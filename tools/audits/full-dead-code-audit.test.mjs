import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {
  walk, declaredTypes, identifierUsedOutside, candidateModules,
  resourceCandidates, audit,
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
    const before = audit(root);
    assert.deepEqual(before.entrypointBlockers, []);
    assert.equal(before.counts.go, 2);
    const fake = path.join(root, "macos", "loop");
    fs.symlinkSync(root, fake, "dir");
    assert.equal(walk(root).length, 7);
    fs.rmSync(path.join(root, "macos/Sources/App.swift"));
    assert.deepEqual(audit(root).entrypointBlockers, ["macOS @main source missing"]);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
