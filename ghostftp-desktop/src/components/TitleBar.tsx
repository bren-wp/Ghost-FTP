import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  Download,
  FolderOpen,
  FolderPlus,
  Info,
  Minus,
  MoreHorizontal,
  Pencil,
  RefreshCw,
  Server,
  Square,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { GhostWordmark } from "./GhostBrand";
import { useLayout } from "@/stores/layoutStore";
import { workspaceLabel } from "@/lib/workspaces";
import { useConnections } from "@/stores/connectionsStore";
import { useSettings } from "@/stores/settingsStore";
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
  const browseLocal = useLayout((s) => s.browseLocal);
  const setBrowseLocal = useLayout((s) => s.setBrowseLocal);
  const browserLayout = useSettings((s) => s.browserLayout);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const disconnect = useConnections((s) => s.disconnect);
  const profile = profiles.find((item) => item.id === activeProfileId) ?? null;
  const workspace = workspaceLabel(dialog, returnDialog);

  const disconnectActive = () => {
    void disconnect().catch((error) => {
      toastError(error, "Couldn't disconnect from the active site");
    });
  };

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
  const [moreOpen, setMoreOpen] = useState(false);
  const [morePosition, setMorePosition] = useState({ top: 0, left: 0 });
  const moreRef = useRef<HTMLDivElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const moreMenuRef = useRef<HTMLDivElement>(null);

  const positionMoreMenu = () => {
    const anchor = moreRef.current?.getBoundingClientRect();
    if (!anchor) return;
    const menuWidth = 174;
    const gutter = 8;
    setMorePosition({
      top: anchor.bottom + 7,
      left: Math.min(
        Math.max(gutter, anchor.right - menuWidth),
        Math.max(gutter, window.innerWidth - menuWidth - gutter),
      ),
    });
  };
  const singlePane = browserLayout === "single";
  const effectivePane: "local" | "remote" = singlePane
    ? browseLocal
      ? "local"
      : "remote"
    : activePane;
  const paneState = paneStates[effectivePane];

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

  useEffect(() => {
    if (!moreOpen) return;
    const focusRaf = requestAnimationFrame(() => {
      moreMenuRef.current
        ?.querySelector<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)')
        ?.focus();
    });
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!moreRef.current?.contains(target) && !moreMenuRef.current?.contains(target)) {
        setMoreOpen(false);
      }
    };
    const onLayout = () => positionMoreMenu();
    positionMoreMenu();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMoreOpen(false);
        moreButtonRef.current?.focus();
      }
    };
    window.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("resize", onLayout);
    window.addEventListener("scroll", onLayout, true);
    return () => {
      cancelAnimationFrame(focusRaf);
      window.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("resize", onLayout);
      window.removeEventListener("scroll", onLayout, true);
    };
  }, [moreOpen]);

  const onMoreMenuKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    const items = Array.from(
      event.currentTarget.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)')
    );
    if (items.length === 0) return;
    event.preventDefault();
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    let next = current;
    if (event.key === "Home") next = 0;
    else if (event.key === "End") next = items.length - 1;
    else if (event.key === "ArrowDown") next = current < 0 ? 0 : (current + 1) % items.length;
    else next = current <= 0 ? items.length - 1 : current - 1;
    items[next]?.focus();
  };

  useEffect(() => {
    if (workspace !== "Files") setMoreOpen(false);
  }, [workspace]);

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
          {singlePane && (
            <Tool
              icon={<FolderOpen size={17}/>}
              label="Local"
              active={browseLocal}
              onClick={() => setBrowseLocal(true)}
            />
          )}
          {profile && (
            <button
              className={`ghost-active-site-chip ${singlePane && !browseLocal ? "active" : ""}`}
              onClick={() => {
                if (singlePane && activeSessionId && browseLocal) {
                  setBrowseLocal(false);
                  return;
                }
                openDialog("siteManager");
              }}
              aria-label={
                singlePane && activeSessionId && browseLocal
                  ? `Show ${profile.name} server files`
                  : `Open ${profile.name} in Sites`
              }
              title={singlePane && activeSessionId && browseLocal ? "Show server files" : "Open this site in Sites"}
            >
              <Server size={15}/>
              <span>{profile.name}</span>
              {activeSessionId && <i className="online" aria-label="Connected"/>}
            </button>
          )}
          {activeSessionId && (
            <Tool icon={<X size={16}/>} label="Disconnect" onClick={disconnectActive}/>
          )}
          <Tool icon={<RefreshCw size={17}/>} label="Refresh" onClick={() => fileAction("refresh", effectivePane)}/>
          <Tool
            icon={<Upload size={17}/>}
            label="Upload"
            disabled={!activeSessionId || (singlePane ? browseLocal && paneStates.local.selectedCount === 0 : paneStates.local.selectedCount === 0)}
            onClick={() => {
              if (singlePane && !browseLocal) {
                window.dispatchEvent(new CustomEvent("ghostftp:pick-upload"));
                return;
              }
              fileAction("upload", "local");
            }}
          />
          <Tool
            icon={<Download size={17}/>}
            label="Download"
            disabled={!activeSessionId || (singlePane && browseLocal) || paneStates.remote.selectedCount === 0}
            onClick={() => fileAction("download", "remote")}
          />
          <Tool icon={<FolderPlus size={17}/>} label="New Folder" disabled={!paneState.canCreateDirectory} onClick={() => fileAction("newFolder", effectivePane)}/>
          <div ref={moreRef} className="ghost-toolbar-more">
            <button
              ref={moreButtonRef}
              type="button"
              className={`ghost-tool-button ${moreOpen ? "active" : ""}`}
              aria-label="More file actions"
              title="More file actions"
              aria-haspopup="menu"
              aria-expanded={moreOpen}
              onClick={() => {
                if (!moreOpen) positionMoreMenu();
                setMoreOpen((open) => !open);
              }}
            >
              <MoreHorizontal size={17}/>
              <span>More</span>
            </button>
            {moreOpen && createPortal(
              <div
                ref={moreMenuRef}
                className="ghost-toolbar-more-menu"
                role="menu"
                aria-label="More file actions"
                onKeyDown={onMoreMenuKeyDown}
                style={{ position: "fixed", top: morePosition.top, left: morePosition.left, right: "auto", zIndex: 200 }}
              >
                <MoreAction
                  icon={<Pencil size={15}/>}
                  label="Rename"
                  disabled={!paneState.hasActiveItem}
                  onClick={() => {
                    setMoreOpen(false);
                    fileAction("rename", effectivePane);
                  }}
                />
                <MoreAction
                  icon={<Info size={15}/>}
                  label="Properties"
                  disabled={!paneState.hasActiveItem}
                  onClick={() => {
                    setMoreOpen(false);
                    fileAction("properties", effectivePane);
                  }}
                />
                <div className="ghost-toolbar-more-separator"/>
                <MoreAction
                  icon={<Trash2 size={15}/>}
                  label="Delete"
                  destructive
                  disabled={paneState.selectedCount === 0}
                  onClick={() => {
                    setMoreOpen(false);
                    fileAction("delete", effectivePane);
                  }}
                />
              </div>,
              document.body,
            )}
          </div>
          <div className="ghost-toolbar-spacer"/>
        </div>
      )}
    </header>
  );
}

function Tool({ icon, label, onClick, disabled = false, active = false }: { icon: React.ReactNode; label: string; onClick?: () => void; disabled?: boolean; active?: boolean }) {
  return <button className={`ghost-tool-button ${active ? "active" : ""}`} aria-label={label} title={label} aria-pressed={active || undefined} onClick={onClick} disabled={disabled || !onClick}>{icon}<span>{label}</span></button>;
}

function MoreAction({ icon, label, onClick, disabled = false, destructive = false }: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  destructive?: boolean;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      className={`ghost-toolbar-more-action ${destructive ? "danger" : ""}`}
      disabled={disabled}
      onClick={onClick}
    >
      {icon}<span>{label}</span>
    </button>
  );
}
