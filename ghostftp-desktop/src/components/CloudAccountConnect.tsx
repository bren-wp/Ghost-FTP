import { useEffect, useRef, useState } from "react";
import { Cloud, ExternalLink, LockKeyhole, X } from "lucide-react";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { ipc } from "@/lib/ipc";
import { useConnections } from "@/stores/connectionsStore";
import { toast } from "@/stores/toastStore";
import { toastError } from "@/lib/errors";
import { useDialog } from "@/hooks/useDialog";

type CloudProvider = "gdrive" | "onedrive" | "dropbox" | "box";
type Availability = { gdrive: boolean; onedrive: boolean; dropbox: boolean; boxDrive: boolean };

const options: { protocol: CloudProvider; name: string; host: string; hint: string }[] = [
  { protocol: "gdrive", name: "Google Drive", host: "drive.google.com", hint: "Google Cloud OAuth desktop Client ID" },
  { protocol: "onedrive", name: "OneDrive", host: "onedrive.com", hint: "Microsoft Entra application Client ID" },
  { protocol: "dropbox", name: "Dropbox", host: "dropbox.com", hint: "Dropbox application key" },
  { protocol: "box", name: "Box", host: "box.com", hint: "Box OAuth application Client ID" },
];

function hasRegisteredApp(provider: CloudProvider, available: Availability | null): boolean {
  if (!available) return false;
  return provider === "box" ? available.boxDrive : available[provider];
}

