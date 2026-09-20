import { useEffect, useRef, useState } from "react";
import type { DuNode } from "@/lib/types";
import type { ColorMode } from "@/stores/diskScanStore";
import { fmtSize } from "@/lib/format";

// A squarified-ish treemap on a Canvas. Thousands of rectangles stay smooth on
// canvas where SVG would choke. Layout is a recursive binary split — each node's
// children are partitioned into two groups of ~equal total size and the box is
// cut along its longer axis, which keeps tiles reasonably square without the
// full squarify bookkeeping. Nested directories draw as inset boxes (WinDirStat
// look); hover reports the deepest node under the cursor, click drills into the
// deepest directory there.

interface Cell {
  node: DuNode;
  x: number;
  y: number;
  w: number;
  h: number;
  depth: number;
}

const MAX_DEPTH = 6;
const PAD = 2;
const MIN_SIDE = 4;

interface SizedChild {
  node: DuNode;
  value: number;
}

/** Split children into two size-balanced groups (input is size-desc sorted). */
function partition(items: SizedChild[]): [SizedChild[], SizedChild[]] {
  const total = items.reduce((a, c) => a + c.value, 0);
  let acc = 0;
  let i = 0;
  while (i < items.length - 1 && acc + items[i].value < total / 2) {
    acc += items[i].value;
    i++;
  }
  // Keep at least one item in each group.
  if (i > items.length - 2) i = items.length - 2;
  return [items.slice(0, i + 1), items.slice(i + 1)];
}

function splitLayout(
  items: SizedChild[],
  x: number,
  y: number,
  w: number,
  h: number,
  out: { node: DuNode; x: number; y: number; w: number; h: number }[]
) {
  if (items.length === 0 || w <= 0 || h <= 0) return;
  if (items.length === 1) {
    out.push({ node: items[0].node, x, y, w, h });
    return;
  }
  const [a, b] = partition(items);
  const aVal = a.reduce((s, c) => s + c.value, 0);
  const total = aVal + b.reduce((s, c) => s + c.value, 0);
  if (total <= 0) return;
  const frac = aVal / total;
  if (w >= h) {
    const aw = w * frac;
    splitLayout(a, x, y, aw, h, out);
    splitLayout(b, x + aw, y, w - aw, h, out);
  } else {
    const ah = h * frac;
    splitLayout(a, x, y, w, ah, out);
    splitLayout(b, x, y + ah, w, h - ah, out);
  }
}

function buildCells(
  node: DuNode,
  x: number,
  y: number,
  w: number,
  h: number,
  depth: number,
  cells: Cell[]
) {
  cells.push({ node, x, y, w, h, depth });
  if (depth >= MAX_DEPTH) return;
  const kids = (node.children ?? []).filter((c) => c.size > 0);
  if (kids.length === 0) return;
  const ix = x + PAD;
  const iy = y + PAD;
  const iw = w - PAD * 2;
  const ih = h - PAD * 2;
  if (iw < MIN_SIDE * 2 || ih < MIN_SIDE * 2) return;
  const rects: { node: DuNode; x: number; y: number; w: number; h: number }[] = [];
  splitLayout(
    kids.map((k) => ({ node: k, value: k.size })),
    ix,
    iy,
    iw,
    ih,
    rects
  );
  for (const r of rects) {
    if (r.w < MIN_SIDE || r.h < MIN_SIDE) {
      cells.push({ node: r.node, x: r.x, y: r.y, w: r.w, h: r.h, depth: depth + 1 });
    } else {
      buildCells(r.node, r.x, r.y, r.w, r.h, depth + 1, cells);
    }
  }
}

// Distinct, pleasant hues for the type palette. Files hash their extension onto
// this ring; directories are drawn as translucent frames, not fills.
const TYPE_HUES = [210, 145, 35, 275, 0, 190, 55, 320, 95, 235, 15, 165];

function extHue(name: string): number {
  const dot = name.lastIndexOf(".");
  const ext = dot > 0 ? name.slice(dot + 1).toLowerCase() : name.toLowerCase();
  let h = 0;
  for (let i = 0; i < ext.length; i++) h = (h * 31 + ext.charCodeAt(i)) >>> 0;
  return TYPE_HUES[h % TYPE_HUES.length];
}

/** Read a Ghost FTP theme token (an "r g b" triple) as a canvas-ready `rgb(...)`. */
function rgbVar(name: string): string {
  const v = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  const [r, g, b] = v.split(/\s+/).map(Number);
  if ([r, g, b].some(Number.isNaN)) return "rgb(120,120,120)";
  return `rgb(${r},${g},${b})`;
}

/** Dark vs light per the active theme's background luminance (works across all
 *  the named `data-theme` palettes, not just "dark"/"light"). */
function isDarkTheme(): boolean {
  const v = getComputedStyle(document.documentElement)
    .getPropertyValue("--bg")
    .trim();
  const [r, g, b] = v.split(/\s+/).map(Number);
  if ([r, g, b].some(Number.isNaN)) return true;
  return 0.2126 * r + 0.7152 * g + 0.0722 * b < 128;
}

function cellFill(cell: Cell, colorMode: ColorMode, dark: boolean): string {
  if (colorMode === "depth") {
    const light = dark
      ? Math.min(60, 22 + cell.depth * 7)
      : Math.max(45, 82 - cell.depth * 7);
    return `hsl(210 45% ${light}%)`;
  }
  const hue = extHue(cell.node.name);
  const l = dark ? 52 : 62;
  return `hsl(${hue} 55% ${l}%)`;
}

