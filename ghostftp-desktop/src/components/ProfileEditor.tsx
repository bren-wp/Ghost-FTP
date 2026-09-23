import { useCallback, useEffect, useId, useRef, useState } from "react";
import { useConnections } from "@/stores/connectionsStore";
import { useSettings } from "@/stores/settingsStore";
import { useDialog } from "@/hooks/useDialog";
import {
  PROTOCOL_DEFAULT_PORT,
  PROTOCOL_LABEL,
  S3_PROVIDER_PRESETS,
  WEBDAV_PROVIDER_PRESETS,
  isObjectProtocol,
  isAgentProtocol,
  type AuthMethod,
  type ConnectionProfile,
  type Protocol,
  type S3Provider,
  type WebdavProvider,
  type DiscoveredAgent,
  type GeneratedKey,
  type SshKeyType,
} from "@/lib/types";
import {
  ShieldOff,
  Cloud,
  Globe,
  Box,
  Eye,
  EyeOff,
  Wand2,
  Check,
  MonitorSmartphone,
  Radar,
  Loader2,
  Link2,
  KeyRound,
  Copy,
  Sparkles,
  FolderOpen,
  X,
} from "lucide-react";
import { open, save } from "@tauri-apps/plugin-dialog";
import { cn } from "@/lib/cn";
import { BrandIcon, protocolIcon } from "@/lib/brandIcons";
import { BRAND_ICONS } from "@/lib/brandIconData";
import { monogram } from "@/lib/format";
import { IconPicker } from "./IconPicker";
import { generatePassword } from "@/lib/password";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { messageOf } from "@/lib/errors";

interface Props {
  profile: ConnectionProfile | null;
  /** Seed values for a NEW connection (e.g. from a ghostftp:// deep link). Ignored
   *  when editing an existing profile. Never carries credentials. */
  prefill?: Partial<ConnectionProfile> | null;
  onClose: () => void;
}

function genId(): string {
  return crypto.randomUUID();
}

/// Best-effort hostname from a WebDAV server URL, for the rail label.
function hostFromUrl(u: string): string {
  try {
    return new URL(u.includes("://") ? u : `https://${u}`).hostname;
  } catch (error) {
    if (!(error instanceof TypeError)) {
      console.warn("Couldn't derive a hostname from the WebDAV URL", error);
    }
    return u;
  }
}

