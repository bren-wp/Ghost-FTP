import { useId, useRef, useState } from "react";
import { useDialog } from "@/hooks/useDialog";
import { toastError } from "@/lib/errors";

interface Props {
  title: string;
  message: string;
  confirmLabel?: string;
  destructive?: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
}

export function ConfirmModal({
  title,
  message,
  confirmLabel = "Confirm",
  destructive,
  onClose,
  onConfirm,
}: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  const [submitting, setSubmitting] = useState(false);
  const closeIfIdle = () => {
    if (!submitting) onClose();
  };
  const confirm = async () => {
    if (submitting) return;
    setSubmitting(true);
    try {
      await onConfirm();
      onClose();
    } catch (error) {
      toastError(error, `Couldn't ${confirmLabel.toLowerCase()}`);
      setSubmitting(false);
    }
  };

  useDialog(panelRef, { onClose: closeIfIdle });
  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={closeIfIdle}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal w-[26rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel p-5 shadow-elev-3"
      >
        <div id={titleId} className="mb-2 text-sm font-semibold">{title}</div>
        <div className="mb-4 whitespace-pre-line text-sm text-text-muted">
          {message}
        </div>
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={closeIfIdle}
            disabled={submitting}
            className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void confirm()}
            disabled={submitting}
            aria-busy={submitting}
            className={
              destructive
                ? "rounded-md bg-danger px-3.5 py-1.5 text-sm font-medium text-white hover:opacity-90 disabled:cursor-wait disabled:opacity-70"
                : "btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:cursor-wait disabled:opacity-70"
            }
          >
            {submitting ? "Working…" : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
