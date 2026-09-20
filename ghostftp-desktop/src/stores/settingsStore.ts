import { create } from "zustand";
import { ipc } from "@/lib/ipc";

export type OverwritePolicy = "overwrite" | "skip" | "rename";
export type SortField = "name" | "size" | "modified";
export type SortDirection = "asc" | "desc";
export type PaneViewMode = "list" | "details" | "grid";
export type PaneDensity = "comfortable" | "compact";
// Remote image previews are opt-in (default off) because a preview means a
// download; local previews are always on and unaffected by this. See Plan 13.
export type RemoteImagePreviews = "off" | "on";
// "single" = one server-focused pane (default); "dual" = local + remote panes.
export type BrowserLayout = "single" | "dual";
export type AppTheme =
  | "dark"
  | "light"
  | "tokyo"
  | "nord"
  | "dracula"
  | "catppuccin"
  | "gruvbox"
  | "solarized"
  | "onedark"
  | "rosepine"
  | "everforest"
  | "ghost-ice"
  | "monokai"
  | "ayu"
  | "palenight"
  | "ghost-midnight";

// `neutral: true` marks the two plain light/dark palettes — they're surfaced as
// the Light/Dark "mode" toggle in Settings, not as named colors. `dark` drives
// which mode tile lights up. Everything else is a named color palette.
export interface ThemeMeta {
  value: AppTheme;
  label: string;
  swatch: string;
  dark: boolean;
  neutral?: boolean;
}

export const APP_THEMES: ThemeMeta[] = [
  { value: "dark", label: "Dark", swatch: "rgb(139 127 246)", dark: true, neutral: true },
  { value: "light", label: "Light", swatch: "rgb(99 88 220)", dark: false, neutral: true },
  { value: "tokyo", label: "Tokyo Night", swatch: "rgb(122 162 247)", dark: true },
  { value: "nord", label: "Nord", swatch: "rgb(136 192 208)", dark: true },
  { value: "dracula", label: "Dracula", swatch: "rgb(189 147 249)", dark: true },
  { value: "catppuccin", label: "Catppuccin", swatch: "rgb(203 166 247)", dark: true },
  { value: "gruvbox", label: "Gruvbox", swatch: "rgb(254 128 25)", dark: true },
  { value: "solarized", label: "Solarized", swatch: "rgb(38 139 210)", dark: true },
  { value: "onedark", label: "One Dark", swatch: "rgb(97 175 239)", dark: true },
  { value: "rosepine", label: "Rosé Pine", swatch: "rgb(196 167 231)", dark: true },
  { value: "everforest", label: "Everforest", swatch: "rgb(167 192 128)", dark: true },
  { value: "monokai", label: "Monokai", swatch: "rgb(249 38 114)", dark: true },
  { value: "ayu", label: "Ayu Mirage", swatch: "rgb(255 167 89)", dark: true },
  { value: "palenight", label: "Palenight", swatch: "rgb(130 170 255)", dark: true },
  { value: "ghost-midnight", label: "Ghost Midnight", swatch: "rgb(47 129 247)", dark: true },
  { value: "ghost-ice", label: "Ghost Ice", swatch: "rgb(9 105 218)", dark: false },
];

/** Named color palettes (everything except the plain Light/Dark neutrals). */
export const COLOR_THEMES = APP_THEMES.filter((t) => !t.neutral);
export type TerminalTheme =
  | "dark"
  | "light"
  | "dracula"
  | "solarized-dark"
  | "gruvbox-dark"
  | "onedark"
  | "rosepine"
  | "everforest";

/** Desktop-notification preference (Plan 16 Phase 3). OS toasts for a curated
 *  set of events, off-window by default so they don't duplicate in-app toasts. */
export interface NotificationSettings {
  enabled: boolean;
  /** Only notify when Ghost FTP isn't the focused window (the default). */
  unfocusedOnly: boolean;
}

interface SettingsState {
  // Appearance
  appTheme: AppTheme;
  /** Optional accent override (hex). "" = use the theme's own accent. */
  accentColor: string;

