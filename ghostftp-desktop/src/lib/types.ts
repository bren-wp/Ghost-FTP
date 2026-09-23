// Shared types mirroring the Rust serde structs in src-tauri/src/.
// Keep these in sync with src-tauri/src/remotefs/mod.rs and profiles/mod.rs.

export type AuthMethod =
  | { kind: "password"; password: string }
  | { kind: "key"; path: string; passphrase?: string }
  | { kind: "agent" }
  // Grant-imported profile: the private key lives in the OS keychain under
  // `keyRef`; it never crosses IPC. Not user-selectable in the editor.
  | { kind: "keyref"; keyRef: string };

// ---- In-app SSH key generation (mirrors src-tauri/src/keys.rs) ----

export type SshKeyType = "ed25519" | "rsa";

/** Request to generate a new SSH keypair on disk. */
export interface GenerateKeyRequest {
  keyType: SshKeyType;
  /** RSA modulus size; ignored for ed25519. Backend defaults to 4096. */
  bits?: number;
  /** Encrypt the private key with this passphrase. Empty/absent = unencrypted. */
  passphrase?: string;
  /** Where to write the private key (may start with `~`). `.pub` sits alongside. */
  path: string;
  /** Trailing comment on the public-key line (e.g. `user@host`). */
  comment?: string;
  /** Overwrite an existing key at `path` instead of erroring. Default false. */
  overwrite?: boolean;
}

/** A generated (or derived) key, as the editor needs it. */
export interface GeneratedKey {
  /** Absolute path the private key was written to (`.pub` sits beside it). */
  path: string;
  /** Full public-key line, ready for the server's `~/.ssh/authorized_keys`. */
  publicKey: string;
  /** OpenSSH-style `SHA256:…` fingerprint. */
  fingerprint: string;
  /** Wire key type: `ssh-ed25519` or `ssh-rsa`. */
  keyType: string;
}

/** Suggested defaults for the generator UI. */
export interface SshKeyDefaults {
  /** Absolute path of the user's `~/.ssh` directory. */
  dir: string;
  /** A ready-to-use, non-colliding private-key path under `dir`. */
  suggestedPath: string;
}

export type Protocol =
  | "sftp"
  | "ftp"
  | "ftps"
  | "s3"
  | "azure"
  | "gcs"
  | "webdav"
  | "http"
  | "dropbox"
  | "onedrive"
  | "gdrive"
  | "box"
  | "shopify"
  | "hubspot"
  | "dynamics"
  | "ghostftp-agent";

export interface ConnectionProfile {
  id: string;
  name: string;
  protocol: Protocol;
  host: string;
  port: number;
  username: string;
  auth: AuthMethod;
  defaultRemotePath?: string;
  /** Optional human-readable note shown in Site Manager. */
  description?: string;
  color?: string;
  /** Connect automatically on app launch (and on a single click in the rail). */
  autoConnect?: boolean;
  // Object-store fields (used when protocol === "s3" or "azure").
  bucket?: string; // bucket (S3) or container (Azure)
  region?: string;
  endpoint?: string;
  account?: string; // Azure storage account name
  // Ghost FTP Agent (protocol === "ghostftp-agent"): the paired daemon's pinned public
  // key (base64). Present === paired; absent === still needs a pairing code.
  agentKey?: string;
  /** Rail/Site Manager folder this server lives in. */
  group?: string;
  /** Persistent Site Manager metadata. */
  favorite?: boolean;
  bookmarked?: boolean;
  tags?: string[];
  /** Unix timestamp (seconds) of the last successful connection. */
  lastUsed?: number;
  /** Manual drag-and-drop position in the rail. Absent = sorted after ordered
   *  profiles, by protocol then name. */
  sortOrder?: number;
  /** Custom rail bubble glyph: an emoji/short string, or a bundled Iconify
   *  key ("prefix:name", see brandIconData). Absent = the name monogram. */
  icon?: string;
  // Bastion/ProxyJump hop (grant-imported profiles only for now — read-only in
  // the editor). Connect to jumpHost, then tunnel to the target.
  jumpHost?: string;
  jumpPort?: number;
  jumpUsername?: string;
}

