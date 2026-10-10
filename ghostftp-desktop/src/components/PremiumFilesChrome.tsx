import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Link2, Plus, ShieldCheck, XCircle } from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { useTransfers } from "@/stores/transfersStore";
import { PROTOCOL_DEFAULT_PORT, type Protocol } from "@/lib/types";

type QuickProtocol = Extract<Protocol, "sftp" | "ftps" | "ftp">;

/**
 * Reference-aligned real Files chrome for Windows/Linux.
 *
 * The inline connect form intentionally never handles credentials. It passes
 * only non-secret fields to the existing validated connection editor, where
 * authentication, host-key confirmation and TLS handling remain authoritative.
 */
export function PremiumFilesChrome() {
  const openNewConnection = useLayout((state) => state.openNewConnection);
  const activeSessionId = useConnections((state) => state.activeSessionId);
  const activeProfileId = useConnections((state) => state.activeProfileId);
  const profiles = useConnections((state) => state.profiles);
  const profile = profiles.find((item) => item.id === activeProfileId);
  const [protocol, setProtocol] = useState<QuickProtocol>("sftp");
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("");
  const [port, setPort] = useState(String(PROTOCOL_DEFAULT_PORT.sftp));

  const openReviewedConnection = () => {
    const parsedPort = Number(port);
    openNewConnection({
      protocol,
      host: host.trim(),
      username: username.trim(),
      port: Number.isInteger(parsedPort) && parsedPort >= 1 && parsedPort <= 65535
        ? parsedPort
        : PROTOCOL_DEFAULT_PORT[protocol],
    });
  };

  return (
    <section className="ghost-premium-files-chrome" aria-label="File manager connection">
      <div className="ghost-premium-files-heading">
        <div>
          <h1>Files</h1>
          <p>Manage local and remote files in one secure workspace</p>
        </div>
        <button type="button" className="ghost-premium-files-cta" onClick={() => openNewConnection()}>
          <Plus size={16} aria-hidden="true" />
          <span>New connection</span>
        </button>
      </div>
      <form
        className="ghost-premium-quickconnect"
        aria-label="Quick connection details"
        onSubmit={(event) => {
          event.preventDefault();
          openReviewedConnection();
        }}
      >
        <label>
          <span>Host</span>
          <input
            aria-label="Quick connection host"
            type="text"
            autoComplete="off"
            spellCheck={false}
            value={host}
            placeholder="Server address"
            onChange={(event) => setHost(event.target.value)}
            maxLength={255}
          />
        </label>
        <label>
          <span>Username</span>
          <input
            aria-label="Quick connection username"
            type="text"
            autoComplete="off"
            spellCheck={false}
            value={username}
            placeholder="Account name"
            onChange={(event) => setUsername(event.target.value)}
            maxLength={255}
          />
        </label>
        <label className="ghost-premium-port">
          <span>Port</span>
          <input
            aria-label="Quick connection port"
            type="number"
            min={1}
            max={65535}
            value={port}
            onChange={(event) => setPort(event.target.value)}
          />
        </label>
        <label className="ghost-premium-protocol">
          <span>Protocol</span>
          <select
            aria-label="Quick connection protocol"
            value={protocol}
            onChange={(event) => {
              const value = event.target.value as QuickProtocol;
              setProtocol(value);
              setPort(String(PROTOCOL_DEFAULT_PORT[value]));
            }}
          >
            <option value="sftp">SFTP</option>
            <option value="ftps">FTPS</option>
            <option value="ftp">FTP</option>
          </select>
        </label>
        <button type="submit" className="ghost-premium-review">
          <Link2 size={15} aria-hidden="true" />
          Review &amp; connect
        </button>
      </form>
      <div className="ghost-premium-connection-state" role="status">
        <ShieldCheck size={14} aria-hidden="true" />
        {activeSessionId && profile
          ? `Connected: ${profile.name} (${profile.protocol.toUpperCase()})`
          : "Disconnected — enter connection details or select a saved site"}
        <span>Credentials are requested only in the secure connection dialog</span>
      </div>
    </section>
  );
}

export function PremiumLiveTransferStrip() {
  const byId = useTransfers((state) => state.byId);
  const openDialog = useLayout((state) => state.openDialog);
  const live = useMemo(
    () => Object.values(byId)
      .filter((transfer) => transfer.status === "transferring" ||
        transfer.status === "queued" || transfer.status === "paused")
      .sort((a, b) => b.startedAt - a.startedAt),
    [byId]
  );
  const recent = live.slice(0, 3);

  return (
    <section className="ghost-premium-transfer-strip" aria-label="Active transfers">
      <div className="ghost-premium-transfer-heading">
        <div>
          <strong>Active transfers</strong>
          <span>{live.length} in progress or queued</span>
        </div>
        <button
          type="button"
          className="ghost-premium-transfer-open"
          onClick={() => openDialog("transferCenter")}
          aria-label="Open Transfer Center"
        >
          View all
        </button>
      </div>
      {recent.length === 0 ? (
        <div className="ghost-premium-transfer-idle" role="status">
          <XCircle size={15} aria-hidden="true" />
          No transfers in progress
        </div>
      ) : (
        <div className="ghost-premium-transfer-list">
          {recent.map((transfer) => {
            const progress = transfer.size > 0
              ? Math.min(100, Math.max(0, (transfer.transferred / transfer.size) * 100))
              : 0;
            const file = transfer.source.split(/[\\/]/).filter(Boolean).at(-1) ?? transfer.source;
            return (
              <button
                type="button"
                key={transfer.id}
                className="ghost-premium-transfer-item"
                onClick={() => openDialog("transferCenter")}
                aria-label={`View ${file} transfer in Transfer Center`}
              >
                {transfer.kind === "upload"
                  ? <ArrowUp size={16} aria-hidden="true" />
                  : <ArrowDown size={16} aria-hidden="true" />}
                <span className="ghost-premium-transfer-name" title={file}>{file}</span>
                <span className="ghost-premium-transfer-state">{transfer.status}</span>
                <span
                  className="ghost-premium-transfer-progress"
                  role="progressbar"
                  aria-label={`${file} progress`}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={Math.round(progress)}
                >
                  <i style={{ width: `${progress}%` }} />
                </span>
                <span className="ghost-premium-transfer-percent">{Math.round(progress)}%</span>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}
