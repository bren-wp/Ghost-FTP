import { createPortal } from "react-dom";
import { useEffect, useState } from "react";
import {
  HardDrive,
  X,
  RefreshCw,
  Ban,
  Loader2,
  ChevronRight,
  Folder,
  Palette,
  AlertCircle,
  Eye,
  Copy,
  Trash2,
} from "lucide-react";
import { useDiskScan, resolveNode } from "@/stores/diskScanStore";
import { useLayout } from "@/stores/layoutStore";
import { toast } from "@/stores/toastStore";
import type { DuNode, ScanStrategy } from "@/lib/types";
import { fmtSize } from "@/lib/format";
import {
  materialIconUrl,
  ContextMenu,
  ConfirmModal,
  type MenuItem,
} from "@ghostftp/file-ui";
import { cn } from "@/lib/cn";
import { DiskTreemap } from "./DiskTreemap";

/** Parent directory of a POSIX/Windows path (its own root when at top). */
function parentDir(p: string): string {
  // POSIX root is its own parent.
  if (p === "/") return "/";
  // A Windows drive root ("C:\" or "C:/") is its own parent — never slice it
  // down to a bare "C:" that no backend can list.
  const driveRoot = p.match(/^[a-zA-Z]:[\\/]/)?.[0];
  const i = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  if (driveRoot && i === 2) return driveRoot;
  if (i <= 0) return p.startsWith("/") ? "/" : p;
  return p.slice(0, i);
}

const STRATEGY_LABEL: Record<ScanStrategy, string> = {
  generic: "Walk",
  shell: "Shell du",
  objectFlat: "Object listing",
};

/** Mount point: renders the overlay only while the explorer is open. */
export function DiskUsageHost() {
  const open = useDiskScan((s) => s.open);
  if (!open) return null;
  return <DiskUsage />;
}

