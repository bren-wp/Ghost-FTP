import { create } from "zustand";
import { ipc, onEditSaved, onEditError } from "@/lib/ipc";
import { useSettings } from "./settingsStore";
import { toast } from "./toastStore";
import { baseName, fmtSize } from "@/lib/format";
import { messageOf, toastError } from "@/lib/errors";
import type { SessionId } from "@/lib/types";

// Tracks active edit sessions so the UI can show a "Editing N files" pill
// + per-file save notifications. The store also installs the global
// editor://saved + editor://error listeners on first use.
// Make the first listener registration a single-flight operation: overlapping
// edits must never subscribe twice and duplicate notifications.
let listenerSetup: Promise<void> | null = null;

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
  builtinEditId: string | null;
  /** Has the global listener been wired? */
  _listenersWired: boolean;

  startEditing: (sessionId: SessionId, remotePath: string) => Promise<void>;
  startBuiltinEditing: (sessionId: SessionId, remotePath: string) => Promise<void>;
  stopEditing: (editId: string) => Promise<void>;
  ensureListeners: () => Promise<void>;
}

export const useEditor = create<EditorState>((set, get) => ({
  edits: {},
  builtinEditId: null,
  _listenersWired: false,

  ensureListeners: async () => {
    if (get()._listenersWired) return;
    if (listenerSetup) return listenerSetup;
    // Register saved + error listeners as one atomic, shared setup attempt.
    // Failed registration tears down the saved listener and remains retryable.
    const setup = (async () => {
      const unlistenSaved = await onEditSaved((e) => {
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
      try {
        await onEditError((e) => {
          const detail = messageOf(e.message);
          toast.error("Save failed", `${baseName(e.remotePath)} — ${detail}`);
          set((s) => {
            const existing = s.edits[e.editId];
            if (!existing) return s;
            return {
              edits: {
                ...s.edits,
                [e.editId]: { ...existing, lastError: detail },
              },
            };
          });
        });
        set({ _listenersWired: true });
      } catch (error) {
        unlistenSaved();
        set({ _listenersWired: false });
        throw error;
      }
    })();
    listenerSetup = setup;
    try {
      await setup;
    } finally {
      if (listenerSetup === setup) listenerSetup = null;
    }
  },

  startEditing: async (sessionId, remotePath) => {
    await get().ensureListeners();
    const editor = useSettings.getState().defaultEditor || undefined;
    let ev;
    try {
      ev = await ipc.startEdit(sessionId, remotePath, editor);
    } catch (e) {
      toastError(e, `Couldn't open ${baseName(remotePath)}`);
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

  startBuiltinEditing: async (sessionId, remotePath) => {
    if (get().builtinEditId) {
      toast.info("Editor already open", "Finish or close the current editor first.");
      return;
    }
    await get().ensureListeners();
    let ev;
    try {
      ev = await ipc.startEdit(sessionId, remotePath, undefined, true);
    } catch (error) {
      toastError(error, "Couldn't open built-in editor");
      throw error;
    }
    set((state) => ({
      builtinEditId: ev.editId,
      edits: {
        ...state.edits,
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
    try {
      await ipc.stopEdit(editId);
      set((s) => {
        const next = { ...s.edits };
        delete next[editId];
        return { edits: next, builtinEditId: s.builtinEditId === editId ? null : s.builtinEditId };
      });
    } catch (error) {
      toastError(error, "Couldn't stop editing session");
    }
  },
}));
