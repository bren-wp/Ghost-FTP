import fs from "node:fs";

const read = (path) => fs.readFileSync(path, "utf8");
const failures = [];

function requireIncludes(file, required, label) {
  const source = read(file);
  for (const needle of required) {
    if (!source.includes(needle)) {
      failures.push(`${file}: missing ${label}: ${needle}`);
    }
  }
}

requireIncludes(
  "src/components/ConfirmModal.tsx",
  [
    "const [submitting, setSubmitting] = useState(false)",
    "await onConfirm();",
    "disabled={submitting}",
    "aria-busy={submitting}",
    "toastError(error",
    "Working…",
  ],
  "duplicate-confirm protection"
);

requireIncludes(
  "src/components/AuthPromptModal.tsx",
  [
    "messageOf(error)",
    "const [submitting, setSubmitting] = useState(false)",
    "const [generating, setGenerating] = useState<number | null>(null)",
    "const [saving, setSaving] = useState(false)",
    "const cancelIfIdle = () =>",
    "disabled={busy}",
    "aria-busy={submitting}",
    "aria-busy={saving}",
    "Submitting…",
    "Updating…",
  ],
  "authentication prompt action locking and redaction"
);

requireIncludes(
  "src/components/ContextMenu.tsx",
  [
    "const [busyIndex, setBusyIndex] = useState<number | null>(null)",
    "await item.onClick();",
    "toastError(error",
    "Math.max(8, window.innerWidth - 220)",
    "aria-busy={busyIndex !== null}",
    "Working…",
  ],
  "context-menu async action contract"
);

requireIncludes(
  "src/components/QuickConnectionDialog.tsx",
  [
    "const actionBusy = busy || keyPicking || testStatus === \"testing\"",
    "const closeIfIdle = () =>",
    "if (!canConnect || actionBusy) return;",
    "disabled={!canConnect||actionBusy}",
    "aria-busy={testStatus===\"testing\"}",
    "aria-busy={busy}",
    "messageOf(error)",
    "Browsing…",
  ],
  "New Connection serialized button actions"
);

requireIncludes(
  "src/components/TransferCenterDialog.tsx",
  [
    "event.stopPropagation(); onPauseResume();",
    "event.stopPropagation(); onRetry();",
    "event.stopPropagation(); onCancel();",
    "disabled={!pausable}",
    "disabled={!retryable}",
    "disabled={!cancelable}",
    "role=\"menuitem\"",
    "disabled={scheduleMode === \"off\" || !selected}",
  ],
  "transfer-row and scheduler button contracts"
);

if (failures.length) {
  console.error(failures.join("\n"));
  process.exit(1);
}

console.log("Interaction hardening contract OK");
