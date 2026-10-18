import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const raw = readFileSync(new URL("../src/lib/transferViewModel.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(raw, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { countTransferStatuses, filterTransferView, recentTransferLog } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);

const epoch = 1_000_000;
const make = (id, status, kind = "upload", age = 10, source = "one.php", destination = "/server/one.php") =>
  ({ id, kind, status, source, destination, size: 100, transferred: 0, startedAt: epoch - age });
const transfers = [
  make("a", "transferring"),
  make("b", "queued", "download", 3600),
  make("c", "paused", "upload", 3601),
  make("d", "done", "download", 86399),
  make("e", "skipped"),
  make("f", "error", "download", 20, "/local/Important.PHP", "/remote/workspace"),
  make("g", "canceled"),
];
const opts = (patch = {}) => ({ tab: "all", query: "", direction: "all", time: "all", nowSeconds: epoch, ...patch });

test("counts status exactly once; skipped is completed and canceled is not", () => {
  assert.deepEqual(countTransferStatuses(transfers), { completed: 2, failed: 1, active: 3 });
  assert.deepEqual(countTransferStatuses([]), { completed: 0, failed: 0, active: 0 });
});

test("status tabs retain semantics and backend item identity", () => {
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "all" })).map(t => t.id), ["a", "b", "c", "d", "e", "f", "g"]);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "active" })).map(t => t.id), ["a", "b", "c"]);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "completed" })).map(t => t.id), ["d", "e"]);
  const failed = filterTransferView(transfers, opts({ tab: "failed" }));
  assert.deepEqual(failed.map(t => t.id), ["f"]);
  assert.strictEqual(failed[0], transfers[5], "retry controls must use original transfer identity");
});

test("direction/search filters narrow failures case insensitively", () => {
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "failed", direction: "upload" })), []);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "failed", direction: "download", query: "  IMPORTANT.php " })).map(t => t.id), ["f"]);
  assert.deepEqual(filterTransferView(transfers, opts({ query: "QUEUED" })).map(t => t.id), ["b"]);
  assert.deepEqual(filterTransferView(transfers, opts({ query: "nonexistent" })), []);
});

test("hour/day inclusive boundaries come from a single supplied clock", () => {
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "active", time: "hour" })).map(t => t.id), ["a", "b"]);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "active", time: "day" })).map(t => t.id), ["a", "b", "c"]);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "completed", time: "day" })).map(t => t.id), ["d", "e"]);
  assert.deepEqual(filterTransferView(transfers, opts({ tab: "failed", time: "hour" })).map(t => t.id), ["f"]);
});

test("selectors never mutate the queue; history is bounded and sorted as a copy", () => {
  const original = structuredClone(transfers);
  const log = recentTransferLog(transfers, epoch - 3600, 2);
  assert.deepEqual(log.map(t => t.id), ["a", "e"]);
  assert.deepEqual(recentTransferLog(transfers, epoch + 1), []);
  assert.deepEqual(recentTransferLog(transfers, 0, -1), []);
  assert.strictEqual(filterTransferView(transfers, opts())[0], transfers[0]);
  assert.deepEqual(transfers, original);
});

test("10k-entry synthetic history is stable and memory bounded", () => {
  const history = Array.from({ length: 10_000 }, (_, i) => make(
    String(i), i % 7 === 0 ? "error" : "done", "download", i, "file_" + i, "/remote/file_" + i
  ));
  Object.freeze(history);
  const filtered = filterTransferView(history, opts({ tab: "failed", time: "hour" }));
  assert.equal(filtered.length, Array.from({ length: 3601 }, (_, i) => i).filter(i => i % 7 === 0).length);
  assert.ok(filtered.every(t => t.status === "error" && t.startedAt >= epoch - 3600));
  assert.equal(recentTransferLog(history, 0).length, 12);
});
