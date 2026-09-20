import { useId, useMemo, useRef, useState } from "react";
import { Copy, FileCode2, Folder, FolderOpen, CopyPlus, Hash, X } from "lucide-react";
import type { DirEntry, SessionId } from "../types";
import { fmtSize, fmtMtime } from "../lib/format";
import { useDialog } from "../hooks/useDialog";
import { useFileUi } from "../context";

interface Props {
  entry: DirEntry;
  sessionId: SessionId | null;
  onClose: () => void;
  onApplied?: () => void;
  onOpenContainingFolder?: () => void;
}

type Tab = "general" | "checksums";

export function PropertiesModal({ entry, sessionId, onClose, onApplied, onOpenContainingFolder }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  const { fs } = useFileUi();
  useDialog(panelRef, { onClose });
  const initialMode = (entry.mode ?? 0o755) & 0o777;
  const [mode, setMode] = useState(initialMode);
  const [tab, setTab] = useState<Tab>("general");
  const [recursive, setRecursive] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checksum, setChecksum] = useState<string | null>(null);
  const [checksumBusy, setChecksumBusy] = useState(false);
  const isDir = entry.kind === "directory";
  const kindLabel = isDir ? "File folder" : entry.kind === "file" ? "File" : entry.kind;
  const octal = useMemo(() => mode.toString(8).padStart(3, "0"), [mode]);

  const setOctal = (value: string) => {
    if (!/^[0-7]{0,3}$/.test(value)) return;
    const parsed = parseInt(value || "0", 8);
    setMode(Number.isFinite(parsed) ? parsed : 0);
  };
  const bit = (m: number) => Boolean(mode & m);
  const toggle = (m: number) => setMode((v) => v ^ m);
  const apply = async () => {
    if (!sessionId || entry.mode == null || mode === initialMode) { onClose(); return; }
    setBusy(true); setError(null);
    try {
      if (recursive && isDir) {
        if (!fs.chmodRecursive) throw new Error("Recursive permissions are not supported by this backend.");
        await fs.chmodRecursive(sessionId, entry.path, mode);
      } else {
        await fs.chmod(sessionId, entry.path, mode);
      }
      onApplied?.(); onClose();
    } catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  };
  const copyPath = async () => { try { await navigator.clipboard.writeText(entry.path); } catch {} };
  const duplicate = async () => {
    if (!sessionId || !fs.duplicate) return;
    setBusy(true); setError(null);
    try { await fs.duplicate(sessionId, entry.path, entry.kind); onApplied?.(); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  };
  const calculateChecksum = async () => {
    if (!sessionId || isDir || !fs.checksum) return;
    setChecksumBusy(true); setError(null);
    try { setChecksum(await fs.checksum(sessionId, entry.path)); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setChecksumBusy(false); }
  };

  return <div className="fixed inset-0 z-modal flex items-center justify-center bg-black/76 p-2" onClick={onClose}>
    <div ref={panelRef} role="dialog" aria-modal="true" aria-labelledby={titleId} className="ghost-properties-dialog flex flex-col overflow-hidden rounded-xl border border-accent/70 bg-[#061a2d] shadow-elev-3" onClick={(e) => e.stopPropagation()}>
      <div className="flex h-[54px] shrink-0 items-center gap-3 border-b border-border px-4"><div className="ghost-dialog-icon"><FileCode2 size={21}/></div><div id={titleId} className="text-[16px] font-semibold">File Properties &amp; Permissions</div><div className="flex-1"/><button onClick={onClose} className="ghost-icon-close"><X size={18}/></button></div>
      <div className="flex h-[43px] shrink-0 items-end gap-2 border-b border-border px-3">
        <button className={`h-[33px] rounded-t-md border px-6 text-[12px] ${tab === "general" ? "border-accent bg-accent/15 text-white" : "border-transparent text-text-muted"}`} onClick={() => setTab("general")}>General</button>
        <button className={`h-[33px] rounded-t-md border px-6 text-[12px] ${tab === "checksums" ? "border-accent bg-accent/15 text-white" : "border-transparent text-text-muted"}`} onClick={() => setTab("checksums")}>Checksums</button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {tab === "general" ? <>
          <div className="mb-4 flex items-center gap-3"><div className="flex h-14 w-14 items-center justify-center rounded-lg bg-indigo-600/55">{isDir ? <Folder size={29}/> : <FileCode2 size={29}/>}</div><div><div className="text-[18px] font-semibold">{entry.name}</div><div className="text-[11px] text-text-muted">{kindLabel}</div></div></div>
          <Detail label="Path"><div className="flex gap-2"><input readOnly className="ghost-ref-input" value={entry.path}/><button className="ghost-mini-button px-2" onClick={() => void copyPath()} title="Copy path"><Copy size={14}/></button></div></Detail>
          <div className="mt-3 grid grid-cols-[92px_1fr] gap-y-2 text-[12px]"><span className="text-text-muted">Size:</span><span>{isDir ? "—" : `${fmtSize(entry.size)} (${entry.size.toLocaleString()} bytes)`}</span><span className="text-text-muted">Created:</span><span>—</span><span className="text-text-muted">Modified:</span><span>{fmtMtime(entry.modified) || "—"}</span><span className="text-text-muted">Accessed:</span><span>—</span></div>
          <div className="my-4 border-t border-border"/>
          <div className="grid grid-cols-[92px_1fr] gap-y-2 text-[12px]"><span className="text-text-muted">Owner:</span><span>—</span><span className="text-text-muted">Group:</span><span>—</span></div>
          {entry.mode != null && <div className="mt-4 border-t border-border pt-4"><div className="mb-3 flex items-center"><strong>Permissions</strong><div className="flex-1"/><span className="mr-2 text-[11px] text-text-muted">Numeric (chmod):</span><input className="ghost-ref-input h-8 w-20" value={octal} onChange={(e) => setOctal(e.target.value)}/></div><div className="grid grid-cols-[1fr_66px_66px_66px] items-center gap-y-2 text-[12px]"><span/><span>Read</span><span>Write</span><span>Execute</span><PermRow label="Owner" bits={[256,128,64]} bit={bit} toggle={toggle}/><PermRow label="Group" bits={[32,16,8]} bit={bit} toggle={toggle}/><PermRow label="Others" bits={[4,2,1]} bit={bit} toggle={toggle}/></div></div>}
          <div className="mt-4 border-t border-border pt-3">
            <div className="mb-2 text-[12px] font-semibold text-text-muted">Advanced</div>
            {isDir && (
              <CheckRow
                label="Apply permissions recursively to all files and folders"
                checked={recursive}
                onChange={setRecursive}
                disabled={!fs.chmodRecursive}
              />
            )}
            <StatusRow label="Transfer timestamps" value="Controlled by the transfer backend" />
            <StatusRow label="Synchronization" value="Configure inclusion in Sync settings" />
          </div>
        </> : <div className="min-h-[360px]"><div className="mb-4 flex items-center gap-3"><div className="ghost-dialog-icon"><Hash size={20}/></div><div><div className="font-semibold">SHA-256</div><div className="text-[11px] text-text-muted">Calculate a cryptographic digest without inventing unsupported results.</div></div></div>{isDir ? <div className="rounded-md border border-border bg-[#051929] p-4 text-[12px] text-text-muted">Checksums are available for files only.</div> : <><button className="ghost-primary-button" disabled={checksumBusy || !fs.checksum} onClick={() => void calculateChecksum()}>{checksumBusy ? "Calculating…" : "Calculate SHA-256"}</button>{checksum && <div className="mt-4 break-all rounded-md border border-border bg-[#041522] p-3 font-mono text-[11px] text-success">{checksum}</div>}{!fs.checksum && <div className="mt-3 text-[11px] text-text-muted">This backend does not provide checksums.</div>}</>}</div>}
        {error && <div className="mt-4 rounded-md border border-danger/35 bg-danger/10 px-3 py-2 text-[11px] text-danger">{error}</div>}
      </div>
      <div className="flex min-h-[62px] shrink-0 items-center border-t border-border bg-[#051929] px-3 py-3"><button className="ghost-mini-button" disabled={!onOpenContainingFolder} onClick={onOpenContainingFolder}><FolderOpen size={14}/> Open Containing Folder</button><button className="ghost-mini-button ml-2" disabled={!fs.duplicate || busy} onClick={() => void duplicate()}><CopyPlus size={14}/> Duplicate</button><div className="flex-1"/><button className="ghost-primary-button" disabled={busy} onClick={() => void apply()}>{busy ? "Applying…" : "Apply"}</button><button className="ghost-mini-button ml-2" onClick={onClose}>Cancel</button></div>
    </div>
  </div>;
}

function Detail({label,children}:{label:string;children:React.ReactNode}) { return <label className="block"><span className="mb-1.5 block text-[10px] text-text-dim">{label}</span>{children}</label>; }
function PermRow({label,bits,bit,toggle}:{label:string;bits:number[];bit:(n:number)=>boolean;toggle:(n:number)=>void}) { return <><span>{label}</span>{bits.map((b)=><span key={b}><input type="checkbox" checked={bit(b)} onChange={() => toggle(b)} className="h-4 w-4 accent-[#169cff]"/></span>)}</>; }
function CheckRow({label,checked,onChange,disabled=false}:{label:string;checked:boolean;onChange:(v:boolean)=>void;disabled?:boolean}) { return <label className={`mb-2 flex items-center gap-2 text-[11.5px] ${disabled ? "opacity-45" : ""}`}><input type="checkbox" disabled={disabled} checked={checked} onChange={(e)=>onChange(e.target.checked)} className="h-4 w-4 accent-[#169cff]"/><span>{label}</span></label>; }
function StatusRow({label,value}:{label:string;value:string}) { return <div className="mb-2 grid grid-cols-[132px_1fr] gap-3 text-[11px]"><span className="text-text-muted">{label}</span><span className="text-text-dim">{value}</span></div>; }
