// Exercise the production graph data math, not a hand-copied algorithm.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const source = readFileSync(new URL("../src/lib/visualizationMath.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { prepareBandwidthGraph, safeDiskPercent, rankDiskItems } =
  await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

test("no transfer samples means no fabricated zero-bandwidth chart", () => {
  const graph = prepareBandwidthGraph([]);
  assert.equal(graph.hasData, false);
  assert.equal(graph.count, 0);
  assert.equal(graph.latest, null);
  assert.equal(graph.peak, 0);
  assert.equal(graph.uploadPoints, "");
  assert.equal(graph.downloadPoints, "");
});

test("the first real measurement remains a visible horizontal line", () => {
  const graph = prepareBandwidthGraph([{ upload: 2048, download: 1024 }]);
  assert.equal(graph.hasData, true);
  assert.equal(graph.count, 1);
  assert.equal(graph.peak, 2048);
  assert.equal(graph.uploadPoints, "0.0,14.0 600.0,14.0");
  assert.equal(graph.downloadPoints, "0.0,60.0 600.0,60.0");
  assert.deepEqual(graph.average, { upload: 2048, download: 1024 });
  assert.equal(graph.lastUploadY, 14);
  assert.equal(graph.lastDownloadY, 60);
});

test("measured zero is data; empty history is not", () => {
  const graph = prepareBandwidthGraph([{ upload: 0, download: 0 }]);
  assert.equal(graph.hasData, true);
  assert.equal(graph.peak, 0);
  assert.deepEqual(graph.latest, { upload: 0, download: 0 });
  assert.equal(graph.uploadPoints, "0.0,106.0 600.0,106.0");
});

test("bad rates never produce invalid SVG geometry or mutate input", () => {
  const samples = [
    { upload: Number.NaN, download: Infinity },
    { upload: -3, download: -Infinity },
    { upload: 12.5, download: 50 },
  ];
  const before = samples.map(item => ({ ...item }));
  const graph = prepareBandwidthGraph(samples);
  assert.deepEqual(graph.latest, { upload: 12.5, download: 50 });
  assert.equal(graph.peak, 50);
  assert.equal(graph.count, 3);
  for (const path of [graph.uploadPoints, graph.downloadPoints]) {
    assert.ok(!/NaN|Infinity|undefined/.test(path));
    for (const xy of path.split(" ")) {
      const [x, y] = xy.split(",").map(Number);
      assert.ok(Number.isFinite(x) && x >= 0 && x <= 600);
      assert.ok(Number.isFinite(y) && y >= 14 && y <= 106);
    }
  }
  assert.deepEqual(samples, before);
});

test("only 48 recent samples are shown; peak and mean use actual data", () => {
  const history = Array.from({ length: 150 }, (_, index) => ({ upload: index, download: index * 2 }));
  const graph = prepareBandwidthGraph(history);
  assert.equal(graph.count, 48);
  assert.deepEqual(graph.latest, { upload: 149, download: 298 });
  assert.equal(graph.peak, 298);
  assert.ok(Math.abs(graph.average.upload - 125.5) < 1e-9);
  assert.ok(Math.abs(graph.average.download - 251) < 1e-9);
  assert.equal(graph.uploadPoints.split(" ").length, 48);
  assert.equal(history.length, 150);
});

test("disk usage percentages clamp malformed and oversized scans", () => {
  assert.equal(safeDiskPercent(0, 100), 0);
  assert.equal(safeDiskPercent(1, 200), 0.5);
  assert.equal(safeDiskPercent(25, 100), 25);
  assert.equal(safeDiskPercent(200, 100), 100);
  for (const invalid of [NaN, Infinity, -Infinity, -1]) {
    assert.equal(safeDiskPercent(invalid, 100), 0);
    assert.equal(safeDiskPercent(10, invalid), 0);
  }
  assert.equal(safeDiskPercent(10, 0), 0);
});

test("disk rank is descending and stable without editing the directory tree", () => {
  const children = [
    { name: "alpha", size: 8 },
    { name: "beta", size: 40 },
    { name: "gamma", size: 40 },
    { name: "unknown", size: Number.NaN },
    { name: "empty", size: 0 },
  ];
  const original = [...children];
  assert.deepEqual(rankDiskItems(children).map(x => x.name),
    ["beta", "gamma", "alpha", "empty", "unknown"]);
  assert.deepEqual(children, original);
  assert.strictEqual(rankDiskItems(children)[0], children[1]);
  assert.notStrictEqual(rankDiskItems(children), children);
});

test("finite extreme bandwidth readings keep averages finite and bounded", () => {
  const max = Number.MAX_VALUE;
  const graph = prepareBandwidthGraph(Array.from({ length: 48 }, (_, i) => ({
    upload: max,
    download: i % 2 === 0 ? max : 0,
  })));
  assert.equal(graph.average.upload, max);
  assert.ok(Number.isFinite(graph.average.download));
  assert.ok(Math.abs(graph.average.download / max - 0.5) < 1e-12);
  assert.equal(graph.peak, max);
  assert.ok(!/Infinity|NaN/.test(graph.uploadPoints + graph.downloadPoints));
});

test("identical nonzero subnormal readings retain their measured average", () => {
  const rate = Number.MIN_VALUE;
  const graph = prepareBandwidthGraph(Array.from({ length: 48 }, () => ({
    upload: rate, download: rate,
  })));
  assert.equal(graph.average.upload, rate);
  assert.equal(graph.average.download, rate);
});
