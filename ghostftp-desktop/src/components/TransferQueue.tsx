import { useEffect, useMemo, useState } from "react";
import {
  AlertCircle,
  CheckCircle2,
  Pause,
  Play,
  RotateCcw,
  Trash2,
  X,
} from "lucide-react";
import { useTransfers } from "@/stores/transfersStore";
import { useConnections } from "@/stores/connectionsStore";
import type { Transfer } from "@/lib/types";

export function TransferQueue() {
  const {
    byId,
    panelOpen,
    setPanelOpen,
    cancel,
    clearFinished,
    initListeners,
    loadInitial,
    pausedAll,
    pauseAll,
    resumeAll,
    pause,
    resume,
    retry,
  } = useTransfers();
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const activeProfile = profiles.find((p) => p.id === activeProfileId);
  const [tab, setTab] = useState<"queue" | "failed" | "completed">("queue");
  const [now, setNow] = useState(Date.now());
  const [logClearedAt, setLogClearedAt] = useState(0);

  useEffect(() => {
    let unsub: (() => void) | undefined;
    void (async () => {
      await loadInitial();
      unsub = await initListeners();
    })();
    return () => unsub?.();
  }, [initListeners, loadInitial]);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const transfers = useMemo(
    () => Object.values(byId).sort((a, b) => b.startedAt - a.startedAt),
    [byId]
  );
  const active = transfers.filter(
    (t) =>
      t.status === "transferring" ||
      t.status === "queued" ||
      t.status === "paused"
  );
  const failed = transfers.filter((t) => t.status === "error");
  const completed = transfers.filter(
    (t) =>
      t.status === "done" ||
      t.status === "skipped" ||
      t.status === "canceled"
  );
  const visible =
    tab === "failed" ? failed : tab === "completed" ? completed : active;
  const visibleLogTransfers = active
    .filter((t) => t.startedAt >= logClearedAt)
    .slice(0, 6)
    .reverse();

  const retryAllFailed = () => {
    void Promise.allSettled(failed.map((transfer) => retry(transfer.id)));
  };

  if (!panelOpen) {
    return (
      <button
        type="button"
        className="ghost-transfer-collapsed"
        onClick={() => setPanelOpen(true)}
      >
        Transfer Queue ({active.length})
      </button>
    );
  }

  return (
    <section className="ghost-bottom-band" aria-label="Transfer activity">
      <div className="ghost-transfer-box">
        <div className="ghost-transfer-tabs" role="tablist" aria-label="Transfer queue filters">
          <button
            type="button"
            role="tab"
            aria-selected={tab === "queue"}
            className={tab === "queue" ? "active" : ""}
            onClick={() => setTab("queue")}
          >
            Transfer Queue ({active.length})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === "failed"}
            className={tab === "failed" ? "active failed" : "failed"}
            onClick={() => setTab("failed")}
          >
            <AlertCircle size={13} /> Failed ({failed.length})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === "completed"}
            className={tab === "completed" ? "active completed" : "completed"}
            onClick={() => setTab("completed")}
          >
            <CheckCircle2 size={13} /> Completed ({completed.length})
          </button>
          <span />
          {failed.length > 0 && (
            <button
              type="button"
              title="Retry all failed transfers"
              aria-label="Retry all failed transfers"
              onClick={retryAllFailed}
            >
              <RotateCcw size={14} />
            </button>
          )}
          <button
            type="button"
            title={pausedAll ? "Resume all transfers" : "Pause all transfers"}
            aria-label={pausedAll ? "Resume all transfers" : "Pause all transfers"}
            onClick={() => (pausedAll ? resumeAll() : pauseAll())}
            disabled={active.length === 0}
          >
            {pausedAll ? <Play size={14} /> : <Pause size={14} />}
          </button>
          <button
            type="button"
            title="Clear completed, canceled and failed transfers"
            aria-label="Clear finished transfers"
            onClick={clearFinished}
            disabled={failed.length + completed.length === 0}
          >
            <Trash2 size={14} />
          </button>
          <button
            type="button"
            title="Hide transfer panel"
            aria-label="Hide transfer panel"
            onClick={() => setPanelOpen(false)}
          >
            <X size={14} />
          </button>
        </div>

        <div className="ghost-transfer-head" aria-hidden="true">
          <span>File</span>
          <span>Direction</span>
          <span>Progress</span>
          <span>Size</span>
          <span>Status</span>
          <span>Speed</span>
          <span>ETA</span>
          <span />
        </div>

        <div className="ghost-transfer-rows">
          {visible.length === 0 ? (
            <div className="ghost-transfer-empty">
              {tab === "queue"
                ? "No active transfers."
                : tab === "failed"
                  ? "No failed transfers."
                  : "No completed transfers."}
            </div>
          ) : (
            visible.map((transfer) => (
              <TransferRow
                key={transfer.id}
                transfer={transfer}
                now={now}
                onCancel={() => cancel(transfer.id)}
                onPause={() => pause(transfer.id)}
                onResume={() => resume(transfer.id)}
                onRetry={() => retry(transfer.id)}
              />
            ))
          )}
        </div>
      </div>

      <div className="ghost-server-log">
        <div className="ghost-log-head">
          <strong>▣ Server Log</strong>
          <span />
          <button
            type="button"
            onClick={() => setLogClearedAt(Date.now())}
            disabled={visibleLogTransfers.length === 0}
          >
            Clear
          </button>
        </div>
        <div className="ghost-log-lines" aria-live="polite">
          <LogLine
            text={
              activeSessionId && activeProfile
                ? `Connected to ${activeProfile.host}:${activeProfile.port}`
                : "Ready — waiting for a server connection."
            }
          />
          {activeSessionId && activeProfile?.defaultRemotePath && (
            <LogLine text={`Remote path: ${activeProfile.defaultRemotePath}`} accent />
          )}
          {visibleLogTransfers.map((transfer) => (
            <LogLine
              key={transfer.id}
              text={`${transfer.kind === "upload" ? "Upload" : "Download"} ${transfer.status}: ${baseName(transfer.source)}`}
              accent={transfer.status === "transferring"}
            />
          ))}
        </div>
      </div>
    </section>
  );
}

