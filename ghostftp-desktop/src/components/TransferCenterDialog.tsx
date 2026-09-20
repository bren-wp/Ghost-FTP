import { useMemo, useState } from "react";
import {
  ArrowDown,
  ArrowUp,
  CalendarClock,
  CheckCircle2,
  Bookmark,
  FolderTree,
  HelpCircle,
  Languages,
  Server,
  Settings,
  Wrench,
  Clock3,
  MoreHorizontal,
  Pause,
  Play,
  Plus,
  RotateCcw,
  Search,
  Trash2,
  XCircle,
} from "lucide-react";
import { useTransfers } from "@/stores/transfersStore";
import { toast } from "@/stores/toastStore";
import type { Transfer } from "@/lib/types";
import { ReferenceWindowControls } from "./ReferenceWindowChrome";
import { GhostMark } from "./GhostBrand";
import { getLocale, setLocale } from "@/lib/i18n";
import { useLayout } from "@/stores/layoutStore";

type FilterTab = "all" | "upload" | "download" | "completed" | "failed";
type ScheduleMode = "off" | "once" | "daily" | "weekly";

interface Props { onClose: () => void }

export function TransferCenterDialog({ onClose }: Props) {
  const byId = useTransfers((s) => s.byId);
  const clearFinished = useTransfers((s) => s.clearFinished);
  const pauseAll = useTransfers((s) => s.pauseAll);
  const resumeAll = useTransfers((s) => s.resumeAll);
  const pausedAll = useTransfers((s) => s.pausedAll);
  const cancel = useTransfers((s) => s.cancel);
  const pause = useTransfers((s) => s.pause);
  const resume = useTransfers((s) => s.resume);
  const retry = useTransfers((s) => s.retry);

  const [tab, setTab] = useState<FilterTab>("all");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [scheduleMode, setScheduleMode] = useState<ScheduleMode>("off");
  const [scheduleDate, setScheduleDate] = useState("");
  const [scheduleTime, setScheduleTime] = useState("13:00");

  const transfers = useMemo(() => Object.values(byId), [byId]);
  const completed = transfers.filter((t) => t.status === "done" || t.status === "skipped").length;
  const failed = transfers.filter((t) => t.status === "error").length;
  const selected = (selectedId ? byId[selectedId] : undefined) ?? transfers[0];

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return transfers.filter((t) => {
      const tabMatch =
        tab === "all" ||
        (tab === "upload" && t.kind === "upload") ||
        (tab === "download" && t.kind === "download") ||
        (tab === "completed" && (t.status === "done" || t.status === "skipped")) ||
        (tab === "failed" && t.status === "error");
      if (!tabMatch) return false;
      if (!q) return true;
      return `${t.source} ${t.destination} ${t.status}`.toLowerCase().includes(q);
    });
  }, [transfers, tab, query]);

  const addTransfer = () => {
    window.dispatchEvent(new CustomEvent("ghostftp:toolbar-action", { detail: { action: "upload" } }));
    onClose();
  };

  const saveSchedule = () => {
    const record = { mode: scheduleMode, date: scheduleDate, time: scheduleTime };
    localStorage.setItem("ghostftp.transferSchedule", JSON.stringify(record));
    toast.success("Transfer schedule saved", scheduleMode === "off" ? "Scheduling is disabled." : `${scheduleMode} at ${scheduleTime}`);
  };

  return (
    <div className="ghost-standalone-view fixed inset-0 z-modal bg-[#041425]" role="dialog" aria-modal="true">
      <div className="ghost-transfer-center flex h-full w-full flex-col overflow-hidden bg-[#061a2d]">
        <TransferCenterTitlebar onClose={onClose} />
        <div className="ghost-transfer-center-heading flex h-[78px] shrink-0 items-center gap-3 border-b border-border px-4">
          <div>
            <div className="text-xl font-semibold">Transfer Center</div>
            <div className="text-[12px] text-text-muted">Manage all file transfers in one place. Upload, download, monitor and automate.</div>
          </div>
          <div className="flex-1" />
          <button className="ghost-primary-button" onClick={addTransfer}><Plus size={14} /> Add Transfer</button>
          <button className="ghost-mini-button" onClick={() => document.getElementById("transfer-scheduler")?.scrollIntoView({ behavior: "smooth" })}><CalendarClock size={14} /> Schedule</button>
          <button className="ghost-mini-button" onClick={clearFinished}><Trash2 size={14} /> Clear Completed</button>
          <button className="ghost-mini-button" title="More transfer actions"><MoreHorizontal size={14} /></button>
        </div>

        <div className="flex items-center gap-2 border-b border-border px-4 py-3 text-[12px]">
          <Tab active={tab === "all"} onClick={() => setTab("all")} label={`All Transfers (${transfers.length})`} />
          <Tab active={tab === "upload"} onClick={() => setTab("upload")} label={`Uploads (${transfers.filter((t) => t.kind === "upload").length})`} icon={<ArrowUp size={14} />} />
          <Tab active={tab === "download"} onClick={() => setTab("download")} label={`Downloads (${transfers.filter((t) => t.kind === "download").length})`} icon={<ArrowDown size={14} />} />
          <Tab active={tab === "completed"} onClick={() => setTab("completed")} label={`Completed (${completed})`} icon={<CheckCircle2 size={14} />} />
          <Tab active={tab === "failed"} onClick={() => setTab("failed")} label={`Failed (${failed})`} icon={<XCircle size={14} />} />
          <div className="flex-1" />
          <div className="relative">
            <Search size={13} className="absolute left-2.5 top-2.5 text-text-dim" />
            <input className="ghost-ref-input h-8 w-52 pl-8" placeholder="Filter transfers…" value={query} onChange={(e) => setQuery(e.target.value)} />
          </div>
          <select className="ghost-ref-input h-8 w-32" value={tab} onChange={(e) => setTab(e.target.value as FilterTab)}>
            <option value="all">All Status</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="upload">Uploads</option>
            <option value="download">Downloads</option>
          </select>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          <div className="grid grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px] border-b border-border px-2 py-2 text-[10px] uppercase tracking-wider text-text-dim">
            <span>#</span><span>Name</span><span>Direction</span><span>Size</span><span>Progress</span><span>Speed</span><span>ETA</span><span>Status</span>
          </div>
          {filtered.length === 0 ? <Empty /> : filtered.map((t, i) => <TransferRow key={t.id} t={t} index={i + 1} selected={selected?.id === t.id} onClick={() => setSelectedId(t.id)} />)}
        </div>

        <div className="grid grid-cols-[1.55fr_.75fr] gap-4 border-t border-border bg-[#051929] p-4">
          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center"><strong>Bandwidth Usage</strong><div className="flex-1" /><span className="text-[11px] text-text-muted">Upload / Download</span></div>
            <div className="relative h-28 overflow-hidden rounded bg-[#041522]">
              <div className="absolute inset-0 bg-[repeating-linear-gradient(0deg,transparent_0_27px,rgba(31,80,115,.36)_28px)]" />
              <svg viewBox="0 0 600 120" preserveAspectRatio="none" className="absolute inset-0 h-full w-full" aria-hidden="true">
                <polyline fill="none" stroke="rgb(42 170 255)" strokeWidth="2.4" points="0,90 48,72 96,81 150,48 205,68 255,44 310,62 365,38 420,57 470,42 520,70 560,52 600,59" />
                <polyline fill="none" stroke="rgb(36 218 139)" strokeWidth="2.2" points="0,105 48,93 96,101 150,78 205,91 255,68 310,82 365,62 420,86 470,71 520,94 560,82 600,90" />
              </svg>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center gap-2"><Clock3 size={16} className="text-accent" /><strong>Active Transfer Details</strong></div>
            {selected ? <TransferDetails t={selected} /> : <div className="text-[12px] text-text-muted">Select a transfer to see details.</div>}
            <div className="mt-4 flex flex-wrap gap-2">
              <button className="ghost-primary-button" disabled={!selected || selected.status === "done" || selected.status === "skipped"} onClick={() => selected && void (selected.status === "paused" ? resume(selected.id) : pause(selected.id))}>{selected?.status === "paused" ? <Play size={14} /> : <Pause size={14} />} {selected?.status === "paused" ? "Resume" : "Pause"}</button>
              <button className="ghost-mini-button" disabled={!selected} onClick={() => selected && void cancel(selected.id)}>Cancel</button>
              <button className="ghost-mini-button" disabled={!selected || selected.status !== "error"} onClick={() => selected && void retry(selected.id)}><RotateCcw size={14} /> Retry</button>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 border-t border-border bg-[#051929] p-4">
          <div id="transfer-scheduler" className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center gap-2"><CalendarClock size={16} className="text-accent" /><strong>Transfer Scheduler</strong></div>
            <div className="mb-3 flex gap-2">
              {(["off", "once", "daily", "weekly"] as ScheduleMode[]).map((mode) => <button key={mode} className={`ghost-mini-button capitalize ${scheduleMode === mode ? "border-accent bg-accent/15 text-white" : ""}`} onClick={() => setScheduleMode(mode)}>{mode}</button>)}
            </div>
            <div className="flex gap-2"><input className="ghost-ref-input h-8" type="date" value={scheduleDate} onChange={(e) => setScheduleDate(e.target.value)} disabled={scheduleMode === "off" || scheduleMode === "daily" || scheduleMode === "weekly"} /><input className="ghost-ref-input h-8" type="time" value={scheduleTime} onChange={(e) => setScheduleTime(e.target.value)} disabled={scheduleMode === "off"} /><button className="ghost-mini-button" onClick={saveSchedule}>Set Schedule</button></div>
          </div>
          <div className="rounded-lg border border-border bg-[#071f35] p-4">
            <div className="mb-3 flex items-center"><strong>Transfer Log</strong><div className="flex-1" /><button className="ghost-mini-button" onClick={() => toast.info("Transfer log", "Live transfer events remain available in the queue history.")}>Clear Log</button></div>
            <div className="max-h-28 overflow-auto font-mono text-[10.5px] leading-5 text-text-muted">
              {transfers.slice(-8).reverse().map((t) => <div key={t.id}><span className={t.status === "error" ? "text-danger" : t.status === "done" ? "text-success" : "text-accent"}>{t.status.toUpperCase()}</span> — {baseName(t.source)} → {baseName(t.destination)}</div>)}
              {transfers.length === 0 && <div>No transfer events yet.</div>}
            </div>
          </div>
        </div>

        <div className="flex items-center border-t border-border bg-[#041522] px-4 py-3">
          <button className="ghost-mini-button" onClick={() => void (pausedAll ? resumeAll() : pauseAll())}>{pausedAll ? <Play size={14} /> : <Pause size={14} />} {pausedAll ? "Resume All" : "Pause All"}</button>
          <div className="flex-1" />
          <div className="text-[11px] text-text-muted">Active: {transfers.filter((t) => t.status === "transferring" || t.status === "queued").length} &nbsp; • &nbsp; Queue: {transfers.length}</div>
        </div>
      </div>
    </div>
  );
}