export const PROTOCOL_DEFAULT_PORT: Record<Protocol, number> = {
  sftp: 22,
  ftp: 21,
  ftps: 21,
  s3: 443,
  azure: 443,
  gcs: 443,
  webdav: 443,
  http: 443,
  dropbox: 443,
  onedrive: 443,
  gdrive: 443,
  box: 443,
  shopify: 443,
  hubspot: 443,
  dynamics: 443,
  "ghostftp-agent": 8722,
};

export const PROTOCOL_LABEL: Record<Protocol, string> = {
  sftp: "SFTP",
  ftp: "FTP",
  ftps: "FTPS",
  s3: "S3",
  azure: "Azure",
  gcs: "GCS",
  webdav: "WebDAV",
  http: "HTTP",
  dropbox: "Dropbox",
  onedrive: "OneDrive",
  gdrive: "Google Drive",
  box: "Box",
  shopify: "Shopify",
  hubspot: "HubSpot",
  dynamics: "Dynamics 365",
  "ghostftp-agent": "Ghost FTP Agent",
};

export function isObjectProtocol(p: Protocol): boolean {
  return p === "s3" || p === "azure" || p === "gcs";
}

/** A Ghost FTP Agent connection controls a whole remote machine, not a login on a
 *  server — it has no username/auth, and is paired with a code instead. */
export function isAgentProtocol(p: Protocol): boolean {
  return p === "ghostftp-agent";
}

/** A daemon found on the LAN by mDNS discovery (mirrors DiscoveredAgent in Rust). */
export interface DiscoveredAgent {
  hostname: string;
  host: string;
  port: number;
  fingerprint: string;
  os: string;
  version: string;
  /** Whether the machine reports an open pairing window right now (null/absent
   *  on daemons too old to say). */
  pairable?: boolean | null;
  /** Id of a saved profile already pinned to this machine's key, if any. */
  pairedProfileId?: string | null;
}

/** Result of a successful pairing (mirrors AgentPairResult in Rust). */
export interface AgentPairResult {
  /** The daemon's static public key (base64) — pin it into the profile. */
  serverKey: string;
  fingerprint: string;
  hostname: string;
  os: string;
}

// ---- Remote control: this machine hosted as a Ghost FTP Agent (mirrors agent_host.rs) ----

/** A controller pinned to THIS machine. */
export interface AgentHostPeer {
  name: string;
  publicKey: string;
  fingerprint: string;
  pairedAt: number;
}

/** The open pairing window, if any. */
export interface AgentHostPairing {
  code: string;
  remainingSecs: number;
}

/** State of the in-app agent host (Settings → Remote control). */
export interface AgentHostStatus {
  enabled: boolean;
  running: boolean;
  port: number;
  hostname: string;
  os: string;
  fingerprint: string;
  allowExec: boolean;
  allowWrite: boolean;
  peers: AgentHostPeer[];
  pairing: AgentHostPairing | null;
}

/** One-click "Add ghostftp-cli to PATH" status (mirrors Rust's
 *  PathStatus). Per-user only — never needs admin. */
export interface PathStatus {
  /** The app-owned bin dir Ghost FTP uses for an explicitly installed CLI. */
  binDir: string;
  /** Whether a ghostftp-cli binary actually exists in binDir. */
  binHasCli: boolean;
  /** Whether ghostftp-cli resolves on PATH right now, and where. */
  onPath: boolean;
  cliLocation: string | null;
  /** Whether Ghost FTP's own managed entry is present (its bin dir on the Windows
   *  per-user Path, or the ~/.local/bin symlink on Linux). */
  managed: boolean;
  /** Short platform note after an add/remove ("Open a new terminal…"). */
  detail: string | null;
}

/** A parsed ghostftp:// deep link forwarded from Rust (mirrors DeepLink there).
 *  Every field optional; never carries a password. */
export interface DeepLink {
  action: "connect" | "pair" | "terminal" | string;
  protocol?: string;
  host?: string;
  port?: number;
  username?: string;
  path?: string;
  name?: string;
  code?: string;
  bucket?: string;
  region?: string;
  endpoint?: string;
  account?: string;
  // ghostftp://grant: issuer base URL + redemption token (docs/grant-links.md).
  issuer?: string;
  token?: string;
}

