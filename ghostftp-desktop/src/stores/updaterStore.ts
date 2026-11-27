import { create } from "zustand";
import { relaunch } from "@tauri-apps/plugin-process";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";
import { messageOf } from "@/lib/errors";
import { redactSensitiveText } from "@/lib/redact";

// In-app auto-updater. The persistent application shell performs a throttled,
// quiet launch check; Help & About → Updates owns all user-facing update actions.
// The native updater verifies each offered package before installation, while
// the process plugin performs the explicit restart. Transport details are never
// surfaced in user-facing messages.
//
// Only the selected version is held in the UI. Native Rust rechecks the offer.
let heldVersion: string | null = null;
let installationUncertain = false;
let restarting = false;

// A fresh native snapshot is required; a missing snapshot is not an empty queue.
// Native exclusive admission is acquired separately before installation.
async function transfersPermitUpdate(): Promise<boolean> {
  try {
    const [transfers, queue] = await Promise.all([ipc.listTransfers(), ipc.transferQueueState()]);
    const terminal = new Set(["done", "skipped", "error", "canceled"]);
    if (queue.waiting.length !== 0 || transfers.some((transfer) => !terminal.has(transfer.status))) {
      toast.warning("Finish transfers before updating", "Complete or cancel queued, running and paused transfers before continuing.");
      return false;
    }
    return true;
  } catch (error) {
    console.warn("Couldn't verify transfer activity", redactSensitiveText(error, 240));
    toast.warning("Update postponed", "Ghost FTP couldn't verify that the transfer queue is idle. Try again after checking Transfers.");
    return false;
  }
}

/** How long to wait between quiet launch checks (persisted via `lastUpdateCheck`
 *  in ghostftp.db so it survives restarts, per the plan). */
const CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000; // 6 hours

function friendlyUpdateError(error: unknown, phase: "check" | "download"): string {
  const detail = messageOf(error);
  console.warn(`Ghost FTP updater ${phase} detail:`, redactSensitiveText(detail, 240));
  const normalized = detail.toLowerCase();

  if (
    normalized.includes("signature") ||
    normalized.includes("verify") ||
    normalized.includes("public key") ||
    normalized.includes("tamper")
  ) {
    return "Ghost FTP couldn't verify the update package. Nothing was installed.";
  }
  if (
    normalized.includes("network") ||
    normalized.includes("http") ||
    normalized.includes("request") ||
    normalized.includes("connect") ||
    normalized.includes("timeout") ||
    normalized.includes("404") ||
    normalized.includes("dns")
  ) {
    return "Ghost FTP couldn't reach the update service. Check your internet connection and try again.";
  }
  return phase === "download"
    ? "Ghost FTP couldn't finish the update. Check your installation before retrying."
    : "Ghost FTP couldn't complete the update check. Please try again later.";
}

export type UpdaterStatus =
  | "idle" // check has not run yet
  | "checked" // no signed in-app updater package offered (manual downloads may be newer)
  | "checking"
  | "available"
  | "downloading"
  | "ready" // downloaded + installed; awaiting restart
  | "error";

interface UpdaterState {
  status: UpdaterStatus;
  /** The offered version (when an update is available). */
  version: string | null;
  currentVersion: string | null;
  /** Newer manually installable version; never an automatic-install offer. */
  manualVersion: string | null;
  recoveryNotice: string | null;
  notes: string | null;
  /** Download progress, bytes. `total` is null until the Started event. */
  downloaded: number;
  total: number | null;
  error: string | null;
  /** Check now. `quiet` swallows the "up to date" toast + any endpoint error
   *  (used for the throttled launch check); a manual check surfaces both. */
  check: (quiet: boolean) => Promise<void>;
  downloadAndInstall: () => Promise<void>;
  restart: () => Promise<void>;
  /** Throttled launch check. Returns a no-op cleanup so it slots into the same
   *  mount pattern as the other startup stores. */
  init: () => Promise<() => void>;
}

