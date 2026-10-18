import type { Transfer } from "./types";

/**
 * A failure shortcut must select a real native queue entry, not an inferred
 * profile or an optimistic UI-only error. Keep the backend's record identity.
 */
export function mostRecentFailedTransferId(
  transfers: readonly Pick<Transfer, "id" | "status" | "startedAt">[],
): string | null {
  let id: string | null = null;
  let newest = -Infinity;
  for (const transfer of transfers) {
    if (transfer.status !== "error") continue;
    const stamp = Number.isFinite(transfer.startedAt) ? transfer.startedAt : -Infinity;
    // Keep the first error as a fallback if all timestamps are corrupt.
    // Equal timestamps retain native insertion order.
    if (id === null || stamp > newest) {
      id = transfer.id;
      newest = stamp;
    }
  }
  return id;
}
