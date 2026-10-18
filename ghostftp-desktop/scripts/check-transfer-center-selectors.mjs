#!/usr/bin/env node
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

// Exercise the actual production TypeScript selectors without a second copy.
const source = readFileSync(new URL("../src/lib/transferCenterSelectors.ts", import.meta.url), "utf8");
const code = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  reportDiagnostics: true,
});
assert.equal(code.diagnostics?.length ?? 0, 0, "selector transpilation must be clean");
const module = { exports: {} };
runInNewContext(code.outputText, { module, exports: module.exports, Number }, { timeout: 2000 });
const { transferBucket, summarizeTransfers, filterTransfers } = module.exports;

const make = (id, status, startedAt = 100_000, extras = {}) => ({
  id, kind: "upload", source: `/local/${id}.php`,
  destination: `/remote/${id}.php`, size: 100, transferred: 0,
  status, startedAt, ...extras,
});
const records = [
  make("running", "transferring"),
  make("queued", "queued"),
  make("paused", "paused"),
  make("complete", "done"),
  make("skipped", "skipped"),
  make("failed", "error", 100_000, { error: "server timed out" }),
  make("canceled", "canceled"),
];
const ids = (rows) => Array.from(rows, (row) => row.id).join(",");
const defaults = {
  tab: "all", query: "", direction: "all", time: "all", nowSeconds: 100_000,
};

assert.equal(transferBucket("canceled"), "other", "cancellation is not failure");
assert.equal(transferBucket("paused"), "active", "paused remains in queue");
assert.equal(JSON.stringify(summarizeTransfers(records)),
  JSON.stringify({ total: 7, active: 3, completed: 2, failed: 1 }));
assert.equal(ids(filterTransfers(records, { ...defaults, tab: "active" })), "running,queued,paused");
assert.equal(ids(filterTransfers(records, { ...defaults, tab: "completed" })), "complete,skipped");
assert.equal(ids(filterTransfers(records, { ...defaults, tab: "failed" })), "failed");
assert.equal(ids(filterTransfers(records, { ...defaults, query: "  SERVER TIMED  " })), "",
  "query searches transfer paths/status, not opaque error strings");
assert.equal(ids(filterTransfers(records, { ...defaults, query: "  /REMOTE/FAILED  " })), "failed");
assert.equal(ids(filterTransfers(records, { ...defaults, direction: "download" })), "");
assert.equal(ids(filterTransfers(records, { ...defaults, tab: "failed", direction: "upload" })), "failed");

// Inclusive boundaries are in seconds, not milliseconds.
const timed = [
  make("boundary", "queued", 96_400),
  make("just-before", "error", 96_399),
  make("today", "done", 99_999, { kind: "download" }),
  make("old", "error", 13_599),
  make("invalid", "error", NaN),
];
assert.equal(ids(filterTransfers(timed, { ...defaults, time: "hour" })), "boundary,today");
assert.equal(ids(filterTransfers(timed, { ...defaults, time: "day" })),
  "boundary,just-before,today", "day uses same captured time");
assert.equal(ids(filterTransfers(timed, { ...defaults, time: "hour", tab: "failed" })), "",
  "time + status filters compose and reject NaN");
assert.equal(ids(filterTransfers(timed, { ...defaults, direction: "download" })), "today");

// Repeated filtering does not mutate the native snapshots or reorder records.
const frozen = Object.freeze(records.map((r) => Object.freeze({ ...r })));
const result = filterTransfers(frozen, defaults);
assert.equal(ids(result), ids(frozen));
assert.equal(result[0], frozen[0]);
assert.equal(frozen[0].status, "transferring");
assert.equal(summarizeTransfers([]).total, 0);
console.log("Transfer selectors: status, filters, boundaries, invalid timestamps and identity OK.");
