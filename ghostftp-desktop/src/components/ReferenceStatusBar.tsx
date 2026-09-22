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

  return <footer className="ghost-reference-statusbar">
    <span className={`dot ${activeSessionId ? "online" : ""}`}/>
    <span>{activeSessionId && profile ? `Connected to ${profile.host} (${profile.protocol.toUpperCase()})` : "Ready"}</span>
    <span className="grow"/>
    <TransferMetrics transfers={transfers}/>
    <StatusClock/>
    <span className="encoding"><i className="dot online"/> UTF-8</span>
  </footer>;
}

function TransferMetrics({ transfers }: { transfers: ReturnType<typeof Object.values<import("@/lib/types").Transfer>> }) {
  const hasLiveTransfer = transfers.some((transfer) => transfer.status === "transferring");
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (!hasLiveTransfer) {
      setNow(Date.now());
      return;
    }
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [hasLiveTransfer]);

  const metrics = useMemo(() => {
    let up = 0, down = 0, active = 0;
    for (const transfer of transfers) {
      if (transfer.status !== "transferring") continue;
      active++;
      const elapsed = Math.max(1, now / 1000 - transfer.startedAt);
      const speed = transfer.transferred / elapsed;
      if (transfer.kind === "upload") up += speed;
      else down += speed;
    }
    return { up, down, active };
  }, [transfers, now]);

  return <>
    <span><Server size={12}/> {metrics.active} transfer{metrics.active === 1 ? "" : "s"} active</span>
    <span className="up"><ArrowUp size={13}/>{formatSpeed(metrics.up)}</span>
    <span className="down"><ArrowDown size={13}/>{formatSpeed(metrics.down)}</span>
  </>;
}

function StatusClock() {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);
  return <span>Local time: {new Date(now).toLocaleTimeString("en-GB",{hour:"2-digit",minute:"2-digit",second:"2-digit",hour12:false})}</span>;
}

function formatSpeed(n:number){if(!n)return "0 B/s";const u=["B/s","KB/s","MB/s","GB/s"];let v=n,i=0;while(v>=1024&&i<u.length-1){v/=1024;i++;}return `${v>=10||i===0?v.toFixed(0):v.toFixed(1)} ${u[i]}`;}