  // Transfers
  overwritePolicy: OverwritePolicy;
  /** Show a per-file conflict prompt before overwriting. When false, apply
   *  `overwritePolicy` silently (the pre-prompt behaviour). */
  promptOnOverwrite: boolean;
  autoOpenTransferPanel: boolean;
  /** Max simultaneous transfers (1–32). Live-applied to the backend queue. */
  transferConcurrency: number;
  /** Automatic retries for transient network/timeout failures (0–8). */
  maxRetryAttempts: number;
  /** Global bandwidth cap in KiB/s (0 = unlimited). Live-applied. */
  transferThrottleKbps: number;
  /** Delta sync: send only changed blocks on re-transfers (Ghost FTP Agent
   *  connections, files ≥ 8 MB). Live-applied; `GHOSTFTP_DELTA=0` overrides. */
  deltaSync: boolean;
  /** Where downloads land. Blank = the OS Downloads folder. */
  defaultDownloadFolder: string;
  /** Command/path used to open files for edit-in-place. Blank = OS default app. */
  defaultEditor: string;

  // File panes
  showHiddenFiles: boolean;
  sortField: SortField;
  sortDirection: SortDirection;
  paneViewMode: PaneViewMode;
  paneDensity: PaneDensity;
  browserLayout: BrowserLayout;
  /** Remote image previews: `"off"` (default) or `"on"`. Local previews are
   *  always on; this only gates the network-fetching remote kind (Plan 13). */
  remoteImagePreviews: RemoteImagePreviews;
  /** Expand the connection rail into a labeled list (names + addresses) instead
   *  of the compact Discord-style bubble strip. */
  railExpanded: boolean;
  /** Rail groups the user has folded shut (by group name). */
  railCollapsedGroups: string[];

  // Terminal
  terminalFontSize: number;
  terminalFontFamily: string;
  terminalTheme: TerminalTheme;
  terminalScrollback: number;
  /** Copy the terminal selection to the clipboard as soon as it's made
   *  (PuTTY-style). On by default. */
  terminalCopyOnSelect: boolean;
  /** Inline ghost-text history suggestions while typing a command (fish/VS
   *  Code style, → to accept). On by default. */
  terminalSuggestions: boolean;

  // Connections
  defaultPort: number;

  // Integrations
  shellIntegration: boolean;
  fileAssociations: boolean;

  // Notifications (Plan 16 Phase 3)
  notifications: NotificationSettings;

  setAppTheme: (t: AppTheme) => void;
  setAccentColor: (hex: string) => void;
  setOverwritePolicy: (p: OverwritePolicy) => void;
  setPromptOnOverwrite: (v: boolean) => void;
  setAutoOpenTransferPanel: (v: boolean) => void;
  setTransferConcurrency: (n: number) => void;
  setMaxRetryAttempts: (n: number) => void;
  setTransferThrottleKbps: (n: number) => void;
  setDeltaSync: (v: boolean) => void;
  setDefaultDownloadFolder: (s: string) => void;
  setDefaultEditor: (s: string) => void;
  setShowHiddenFiles: (v: boolean) => void;
  setSortField: (f: SortField) => void;
  setSortDirection: (d: SortDirection) => void;
  setPaneViewMode: (m: PaneViewMode) => void;
  setPaneDensity: (d: PaneDensity) => void;
  setBrowserLayout: (l: BrowserLayout) => void;
  setRemoteImagePreviews: (v: RemoteImagePreviews) => void;
  setRailExpanded: (v: boolean) => void;
  toggleRailGroup: (name: string) => void;
  setTerminalFontSize: (n: number) => void;
  setTerminalFontFamily: (s: string) => void;
  setTerminalTheme: (t: TerminalTheme) => void;
  setTerminalScrollback: (n: number) => void;
  setTerminalCopyOnSelect: (v: boolean) => void;
  setTerminalSuggestions: (v: boolean) => void;
  setDefaultPort: (n: number) => void;
  setShellIntegration: (v: boolean) => void;
  setFileAssociations: (v: boolean) => void;
  setNotifications: (v: NotificationSettings) => void;
}

const STORAGE_KEY = "ghostftp.settings.v1";

