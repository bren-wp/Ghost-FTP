import { useEffect, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  Download,
  FolderPlus,
  Info,
  Minus,
  Pencil,
  RefreshCw,
  Server,
  Square,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { GhostWordmark } from "./GhostBrand";
import { type AppDialog, useLayout } from "@/stores/layoutStore";
import { useConnections } from "@/stores/connectionsStore";
import { PRODUCT_VERSION_BADGE } from "@/lib/release";
import { toastError } from "@/lib/errors";

type PaneTarget = "local" | "remote" | "active";
type FileAction = "refresh" | "upload" | "download" | "newFolder" | "delete" | "rename" | "properties";
type PaneActionState = {
  paneId: "local" | "remote";
  selectedCount: number;
  hasActiveItem: boolean;
  hasSession: boolean;
  canCreateDirectory: boolean;
  focused: boolean;
};

const WORKSPACE_DIALOGS = new Set<AppDialog>([
  "settings", "siteManager", "transferCenter", "sync", "help", "updates",
  "cloudStorage", "schedules", "activityLogs", "about",
]);

function currentWorkspace(dialog: AppDialog | null, returnDialog: AppDialog | null) {
  const active = dialog && WORKSPACE_DIALOGS.has(dialog) ? dialog : returnDialog;
  if (active === "siteManager" || active === "cloudStorage") return active === "cloudStorage" ? "Cloud Storage" : "Sites";
  if (active === "transferCenter" || active === "schedules" || active === "activityLogs") return "Transfers";
  if (active === "sync") return "Sync & Backup";
  if (active === "settings") return "Settings";
  if (active === "about" || active === "help" || active === "updates") return "Help & About";
  return "Files";
}

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
  const dialog = useLayout((s) => s.dialog);
  const returnDialog = useLayout((s) => s.returnDialog);
  const openDialog = useLayout((s) => s.openDialog);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const disconnect = useConnections((s) => s.disconnect);
  const profile = profiles.find((item) => item.id === activeProfileId) ?? null;
  const workspace = currentWorkspace(dialog, returnDialog);

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

  return (
    <header className="ghost-app-header ghost-simple-header">
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
        <div className="ghost-current-workspace" data-tauri-drag-region>{workspace}</div>
        <div className="ghost-window-title-spacer" data-tauri-drag-region />
        <div className="ghost-window-controls">
          <button aria-label="Minimize" onClick={() => void safeWindowAction("minimize")}><Minus size={14}/></button>
          <button aria-label="Maximize or restore" onClick={() => void safeWindowAction("maximize")}><Square size={12}/></button>
          <button className="danger" aria-label="Close" onClick={() => void safeWindowAction("close")}><X size={15}/></button>
        </div>
      </div>

      {workspace === "Files" && (
        <div className="ghost-toolbar-row ghost-simple-toolbar">
          <button
            className="ghost-active-site-chip"
            onClick={() => openDialog("siteManager")}
            title="Open Sites"
          >
            <Server size={15}/>
            <span>{profile ? profile.name : "Choose a site"}</span>
            {activeSessionId && <i className="online" aria-label="Connected"/>}
          </button>
          {activeSessionId && (
            <Tool icon={<X size={16}/>} label="Disconnect" onClick={() => void disconnect()}/>
          )}
          <Tool icon={<RefreshCw size={17}/>} label="Refresh" onClick={() => fileAction("refresh")}/>
          <Tool icon={<Upload size={17}/>} label="Upload" disabled={!activeSessionId || paneStates.local.selectedCount === 0} onClick={() => fileAction("upload", "local")}/>
          <Tool icon={<Download size={17}/>} label="Download" disabled={!activeSessionId || paneStates.remote.selectedCount === 0} onClick={() => fileAction("download", "remote")}/>
          <Tool icon={<FolderPlus size={17}/>} label="New Folder" disabled={!paneState.canCreateDirectory} onClick={() => fileAction("newFolder")}/>
          <Tool icon={<Pencil size={17}/>} label="Rename" disabled={!paneState.hasActiveItem} onClick={() => fileAction("rename")}/>
          <Tool icon={<Trash2 size={17}/>} label="Delete" disabled={paneState.selectedCount === 0} onClick={() => fileAction("delete")}/>
          <Tool icon={<Info size={17}/>} label="Properties" disabled={!paneState.hasActiveItem} onClick={() => fileAction("properties")}/>
          <div className="ghost-toolbar-spacer"/>
        </div>
      )}
    </header>
  );
}

function Tool({ icon, label, onClick, disabled = false }: { icon: React.ReactNode; label: string; onClick?: () => void; disabled?: boolean }) {
  return <button className="ghost-tool-button" aria-label={label} title={label} onClick={onClick} disabled={disabled || !onClick}>{icon}<span>{label}</span></button>;
}
