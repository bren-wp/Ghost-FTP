import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Link2, ShieldCheck, XCircle } from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { useTransfers } from "@/stores/transfersStore";
import { PROTOCOL_DEFAULT_PORT, type ConnectionProfile, type Protocol } from "@/lib/types";

type QuickProtocol = Extract<Protocol, "sftp" | "ftps" | "ftp">;

/**
 * Reference-aligned real Files chrome for Windows/Linux.
 *
 * Quick Connect uses the real ephemeral backend with a transient password.
 * Host-key verification and TLS are enforced by the existing Rust protocol
 * adapters; the form never saves a profile or transfers secrets to storage.
 */
export function PremiumFilesChrome() {
  const activeSessionId = useConnections((state) => state.activeSessionId);
  const connecting = useConnections((state) => state.connecting);
  const connectTemporary = useConnections((state) => state.connectTemporary);
  const activeProfileId = useConnections((state) => state.activeProfileId);
  const profiles = useConnections((state) => state.profiles);
  const profile = profiles.find((item) => item.id === activeProfileId);
  const [protocol, setProtocol] = useState<QuickProtocol>("sftp");
  const [host, setHost] = useState("");
  const [username, setUsername] = useState("");
  const [port, setPort] = useState(String(PROTOCOL_DEFAULT_PORT.sftp));
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [inputError, setInputError] = useState<string | null>(null);

  const parsedPort = Number(port);
  const canConnect = host.trim().length > 0 && username.trim().length > 0 &&
    Number.isInteger(parsedPort) && parsedPort >= 1 && parsedPort <= 65535;

  // Quick Connect must open a real ephemeral backend session. It never
  // creates a saved Site or launches the new-profile editor.
  const connectNow = async () => {
    if (submitting || connecting) return;
    if (!canConnect) {
      setInputError("Enter a host, username and valid port (1–65535).");
      return;
    }
    const profile: ConnectionProfile = {
      id: crypto.randomUUID(),
      name: host.trim(),
      protocol,
      host: host.trim(),
      port: parsedPort,
      username: username.trim(),
      auth: { kind: "password", password },
      defaultRemotePath: ".",
      autoConnect: false,
    };
    setSubmitting(true);
    setInputError(null);
    try {
      await connectTemporary(profile);
      setPassword("");
    } catch {
      // The store already shows a redacted backend error; never echo secrets.
      setInputError("Connection failed. Check the server and authentication details.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="ghost-premium-files-chrome" aria-label="File manager connection">
      <div className="ghost-premium-files-heading">
        <div>
          <h1>Files</h1>
          <p>Manage local and remote files in one secure workspace</p>
        </div>
      </div>
      <form
        className="ghost-premium-quickconnect"
        aria-label="Quick connection details"
        onSubmit={(event) => {
          event.preventDefault();
          void connectNow();
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
        <label className="ghost-premium-password">
          <span>Password</span>
          <input
            aria-label="Quick connection password"
            type="password"
            autoComplete="off"
            spellCheck={false}
            value={password}
            placeholder="Session password"
            onChange={(event) => setPassword(event.target.value)}
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
        <button type="submit" className="ghost-premium-review"
          disabled={submitting || connecting || !canConnect}
          aria-busy={submitting || connecting}>
          <Link2 size={15} aria-hidden="true" />
          {submitting || connecting ? "Connecting…" : "Connect"}
        </button>
      </form>
      {inputError && <p role="alert" className="ghost-premium-connection-error">{inputError}</p>}
      <div className="ghost-premium-connection-state" role="status">
        <ShieldCheck size={14} aria-hidden="true" />
        {activeSessionId && profile
          ? `Connected: ${profile.name} (${profile.protocol.toUpperCase()})`
          : "Disconnected — enter connection details or select a saved site"}
        <span>Quick Connect is temporary. Use New connection to save a site.</span>
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
