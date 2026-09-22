import { create } from "zustand";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";

// In-app auto-updater (Plan 16 Phase 1/2). Drives the Tauri updater plugin from
// the frontend: check a signed Ghost FTP release service `latest.json`, download the
// signed artifact (verified against the pubkey in tauri.conf.json), and relaunch
// to apply. The signature check is the plugin's, not ours — the app never
// installs an unverified update.
//
// `heldUpdate` keeps the (non-serializable) plugin Update object between the
// check and the download; only plain, renderable fields live in the store.

let heldUpdate: Update | null = null;

/** How long to wait between quiet launch checks (persisted via `lastUpdateCheck`
 *  in ghostftp.db so it survives restarts, per the plan). */
const CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000; // 6 hours

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
  /** Session-only: hide the launch prompt without changing anything persisted. */
  dismissed: boolean;

  /** Check now. `quiet` swallows the "up to date" toast + any endpoint error
   *  (used for the throttled launch check); a manual check surfaces both. */
  check: (quiet: boolean) => Promise<void>;
  downloadAndInstall: () => Promise<void>;
  restart: () => Promise<void>;
  dismiss: () => void;
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
  dismissed: false,

  check: async (quiet) => {
    if (get().status === "checking" || get().status === "downloading") return;
    set({ status: "checking", error: null });
    // Record the check time regardless of outcome so we don't hammer the endpoint.
    void ipc.settingsSet("lastUpdateCheck", JSON.stringify(Date.now())).catch((error) => {
      if (quiet) console.warn("Couldn't persist update-check time", error);
      else toast.warning("Update check will not be remembered", String(error));
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
          dismissed: false,
        });
      } else {
        heldUpdate = null;
        set({ status: "idle", version: null, notes: null });
        if (!quiet) toast.success("Ghost FTP is up to date");
      }
    } catch (e) {
      heldUpdate = null;
      // A missing/unbuilt latest.json (404) is expected until the signed release
      // pipeline is live — never nag on launch. Manual checks show the error.
      if (quiet) {
        set({ status: "idle" });
      } else {
        set({ status: "error", error: String(e) });
        toast.error(`Update check failed: ${e}`);
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
      set({ status: "ready" });
    } catch (e) {
      set({ status: "error", error: String(e) });
      toast.error(`Update download failed: ${e}`);
    }
  },

  restart: async () => {
    try {
      await relaunch();
    } catch (e) {
      toast.error(`Couldn't restart: ${e}`);
    }
  },

  dismiss: () => set({ dismissed: true }),

  init: async () => {
    try {
      const all = await ipc.settingsGetAll();
      const raw = all["lastUpdateCheck"];
      const last = raw ? Number(JSON.parse(raw)) : 0;
      if (!Number.isFinite(last) || Date.now() - last > CHECK_INTERVAL_MS) {
        await get().check(true);
      }
    } catch {
      // No backend (mock) — skip the launch check.
    }
    return () => {};
  },
}));