// ---- Access grants (ghostftp://grant; docs/grant-links.md) ----

/** Bastion/ProxyJump hop described by a grant manifest. */
export interface GrantJump {
  host: string;
  port?: number;
  username?: string;
}

export interface GrantConnection {
  name?: string;
  protocol: string;
  host: string;
  port?: number;
  username: string;
  path?: string;
  jump?: GrantJump;
}

/** What the issuer offers (fetched before consent, shown in the dialog). */
export interface GrantManifest {
  version: number;
  issuer: string;
  name: string;
  group?: string;
  expiresAt?: string;
  auth: { type: string };
  connections: GrantConnection[];
}

/** Result of the key-exchange + import after the user accepts a grant. */
export interface GrantImportResult {
  group: string;
  imported: ConnectionProfile[];
  failed: { name: string; error: string }[];
}

// ---- Importers ----

export interface ProfilePreview {
  previewId: string;
  name: string;
  protocol: Protocol;
  host: string;
  port: number;
  username: string;
  identityFile?: string;
  note?: string;
}

export type ImporterKind = "openssh" | "filezilla" | "putty";

export interface ImporterPaths {
  openssh: string | null;
  filezilla: string | null;
  putty: string | null;
}

// ---- Sync ----

export type SyncDirection = "localToRemote" | "remoteToLocal";
export type SyncStrategy = "additive" | "mirror";
export type SyncReason = "missing" | "newer" | "sizeChanged" | "edited";
/** How a pair materializes files locally. `mirror` moves whole files eagerly;
 *  `onDemand` registers OneDrive-style placeholders that hydrate on open
 *  (Windows-only, inert elsewhere). */
export type SyncMode = "mirror" | "onDemand";

export interface SyncFile {
  relative: string;
  sourcePath: string;
  destinationPath: string;
  size: number;
  reason: SyncReason;
}

export interface SyncDelete {
  relative: string;
  path: string;
  kind: FileKind;
  size: number;
}

export interface SyncPlan {
  direction: SyncDirection;
  strategy: SyncStrategy;
  localRoot: string;
  remoteRoot: string;
  copies: SyncFile[];
  deletes: SyncDelete[];
  totalBytes: number;
}

// ---- Folder Sync (continuous watched sync pairs; mirrors foldersync.rs) ----

export type PairState = "idle" | "scanning" | "syncing" | "error";

/** A configured, persisted sync pair (the user-editable half). */
export interface SyncPair {
  id: string; // "" when creating; backend assigns a uuid
  name: string;
  localRoot: string;
  profileId: string; // a ConnectionProfile.id
  remoteRoot: string;
  direction: SyncDirection;
  strategy: SyncStrategy;
  mode: SyncMode; // default "mirror"
  enabled: boolean;
  pollIntervalSecs: number; // default 60
  exclude: string[]; // gitignore-style patterns; never pushed nor mirror-deleted
  mirrorDeleteCap: number; // max deletes per Mirror reconcile; 0 = unlimited
}

/** A sync pair plus its live runtime status (what the backend returns). */
export interface PairView extends SyncPair {
  running: boolean;
  state: PairState;
  inFlight: number;
  lastSynced: number | null; // ms epoch
  lastError: string | null;
}

// ---- On-demand virtual folders (mirrors src-tauri/src/virtualfs) ----

/** Live status of one on-demand sync root (placeholders that hydrate on open). */
export interface VirtualFsRootStatus {
  pairId: string;
  localRoot: string;
  running: boolean; // a provider is connected and serving hydration callbacks
  lastError: string | null;
}

// ---- Disk Usage Explorer (mirrors src-tauri/src/diskscan.rs) ----

export type ScanState = "scanning" | "done" | "error" | "canceled";
/** Which strategy produced the scan. Phase 1 is always "generic"; the exec and
 *  object-store fast paths report "shell" / "objectFlat". */
export type ScanStrategy = "generic" | "shell" | "objectFlat";

/** One node in the aggregated size tree. `size` is total bytes under the node
 *  (own size for a file, sum of descendants for a directory). */
