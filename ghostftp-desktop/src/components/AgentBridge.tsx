import { useEffect, useId, useRef } from "react";
import { Ban, Check, ShieldCheck, Terminal as TerminalIcon } from "lucide-react";
import { useBridge } from "@/stores/bridgeStore";
import { useDialog } from "@/hooks/useDialog";
import { toastError } from "@/lib/errors";
import { cn } from "@/lib/cn";
import type { BridgeApproval } from "@/lib/types";

// Phrase the approval prompt per operation kind.
const APPROVAL_COPY: Record<string, { title: string; foot: string }> = {
  exec: { title: "Agent wants to run a command", foot: "Runs over your authenticated SSH session." },
  read: { title: "Agent wants to read from the server", foot: "Reads through your authenticated Ghost FTP session." },
  download: { title: "Agent wants to download a file", foot: "Downloads through Ghost FTP's transfer engine." },
  upload: { title: "Agent wants to upload a file", foot: "Uploads through Ghost FTP's transfer engine." },
  upload_dir: { title: "Agent wants to upload a directory", foot: "Uploads the whole tree through Ghost FTP's transfer engine." },
  sync: { title: "Agent wants to sync directories", foot: "Copies through Ghost FTP's transfer engine; mirror mode deletes destination files missing from the source." },
  search: { title: "Agent wants to search the server", foot: "Searches through your authenticated Ghost FTP session." },
  skill: { title: "Agent wants to run a Skill across servers", foot: "Runs the skill's steps on every listed server through your authenticated Ghost FTP sessions." },
};
function approvalCopy(kind: string) {
  return (
    APPROVAL_COPY[kind] ?? {
      title: "Agent wants to run an operation",
      foot: "Runs through your authenticated Ghost FTP session.",
    }
  );
}

// Pull the load-bearing numbers out of a directory-upload / sync summary so
// the modal surfaces them as badges (risk obvious), not just prose. The full
// summary is always shown verbatim above; these only re-emphasize it, so a
// non-matching summary simply renders no badges.
function summaryStats(
  kind: string,
  command: string
): { label: string; danger?: boolean }[] {
  const stats: { label: string; danger?: boolean }[] = [];
  if (kind === "upload_dir") {
    const m = command.match(/\((\d+) files?, (.+?) total, overwrite: (yes|no)\)/);
    if (m) {
      stats.push({ label: `${m[1]} files` }, { label: m[2] });
      if (m[3] === "yes")
        stats.push({ label: "overwrites existing files", danger: true });
    }
  } else if (kind === "sync") {
    if (/dry run — no changes/.test(command)) {
      stats.push({ label: "dry run — nothing changes" });
      return stats;
    }
    const copy = command.match(/copy (\d+) files? \((.+?)\)/);
    if (copy) stats.push({ label: `copies ${copy[1]} files` }, { label: copy[2] });
    const del = command.match(/delete (\d+) files? on the destination \(mirror\)/);
    if (del)
      stats.push({ label: `DELETES ${del[1]} files on the destination`, danger: true });
  } else if (kind === "skill") {
    // "Run skill "X" on N server(s) (…) — M step(s) each"
    const servers = command.match(/on (\d+) servers? \(/);
    if (servers)
      stats.push({ label: `${servers[1]} server${servers[1] === "1" ? "" : "s"}`, danger: true });
    const steps = command.match(/— (\d+) steps? each/);
    if (steps) stats.push({ label: `${steps[1]} step${steps[1] === "1" ? "" : "s"} each` });
  }
  return stats;
}

// Always-mounted: boots the bridge store (status + event listeners) and renders
// the per-command approval modal whenever a request is pending.
export function AgentBridgeHost() {
  const approvals = useBridge((s) => s.approvals);
  const respond = useBridge((s) => s.respond);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    useBridge
      .getState()
      .init()
      .then((c) => {
        if (cancelled) c();
        else cleanup = c;
      })
      .catch((error) => {
        toastError(error, "Couldn't initialize Agent Bridge");
      });
    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  const pending = approvals[0];
  if (!pending) return null;
  return (
    <ApprovalModal
      approval={pending}
      count={approvals.length}
      onApprove={() => respond(pending.requestId, "approve")}
      onDeny={() => respond(pending.requestId, "deny")}
    />
  );
}

function ApprovalModal({
  approval,
  count,
  onApprove,
  onDeny,
}: {
  approval: BridgeApproval;
  count: number;
  onApprove: () => void;
  onDeny: () => void;
}) {
  const copy = approvalCopy(approval.kind);
  const stats = summaryStats(approval.kind, approval.command);
  const allowSudo = useBridge((s) => s.status.policy.allowSudo);
  const sudoNote =
    allowSudo && approval.kind === "exec" && /\bsudo\b/.test(approval.command);
  const panelRef = useRef<HTMLDivElement>(null);
  const denyRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  // Escape denies (the safe choice); Deny takes initial focus so a stray
  // keypress never auto-approves an agent command.
  useDialog(panelRef, { onClose: onDeny, initialFocus: denyRef });
  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[28rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-5 py-3.5">
          <ShieldCheck size={15} className="text-accent" />
          <span id={titleId} className="text-[15px] font-semibold tracking-tight">
            {copy.title}
          </span>
        </div>
        <div className="px-5 py-4">
          <div className="mb-2 text-xs text-text-muted">
            On <span className="font-medium text-text">{approval.sessionName}</span>{" "}
            via the Agent Bridge:
          </div>
          <pre className="selectable max-h-40 max-w-full overflow-auto whitespace-pre-wrap [overflow-wrap:anywhere] rounded-md border border-border bg-bg-subtle px-3 py-2 font-mono text-xs text-text">
            {approval.command}
          </pre>
          {stats.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1.5">
              {stats.map((s) => (
                <span
                  key={s.label}
                  className={cn(
                    "rounded-md border px-2 py-0.5 text-[11px] font-medium",
                    s.danger
                      ? "border-danger/40 bg-danger/10 text-danger"
                      : "border-border bg-bg-subtle text-text-muted"
                  )}
                >
                  {s.label}
                </span>
              ))}
            </div>
          )}
          <div className="mt-3 flex items-center gap-1.5 text-[11px] text-text-dim">
            <TerminalIcon size={11} /> {copy.foot}
          </div>
          {sudoNote && (
            <div className="mt-2 flex items-center gap-1.5 text-[11px] text-accent">
              <ShieldCheck size={11} /> Ghost FTP will answer sudo's prompt with this
              connection's password.
            </div>
          )}
        </div>
        <div className="flex items-center justify-between gap-2 border-t border-border px-5 py-3">
          <span className="text-[11px] text-text-dim">
            {count > 1 ? `${count - 1} more queued` : ""}
          </span>
          <div className="flex gap-2">
            <button
              ref={denyRef}
              onClick={onDeny}
              className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <Ban size={13} /> Deny
            </button>
            <button
              onClick={onApprove}
              className="btn-accent flex items-center gap-1 rounded-md px-3.5 py-1.5 text-sm font-medium text-white"
            >
              <Check size={14} /> Approve &amp; run
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

