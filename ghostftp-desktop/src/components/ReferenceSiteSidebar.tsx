import {
  ChevronDown,
  ChevronRight,
  Clock3,
  Cloud,
  FolderOpen,
  FolderSync,
  Plus,
  ScrollText,
  Server,
  Settings,
  ArrowUpDown,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";

export function ReferenceSiteSidebar() {
  const profiles = useConnections((s) => s.profiles);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const sessions = useConnections((s) => s.sessions);
  const connect = useConnections((s) => s.connect);
  const setActiveSession = useConnections((s) => s.setActiveSession);
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const [expanded, setExpanded] = useState(true);

  const visibleProfiles = useMemo(() => profiles.slice(0, 9), [profiles]);

  const activate = async (profileId: string) => {
    const live = sessions.find((s) => s.profileId === profileId);
    if (live) {
      setActiveSession(live.sessionId);
      return;
    }
    await connect(profileId);
  };

  const openSync = () => {
    window.dispatchEvent(new CustomEvent("ghostftp:open-sync"));
  };

  return (
    <aside className="ghost-sites-panel" aria-label="Sites and File Manager navigation">
      <button
        type="button"
        className="ghost-sites-group"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        {expanded ? <ChevronDown size={13}/> : <ChevronRight size={13}/>}
        <Server size={14}/>
        <span>My Sites</span>
      </button>

      {expanded && (
        <div className="ghost-sites-list">
          {visibleProfiles.length === 0 ? (
            <button
              type="button"
              className="ghost-site-row ghost-site-empty"
              onClick={() => openNewConnection()}
            >
              <Plus size={14}/><span>Add your first server</span>
            </button>
          ) : visibleProfiles.map((p) => {
            const connected = sessions.some((s) => s.profileId === p.id);
            const active = p.id === activeProfileId;
            return (
              <button
                type="button"
                key={p.id}
                className={`ghost-site-row ${active ? "active" : ""}`}
                onClick={() => void activate(p.id)}
                title={`${p.name} · ${p.host}`}
              >
                <Server size={14}/>
                <span className="ghost-site-name">{p.name}</span>
                <i className={connected ? "online" : ""}/>
              </button>
            );
          })}
        </div>
      )}

      <nav className="ghost-file-nav" aria-label="File Manager sections">
        <SidebarAction
          icon={<ArrowUpDown size={15}/>}
          label="Transfer Center"
          onClick={() => openDialog("transferCenter")}
        />
        <SidebarAction
          icon={<FolderOpen size={15}/>}
          label="File Manager"
          active
          onClick={() => useLayout.getState().closeDialog()}
        />
        <SidebarAction icon={<FolderSync size={15}/>} label="Sync & Backup" onClick={openSync}/>
        <SidebarAction
          icon={<Cloud size={15}/>}
          label="Cloud Storage"
          onClick={() => openDialog("siteManager")}
        />
        <SidebarAction
          icon={<Clock3 size={15}/>}
          label="Schedules"
          onClick={() => openDialog("transferCenter")}
        />
        <SidebarAction
          icon={<ScrollText size={15}/>}
          label="Activity Logs"
          onClick={() => openDialog("transferCenter")}
        />
        <SidebarAction
          icon={<Settings size={15}/>}
          label="Settings"
          onClick={() => openDialog("settings")}
        />
      </nav>

      <div className="ghost-sites-spacer"/>
    </aside>
  );
}

function SidebarAction({
  icon,
  label,
  onClick,
  active = false,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      className={`ghost-file-nav-row ${active ? "active" : ""}`}
      onClick={onClick}
      aria-current={active ? "page" : undefined}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
