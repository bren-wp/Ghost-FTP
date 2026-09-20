import { create } from "zustand";
import type { ConnectionProfile } from "@/lib/types";

// Which app-level modal is open (mutually exclusive). Centralized here so the
// command palette, keyboard shortcuts and title-bar menus can all open them
// without prop-threading through App.
export type AppDialog =
  | "settings"
  | "newConnection"
  | "import"
  | "about"
  | "agentBridge"
  | "grant"
  | "siteManager"
  | "transferCenter"
  | "sync"
  | "help"
  | "updates"
  | "cloudStorage"
  | "schedules"
  | "activityLogs";

/** Seed for the grant consent dialog, parsed from a ghostftp://grant deep link. */
export interface GrantPrefill {
  issuer: string;
  token: string;
  name?: string;
}

interface LayoutState {
  terminalOpen: boolean;
  setTerminalOpen: (open: boolean) => void;
  toggleTerminal: () => void;

  // The Agent console is a dockable bottom panel (not a modal) so it can sit
  // open alongside the file browser and the Bridge control panel.
  consoleOpen: boolean;
  setConsoleOpen: (open: boolean) => void;
  toggleConsole: () => void;

  dialog: AppDialog | null;
  openDialog: (d: AppDialog) => void;
  closeDialog: () => void;

  // Prefill for the New Connection editor, set by a ghostftp:// deep link so the
  // editor opens pointed at the right server (never auto-connecting). Cleared
  // when the editor closes.
  connectionPrefill: Partial<ConnectionProfile> | null;
  openNewConnection: (prefill?: Partial<ConnectionProfile>) => void;

  // Seed for the grant consent dialog (ghostftp://grant deep link). Cleared when
  // the dialog closes.
  grantPrefill: GrantPrefill | null;
  openGrant: (prefill: GrantPrefill) => void;

  // When true, the file browser shows the LOCAL filesystem instead of the
  // active server. Toggled by the rail's "Local" home bubble; cleared when a
  // server bubble is selected.
  browseLocal: boolean;
  setBrowseLocal: (v: boolean) => void;

  // A one-shot "show this path in the file browser" request (from the Disk
  // Usage explorer's "Reveal" action). The active browser consumes it and
  // clears it. `path` is the directory to open (a file's parent).
  revealTarget: { sessionId: string; path: string } | null;
  requestReveal: (sessionId: string, path: string) => void;
  clearReveal: () => void;

  paletteOpen: boolean;
  setPaletteOpen: (v: boolean) => void;
  togglePalette: () => void;

  shortcutsOpen: boolean;
  setShortcutsOpen: (v: boolean) => void;
}

export const useLayout = create<LayoutState>((set) => ({
  terminalOpen: false,
  setTerminalOpen: (open) => set({ terminalOpen: open }),
  toggleTerminal: () => set((s) => ({ terminalOpen: !s.terminalOpen })),

  consoleOpen: false,
  setConsoleOpen: (open) => set({ consoleOpen: open }),
  toggleConsole: () => set((s) => ({ consoleOpen: !s.consoleOpen })),

  dialog: null,
  openDialog: (d) => set({ dialog: d }),
  closeDialog: () =>
    set({ dialog: null, connectionPrefill: null, grantPrefill: null }),

  connectionPrefill: null,
  openNewConnection: (prefill) =>
    set({ dialog: "newConnection", connectionPrefill: prefill ?? null }),

  grantPrefill: null,
  openGrant: (prefill) => set({ dialog: "grant", grantPrefill: prefill }),

  browseLocal: false,
  setBrowseLocal: (v) => set({ browseLocal: v }),

  revealTarget: null,
  requestReveal: (sessionId, path) => set({ revealTarget: { sessionId, path } }),
  clearReveal: () => set({ revealTarget: null }),

  paletteOpen: false,
  setPaletteOpen: (v) => set({ paletteOpen: v }),
  togglePalette: () => set((s) => ({ paletteOpen: !s.paletteOpen })),

  shortcutsOpen: false,
  setShortcutsOpen: (v) => set({ shortcutsOpen: v }),
}));
