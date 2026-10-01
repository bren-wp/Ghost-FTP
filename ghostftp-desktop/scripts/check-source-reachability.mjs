#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const desktopRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const projectRoot = path.resolve(desktopRoot, "..");
const sourceRoots = [
  path.join(desktopRoot, "src"),
  path.join(desktopRoot, "packages", "file-ui", "src"),
];
const ignored = new Set([
  path.join(desktopRoot, "src", "vite-env.d.ts"),
]);

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else if (/\.(?:ts|tsx)$/.test(entry.name) && !ignored.has(full)) out.push(full);
  }
  return out;
}

const files = sourceRoots.flatMap(walk);
const fileSet = new Set(files.map((file) => path.normalize(file)));
const roots = [
  path.join(desktopRoot, "src", "main.tsx"),
  path.join(desktopRoot, "packages", "file-ui", "src", "index.ts"),
].map((file) => path.normalize(file));

const importPattern =
  /(?:import|export)\s+(?:type\s+)?(?:[^"'()]*?\s+from\s+)?["']([^"']+)["']|import\s*\(\s*["']([^"']+)["']\s*\)/g;

function resolveModule(fromFile, specifier) {
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
      specifier.slice("@ghostftp/file-ui/".length)
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
  ].map((file) => path.normalize(file));

  return candidates.find((candidate) => fileSet.has(candidate)) ?? null;
}

const graph = new Map(files.map((file) => [path.normalize(file), new Set()]));
for (const file of files) {
  const normalized = path.normalize(file);
  const source = fs.readFileSync(file, "utf8");
  importPattern.lastIndex = 0;
  let match;
  while ((match = importPattern.exec(source))) {
    const target = resolveModule(file, match[1] ?? match[2]);
    if (target) graph.get(normalized).add(target);
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

const unreachable = [...fileSet]
  .filter((file) => !reachable.has(file))
  .map((file) => path.relative(projectRoot, file).replaceAll("\\", "/"))
  .sort();

if (unreachable.length) {
  console.error("Dead frontend source files detected:");
  for (const file of unreachable) console.error(` - ${file}`);
  console.error("Delete them or make the intended entrypoint/import explicit.");
  process.exit(1);
}

console.log(`Frontend source reachability OK: ${reachable.size} TypeScript files reachable.`);
