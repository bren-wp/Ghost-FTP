import { useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Server } from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { liveTransferRate, useTransfers, type TransferRateSample } from "@/stores/transfersStore";
import { useLayout } from "@/stores/layoutStore";
import type { Transfer } from "@/lib/types";

export function ReferenceStatusBar() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const transfersById = useTransfers((s) => s.byId);
  const rateById = useTransfers((s) => s.rateById);
  const transfers = useMemo(() => Object.values(transfersById), [transfersById]);
  const openDialog = useLayout((s) => s.openDialog);
  const profile = profiles.find((p) => p.id === activeProfileId);

  return <footer className="ghost-reference-statusbar">
    <span className={`dot ${activeSessionId ? "online" : ""}`}/>
    <span>{activeSessionId && profile ? `Connected to ${profile.host} (${profile.protocol.toUpperCase()})` : "Ready"}</span>
    <span className="grow"/>
    <TransferMetrics transfers={transfers} rateById={rateById} onOpen={() => openDialog("transferCenter")}/>
    <StatusClock/>
    <span className="encoding"><i className="dot online"/> UTF-8</span>
  </footer>;
}

function TransferMetrics({
  transfers,
  rateById,
  onOpen,
}: {
  transfers: Transfer[];
  rateById: Record<string, TransferRateSample>;
  onOpen: () => void;
}) {
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
      const speed = liveTransferRate(transfer, rateById[transfer.id], now);
      if (transfer.kind === "upload") up += speed;
      else down += speed;
    }
    return { up, down, active };
  }, [transfers, rateById, now]);

  return (
    <button
      type="button"
      className="ghost-status-transfer-link"
      onClick={onOpen}
      aria-label="Open Transfers"
      title="Open Transfers"
    >
      <span><Server size={12}/> {metrics.active} transfer{metrics.active === 1 ? "" : "s"} active</span>
      <span className="up"><ArrowUp size={13}/>{formatSpeed(metrics.up)}</span>
      <span className="down"><ArrowDown size={13}/>{formatSpeed(metrics.down)}</span>
    </button>
  );
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