export interface DuNode {
  name: string;
  path: string;
  kind: FileKind;
  size: number;
  children?: DuNode[];
}

/** A scan snapshot: live progress while scanning, the `tree` once done. */
export interface ScanSnapshot {
  id: string;
  sessionId: string;
  root: string;
  state: ScanState;
  strategy: ScanStrategy;
  dirsScanned: number;
  filesFound: number;
  totalBytes: number;
  error?: string;
  /** Why the fast path fell back to the walk, when it did. */
  note?: string;
  tree?: DuNode;
  startedAt: number;
}

/** The lightweight `diskscan://progress` event body. */
export interface DiskScanProgress {
  id: string;
  dirsScanned: number;
  filesFound: number;
  totalBytes: number;
  strategy: ScanStrategy;
}

// ---- Directory Diff (mirrors src-tauri/src/diff.rs) ----

export type DiffClass = "onlyInA" | "onlyInB" | "different" | "same";
/** Why a both-present file was classified "different". */
export type DiffReason = "size" | "content";
export type DiffRunState = "comparing" | "done" | "error" | "canceled";
export type DiffPhase = "walkingA" | "walkingB" | "hashing";

/** One path in the diff, with whichever side(s) it appears on. */
export interface DiffEntry {
  relative: string;
  class: DiffClass;
  reason?: DiffReason;
  aPath?: string;
  aSize?: number;
  aModified?: number;
  bPath?: string;
  bSize?: number;
  bModified?: number;
  /** Set when `--hash` couldn't hash a side; the size classification stands. */
  hashError?: string;
}

export interface DiffSummary {
  onlyInA: number;
  onlyInB: number;
  different: number;
  same: number;
  total: number;
}

export interface DiffResult {
  rootA: string;
  rootB: string;
  hashed: boolean;
  summary: DiffSummary;
  entries: DiffEntry[];
}

/** A diff snapshot: live counts while comparing, the `result` once done. */
export interface DiffSnapshot {
  id: string;
  sessionA: string;
  pathA: string;
  sessionB: string;
  pathB: string;
  hashed: boolean;
  state: DiffRunState;
  phase: DiffPhase;
  filesA: number;
  filesB: number;
  error?: string;
  result?: DiffResult;
  startedAt: number;
}

/** The lightweight `diff://progress` event body. */
export interface DiffProgress {
  id: string;
  phase: DiffPhase;
  filesA: number;
  filesB: number;
}

// ---- Duplicate finder (mirrors src-tauri/src/dedupe.rs) ----

export type DedupeMode = "name" | "hash";
export type DedupeRunState = "scanning" | "done" | "error" | "canceled";
export type DedupePhase = "walking" | "hashing";

/** One file inside a duplicate group. */
export interface DedupeFile {
  path: string;
  size: number;
  modified: number;
}

/** A set of files believed identical; `keep` is the suggested survivor index. */
export interface DedupeGroup {
  key: string;
  size: number;
  hash?: string;
  files: DedupeFile[];
  keep: number;
}

export interface DedupeSummary {
  filesScanned: number;
  groups: number;
  /** Files that would be deleted keeping one per group. */
  duplicateFiles: number;
  /** Bytes reclaimed keeping one per group. */
  wastedBytes: number;
  hashErrors: number;
}

export interface DedupeResult {
  root: string;
  mode: DedupeMode;
  summary: DedupeSummary;
  groups: DedupeGroup[];
}

/** A dedupe snapshot: live counts while scanning, the `result` once done. */
export interface DedupeSnapshot {
  id: string;
  sessionId: string;
  path: string;
  mode: DedupeMode;
  state: DedupeRunState;
  phase: DedupePhase;
  filesFound: number;
  hashed: number;
  error?: string;
  result?: DedupeResult;
  startedAt: number;
}

/** The lightweight `dedupe://progress` event body. */
export interface DedupeProgress {
  id: string;
  phase: DedupePhase;
  filesFound: number;
  hashed: number;
}

// ---- Fleet Search (mirrors src-tauri/src/search.rs) ----

