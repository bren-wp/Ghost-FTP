import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import type {
  ConnectionProfile,
  GenerateKeyRequest,
  GeneratedKey,
  SshKeyDefaults,
  DirEntry,
  Capabilities,
  EditErrorEvent,
  EditSavedEvent,
  EditStartedEvent,
  HostDecision,
  HostPromptEvent,
  AuthPromptEvent,
  AuthChangedEvent,
  ImporterPaths,
  ProfilePreview,
  SessionId,
  SyncDirection,
  SyncPlan,
  SyncStrategy,
  TerminalDataEvent,
  TerminalExitEvent,
  Transfer,
  TransferQueueState,
  OverwritePolicy,
  BridgeStatus,
  BridgeActivity,
  BridgeApproval,
  ApprovalDecision,
  ApprovalPolicy,
  AgentExecStart,
  AgentOutput,
  SavedCommand,
  Snippet,
  Skill,
  SkillRunResult,
  SkillDryRunResult,
  DiscoveredAgent,
  AgentPairResult,
  AgentHostStatus,
  CliStatus,
  CliUpdateMode,
  PathStatus,
  DeepLink,
  GrantManifest,
  GrantImportResult,
  SyncPair,
  PairView,
  VirtualFsRootStatus,
  ScanSnapshot,
  DiskScanProgress,
  DiffSnapshot,
  DiffProgress,
  DedupeSnapshot,
  DedupeProgress,
  SearchQuery,
  SearchSnapshot,
  SearchProgress,
  SearchHitBatch,
  BackupSummary,
} from "./types";

// Typed wrappers around the Tauri command surface. The string names must match
// the #[tauri::command] handlers registered in src-tauri/src/commands.rs.

