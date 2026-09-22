import { useCallback, useEffect, useRef, useState } from "react";
import {
  MonitorSmartphone,
  KeyRound,
  Loader2,
  Copy,
  Check,
  Trash2,
  ShieldAlert,
} from "lucide-react";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import type { AgentHostStatus } from "@/lib/types";
import { onAgentHostPaired } from "@/lib/ipc";
import { cn } from "@/lib/cn";

/// Settings → Remote control. Lets THIS machine be driven from the user's other
/// Ghost FTPs: flip it on, click "Show pairing code", read the six digits into the
/// other device. No separate daemon download, no terminal — the app hosts the
/// same ghostftp-agentd stack in-process.
export function RemoteControlSettings() {
  const [status, setStatus] = useState<AgentHostStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  const refresh = useCallback(async () => {
    try {
      setStatus(await ipc.agentHostStatus());
    } catch (e) {
      toast.error("Remote control unavailable", String(e));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // A controller finished pairing — refresh the peer list and celebrate.
  useEffect(() => {
    let disposed = false;
    let unlisten: (() => void) | undefined;
    const registration = onAgentHostPaired((e) => {
      toast.success("New machine paired", `${e.name} can now control this computer`);
      void refresh();
    });

    void registration
      .then((cleanup) => {
        if (disposed) cleanup();
        else unlisten = cleanup;
      })
      .catch((error) => {
        toast.error("Remote-control listener unavailable", String(error));
      });

    return () => {
      disposed = true;
      unlisten?.();
    };
  }, [refresh]);

  // While a pairing window is open, tick the countdown down once a second.
  const hasWindow = !!status?.pairing;
  useEffect(() => {
    if (!hasWindow) return;
    const t = setInterval(() => {
      setStatus((s) =>
        s?.pairing
          ? {
              ...s,
              pairing:
                s.pairing.remainingSecs <= 1
                  ? null
                  : { ...s.pairing, remainingSecs: s.pairing.remainingSecs - 1 },
            }
          : s
      );
    }, 1000);
    return () => clearInterval(t);
  }, [hasWindow]);

  const run = async (fn: () => Promise<AgentHostStatus>) => {
    setBusy(true);
    try {
      setStatus(await fn());
    } catch (e) {
      toast.error("Remote control", String(e));
    } finally {
      setBusy(false);
    }
  };

  const toggleEnabled = (v: boolean) =>
    run(() => ipc.agentHostSetEnabled(v));

  const copyCode = async () => {
    if (!status?.pairing) return;
    try {
      await navigator.clipboard.writeText(status.pairing.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (error) {
      toast.warning(
        "Couldn't copy the pairing code",
        `The code remains visible on screen. ${String(error)}`
      );
    }
  };

  if (!status) {
    return (
      <div className="flex items-center gap-2 text-xs text-text-dim">
        <Loader2 size={13} className="animate-spin" /> Loading…
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-sm">
            <MonitorSmartphone size={14} className="text-accent" />
            Let my other machines control this computer
          </div>
          <div className="mt-1 text-xs text-text-dim">
            Runs a tiny encrypted agent so another Ghost FTP on your network can browse,
            transfer files, and (if you allow it) run commands here. Off by default.
            {status.running && (
              <>
                {" "}
                Listening on port{" "}
                <span className="font-mono text-text-muted">{status.port}</span>.
              </>
            )}
          </div>
        </div>
        <Toggle checked={status.enabled} disabled={busy} onChange={toggleEnabled} />
      </div>

      {status.running && (
        <>
          {/* Pairing */}
          <div className="rounded-lg border border-border bg-bg-subtle p-3">
            {status.pairing ? (
              <div className="space-y-2">
                <div className="text-xs text-text-muted">
                  On your other machine: New Connection → Ghost FTP Agent → pick{" "}
                  <span className="font-medium text-text">
                    {status.hostname || "this machine"}
                  </span>{" "}
                  → enter this code.
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex-1 rounded-md border border-accent/40 bg-accent/10 px-3 py-2 text-center font-mono text-2xl font-semibold tracking-[0.4em] text-text">
                    {status.pairing.code}
                  </div>
                  <button
                    type="button"
                    onClick={copyCode}
                    className="rounded-md border border-border bg-bg-panel p-2 text-text-muted hover:bg-bg-hover hover:text-text"
                    title="Copy code"
                  >
                    {copied ? (
                      <Check size={14} className="text-success" />
                    ) : (
                      <Copy size={14} />
                    )}
                  </button>
                </div>
                <div className="flex items-center justify-between text-[11px] text-text-dim">
                  <span>Expires in {formatCountdown(status.pairing.remainingSecs)}</span>
                  <button
                    type="button"
                    onClick={() => run(() => ipc.agentHostClosePairing())}
                    className="underline hover:text-text"
                  >
                    Close
                  </button>
                </div>
              </div>
            ) : (
              <button
                type="button"
                disabled={busy}
                onClick={() => run(() => ipc.agentHostOpenPairing())}
                className="btn-accent flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
              >
                <KeyRound size={13} /> Show pairing code
              </button>
            )}
          </div>

          {/* What controllers may do here */}
          <div className="space-y-2">
            <div className="text-xs font-medium text-text">
              What paired machines may do here
            </div>
            <PolicyToggle
              label="Run commands (exec)"
              help="Allow paired controllers — and any AI agent driving them — to run shell commands on this machine. The headline capability; turn off for a browse/read-only foothold."
              checked={status.allowExec}
              disabled={busy}
              onChange={(v) =>
                run(() => ipc.agentHostSetPolicy(v, status.allowWrite))
              }
            />
            <PolicyToggle
              label="Write files (upload, delete, rename)"
              help="Allow paired controllers to modify this machine's filesystem. Reads and directory listings are always allowed once paired."
              checked={status.allowWrite}
              disabled={busy}
              onChange={(v) =>
                run(() => ipc.agentHostSetPolicy(status.allowExec, v))
              }
            />
            {status.allowExec && (
              <div className="flex items-start gap-1.5 text-[11px] text-warning">
                <ShieldAlert size={12} className="mt-0.5 shrink-0" />
                Paired machines can run commands here. Only pair devices you trust,
                and revoke any you no longer use below.
              </div>
            )}
          </div>

          {/* Paired controllers */}
          <div className="space-y-1.5">
            <div className="text-xs font-medium text-text">
              Paired machines ({status.peers.length})
            </div>
            {status.peers.length === 0 ? (
              <div className="text-[11px] text-text-dim">
                None yet. Click “Show pairing code” and enter it on another machine.
              </div>
            ) : (
              status.peers.map((p) => (
                <div
                  key={p.publicKey}
                  className="flex items-center justify-between rounded-md border border-border bg-bg-subtle px-2.5 py-1.5"
                >
                  <div className="min-w-0">
                    <div className="truncate text-xs font-medium text-text">
                      {p.name}
                    </div>
                    <div className="truncate font-mono text-[10px] text-text-dim">
                      {p.fingerprint}
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => run(() => ipc.agentHostRevokePeer(p.publicKey))}
                    className="ml-2 flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[11px] text-text-dim hover:bg-danger/10 hover:text-danger disabled:opacity-50"
                    title="Revoke — this machine can no longer connect"
                  >
                    <Trash2 size={12} /> Revoke
                  </button>
                </div>
              ))
            )}
          </div>

          <div className="text-[11px] text-text-dim">
            This machine’s key: <span className="font-mono">{status.fingerprint}</span>
            {" · "}
            {status.os}
            {navigator.platform.startsWith("Win") && (
              <> · Windows may ask to allow Ghost FTP through the firewall the first time.</>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function formatCountdown(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

function PolicyToggle({
  label,
  help,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  help: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="min-w-0 flex-1">
        <div className="text-sm">{label}</div>
        <div className="text-[11px] text-text-dim">{help}</div>
      </div>
      <Toggle checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}

function Toggle({
  checked,
  disabled,
  onChange,
}: {
  checked: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
}) {
  const ref = useRef<HTMLButtonElement>(null);
  return (
    <button
      ref={ref}
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-5 w-9 shrink-0 rounded-full border transition-colors disabled:opacity-50",
        checked
          ? "bg-accent border-accent"
          : "bg-bg-subtle border-border hover:border-text-dim"
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow-elev-1 transition-transform",
          checked ? "translate-x-4" : "translate-x-0"
        )}
      />
    </button>
  );
}
