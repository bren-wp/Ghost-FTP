// Ghost FTP by Brendigo
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

// Test the real production formatting/ETA implementation without loading its
// React rendering dependencies. TypeScript removes Transfer's type annotation.
const source = readFileSync(new URL("../src/components/TransferCenterParts.tsx", import.meta.url), "utf8");
const offset = source.indexOf("export function transferProgressPercent(");
assert.ok(offset >= 0, "transfer display utilities must remain testable");
const { outputText, diagnostics } = ts.transpileModule(source.slice(offset), {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { transferProgressPercent, etaOf, formatRate, formatBytes, baseName } =
  await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

test("zero-byte files and finite binary units are displayed precisely", () => {
  assert.equal(formatBytes(0), "0 B");
  assert.equal(formatBytes(1023), "1023 B");
  assert.equal(formatBytes(1024), "1.0 KB");
  assert.equal(formatBytes(1536), "1.5 KB");
  assert.equal(formatRate(1024), "1.0 KB/s");
  assert.equal(formatRate(0), "—");
});

test("negative and non-finite counters are never displayed as throughput", () => {
  for (const bad of [NaN, Infinity, -Infinity, -1]) {
    assert.equal(formatBytes(bad), "—");
    assert.equal(formatRate(bad), "—");
  }
  assert.equal(formatRate(1), "1 B/s");
});

test("progress is always finite and within 0–100 percent", () => {
  assert.equal(transferProgressPercent({ size: 100, transferred: 25 }), 25);
  assert.equal(transferProgressPercent({ size: 100, transferred: 999 }), 100);
  assert.equal(transferProgressPercent({ size: 0, transferred: 0 }), 0);
  assert.equal(transferProgressPercent({ size: 0, transferred: 0, status: "done" }), 100,
    "a confirmed empty-file transfer must display 100% completion");
  assert.equal(transferProgressPercent({ size: 0, transferred: 0, status: "error" }), 0);
  assert.equal(transferProgressPercent({ size: 0, transferred: 0, status: "paused" }), 0);
  assert.equal(transferProgressPercent({ size: NaN, transferred: 20 }), 0);
  assert.equal(transferProgressPercent({ size: 100, transferred: Infinity }), 0);
  assert.equal(transferProgressPercent({ size: 100, transferred: -3 }), 0);
});

test("ETA rejects invalid speeds and malformed counters", () => {
  const transfer = { size: 1000, transferred: 0, status: "transferring" };
  assert.equal(etaOf(transfer, 100), "00:10");
  assert.equal(etaOf({ ...transfer, transferred: 500 }, 100), "00:05");
  for (const speed of [0, -1, Infinity, NaN, -Infinity]) {
    assert.equal(etaOf(transfer, speed), "—");
  }
  assert.equal(etaOf({ ...transfer, size: NaN }, 100), "—");
  assert.equal(etaOf({ ...transfer, transferred: -1 }, 100), "—");
  assert.equal(etaOf({ ...transfer, transferred: Infinity }, 100), "—");
  assert.equal(etaOf({ ...transfer, status: "done" }, 100), "—");
});

test("paths preserve both Windows and POSIX basename behavior", () => {
  assert.equal(baseName("C:\\tmp\\source.txt"), "source.txt");
  assert.equal(baseName("/tmp/source.txt"), "source.txt");
  assert.equal(baseName(""), "transfer");
});

test("bandwidth graph preserves real measurements and uses clear empty state", () => {
  const begin = source.indexOf("export function BandwidthChart(");
  const end = source.indexOf("export function transferProgressPercent(", begin);
  assert.ok(begin >= 0 && end > begin);
  const chart = source.slice(begin, end);
  assert.ok(chart.includes("prepareBandwidthGraph(history)"));
  assert.ok(chart.includes("graph.hasData"));
  assert.ok(chart.includes("No bandwidth measurements yet."));
  assert.ok(chart.includes("graph.uploadPoints"));
  assert.ok(chart.includes("graph.downloadPoints"));
  assert.ok(chart.includes("graph.average.upload"));
  assert.ok(chart.includes("aria-label={description}"));
  assert.ok(!chart.includes("[{ upload: 0, download: 0 }, { upload: 0, download: 0 }]"),
    "missing readings must not render as fabricated zero throughput");
});

test("keyboard actions inside transfer rows are not stolen by row selection", () => {
  const begin = source.indexOf("export function TransferRow(");
  const end = source.indexOf("export function TransferDetails(", begin);
  assert.ok(begin >= 0 && end > begin);
  const row = source.slice(begin, end);
  assert.ok(row.includes("event.target !== event.currentTarget"),
    "ignore bubbled Enter and Space from native Pause/Retry/Cancel buttons");
  assert.ok(row.includes('event.key === "Enter" || event.key === " "'),
    "the row itself must remain keyboard-selectable");
  assert.ok(row.includes("event.stopPropagation(); onPauseResume();"));
  assert.ok(row.includes("event.stopPropagation(); onRetry();"));
  assert.ok(row.includes("event.stopPropagation(); onCancel();"));
  assert.ok(row.includes('title={transfer.source} dir="auto"'),
    "full file path must remain inspectable for truncated names");
});


test("detail progress and bandwidth chart stay accessible in compact desktop panels", () => {
  const detailStart = source.indexOf("export function TransferDetails(");
  const chartStart = source.indexOf("export function BandwidthChart(", detailStart);
  assert.ok(detailStart > 0 && chartStart > detailStart);
  const detail = source.slice(detailStart, chartStart);
  assert.ok(detail.includes('role="progressbar"'));
  assert.ok(detail.includes('aria-valuenow={Number(pct.toFixed(0))}'));
  const chart = source.slice(chartStart, source.indexOf("export function transferProgressPercent(", chartStart));
  assert.ok(chart.includes('className="relative h-24 w-full'),
    "bandwidth graph must stay within the fixed-height transfer summary panel");
  assert.ok(chart.includes("Average ↑"));
  assert.ok(chart.includes("graph.hasData"));
});

test("native transfer dialogs handle picker failures and reject changed sessions", () => {
  const dialog = readFileSync(new URL("../src/components/TransferCenterDialog.tsx", import.meta.url), "utf8");
  const exporting = dialog.slice(dialog.indexOf("const exportTransferHistory"), dialog.indexOf("const transfers ="));
  assert.ok(exporting.indexOf("try {") < exporting.indexOf("await save("));
  assert.ok(exporting.includes("toastError(error,"));
  const adding = dialog.slice(dialog.indexOf("const addTransfer ="), dialog.indexOf("// These counters"));
  assert.ok(adding.indexOf("await open(") < adding.indexOf("useConnections.getState()"));
  assert.ok(adding.includes("currentConnection.activeSessionId !== activeSessionId"));
  assert.ok(adding.includes("currentConnection.activeProfileId !== activeProfileId"));
  assert.ok(adding.indexOf("currentConnection.activeSessionId !== activeSessionId") < adding.indexOf("await enqueueUploads("));
});