export type PersistedSettings = Omit<
  SettingsState,
  | "setAppTheme"
  | "setAccentColor"
  | "setOverwritePolicy"
  | "setPromptOnOverwrite"
  | "setAutoOpenTransferPanel"
  | "setTransferConcurrency"
  | "setMaxRetryAttempts"
  | "setTransferThrottleKbps"
  | "setDeltaSync"
  | "setDefaultDownloadFolder"
  | "setDefaultEditor"
  | "setShowHiddenFiles"
  | "setSortField"
  | "setSortDirection"
  | "setPaneViewMode"
  | "setPaneDensity"
  | "setBrowserLayout"
  | "setRemoteImagePreviews"
  | "setRailExpanded"
  | "toggleRailGroup"
  | "setTerminalFontSize"
  | "setTerminalFontFamily"
  | "setTerminalTheme"
  | "setTerminalScrollback"
  | "setTerminalCopyOnSelect"
  | "setTerminalSuggestions"
  | "setDefaultPort"
  | "setShellIntegration"
  | "setFileAssociations"
  | "setNotifications"
>;

const DEFAULTS: PersistedSettings = {
  appTheme: "dark",
  accentColor: "",
  overwritePolicy: "overwrite",
  promptOnOverwrite: true,
  autoOpenTransferPanel: true,
  transferConcurrency: 3,
  maxRetryAttempts: 3,
  transferThrottleKbps: 0,
  deltaSync: true,
  defaultDownloadFolder: "",
  defaultEditor: "",
  // On by default: Ghost FTP's work is server admin, where the interesting files are
  // dotfiles (.htaccess, .env, .ssh/, .git/). Hiding them by default made them
  // look absent rather than filtered.
  showHiddenFiles: true,
  sortField: "name",
  sortDirection: "asc",
  paneViewMode: "details",
  paneDensity: "comfortable",
  browserLayout: "dual",
  remoteImagePreviews: "off",
  railExpanded: true,
  railCollapsedGroups: [],
  terminalFontSize: 13,
  terminalFontFamily:
    '"JetBrains Mono", "Fira Code", "Cascadia Code", Consolas, monospace',
  terminalTheme: "dark",
  terminalScrollback: 5000,
  terminalCopyOnSelect: true,
  terminalSuggestions: true,
  defaultPort: 22,
  shellIntegration: false,
  fileAssociations: false,
  notifications: { enabled: false, unfocusedOnly: true },
};

// The persisted setting keys, in one place — also the authoritative allow-list
// the localStorage→ghostftp.db migration filters against.
export const SETTINGS_KEYS = Object.keys(DEFAULTS) as (keyof PersistedSettings)[];

/** Read the pre-paint snapshot Rust injected on `window.__GHOSTFTP_SETTINGS__`
 *  (Plan 12 Phase 2) — present in the main window, absent in JS-spawned
 *  popouts and mock builds. */
function readInjected(): Partial<PersistedSettings> | null {
  try {
    const inj = (globalThis as { __GHOSTFTP_SETTINGS__?: unknown }).__GHOSTFTP_SETTINGS__;
    if (inj && typeof inj === "object") return inj as Partial<PersistedSettings>;
  } catch {
    // ignore
  }
  return null;
}

/** Keep only recognised setting keys from an arbitrary object. */
function pickKnown(obj: Partial<PersistedSettings>): Partial<PersistedSettings> {
  const out: Partial<PersistedSettings> = {};
  for (const k of SETTINGS_KEYS) {
    const v = (obj as Record<string, unknown>)[k];
    if (v !== undefined) (out as Record<string, unknown>)[k] = v;
  }
  return out;
}

function load(): PersistedSettings {
  const injected = readInjected();
  if (injected) return { ...DEFAULTS, ...pickKnown(injected) };
  // No injection (popout / mock / first paint before the script ran) — start
  // from defaults; `hydrateFromDb()` below reconciles from ghostftp.db async.
  return DEFAULTS;
}

/** Persist one setting to ghostftp.db — the single source of truth (Plan 12 Phase
 *  2). Fire-and-forget: a missing backend (mock) just skips the write, the
 *  in-memory value still applies for the session. */
function persistKey<K extends keyof PersistedSettings>(key: K, value: PersistedSettings[K]) {
  ipc.settingsSet(String(key), JSON.stringify(value)).catch(() => {
    // no backend — ignore
  });
}

const initial = load();

function mutate<K extends keyof PersistedSettings>(
  set: (fn: (s: SettingsState) => Partial<SettingsState>) => void,
  _get: () => SettingsState,
  key: K,
  value: PersistedSettings[K]
) {
  set(() => ({ [key]: value }) as Partial<SettingsState>);
  persistKey(key, value);
}

