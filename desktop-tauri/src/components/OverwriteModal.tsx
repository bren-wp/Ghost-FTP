import { useId, useRef, useState } from "react";
import { FileWarning, FolderSync, Copy, SkipForward } from "lucide-react";
import { useDialog } from "@/hooks/useDialog";
import { useConflicts, type ConflictInfo } from "@/stores/conflictStore";
import { fmtSize, fmtMtime } from "@/lib/format";
import { cn } from "@/lib/cn";

// FileZilla-style overwrite prompt. Always mounted; renders whenever the
// conflict store has a pending question (the head of its queue). Escape / click
// outside = Cancel (the safe choice — it stops the whole transfer).
export function OverwriteDialogHost() {
  const pending = useConflicts((s) => s.queue[0]);
  const respond = useConflicts((s) => s.respond);
  if (!pending) return null;
  // Key on the resolver so each queued conflict gets a fresh dialog (resetting
  // the checkboxes) instead of reusing the previous one's component state.
  return (
    <OverwriteModal
      key={pending.remaining + pending.name + pending.destDir}
      info={pending}
      onResolve={respond}
    />
  );
}

function OverwriteModal({
  info,
  onResolve,
}: {
  info: ConflictInfo;
  onResolve: (d: {
    action: "overwrite" | "rename" | "skip" | "cancel";
    applyToAll: boolean;
    remember: boolean;
  }) => void;
}) {
  const [applyToAll, setApplyToAll] = useState(false);
  const [remember, setRemember] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  // Escape = Cancel (stops the transfer) — the non-destructive default.
  const cancel = () => onResolve({ action: "cancel", applyToAll: false, remember: false });
  useDialog(panelRef, { onClose: cancel });

  const pick = (action: "overwrite" | "rename" | "skip") =>
    onResolve({ action, applyToAll, remember });

  const where = info.side === "local" ? "on this computer" : "on the server";
  const hasMeta =
    info.existingSize != null ||
    info.sourceSize != null ||
    info.existingModified != null ||
    info.sourceModified != null;

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={cancel}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal w-[30rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-5 py-3.5">
          <FileWarning size={15} className="text-warning" />
          <span id={titleId} className="text-[15px] font-semibold tracking-tight">
            {info.kind === "directory" ? "Folder already exists" : "File already exists"}
          </span>
        </div>

        <div className="px-5 py-4">
          <div className="text-sm text-text-muted">
            A {info.kind === "directory" ? "folder" : "file"} named{" "}
            <span className="font-medium text-text">{info.name}</span> already
            exists {where} in:
          </div>
          <div
            className="mt-1 truncate font-mono text-xs text-text-dim"
            title={info.destDir}
          >
            {info.destDir}
          </div>

          {hasMeta && (
            <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
              <FileFacts
                label="Existing"
                size={info.existingSize}
                modified={info.existingModified}
              />
              <FileFacts
                label="New"
                size={info.sourceSize}
                modified={info.sourceModified}
                accent
              />
            </div>
          )}

          <div className="mt-4 space-y-2">
            <Check checked={applyToAll} onChange={setApplyToAll}>
              Apply to the rest of this transfer
              {info.remaining > 0 ? ` (${info.remaining} more)` : ""}
            </Check>
            <Check checked={remember} onChange={setRemember}>
              Remember my choice as the default
            </Check>
          </div>
        </div>

        <div className="flex items-center justify-between gap-2 border-t border-border px-5 py-3">
          <button
            onClick={cancel}
            className="rounded-md px-3 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
          >
            Cancel transfer
          </button>
          <div className="flex gap-2">
            <button
              onClick={() => pick("skip")}
              className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <SkipForward size={13} /> Skip
            </button>
            <button
              onClick={() => pick("rename")}
              className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <Copy size={13} /> Keep both
            </button>
            <button
              onClick={() => pick("overwrite")}
              className="btn-accent flex items-center gap-1 rounded-md px-3.5 py-1.5 text-sm font-medium text-white"
            >
              <FolderSync size={14} /> Overwrite
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function FileFacts({
  label,
  size,
  modified,
  accent,
}: {
  label: string;
  size?: number;
  modified?: number;
  accent?: boolean;
}) {
  return (
    <div
      className={cn(
        "rounded-md border px-2.5 py-1.5",
        accent ? "border-accent/30 bg-accent/5" : "border-border bg-bg-subtle"
      )}
    >
      <div className="text-[10px] font-semibold uppercase tracking-wider text-text-dim">
        {label}
      </div>
      <div className="tabular-nums text-text">
        {size != null ? fmtSize(size) : "—"}
      </div>
      {modified != null && (
        <div className="text-[11px] text-text-dim">{fmtMtime(modified)}</div>
      )}
    </div>
  );
}

function Check({
  checked,
  onChange,
  children,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  children: React.ReactNode;
}) {
  return (
    <label className="flex cursor-pointer select-none items-center gap-2 text-sm text-text-muted">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-3.5 w-3.5 accent-[rgb(var(--accent))]"
      />
      {children}
    </label>
  );
}