/** Cloud account setup for Windows/Linux. Tokens never enter React state. */
export function CloudAccountConnect({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const panelRef = useRef<HTMLElement>(null);
  const saveProfile = useConnections((state) => state.saveProfile);
  const [provider, setProvider] = useState<CloudProvider>("gdrive");
  const [available, setAvailable] = useState<Availability | null>(null);
  const [accountName, setAccountName] = useState("");
  const [clientId, setClientId] = useState("");
  const [advanced, setAdvanced] = useState(false);
  const [busy, setBusy] = useState(false);
  const selected = options.find((option) => option.protocol === provider)!;
  const configured = hasRegisteredApp(provider, available);
  useDialog(panelRef, { onClose: () => { if (!busy) onClose(); } });

  useEffect(() => {
    let live = true;
    void ipc.cloudOAuthAvailability()
      .then((status) => { if (live) setAvailable(status); })
      .catch(() => { if (live) setAvailable(null); });
    return () => { live = false; };
  }, []);

  const onProviderChange = (next: CloudProvider) => {
    if (busy) return;
    setProvider(next);
    setClientId("");
    setAccountName("");
    setAdvanced(false);
  };

  const connect = async () => {
    if (busy) return;
    if (!configured && !clientId.trim()) {
      setAdvanced(true);
      toast.warning("OAuth application setup required", "Enter the public application Client ID to connect this account.");
      return;
    }
    const profileId = crypto.randomUUID();
    setBusy(true);
    try {
      const id = clientId.trim() || undefined;
      const result = provider === "gdrive"
        ? await ipc.gdriveAuthorize(profileId, id)
        : provider === "onedrive"
          ? await ipc.onedriveAuthorize(profileId, id)
          : provider === "dropbox"
            ? await ipc.dropboxAuthorize(profileId, id)
            : await ipc.boxAuthorize(profileId, id);
      const profile: ConnectionProfile = {
        id: profileId,
        protocol: provider as Protocol,
        name: accountName.trim() || result.accountLabel.trim() || selected.name,
        host: selected.host,
        port: 443,
        username: "",
        auth: { kind: "password", password: "" },
        group: "Cloud",
        autoConnect: false,
        defaultRemotePath: "/",
      };
      try {
        await saveProfile(profile);
      } catch (error) {
        // The remote sign-in succeeded but the profile could not be persisted.
        // Drop the new keychain entry so it cannot become an orphaned credential.
        await ipc.cloudOAuthDiscard(profileId, provider).catch(() => undefined);
        throw error;
      }
      toast.success("Cloud account connected", profile.name);
      onSaved();
      onClose();
    } catch (error) {
      toastError(error, "Could not connect cloud account");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[75] flex items-center justify-center bg-black/70 px-3 py-5" role="presentation">
      <section
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Connect cloud account"
        className="max-h-full w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-[#071d30] p-5 shadow-2xl"
      >
        <header className="mb-5 flex items-center gap-3">
          <Cloud size={24} className="text-accent" />
          <div className="min-w-0 flex-1">
            <h2 className="text-lg font-semibold">Connect cloud account</h2>
            <p className="text-xs text-text-muted">Use your provider's secure browser sign-in. No cloud password is entered here.</p>
          </div>
          <button type="button" aria-label="Close cloud connection" onClick={onClose} disabled={busy} className="ghost-mini-button">
            <X size={16} />
          </button>
        </header>
        <fieldset disabled={busy} className="space-y-4">
          <legend className="mb-2 text-sm font-semibold">Choose a service</legend>
          <div className="grid grid-cols-2 gap-2">
            {options.map((option) => (
              <button
                key={option.protocol}
                type="button"
                aria-pressed={provider === option.protocol}
                className={provider === option.protocol
                  ? "rounded-lg border border-accent bg-accent/15 px-3 py-3 text-left text-sm font-semibold"
                  : "rounded-lg border border-border bg-[#092337] px-3 py-3 text-left text-sm hover:border-accent"}
                onClick={() => onProviderChange(option.protocol)}
              >
                {option.name}
              </button>
            ))}
          </div>
          <label className="block text-xs text-text-muted">
            Connection name (optional)
            <input
              className="ghost-ref-input mt-1 w-full"
              maxLength={100}
              value={accountName}
              onChange={(event) => setAccountName(event.target.value)}
              placeholder={selected.name}
            />
          </label>
          <div className="rounded-lg border border-border bg-[#0b263b] p-3 text-xs text-text-muted">
            <div className="flex items-start gap-2">
              <LockKeyhole size={16} className="mt-0.5 shrink-0 text-accent" />
              <p>
                Sign-in uses OAuth 2.0 with PKCE, CSRF protection and a local loopback redirect.
                Access and refresh tokens remain in your operating system's credential store.
              </p>
            </div>
            {!configured && (
              <p className="mt-3 text-warning">
                This build has no registered {selected.name} OAuth application ID. Enter your public Client ID to authorize this service.
              </p>
            )}
            <button
              type="button"
              className="mt-3 flex items-center gap-1 text-accent hover:underline"
              aria-expanded={advanced}
              onClick={() => setAdvanced((value) => !value)}
            >
              <ExternalLink size={13} />
              {advanced ? "Hide application setup" : "Advanced: OAuth application setup"}
            </button>
            {advanced && (
              <label className="mt-3 block">
                {selected.hint}
                <input
                  className="ghost-ref-input mt-1 w-full"
                  autoComplete="off"
                  spellCheck={false}
                  maxLength={256}
                  value={clientId}
                  onChange={(event) => setClientId(event.target.value)}
                  placeholder="Public OAuth Client ID (not a client secret)"
                />
                <span className="mt-1 block text-[11px] leading-5">
                  Register a desktop/public application with redirect URI http://localhost:53682/.
                  Paste only its public ID, never a client secret or account password. The ID is saved with tokens in the OS credential store.
                </span>
              </label>
            )}
          </div>
        </fieldset>
        <footer className="mt-5 flex justify-end gap-2">
          <button type="button" className="ghost-mini-button" disabled={busy} onClick={onClose}>Cancel</button>
          <button type="button" className="ghost-primary-button" disabled={busy || available === null && !clientId.trim()} onClick={() => void connect()}>
            <ExternalLink size={15} />
            {busy ? "Finish sign-in in your browser…" : "Connect in browser"}
          </button>
        </footer>
      </section>
    </div>
  );
}