export const ipc = {
  openExternalUrl: (url: string) => invoke<void>("open_external_url", { url }),

  listProfiles: () => invoke<ConnectionProfile[]>("list_profiles"),
  exportProfiles: (path: string) => invoke<number>("export_profiles", { path }),

  saveProfile: (profile: ConnectionProfile) =>
    invoke<void>("save_profile", { profile }),

  duplicateProfile: (id: string) =>
    invoke<ConnectionProfile>("duplicate_profile", { id }),

  /** Persist a manual rail order: every profile id, in display order. */
  reorderProfiles: (ids: string[]) =>
    invoke<void>("reorder_profiles", { ids }),

  deleteProfile: (id: string) => invoke<void>("delete_profile", { id }),

  // ---- In-app SSH key generation ----

  /** Suggested `~/.ssh` dir + a non-colliding filename for a new key. */
  sshKeyDefaults: () => invoke<SshKeyDefaults>("ssh_key_defaults"),

  /** Generate a keypair, write the private key + `.pub`, return the public key. */
  generateSshKey: (req: GenerateKeyRequest) =>
    invoke<GeneratedKey>("generate_ssh_key", { req }),

  /** Derive the public-key line for an existing private key path (needs the
   *  passphrase if the key is encrypted). */
  sshPublicKeyFor: (path: string, passphrase?: string) =>
    invoke<GeneratedKey>("ssh_public_key_for", { path, passphrase }),

  testProfileConnection: (profileId: string) =>
    invoke<void>("test_profile_connection", { profileId }),

  testEphemeralConnection: (profile: ConnectionProfile) =>
    invoke<void>("test_ephemeral_connection", { profile }),

  connect: (profileId: string) =>
    invoke<SessionId>("connect", { profileId }),

  /** Open an in-memory session without persisting a profile or credential. */
  connectEphemeral: (profile: ConnectionProfile) =>
    invoke<SessionId>("connect_ephemeral", { profile }),

  disconnect: (sessionId: SessionId) =>
    invoke<void>("disconnect", { sessionId }),

  // ---- Ghost FTP Agent (Ghost FTP-to-Ghost FTP remote control) ----

  /** Browse the LAN for ghostftp-agentd daemons over mDNS (best-effort). */
  discoverAgents: () => invoke<DiscoveredAgent[]>("discover_agents"),

  /** This controller's own agent public key, for the pairing UI. */
  agentPublicKey: () => invoke<string>("agent_public_key"),

  /** Pair with a ghostftp-agentd at host:port using a one-time 6-digit code.
   *  Returns the daemon's key to pin into the profile; persists nothing itself,
   *  so a failed pairing leaves no half-configured connection behind. */
  pairAgent: (host: string, port: number, code: string) =>
    invoke<AgentPairResult>("pair_agent", { host, port, code }),

  /** Run the interactive Dropbox OAuth flow for a profile (opens the browser,
   *  catches the loopback redirect, stores tokens in the OS keychain). Resolves
   *  with the authorized account label. The editor saves the profile on success. */
  dropboxAuthorize: (profileId: string) =>
    invoke<{ accountLabel: string }>("dropbox_authorize", { profileId }),

  /** Run the interactive OneDrive OAuth flow for a profile (Microsoft Graph). */
  onedriveAuthorize: (profileId: string) =>
    invoke<{ accountLabel: string }>("onedrive_authorize", { profileId }),

  /** Run the interactive Google Drive OAuth flow for a profile. */
  gdriveAuthorize: (profileId: string) =>
    invoke<{ accountLabel: string }>("gdrive_authorize", { profileId }),

  /** Run the interactive Box OAuth flow for a profile. */
  boxAuthorize: (profileId: string) =>
    invoke<{ accountLabel: string }>("box_authorize", { profileId }),

  /** Run the interactive Dynamics 365 (delegated) OAuth flow for a profile —
   *  the OneDrive pattern with the org's `user_impersonation` scope. `host` is
   *  the environment URL (`{org}.crm.dynamics.com`) the editor collected. */
  dynamicsAuthorize: (profileId: string, host: string) =>
    invoke<{ accountLabel: string }>("dynamics_authorize", { profileId, host }),

  // ---- Remote control: host THIS machine as a Ghost FTP Agent (Settings) ----

  /** Current state of the in-app agent host (enabled, running, policy, peers,
   *  open pairing window). */
  agentHostStatus: () => invoke<AgentHostStatus>("agent_host_status"),

  /** Turn the host on/off; optionally set the listen port. */
  agentHostSetEnabled: (enabled: boolean, port?: number) =>
    invoke<AgentHostStatus>("agent_host_set_enabled", { enabled, port }),

  /** Open a pairing window and get the fresh 6-digit code to read aloud. */
  agentHostOpenPairing: () =>
    invoke<AgentHostStatus>("agent_host_open_pairing"),

  /** Close the pairing window early (e.g. the dialog closed). */
  agentHostClosePairing: () =>
    invoke<AgentHostStatus>("agent_host_close_pairing"),

  /** Set what paired controllers may do on THIS machine. */
  agentHostSetPolicy: (allowExec: boolean, allowWrite: boolean) =>
    invoke<AgentHostStatus>("agent_host_set_policy", { allowExec, allowWrite }),

  /** Un-pin a controller by its public key. */
  agentHostRevokePeer: (publicKey: string) =>
    invoke<AgentHostStatus>("agent_host_revoke_peer", { publicKey }),

  // --- CLI updater (Plan 10 Phase 0c/0d): keep ghostftp-cli in step with the app.
  cliUpdaterStatus: () => invoke<CliStatus>("cli_updater_status"),
  cliUpdaterCheck: () => invoke<CliStatus>("cli_updater_check"),
  cliUpdaterUpdate: () => invoke<CliStatus>("cli_updater_update"),
  cliUpdaterSetMode: (mode: CliUpdateMode) =>
    invoke<CliStatus>("cli_updater_set_mode", { mode }),

  // ---- One-click "Add ghostftp-cli to PATH" (Plan 16 Phase 4): per-user, no admin.
  pathStatus: () => invoke<PathStatus>("path_status"),
  pathAdd: () => invoke<PathStatus>("path_add"),
  pathRemove: () => invoke<PathStatus>("path_remove"),

  listDirectory: (sessionId: SessionId, path: string) =>
    invoke<DirEntry[]>("list_directory", { sessionId, path }),

  capabilities: (sessionId: SessionId) =>
    invoke<Capabilities>("capabilities", { sessionId }),

  // Base64 of a small local file, for image thumbnails in the grid view.
  readFilePreview: (sessionId: SessionId, path: string) =>
    invoke<string>("read_file_preview", { sessionId, path }),

  // Base64 of a Rust-decoded+downscaled thumbnail for a remote (or local) image
  // (Plan 13 Phase 1). `signal` is the file's change token (an object-store ETag
  // or "size:mtime") — it keys the disk cache so edits invalidate. Rejects for
  // oversized/undecodable/unsupported files; the adapter maps that to "no
  // preview" → the type icon.
  previewThumbnail: (
    sessionId: SessionId,
    path: string,
    fileSize: number,
    signal: string
  ) => invoke<string>("preview_thumbnail", { sessionId, path, fileSize, signal }),

  openTerminal: (sessionId: SessionId, cols: number, rows: number) =>
    invoke<string>("open_terminal", { sessionId, cols, rows }),

  terminalWrite: (terminalId: string, data: string) =>
    invoke<void>("terminal_write", { terminalId, data }),

  terminalResize: (terminalId: string, cols: number, rows: number) =>
    invoke<void>("terminal_resize", { terminalId, cols, rows }),

  closeTerminal: (terminalId: string) =>
    invoke<void>("close_terminal", { terminalId }),

  // ---- Command snippets (Plan 11 Phase 4) ----
  // Every mutation returns the full, re-ordered list (most-used first).
  snippetList: () => invoke<Snippet[]>("snippet_list"),
  snippetSave: (snippet: Snippet) =>
    invoke<Snippet[]>("snippet_save", { snippet }),
  snippetDelete: (id: string) => invoke<Snippet[]>("snippet_delete", { id }),
  /** Record one insertion so the snippet floats up the ordering. */
  snippetRun: (id: string) => invoke<Snippet[]>("snippet_run", { id }),

  startDownload: (
    sessionId: SessionId,
    remotePath: string,
    localDir: string,
    overwritePolicy?: OverwritePolicy
  ) =>
    invoke<string>("start_download", {
      sessionId,
      remotePath,
      localDir,
      overwritePolicy,
    }),

  startUpload: (
    sessionId: SessionId,
    localPath: string,
    remoteDir: string,
    overwritePolicy?: OverwritePolicy
  ) =>
    invoke<string>("start_upload", {
      sessionId,
      localPath,
      remoteDir,
      overwritePolicy,
    }),

  cancelTransfer: (transferId: string) =>
    invoke<void>("cancel_transfer", { transferId }),

  listTransfers: () => invoke<Transfer[]>("list_transfers"),

  transferPause: (transferId: string) =>
    invoke<void>("transfer_pause", { transferId }),

  transferResume: (transferId: string) =>
    invoke<void>("transfer_resume", { transferId }),

  transferRetry: (transferId: string) =>
    invoke<void>("transfer_retry", { transferId }),

  transferMove: (transferId: string, direction: "up" | "down") =>
    invoke<void>("transfer_move", { transferId, direction }),

  transferPauseAll: () => invoke<void>("transfer_pause_all"),

  transferResumeAll: () => invoke<void>("transfer_resume_all"),

  transferSetConcurrency: (count: number) =>
    invoke<void>("transfer_set_concurrency", { count }),

  transferSetMaxRetries: (attempts: number) =>
    invoke<void>("transfer_set_max_retries", { attempts }),

  transferSetThrottle: (kbps: number) =>
    invoke<void>("transfer_set_throttle", { kbps }),

  transferSetDeltaSync: (enabled: boolean) =>
    invoke<void>("transfer_set_delta_sync", { enabled }),

  transferQueueState: () => invoke<TransferQueueState>("transfer_queue_state"),

  startDirectoryDownload: (
    sessionId: SessionId,
    remoteDir: string,
    localDir: string,
    overwritePolicy?: OverwritePolicy
  ) =>
    invoke<string[]>("start_directory_download", {
      sessionId,
      remoteDir,
      localDir,
      overwritePolicy,
    }),

  startDirectoryUpload: (
    sessionId: SessionId,
    localDir: string,
    remoteDir: string,
    overwritePolicy?: OverwritePolicy
  ) =>
    invoke<string[]>("start_directory_upload", {
      sessionId,
      localDir,
      remoteDir,
      overwritePolicy,
    }),

  renamePath: (sessionId: SessionId, from: string, to: string) =>
    invoke<void>("rename_path", { sessionId, from, to }),

  deletePath: (sessionId: SessionId, path: string, recursive: boolean) =>
    invoke<void>("delete_path", { sessionId, path, recursive }),

  createDirectory: (sessionId: SessionId, path: string) =>
    invoke<void>("create_directory", { sessionId, path }),

  chmodPath: (sessionId: SessionId, path: string, mode: number) =>
    invoke<void>("chmod_path", { sessionId, path, mode }),

  chmodPathRecursive: (sessionId: SessionId, path: string, mode: number) =>
    invoke<void>("chmod_path_recursive", { sessionId, path, mode }),

  checksumPath: (sessionId: SessionId, path: string) =>
    invoke<string>("checksum_path", { sessionId, path }),

  // Copy a file/folder alongside the original; backend picks a free name.
  duplicatePath: (sessionId: SessionId, path: string) =>
    invoke<void>("duplicate_path", { sessionId, path }),

  // Build a tar.gz/zip of a remote dir on the server, then download it. Returns
  // the resulting transfer id (the archive shows in the transfers panel).
  startArchiveDownload: (
    sessionId: SessionId,
    remotePath: string,
    format: "tar.gz" | "zip"
  ) =>
    invoke<string>("start_archive_download", {
      sessionId,
      remotePath,
      format,
    }),

  respondToHostPrompt: (requestId: string, decision: HostDecision) =>
    invoke<void>("respond_to_host_prompt", { requestId, decision }),

  // Answer a keyboard-interactive auth prompt; pass null to cancel (aborts connect).
  respondToAuthPrompt: (requestId: string, responses: string[] | null) =>
    invoke<void>("respond_to_auth_prompt", { requestId, responses }),

  importerDefaultPaths: () => invoke<ImporterPaths>("importer_default_paths"),

  importOpenssh: (path?: string) =>
    invoke<ProfilePreview[]>("import_openssh", { path }),

  importFilezilla: (path?: string) =>
    invoke<ProfilePreview[]>("import_filezilla", { path }),

  importPutty: () => invoke<ProfilePreview[]>("import_putty"),

  saveImportedProfiles: (previews: ProfilePreview[]) =>
    invoke<number>("save_imported_profiles", { previews }),

  // ---- Access grants (ghostftp://grant; docs/grant-links.md) ----
  /** Fetch the manifest a grant token offers (before consent). */
  fetchGrantManifest: (issuer: string, token: string) =>
    invoke<GrantManifest>("fetch_grant_manifest", { issuer, token }),

  /** After consent: generate a key, upload the public half, import the
   *  granted servers as ordinary profiles. */
  acceptGrant: (issuer: string, token: string, manifest: GrantManifest) =>
    invoke<GrantImportResult>("accept_grant", { issuer, token, manifest }),

  syncPlan: (
    sessionId: SessionId,
    localPath: string,
    remotePath: string,
    direction: SyncDirection,
    strategy: SyncStrategy
  ) =>
    invoke<SyncPlan>("sync_plan", {
      sessionId,
      localPath,
      remotePath,
      direction,
      strategy,
    }),

  syncExecute: (sessionId: SessionId, plan: SyncPlan) =>
    invoke<string[]>("sync_execute", { sessionId, plan }),

  startEdit: (sessionId: SessionId, remotePath: string, editor?: string) =>
    invoke<EditStartedEvent>("start_edit", { sessionId, remotePath, editor }),

  stopEdit: (editId: string) => invoke<void>("stop_edit", { editId }),

  // Agent Bridge
  bridgeStart: () => invoke<BridgeStatus>("bridge_start"),
  bridgeStop: () => invoke<BridgeStatus>("bridge_stop"),
  bridgeSetEnabled: (enabled: boolean) =>
    invoke<BridgeStatus>("bridge_set_enabled", { enabled }),
  bridgeStatus: () => invoke<BridgeStatus>("bridge_status"),
  bridgeSetSessionAccess: (sessionId: SessionId, enabled: boolean) =>
    invoke<BridgeStatus>("bridge_set_session_access", { sessionId, enabled }),
  bridgeSetPolicy: (policy: ApprovalPolicy) =>
    invoke<BridgeStatus>("bridge_set_policy", { policy }),
  bridgeSetActiveSession: (sessionId: SessionId | null) =>
    invoke<void>("bridge_set_active_session", { sessionId }),
  bridgeRegisterMcp: (url: string, token: string) =>
    invoke<string>("bridge_register_mcp", { url, token }),
  // ---- Service credentials (Plan 12 Phase 1) ----
  // Secrets never cross IPC as readable values: the frontend writes (one-way)
  // and asks whether one exists, but never reads the value back. Rust fetches
  // it from the OS keychain at the point of use.
  /** Store (or clear, if `value` is "") a service credential by purpose. */
  setApiKey: (purpose: string, value: string) =>
    invoke<void>("set_api_key", { purpose, value }),
  /** Whether a credential exists for `purpose` (for the Set/••••/Clear UI). */
  apiKeyStatus: (purpose: string) =>
    invoke<boolean>("api_key_status", { purpose }),

  // ---- Settings (Plan 12 Phase 2): ghostftp.db is the source of truth ----
  /** Every setting as `key -> raw JSON value`. */
  settingsGetAll: () =>
    invoke<Record<string, string>>("settings_get_all"),
  /** Upsert one setting (`value` is a JSON-stringified value). */
  settingsSet: (key: string, value: string) =>
    invoke<void>("settings_set", { key, value }),
  /** Delete one setting row (reset-to-default, e.g. clearing a shortcut override). */
  settingsDelete: (key: string) =>
    invoke<void>("settings_delete", { key }),
  /** Bulk-upsert (the one-time localStorage import). */
  settingsSetAll: (values: Record<string, string>) =>
    invoke<void>("settings_set_all", { values }),

  // ---- Encrypted backup / restore (Plan 12 Phase 4) ----
  /** Write an encrypted backup to `path`, protected by `password`. */
  backupExport: (path: string, password: string) =>
    invoke<BackupSummary>("backup_export", { path, password }),
  /** Decrypt + report what's inside a backup, without applying it. */
  backupInspect: (path: string, password: string) =>
    invoke<BackupSummary>("backup_inspect", { path, password }),
  /** Restore a backup (staged for next launch; credentials injected now). */
  backupImport: (path: string, password: string) =>
    invoke<BackupSummary>("backup_import", { path, password }),

  respondToBridgeApproval: (requestId: string, decision: ApprovalDecision) =>
    invoke<void>("respond_to_bridge_approval", { requestId, decision }),
  bridgeActivity: () => invoke<BridgeActivity[]>("bridge_activity"),
  bridgeClearActivity: () => invoke<void>("bridge_clear_activity"),
  // Saved commands (pre-approved; managed only here, never over the bridge).
  bridgeListCommands: () => invoke<SavedCommand[]>("bridge_list_commands"),
  bridgeSaveCommand: (command: SavedCommand) =>
    invoke<SavedCommand[]>("bridge_save_command", { command }),
  bridgeDeleteCommand: (id: string) =>
    invoke<SavedCommand[]>("bridge_delete_command", { id }),
  // Skills (Plan 8): fleet automations. Hand-authored skills are born approved;
  // AI proposals are approved via bridgeApproveSkill before they can run.
  bridgeListSkills: () => invoke<Skill[]>("bridge_list_skills"),
  bridgeSaveSkill: (skill: Skill) => invoke<Skill[]>("bridge_save_skill", { skill }),
  bridgeDeleteSkill: (id: string) => invoke<Skill[]>("bridge_delete_skill", { id }),
  bridgeApproveSkill: (id: string) => invoke<Skill[]>("bridge_approve_skill", { id }),
  bridgeRunSkill: (
    name: string,
    params: Record<string, string>,
    targets: string[] | null,
    dryRun: boolean,
  ) =>
    invoke<SkillRunResult | SkillDryRunResult>("bridge_run_skill", {
      name,
      params,
      targets,
      dryRun,
    }),
  // Write the agent console text to disk (Downloads) and return the saved path.
  exportAgentLog: (content: string, name: string) =>
    invoke<string>("export_agent_log", { content, name }),

  // ---- Folder Sync (continuous watched sync pairs) ----
  // Every command returns the full, updated list of pairs.
  folderSyncList: () => invoke<PairView[]>("foldersync_list"),
  folderSyncUpsert: (pair: SyncPair) =>
    invoke<PairView[]>("foldersync_upsert", { pair }),
  folderSyncRemove: (id: string) =>
    invoke<PairView[]>("foldersync_remove", { id }),
  folderSyncSetEnabled: (id: string, enabled: boolean) =>
    invoke<PairView[]>("foldersync_set_enabled", { id, enabled }),
  folderSyncSyncNow: (id: string) =>
    invoke<PairView[]>("foldersync_sync_now", { id }),

  // ---- On-demand virtual folders (Plan 9) ----
  /** Whether on-demand placeholders are available (Windows + `virtualfs` build). */
  virtualFsSupported: () => invoke<boolean>("virtualfs_supported"),
  /** Live status of every registered on-demand sync root. */
  virtualFsStatus: () => invoke<VirtualFsRootStatus[]>("virtualfs_status"),
  /** Explorer-style "free up space": dehydrate a pair's hydrated files back to
   *  placeholders. Returns the count dehydrated. */
  virtualFsFreeUpSpace: (pairId: string) =>
    invoke<number>("virtualfs_free_up_space", { pairId }),

  // ---- Disk Usage Explorer ----
  /** Kick off a recursive size scan of `path` on a session; returns a scan id. */
  diskScanStart: (sessionId: SessionId, path: string) =>
    invoke<string>("diskscan_start", { sessionId, path }),
  /** Lightweight status (live counts while scanning). */
  diskScanStatus: (scanId: string) =>
    invoke<ScanSnapshot>("diskscan_status", { scanId }),
  /** Full snapshot including the aggregated tree (present once done). */
  diskScanTree: (scanId: string) =>
    invoke<ScanSnapshot>("diskscan_tree", { scanId }),
  diskScanCancel: (scanId: string) =>
    invoke<void>("diskscan_cancel", { scanId }),
  /** Drop a finished scan from the backend (tab closed). */
  diskScanForget: (scanId: string) =>
    invoke<void>("diskscan_forget", { scanId }),

  /** Start a two-tree directory diff; returns a diff id. Each side is a session
   *  id (or LOCAL_SESSION) + a path. `hash` confirms same-size files by content. */
  diffStart: (
    sessionA: SessionId,
    pathA: string,
    sessionB: SessionId,
    pathB: string,
    hash: boolean
  ) =>
    invoke<string>("diff_start", { sessionA, pathA, sessionB, pathB, hash }),
  /** Lightweight status (live counts while comparing). */
  diffStatus: (diffId: string) =>
    invoke<DiffSnapshot>("diff_status", { diffId }),
  /** Full snapshot including the result (present once done). */
  diffResult: (diffId: string) =>
    invoke<DiffSnapshot>("diff_result", { diffId }),
  diffCancel: (diffId: string) => invoke<void>("diff_cancel", { diffId }),
  /** Drop a finished diff from the backend (view closed). */
  diffForget: (diffId: string) => invoke<void>("diff_forget", { diffId }),

  /** Start a duplicate scan under `path` on a session (or LOCAL_SESSION);
   *  returns a dedupe id. `hash` groups by content instead of name+size. */
  dedupeStart: (sessionId: SessionId, path: string, hash: boolean) =>
    invoke<string>("dedupe_start", { sessionId, path, hash }),
  /** Lightweight status (live counts while scanning). */
  dedupeStatus: (dedupeId: string) =>
    invoke<DedupeSnapshot>("dedupe_status", { dedupeId }),
  /** Full snapshot including the result (present once done). */
  dedupeResult: (dedupeId: string) =>
    invoke<DedupeSnapshot>("dedupe_result", { dedupeId }),
  dedupeCancel: (dedupeId: string) =>
    invoke<void>("dedupe_cancel", { dedupeId }),
  /** Drop a finished scan from the backend (view closed). */
  dedupeForget: (dedupeId: string) =>
    invoke<void>("dedupe_forget", { dedupeId }),
  /** Delete an explicit list of paths on a session. Returns per-path errors
   *  (empty = everything deleted). */
  dedupeDelete: (sessionId: SessionId, paths: string[]) =>
    invoke<string[]>("dedupe_delete", { sessionId, paths }),

  /** Start a fleet search under `path` on a session; returns a search id. */
  searchStart: (sessionId: SessionId, path: string, query: SearchQuery) =>
    invoke<string>("search_start", { sessionId, path, query }),
  /** Lightweight status (live counts while searching). */
  searchStatus: (searchId: string) =>
    invoke<SearchSnapshot>("search_status", { searchId }),
  /** Full snapshot including the hit list (present once done). */
  searchResult: (searchId: string) =>
    invoke<SearchSnapshot>("search_result", { searchId }),
  searchCancel: (searchId: string) =>
    invoke<void>("search_cancel", { searchId }),
  /** Drop a finished search from the backend (panel closed). */
  searchForget: (searchId: string) =>
    invoke<void>("search_forget", { searchId }),
};

