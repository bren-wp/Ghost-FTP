// Ghost FTP by Brendigo
import type { Transfer } from "./types";

/**
 * Pure Transfer Center view state. This module must never mutate backend
 * transfers, confirm completion, retry a job, or write the persisted queue.
 * The native transfer engine remains the only source of transfer truth.
 */
export type TransferStatusTab = "all" | "active" | "completed" | "failed";
export type TransferDirectionFilter = "all" | "upload" | "download";
export type TransferTimeFilter = "all" | "hour" | "day" | "week" | "month";

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
  const normalized = options.query.trim().toLowerCase();
  // Ghost FTP by Brendigo: fixed rolling windows, independent of locale/DST.
  const windowSeconds = {
    all: null,
    hour: 3_600,
    day: 86_400,
    week: 7 * 86_400,
    month: 30 * 86_400,
  }[options.time];

  return transfers.filter((transfer) => {
    if (options.tab === "active" && !isActive(transfer.status)) return false;
    if (options.tab === "completed" && !isCompleted(transfer.status)) return false;
    if (options.tab === "failed" && transfer.status !== "error") return false;

    if (options.direction !== "all" && transfer.kind !== options.direction) return false;
    if (windowSeconds !== null) {
      // Server clocks, imported history and corrupt timestamps must never
      // make future-dated transfers appear in rolling history filters.
      if (
        !Number.isFinite(options.nowSeconds) ||
        !Number.isFinite(transfer.startedAt) ||
        transfer.startedAt > options.nowSeconds ||
        transfer.startedAt < options.nowSeconds - windowSeconds
      ) return false;
    }

    if (!normalized) return true;
    return `${transfer.source} ${transfer.destination} ${transfer.status}`
      .toLowerCase()
      .includes(normalized);
  });
}

/** Return a sorted copy; never reorder the live queue while showing its log. */
export function recentTransferLog(
  transfers: readonly Transfer[],
  clearedAtSeconds: number,
  limit = 12,
): Transfer[] {
  // Render just the newest k rows, not a full copy and O(n log n) sort
  // of all persisted transfers on every progress update. Default k is 12.
  // Strict timestamp comparisons preserve original input order for ties.
  const capacity = Number.isFinite(limit)
    ? Math.min(transfers.length, Math.max(0, Math.floor(limit)))
    : 0;
  if (capacity === 0 || !Number.isFinite(clearedAtSeconds)) return [];
  const newest: Transfer[] = [];
  for (const transfer of transfers) {
    if (!Number.isFinite(transfer.startedAt) || transfer.startedAt < 0 ||
        transfer.startedAt <= clearedAtSeconds) continue;
    let at = newest.findIndex((item) => item.startedAt < transfer.startedAt);
    if (at < 0) at = newest.length;
    if (at >= capacity) continue;
    newest.splice(at, 0, transfer);
    if (newest.length > capacity) newest.pop();
  }
  return newest;
}

/** View-only sorting: never reprioritize persisted/native transfer work. */
export type TransferSortOrder = "original" | "newest" | "oldest" | "largest" | "name";

export function sortTransferView(
  transfers: readonly Transfer[],
  order: TransferSortOrder,
): Transfer[] {
  if (order === "original") return [...transfers];
  const sortable = transfers.map((transfer, index) => ({ transfer, index }));
  if (order === "name") {
    const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });
    // Extract each basename once; sorting a large history must not repeatedly
    // split both file paths on every comparator invocation.
    const named = sortable.map((item) => ({
      ...item,
      name: String(item.transfer.source ?? "").split(/[/\\]/).filter(Boolean).pop() ?? "",
    }));
    named.sort((a, b) => collator.compare(a.name, b.name) || a.index - b.index);
    return named.map(({ transfer }) => transfer);
  }
  const ranked = sortable.map((item) => {
    const value = order === "largest" ? item.transfer.size : item.transfer.startedAt;
    return { ...item, value: Number.isFinite(value) && value >= 0 ? value : null };
  });
  ranked.sort((a, b) => {
    if (a.value === null && b.value !== null) return 1;
    if (b.value === null && a.value !== null) return -1;
    if (a.value === null || b.value === null) return a.index - b.index;
    const cmp = order === "oldest" ? a.value - b.value : b.value - a.value;
    return cmp || a.index - b.index;
  });
  return ranked.map(({ transfer }) => transfer);
}
