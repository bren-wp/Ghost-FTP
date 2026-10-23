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
const ignoredDirs = new Set([".git", "node_modules", ".gradle", ".build", "build", "target", "dist", ".swiftpm", ".idea", "coverage"]);
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

export function declaredTypes(source, language) {
  const regex = language === "kotlin"
    ? /^(?:(?:public|internal|private|protected|data|sealed|open|abstract|final|enum|annotation|value|expect|actual)\s+)*(?:class|object|interface|typealias)\s+([A-Za-z_]\w*)\b/gm
    : /^(?:(?:public|internal|private|fileprivate|open|final|indirect|@MainActor|@Observable)\s+)*(?:class|struct|enum|actor|protocol|typealias)\s+([A-Za-z_]\w*)\b/gm;
  return [...source.matchAll(regex)].map((m) => m[1]);
}

export function identifierUsedOutside(name, sourceFile, entries) {
  const token = new RegExp("\\b" + name + "\\b");
  return entries.some(([file, text]) => file !== sourceFile && token.test(text));
}

export function candidateModules(entries, language, external = []) {
  const others = [...entries, ...external];
  return entries.flatMap(([file, source]) => {
    if (language === "swift" && /@main\b/.test(source)) return [];
    // Activity and service classes may be referenced by AndroidManifest.xml.
    const names = declaredTypes(source, language);
    if (names.some((name) => identifierUsedOutside(name, file, others))) return [];
    return [{
      file, declarations: names,
      reason: "no textual cross-file type reference; OS entrypoints, extensions, reflection and test fixtures may still use it",
    }];
  });
}

/**
 * Symbol-level advisory scan for ALL maintained native/source languages.
 * A sole textual mention of a private function is a review candidate, NOT
 * a proof of unreachability: reflection, callbacks, build tags, macros and
 * selectors are deliberate blind spots. Preserve platform entrypoints.
 */