export async function onEditSaved(
  cb: (event: EditSavedEvent) => void
): Promise<UnlistenFn> {
  return listen<EditSavedEvent>("editor://saved", (e) => cb(e.payload));
}

export async function onEditError(
  cb: (event: EditErrorEvent) => void
): Promise<UnlistenFn> {
  return listen<EditErrorEvent>("editor://error", (e) => cb(e.payload));
}

export async function onHostPrompt(
  cb: (event: HostPromptEvent) => void
): Promise<UnlistenFn> {
  return listen<HostPromptEvent>("host://prompt", (e) => cb(e.payload));
}

export async function onAuthPrompt(
  cb: (event: AuthPromptEvent) => void
): Promise<UnlistenFn> {
  return listen<AuthPromptEvent>("auth://prompt", (e) => cb(e.payload));
}

export async function onAuthChanged(
  cb: (event: AuthChangedEvent) => void
): Promise<UnlistenFn> {
  return listen<AuthChangedEvent>("auth://changed", (e) => cb(e.payload));
}

export async function onTerminalData(
  cb: (event: TerminalDataEvent) => void
): Promise<UnlistenFn> {
  return listen<TerminalDataEvent>("terminal://data", (e) => cb(e.payload));
}

export async function onTerminalExit(
  cb: (event: TerminalExitEvent) => void
): Promise<UnlistenFn> {
  return listen<TerminalExitEvent>("terminal://exit", (e) => cb(e.payload));
}

