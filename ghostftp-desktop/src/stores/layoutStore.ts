import { create } from "zustand";
import type { ConnectionProfile } from "@/lib/types";

// Which primary workspace or transient overlay is active. Centralized here so
// navigation, shortcuts and contextual actions can route without prop-threading.
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
  | "updates";

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

  // The Agent console is a dockable bottom panel so it can sit
  // open alongside the file browser and the Bridge control panel.
  consoleOpen: boolean;
  setConsoleOpen: (open: boolean) => void;
  toggleConsole: () => void;

  dialog: AppDialog | null;
  /** Parent workspace restored after a transient editor/import/consent dialog closes. */
  returnDialog: AppDialog | null;
  openDialog: (d: AppDialog) => void;
  closeDialog: () => void;
  showFiles: () => void;

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
  // server is selected.
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
  returnDialog: null,
  openDialog: (d) =>
    set((state) => {
      const transient = d === "import" || d === "agentBridge";
      return {
        dialog: d,
        returnDialog:
          transient && state.dialog && state.dialog !== d
            ? state.dialog
            : transient
              ? state.returnDialog
              : null,
        connectionPrefill: null,
        grantPrefill: null,
      };
    }),
  closeDialog: () =>
    set((state) => {
      const transient =
        state.dialog === "newConnection" ||
        state.dialog === "import" ||
        state.dialog === "grant" ||
        state.dialog === "agentBridge";
      return {
        dialog: transient ? state.returnDialog : null,
        returnDialog: null,
        connectionPrefill: null,
        grantPrefill: null,
      };
    }),
  showFiles: () =>
    set({
      dialog: null,
      returnDialog: null,
      connectionPrefill: null,
      grantPrefill: null,
    }),

  connectionPrefill: null,
  openNewConnection: (prefill) =>
    set((state) => ({
      dialog: "newConnection",
      returnDialog:
        state.dialog && state.dialog !== "newConnection"
          ? state.dialog
          : state.returnDialog,
      connectionPrefill: prefill ?? null,
    })),

  grantPrefill: null,
  openGrant: (prefill) =>
    set((state) => ({
      dialog: "grant",
      returnDialog:
        state.dialog && state.dialog !== "grant"
          ? state.dialog
          : state.returnDialog,
      grantPrefill: prefill,
    })),

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