/// Normalize a Shopify store domain for the profile host: strip any
/// scheme/path, and append `.myshopify.com` when only the shop name was typed.
function normalizeShopDomain(s: string): string {
  const t = s
    .trim()
    .toLowerCase()
    .replace(/^https?:\/\//, "")
    .replace(/\/.*$/, "");
  if (!t) return t;
  return t.endsWith(".myshopify.com") ? t : `${t}.myshopify.com`;
}

/// Normalize a Dynamics 365 environment URL for the profile host: strip any
/// scheme/path, and append `.crm.dynamics.com` when only the org name was
/// typed (regional variants like `.crm4.dynamics.com` are kept as typed).
function normalizeDynamicsUrl(s: string): string {
  const t = s
    .trim()
    .toLowerCase()
    .replace(/^https?:\/\//, "")
    .replace(/\/.*$/, "");
  if (!t) return t;
  return t.includes(".") ? t : `${t}.crm.dynamics.com`;
}

/// Guess which provider a saved S3 profile belongs to from its endpoint URL.
/// We never need this to be perfect — it just preselects a preset button.
function guessProvider(endpoint?: string): S3Provider {
  if (!endpoint) return "aws";
  const e = endpoint.toLowerCase();
  if (e.includes("r2.cloudflarestorage.com")) return "r2";
  if (e.includes("backblazeb2.com")) return "b2";
  if (e.includes("wasabisys.com")) return "wasabi";
  if (e.includes("digitaloceanspaces.com")) return "spaces";
  if (e.includes("storjshare.io")) return "storj";
  if (e.includes("your-objectstorage.com")) return "hetzner";
  if (e.includes("scw.cloud")) return "scaleway";
  if (e.includes("oraclecloud.com")) return "oci";
  if (e.includes("cloud-object-storage.appdomain.cloud")) return "ibm";
  if (e.includes("supabase.co")) return "supabase";
  // A bare endpoint we don't recognize is some self-hosted / niche S3 server.
  return "generic";
}

/// Protocol picker, grouped for the left rail. New backends slot into a group
/// instead of stretching a flat grid taller — the rail just scrolls.
const PROTOCOL_GROUPS: { label: string; items: Protocol[] }[] = [
  { label: "Servers", items: ["sftp", "ftp", "ftps"] },
  { label: "Object storage", items: ["s3", "azure", "gcs"] },
  { label: "Web", items: ["webdav", "http"] },
  { label: "Cloud drives", items: ["dropbox", "onedrive", "gdrive", "box"] },
  { label: "Commerce", items: ["shopify", "hubspot", "dynamics"] },
  { label: "Machine", items: ["ghostftp-agent"] },
];

// Real vendor marks for the provider preset buttons (Plan 14 Phase 3). A
// provider with no bundled logo (Storj, the "generic"/self-hosted presets)
// falls back to the neutral lucide glyph in the button.
const S3_PROVIDER_ICON: Partial<Record<S3Provider, string>> = {
  aws: "logos:aws-s3",
  r2: "logos:cloudflare",
  b2: "simple-icons:backblaze",
  wasabi: "simple-icons:wasabi",
  spaces: "logos:digital-ocean",
  minio: "simple-icons:minio",
  hetzner: "simple-icons:hetzner",
  scaleway: "simple-icons:scaleway",
  oci: "logos:oracle",
  ibm: "logos:ibm",
  supabase: "logos:supabase-icon",
};

const WEBDAV_PROVIDER_ICON: Partial<Record<WebdavProvider, string>> = {
  nextcloud: "simple-icons:nextcloud",
  owncloud: "simple-icons:owncloud",
  storagebox: "simple-icons:hetzner",
};

export function ProfileEditor({ profile, prefill, onClose }: Props) {
  const saveProfile = useConnections((s) => s.saveProfile);
  const connectProfile = useConnections((s) => s.connect);
  const profiles = useConnections((s) => s.profiles);
  const defaultPort = useSettings((s) => s.defaultPort);

  // Existing rail groups, offered as datalist suggestions for the Group field.
  const knownGroups = Array.from(
    new Set(profiles.map((p) => p.group).filter((g): g is string => !!g))
  );

  // Seed values: the profile being edited, or a deep-link prefill for a new
  // one. `profile` still gates "is this an edit" (title, pairing persistence).
  const seed = profile ?? prefill ?? null;

  // Stable id for the lifetime of the editor, so pairing (which persists to this
  // id) and a later Save target the same profile.
  const [id] = useState(profile?.id ?? genId());
  const [name, setName] = useState(seed?.name ?? "");
  const [protocol, setProtocol] = useState<Protocol>(seed?.protocol ?? "sftp");
  // Ghost FTP Agent: the pinned daemon key. Set by pairing; carried through Save.
  const [agentKey, setAgentKey] = useState<string | undefined>(seed?.agentKey);
  const [host, setHost] = useState(seed?.host ?? "");
  const [port, setPort] = useState(seed?.port ?? defaultPort);
  const [portTouched, setPortTouched] = useState<boolean>(!!seed?.port);
  const [username, setUsername] = useState(seed?.username ?? "");
  const [authKind, setAuthKind] = useState<AuthMethod["kind"]>(
    profile?.auth.kind ?? "password"
  );
  const [password, setPassword] = useState(
    profile?.auth.kind === "password" ? profile.auth.password : ""
  );
  const [keyPath, setKeyPath] = useState(
    profile?.auth.kind === "key" ? profile.auth.path : ""
  );
  const [passphrase, setPassphrase] = useState(
    profile?.auth.kind === "key" ? profile.auth.passphrase ?? "" : ""
  );
  const [defaultRemotePath, setDefaultRemotePath] = useState(
    seed?.defaultRemotePath ?? "."
  );
  const [autoConnect, setAutoConnect] = useState(profile?.autoConnect ?? false);
  const [group, setGroup] = useState(seed?.group ?? "");
  // Custom rail bubble glyph: emoji/short string, or a bundled Iconify key
  // (picked via the IconPicker grid; custom keys still possible via search).
  const [icon, setIcon] = useState(seed?.icon ?? "");
  // A value with a colon is treated as an Iconify key; unknown keys warn and
  // the rail falls back to the name monogram.
  const iconUnknown = icon.trim().includes(":") && !BRAND_ICONS[icon.trim()];

  // S3-only state.
  const [bucket, setBucket] = useState(seed?.bucket ?? "");
  const [region, setRegion] = useState(seed?.region ?? "us-east-1");
  const [endpoint, setEndpoint] = useState(seed?.endpoint ?? "");
  const [s3Provider, setS3Provider] = useState<S3Provider>(
    seed?.protocol === "s3" ? guessProvider(seed.endpoint) : "aws"
  );
  // Azure-only state.
  const [azureAccount, setAzureAccount] = useState(seed?.account ?? "");

  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });

  const onProtocolChange = (p: Protocol) => {
    setProtocol(p);
    if (!portTouched) {
      setPort(PROTOCOL_DEFAULT_PORT[p]);
    }
    if (isObjectProtocol(p)) {
      // Object stores authenticate by key material, not interactive login. S3 /
      // Azure carry it in the password field; GCS uses a service-account JSON,
      // which defaults to a key *file* path (Password mode pastes the JSON).
      setAuthKind(p === "gcs" ? "key" : "password");
    }
    if ((p === "ftp" || p === "ftps") && authKind !== "password") {
      setAuthKind("password");
    }
  };

  const onS3ProviderChange = (p: S3Provider) => {
    setS3Provider(p);
    const preset = S3_PROVIDER_PRESETS[p];
    setRegion(preset.defaultRegion);
    if (p === "aws") {
      setEndpoint("");
    }
  };

  const isFtp = protocol === "ftp" || protocol === "ftps";
  const isS3 = protocol === "s3";
  const isAzure = protocol === "azure";
  const isGcs = protocol === "gcs";
  const isWebdav = protocol === "webdav";
  const isHttp = protocol === "http";
  const isDropbox = protocol === "dropbox";
  const isOnedrive = protocol === "onedrive";
  const isGdrive = protocol === "gdrive";
  const isBox = protocol === "box";
  const isShopify = protocol === "shopify";
  const isHubSpot = protocol === "hubspot";
  const isDynamics = protocol === "dynamics";
  const isCloudOAuth = isDropbox || isOnedrive || isGdrive || isBox;
  const isObject = isObjectProtocol(protocol);
  const isAgent = isAgentProtocol(protocol);

  // OAuth clouds (Dropbox/OneDrive/…): authorization state. Editing an
  // already-authorized profile (its account label is persisted) starts authorized.
  const [cloudAuthed, setCloudAuthed] = useState<boolean>(
    (seed?.protocol === "dropbox" ||
      seed?.protocol === "onedrive" ||
      seed?.protocol === "gdrive" ||
      seed?.protocol === "box") &&
      !!seed?.account
  );
  const [cloudAccount, setCloudAccount] = useState<string>(seed?.account ?? "");

  // Shopify: the Admin credential lives in the OS keychain
  // (`shopify:{profile_id}`) — never in profiles.json. The editor only ever
  // sets it (one-way) and asks whether one already exists.
  const [shopifyMode, setShopifyMode] = useState<"token" | "client">("token");
  const [shopifyToken, setShopifyToken] = useState("");
  const [shopifyClientId, setShopifyClientId] = useState("");
  const [shopifyClientSecret, setShopifyClientSecret] = useState("");
  const [shopifySecretSaved, setShopifySecretSaved] = useState(false);
  useEffect(() => {
    if (protocol === "shopify") {
      ipc
        .apiKeyStatus(`shopify:${id}`)
        .then(setShopifySecretSaved)
        .catch((error) => toast.error("Couldn't read Shopify credential status", String(error)));
    }
  }, [protocol, id]);

  /// The Shopify credential as stored in the keychain: the bare access token,
  /// or `client_id:client_secret` for a Dev Dashboard app.
  const shopifySecretEntered =
    !!shopifyToken.trim() ||
    !!shopifyClientId.trim() ||
    !!shopifyClientSecret.trim();
  const shopifySecret =
    shopifyMode === "token"
      ? shopifyToken.trim()
      : `${shopifyClientId.trim()}:${shopifyClientSecret.trim()}`;
  const shopifyDomainOk = /^[a-z0-9][a-z0-9-]*\.myshopify\.com$/.test(
    normalizeShopDomain(host)
  );
  const shopifySecretOk =
    shopifySecretSaved ||
    (shopifyMode === "token"
      ? !!shopifyToken.trim()
      : !!shopifyClientId.trim() && !!shopifyClientSecret.trim());

  // HubSpot: the private-app access token (`pat-…`) lives in the OS keychain
  // (`hubspot:{profile_id}`) — never in profiles.json. The editor only ever
  // sets it (one-way) and asks whether one already exists.
  const [hubspotToken, setHubspotToken] = useState("");
  const [hubspotSecretSaved, setHubspotSecretSaved] = useState(false);
  useEffect(() => {
    if (protocol === "hubspot") {
      ipc
        .apiKeyStatus(`hubspot:${id}`)
        .then(setHubspotSecretSaved)
        .catch((error) => toast.error("Couldn't read HubSpot credential status", String(error)));
    }
  }, [protocol, id]);

  const hubspotSecretOk = hubspotSecretSaved || !!hubspotToken.trim();
  /// HubSpot private-app tokens start with `pat-`; anything else is almost
  /// certainly the wrong credential pasted in — warn, don't block.
  const hubspotPatWarn =
    !!hubspotToken.trim() && !hubspotToken.trim().startsWith("pat-");

  // Dynamics 365: two auth modes. Client credentials: the app-registration
  // credential lives in the OS keychain (`dynamics:{profile_id}`,
  // `tenant_id:client_id:client_secret`) — never in profiles.json; the editor
  // only ever sets it (one-way) and asks whether one already exists. Delegated
  // ("Sign in with Microsoft"): the OAuth TokenSet lives in the keychain under
  // the `ghostftp-dynamics` service (set by `dynamics_authorize`); the authorized
  // account label is persisted on the profile, like the OAuth clouds. The
  // backend prefers the client-credentials blob when both exist, so saving in
  // delegated mode clears the blob.
  const [dynamicsMode, setDynamicsMode] = useState<"client" | "delegated">(
    seed?.protocol === "dynamics" && !!seed?.account ? "delegated" : "client"
  );
  const [dynamicsTenantId, setDynamicsTenantId] = useState("");
  const [dynamicsClientId, setDynamicsClientId] = useState("");
  const [dynamicsClientSecret, setDynamicsClientSecret] = useState("");
  const [dynamicsSecretSaved, setDynamicsSecretSaved] = useState(false);
  const [dynamicsAuthed, setDynamicsAuthed] = useState<boolean>(
    seed?.protocol === "dynamics" && !!seed?.account
  );
  const [dynamicsAccount, setDynamicsAccount] = useState<string>(
    seed?.protocol === "dynamics" ? (seed?.account ?? "") : ""
  );
  useEffect(() => {
    if (protocol === "dynamics") {
      ipc
        .apiKeyStatus(`dynamics:${id}`)
        .then(setDynamicsSecretSaved)
        .catch((error) => toast.error("Couldn't read Dynamics credential status", String(error)));
    }
  }, [protocol, id]);

  /// The Dynamics credential as stored in the keychain:
  /// `tenant_id:client_id:client_secret` (the Shopify client-mode shape).
  const dynamicsSecretEntered =
    !!dynamicsTenantId.trim() ||
    !!dynamicsClientId.trim() ||
    !!dynamicsClientSecret.trim();
  const dynamicsSecret = `${dynamicsTenantId.trim()}:${dynamicsClientId.trim()}:${dynamicsClientSecret.trim()}`;
  const dynamicsUrlOk = /^[a-z0-9][a-z0-9-]*\.crm\d*\.dynamics\.com$/.test(
    normalizeDynamicsUrl(host)
  );
  const dynamicsCredOk =
    dynamicsMode === "client"
      ? dynamicsSecretSaved ||
        (!!dynamicsTenantId.trim() &&
          !!dynamicsClientId.trim() &&
          !!dynamicsClientSecret.trim())
      : dynamicsAuthed;

  /// Build the profile from the current form state. Shared by Save and by the
  /// pairing flow (which must persist the profile before it can pair by id).
  const buildProfile = (): ConnectionProfile => {
    let auth: AuthMethod;
    if (profile?.auth.kind === "keyref") {
      // Grant-managed key: lives in the OS keychain, never edited here.
      auth = profile.auth;
    } else if (authKind === "password") {
      auth = { kind: "password", password };
    } else if (authKind === "key") {
      auth = { kind: "key", path: keyPath, passphrase: passphrase || undefined };
    } else {
      auth = { kind: "agent" };
    }
    return {
      id,
      name:
        name ||
        (isS3
          ? `${bucket}@${s3Provider}`
          : isAzure
            ? `${azureAccount}/${bucket}`
            : isGcs
              ? `${bucket}@gcs`
              : isWebdav || isHttp
                ? `${username ? `${username}@` : ""}${hostFromUrl(endpoint)}`
                : isCloudOAuth
                  ? cloudAccount || PROTOCOL_LABEL[protocol]
                  : isShopify
                    ? normalizeShopDomain(host) || "Shopify"
                    : isHubSpot
                      ? "HubSpot"
                      : isDynamics
                        ? normalizeDynamicsUrl(host) || "Dynamics 365"
                        : isAgent
                        ? `Agent @ ${host}`
                        : `${username}@${host}`),
      protocol,
      host: isObject
        ? endpoint ||
          (isAzure
            ? "blob.core.windows.net"
            : isGcs
              ? "storage.googleapis.com"
              : "s3.amazonaws.com")
        : isWebdav || isHttp
          ? hostFromUrl(endpoint)
          : isCloudOAuth
            ? `${protocol}.com`
            : isHubSpot
              ? "api.hubapi.com"
              : isShopify
                ? normalizeShopDomain(host)
                : isDynamics
                  ? normalizeDynamicsUrl(host)
                  : host,
      port,
      username: isAzure
        ? azureAccount
        : isAgent || isGcs || isCloudOAuth || isShopify || isHubSpot || isDynamics
          ? ""
          : username,
      auth:
        isAgent || isCloudOAuth || isShopify || isHubSpot || isDynamics
          ? { kind: "password", password: "" }
          : auth,
      defaultRemotePath: defaultRemotePath || undefined,
      color: profile?.color,
      autoConnect: autoConnect || undefined,
      bucket: isObject ? bucket : undefined,
      region: isS3 ? region : undefined,
      endpoint: isObject || isWebdav || isHttp ? endpoint || undefined : undefined,
      account: isAzure
        ? azureAccount
        : isCloudOAuth
          ? cloudAccount || undefined
          : isDynamics && dynamicsMode === "delegated"
            ? dynamicsAccount || undefined
            : undefined,
      agentKey: isAgent ? agentKey : undefined,
      group: group.trim() || undefined,
      icon: icon.trim() || undefined,
      sortOrder: profile?.sortOrder,
      // Bastion hop (grant-imported only): carried through untouched.
      jumpHost: profile?.jumpHost,
      jumpPort: profile?.jumpPort,
      jumpUsername: profile?.jumpUsername,
    };
  };

  const save = async () => {
    // Shopify: persist the credential to the OS keychain first (one-way set —
    // it never lands in profiles.json). Blank fields keep the saved one.
    if (isShopify && shopifySecretEntered) {
      await ipc.setApiKey(`shopify:${id}`, shopifySecret);
    }
    // HubSpot: same one-way keychain set for the private-app token.
    if (isHubSpot && hubspotToken.trim()) {
      await ipc.setApiKey(`hubspot:${id}`, hubspotToken.trim());
    }
    // Dynamics: same one-way keychain set for the app-registration credential.
    // In delegated mode the blob is cleared instead — the backend prefers it
    // over the OAuth tokens when both exist.
    if (isDynamics && dynamicsMode === "client" && dynamicsSecretEntered) {
      await ipc.setApiKey(`dynamics:${id}`, dynamicsSecret);
    }
    if (isDynamics && dynamicsMode === "delegated") {
      await ipc.setApiKey(`dynamics:${id}`, "");
    }
    await saveProfile(buildProfile());
    onClose();
  };

  /// Pair with the daemon at host:port using a 6-digit code. Nothing is
  /// persisted until the daemon acknowledges, so a failed attempt leaves no
  /// half-configured connection behind. On success the profile is saved right
  /// away (pairing IS the consent step — demanding another Save was a dead
  /// end), the editor closes, and the connection opens.
  const pair = async (code: string): Promise<void> => {
    const res = await ipc.pairAgent(host.trim(), port, code);
    // buildProfile() reads state that React hasn't committed yet — pass the
    // fresh values through explicitly.
    const finalName = name || res.hostname || `Agent @ ${host}`;
    setAgentKey(res.serverKey);
    if (!name) setName(finalName);
    // Android's process directory is not shared storage. Start new/default
    // connections where the companion's storage permission applies.
    const initialPath = res.os === "android" && (!defaultRemotePath.trim() || defaultRemotePath === ".")
      ? "/sdcard"
      : defaultRemotePath;
    await saveProfile({
      ...buildProfile(),
      name: finalName,
      agentKey: res.serverKey,
      defaultRemotePath: initialPath || undefined,
    });
    toast.success(
      `Paired with ${res.hostname || host}`,
      `${res.os} · ${res.fingerprint}`
    );
    onClose();
    void connectProfile(id).catch((error) => {
      // The connection store already shows the actionable toast. Handle the
      // rejection here as well so a failed post-pair connect never becomes an
      // unhandled promise rejection.
      console.warn("Couldn't connect after pairing", messageOf(error));
    });
  };

  const gcsCredOk = authKind === "key" ? !!keyPath : !!password;
  const canSave = isS3
    ? !!bucket && !!username && !!password
    : isAzure
      ? !!azureAccount && !!bucket && !!password
      : isGcs
        ? !!bucket && gcsCredOk
        : isWebdav
          ? !!endpoint && !!password
          : isHttp
            ? !!endpoint
            : isShopify
              ? shopifyDomainOk && shopifySecretOk
              : isHubSpot
                ? hubspotSecretOk
                : isDynamics
                  ? dynamicsUrlOk && dynamicsCredOk
                  : isCloudOAuth
                ? cloudAuthed
                : isAgent
                  ? !!host && !!agentKey
                  : !!host && !!username;
  // Name what's still required so a disabled Save isn't a dead end.
  const missing = (
    isS3
      ? [!bucket && "bucket", !username && "access key ID", !password && "secret key"]
      : isAzure
        ? [!azureAccount && "account", !bucket && "container", !password && "access key"]
        : isGcs
          ? [
              !bucket && "bucket",
              !gcsCredOk && (authKind === "key" ? "key file path" : "JSON key"),
            ]
          : isWebdav
            ? [!endpoint && "server URL", !password && "password / token"]
            : isHttp
              ? [!endpoint && "URL"]
              : isShopify
                ? [
                    !shopifyDomainOk && "store domain",
                    !shopifySecretOk && "API credential",
                  ]
                : isHubSpot
                  ? [!hubspotSecretOk && "private-app token"]
                  : isDynamics
                    ? [
                        !dynamicsUrlOk && "environment URL",
                        !dynamicsCredOk &&
                          (dynamicsMode === "client"
                            ? "client credentials"
                            : "Microsoft sign-in"),
                      ]
                    : isCloudOAuth
                ? [!cloudAuthed && `${PROTOCOL_LABEL[protocol]} authorization`]
                : isAgent
                  ? [!host && "host", !agentKey && "pairing"]
                  : [!host && "host", !username && "username"]
  ).filter(Boolean);

  return (
    <div className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal flex max-h-[88vh] w-[52rem] max-w-[94vw] overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        {/* Left rail — grouped protocol picker. Scrolls as the list grows so the
            dialog stays wide-and-short instead of tall-and-narrow. */}
        <nav className="flex w-52 shrink-0 flex-col gap-3 overflow-y-auto border-r border-border bg-bg-subtle/50 p-2">
          <div
            id={titleId}
            className="px-2 pb-0.5 pt-1 text-[15px] font-semibold tracking-tight"
          >
            {profile ? "Edit connection" : "New connection"}
          </div>
          {PROTOCOL_GROUPS.map((grp) => (
            <div key={grp.label}>
              <div className="px-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
                {grp.label}
              </div>
              <div className="flex flex-col gap-0.5">
                {grp.items.map((p) => (
                  <ProtocolButton
                    key={p}
                    active={protocol === p}
                    onClick={() => onProtocolChange(p)}
                    protocol={p}
                    label={PROTOCOL_LABEL[p]}
                  />
                ))}
              </div>
            </div>
          ))}
        </nav>

        {/* Right pane — the form for the selected protocol. */}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex items-center gap-2 border-b border-border px-5 py-3">
            <span className="text-sm font-semibold">{PROTOCOL_LABEL[protocol]}</span>
            <span className="text-[11px] text-text-dim">{protocolHint(protocol)}</span>
            <div className="flex-1" />
            <button
              onClick={onClose}
              aria-label="Close"
              className="rounded-md p-1.5 text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <X size={14} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto px-5 py-4">
        <div className="flex gap-2">
          <Field label="Name" className="flex-1">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={isS3 ? "my-bucket" : "my-prod-box"}
              className={inputCls}
            />
          </Field>
          <Field label="Icon (optional)" className="w-40">
            <IconPicker
              value={icon}
              onChange={setIcon}
              fallback={monogram(name || "?")}
            />
          </Field>
          <Field label="Group (optional)" className="w-40">
            <input
              value={group}
              onChange={(e) => setGroup(e.target.value)}
              placeholder="e.g. Production"
              list="profile-editor-groups"
              className={inputCls}
            />
            <datalist id="profile-editor-groups">
              {knownGroups.map((g) => (
                <option key={g} value={g} />
              ))}
            </datalist>
          </Field>
        </div>

        {iconUnknown && (
          <Hint tone="warn">
            Unknown icon key — the bubble falls back to the name monogram.
            Bundled keys look like <code>mdi:rocket-launch</code>.
          </Hint>
        )}

        {isS3 ? (
          <S3Section
            provider={s3Provider}
            onProviderChange={onS3ProviderChange}
            bucket={bucket}
            setBucket={setBucket}
            region={region}
            setRegion={setRegion}
            endpoint={endpoint}
            setEndpoint={setEndpoint}
            accessKey={username}
            setAccessKey={setUsername}
            secretKey={password}
            setSecretKey={setPassword}
          />
        ) : isAzure ? (
          <AzureSection
            account={azureAccount}
            setAccount={setAzureAccount}
            container={bucket}
            setContainer={setBucket}
            accessKey={password}
            setAccessKey={setPassword}
            endpoint={endpoint}
            setEndpoint={setEndpoint}
          />
        ) : isGcs ? (
          <GcsSection
            bucket={bucket}
            setBucket={setBucket}
            keyMode={authKind === "key" ? "file" : "paste"}
            setKeyMode={(m) => setAuthKind(m === "file" ? "key" : "password")}
            keyPath={keyPath}
            setKeyPath={setKeyPath}
            keyJson={password}
            setKeyJson={setPassword}
          />
        ) : isWebdav ? (
          <WebdavSection
            url={endpoint}
            setUrl={setEndpoint}
            username={username}
            setUsername={setUsername}
            password={password}
            setPassword={setPassword}
          />
        ) : isHttp ? (
          <HttpSection
            url={endpoint}
            setUrl={setEndpoint}
            username={username}
            setUsername={setUsername}
            password={password}
            setPassword={setPassword}
          />
        ) : isShopify ? (
          <ShopifySection
            domain={host}
            setDomain={setHost}
            mode={shopifyMode}
            setMode={setShopifyMode}
            token={shopifyToken}
            setToken={setShopifyToken}
            clientId={shopifyClientId}
            setClientId={setShopifyClientId}
            clientSecret={shopifyClientSecret}
            setClientSecret={setShopifyClientSecret}
            secretSaved={shopifySecretSaved}
          />
        ) : isHubSpot ? (
          <HubSpotSection
            token={hubspotToken}
            setToken={setHubspotToken}
            secretSaved={hubspotSecretSaved}
            patWarn={hubspotPatWarn}
          />
        ) : isDynamics ? (
          <DynamicsSection
            url={host}
            setUrl={setHost}
            urlOk={dynamicsUrlOk}
            mode={dynamicsMode}
            setMode={setDynamicsMode}
            tenantId={dynamicsTenantId}
            setTenantId={setDynamicsTenantId}
            clientId={dynamicsClientId}
            setClientId={setDynamicsClientId}
            clientSecret={dynamicsClientSecret}
            setClientSecret={setDynamicsClientSecret}
            secretSaved={dynamicsSecretSaved}
            authorize={(pid) =>
              ipc.dynamicsAuthorize(pid, normalizeDynamicsUrl(host))
            }
            profileId={id}
            authed={dynamicsAuthed}
            account={dynamicsAccount}
            onAuthorized={(label) => {
              setDynamicsAuthed(true);
              setDynamicsAccount(label);
              if (!name) setName(label || normalizeDynamicsUrl(host));
            }}
            onReset={() => {
              setDynamicsAuthed(false);
              setDynamicsAccount("");
            }}
          />
        ) : isCloudOAuth ? (
          <OAuthConnectSection
            label={PROTOCOL_LABEL[protocol]}
            authorize={
              isDropbox
                ? ipc.dropboxAuthorize
                : isOnedrive
                  ? ipc.onedriveAuthorize
                  : isGdrive
                    ? ipc.gdriveAuthorize
                    : ipc.boxAuthorize
            }
            profileId={id}
            authed={cloudAuthed}
            account={cloudAccount}
            onAuthorized={(label) => {
              setCloudAuthed(true);
              setCloudAccount(label);
              if (!name) setName(label || PROTOCOL_LABEL[protocol]);
            }}
            onReset={() => {
              setCloudAuthed(false);
              setCloudAccount("");
            }}
          />
        ) : isAgent ? (
          <AgentSection
            profileId={id}
            host={host}
            setHost={setHost}
            port={port}
            setPort={setPort}
            setPortTouched={setPortTouched}
            paired={!!agentKey}
            onPair={pair}
            onUnpair={() => setAgentKey(undefined)}
          />
        ) : (
          <>
            <div className="flex gap-2">
              <Field label="Host" className="flex-1">
                <input
                  value={host}
                  onChange={(e) => setHost(e.target.value)}
                  placeholder=""
                  className={inputCls}
                />
              </Field>
              <Field label="Port" className="w-24">
                <input
                  type="number"
                  value={port}
                  onChange={(e) => {
                    setPort(
                      parseInt(e.target.value) || PROTOCOL_DEFAULT_PORT[protocol]
                    );
                    setPortTouched(true);
                  }}
                  className={inputCls}
                />
              </Field>
            </div>

            <Field label="Username">
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder={isFtp ? "anonymous" : "root"}
                className={inputCls}
              />
            </Field>

            {profile?.jumpHost && (
              <Hint>
                via bastion{" "}
                <span className="font-mono text-text-dim">
                  {profile.jumpUsername ? `${profile.jumpUsername}@` : ""}
                  {profile.jumpHost}:{profile.jumpPort ?? 22}
                </span>
              </Hint>
            )}

            {profile?.auth.kind === "keyref" ? (
              <Hint>
                Managed key (access grant) — stored in your OS keychain.
              </Hint>
            ) : (
              <>
            <Field label="Auth">
              <select
                value={authKind}
                onChange={(e) => setAuthKind(e.target.value as AuthMethod["kind"])}
                className={inputCls}
              >
                <option value="password">Password</option>
                <option value="key" disabled={isFtp}>
                  Private key file{isFtp ? " — SFTP only" : ""}
                </option>
                <option value="agent" disabled={isFtp}>
                  SSH agent{isFtp ? " — SFTP only" : ""}
                </option>
              </select>
            </Field>

            {authKind === "agent" && (
              <Hint>
                Uses the running SSH agent for authentication.{" "}
                <span className="font-mono text-text-dim">
                  {/* eslint-disable-next-line no-undef */}
                  {navigator.platform.startsWith("Win")
                    ? "\\\\.\\pipe\\openssh-ssh-agent (OpenSSH Authentication Agent service)"
                    : "$SSH_AUTH_SOCK"}
                </span>
                . Run <span className="font-mono">ssh-add</span> first to load
                your keys.
              </Hint>
            )}

            {authKind === "password" && (
              <Field label="Password">
                <PasswordInput value={password} onChange={setPassword} />
              </Field>
            )}

            {authKind === "key" && (
              <KeyAuthSection
                keyPath={keyPath}
                setKeyPath={setKeyPath}
                passphrase={passphrase}
                setPassphrase={setPassphrase}
                username={username}
                host={host}
              />
            )}
              </>
            )}
          </>
        )}

        <Field label={isObject ? "Default key prefix" : "Default remote path"}>
          <input
            value={defaultRemotePath}
            onChange={(e) => setDefaultRemotePath(e.target.value)}
            placeholder={isObject ? "" : isFtp ? "/" : "."}
            className={inputCls}
          />
        </Field>

        <label className="mb-3 flex cursor-pointer items-center gap-2.5 rounded-md border border-border bg-bg-subtle px-2.5 py-2">
          <input
            type="checkbox"
            checked={autoConnect}
            onChange={(e) => setAutoConnect(e.target.checked)}
            className="h-3.5 w-3.5 shrink-0 accent-[rgb(var(--accent))]"
          />
          <span className="min-w-0">
            <span className="block text-xs font-medium text-text">
              Auto-connect on startup
            </span>
            <span className="block text-[11px] text-text-dim">
              Open this server automatically when Ghost FTP launches.
            </span>
          </span>
        </label>

        {isFtp && (
          <Hint tone="warn">
            <ShieldOff size={11} className="mr-1 inline-block align-text-bottom" />
            FTP has no integrated terminal and{" "}
            {protocol === "ftp"
              ? "transmits credentials in clear text"
              : "uses TLS for the control channel only"}
            . Prefer SFTP when the server supports it.
          </Hint>
        )}
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
            {!canSave && (
              <span className="mr-auto text-[11px] text-text-dim">
                Needs {missing.join(", ")}
              </span>
            )}
            <button
              className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover"
              onClick={onClose}
            >
              Cancel
            </button>
            <button
              className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:opacity-40 disabled:cursor-not-allowed"
              onClick={save}
              disabled={!canSave}
              title={canSave ? undefined : `Fill in: ${missing.join(", ")}`}
            >
              Save
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/// SFTP private-key auth: the key path + passphrase, plus the "no more PuTTYgen"
/// helpers — generate a fresh keypair in place, or copy the public half of an
/// existing key to install on the server. Generating writes the private key (and
/// a `.pub` beside it), points this connection at it, and surfaces the public-key
/// line ready to paste into the server's `~/.ssh/authorized_keys`.
function KeyAuthSection({
  keyPath,
  setKeyPath,
  passphrase,
  setPassphrase,
  username,
  host,
}: {
  keyPath: string;
  setKeyPath: (v: string) => void;
  passphrase: string;
  setPassphrase: (v: string) => void;
  username: string;
  host: string;
}) {
  const [showGen, setShowGen] = useState(false);
  // The last generated/derived public key, shown in a copyable box. `justSaved`
  // distinguishes a key we just wrote (show the saved path) from one derived
  // from an existing private key (copy-only).
  const [pubKey, setPubKey] = useState<GeneratedKey | null>(null);
  const [justSaved, setJustSaved] = useState(false);
  const [copied, setCopied] = useState(false);
  const [derivingBusy, setDerivingBusy] = useState(false);

  // Generator form state.
  const [genType, setGenType] = useState<SshKeyType>("ed25519");
  const [genPass, setGenPass] = useState("");
  const [genPath, setGenPath] = useState("");
  const [genBusy, setGenBusy] = useState(false);
  const [overwrite, setOverwrite] = useState(false);

  // Public-key comment: mirrors ssh-keygen's `user@host` default.
  const comment = `${username || "ghostftp"}@${host || "ghostftp"}`;

  const toggleGenerator = useCallback(async () => {
    setShowGen((s) => !s);
    if (!genPath) {
      try {
        setGenPath((await ipc.sshKeyDefaults()).suggestedPath);
      } catch (error) {
        toast.warning("Couldn't suggest a default key path", String(error));
      }
    }
  }, [genPath]);

  const browsePrivateKey = async () => {
    const picked = await open({
      title: "Select a private key file",
      multiple: false,
      directory: false,
    });
    if (typeof picked === "string") setKeyPath(picked);
  };

  const browseSaveLocation = async () => {
    const picked = await save({
      title: "Save new private key as",
      defaultPath: genPath || undefined,
    });
    if (picked) setGenPath(picked);
  };

  const copyLine = async (line: string) => {
    try {
      await navigator.clipboard.writeText(line);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
      return true;
    } catch (error) {
      toast.warning(
        "Couldn't copy public key",
        `The key is still available in the field. ${String(error)}`
      );
      return false;
    }
  };

  const doGenerate = async () => {
    if (!genPath.trim()) return;
    setGenBusy(true);
    try {
      const res = await ipc.generateSshKey({
        keyType: genType,
        bits: genType === "rsa" ? 4096 : undefined,
        passphrase: genPass || undefined,
        path: genPath.trim(),
        comment,
        overwrite,
      });
      // Point the connection at the fresh key and carry its passphrase across.
      setKeyPath(res.path);
      setPassphrase(genPass);
      setPubKey(res);
      setJustSaved(true);
      setShowGen(false);
      setOverwrite(false);
      const copiedOk = await copyLine(res.publicKey);
      toast.success(
        copiedOk ? "Key created & public key copied" : "Key created",
        `Saved to ${res.path}`
      );
    } catch (e) {
      const msg = String(e);
      if (/already exists/i.test(msg) && !overwrite) {
        // Offer the overwrite escape hatch instead of a dead end.
        setOverwrite(true);
        toast.error(
          "A key already exists there",
          "Enable overwrite to replace it, or change the name."
        );
      } else {
        toast.error("Key generation failed", msg);
      }
    } finally {
      setGenBusy(false);
    }
  };

  const derivePublic = async () => {
    if (!keyPath.trim()) return;
    setDerivingBusy(true);
    try {
      const res = await ipc.sshPublicKeyFor(keyPath.trim(), passphrase || undefined);
      setPubKey(res);
      setJustSaved(false);
      const copiedOk = await copyLine(res.publicKey);
      if (copiedOk)
        toast.success("Public key copied", "Add it to the server's authorized_keys.");
    } catch (e) {
      const msg = String(e);
      toast.error(
        "Couldn't read the public key",
        /passphrase|decrypt|password|cipher/i.test(msg)
          ? "If the key is encrypted, enter its passphrase above first."
          : msg
      );
    } finally {
      setDerivingBusy(false);
    }
  };

  return (
    <>
      <Field label="Key path">
        <div className="flex gap-1.5">
          <input
            value={keyPath}
            onChange={(e) => setKeyPath(e.target.value)}
            placeholder="~/.ssh/id_ed25519"
            className={cn(inputCls, "flex-1")}
          />
          <button
            type="button"
            onClick={browsePrivateKey}
            title="Browse for an existing private key"
            aria-label="Browse for an existing private key"
            className="flex shrink-0 items-center rounded-md border border-border bg-bg-subtle px-2.5 text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <FolderOpen size={14} />
          </button>
        </div>
      </Field>

      <Field label="Passphrase (optional)">
        <input
          type="password"
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
          placeholder="Unlocks an encrypted key"
          autoComplete="off"
          className={inputCls}
        />
      </Field>

      {/* Make a new key, or copy the public half of an existing one. */}
      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={toggleGenerator}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-[11.5px] font-medium transition-colors",
            showGen
              ? "border-accent/40 bg-accent-soft text-accent"
              : "border-border bg-bg-subtle text-text-muted hover:bg-bg-hover hover:text-text"
          )}
        >
          <Sparkles size={12} />
          Generate new key…
        </button>
        {keyPath.trim() && (
          <button
            type="button"
            onClick={derivePublic}
            disabled={derivingBusy}
            className="inline-flex items-center gap-1.5 rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-[11.5px] text-text-muted transition-colors hover:bg-bg-hover hover:text-text disabled:opacity-50"
          >
            {derivingBusy ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Copy size={12} />
            )}
            Copy public key
          </button>
        )}
      </div>

      {/* Generator panel */}
      {showGen && (
        <div className="mb-3 rounded-md border border-border bg-bg-subtle p-2.5">
          <div className="mb-2 flex items-center gap-1.5 text-xs font-medium text-text">
            <KeyRound size={13} className="text-accent" />
            Generate a new SSH key
          </div>

          <Field label="Type">
            <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-panel p-1">
              {(
                [
                  ["ed25519", "Ed25519", "Modern · recommended"],
                  ["rsa", "RSA 4096", "Maximum compatibility"],
                ] as const
              ).map(([val, label, sub]) => (
                <button
                  key={val}
                  type="button"
                  onClick={() => setGenType(val)}
                  className={cn(
                    "flex flex-col items-start rounded-sm px-2 py-1.5 text-left transition-colors",
                    genType === val
                      ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                      : "text-text-muted hover:bg-bg-hover hover:text-text"
                  )}
                >
                  <span className="text-[11px] font-semibold">{label}</span>
                  <span className="text-[10px] text-text-dim">{sub}</span>
                </button>
              ))}
            </div>
          </Field>

          <Field label="Save to">
            <div className="flex gap-1.5">
              <input
                value={genPath}
                onChange={(e) => setGenPath(e.target.value)}
                placeholder="~/.ssh/ghostftp_ed25519"
                spellCheck={false}
                className={cn(inputCls, "flex-1 font-mono text-[11px]")}
              />
              <button
                type="button"
                onClick={browseSaveLocation}
                title="Choose a location"
                aria-label="Choose a save location"
                className="flex shrink-0 items-center rounded-md border border-border bg-bg-panel px-2.5 text-text-muted hover:bg-bg-hover hover:text-text"
              >
                <FolderOpen size={14} />
              </button>
            </div>
          </Field>

          <Field label="Passphrase (optional)">
            <input
              type="password"
              value={genPass}
              onChange={(e) => setGenPass(e.target.value)}
              placeholder="Encrypts the private key at rest"
              autoComplete="new-password"
              className={inputCls}
            />
          </Field>

          {overwrite && (
            <label className="mb-2 flex cursor-pointer items-center gap-2 text-[11px] text-danger">
              <input
                type="checkbox"
                checked={overwrite}
                onChange={(e) => setOverwrite(e.target.checked)}
                className="h-3.5 w-3.5 accent-[rgb(var(--accent))]"
              />
              Overwrite the existing key at this path
            </label>
          )}

          <button
            type="button"
            onClick={doGenerate}
            disabled={genBusy || !genPath.trim()}
            className="btn-accent flex w-full items-center justify-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {genBusy ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <Sparkles size={14} />
            )}
            {genBusy ? "Generating…" : "Generate key"}
          </button>
          <div className="mt-1.5 text-[11px] leading-relaxed text-text-dim">
            Ghost FTP writes the private key here (and a{" "}
            <span className="font-mono">.pub</span> beside it) and points this
            connection at it. Then add the public key to the server.
          </div>
        </div>
      )}

      {/* Public-key result — copyable, with the "put this on the server" nudge. */}
      {pubKey && (
        <div className="mb-3 rounded-md border border-success/30 bg-success/5 p-2.5">
          <div className="mb-1.5 flex items-center gap-1.5 text-[11.5px] font-medium text-text">
            <Check size={13} className="shrink-0 text-success" />
            <span>
              {justSaved ? "Key created" : "Public key"} — add it to the server's{" "}
              <span className="font-mono text-text-muted">
                ~/.ssh/authorized_keys
              </span>
            </span>
          </div>
          <textarea
            readOnly
            value={pubKey.publicKey}
            rows={3}
            spellCheck={false}
            onFocus={(e) => e.currentTarget.select()}
            className={cn(
              inputCls,
              "resize-none break-all font-mono text-[11px] leading-snug"
            )}
          />
          <div className="mt-1.5 flex items-center justify-between gap-2">
            <span
              className="min-w-0 truncate font-mono text-[10px] text-text-dim"
              title={pubKey.fingerprint}
            >
              {pubKey.fingerprint}
            </span>
            <button
              type="button"
              onClick={() => copyLine(pubKey.publicKey)}
              className="inline-flex shrink-0 items-center gap-1 rounded-md border border-border bg-bg-subtle px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
            >
              {copied ? (
                <Check size={11} className="text-accent" />
              ) : (
                <Copy size={11} />
              )}
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          {justSaved && (
            <div className="mt-1.5 text-[10.5px] text-text-dim">
              Saved to <span className="font-mono">{pubKey.path}</span>
            </div>
          )}
        </div>
      )}
    </>
  );
}

