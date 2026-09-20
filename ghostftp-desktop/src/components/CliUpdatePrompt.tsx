import { useEffect } from "react";
import { ArrowUpCircle, X } from "lucide-react";
import { useCliUpdater } from "@/stores/cliUpdaterStore";

/**
 * Non-blocking prompt for a stale `ghostftp-cli` (Plan 10 Phase 0c/0d). Mounted at
 * the app root; it inits the CLI-updater store and, when the installed CLI lags
 * the app AND the user's preference is `ask`, shows a dismissible bar above the
 * status bar with Update now / Not now / Always update automatically. `auto`
 * updates silently (no prompt); `off` never checks — so this renders nothing in
 * either of those modes.
 */
export function CliUpdatePrompt() {
  const status = useCliUpdater((s) => s.status);
  const busy = useCliUpdater((s) => s.busy);
  const dismissed = useCliUpdater((s) => s.dismissed);
  const update = useCliUpdater((s) => s.update);
  const setMode = useCliUpdater((s) => s.setMode);
  const dismiss = useCliUpdater((s) => s.dismiss);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    useCliUpdater
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
    status?.installed === true &&
    status.stale === true &&
    status.mode === "ask";

  if (!show || !status) return null;

  return (
    <div className="flex items-center gap-3 border-t border-accent/40 bg-accent/10 px-4 py-2 text-sm">
      <ArrowUpCircle className="h-4 w-4 shrink-0 text-accent" />
      <span className="min-w-0 flex-1 truncate text-text">
        A newer <span className="font-medium">ghostftp-cli</span> is available —
        yours is v{status.cliVersion} but Ghost FTP is v{status.appVersion}.
      </span>
      <button
        onClick={() => void update()}
        disabled={busy}
        className="shrink-0 rounded-md bg-accent px-2.5 py-1 text-xs font-medium text-white hover:bg-accent/90 disabled:opacity-60"
      >
        {busy ? "Updating…" : "Update now"}
      </button>
      <button
        onClick={() => void setMode("auto")}
        disabled={busy}
        className="shrink-0 rounded-md border border-border px-2.5 py-1 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-60"
      >
        Always update automatically
      </button>
      <button
        onClick={dismiss}
        title="Not now"
        className="shrink-0 rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
