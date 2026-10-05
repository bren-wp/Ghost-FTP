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

function requireExcludes(file, forbidden, label) {
  const source = read(file);
  for (const needle of forbidden) {
    if (source.includes(needle)) {
      failures.push(`${file}: forbidden ${label}: ${needle}`);
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
  "packages/file-ui/src/components/ConfirmModal.tsx",
  [
    "const [submitting, setSubmitting] = useState(false)",
    "await onConfirm();",
    "disabled={submitting}",
    "aria-busy={submitting}",
    "Action failed. Please try again.",
    "Working…",
  ],
  "shared file-ui confirm serialization"
);

requireIncludes(
  "packages/file-ui/src/components/PromptModal.tsx",
  [
    "const [submitting, setSubmitting] = useState(false)",
    "await onSubmit(value);",
    "disabled={submitting}",
    "aria-busy={submitting}",
    "Action failed. Please try again.",
    "Working…",
  ],
  "shared file-ui prompt serialization"
);

requireIncludes(
  "packages/file-ui/src/components/FilePane.tsx",
  [
    "setError(errorText(e));",
    "throw e;",
    "const error = new Error(`invalid mode",
    "setError(error.message);",
    "throw error;",
  ],
  "shared file-ui operation failure propagation"
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
    "const clearSecrets = () =>",
    "clearSecrets();",
    'aria-label="Close connection dialog"',
    "Browsing…",
  ],
  "New Connection serialized button actions"
);

requireExcludes(
  "src/components/QuickConnectionDialog.tsx",
  ["console.debug(", "messageOf(error)"],
  "raw quick-connect diagnostics"
);

requireExcludes(
  "src/components/SiteManagerDialog.tsx",
  ["console.debug("],
  "raw site-manager connection diagnostics"
);

requireIncludes(
  "src/stores/transfersStore.ts",
  [
    "await start(item, policy);",
    "existing.set(name, {",
    "same basename in one batch",
  ],
  "batch destination reservation"
);

requireIncludes(
  "src/stores/syncStore.ts",
  [
    "const pairs = await ipc.folderSyncUpsert(pair);",
    "throw error;",
    'toastError(error, "Couldn\'t load Sync & Backup")',
  ],
  "sync mutation failure propagation"
);

requireIncludes(
  "src/components/SyncSettings.tsx",
  [
    "if (!canSubmit || busy) return;",
    'toastError(error, "Couldn\'t create sync pair")',
    'placeholder="Local folder path"',
    'placeholder="Remote folder path"',
    "aria-busy={busy}",
  ],
  "cross-platform sync form failure handling"
);

requireExcludes(
  "src/components/SyncSettings.tsx",
  ["console.warn(", "C:\\\\path\\\\to\\\\folder"],
  "raw diagnostics or Windows-only sync hint"
);

requireIncludes(
  "scripts/init-icons.mjs",
  [
    "fileURLToPath(import.meta.url)",
    "Required production icon missing:",
    "Production desktop icons verified.",
    "<title>Ghost FTP</title>",
  ],
  "fail-closed production desktop assets"
);

requireExcludes(
  "scripts/init-icons.mjs",
  ["makePng", "placeholder icon", "file-ssh placeholder"],
  "generated fallback branding"
);

requireIncludes(
  "src/App.tsx",
  [
    'import { messageOf, toastError } from "./lib/errors";',
    'console.error("Couldn\'t initialize Sync & Backup", messageOf(error));',
    'toastError(error, "Couldn\'t initialize Sync & Backup");',
    'console.error("Couldn\'t initialize transfer activity", messageOf(error));',
    'toastError(error, "Couldn\'t register deep-link handler");',
  ],
  "redacted application initialization failures"
);

requireExcludes(
  "src/App.tsx",
  [
    'toast.error("Couldn\'t initialize Sync & Backup", String(error))',
    'toast.error("Couldn\'t initialize application settings", String(error))',
    'toast.error("Couldn\'t initialize transfer activity", String(error))',
    'toast.error("Couldn\'t register deep-link handler", String(error))',
    'console.error("Couldn\'t initialize app updates", error)',
  ],
  "raw application initialization errors"
);

for (const storeFile of [
  "src/stores/searchStore.ts",
  "src/stores/dedupeStore.ts",
  "src/stores/diskScanStore.ts",
]) {
  requireIncludes(
    storeFile,
    ["messageOf(error)", "toastError(error"],
    "redacted async store failures"
  );
  requireExcludes(
    storeFile,
    ["String(error)", "String(e)"],
    "raw async store failures"
  );
}

for (const storeFile of [
  "src/stores/bridgeStore.ts",
  "src/stores/skillsStore.ts",
  "src/stores/snippetsStore.ts",
]) {
  requireIncludes(
    storeFile,
    ["messageOf(", "toastError("],
    "redacted Agent Bridge async failures"
  );
  requireExcludes(
    storeFile,
    ["String(error)", "String(e)"],
    "raw Agent Bridge async failures"
  );
}

requireIncludes(
  "src/stores/bindingsStore.ts",
  ['import { messageOf } from "@/lib/errors";', "messageOf(error)"],
  "redacted shortcut persistence diagnostics"
);

requireExcludes(
  "src/stores/bindingsStore.ts",
  ["String(error)", "String(e)"],
  "raw shortcut persistence failures"
);

requireIncludes(
  "src/lib/secretMigration.ts",
  ['import { messageOf } from "@/lib/errors";', "messageOf(error)"],
  "redacted settings migration diagnostics"
);

requireExcludes(
  "src/lib/secretMigration.ts",
  ['console.warn("Couldn\'t apply Ghost FTP settings default migrations", error)', 'console.warn("Couldn\'t migrate legacy Ghost FTP settings", error)'],
  "raw settings migration diagnostics"
);

for (const storeFile of [
  "src/stores/diffStore.ts",
  "src/stores/editorStore.ts",
]) {
  requireIncludes(
    storeFile,
    ["messageOf(", "toastError("],
    "redacted user-visible store failures"
  );
  requireExcludes(
    storeFile,
    ["String(error)", "String(e)"],
    "raw user-visible store failures"
  );
}

for (const componentFile of [
  "src/components/HostKeyModal.tsx",
  "src/components/AgentBridge.tsx",
  "src/components/SkillsPanel.tsx",
]) {
  requireIncludes(
    componentFile,
    ["toastError("],
    "redacted component failures"
  );
  requireExcludes(
    componentFile,
    ["String(error)", "String(e)"],
    "raw component failures"
  );
}

requireIncludes(
  "src/components/Settings.tsx",
  ['import { messageOf, toastError } from "@/lib/errors";', "setShellDetail(messageOf(error))"],
  "redacted Settings diagnostics"
);
requireExcludes(
  "src/components/Settings.tsx",
  ["error instanceof Error?error.message:String(error)"],
  "raw Settings diagnostics"
);

for (const componentFile of [
  "src/components/SyncDialog.tsx",
  "src/components/ImportDialog.tsx",
]) {
  requireIncludes(
    componentFile,
    ["messageOf("],
    "redacted dialog failures"
  );
  requireExcludes(
    componentFile,
    ["String(error)", "String(e)"],
    "raw dialog failures"
  );
}

requireIncludes(
  "src/components/TitleBar.tsx",
  [
    'import { createPortal } from "react-dom"',
    "const moreMenuRef = useRef<HTMLDivElement>(null)",
    "createPortal(",
    "document.body",
    'style={{ position: "fixed"',
    "const disconnectActive = () =>",
    'toastError(error, "Couldn\'t disconnect from the active site")',
    'label="Disconnect" onClick={disconnectActive}',
  ],
  "Title bar overflow and disconnect failure-handling contract"
);

requireIncludes(
  "src/components/Settings.tsx",
  [
    'type SettingsAsyncMutation = "reset" | "notifications" | "shell"',
    "const asyncMutationInFlight = useRef(false)",
    "if (asyncMutationInFlight.current) return;",
    'runSettingsAsyncMutation("reset"',
    'runMutation("notifications"',
    'runMutation("shell"',
    "disabled={asyncMutation !== null}",
    "aria-busy={asyncMutation !== null}",
  ],
  "Settings async mutation serialization"
);

requireIncludes(
  "src/stores/settingsStore.ts",
  [
    "let settingsPersistenceTail: Promise<void> = Promise.resolve()",
    "let transferEngineSettingsTail: Promise<void> = Promise.resolve()",
    "function enqueueSettingsPersistence<T>(task: () => Promise<T>): Promise<T>",
    "function enqueueTransferEngineSettings(task: () => Promise<void>): Promise<void>",
    "const durableSettings = structuredClone(initial)",
    "const settingsMutationRevision = new Map<keyof PersistedSettings, number>()",
    "function rememberDurableSetting<K extends keyof PersistedSettings>(",
    "function mutateLiveTransferSetting<K extends keyof PersistedSettings>(",
    "rollbackValue = await persistence;",
    "if (settingsMutationRevision.get(key) !== revision) return;",
    "await persistKey(key, rollbackValue);",
    "await applyNative(rollbackValue);",
    "await enqueueSettingsPersistence(async () => {",
    "rememberDurableSetting(key, previous[key]);",
    "await applyTransferEngineSnapshot(previous);",
    "await (previous.shellIntegration ? ipc.pathAdd() : ipc.pathRemove());",
    '"transferConcurrency",',
    '"maxRetryAttempts",',
    '"transferThrottleKbps",',
    '"deltaSync",',
    "async function applyTransferEngineSnapshot(snapshot: PersistedSettings): Promise<void>",
  ],
  "transactional Settings persistence and live transfer-engine serialization"
);

requireIncludes(
  "src/components/SyncSettings.tsx",
  [
    'const [mutating, setMutating] = useState<"toggle" | "sync" | "remove" | null>(null)',
    "const mutationInFlight = useRef(false)",
    "if (mutationInFlight.current) return;",
    "mutationInFlight.current = true;",
    "mutationInFlight.current = false;",
    'runPairMutation("sync"',
    'runPairMutation("toggle"',
    'runPairMutation("remove"',
    "aria-busy={mutating !== null || freeing}",
    "disabled={mutating !== null || freeing}",
    "disabled?: boolean",
  ],
  "sync-pair mutation serialization"
);

requireIncludes(
  "src/components/AgentBridge.tsx",
  [
    'const [bridgeMutation, setBridgeMutation] = useState<',
    "const bridgeMutationInFlight = useRef(false)",
    "if (bridgeMutationInFlight.current) return;",
    "bridgeMutationInFlight.current = true;",
    "bridgeMutationInFlight.current = false;",
    'runBridgeMutation("master"',
    'runBridgeMutation("endpoint"',
    'runBridgeMutation("session"',
    'runBridgeMutation("policy"',
    'disabled={bridgeMutation !== null}',
    'aria-busy={bridgeMutation === "endpoint"}',
  ],
  "Agent Bridge mutation serialization"
);

requireIncludes(
  "src-tauri/src/path_integration.rs",
  [
    "std::env::split_paths(&path)",
    "fn cli_on_path(",
    "cli_path_lookup_does_not_require_a_shell_process",
  ],
  "shell-free PATH status contract"
);

requireIncludes(
  "src/components/TransferCenterDialog.tsx",
  [
    "event.stopPropagation(); onPauseResume();",
    "event.stopPropagation(); onRetry();",
    "event.stopPropagation(); onCancel();",
    "disabled={!pausable || busy}",
    "disabled={!retryable || busy}",
    "disabled={!cancelable || busy}",
    "const backendActionInFlight = useRef(new Set<string>())",
    "if (backendActionInFlight.current.has(key)) return;",
    "backendActionInFlight.current.add(key);",
    "backendActionInFlight.current.delete(key);",
    'backendActionBusy.has("queue")',
    'backendActionBusy.has(`transfer:${transfer.id}`)',
    "aria-busy={busy}",
    "role=\"menuitem\"",
    "disabled={scheduleMode === \"off\" || !selected}",
    "Export History…",
    "void exportTransferHistory();",
    "const count = await exportHistory(path);",
  ],
  "serialized transfer-row, queue and scheduler button contracts"
);

if (failures.length) {
  console.error(failures.join("\n"));
  process.exit(1);
}

console.log("Interaction hardening contract OK");