/// Ghost FTP Agent connection editor: host/port, LAN discovery, and the pairing
/// ceremony. Controls a whole remote machine (no login/auth) — it's paired once
/// with a 6-digit code the daemon prints, then keyed by the pinned daemon key.
function AgentSection({
  profileId,
  host,
  setHost,
  port,
  setPort,
  setPortTouched,
  paired,
  onPair,
  onUnpair,
}: {
  profileId: string;
  host: string;
  setHost: (v: string) => void;
  port: number;
  setPort: (v: number) => void;
  setPortTouched: (v: boolean) => void;
  paired: boolean;
  onPair: (code: string) => Promise<void>;
  onUnpair: () => void;
}) {
  const [code, setCode] = useState("");
  const [pairing, setPairing] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [found, setFound] = useState<DiscoveredAgent[] | null>(null);
  const codeRef = useRef<HTMLInputElement>(null);

  const scan = useCallback(async () => {
    setScanning(true);
    try {
      setFound(await ipc.discoverAgents());
    } catch (e) {
      toast.error("Scan failed", String(e));
    } finally {
      setScanning(false);
    }
  }, []);

  // Scan the moment the Ghost FTP Agent form opens — finding machines shouldn't
  // require knowing there's a button for it.
  useEffect(() => {
    void scan();
  }, [scan]);

  const doPair = async () => {
    setPairing(true);
    try {
      await onPair(code.trim());
      setCode("");
    } catch (e) {
      toast.error("Pairing failed", String(e));
    } finally {
      setPairing(false);
    }
  };

  const codeValid = /^\d{6}$/.test(code.trim());
  // The machine the host/port fields currently point at, if it was discovered.
  const selectedMachine = found?.find((d) => d.host === host && d.port === port);

  return (
    <>
      <div className="flex gap-2">
        <Field label="Host / IP" className="flex-1">
          <input
            value={host}
            onChange={(e) => setHost(e.target.value)}
            placeholder="192.168.1.42"
            className={inputCls}
          />
        </Field>
        <Field label="Port" className="w-24">
          <input
            type="number"
            value={port}
            onChange={(e) => {
              setPort(parseInt(e.target.value) || PROTOCOL_DEFAULT_PORT["ghostftp-agent"]);
              setPortTouched(true);
            }}
            className={inputCls}
          />
        </Field>
      </div>

      {/* LAN discovery — runs automatically on open; the button rescans. */}
      <div className="mb-3">
        <button
          type="button"
          onClick={scan}
          disabled={scanning}
          className="flex items-center gap-1.5 rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-50"
        >
          {scanning ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <Radar size={12} />
          )}
          {scanning ? "Scanning…" : found ? "Rescan network" : "Scan local network"}
        </button>
        {found && found.length === 0 && !scanning && (
          <div className="mt-1.5 text-[11px] text-text-dim">
            No devices found. Enable Ghost FTP Agent on your phone or TV, or Remote
            control in the desktop Ghost FTP app. Keep both devices on the same
            network, or enter the device's IP above.
          </div>
        )}
        {found && found.length > 0 && (
          <div className="mt-1.5 flex flex-col gap-1">
            {found.map((d) => {
              const selected = d.host === host && d.port === port;
              const isThisProfile = d.pairedProfileId === profileId;
              return (
                <button
                  key={d.fingerprint + d.host}
                  type="button"
                  onClick={() => {
                    setHost(d.host);
                    setPort(d.port);
                    setPortTouched(true);
                    // Lead the user to the next step instead of silently
                    // filling a field they may not be looking at.
                    if (!paired) codeRef.current?.focus();
                  }}
                  className={cn(
                    "flex items-center justify-between rounded-md border px-2.5 py-1.5 text-left text-xs transition-colors",
                    selected
                      ? "border-accent bg-accent/10"
                      : "border-border bg-bg-subtle hover:bg-bg-hover"
                  )}
                >
                  <span className="min-w-0">
                    <span className="flex min-w-0 items-center gap-1.5">
                      <span className="truncate font-medium text-text">
                        {d.hostname || d.host}
                      </span>
                      {selected && <Check size={11} className="shrink-0 text-accent" />}
                    </span>
                    <span className="block truncate text-[10px] text-text-dim">
                      {d.os} · {d.host}:{d.port} · {d.fingerprint}
                    </span>
                  </span>
                  {d.pairedProfileId ? (
                    <span className="ml-2 shrink-0 rounded-full border border-success/40 bg-success/10 px-1.5 py-0.5 text-[9px] font-medium text-success">
                      {isThisProfile ? "this connection" : "paired"}
                    </span>
                  ) : d.pairable ? (
                    <span className="ml-2 shrink-0 rounded-full border border-accent/40 bg-accent/10 px-1.5 py-0.5 text-[9px] font-medium text-accent">
                      ready to pair
                    </span>
                  ) : (
                    <MonitorSmartphone size={13} className="ml-2 shrink-0 text-text-dim" />
                  )}
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Pairing */}
      {paired ? (
        <div className="mb-3 flex items-center justify-between rounded-md border border-success/30 bg-success/10 px-2.5 py-2">
          <span className="flex items-center gap-1.5 text-xs text-text">
            <Check size={13} className="text-success" />
            Paired — the daemon's key is pinned.
          </span>
          <button
            type="button"
            onClick={onUnpair}
            className="text-[11px] text-text-dim underline hover:text-text"
          >
            Re-pair
          </button>
        </div>
      ) : (
        <div className="mb-3 rounded-md border border-border bg-bg-subtle p-2.5">
          <div className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-text">
            <Link2 size={13} className="text-accent" />
            {selectedMachine
              ? `Pair with ${selectedMachine.hostname || host}`
              : "Pair with this machine"}
          </div>
          <div className="mb-2 text-[11px] leading-relaxed text-text-dim">
            {selectedMachine?.os === "android" ? (
              <>
                On your phone or TV, open Ghost FTP Agent, grant storage access,
                enable the agent, and select Show pairing code. Enter that
                6-digit code here.
              </>
            ) : selectedMachine && selectedMachine.pairable === false ? (
              <>
                <span className="font-medium text-text-muted">
                  {selectedMachine.hostname || host}
                </span>{" "}
                isn't accepting pairing right now. On that machine run{" "}
                <span className="font-mono text-text-muted">ghostftp-agentd pair</span> (or
                open its Ghost FTP app → Settings → Remote control → Show pairing code),
                then type the code here.
              </>
            ) : selectedMachine ? (
              <>
                Type the 6-digit code showing on{" "}
                <span className="font-medium text-text-muted">
                  {selectedMachine.hostname || host}
                </span>
                . No code there? Run{" "}
                <span className="font-mono text-text-muted">ghostftp-agentd pair</span> on
                it, or open its Ghost FTP app → Settings → Remote control → Show pairing
                code.
              </>
            ) : (
              <>
                On the machine you want to control, run{" "}
                <span className="font-mono text-text-muted">ghostftp-agentd pair</span> (or
                open its Ghost FTP app → Settings → Remote control → Show pairing code) and
                type the 6-digit code below. Pairing sets up an end-to-end encrypted
                link and pins the machine's key — you won't need the code again.
              </>
            )}
          </div>
          <div className="flex gap-2">
            <input
              ref={codeRef}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              onKeyDown={(e) => {
                if (e.key === "Enter" && codeValid && host && !pairing) void doPair();
              }}
              placeholder="000000"
              inputMode="numeric"
              className={cn(inputCls, "flex-1 text-center font-mono tracking-[0.3em]")}
            />
            <button
              type="button"
              onClick={doPair}
              disabled={!codeValid || !host || pairing}
              className="btn-accent flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
            >
              {pairing && <Loader2 size={13} className="animate-spin" />}
              {pairing ? "Pairing…" : "Pair"}
            </button>
          </div>
        </div>
      )}
    </>
  );
}

function AzureSection({
  account,
  setAccount,
  container,
  setContainer,
  accessKey,
  setAccessKey,
  endpoint,
  setEndpoint,
}: {
  account: string;
  setAccount: (v: string) => void;
  container: string;
  setContainer: (v: string) => void;
  accessKey: string;
  setAccessKey: (v: string) => void;
  endpoint: string;
  setEndpoint: (v: string) => void;
}) {
  return (
    <>
      <Hint>
        Azure Blob Storage. Use the storage account name and an account key
        (Portal → Storage account → Access keys). Custom endpoints target
        Azurite (local emulator) or sovereign clouds.
      </Hint>

      <div className="flex gap-2">
        <Field label="Storage account" className="flex-1">
          <input
            value={account}
            onChange={(e) => setAccount(e.target.value)}
            placeholder="mystorageacct"
            className={inputCls}
          />
        </Field>
        <Field label="Container" className="flex-1">
          <input
            value={container}
            onChange={(e) => setContainer(e.target.value)}
            placeholder="my-container"
            className={inputCls}
          />
        </Field>
      </div>

      <Field label="Access key">
        <input
          type="password"
          value={accessKey}
          onChange={(e) => setAccessKey(e.target.value)}
          className={inputCls}
        />
      </Field>

      <Field label="Endpoint (optional)">
        <input
          value={endpoint}
          onChange={(e) => setEndpoint(e.target.value)}
          placeholder="(leave blank for public Azure)"
          className={inputCls}
        />
      </Field>
    </>
  );
}

/// OAuth cloud connect (Dropbox / OneDrive / …). Like agent pairing: authorizing
/// runs a browser flow and stores tokens in the OS keychain keyed by the profile
/// id, then the editor persists the profile. No password lives in Ghost FTP.
function OAuthConnectSection({
  label,
  authorize: runAuthorize,
  profileId,
  authed,
  account,
  onAuthorized,
  onReset,
}: {
  label: string;
  authorize: (profileId: string) => Promise<{ accountLabel: string }>;
  profileId: string;
  authed: boolean;
  account: string;
  onAuthorized: (label: string) => void;
  onReset: () => void;
}) {
  const [busy, setBusy] = useState(false);

  const authorize = async () => {
    setBusy(true);
    try {
      const res = await runAuthorize(profileId);
      onAuthorized(res.accountLabel);
      toast.success(
        `Connected to ${label}`,
        res.accountLabel ? `Authorized as ${res.accountLabel}` : undefined
      );
    } catch (e) {
      toast.error(`${label} authorization failed`, String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Hint>
        Connect a {label} account. Authorizing opens your browser once; Ghost FTP
        stores the refresh token in your OS keychain and never sees your {label}{" "}
        password.
      </Hint>

      {authed ? (
        <div className="mb-3 flex items-center justify-between rounded-md border border-success/30 bg-success/10 px-2.5 py-2">
          <span className="flex items-center gap-1.5 text-xs text-text">
            <Check size={13} className="text-success" />
            Connected{account ? ` as ${account}` : ""}.
          </span>
          <button
            type="button"
            onClick={onReset}
            className="text-[11px] text-text-dim underline hover:text-text"
          >
            Reconnect
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={authorize}
          disabled={busy}
          className="btn-accent mb-3 flex w-full items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50"
        >
          {busy ? <Loader2 size={14} className="animate-spin" /> : <Box size={14} />}
          {busy ? "Waiting for authorization…" : `Connect with ${label}`}
        </button>
      )}
    </>
  );
}

function HttpSection({
  url,
  setUrl,
  username,
  setUsername,
  password,
  setPassword,
}: {
  url: string;
  setUrl: (v: string) => void;
  username: string;
  setUsername: (v: string) => void;
  password: string;
  setPassword: (v: string) => void;
}) {
  const [auth, setAuth] = useState<boolean>(!!username);
  return (
    <>
      <Hint>
        Read-only browse of any static file server. Point at a directory with an
        autoindex (nginx / Apache) to browse it, or paste a direct file URL to
        pull a single artifact. No uploads, renames, or deletes.
      </Hint>

      <Field label="URL">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder=""
          className={inputCls}
        />
      </Field>

      <label className="mb-3 flex cursor-pointer items-center gap-2.5 rounded-md border border-border bg-bg-subtle px-2.5 py-2">
        <input
          type="checkbox"
          checked={auth}
          onChange={(e) => {
            setAuth(e.target.checked);
            if (!e.target.checked) {
              setUsername("");
              setPassword("");
            }
          }}
          className="h-3.5 w-3.5 shrink-0 accent-[rgb(var(--accent))]"
        />
        <span className="text-xs font-medium text-text">
          Server needs HTTP Basic auth
        </span>
      </label>

      {auth && (
        <>
          <Field label="Username">
            <input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={inputCls}
            />
          </Field>
          <Field label="Password">
            <PasswordInput value={password} onChange={setPassword} />
          </Field>
        </>
      )}
    </>
  );
}

function WebdavSection({
  url,
  setUrl,
  username,
  setUsername,
  password,
  setPassword,
}: {
  url: string;
  setUrl: (v: string) => void;
  username: string;
  setUsername: (v: string) => void;
  password: string;
  setPassword: (v: string) => void;
}) {
  const [provider, setProvider] = useState<WebdavProvider>("nextcloud");
  // Bearer mode = no username (the value in `password` is the token).
  const [mode, setMode] = useState<"basic" | "bearer">(
    !username && password ? "bearer" : "basic"
  );
  const preset = WEBDAV_PROVIDER_PRESETS[provider];

  const applyPreset = (p: WebdavProvider) => {
    setProvider(p);
    const tpl = WEBDAV_PROVIDER_PRESETS[p].urlHint;
    // Prefill the URL template, substituting a known username where it appears.
    setUrl(username ? tpl.replace("<user>", username) : tpl);
  };

  return (
    <>
      <Field label="Provider">
        <div className="grid grid-cols-4 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(Object.keys(WEBDAV_PROVIDER_PRESETS) as WebdavProvider[]).map((p) => {
            const data = WEBDAV_PROVIDER_PRESETS[p];
            return (
              <button
                key={p}
                type="button"
                onClick={() => applyPreset(p)}
                className={
                  "flex flex-col items-start rounded-sm px-2 py-1.5 text-left transition-colors " +
                  (provider === p
                    ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                    : "text-text-muted hover:bg-bg-hover hover:text-text")
                }
              >
                <span className="flex items-center gap-1 text-[11px] font-semibold">
                  <span
                    className={cn(
                      "flex h-3.5 w-3.5 shrink-0 items-center justify-center",
                      provider === p ? "text-accent" : "text-text-dim"
                    )}
                  >
                    {WEBDAV_PROVIDER_ICON[p] ? (
                      <BrandIcon icon={WEBDAV_PROVIDER_ICON[p]!} size={13} />
                    ) : (
                      <Globe size={11} />
                    )}
                  </span>
                  {data.label}
                </span>
                <span className="text-[10px] text-text-dim">{data.vendor}</span>
              </button>
            );
          })}
        </div>
      </Field>

      <Hint>{preset.description}</Hint>

      <Field label="Server URL">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={preset.urlHint}
          className={inputCls}
        />
      </Field>

      <Field label="Auth">
        <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(["basic", "bearer"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => {
                setMode(m);
                if (m === "bearer") setUsername("");
              }}
              className={
                "rounded-sm px-2 py-1.5 text-[11px] font-semibold transition-colors " +
                (mode === m
                  ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                  : "text-text-muted hover:bg-bg-hover hover:text-text")
              }
            >
              {m === "basic" ? "Username + password" : "Bearer token"}
            </button>
          ))}
        </div>
      </Field>

      {mode === "basic" && (
        <Field label="Username">
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="alice"
            className={inputCls}
          />
        </Field>
      )}

      <Field label={mode === "basic" ? "Password" : "Bearer token"}>
        <PasswordInput value={password} onChange={setPassword} />
      </Field>
    </>
  );
}

function ShopifySection({
  domain,
  setDomain,
  mode,
  setMode,
  token,
  setToken,
  clientId,
  setClientId,
  clientSecret,
  setClientSecret,
  secretSaved,
}: {
  domain: string;
  setDomain: (v: string) => void;
  mode: "token" | "client";
  setMode: (m: "token" | "client") => void;
  token: string;
  setToken: (v: string) => void;
  clientId: string;
  setClientId: (v: string) => void;
  clientSecret: string;
  setClientSecret: (v: string) => void;
  secretSaved: boolean;
}) {
  return (
    <>
      <Field label="Store domain">
        <input
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          placeholder="my-store.myshopify.com"
          className={inputCls}
        />
      </Field>

      <Hint>
        Browse and edit theme files like an FTP site. Prefer an
        unpublished/duplicate theme — changes deploy instantly.
      </Hint>

      <Field label="Auth">
        <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(["token", "client"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setMode(m)}
              className={
                "rounded-sm px-2 py-1.5 text-[11px] font-semibold transition-colors " +
                (mode === m
                  ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                  : "text-text-muted hover:bg-bg-hover hover:text-text")
              }
            >
              {m === "token" ? "Access token" : "Client ID + Secret"}
            </button>
          ))}
        </div>
      </Field>

      {mode === "token" ? (
        <Field label="Admin API access token">
          <PasswordInput value={token} onChange={setToken} />
        </Field>
      ) : (
        <>
          <Field label="Client ID">
            <PasswordInput value={clientId} onChange={setClientId} />
          </Field>
          <Field label="Client secret">
            <PasswordInput value={clientSecret} onChange={setClientSecret} />
          </Field>
        </>
      )}

      <Hint>
        Stored in your OS keychain — never in the connections file.
        {secretSaved
          ? " A credential is already saved; leave the fields blank to keep it."
          : ""}
      </Hint>
    </>
  );
}

