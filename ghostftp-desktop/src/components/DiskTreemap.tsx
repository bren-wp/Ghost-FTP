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
  // Normalize values first: large valid sizes must not overflow sums.
  const scale = items.reduce((largest, item) => Math.max(largest, item.value), 0);
  const total = items.reduce((sum, item) => sum + item.value / scale, 0);
  let acc = 0;
  let i = 0;
  while (i < items.length - 1 && acc + items[i].value / scale < total / 2) {
    acc += items[i].value / scale;
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
  if (items.length === 0 || !Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) return;
  if (items.length === 1) {
    out.push({ node: items[0].node, x, y, w, h });
    return;
  }
  const [a, b] = partition(items);
  const scale = items.reduce((largest, item) => Math.max(largest, item.value), 0);
  const aVal = a.reduce((sum, item) => sum + item.value / scale, 0);
  const total = aVal + b.reduce((sum, item) => sum + item.value / scale, 0);
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
  const kids = (node.children ?? []).filter((c) => Number.isFinite(c.size) && c.size > 0);
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
  const [pixelRatio, setPixelRatio] = useState(1);
  const hasDrawableData =
    Number.isFinite(root.size) && root.size > 0 &&
    (root.children?.some((child) => Number.isFinite(child.size) && child.size > 0) ?? false);
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

  // Device pixel ratio can change when moving a window between monitors.
  // Cap it to avoid allocating enormous canvas bitmaps on ultra-high DPI.
  useEffect(() => {
    const update = () => {
      const device = window.devicePixelRatio;
      setPixelRatio(Number.isFinite(device) ? Math.min(2.5, Math.max(1, device)) : 1);
    };
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
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
    const dpr = pixelRatio;
    canvas.width = Math.max(1, Math.round(size.w * dpr));
    canvas.height = Math.max(1, Math.round(size.h * dpr));
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, size.w, size.h);

    const cells: Cell[] = [];
    if (hasDrawableData) {
      buildCells(root, 0, 0, size.w, size.h, 0, cells);
    }
    cellsRef.current = cells;

    // Parents first (already ordered that way), so nested children paint on top.
    for (const cell of cells) {
      const isLeaf =
        cell.node.kind !== "directory" ||
        !cell.node.children ||
        cell.node.children.length === 0 ||
        cell.depth >= MAX_DEPTH ||
        cell.w < MIN_SIDE * 2 ||
        cell.h < MIN_SIDE * 2;
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
        ctx.strokeRect(
          hc.x + 0.75,
          hc.y + 0.75,
          Math.max(0, hc.w - 1.5),
          Math.max(0, hc.h - 1.5),
        );
      }
    }
  }, [root, size, pixelRatio, hasDrawableData, colorMode, hoverPath, themeTick]);

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

  const onCanvasKeyDown = (event: React.KeyboardEvent<HTMLCanvasElement>) => {
    const folders = cellsRef.current.filter(
      (cell) => cell.depth > 0 && cell.node.kind === "directory" &&
        cell.w >= MIN_SIDE && cell.h >= MIN_SIDE,
    );
    if (folders.length === 0) return;
    const index = folders.findIndex((cell) => cell.node.path === hoverPath);
    const current = folders[index >= 0 ? index : 0];
    const forward = event.key === "ArrowRight" || event.key === "ArrowDown";
    const backward = event.key === "ArrowLeft" || event.key === "ArrowUp";
    if (forward || backward || event.key === "Home" || event.key === "End") {
      event.preventDefault();
      const nextIndex = event.key === "Home" ? 0 :
        event.key === "End" ? folders.length - 1 :
        index < 0 ? 0 :
        (index + (forward ? 1 : -1) + folders.length) % folders.length;
      const next = folders[nextIndex];
      setHoverPath(next.node.path);
      onHover(next.node);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onDrill(current.node);
    } else if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
      event.preventDefault();
      const rect = canvasRef.current?.getBoundingClientRect();
      if (rect) onContext(current.node, rect.left + current.x + 8, rect.top + current.y + 8);
    }
  };

  return (
    <div ref={wrapRef} className="relative h-full w-full overflow-hidden">
      <canvas
        ref={canvasRef}
        style={{ width: size.w, height: size.h }}
        className="block cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent"
        role="group"
        tabIndex={hasDrawableData ? 0 : -1}
        aria-label={"Disk usage map for " + root.name + ". Arrow keys select folders, Enter opens, Shift+F10 shows actions."}
        onKeyDown={onCanvasKeyDown}
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
      {!hasDrawableData && (
        <div className="absolute inset-0 flex items-center justify-center px-4 text-center text-xs text-text-dim" role="status">
          {root.size === 0
            ? "Nothing to show — this folder is empty (" + fmtSize(0) + ")."
            : "No measurable items available for this folder."}
        </div>
      )}
    </div>
  );
}
