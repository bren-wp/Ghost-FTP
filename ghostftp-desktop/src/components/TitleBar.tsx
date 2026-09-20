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
import { openOfficialUrl } from "@/lib/external";
import { PRODUCT_VERSION_BADGE } from "@/lib/release";

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
  const quickConnectRef = useRef<HTMLDivElement>(null);
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
      { label: "Help Center", run: () => openOfficialUrl("/support/") },
      { label: "Documentation", run: () => openOfficialUrl("/docs/") },
      { label: "Check for Updates", run: () => openDialog("about") },
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

  useEffect(() => {
    if (!protocolMenu) return;
    const close = (event: MouseEvent) => {
      if (!quickConnectRef.current?.contains(event.target as Node)) setProtocolMenu(false);
    };
    const esc = (event: KeyboardEvent) => {
      if (event.key === "Escape") setProtocolMenu(false);
    };
    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", esc);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", esc);
    };
  }, [protocolMenu]);

  const focusTopMenu = (name: string) => {
    requestAnimationFrame(() => {
      menuWrap.current?.querySelector<HTMLButtonElement>(`[data-menu-trigger="${name}"]`)?.focus();
    });
  };
  const openMenuFromKeyboard = (name: string) => {
    setMenu(name);
    requestAnimationFrame(() => {
      menuWrap.current?.querySelector<HTMLButtonElement>(`[data-menu-anchor="${name}"] [role="menuitem"]:not(:disabled)`)?.focus();
    });
  };
  const moveTopMenu = (name: string, delta: number) => {
    const names = Object.keys(menus);
    const index = names.indexOf(name);
    const next = names[(index + delta + names.length) % names.length];
    if (menu) openMenuFromKeyboard(next);
    else focusTopMenu(next);
  };
  const menuListKeyDown = (event: React.KeyboardEvent<HTMLDivElement>, name: string) => {
    const items = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)'));
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const delta = event.key === "ArrowDown" ? 1 : -1;
      items[(Math.max(current, 0) + delta + items.length) % items.length]?.focus();
    } else if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      (event.key === "Home" ? items[0] : items[items.length - 1])?.focus();
    } else if (event.key === "Escape") {
      event.preventDefault();
      setMenu(null);
      focusTopMenu(name);
    } else if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
      event.preventDefault();
      moveTopMenu(name, event.key === "ArrowRight" ? 1 : -1);
    }
  };

  return <header className="ghost-app-header">
    <div
      className="ghost-title-row"
      onDoubleClick={(event) => {
        if ((event.target as HTMLElement).closest("button,select,input")) return;
        safeWindowAction("maximize");
      }}
    >
      <div className="ghost-title-left" data-tauri-drag-region>
        <GhostWordmark compact />
        <span className="ghost-version-badge">{PRODUCT_VERSION_BADGE}</span>
      </div>
      <div className="ghost-title-menu-wrap" ref={menuWrap}>
        <nav className="ghost-menu" aria-label="Application menu">
          {Object.keys(menus).map((name) => <div className="ghost-menu-anchor" data-menu-anchor={name} key={name}>
            <button
              className={menu === name ? "active" : ""}
              aria-haspopup="menu"
              aria-expanded={menu === name}
              data-menu-trigger={name}
              onKeyDown={(event) => {
                if (event.key === "ArrowDown") { event.preventDefault(); openMenuFromKeyboard(name); }
                else if (event.key === "ArrowRight" || event.key === "ArrowLeft") { event.preventDefault(); moveTopMenu(name, event.key === "ArrowRight" ? 1 : -1); }
              }}
              onPointerEnter={() => { if (menu && menu !== name) setMenu(name); }}
              onClick={() => setMenu((value) => value === name ? null : name)}
            >
              {name}
            </button>
            {menu === name && (
              <div className="ghost-menu-popover" role="menu" onKeyDown={(event) => menuListKeyDown(event, name)}>
                {menus[name].map((item, idx) =>
                  "separator" in item
                    ? <div key={idx} className="ghost-menu-separator"/>
                    : <button key={idx} role="menuitem" disabled={item.disabled} onClick={() => { setMenu(null); item.run(); }}>{item.label}</button>)}
              </div>
            )}
          </div>)}
        </nav>
      </div>
      <div className="ghost-window-title-spacer" data-tauri-drag-region />
      <div className="ghost-window-controls">
        <button aria-label="Minimize" onClick={() => safeWindowAction("minimize")}><Minus size={14}/></button>
        <button aria-label="Maximize or restore" onClick={() => safeWindowAction("maximize")}><Square size={12}/></button>
        <button className="danger" aria-label="Close" onClick={() => safeWindowAction("close")}><X size={15}/></button>
      </div>
    </div>

    <div className="ghost-quick-row">
      <label>Host<input value={host} onChange={(e) => setHost(e.target.value)} placeholder=""/></label>
      <label>Username<input value={username} onChange={(e) => setUsername(e.target.value)} placeholder=""/></label>
      <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••"/></label>
      <label className="ghost-port-field">Port<input type="number" min={1} max={65535} value={port} onChange={(e) => setPort(Number(e.target.value) || PROTOCOL_DEFAULT_PORT[protocol])}/></label>
      <div className="ghost-quick-connect" ref={quickConnectRef}>
        <button className="ghost-quick-connect-main" disabled={quickBusy} onClick={() => void quickConnect()}>{quickBusy ? "Connecting…" : "Quick Connect"}</button>
        <button className="ghost-quick-connect-menu" aria-label="Quick Connect protocol" aria-haspopup="menu" aria-expanded={protocolMenu} onKeyDown={(event) => { if (event.key === "ArrowDown") { event.preventDefault(); setProtocolMenu(true); requestAnimationFrame(() => quickConnectRef.current?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus()); } }} onClick={() => setProtocolMenu((value) => !value)}><ChevronDown size={13}/></button>
        {protocolMenu && <div className="ghost-quick-protocol-menu" role="menu" onKeyDown={(event) => { const items=Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]')); const current=items.indexOf(document.activeElement as HTMLButtonElement); if(event.key==="ArrowDown"||event.key==="ArrowUp"){event.preventDefault(); const delta=event.key==="ArrowDown"?1:-1; items[(Math.max(current,0)+delta+items.length)%items.length]?.focus();} else if(event.key==="Escape"){event.preventDefault();setProtocolMenu(false);quickConnectRef.current?.querySelector<HTMLButtonElement>(".ghost-quick-connect-menu")?.focus();} }}>
          {(["ftp", "ftps", "sftp"] as Protocol[]).map((item) => <button key={item} role="menuitem" className={item === protocol ? "active" : ""} onClick={() => chooseQuickProtocol(item)}>{item.toUpperCase()} <span>{PROTOCOL_DEFAULT_PORT[item]}</span></button>)}
        </div>}
      </div>
      <div className="ghost-title-actions ghost-quick-actions">
        <button className="ghost-mini-button" aria-label="Settings" title="Settings" onClick={() => openDialog("settings")}><Settings size={14}/><span>Settings</span></button>
        <div className="ghost-language-menu"><Languages size={14}/><select aria-label="Language" value={locale} onChange={(e) => setLocale(e.target.value as any)}><option value="en">English (English)</option><option value="hr">Hrvatski (Croatian)</option><option value="de">Deutsch (German)</option><option value="fr">Français (French)</option><option value="es">Español (Spanish)</option><option value="it">Italiano (Italian)</option><option value="pt">Português (Portuguese)</option><option value="nl">Nederlands (Dutch)</option><option value="pl">Polski (Polish)</option><option value="sl">Slovenščina (Slovenian)</option><option value="sr">Srpski (Serbian)</option><option value="bs">Bosanski (Bosnian)</option><option value="mk">Македонски (Macedonian)</option><option value="sq">Shqip (Albanian)</option></select><ChevronDown size={12}/></div>
      </div>
    </div>

    <div className="ghost-toolbar-row">
      <div className="ghost-sites-toolbar-head">
        <button className="ghost-sites-toolbar-title" onClick={() => openDialog("siteManager")}><span className="ghost-sites-ring">◉</span><span>Sites</span></button>
        <span/>
        <button aria-label="New site" title="New site" onClick={() => openNewConnection()}>＋</button>
        <button aria-label="Open Site Manager" title="Open Site Manager" onClick={() => openDialog("siteManager")}><ChevronDown size={14}/></button>
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