function HubSpotSection({
  token,
  setToken,
  secretSaved,
  patWarn,
}: {
  token: string;
  setToken: (v: string) => void;
  secretSaved: boolean;
  patWarn: boolean;
}) {
  return (
    <>
      <Field label="Private app access token">
        <PasswordInput value={token} onChange={setToken} />
      </Field>

      <Hint>
        No URL needed — the token identifies your portal; the API base
        (api.hubapi.com) is fixed. Create a private app in your HubSpot portal
        (Settings → Integrations → Private Apps). Each scope unlocks a root:{" "}
        <code>content</code> for the design roots, <code>files</code> for the
        File Manager, <code>hubdb</code> for HubDB tables — any one is enough
        to connect; roots without their scope stay hidden.
      </Hint>

      <Hint tone="warn">
        Prefer the draft environment — published writes deploy instantly.
      </Hint>

      {patWarn && (
        <Hint tone="warn">
          Private-app tokens start with <code>pat-</code> — double-check that
          you pasted the right credential.
        </Hint>
      )}

      <Hint>
        Stored in your OS keychain — never in the connections file.
        {secretSaved
          ? " A token is already saved; leave the field blank to keep it."
          : ""}
      </Hint>
    </>
  );
}

function DynamicsSection({
  url,
  setUrl,
  urlOk,
  mode,
  setMode,
  tenantId,
  setTenantId,
  clientId,
  setClientId,
  clientSecret,
  setClientSecret,
  secretSaved,
  authorize,
  profileId,
  authed,
  account,
  onAuthorized,
  onReset,
}: {
  url: string;
  setUrl: (v: string) => void;
  urlOk: boolean;
  mode: "client" | "delegated";
  setMode: (m: "client" | "delegated") => void;
  tenantId: string;
  setTenantId: (v: string) => void;
  clientId: string;
  setClientId: (v: string) => void;
  clientSecret: string;
  setClientSecret: (v: string) => void;
  secretSaved: boolean;
  authorize: (profileId: string) => Promise<{ accountLabel: string }>;
  profileId: string;
  authed: boolean;
  account: string;
  onAuthorized: (label: string) => void;
  onReset: () => void;
}) {
  return (
    <>
      <Field label="Environment URL">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="my-org.crm.dynamics.com"
          className={inputCls}
        />
      </Field>

      <Hint tone="warn">
        Edits publish immediately — prefer a dev/sandbox environment.
      </Hint>

      <Field label="Auth">
        <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(["client", "delegated"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setMode(m)}
              className={
                "rounded-sm px-2 py-1.5 text-[11px] font-semibold transition-colors " +
                (mode === m
                  ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                  : "text-text-muted hover:bg-bg-hover hover:text-text")
              }
            >
              {m === "client" ? "Client credentials" : "Sign in with Microsoft"}
            </button>
          ))}
        </div>
      </Field>

      {mode === "client" ? (
        <>
          <Hint>
            Register an app in Microsoft Entra ID and add it as an application
            user on the environment (Power Platform admin center → Environments
            → Settings → Users + permissions → Application users).
          </Hint>

          <Field label="Tenant ID">
            <PasswordInput value={tenantId} onChange={setTenantId} />
          </Field>
          <Field label="Client ID">
            <PasswordInput value={clientId} onChange={setClientId} />
          </Field>
          <Field label="Client secret">
            <PasswordInput value={clientSecret} onChange={setClientSecret} />
          </Field>

          <Hint>
            Stored in your OS keychain — never in the connections file.
            {secretSaved
              ? " A credential is already saved; leave the fields blank to keep it."
              : ""}
          </Hint>
        </>
      ) : urlOk ? (
        <OAuthConnectSection
          label="Microsoft"
          authorize={authorize}
          profileId={profileId}
          authed={authed}
          account={account}
          onAuthorized={onAuthorized}
          onReset={onReset}
        />
      ) : (
        <Hint tone="warn">
          Enter a valid environment URL above before signing in — the delegated
          flow is scoped to that organization.
        </Hint>
      )}
    </>
  );
}

