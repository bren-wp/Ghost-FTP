// Execute the TypeScript shortcut implementation with simulated OS keyboard events.
// This runs in the mandatory npm run check:ui workflow, not only a static grep.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

const source = readFileSync(new URL("../src/lib/shortcuts.ts", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.equal((compiled.diagnostics ?? []).filter((d) => d.category === ts.DiagnosticCategory.Error).length, 0);

function platform(platformName, userAgent = platformName) {
  const exports = {};
  runInNewContext(compiled.outputText, { exports, navigator: { platform: platformName, userAgent } }, { timeout: 1000 });
  return exports;
}
function key(keyName, keys = {}) {
  return { key: keyName, ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, ...keys };
}

const mac = platform("MacIntel", "Macintosh");
assert.equal(mac.keyCombo(key("t", { metaKey: true })), "mod+t");
assert.equal(mac.keyCombo(key("t", { ctrlKey: true })), "ctrl+t");
assert.equal(mac.keyCombo(key("r", { ctrlKey: true })), "ctrl+r");
assert.notEqual(mac.keyCombo(key("k", { ctrlKey: true })), "mod+k");
assert.notEqual(mac.keyCombo(key("r", { ctrlKey: true })), "mod+r");
assert.equal(mac.keyCombo(key("d", { ctrlKey: true, shiftKey: true })), "ctrl+shift+d");
assert.equal(mac.keyCombo(key("d", { metaKey: true, shiftKey: true })), "mod+shift+d");
assert.equal(mac.keyCombo(key("d", { ctrlKey: true, metaKey: true, shiftKey: true })), "mod+ctrl+shift+d");
assert.equal(mac.formatCombo("mod+k"), "⌘ K");

const windows = platform("Win32", "Windows NT");
assert.equal(windows.keyCombo(key("k", { ctrlKey: true })), "mod+k");
assert.equal(windows.keyCombo(key("k", { metaKey: true })), "meta+k");
assert.equal(windows.keyCombo(key("K", { ctrlKey: true, shiftKey: true })), "mod+shift+k");

const linux = platform("Linux x86_64", "Linux");
assert.equal(linux.keyCombo(key("k", { ctrlKey: true })), "mod+k");
assert.equal(linux.keyCombo(key("k", { metaKey: true })), "meta+k");
assert.equal(linux.keyCombo(key(" ", { ctrlKey: true, altKey: true })), "mod+alt+space");
console.log("macOS Command/Control and Windows/Linux shortcut regression tests passed.");
