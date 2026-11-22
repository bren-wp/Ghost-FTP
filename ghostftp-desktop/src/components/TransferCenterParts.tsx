// Ghost FTP by Brendigo
// Presentation-only transfer widgets. The queue, network calls, scheduler and
// history stay in TransferCenterDialog; no mutable backend state lives here.
import type { ReactNode } from "react";
import { Activity, Pause, Play, RotateCcw, XCircle } from "lucide-react";
import type { Transfer } from "@/lib/types";
import { messageOf } from "@/lib/errors";
import { prepareBandwidthGraph, type BandwidthSample } from "@/lib/visualizationMath";

export type { BandwidthSample } from "@/lib/visualizationMath";

export function Tab({
  label,
  icon,
  active = false,
  onClick,
}: {
  label: string;
  icon?: ReactNode;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`flex items-center gap-1.5 whitespace-nowrap rounded-md border px-3 py-2 ${
        active
          ? "border-accent bg-accent/15 text-white"
          : "border-transparent text-text-muted hover:bg-bg-hover"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}

export function Empty({ hasHistory, onClearFilters }: { hasHistory: boolean; onClearFilters: () => void }) {
  return (
    <div className="grid min-h-48 place-items-center px-6 py-8 text-text-muted">
      <div className="w-full max-w-md rounded-xl border border-border bg-[#071f35] px-6 py-7 text-center shadow-[inset_0_1px_0_rgba(255,255,255,.02)]">
        <Activity size={28} className="mx-auto mb-3 text-accent" />
        <div className="font-semibold text-text">
          {hasHistory ? "No matching transfers" : "No transfers yet"}
        </div>
        <div className="mx-auto mt-1 max-w-sm text-[12px] leading-5">
          {hasHistory
            ? "No transfer matches the current status, search, direction or time filters."
            : "Start an actual upload or download from Files to populate transfer history."}
        </div>
        {hasHistory && (
          <button type="button" className="ghost-mini-button mx-auto mt-4" onClick={onClearFilters}>
            Clear filters
          </button>
        )}
      </div>
    </div>
  );
}

export function TransferRow({
  transfer,
  rate,
  index,
  selected,
  busy,
  onClick,
  onPauseResume,
  onCancel,
  onRetry,
}: {
  transfer: Transfer;
  rate: number;
  index: number;
  selected: boolean;
  busy: boolean;
  onClick: () => void;
  onPauseResume: () => void;
  onCancel: () => void;
  onRetry: () => void;
}) {
  const pct = transferProgressPercent(transfer);
  const pausable = transfer.status === "transferring" || transfer.status === "queued" || transfer.status === "paused";
  const cancelable = pausable;
  const retryable = transfer.status === "error";

  return (
    <div
      role="row"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(event) => {
        // Native button keyboard activation must not be intercepted by the
        // enclosing selectable row (Enter/Space bubble from child buttons).
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onClick();
        }
      }}
      aria-selected={selected}
      aria-busy={busy}
      className={`grid w-full grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px_106px] items-center border-b border-border-subtle px-2 py-2.5 text-left text-[11.5px] outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-accent ${
        selected ? "bg-accent/10" : "bg-transparent hover:bg-bg-hover"
      }`}
    >
      <span className="text-text-dim">{index}</span>
      <span className="truncate font-medium" title={transfer.source} dir="auto">{baseName(transfer.source)}</span>
      <span className={transfer.kind === "upload" ? "text-accent" : "text-success"}>
        {transfer.kind === "upload" ? "↑ Upload" : "↓ Download"}
      </span>
      <span className="text-text-muted">{formatBytes(transfer.size || 0)}</span>
      <span className="flex items-center gap-2">
        <span
          role="progressbar"
          aria-label={`${baseName(transfer.source)} transfer progress`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Number(pct.toFixed(0))}
          className="h-3 flex-1 overflow-hidden rounded border border-border bg-[#03111d]"
        >
          <span
            className={`block h-full ${
              transfer.status === "error"
                ? "bg-danger"
                : transfer.status === "done"
                  ? "bg-success"
                  : "bg-accent-strong"
            }`}
            style={{ width: `${pct}%` }}
          />
        </span>
        <span className="w-8 text-right text-[10px]">{pct.toFixed(0)}%</span>
      </span>
      <span className="text-text-muted">{formatRate(rate)}</span>
      <span className="text-text-muted">{etaOf(transfer, rate)}</span>
      <span
        className={
          transfer.status === "error"
            ? "text-danger"
            : transfer.status === "done"
              ? "text-success"
              : transfer.status === "paused"
                ? "text-warning"
                : "text-accent"
        }
      >
        {transfer.status}
      </span>
      <span className="flex justify-end gap-1" role="group" aria-label={`Actions for ${baseName(transfer.source)}`}>
        <button
          type="button"
          className="ghost-row-action"
          disabled={!pausable || busy}
          aria-label={transfer.status === "paused" ? "Resume transfer" : "Pause transfer"}
          title={transfer.status === "paused" ? "Resume" : "Pause"}
          onClick={(event) => { event.stopPropagation(); onPauseResume(); }}
        >
          {transfer.status === "paused" ? <Play size={13}/> : <Pause size={13}/>}
        </button>
        <button
          type="button"
          className="ghost-row-action"
          disabled={!retryable || busy}
          aria-label="Retry transfer"
          title="Retry"
          onClick={(event) => { event.stopPropagation(); onRetry(); }}
        >
          <RotateCcw size={13}/>
        </button>
        <button
          type="button"
          className="ghost-row-action"
          disabled={!cancelable || busy}
          aria-label="Cancel transfer"
          title="Cancel"
          onClick={(event) => { event.stopPropagation(); onCancel(); }}
        >
          <XCircle size={13}/>
        </button>
      </span>
    </div>
  );
}

export function TransferDetails({ transfer, rate }: { transfer: Transfer; rate: number }) {
  const pct = transferProgressPercent(transfer);
  return (
    <div className="space-y-1 text-[11px] text-text-muted">
      <div className="truncate font-semibold text-text" title={transfer.source} dir="auto">
        {baseName(transfer.source)}
      </div>
      <div
        role="progressbar"
        aria-label={"Progress for " + baseName(transfer.source)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Number(pct.toFixed(0))}
        className="h-3 overflow-hidden rounded border border-border bg-[#03111d]"
      >
        <div
          className="h-full bg-accent-strong"
          style={{ width: `${pct}%` }}
        />
      </div>
      <div>
        Transferred: {formatBytes(transfer.transferred)} / {formatBytes(transfer.size)}
      </div>
      <div>Speed: {formatRate(rate)}</div>
      <div>ETA: {etaOf(transfer, rate)}</div>
      <div>
        Status: <span className="text-accent">{transfer.status}</span>
      </div>
      {transfer.error && <div role="alert" className="break-words text-danger">{messageOf(transfer.error)}</div>}
    </div>
  );
}

export function BandwidthChart({
  history,
  active = false,
}: {
  history: BandwidthSample[];
  active?: boolean;
}) {
  const graph = prepareBandwidthGraph(history);
  const measuredRate = (value: number) => value === 0 ? "0 B/s" : formatRate(value);
  const latestUpload = graph.latest ? measuredRate(graph.latest.upload) : "—";
  const latestDownload = graph.latest ? measuredRate(graph.latest.download) : "—";
  const description = graph.hasData
    ? "Transfer bandwidth over " + graph.count + " measured samples. Upload now " +
      latestUpload + ", download now " + latestDownload +
      ", maximum observed " + measuredRate(graph.peak) + "."
    : "No measured transfer bandwidth yet.";

  return (
    <section className="min-w-0" aria-label="Transfer bandwidth graph">
      <div className="mb-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-text-muted">
        <span className="whitespace-nowrap" aria-label={"Current upload rate: " + latestUpload}>
          <span className="font-semibold text-accent">↑ Upload</span> {latestUpload}
        </span>
        <span className="whitespace-nowrap" aria-label={"Current download rate: " + latestDownload}>
          <span className="font-semibold text-success">↓ Download</span> {latestDownload}
        </span>
        <span className="whitespace-nowrap">Peak {graph.hasData ? measuredRate(graph.peak) : "—"}</span>
        <span className="whitespace-nowrap">
          Average ↑ {graph.hasData ? measuredRate(graph.average.upload) : "—"} ·
          {" "}↓ {graph.hasData ? measuredRate(graph.average.download) : "—"}
        </span>
      </div>
      <div className="relative h-24 w-full min-w-0 overflow-hidden rounded-md border border-border-subtle bg-[#041522]">
        {graph.hasData ? (
          <svg
            viewBox="0 0 600 120"
            preserveAspectRatio="none"
            className="h-full w-full"
            role="img"
            aria-label={description}
          >
            <title>{description}</title>
            <g stroke="rgb(104 151 189)" strokeOpacity="0.22" strokeWidth="1" vectorEffect="non-scaling-stroke">
              <line x1="0" x2="600" y1="14" y2="14" />
              <line x1="0" x2="600" y1="60" y2="60" />
              <line x1="0" x2="600" y1="106" y2="106" />
              <line x1="150" x2="150" y1="14" y2="106" />
              <line x1="300" x2="300" y1="14" y2="106" />
              <line x1="450" x2="450" y1="14" y2="106" />
            </g>
            <polyline
              fill="none"
              stroke="rgb(56 171 255)"
              strokeWidth="2.6"
              vectorEffect="non-scaling-stroke"
              strokeLinecap="round"
              strokeLinejoin="round"
              points={graph.uploadPoints}
            />
            <polyline
              fill="none"
              stroke="rgb(36 218 139)"
              strokeWidth="2.5"
              vectorEffect="non-scaling-stroke"
              strokeLinecap="round"
              strokeLinejoin="round"
              points={graph.downloadPoints}
            />
            <circle cx="598" cy={graph.lastUploadY} r="3" fill="rgb(56 171 255)" />
            <circle cx="598" cy={graph.lastDownloadY} r="3" fill="rgb(36 218 139)" />
          </svg>
        ) : (
          <div className="flex h-full items-center justify-center px-3 text-center text-xs text-text-muted" role="status">
            No bandwidth measurements yet. Start a transfer to display real upload and download rates.
          </div>
        )}
      </div>
      <div className="mt-1.5 flex flex-wrap justify-between gap-x-3 gap-y-1 text-[10px] text-text-dim">
        <span>{graph.count} measured sample{graph.count === 1 ? "" : "s"} · newest on right</span>
        <span>{active ? "Live" : graph.hasData ? "Last recorded activity" : "Waiting for activity"}</span>
      </div>
    </section>
  );
}


export function transferProgressPercent(
  transfer: Pick<Transfer, "size" | "transferred"> & Partial<Pick<Transfer, "status">>,
): number {
  // A server-confirmed zero-byte transfer is complete even though there are
  // no bytes from which to calculate a ratio. Never label an unfinished or
  // failed transfer as 100% merely because its size is unknown.
  if (transfer.status === "done") return 100;
  if (!Number.isFinite(transfer.size) || transfer.size <= 0 ||
      !Number.isFinite(transfer.transferred) || transfer.transferred < 0) return 0;
  return Math.max(0, Math.min(100, (transfer.transferred / transfer.size) * 100));
}

export function etaOf(transfer: Transfer, speed: number) {
  if (
    !Number.isFinite(speed) || speed <= 0 ||
    !Number.isFinite(transfer.size) ||
    !Number.isFinite(transfer.transferred) ||
    transfer.transferred < 0 ||
    transfer.status !== "transferring" ||
    transfer.size <= transfer.transferred
  ) {
    return "—";
  }
  const seconds = Math.ceil((transfer.size - transfer.transferred) / speed);
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

export function baseName(path: string) {
  return path.split(/[\\/]/).filter(Boolean).pop() || path || "transfer";
}

export function formatRate(bytesPerSecond: number) {
  return Number.isFinite(bytesPerSecond) && bytesPerSecond > 0
    ? `${formatBytes(bytesPerSecond)}/s` : "—";
}

export function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  // A real zero-byte file is not an unknown size.
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let index = 0;
  let value = bytes;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}