function GcsSection({
  bucket,
  setBucket,
  keyMode,
  setKeyMode,
  keyPath,
  setKeyPath,
  keyJson,
  setKeyJson,
}: {
  bucket: string;
  setBucket: (v: string) => void;
  keyMode: "file" | "paste";
  setKeyMode: (v: "file" | "paste") => void;
  keyPath: string;
  setKeyPath: (v: string) => void;
  keyJson: string;
  setKeyJson: (v: string) => void;
}) {
  return (
    <>
      <Hint>
        Google Cloud Storage. Create a service account with Storage access, then
        use its JSON key (Cloud console → IAM &amp; Admin → Service accounts →
        Keys). Point at the downloaded file, or paste the key directly.
      </Hint>

      <Field label="Bucket">
        <input
          value={bucket}
          onChange={(e) => setBucket(e.target.value)}
          placeholder="my-gcs-bucket"
          className={inputCls}
        />
      </Field>

      <Field label="Service account key">
        <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(["file", "paste"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setKeyMode(m)}
              className={
                "rounded-sm px-2 py-1.5 text-[11px] font-semibold transition-colors " +
                (keyMode === m
                  ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                  : "text-text-muted hover:bg-bg-hover hover:text-text")
              }
            >
              {m === "file" ? "Key file path" : "Paste JSON"}
            </button>
          ))}
        </div>
      </Field>

      {keyMode === "file" ? (
        <Field label="Key file path">
          <input
            value={keyPath}
            onChange={(e) => setKeyPath(e.target.value)}
            placeholder="~/keys/service-account.json"
            className={inputCls}
          />
        </Field>
      ) : (
        <Field label="Service account JSON">
          <textarea
            value={keyJson}
            onChange={(e) => setKeyJson(e.target.value)}
            placeholder='{ "type": "service_account", ... }'
            spellCheck={false}
            rows={5}
            className={cn(inputCls, "resize-y font-mono text-[11px] leading-snug")}
          />
        </Field>
      )}
    </>
  );
}

