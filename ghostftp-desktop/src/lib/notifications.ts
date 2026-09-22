// Desktop notifications (Plan 16 Phase 3). A small, curated set of OS toasts for
// events that matter when Ghost FTP isn't focused: a transfer batch draining the
// active queue, a folder-sync pair entering its error state, and an
// edit-in-place save failing (the user may have Ghost FTP hidden while editing).
//
// The events are decided and gated *here* rather than in the Rust backend
// because the frontend already tracks all of this state (the transfer stream,
// the sync store, the editor error event) and can cheaply check window focus.
// Everything is behind the `notifications` setting (opt-in, unfocused-only),
// permission is never requested by a background event; Preferences owns the explicit opt-in, and clicking a
// toast focuses the window (and, where it applies, opens the relevant panel).
import {
  isPermissionGranted,
  requestPermission,
  sendNotification,
  onAction,
} from "@tauri-apps/plugin-notification";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useSettings } from "@/stores/settingsStore";
import { useSync } from "@/stores/syncStore";
import { onTransferEvent, onEditError } from "@/lib/ipc";
import { useLayout } from "@/stores/layoutStore";
import { useTransfers } from "@/stores/transfersStore";

type Permission = "unknown" | "granted" | "denied";
let permission: Permission = "unknown";

/** Passive permission probe used by background notification events. */
async function ensurePermission(): Promise<boolean> {
  if (permission === "granted") return true;
  if (permission === "denied") return false;
  try {
    const granted = await isPermissionGranted();
    permission = granted ? "granted" : "denied";
    return granted;
  } catch {
    permission = "denied";
    return false;
  }
}

/** Explicit Preferences action. This is the only path allowed to trigger an OS permission prompt. */
export async function requestDesktopNotificationPermission(): Promise<boolean> {
  try {
    let granted = await isPermissionGranted();
    if (!granted) granted = (await requestPermission()) === "granted";
    permission = granted ? "granted" : "denied";
    return granted;
  } catch {
    permission = "denied";
    return false;
  }
}

async function windowFocused(): Promise<boolean> {
  try {
    return await getCurrentWindow().isFocused();
  } catch {
    // Fall back to the DOM's own idea of focus if the window API is unavailable.
    try {
      return document.hasFocus();
    } catch {
      return true; // assume focused → suppress, the safe default
    }
  }
}

// The panel to open if the just-sent notification is clicked. Notifications are
// rare and the user clicks the one they just saw, so tracking the latest route
// is enough (and robust across the platforms' varied toast-click support).
let lastRoute: (() => void) | null = null;

async function notify(title: string, body: string, route?: () => void): Promise<boolean> {
  const s = useSettings.getState().notifications;
  if (!s.enabled) return false;
  if (s.unfocusedOnly && (await windowFocused())) return false;
  if (!(await ensurePermission())) return false;
  lastRoute = route ?? null;
  try {
    sendNotification({ title, body });
    return true;
  } catch {
    return false;
  }
}

/** Wire the curated events. Returns a cleanup that removes every listener. */
export function initNotifications(): () => void {
  const cleanups: Array<() => void> = [];
  // The event subscriptions resolve asynchronously, but React (StrictMode in the
  // main window) can run cleanup *before* they resolve. Guard with a flag so a
  // subscription that lands after teardown immediately unsubscribes — otherwise
  // the double-mount leaks a listener and every toast fires twice.
  let cancelled = false;
  const track = (p: Promise<() => void>) => {
    void p
      .then((un) => {
        if (cancelled) un();
        else cleanups.push(un);
      })
      .catch((error) => {
        console.warn("Couldn't attach a desktop-notification event listener", error);
      });
  };

  // ---- Transfer batches: notify when the active queue drains ----
  // Track in-flight transfer ids; when the last one finishes, summarise the
  // batch (done / failed counts) as a single toast — never one-per-file.
  const active = new Set<string>();
  let batchTotal = 0;
  let batchDone = 0;
  let batchFailed = 0;

  const flushBatch = () => {
    if (batchTotal === 0) return;
    const done = batchDone;
    const failed = batchFailed;
    batchTotal = batchDone = batchFailed = 0;
    const route = () => {
      try {
        useTransfers.getState().setPanelOpen(true);
      } catch (error) {
        console.warn("Couldn't route a transfer notification back to the queue", error);
      }
    };
    if (failed > 0) {
      void notify(
        "Transfers finished with errors",
        `${done} done · ${failed} failed`,
        route
      );
    } else {
      void notify(
        "Transfers complete",
        `${done} file${done === 1 ? "" : "s"} transferred`,
        route
      );
    }
  };

  track(onTransferEvent((kind, t) => {
    const id = t.id;
    switch (kind) {
      case "added":
      case "progress":
        // `added` is the entry point; `progress` also catches a transfer whose
        // `added` we missed (listener attached mid-flight).
        if (!active.has(id)) {
          active.add(id);
          batchTotal++;
        }
        break;
      case "done":
        if (active.delete(id)) {
          batchDone++;
          if (active.size === 0) flushBatch();
        }
        break;
      case "error":
        if (active.delete(id)) {
          batchFailed++;
          if (active.size === 0) flushBatch();
        }
        break;
      case "updated":
        // Canceled rows emit no terminal event — without this they'd wedge the
        // batch forever. A paused row is still in flight, so it stays in
        // `active` (it emits nothing until resumed).
        if (t.status === "canceled" && active.delete(id)) {
          if (active.size === 0) flushBatch();
        }
        break;
    }
  }));

  // ---- Edit-in-place save failures ----
  track(onEditError((e) => {
    const name = e.remotePath.split(/[\\/]/).pop() || e.remotePath;
    void notify("Couldn't save edit", `${name}: ${e.message}`);
  }));

  // ---- Folder-sync pairs entering their error state ----
  // Diff the sync store's pair list on every change; notify on a transition
  // *into* "error" (not while it stays there) so we don't repeat.
  const prevErr = new Map<string, boolean>();
  // Seed with the current state so a pair already in error at boot doesn't
  // fire a stale toast.
  for (const p of useSync.getState().pairs ?? []) prevErr.set(p.id, p.state === "error");
  const unsubSync = useSync.subscribe((state) => {
    for (const p of state.pairs ?? []) {
      const was = prevErr.get(p.id) ?? false;
      const now = p.state === "error";
      if (now && !was) {
        void notify(
          "Folder sync error",
          `${p.name}: ${p.lastError ?? "sync failed"}`,
          () => {
            try {
              useLayout.getState().openDialog("settings");
            } catch (error) {
              console.warn("Couldn't route a sync notification to Preferences", error);
            }
          }
        );
      }
      prevErr.set(p.id, now);
    }
  });
  cleanups.push(unsubSync);

  // ---- Click-to-focus (best-effort; toast-click support varies by OS) ----
  void onAction(() => {
    void getCurrentWindow().setFocus().catch((error) => {
      console.warn("Couldn't focus Ghost FTP from a notification action", error);
    });
    try {
      lastRoute?.();
    } catch (error) {
      console.warn("Couldn't run the notification route action", error);
    }
  })
    .then((listener) => {
      const un = () => {
        void listener.unregister().catch((error) =>
          console.warn("Couldn't unregister the notification action listener", error)
        );
      };
      if (cancelled) un();
      else cleanups.push(un);
    })
    .catch((error) => {
      console.warn("Notification action callbacks are unavailable on this platform", error);
    });

  return () => {
    cancelled = true;
    for (const c of cleanups) {
      try {
        c();
      } catch (error) {
        console.warn("Couldn't clean up a desktop-notification listener", error);
      }
    }
  };
}
