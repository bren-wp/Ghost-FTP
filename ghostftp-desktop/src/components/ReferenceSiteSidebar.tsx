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
  ShieldCheck,
  Zap,
  Sparkles,
  MonitorSmartphone,
  CircleDot,
} from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
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

  const visibleProfiles = useMemo(() => profiles, [profiles]);

  const activate = async (profileId: string) => {
    const live = sessions.find((s) => s.profileId === profileId);
    if (live) {
      setActiveSession(live.sessionId);
      return;
    }
    try {
      await connect(profileId);
    } catch {
      // connectionsStore already shows the protocol/backend failure to the user.
      // Keep sidebar activation from leaking an unhandled rejected promise.
    }
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
        <SidebarAction icon={<FolderSync size={15}/>} label="Sync & Backup" onClick={() => openDialog("sync")}/>
        <SidebarAction
          icon={<Cloud size={15}/>}
          label="Cloud Storage"
          onClick={() => openDialog("cloudStorage")}
        />
        <SidebarAction
          icon={<Clock3 size={15}/>}
          label="Schedules"
          onClick={() => openDialog("schedules")}
        />
        <SidebarAction
          icon={<ScrollText size={15}/>}
          label="Activity Logs"
          onClick={() => openDialog("activityLogs")}
        />
        <SidebarAction
          icon={<Settings size={15}/>}
          label="Settings"
          onClick={() => openDialog("settings")}
        />
      </nav>

      <div className="ghost-sites-spacer"/>
      <div className="ghost-sidebar-capabilities" aria-label="Ghost FTP capabilities">
        <CapabilityNote icon={<ShieldCheck size={14}/>} label="Secure Connections"/>
        <CapabilityNote icon={<Zap size={14}/>} label="Fast Transfers"/>
        <CapabilityNote icon={<Sparkles size={14}/>} label="Modern Interface"/>
        <CapabilityNote icon={<MonitorSmartphone size={14}/>} label="Cross-Platform"/>
        <CapabilityNote icon={<CircleDot size={14}/>} label="Built for Creators"/>
      </div>
    </aside>
  );
}

function CapabilityNote({ icon, label }: { icon: ReactNode; label: string }) {
  return (
    <div className="ghost-sidebar-capability" role="note">
      {icon}
      <span>{label}</span>
    </div>
  );
}

function SidebarAction({
  icon,
  label,
  onClick,
  active = false,
}: {
  icon: ReactNode;
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
