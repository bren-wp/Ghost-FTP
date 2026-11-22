// Ghost FTP by Brendigo
import { useEffect, useMemo, useRef, useState } from "react";
import { open, save } from "@tauri-apps/plugin-dialog";
import {
  Activity,
  CheckCircle2,
  FolderTree,
  FileDown,
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
import {
  countTransferStatuses,
  filterTransferView,
  sortTransferView,
  type TransferSortOrder,
  recentTransferLog,
  type TransferStatusTab,
  type TransferDirectionFilter,
  type TransferTimeFilter,
} from "@/lib/transferViewModel";
import { useDialog } from "@/hooks/useDialog";
import { messageOf, toastError } from "@/lib/errors";
import { toast } from "@/stores/toastStore";
import { useTransferSchedule } from "@/stores/transferScheduleStore";
import { mostRecentFailedTransferId } from "@/lib/transferFailureNavigation";
import { Tab, Empty, TransferRow, TransferDetails, BandwidthChart, baseName, type BandwidthSample } from "./TransferCenterParts";

interface Props {
  onClose: () => void;
}

export function TransferCenterDialog({ onClose }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const byId = useTransfers((state) => state.byId);
  const rateById = useTransfers((state) => state.rateById);
  const clearCompleted = useTransfers((state) => state.clearCompleted);
  const exportHistory = useTransfers((state) => state.exportHistory);
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

  const [tab, setTab] = useState<TransferStatusTab>("all");
  const [query, setQuery] = useState("");
  const [directionFilter, setDirectionFilter] = useState<TransferDirectionFilter>("all");
  const [timeFilter, setTimeFilter] = useState<TransferTimeFilter>("all");
  const [sortOrder, setSortOrder] = useState<TransferSortOrder>("original");
  const [historyClock, setHistoryClock] = useState(() => Date.now() / 1000);
  useEffect(() => {
    if (timeFilter === "all") return;
    setHistoryClock(Date.now() / 1000);
    // Rolling history must expire even when the transfer queue is idle.
    const timer = window.setInterval(() => setHistoryClock(Date.now() / 1000), 60_000);
    return () => window.clearInterval(timer);
  }, [timeFilter]);
  const [moreOpen, setMoreOpen] = useState(false);
  const moreRef = useRef<HTMLDivElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const schedulerRef = useRef<HTMLDivElement>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const focusedFailureRowRef = useRef<HTMLDivElement>(null);
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
  const backendActionInFlight = useRef(new Set<string>());
  const [backendActionBusy, setBackendActionBusy] = useState<Set<string>>(
    () => new Set()
  );

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

  const exportTransferHistory = async () => {
    const path = await save({
      title: "Export transfer history",
      defaultPath: "GhostFTP-transfer-history.csv",
      filters: [{ name: "CSV", extensions: ["csv"] }],
    });
    if (!path) return;

    try {
      const count = await exportHistory(path);
      toast.success(
        "Transfer history exported",
        `${count} transfer${count === 1 ? "" : "s"} saved`,
      );
    } catch (error) {
      toastError(error, "Couldn't export transfer history");
    }
  };

  const transfers = useMemo(() => Object.values(byId), [byId]);
  const { completed, failed, active: activeFilterCount } = useMemo(
    () => countTransferStatuses(transfers),
    [transfers]
  );
  const filtered = useMemo(
    () => filterTransferView(transfers, {
      tab,
      query,
      direction: directionFilter,
      time: timeFilter,
      nowSeconds: historyClock,
    }),
    [transfers, tab, query, directionFilter, timeFilter, historyClock]
  );


  const sortedTransfers = useMemo(
    () => sortTransferView(filtered, sortOrder),
    [filtered, sortOrder]
  );

  // Review the latest real backend failure in one action, regardless of
  // previous text/direction/time filters. This never retries automatically.
  const reviewLatestFailure = () => {
    const id = mostRecentFailedTransferId(transfers);
    setQuery("");
    setDirectionFilter("all");
    setTimeFilter("all");
    setTab("failed");
    setSelectedId(id);
    if (id) setDetailsOpen(true);
  };

  // Keep the selected transfer stable when filters/search change. Actions and
  // details must continue to refer to the real queue item, while the effect
  // below clears selection once that item is no longer visible.
  const selected = selectedId ? byId[selectedId] ?? null : null;

  useEffect(() => {
    if (selectedId && !filtered.some((transfer) => transfer.id === selectedId)) {
      setSelectedId(null);
    }
  }, [filtered, selectedId]);

  // Keep the selected failed native queue item visible in long histories.
  useEffect(() => {
    if (tab === "failed" && selectedId) {
      focusedFailureRowRef.current?.scrollIntoView({ block: "nearest" });
    }
  }, [tab, selectedId]);

  const logTransfers = useMemo(
    () => recentTransferLog(transfers, logClearedAt),
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
    key: string,
    failureTitle: string,
    operation: () => Promise<void>
  ) => {
    if (backendActionInFlight.current.has(key)) return;
    backendActionInFlight.current.add(key);
    setBackendActionBusy(new Set(backendActionInFlight.current));
    try {
      await operation();
    } catch (error) {
      toastError(error, failureTitle);
    } finally {
      backendActionInFlight.current.delete(key);
      setBackendActionBusy(new Set(backendActionInFlight.current));
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

  // These counters describe the real backend snapshot, never seeded sample
  // jobs. Reuse the same totals as the filter tabs and queue actions.
  const activeCount = activeFilterCount;

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
                  disabled={(activeCount === 0 && !pausedAll) || backendActionBusy.has("queue")}
                  aria-busy={backendActionBusy.has("queue")}
                  onClick={() => {
                    setMoreOpen(false);
                    void runBackendAction(
                      "queue",
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
                  onClick={() => {
                    setMoreOpen(false);
                    void exportTransferHistory();
                  }}
                >
                  <FileDown size={14}/>
                  <span>Export History…</span>
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

        <section className="ghost-transfer-overview" aria-label="Live transfer overview">
          {([
            { filter: "all", title: "Total transfers", count: transfers.length, icon: Activity },
            { filter: "active", title: "Active & queued", count: activeFilterCount, icon: Play },
            { filter: "completed", title: "Completed", count: completed, icon: CheckCircle2 },
            { filter: "failed", title: "Failed", count: failed, icon: XCircle },
          ] as const).map(({ filter, title, count, icon: Icon }) => (
            <button
              key={filter}
              type="button"
              className={`ghost-transfer-overview-card${tab === filter ? " is-selected" : ""}`}
              onClick={() => filter === "failed" ? reviewLatestFailure() : setTab(filter)}
              aria-pressed={tab === filter}
              aria-label={`${title}: ${count}. Show ${filter === "all" ? "all" : filter} transfers`}
            >
              <span className="ghost-transfer-overview-label">
                <Icon size={16} aria-hidden="true" />
                {title}
              </span>
              <strong className="ghost-transfer-overview-count">{count}</strong>
            </button>
          ))}
        </section>

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
            onClick={reviewLatestFailure}
            label={`Failed (${failed})`}
            icon={<XCircle size={14} />}
          />
          <div className="flex-1" />
          <button
            type="button"
            className="ghost-mini-button"
            disabled={failed === 0}
            aria-label={`Review ${failed} failed transfers, newest first`}
            onClick={reviewLatestFailure}
          >
            <XCircle size={14} aria-hidden="true" /> Review failures ({failed})
          </button>
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
            onChange={(event) => setDirectionFilter(event.target.value as TransferDirectionFilter)}
          >
            <option value="all">All Directions</option>
            <option value="upload">Uploads</option>
            <option value="download">Downloads</option>
          </select>
          <select
            className="ghost-ref-input h-8 w-28"
            aria-label="Transfer time filter"
            value={timeFilter}
            onChange={(event) => setTimeFilter(event.target.value as TransferTimeFilter)}
          >
            <option value="all">Any Time</option>
            <option value="hour">Last Hour</option>
            <option value="day">Last 24h</option>
            <option value="week">Last 7 days</option>
            <option value="month">Last 30 days</option>
          </select>
          <select
            className="ghost-ref-input h-8 w-36"
            aria-label="Transfer sort order"
            value={sortOrder}
            onChange={(event) => setSortOrder(event.target.value as TransferSortOrder)}
          >
            <option value="original">Original order</option>
            <option value="newest">Newest first</option>
            <option value="oldest">Oldest first</option>
            <option value="largest">Largest first</option>
            <option value="name">File name (A–Z)</option>
          </select>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          {filtered.length === 0 ? (
            <Empty
              hasHistory={transfers.length > 0}
              onClearFilters={() => {
                setTab("all");
                setQuery("");
                setDirectionFilter("all");
                setTimeFilter("all");
              }}
            />
          ) : (
            <>
              <div className="sticky top-0 z-10 grid min-w-[920px] grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px_106px] border-b border-border bg-[#061a2d] px-2 py-2 text-[10px] uppercase tracking-wider text-text-dim">
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
                {sortedTransfers.map((transfer, index) => (
                  <div
                    key={transfer.id}
                    ref={selected?.id === transfer.id && transfer.status === "error" ? focusedFailureRowRef : undefined}
                  >
                    <TransferRow
                    transfer={transfer}
                    rate={liveTransferRate(transfer, rateById[transfer.id])}
                    index={index + 1}
                    selected={selected?.id === transfer.id}
                    onClick={() => setSelectedId(transfer.id)}
                    busy={backendActionBusy.has(`transfer:${transfer.id}`)}
                    onPauseResume={() => void runBackendAction(
                      `transfer:${transfer.id}`,
                      transfer.status === "paused" ? "Couldn't resume transfer" : "Couldn't pause transfer",
                      () => transfer.status === "paused" ? resume(transfer.id) : pause(transfer.id)
                    )}
                    onCancel={() => void runBackendAction(
                      `transfer:${transfer.id}`,
                      "Couldn't cancel transfer",
                      () => cancel(transfer.id)
                    )}
                    onRetry={() => void runBackendAction(
                      `transfer:${transfer.id}`,
                      "Couldn't retry transfer",
                      () => retry(transfer.id)
                    )}
                  />
                  {selected?.id === transfer.id && transfer.status === "error" && (
                    <div
                      className="flex flex-wrap items-center gap-3 border-b border-danger/30 bg-danger/5 px-4 py-3 text-[11px]"
                      role="group"
                      aria-label="Selected transfer failure details"
                    >
                      <div className="min-w-0 flex-1">
                        <strong className="text-danger">Why this transfer failed</strong>
                        <p className="mt-1 break-words text-text-muted">
                          {transfer.error ? messageOf(transfer.error) : "The backend did not provide an error reason."}
                        </p>
                        <p className="mt-1 text-text-dim">
                          Check the connection, remote path and permissions before retrying.
                        </p>
                      </div>
                      <button
                        type="button"
                        className="ghost-mini-button"
                        disabled={backendActionBusy.has(`transfer:${transfer.id}`)}
                        aria-busy={backendActionBusy.has(`transfer:${transfer.id}`)}
                        onClick={() => void runBackendAction(
                          `transfer:${transfer.id}`,
                          "Couldn't retry transfer",
                          () => retry(transfer.id)
                        )}
                      >
                        <RotateCcw size={13} aria-hidden="true" /> Retry this transfer
                      </button>
                    </div>
                  )}
                  </div>
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
                disabled={!selected || !canPauseSelected || (selected ? backendActionBusy.has(`transfer:${selected.id}`) : false)}
                aria-busy={selected ? backendActionBusy.has(`transfer:${selected.id}`) : false}
                onClick={() =>
                  selected &&
                  void runBackendAction(
                    `transfer:${selected.id}`,
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
                disabled={!selected || !canCancelSelected || (selected ? backendActionBusy.has(`transfer:${selected.id}`) : false)}
                aria-busy={selected ? backendActionBusy.has(`transfer:${selected.id}`) : false}
                onClick={() => selected && void runBackendAction(
                  `transfer:${selected.id}`,
                  "Couldn't cancel transfer",
                  () => cancel(selected.id)
                )}
              >
                Cancel
              </button>
              <button
                type="button"
                className="ghost-mini-button"
                disabled={!selected || selected.status !== "error" || (selected ? backendActionBusy.has(`transfer:${selected.id}`) : false)}
                aria-busy={selected ? backendActionBusy.has(`transfer:${selected.id}`) : false}
                onClick={() => selected && void runBackendAction(
                  `transfer:${selected.id}`,
                  "Couldn't retry transfer",
                  () => retry(selected.id)
                )}
              >
                <RotateCcw size={14} /> Retry
              </button>
            </div>
            <label className="mt-3 grid grid-cols-[70px_1fr] items-center gap-2 text-[11px] text-text-muted">
              <span>Priority</span>
              <select
                className="ghost-ref-input h-8"
                defaultValue="normal"
                disabled={!selected || (selected ? backendActionBusy.has(`transfer:${selected.id}`) : false)}
                onChange={(event) => {
                  if (!selected) return;
                  if (event.target.value === "high") void runBackendAction(
                    `transfer:${selected.id}`,
                    "Couldn't raise transfer priority",
                    () => move(selected.id, "up")
                  );
                  if (event.target.value === "low") void runBackendAction(
                    `transfer:${selected.id}`,
                    "Couldn't lower transfer priority",
                    () => move(selected.id, "down")
                  );
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

