import { create } from "zustand";
import { ipc, onDedupeEvent } from "@/lib/ipc";
import { toast } from "./toastStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import type {
  DedupePhase,
  DedupeProgress,
  DedupeResult,
  DedupeRunState,
  DedupeSnapshot,
  SessionId,
} from "@/lib/types";

interface DedupeStoreState {
  /** Whether the duplicates overlay is open. */
  open: boolean;
  dedupeId: string | null;

  sessionId: SessionId;
  path: string;
  hash: boolean;

  state: DedupeRunState | null;
  phase: DedupePhase;
  filesFound: number;
  hashed: number;
  error: string | null;
  result: DedupeResult | null;

  /** Non-keeper paths the user has UNchecked (spared from deletion). */
  excluded: Set<string>;
  /** Paths successfully deleted from this view. */
  deleted: Set<string>;
  deleting: boolean;

  unlisten: (() => void) | null;

  /** Open the overlay preset to this connection + folder. */
  openFor: (sessionId: SessionId, path: string) => void;
  setPath: (path: string) => void;
  setHash: (on: boolean) => void;
  /** Start (or restart) the scan with the current settings. */
  run: () => Promise<void>;
  cancel: () => Promise<void>;
  close: () => void;
  /** Spare / re-include a non-keeper path for deletion. */
  toggleExcluded: (path: string) => void;
  /** The paths "Delete selected" would remove. */
  selectedPaths: () => string[];
  /** Delete every checked duplicate (the caller confirms first). */
  deleteSelected: () => Promise<void>;
}

function fromSnapshot(s: DedupeSnapshot) {
  return {
    state: s.state,
    phase: s.phase,
    filesFound: s.filesFound,
    hashed: s.hashed,
    error: s.error ?? null,
    result: s.result ?? null,
  };
}

export const useDedupe = create<DedupeStoreState>((set, get) => ({
  open: false,
  dedupeId: null,
  sessionId: LOCAL_SESSION,
  path: "",
  hash: false,
  state: null,
  phase: "walking",
  filesFound: 0,
  hashed: 0,
  error: null,
  result: null,
  excluded: new Set(),
  deleted: new Set(),
  deleting: false,
  unlisten: null,

  openFor: (sessionId, path) => {
    set({
      open: true,
      dedupeId: null,
      sessionId,
      path,
      hash: false,
      state: null,
      phase: "walking",
      filesFound: 0,
      hashed: 0,
      error: null,
      result: null,
      excluded: new Set(),
      deleted: new Set(),
      deleting: false,
    });
  },

  setPath: (path) => set({ path }),
  setHash: (on) => set({ hash: on }),

  run: async () => {
    const prev = get();
    if (prev.dedupeId) {
      void ipc.dedupeForget(prev.dedupeId).catch((error) =>
        console.warn("Couldn't forget previous duplicate scan", error)
      );
    }
    prev.unlisten?.();

    const unlisten = await onDedupeEvent((kind, payload) => {
      if (!payload || payload.id !== get().dedupeId) return;
      if (kind === "progress") {
        const p = payload as DedupeProgress;
        set({ phase: p.phase, filesFound: p.filesFound, hashed: p.hashed });
      } else {
        set(fromSnapshot(payload as DedupeSnapshot));
      }
    });

    const { sessionId, path, hash } = prev;
    set({
      dedupeId: null,
      state: "scanning",
      phase: "walking",
      filesFound: 0,
      hashed: 0,
      error: null,
      result: null,
      excluded: new Set(),
      deleted: new Set(),
      unlisten,
    });

    try {
      const dedupeId = await ipc.dedupeStart(sessionId, path, hash);
      set({ dedupeId });
      // Reconcile: the scan may have settled before `dedupeId` was set, so a
      // terminal event could have been missed above.
      const snap = await ipc.dedupeResult(dedupeId);
      if (snap.state !== "scanning") set(fromSnapshot(snap));
      else set({ phase: snap.phase, filesFound: snap.filesFound, hashed: snap.hashed });
    } catch (e) {
      set({ state: "error", error: String(e) });
      toast.error("Couldn't start the scan", String(e));
    }
  },

  cancel: async () => {
    const { dedupeId } = get();
    if (!dedupeId) return;
    try {
      await ipc.dedupeCancel(dedupeId);
    } catch (error) {
      set({ error: String(error) });
      toast.error("Couldn't cancel duplicate scan", String(error));
    }
  },

  close: () => {
    const { dedupeId, unlisten } = get();
    if (dedupeId) {
      void ipc.dedupeForget(dedupeId).catch((error) =>
        console.warn("Couldn't forget duplicate scan", error)
      );
    }
    unlisten?.();
    set({
      open: false,
      dedupeId: null,
      state: null,
      result: null,
      error: null,
      unlisten: null,
    });
  },

  toggleExcluded: (path) =>
    set((s) => {
      const excluded = new Set(s.excluded);
      if (excluded.has(path)) excluded.delete(path);
      else excluded.add(path);
      return { excluded };
    }),

  selectedPaths: () => {
    const { result, excluded, deleted } = get();
    if (!result) return [];
    const paths: string[] = [];
    for (const g of result.groups) {
      g.files.forEach((f, i) => {
        if (i !== g.keep && !excluded.has(f.path) && !deleted.has(f.path)) {
          paths.push(f.path);
        }
      });
    }
    return paths;
  },

  deleteSelected: async () => {
    const { sessionId } = get();
    const paths = get().selectedPaths();
    if (paths.length === 0) return;
    set({ deleting: true });
    try {
      const errors = await ipc.dedupeDelete(sessionId, paths);
      const failed = new Set(errors.map((e) => e.split(": ")[0]));
      const deleted = new Set(get().deleted);
      for (const p of paths) if (!failed.has(p)) deleted.add(p);
      set({ deleted });
      if (errors.length === 0) {
        toast.success(
          "Duplicates deleted",
          `${paths.length} file${paths.length === 1 ? "" : "s"} removed`
        );
      } else {
        toast.warning(
          "Some deletions failed",
          `${paths.length - errors.length} deleted, ${errors.length} failed`
        );
      }
    } catch (e) {
      toast.error("Delete failed", String(e));
    } finally {
      set({ deleting: false });
    }
  },
}));
