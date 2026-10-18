/**
 * One guarded async boundary for the native save dialog AND the export IPC.
 * A rejected OS picker must not escape a fire-and-forget React button handler
 * as an unhandled promise rejection.
 */
export type TransferExportOutcome = "saved" | "cancelled" | "failed";

export async function runTransferHistoryExport(
  chooseDestination: () => Promise<string | null>,
  exportToDestination: (path: string) => Promise<number>,
  onSuccess: (count: number) => void,
  onFailure: (error: unknown) => void,
): Promise<TransferExportOutcome> {
  try {
    const destination = await chooseDestination();
    if (!destination) return "cancelled";
    const count = await exportToDestination(destination);
    onSuccess(count);
    return "saved";
  } catch (error) {
    onFailure(error);
    return "failed";
  }
}