export async function onBridgeApproval(
  cb: (event: BridgeApproval) => void
): Promise<UnlistenFn> {
  return listen<BridgeApproval>("bridge://approval", (e) => cb(e.payload));
}

export async function onBridgeActivity(
  cb: (event: BridgeActivity) => void
): Promise<UnlistenFn> {
  return listen<BridgeActivity>("bridge://activity", (e) => cb(e.payload));
}

export async function onAgentExecStart(
  cb: (event: AgentExecStart) => void
): Promise<UnlistenFn> {
  return listen<AgentExecStart>("agent://exec-start", (e) => cb(e.payload));
}

/** The AI proposed a new Skill over the bridge (awaiting human approval). */
export async function onBridgeSkillProposed(
  cb: (skill: Skill) => void
): Promise<UnlistenFn> {
  return listen<Skill>("bridge://skill-proposed", (e) => cb(e.payload));
}

export async function onAgentOutput(
  cb: (event: AgentOutput) => void
): Promise<UnlistenFn> {
  return listen<AgentOutput>("agent://output", (e) => cb(e.payload));
}

/** A controller finished pairing with THIS machine's hosted agent. */
export async function onAgentHostPaired(
  cb: (event: { name: string; key: string }) => void
): Promise<UnlistenFn> {
  return listen<{ name: string; key: string }>("agent-host://paired", (e) =>
    cb(e.payload)
  );
}

