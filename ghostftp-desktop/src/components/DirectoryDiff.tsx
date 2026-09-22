import { createPortal } from "react-dom";
import { useEffect, useMemo } from "react";
import {
  GitCompareArrows,
  X,
  Ban,
  Loader2,
  RefreshCw,
  ArrowLeftRight,
  ArrowLeft,
  ArrowRight,
  AlertCircle,
  Eye,
  Copy,
  Hash,
  Play,
} from "lucide-react";
import { useDiff } from "@/stores/diffStore";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { toast } from "@/stores/toastStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import { ipc } from "@/lib/ipc";
import type { DiffClass, DiffEntry, SessionId } from "@/lib/types";
import { fmtSize } from "@/lib/format";
import { cn } from "@/lib/cn";

/** Parent directory of a POSIX/Windows path (its own root at the top). */
function parentDir(p: string): string {
  if (p === "/") return "/";
  const driveRoot = p.match(/^[a-zA-Z]:[\\/]/)?.[0];
  const i = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  if (driveRoot && i === 2) return driveRoot;
  if (i <= 0) return p.startsWith("/") ? "/" : p;
  return p.slice(0, i);
}

/** Join a scan root and a POSIX relative path, keeping the root's separator. */
function joinPath(root: string, rel: string): string {
  const sep = root.includes("\\") && !root.includes("/") ? "\\" : "/";
  return root.endsWith("/") || root.endsWith("\\") ? root + rel : root + sep + rel;
}

/** Per-class visual treatment: a label, an accent colour, and a dot. */
const CLASS_META: Record<
  DiffClass,
  { label: string; text: string; dot: string }
> = {
  onlyInA: { label: "Only in A", text: "text-emerald-400", dot: "bg-emerald-400" },
  onlyInB: { label: "Only in B", text: "text-sky-400", dot: "bg-sky-400" },
  different: { label: "Differs", text: "text-amber-400", dot: "bg-amber-400" },
  same: { label: "Same", text: "text-text-dim", dot: "bg-text-dim" },
};

/** Mount point: renders the overlay only while a diff is open. */
export function DirectoryDiffHost() {
  const open = useDiff((s) => s.open);
  if (!open) return null;
  return <DirectoryDiff />;
}

