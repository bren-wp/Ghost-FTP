import { create } from "zustand";
import { ipc, onTransferEvent, onTransferQueue } from "@/lib/ipc";
import { useSettings } from "./settingsStore";
import { toast } from "./toastStore";
import { useConflicts, type ConflictDecision } from "./conflictStore";
import { baseName } from "@/lib/format";
import { toastError } from "@/lib/errors";
import {
  LOCAL_SESSION,
  type SessionId,
  type Transfer,
  type OverwritePolicy,
} from "@/lib/types";

/** A file/folder queued for transfer. Lighter than DirEntry so native-picker
 *  paths (which have no metadata) can be enqueued too. */
export interface TransferItem {
  path: string;
  kind: "file" | "directory";
  size?: number;
  modified?: number;
}

/** Map a DirEntry (kind may be symlink/other) to a TransferItem (file/dir). */
export function toTransferItem(e: {
  path: string;
  kind: string;
  size?: number;
  modified?: number | null;
}): TransferItem {
  return {
    path: e.path,
    kind: e.kind === "directory" ? "directory" : "file",
    size: e.size,
    modified: e.modified ?? undefined,
  };
}

export interface TransferRateSample {
  bytesPerSecond: number;
  sampledAt: number;
}

interface TransfersState {
  byId: Record<string, Transfer>;
  /** Rolling rates derived from real backend progress byte deltas. */
  rateById: Record<string, TransferRateSample>;
  /** Waiting (queued) transfer ids in FIFO order — position = index + 1. */
  queue: string[];
  pausedAll: boolean;
  concurrency: number;
  throttleKbps: number;

  initListeners: () => Promise<() => void>;
  loadInitial: () => Promise<void>;

  // Low-level starts. `policy` overrides the default overwrite policy for this
  // one transfer (used after the user answers the conflict prompt).
  download: (
    sessionId: SessionId,
    remotePath: string,
    localDir: string,
    policy?: OverwritePolicy
  ) => Promise<string>;
  upload: (
    sessionId: SessionId,
    localPath: string,
    remoteDir: string,
    policy?: OverwritePolicy
  ) => Promise<string>;
  downloadDir: (
    sessionId: SessionId,
    remoteDir: string,
    localDir: string,
    policy?: OverwritePolicy
  ) => Promise<string[]>;
  uploadDir: (
    sessionId: SessionId,
    localDir: string,
    remoteDir: string,
    policy?: OverwritePolicy
  ) => Promise<string[]>;
  // Batch entry points: detect name collisions against the destination and
  // prompt (FileZilla-style) before starting each transfer.
  enqueueDownloads: (
    sessionId: SessionId,
    items: TransferItem[],
    localDir: string
  ) => Promise<void>;
  enqueueUploads: (
    sessionId: SessionId,
    items: TransferItem[],
    remoteDir: string
  ) => Promise<void>;
  cancel: (id: string) => Promise<void>;
  pause: (id: string) => Promise<void>;
  resume: (id: string) => Promise<void>;
  retry: (id: string) => Promise<void>;
  move: (id: string, dir: "up" | "down") => Promise<void>;
  pauseAll: () => Promise<void>;
  resumeAll: () => Promise<void>;
  setConcurrency: (n: number) => Promise<void>;
  setThrottle: (kbps: number) => Promise<void>;
  clearFinished: () => void;
  clearCompleted: () => void;
}

/** Map a conflict action to the backend overwrite policy. "skip"/"cancel" are
 *  handled by the caller (they don't start a transfer), so this only covers the
 *  two that do. */
function actionToPolicy(action: ConflictDecision["action"]): OverwritePolicy {
  return action === "rename" ? "rename" : "overwrite";
}

/** Shared conflict loop. Lists the destination once, then for each item either
 *  starts it (no collision, or user chose overwrite/rename) or skips it. A
 *  remembered "apply to all" decision short-circuits later prompts; "cancel"
 *  stops the batch. */
