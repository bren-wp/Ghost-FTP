import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {
  walk, declaredTypes, identifierUsedOutside, candidateModules,
  audit, privateSymbolCandidates, rustTestModuleRanges,
} from "./full-dead-code-audit.mjs";

const temporary = (callback) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-desktop-audit-"));
  const put = (name, content) => {
    const location = path.join(root, name);
    fs.mkdirSync(path.dirname(location), {recursive: true});
    fs.writeFileSync(location, content);
  };
  try { callback(root, put); }
  finally { fs.rmSync(root, {recursive: true, force: true}); }
};

test("Swift module inventory conservatively treats OS entrypoints as reachable", () => {
  assert.deepEqual(declaredTypes("struct Pane: View {}\nfinal class Store {}\n", "swift"), ["Pane", "Store"]);
  assert.throws(() => declaredTypes("class Example", "java"), /Unsupported native module audit language/);
  assert.equal(identifierUsedOutside("Profile", "A.swift", [["A.swift","struct Profile"], ["B.swift","Profile()"]]), true);
  assert.equal(identifierUsedOutside("Profile", "A.swift", [["A.swift","struct Profile"], ["B.swift","Profiles()"]]), false);
  assert.deepEqual(candidateModules([["App.swift","@main\nstruct App {}"]], "swift"), []);
  assert.deepEqual(candidateModules([["Extension.swift","extension String { var sanitized: String { self } }"]], "swift"), []);
});

test("maintained platform entrypoints are checked and symlink recursion is blocked", () => temporary((root, put) => {
  put("ghostftp-desktop/src/main.tsx", "export {}");
  put("ghostftp-desktop/src-tauri/src/lib.rs", "pub fn work() {}");
  put("macos/Sources/App.swift", "@main\nstruct App {}");
  put("tools/ghostftp-installer/main.go", "package main\nfunc main() {}");
  put("tools/ghostftp-runtime/main.go", "package main\nfunc main() {}");
  put("docs/build/STATUS.md", "documentation is a source file");
  assert.deepEqual(audit(root).entrypointBlockers, []);
  assert.equal(audit(root).counts.go, 2);
  fs.symlinkSync(root, path.join(root, "macos", "loop"), "dir");
  assert.equal(walk(root).length, 6);
  fs.rmSync(path.join(root, "macos/Sources/App.swift"));
  assert.deepEqual(audit(root).entrypointBlockers, ["macOS @main source missing"]);
}));

test("source symbol audit is advisory across Swift, Rust, TypeScript and Go", () => {
  assert.deepEqual(privateSymbolCandidates([["macos/View.swift","private func orphan() {}"]], "swift").map(x=>x.name), ["orphan"]);
  assert.deepEqual(privateSymbolCandidates([["src/engine.rs","fn orphan_worker() {}"]], "rust").map(x=>x.name), ["orphan_worker"]);
  assert.deepEqual(privateSymbolCandidates([["src/ui.ts","const orphanArrow = () => 1;"]], "typescript").map(x=>x.name), ["orphanArrow"]);
  assert.deepEqual(privateSymbolCandidates([["tools/main.go","func unusedWorker() {}"]], "go").map(x=>x.name), ["unusedWorker"]);
  assert.deepEqual(privateSymbolCandidates([["tools/main.go","func main() {}\nfunc init() {}"]], "go"), []);
  assert.throws(() => privateSymbolCandidates([], "java"), /Unknown symbol audit language/);
});

test("Rust production and test module scopes remain distinct", () => {
  const rust = `fn before() {}
#[cfg(test)]
mod tests {
  #[test]
  fn unitTest() { let json = "{not-a-module-brace}"; }
  fn helperOnlyForTests() {}
  // irrelevant closing brace: }
}
fn after() {}
#[tokio::test]
async fn asyncTest() {}
fn last() {}`;
  assert.equal(rustTestModuleRanges(rust).length, 1);
  assert.deepEqual(privateSymbolCandidates([["src/lib.rs", rust]], "rust").map(x=>x.name), ["before", "after", "last"]);
});

test("whole-project inventory sees desktop scripts and excludes generated files", () => temporary((root, put) => {
  for (const [name, content] of [
    ["ghostftp-desktop/src/main.tsx", "function unusedUi() {}"],
    ["ghostftp-desktop/src-tauri/src/lib.rs", "fn unusedRust() {}"],
    ["ghostftp-desktop/src-tauri/tests/protocol_e2e.rs", "#[test]\nfn realTest() {}"],
    ["macos/Sources/App.swift", "@main\nstruct App {}\nprivate func unusedSwift() {}"],
    ["tools/ghostftp-installer/main.go", "package main\nfunc main() {}\nfunc unusedInstaller() {}"],
    ["tools/ghostftp-runtime/main.go", "package main\nfunc main() {}"],
    ["ghostftp-desktop/target/generated.rs", "generated"],
  ]) put(name, content);
  const report = audit(root);
  assert.deepEqual(report.entrypointBlockers, []);
  assert.ok(report.advisoryPrivateSymbols.desktopRust.some(x=>x.name==="unusedRust"));
  assert.ok(!report.advisoryPrivateSymbols.desktopRust.some(x=>x.name==="realTest"));
  assert.ok("macosSwift" in report.advisoryPrivateSymbols);
  assert.ok("desktopTypeScript" in report.advisoryPrivateSymbols);
  assert.ok("installerRuntimeGo" in report.advisoryPrivateSymbols);
  assert.equal(report.counts.rust, 2);
}));

test("obsolete mock server automation remains prohibited", () => {
  const scriptsDir = path.resolve("ghostftp-desktop/scripts");
  const old = walk(scriptsDir).filter(p => /\.(?:mjs|js)$/.test(p) && /npm[^\n]*dev:mock/.test(fs.readFileSync(p, "utf8")));
  assert.deepEqual(old, []);
});