export type SearchKind = "name" | "content";
export type SearchRunState = "searching" | "done" | "error" | "canceled";
/** Which strategy ran: "generic" walk, "shell" (server-side rg/grep/find), or
 *  "objectFlat" (a bucket key listing, name search only). */
export type SearchStrategy = "generic" | "shell" | "objectFlat";

/** A search request sent to `search_start` (camelCase mirrors SearchQuery). */
export interface SearchQuery {
  pattern: string;
  kind: SearchKind;
  /** Content: treat the pattern as a regex. Ignored for name search. */
  regex: boolean;
  caseSensitive: boolean;
  includeGlobs: string[];
  excludeGlobs: string[];
  /** Allow content search to download files on grep-less backends. */
  contentRemote: boolean;
  maxResults: number;
  maxFileBytes: number;
}

/** One search hit. Name hits carry the entry; content hits add line/column/preview. */
export interface SearchHit {
  path: string;
  relative: string;
  isDir: boolean;
  size: number;
  line?: number;
  column?: number;
  preview?: string;
}

/** A search snapshot: live counts while searching, the `hits` once done. */
export interface SearchSnapshot {
  id: string;
  sessionId: string;
  root: string;
  kind: SearchKind;
  state: SearchRunState;
  strategy: SearchStrategy;
  filesScanned: number;
  hitCount: number;
  truncated: boolean;
  note?: string;
  error?: string;
  hits?: SearchHit[];
  startedAt: number;
}

/** The lightweight `search://progress` event body. */
export interface SearchProgress {
  id: string;
  strategy: SearchStrategy;
  filesScanned: number;
  hitCount: number;
}

/** A batch of newly-found hits streamed over `search://hit`. */
export interface SearchHitBatch {
  id: string;
  hits: SearchHit[];
}

// ---- Edit-in-place ----

export interface EditStartedEvent {
  editId: string;
  sessionId: string;
  remotePath: string;
  localTempPath: string;
}

export interface EditSavedEvent {
  editId: string;
  remotePath: string;
  bytes: number;
}

export interface EditErrorEvent {
  editId: string;
  remotePath: string;
  message: string;
}

export type S3Provider =
  | "aws"
  | "r2"
  | "b2"
  | "wasabi"
  | "spaces"
  | "minio"
  | "storj"
  | "hetzner"
  | "scaleway"
  | "oci"
  | "ibm"
  | "supabase"
  | "generic";

export interface S3ProviderPreset {
  label: string;
  /** Short vendor sub-label under the button (e.g. "Amazon", "Cloudflare"). */
  vendor: string;
  description: string;
  endpointHint: string; // displayed as placeholder
  defaultRegion: string;
}

