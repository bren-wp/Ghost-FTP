import { create } from "zustand";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";
import { messageOf } from "@/lib/errors";

// In-app auto-updater. The persistent application shell performs a throttled,
// quiet launch check; Help & About → Updates owns all user-facing update actions.
// The native updater verifies each offered package before installation, while
// the process plugin performs the explicit restart. Transport details are never
// surfaced in user-facing messages.
//
// `heldUpdate` keeps the non-serializable plugin Update object between the
// check and download; only plain, renderable fields live in the store.

let heldUpdate: Update | null = null;

/** How long to wait between quiet launch checks (persisted via `lastUpdateCheck`
 *  in ghostftp.db so it survives restarts, per the plan). */
const CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000; // 6 hours

function friendlyUpdateError(error: unknown, phase: "check" | "download"): string {
  const detail = messageOf(error);
  console.warn(`Ghost FTP updater ${phase} detail:`, detail);
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
    ? "Ghost FTP couldn't install the update. Your current installation was left unchanged."
    : "Ghost FTP couldn't complete the update check. Please try again later.";
}

export type UpdaterStatus =
  | "idle" // no check run yet, or up to date
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
  notes: null,
  downloaded: 0,
  total: null,
  error: null,

  check: async (quiet) => {
    if (get().status === "checking" || get().status === "downloading") return;
    set({ status: "checking", error: null });
    // Record the check time regardless of outcome so we don't hammer the endpoint.
    void ipc.settingsSet("lastUpdateCheck", JSON.stringify(Date.now())).catch((error) => {
      if (quiet) console.warn("Couldn't persist update-check time", messageOf(error));
      else toast.warning("Update check will not be remembered", messageOf(error));
    });
    try {
      const update = await check();
      if (update && update.available) {
        heldUpdate = update;
        set({
          status: "available",
          version: update.version,
          currentVersion: update.currentVersion,
          notes: update.body ?? null,
        });
      } else {
        heldUpdate = null;
        set({ status: "idle", version: null, notes: null });
        if (!quiet) toast.success("Ghost FTP is up to date");
      }
    } catch (e) {
      heldUpdate = null;
      // Quiet launch checks never nag. Manual checks show only a safe,
      // user-facing summary; technical transport details stay in the console.
      if (quiet) {
        console.warn("Quiet update check failed", messageOf(e));
        set({ status: "idle" });
      } else {
        const message = friendlyUpdateError(e, "check");
        set({ status: "error", error: message });
        toast.error("Update check failed", message);
      }
    }
  },

  downloadAndInstall: async () => {
    if (!heldUpdate || get().status === "downloading") return;
    set({ status: "downloading", downloaded: 0, total: null, error: null });
    try {
      await heldUpdate.downloadAndInstall((event) => {
        switch (event.event) {
          case "Started":
            set({ total: event.data.contentLength ?? null, downloaded: 0 });
            break;
          case "Progress":
            set((s) => ({ downloaded: s.downloaded + event.data.chunkLength }));
            break;
          case "Finished":
            break;
        }
      });
      heldUpdate = null;
      set({ status: "ready" });
    } catch (e) {
      const message = friendlyUpdateError(e, "download");
      // Keep the verified Update handle so a transient network/install error
      // can be retried without discarding the already offered release.
      set({
        status: heldUpdate ? "available" : "error",
        error: message,
        downloaded: 0,
        total: null,
      });
      toast.error("Update download failed", message);
    }
  },

  restart: async () => {
    try {
      await relaunch();
    } catch (e) {
      toast.error("Couldn't restart", messageOf(e));
    }
  },

  init: async () => {
    try {
      const all = await ipc.settingsGetAll();
      const raw = all["lastUpdateCheck"];
      const last = raw ? Number(JSON.parse(raw)) : 0;
      if (!Number.isFinite(last) || Date.now() - last > CHECK_INTERVAL_MS) {
        await get().check(true);
      }
    } catch (error) {
      console.warn("Couldn't initialize the quiet update check", messageOf(error));
    }
    return () => {};
  },
}));
