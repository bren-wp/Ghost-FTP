import { useEffect, useMemo, useRef, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  ChevronDown, Download, FolderPlus, Info, Languages, Link2, Minus, Pencil,
  RefreshCw, Settings, Square, Trash2, Upload, X,
} from "lucide-react";
import { GhostWordmark } from "./GhostBrand";
import { useLayout } from "@/stores/layoutStore";
import { useConnections } from "@/stores/connectionsStore";
import { getLocale, setLocale } from "@/lib/i18n";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";

const SITE = "https://ghostftp.com";
type PaneTarget = "local" | "remote" | "active";
type FileAction = "refresh" | "upload" | "download" | "newFolder" | "delete" | "rename" | "properties";
type MenuItem = { label: string; run: () => void; disabled?: boolean } | { separator: true };

function fileAction(action: FileAction, pane: PaneTarget = "active") {
  const target = pane === "active" ? undefined : pane;
  window.dispatchEvent(new CustomEvent("ghostftp:toolbar-action", { detail: { action, target } }));
}
function safeWindowAction(action: "minimize" | "maximize" | "close") {
  try {
    const win = getCurrentWindow();
    if (action === "minimize") void win.minimize();
    if (action === "maximize") void win.toggleMaximize();
    if (action === "close") void win.close();
  } catch {}
}