export const useSettings = create<SettingsState>((set, get) => ({
  ...initial,

  setAppTheme: (t) => mutate(set, get, "appTheme", t),
  setAccentColor: (hex) => mutate(set, get, "accentColor", hex),
  setOverwritePolicy: (p) => mutate(set, get, "overwritePolicy", p),
  setPromptOnOverwrite: (v) => mutate(set, get, "promptOnOverwrite", v),
  setAutoOpenTransferPanel: (v) =>
    mutate(set, get, "autoOpenTransferPanel", v),
  setTransferConcurrency: (n) => {
    const clamped = Math.max(1, Math.min(32, Math.round(n)));
    mutate(set, get, "transferConcurrency", clamped);
    // Live-apply to the running queue; the persisted value covers next launch.
    ipc.transferSetConcurrency(clamped).catch(() => {});
  },
  setMaxRetryAttempts: (n) => {
    const clamped = Math.max(0, Math.min(8, Math.round(n)));
    mutate(set, get, "maxRetryAttempts", clamped);
    ipc.transferSetMaxRetries(clamped).catch(() => {});
  },
  setTransferThrottleKbps: (n) => {
    const clamped = Math.max(0, Math.round(n));
    mutate(set, get, "transferThrottleKbps", clamped);
    ipc.transferSetThrottle(clamped).catch(() => {});
  },
  setDeltaSync: (v) => {
    mutate(set, get, "deltaSync", v);
    // Live-apply to the transfer engine; the persisted value covers next launch.
    ipc.transferSetDeltaSync(v).catch(() => {});
  },
  setDefaultDownloadFolder: (s) =>
    mutate(set, get, "defaultDownloadFolder", s),
  setDefaultEditor: (s) => mutate(set, get, "defaultEditor", s),
  setShowHiddenFiles: (v) => mutate(set, get, "showHiddenFiles", v),
  setSortField: (f) => mutate(set, get, "sortField", f),
  setSortDirection: (d) => mutate(set, get, "sortDirection", d),
  setPaneViewMode: (m) => mutate(set, get, "paneViewMode", m),
  setPaneDensity: (d) => mutate(set, get, "paneDensity", d),
  setBrowserLayout: (l) => mutate(set, get, "browserLayout", l),
  setRemoteImagePreviews: (v) => mutate(set, get, "remoteImagePreviews", v),
  setRailExpanded: (v) => mutate(set, get, "railExpanded", v),
  toggleRailGroup: (name) => {
    const cur = get().railCollapsedGroups;
    mutate(
      set,
      get,
      "railCollapsedGroups",
      cur.includes(name) ? cur.filter((g) => g !== name) : [...cur, name]
    );
  },
  setTerminalFontSize: (n) =>
    mutate(set, get, "terminalFontSize", Math.max(8, Math.min(32, Math.round(n)))),
  setTerminalFontFamily: (s) => mutate(set, get, "terminalFontFamily", s),
  setTerminalTheme: (t) => mutate(set, get, "terminalTheme", t),
  setTerminalScrollback: (n) =>
    mutate(
      set,
      get,
      "terminalScrollback",
      Math.max(100, Math.min(100000, Math.round(n)))
    ),
  setTerminalCopyOnSelect: (v) =>
    mutate(set, get, "terminalCopyOnSelect", v),
  setTerminalSuggestions: (v) =>
    mutate(set, get, "terminalSuggestions", v),
  setDefaultPort: (n) => mutate(set, get, "defaultPort", Math.max(1, Math.min(65535, Math.round(n)))),
  setShellIntegration: (v) => mutate(set, get, "shellIntegration", v),
  setFileAssociations: (v) => mutate(set, get, "fileAssociations", v),
  setNotifications: (v) => mutate(set, get, "notifications", v),
}));

/** Reload settings from ghostftp.db and merge them into the live store. Used by
 *  windows that didn't get the pre-paint injection (popouts) and to reflect a
 *  freshly-completed localStorage→ghostftp.db migration in the current session. */
/** Push the persisted transfer-engine preferences into the native queue.
 * Tauri initializes the engine with safe defaults, then this reconciles the
 * user's saved limits immediately after startup/hydration. */
export function applyTransferEngineSettings(): void {
  const state = useSettings.getState();
  ipc.transferSetConcurrency(state.transferConcurrency).catch(() => {});
  ipc.transferSetMaxRetries(state.maxRetryAttempts).catch(() => {});
  ipc.transferSetThrottle(state.transferThrottleKbps).catch(() => {});
  ipc.transferSetDeltaSync(state.deltaSync).catch(() => {});
}