function TransferCenterTitlebar({ onClose }: { onClose: () => void }) {
  const openDialog = useLayout((state) => state.openDialog);
  const locale = getLocale();
  const item = (label: string, icon: React.ReactNode, action: () => void, active = false) => (
    <button className={`ghost-transfer-nav-item ${active ? "active" : ""}`} onClick={action}>{icon}<span>{label}</span></button>
  );
  return <div className="ghost-transfer-titlebar" data-tauri-drag-region>
    <div className="ghost-transfer-brand" data-tauri-drag-region><GhostMark size={34}/><div><div className="font-semibold">Ghost FTP</div><div>Transfer Center</div></div></div>
    <nav className="ghost-transfer-nav" aria-label="Transfer Center navigation">
      {item("Sites", <FolderTree size={15}/>, onClose)}
      {item("Transfers", <ArrowUp size={15}/>, () => {}, true)}
      {item("Server", <Server size={15}/>, () => openDialog("siteManager"))}
      {item("Bookmarks", <Bookmark size={15}/>, () => openDialog("siteManager"))}
      {item("Tools", <Wrench size={15}/>, () => openDialog("settings"))}
      {item("Settings", <Settings size={15}/>, () => openDialog("settings"))}
      {item("Help", <HelpCircle size={15}/>, () => openDialog("about"))}
    </nav>
    <label className="ghost-transfer-language"><Languages size={14}/><select aria-label="Language" value={locale} onChange={(event) => setLocale(event.target.value as any)}><option value="en">English (English)</option><option value="hr">Hrvatski (Croatian)</option><option value="de">Deutsch (German)</option><option value="fr">Français (French)</option><option value="es">Español (Spanish)</option><option value="it">Italiano (Italian)</option><option value="pt">Português (Portuguese)</option><option value="nl">Nederlands (Dutch)</option><option value="pl">Polski (Polish)</option><option value="sl">Slovenščina (Slovenian)</option><option value="sr">Srpski (Serbian)</option><option value="bs">Bosanski (Bosnian)</option><option value="mk">Македонски (Macedonian)</option><option value="sq">Shqip (Albanian)</option></select></label>
    <ReferenceWindowControls/>
  </div>;
}