export const useUpdater = create<UpdaterState>((set, get) => ({
  status: "idle",
  version: null,
  currentVersion: null,
  manualVersion: null,
  recoveryNotice: null,
  notes: null,
  downloaded: 0,
  total: null,
  error: null,

  check: async (quiet) => {
    if (installationUncertain || get().status === "checking" || get().status === "downloading" || get().status === "ready") return;
    set({ status: "checking", error: null, manualVersion: null });
    // Record the check time regardless of outcome so we don't hammer the endpoint.
    void ipc.settingsSet("lastUpdateCheck", JSON.stringify(Date.now())).catch((error) => {
      if (quiet) console.warn("Couldn't persist update-check time", redactSensitiveText(error, 240));
      else toast.warning("Update check will not be remembered", redactSensitiveText(error, 240));
    });
    // Only explicit user checks query manual availability; signed launch checks
    // retain their existing throttling and trust boundary.
    if (!quiet) {
      try {
        const manualVersion = await ipc.checkManualUpdate();
        set({ manualVersion });
      } catch (error) {
        console.warn("Manual update availability unknown", redactSensitiveText(error, 240));
      }
    }
    try {
      const update = await ipc.checkSignedUpdate();
      if (update) {
        heldVersion = update.version;
        set({
          status: "available",
          version: update.version,
          currentVersion: update.currentVersion,
          notes: update.body ?? null,
        });
      } else {
        heldVersion = null;
        // A missing signed updater offer says nothing about newer unsigned
        // Windows/Linux packages published on GitHub. Never tell the user
        // they are up to date based solely on the signed-updater endpoint.
        set({ status: "checked", version: null, notes: null });
        if (!quiet) toast.info(
          "No signed automatic update available",
          "Check official Downloads for newer manually installable Windows/Linux editions."
        );
      }
    } catch (e) {
      heldVersion = null;
      // Quiet launch checks never nag. Manual checks show only a safe,
      // user-facing summary; technical transport details stay in the console.
      if (quiet) {
        console.warn("Quiet update check failed", redactSensitiveText(e, 240));
        set({ status: "idle" });
      } else {
        const message = friendlyUpdateError(e, "check");
        set({ status: "error", error: message });
        toast.error("Update check failed", message);
      }
    }
  },

  downloadAndInstall: async () => {
    if (!heldVersion || installationUncertain || get().status !== "available") return;
    const target = heldVersion;
    // Reserve the operation synchronously, including while awaiting preflight.
    set({ status: "downloading", downloaded: 0, total: null, error: null });
    if (!(await transfersPermitUpdate())) {
      set({ status: "available" });
      return;
    }
    try {
      await ipc.installSignedUpdate(target, (progress) => {
        set({ downloaded: progress.downloaded, total: progress.total });
      });
      heldVersion = null;
      set({ status: "ready" });
    } catch (error) {
      console.warn("Native update failed", redactSensitiveText(error, 240));
      const phase = typeof error === "object" && error !== null && "phase" in error ? error.phase : null;
      if (phase === "prepare" || phase === "download") {
        set({ status: "available", downloaded: 0, total: null });
        toast.warning("Update postponed", phase === "prepare"
          ? "Finish transfers, close remote editors, disable folder sync and complete pending backup restores before updating."
          : "Ghost FTP couldn't download or verify the update. Try again.");
        return;
      }
      heldVersion = null;
      installationUncertain = phase !== "check";
      const message = installationUncertain
        ? "Installation status is uncertain. Close and reopen Ghost FTP before starting transfers or retrying."
        : "The update offer changed or is unavailable. Check again.";
      set({ status: "error", error: message, downloaded: 0, total: null });
      toast.error("Update failed", message);
    }
  },

  restart: async () => {
    if (get().status !== "ready" || restarting) return;
    restarting = true;
    try {
      if (!(await transfersPermitUpdate())) return;
      await relaunch();
    } catch (e) {
      console.warn("Updater restart failed", redactSensitiveText(e, 240));
      toast.error("Couldn't restart", "Close and reopen Ghost FTP when all transfers are finished.");
    } finally {
      restarting = false;
    }
  },

  init: async () => {
    try {
      set({ recoveryNotice: await ipc.updateRecoveryNotice() });
    } catch (error) {
      console.warn("Update recovery status unavailable", redactSensitiveText(error, 240));
      set({ recoveryNotice: "Ghost FTP couldn't read the previous update status. Check your installation and backups before updating." });
    }
    try {
      const all = await ipc.settingsGetAll();
      const raw = all["lastUpdateCheck"];
      const last = raw ? Number(JSON.parse(raw)) : 0;
      if (!Number.isFinite(last) || Date.now() - last > CHECK_INTERVAL_MS) {
        await get().check(true);
      }
    } catch (error) {
      console.warn("Couldn't initialize the quiet update check", redactSensitiveText(error, 240));
    }
    return () => {};
  },
}));
