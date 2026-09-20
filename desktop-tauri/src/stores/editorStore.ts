import { create } from "zustand";
import { ipc, onEditSaved, onEditError } from "@/lib/ipc";
import { useSettings } from "./settingsStore";
import { toast } from "./toastStore";
import { baseName, fmtSize } from "@/lib/format";
import type { SessionId } from "@/lib/types";

// Tracks active edit sessions so the UI can show a "Editing N files" pill
// + per-file save notifications. The store also installs the global
// editor://saved + editor://error listeners on first use.

export interface ActiveEdit {
  editId: string;
  sessionId: SessionId;
  remotePath: string;
  localTempPath: string;
  /** Most recent save (unix ms). 0 means "not saved since opening". */
  lastSavedAt: number;
  /** Most recent saved size in bytes; null if never saved. */
  lastBytes: number | null;
  /** Last error message, if any. */
  lastError: string | null;
}

interface EditorState {
  edits: Record<string, ActiveEdit>;
  /** Has the global listener been wired? */
  _listenersWired: boolean;

  startEditing: (sessionId: SessionId, remotePath: string) => Promise<void>;
  stopEditing: (editId: string) => Promise<void>;
  ensureListeners: () => Promise<void>;
}

export const useEditor = create<EditorState>((set, get) => ({
  edits: {},
  _listenersWired: false,

  ensureListeners: async () => {
    if (get()._listenersWired) return;
    set({ _listenersWired: true });
    await onEditSaved((e) => {
      // Make the round-trip visible: an external-editor save is silent
      // otherwise, so the user can't tell the change actually uploaded.
      toast.success(
        "Saved to server",
        `${baseName(e.remotePath)} · ${fmtSize(e.bytes)}`
      );
      set((s) => {
        const existing = s.edits[e.editId];
        if (!existing) return s;
        return {
          edits: {
            ...s.edits,
            [e.editId]: {
              ...existing,
              lastSavedAt: Date.now(),
              lastBytes: e.bytes,
              lastError: null,
            },
          },
        };
      });
    });
    await onEditError((e) => {
      toast.error("Save failed", `${baseName(e.remotePath)} — ${e.message}`);
      set((s) => {
        const existing = s.edits[e.editId];
        if (!existing) return s;
        return {
          edits: {
            ...s.edits,
            [e.editId]: { ...existing, lastError: e.message },
          },
        };
      });
    });
  },

  startEditing: async (sessionId, remotePath) => {
    await get().ensureListeners();
    const editor = useSettings.getState().defaultEditor || undefined;
    let ev;
    try {
      ev = await ipc.startEdit(sessionId, remotePath, editor);
    } catch (e) {
      toast.error("Couldn't open editor", `${baseName(remotePath)} — ${e}`);
      throw e;
    }
    toast.info(
      "Opened for editing",
      `${baseName(remotePath)} — saving in your editor uploads it back`
    );
    set((s) => ({
      edits: {
        ...s.edits,
        [ev.editId]: {
          editId: ev.editId,
          sessionId: ev.sessionId,
          remotePath: ev.remotePath,
          localTempPath: ev.localTempPath,
          lastSavedAt: 0,
          lastBytes: null,
          lastError: null,
        },
      },
    }));
  },

  stopEditing: async (editId) => {
    await ipc.stopEdit(editId).catch(() => {});
    set((s) => {
      const next = { ...s.edits };
      delete next[editId];
      return { edits: next };
    });
  },
}));