function S3Section({
  provider,
  onProviderChange,
  bucket,
  setBucket,
  region,
  setRegion,
  endpoint,
  setEndpoint,
  accessKey,
  setAccessKey,
  secretKey,
  setSecretKey,
}: {
  provider: S3Provider;
  onProviderChange: (p: S3Provider) => void;
  bucket: string;
  setBucket: (v: string) => void;
  region: string;
  setRegion: (v: string) => void;
  endpoint: string;
  setEndpoint: (v: string) => void;
  accessKey: string;
  setAccessKey: (v: string) => void;
  secretKey: string;
  setSecretKey: (v: string) => void;
}) {
  const preset = S3_PROVIDER_PRESETS[provider];
  return (
    <>
      <Field label="Provider">
        <div className="grid grid-cols-3 gap-1 rounded-md border border-border bg-bg-subtle p-1">
          {(Object.keys(S3_PROVIDER_PRESETS) as S3Provider[]).map((p) => {
            const data = S3_PROVIDER_PRESETS[p];
            return (
              <button
                key={p}
                type="button"
                onClick={() => onProviderChange(p)}
                className={
                  "flex flex-col items-start rounded-sm px-2 py-1.5 text-left transition-colors " +
                  (provider === p
                    ? "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
                    : "text-text-muted hover:bg-bg-hover hover:text-text")
                }
              >
                <span className="flex items-center gap-1 text-[11px] font-semibold">
                  <span
                    className={cn(
                      "flex h-3.5 w-3.5 shrink-0 items-center justify-center",
                      provider === p ? "text-accent" : "text-text-dim"
                    )}
                  >
                    {S3_PROVIDER_ICON[p] ? (
                      <BrandIcon icon={S3_PROVIDER_ICON[p]!} size={13} />
                    ) : (
                      <Cloud size={11} />
                    )}
                  </span>
                  {data.label}
                </span>
                <span className="text-[10px] text-text-dim">{data.vendor}</span>
              </button>
            );
          })}
        </div>
      </Field>

      <Hint>{preset.description}</Hint>

      <Field label="Bucket">
        <input
          value={bucket}
          onChange={(e) => setBucket(e.target.value)}
          placeholder="my-bucket"
          className={inputCls}
        />
      </Field>

      <div className="flex gap-2">
        <Field label="Region" className="w-32">
          <input
            value={region}
            onChange={(e) => setRegion(e.target.value)}
            className={inputCls}
          />
        </Field>
        <Field label={provider === "aws" ? "Endpoint (optional)" : "Endpoint"} className="flex-1">
          <input
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
            placeholder={preset.endpointHint}
            className={inputCls}
          />
        </Field>
      </div>

      <Field label="Access key ID">
        <input
          value={accessKey}
          onChange={(e) => setAccessKey(e.target.value)}
          placeholder="AKIA…"
          className={inputCls}
        />
      </Field>
      <Field label="Secret access key">
        <input
          type="password"
          value={secretKey}
          onChange={(e) => setSecretKey(e.target.value)}
          className={inputCls}
        />
      </Field>
    </>
  );
}