function DirectoryDiff() {
  const sideA = useDiff((s) => s.sideA);
  const sideB = useDiff((s) => s.sideB);
  const hash = useDiff((s) => s.hash);
  const state = useDiff((s) => s.state);
  const phase = useDiff((s) => s.phase);
  const filesA = useDiff((s) => s.filesA);
  const filesB = useDiff((s) => s.filesB);
  const error = useDiff((s) => s.error);
  const result = useDiff((s) => s.result);
  const filters = useDiff((s) => s.filters);
  const setSideB = useDiff((s) => s.setSideB);
  const setHash = useDiff((s) => s.setHash);
  const run = useDiff((s) => s.run);
  const cancel = useDiff((s) => s.cancel);
  const close = useDiff((s) => s.close);
  const toggleFilter = useDiff((s) => s.toggleFilter);

  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);
  const requestReveal = useLayout((s) => s.requestReveal);

  // Connection options for the side pickers: local + every live session.
  const options = useMemo(() => {
    const nameOf = (profileId: string) =>
      profiles.find((p) => p.id === profileId)?.name ?? "connection";
    return [
      { id: LOCAL_SESSION, name: "Local filesystem" },
      ...sessions.map((s) => ({ id: s.sessionId, name: nameOf(s.profileId) })),
    ];
  }, [sessions, profiles]);

  const nameFor = (sessionId: SessionId) =>
    options.find((o) => o.id === sessionId)?.name ?? sessionId;

  // Esc closes (matches every other overlay).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);

  const comparing = state === "comparing";
  const sidesReady =
    sideA.path.trim() !== "" &&
    sideB.path.trim() !== "" &&
    !(sideA.sessionId === sideB.sessionId && sideA.path === sideB.path);

  const reveal = (sessionId: SessionId, path: string, isDir: boolean) => {
    requestReveal(sessionId, isDir ? path : parentDir(path));
    close();
  };

  const copyPath = (path: string) => {
    void navigator.clipboard.writeText(path)
      .then(() => toast.info("Path copied", path))
      .catch((error) => toast.error("Couldn't copy path", String(error)));
  };

  // Copy-across reuses the local↔remote transfer engine, so it needs exactly
  // one local side (remote↔remote transfers aren't supported by the engine).
  const copySupported =
    (sideA.sessionId === LOCAL_SESSION) !== (sideB.sessionId === LOCAL_SESSION);

  const copyAcross = async (e: DiffEntry, dir: "a2b" | "b2a") => {
    const srcPath = dir === "a2b" ? e.aPath : e.bPath;
    if (!srcPath) return;
    const srcSession = dir === "a2b" ? sideA.sessionId : sideB.sessionId;
    const destSession = dir === "a2b" ? sideB.sessionId : sideA.sessionId;
    const destRoot = dir === "a2b" ? sideB.path : sideA.path;
    const destDir = parentDir(joinPath(destRoot, e.relative));
    try {
      // Best-effort: make sure a nested destination directory exists.
      if (e.relative.includes("/")) {
        await ipc.createDirectory(destSession, destDir).catch((error) => {
          console.warn("Couldn't pre-create destination directory; transfer will continue", error);
        });
      }
      if (srcSession === LOCAL_SESSION) {
        await ipc.startUpload(destSession, srcPath, destDir, "overwrite");
      } else {
        await ipc.startDownload(srcSession, srcPath, destDir, "overwrite");
      }
      toast.success(
        "Copy queued",
        `${e.relative} → ${dir === "a2b" ? nameFor(sideB.sessionId) : nameFor(sideA.sessionId)}`
      );
    } catch (err) {
      toast.error("Copy failed", String(err));
    }
  };

  const phaseLabel =
    phase === "walkingA"
      ? "Scanning side A…"
      : phase === "walkingB"
        ? "Scanning side B…"
        : "Hashing files…";

  const visible = (result?.entries ?? []).filter((e) => filters[e.class]);

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <GitCompareArrows size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Directory Diff</span>

        {/* Side A (fixed) ↔ Side B (editable picker) */}
        <div className="flex min-w-0 flex-1 items-center gap-2 text-xs">
          <span
            className="min-w-0 max-w-[22rem] truncate rounded bg-bg-subtle px-2 py-1 font-mono"
            title={`${nameFor(sideA.sessionId)}:${sideA.path}`}
          >
            <span className="text-text-dim">{nameFor(sideA.sessionId)}</span>
            <span className="text-text-dim">:</span>
            {sideA.path}
          </span>
          <ArrowLeftRight size={13} className="shrink-0 text-text-dim" />
          <select
            value={sideB.sessionId}
            onChange={(e) => setSideB({ sessionId: e.target.value })}
            disabled={comparing}
            className="shrink-0 rounded border border-border bg-bg-subtle px-1.5 py-1 text-xs text-text focus:border-accent focus:outline-none disabled:opacity-40"
            title="Side B connection"
          >
            {options.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
          <input
            value={sideB.path}
            onChange={(e) => setSideB({ path: e.target.value })}
            onKeyDown={(e) => {
              if (e.key === "Enter" && sidesReady && !comparing) void run();
            }}
            disabled={comparing}
            spellCheck={false}
            placeholder="/path/on/side/B"
            className="min-w-0 flex-1 rounded border border-border bg-bg-subtle px-2 py-1 font-mono text-xs text-text focus:border-accent focus:outline-none disabled:opacity-40"
          />
        </div>

        <button
          onClick={() => setHash(!hash)}
          disabled={comparing}
          className={cn(
            "flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-[11px] disabled:opacity-40",
            hash
              ? "border-accent bg-accent/10 text-accent"
              : "border-border text-text-muted hover:bg-bg-hover hover:text-text"
          )}
          title="Confirm same-size files by content hash (sha256). Slower — it reads every same-size file."
        >
          <Hash size={12} /> Hash
        </button>

        {comparing ? (
          <button
            onClick={() => cancel()}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
            title="Stop the comparison"
          >
            <Ban size={12} /> Cancel
          </button>
        ) : (
          <button
            onClick={() => run()}
            disabled={!sidesReady}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title={sidesReady ? "Run the comparison" : "Choose a different side B first"}
          >
            {result ? <RefreshCw size={12} /> : <Play size={12} />}
            {result ? "Rerun" : "Compare"}
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

      {/* Progress banner */}
      {comparing && (
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1.5 text-xs text-text-muted">
          <Loader2 size={13} className="animate-spin text-accent" />
          <span>
            {phaseLabel} A: {filesA.toLocaleString()} files · B:{" "}
            {filesB.toLocaleString()} files
          </span>
        </div>
      )}

      {/* Filter / summary bar */}
      {result && (
        <div className="flex shrink-0 flex-wrap items-center gap-1.5 border-b border-border bg-bg-panel px-3 py-1.5">
          {(Object.keys(CLASS_META) as DiffClass[]).map((cls) => {
            const count =
              cls === "onlyInA"
                ? result.summary.onlyInA
                : cls === "onlyInB"
                  ? result.summary.onlyInB
                  : cls === "different"
                    ? result.summary.different
                    : result.summary.same;
            const on = filters[cls];
            return (
              <button
                key={cls}
                onClick={() => toggleFilter(cls)}
                className={cn(
                  "flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px]",
                  on
                    ? "border-border bg-bg-subtle text-text"
                    : "border-transparent text-text-dim hover:bg-bg-hover"
                )}
                title={`${on ? "Hide" : "Show"} ${CLASS_META[cls].label}`}
              >
                <span className={cn("h-2 w-2 rounded-full", CLASS_META[cls].dot)} />
                {CLASS_META[cls].label}
                <span className="tabular-nums text-text-dim">{count}</span>
              </button>
            );
          })}
          <span className="ml-auto text-[11px] text-text-dim">
            {result.summary.total.toLocaleString()} paths
            {result.hashed ? " · hashed" : ""}
          </span>
        </div>
      )}

      {/* Body */}
      {error ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-md flex-col items-center gap-2 text-center">
            <AlertCircle size={22} className="text-danger" />
            <div className="text-sm font-medium">Comparison failed</div>
            <div className="text-xs text-text-dim">{error}</div>
          </div>
        </div>
      ) : !result && !comparing ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-sm flex-col items-center gap-2 text-center text-text-dim">
            <GitCompareArrows size={26} />
            <div className="text-sm">
              Pick a side B connection and path, then Compare.
            </div>
          </div>
        </div>
      ) : (
        <DiffRows
          entries={visible}
          onReveal={reveal}
          onCopyPath={copyPath}
          onCopyAcross={copyAcross}
          copySupported={copySupported}
          aName={nameFor(sideA.sessionId)}
          bName={nameFor(sideB.sessionId)}
          sessionA={sideA.sessionId}
          sessionB={sideB.sessionId}
        />
      )}
    </div>,
    document.body
  );
}

