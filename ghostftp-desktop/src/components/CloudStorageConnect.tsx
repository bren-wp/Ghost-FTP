import { useRef, useState } from "react";
import { open as openNativeDialog } from "@tauri-apps/plugin-dialog";
import { Cloud, Eye, EyeOff, FolderOpen, LockKeyhole, X } from "lucide-react";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { useConnections } from "@/stores/connectionsStore";
import { useDialog } from "@/hooks/useDialog";
import { toast } from "@/stores/toastStore";
import { toastError } from "@/lib/errors";
import { ipc } from "@/lib/ipc";

type Provider = "s3" | "b2" | "r2" | "gcs" | "webdav" | "nextcloud" | "mega" | "proton";
type LiveProvider = Exclude<Provider, "mega" | "proton">;
const providers: { id: Provider; name: string; explanation: string }[] = [
  { id: "s3", name: "Amazon S3 / compatible", explanation: "Bucket, access key, secret key and region" },
  { id: "b2", name: "Backblaze B2", explanation: "B2 S3 endpoint and application key" },
  { id: "r2", name: "Cloudflare R2", explanation: "R2 S3 endpoint and API key" },
  { id: "gcs", name: "Google Cloud Storage", explanation: "Bucket and service-account JSON file" },
  { id: "webdav", name: "WebDAV", explanation: "HTTPS URL, username and password" },
  { id: "nextcloud", name: "Nextcloud", explanation: "Nextcloud HTTPS address and app password" },
  { id: "mega", name: "MEGA", explanation: "External official MEGAcmd; native integration unavailable" },
  { id: "proton", name: "Proton Drive", explanation: "External official CLI; native integration unavailable" },
];

function secureEndpoint(value: string, allowPath: boolean): URL {
  const url = new URL(value.trim());
  if (url.protocol !== "https:" || !url.hostname || url.username || url.password ||
      url.search || url.hash || (!allowPath && url.pathname !== "/")) {
    throw new Error("Use an HTTPS endpoint without credentials, query, fragment or unexpected path.");
  }
  return url;
}

function nextcloudDavUrl(raw: string, username: string): URL {
  const url = secureEndpoint(raw, true);
  if (!url.pathname.includes("/remote.php/dav/files/")) {
    const prefix = url.pathname.replace(/\/+$/, "");
    url.pathname = prefix + "/remote.php/dav/files/" + encodeURIComponent(username) + "/";
  }
  return url;
}

function validBucket(value: string): string {
  const bucket = value.trim();
  if (bucket.length < 3 || bucket.length > 63 || !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(bucket)) {
    throw new Error("Bucket must contain 3–63 valid letters, numbers, dots, underscores or hyphens.");
  }
  return bucket;
}

/** Native protocols only. Never persists credentials in frontend storage. */
export function buildStorageProfile(values: {
  provider: LiveProvider; id: string; name: string; bucket: string; region: string;
  endpoint: string; username: string; secret: string; keyPath: string;
}): ConnectionProfile {
  const { provider, id, name } = values;
  const isObject = ["s3", "b2", "r2", "gcs"].includes(provider);
  const username = values.username.trim();
  const bucket = isObject ? validBucket(values.bucket) : undefined;
  const region = values.region.trim();
  let url: URL | undefined;
  let endpoint: string | undefined;

  if (provider === "b2") {
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(region)) {
      throw new Error("Enter the B2 region from the bucket endpoint, e.g. us-west-004.");
    }
    endpoint = "https://s3." + region + ".backblazeb2.com";
    url = secureEndpoint(endpoint, false);
  } else if (provider === "r2" || (provider === "s3" && values.endpoint.trim())) {
    url = secureEndpoint(values.endpoint, false);
    endpoint = url.origin;
  } else if (provider === "webdav" || provider === "nextcloud") {
    if (!username) throw new Error("Enter your WebDAV or Nextcloud username.");
    url = provider === "nextcloud"
      ? nextcloudDavUrl(values.endpoint, username)
      : secureEndpoint(values.endpoint, true);
    endpoint = url.toString();
  }

  if (provider === "gcs") {
    if (!values.keyPath.trim()) throw new Error("Select a Google service-account JSON key file.");
  } else if (!values.secret) {
    throw new Error("Enter your access secret or application password.");
  }
  if (["s3", "b2", "r2"].includes(provider) && !username) throw new Error("Enter the access key ID.");
  if (["s3", "r2"].includes(provider) && !region) throw new Error("Enter the region.");

  const host = provider === "gcs" ? "storage.googleapis.com" : url?.hostname ?? "s3.amazonaws.com";
  const port = url?.port ? Number(url.port) : 443;
  return {
    id,
    name: name.trim() || providers.find((item) => item.id === provider)!.name + (bucket ? " · " + bucket : ""),
    protocol: (provider === "gcs" ? "gcs" : provider === "nextcloud" || provider === "webdav" ? "webdav" : "s3") as Protocol,
    host, port, username,
    auth: provider === "gcs"
      ? { kind: "key", path: values.keyPath.trim() }
      : { kind: "password", password: values.secret },
    bucket, region: isObject && provider !== "gcs" ? (provider === "r2" ? "auto" : region) : undefined,
    endpoint, group: "Cloud", defaultRemotePath: "/", autoConnect: false,
  };
}