// Curated S3-compatible presets. Each is purely a New-Connection convenience:
// the backend (`session/object.rs`) treats every non-AWS endpoint the same —
// path-style addressing, credentials from access-key/secret. Providers with a
// per-account endpoint (R2, MinIO, Spaces, Hetzner, OCI, IBM, Supabase) show the
// template as a placeholder; the user fills in their account/region/namespace.
export const S3_PROVIDER_PRESETS: Record<S3Provider, S3ProviderPreset> = {
  aws: {
    label: "AWS S3",
    vendor: "Amazon",
    description: "Native Amazon S3; endpoint derived from region.",
    endpointHint: "(leave blank — derived from region)",
    defaultRegion: "us-east-1",
  },
  r2: {
    label: "Cloudflare R2",
    vendor: "Cloudflare",
    description: "S3-compatible. Region is always 'auto'.",
    endpointHint: "https://<account>.r2.cloudflarestorage.com",
    defaultRegion: "auto",
  },
  b2: {
    label: "Backblaze B2",
    vendor: "Backblaze",
    description: "S3-compatible. Endpoint is per-bucket region.",
    endpointHint: "https://s3.us-west-002.backblazeb2.com",
    defaultRegion: "us-west-002",
  },
  wasabi: {
    label: "Wasabi",
    vendor: "Wasabi",
    description: "S3-compatible hot storage. Endpoint is region-specific.",
    endpointHint: "https://s3.us-east-1.wasabisys.com",
    defaultRegion: "us-east-1",
  },
  spaces: {
    label: "DO Spaces",
    vendor: "DigitalOcean",
    description: "DigitalOcean Spaces. Endpoint is the datacenter region.",
    endpointHint: "https://nyc3.digitaloceanspaces.com",
    defaultRegion: "nyc3",
  },
  minio: {
    label: "MinIO",
    vendor: "Self-hosted",
    description: "Self-hosted MinIO. Point the endpoint at your server.",
    endpointHint: "",
    defaultRegion: "us-east-1",
  },
  storj: {
    label: "Storj",
    vendor: "Storj DCS",
    description: "Storj S3-compatible gateway. Region is ignored.",
    endpointHint: "https://gateway.storjshare.io",
    defaultRegion: "us-east-1",
  },
  hetzner: {
    label: "Hetzner",
    vendor: "Hetzner",
    description: "Hetzner Object Storage. Endpoint is the location.",
    endpointHint: "https://fsn1.your-objectstorage.com",
    defaultRegion: "fsn1",
  },
  scaleway: {
    label: "Scaleway",
    vendor: "Scaleway",
    description: "Scaleway Object Storage. Endpoint is region-specific.",
    endpointHint: "https://s3.fr-par.scw.cloud",
    defaultRegion: "fr-par",
  },
  oci: {
    label: "Oracle OCI",
    vendor: "Oracle",
    description: "OCI Object Storage (S3 compat). Endpoint carries the namespace.",
    endpointHint: "https://<namespace>.compat.objectstorage.us-ashburn-1.oraclecloud.com",
    defaultRegion: "us-ashburn-1",
  },
  ibm: {
    label: "IBM COS",
    vendor: "IBM Cloud",
    description: "IBM Cloud Object Storage (S3 compat). Endpoint is region-specific.",
    endpointHint: "https://s3.us-south.cloud-object-storage.appdomain.cloud",
    defaultRegion: "us-south",
  },
  supabase: {
    label: "Supabase",
    vendor: "Supabase",
    description: "Supabase Storage S3 endpoint. Region matches the project.",
    endpointHint: "https://<project>.supabase.co/storage/v1/s3",
    defaultRegion: "us-east-1",
  },
  generic: {
    label: "S3-compatible",
    vendor: "Self-hosted",
    description: "Any S3 API server (Ceph RGW, Garage, SeaweedFS, …).",
    endpointHint: "",
    defaultRegion: "us-east-1",
  },
};

export type WebdavProvider = "nextcloud" | "owncloud" | "storagebox" | "generic";

export interface WebdavProviderPreset {
  label: string;
  vendor: string;
  description: string;
  /** URL template shown as a placeholder; `<user>` is substituted from the
   *  username when a preset is applied. */
  urlHint: string;
  /** Whether this preset expects a username (Basic auth). */
  wantsUser: boolean;
}

// WebDAV presets (the Phase-0 preset idea generalized beyond S3). Selecting one
// prefills the server-URL template; the user swaps in their host/username.
export const WEBDAV_PROVIDER_PRESETS: Record<WebdavProvider, WebdavProviderPreset> = {
  nextcloud: {
    label: "Nextcloud",
    vendor: "Nextcloud",
    description: "Use an app password (Settings → Security → Devices & sessions).",
    urlHint: "",
    wantsUser: true,
  },
  owncloud: {
    label: "ownCloud",
    vendor: "ownCloud",
    description: "Same DAV path as Nextcloud; an app password is recommended.",
    urlHint: "",
    wantsUser: true,
  },
  storagebox: {
    label: "Storage Box",
    vendor: "Hetzner",
    description: "Hetzner Storage Box over WebDAV. Enable WebDAV in the panel.",
    urlHint: "https://<user>.your-storagebox.de",
    wantsUser: true,
  },
  generic: {
    label: "Generic",
    vendor: "WebDAV",
    description: "Any WebDAV server. Basic auth, or leave the user blank for a bearer token.",
    urlHint: "",
    wantsUser: true,
  },
};

// File-system types live in the @ghostftp/file-ui package (the open-source UI owns
// them) and are re-exported here so the rest of the app keeps importing from
// "@/lib/types" unchanged. Single source of truth, no duplication.
export {
  LOCAL_SESSION,
  type FileKind,
  type DirEntry,
  type Capabilities,
  type SessionId,
} from "@ghostftp/file-ui";

