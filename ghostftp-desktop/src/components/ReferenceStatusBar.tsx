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
  const sessions = useConnections((s) => s.sessions);
  const checkHealth = useConnections((s) => s.checkHealth);
  const transfersById = useTransfers((s) => s.byId);
  const rateById = useTransfers((s) => s.rateById);
  const transfers = useMemo(() => Object.values(transfersById), [transfersById]);
  const openDialog = useLayout((s) => s.openDialog);
  const profile = profiles.find((p) => p.id === activeProfileId);
  const activeSession = sessions.find((session) => session.sessionId === activeSessionId);
  const healthSupported = !!profile && ["ftp", "ftps", "sftp"].includes(profile.protocol);
  const health = activeSession?.health ?? "unknown";

  useEffect(() => {
    if (!activeSessionId || !healthSupported) return;

    const probe = () => {
      if (document.visibilityState === "visible") {
        void checkHealth(activeSessionId);
      }
    };
    const interval = window.setInterval(probe, 30_000);
    window.addEventListener("focus", probe);
    document.addEventListener("visibilitychange", probe);

    return () => {
      window.clearInterval(interval);
      window.removeEventListener("focus", probe);
      document.removeEventListener("visibilitychange", probe);
    };
  }, [activeSessionId, healthSupported, checkHealth]);

  const healthLabel =
    health === "healthy"
      ? "Healthy"
      : health === "checking"
        ? "Checking"
        : health === "unhealthy"
          ? "Needs attention"
          : "Connected";

  return <footer className="ghost-reference-statusbar">
    {activeSessionId && healthSupported ? (
      <button
        type="button"
        className={`ghost-status-health ${health}`}
        onClick={() => void checkHealth(activeSessionId)}
        disabled={health === "checking"}
        aria-label="Check connection health"
        title="Check connection health"
      >
        <span className={`dot ${health === "healthy" ? "online" : health === "unhealthy" ? "unhealthy" : ""}`}/>
        {healthLabel}
      </button>
    ) : (
      <span className={`dot ${activeSessionId ? "online" : ""}`}/>
    )}
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
    // Background windows should not wake React once per second just to
    // redraw an invisible clock; refresh immediately on foreground return.
    let interval: number | undefined;
    const syncVisibility = () => {
      if (interval !== undefined) window.clearInterval(interval);
      interval = undefined;
      if (document.visibilityState === "visible") {
        setNow(Date.now());
        interval = window.setInterval(() => setNow(Date.now()), 1000);
      }
    };
    syncVisibility();
    document.addEventListener("visibilitychange", syncVisibility);
    return () => {
      if (interval !== undefined) window.clearInterval(interval);
      document.removeEventListener("visibilitychange", syncVisibility);
    };
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