export function CloudStorageConnect({
  onClose, onBack, onSaved,
}: { onClose: () => void; onBack: () => void; onSaved: () => void }) {
  const panelRef = useRef<HTMLElement>(null);
  const saveProfile = useConnections((state) => state.saveProfile);
  const [provider, setProvider] = useState<Provider>("s3");
  const [name, setName] = useState("");
  const [bucket, setBucket] = useState("");
  const [region, setRegion] = useState("us-east-1");
  const [endpoint, setEndpoint] = useState("");
  const [username, setUsername] = useState("");
  const [secret, setSecret] = useState("");
  const [keyPath, setKeyPath] = useState("");
  const [showSecret, setShowSecret] = useState(false);
  const [busy, setBusy] = useState<"idle" | "test" | "save">("idle");
  const [verified, setVerified] = useState(false);
  const [message, setMessage] = useState("");
  const selected = providers.find((item) => item.id === provider)!;
  const supported = provider !== "mega" && provider !== "proton";
  useDialog(panelRef, { onClose: () => { if (busy === "idle") onClose(); } });

  const changeProvider = (next: Provider) => {
    if (busy !== "idle") return;
    setProvider(next); setName(""); setBucket(""); setUsername(""); setSecret("");
    setKeyPath(""); setShowSecret(false);
    setRegion(next === "r2" ? "auto" : next === "b2" ? "" : "us-east-1");
    setEndpoint(""); setVerified(false); setMessage("");
  };
  const update = (setter: (value: string) => void, value: string) => {
    setter(value); setVerified(false); setMessage("");
  };
  const makeProfile = () => buildStorageProfile({
    provider: provider as LiveProvider, id: crypto.randomUUID(), name, bucket, region,
    endpoint, username, secret, keyPath,
  });
  const probe = async () => {
    if (!supported || busy !== "idle") return;
    let profile: ConnectionProfile;
    try { profile = makeProfile(); } catch (error) { toastError(error, "Check cloud connection fields"); return; }
    setBusy("test");
    try {
      await ipc.probeCloudStorage(profile);
      setVerified(true);
      setMessage("Connection verified with live read-only listing or WebDAV PROPFIND.");
      toast.success("Cloud connection verified");
    } catch (error) {
      setVerified(false);
      setMessage("Unable to verify. Check HTTPS, access keys, bucket and listing permissions.");
      toastError(error, "Cloud connection test failed");
    } finally { setBusy("idle"); }
  };
  const save = async () => {
    if (!supported || busy !== "idle") return;
    let profile: ConnectionProfile;
    try { profile = makeProfile(); } catch (error) { toastError(error, "Check cloud connection fields"); return; }
    setBusy("save");
    try {
      await saveProfile(profile);
      setSecret(""); setKeyPath("");
      if (verified) toast.success("Cloud connection saved", profile.name);
      else toast.warning("Saved without verification", "Test the connection before transferring files.");
      onSaved(); onClose();
    } catch (error) {
      toastError(error, "Could not save cloud connection");
    } finally { setBusy("idle"); }
  };

  return (
    <div className="fixed inset-0 z-[75] flex items-center justify-center bg-black/70 p-3" role="presentation">
      <section ref={panelRef} role="dialog" aria-modal="true" aria-label="Connect cloud storage"
        className="max-h-full w-full max-w-2xl overflow-y-auto rounded-2xl border border-border bg-[#071d30] p-5 shadow-2xl">
        <header className="mb-4 flex items-start gap-3">
          <Cloud size={24} className="text-accent"/>
          <div className="min-w-0 flex-1">
            <h2 className="text-lg font-semibold">Connect cloud storage</h2>
            <p className="text-xs text-text-muted">Select a storage service. Secrets go to the OS credential store, not to profile JSON.</p>
          </div>
          <button type="button" className="ghost-mini-button" aria-label="Close storage setup"
            disabled={busy !== "idle"} onClick={onClose}><X size={16}/></button>
        </header>
        <fieldset disabled={busy !== "idle"} className="space-y-4">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {providers.map((option) => (
              <button type="button" key={option.id} aria-pressed={provider === option.id}
                title={option.explanation}
                className={provider === option.id
                  ? "rounded-lg border border-accent bg-accent/15 p-2 text-left text-xs font-semibold"
                  : "rounded-lg border border-border bg-[#092337] p-2 text-left text-xs hover:border-accent"}
                onClick={() => changeProvider(option.id)}>
                {option.name}
                {(option.id === "mega" || option.id === "proton") &&
                  <span className="block text-[10px] font-normal text-text-muted">External tool</span>}
              </button>
            ))}
          </div>
          {!supported ? (
            <div className="rounded-lg border border-border bg-[#092337] p-4 text-sm">
              <h3 className="font-semibold">{selected.name}: native connection unavailable</h3>
              <p className="mt-2 text-xs leading-5 text-text-muted">
                {provider === "mega"
                  ? "MEGA offers official MEGAcmd on Windows and Linux. Ghost FTP does not yet implement MEGA encryption and file browsing; use the official tool independently."
                  : "The official Proton Drive CLI works on Windows and Linux. Proton's SDK is not yet ready for commercial third-party production use, so Ghost FTP does not ask for Proton passwords or encryption keys."}
              </p>
              <button type="button" className="ghost-mini-button mt-3" onClick={() => {
                void ipc.openExternalUrl(provider === "mega"
                  ? "https://mega.nz/cmd" : "https://proton.me/blog/proton-drive-cli")
                  .catch((error) => toastError(error, "Could not open official documentation"));
              }}>Open official provider instructions</button>
            </div>
          ) : (
            <>
              <p className="text-xs text-text-muted">{selected.explanation}. This connection uses provider keys or a WebDAV app password.</p>
              <label className="block text-xs text-text-muted">Connection name (optional)
                <input className="ghost-ref-input mt-1 w-full" maxLength={100} value={name}
                  onChange={(e) => update(setName, e.target.value)} placeholder={selected.name}/>
              </label>
              {["s3", "b2", "r2", "gcs"].includes(provider) && (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <label className="text-xs text-text-muted">Bucket
                    <input className="ghost-ref-input mt-1 w-full" value={bucket}
                      onChange={(e) => update(setBucket, e.target.value)} autoComplete="off" spellCheck={false}
                      placeholder="your-bucket"/>
                  </label>
                  {provider !== "gcs" && <label className="text-xs text-text-muted">Region
                    <input className="ghost-ref-input mt-1 w-full" value={region} autoComplete="off"
                      onChange={(e) => update(setRegion, e.target.value)}
                      placeholder={provider === "b2" ? "us-west-004" : "us-east-1"}/>
                  </label>}
                </div>
              )}
              {["s3", "r2", "webdav", "nextcloud"].includes(provider) && (
                <label className="block text-xs text-text-muted">
                  {provider === "nextcloud" ? "Nextcloud server or full WebDAV URL"
                    : provider === "webdav" ? "WebDAV HTTPS endpoint" : "Custom HTTPS S3 endpoint"}
                  <input type="url" className="ghost-ref-input mt-1 w-full" value={endpoint}
                    onChange={(e) => update(setEndpoint, e.target.value)} autoComplete="off"
                    placeholder={provider === "nextcloud" ? "https://cloud.example.com"
                      : provider === "webdav" ? "https://example.com/dav/"
                      : provider === "r2" ? "https://ACCOUNT_ID.r2.cloudflarestorage.com"
                      : "Optional: https://s3.example.com"}/>
                </label>
              )}
              {provider === "b2" && <p className="text-xs text-text-muted">
                Enter the B2 region shown by your bucket endpoint; the HTTPS S3 URL is generated automatically.
                Use an application key ID and key, not the master key.
              </p>}
              {provider === "gcs" ? (
                <div className="text-xs text-text-muted">
                  <span>Service-account JSON key file</span>
                  <div className="mt-1 flex items-center gap-2">
                    <input className="ghost-ref-input min-w-0 flex-1" readOnly value={keyPath}
                      aria-label="Selected service-account key" placeholder="Select a JSON key"/>
                    <button className="ghost-mini-button" type="button" onClick={() => {
                      void openNativeDialog({ multiple: false, directory: false,
                        filters: [{ name: "Service account JSON", extensions: ["json"] }] })
                        .then((value) => { if (typeof value === "string") update(setKeyPath, value); })
                        .catch((error) => toastError(error, "Could not select JSON key"));
                    }}><FolderOpen size={14}/> Browse</button>
                  </div>
                  <p className="mt-1">Protect this private key file with operating-system permissions and grant minimal bucket access.</p>
                </div>
              ) : (
                <>
                  <label className="block text-xs text-text-muted">
                    {["s3", "b2", "r2"].includes(provider) ? "Access key ID" : "Username"}
                    <input className="ghost-ref-input mt-1 w-full" autoComplete="off" value={username}
                      onChange={(e) => update(setUsername, e.target.value)}/>
                  </label>
                  <label className="block text-xs text-text-muted">
                    {["s3", "b2", "r2"].includes(provider) ? "Secret access key" : "Application password / access token"}
                    <div className="mt-1 flex items-center gap-2">
                      <input type={showSecret ? "text" : "password"}
                        className="ghost-ref-input min-w-0 flex-1" autoComplete="off" value={secret}
                        onChange={(e) => update(setSecret, e.target.value)}/>
                      <button className="ghost-mini-button" type="button" onClick={() => setShowSecret((x) => !x)}
                        aria-label={showSecret ? "Hide secret" : "Show secret"}>
                        {showSecret ? <EyeOff size={14}/> : <Eye size={14}/>}
                      </button>
                    </div>
                    {(provider === "nextcloud" || provider === "webdav") &&
                      <span className="mt-1 block">Use a revocable app password instead of the primary account password.</span>}
                  </label>
                </>
              )}
              <p className="flex items-start gap-2 text-xs text-text-muted">
                <LockKeyhole size={14} className="mt-0.5 shrink-0 text-accent"/>
                HTTPS certificate verification is mandatory. Test Connection uses a real read-only listing
                or WebDAV PROPFIND; a write-only key may not have permission to list.
              </p>
              {message && <div role="status" className={verified
                ? "rounded-lg border border-success/40 p-2 text-xs text-success"
                : "rounded-lg border border-warning/40 p-2 text-xs text-warning"}>{message}</div>}
            </>
          )}
        </fieldset>
        <footer className="mt-5 flex flex-wrap justify-end gap-2">
          <button type="button" className="ghost-mini-button" disabled={busy !== "idle"} onClick={onBack}>Back to account sign-in</button>
          <button type="button" className="ghost-mini-button" disabled={busy !== "idle"} onClick={onClose}>Cancel</button>
          {supported && <>
            <button type="button" className="ghost-mini-button" disabled={busy !== "idle"} onClick={() => void probe()}>
              {busy === "test" ? "Testing…" : "Test Connection"}
            </button>
            <button type="button" className="ghost-primary-button" disabled={busy !== "idle"} onClick={() => void save()}>
              {busy === "save" ? "Saving…" : "Save Connection"}
            </button>
          </>}
        </footer>
      </section>
    </div>
  );
}