/** CLI version-drift status changed (startup check, or an update finished). */
export async function onCliUpdaterStatus(
  cb: (status: CliStatus) => void
): Promise<UnlistenFn> {
  return listen<CliStatus>("cli-updater://status", (e) => cb(e.payload));
}

/** A folder-sync pair changed (config edited, or a background sync updated its
 *  runtime state). Payload is empty — re-fetch the list on receipt. */
export async function onFolderSyncChanged(
  cb: () => void
): Promise<UnlistenFn> {
  return listen("foldersync://changed", () => cb());
}

/** A ghostftp:// deep link was opened (from a hosting panel like a hosting panel). */
export async function onDeepLink(
  cb: (link: DeepLink) => void
): Promise<UnlistenFn> {
  return listen<DeepLink>("deep-link://open", (e) => cb(e.payload));
}

/** Disk-usage scan lifecycle. `progress` carries live counts; the terminal
 *  events (`done`/`error`/`canceled`) carry a full ScanSnapshot with the tree. */
export async function onDiskScanEvent(
  cb: (
    kind: "progress" | "done" | "error" | "canceled",
    payload: DiskScanProgress | ScanSnapshot
  ) => void
): Promise<UnlistenFn> {
  const unsubs = await Promise.all([
    listen<DiskScanProgress>("diskscan://progress", (e) =>
      cb("progress", e.payload)
    ),
    listen<ScanSnapshot>("diskscan://done", (e) => cb("done", e.payload)),
    listen<ScanSnapshot>("diskscan://error", (e) => cb("error", e.payload)),
    listen<ScanSnapshot>("diskscan://canceled", (e) =>
      cb("canceled", e.payload)
    ),
  ]);
  return () => {
    unsubs.forEach((u) => u());
  };
}