const MAX_ROWS = 2000;

function DiffRows({
  entries,
  onReveal,
  onCopyPath,
  onCopyAcross,
  copySupported,
  aName,
  bName,
  sessionA,
  sessionB,
}: {
  entries: DiffEntry[];
  onReveal: (sessionId: SessionId, path: string, isDir: boolean) => void;
  onCopyPath: (path: string) => void;
  onCopyAcross: (e: DiffEntry, dir: "a2b" | "b2a") => void;
  copySupported: boolean;
  aName: string;
  bName: string;
  sessionA: SessionId;
  sessionB: SessionId;
}) {
  const shown = entries.slice(0, MAX_ROWS);
  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      {/* Column header */}
      <div className="sticky top-0 z-sticky flex items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
        <span className="w-4 shrink-0" />
        <span className="min-w-0 flex-1">Path</span>
        <span className="w-24 shrink-0 text-right" title={aName}>
          A · {aName}
        </span>
        <span className="w-24 shrink-0 text-right" title={bName}>
          B · {bName}
        </span>
        <span className="w-24 shrink-0" />
      </div>

      {entries.length === 0 ? (
        <div className="px-3 py-8 text-center text-xs text-text-dim">
          No entries match the current filters.
        </div>
      ) : (
        shown.map((e) => {
          const meta = CLASS_META[e.class];
          return (
            <div
              key={e.relative}
              className="group flex items-center gap-2 border-b border-border-subtle px-3 py-1.5 text-xs hover:bg-bg-hover"
              title={e.relative}
            >
              <span
                className={cn("h-2 w-2 shrink-0 rounded-full", meta.dot)}
                title={meta.label}
              />
              <div className="min-w-0 flex-1">
                <span className="truncate font-mono">{e.relative}</span>
                {e.class === "different" && e.reason && (
                  <span className="ml-2 rounded-sm bg-bg-subtle px-1 py-0.5 text-[9px] uppercase tracking-wide text-amber-400">
                    {e.reason}
                  </span>
                )}
                {e.hashError && (
                  <span
                    className="ml-2 text-[10px] text-text-dim"
                    title={e.hashError}
                  >
                    hash n/a
                  </span>
                )}
              </div>
              <span
                className={cn(
                  "w-24 shrink-0 text-right tabular-nums",
                  e.aSize == null ? "text-text-dim" : meta.text
                )}
              >
                {e.aSize == null ? "—" : fmtSize(e.aSize)}
              </span>
              <span
                className={cn(
                  "w-24 shrink-0 text-right tabular-nums",
                  e.bSize == null ? "text-text-dim" : meta.text
                )}
              >
                {e.bSize == null ? "—" : fmtSize(e.bSize)}
              </span>
              {/* Row actions (copy across, reveal, copy path) */}
              <div className="flex w-24 shrink-0 items-center justify-end gap-1 opacity-0 group-hover:opacity-100">
                {e.aPath && (
                  <button
                    onClick={() => onCopyAcross(e, "a2b")}
                    disabled={!copySupported}
                    className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-30"
                    title={
                      copySupported
                        ? `Copy A → B (overwrite): ${e.relative}`
                        : "Copy across needs one local side (remote↔remote transfers aren't supported yet)"
                    }
                  >
                    <ArrowRight size={12} />
                  </button>
                )}
                {e.bPath && (
                  <button
                    onClick={() => onCopyAcross(e, "b2a")}
                    disabled={!copySupported}
                    className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-30"
                    title={
                      copySupported
                        ? `Copy B → A (overwrite): ${e.relative}`
                        : "Copy across needs one local side (remote↔remote transfers aren't supported yet)"
                    }
                  >
                    <ArrowLeft size={12} />
                  </button>
                )}
                {e.aPath && (
                  <button
                    onClick={() => onReveal(sessionA, e.aPath!, false)}
                    className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                    title="Reveal on side A"
                  >
                    <Eye size={12} />
                  </button>
                )}
                {e.bPath && (
                  <button
                    onClick={() => onReveal(sessionB, e.bPath!, false)}
                    className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                    title="Reveal on side B"
                  >
                    <Eye size={12} />
                  </button>
                )}
                <button
                  onClick={() => onCopyPath(e.aPath ?? e.bPath ?? e.relative)}
                  className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                  title="Copy path"
                >
                  <Copy size={12} />
                </button>
              </div>
            </div>
          );
        })
      )}
      {entries.length > MAX_ROWS && (
        <div className="px-3 py-2 text-center text-[10px] text-text-dim">
          +{(entries.length - MAX_ROWS).toLocaleString()} more not shown
        </div>
      )}
    </div>
  );
}