export function TitleBar() {
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const disconnect = useConnections((s) => s.disconnect);
  const connectTemporary = useConnections((s) => s.connectTemporary);
  const [protocol, setProtocol] = useState<Protocol>("ftp");
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [port, setPort] = useState(21);
  const [menu, setMenu] = useState<string | null>(null);
  const [protocolMenu, setProtocolMenu] = useState(false);
  const [quickBusy, setQuickBusy] = useState(false);
  const menuWrap = useRef<HTMLDivElement>(null);
  const locale = getLocale();

  const quickConnect = async () => {
    if (quickBusy) return;
    const normalizedPort =
      Number.isInteger(port) && port >= 1 && port <= 65535
        ? port
        : PROTOCOL_DEFAULT_PORT[protocol];
    if (!host.trim() || !username.trim()) {
      openNewConnection({
        protocol,
        host,
        username,
        port: normalizedPort,
        auth: { kind: "password", password },
        name: host || "New server",
      });
      return;
    }
    const profile: ConnectionProfile = {
      id: crypto.randomUUID(),
      name: host.trim(),
      protocol,
      host: host.trim(),
      port: normalizedPort,
      username: username.trim(),
      auth: { kind: "password", password },
      defaultRemotePath: ".",
      autoConnect: false,
    };
    setQuickBusy(true);
    try {
      await connectTemporary(profile);
      setPassword("");
    } finally {
      setQuickBusy(false);
    }
  };

  const chooseQuickProtocol = (next: Protocol) => {
    setProtocol(next);
    setPort(PROTOCOL_DEFAULT_PORT[next]);
    setProtocolMenu(false);
  };

  const menus = useMemo<Record<string, MenuItem[]>>(() => ({
    File: [
      { label: "New Connection…", run: () => openNewConnection() },
      { label: "Site Manager…", run: () => openDialog("siteManager") },
      { label: "Import Sites…", run: () => openDialog("import") },
      { separator: true },
      { label: "Exit", run: () => safeWindowAction("close") },
    ],
    Edit: [
      { label: "Rename", run: () => fileAction("rename") },
      { label: "Delete", run: () => fileAction("delete") },
      { label: "Properties", run: () => fileAction("properties") },
      { separator: true },
      { label: "Preferences…", run: () => openDialog("settings") },
    ],
    View: [
      { label: "Refresh", run: () => fileAction("refresh") },
      { label: "Transfer Center", run: () => openDialog("transferCenter") },
      { label: "Site Manager", run: () => openDialog("siteManager") },
    ],
    Transfer: [
      { label: "Upload", run: () => fileAction("upload", "local") },
      { label: "Download", run: () => fileAction("download", "remote"), disabled: !activeSessionId },
      { separator: true },
      { label: "Transfer Center", run: () => openDialog("transferCenter") },
    ],
    Server: [
      { label: "Connect…", run: () => openNewConnection() },
      { label: "Disconnect", run: () => void disconnect(), disabled: !activeSessionId },
      { label: "Refresh", run: () => fileAction("refresh", "remote"), disabled: !activeSessionId },
    ],
    Bookmarks: [{ label: "Site Manager…", run: () => openDialog("siteManager") }],
    Tools: [
      { label: "Transfer Center", run: () => openDialog("transferCenter") },
      { label: "Preferences…", run: () => openDialog("settings") },
    ],
    Help: [
      { label: "Help Center", run: () => window.open(`${SITE}/support/`, "_blank", "noopener,noreferrer") },
      { label: "Documentation", run: () => window.open(`${SITE}/docs/`, "_blank", "noopener,noreferrer") },
      { label: "Check for Updates", run: () => window.open(`${SITE}/download/`, "_blank", "noopener,noreferrer") },
      { separator: true },
      { label: "About Ghost FTP", run: () => openDialog("about") },
    ],
  }), [activeSessionId, disconnect, openDialog, openNewConnection]);

  useEffect(() => {
    if (!menu) return;
    const close = (event: MouseEvent) => { if (!menuWrap.current?.contains(event.target as Node)) setMenu(null); };
    const esc = (event: KeyboardEvent) => { if (event.key === "Escape") setMenu(null); };
    document.addEventListener("mousedown", close); document.addEventListener("keydown", esc);
    return () => { document.removeEventListener("mousedown", close); document.removeEventListener("keydown", esc); };
  }, [menu]);

  return <header className="ghost-app-header">
    <div className="ghost-title-row" ref={menuWrap} data-tauri-drag-region onDoubleClick={() => safeWindowAction("maximize")}>
      <div className="ghost-title-left" data-tauri-drag-region>
        <GhostWordmark compact />
        <span className="ghost-rc-badge">RC11</span>
      </div>
      <nav className="ghost-menu ghost-title-menu" aria-label="Application menu">
        {Object.keys(menus).map((name) => <div className="ghost-menu-anchor" key={name}>
          <button
            className={menu === name ? "active" : ""}
            aria-haspopup="menu"
            aria-expanded={menu === name}
            onClick={() => setMenu((v) => v === name ? null : name)}
          >
            {name}
          </button>
          {menu === name && <div className="ghost-menu-popover" role="menu">{menus[name].map((item, idx) =>
            "separator" in item ? <div key={idx} className="ghost-menu-separator"/> : <button key={idx} role="menuitem" disabled={item.disabled} onClick={() => { setMenu(null); item.run(); }}>{item.label}</button>)}</div>}
        </div>)}
      </nav>
      <div className="ghost-window-title-spacer" data-tauri-drag-region />
      <div className="ghost-window-controls">
        <button aria-label="Minimize" onClick={() => safeWindowAction("minimize")}><Minus size={14}/></button>
        <button aria-label="Maximize" onClick={() => safeWindowAction("maximize")}><Square size={12}/></button>
        <button className="danger" aria-label="Close" onClick={() => safeWindowAction("close")}><X size={15}/></button>
      </div>
    </div>

    <div className="ghost-quick-row">
      <label>Host<input value={host} onChange={(e) => setHost(e.target.value)} placeholder=""/></label>
      <label>Username<input value={username} onChange={(e) => setUsername(e.target.value)} placeholder=""/></label>
      <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••"/></label>
      <label className="ghost-port-field">Port<input type="number" min={1} max={65535} value={port} onChange={(e) => setPort(Number(e.target.value) || PROTOCOL_DEFAULT_PORT[protocol])}/></label>
      <div className="ghost-quick-connect">
        <button className="ghost-quick-connect-main" disabled={quickBusy} onClick={() => void quickConnect()}>{quickBusy ? "Connecting…" : "Quick Connect"}</button>
        <button className="ghost-quick-connect-menu" aria-label="Quick Connect protocol" aria-expanded={protocolMenu} onClick={() => setProtocolMenu((v) => !v)}><ChevronDown size={13}/></button>
        {protocolMenu && <div className="ghost-quick-protocol-menu" role="menu">
          {(["ftp", "ftps", "sftp"] as Protocol[]).map((item) => <button key={item} role="menuitem" className={item === protocol ? "active" : ""} onClick={() => chooseQuickProtocol(item)}>{item.toUpperCase()} <span>{PROTOCOL_DEFAULT_PORT[item]}</span></button>)}
        </div>}
      </div>
      <div className="ghost-quick-actions">
        <button className="ghost-mini-button" aria-label="Settings" title="Settings" onClick={() => openDialog("settings")}><Settings size={14}/><span>Settings</span></button>
        <div className="ghost-language-menu"><Languages size={14}/><select aria-label="Language" value={locale} onChange={(e) => setLocale(e.target.value as any)}><option value="en">English (English)</option><option value="hr">Hrvatski (Croatian)</option><option value="de">Deutsch (German)</option><option value="fr">Français (French)</option><option value="es">Español (Spanish)</option><option value="it">Italiano (Italian)</option><option value="pt">Português (Portuguese)</option><option value="nl">Nederlands (Dutch)</option><option value="pl">Polski (Polish)</option><option value="sl">Slovenščina (Slovenian)</option><option value="sr">Srpski (Serbian)</option><option value="bs">Bosanski (Bosnian)</option><option value="mk">Македонски (Macedonian)</option><option value="sq">Shqip (Albanian)</option></select><ChevronDown size={12}/></div>
      </div>
    </div>

    <div className="ghost-toolbar-row">
      <div className="ghost-sites-toolbar-head">
        <button className="ghost-sites-toolbar-title" onClick={() => openDialog("siteManager")}><span className="ghost-sites-ring">◉</span><span>Sites</span></button>
        <span/>
        <button aria-label="New site" title="New site" onClick={() => openNewConnection()}>＋</button>
        <button aria-label="Site Manager" title="Site Manager" onClick={() => openDialog("siteManager")}>×</button>
      </div>
      <Tool icon={<Link2 size={17}/>} label="Connect" onClick={() => openNewConnection()}/>
      <Tool icon={<X size={17}/>} label="Disconnect" disabled={!activeSessionId} onClick={() => void disconnect()}/>
      <Tool icon={<RefreshCw size={17}/>} label="Refresh" onClick={() => fileAction("refresh")}/>
      <Tool icon={<Upload size={17}/>} label="Upload" onClick={() => fileAction("upload", "local")}/>
      <Tool icon={<Download size={17}/>} label="Download" disabled={!activeSessionId} onClick={() => fileAction("download", "remote")}/>
      <Tool icon={<FolderPlus size={17}/>} label="New Folder" onClick={() => fileAction("newFolder")}/>
      <Tool icon={<Trash2 size={17}/>} label="Delete" onClick={() => fileAction("delete")}/>
      <Tool icon={<Pencil size={17}/>} label="Rename" onClick={() => fileAction("rename")}/>
      <Tool icon={<Info size={17}/>} label="Properties" onClick={() => fileAction("properties")}/>
      <div className="ghost-toolbar-spacer"/>
    </div>
  </header>;
}

function Tool({ icon, label, onClick, disabled = false }: { icon: React.ReactNode; label: string; onClick?: () => void; disabled?: boolean }) {
  return <button className="ghost-tool-button" aria-label={label} title={label} onClick={onClick} disabled={disabled}>{icon}<span>{label}</span></button>;
}
