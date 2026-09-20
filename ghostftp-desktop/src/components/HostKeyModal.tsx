import { useEffect, useId, useRef, useState } from "react";
import {
  ShieldCheck,
  ShieldAlert,
  KeyRound,
  X,
} from "lucide-react";
import { ipc, onHostPrompt } from "@/lib/ipc";
import { useDialog } from "@/hooks/useDialog";
import type { HostDecision, HostPromptEvent } from "@/lib/types";

// Mounted once at the top of the tree. Listens for `host://prompt` events
// from the SSH connect handshake and shows the user the fingerprint of an
// unknown or changed host key. Queues prompts in case the user is connecting
// to several profiles quickly.
export function HostKeyModal() {
  const [queue, setQueue] = useState<HostPromptEvent[]>([]);

  useEffect(() => {
    let unlisten: (() => void) | null = null;
    onHostPrompt((e) => setQueue((q) => [...q, e])).then((u) => {
      unlisten = u;
    });
    return () => {
      if (unlisten) unlisten();
    };
  }, []);

  if (queue.length === 0) return null;
  const current = queue[0];

  const respond = async (decision: HostDecision) => {
    try {
      await ipc.respondToHostPrompt(current.requestId, decision);
    } finally {
      setQueue((q) => q.slice(1));
    }
  };

  return <HostKeyDialog event={current} onRespond={respond} />;
}

function HostKeyDialog({
  event,
  onRespond,
}: {
  event: HostPromptEvent;
  onRespond: (d: HostDecision) => void;
}) {
  const isMismatch = event.kind === "mismatch";
  const panelRef = useRef<HTMLDivElement>(null);
  const rejectRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  // Escape rejects (the safe choice) and the safe action takes initial focus —
  // pressing Enter on open can never silently trust an unverified key.
  useDialog(panelRef, {
    onClose: () => onRespond("reject"),
    initialFocus: rejectRef,
  });
  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[30rem] max-w-[92vw] overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div
          className={`flex items-center gap-2.5 border-b px-4 py-3 ${
            isMismatch
              ? "border-danger/30 bg-danger-soft/40"
              : "border-border bg-bg-subtle"
          }`}
        >
          <div
            className={`flex h-7 w-7 items-center justify-center rounded-md ${
              isMismatch
                ? "bg-danger/20 text-danger"
                : "bg-accent-soft text-accent"
            }`}
          >
            {isMismatch ? <ShieldAlert size={15} /> : <ShieldCheck size={15} />}
          </div>
          <div className="min-w-0 flex-1">
            <div id={titleId} className="text-[13px] font-semibold">
              {isMismatch ? "Host key has changed" : "Unknown host"}
            </div>
            <div className="truncate font-mono text-[11px] text-text-dim">
              {event.host}:{event.port}
            </div>
          </div>
          <button
            onClick={() => onRespond("reject")}
            className="rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text"
            title="Cancel"
          >
            <X size={13} />
          </button>
        </div>

        <div className="px-4 py-4">
          {isMismatch ? (
            <p className="mb-3 text-[12.5px] leading-relaxed text-text-muted">
              The server presented a host key that does <strong>not</strong>{" "}
              match the one previously recorded in <code>~/.ssh/known_hosts</code>.
              This could mean the server was re-keyed — or that you are being
              man-in-the-middled. Verify with the host operator before
              continuing.
            </p>
          ) : (
            <p className="mb-3 text-[12.5px] leading-relaxed text-text-muted">
              This is the first time Ghost FTP has connected to this host. Verify the
              fingerprint matches what the server operator told you, then choose
              whether to trust it permanently or accept just this once.
            </p>
          )}

          <FingerprintRow
            label="Server"
            keyType={event.keyType}
            fingerprint={event.fingerprint}
            tone={isMismatch ? "danger" : "neutral"}
          />
          {isMismatch && event.storedFingerprint && (
            <FingerprintRow
              label="Saved"
              keyType={event.keyType}
              fingerprint={event.storedFingerprint}
              tone="neutral"
            />
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border bg-bg-subtle px-4 py-3">
          <button
            ref={rejectRef}
            onClick={() => onRespond("reject")}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover"
          >
            Reject
          </button>
          <button
            onClick={() => onRespond("accept")}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover"
          >
            Accept once
          </button>
          <button
            onClick={() => onRespond("trust")}
            className={
              isMismatch
                ? "rounded-md bg-danger px-3 py-1.5 text-xs font-medium text-white hover:opacity-90"
                : "btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white"
            }
          >
            Trust &amp; save
          </button>
        </div>
      </div>
    </div>
  );
}

function FingerprintRow({
  label,
  keyType,
  fingerprint,
  tone,
}: {
  label: string;
  keyType: string;
  fingerprint: string;
  tone: "neutral" | "danger";
}) {
  return (
    <div className="mb-2 flex items-center gap-2 rounded-md border border-border bg-bg-subtle px-2.5 py-2 last:mb-0">
      <KeyRound
        size={13}
        className={tone === "danger" ? "text-danger" : "text-accent"}
      />
      <div className="min-w-0 flex-1">
        <div className="text-[10px] font-medium uppercase tracking-wider text-text-dim">
          {label} · {keyType}
        </div>
        <div className="selectable truncate font-mono text-[11px]">
          {fingerprint}
        </div>
      </div>
    </div>
  );
}
