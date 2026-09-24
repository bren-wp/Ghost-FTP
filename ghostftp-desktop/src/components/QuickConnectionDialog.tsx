import { useEffect, useMemo, useRef, useState } from "react";
import { open as openNativeDialog } from "@tauri-apps/plugin-dialog";
import { Bookmark, ChevronDown, Eye, EyeOff, FolderOpen, KeyRound, Link2, Radio, Settings2, X } from "lucide-react";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";
import { useConnections } from "@/stores/connectionsStore";
import { useDialog } from "@/hooks/useDialog";
import { GhostMark } from "./GhostBrand";
import { ipc } from "@/lib/ipc";
import { messageOf, toastError } from "@/lib/errors";

interface Props {
  prefill?: Partial<ConnectionProfile> | null;
  onClose: () => void;
  saveByDefault?: boolean;
  cancelLabel?: string;
}

export function QuickConnectionDialog({ prefill, onClose, saveByDefault = false, cancelLabel = "Cancel" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const saveProfile = useConnections((s) => s.saveProfile);
  const connectProfile = useConnections((s) => s.connect);
  const connectTemporary = useConnections((s) => s.connectTemporary);
  const initialProtocol = (prefill?.protocol === "ftp" || prefill?.protocol === "ftps" || prefill?.protocol === "sftp" ? prefill.protocol : "sftp") as Protocol;
  const [protocol, setProtocol] = useState<Protocol>(initialProtocol);
  const [host, setHost] = useState(prefill?.host ?? "");
  const [port, setPort] = useState(prefill?.port ?? PROTOCOL_DEFAULT_PORT[initialProtocol]);
  const [username, setUsername] = useState(prefill?.username ?? "");
  const [password, setPassword] = useState(prefill?.auth?.kind === "password" ? prefill.auth.password : "");
  const [showPassword, setShowPassword] = useState(false);
  const [useKey, setUseKey] = useState(prefill?.auth?.kind === "key");
  const [keyPath, setKeyPath] = useState(prefill?.auth?.kind === "key" ? prefill.auth.path : "");
  const [keyPassphrase, setKeyPassphrase] = useState(prefill?.auth?.kind === "key" ? (prefill.auth.passphrase ?? "") : "");
  const [showKeyPassphrase, setShowKeyPassphrase] = useState(false);
  const [remember, setRemember] = useState(saveByDefault);
  const [advanced, setAdvanced] = useState(false);
  const [remotePath, setRemotePath] = useState(prefill?.defaultRemotePath ?? ".");
  const [name, setName] = useState(prefill?.name ?? "");
  const [busy, setBusy] = useState(false);
  const [keyPicking, setKeyPicking] = useState(false);
  const [testStatus, setTestStatus] = useState<"idle"|"testing"|"ok"|"error">("idle");
  const actionBusy = busy || keyPicking || testStatus === "testing";
  const closeIfIdle = () => {
    if (!actionBusy) onClose();
  };
  useDialog(panelRef, { onClose: closeIfIdle });

  const validPort = Number.isInteger(port) && port >= 1 && port <= 65535;
  const keyReady = protocol !== "sftp" || !useKey || keyPath.trim().length > 0;
  const canConnect =
    host.trim().length > 0 &&
    username.trim().length > 0 &&
    validPort &&
    keyReady;
  const effectiveName = useMemo(() => name.trim() || host.trim() || "New server", [name, host]);

  const makeProfile = (): ConnectionProfile => ({
    id: crypto.randomUUID(),
    name: effectiveName,
    protocol,
    host: host.trim(),
    port,
    username: username.trim(),
    auth: useKey && protocol === "sftp" ? { kind: "key", path: keyPath.trim(), passphrase: keyPassphrase || undefined } : { kind: "password", password },
    defaultRemotePath: remotePath.trim() || ".",
    autoConnect: false,
    group: remember ? "My Sites" : undefined,
  });

  useEffect(() => {
    setTestStatus("idle");
  }, [protocol, host, port, username, password, useKey, keyPath, keyPassphrase, remotePath]);

  const submit = async (connectNow: boolean) => {
    if (!canConnect || actionBusy) return;
    setBusy(true);
    const profile = makeProfile();
    const ephemeral = !remember;
    try {
      if (connectNow && ephemeral) {
        try {
          await connectTemporary(profile);
        } catch (error) {
          // connectionsStore already presents the structured connection error.
          console.debug(
            "Quick connection failure was surfaced by the connections store",
            messageOf(error)
          );
          return;
        }
      } else {
        try {
          await saveProfile(profile);
        } catch (error) {
          toastError(error, `Couldn't save ${profile.name}`);
          return;
        }
        if (connectNow) {
          try {
            await connectProfile(profile.id);
          } catch (error) {
            // The profile is safely persisted; connectionsStore already surfaced
            // the real FTP/FTPS/SFTP connection failure.
            console.debug(
              "Saved-profile connection failure was surfaced by the connections store",
              messageOf(error)
            );
            return;
          }
        }
      }
      onClose();
    } finally {
      setBusy(false);
    }
  };

  const chooseKey = async () => {
    if (actionBusy) return;
    setKeyPicking(true);
    try {
      const picked = await openNativeDialog({ multiple: false, directory: false, title: "Select SSH private key" });
      if (typeof picked === "string") setKeyPath(picked);
    } catch (error) {
      toastError(error, "Couldn't open the SSH private key picker");
    } finally {
      setKeyPicking(false);
    }
  };

  const testConnection = async () => {
    if (!canConnect || actionBusy) return;
    setTestStatus("testing");
    try {
      await ipc.testEphemeralConnection(makeProfile());
      setTestStatus("ok");
    } catch (error) {
      setTestStatus("error");
      toastError(error, "Connection test failed");
    }
  };

  return (
    <div className="ghost-transient-overlay fixed inset-0 z-modal flex items-center justify-center bg-black/70 p-4">
      <div ref={panelRef} role="dialog" aria-modal="true" className="ghost-new-connection-dialog flex w-[min(752px,94vw)] flex-col overflow-hidden rounded-xl border border-accent/70 bg-[#061a2d] shadow-[0_0_0_1px_rgba(65,181,255,.08),0_30px_90px_rgba(0,0,0,.7),0_0_38px_rgba(31,149,255,.15)]">
        <div className="ghost-new-connection-head flex shrink-0 items-center gap-3 border-b border-border px-5 py-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#0b3151]"><GhostMark size={34}/></div>
          <div><div className="text-[19px] font-semibold">New Connection</div><div className="text-[12px] text-text-muted">Enter your server details, then connect or optionally save it in Sites.</div></div>
          <div className="flex-1"/><button type="button" disabled={actionBusy} onClick={closeIfIdle} className="rounded-md p-2 text-text-muted hover:bg-bg-hover hover:text-white disabled:cursor-not-allowed disabled:opacity-50"><X size={18}/></button>
        </div>

        <div className="ghost-new-connection-body min-h-0 overflow-y-auto p-5">
          {remember && <Field label="Site name"><input value={name} onChange={(e)=>setName(e.target.value)} placeholder="e.g. Main web server" disabled={actionBusy}/></Field>}
          <div className="grid grid-cols-[1.1fr_1.7fr_.55fr] gap-3">
            <Field label="Protocol"><select value={protocol} disabled={actionBusy} onChange={(e)=>{const p=e.target.value as Protocol;setProtocol(p);setPort(PROTOCOL_DEFAULT_PORT[p]);}}><option value="sftp">SFTP (SSH File Transfer)</option><option value="ftp">FTP</option><option value="ftps">FTPS (FTP over TLS)</option></select></Field>
            <Field label="Host / Address"><input value={host} disabled={actionBusy} onChange={(e)=>setHost(e.target.value)} placeholder="e.g. ftp.your-domain.tld or 192.0.2.10"/></Field>
            <Field label="Port"><input type="number" min={1} max={65535} value={port} disabled={actionBusy} onChange={(e)=>setPort(Number(e.target.value))} aria-label="Server port"/></Field>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-3">
            <Field label="Username"><input value={username} disabled={actionBusy} onChange={(e)=>setUsername(e.target.value)} placeholder="Enter username"/></Field>
            {protocol === "sftp" ? (
              <Field label="Authentication">
                <div className="ghost-auth-choice" role="group" aria-label="SFTP authentication method">
                  <button
                    type="button"
                    disabled={actionBusy}
                    className={!useKey ? "active" : ""}
                    aria-pressed={!useKey}
                    onClick={()=>setUseKey(false)}
                  >
                    Password
                  </button>
                  <button
                    type="button"
                    disabled={actionBusy}
                    className={useKey ? "active" : ""}
                    aria-pressed={useKey}
                    onClick={()=>setUseKey(true)}
                  >
                    <KeyRound size={13}/> Private key
                  </button>
                </div>
              </Field>
            ) : (
              <Field label="Password"><div className="relative"><input type={showPassword?'text':'password'} value={password} disabled={actionBusy} onChange={(e)=>setPassword(e.target.value)} placeholder="Enter password" className="pr-10"/><button type="button" disabled={actionBusy} aria-label={showPassword?"Hide password":"Show password"} className="absolute right-2 top-1/2 -translate-y-1/2 text-text-dim disabled:cursor-not-allowed disabled:opacity-50" onClick={()=>setShowPassword(v=>!v)}>{showPassword?<EyeOff size={16}/>:<Eye size={16}/>}</button></div></Field>
            )}
          </div>

          {protocol === "sftp" && !useKey && (
            <div className="mt-3">
              <Field label="Password">
                <div className="relative">
                  <input type={showPassword?'text':'password'} value={password} disabled={actionBusy} onChange={(e)=>setPassword(e.target.value)} placeholder="Enter password" className="pr-10"/>
                  <button type="button" disabled={actionBusy} aria-label={showPassword?"Hide password":"Show password"} className="absolute right-2 top-1/2 -translate-y-1/2 text-text-dim disabled:cursor-not-allowed disabled:opacity-50" onClick={()=>setShowPassword(v=>!v)}>{showPassword?<EyeOff size={16}/>:<Eye size={16}/>}</button>
                </div>
              </Field>
            </div>
          )}

          {protocol === "sftp" && useKey && (
            <div className="mt-3 grid grid-cols-[minmax(0,1fr)_auto] gap-2">
              <input
                className="ghost-ref-input"
                value={keyPath}
                disabled={actionBusy}
                onChange={(e)=>setKeyPath(e.target.value)}
                placeholder="Select private key file…"
                aria-label="Private key file"
              />
              <button
                type="button"
                className="ghost-mini-button"
                disabled={actionBusy}
                aria-busy={keyPicking}
                onClick={()=>void chooseKey()}
                aria-label="Choose private key file"
              >
                <FolderOpen size={14}/> {keyPicking ? "Browsing…" : "Browse"}
              </button>
              <div className="relative col-span-2">
                <input
                  className="ghost-ref-input pr-10"
                  type={showKeyPassphrase?"text":"password"}
                  value={keyPassphrase}
                  disabled={actionBusy}
                  onChange={(e)=>setKeyPassphrase(e.target.value)}
                  placeholder="Private key passphrase (optional)"
                />
                <button
                  type="button"
                  disabled={actionBusy}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-text-dim disabled:cursor-not-allowed disabled:opacity-50"
                  onClick={()=>setShowKeyPassphrase(v=>!v)}
                  aria-label={showKeyPassphrase?"Hide key passphrase":"Show key passphrase"}
                >
                  {showKeyPassphrase?<EyeOff size={16}/>:<Eye size={16}/>}
                </button>
              </div>
            </div>
          )}

          <div className="mt-4">
            <ConnectionModeStatus protocol={protocol}/>
          </div>

          <div className="mt-4 flex items-center gap-3"><Check checked={remember} disabled={actionBusy} onChange={setRemember} label="Save this connection in Sites" icon={<Bookmark size={15}/>}/><div className="flex-1"/><button type="button" className="ghost-mini-button" disabled={!canConnect||actionBusy} onClick={()=>void testConnection()} aria-live="polite" aria-busy={testStatus==="testing"}><Radio size={14}/>{testStatus==="testing"?"Testing…":testStatus==="ok"?"Connection OK":testStatus==="error"?"Test Failed":"Test Connection"}</button></div>

          <div className="mt-4 overflow-hidden rounded-md border border-border bg-[#051929]">
            <button type="button" disabled={actionBusy} onClick={()=>setAdvanced(v=>!v)} className="flex w-full items-center gap-2 px-4 py-3 text-left text-[12px] text-text-muted hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-60"><Settings2 size={16}/><span>Advanced Settings</span><div className="flex-1"/><ChevronDown size={14} className={advanced?'rotate-180':''}/></button>
            {advanced && <div className="grid grid-cols-2 gap-3 border-t border-border p-4"><Field label="Default remote path"><input value={remotePath} disabled={actionBusy} onChange={(e)=>setRemotePath(e.target.value)} placeholder="/var/www"/></Field>{(protocol==="ftp"||protocol==="ftps")&&<Field label="Connection mode"><input value="Passive" readOnly/></Field>}</div>}
          </div>
        </div>

        <div className="ghost-new-connection-actions flex shrink-0 items-center border-t border-border bg-[#051929] px-5 py-4">
          <button type="button" className="ghost-mini-button" disabled={actionBusy} onClick={closeIfIdle}>{cancelLabel}</button><div className="flex-1"/>
          {remember && <button type="button" disabled={!canConnect||actionBusy} className="ghost-mini-button mr-2" onClick={()=>void submit(false)}><Bookmark size={14}/> {busy?'Saving…':'Save'}</button>}
          <button type="button" disabled={!canConnect||actionBusy} aria-busy={busy} className="ghost-primary-button" onClick={()=>void submit(true)}><Link2 size={15}/> {busy?'Connecting…':'Connect'}</button>
        </div>
      </div>
    </div>
  );
}

function ConnectionModeStatus({protocol}:{protocol:Protocol}) {
  const ftp = protocol === "ftp" || protocol === "ftps";
  return <div className="flex items-center gap-2 text-[12px] text-text-muted">
    <Radio size={15}/>
    <span>{ftp ? "Passive mode" : "SSH transport"}</span>
    <span className="rounded border border-success/25 bg-success/10 px-2 py-0.5 text-[10px] text-success">
      {ftp ? "Recommended" : "Secure"}
    </span>
  </div>;
}
function Field({label,children}:{label:string;children:React.ReactNode}){return <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-text-muted">{label}</span><div className="ghost-ref-field">{children}</div></label>}
function Check({checked,onChange,label,icon,disabled=false}:{checked:boolean;onChange:(v:boolean)=>void;label:string;icon?:React.ReactNode;disabled?:boolean}){return <label className={`flex items-center gap-2 text-[12px] ${disabled?'opacity-40':'cursor-pointer'} text-text-muted`}><input type="checkbox" disabled={disabled} checked={checked} onChange={(e)=>onChange(e.target.checked)} className="accent-[#189dff]"/>{icon}<span>{label}</span></label>}
