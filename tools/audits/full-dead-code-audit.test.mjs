import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {
  walk, declaredTypes, identifierUsedOutside, candidateModules,
  resourceCandidates, audit, privateSymbolCandidates, rustTestModuleRanges,
} from "./full-dead-code-audit.mjs";

test("Kotlin and Swift declaration parsing is intentionally conservative", () => {
  assert.deepEqual(declaredTypes("data class Profile(val id: String)\nobject Vault {}\n", "kotlin"), ["Profile", "Vault"]);
  assert.deepEqual(declaredTypes("struct Pane: View {}\nfinal class Store {}\n", "swift"), ["Pane", "Store"]);
  assert.equal(identifierUsedOutside("Profile", "A.kt", [["A.kt", "class Profile"], ["B.kt", "Profile()"]]), true);
  assert.equal(identifierUsedOutside("Profile", "A.kt", [["A.kt", "class Profile"], ["B.kt", "Profiles()"]]), false);
});

test("only advisory orphan candidates, never deletion decisions", () => {
  const kotlin = [["A.kt", "class A"], ["B.kt", "class B { val a = A() }"]];
  assert.deepEqual(candidateModules(kotlin, "kotlin"), [{ file: "B.kt", declarations: ["B"], reason: "no textual cross-file type reference; OS entrypoints, extensions, reflection and test fixtures may still use it" }]);
  assert.deepEqual(candidateModules([["App.swift", "@main\nstruct App {}"]], "swift"), []);
  assert.deepEqual(resourceCandidates(["res/drawable/icon.xml", "res/drawable/unused.xml"],
    [["Main.kt", "R.drawable.icon"]]), ["res/drawable/unused.xml"]);
});

