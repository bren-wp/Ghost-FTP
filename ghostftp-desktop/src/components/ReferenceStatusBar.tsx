import { useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Server } from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useTransfers } from "@/stores/transfersStore";

export function ReferenceStatusBar() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const transfersById = useTransfers((s) => s.byId);
  const transfers = useMemo(() => Object.values(transfersById), [transfersById]);
  const profile = profiles.find((p) => p.id === activeProfileId);
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const id = window.setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(id); }, []);

  const metrics = useMemo(() => {
    let up = 0, down = 0, active = 0;
    for (const t of transfers) {
      if (t.status !== "transferring" && t.status !== "queued") continue;
      active++;
      const elapsed = Math.max(1, (now - t.startedAt) / 1000);
      const speed = t.transferred / elapsed;
      if (t.kind === "upload") up += speed; else down += speed;
    }
    return { up, down, active };
  }, [transfers, now]);

  return <footer className="ghost-reference-statusbar">
    <span className={`dot ${activeSessionId ? "online" : ""}`}/>
    <span>{activeSessionId && profile ? `Connected to ${profile.host}` : "Ready"}</span>
    <span className="grow"/>
    <span><Server size={12}/> {metrics.active} transfer{metrics.active === 1 ? "" : "s"} active</span>
    <span className="up"><ArrowUp size={13}/>{formatSpeed(metrics.up)}</span>
    <span className="down"><ArrowDown size={13}/>{formatSpeed(metrics.down)}</span>
    <span>Server time: {new Date(now).toLocaleTimeString("en-GB",{hour:"2-digit",minute:"2-digit",hour12:false})}</span>
    <span className="encoding"><i className="dot online"/> UTF-8</span>
  </footer>;
}
function formatSpeed(n:number){if(!n)return "0 B/s";const u=["B/s","KB/s","MB/s","GB/s"];let v=n,i=0;while(v>=1024&&i<u.length-1){v/=1024;i++;}return `${v>=10||i===0?v.toFixed(0):v.toFixed(1)} ${u[i]}`;}
