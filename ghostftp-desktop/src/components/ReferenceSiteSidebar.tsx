import {
  ArrowUpDown,
  FolderOpen,
  FolderSync,
  HelpCircle,
  Keyboard,
  Plus,
  ScanSearch,
  Server,
  Settings,
  Terminal,
} from "lucide-react";
import type { ReactNode } from "react";
import { useLayout } from "@/stores/layoutStore";
import { resolveWorkspaceDialog } from "@/lib/workspaces";
import { useConnections } from "@/stores/connectionsStore";
import { useDedupe } from "@/stores/dedupeStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";

export function ReferenceSiteSidebar() {
  const dialog = useLayout((s) => s.dialog);
  const returnDialog = useLayout((s) => s.returnDialog);
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const showFiles = useLayout((s) => s.showFiles);
  const current = resolveWorkspaceDialog(dialog, returnDialog);
  const terminalOpen = useLayout((s) => s.terminalOpen);
  const setTerminalOpen = useLayout((s) => s.setTerminalOpen);
  const toggleTerminal = useLayout((s) => s.toggleTerminal);
  const paletteOpen = useLayout((s) => s.paletteOpen);
  const setPaletteOpen = useLayout((s) => s.setPaletteOpen);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const duplicatesOpen = useDedupe((s) => s.open);
  const openDuplicates = useDedupe((s) => s.openFor);
  const supportsTerminal = !!activeSessionId && profiles.some(
    (profile) => profile.id === activeProfileId && profile.protocol === "sftp",
  );

  const openTerminal = () => {
    if (!supportsTerminal) return;
    if (current !== null) {
      showFiles();
      setTerminalOpen(true);
    } else {
      toggleTerminal();
    }
  };

  return (
    <aside className="ghost-sites-panel ghost-primary-sidebar" aria-label="Ghost FTP navigation">
      <button type="button" className="ghost-sidebar-new" aria-label="New connection" title="New connection" onClick={() => openNewConnection()}>
        <Plus size={16}/><span>New connection</span>
      </button>

      <nav className="ghost-file-nav ghost-primary-nav" aria-label="Primary navigation">
        <SidebarAction
          icon={<FolderOpen size={16}/>} label="Files" active={current === null}
          onClick={showFiles}
        />
        <SidebarAction
          icon={<Server size={16}/>} label="Sites"
          active={current === "siteManager"}
          onClick={() => openDialog("siteManager")}
        />
        <SidebarAction
          icon={<ArrowUpDown size={16}/>} label="Transfers"
          active={current === "transferCenter"}
          onClick={() => openDialog("transferCenter")}
        />
        <SidebarAction
          icon={<FolderSync size={16}/>} label="Sync & Backup"
          active={current === "sync"}
          onClick={() => openDialog("sync")}
        />
        <SidebarAction
          icon={<Settings size={16}/>} label="Settings"
          active={current === "settings"}
          onClick={() => openDialog("settings")}
        />
      </nav>

      <nav className="ghost-file-nav ghost-primary-nav ghost-tools-nav" aria-label="Quick tools">
        <span className="ghost-sidebar-section-label">Quick tools</span>
        <SidebarAction
          icon={<Terminal size={16}/>} label="Terminal"
          active={terminalOpen && current === null}
          disabled={!supportsTerminal}
          onClick={openTerminal}
        />
        <SidebarAction
          icon={<Keyboard size={16}/>} label="Commands"
          active={paletteOpen}
          onClick={() => setPaletteOpen(true)}
        />
        <SidebarAction
          icon={<ScanSearch size={16}/>} label="Duplicates"
          active={duplicatesOpen}
          onClick={() => openDuplicates(LOCAL_SESSION, "")}
        />
      </nav>

      <div className="ghost-sites-spacer"/>
      <SidebarAction
        icon={<HelpCircle size={16}/>} label="Help & About"
        active={current === "about" || current === "help" || current === "updates"}
        onClick={() => openDialog("about")}
      />
    </aside>
  );
}

function SidebarAction({ icon, label, onClick, active = false, disabled = false }: {
  icon: ReactNode; label: string; onClick: () => void; active?: boolean; disabled?: boolean;
}) {
  return (
    <button
      type="button"
      className={`ghost-file-nav-row ${active ? "active" : ""}`}
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      aria-current={active ? "page" : undefined}
    >
      {icon}<span>{label}</span>
    </button>
  );
}
