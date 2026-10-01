#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const desktopRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const projectRoot = path.resolve(desktopRoot, "..");

function walk(dir, predicate = () => true) {
  if (!fs.existsSync(dir)) return [];
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full, predicate));
    else if (predicate(full)) out.push(path.normalize(full));
  }
  return out;
}
const relative = (file) => path.relative(projectRoot, file).replaceAll("\\", "/");

// TypeScript/React production entrypoint graph.
const tsSourceRoots = [
  path.join(desktopRoot, "src"),
  path.join(desktopRoot, "packages", "file-ui", "src"),
];
const tsIgnored = new Set([path.normalize(path.join(desktopRoot, "src", "vite-env.d.ts"))]);
const tsFiles = tsSourceRoots
  .flatMap((root) => walk(root, (file) => /\.(?:ts|tsx)$/.test(file)))
  .filter((file) => !tsIgnored.has(file));
const tsFileSet = new Set(tsFiles);
const tsRoots = [
  path.join(desktopRoot, "src", "main.tsx"),
  path.join(desktopRoot, "packages", "file-ui", "src", "index.ts"),
].map(path.normalize);
const tsImportPattern =
  /(?:import|export)\s+(?:type\s+)?(?:[^"'()]*?\s+from\s+)?["']([^"']+)["']|import\s*\(\s*["']([^"']+)["']\s*\)/g;

function resolveTsModule(fromFile, specifier) {
  let base = null;
  if (specifier.startsWith("@/")) {
    base = path.join(desktopRoot, "src", specifier.slice(2));
  } else if (specifier === "@ghostftp/file-ui") {
    base = path.join(desktopRoot, "packages", "file-ui", "src", "index");
  } else if (specifier.startsWith("@ghostftp/file-ui/")) {
    base = path.join(
      desktopRoot,
      "packages",
      "file-ui",
      "src",
      specifier.slice("@ghostftp/file-ui/".length),
    );
  } else if (specifier.startsWith(".")) {
    base = path.resolve(path.dirname(fromFile), specifier);
  } else {
    return null;
  }
  const candidates = [
    base,
    `${base}.ts`,
    `${base}.tsx`,
    path.join(base, "index.ts"),
    path.join(base, "index.tsx"),
  ].map(path.normalize);
  return candidates.find((candidate) => tsFileSet.has(candidate)) ?? null;
}

const tsGraph = new Map(tsFiles.map((file) => [file, new Set()]));
for (const file of tsFiles) {
  const source = fs.readFileSync(file, "utf8");
  tsImportPattern.lastIndex = 0;
  let match;
  while ((match = tsImportPattern.exec(source))) {
    const target = resolveTsModule(file, match[1] ?? match[2]);
    if (target) tsGraph.get(file).add(target);
  }
}
const tsReachable = new Set();
const tsStack = [...tsRoots];
while (tsStack.length) {
  const file = tsStack.pop();
  if (!file || tsReachable.has(file)) continue;
  tsReachable.add(file);
  for (const next of tsGraph.get(file) ?? []) tsStack.push(next);
}
const unreachableTs = [...tsFileSet]
  .filter((file) => !tsReachable.has(file))
  .map(relative)
  .sort();

// Rust module graph. Compiler/clippy cannot see an orphan .rs file that no
// crate root includes, so prove source-file reachability before those checks.
const rustCrates = [
  path.join(desktopRoot, "src-tauri", "src"),
  path.join(desktopRoot, "src-tauri", "ghostftp-cli", "src"),
  path.join(desktopRoot, "src-tauri", "ghostftp-agent-proto", "src"),
  path.join(desktopRoot, "src-tauri", "ghostftp-agentd", "src"),
];
const rustModulePattern =
  /(?:#\s*\[\s*path\s*=\s*"([^"]+)"\s*\]\s*)?(?:pub(?:\s*\([^)]*\))?\s+)?mod\s+([A-Za-z_][A-Za-z0-9_]*)\s*;/g;
const rustIncludePattern = /include!\s*\(\s*"([^"]+)"\s*\)/g;
const stripRustComments = (source) =>
  source.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
let rustReachableCount = 0;
const unreachableRust = [];

for (const crateRoot of rustCrates) {
  const files = walk(crateRoot, (file) => file.endsWith(".rs"));
  const fileSet = new Set(files);
  const graph = new Map(files.map((file) => [file, new Set()]));
  const roots = ["lib.rs", "main.rs"]
    .map((name) => path.normalize(path.join(crateRoot, name)))
    .filter((file) => fileSet.has(file));

  const moduleBaseDir = (file) => {
    const stem = path.basename(file, ".rs");
    return stem === "lib" || stem === "main" || stem === "mod"
      ? path.dirname(file)
      : path.join(path.dirname(file), stem);
  };

  for (const file of files) {
    const source = stripRustComments(fs.readFileSync(file, "utf8"));
    rustModulePattern.lastIndex = 0;
    let match;
    while ((match = rustModulePattern.exec(source))) {
      const override = match[1];
      const name = match[2];
      const candidates = override
        ? [path.resolve(path.dirname(file), override)]
        : [
            path.join(moduleBaseDir(file), `${name}.rs`),
            path.join(moduleBaseDir(file), name, "mod.rs"),
          ];
      const target = candidates.map(path.normalize).find((candidate) => fileSet.has(candidate));
      if (target) graph.get(file).add(target);
    }
    rustIncludePattern.lastIndex = 0;
    while ((match = rustIncludePattern.exec(source))) {
      const target = path.normalize(path.resolve(path.dirname(file), match[1]));
      if (fileSet.has(target)) graph.get(file).add(target);
    }
  }

  const reachable = new Set();
  const stack = [...roots];
  while (stack.length) {
    const file = stack.pop();
    if (!file || reachable.has(file)) continue;
    reachable.add(file);
    for (const next of graph.get(file) ?? []) stack.push(next);
  }
  rustReachableCount += reachable.size;
  unreachableRust.push(...files.filter((file) => !reachable.has(file)).map(relative));
}

// Operational scripts are build/release entrypoints, not imported modules.
// Require a real workflow, package, runbook or helper reference for each.
const utilityDirs = [
  path.join(projectRoot, ".github", "scripts"),
  path.join(projectRoot, "updates", "scripts"),
  path.join(projectRoot, "android", "scripts"),
];
const utilityFiles = utilityDirs.flatMap((dir) =>
  walk(dir, (file) => /\.(?:mjs|sh)$/.test(file)),
);
const referenceFiles = [
  ...walk(path.join(projectRoot, ".github", "workflows"), (file) => /\.ya?ml$/.test(file)),
  ...walk(path.join(projectRoot, ".github", "scripts"), (file) => /\.(?:mjs|sh)$/.test(file)),
  ...walk(path.join(projectRoot, "updates"), (file) => /\.(?:md|mjs|sh)$/.test(file)),
  ...walk(path.join(projectRoot, "android"), (file) => /\.(?:md|mjs|sh|kts)$/.test(file)),
  path.join(projectRoot, "ghostftp-desktop", "package.json"),
  path.join(projectRoot, "docs", "development", "BUILDING.md"),
  path.join(projectRoot, "docs", "release", "PROCESS.md"),
].filter((file) => fs.existsSync(file));

const unreferencedUtilities = utilityFiles
  .filter((utility) => {
    const rel = relative(utility);
    const basename = path.basename(utility);
    return !referenceFiles.some((ref) => {
      if (path.normalize(ref) === path.normalize(utility)) return false;
      const source = fs.readFileSync(ref, "utf8");
      return source.includes(rel) || source.includes(basename);
    });
  })
  .map(relative)
  .sort();

const failures = [];
if (unreachableTs.length) {
  failures.push("Dead frontend source files detected:", ...unreachableTs.map((file) => ` - ${file}`));
}
if (unreachableRust.length) {
  failures.push(
    "Orphan Rust source files outside every crate module graph:",
    ...unreachableRust.sort().map((file) => ` - ${file}`),
  );
}
if (unreferencedUtilities.length) {
  failures.push(
    "Unreferenced CI/update/Android utility scripts detected:",
    ...unreferencedUtilities.map((file) => ` - ${file}`),
  );
}
if (failures.length) {
  console.error(failures.join("\n"));
  console.error(
    "Delete only after proving a file is outside runtime/build/test/release/compatibility paths, or add its real operational reference.",
  );
  process.exit(1);
}

console.log(
  `Source reachability OK: ${tsReachable.size} TypeScript files, ${rustReachableCount} Rust files, ${utilityFiles.length} operational utility scripts.`,
);