// Local re-import so types defined in THIS file (e.g. SyncDelete) can still
// reference FileKind below.
import type { FileKind } from "@ghostftp/file-ui";

export interface TerminalDataEvent {
  terminalId: string;
  data: string;
}

export interface TerminalExitEvent {
  terminalId: string;
  code: number | null;
}

// Server-side asks the user whether to trust a host fingerprint we've never
// seen, or whose stored key has changed (potential MITM). The user replies via
// `respondToHostPrompt`. Mirrors src-tauri/src/session/mod.rs.
export type HostPromptKind = "unknown" | "mismatch";
export type HostDecision = "accept" | "trust" | "reject";

export interface HostPromptEvent {
  requestId: string;
  host: string;
  port: number;
  keyType: string;
  fingerprint: string;
  storedFingerprint?: string | null;
  kind: HostPromptKind;
}

// ---- Keyboard-interactive auth (e.g. forced password change for a temp pw) ----

export interface AuthPromptField {
  prompt: string;
  /** Whether the server wants the typed value echoed (false for passwords). */
  echo: boolean;
}

export interface AuthPromptEvent {
  requestId: string;
  profileId: string;
  host: string;
  name: string;
  instructions: string;
  prompts: AuthPromptField[];
}

export interface AuthChangedEvent {
  profileId: string;
}

// ---- Agent Bridge ----

export interface ApprovalPolicy {
  /** Master switch — approve every agent request on enabled sessions. */
  allowAll: boolean;
  /** Auto-approve read-only ops (list_dir, read_file, download, search). */
  autoRead: boolean;
  /** Auto-approve shell commands that look read-only (best-effort heuristic). */
  autoSafeExec: boolean;
  /** Answer sudo's prompt with the connection password (password-auth only).
   *  A capability gate, not an auto-approve — sudo commands still prompt. */
  allowSudo: boolean;
}

export interface BridgeStatus {
  running: boolean;
  /** Master on/off switch (persisted, default off). When false nothing listens. */
  enabled: boolean;
  url: string | null;
  port: number | null;
  token: string | null;
  enabledSessions: string[];
  policy: ApprovalPolicy;
}

export interface BridgeContext {
  bridgeRunning: boolean;
  policy: ApprovalPolicy;
  activeSessionId: string | null;
  sessions: Array<{
    id: string;
    name: string;
    protocol: string;
    host: string;
    canExec: boolean;
  }>;
  savedCommands: Array<{
    name: string;
    description: string;
  }>;
}

export interface BridgeActivity {
  id: string;
  sessionId: string;
  kind: string; // "exec" | "read" | "download" | "upload" | "upload_dir" | "sync" | "search" | "denied" | "error"
  detail: string;
  ok: boolean;
  at: number; // unix millis
}

export interface BridgeApproval {
  requestId: string;
  sessionId: string;
  sessionName: string;
  /** Operation kind: "exec" | "read" | "download" | "upload" | "upload_dir"
   *  | "sync" | "search". */
  kind: string;
  /** Human-readable summary of what the agent wants to do. */
  command: string;
}

export type ApprovalDecision = "approve" | "deny";

/** A user-defined, pre-approved command the agent can run by name. The agent
 *  only supplies the name; the exact `command` was vetted by the user, so it
 *  runs with no approval prompt. Managed only in Ghost FTP's UI (never over the
 *  bridge), so the agent can't author one. */
export interface SavedCommand {
  id: string;
  name: string;
  command: string;
  description: string;
}

// ---- Command snippets: the low-friction, single-session
// counterpart to Fleet Skills — a saved command line with optional {{variable}}
// placeholders, inserted into a live shell with one keystroke. ----
export interface Snippet {
  id: string;
  name: string;
  body: string;
  /** Optional grouping label shown in the panel. */
  folder: string | null;
  /** Times inserted — drives palette ordering (most-used first). */
  useCount: number;
  createdMs: number;
  updatedMs: number;
}

// ---- Skills: parameterized, fleet-targetable, AI-authorable ----

