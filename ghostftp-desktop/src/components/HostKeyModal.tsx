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
import { toastError } from "@/lib/errors";

// Mounted once at the top of the tree. Listens for `host://prompt` events
// from the SSH connect handshake and shows the user the fingerprint of an
// unknown or changed host key. Queues prompts in case the user is connecting
// to several profiles quickly.
export function HostKeyModal() {
  const [queue, setQueue] = useState<HostPromptEvent[]>([]);
  const pendingRequestRef = useRef<string | null>(null);
  const [pendingRequestId, setPendingRequestId] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;
    let unlisten: (() => void) | undefined;
    const registration = onHostPrompt((e) =>
      setQueue((q) => q.some((item) => item.requestId === e.requestId) ? q : [...q, e])
    );

    void registration
      .then((cleanup) => {
        if (disposed) cleanup();
        else unlisten = cleanup;
      })
      .catch((error) => {
        toastError(error, "Host-key verification unavailable");
      });

    return () => {
      disposed = true;
      unlisten?.();
    };
  }, []);

  if (queue.length === 0) return null;
  const current = queue[0];

  const respond = async (decision: HostDecision) => {
    // The ref locks synchronously, preventing double clicks and Escape from
    // sending contradictory decisions before React has re-rendered.
    if (pendingRequestRef.current !== null) return;
    const requestId = current.requestId;
    pendingRequestRef.current = requestId;
    setPendingRequestId(requestId);
    try {
      await ipc.respondToHostPrompt(requestId, decision);
      // A slow response must never dequeue a different, newer request.
      setQueue((q) => q.filter((item) => item.requestId !== requestId));
    } catch (error) {
      toastError(error, "Couldn't answer host-key prompt");
    } finally {
      pendingRequestRef.current = null;
      setPendingRequestId(null);
    }
  };

  return <HostKeyDialog key={current.requestId} event={current} pending={pendingRequestId === current.requestId} onRespond={respond} />;
}

function HostKeyDialog({
  event,
  pending,
  onRespond,
}: {
  event: HostPromptEvent;
  pending: boolean;
  onRespond: (d: HostDecision) => void;
}) {
  const isMismatch = event.kind === "mismatch";
  const [verifiedMismatch, setVerifiedMismatch] = useState(false);
  const unsafeDecisionDisabled = pending || (isMismatch && !verifiedMismatch);
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
        aria-busy={pending}
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
            type="button"
            disabled={pending}
            onClick={() => onRespond("reject")}
            aria-label="Reject host key and close"
            className="rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-50"
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
          {isMismatch && (
            <label className="mt-3 flex cursor-pointer items-start gap-2 rounded-md border border-danger/30 bg-danger-soft/20 p-2.5 text-xs leading-relaxed">
              <input
                type="checkbox"
                checked={verifiedMismatch}
                disabled={pending}
                onChange={(e) => setVerifiedMismatch(e.target.checked)}
                className="mt-0.5 shrink-0"
              />
              I verified the changed server fingerprint with the server operator using a trusted channel.
            </label>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border bg-bg-subtle px-4 py-3">
          <button
            ref={rejectRef}
            type="button"
            disabled={pending}
            onClick={() => onRespond("reject")}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover disabled:opacity-50"
          >
            Reject
          </button>
          <button
            type="button"
            disabled={unsafeDecisionDisabled}
            onClick={() => onRespond("accept")}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-40"
          >
            Accept once
          </button>
          <button
            type="button"
            disabled={unsafeDecisionDisabled}
            onClick={() => onRespond("trust")}
            className={
              isMismatch
                ? "rounded-md bg-danger px-3 py-1.5 text-xs font-medium text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
                : "btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
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