export async function hydrateFromDb(): Promise<void> {
  try {
    const raw = await ipc.settingsGetAll();
    const parsed: Partial<PersistedSettings> = {};
    for (const k of SETTINGS_KEYS) {
      const v = raw[k as string];
      if (v !== undefined) {
        try {
          (parsed as Record<string, unknown>)[k] = JSON.parse(v);
        } catch {
          // skip a corrupt row
        }
      }
    }
    const known = pickKnown(parsed);
    if (Object.keys(known).length) {
      useSettings.setState(known as Partial<SettingsState>);
    }
    applyTransferEngineSettings();
  } catch {
    // no backend — ignore
  }
}

// Popped-out terminal windows and mock builds never receive the Rust pre-paint
// injection, so seed them from ghostftp.db right after boot.
if (!readInjected()) {
  void hydrateFromDb();
}

export function captureSettingsSnapshot(): PersistedSettings {
  const state = useSettings.getState();
  const snapshot = {} as PersistedSettings;
  for (const key of SETTINGS_KEYS) {
    (snapshot as Record<string, unknown>)[key] = structuredClone(
      (state as unknown as Record<string, unknown>)[key]
    );
  }
  return snapshot;
}

/** Restore a complete preference snapshot, including live transfer-engine limits.
 * Used by Preferences → Cancel so Cancel really cancels changes even though the
 * settings store applies previews immediately. */
export function restoreSettingsSnapshot(snapshot: PersistedSettings): void {
  useSettings.setState({ ...snapshot } as Partial<SettingsState>);
  for (const key of SETTINGS_KEYS) persistKey(key, snapshot[key]);
  ipc.transferSetConcurrency(snapshot.transferConcurrency).catch(() => {});
  ipc.transferSetMaxRetries(snapshot.maxRetryAttempts).catch(() => {});
  ipc.transferSetThrottle(snapshot.transferThrottleKbps).catch(() => {});
  ipc.transferSetDeltaSync(snapshot.deltaSync).catch(() => {});
  (snapshot.shellIntegration ? ipc.pathAdd() : ipc.pathRemove()).catch(() => {});
}

export const TERMINAL_THEMES: Record<
  TerminalTheme,
  { background: string; foreground: string; cursor: string }
> = {
  dark: { background: "#0b0d10", foreground: "#e5e7eb", cursor: "#3b82f6" },
  light: { background: "#fafafa", foreground: "#1f2937", cursor: "#3b82f6" },
  dracula: {
    background: "#282a36",
    foreground: "#f8f8f2",
    cursor: "#ff79c6",
  },
  "solarized-dark": {
    background: "#002b36",
    foreground: "#839496",
    cursor: "#93a1a1",
  },
  "gruvbox-dark": {
    background: "#282828",
    foreground: "#ebdbb2",
    cursor: "#fe8019",
  },
  onedark: {
    background: "#282c34",
    foreground: "#abb2bf",
    cursor: "#61afef",
  },
  rosepine: {
    background: "#191724",
    foreground: "#e0def4",
    cursor: "#c4a7e7",
  },
  everforest: {
    background: "#2d353b",
    foreground: "#d3c6aa",
    cursor: "#a7c080",
  },
};

/** Restore every user-facing preference to the Ghost FTP defaults and persist
 *  the reset to the native settings database. Live transfer limits are also
 *  applied immediately so Reset behaves as a real action rather than a reload. */
export function resetSettingsToDefaults(): void {
  useSettings.setState({ ...DEFAULTS } as Partial<SettingsState>);
  for (const key of SETTINGS_KEYS) persistKey(key, DEFAULTS[key]);
  ipc.transferSetConcurrency(DEFAULTS.transferConcurrency).catch(() => {});
  ipc.transferSetMaxRetries(DEFAULTS.maxRetryAttempts).catch(() => {});
  ipc.transferSetThrottle(DEFAULTS.transferThrottleKbps).catch(() => {});
  ipc.transferSetDeltaSync(DEFAULTS.deltaSync).catch(() => {});
  (DEFAULTS.shellIntegration ? ipc.pathAdd() : ipc.pathRemove()).catch(() => {});
}
