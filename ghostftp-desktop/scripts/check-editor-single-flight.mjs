import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import ts from "typescript";

const src = readFileSync(new URL("../src/lib/singleFlightGate.ts", import.meta.url), "utf8");
const code = ts.transpileModule(src, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
}).outputText;
const { createSingleFlightGate } = await import(
  "data:text/javascript;base64," + Buffer.from(code).toString("base64")
);

const gate = createSingleFlightGate();
assert.equal(gate.tryAcquire(), true, "first click reserves the editor");
assert.equal(gate.tryAcquire(), false, "second click cannot start another native session");
gate.release();
assert.equal(gate.tryAcquire(), true, "failed IPC/listener attempts must be retryable");
assert.equal(gate.tryAcquire(), false);
gate.release();
assert.equal(gate.tryAcquire(), true, "closing and opening again remains possible");
gate.release();

const store = readFileSync(new URL("../src/stores/editorStore.ts", import.meta.url), "utf8");
assert.match(store, /if \(get\(\)\.builtinEditId \|\| !builtinLaunchGate\.tryAcquire\(\)\)/);
const begin = store.indexOf("startBuiltinEditing: async");
const end = store.indexOf("stopEditing: async", begin);
const launch = store.slice(begin, end);
assert.ok(launch.indexOf("builtinLaunchGate.tryAcquire()") < launch.indexOf("await get().ensureListeners()"),
  "acquire synchronously before listener and native IPC waits");
assert.ok(launch.includes("} finally {") && launch.includes("builtinLaunchGate.release();"),
  "always release after success, failure or cancellation");
assert.equal((launch.match(/ipc\.startEdit\(/g) ?? []).length, 1,
  "a launch attempt may create only one native edit session");
console.log("Built-in editor single-flight, retry and source integration verified.");
