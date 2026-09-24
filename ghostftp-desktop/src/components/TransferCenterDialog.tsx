import { useEffect, useMemo, useRef, useState } from "react";
import { open } from "@tauri-apps/plugin-dialog";
import {
  Activity,
  CheckCircle2,
  FolderTree,
  Pause,
  Play,
  Plus,
  MoreHorizontal,
  Clock3,
  RotateCcw,
  Search,
  Trash2,
  XCircle,
} from "lucide-react";
import { liveTransferRate, useTransfers } from "@/stores/transfersStore";
import { useConnections } from "@/stores/connectionsStore";
import type { Transfer } from "@/lib/types";
import { useDialog } from "@/hooks/useDialog";
import { toastError } from "@/lib/errors";
import { useTransferSchedule } from "@/stores/transferScheduleStore";

type FilterTab = "all" | "active" | "completed" | "failed";
type DirectionFilter = "all" | "upload" | "download";
type TimeFilter = "all" | "hour" | "day";
type BandwidthSample = { upload: number; download: number };

interface Props {
  onClose: () => void;
}

export function TransferCenterDialog({ onClose }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const byId = useTransfers((state) => state.byId);
  const rateById = useTransfers((state) => state.rateById);
  const clearCompleted = useTransfers((state) => state.clearCompleted);
  const pauseAll = useTransfers((state) => state.pauseAll);
  const resumeAll = useTransfers((state) => state.resumeAll);
  const pausedAll = useTransfers((state) => state.pausedAll);
  const cancel = useTransfers((state) => state.cancel);
  const pause = useTransfers((state) => state.pause);
  const resume = useTransfers((state) => state.resume);
  const retry = useTransfers((state) => state.retry);
  const move = useTransfers((state) => state.move);
  const enqueueUploads = useTransfers((state) => state.enqueueUploads);
  const activeSessionId = useConnections((state) => state.activeSessionId);
  const activeProfileId = useConnections((state) => state.activeProfileId);
  const profiles = useConnections((state) => state.profiles);
  const activeProfile = profiles.find((profile) => profile.id === activeProfileId);
  const uploadTarget = activeProfile?.defaultRemotePath?.trim() || ".";

  const [tab, setTab] = useState<FilterTab>("all");
  const [query, setQuery] = useState("");
  const [directionFilter, setDirectionFilter] = useState<DirectionFilter>("all");
  const [timeFilter, setTimeFilter] = useState<TimeFilter>("all");
  const [moreOpen, setMoreOpen] = useState(false);
  const moreRef = useRef<HTMLDivElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const schedulerRef = useRef<HTMLDivElement>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const scheduleMode = useTransferSchedule((state) => state.mode);
  const scheduleDate = useTransferSchedule((state) => state.date);
  const scheduleTime = useTransferSchedule((state) => state.time);
  const scheduledTransferId = useTransferSchedule((state) => state.transferId);
  const scheduleArmed = useTransferSchedule((state) => state.armed);
  const setScheduleMode = useTransferSchedule((state) => state.setMode);
  const setScheduleDate = useTransferSchedule((state) => state.setDate);
  const setScheduleTime = useTransferSchedule((state) => state.setTime);
  const setScheduledTransferId = useTransferSchedule((state) => state.setTarget);
  const setScheduleArmed = useTransferSchedule((state) => state.setArmed);
  const [logClearedAt, setLogClearedAt] = useState(0);
  const [bandwidthHistory, setBandwidthHistory] = useState<BandwidthSample[]>([]);

  useDialog(panelRef, { onClose, trapFocus: false });

  useEffect(() => {
    if (!moreOpen) return;
    const raf = requestAnimationFrame(() => {
      moreRef.current
        ?.querySelector<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)')
        ?.focus();
    });
    const onDown = (event: MouseEvent) => {
      if (!moreRef.current?.contains(event.target as Node)) setMoreOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMoreOpen(false);
        moreButtonRef.current?.focus();
      }
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      cancelAnimationFrame(raf);
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [moreOpen]);

  const onMoreMenuKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const items = Array.from(
      event.currentTarget.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)')
    );
    if (event.key === "Escape") {
      event.preventDefault();
      setMoreOpen(false);
      moreButtonRef.current?.focus();
      return;
    }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key) || items.length === 0) return;
    event.preventDefault();
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    const next =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? items.length - 1
          : event.key === "ArrowDown"
            ? (Math.max(current, -1) + 1) % items.length
            : (current <= 0 ? items.length : current) - 1;
    items[next]?.focus();
  };

  const transfers = useMemo(() => Object.values(byId), [byId]);
  const completed = transfers.filter(
    (transfer) => transfer.status === "done" || transfer.status === "skipped"
  ).length;
  const failed = transfers.filter((transfer) => transfer.status === "error").length;
  const activeFilterCount = transfers.filter(
    (transfer) =>
      transfer.status === "transferring" ||
      transfer.status === "queued" ||
      transfer.status === "paused"
  ).length;

  const filtered = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return transfers.filter((transfer) => {
      const tabMatch =
        tab === "all" ||
        (tab === "active" &&
          (transfer.status === "transferring" ||
            transfer.status === "queued" ||
            transfer.status === "paused")) ||
        (tab === "completed" &&
          (transfer.status === "done" || transfer.status === "skipped")) ||
        (tab === "failed" && transfer.status === "error");
      if (!tabMatch) return false;

      if (directionFilter !== "all" && transfer.kind !== directionFilter) return false;

      const now = Date.now() / 1000;
      if (timeFilter === "hour" && transfer.startedAt < now - 3600) return false;
      if (timeFilter === "day" && transfer.startedAt < now - 86400) return false;

      if (!normalizedQuery) return true;
      return `${transfer.source} ${transfer.destination} ${transfer.status}`
        .toLowerCase()
        .includes(normalizedQuery);
    });
  }, [transfers, tab, query, directionFilter, timeFilter]);

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

  const hasLiveTransfer = transfers.some(
    (transfer) => transfer.status === "transferring"
  );

  // Sample the rolling rates derived from real backend progress events. Idle
  // views do not repaint, and a stalled transfer naturally samples as 0 once
  // its last progress event becomes stale.
  useEffect(() => {
    if (!hasLiveTransfer) return;

    const sample = () => {
      const state = useTransfers.getState();
      const now = Date.now();
      const next = Object.values(state.byId).reduce<BandwidthSample>(
        (sum, transfer) => {
          const speed = liveTransferRate(
            transfer,
            state.rateById[transfer.id],
            now
          );
          sum[transfer.kind] += speed;
          return sum;
        },
        { upload: 0, download: 0 }
      );
      setBandwidthHistory((history) => [...history, next].slice(-48));
    };

    sample();
    const timer = window.setInterval(sample, 1000);
    return () => window.clearInterval(timer);
  }, [hasLiveTransfer]);

  const runBackendAction = async (
    failureTitle: string,
    operation: () => Promise<void>
  ) => {
    try {
      await operation();
    } catch (error) {
      toastError(error, failureTitle);
    }
  };

  const revealScheduler = () => {
    setDetailsOpen(true);
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        schedulerRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
        schedulerRef.current?.focus();
      });
    });
  };

  const addTransfer = async (kind: "files" | "folder" = "files") => {
    if (!activeSessionId) return;
    try {
      const picked = await open({
        multiple: kind === "files",
        directory: kind === "folder",
        title: kind === "folder" ? "Add folder transfer" : "Add file transfer",
      });
      if (!picked) return;

      const paths = Array.isArray(picked) ? picked : [picked];
      if (paths.length === 0) return;

      await enqueueUploads(
        activeSessionId,
        paths.map((path) => ({
          path,
          kind: kind === "folder" ? "directory" as const : "file" as const,
        })),
        uploadTarget
      );
    } catch (error) {
      toastError(error, "Couldn't add the transfer");
    }
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
    <div
      className="ghost-workspace-view ghost-standalone-view bg-[#041425]"
      role="region"
      aria-label="Transfers"
    >
      <div
        ref={panelRef}
        className="ghost-transfer-center flex h-full w-full flex-col overflow-hidden bg-[#061a2d]"
      >
        <div className="ghost-transfer-center-heading flex h-[78px] shrink-0 items-center gap-3 border-b border-border px-4">
          <div>
            <div className="text-xl font-semibold">Transfers</div>
            <div className="text-[12px] text-text-muted">
              Manage real uploads, downloads, queue state and transfer history.
            </div>
          </div>
          <div className="flex-1" />
          <button
            type="button"
            className="ghost-primary-button"
            disabled={!activeSessionId}
            title={activeSessionId ? `Upload files to ${uploadTarget}` : "Connect to a server first"}
            onClick={() => void addTransfer("files")}
          >
            <Plus size={14} /> Add Transfer
          </button>
          <div className="relative" ref={moreRef}>
            <button
              ref={moreButtonRef}
              type="button"
              className="ghost-mini-button"
              aria-haspopup="menu"
              aria-expanded={moreOpen}
              onClick={() => setMoreOpen((value) => !value)}
            >
              <MoreHorizontal size={15}/> More
            </button>
            {moreOpen && (
              <div className="ghost-transfer-more-menu" role="menu" onKeyDown={onMoreMenuKeyDown}>
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMoreOpen(false);
                    setDetailsOpen((open) => !open);
                  }}
                >
                  <Activity size={14}/>
                  <span>{detailsOpen ? "Hide Details" : "Show Details"}</span>
                </button>
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMoreOpen(false);
                    revealScheduler();
                  }}
                >
                  <Clock3 size={14}/>
                  <span>Schedule Transfer…</span>
                </button>
                <button
                  type="button"
                  role="menuitem"
                  disabled={!activeSessionId}
                  onClick={() => {
                    setMoreOpen(false);
                    void addTransfer("folder");
                  }}
                >
                  <FolderTree size={14}/>
                  <span>Add Folder Transfer…</span>
                </button>
                <button
                  type="button"
                  role="menuitem"
                  disabled={activeCount === 0 && !pausedAll}
                  onClick={() => {
                    setMoreOpen(false);
                    void runBackendAction(
                      pausedAll ? "Couldn't resume the transfer queue" : "Couldn't pause the transfer queue",
                      () => pausedAll ? resumeAll() : pauseAll()
                    );
                  }}
                >
                  {pausedAll ? <Play size={14}/> : <Pause size={14}/>}
                  <span>{pausedAll ? "Resume All" : "Pause All"}</span>
                </button>
                <button
                  type="button"
                  role="menuitem"
                  disabled={completed === 0}
                  onClick={() => {
                    setMoreOpen(false);
                    clearCompleted();
                  }}
                >
                  <Trash2 size={14}/>
                  <span>Clear Completed</span>
                </button>
              </div>
            )}
          </div>
        </div>

        <div className="ghost-transfer-filters flex items-center gap-2 border-b border-border px-4 py-3 text-[12px]">
          <Tab
            active={tab === "all"}
            onClick={() => setTab("all")}
            label={`All Transfers (${transfers.length})`}
          />
          <Tab
            active={tab === "active"}
            onClick={() => setTab("active")}
            label={`Active (${activeFilterCount})`}
            icon={<Play size={14} />}
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
            aria-label="Transfer direction filter"
            value={directionFilter}
            onChange={(event) => setDirectionFilter(event.target.value as DirectionFilter)}
          >
            <option value="all">All Directions</option>
            <option value="upload">Uploads</option>
            <option value="download">Downloads</option>
          </select>
          <select
            className="ghost-ref-input h-8 w-28"
            aria-label="Transfer time filter"
            value={timeFilter}
            onChange={(event) => setTimeFilter(event.target.value as TimeFilter)}
          >
            <option value="all">Any Time</option>
            <option value="hour">Last Hour</option>
            <option value="day">Last 24h</option>
          </select>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          {filtered.length === 0 ? (
            <Empty />
          ) : (
            <>
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
                {filtered.map((transfer, index) => (
                  <TransferRow
                    key={transfer.id}
                    transfer={transfer}
                    rate={liveTransferRate(transfer, rateById[transfer.id])}
                    index={index + 1}
                    selected={selected?.id === transfer.id}
                    onClick={() => setSelectedId(transfer.id)}
                    onPauseResume={() => void runBackendAction(
                      transfer.status === "paused" ? "Couldn't resume transfer" : "Couldn't pause transfer",
                      () => transfer.status === "paused" ? resume(transfer.id) : pause(transfer.id)
                    )}
                    onCancel={() => void runBackendAction("Couldn't cancel transfer", () => cancel(transfer.id))}
                    onRetry={() => void runBackendAction("Couldn't retry transfer", () => retry(transfer.id))}
                  />
                ))}
              </div>
            </>
          )}
        </div>

        {detailsOpen && (
          <>
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
              <TransferDetails
                transfer={selected}
                rate={liveTransferRate(selected, rateById[selected.id])}
              />
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
                  void runBackendAction(
                    selected.status === "paused" ? "Couldn't resume transfer" : "Couldn't pause transfer",
                    () => selected.status === "paused" ? resume(selected.id) : pause(selected.id)
                  )
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
                onClick={() => selected && void runBackendAction("Couldn't cancel transfer", () => cancel(selected.id))}
              >
                Cancel
              </button>
              <button
                type="button"
                className="ghost-mini-button"
                disabled={!selected || selected.status !== "error"}
                onClick={() => selected && void runBackendAction("Couldn't retry transfer", () => retry(selected.id))}
              >
                <RotateCcw size={14} /> Retry
              </button>
            </div>
            <label className="mt-3 grid grid-cols-[70px_1fr] items-center gap-2 text-[11px] text-text-muted">
              <span>Priority</span>
              <select
                className="ghost-ref-input h-8"
                defaultValue="normal"
                disabled={!selected}
                onChange={(event) => {
                  if (!selected) return;
                  if (event.target.value === "high") void runBackendAction("Couldn't raise transfer priority", () => move(selected.id, "up"));
                  if (event.target.value === "low") void runBackendAction("Couldn't lower transfer priority", () => move(selected.id, "down"));
                  event.currentTarget.value = "normal";
                }}
              >
                <option value="high">High — move up</option>
                <option value="normal">Normal</option>
                <option value="low">Low — move down</option>
              </select>
            </label>
          </div>
        </div>

        <div className="ghost-transfer-lower-grid grid grid-cols-2 gap-4 border-t border-border bg-[#051929] p-4">
          <div
            ref={schedulerRef}
            tabIndex={-1}
            className="ghost-transfer-scheduler rounded-lg border border-border bg-[#071f35] p-4 outline-none focus-visible:ring-2 focus-visible:ring-accent/50"
          >
            <div className="mb-1 flex items-center gap-2">
              <Clock3 size={16} className="text-accent" />
              <strong>Transfer Scheduler</strong>
            </div>
            <div className="mb-3 text-[10.5px] text-text-muted">
              Schedule the selected transfer for later or recurring retry.
            </div>
            <div className="grid grid-cols-4 gap-1 rounded-md border border-border bg-[#051929] p-1">
              {(["off","once","daily","weekly"] as const).map((mode) => (
                <button
                  key={mode}
                  type="button"
                  className={`h-8 rounded text-[11px] capitalize ${scheduleMode === mode ? "bg-accent-strong text-white" : "text-text-muted hover:bg-bg-hover"}`}
                  onClick={() => {
                    setScheduleMode(mode);
                    if (mode === "off") setScheduleArmed(false);
                  }}
                >
                  {mode}
                </button>
              ))}
            </div>
            <div className="mt-3 grid grid-cols-[1fr_120px] gap-2">
              <input
                type="date"
                className="ghost-ref-input h-9"
                value={scheduleDate}
                aria-label={scheduleMode === "once" ? "Run date" : "Start date"}
                title={scheduleMode === "once" ? "Run date" : "Recurring schedule start date"}
                disabled={scheduleMode === "off"}
                onChange={(event) => setScheduleDate(event.target.value)}
              />
              <input
                type="time"
                className="ghost-ref-input h-9"
                value={scheduleTime}
                disabled={scheduleMode === "off"}
                onChange={(event) => setScheduleTime(event.target.value)}
              />
            </div>
            <div className="mt-3 flex items-center gap-2">
              <div className="min-w-0 flex-1 truncate text-[10.5px] text-text-muted">
                {selected
                  ? `Target: ${baseName(selected.source)}`
                  : scheduledTransferId
                    ? "Scheduled target retained"
                    : "Select a transfer first"}
              </div>
              <button
                type="button"
                className="ghost-primary-button"
                disabled={scheduleMode === "off" || !selected}
                onClick={() => {
                  if (!selected) return;
                  setScheduledTransferId(selected.id);
                  setScheduleArmed(true);
                }}
              >
                Set Schedule
              </button>
            </div>
            {scheduleArmed && scheduleMode !== "off" && (
              <div className="mt-2 rounded-md border border-success/25 bg-success/10 px-2.5 py-1.5 text-[10.5px] text-success">
                Schedule active — {scheduleMode}
                {scheduleMode === "once" ? " on " : " from "}
                {scheduleDate} at {scheduleTime}
              </div>
            )}
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
          </>
        )}

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

function Empty() {
  return (
    <div className="grid min-h-48 place-items-center px-6 py-8 text-text-muted">
      <div className="w-full max-w-md rounded-xl border border-border bg-[#071f35] px-6 py-7 text-center shadow-[inset_0_1px_0_rgba(255,255,255,.02)]">
        <Activity size={28} className="mx-auto mb-3 text-accent" />
        <div className="font-semibold text-text">No transfers in this view</div>
        <div className="mx-auto mt-1 max-w-sm text-[12px] leading-5">
          Start an upload or download from Files, or change the active filters.
        </div>
      </div>
    </div>
  );
}

function TransferRow({
  transfer,
  rate,
  index,
  selected,
  onClick,
  onPauseResume,
  onCancel,
  onRetry,
}: {
  transfer: Transfer;
  rate: number;
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

function TransferDetails({ transfer, rate }: { transfer: Transfer; rate: number }) {
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
      <div>Speed: {formatRate(rate)}</div>
      <div>ETA: {etaOf(transfer, rate)}</div>
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
      <div className="relative h-24 overflow-hidden rounded border border-border-subtle bg-[#041522]">
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

function etaOf(transfer: Transfer, speed: number) {
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
