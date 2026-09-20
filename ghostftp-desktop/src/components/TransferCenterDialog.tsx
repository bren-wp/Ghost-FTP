import { useEffect, useMemo, useRef, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  Activity,
  ArrowDown,
  ArrowUp,
  Bookmark,
  CheckCircle2,
  FolderTree,
  HelpCircle,
  Languages,
  Pause,
  Play,
  Plus,
  RotateCcw,
  Search,
  Server,
  Settings,
  Trash2,
  Wrench,
  XCircle,
} from "lucide-react";
import { useTransfers } from "@/stores/transfersStore";
import type { Transfer } from "@/lib/types";
import { ReferenceWindowControls } from "./ReferenceWindowChrome";
import { GhostMark } from "./GhostBrand";
import { getLocale, setLocale } from "@/lib/i18n";
import { useLayout } from "@/stores/layoutStore";

type FilterTab = "all" | "upload" | "download" | "completed" | "failed";
type BandwidthSample = { upload: number; download: number };

interface Props {
  onClose: () => void;
}

export function TransferCenterDialog({ onClose }: Props) {
  const byId = useTransfers((state) => state.byId);
  const clearFinished = useTransfers((state) => state.clearFinished);
  const pauseAll = useTransfers((state) => state.pauseAll);
  const resumeAll = useTransfers((state) => state.resumeAll);
  const pausedAll = useTransfers((state) => state.pausedAll);
  const cancel = useTransfers((state) => state.cancel);
  const pause = useTransfers((state) => state.pause);
  const resume = useTransfers((state) => state.resume);
  const retry = useTransfers((state) => state.retry);
  const concurrency = useTransfers((state) => state.concurrency);
  const throttleKbps = useTransfers((state) => state.throttleKbps);

  const [tab, setTab] = useState<FilterTab>("all");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [logClearedAt, setLogClearedAt] = useState(0);
  const [bandwidthHistory, setBandwidthHistory] = useState<BandwidthSample[]>([]);
  const previousTotals = useRef({
    at: Date.now(),
    upload: 0,
    download: 0,
  });

  const transfers = useMemo(() => Object.values(byId), [byId]);
  const completed = transfers.filter(
    (transfer) => transfer.status === "done" || transfer.status === "skipped"
  ).length;
  const failed = transfers.filter((transfer) => transfer.status === "error").length;

  const filtered = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return transfers.filter((transfer) => {
      const tabMatch =
        tab === "all" ||
        (tab === "upload" && transfer.kind === "upload") ||
        (tab === "download" && transfer.kind === "download") ||
        (tab === "completed" &&
          (transfer.status === "done" || transfer.status === "skipped")) ||
        (tab === "failed" && transfer.status === "error");
      if (!tabMatch) return false;
      if (!normalizedQuery) return true;
      return `${transfer.source} ${transfer.destination} ${transfer.status}`
        .toLowerCase()
        .includes(normalizedQuery);
    });
  }, [transfers, tab, query]);

  const selected = selectedId
    ? filtered.find((transfer) => transfer.id === selectedId) ?? null
    : null;

  useEffect(() => {
    if (selectedId && !filtered.some((transfer) => transfer.id === selectedId)) {
      setSelectedId(null);
    }
  }, [filtered, selectedId]);

  const logTransfers = useMemo(
    () =>
      transfers
        .filter((transfer) => transfer.startedAt > logClearedAt)
        .sort((a, b) => b.startedAt - a.startedAt)
        .slice(0, 12),
    [transfers, logClearedAt]
  );

  // Sample real byte deltas from the native transfer store. This intentionally
  // replaces the old decorative bandwidth curves with actual queue activity.
  useEffect(() => {
    const sample = () => {
      const snapshot = Object.values(useTransfers.getState().byId);
      const now = Date.now();
      const totals = snapshot.reduce(
        (sum, transfer) => {
          sum[transfer.kind] += transfer.transferred;
          return sum;
        },
        { upload: 0, download: 0 }
      );
      const previous = previousTotals.current;
      const elapsedSeconds = Math.max(0.25, (now - previous.at) / 1000);
      const next: BandwidthSample = {
        upload: Math.max(0, (totals.upload - previous.upload) / elapsedSeconds),
        download: Math.max(
          0,
          (totals.download - previous.download) / elapsedSeconds
        ),
      };
      previousTotals.current = { at: now, ...totals };
      setBandwidthHistory((history) => [...history, next].slice(-48));
    };

    sample();
    const timer = window.setInterval(sample, 1000);
    return () => window.clearInterval(timer);
  }, []);

  const addTransfer = () => {
    window.dispatchEvent(
      new CustomEvent("ghostftp:toolbar-action", {
        detail: { action: "upload" },
      })
    );
    onClose();
  };

  const activeCount = transfers.filter(
    (transfer) =>
      transfer.status === "transferring" || transfer.status === "queued"
  ).length;

  const canPauseSelected =
    selected?.status === "transferring" ||
    selected?.status === "queued" ||
    selected?.status === "paused";
  const canCancelSelected =
    selected?.status === "transferring" ||
    selected?.status === "queued" ||
    selected?.status === "paused";

  return (
    <section
      className="ghost-app-view ghost-standalone-view bg-[#041425]"
      aria-label="Transfer Center"
    >
      <div className="ghost-transfer-center flex h-full w-full flex-col overflow-hidden bg-[#061a2d]">
        <TransferCenterTitlebar onClose={onClose} />

        <div className="ghost-transfer-center-heading flex h-[78px] shrink-0 items-center gap-3 border-b border-border px-4">
          <div>
            <div className="text-xl font-semibold">Transfer Center</div>
            <div className="text-[12px] text-text-muted">
              Manage real uploads, downloads, queue state and transfer history.
            </div>
          </div>
          <div className="flex-1" />
          <button type="button" className="ghost-primary-button" onClick={addTransfer}>
            <Plus size={14} /> Add Transfer
          </button>
          <button
            type="button"
            className="ghost-mini-button"
            disabled={activeCount === 0 && !pausedAll}
            onClick={() => void (pausedAll ? resumeAll() : pauseAll())}
          >
            {pausedAll ? <Play size={14} /> : <Pause size={14} />}
            {pausedAll ? "Resume All" : "Pause All"}
          </button>
          <button
            type="button"
            className="ghost-mini-button"
            disabled={completed === 0 && failed === 0}
            onClick={clearFinished}
          >
            <Trash2 size={14} /> Clear Finished
          </button>
        </div>

        <div className="ghost-transfer-filters flex items-center gap-2 border-b border-border px-4 py-3 text-[12px]">
          <Tab
            active={tab === "all"}
            onClick={() => setTab("all")}
            label={`All Transfers (${transfers.length})`}
          />
          <Tab
            active={tab === "upload"}
            onClick={() => setTab("upload")}
            label={`Uploads (${transfers.filter((transfer) => transfer.kind === "upload").length})`}
            icon={<ArrowUp size={14} />}
          />
          <Tab
            active={tab === "download"}
            onClick={() => setTab("download")}
            label={`Downloads (${transfers.filter((transfer) => transfer.kind === "download").length})`}
            icon={<ArrowDown size={14} />}
          />
          <Tab
            active={tab === "completed"}
            onClick={() => setTab("completed")}
            label={`Completed (${completed})`}
            icon={<CheckCircle2 size={14} />}
          />
          <Tab
            active={tab === "failed"}
            onClick={() => setTab("failed")}
            label={`Failed (${failed})`}
            icon={<XCircle size={14} />}
          />
          <div className="flex-1" />
          <div className="relative">
            <Search
              size={13}
              className="pointer-events-none absolute left-2.5 top-2.5 text-text-dim"
            />
            <input
              className="ghost-ref-input h-8 w-52 pl-8"
              placeholder="Filter transfers…"
              aria-label="Filter transfers"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>
          <select
            className="ghost-ref-input h-8 w-32"
            aria-label="Transfer filter"
            value={tab}
            onChange={(event) => setTab(event.target.value as FilterTab)}
          >
            <option value="all">All Status</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="upload">Uploads</option>
            <option value="download">Downloads</option>
          </select>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          <div className="grid min-w-[920px] grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px_106px] border-b border-border px-2 py-2 text-[10px] uppercase tracking-wider text-text-dim">
            <span>#</span>
            <span>Name</span>
            <span>Direction</span>
            <span>Size</span>
            <span>Progress</span>
            <span>Speed</span>
            <span>ETA</span>
            <span>Status</span>
            <span className="text-right">Actions</span>
          </div>
          <div className="min-w-[920px]">
            {filtered.length === 0 ? (
              <Empty />
            ) : (
              filtered.map((transfer, index) => (
                <TransferRow
                  key={transfer.id}
                  transfer={transfer}
                  index={index + 1}
                  selected={selected?.id === transfer.id}
                  onClick={() => setSelectedId(transfer.id)}
                  onPauseResume={() => void (transfer.status === "paused" ? resume(transfer.id) : pause(transfer.id))}
                  onCancel={() => void cancel(transfer.id)}
                  onRetry={() => void retry(transfer.id)}
                />
              ))
            )}
          </div>
        </div>

        <div className="ghost-transfer-summary-grid grid grid-cols-[1.55fr_.75fr] gap-4 border-t border-border bg-[#051929] p-4">
          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center gap-2">
              <Activity size={16} className="text-accent" />
              <strong>Bandwidth Usage</strong>
              <div className="flex-1" />
              <span className="text-[11px] text-text-muted">
                Live native transfer sampling
              </span>
            </div>
            <BandwidthChart history={bandwidthHistory} />
          </div>

          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center gap-2">
              <Activity size={16} className="text-accent" />
              <strong>Active Transfer Details</strong>
            </div>
            {selected ? (
              <TransferDetails transfer={selected} />
            ) : (
              <div className="text-[12px] text-text-muted">
                Select a transfer to see details.
              </div>
            )}
            <div className="mt-4 flex flex-wrap gap-2">
              <button
                type="button"
                className="ghost-primary-button"
                disabled={!selected || !canPauseSelected}
                onClick={() =>
                  selected &&
                  void (selected.status === "paused"
                    ? resume(selected.id)
                    : pause(selected.id))
                }
              >
                {selected?.status === "paused" ? (
                  <Play size={14} />
                ) : (
                  <Pause size={14} />
                )}
                {selected?.status === "paused" ? "Resume" : "Pause"}
              </button>
              <button
                type="button"
                className="ghost-mini-button"
                disabled={!selected || !canCancelSelected}
                onClick={() => selected && void cancel(selected.id)}
              >
                Cancel
              </button>
              <button
                type="button"
                className="ghost-mini-button"
                disabled={!selected || selected.status !== "error"}
                onClick={() => selected && void retry(selected.id)}
              >
                <RotateCcw size={14} /> Retry
              </button>
            </div>
          </div>
        </div>

        <div className="ghost-transfer-lower-grid grid grid-cols-2 gap-4 border-t border-border bg-[#051929] p-4">
          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center gap-2">
              <Settings size={16} className="text-accent" />
              <strong>Queue Controls</strong>
            </div>
            <div className="grid grid-cols-2 gap-3 text-[11px]">
              <QueueMetric label="Active / queued" value={String(activeCount)} />
              <QueueMetric
                label="Concurrency"
                value={String(concurrency)}
              />
              <QueueMetric
                label="Speed limit"
                value={throttleKbps > 0 ? `${formatBytes(throttleKbps * 1024)}/s` : "No limit"}
              />
              <QueueMetric
                label="Queue state"
                value={pausedAll ? "Paused" : "Running"}
              />
            </div>
            <div className="mt-4 text-[10.5px] leading-5 text-text-dim">
              Persistent scheduling is not exposed here until the native scheduler
              is implemented. Ghost FTP does not simulate scheduled transfers.
              Concurrency, retry and speed policies are configured in Preferences.
            </div>
          </div>

          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center">
              <strong>Transfer Log</strong>
              <div className="flex-1" />
              <button
                type="button"
                className="ghost-mini-button"
                disabled={logTransfers.length === 0}
                onClick={() => setLogClearedAt(Date.now() / 1000)}
              >
                Clear Log
              </button>
            </div>
            <div className="max-h-28 overflow-auto font-mono text-[10.5px] leading-5 text-text-muted">
              {logTransfers.map((transfer) => (
                <div key={transfer.id}>
                  <span
                    className={
                      transfer.status === "error"
                        ? "text-danger"
                        : transfer.status === "done"
                          ? "text-success"
                          : "text-accent"
                    }
                  >
                    {transfer.status.toUpperCase()}
                  </span>
                  {" — "}
                  {baseName(transfer.source)} → {baseName(transfer.destination)}
                </div>
              ))}
              {logTransfers.length === 0 && <div>No transfer events in this view.</div>}
            </div>
          </div>
        </div>

        <div className="flex items-center border-t border-border bg-[#041522] px-4 py-3">
          <span
            className={`mr-2 h-2 w-2 rounded-full ${
              pausedAll ? "bg-warning" : activeCount > 0 ? "bg-accent" : "bg-success"
            }`}
          />
          <span className="text-[11px] text-text-muted">
            {pausedAll
              ? "Queue paused"
              : activeCount > 0
                ? `${activeCount} transfer${activeCount === 1 ? "" : "s"} active or queued`
                : "Ready"}
          </span>
          <div className="flex-1" />
          <div className="text-[11px] text-text-muted">
            Completed: {completed} &nbsp; • &nbsp; Failed: {failed} &nbsp; • &nbsp;
            Queue: {transfers.length}
          </div>
        </div>
      </div>
    </section>
  );
}