function TransferRow({
  transfer: t,
  now,
  onCancel,
  onPause,
  onResume,
  onRetry,
}: {
  transfer: Transfer;
  now: number;
  onCancel: () => void;
  onPause: () => void;
  onResume: () => void;
  onRetry: () => void;
}) {
  const pct =
    t.size > 0
      ? Math.min(100, Math.round((t.transferred / t.size) * 100))
      : t.status === "done" || t.status === "skipped"
        ? 100
        : 0;
  const elapsed = Math.max(1, (now - t.startedAt) / 1000);
  const speed = t.status === "transferring" ? t.transferred / elapsed : 0;
  const eta =
    speed > 0 && t.size > t.transferred ? (t.size - t.transferred) / speed : 0;
  const status = transferStatusLabel(t);
  const active =
    t.status === "transferring" ||
    t.status === "queued" ||
    t.status === "paused";

  return (
    <div className="ghost-transfer-row" data-status={t.status}>
      <span className="file">▧ {baseName(t.source)}</span>
      <span className={t.kind}>{t.kind === "upload" ? "↑ Upload" : "↓ Download"}</span>
      <span className="progress">
        <progress
          className="ghost-transfer-progress"
          max={100}
          value={pct}
          aria-label={`${baseName(t.source)} transfer progress`}
        />
        <em>{pct}%</em>
      </span>
      <span>{formatBytes(t.size)}</span>
      <span className={`status ${t.status}`}>{status}</span>
      <span>{formatSpeed(speed)}</span>
      <span>{formatEta(eta)}</span>
      <span className="actions">
        {t.status === "paused" ? (
          <button type="button" onClick={onResume} title="Resume" aria-label="Resume transfer">
            <Play size={12} />
          </button>
        ) : t.status === "transferring" || t.status === "queued" ? (
          <button type="button" onClick={onPause} title="Pause" aria-label="Pause transfer">
            <Pause size={12} />
          </button>
        ) : t.status === "error" ? (
          <button type="button" onClick={onRetry} title="Retry" aria-label="Retry transfer">
            <RotateCcw size={12} />
          </button>
        ) : null}
        {active && (
          <button type="button" onClick={onCancel} title="Cancel" aria-label="Cancel transfer">
            <X size={12} />
          </button>
        )}
      </span>
    </div>
  );
}

function transferStatusLabel(transfer: Transfer) {
  switch (transfer.status) {
    case "done":
      return "Completed";
    case "skipped":
      return "Skipped";
    case "canceled":
      return "Canceled";
    case "error":
      return transfer.retryAttempt ? `Retry ${transfer.retryAttempt}` : "Failed";
    case "paused":
      return "Paused";
    case "queued":
      return "Queued";
    case "transferring":
      return "Transferring";
  }
}

function LogLine({ text, accent = false }: { text: string; accent?: boolean }) {
  const time = new Date().toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
  return (
    <div>
      <time>{time}</time>
      <span className={accent ? "accent" : ""}>{text}</span>
    </div>
  );
}

function baseName(path: string) {
  return path.replace(/\\/g, "/").split("/").filter(Boolean).pop() || path;
}

function formatBytes(n: number) {
  if (!n) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let value = n;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value >= 10 || index === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[index]}`;
}

function formatSpeed(n: number) {
  return n > 0 ? `${formatBytes(n)}/s` : "—";
}

function formatEta(sec: number) {
  if (!sec || !Number.isFinite(sec)) return "—";
  const seconds = Math.max(0, Math.round(sec));
  const minutes = Math.floor(seconds / 60);
  return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
}
