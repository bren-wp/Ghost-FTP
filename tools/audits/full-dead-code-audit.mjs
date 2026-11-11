#!/usr/bin/env node
/**
 * Conservative repository-wide dead-code inventory.
 * No file or public API is deleted automatically. Compiler, runtime, OS and
 * compatibility evidence is required before acting on advisory candidates.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const ignoredDirs = new Set([".git", "node_modules", ".gradle", ".build", "build", "target", "dist", ".idea", "coverage"]);
const posix = (value) => value.replaceAll("\\", "/");
const fromRoot = (file) => posix(path.relative(root, file));

export function walk(directory) {
  if (!fs.existsSync(directory)) return [];
  const result = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const next = path.join(directory, entry.name);
    if (entry.isSymbolicLink()) continue;
    if (entry.isDirectory()) {
      // docs/build is tracked documentation, unlike generated build outputs.
      if (ignoredDirs.has(entry.name) && !(entry.name === "build" && path.basename(directory) === "docs")) continue;
      result.push(...walk(next));
    } else if (entry.isFile()) result.push(next);
  }
  return result;
}

/**
 * Find the actual lexical brace span of Rust #[cfg(test)] mod blocks.
 * The old implementation suppressed every subsequent production function
 * after the first test module, including functions declared OUTSIDE it.
 * This matcher is conservative: it excludes obvious test modules, never
 * deletes code, and returns advisory symbols for compiler verification.
 */
export function rustTestModuleRanges(source) {
  const ranges = [];
  const headers = /#\[\s*cfg\s*\(\s*test\s*\)\s*\]\s*(?:#\[[^\]]+\]\s*)*mod\s+\w+\s*\{/g;
  for (const header of source.matchAll(headers)) {
    const start = source.indexOf("{", header.index);
    if (start < 0) continue;
    let depth = 0;
    let finish = source.length;
    let mode = "code";
    for (let index = start; index < source.length; index++) {
      const ch = source[index], next = source[index + 1];
      if (mode === "line") { if (ch === "\n") mode = "code"; continue; }
      if (mode === "block") {
        if (ch === "*" && next === "/") { mode = "code"; index++; }
        continue;
      }
      if (mode === "string") {
        if (ch === "\\") { index++; continue; }
        if (ch === '"') mode = "code";
        continue;
      }
      if (ch === "/" && next === "/") { mode = "line"; index++; continue; }
      if (ch === "/" && next === "*") { mode = "block"; index++; continue; }
      if (ch === '"') { mode = "string"; continue; }
      if (ch === "{") depth++;
      else if (ch === "}" && --depth === 0) {
        finish = index + 1;
        break;
      }
    }
    ranges.push([header.index, finish]);
  }
  return ranges;
}

/**
 * Symbol-level advisory scan for ALL maintained native/source languages.
 * A sole textual mention of a private function is a review candidate, NOT
 * a proof of unreachability: reflection, callbacks, build tags, macros and
 * selectors are deliberate blind spots. Preserve platform entrypoints.
 */
export function privateSymbolCandidates(entries, language, external = []) {
  const patterns = {

    rust: /^\s*(?:(?:pub\((?:crate|super|self)\)\s+)?)(?:async\s+)?fn\s+([A-Za-z_]\w*)\s*\(/gm,
    // Include unexported arrow callbacks as advisory symbols, in addition
    // to ordinary local functions. Dynamic UI consumers still need review.
    typescript: /^\s*(?:(?!export\b)(?:async\s+)?function\s+([A-Za-z_]\w*)\s*\(|(?:const|let)\s+([A-Za-z_]\w*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_]\w*)\s*=>)/gm,
    go: /^func\s+(?:\([^)]*\)\s*)?([a-z_]\w*)\s*\(/gm,
  };
  const pattern = patterns[language];
  if (!pattern) throw new Error("Unknown symbol audit language: " + language);
  const referents = [...entries, ...external].map(([, code]) => code).join("\n");
  const testRanges = language === "rust"
    ? new Map(entries.map(([file, source]) => [file, rustTestModuleRanges(source)]))
    : new Map();
  const reserved = new Set(["init", "main", "body"]);
  return entries.flatMap(([file, source]) => [...source.matchAll(pattern)]
    .filter((m) => !reserved.has(m[1] || m[2]))
    // Exclude only Rust functions INSIDE a cfg(test) module or with an
    // immediately adjacent test attribute. Do not suppress later production
    // functions in the same Rust source file.
    .filter((m) => language !== "rust" ||
      (!(testRanges.get(file) || []).some(([start, end]) =>
        m.index >= start && m.index < end) &&
       !/#\[\s*(?:cfg\(test\)|(?:tokio::)?test)\s*\]\s*$/.test(
         source.slice(Math.max(0, m.index - 160), m.index))))
    .filter((m) => {
      const name = m[1] || m[2];
      const occurrences = referents.match(new RegExp("\\b" + name + "\\b", "g")) || [];
      return occurrences.length === 1;
    }).map((m) => ({
      file,
      name: m[1] || m[2],
      line: source.slice(0, m.index).split("\n").length,
      reason: "Single textual mention of nonpublic function across source and tests; callback/reflection/macro/build-tag references must be inspected before removal",
    })));
}