export function privateSymbolCandidates(entries, language, external = []) {
  const patterns = {
    kotlin: /^\s*(?:private|internal)\s+(?:suspend\s+)?fun\s+([A-Za-z_]\w*)\s*\(/gm,
    swift: /^\s*(?:private|fileprivate)\s+(?:static\s+|class\s+)?func\s+([A-Za-z_]\w*)\s*\(/gm,
    rust: /^\s*(?:(?:pub\((?:crate|super|self)\)\s+)?)(?:async\s+)?fn\s+([A-Za-z_]\w*)\s*\(/gm,
    typescript: /^\s*(?!export\b)(?:async\s+)?function\s+([A-Za-z_]\w*)\s*\(/gm,
    go: /^func\s+([a-z_]\w*)\s*\(/gm,
  };
  const pattern = patterns[language];
  if (!pattern) throw new Error("Unknown symbol audit language: " + language);
  const referents = [...entries, ...external].map(([, code]) => code).join("\n");
  const reserved = new Set(["init", "main", "body", "onCreate", "onDestroy", "onStart",
    "onStop", "onResume", "onPause", "onSaveInstanceState", "onActivityResult"]);
  return entries.flatMap(([file, source]) => [...source.matchAll(pattern)]
    .filter((m) => !reserved.has(m[1]))
    // Most Rust modules put #[cfg(test)] mod tests at file end; those
    // functions are test entrypoints, not production-orphan candidates.
    // This conservative exclusion may hide functions after the test module,
    // therefore all reported symbols remain advisory only.
    .filter((m) => language !== "rust" ||
      (!source.slice(0, m.index).includes("#[cfg(test)]") &&
       !/#\\[(?:tokio::)?test\\]\\s*$/.test(source.slice(Math.max(0, m.index - 140), m.index))))
    .filter((m) => {
      const name = m[1];
      const occurrences = referents.match(new RegExp("\\b" + name + "\\b", "g")) || [];
      return occurrences.length === 1;
    }).map((m) => ({
      file,
      name: m[1],
      line: source.slice(0, m.index).split("\n").length,
      reason: "Single textual mention of nonpublic function across source and tests; callback/reflection/macro/build-tag references must be inspected before removal",
    })));
}

export function resourceCandidates(resources, sources) {
  const references = sources.map(([, text]) => text).join("\n");
  return resources.filter((file) => {
    const name = path.basename(file).replace(/\.(?:xml|png|webp|jpg|jpeg)$/i, "");
    return !new RegExp("\\b" + name.replace(/[^A-Za-z0-9_]/g, "") + "\\b").test(references);
  });
}

export function audit(directory = root) {
  const files = walk(directory);
  const inside = (prefix) => path.join(directory, prefix);
  const lookup = (prefix, suffix) => files.filter((p) => p.startsWith(inside(prefix)) && p.endsWith(suffix))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const kotlin = lookup("android/app/src/main/java/", ".kt");
  const kotlinTests = [...lookup("android/app/src/test/", ".kt"), ...lookup("android/app/src/androidTest/", ".kt")];
  const swift = lookup("macos/Sources/", ".swift");
  const swiftTests = lookup("macos/Tests/", ".swift");
  const manifestFile = inside("android/app/src/main/AndroidManifest.xml");
  const manifest = fs.existsSync(manifestFile) ? fs.readFileSync(manifestFile, "utf8") : "";
  const swiftPackage = inside("macos/Package.swift");
  const packageText = fs.existsSync(swiftPackage) ? fs.readFileSync(swiftPackage, "utf8") : "";
  const androidResources = files.filter((p) => p.startsWith(inside("android/app/src/main/res/")));
  const xmlReferences = androidResources.filter((p) => p.endsWith(".xml"))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const resources = androidResources.filter((p) => /\/(?:drawable|mipmap[^/]*|xml)\//.test(posix(p)) && /\.(?:xml|png|webp|jpg|jpeg)$/i.test(p));
  const go = files.filter((p) => /\/tools\/ghostftp-(?:installer|runtime)\/.*\.go$/.test(posix(p)));
  const goSources = go.filter((p) => !p.endsWith("_test.go"))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const goTests = go.filter((p) => p.endsWith("_test.go"))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const rust = files.filter((p) => p.startsWith(inside("ghostftp-desktop/src-tauri/")) && p.endsWith(".rs"));
  const rustSources = rust.map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const typescript = files.filter((p) => p.startsWith(inside("ghostftp-desktop/")) && /\.(?:ts|tsx)$/.test(p));
  const tsSources = typescript.filter((p) => !/\.(?:test|spec)\.(?:ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const tsTests = typescript.filter((p) => /\.(?:test|spec)\.(?:ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const scripts = files.filter((p) => /\.(?:sh|mjs|py)$/.test(p) &&
    [".github/scripts/", "ghostftp-desktop/scripts/", "android/scripts/", "macos/scripts/", "updates/scripts/", "tools/"]
      .some((prefix) => p.startsWith(inside(prefix))));
  const refs = files.filter((p) => /\.(?:md|mjs|js|json|yaml|yml|sh|py|kts|rs|kt|swift|ts|tsx)$/.test(p))
    .map((p) => [posix(path.relative(directory, p)), fs.readFileSync(p, "utf8")]);
  const possibleUnreferencedScripts = scripts.filter((file) => {
    const relative = posix(path.relative(directory, file));
    const name = path.basename(file);
    return !refs.some(([other, text]) => other !== relative && (text.includes(relative) || text.includes(name)));
  }).map((p) => posix(path.relative(directory, p))).sort();
  const blockers = [];
  if (!manifest.includes("SplashActivity") || !kotlin.some(([p]) => p.endsWith("/SplashActivity.kt"))) blockers.push("Android launcher missing");
  if (!swift.some(([, source]) => /@main\b/.test(source))) blockers.push("macOS @main source missing");
  for (const name of ["ghostftp-installer", "ghostftp-runtime"]) {
    if (!go.some((p) => posix(p).endsWith("/" + name + "/main.go"))) blockers.push("Go tool entrypoint missing: " + name);
  }
  if (!rust.some((p) => posix(p).endsWith("/src/lib.rs"))) blockers.push("Tauri lib.rs missing");
  if (!typescript.some((p) => posix(p).endsWith("/src/main.tsx"))) blockers.push("React main.tsx missing");
  const relative = (p) => posix(path.relative(directory, p));
  return {
    method: "Source-level conservative reference inventory; CANDIDATES are not proven dead code",
    counts: {
      workingTreeFiles: files.length,
      typescript: typescript.length,
      rust: rust.length,
      kotlin: kotlin.length,
      kotlinTests: kotlinTests.length,
      androidResources: androidResources.length,
      swift: swift.length,
      swiftTests: swiftTests.length,
      go: go.length,
      operationalScripts: scripts.length,
    },
    candidateFiles: {
      androidKotlin: candidateModules(kotlin, "kotlin", [...kotlinTests, ["AndroidManifest.xml", manifest]]).sort((a, b) => a.file.localeCompare(b.file)),
      macosSwift: candidateModules(swift, "swift", [...swiftTests, ["Package.swift", packageText]]).sort((a, b) => a.file.localeCompare(b.file)),
      androidResources: resourceCandidates(resources, [...kotlin, ...kotlinTests, ...xmlReferences, ["AndroidManifest.xml", manifest]]).map(relative).sort(),
      operationalScripts: possibleUnreferencedScripts,
    },
    advisoryPrivateSymbols: {
      androidKotlin: privateSymbolCandidates(kotlin, "kotlin", kotlinTests),
      macosSwift: privateSymbolCandidates(swift, "swift", swiftTests),
      desktopRust: privateSymbolCandidates(rustSources, "rust"),
      desktopTypeScript: privateSymbolCandidates(tsSources, "typescript", tsTests),
      installerRuntimeGo: privateSymbolCandidates(goSources, "go", goTests),
    },
    entrypointBlockers: blockers,
    requiredFollowUp: [
      "TS/Rust graph: npm run check:dead, cargo check/test/clippy on all targets and features",
      "Android: Gradle unit tests, Android Lint, R8 shrinker, manifest and resources, emulator",
      "macOS: swift build/test, package contract and actual macOS app launch",
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
    console.log("Candidate counts (Kotlin/Swift/resources/scripts): " +
      Object.values(result.candidateFiles).map((items) => items.length).join("/"));
    console.log("Advisory candidates MUST NOT be removed solely from this report.");
    console.log("Unreferenced nonpublic function candidates by platform: " +
      Object.entries(result.advisoryPrivateSymbols).map(([name, items]) =>
        name + "=" + items.length).join(", "));
    if (result.entrypointBlockers.length) console.error(result.entrypointBlockers.join("\n"));
  }
  if (result.entrypointBlockers.length) process.exitCode = 1;
}
