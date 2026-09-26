import { useEffect, useState } from "react";
import {
  Info,
  Loader2,
  CheckCircle2,
  TerminalSquare,
  Plus,
  Trash2,
  Bell,
  RefreshCw,
  ArrowUpCircle,
  Download,
} from "lucide-react";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { useSettings } from "@/stores/settingsStore";
import { useUpdater } from "@/stores/updaterStore";
import type { PathStatus } from "@/lib/types";
import { getVersion } from "@tauri-apps/api/app";
import { cn } from "@/lib/cn";

const RELEASES_URL = "https://ghostftp.com/download";

/// Settings → About. The app's identity/version, plus the trust/UX fundamentals
/// from Plan 16: the one-click "Add ghostftp-cli to PATH" row (Phase 4). The updater
/// card (Phase 2) and notification toggles (Phase 3) join this section as they land.
export function AboutSettings() {
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const v = await getVersion();
        if (!cancelled) setVersion(v);
      } catch {
        /* not in a Tauri context (mock/browser) — just omit the version */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h3 className="mb-1 flex items-center gap-2 text-sm font-medium text-text">
          <Info size={15} className="text-accent" />
          Ghost FTP{version ? ` ${version}` : ""}
        </h3>
        <p className="text-xs text-text-muted">
          A modern client for SFTP · FTP · S3 · Azure — plus the Agent Bridge.
        </p>
      </div>

      <UpdaterCard />

      <NotificationsRow />

      <PathRow />
    </div>
  );
}

/// In-app updater (Plan 16 Phase 1/2). Manual "Check for updates", the current
/// status, and the download → restart flow. The launch check + the prompt bar
/// live in UpdatePrompt; this is the on-demand surface.
function UpdaterCard() {
  const status = useUpdater((s) => s.status);
  const version = useUpdater((s) => s.version);
  const downloaded = useUpdater((s) => s.downloaded);
  const total = useUpdater((s) => s.total);
  const error = useUpdater((s) => s.error);
  const check = useUpdater((s) => s.check);
  const download = useUpdater((s) => s.downloadAndInstall);
  const restart = useUpdater((s) => s.restart);

  const busy = status === "checking" || status === "downloading";
  const pct =
    total && total > 0 ? Math.min(100, Math.round((downloaded / total) * 100)) : null;

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2 text-sm font-medium text-text">
        <ArrowUpCircle size={15} className="text-accent" />
        Updates
      </div>

      <div className="rounded-lg border border-border bg-bg-subtle/50 p-3 text-sm">
        {status === "available" || status === "downloading" || status === "ready" ? (
          <div className="flex items-start gap-2">
            <ArrowUpCircle size={15} className="mt-0.5 shrink-0 text-accent" />
            <div className="min-w-0">
              {status === "available" && (
                <div className="text-text">Ghost FTP {version} is available.</div>
              )}
              {status === "downloading" && (
                <div className="text-text">
                  Downloading update{pct !== null ? ` — ${pct}%` : "…"}
                </div>
              )}
              {status === "ready" && (
                <div className="text-text">
                  Update downloaded — restart Ghost FTP to finish installing.
                </div>
              )}
            </div>
          </div>
        ) : status === "error" ? (
          <div className="text-text-muted">{error ?? "Update check failed."}</div>
        ) : status === "checking" ? (
          <div className="flex items-center gap-2 text-text-muted">
            <Loader2 size={15} className="animate-spin" /> Checking…
          </div>
        ) : (
          <div className="flex items-start gap-2">
            <CheckCircle2 size={15} className="mt-0.5 shrink-0 text-emerald-500" />
            <div className="text-text">You're on the latest version.</div>
          </div>
        )}

        <div className="mt-3 flex flex-wrap items-center gap-2">
          {status === "available" ? (
            <button
              onClick={() => void download()}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90"
            >
              <Download size={13} /> Download &amp; install
            </button>
          ) : status === "ready" ? (
            <button
              onClick={() => void restart()}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90"
            >
              <RefreshCw size={13} /> Restart now
            </button>
          ) : (
            <button
              onClick={() => void check(false)}
              disabled={busy}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90 disabled:opacity-60"
            >
              {busy ? <Loader2 size={13} className="animate-spin" /> : <RefreshCw size={13} />}
              Check for updates
            </button>
          )}
          <button
            onClick={() => window.open(RELEASES_URL, "_blank", "noopener,noreferrer")}
            className="rounded-md border border-border px-3 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text"
          >
            Release notes
          </button>
        </div>
      </div>
    </div>
  );
}

/// Desktop-notification toggles (Plan 16 Phase 3). OS toasts for a curated set of
/// events (transfer batch done/failed, folder-sync error, edit-in-place save
/// failure). Off-window by default so they don't double up with in-app toasts.
function NotificationsRow() {
  const notifications = useSettings((s) => s.notifications);
  const setNotifications = useSettings((s) => s.setNotifications);

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2 text-sm font-medium text-text">
        <Bell size={15} className="text-accent" />
        Desktop notifications
      </div>
      <p className="mb-2 text-xs text-text-muted">
        OS toasts for transfers finishing, folder-sync errors, and failed
        edit-in-place saves — so you see them when Ghost FTP is in the background.
      </p>
      <div className="flex flex-col gap-2">
        <Toggle
          label="Enable desktop notifications"
          checked={notifications.enabled}
          onChange={(v) => setNotifications({ ...notifications, enabled: v })}
        />
        <Toggle
          label="Only when Ghost FTP isn't focused"
          help="Skip toasts while the window is in the foreground (in-app toasts already cover it)."
          checked={notifications.unfocusedOnly}
          disabled={!notifications.enabled}
          onChange={(v) => setNotifications({ ...notifications, unfocusedOnly: v })}
        />
      </div>
    </div>
  );
}

