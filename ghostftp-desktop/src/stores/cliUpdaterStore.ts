import { create } from "zustand";
import { ipc, onCliUpdaterStatus } from "@/lib/ipc";
import { toast } from "./toastStore";
import type { CliStatus, CliUpdateMode } from "@/lib/types";

// CLI version-drift store (Plan 10 Phase 0c/0d). Mirrors the backend
// `cli_updater` subsystem: it holds the last status, subscribes to
// `cli-updater://status`, and drives the status-bar prompt + Settings section.
// `dismissed` is UI-only (per session) so the pill can be hidden without
// changing the persisted `ask`/`auto`/`off` preference.

interface CliUpdaterState {
  status: CliStatus | null;
  busy: boolean;
  dismissed: boolean;
  init: () => Promise<() => void>;
  refresh: () => Promise<void>;
  update: () => Promise<void>;
  setMode: (mode: CliUpdateMode) => Promise<void>;
  dismiss: () => void;
}

export const useCliUpdater = create<CliUpdaterState>((set, get) => ({
  status: null,
  busy: false,
  dismissed: false,

  init: async () => {
    const unlisten = await onCliUpdaterStatus((status) => set({ status }));
    // Pull the current status once on mount (also runs the fresh check).
    try {
      const status = await ipc.cliUpdaterStatus();
      set({ status });
    } catch (error) {
      console.warn("Couldn't read the initial Ghost FTP CLI updater status", error);
    }
    return unlisten;
  },

  refresh: async () => {
    try {
      const status = await ipc.cliUpdaterCheck();
      set({ status });
    } catch (e) {
      toast.error(`CLI check failed: ${e}`);
    }
  },

  update: async () => {
    if (get().busy) return;
    set({ busy: true });
    try {
      const status = await ipc.cliUpdaterUpdate();
      set({ status, dismissed: true });
      toast.success(status.message ?? "ghostftp-cli updated");
    } catch (e) {
      toast.error(`ghostftp-cli update failed: ${e}`);
    } finally {
      set({ busy: false });
    }
  },

  setMode: async (mode) => {
    set({ busy: true });
    try {
      const status = await ipc.cliUpdaterSetMode(mode);
      set({ status });
    } catch (e) {
      toast.error(`Couldn't change setting: ${e}`);
    } finally {
      set({ busy: false });
    }
  },

  dismiss: () => set({ dismissed: true }),
}));
