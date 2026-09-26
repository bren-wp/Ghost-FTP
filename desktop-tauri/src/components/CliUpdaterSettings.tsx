import { useEffect } from "react";
import { Terminal, Loader2, CheckCircle2, AlertTriangle, Download } from "lucide-react";
import { useCliUpdater } from "@/stores/cliUpdaterStore";
import type { CliUpdateMode } from "@/lib/types";
import { cn } from "@/lib/cn";

const MODES: { value: CliUpdateMode; label: string; help: string }[] = [
  { value: "ask", label: "Ask me", help: "Prompt when the CLI is out of date." },
  { value: "auto", label: "Automatic", help: "Update silently on launch." },
  { value: "off", label: "Off", help: "Never check." },
];

/// Settings → ghostftp-cli (Plan 10 Phase 0c/0d). Shows whether the standalone
/// `ghostftp-cli` is installed and current, lets the user update it, and persists the
/// ask / auto / off preference. `ghostftp-cli` ships as a separate download from the
/// app, so it can lag after an app update — this is where drift is surfaced and
/// resolved.
export function CliUpdaterSettings() {
  const status = useCliUpdater((s) => s.status);
  const busy = useCliUpdater((s) => s.busy);
  const refresh = useCliUpdater((s) => s.refresh);
  const update = useCliUpdater((s) => s.update);
  const setMode = useCliUpdater((s) => s.setMode);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const mode = status?.mode ?? "ask";

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h3 className="mb-1 flex items-center gap-2 text-sm font-medium text-text">
          <Terminal size={15} className="text-accent" />
          ghostftp-cli
        </h3>
        <p className="text-xs text-text-muted">
          The command-line client ships as a separate download from the app, so it
          can fall behind after an app update. Keep it in step here.
        </p>
      </div>

      {/* Status card */}
      <div className="rounded-lg border border-border bg-bg-subtle/50 p-3 text-sm">
        {!status ? (
          <div className="flex items-center gap-2 text-text-muted">
            <Loader2 size={15} className="animate-spin" /> Checking…
          </div>
        ) : !status.installed ? (
          <div className="flex items-start gap-2">
            <AlertTriangle size={15} className="mt-0.5 shrink-0 text-amber-500" />
            <div className="min-w-0">
              <div className="text-text">ghostftp-cli isn't installed (or isn't on your PATH).</div>
              <p className="mt-0.5 text-xs text-text-muted">
                Install it from the release page, or click Install to download the
                matching build into Ghost FTP's data folder.
              </p>
            </div>
          </div>
        ) : status.stale ? (
          <div className="flex items-start gap-2">
            <AlertTriangle size={15} className="mt-0.5 shrink-0 text-amber-500" />
            <div className="min-w-0">
              <div className="text-text">
                ghostftp-cli v{status.cliVersion} is older than Ghost FTP v{status.appVersion}.
              </div>
              {status.cliPath && (
                <p className="mt-0.5 truncate text-xs text-text-muted">{status.cliPath}</p>
              )}
            </div>
          </div>
        ) : (
          <div className="flex items-start gap-2">
            <CheckCircle2 size={15} className="mt-0.5 shrink-0 text-emerald-500" />
            <div className="min-w-0">
              <div className="text-text">ghostftp-cli v{status.cliVersion} is up to date.</div>
              {status.cliPath && (
                <p className="mt-0.5 truncate text-xs text-text-muted">{status.cliPath}</p>
              )}
            </div>
          </div>
        )}

        {status?.message && (
          <p className="mt-2 border-t border-border/60 pt-2 text-xs text-text-muted">
            {status.message}
          </p>
        )}

        <div className="mt-3 flex gap-2">
          <button
            onClick={() => void update()}
            disabled={busy}
            className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent/90 disabled:opacity-60"
          >
            {busy ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <Download size={13} />
            )}
            {!status?.installed ? "Install ghostftp-cli" : busy ? "Updating…" : "Update now"}
          </button>
          <button
            onClick={() => void refresh()}
            disabled={busy}
            className="rounded-md border border-border px-3 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-60"
          >
            Check again
          </button>
        </div>
      </div>

      {/* Preference */}
      <div>
        <div className="mb-1.5 text-sm font-medium text-text">When the CLI is out of date</div>
        <div className="flex flex-col gap-1.5">
          {MODES.map((m) => (
            <button
              key={m.value}
              onClick={() => void setMode(m.value)}
              disabled={busy}
              className={cn(
                "flex items-center justify-between rounded-md border px-3 py-2 text-left text-sm disabled:opacity-60",
                mode === m.value
                  ? "border-accent bg-accent/10 text-text"
                  : "border-border text-text-muted hover:bg-bg-hover hover:text-text"
              )}
            >
              <span>
                <span className="font-medium">{m.label}</span>
                <span className="ml-2 text-xs text-text-muted">{m.help}</span>
              </span>
              {mode === m.value && <CheckCircle2 size={15} className="text-accent" />}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
