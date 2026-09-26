import { create } from "zustand";

// A FileZilla-style "this file already exists" prompt, modelled as a queue of
// questions the transfer code awaits. A batch lists its destination directory
// once, then for every name collision calls `ask(...)` and blocks on the user's
// choice. "Apply to all" lets the caller reuse one answer for the rest of the
// batch; "remember" persists it as the default overwrite policy.

export type ConflictAction = "overwrite" | "rename" | "skip" | "cancel";

export interface ConflictInfo {
  /** Basename of the colliding item. */
  name: string;
  /** Destination directory the collision is in. */
  destDir: string;
  /** Which side already has the file (download → local, upload → remote). */
  side: "local" | "remote";
  kind: "file" | "directory";
  /** How many items still follow in this batch (drives the "apply to all" hint). */
  remaining: number;
  // Optional metadata for a source-vs-existing comparison in the dialog.
  existingSize?: number;
  existingModified?: number;
  sourceSize?: number;
  sourceModified?: number;
}

export interface ConflictDecision {
  action: ConflictAction;
  /** Reuse this decision for the remaining collisions in the batch. */
  applyToAll: boolean;
  /** Persist this action as the default overwrite policy. */
  remember: boolean;
}

interface Pending extends ConflictInfo {
  resolve: (d: ConflictDecision) => void;
}

interface ConflictState {
  /** FIFO of pending prompts; the host renders `queue[0]`. */
  queue: Pending[];
  ask: (info: ConflictInfo) => Promise<ConflictDecision>;
  respond: (decision: ConflictDecision) => void;
}

export const useConflicts = create<ConflictState>((set, get) => ({
  queue: [],

  ask: (info) =>
    new Promise<ConflictDecision>((resolve) => {
      set((s) => ({ queue: [...s.queue, { ...info, resolve }] }));
    }),

  respond: (decision) => {
    const head = get().queue[0];
    if (!head) return;
    set((s) => ({ queue: s.queue.slice(1) }));
    head.resolve(decision);
  },
}));
