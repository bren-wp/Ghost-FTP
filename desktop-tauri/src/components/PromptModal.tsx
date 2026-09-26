import { useEffect, useId, useRef, useState } from "react";
import { useDialog } from "@/hooks/useDialog";

interface Props {
  title: string;
  label?: string;
  initialValue?: string;
  okLabel?: string;
  onClose: () => void;
  onSubmit: (value: string) => void;
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
  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose, initialFocus: inputRef });

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const submit = () => {
    if (value.trim().length === 0) return;
    onSubmit(value);
    onClose();
  };

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60"
      onClick={onClose}
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
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") submit();
            else if (e.key === "Escape") onClose();
          }}
          className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
        />
        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={submit}
            className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white"
          >
            {okLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
