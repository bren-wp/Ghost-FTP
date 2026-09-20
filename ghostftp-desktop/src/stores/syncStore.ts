import { create } from "zustand";
import { ipc, onFolderSyncChanged } from "@/lib/ipc";
import { toast } from "./toastStore";
import type { PairView, SyncPair } from "@/lib/types";

// Frontend view of the Folder Sync engine. The Rust backend owns the pairs,
// watches the folders and runs the transfers; every command returns the full,
// updated list so we just mirror what it hands back. A "foldersync://changed"
// event fires whenever a background sync moves a pair's state — we re-fetch.
interface SyncStoreState {
  pairs: PairView[];
  loaded: boolean;

  /** Fetch the current list and attach the change listener. Returns the
   *  unlisten fn — call it on teardown. */
  init: () => Promise<() => void>;
  refresh: () => Promise<void>;
  upsert: (pair: SyncPair) => Promise<void>;
  remove: (id: string) => Promise<void>;
  setEnabled: (id: string, enabled: boolean) => Promise<void>;
  syncNow: (id: string) => Promise<void>;
}

export const useSync = create<SyncStoreState>((set, get) => ({
  pairs: [],
  loaded: false,

  init: async () => {
    if (!get().loaded) {
      set({ loaded: true });
      try {
        set({ pairs: await ipc.folderSyncList() });
      } catch {
        // backend not ready yet — the listener below still attaches
      }
    }
    const un = await onFolderSyncChanged(() => {
      void get().refresh();
    });
    return un;
  },

  refresh: async () => {
    try {
      set({ pairs: await ipc.folderSyncList() });
    } catch {
      // ignore — a transient failure; the next event will reconcile
    }
  },

  upsert: async (pair) => {
    try {
      set({ pairs: await ipc.folderSyncUpsert(pair) });
      toast.success(pair.id ? "Sync pair updated" : "Sync pair created", pair.name);
    } catch (e) {
      toast.error("Couldn't save sync pair", String(e));
    }
  },

  remove: async (id) => {
    try {
      set({ pairs: await ipc.folderSyncRemove(id) });
    } catch (e) {
      toast.error("Couldn't remove sync pair", String(e));
    }
  },

  setEnabled: async (id, enabled) => {
    try {
      set({ pairs: await ipc.folderSyncSetEnabled(id, enabled) });
    } catch (e) {
      toast.error("Couldn't update sync pair", String(e));
      void get().refresh();
    }
  },

  syncNow: async (id) => {
    try {
      set({ pairs: await ipc.folderSyncSyncNow(id) });
    } catch (e) {
      toast.error("Couldn't start sync", String(e));
    }
  },
}));
