// Ghost FTP by Brendigo
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

// Exercise the production conflict store with a minimal deterministic Zustand
// adapter. Importing a duplicate/reimplemented conflict queue would miss bugs.
const original = readFileSync(new URL("../src/stores/conflictStore.ts", import.meta.url), "utf8");
assert.ok(original.includes('import { create } from "zustand";'));
const adapter = `
const create = initializer => {
  let state;
  const get = () => state;
  const set = updater => {
    const patch = typeof updater === "function" ? updater(state) : updater;
    state = { ...state, ...patch };
  };
  const store = selector => selector(state);
  state = initializer(set, get);
  store.getState = get;
  return store;
};
`;
const source = adapter + original.replace('import { create } from "zustand";', "");
const { outputText, diagnostics } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { useConflicts } = await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

const input = () => ({
  name: "identical.txt",
  destDir: "/remote",
  side: "remote",
  kind: "file",
  remaining: 2,
});
const choice = action => ({ action, applyToAll: false, remember: false });

test("identical queued prompts receive unique, monotonic IDs", async () => {
  const first = useConflicts.getState().ask(input());
  const second = useConflicts.getState().ask(input());
  const queue = useConflicts.getState().queue;
  assert.equal(queue.length, 2);
  assert.notEqual(queue[0].promptId, queue[1].promptId);
  assert.equal(queue[1].promptId, queue[0].promptId + 1);
  useConflicts.getState().respond(queue[0].promptId, choice("skip"));
  useConflicts.getState().respond(queue[1].promptId, choice("rename"));
  assert.equal((await first).action, "skip");
  assert.equal((await second).action, "rename");
  assert.equal(useConflicts.getState().queue.length, 0);
});

test("stale double-click, Escape or delayed handler cannot resolve the next prompt", async () => {
  const first = useConflicts.getState().ask(input());
  const second = useConflicts.getState().ask(input());
  const [a, b] = useConflicts.getState().queue;
  useConflicts.getState().respond(a.promptId, choice("overwrite"));
  useConflicts.getState().respond(a.promptId, choice("overwrite"));
  useConflicts.getState().respond(a.promptId, choice("cancel"));
  assert.equal((await first).action, "overwrite");
  assert.equal(useConflicts.getState().queue.length, 1);
  assert.equal(useConflicts.getState().queue[0].promptId, b.promptId);
  useConflicts.getState().respond(b.promptId, choice("skip"));
  assert.equal((await second).action, "skip");
  assert.equal(useConflicts.getState().queue.length, 0);
});

test("unknown IDs cannot mutate or dismiss a different pending dialog", async () => {
  const pending = useConflicts.getState().ask(input());
  const id = useConflicts.getState().queue[0].promptId;
  useConflicts.getState().respond(id + 999, choice("overwrite"));
  useConflicts.getState().respond(id - 1, choice("cancel"));
  assert.equal(useConflicts.getState().queue.length, 1);
  useConflicts.getState().respond(id, choice("cancel"));
  assert.equal((await pending).action, "cancel");
});

test("modal binds key and confirmation callback to the same prompt identity", () => {
  const ui = readFileSync(new URL("../src/components/OverwriteModal.tsx", import.meta.url), "utf8");
  assert.ok(ui.includes("key={pending.promptId}"), "identical prompts must reset checkbox state");
  assert.ok(ui.includes("respond(pending.promptId, decision)"), "stale callbacks must carry the source prompt ID");
  assert.ok(!ui.includes("key={pending.remaining + pending.name + pending.destDir}"));
});
