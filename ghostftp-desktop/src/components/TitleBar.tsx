import { useEffect, useMemo, useRef, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  ChevronDown, Download, FolderPlus, Info, Languages, Link2, Minus, Pencil,
  RefreshCw, Settings, Shield, Square, Trash2, Upload, X,
} from "lucide-react";
import { GhostWordmark } from "./GhostBrand";
import { useLayout } from "@/stores/layoutStore";
import { useConnections } from "@/stores/connectionsStore";
import { getLocale, setLocale } from "@/lib/i18n";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";
import { PRODUCT_VERSION_BADGE } from "@/lib/release";
import { toastError } from "@/lib/errors";

type PaneTarget = "local" | "remote" | "active";
type FileAction = "refresh" | "upload" | "download" | "newFolder" | "delete" | "rename" | "properties";
type MenuItem = { label: string; run: () => void; disabled?: boolean } | { separator: true };
type PaneActionState = {
  paneId: "local" | "remote";
  selectedCount: number;
  hasActiveItem: boolean;
  hasSession: boolean;
  canCreateDirectory: boolean;
  focused: boolean;
};

function fileAction(action: FileAction, pane: PaneTarget = "active") {
  const target = pane === "active" ? undefined : pane;
  window.dispatchEvent(new CustomEvent("ghostftp:toolbar-action", { detail: { action, target } }));
}
async function safeWindowAction(action: "minimize" | "maximize" | "close") {
  try {
    const win = getCurrentWindow();
    if (action === "minimize") await win.minimize();
    if (action === "maximize") await win.toggleMaximize();
    if (action === "close") await win.close();
  } catch (error) {
    toastError(error, `Couldn't ${action === "maximize" ? "maximize or restore" : action} Ghost FTP`);
  }
}

export function TitleBar() {
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const disconnect = useConnections((s) => s.disconnect);
  const connectTemporary = useConnections((s) => s.connectTemporary);
  const [protocol, setProtocol] = useState<Protocol>("sftp");
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [port, setPort] = useState(22);
  const [menu, setMenu] = useState<string | null>(null);
  const [protocolMenu, setProtocolMenu] = useState(false);
  const [quickBusy, setQuickBusy] = useState(false);
  const emptyPane = (paneId: "local" | "remote"): PaneActionState => ({
    paneId,
    selectedCount: 0,
    hasActiveItem: false,
    hasSession: paneId === "local",
    canCreateDirectory: paneId === "local",
    focused: paneId === "local",
  });
  const [paneStates, setPaneStates] = useState<Record<"local" | "remote", PaneActionState>>({
    local: emptyPane("local"),
    remote: emptyPane("remote"),
  });
  const [activePane, setActivePane] = useState<"local" | "remote">("local");
  const paneState = paneStates[activePane];
  const menuWrap = useRef<HTMLDivElement>(null);
  const quickConnectRef = useRef<HTMLDivElement>(null);
  const locale = getLocale();

  useEffect(() => {
    const handler = (event: Event) => {
      const custom = event as CustomEvent<PaneActionState>;
      if (!custom.detail) return;
      setPaneStates((current) => ({ ...current, [custom.detail.paneId]: custom.detail }));
      if (custom.detail.focused) setActivePane(custom.detail.paneId);
    };
    window.addEventListener("ghostftp:pane-action-state", handler as EventListener);
    return () => window.removeEventListener("ghostftp:pane-action-state", handler as EventListener);
  }, []);

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
    } catch {
      // connectionsStore already surfaces the structured backend error.
      // Swallow it here so a failed toolbar Quick Connect does not become an
      // unhandled promise rejection in WebView2.
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
      { label: "Exit", run: () => void safeWindowAction("close") },
    ],
    Edit: [
      { label: "Rename", run: () => fileAction("rename"), disabled: !paneState.hasActiveItem },
      { label: "Delete", run: () => fileAction("delete"), disabled: paneState.selectedCount === 0 },
      { label: "Properties", run: () => fileAction("properties"), disabled: !paneState.hasActiveItem },
      { separator: true },
      { label: "Preferences…", run: () => openDialog("settings") },
    ],
    View: [
      { label: "Refresh", run: () => fileAction("refresh"), disabled: !paneState.hasSession },
      { label: "Transfer Center", run: () => openDialog("transferCenter") },
      { label: "Site Manager", run: () => openDialog("siteManager") },
    ],
    Transfer: [
      { label: "Upload", run: () => fileAction("upload", "local"), disabled: !activeSessionId || paneStates.local.selectedCount === 0 },
      { label: "Download", run: () => fileAction("download", "remote"), disabled: !activeSessionId || paneStates.remote.selectedCount === 0 },
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
      { label: "Help Center", run: () => openDialog("help") },
      { label: "Documentation", run: () => openDialog("help") },
      { label: "Check for Updates", run: () => openDialog("updates") },
      { separator: true },
      { label: "About Ghost FTP", run: () => openDialog("about") },
    ],
  }), [activeSessionId, disconnect, openDialog, openNewConnection, paneState, paneStates]);

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
        void safeWindowAction("maximize");
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
        <button aria-label="Minimize" onClick={() => void safeWindowAction("minimize")}><Minus size={14}/></button>
        <button aria-label="Maximize or restore" onClick={() => void safeWindowAction("maximize")}><Square size={12}/></button>
        <button className="danger" aria-label="Close" onClick={() => void safeWindowAction("close")}><X size={15}/></button>
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
        <button className="ghost-sites-toolbar-title" onClick={() => openDialog("siteManager")}><Shield size={15}/><span>Sites</span></button>
        <span/>
        <button aria-label="New site" title="New site" onClick={() => openNewConnection()}>＋</button>
        <button aria-label="Open Site Manager" title="Open Site Manager" onClick={() => openDialog("siteManager")}><ChevronDown size={14}/></button>
      </div>
      <Tool icon={<Link2 size={17}/>} label="Connect" onClick={() => openNewConnection()}/>
      <Tool icon={<X size={17}/>} label="Disconnect" disabled={!activeSessionId} onClick={() => void disconnect()}/>
      <Tool icon={<RefreshCw size={17}/>} label="Refresh" onClick={() => fileAction("refresh")}/>
      <Tool icon={<Upload size={17}/>} label="Upload" disabled={!activeSessionId || paneStates.local.selectedCount === 0} onClick={() => fileAction("upload", "local")}/>
      <Tool icon={<Download size={17}/>} label="Download" disabled={!activeSessionId || paneStates.remote.selectedCount === 0} onClick={() => fileAction("download", "remote")}/>
      <Tool icon={<FolderPlus size={17}/>} label="New Folder" disabled={!paneState.canCreateDirectory} onClick={() => fileAction("newFolder")}/>
      <Tool icon={<Trash2 size={17}/>} label="Delete" disabled={paneState.selectedCount === 0} onClick={() => fileAction("delete")}/>
      <Tool icon={<Pencil size={17}/>} label="Rename" disabled={!paneState.hasActiveItem} onClick={() => fileAction("rename")}/>
      <Tool icon={<Info size={17}/>} label="Properties" disabled={!paneState.hasActiveItem} onClick={() => fileAction("properties")}/>
      <div className="ghost-toolbar-spacer"/>
    </div>
  </header>;
}

function Tool({ icon, label, onClick, disabled = false }: { icon: React.ReactNode; label: string; onClick?: () => void; disabled?: boolean }) {
  return <button className="ghost-tool-button" aria-label={label} title={label} onClick={onClick} disabled={disabled || !onClick}>{icon}<span>{label}</span></button>;
}