function DiskUsage() {
  const {
    root,
    state,
    strategy,
    dirsScanned,
    filesFound,
    totalBytes,
    error,
    note,
    tree,
    crumbs,
    colorMode,
  } = useDiskScan();
  const rescan = useDiskScan((s) => s.rescan);
  const cancel = useDiskScan((s) => s.cancel);
  const close = useDiskScan((s) => s.close);
  const drillTo = useDiskScan((s) => s.drillTo);
  const navigateTo = useDiskScan((s) => s.navigateTo);
  const setColorMode = useDiskScan((s) => s.setColorMode);
  const removeNode = useDiskScan((s) => s.removeNode);
  const sessionId = useDiskScan((s) => s.sessionId);
  const requestReveal = useLayout((s) => s.requestReveal);

  const [hover, setHover] = useState<DuNode | null>(null);
  const [menu, setMenu] = useState<{ x: number; y: number; items: MenuItem[] } | null>(
    null
  );
  const [confirmDel, setConfirmDel] = useState<DuNode | null>(null);

  const reveal = (node: DuNode) => {
    if (!sessionId) return;
    const dir = node.kind === "directory" ? node.path : parentDir(node.path);
    requestReveal(sessionId, dir);
    close();
  };

  const copyPath = (node: DuNode) => {
    void navigator.clipboard.writeText(node.path)
      .then(() => toast.info("Path copied", node.path))
      .catch((error) => toast.error("Couldn't copy path", String(error)));
  };

  const openMenu = (node: DuNode, x: number, y: number) => {
    const isRoot = node.path === (tree?.path ?? "");
    const items: MenuItem[] = [
      {
        label:
          node.kind === "directory"
            ? "Open in file browser"
            : "Reveal in file browser",
        icon: <Eye size={12} />,
        onClick: () => reveal(node),
      },
      {
        label: "Copy path",
        icon: <Copy size={12} />,
        onClick: () => copyPath(node),
        separatorAfter: !isRoot,
      },
    ];
    if (!isRoot) {
      items.push({
        label: "Delete",
        icon: <Trash2 size={12} />,
        destructive: true,
        onClick: () => setConfirmDel(node),
      });
    }
    setMenu({ x, y, items });
  };

  // Esc closes the explorer (matches every other overlay).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);

  const current = resolveNode(tree, crumbs);
  const scanning = state === "scanning";

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <HardDrive size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Disk Usage</span>

        {/* Breadcrumb into the tree */}
        <div className="flex min-w-0 flex-1 items-center overflow-x-auto whitespace-nowrap text-xs [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <button
            onClick={() => navigateTo(-1)}
            className={cn(
              "max-w-[20rem] truncate rounded px-1 py-0.5 font-mono",
              crumbs.length === 0
                ? "font-medium text-text"
                : "text-text-muted hover:bg-bg-hover hover:text-text"
            )}
            title={root}
          >
            {root || "/"}
          </button>
          {crumbs.map((seg, i) => (
            <span key={i} className="flex items-center">
              <ChevronRight size={12} className="shrink-0 text-text-dim" />
              <button
                onClick={() => navigateTo(i)}
                className={cn(
                  "max-w-[14rem] truncate rounded px-1 py-0.5 font-mono",
                  i === crumbs.length - 1
                    ? "font-medium text-text"
                    : "text-text-muted hover:bg-bg-hover hover:text-text"
                )}
              >
                {seg}
              </button>
            </span>
          ))}
        </div>

        {/* Strategy badge (+ fallback note) */}
        <span
          className="shrink-0 rounded-sm bg-bg-subtle px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wider text-text-dim"
          title={note ? note : `Scan strategy: ${STRATEGY_LABEL[strategy]}`}
        >
          {STRATEGY_LABEL[strategy]}
        </span>
        {note && (
          <span
            className="hidden shrink-0 truncate text-[10px] text-text-dim lg:inline"
            title={note}
          >
            {note}
          </span>
        )}

        <button
          onClick={() =>
            setColorMode(colorMode === "type" ? "depth" : "type")
          }
          className="flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
          title="Toggle treemap colouring (by file type / by depth)"
        >
          <Palette size={12} />
          {colorMode === "type" ? "By type" : "By depth"}
        </button>

        {scanning ? (
          <button
            onClick={() => cancel()}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
            title="Stop the scan"
          >
            <Ban size={12} /> Cancel
          </button>
        ) : (
          <button
            onClick={() => rescan()}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
            title="Re-scan this folder"
          >
            <RefreshCw size={12} /> Rescan
          </button>
        )}
        <button
          onClick={() => close()}
          className="shrink-0 rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-danger"
          title="Close (Esc)"
        >
          <X size={16} />
        </button>
      </div>

      {/* Progress banner while scanning */}
      {scanning && (
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1.5 text-xs text-text-muted">
          <Loader2 size={13} className="animate-spin text-accent" />
          <span>
            Scanning… {dirsScanned.toLocaleString()} folders,{" "}
            {filesFound.toLocaleString()} files ·{" "}
            <span className="font-medium text-text">{fmtSize(totalBytes)}</span>
          </span>
        </div>
      )}

      {/* Body */}
      {error ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-md flex-col items-center gap-2 text-center">
            <AlertCircle size={22} className="text-danger" />
            <div className="text-sm font-medium">Scan failed</div>
            <div className="text-xs text-text-dim">{error}</div>
            <button
              onClick={() => rescan()}
              className="mt-1 rounded-md border border-border px-3 py-1 text-xs hover:bg-bg-hover"
            >
              Try again
            </button>
          </div>
        </div>
      ) : (
        <div className="flex min-h-0 flex-1">
          {/* Treemap */}
          <div className="relative flex min-w-0 flex-1 flex-col border-r border-border">
            <div className="min-h-0 flex-1 p-2">
              {current && !scanning ? (
                <DiskTreemap
                  root={current}
                  colorMode={colorMode}
                  onDrill={(n) => drillTo(n)}
                  onHover={setHover}
                  onContext={openMenu}
                />
              ) : (
                <div className="flex h-full items-center justify-center text-xs text-text-dim">
                  {scanning ? "Building the map…" : "No data"}
                </div>
              )}
            </div>
            {/* Hover readout */}
            <div className="flex h-6 shrink-0 items-center gap-2 border-t border-border bg-bg-panel px-3 text-[11px] text-text-dim">
              {hover ? (
                <>
                  <span className="min-w-0 flex-1 truncate font-mono" title={hover.path}>
                    {hover.path}
                  </span>
                  <span className="shrink-0 font-medium text-text">
                    {fmtSize(hover.size)}
                  </span>
                </>
              ) : current ? (
                <span className="truncate">
                  {current.name} · {fmtSize(current.size)} ·{" "}
                  {(current.children?.length ?? 0).toLocaleString()} items — click a box to drill in
                </span>
              ) : null}
            </div>
          </div>

          {/* Size-ranked list */}
          <RankedList
            node={current}
            onDrill={(n) => drillTo(n)}
            onContext={openMenu}
          />
        </div>
      )}

      {menu && (
        <ContextMenu
          x={menu.x}
          y={menu.y}
          items={menu.items}
          onClose={() => setMenu(null)}
        />
      )}
      {confirmDel && (
        <ConfirmModal
          title={`Delete ${confirmDel.name}?`}
          message={
            `Path: ${confirmDel.path}` +
            (confirmDel.kind === "directory"
              ? `\nThis folder and everything under it (${fmtSize(
                  confirmDel.size
                )}) will be permanently removed.`
              : `\n${fmtSize(confirmDel.size)} will be permanently removed.`)
          }
          destructive
          confirmLabel="Delete"
          onClose={() => setConfirmDel(null)}
          onConfirm={() => {
            const n = confirmDel;
            setConfirmDel(null);
            void removeNode(n);
          }}
        />
      )}
    </div>,
    document.body
  );
}