/// A compact labeled switch matching the settings look.
function Toggle({
  label,
  help,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  help?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2 text-left text-sm disabled:opacity-50",
        "hover:bg-bg-hover"
      )}
    >
      <span className="min-w-0">
        <span className="text-text">{label}</span>
        {help && <span className="mt-0.5 block text-xs text-text-muted">{help}</span>}
      </span>
      <span
        className={cn(
          "relative h-5 w-9 shrink-0 rounded-full transition-colors",
          checked ? "bg-accent" : "bg-border"
        )}
      >
        <span
          className={cn(
            "absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform",
            checked ? "translate-x-4" : "translate-x-0.5"
          )}
        />
      </span>
    </button>
  );
}

/// One-click "Add ghostftp-cli to PATH" (Plan 16 Phase 4). Per-user only, so it never
/// needs admin. `ghostftp-cli` ships as a separate binary from the app; once it's on
/// PATH it works in any terminal. On Windows this writes the per-user
/// `HKCU\Environment\Path` (new terminals pick it up immediately); on Linux
/// it symlinks into `~/.local/bin` and, if needed, adds one guarded line to the
/// shell profile.
function PathRow() {
  const [status, setStatus] = useState<PathStatus | null>(null);
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      setStatus(await ipc.pathStatus());
    } catch (e) {
      toast.error(`Couldn't read PATH status: ${e}`);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const add = async () => {
    setBusy(true);
    try {
      const next = await ipc.pathAdd();
      setStatus(next);
      toast.success(next.detail ?? "Added ghostftp-cli to your PATH.");
    } catch (e) {
      toast.error(`Add to PATH failed: ${e}`);
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    try {
      const next = await ipc.pathRemove();
      setStatus(next);
      toast.success(next.detail ?? "Removed ghostftp-cli from your PATH.");
    } catch (e) {
      toast.error(`Remove from PATH failed: ${e}`);
    } finally {
      setBusy(false);
    }
  };

  // Prefer the managed flag for the primary action: right after an add, our
  // registry/symlink entry is present even though this already-running process's
  // own PATH won't reflect it until a new terminal (or app restart).
  const managed = status?.managed ?? false;
  const onPath = status?.onPath ?? false;
  const canInstallFirst = status ? !status.binHasCli && !onPath && !managed : false;

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2 text-sm font-medium text-text">
        <TerminalSquare size={15} className="text-accent" />
        ghostftp-cli on PATH
      </div>
      <p className="mb-2 text-xs text-text-muted">
        将 <code className="text-text">ghostftp-cli</code> 命令加入 PATH，使其可在任意终端使用。仅对当前用户生效，无需管理员权限。
      </p>

      <div className="rounded-lg border border-border bg-bg-subtle/50 p-3 text-sm">
        {!status ? (
          <div className="flex items-center gap-2 text-text-muted">
            <Loader2 size={15} className="animate-spin" /> Checking…
          </div>
        ) : managed ? (
          <div className="flex items-start gap-2">
            <CheckCircle2 size={15} className="mt-0.5 shrink-0 text-emerald-500" />
            <div className="min-w-0">
              <div className="text-text">ghostftp-cli is on your PATH (managed by Ghost FTP).</div>
              <p className="mt-0.5 truncate text-xs text-text-muted">{status.binDir}</p>
            </div>
          </div>
        ) : onPath ? (
          <div className="flex items-start gap-2">
            <CheckCircle2 size={15} className="mt-0.5 shrink-0 text-emerald-500" />
            <div className="min-w-0">
              <div className="text-text">ghostftp-cli is already on your PATH.</div>
              {status.cliLocation && (
                <p className="mt-0.5 truncate text-xs text-text-muted">
                  {status.cliLocation}
                </p>
              )}
            </div>
          </div>
        ) : canInstallFirst ? (
          <div className="min-w-0">
            <div className="text-text">ghostftp-cli isn't installed yet.</div>
            <p className="mt-0.5 text-xs text-text-muted">
              Install it from the <span className="font-medium">ghostftp-cli</span> tab
              first, then add it to your PATH here.
            </p>
          </div>
        ) : (
          <div className="min-w-0">
            <div className="text-text">ghostftp-cli isn't on your PATH.</div>
            <p className="mt-0.5 truncate text-xs text-text-muted">{status.binDir}</p>
          </div>
        )}

        <div className="mt-3 flex gap-2">
          {managed ? (
            <button
              onClick={() => void remove()}
              disabled={busy}
              className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-60"
            >
              {busy ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Trash2 size={13} />
              )}
              Remove from PATH
            </button>
          ) : (
            <button
              onClick={() => void add()}
              disabled={busy || onPath || canInstallFirst}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90 disabled:opacity-60"
            >
              {busy ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Plus size={13} />
              )}
              Add to PATH
            </button>
          )}
          <button
            onClick={() => void load()}
            disabled={busy}
            className="rounded-md border border-border px-3 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-60"
          >
            Refresh
          </button>
        </div>
      </div>
    </div>
  );
}