/** Directory-diff lifecycle. `progress` carries live file counts + phase; the
 *  terminal events (`done`/`error`/`canceled`) carry a full DiffSnapshot. */
export async function onDiffEvent(
  cb: (
    kind: "progress" | "done" | "error" | "canceled",
    payload: DiffProgress | DiffSnapshot
  ) => void
): Promise<UnlistenFn> {
  const unsubs = await Promise.all([
    listen<DiffProgress>("diff://progress", (e) => cb("progress", e.payload)),
    listen<DiffSnapshot>("diff://done", (e) => cb("done", e.payload)),
    listen<DiffSnapshot>("diff://error", (e) => cb("error", e.payload)),
    listen<DiffSnapshot>("diff://canceled", (e) => cb("canceled", e.payload)),
  ]);
  return () => {
    unsubs.forEach((u) => u());
  };
}

/** Dedupe lifecycle. `progress` carries live counts + phase; the terminal
 *  events (`done`/`error`/`canceled`) carry a full DedupeSnapshot. */
export async function onDedupeEvent(
  cb: (
    kind: "progress" | "done" | "error" | "canceled",
    payload: DedupeProgress | DedupeSnapshot
  ) => void
): Promise<UnlistenFn> {
  const unsubs = await Promise.all([
    listen<DedupeProgress>("dedupe://progress", (e) => cb("progress", e.payload)),
    listen<DedupeSnapshot>("dedupe://done", (e) => cb("done", e.payload)),
    listen<DedupeSnapshot>("dedupe://error", (e) => cb("error", e.payload)),
    listen<DedupeSnapshot>("dedupe://canceled", (e) => cb("canceled", e.payload)),
  ]);
  return () => {
    unsubs.forEach((u) => u());
  };
}