test("all platform roots are checked; no symlink recursion", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-dead-audit-"));
  const put = (name, content) => {
    const location = path.join(root, name);
    fs.mkdirSync(path.dirname(location), { recursive: true });
    fs.writeFileSync(location, content);
    return location;
  };
  try {
    put("ghostftp-desktop/src/main.tsx", "export {}");
    put("ghostftp-desktop/src-tauri/src/lib.rs", "pub fn work() {}");
    put("android/app/src/main/AndroidManifest.xml", "<activity android:name=\".SplashActivity\"/>");
    put("android/app/src/main/java/app/SplashActivity.kt", "class SplashActivity");
    put("macos/Sources/App.swift", "@main\nstruct App {}");
    put("tools/ghostftp-installer/main.go", "package main\nfunc main() {}");
    put("tools/ghostftp-runtime/main.go", "package main\nfunc main() {}");
    put("docs/build/STATUS.md", "build documentation is source");
    put("android/app/build/generated.txt", "generated - ignore");
    const before = audit(root);
    assert.deepEqual(before.entrypointBlockers, []);
    assert.equal(before.counts.go, 2);
    const fake = path.join(root, "macos", "loop");
    fs.symlinkSync(root, fake, "dir");
    assert.equal(walk(root).length, 8);
    fs.rmSync(path.join(root, "macos/Sources/App.swift"));
    assert.deepEqual(audit(root).entrypointBlockers, ["macOS @main source missing"]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("obsolete npm dev:mock automation must not return", () => {
  const scriptsDir = path.resolve("ghostftp-desktop/scripts");
  const old = walk(scriptsDir).filter((p) => /\.(?:mjs|js)$/.test(p) && /npm[^\n]*dev:mock/.test(fs.readFileSync(p, "utf8")));
  assert.deepEqual(old, [], "deprecated unconfigured mock server scripts are not runnable QA");
});

test("cross-platform nonpublic symbol candidates are advisory and test references count", () => {
  const kt = [["android/Prod.kt", "class Feature {\\n  private fun lost() {}\\n  private fun keep() {}\\n}"],
    ["android/Test.kt", "fun test() { keep() }"]];
  const source = [[kt[0][0], kt[0][1].replaceAll("\\n", "\n")]];
  const external = [[kt[1][0], kt[1][1]]];
  assert.deepEqual(privateSymbolCandidates(source, "kotlin", external).map(x => x.name), ["lost"]);
  assert.deepEqual(privateSymbolCandidates([["macos/View.swift", "private func oldCallback() {}"]], "swift")
    .map(x => x.name), ["oldCallback"]);
  assert.deepEqual(privateSymbolCandidates([["rust/client.rs", "fn orphan_test_helper() {}"]], "rust")
    .map(x => x.name), ["orphan_test_helper"]);
  assert.deepEqual(privateSymbolCandidates([["src/cache.ts", "function unusedLocal() {}"]], "typescript")
    .map(x => x.name), ["unusedLocal"]);
  assert.deepEqual(privateSymbolCandidates([["tools/main.go", "func orphanWorker() {}"]], "go")
    .map(x => x.name), ["orphanWorker"]);
  assert.deepEqual(privateSymbolCandidates([["tools/main.go", "func main() {}\nfunc init() {}"]], "go"), []);
  assert.throws(() => privateSymbolCandidates([], "java"), /Unknown symbol audit language/);
});

test("dead code audit tracks language-specific symbol reports without deleting any files", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-symbol-audit-"));
  try {
    for (const [name, content] of [
      ["android/app/src/main/java/org/Main.kt", "class Demo { private fun unusedHelper() {} }"],
      ["macos/Sources/App.swift", "@main\nstruct App { }\nprivate func unusedSwift() {}"],
      ["ghostftp-desktop/src-tauri/src/lib.rs", "fn unusedRust() {}"],
      ["ghostftp-desktop/src-tauri/tests/protocol_e2e.rs", "#[test]\\nfn realIntegrationTest() {}"],
      ["ghostftp-desktop/src/main.tsx", "function unusedUi() {}"],
      ["tools/ghostftp-installer/main.go", "package main\nfunc main() {}\nfunc unusedInstaller() {}"],
      ["tools/ghostftp-runtime/main.go", "package main\nfunc main() {}"],
      ["android/app/src/main/AndroidManifest.xml", "<activity android:name=\".SplashActivity\"/>"],
      ["android/app/src/main/java/org/SplashActivity.kt", "class SplashActivity"]
    ]) {
      const full = path.join(root, name);
      fs.mkdirSync(path.dirname(full), {recursive:true});
      fs.writeFileSync(full, content);
    }
    const report = audit(root);
    assert.deepEqual(report.entrypointBlockers, []);
    assert.ok("desktopRust" in report.advisoryPrivateSymbols);
    assert.ok(report.advisoryPrivateSymbols.desktopRust.some(x => x.name === "unusedRust"));
    assert.ok(!report.advisoryPrivateSymbols.desktopRust.some(x => x.name === "realIntegrationTest"),
      "Rust integration test harness entrypoints are not dead production functions");
    assert.ok("desktopTypeScript" in report.advisoryPrivateSymbols);
    assert.ok("installerRuntimeGo" in report.advisoryPrivateSymbols);
  } finally {
    fs.rmSync(root, {recursive:true, force:true});
  }
});

test("Rust private function inventory excludes cfg(test) and test attribute entrypoints", () => {
  const rust = `fn productionOrphan() {}
#[cfg(test)]
mod tests {
    #[test]
    fn exercisedOnlyByHarness() {}
    fn testHelper() {}
}`;
  assert.deepEqual(privateSymbolCandidates([["src/lib.rs", rust]], "rust")
    .map(x => x.name), ["productionOrphan"]);
});

test("Rust test-only scope ends at matching closing brace, not end of file", () => {
  const rust = \`fn productionBefore() {}
#[cfg(test)]
mod tests {
  #[test]
  fn testCase() { let json = "{not-a-module-brace}"; }
  fn fixtureInsideTest() {}
  // a brace in a comment must not count: }
  mod child { fn nestedTestOnly() {} }
}
fn productionAfter() {}
#[cfg(test)]
fn testOnlyOutsideModule() {}
fn productionLast() {}\`;
  const names = privateSymbolCandidates([["src/lib.rs", rust]], "rust")
    .map((x) => x.name);
  assert.deepEqual(names, ["productionBefore", "productionAfter", "productionLast"]);
  assert.equal(rustTestModuleRanges(rust).length, 1);
});

test("Rust adjacent #[cfg(test)] and #[tokio::test] exclude only attributed functions", () => {
  const rust = \`fn prod1() {}
#[cfg(test)]
fn setupForTests() {}
fn prod2() {}
#[tokio::test]
async fn asyncTest() {}
fn prod3() {}\`;
  assert.deepEqual(privateSymbolCandidates([["src/lib.rs", rust]], "rust")
    .map((x) => x.name), ["prod1", "prod2", "prod3"]);
});

test("Cross-platform inventory includes unexported TypeScript arrows and Go receiver methods", () => {
  const ts = [
    ["desktop/src/feature.ts", "const unusedArrow = (value) => value;\nconst usedArrow = () => 1;\nusedArrow();"],
  ];
  assert.deepEqual(privateSymbolCandidates(ts, "typescript").map(x => x.name),
    ["unusedArrow"]);
  const go = [
    ["tools/ghostftp-runtime/runtime.go", "package main\nfunc (r *Runtime) unusedWorker() {}\nfunc (r *Runtime) usedWorker() {}"],
    ["tools/ghostftp-runtime/main.go", "func initRuntime(r *Runtime) { r.usedWorker() }"],
  ];
  assert.ok(privateSymbolCandidates(go, "go")
    .some(x => x.name === "unusedWorker"));
  assert.ok(!privateSymbolCandidates(go, "go")
    .some(x => x.name === "usedWorker"));
  assert.deepEqual(privateSymbolCandidates([
    ["src/exported.ts", "export const keep = () => 123;\nexport function other() {}"]
  ], "typescript"), []);
});
