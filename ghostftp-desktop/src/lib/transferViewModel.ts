import type { Transfer } from "./types";

/**
 * Pure Transfer Center view state. This module must never mutate backend
 * transfers, confirm completion, retry a job, or write the persisted queue.
 * The native transfer engine remains the only source of transfer truth.
 */
export type TransferStatusTab = "all" | "active" | "completed" | "failed";
export type TransferDirectionFilter = "all" | "upload" | "download";
export type TransferTimeFilter = "all" | "hour" | "day";

export interface TransferViewFilters {
  tab: TransferStatusTab;
  query: string;
  direction: TransferDirectionFilter;
  time: TransferTimeFilter;
  nowSeconds: number;
}

export interface TransferCounts {
  completed: number;
  failed: number;
  active: number;
}

function isActive(status: Transfer["status"]): boolean {
  return status === "transferring" || status === "queued" || status === "paused";
}

function isCompleted(status: Transfer["status"]): boolean {
  return status === "done" || status === "skipped";
}

export function countTransferStatuses(transfers: readonly Transfer[]): TransferCounts {
  const totals: TransferCounts = { completed: 0, failed: 0, active: 0 };
  for (const transfer of transfers) {
    if (isCompleted(transfer.status)) totals.completed += 1;
    else if (transfer.status === "error") totals.failed += 1;
    else if (isActive(transfer.status)) totals.active += 1;
  }
  return totals;
}

/** Preserve input ordering and reference identity, including backend retries. */
export function filterTransferView(
  transfers: readonly Transfer[],
  options: TransferViewFilters,
): Transfer[] {
  const normalized = options.query.trim().toLocaleLowerCase();
  const windowSeconds = options.time === "hour" ? 3_600 : options.time === "day" ? 86_400 : null;

  return transfers.filter((transfer) => {
    if (options.tab === "active" && !isActive(transfer.status)) return false;
    if (options.tab === "completed" && !isCompleted(transfer.status)) return false;
    if (options.tab === "failed" && transfer.status !== "error") return false;

    if (options.direction !== "all" && transfer.kind !== options.direction) return false;
    if (windowSeconds !== null && transfer.startedAt < options.nowSeconds - windowSeconds) return false;

    if (!normalized) return true;
    return `${transfer.source} ${transfer.destination} ${transfer.status}`
      .toLocaleLowerCase()
      .includes(normalized);
  });
}

/** Return a sorted copy; never reorder the live queue while showing its log. */
export function recentTransferLog(
  transfers: readonly Transfer[],
  clearedAtSeconds: number,
  limit = 12,
): Transfer[] {
  return transfers
    .filter((transfer) => transfer.startedAt > clearedAtSeconds)
    .sort((a, b) => b.startedAt - a.startedAt)
    .slice(0, Math.max(0, Math.floor(limit)));
}