export function DiskTreemap({
  root,
  colorMode,
  onDrill,
  onHover,
  onContext,
}: {
  root: DuNode;
  colorMode: ColorMode;
  /** Drill into a directory the user clicked. */
  onDrill: (node: DuNode) => void;
  /** The deepest node under the cursor (null when the cursor leaves). */
  onHover: (node: DuNode | null) => void;
  /** Right-click on the deepest node under the cursor. */
  onContext: (node: DuNode, x: number, y: number) => void;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 0, h: 0 });
  const cellsRef = useRef<Cell[]>([]);
  const [hoverPath, setHoverPath] = useState<string | null>(null);
  // Bumped when the app theme changes so the canvas repaints with fresh tokens.
  const [themeTick, setThemeTick] = useState(0);

  // Track the pixel box.
  useEffect(() => {
    const el = wrapRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      const r = entries[0].contentRect;
      setSize((prev) =>
        prev.w === Math.floor(r.width) && prev.h === Math.floor(r.height)
          ? prev
          : { w: Math.floor(r.width), h: Math.floor(r.height) }
      );
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Repaint on theme switches (the app toggles `data-theme` on <html>).
  useEffect(() => {
    const mo = new MutationObserver(() => setThemeTick((t) => t + 1));
    mo.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-theme"],
    });
    return () => mo.disconnect();
  }, []);

  // Recompute cells + repaint whenever the node, size, or palette change.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || size.w === 0 || size.h === 0) return;
    const dark = isDarkTheme();
    const dpr = window.devicePixelRatio || 1;
    canvas.width = size.w * dpr;
    canvas.height = size.h * dpr;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, size.w, size.h);

    const cells: Cell[] = [];
    if (root.size > 0 && (root.children?.length ?? 0) > 0) {
      buildCells(root, 0, 0, size.w, size.h, 0, cells);
    }
    cellsRef.current = cells;

    // Parents first (already ordered that way), so nested children paint on top.
    for (const cell of cells) {
      const isLeaf =
        cell.node.kind !== "directory" ||
        !cell.node.children ||
        cell.node.children.length === 0 ||
        cell.depth >= MAX_DEPTH;
      if (cell.depth === 0) {
        // The container itself: neutral backdrop from the theme.
        ctx.fillStyle = rgbVar("--bg-subtle");
        ctx.fillRect(cell.x, cell.y, cell.w, cell.h);
        continue;
      }
      if (isLeaf) {
        ctx.fillStyle = cellFill(cell, colorMode, dark);
        ctx.fillRect(cell.x, cell.y, cell.w, cell.h);
        // Hairline separator.
        ctx.strokeStyle = dark ? "rgba(0,0,0,0.35)" : "rgba(0,0,0,0.15)";
        ctx.lineWidth = 0.5;
        ctx.strokeRect(cell.x + 0.25, cell.y + 0.25, cell.w - 0.5, cell.h - 0.5);
      }
    }

    // Highlight the hovered cell.
    if (hoverPath) {
      const hc = cells.find((c) => c.node.path === hoverPath);
      if (hc) {
        ctx.strokeStyle = dark ? "#fff" : "#111";
        ctx.lineWidth = 1.5;
        ctx.strokeRect(hc.x + 0.75, hc.y + 0.75, hc.w - 1.5, hc.h - 1.5);
      }
    }
  }, [root, size, colorMode, hoverPath, themeTick]);

  const cellAt = (px: number, py: number, wantDir: boolean): Cell | null => {
    // Deepest cell (last in array among containers) wins.
    let best: Cell | null = null;
    for (const c of cellsRef.current) {
      if (c.depth === 0) continue;
      if (px < c.x || px > c.x + c.w || py < c.y || py > c.y + c.h) continue;
      if (wantDir && c.node.kind !== "directory") continue;
      if (!best || c.depth >= best.depth) best = c;
    }
    return best;
  };

  const onMove = (e: React.MouseEvent) => {
    const rect = canvasRef.current!.getBoundingClientRect();
    const cell = cellAt(e.clientX - rect.left, e.clientY - rect.top, false);
    const path = cell?.node.path ?? null;
    if (path !== hoverPath) {
      setHoverPath(path);
      onHover(cell?.node ?? null);
    }
  };

  const onLeave = () => {
    setHoverPath(null);
    onHover(null);
  };

  const onClick = (e: React.MouseEvent) => {
    const rect = canvasRef.current!.getBoundingClientRect();
    const cell = cellAt(e.clientX - rect.left, e.clientY - rect.top, true);
    if (cell) onDrill(cell.node);
  };

  const onContextMenu = (e: React.MouseEvent) => {
    const rect = canvasRef.current!.getBoundingClientRect();
    const cell = cellAt(e.clientX - rect.left, e.clientY - rect.top, false);
    if (cell) {
      e.preventDefault();
      onContext(cell.node, e.clientX, e.clientY);
    }
  };

  return (
    <div ref={wrapRef} className="relative h-full w-full overflow-hidden">
      <canvas
        ref={canvasRef}
        style={{ width: size.w, height: size.h }}
        className="block cursor-pointer"
        onMouseMove={onMove}
        onMouseLeave={onLeave}
        onClick={onClick}
        onContextMenu={onContextMenu}
        title={
          hoverPath
            ? undefined
            : "Click a box to drill into a folder"
        }
      />
      {root.size === 0 && (
        <div className="absolute inset-0 flex items-center justify-center text-xs text-text-dim">
          Nothing to show — this folder is empty ({fmtSize(0)}).
        </div>
      )}
    </div>
  );
}
