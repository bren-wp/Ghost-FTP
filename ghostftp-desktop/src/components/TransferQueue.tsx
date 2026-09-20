import { useEffect, useMemo, useState } from "react";
import { AlertCircle, CheckCircle2, Pause, Play, Trash2, X } from "lucide-react";
import { useTransfers } from "@/stores/transfersStore";
import { useConnections } from "@/stores/connectionsStore";
import type { Transfer } from "@/lib/types";

export function TransferQueue() {
  const {
    byId, panelOpen, setPanelOpen, cancel, clearFinished, initListeners, loadInitial,
    pausedAll, pauseAll, resumeAll, pause, resume, retry,
  } = useTransfers();
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const activeProfile = profiles.find((p) => p.id === activeProfileId);
  const [tab, setTab] = useState<"queue" | "failed" | "completed">("queue");
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    let unsub: (() => void) | undefined;
    void (async () => { await loadInitial(); unsub = await initListeners(); })();
    return () => unsub?.();
  }, [initListeners, loadInitial]);
  useEffect(() => { const id = window.setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(id); }, []);

  const transfers = useMemo(() => Object.values(byId).sort((a, b) => b.startedAt - a.startedAt), [byId]);
  const failed = transfers.filter((t) => t.status === "error");
  const completed = transfers.filter((t) => t.status === "done");
  const visible = tab === "failed" ? failed : tab === "completed" ? completed : transfers;
  const active = transfers.filter((t) => t.status === "transferring" || t.status === "queued" || t.status === "paused");

  if (!panelOpen) return <button className="ghost-transfer-collapsed" onClick={() => setPanelOpen(true)}>Transfer Queue ({active.length})</button>;

  return <section className="ghost-bottom-band">
    <div className="ghost-transfer-box">
      <div className="ghost-transfer-tabs">
        <button className={tab === "queue" ? "active" : ""} onClick={() => setTab("queue")}>Transfer Queue ({active.length || transfers.length})</button>
        <button className={tab === "failed" ? "active failed" : "failed"} onClick={() => setTab("failed")}><AlertCircle size={13}/> Failed</button>
        <button className={tab === "completed" ? "active completed" : "completed"} onClick={() => setTab("completed")}><CheckCircle2 size={13}/> Completed</button>
        <span />
        <button title={pausedAll ? "Resume all" : "Pause all"} onClick={() => pausedAll ? resumeAll() : pauseAll()}>{pausedAll ? <Play size={14}/> : <Pause size={14}/>}</button>
        <button title="Clear finished" onClick={clearFinished}><Trash2 size={14}/></button>
        <button title="Hide transfer panel" onClick={() => setPanelOpen(false)}><X size={14}/></button>
      </div>
      <div className="ghost-transfer-head"><span>File</span><span>Direction</span><span>Progress</span><span>Size</span><span>Status</span><span>Speed</span><span>ETA</span><span/></div>
      <div className="ghost-transfer-rows">
        {visible.length === 0 ? <div className="ghost-transfer-empty">No {tab === "queue" ? "transfers" : tab}.</div> : visible.map((t) => <TransferRow key={t.id} transfer={t} now={now} onCancel={() => cancel(t.id)} onPause={() => pause(t.id)} onResume={() => resume(t.id)} onRetry={() => retry(t.id)}/>) }
      </div>
    </div>
    <div className="ghost-server-log">
      <div className="ghost-log-head"><strong>▣ Server Log</strong><span/><button onClick={() => setNow(Date.now())}>Clear</button></div>
      <div className="ghost-log-lines">
        <LogLine text={activeSessionId && activeProfile ? `Connected to ${activeProfile.host}:${activeProfile.port}` : "Ready — waiting for a server connection."}/>
        {activeSessionId && activeProfile?.defaultRemotePath && <LogLine text={`Remote path: ${activeProfile.defaultRemotePath}`} accent/>}
        {active.slice(0,6).reverse().map((t) => <LogLine key={t.id} text={`${t.kind === "upload" ? "Upload" : "Download"} ${t.status}: ${baseName(t.source)}`} accent={t.status === "transferring"}/>) }
      </div>
    </div>
  </section>;
}

function TransferRow({ transfer: t, now, onCancel, onPause, onResume, onRetry }: { transfer: Transfer; now: number; onCancel: () => void; onPause: () => void; onResume: () => void; onRetry: () => void }) {
  const pct = t.size > 0 ? Math.min(100, Math.round((t.transferred / t.size) * 100)) : t.status === "done" ? 100 : 0;
  const elapsed = Math.max(1, (now - t.startedAt) / 1000);
  const speed = t.transferred / elapsed;
  const eta = speed > 0 && t.size > t.transferred ? (t.size - t.transferred) / speed : 0;
  const status = t.status === "done" ? "Completed" : t.status === "error" ? "Failed" : t.status === "paused" ? "Paused" : t.status === "queued" ? "Queued" : "Transferring";
  return <div className="ghost-transfer-row">
    <span className="file">▧ {baseName(t.source)}</span>
    <span className={t.kind}>{t.kind === "upload" ? "↑ Upload" : "↓ Download"}</span>
    <span className="progress"><i><b style={{ width: `${pct}%` }}/></i><em>{pct}%</em></span>
    <span>{formatBytes(t.size)}</span>
    <span className={`status ${t.status}`}>{status}</span>
    <span>{formatSpeed(speed)}</span>
    <span>{formatEta(eta)}</span>
    <span className="actions">{t.status === "paused" ? <button onClick={onResume} title="Resume"><Play size={12}/></button> : (t.status === "transferring" || t.status === "queued") ? <button onClick={onPause} title="Pause"><Pause size={12}/></button> : t.status === "error" ? <button onClick={onRetry} title="Retry">↻</button> : null}<button onClick={onCancel} title="Cancel"><X size={12}/></button></span>
  </div>;
}

function LogLine({ text, accent = false }: { text: string; accent?: boolean }) {
  const time = new Date().toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
  return <div><time>{time}</time><span className={accent ? "accent" : ""}>{text}</span></div>;
}
function baseName(path: string) { return path.replace(/\\/g, "/").split("/").filter(Boolean).pop() || path; }
function formatBytes(n: number) { if (!n) return "—"; const u=["B","KB","MB","GB"]; let v=n,i=0; while(v>=1024&&i<u.length-1){v/=1024;i++;} return `${v>=10||i===0?v.toFixed(0):v.toFixed(1)} ${u[i]}`; }
function formatSpeed(n: number) { return n > 0 ? `${formatBytes(n)}/s` : "—"; }
function formatEta(sec: number) { if (!sec || !Number.isFinite(sec)) return "—"; const s=Math.max(0,Math.round(sec)); const m=Math.floor(s/60); return `${String(Math.floor(m/60)).padStart(2,"0")}:${String(m%60).padStart(2,"0")}:${String(s%60).padStart(2,"0")}`; }