function Tab({ label, icon, active = false, onClick }: { label: string; icon?: React.ReactNode; active?: boolean; onClick: () => void }) {
  return <button onClick={onClick} className={`flex items-center gap-1.5 rounded-md border px-3 py-2 ${active ? "border-accent bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover"}`}>{icon}{label}</button>;
}

function Empty() {
  return <div className="grid min-h-56 place-items-center text-text-muted"><div className="text-center"><ArrowUp size={30} className="mx-auto mb-3 text-accent" /><div className="font-semibold text-text">No matching transfers</div><div className="mt-1 text-[12px]">Start an upload or download, or change the current filters.</div></div></div>;
}

function TransferRow({ t, index, selected, onClick }: { t: Transfer; index: number; selected: boolean; onClick: () => void }) {
  const pct = t.size > 0 ? Math.max(0, Math.min(100, (t.transferred / t.size) * 100)) : 0;
  return <button onClick={onClick} className={`grid w-full grid-cols-[36px_minmax(220px,1.4fr)_95px_90px_160px_90px_90px_100px] items-center border-0 border-b border-border-subtle px-2 py-2.5 text-left text-[11.5px] ${selected ? "bg-accent/10" : "bg-transparent hover:bg-bg-hover"}`}>
    <span className="text-text-dim">{index}</span><span className="truncate font-medium">{baseName(t.source)}</span><span className={t.kind === "upload" ? "text-accent" : "text-success"}>{t.kind === "upload" ? "↑ Upload" : "↓ Download"}</span><span className="text-text-muted">{formatBytes(t.size || 0)}</span><span className="flex items-center gap-2"><span className="h-3 flex-1 overflow-hidden rounded border border-border bg-[#03111d]"><span className={`block h-full ${t.status === "error" ? "bg-danger" : t.status === "done" ? "bg-success" : "bg-accent-strong"}`} style={{ width: `${pct}%` }} /></span><span className="w-8 text-right text-[10px]">{pct.toFixed(0)}%</span></span><span className="text-text-muted">{formatBytes(speedOf(t))}/s</span><span className="text-text-muted">{etaOf(t)}</span><span className={t.status === "error" ? "text-danger" : t.status === "done" ? "text-success" : t.status === "paused" ? "text-warning" : "text-accent"}>{t.status}</span>
  </button>;
}