function TransferCenterTitlebar({ onClose }: { onClose: () => void }) {
  const openDialog = useLayout((state) => state.openDialog);
  const locale = getLocale();

  const item = (
    label: string,
    icon: React.ReactNode,
    action?: () => void,
    active = false
  ) => (
    <button
      type="button"
      className={`ghost-transfer-nav-item ${active ? "active" : ""}`}
      onClick={action}
      disabled={active || !action}
      aria-current={active ? "page" : undefined}
    >
      {icon}
      <span>{label}</span>
    </button>
  );

  const toggleMaximize = () => {
    try {
      void getCurrentWindow().toggleMaximize();
    } catch {
      // Native window operation; browser/source preview has no Tauri window.
    }
  };

  return (
    <div
      className="ghost-transfer-titlebar"
      data-tauri-drag-region
      onDoubleClick={toggleMaximize}
    >
      <div className="ghost-transfer-brand" data-tauri-drag-region>
        <GhostMark size={34} />
        <div>
          <div className="font-semibold">Ghost FTP</div>
          <div>Transfer Center</div>
        </div>
      </div>
      <nav className="ghost-transfer-nav" aria-label="Transfer Center navigation">
        {item("Sites", <FolderTree size={15} />, onClose)}
        {item("Transfers", <ArrowUp size={15} />, undefined, true)}
        {item("Server", <Server size={15} />, () => openDialog("siteManager"))}
        {item("Bookmarks", <Bookmark size={15} />, () => openDialog("siteManager"))}
        {item("Tools", <Wrench size={15} />, () => openDialog("settings"))}
        {item("Settings", <Settings size={15} />, () => openDialog("settings"))}
        {item("Help", <HelpCircle size={15} />, () => openDialog("about"))}
      </nav>
      <label className="ghost-transfer-language">
        <Languages size={14} />
        <select
          aria-label="Language"
          value={locale}
          onChange={(event) => setLocale(event.target.value as any)}
        >
          <option value="en">English (English)</option>
          <option value="hr">Hrvatski (Croatian)</option>
          <option value="de">Deutsch (German)</option>
          <option value="fr">Français (French)</option>
          <option value="es">Español (Spanish)</option>
          <option value="it">Italiano (Italian)</option>
          <option value="pt">Português (Portuguese)</option>
          <option value="nl">Nederlands (Dutch)</option>
          <option value="pl">Polski (Polish)</option>
          <option value="sl">Slovenščina (Slovenian)</option>
          <option value="sr">Srpski (Serbian)</option>
          <option value="bs">Bosanski (Bosnian)</option>
          <option value="mk">Македонски (Macedonian)</option>
          <option value="sq">Shqip (Albanian)</option>
        </select>
      </label>
      <ReferenceWindowControls onClose={onClose} />
    </div>
  );
}