async function runBatch(
  destSessionId: SessionId,
  destDir: string,
  side: "local" | "remote",
  items: TransferItem[],
  start: (item: TransferItem, policy: OverwritePolicy) => Promise<unknown>
): Promise<void> {
  const settings = useSettings.getState();
  const defaultPolicy = settings.overwritePolicy;

  // "Don't ask" mode (the user ticked "remember" before, or turned prompts off
  // in Settings): apply the saved default silently — the original behaviour.
  if (!settings.promptOnOverwrite) {
    for (const item of items) {
      try {
        await start(item, defaultPolicy);
      } catch (error) {
        toastError(error, `Couldn't queue ${baseName(item.path)}`);
      }
    }
    return;
  }

  // Snapshot the destination's names so we can spot collisions. If it can't be
  // listed (e.g. doesn't exist yet), assume no conflicts.
  const existing = new Map<string, { size: number; modified?: number }>();
  try {
    const list = await ipc.listDirectory(destSessionId, destDir);
    for (const e of list) {
      existing.set(e.name, { size: e.size, modified: e.modified ?? undefined });
    }
  } catch (error) {
    toastError(error, "Couldn't inspect the destination for existing files");
    return;
  }

  let remembered: ConflictDecision | null = null;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    const name = baseName(item.path);
    const hit = existing.get(name);
    let policy = defaultPolicy;

    if (hit) {
      const decision: ConflictDecision =
        remembered ??
        (await useConflicts.getState().ask({
          name,
          destDir,
          side,
          kind: item.kind,
          remaining: items.length - i - 1,
          existingSize: item.kind === "file" ? hit.size : undefined,
          existingModified: hit.modified ?? undefined,
          sourceSize: item.kind === "file" ? item.size : undefined,
          sourceModified: item.modified,
        }));

      if (decision.applyToAll) remembered = decision;
      if (decision.remember && decision.action !== "cancel") {
        // Persist as the default AND stop asking from now on.
        settings.setOverwritePolicy(
          decision.action === "skip" ? "skip" : actionToPolicy(decision.action)
        );
        settings.setPromptOnOverwrite(false);
      }
      if (decision.action === "cancel") break;
      if (decision.action === "skip") continue;
      policy = actionToPolicy(decision.action);
    }

    try {
      await start(item, policy);
    } catch (error) {
      toastError(error, `Couldn't queue ${baseName(item.path)}`);
    }
  }
}

function indexBy(transfers: Transfer[]): Record<string, Transfer> {
  return transfers.reduce<Record<string, Transfer>>((acc, t) => {
    acc[t.id] = t;
    return acc;
  }, {});
}

const transferRateBaselines = new Map<string, { bytes: number; at: number }>();
let transferEventRevision = 0;
let queueEventRevision = 0;
const RATE_STALE_AFTER_MS = 2_500;
const RATE_EMA_WEIGHT = 0.35;

export function liveTransferRate(
  transfer: Transfer,
  sample: TransferRateSample | undefined,
  now = Date.now()
): number {
  if (
    transfer.status !== "transferring" ||
    !sample ||
    now - sample.sampledAt > RATE_STALE_AFTER_MS
  ) {
    return 0;
  }
  return sample.bytesPerSecond;
}

function rateFromEvent(
  kind: string,
  transfer: Transfer,
  previous: TransferRateSample | undefined
): TransferRateSample {
  const now = Date.now();

  if (transfer.status !== "transferring") {
    transferRateBaselines.delete(transfer.id);
    return { bytesPerSecond: 0, sampledAt: now };
  }

  const baseline = transferRateBaselines.get(transfer.id);
  transferRateBaselines.set(transfer.id, { bytes: transfer.transferred, at: now });

  if (
    kind !== "progress" ||
    !baseline ||
    transfer.transferred < baseline.bytes
  ) {
    return { bytesPerSecond: 0, sampledAt: now };
  }

  const elapsedMs = now - baseline.at;
  if (elapsedMs < 40) {
    return previous
      ? { ...previous, sampledAt: now }
      : { bytesPerSecond: 0, sampledAt: now };
  }

  const instant =
    ((transfer.transferred - baseline.bytes) * 1000) / elapsedMs;
  const bytesPerSecond =
    previous && previous.bytesPerSecond > 0
      ? previous.bytesPerSecond * (1 - RATE_EMA_WEIGHT) +
        instant * RATE_EMA_WEIGHT
      : instant;

  return {
    bytesPerSecond: Math.max(0, bytesPerSecond),
    sampledAt: now,
  };
}