function TransferDetails({ t }: { t: Transfer }) {
  const pct = t.size > 0 ? Math.max(0, Math.min(100, (t.transferred / t.size) * 100)) : 0;
  return <div className="space-y-1 text-[11px] text-text-muted"><div className="truncate font-semibold text-text">{baseName(t.source)}</div><div className="h-3 overflow-hidden rounded border border-border bg-[#03111d]"><div className="h-full bg-accent-strong" style={{ width: `${pct}%` }} /></div><div>Transferred: {formatBytes(t.transferred)} / {formatBytes(t.size)}</div><div>Speed: {formatBytes(speedOf(t))}/s</div><div>ETA: {etaOf(t)}</div><div>Status: <span className="text-accent">{t.status}</span></div>{t.error && <div className="text-danger">{t.error}</div>}</div>;
}

function speedOf(t: Transfer) { const secs = Math.max(1, (Date.now() / 1000) - t.startedAt); return t.status === "transferring" ? t.transferred / secs : 0; }
function etaOf(t: Transfer) { const speed = speedOf(t); if (!speed || t.status !== "transferring" || t.size <= t.transferred) return "—"; const secs = Math.max(0, Math.ceil((t.size - t.transferred) / speed)); const m = Math.floor(secs / 60); const s = secs % 60; return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`; }
function baseName(p: string) { return p.split(/[\\/]/).filter(Boolean).pop() || p || "transfer"; }
function formatBytes(n: number) { if (!n) return "—"; const u = ["B", "KB", "MB", "GB", "TB"]; let i = 0, v = n; while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; } return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${u[i]}`; }
