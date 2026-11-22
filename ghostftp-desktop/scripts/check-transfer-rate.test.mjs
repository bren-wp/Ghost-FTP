// Ghost FTP by Brendigo
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

// Execute the production sampling implementation, not a reimplemented test
// double. Its external Zustand dependencies are deliberately excluded.
const source = readFileSync(new URL("../src/stores/transfersStore.ts", import.meta.url), "utf8");
const start = source.indexOf("const transferRateBaselines = new Map");
const end = source.indexOf("export const useTransfers = create<TransfersState>", start);
assert.ok(start >= 0 && end > start, "transfer sampling code must remain testable");
const extracted = source.slice(start, end) +
  "\nexport { rateFromEvent, transferRateBaselines };\n";
const { outputText, diagnostics } = ts.transpileModule(extracted, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { rateFromEvent, transferRateBaselines, liveTransferRate } =
  await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

const transfer = (id, transferred, status = "transferring") =>
  ({ id, transferred, status });
const originalNow = Date.now;
let clock = 0;
Date.now = () => clock;

test.after(() => { Date.now = originalNow; });

test("bursty progress preserves a 40ms baseline and all transferred bytes", () => {
  transferRateBaselines.clear();
  clock = 1000;
  let sample = rateFromEvent("progress", transfer("fast", 0));
  for (let i = 1; i < 10; i++) {
    clock = 1000 + i * 4;
    sample = rateFromEvent("progress", transfer("fast", i * 100), sample);
    assert.equal(sample.bytesPerSecond, 0);
  }
  clock = 1040;
  sample = rateFromEvent("progress", transfer("fast", 1000), sample);
  assert.equal(sample.bytesPerSecond, 25_000,
    "all 1,000 bytes over the full 40ms must count, not just the last 100");
  assert.equal(liveTransferRate(transfer("fast", 1000), sample, clock), 25_000);
});

test("speed expires during stalled transfers even with more sub-40ms events", () => {
  transferRateBaselines.clear();
  clock = 2000;
  rateFromEvent("progress", transfer("stall", 0));
  clock = 2050;
  let sample = rateFromEvent("progress", transfer("stall", 1000));
  assert.equal(sample.bytesPerSecond, 20_000);
  clock = 2055;
  const interim = rateFromEvent("progress", transfer("stall", 1000), sample);
  assert.equal(interim.sampledAt, sample.sampledAt, "do not freshen an unchanged speed sample");
  assert.equal(liveTransferRate(transfer("stall", 1000), interim, 4600), 0);
  clock = 5000;
  sample = rateFromEvent("progress", transfer("stall", 1000), interim);
  assert.equal(sample.bytesPerSecond, 0, "stale EMA must not leak through new samples");
});

test("counter reset, clock rollback and terminal state cannot retain old speed", () => {
  transferRateBaselines.clear();
  clock = 10_000;
  rateFromEvent("progress", transfer("restart", 100));
  clock = 10_100;
  let sample = rateFromEvent("progress", transfer("restart", 500));
  assert.equal(sample.bytesPerSecond, 4000);
  clock = 10_110;
  sample = rateFromEvent("progress", transfer("restart", 10), sample);
  assert.equal(sample.bytesPerSecond, 0);
  clock = 10_105;
  sample = rateFromEvent("progress", transfer("restart", 20), sample);
  assert.equal(sample.bytesPerSecond, 0);
  assert.equal(liveTransferRate(transfer("restart", 20), sample, 10_104), 0);
  clock = 10_200;
  sample = rateFromEvent("done", transfer("restart", 20, "done"), sample);
  assert.equal(sample.bytesPerSecond, 0);
  assert.equal(transferRateBaselines.has("restart"), false);
});

test("invalid byte counters, invalid samples and non-finite clocks fail closed", () => {
  transferRateBaselines.clear();
  clock = 20_000;
  assert.equal(rateFromEvent("progress", transfer("bad", NaN)).bytesPerSecond, 0);
  clock = 20_050;
  assert.equal(rateFromEvent("progress", transfer("bad", Infinity)).bytesPerSecond, 0);
  assert.equal(liveTransferRate(transfer("bad", 0), { bytesPerSecond: Infinity, sampledAt: clock }, clock), 0);
  assert.equal(liveTransferRate(transfer("bad", 0), { bytesPerSecond: -1, sampledAt: clock }, clock), 0);
  assert.equal(liveTransferRate(transfer("bad", 0), { bytesPerSecond: 1, sampledAt: NaN }, clock), 0);
  assert.equal(liveTransferRate(transfer("bad", 0), { bytesPerSecond: 1, sampledAt: clock }, Infinity), 0);
  assert.equal(liveTransferRate(transfer("bad", 0, "paused"), { bytesPerSecond: 999, sampledAt: clock }, clock), 0);
});
