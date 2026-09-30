import { useId, useRef, useState } from "react";
import { useDialog } from "../hooks/useDialog";

interface Props {
  title: string;
  label?: string;
  initialValue?: string;
  okLabel?: string;
  onClose: () => void;
  onSubmit: (value: string) => void | Promise<void>;
}

export function PromptModal({
  title,
  label,
  initialValue = "",
  okLabel = "OK",
  onClose,
  onSubmit,
}: Props) {
  const [value, setValue] = useState(initialValue);
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();

  const closeIfIdle = () => {
    if (!submitting) onClose();
  };

  const submit = async () => {
    if (submitting || value.trim().length === 0) return;
    setSubmitting(true);
    setFailure(null);
    try {
      await onSubmit(value);
      onClose();
    } catch {
      setFailure("Action failed. Please try again.");
      setSubmitting(false);
    }
  };

  useDialog(panelRef, { onClose: closeIfIdle, initialFocus: inputRef });

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60"
      onClick={closeIfIdle}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal w-[24rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel p-5 shadow-elev-3"
      >
        <div id={titleId} className="mb-2 text-sm font-semibold">{title}</div>
        {label && (
          <div className="mb-2 text-xs text-text-dim">{label}</div>
        )}
        <input
          ref={inputRef}
          value={value}
          disabled={submitting}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void submit();
            else if (e.key === "Escape") closeIfIdle();
          }}
          className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent disabled:cursor-wait disabled:opacity-70"
        />
        {failure && (
          <div role="alert" className="mt-2 text-xs text-danger">
            {failure}
          </div>
        )}
        <div className="mt-4 flex justify-end gap-2">
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
            onClick={() => void submit()}
            disabled={submitting || value.trim().length === 0}
            aria-busy={submitting}
            className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:cursor-wait disabled:opacity-70"
          >
            {submitting ? "Working…" : okLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
