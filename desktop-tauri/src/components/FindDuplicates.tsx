import { createPortal } from "react-dom";
import { useEffect, useState } from "react";
import {
  CopyX,
  X,
  Ban,
  Loader2,
  RefreshCw,
  AlertCircle,
  Eye,
  Copy,
  Hash,
  Play,
  Trash2,
} from "lucide-react";
import { useDedupe } from "@/stores/dedupeStore";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { toast } from "@/stores/toastStore";
import { ConfirmModal } from "./ConfirmModal";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import type { SessionId } from "@/lib/types";
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

/** Mount point: renders the overlay only while a scan is open. */
export function FindDuplicatesHost() {
  const open = useDedupe((s) => s.open);
  if (!open) return null;
  return <FindDuplicates />;
}

function FindDuplicates() {
  const sessionId = useDedupe((s) => s.sessionId);
  const path = useDedupe((s) => s.path);
  const hash = useDedupe((s) => s.hash);
  const state = useDedupe((s) => s.state);
  const phase = useDedupe((s) => s.phase);
  const filesFound = useDedupe((s) => s.filesFound);
  const hashed = useDedupe((s) => s.hashed);
  const error = useDedupe((s) => s.error);
  const result = useDedupe((s) => s.result);
  const excluded = useDedupe((s) => s.excluded);
  const deleted = useDedupe((s) => s.deleted);
  const deleting = useDedupe((s) => s.deleting);
  const setPath = useDedupe((s) => s.setPath);
  const setHash = useDedupe((s) => s.setHash);
  const run = useDedupe((s) => s.run);
  const cancel = useDedupe((s) => s.cancel);
  const close = useDedupe((s) => s.close);
  const toggleExcluded = useDedupe((s) => s.toggleExcluded);
  const selectedPaths = useDedupe((s) => s.selectedPaths);
  const deleteSelected = useDedupe((s) => s.deleteSelected);

  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);
  const requestReveal = useLayout((s) => s.requestReveal);

  const [confirming, setConfirming] = useState(false);

  const name =
    sessionId === LOCAL_SESSION
      ? "Local filesystem"
      : (profiles.find(
          (p) => p.id === sessions.find((s) => s.sessionId === sessionId)?.profileId
        )?.name ?? sessionId);

  // Esc closes (matches every other overlay).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);

  const scanning = state === "scanning";
  const selected = selectedPaths();
  const selectedBytes = result
    ? result.groups.reduce(
        (acc, g) =>
          acc +
          g.files.filter(
            (f, i) => i !== g.keep && selected.includes(f.path)
          ).length *
            g.size,
        0
      )
    : 0;

  const reveal = (sessionId: SessionId, p: string) => {
    requestReveal(sessionId, parentDir(p));
    close();
  };

  const copyPath = (p: string) => {
    navigator.clipboard.writeText(p);
    toast.info("Path copied", p);
  };

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <CopyX size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Find Duplicates</span>

        <div className="flex min-w-0 flex-1 items-center gap-2 text-xs">
          <span className="shrink-0 rounded bg-bg-subtle px-2 py-1 text-text-dim">
            {name}
          </span>
          <input
            value={path}
            onChange={(e) => setPath(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && path.trim() !== "" && !scanning) void run();
            }}
            disabled={scanning}
            spellCheck={false}
            placeholder="/path/to/scan"
            className="min-w-0 flex-1 rounded border border-border bg-bg-subtle px-2 py-1 font-mono text-xs text-text focus:border-accent focus:outline-none disabled:opacity-40"
          />
        </div>

        <button
          onClick={() => setHash(!hash)}
          disabled={scanning}
          className={cn(
            "flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-[11px] disabled:opacity-40",
            hash
              ? "border-accent bg-accent/10 text-accent"
              : "border-border text-text-muted hover:bg-bg-hover hover:text-text"
          )}
          title="Group by content hash (sha256) instead of name+size. Catches duplicates with unrelated names — slower, it reads every same-size file."
        >
          <Hash size={12} /> Hash
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
            onClick={() => run()}
            disabled={path.trim() === ""}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Scan for duplicates"
          >
            {result ? <RefreshCw size={12} /> : <Play size={12} />}
            {result ? "Rescan" : "Scan"}
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
      {scanning && (
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1.5 text-xs text-text-muted">
          <Loader2 size={13} className="animate-spin text-accent" />
          <span>
            {phase === "hashing"
              ? `Hashing files… ${hashed.toLocaleString()} done`
              : `Scanning… ${filesFound.toLocaleString()} files found`}
          </span>
        </div>
      )}

      {/* Summary / action bar */}
      {result && (
        <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-border bg-bg-panel px-3 py-1.5 text-[11px] text-text-muted">
          <span>
            <span className="font-semibold text-text">{result.summary.groups}</span>{" "}
            groups ·{" "}
            <span className="font-semibold text-text">
              {result.summary.duplicateFiles}
            </span>{" "}
            duplicates ·{" "}
            <span className="font-semibold text-amber-400">
              {fmtSize(result.summary.wastedBytes)}
            </span>{" "}
            reclaimable
          </span>
          <span className="text-text-dim">
            {result.summary.filesScanned.toLocaleString()} files scanned
            {result.mode === "hash" ? " · hashed" : " · name match"}
            {result.summary.hashErrors > 0
              ? ` · ${result.summary.hashErrors} hash errors`
              : ""}
          </span>
          <button
            onClick={() => setConfirming(true)}
            disabled={selected.length === 0 || deleting}
            className="ml-auto flex items-center gap-1 rounded-md border border-danger/40 px-2 py-1 text-[11px] text-danger hover:bg-danger/10 disabled:opacity-40"
            title="Delete the checked duplicates (keeps one file per group)"
          >
            {deleting ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Trash2 size={12} />
            )}
            Delete {selected.length} selected
            {selected.length > 0 ? ` (${fmtSize(selectedBytes)})` : ""}
          </button>
        </div>
      )}

      {/* Body */}
      {error ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-md flex-col items-center gap-2 text-center">
            <AlertCircle size={22} className="text-danger" />
            <div className="text-sm font-medium">Scan failed</div>
            <div className="text-xs text-text-dim">{error}</div>
          </div>
        </div>
      ) : !result && !scanning ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-sm flex-col items-center gap-2 text-center text-text-dim">
            <CopyX size={26} />
            <div className="text-sm">
              Scan this folder for duplicate files — the{" "}
              <span className="font-mono">name_1.ext</span> copies uploads leave
              behind, or exact content duplicates with Hash on.
            </div>
          </div>
        </div>
      ) : result && result.groups.length === 0 ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-sm flex-col items-center gap-2 text-center text-text-dim">
            <CopyX size={26} />
            <div className="text-sm">No duplicates found — this tree is clean.</div>
            {!hash && (
              <div className="text-xs">
                Tip: rescan with <span className="font-mono">Hash</span> to catch
                duplicates with different names.
              </div>
            )}
          </div>
        </div>
      ) : result ? (
        <div className="min-h-0 flex-1 overflow-y-auto">
          {result.groups.slice(0, MAX_GROUPS).map((g) => (
            <div key={g.key} className="border-b border-border-subtle">
              {/* Group header */}
              <div className="flex items-center gap-2 bg-bg-subtle px-3 py-1 text-[11px] text-text-muted">
                <span className="min-w-0 flex-1 truncate font-mono" title={g.key}>
                  {g.key}
                </span>
                <span className="shrink-0">
                  {g.files.length} files · {fmtSize(g.size)} each ·{" "}
                  <span className="text-amber-400">
                    {fmtSize(g.size * (g.files.length - 1))} wasted
                  </span>
                </span>
              </div>
              {/* Group files */}
              {g.files.map((f, i) => {
                const isKeep = i === g.keep;
                const isDeleted = deleted.has(f.path);
                const checked = !isKeep && !excluded.has(f.path) && !isDeleted;
                return (
                  <div
                    key={f.path}
                    className={cn(
                      "group flex items-center gap-2 px-3 py-1 text-xs hover:bg-bg-hover",
                      isDeleted && "opacity-40"
                    )}
                    title={f.path}
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={isKeep || isDeleted || deleting}
                      onChange={() => toggleExcluded(f.path)}
                      className="h-3 w-3 shrink-0 accent-[var(--color-accent)]"
                      title={
                        isKeep
                          ? "Suggested keeper — kept"
                          : checked
                            ? "Will be deleted"
                            : "Spared from deletion"
                      }
                    />
                    <span
                      className={cn(
                        "min-w-0 flex-1 truncate font-mono",
                        isDeleted && "line-through"
                      )}
                    >
                      {f.path}
                    </span>
                    {isKeep && (
                      <span className="shrink-0 rounded-sm bg-emerald-400/10 px-1 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-emerald-400">
                        keep
                      </span>
                    )}
                    {isDeleted && (
                      <span className="shrink-0 rounded-sm bg-danger/10 px-1 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-danger">
                        deleted
                      </span>
                    )}
                    <span className="w-20 shrink-0 text-right tabular-nums text-text-dim">
                      {fmtSize(f.size)}
                    </span>
                    <div className="flex w-12 shrink-0 items-center justify-end gap-1 opacity-0 group-hover:opacity-100">
                      <button
                        onClick={() => reveal(sessionId, f.path)}
                        className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                        title="Reveal in browser"
                      >
                        <Eye size={12} />
                      </button>
                      <button
                        onClick={() => copyPath(f.path)}
                        className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                        title="Copy path"
                      >
                        <Copy size={12} />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
          {result.groups.length > MAX_GROUPS && (
            <div className="px-3 py-2 text-center text-[10px] text-text-dim">
              +{(result.groups.length - MAX_GROUPS).toLocaleString()} more groups
              not shown — clean these first, then rescan
            </div>
          )}
        </div>
      ) : null}

      {confirming && (
        <ConfirmModal
          title={`Delete ${selected.length} duplicate${selected.length === 1 ? "" : "s"}?`}
          message={`This permanently deletes ${selected.length} file${selected.length === 1 ? "" : "s"} (${fmtSize(selectedBytes)}) from ${name}, keeping one copy per group. This can't be undone.`}
          destructive
          confirmLabel="Delete"
          onClose={() => setConfirming(false)}
          onConfirm={() => {
            setConfirming(false);
            void deleteSelected();
          }}
        />
      )}
    </div>,
    document.body
  );
}

const MAX_GROUPS = 500;