export const useTransfers = create<TransfersState>((set, get) => ({
  byId: {},
  rateById: {},
  queue: [],
  pausedAll: false,
  concurrency: 3,
  throttleKbps: 0,

  initListeners: async () => {
    let unlistenEvents: (() => void) | null = null;
    try {
      unlistenEvents = await onTransferEvent((kind, t) => {
        transferEventRevision++;
        // Keep the transfer snapshot plus rolling rate telemetry from actual
        // backend progress byte deltas. Updated/paused/error/done events reset
        // the sample instead of leaving stale speed visible.
        set((state) => ({
          byId: { ...state.byId, [t.id]: t },
          rateById: {
            ...state.rateById,
            [t.id]: rateFromEvent(kind, t, state.rateById[t.id]),
          },
        }));
        // Surface terminal outcomes as toasts so background transfers aren't silent.
        if (kind === "done") {
          const verb = t.kind === "upload" ? "Uploaded" : "Downloaded";
          if (t.status === "skipped") {
            toast.info("Transfer skipped", baseName(t.source));
          } else {
            toast.success("Transfer complete", `${verb} ${baseName(t.source)}`);
          }
        } else if (kind === "error") {
          toast.error("Transfer failed", t.error || baseName(t.source));
        }
      });

      const unlistenQueue = await onTransferQueue((q) => {
        queueEventRevision++;
        set({
          queue: q.waiting,
          pausedAll: q.pausedAll,
          concurrency: q.concurrency,
          throttleKbps: q.throttleKbps,
        });
      });

      return () => {
        unlistenEvents?.();
        unlistenQueue();
      };
    } catch (error) {
      // If queue-listener registration fails after the transfer listener is
      // already live, release the partial registration before surfacing the
      // initialization failure.
      unlistenEvents?.();
      throw error;
    }
  },

  loadInitial: async () => {
    const transferRevisionAtStart = transferEventRevision;
    const queueRevisionAtStart = queueEventRevision;
    const [list, q] = await Promise.all([
      ipc.listTransfers(),
      ipc.transferQueueState(),
    ]);

    const initialById = indexBy(list);
    set((state) => {
      const transferChangedWhileLoading =
        transferEventRevision !== transferRevisionAtStart;
      const queueChangedWhileLoading = queueEventRevision !== queueRevisionAtStart;

      if (!transferChangedWhileLoading) {
        transferRateBaselines.clear();
      }

      return {
        // A live event observed after snapshot loading began is newer than the
        // snapshot. Preserve it while still importing transfers that existed
        // before listener registration completed.
        byId: transferChangedWhileLoading
          ? { ...initialById, ...state.byId }
          : initialById,
        rateById: transferChangedWhileLoading ? state.rateById : {},
        queue: queueChangedWhileLoading ? state.queue : q.waiting,
        pausedAll: queueChangedWhileLoading ? state.pausedAll : q.pausedAll,
        concurrency: queueChangedWhileLoading ? state.concurrency : q.concurrency,
        throttleKbps: queueChangedWhileLoading
          ? state.throttleKbps
          : q.throttleKbps,
      };
    });
  },

  download: async (sessionId, remotePath, localDir, policy) => {
    const p = policy ?? useSettings.getState().overwritePolicy;
    return ipc.startDownload(sessionId, remotePath, localDir, p);
  },

  upload: async (sessionId, localPath, remoteDir, policy) => {
    const p = policy ?? useSettings.getState().overwritePolicy;
    return ipc.startUpload(sessionId, localPath, remoteDir, p);
  },

  downloadDir: async (sessionId, remoteDir, localDir, policy) => {
    const p = policy ?? useSettings.getState().overwritePolicy;
    return ipc.startDirectoryDownload(sessionId, remoteDir, localDir, p);
  },

  uploadDir: async (sessionId, localDir, remoteDir, policy) => {
    const p = policy ?? useSettings.getState().overwritePolicy;
    return ipc.startDirectoryUpload(sessionId, localDir, remoteDir, p);
  },

  enqueueDownloads: async (sessionId, items, localDir) => {
    await runBatch(LOCAL_SESSION, localDir, "local", items, (item, policy) =>
      item.kind === "directory"
        ? get().downloadDir(sessionId, item.path, localDir, policy)
        : get().download(sessionId, item.path, localDir, policy)
    );
  },

  enqueueUploads: async (sessionId, items, remoteDir) => {
    await runBatch(sessionId, remoteDir, "remote", items, (item, policy) =>
      item.kind === "directory"
        ? get().uploadDir(sessionId, item.path, remoteDir, policy)
        : get().upload(sessionId, item.path, remoteDir, policy)
    );
  },

  cancel: async (id) => {
    await ipc.cancelTransfer(id);
  },

  pause: async (id) => {
    await ipc.transferPause(id);
  },

  resume: async (id) => {
    await ipc.transferResume(id);
  },

  retry: async (id) => {
    await ipc.transferRetry(id);
  },

  move: async (id, dir) => {
    await ipc.transferMove(id, dir);
  },

  pauseAll: async () => {
    await ipc.transferPauseAll();
  },

  resumeAll: async () => {
    await ipc.transferResumeAll();
  },

  setConcurrency: async (n) => {
    await ipc.transferSetConcurrency(n);
  },

  setThrottle: async (kbps) => {
    await ipc.transferSetThrottle(kbps);
  },

  clearFinished: () =>
    set((s) => {
      const next: Record<string, Transfer> = {};
      for (const t of Object.values(s.byId)) {
        // Paused rows are still in flight — keep them like active ones.
        if (
          t.status === "transferring" ||
          t.status === "queued" ||
          t.status === "paused"
        ) {
          next[t.id] = t;
        }
      }
      const rateById: Record<string, TransferRateSample> = {};
      for (const id of Object.keys(next)) {
        const sample = s.rateById[id];
        if (sample) rateById[id] = sample;
      }
      for (const id of [...transferRateBaselines.keys()]) {
        if (!next[id]) transferRateBaselines.delete(id);
      }
      return { byId: next, rateById };
    }),

  clearCompleted: () =>
    set((s) => {
      const next: Record<string, Transfer> = {};
      for (const t of Object.values(s.byId)) {
        if (t.status !== "done" && t.status !== "skipped" && t.status !== "canceled") {
          next[t.id] = t;
        }
      }
      const rateById: Record<string, TransferRateSample> = {};
      for (const id of Object.keys(next)) {
        const sample = s.rateById[id];
        if (sample) rateById[id] = sample;
      }
      for (const id of [...transferRateBaselines.keys()]) {
        if (!next[id]) transferRateBaselines.delete(id);
      }
      return { byId: next, rateById };
    }),
}));
