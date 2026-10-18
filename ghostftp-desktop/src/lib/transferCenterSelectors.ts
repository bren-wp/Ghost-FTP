import type { Transfer, TransferStatus } from "./types";

/**
 * Pure presentation selectors for the transfer queue.
 *
 * The native backend remains authoritative for transfer state, ordering,
 * errors and retries. No counters or status are persisted in this module.
 */
export type TransferFilterTab = "all" | "active" | "completed" | "failed";
export type TransferDirectionFilter = "all" | Transfer["kind"];
export type TransferTimeFilter = "all" | "hour" | "day";
type TransferBucket = Exclude<TransferFilterTab, "all"> | "other";

export interface TransferFilters {
  tab: TransferFilterTab;
  query: string;
  direction: TransferDirectionFilter;
  time: TransferTimeFilter;
  /** Unix seconds, captured once by the caller for a consistent filter pass. */
  nowSeconds: number;
}

export interface TransferCounts {
  total: number;
  active: number;
  completed: number;
  failed: number;
}

/** Keep counters and tabs on the same status policy. Canceled is not failed. */
export function transferBucket(status: TransferStatus): TransferBucket {
  switch (status) {
    case "queued":
    case "transferring":
    case "paused":
      return "active";
    case "done":
    case "skipped":
      return "completed";
    case "error":
      return "failed";
    default:
      return "other";
  }
}

export function summarizeTransfers(transfers: readonly Transfer[]): TransferCounts {
  const counts: TransferCounts = { total: 0, active: 0, completed: 0, failed: 0 };
  for (const transfer of transfers) {
    counts.total++;
    const bucket = transferBucket(transfer.status);
    if (bucket !== "other") counts[bucket]++;
  }
  return counts;
}

/** Retains backend order and object identity; never mutates transfer records. */
export function filterTransfers(
  transfers: readonly Transfer[],
  filters: TransferFilters,
): Transfer[] {
  const query = filters.query.trim().toLowerCase();
  const cutoff = filters.time === "hour"
    ? filters.nowSeconds - 3600
    : filters.time === "day"
      ? filters.nowSeconds - 86400
      : null;

  return transfers.filter((transfer) => {
    if (filters.tab !== "all" && transferBucket(transfer.status) !== filters.tab) {
      return false;
    }
    if (filters.direction !== "all" && transfer.kind !== filters.direction) {
      return false;
    }
    if (cutoff !== null &&
      (!Number.isFinite(transfer.startedAt) || transfer.startedAt < cutoff)) {
      return false;
    }
    return !query ||
      `${transfer.source} ${transfer.destination} ${transfer.status}`
        .toLowerCase()
        .includes(query);
  });
}