function Tab({
  label,
  icon,
  active = false,
  onClick,
}: {
  label: string;
  icon?: React.ReactNode;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-md border px-3 py-2 ${
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

function Empty() {
  return (
    <div className="grid min-h-48 place-items-center px-6 py-8 text-text-muted">
      <div className="w-full max-w-md rounded-xl border border-border bg-[#071f35] px-6 py-7 text-center shadow-[inset_0_1px_0_rgba(255,255,255,.02)]">
        <Activity size={28} className="mx-auto mb-3 text-accent" />
        <div className="font-semibold text-text">No transfers in this view</div>
        <div className="mx-auto mt-1 max-w-sm text-[12px] leading-5">
          Start a real upload or download from File Manager, or change the active filters.
        </div>
      </div>
    </div>
  );
}

function TransferRow({
  transfer,
  index,
  selected,
  onClick,
  onPauseResume,
  onCancel,
  onRetry,
}: {
  transfer: Transfer;
  index: number;
  selected: boolean;
  onClick: () => void;
  onPauseResume: () => void;
  onCancel: () => void;
  onRetry: () => void;
}) {
  const pct =
    transfer.size > 0
      ? Math.max(0, Math.min(100, (transfer.transferred / transfer.size) * 100))
      : 0;
  const pausable = transfer.status === "transferring" || transfer.status === "queued" || transfer.status === "paused";
  const cancelable = pausable;
  const retryable = transfer.status === "error";

  return (
    <div
      role="row"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onClick();
        }
      }}
      aria-selected={selected}
      className={`grid w-full grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px_106px] items-center border-b border-border-subtle px-2 py-2.5 text-left text-[11.5px] outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-accent ${
        selected ? "bg-accent/10" : "bg-transparent hover:bg-bg-hover"
      }`}
    >
      <span className="text-text-dim">{index}</span>
      <span className="truncate font-medium">{baseName(transfer.source)}</span>
      <span className={transfer.kind === "upload" ? "text-accent" : "text-success"}>
        {transfer.kind === "upload" ? "↑ Upload" : "↓ Download"}
      </span>
      <span className="text-text-muted">{formatBytes(transfer.size || 0)}</span>
      <span className="flex items-center gap-2">
        <span className="h-3 flex-1 overflow-hidden rounded border border-border bg-[#03111d]">
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
      <span className="text-text-muted">{formatRate(speedOf(transfer))}</span>
      <span className="text-text-muted">{etaOf(transfer)}</span>
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
          disabled={!pausable}
          aria-label={transfer.status === "paused" ? "Resume transfer" : "Pause transfer"}
          title={transfer.status === "paused" ? "Resume" : "Pause"}
          onClick={(event) => { event.stopPropagation(); onPauseResume(); }}
        >
          {transfer.status === "paused" ? <Play size={13}/> : <Pause size={13}/>}
        </button>
        <button
          type="button"
          className="ghost-row-action"
          disabled={!retryable}
          aria-label="Retry transfer"
          title="Retry"
          onClick={(event) => { event.stopPropagation(); onRetry(); }}
        >
          <RotateCcw size={13}/>
        </button>
        <button
          type="button"
          className="ghost-row-action"
          disabled={!cancelable}
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

function TransferDetails({ transfer }: { transfer: Transfer }) {
  const pct =
    transfer.size > 0
      ? Math.max(0, Math.min(100, (transfer.transferred / transfer.size) * 100))
      : 0;
  return (
    <div className="space-y-1 text-[11px] text-text-muted">
      <div className="truncate font-semibold text-text">
        {baseName(transfer.source)}
      </div>
      <div className="h-3 overflow-hidden rounded border border-border bg-[#03111d]">
        <div
          className="h-full bg-accent-strong"
          style={{ width: `${pct}%` }}
        />
      </div>
      <div>
        Transferred: {formatBytes(transfer.transferred)} / {formatBytes(transfer.size)}
      </div>
      <div>Speed: {formatRate(speedOf(transfer))}</div>
      <div>ETA: {etaOf(transfer)}</div>
      <div>
        Status: <span className="text-accent">{transfer.status}</span>
      </div>
      {transfer.error && <div className="text-danger">{transfer.error}</div>}
    </div>
  );
}

function BandwidthChart({ history }: { history: BandwidthSample[] }) {
  const data =
    history.length > 1
      ? history
      : [
          { upload: 0, download: 0 },
          { upload: 0, download: 0 },
        ];
  const peak = Math.max(0, ...data.flatMap((sample) => [sample.upload, sample.download]));
  const scaleMax = Math.max(1, peak);
  const points = (key: keyof BandwidthSample) =>
    data
      .map((sample, index) => {
        const x = data.length === 1 ? 0 : (index / (data.length - 1)) * 600;
        const y = 110 - (sample[key] / scaleMax) * 92;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(" ");

  const latest = data[data.length - 1];
  return (
    <div>
      <div className="mb-2 flex flex-wrap gap-x-5 gap-y-1 text-[10.5px] text-text-muted">
        <span>
          <span className="text-accent">↑</span> Upload {formatRate(latest.upload)}
        </span>
        <span>
          <span className="text-success">↓</span> Download {formatRate(latest.download)}
        </span>
        <span className="ml-auto">Peak {formatRate(peak)}</span>
      </div>
      <div className="relative h-28 overflow-hidden rounded border border-border-subtle bg-[#041522]">
        <div className="absolute inset-0 bg-[repeating-linear-gradient(0deg,transparent_0_27px,rgba(31,80,115,.36)_28px)]" />
        <svg
          viewBox="0 0 600 120"
          preserveAspectRatio="none"
          className="absolute inset-0 h-full w-full"
          aria-label="Live upload and download bandwidth history"
          role="img"
        >
          <polyline
            fill="none"
            stroke="rgb(42 170 255)"
            strokeWidth="2.4"
            points={points("upload")}
          />
          <polyline
            fill="none"
            stroke="rgb(36 218 139)"
            strokeWidth="2.2"
            points={points("download")}
          />
        </svg>
      </div>
    </div>
  );
}

function QueueMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border-subtle bg-[#051929] px-3 py-2">
      <div className="text-[10px] text-text-dim">{label}</div>
      <div className="mt-0.5 font-medium text-text">{value}</div>
    </div>
  );
}

function speedOf(transfer: Transfer) {
  const seconds = Math.max(1, Date.now() / 1000 - transfer.startedAt);
  return transfer.status === "transferring"
    ? transfer.transferred / seconds
    : 0;
}

function etaOf(transfer: Transfer) {
  const speed = speedOf(transfer);
  if (
    !speed ||
    transfer.status !== "transferring" ||
    transfer.size <= transfer.transferred
  ) {
    return "—";
  }
  const seconds = Math.max(
    0,
    Math.ceil((transfer.size - transfer.transferred) / speed)
  );
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

function baseName(path: string) {
  return path.split(/[\\/]/).filter(Boolean).pop() || path || "transfer";
}

function formatRate(bytesPerSecond: number) {
  return bytesPerSecond > 0 ? `${formatBytes(bytesPerSecond)}/s` : "—";
}

function formatBytes(bytes: number) {
  if (!bytes) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let index = 0;
  let value = bytes;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}