/** Fleet-search lifecycle. `progress` carries live counts + strategy, `hit`
 *  streams batches of new hits, and the terminal events (`done`/`error`/
 *  `canceled`) carry a full SearchSnapshot (fetch the hit list on `done`). */
export async function onSearchEvent(
  cb: (
    kind: "progress" | "hit" | "done" | "error" | "canceled",
    payload: SearchProgress | SearchHitBatch | SearchSnapshot
  ) => void
): Promise<UnlistenFn> {
  const unsubs = await Promise.all([
    listen<SearchProgress>("search://progress", (e) => cb("progress", e.payload)),
    listen<SearchHitBatch>("search://hit", (e) => cb("hit", e.payload)),
    listen<SearchSnapshot>("search://done", (e) => cb("done", e.payload)),
    listen<SearchSnapshot>("search://error", (e) => cb("error", e.payload)),
    listen<SearchSnapshot>("search://canceled", (e) => cb("canceled", e.payload)),
  ]);
  return () => {
    unsubs.forEach((u) => u());
  };
}

export async function onTransferEvent(
  cb: (
    kind: "added" | "progress" | "done" | "error" | "updated",
    t: Transfer
  ) => void
): Promise<UnlistenFn> {
  const unsubs = await Promise.all([
    listen<Transfer>("transfer://added", (e) => cb("added", e.payload)),
    listen<Transfer>("transfer://progress", (e) => cb("progress", e.payload)),
    listen<Transfer>("transfer://done", (e) => cb("done", e.payload)),
    listen<Transfer>("transfer://error", (e) => cb("error", e.payload)),
    listen<Transfer>("transfer://updated", (e) => cb("updated", e.payload)),
  ]);
  return () => {
    unsubs.forEach((u) => u());
  };
}

export async function onTransferQueue(
  cb: (q: TransferQueueState) => void
): Promise<UnlistenFn> {
  return listen<TransferQueueState>("transfer://queue", (e) => cb(e.payload));
}