export function audit(directory = root) {
  const files = walk(directory);
  const inside = (prefix) => path.join(directory, prefix);
  const go = files.filter((p) => /\/tools\/ghostftp-(?:installer|runtime)\/.*\.go$/.test(posix(p)));
  const goSources = go.filter((p) => !p.endsWith("_test.go"))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const goTests = go.filter((p) => p.endsWith("_test.go"))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const rust = files.filter((p) => p.startsWith(inside("ghostftp-desktop/src-tauri/")) && p.endsWith(".rs"));
  // Rust integration tests are binaries with harness-registered #[test]
  // entrypoints, never production functions. Include them as references,
  // but never report their own test functions as unused application code.
  const rustTests = rust.filter((p) => /\/(?:tests|test)\//.test(posix(p)))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const rustSources = rust.filter((p) => !/\/(?:tests|test)\//.test(posix(p)))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const typescript = files.filter((p) => p.startsWith(inside("ghostftp-desktop/")) && /\.(?:ts|tsx)$/.test(p));
  const tsSources = typescript.filter((p) => !/\.(?:test|spec)\.(?:ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const tsTests = typescript.filter((p) => /\.(?:test|spec)\.(?:ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const scripts = files.filter((p) => /\.(?:sh|mjs|py)$/.test(p) &&
    [".github/scripts/", "ghostftp-desktop/scripts/", "updates/scripts/", "tools/"]
      .some((prefix) => p.startsWith(inside(prefix))));
  const refs = files.filter((p) => /\.(?:md|mjs|js|json|yaml|yml|sh|py|kts|rs|kt|ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const possibleUnreferencedScripts = scripts.filter((file) => {
    const relative = posix(path.relative(directory, file));
    const name = path.basename(file);
    return !refs.some(([other, text]) => other !== relative && (text.includes(relative) || text.includes(name)));
  }).map((p) => posix(path.relative(directory, p))).sort();
  const blockers = [];
  for (const name of ["ghostftp-installer", "ghostftp-runtime"]) {
    if (!go.some((p) => posix(p).endsWith("/" + name + "/main.go"))) blockers.push("Go tool entrypoint missing: " + name);
  }
  if (!rust.some((p) => posix(p).endsWith("/src/lib.rs"))) blockers.push("Tauri lib.rs missing");
  if (!typescript.some((p) => posix(p).endsWith("/src/main.tsx"))) blockers.push("React main.tsx missing");
  return {
    method: "Source-level conservative reference inventory; CANDIDATES are not proven dead code",
    counts: {
      workingTreeFiles: files.length,
      typescript: typescript.length,
      rust: rust.length,
      go: go.length,
      operationalScripts: scripts.length,
    },
    candidateFiles: {
      operationalScripts: possibleUnreferencedScripts,
    },
    advisoryPrivateSymbols: {
      desktopRust: privateSymbolCandidates(rustSources, "rust", rustTests),
      desktopTypeScript: privateSymbolCandidates(tsSources, "typescript", tsTests),
      installerRuntimeGo: privateSymbolCandidates(goSources, "go", goTests),
    },
    entrypointBlockers: blockers,
    requiredFollowUp: [
      "TS/Rust graph: npm run check:dead, cargo check/test/clippy on all targets and features",
      "Go: go test, go vet and GOOS-specific build tags",
      "Shell/CI: references, platform builds, release and rollback workflow",
      "Only delete after compiler evidence plus dynamic compatibility and user-data review",
    ],
  };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = audit();
  if (process.argv.includes("--json")) console.log(JSON.stringify(result, null, 2));
  else {
    console.log("Full repository source inventory: " + JSON.stringify(result.counts));
    console.log("Candidate counts (scripts): " +
      Object.values(result.candidateFiles).map((items) => items.length).join("/"));
    console.log("Advisory candidates MUST NOT be removed solely from this report.");
    console.log("Unreferenced nonpublic function candidates by platform: " +
      Object.entries(result.advisoryPrivateSymbols).map(([name, items]) =>
        name + "=" + items.length).join(", "));
    if (result.entrypointBlockers.length) console.error(result.entrypointBlockers.join("\n"));
  }
  if (result.entrypointBlockers.length) process.exitCode = 1;
}
