import {
  ArrowUpDown,
  FolderOpen,
  FolderSync,
  HelpCircle,
  Plus,
  Server,
  Settings,
} from "lucide-react";
import type { ReactNode } from "react";
import { type AppDialog, useLayout } from "@/stores/layoutStore";

const WORKSPACE_DIALOGS = new Set<AppDialog>([
  "settings", "siteManager", "transferCenter", "sync", "help", "updates", "about",
]);

function workspace(dialog: AppDialog | null, returnDialog: AppDialog | null): AppDialog | null {
  if (dialog && WORKSPACE_DIALOGS.has(dialog)) return dialog;
  if (returnDialog && WORKSPACE_DIALOGS.has(returnDialog)) return returnDialog;
  return null;
}

export function ReferenceSiteSidebar() {
  const dialog = useLayout((s) => s.dialog);
  const returnDialog = useLayout((s) => s.returnDialog);
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const showFiles = useLayout((s) => s.showFiles);
  const current = workspace(dialog, returnDialog);

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

      <div className="ghost-sites-spacer"/>
      <SidebarAction
        icon={<HelpCircle size={16}/>} label="Help & About"
        active={current === "about" || current === "help" || current === "updates"}
        onClick={() => openDialog("about")}
      />
    </aside>
  );
}

function SidebarAction({ icon, label, onClick, active = false }: {
  icon: ReactNode; label: string; onClick: () => void; active?: boolean;
}) {
  return (
    <button
      type="button"
      className={`ghost-file-nav-row ${active ? "active" : ""}`}
      onClick={onClick}
      aria-label={label}
      title={label}
      aria-current={active ? "page" : undefined}
    >
      {icon}<span>{label}</span>
    </button>
  );
}
