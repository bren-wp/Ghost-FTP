import { ChevronDown, ChevronRight, Plus, Server } from "lucide-react";
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

  return (
    <aside className="ghost-sites-panel">
      <button className="ghost-sites-group" onClick={() => setExpanded((v) => !v)}>
        {expanded ? <ChevronDown size={13}/> : <ChevronRight size={13}/>}<Server size={14}/><span>My Sites</span>
      </button>
      {expanded && <div className="ghost-sites-list">
        {visibleProfiles.length === 0 ? (
          <button className="ghost-site-row ghost-site-empty" onClick={() => openNewConnection()}>
            <Plus size={14}/><span>Add your first server</span>
          </button>
        ) : visibleProfiles.map((p) => {
          const connected = sessions.some((s) => s.profileId === p.id);
          const active = p.id === activeProfileId;
          return (
            <button key={p.id} className={`ghost-site-row ${active ? "active" : ""}`} onClick={() => void activate(p.id)} title={`${p.name} · ${p.host}`}>
              <Server size={14}/><span className="ghost-site-name">{p.name}</span><i className={connected ? "online" : ""}/>
            </button>
          );
        })}
      </div>}
      <div className="ghost-sites-spacer"/>
    </aside>
  );
}