const MAX_ROWS = 500;

function RankedList({
  node,
  onDrill,
  onContext,
}: {
  node: DuNode | null;
  onDrill: (n: DuNode) => void;
  onContext: (node: DuNode, x: number, y: number) => void;
}) {
  const children = node?.children ?? [];
  const parentSize = node?.size ?? 0;
  const shown = children.slice(0, MAX_ROWS);

  return (
    <div className="flex w-80 shrink-0 flex-col bg-bg-panel">
      <div className="flex shrink-0 items-center justify-between border-b border-border bg-bg-subtle px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
        <span>Biggest items</span>
        <span>{children.length.toLocaleString()}</span>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">
        {children.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-text-dim">
            Nothing here.
          </div>
        ) : (
          shown.map((c) => {
            const pct = parentSize > 0 ? (c.size / parentSize) * 100 : 0;
            const isDir = c.kind === "directory";
            const matUrl =
              c.kind === "file" ? materialIconUrl(c.name) : undefined;
            return (
              <button
                key={c.path}
                onClick={() => isDir && onDrill(c)}
                onContextMenu={(e) => {
                  e.preventDefault();
                  onContext(c, e.clientX, e.clientY);
                }}
                title={c.path}
                className={cn(
                  "flex w-full items-center gap-2 border-b border-border-subtle px-3 py-1.5 text-left text-xs",
                  isDir
                    ? "cursor-pointer hover:bg-bg-hover"
                    : "cursor-default"
                )}
              >
                {matUrl ? (
                  <img
                    src={matUrl}
                    width={14}
                    height={14}
                    alt=""
                    draggable={false}
                    className="shrink-0"
                  />
                ) : (
                  <Folder size={14} className="shrink-0 text-accent" />
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="min-w-0 truncate">{c.name}</span>
                    <span className="shrink-0 tabular-nums text-text-dim">
                      {fmtSize(c.size)}
                    </span>
                  </div>
                  {/* Percentage-of-parent bar */}
                  <div className="mt-1 h-1 w-full overflow-hidden rounded-full bg-bg-subtle">
                    <div
                      className="h-full rounded-full bg-accent"
                      style={{ width: `${Math.max(1, pct)}%` }}
                    />
                  </div>
                </div>
                <span className="w-10 shrink-0 text-right tabular-nums text-[10px] text-text-dim">
                  {pct >= 0.1 ? `${pct.toFixed(0)}%` : "<1%"}
                </span>
              </button>
            );
          })
        )}
        {children.length > MAX_ROWS && (
          <div className="px-3 py-2 text-center text-[10px] text-text-dim">
            +{(children.length - MAX_ROWS).toLocaleString()} more not shown
          </div>
        )}
      </div>
    </div>
  );
}
