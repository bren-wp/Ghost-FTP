import { create } from "zustand";
import { ipc, onDiffEvent } from "@/lib/ipc";
import { toast } from "./toastStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import type {
  DiffClass,
  DiffPhase,
  DiffProgress,
  DiffResult,
  DiffRunState,
  DiffSnapshot,
  SessionId,
} from "@/lib/types";

/** One side of a diff: which connection and which directory. */
export interface DiffSide {
  sessionId: SessionId;
  path: string;
}

/** Which classes the row list shows. "same" is off by default — the point of a
 *  diff is the differences. */
export type DiffFilters = Record<DiffClass, boolean>;

const DEFAULT_FILTERS: DiffFilters = {
  onlyInA: true,
  onlyInB: true,
  different: true,
  same: false,
};

interface DiffStoreState {
  /** Whether the diff overlay is open. */
  open: boolean;
  diffId: string | null;

  sideA: DiffSide;
  sideB: DiffSide;
  hash: boolean;

  state: DiffRunState | null;
  phase: DiffPhase;
  filesA: number;
  filesB: number;
  error: string | null;
  result: DiffResult | null;

  filters: DiffFilters;

  unlisten: (() => void) | null;

  /** Open the overlay with side A preset; the user picks side B, then runs. */
  openFor: (sessionId: SessionId, path: string) => void;
  setSideB: (side: Partial<DiffSide>) => void;
  setHash: (on: boolean) => void;
  /** Start (or restart) the comparison with the current sides + hash setting. */
  run: () => Promise<void>;
  cancel: () => Promise<void>;
  close: () => void;
  toggleFilter: (cls: DiffClass) => void;
}

function fromSnapshot(s: DiffSnapshot) {
  return {
    state: s.state,
    phase: s.phase,
    filesA: s.filesA,
    filesB: s.filesB,
    error: s.error ?? null,
    result: s.result ?? null,
    hash: s.hashed,
  };
}

export const useDiff = create<DiffStoreState>((set, get) => ({
  open: false,
  diffId: null,
  sideA: { sessionId: LOCAL_SESSION, path: "" },
  sideB: { sessionId: LOCAL_SESSION, path: "" },
  hash: false,
  state: null,
  phase: "walkingA",
  filesA: 0,
  filesB: 0,
  error: null,
  result: null,
  filters: { ...DEFAULT_FILTERS },
  unlisten: null,

  openFor: (sessionId, path) => {
    // Default side B to the local filesystem at the same path (the common
    // "compare this server folder against my local copy" case). If side A is
    // already local, leave B's path blank for the user to fill in.
    const sideB: DiffSide =
      sessionId === LOCAL_SESSION
        ? { sessionId: LOCAL_SESSION, path: "" }
        : { sessionId: LOCAL_SESSION, path };
    set({
      open: true,
      diffId: null,
      sideA: { sessionId, path },
      sideB,
      hash: false,
      state: null,
      phase: "walkingA",
      filesA: 0,
      filesB: 0,
      error: null,
      result: null,
      filters: { ...DEFAULT_FILTERS },
    });
  },

  setSideB: (side) => set((s) => ({ sideB: { ...s.sideB, ...side } })),
  setHash: (on) => set({ hash: on }),

  run: async () => {
    const prev = get();
    if (prev.diffId) ipc.diffForget(prev.diffId).catch(() => {});
    prev.unlisten?.();

    const unlisten = await onDiffEvent((kind, payload) => {
      if (!payload || payload.id !== get().diffId) return;
      if (kind === "progress") {
        const p = payload as DiffProgress;
        set({ phase: p.phase, filesA: p.filesA, filesB: p.filesB });
      } else {
        set(fromSnapshot(payload as DiffSnapshot));
      }
    });

    const { sideA, sideB, hash } = prev;
    set({
      diffId: null,
      state: "comparing",
      phase: "walkingA",
      filesA: 0,
      filesB: 0,
      error: null,
      result: null,
      unlisten,
    });

    try {
      const diffId = await ipc.diffStart(
        sideA.sessionId,
        sideA.path,
        sideB.sessionId,
        sideB.path,
        hash
      );
      set({ diffId });
      // Reconcile: the diff may have settled before `diffId` was set, so a
      // terminal event could have been missed above.
      const snap = await ipc.diffResult(diffId);
      if (snap.state !== "comparing") set(fromSnapshot(snap));
      else
        set({
          phase: snap.phase,
          filesA: snap.filesA,
          filesB: snap.filesB,
        });
    } catch (e) {
      set({ state: "error", error: String(e) });
      toast.error("Couldn't start the diff", String(e));
    }
  },

  cancel: async () => {
    const { diffId } = get();
    if (diffId) await ipc.diffCancel(diffId).catch(() => {});
  },

  close: () => {
    const { diffId, unlisten } = get();
    if (diffId) ipc.diffForget(diffId).catch(() => {});
    unlisten?.();
    set({
      open: false,
      diffId: null,
      state: null,
      result: null,
      error: null,
      unlisten: null,
    });
  },

  toggleFilter: (cls) =>
    set((s) => ({ filters: { ...s.filters, [cls]: !s.filters[cls] } })),
}));
