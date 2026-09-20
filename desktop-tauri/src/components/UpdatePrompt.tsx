import { useEffect } from "react";
import { ArrowUpCircle, X, RefreshCw, Loader2 } from "lucide-react";
import { useUpdater } from "@/stores/updaterStore";

/**
 * Non-blocking in-app update prompt (Plan 16 Phase 1/2). Mounted at the app root;
 * it runs the throttled launch check and, when an update is available, shows a
 * dismissible bar above the status bar: Download → a progress bar → Restart to
 * update. Renders nothing when up to date, dismissed, or still checking.
 */
export function UpdatePrompt() {
  const status = useUpdater((s) => s.status);
  const version = useUpdater((s) => s.version);
  const downloaded = useUpdater((s) => s.downloaded);
  const total = useUpdater((s) => s.total);
  const dismissed = useUpdater((s) => s.dismissed);
  const download = useUpdater((s) => s.downloadAndInstall);
  const restart = useUpdater((s) => s.restart);
  const dismiss = useUpdater((s) => s.dismiss);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    useUpdater
      .getState()
      .init()
      .then((c) => {
        if (cancelled) c();
        else cleanup = c;
      });
    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  const show =
    !dismissed &&
    (status === "available" || status === "downloading" || status === "ready");
  if (!show) return null;

  const pct =
    total && total > 0 ? Math.min(100, Math.round((downloaded / total) * 100)) : null;

  return (
    <div className="flex items-center gap-3 border-t border-accent/40 bg-accent/10 px-4 py-2 text-sm">
      <ArrowUpCircle className="h-4 w-4 shrink-0 text-accent" />

      {status === "available" && (
        <>
          <span className="min-w-0 flex-1 truncate text-text">
            <span className="font-medium">Ghost FTP {version}</span> is available.
          </span>
          <button
            onClick={() => void download()}
            className="shrink-0 rounded-md bg-accent px-2.5 py-1 text-xs font-medium text-white hover:bg-accent/90"
          >
            Download &amp; install
          </button>
        </>
      )}

      {status === "downloading" && (
        <>
          <span className="flex min-w-0 flex-1 items-center gap-2 text-text">
            <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-accent" />
            <span className="shrink-0">Downloading update…</span>
            <span className="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-border">
              <span
                className="block h-full bg-accent transition-[width]"
                style={{ width: pct !== null ? `${pct}%` : "40%" }}
              />
            </span>
            {pct !== null && (
              <span className="shrink-0 tabular-nums text-xs text-text-muted">{pct}%</span>
            )}
          </span>
        </>
      )}

      {status === "ready" && (
        <>
          <span className="min-w-0 flex-1 truncate text-text">
            Update downloaded — restart Ghost FTP to finish installing.
          </span>
          <button
            onClick={() => void restart()}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-md bg-accent px-2.5 py-1 text-xs font-medium text-white hover:bg-accent/90"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Restart now
          </button>
        </>
      )}

      {status !== "downloading" && (
        <button
          onClick={dismiss}
          title="Not now"
          className="shrink-0 rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
