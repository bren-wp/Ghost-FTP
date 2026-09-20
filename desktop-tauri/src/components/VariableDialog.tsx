import { useId, useMemo, useRef, useState } from "react";
import { useDialog } from "@/hooks/useDialog";
import {
  extractVariables,
  resolveVariables,
  stripTrailingNewline,
} from "@/lib/snippets";
import type { Snippet } from "@/lib/types";

/// The snippet-insert dialog (Plan 11 Phase 4). Shown when a snippet needs
/// values for its `{{variable}}` placeholders and/or is multi-line — it
/// collects each value, previews the resolved command, and confirms the insert.
/// Single-line snippets with no variables skip this entirely (injected
/// directly). The insert never appends a trailing newline, so the command lands
/// at the prompt for the user to run.
export function VariableDialog({
  snippet,
  onCancel,
  onInsert,
}: {
  snippet: Snippet;
  onCancel: () => void;
  onInsert: (values: Record<string, string>) => void;
}) {
  const vars = useMemo(() => extractVariables(snippet.body), [snippet.body]);
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(vars.map((v) => [v, ""]))
  );

  const panelRef = useRef<HTMLDivElement>(null);
  const firstFieldRef = useRef<HTMLInputElement>(null);
  const titleId = useId();
  useDialog(panelRef, {
    onClose: onCancel,
    initialFocus: vars.length ? firstFieldRef : undefined,
  });

  const preview = stripTrailingNewline(resolveVariables(snippet.body, values));
  const multiline = preview.includes("\n");
  const lineCount = preview.split("\n").length;

  const submit = () => onInsert(values);

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onCancel}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          // Enter submits from a single-line field; Shift+Enter / textareas are
          // untouched. Esc is handled by useDialog.
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            submit();
          }
        }}
        className="anim-modal flex w-[30rem] max-w-[92vw] flex-col rounded-xl border border-border bg-bg-panel p-5 shadow-elev-3"
      >
        <div id={titleId} className="mb-1 text-sm font-semibold">
          Insert snippet
        </div>
        <div className="mb-3 truncate text-xs text-text-dim">
          {snippet.name || "(unnamed)"}
        </div>

        {vars.length > 0 && (
          <div className="mb-3 flex flex-col gap-2.5">
            {vars.map((name, i) => (
              <label key={name} className="flex flex-col gap-1">
                <span className="font-mono text-[11px] text-text-muted">
                  {name}
                </span>
                <input
                  ref={i === 0 ? firstFieldRef : undefined}
                  value={values[name] ?? ""}
                  onChange={(e) =>
                    setValues((v) => ({ ...v, [name]: e.target.value }))
                  }
                  placeholder={`Value for ${name}…`}
                  className="rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
                />
              </label>
            ))}
          </div>
        )}

        <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
          Preview
        </div>
        <pre className="mb-3 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-bg-subtle px-2.5 py-2 font-mono text-xs text-text">
          {preview || " "}
        </pre>

        {multiline && (
          <div className="mb-3 rounded-md border border-warning/30 bg-warning/10 px-2.5 py-2 text-[11px] leading-relaxed text-warning">
            This snippet is {lineCount} lines. Each line before the last runs as
            it's inserted; the final line waits at the prompt for you to press
            Enter.
          </div>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={submit}
            className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white"
          >
            Insert
          </button>
        </div>
      </div>
    </div>
  );
}