function protocolHint(p: Protocol): string {
  switch (p) {
    case "sftp":
      return "SSH · :22";
    case "ftp":
      return "Plain · :21";
    case "ftps":
      return "TLS · :21";
    case "s3":
      return "Object · :443";
    case "azure":
      return "Blob · :443";
    case "gcs":
      return "Object · :443";
    case "webdav":
      return "HTTP · :443";
    case "http":
      return "Read-only · :443";
    case "dropbox":
      return "OAuth · Cloud";
    case "onedrive":
      return "OAuth · Cloud";
    case "gdrive":
      return "OAuth · Cloud";
    case "box":
      return "OAuth · Cloud";
    case "shopify":
      return "Theme files · :443";
    case "hubspot":
      return "Design files · :443";
    case "dynamics":
      return "Web resources · :443";
    case "ghostftp-agent":
      return "Machine · :8722";
  }
}

function ProtocolButton({
  active,
  onClick,
  protocol,
  label,
}: {
  active: boolean;
  onClick: () => void;
  protocol: Protocol;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "true" : undefined}
      className={
        "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[13px] transition-colors " +
        (active
          ? "bg-accent-soft font-medium text-accent"
          : "text-text-muted hover:bg-bg-hover hover:text-text")
      }
    >
      {/* Real brand/protocol logo. Colour marks (S3/Azure/Dropbox…) keep their
          own palette; monochrome glyphs (SSH/FTP/HTTP/lighthouse) follow the
          button's text colour via currentColor. Fixed 16px slot keeps labels
          aligned regardless of each mark's aspect ratio. */}
      <span
        className={cn(
          "flex h-4 w-4 shrink-0 items-center justify-center",
          active ? "text-accent" : "text-text-dim"
        )}
      >
        <BrandIcon icon={protocolIcon(protocol)} size={15} />
      </span>
      <span className="truncate">{label}</span>
    </button>
  );
}