export type SkillStatus = "approved" | "proposed";

/** One named input a Skill's steps interpolate via `${name}`. */
export interface SkillParam {
  name: string;
  description: string;
  required: boolean;
  default: string | null;
}

/** One linear step of a Skill: a shell command template (`${param}` allowed). */
export interface SkillStep {
  name: string;
  command: string;
}

/** Which connected servers a Skill runs on by default. */
export interface TargetSelector {
  all: boolean;
  sessions: string[];
}

/** A saved, AI-authorable Skill: a named, parameterized, multi-step workflow the
 *  agent (or the user) can run across one or many connected servers. Proposed
 *  skills are AI-authored and need one human approval before they can run. */
export interface Skill {
  id: string;
  name: string;
  description: string;
  params: SkillParam[];
  steps: SkillStep[];
  targets: TargetSelector;
  status: SkillStatus;
  createdBy: string; // "user" | "ai"
  stopOnError: boolean;
}

/** Result of one step on one target in a skill run. */
export interface SkillStepResult {
  step: number;
  name: string;
  command: string;
  ok: boolean;
  exitCode?: number | null;
  stdout?: string;
  stderr?: string;
  truncated?: boolean;
  timedOut?: boolean;
  error?: string;
}

export interface SkillTargetResult {
  sessionId: string;
  sessionName: string;
  ok: boolean;
  steps: SkillStepResult[];
}

export interface SkillSkipped {
  target: string;
  reason: string;
}

/** Aggregated result of running a skill (real run). */
export interface SkillRunResult {
  skill: string;
  status: string;
  targetCount: number;
  succeeded: number;
  failed: number;
  results: SkillTargetResult[];
  skipped: SkillSkipped[];
}

/** Result of a dry-run: resolved commands per target, nothing executed. */
export interface SkillDryRunResult {
  dryRun: true;
  skill: string;
  proposal: boolean;
  stepCount: number;
  targets: { sessionId: string; sessionName: string; commands: string[] }[];
  skipped: SkillSkipped[];
  needsApproval: boolean;
}

// Live agent console (streamed exec output + op feed).
export interface AgentExecStart {
  opId: string;
  sessionId: string;
  sessionName: string;
  command: string;
}

export interface AgentOutput {
  opId: string;
  stream: "stdout" | "stderr";
  chunk: string;
}

export interface AgentConsoleEntry {
  id: string;
  sessionId: string;
  sessionName?: string;
  kind: string; // exec | read | download | upload | search | denied | error
  command: string; // command (exec) or op summary
  output: string; // streamed stdout/stderr (exec only)
  status: "running" | "done";
  ok?: boolean;
  at: number;
}

export type TransferKind = "download" | "upload";
export type TransferStatus =
  | "queued"
  | "transferring"
  | "paused"
  | "done"
  | "skipped"
  | "error"
  | "canceled";
export type OverwritePolicy = "overwrite" | "skip" | "rename";

export interface Transfer {
  id: string;
  kind: TransferKind;
  source: string;
  destination: string;
  size: number;
  transferred: number;
  status: TransferStatus;
  error?: string;
  /** Set while the backend auto-retries a failed transfer (`error` reads
   *  "retrying in Ns (attempt N/3)"). */
  retryAttempt?: number;
  /** Delta-sync stats, present only when the transfer ran over the
   *  Ghost FTP Agent backend with delta enabled. `sent` = literal bytes that
   *  crossed the wire; `reused` = bytes sourced from the basis instead.
   *  `sent + reused` equals the total file size. Set before finalize, so it
   *  persists on done transfers. */
  delta?: { sent: number; reused: number };
  startedAt: number;
}

/** Live transfer-queue snapshot — payload of `transfer://queue` and
 *  the return of `transfer_queue_state`. `waiting` holds the FIFO order of
 *  queued transfer ids. */
export interface TransferQueueState {
  waiting: string[];
  pausedAll: boolean;
  concurrency: number;
  throttleKbps: number;
}

/** What an encrypted backup contains. */
export interface BackupSummary {
  profiles: number;
  credentials: number;
  hasBridge: boolean;
  hasSync: boolean;
  dbBytes: number;
}
