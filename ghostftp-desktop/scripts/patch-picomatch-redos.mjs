/**
 * Ghost FTP emergency build-time mitigation for CVE-2026-112366.
 * Three pinned npm dependencies have no published official security fix.
 * --verify requires a local patch and bounded regression tests; it never writes.
 */
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";

// Self-contained inside Desktop-Source.zip, not dependent on .github files.
const desktop = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const lock = JSON.parse(readFileSync(join(desktop, "package-lock.json"), "utf8"));
const versions = new Map([
  ["node_modules/picomatch", "2.3.2"],
  ["node_modules/tinyglobby/node_modules/picomatch", "4.0.4"],
  ["node_modules/vite/node_modules/picomatch", "4.0.7"],
]);
const marker = "// Ghost FTP CVE-2026-112366 locally hardened repeated extglob";
const unsafeOptOut = [
  "  if (options.maxExtglobRecursion === false) {",
  "    return { risky: false };",
  "  }",
  "",
].join("\n");
const anchor = [
  "  return { risky: false };",
  "};",
  "",
  "/**",
  " * Parse the given input string.",
].join("\n");
const replacement = [
  "  " + marker,
  "  // Guard against nested unbounded quantifiers from +(*) and +(*a).",
  "  // Earlier known-safe class rewrites still execute first.",
  "  // Unhandled inner wildcard branches become literal extglobs.",
  "  if (branches.some(branch => branch.includes('*'))) {",
  "    return { risky: true };",
  "  }",
  "",
  ...anchor.split("\n"),
].join("\n");
const count = (body, needle) => body.split(needle).length - 1;

export function checkSources({ write = false } = {}) {
  const actual = Object.entries(lock.packages ?? {})
    .filter(([name]) => name.endsWith("/picomatch"))
    .map(([name, info]) => [name, info.version])
    .sort((a, b) => a[0].localeCompare(b[0]));
  const expected = [...versions.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  assert.deepEqual(actual, expected, "picomatch dependency graph changed: manual patch review required");

  for (const [packagePath, version] of versions) {
    const file = join(desktop, packagePath, "lib", "parse.js");
    assert.ok(statSync(file).isFile(), "missing parser: " + packagePath);
    let source = readFileSync(file, "utf8");
    if (!source.includes(marker)) {
      assert.ok(write, packagePath + " is not locally patched");
      assert.equal(count(source, unsafeOptOut), 1, "picomatch safety opt-out changed");
      assert.equal(count(source, anchor), 1, "picomatch analyzeRepeatedExtglob changed");
      assert.equal(count(source, "const analyzeRepeatedExtglob = (body, options) => {"), 1);
      source = source.replace(unsafeOptOut, "").replace(anchor, replacement);
      writeFileSync(file, source, "utf8");
    }
    const updated = readFileSync(file, "utf8");
    assert.equal(count(updated, marker), 1, "missing/duplicated local patch");
    assert.ok(updated.includes(replacement), "patched security guard was modified");
    assert.ok(!updated.includes(unsafeOptOut), "unsafe recursion opt-out remains");
    assert.equal(
      JSON.parse(readFileSync(join(desktop, packagePath, "package.json"), "utf8")).version,
      version,
      "installed parser differs from locked version"
    );
  }
}

function runBoundedRegressions() {
  for (const [packagePath] of versions) {
    const packageDir = join(desktop, packagePath);
    const js = [
      "const assert = require('node:assert/strict');",
      "const p = require(process.argv[1]);",
      "for (const pattern of ['+(*)', '+(*a)', '*(*)', '+(a*)', '+(*.js)']) {",
      "  assert.equal(p.isMatch('a'.repeat(512) + '!', pattern), false, pattern);",
      "}",
      "for (const [pattern, value, expected] of [",
      "  ['**/*.{js,ts}', 'src/main.ts', true],",
      "  ['src/@(app|index).ts', 'src/app.ts', true],",
      "  ['src/@(app|index).ts', 'src/other.ts', false],",
      "  ['*.js', 'file.js', true],",
      "]) { assert.equal(p.isMatch(value, pattern), expected, pattern); }",
    ].join("\n");
    execFileSync(process.execPath, ["-e", js, packageDir], {
      timeout: 3500,
      windowsHide: true,
      stdio: "pipe",
    });
  }
}

const verify = process.argv.includes("--verify");
checkSources({ write: !verify });
runBoundedRegressions();
console.log(
  "Ghost FTP locally patched all 3 picomatch copies for CVE-2026-112366; upstream npm advisory remains unresolved."
);