function Hint({
  children,
  tone,
}: {
  children: React.ReactNode;
  tone?: "warn";
}) {
  return (
    <div
      className={
        "mb-3 rounded-md border px-2.5 py-2 text-[11.5px] leading-relaxed " +
        (tone === "warn"
          ? "border-danger/30 bg-danger-soft/60 text-text-muted"
          : "border-border bg-bg-subtle text-text-muted")
      }
    >
      {children}
    </div>
  );
}

const inputCls =
  "w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent";

function Field({
  label,
  className,
  children,
}: {
  label: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <label className={`mb-3 block ${className ?? ""}`}>
      <div className="mb-1 text-xs text-text-muted">{label}</div>
      {children}
    </label>
  );
}

/**
 * Password input with a reveal toggle and a "Generate strong password" action
 * that fills the field and copies the value to the clipboard — for the common
 * case of creating a brand-new server account while adding the connection.
 */
function PasswordInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const [reveal, setReveal] = useState(false);
  const [copied, setCopied] = useState(false);

  async function generate() {
    const pw = generatePassword();
    onChange(pw);
    setReveal(true); // so the user can see/verify what was generated
    try {
      await navigator.clipboard.writeText(pw);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (error) {
      toast.warning(
        "Password generated, but couldn't copy it",
        `The generated password remains in the field. ${String(error)}`
      );
    }
  }

  return (
    <div>
      <div className="relative">
        <input
          type={reveal ? "text" : "password"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(inputCls, "pr-9")}
          autoComplete="off"
          spellCheck={false}
        />
        <button
          type="button"
          onClick={() => setReveal((r) => !r)}
          title={reveal ? "Hide password" : "Show password"}
          aria-label={reveal ? "Hide password" : "Show password"}
          className="absolute inset-y-0 right-0 flex items-center px-2.5 text-text-dim hover:text-text"
        >
          {reveal ? <EyeOff size={13} /> : <Eye size={13} />}
        </button>
      </div>
      <button
        type="button"
        onClick={generate}
        className="mt-1.5 inline-flex items-center gap-1 rounded-md border border-border bg-bg-subtle px-2 py-1 text-[11.5px] text-text-muted transition-colors hover:bg-bg-hover hover:text-text"
      >
        {copied ? (
          <Check size={11} className="text-accent" />
        ) : (
          <Wand2 size={11} />
        )}
        {copied ? "Generated & copied" : "Generate strong password"}
      </button>
    </div>
  );
}
