import { create } from "zustand";
import { ipc, onFolderSyncChanged } from "@/lib/ipc";
import { toast } from "./toastStore";
import { toastError } from "@/lib/errors";
import type { PairView, SyncPair } from "@/lib/types";

interface SyncStoreState {
  pairs: PairView[];
  loaded: boolean;

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
      try {
        set({ pairs: await ipc.folderSyncList(), loaded: true });
      } catch (error) {
        set({ loaded: false });
        toastError(error, "Couldn't load Sync & Backup");
      }
    }
    const un = await onFolderSyncChanged(() => {
      void get().refresh();
    });
    return un;
  },

  refresh: async () => {
    try {
      set({ pairs: await ipc.folderSyncList(), loaded: true });
    } catch (error) {
      toastError(error, "Couldn't refresh Sync & Backup");
    }
  },

  upsert: async (pair) => {
    const pairs = await ipc.folderSyncUpsert(pair);
    set({ pairs });
    toast.success(pair.id ? "Sync pair updated" : "Sync pair created", pair.name);
  },

  remove: async (id) => {
    set({ pairs: await ipc.folderSyncRemove(id) });
  },

  setEnabled: async (id, enabled) => {
    try {
      set({ pairs: await ipc.folderSyncSetEnabled(id, enabled) });
    } catch (error) {
      void get().refresh();
      throw error;
    }
  },

  syncNow: async (id) => {
    set({ pairs: await ipc.folderSyncSyncNow(id) });
  },
}));
