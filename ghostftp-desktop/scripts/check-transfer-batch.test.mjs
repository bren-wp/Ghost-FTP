// Ghost FTP by Brendigo
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

// Run the real production batch/conflict implementation, not a reimplementation.
// Replace only its outer Zustand, Tauri and UI dependencies with deterministic fakes.
const source = readFileSync(new URL("../src/stores/transfersStore.ts", import.meta.url), "utf8");
const begin = source.indexOf("function actionToPolicy(");
const end = source.indexOf("function isActiveTransfer(", begin);
assert.ok(begin >= 0 && end > begin, "production runBatch implementation must remain present");

const environment = `
const harness = { existing: [], decisions: [], asks: [], warnings: [], errors: [], listCalls: 0, failList: false };
const settings = {
  promptOnOverwrite: true,
  overwritePolicy: "overwrite",
  writes: [],
  setOverwritePolicy(value) { this.overwritePolicy = value; this.writes.push(["policy", value]); },
  setPromptOnOverwrite(value) { this.promptOnOverwrite = value; this.writes.push(["prompt", value]); },
};
const useSettings = { getState: () => settings };
const useConflicts = { getState: () => ({
  ask: async info => {
    harness.asks.push(info);
    const decision = harness.decisions.shift();
    if (!decision) throw Error("unexpected extra conflict prompt");
    return decision;
  },
}) };
const ipc = { listDirectory: async () => {
  harness.listCalls++;
  if (harness.failList) throw Error("test destination list failure");
  return harness.existing;
} };
const toast = { warning: (...args) => harness.warnings.push(args) };
const toastError = (...args) => harness.errors.push(args);
const baseName = path => path.replaceAll(String.fromCharCode(92), "/").split("/").filter(Boolean).at(-1) || path;
export { harness, settings };
`;
const { outputText, diagnostics } = ts.transpileModule(
  environment + source.slice(begin, end) + "\nexport { runBatch };\n",
  {
    compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
    reportDiagnostics: true,
  },
);
assert.deepEqual(diagnostics ?? [], []);
const { runBatch, harness, settings } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);

const file = name => ({ path: "/source/" + name, kind: "file", size: 123 });
const choice = (action, remember = false, applyToAll = false) =>
  ({ action, remember, applyToAll });
function reset() {
  harness.existing = [];
  harness.decisions = [];
  harness.asks = [];
  harness.warnings = [];
  harness.errors = [];
  harness.listCalls = 0;
  harness.failList = false;
  settings.promptOnOverwrite = true;
  settings.overwritePolicy = "overwrite";
  settings.writes = [];
}
function capture() {
  const started = [];
  return { started, start: async (item, policy) => { started.push([item.path, policy]); } };
}

test("remember overwrite immediately suppresses extra prompts in the same batch", async () => {
  reset();
  harness.existing = [{name: "a.txt", size: 4}, {name: "b.txt", size: 8}];
  harness.decisions = [choice("overwrite", true, false)];
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt"), file("b.txt")], start);
  assert.deepEqual(started, [["/source/a.txt", "overwrite"], ["/source/b.txt", "overwrite"]]);
  assert.equal(harness.asks.length, 1, "remember must apply immediately even without apply-to-all");
  assert.equal(settings.promptOnOverwrite, false);
  assert.deepEqual(settings.writes, [["policy", "overwrite"], ["prompt", false]],
    "durable settings are only updated once for the newly remembered action");
});

test("remember skip suppresses all remaining name collisions but still queues new files", async () => {
  reset();
  harness.existing = [{name: "a.txt", size: 4}, {name: "b.txt", size: 8}];
  harness.decisions = [choice("skip", true, false)];
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt"), file("fresh.txt"), file("b.txt")], start);
  assert.deepEqual(started, [["/source/fresh.txt", "skip"]]);
  assert.equal(harness.asks.length, 1);
  assert.equal(settings.overwritePolicy, "skip");
  assert.equal(settings.promptOnOverwrite, false);
});

test("apply-to-all without remember is limited to this batch and does not change defaults", async () => {
  reset();
  harness.existing = [{name: "a.txt", size: 4}, {name: "b.txt", size: 8}];
  harness.decisions = [choice("rename", false, true)];
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt"), file("b.txt")], start);
  assert.deepEqual(started, [["/source/a.txt", "rename"], ["/source/b.txt", "rename"]]);
  assert.equal(harness.asks.length, 1);
  assert.equal(settings.promptOnOverwrite, true);
  assert.deepEqual(settings.writes, []);
});

test("cancel aborts without changing durable settings or starting transfers", async () => {
  reset();
  harness.existing = [{name: "a.txt", size: 4}];
  harness.decisions = [choice("cancel", false, false)];
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt"), file("new.txt")], start);
  assert.deepEqual(started, []);
  assert.deepEqual(settings.writes, []);
  assert.equal(harness.asks.length, 1);
});

test("failed destination listing is fail-closed and does not queue potentially destructive work", async () => {
  reset();
  harness.failList = true;
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt")], start);
  assert.deepEqual(started, []);
  assert.equal(harness.errors.length, 1);
  assert.equal(harness.asks.length, 0);
});

test("silent mode prevents duplicate basename enqueueing and skips destination queries", async () => {
  reset();
  settings.promptOnOverwrite = false;
  const { started, start } = capture();
  await runBatch("remote", "/", "remote", [file("a.txt"), file("a.txt"), file("b.txt")], start);
  assert.deepEqual(started, [["/source/a.txt", "overwrite"], ["/source/b.txt", "overwrite"]]);
  assert.equal(harness.warnings.length, 1);
  assert.equal(harness.listCalls, 0);
});
