// Execute the production treemap layout and check keyboard access contracts.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const source = readFileSync(new URL("../src/components/DiskTreemap.tsx", import.meta.url), "utf8");
const usage = readFileSync(new URL("../src/components/DiskUsage.tsx", import.meta.url), "utf8");
const begin = source.indexOf("interface Cell {");
const end = source.indexOf("const TYPE_HUES", begin);
assert.ok(begin > 0 && end > begin, "actual layout must remain extractable for geometry tests");
const extracted = source.slice(begin, end) + "\nexport { buildCells };\n";
const { outputText, diagnostics } = ts.transpileModule(extracted, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { buildCells } = await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

const file = (name, size) =>
  ({ name, path: "/" + name, kind: "file", size });
const root = (children, size = 100) =>
  ({ name: "root", path: "/", kind: "directory", size, children });

test("valid child areas stay within the canvas and never produce NaN", () => {
  const cells = [];
  buildCells(root([
    file("big", 70), file("small", 30),
    file("nan", Number.NaN), file("infinity", Infinity), file("negative", -4), file("empty", 0),
  ]), 0, 0, 600, 240, 0, cells);
  assert.deepEqual(cells.map(c => c.node.name).sort(), ["big", "root", "small"]);
  for (const cell of cells) {
    assert.ok([cell.x, cell.y, cell.w, cell.h].every(Number.isFinite));
    assert.ok(cell.x >= 0 && cell.y >= 0 && cell.w > 0 && cell.h > 0);
    assert.ok(cell.x + cell.w <= 600 && cell.y + cell.h <= 240);
  }
});

test("near-floating-point maximum file sizes preserve finite partition geometry", () => {
  const cells = [];
  buildCells(root([file("a", 1e308), file("b", 1e308)], 1e308), 0, 0, 800, 400, 0, cells);
  assert.equal(cells.length, 3);
  for (const cell of cells) {
    assert.ok([cell.x, cell.y, cell.w, cell.h].every(Number.isFinite));
    assert.ok(cell.w > 0 && cell.h > 0);
  }
});

test("invalid canvas dimensions never leak unrenderable rectangles", () => {
  for (const [w, h] of [[NaN, 200], [Infinity, 200], [600, -1], [0, 200]]) {
    const cells = [];
    buildCells(root([file("file", 100)]), 0, 0, w, h, 0, cells);
    assert.deepEqual(cells, []);
  }
});

test("deeply nested folders honor a bounded recursive display depth", () => {
  let node = file("leaf", 50);
  for (let i = 0; i < 15; i++) {
    node = { name: "dir" + i, path: "/dir" + i, kind: "directory", size: 50, children: [node] };
  }
  const cells = [];
  buildCells(root([node], 50), 0, 0, 800, 500, 0, cells);
  assert.ok(cells.length > 2 && cells.length <= 7);
  assert.ok(cells.every(c => c.depth <= 6));
});

test("treemap is accessible from keyboard and supports correct high-DPI redraw", () => {
  for (const fragment of [
    "onKeyDown={onCanvasKeyDown}",
    "tabIndex={hasDrawableData ? 0 : -1}",
    'event.key === "ArrowRight"',
    'event.key === "Home"',
    'event.key === "End"',
    'event.key === "Enter"',
    'event.key === "ContextMenu"',
    'event.shiftKey && event.key === "F10"',
    "Math.round(size.w * dpr)",
    "Number.isFinite(child.size)",
    "window.addEventListener(\"resize\", update)",
  ]) assert.ok(source.includes(fragment), "Missing chart keyboard/DPI guard: " + fragment);
  assert.ok(usage.includes("rankDiskItems(children)"));
  assert.ok(usage.includes("safeDiskPercent(c.size, parentSize)"));
  assert.ok(usage.includes("onContext(c, box.left + 12, box.bottom)"));
  assert.ok(!usage.includes("Math.max(1, pct)"), "zero-byte items must not draw fake one-percent bars");
});
