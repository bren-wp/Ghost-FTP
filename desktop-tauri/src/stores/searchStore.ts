import { create } from "zustand";
import { ipc, onSearchEvent } from "@/lib/ipc";
import { toast } from "./toastStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import type {
  SearchHit,
  SearchHitBatch,
  SearchKind,
  SearchProgress,
  SearchQuery,
  SearchRunState,
  SearchSnapshot,
  SearchStrategy,
  SessionId,
} from "@/lib/types";

/** Split a raw "*.rs, *.toml" glob field into a trimmed, non-empty list. */
function splitGlobs(raw: string): string[] {
  return raw
    .split(/[,\s]+/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

interface SearchStoreState {
  /** Whether the search overlay is open. */
  open: boolean;
  searchId: string | null;

  /** Where to search: the connection + root directory (side is fixed once open). */
  sessionId: SessionId;
  root: string;

  // Query knobs (bound to the input row).
  pattern: string;
  kind: SearchKind;
  regex: boolean;
  caseSensitive: boolean;
  include: string;
  exclude: string;
  contentRemote: boolean;

  // Run state.
  state: SearchRunState | null;
  strategy: SearchStrategy;
  filesScanned: number;
  hitCount: number;
  truncated: boolean;
  note: string | null;
  error: string | null;
  hits: SearchHit[];

  unlisten: (() => void) | null;

  /** Open the overlay rooted at a connection + path; the user types + runs. */
  openFor: (sessionId: SessionId, path: string) => void;
  setPattern: (p: string) => void;
  setKind: (k: SearchKind) => void;
  setRegex: (on: boolean) => void;
  setCaseSensitive: (on: boolean) => void;
  setInclude: (s: string) => void;
  setExclude: (s: string) => void;
  setContentRemote: (on: boolean) => void;
  /** Start (or restart) the search with the current query. */
  run: () => Promise<void>;
  cancel: () => Promise<void>;
  close: () => void;
}

function fromSnapshot(s: SearchSnapshot) {
  return {
    state: s.state,
    strategy: s.strategy,
    filesScanned: s.filesScanned,
    hitCount: s.hitCount,
    truncated: s.truncated,
    note: s.note ?? null,
    error: s.error ?? null,
    // The done snapshot carries the authoritative hit list; prefer it over the
    // streamed batches (dedupes any race on `searchId`).
    ...(s.hits ? { hits: s.hits } : {}),
  };
}

const RESET = {
  searchId: null,
  state: null as SearchRunState | null,
  strategy: "generic" as SearchStrategy,
  filesScanned: 0,
  hitCount: 0,
  truncated: false,
  note: null,
  error: null,
  hits: [] as SearchHit[],
};

export const useSearch = create<SearchStoreState>((set, get) => ({
  open: false,
  sessionId: LOCAL_SESSION,
  root: "",
  pattern: "",
  kind: "name",
  regex: false,
  caseSensitive: false,
  include: "",
  exclude: "",
  contentRemote: false,
  ...RESET,
  unlisten: null,

  openFor: (sessionId, path) => {
    get().unlisten?.();
    set({
      open: true,
      sessionId,
      root: path,
      pattern: "",
      kind: "name",
      regex: false,
      caseSensitive: false,
      include: "",
      exclude: "",
      contentRemote: false,
      unlisten: null,
      ...RESET,
    });
  },

  setPattern: (pattern) => set({ pattern }),
  setKind: (kind) => set({ kind }),
  setRegex: (regex) => set({ regex }),
  setCaseSensitive: (caseSensitive) => set({ caseSensitive }),
  setInclude: (include) => set({ include }),
  setExclude: (exclude) => set({ exclude }),
  setContentRemote: (contentRemote) => set({ contentRemote }),

  run: async () => {
    const prev = get();
    if (!prev.pattern.trim()) return;
    if (prev.searchId) ipc.searchForget(prev.searchId).catch(() => {});
    prev.unlisten?.();

    const unlisten = await onSearchEvent((kind, payload) => {
      if (!payload || payload.id !== get().searchId) return;
      if (kind === "progress") {
        const p = payload as SearchProgress;
        set({
          strategy: p.strategy,
          filesScanned: p.filesScanned,
          hitCount: p.hitCount,
        });
      } else if (kind === "hit") {
        const b = payload as SearchHitBatch;
        set((s) => ({ hits: [...s.hits, ...b.hits] }));
      } else {
        set(fromSnapshot(payload as SearchSnapshot));
      }
    });

    const query: SearchQuery = {
      pattern: prev.pattern,
      kind: prev.kind,
      regex: prev.regex,
      caseSensitive: prev.caseSensitive,
      includeGlobs: splitGlobs(prev.include),
      excludeGlobs: splitGlobs(prev.exclude),
      contentRemote: prev.contentRemote,
      maxResults: 1000,
      maxFileBytes: 8 * 1024 * 1024,
    };

    set({ ...RESET, state: "searching", unlisten });

    try {
      const searchId = await ipc.searchStart(prev.sessionId, prev.root, query);
      set({ searchId });
      // Reconcile: the search may have settled before `searchId` was set, so a
      // terminal event could have been missed above.
      const snap = await ipc.searchResult(searchId);
      if (snap.state !== "searching") set(fromSnapshot(snap));
    } catch (e) {
      set({ state: "error", error: String(e) });
      toast.error("Couldn't start the search", String(e));
    }
  },

  cancel: async () => {
    const { searchId } = get();
    if (searchId) await ipc.searchCancel(searchId).catch(() => {});
  },

  close: () => {
    const { searchId, unlisten } = get();
    if (searchId) ipc.searchForget(searchId).catch(() => {});
    unlisten?.();
    set({ open: false, unlisten: null, ...RESET });
  },
}));
