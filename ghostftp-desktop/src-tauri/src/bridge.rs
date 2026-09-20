//! Agent Bridge — a localhost HTTP endpoint that lets a local AI agent (Claude
//! Code, Cursor, …) help the user through Ghost FTP's already-authenticated sessions,
//! without installing anything on the remote server or handing the agent any
//! credentials.
//!
//! Capabilities exposed to the agent (REST + MCP):
//!   * `exec` — run a diagnostic/status command (SSH or Ghost FTP Agent),
//!   * `list_dir` / `read_file` / `search` — inspect the remote filesystem,
//!   * `download` / `upload` / `upload_dir` — move files through Ghost FTP's
//!     transfer engine,
//!   * `sync` — one-way directory sync (plan/dry-run + gated execute),
//!   * `server_info` / `list_sessions` — context about what's connected.
//!
//! Security model:
//!   * bound to 127.0.0.1 only,
//!   * guarded by a per-launch bearer token,
//!   * every session must be explicitly opted in ("Allow agent access"),
//!   * each side-effecting request is approved interactively in the Ghost FTP UI
//!     (emit event → await oneshot → resolve), UNLESS the user has relaxed the
//!     approval policy (allow-all, auto-approve read-only ops, or auto-approve
//!     safe read-only commands). The policy + the per-profile allow-list are
//!     persisted to `bridge.json` so they survive restarts/reconnects.

use crate::AppState;
use anyhow::{anyhow, Context, Result};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::collections::{HashMap, HashSet};
use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tauri::{AppHandle, Emitter, Manager};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::{TcpListener, TcpStream};
use tokio::sync::{oneshot, Mutex};
use uuid::Uuid;

use crate::session::{ExecStream, Session};
use crate::sync::{SyncDirection, SyncStrategy};
use crate::transfer::OverwritePolicy;

const APPROVAL_TIMEOUT: Duration = Duration::from_secs(120);
const MAX_ACTIVITY: usize = 200;
const MAX_BODY: usize = 1024 * 1024; // 1 MiB request bodies
const MAX_HEADERS: usize = 64 * 1024;
const MAX_READ_FILE: usize = 256 * 1024; // cap ghostftp_read_file output at 256 KiB
const MAX_EXEC_BYTES: usize = 512 * 1024; // cap exec output at 512 KiB
const EXEC_TIMEOUT: Duration = Duration::from_secs(60); // default; kill a hung/streaming command
/// Bounds for a caller-supplied exec timeout (`timeoutMs`). The floor keeps a
/// typo like `1` from insta-killing every command; the ceiling (15 min) keeps
/// a runaway agent from parking a command forever. Shared with `ghostftp-agentd` via
/// the proto crate so the daemon and the bridge clamp to the *same* ceiling
/// (Plan 10 Phase 0e — a daemon-private 10-min cap used to silently shorten a
/// bridge-accepted 15-min timeout on paired-agent targets).
use ghostftp_agent_proto::msg::{EXEC_TIMEOUT_MS_MAX, EXEC_TIMEOUT_MS_MIN};
const SEARCH_MAX_RESULTS: usize = 200;
const SEARCH_MAX_DEPTH: usize = 6;

/// Filename of the endpoint discovery file written next to `bridge.json` while
/// the bridge is running. `ghostftp-cli agent …` reads the URL + token from it, so
/// the AI never has to handle either. Deleted when the bridge stops.
const DISCOVERY_FILE: &str = "agent-endpoint.json";

/// How much approval friction the user has opted to remove. Every field
/// defaults to `false` — i.e. the original "approve everything" behaviour.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct ApprovalPolicy {
    /// Master switch — approve every agent request on enabled sessions.
    pub allow_all: bool,
    /// Auto-approve read-only operations (list_dir, read_file, search). Note:
    /// download/upload are treated as writes (they touch a local/remote disk).
    pub auto_read: bool,
    /// Auto-approve commands that look read-only (best-effort heuristic).
    pub auto_safe_exec: bool,
    /// Answer sudo's password prompt with the connection's own password when an
    /// (approved) command runs `sudo`. Off by default; only takes effect on
    /// password-auth sessions — key/agent logins have no password to reuse.
    /// This is a capability gate, NOT an auto-approve: a sudo command still
    /// prompts for approval unless `allow_all`/`auto_safe_exec` say otherwise.
    pub allow_sudo: bool,
}

/// A user-defined, pre-approved command the agent can run by NAME. The agent
/// only ever supplies the name; the exact `command` string was written and
/// vetted by the user when they saved it, so running one skips the approval
/// prompt (it still requires the target session to be opted in). Global — a
/// `name -> command` entry runnable on any bridge-enabled SSH server. The agent
/// can list and run these over the bridge but can never create/edit/delete them
/// (that is local-UI-only), so it cannot self-author a "pre-approved" command.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SavedCommand {
    pub id: String,
    pub name: String,
    pub command: String,
    #[serde(default)]
    pub description: String,
}

fn default_true() -> bool {
    true
}

/// A saved, AI-authorable Skill (Plan 8): a named, parameterized, multi-step
/// workflow the agent (or the user) can run across one or many connected servers.
/// It's the next abstraction above a [`SavedCommand`] — where a saved command is
/// one pre-approved string runnable by name, a Skill is a sequence of shell steps
/// with `${param}` placeholders, a default target selector, and a proposal gate.
///
/// The store is deliberately linear (no branching/looping — see Plan 8 Risks):
/// each step is a shell command template run on every resolved target, reusing
/// the bridge's exec + approval machinery. Steps run in order per target; a
/// failing step halts that target's remaining steps when `stop_on_error` is set.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct Skill {
    pub id: String,
    pub name: String,
    pub description: String,
    pub params: Vec<SkillParam>,
    pub steps: Vec<SkillStep>,
    /// Default set of servers to run on; a run call may override it.
    pub targets: TargetSelector,
    /// `approved` = runnable; `proposed` = AI-authored, awaiting one human
    /// approval in Ghost FTP's Skills panel before it can run.
    pub status: SkillStatus,
    /// Provenance for the UI: `"user"` (hand-authored, born approved) or `"ai"`
    /// (proposed over the bridge).
    pub created_by: String,
    /// Halt a target's remaining steps after the first failing step. Default true.
    #[serde(default = "default_true")]
    pub stop_on_error: bool,
}

/// One named input a Skill's steps interpolate via `${name}`. Missing required
/// params fail the run before anything executes; optional params fall back to
/// `default` (or the empty string).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct SkillParam {
    pub name: String,
    pub description: String,
    pub required: bool,
    pub default: Option<String>,
}

/// One linear step of a Skill: a shell command template. `${param}` placeholders
/// are substituted at run time before the command reaches the exec path.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct SkillStep {
    /// Optional label shown in the console / audit log; falls back to the command.
    pub name: String,
    pub command: String,
}

/// Which connected servers a Skill runs on. `all` = every enabled, exec-capable
/// session (SSH or Ghost FTP Agent); otherwise the explicit `sessions` list (names or
/// ids). A run call can override this selector entirely.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct TargetSelector {
    pub all: bool,
    pub sessions: Vec<String>,
}

/// A Skill's lifecycle state. AI-authored Skills land as `Proposed` and can't run
/// until a human approves them in the Skills panel — so the agent can't
/// self-grant a destructive fleet workflow (Plan 8 Safety).
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SkillStatus {
    /// Runnable. Hand-authored Skills and approved proposals.
    #[default]
    Approved,
    /// AI-authored, awaiting one human approval before it can run.
    Proposed,
}

/// On-disk shape of `bridge.json`. Session ids are per-connect UUIDs, so the
/// allow-list is keyed by *profile* id and re-applied when a session connects.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
struct PersistedConfig {
    /// Master on/off switch. Default false (serde `default` => old configs
    /// without the key stay off), so local work exposes nothing until opted in.
    enabled: bool,
    enabled_profiles: Vec<String>,
    policy: ApprovalPolicy,
    /// User-defined pre-approved commands the agent can run by name.
    #[serde(default)]
    saved_commands: Vec<SavedCommand>,
    /// Saved Skills (Plan 8).
    #[serde(default)]
    skills: Vec<Skill>,
    /// One-time flag: existing saved commands have been seeded as single-step
    /// Skills. Prevents re-seeding on every launch.
    #[serde(default)]
    skills_migrated: bool,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct BridgeStatus {
    pub running: bool,
    /// Master switch state (persisted). When false nothing listens at all.
    pub enabled: bool,
    pub url: Option<String>,
    pub port: Option<u16>,
    pub token: Option<String>,
    pub enabled_sessions: Vec<String>,
    pub policy: ApprovalPolicy,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ActivityEntry {
    pub id: String,
    pub session_id: String,
    pub kind: String, // "exec" | "read" | "download" | "upload" | "upload_dir" | "sync" | "diff" | "search" | "denied" | "error"
    pub detail: String,
    pub ok: bool,
    pub at: i64, // unix millis
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ApprovalRequest {
    pub request_id: String,
    pub session_id: String,
    pub session_name: String,
    /// Operation kind ("exec", "read", "download", "upload", "search") so the
    /// UI can phrase the prompt appropriately.
    pub kind: String,
    /// Human-readable summary of what the agent wants to do.
    pub command: String,
}

#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ApprovalDecision {
    Approve,
    Deny,
}

/// Risk class of an operation, used to decide auto-approval.
#[derive(Clone, Copy, PartialEq, Eq)]
enum OpClass {
    Read,
    Write,
    Exec,
}

struct Running {
    port: u16,
    token: String,
    /// Path of the discovery file written on start, removed on stop.
    endpoint_path: PathBuf,
    shutdown: Option<oneshot::Sender<()>>,
}

#[derive(Default)]
pub struct BridgeState {
    running: Mutex<Option<Running>>,
    /// Master on/off switch (the persisted `enabled` key). Default false:
    /// while off the bridge never starts and no token/discovery file exists.
    enabled_master: Mutex<bool>,
    /// Runtime allow-list, keyed by session id (per-connect UUID).
    pub(crate) enabled: Mutex<HashSet<String>>,
    /// Persistent allow-list, keyed by profile id; re-applied on connect.
    enabled_profiles: Mutex<HashSet<String>>,
    policy: Mutex<ApprovalPolicy>,
    /// User-defined pre-approved commands (global name -> command list).
    saved_commands: Mutex<Vec<SavedCommand>>,
    /// Saved Skills (Plan 8): AI-authorable, multi-step, fleet-targetable.
    skills: Mutex<Vec<Skill>>,
    /// Whether saved commands have been seeded into Skills (one-time).
    skills_migrated: Mutex<bool>,
    approvals: Mutex<HashMap<String, oneshot::Sender<ApprovalDecision>>>,
    activity: Mutex<Vec<ActivityEntry>>,
    config_path: Option<PathBuf>,
    /// The frontend's currently focused session id, if any. Not persisted.
    pub(crate) active_session_id: Mutex<Option<String>>,
}

fn now_millis() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

/// Effective exec timeout for an optional caller-supplied `timeoutMs`:
/// absent → the 60 s default; present → clamped to [1 s, 15 min].
fn exec_timeout_from(timeout_ms: Option<u64>) -> Duration {
    match timeout_ms {
        Some(ms) => Duration::from_millis(ms.clamp(EXEC_TIMEOUT_MS_MIN, EXEC_TIMEOUT_MS_MAX)),
        None => EXEC_TIMEOUT,
    }
}

fn activity(kind: &str, session_id: &str, detail: String, ok: bool) -> ActivityEntry {
    ActivityEntry {
        id: Uuid::new_v4().to_string(),
        session_id: session_id.to_string(),
        kind: kind.to_string(),
        detail,
        ok,
        at: now_millis(),
    }
}

/// Publish the live endpoint (`{url, port, token, pid, version}`) so the CLI can
/// reach the bridge without the user or the agent ever copying a URL or token.
/// On Unix the file is locked down to the owner (0600); on Windows it inherits
/// the per-user `%APPDATA%` ACL — the same trust boundary `profiles.json`
/// already relies on (any process running as this user can read it).
fn write_discovery_file(path: &PathBuf, port: u16, token: &str) {
    let body = json!({
        "url": format!("http://127.0.0.1:{port}"),
        "port": port,
        "token": token,
        "pid": std::process::id(),
        "version": env!("CARGO_PKG_VERSION"),
    });
    let Ok(bytes) = serde_json::to_vec_pretty(&body) else {
        return;
    };
    if std::fs::write(path, &bytes).is_ok() {
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600));
        }
    }
}

/// Best-effort classifier: is this shell command read-only? Conservative by
/// design — anything with shell chaining/redirection, a known command-runner
/// (which could exec an arbitrary mutating command via args alone), or a binary
/// not on the curated safe list, is treated as *not* read-only (so it still
/// needs approval unless allow-all is on). This is a heuristic, never a security
/// boundary — the binary+args space is too large to fully reason about, so we
/// only auto-approve commands we're confident neither mutate state nor launch
/// another program. Anything else falls back to interactive approval.
fn is_read_only_command(cmd: &str) -> bool {
    let trimmed = cmd.trim();
    if trimmed.is_empty() {
        return false;
    }
    // Reject anything that could chain/redirect/substitute into a mutating command.
    const DANGEROUS: &[&str] = &[
        ";", "&&", "||", "|", ">", "<", "`", "$(", "${", "&", "\n", "\r",
    ];
    for d in DANGEROUS {
        if trimmed.contains(d) {
            return false;
        }
    }
    let first = trimmed.split_whitespace().next().unwrap_or("");
    let bin = first.rsplit('/').next().unwrap_or(first);
    // Binaries that run another program or mutate system/network state using
    // only flags/args (no shell metacharacters) — e.g. `env CMD`, `ip netns
    // exec <ns> CMD`, `date -s`, `dmesg -C`. These can never be auto-approved.
    const NEVER_SAFE: &[&str] = &[
        "env",
        "sudo",
        "doas",
        "su",
        "xargs",
        "watch",
        "timeout",
        "nohup",
        "nice",
        "ionice",
        "time",
        "ssh",
        "scp",
        "rsync",
        "sh",
        "bash",
        "zsh",
        "dash",
        "ksh",
        "fish",
        "perl",
        "python",
        "python3",
        "ruby",
        "node",
        "php",
        "lua",
        "eval",
        "exec",
        "find",
        "ip",
        "ifconfig",
        "iptables",
        "nft",
        "date",
        "hostname",
        "hostnamectl",
        "dmesg",
        "ss",
        "systemctl",
        "service",
        "kill",
        "pkill",
        "killall",
        "mount",
        "umount",
        "modprobe",
        "sysctl",
        "tee",
        "dd",
        "crontab",
    ];
    if NEVER_SAFE.contains(&bin) {
        return false;
    }
    // Curated binaries that neither mutate state nor launch other programs with
    // their normal invocations. Deliberately excludes find/sed/awk/sort/uniq
    // (write/exec/-i/-o flags) and anything in NEVER_SAFE.
    const READ_ONLY: &[&str] = &[
        "ls",
        "cat",
        "pwd",
        "whoami",
        "id",
        "uname",
        "uptime",
        "df",
        "du",
        "free",
        "ps",
        "stat",
        "head",
        "tail",
        "wc",
        "file",
        "echo",
        "printenv",
        "which",
        "type",
        "tree",
        "readlink",
        "realpath",
        "dirname",
        "basename",
        "lsblk",
        "lscpu",
        "netstat",
        "cut",
        "grep",
        "egrep",
        "fgrep",
        "md5sum",
        "sha1sum",
        "sha256sum",
        "groups",
        "getent",
        "lsof",
        "nproc",
        "vmstat",
        "cal",
        "arch",
        "lsusb",
        "lspci",
        "uptime",
    ];
    READ_ONLY.contains(&bin)
}

/// Best-effort: does this command invoke `sudo`? Splits on shell word
/// boundaries so `sudo …`, `cmd | sudo …`, `;sudo`, `$(sudo …)` all match.
/// Used only to decide whether to feed the connection password; never a
/// security boundary. We deliberately don't special-case `sudo -n`: priming
/// the credential cache simply makes a later `sudo -n` succeed too, which is
/// the desired outcome, and a blanket `-n` check would mis-fire on unrelated
/// flags like `grep -n`.
fn command_uses_sudo(cmd: &str) -> bool {
    let is_sep = |c: char| c.is_whitespace() || "|&;()<>".contains(c);
    cmd.split(is_sep)
        .any(|tok| tok.rsplit('/').next().unwrap_or(tok) == "sudo")
}

/// The reusable login password for a session, if it authenticated with one.
/// Key/agent logins return None (nothing to hand to sudo).
fn connection_password(ssh: &crate::session::SshSession) -> Option<String> {
    match &ssh.profile.auth {
        crate::profiles::AuthMethod::Password { password } => Some(password.clone()),
        _ => None,
    }
}

impl BridgeState {
    pub fn new() -> Self {
        Self::default()
    }

    /// Load (or initialise) bridge config from the app data dir. Mirrors
    /// `ProfileStore::load_or_create`. Falls back to defaults on any error.
    pub fn load_or_create(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).ok();
        let path = dir.join("bridge.json");
        let mut cfg: PersistedConfig = if path.exists() {
            std::fs::read(&path)
                .ok()
                .and_then(|b| serde_json::from_slice(&b).ok())
                .unwrap_or_default()
        } else {
            PersistedConfig::default()
        };
        // One-time migration: seed each existing saved command as a single-step,
        // approved Skill so the shipped saved-commands survive into the Skills
        // store. Non-destructive — the saved commands themselves are left intact
        // for `ghostftp_run_command`. Guarded so it runs at most once.
        if !cfg.skills_migrated {
            for c in &cfg.saved_commands {
                cfg.skills.push(migrate_command_to_skill(c));
            }
            cfg.skills_migrated = true;
            if let Ok(bytes) = serde_json::to_vec_pretty(&cfg) {
                let _ = std::fs::write(&path, bytes);
            }
        }
        Ok(Self {
            enabled_master: Mutex::new(cfg.enabled),
            enabled_profiles: Mutex::new(cfg.enabled_profiles.into_iter().collect()),
            policy: Mutex::new(cfg.policy),
            saved_commands: Mutex::new(cfg.saved_commands),
            skills: Mutex::new(cfg.skills),
            skills_migrated: Mutex::new(cfg.skills_migrated),
            config_path: Some(path),
            ..Default::default()
        })
    }

    async fn persist(&self) {
        let Some(path) = self.config_path.as_ref() else {
            return;
        };
        let cfg = PersistedConfig {
            enabled: *self.enabled_master.lock().await,
            enabled_profiles: self.enabled_profiles.lock().await.iter().cloned().collect(),
            policy: *self.policy.lock().await,
            saved_commands: self.saved_commands.lock().await.clone(),
            skills: self.skills.lock().await.clone(),
            skills_migrated: *self.skills_migrated.lock().await,
        };
        if let Ok(bytes) = serde_json::to_vec_pretty(&cfg) {
            let _ = std::fs::write(path, bytes);
        }
    }

    pub async fn status(&self) -> BridgeStatus {
        let running = self.running.lock().await;
        let enabled = *self.enabled_master.lock().await;
        let enabled_sessions = self.enabled.lock().await.iter().cloned().collect();
        let policy = *self.policy.lock().await;
        match running.as_ref() {
            Some(r) => BridgeStatus {
                running: true,
                enabled,
                url: Some(format!("http://127.0.0.1:{}", r.port)),
                port: Some(r.port),
                token: Some(r.token.clone()),
                enabled_sessions,
                policy,
            },
            None => BridgeStatus {
                running: false,
                enabled,
                url: None,
                port: None,
                token: None,
                enabled_sessions,
                policy,
            },
        }
    }

    pub async fn start(self: &Arc<Self>, app: AppHandle) -> Result<BridgeStatus> {
        if self.running.lock().await.is_some() {
            return Ok(self.status().await);
        }
        let listener = TcpListener::bind(("127.0.0.1", 0)).await?;
        let port = listener.local_addr()?.port();
        let token = format!("{}{}", Uuid::new_v4().simple(), Uuid::new_v4().simple());

        // Publish the live endpoint so `ghostftp-cli agent …` can find it with no
        // URL/token handling by the user or the agent. Re-resolve the data dir
        // from `app` here (not from config_path) so it works even on the
        // load_or_create default-fallback path where config_path is None.
        let dir = app
            .path()
            .app_data_dir()
            .unwrap_or_else(|_| PathBuf::from("."));
        std::fs::create_dir_all(&dir).ok();
        let endpoint_path = dir.join(DISCOVERY_FILE);
        write_discovery_file(&endpoint_path, port, &token);

        let (tx, rx) = oneshot::channel();

        let state = self.clone();
        let app_for_task = app.clone();
        let token_for_task = token.clone();
        tokio::spawn(async move {
            serve(app_for_task, state, listener, token_for_task, rx).await;
        });

        *self.running.lock().await = Some(Running {
            port,
            token,
            endpoint_path,
            shutdown: Some(tx),
        });
        Ok(self.status().await)
    }

    pub async fn stop(&self) {
        if let Some(mut r) = self.running.lock().await.take() {
            // Pull the published endpoint so the CLI immediately sees "not
            // running" instead of a stale URL/token.
            let _ = std::fs::remove_file(&r.endpoint_path);
            if let Some(tx) = r.shutdown.take() {
                let _ = tx.send(());
            }
        }
    }

    /// The master on/off switch the UI exposes (persisted, default off). Turning
    /// it on starts the bridge (binding a port, minting a token, publishing the
    /// discovery file) and makes it auto-start on the next launch; turning it
    /// off stops the bridge and removes the token/discovery file, so local work
    /// exposes nothing. Per-session grants are left intact across a toggle.
    pub async fn set_enabled(self: &Arc<Self>, app: AppHandle, on: bool) -> Result<BridgeStatus> {
        *self.enabled_master.lock().await = on;
        self.persist().await;
        if on {
            self.start(app).await?;
        } else {
            self.stop().await;
        }
        Ok(self.status().await)
    }

    /// Called once on launch: bring the bridge up only if the user left the
    /// master switch on, so the AI path survives app restarts.
    pub async fn auto_start_if_enabled(self: &Arc<Self>, app: AppHandle) {
        if *self.enabled_master.lock().await {
            if let Err(e) = self.start(app).await {
                tracing::warn!(?e, "agent bridge auto-start failed");
            }
        }
    }

    /// Grant/revoke agent access for a session. Mirrors the grant onto the
    /// persistent, profile-keyed allow-list so it survives reconnects.
    pub async fn set_access(&self, app: &AppHandle, session_id: &str, enabled: bool) {
        {
            let mut set = self.enabled.lock().await;
            if enabled {
                set.insert(session_id.to_string());
            } else {
                set.remove(session_id);
            }
        }
        if let Some(sess) = app.state::<AppState>().sessions.get(session_id).await {
            let profile_id = sess.profile().id.clone();
            let mut profs = self.enabled_profiles.lock().await;
            if enabled {
                profs.insert(profile_id);
            } else {
                profs.remove(&profile_id);
            }
        }
        self.persist().await;
    }

    /// Called when a session connects: auto-enable it if its profile was
    /// previously granted access.
    pub async fn on_session_connected(&self, session_id: &str, profile_id: &str) {
        if self.enabled_profiles.lock().await.contains(profile_id) {
            self.enabled.lock().await.insert(session_id.to_string());
        }
    }

    pub async fn set_policy(&self, policy: ApprovalPolicy) {
        *self.policy.lock().await = policy;
        self.persist().await;
    }

    pub async fn set_active_session(&self, session_id: Option<String>) {
        *self.active_session_id.lock().await = session_id;
    }

    async fn is_enabled(&self, session_id: &str) -> bool {
        self.enabled.lock().await.contains(session_id)
    }

    /// Decide whether the current policy auto-approves an operation.
    async fn auto_approved(&self, class: OpClass, exec_cmd: Option<&str>) -> bool {
        let p = *self.policy.lock().await;
        if p.allow_all {
            return true;
        }
        match class {
            OpClass::Read => p.auto_read,
            OpClass::Write => false,
            OpClass::Exec => {
                p.auto_safe_exec && exec_cmd.map(is_read_only_command).unwrap_or(false)
            }
        }
    }

    pub async fn resolve_approval(
        &self,
        request_id: &str,
        decision: ApprovalDecision,
    ) -> Result<()> {
        let tx = self
            .approvals
            .lock()
            .await
            .remove(request_id)
            .ok_or_else(|| anyhow!("no pending approval {request_id}"))?;
        tx.send(decision)
            .map_err(|_| anyhow!("approval receiver dropped"))?;
        Ok(())
    }

    /// Emit a `bridge://approval` event and block until the user clicks
    /// Approve/Deny (or it times out). Returns true only on explicit approval.
    async fn request_approval(
        &self,
        app: &AppHandle,
        session_id: &str,
        session_name: &str,
        kind: &str,
        summary: &str,
    ) -> bool {
        let request_id = Uuid::new_v4().to_string();
        let (tx, rx) = oneshot::channel();
        self.approvals.lock().await.insert(request_id.clone(), tx);
        let _ = app.emit(
            "bridge://approval",
            ApprovalRequest {
                request_id: request_id.clone(),
                session_id: session_id.to_string(),
                session_name: session_name.to_string(),
                kind: kind.to_string(),
                command: summary.to_string(),
            },
        );
        let approved = matches!(
            tokio::time::timeout(APPROVAL_TIMEOUT, rx).await,
            Ok(Ok(ApprovalDecision::Approve))
        );
        // If it timed out, the entry is still in the map — clean it up.
        self.approvals.lock().await.remove(&request_id);
        approved
    }

    async fn log(&self, app: &AppHandle, entry: ActivityEntry) {
        {
            let mut v = self.activity.lock().await;
            v.push(entry.clone());
            let len = v.len();
            if len > MAX_ACTIVITY {
                v.drain(0..len - MAX_ACTIVITY);
            }
        }
        let _ = app.emit("bridge://activity", entry);
    }

    pub async fn recent_activity(&self) -> Vec<ActivityEntry> {
        self.activity.lock().await.clone()
    }

    /// Clear the in-memory activity log (the History panel's Clear button).
    pub async fn clear_activity(&self) {
        self.activity.lock().await.clear();
    }

    // ---- Saved commands (pre-approved, local-UI-managed) ----

    pub async fn list_commands(&self) -> Vec<SavedCommand> {
        self.saved_commands.lock().await.clone()
    }

    /// Insert or update a saved command (keyed by id; a blank id mints a fresh
    /// uuid). Returns the full updated list. Reachable only from the local Ghost FTP
    /// UI (a Tauri command), never over the bridge — so the agent can't author a
    /// "pre-approved" command.
    pub async fn upsert_command(&self, mut cmd: SavedCommand) -> Vec<SavedCommand> {
        if cmd.id.trim().is_empty() {
            cmd.id = Uuid::new_v4().to_string();
        }
        {
            let mut list = self.saved_commands.lock().await;
            if let Some(existing) = list.iter_mut().find(|c| c.id == cmd.id) {
                *existing = cmd;
            } else {
                list.push(cmd);
            }
        }
        self.persist().await;
        self.saved_commands.lock().await.clone()
    }

    pub async fn delete_command(&self, id: &str) -> Vec<SavedCommand> {
        self.saved_commands.lock().await.retain(|c| c.id != id);
        self.persist().await;
        self.saved_commands.lock().await.clone()
    }

    /// Find a saved command by name (case-insensitive, first match).
    async fn find_command(&self, name: &str) -> Option<SavedCommand> {
        let want = name.trim().to_lowercase();
        self.saved_commands
            .lock()
            .await
            .iter()
            .find(|c| c.name.trim().to_lowercase() == want)
            .cloned()
    }

    // ---- Skills (Plan 8): parameterized, fleet-targetable, AI-authorable) ----

    pub async fn list_skills(&self) -> Vec<Skill> {
        self.skills.lock().await.clone()
    }

    /// Insert or update a Skill (keyed by id; a blank id mints a uuid). Local-UI
    /// only (a Tauri command) — the bridge path is [`Self::propose_skill`], which
    /// forces a proposal. A hand-authored Skill is born `Approved`; editing one
    /// preserves whatever status it already had unless the caller changed it.
    pub async fn upsert_skill(&self, mut skill: Skill) -> Vec<Skill> {
        if skill.id.trim().is_empty() {
            skill.id = Uuid::new_v4().to_string();
        }
        if skill.created_by.trim().is_empty() {
            skill.created_by = "user".into();
        }
        {
            let mut list = self.skills.lock().await;
            if let Some(existing) = list.iter_mut().find(|s| s.id == skill.id) {
                *existing = skill;
            } else {
                list.push(skill);
            }
        }
        self.persist().await;
        self.skills.lock().await.clone()
    }

    pub async fn delete_skill(&self, id: &str) -> Vec<Skill> {
        self.skills.lock().await.retain(|s| s.id != id);
        self.persist().await;
        self.skills.lock().await.clone()
    }

    /// Approve a proposed Skill (local-UI only) — the one human gate that makes an
    /// AI-authored Skill runnable.
    pub async fn approve_skill(&self, id: &str) -> Vec<Skill> {
        {
            let mut list = self.skills.lock().await;
            if let Some(s) = list.iter_mut().find(|s| s.id == id) {
                s.status = SkillStatus::Approved;
            }
        }
        self.persist().await;
        self.skills.lock().await.clone()
    }

    /// Save an AI-authored Skill over the bridge. Always forced to `Proposed` /
    /// `created_by = "ai"` with a fresh id, so the agent can neither approve its
    /// own workflow nor overwrite an existing (approved) Skill. Returns the saved
    /// proposal.
    pub async fn propose_skill(&self, mut skill: Skill) -> Skill {
        skill.id = Uuid::new_v4().to_string();
        skill.status = SkillStatus::Proposed;
        skill.created_by = "ai".into();
        self.skills.lock().await.push(skill.clone());
        self.persist().await;
        skill
    }

    /// Find a Skill by id or name (case-insensitive name, first match).
    async fn find_skill(&self, name_or_id: &str) -> Option<Skill> {
        let want = name_or_id.trim().to_lowercase();
        self.skills
            .lock()
            .await
            .iter()
            .find(|s| s.id == name_or_id || s.name.trim().to_lowercase() == want)
            .cloned()
    }
}

/// Seed a saved command as a single-step, approved Skill (one-time migration).
fn migrate_command_to_skill(c: &SavedCommand) -> Skill {
    Skill {
        id: Uuid::new_v4().to_string(),
        name: c.name.clone(),
        description: if c.description.trim().is_empty() {
            "Migrated from a saved command.".to_string()
        } else {
            c.description.clone()
        },
        params: Vec::new(),
        steps: vec![SkillStep {
            name: String::new(),
            command: c.command.clone(),
        }],
        targets: TargetSelector::default(),
        status: SkillStatus::Approved,
        created_by: "user".into(),
        stop_on_error: true,
    }
}

/// Opt-in check → policy/approval gate. On success the caller proceeds; on
/// failure it returns the contained `(status, json)` response. Denials are
/// logged here so callers only log their own success/error paths.
// These fields are deliberately kept explicit at the authorization boundary.
#[allow(clippy::too_many_arguments)]
async fn gate(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    session_name: &str,
    class: OpClass,
    kind: &str,
    summary: &str,
    exec_cmd: Option<&str>,
) -> Result<(), (u16, Value)> {
    if !state.is_enabled(session_id).await {
        return Err((
            403,
            json!({
                "error": format!(
                    "connection '{}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.",
                    session_name
                )
            }),
        ));
    }
    // Proactively heal a transport that died while the session sat idle, so the
    // op below runs on a live connection instead of eating the first failure.
    // SSH-only, best-effort — a reconnect failure surfaces as the op's own error.
    if let Some(ssh) = app.state::<AppState>().sessions.get_ssh(session_id).await {
        let _ = ssh.ensure_alive().await;
    }
    if state.auto_approved(class, exec_cmd).await {
        return Ok(());
    }
    if state
        .request_approval(app, session_id, session_name, kind, summary)
        .await
    {
        Ok(())
    } else {
        state
            .log(
                app,
                activity("denied", session_id, summary.to_string(), false),
            )
            .await;
        Err((
            403,
            json!({"error": "the user denied or did not respond to the approval prompt in Ghost FTP. Ask before trying again."}),
        ))
    }
}

// ---- HTTP server (hand-rolled, localhost-only, one request per connection) ----

async fn serve(
    app: AppHandle,
    state: Arc<BridgeState>,
    listener: TcpListener,
    token: String,
    mut shutdown: oneshot::Receiver<()>,
) {
    loop {
        tokio::select! {
            _ = &mut shutdown => break,
            accepted = listener.accept() => {
                let Ok((stream, _peer)) = accepted else { continue };
                let app = app.clone();
                let state = state.clone();
                let token = token.clone();
                tokio::spawn(async move {
                    let _ = handle_conn(app, state, stream, token).await;
                });
            }
        }
    }
}

struct Request {
    method: String,
    path: String,
    auth: Option<String>,
    body: Vec<u8>,
}

fn find_subslice(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack.windows(needle.len()).position(|w| w == needle)
}

async fn read_request(stream: &mut TcpStream) -> Result<Request> {
    let mut buf = Vec::new();
    let mut tmp = [0u8; 4096];
    let header_end = loop {
        if let Some(pos) = find_subslice(&buf, b"\r\n\r\n") {
            break pos + 4;
        }
        if buf.len() > MAX_HEADERS {
            return Err(anyhow!("request headers too large"));
        }
        let n = stream.read(&mut tmp).await?;
        if n == 0 {
            return Err(anyhow!("connection closed before headers"));
        }
        buf.extend_from_slice(&tmp[..n]);
    };

    let header_text = String::from_utf8_lossy(&buf[..header_end]).to_string();
    let mut lines = header_text.split("\r\n");
    let request_line = lines.next().unwrap_or("");
    let mut parts = request_line.split_whitespace();
    let method = parts.next().unwrap_or("").to_string();
    let path = parts.next().unwrap_or("").to_string();

    let mut content_length = 0usize;
    let mut auth = None;
    for line in lines {
        if let Some((k, v)) = line.split_once(':') {
            let key = k.trim().to_ascii_lowercase();
            let val = v.trim();
            if key == "content-length" {
                content_length = val.parse().unwrap_or(0);
            } else if key == "authorization" {
                auth = Some(val.to_string());
            }
        }
    }
    if content_length > MAX_BODY {
        return Err(anyhow!("request body too large"));
    }

    let mut body = buf[header_end..].to_vec();
    while body.len() < content_length {
        let n = stream.read(&mut tmp).await?;
        if n == 0 {
            break;
        }
        body.extend_from_slice(&tmp[..n]);
    }
    body.truncate(content_length);

    Ok(Request {
        method,
        path,
        auth,
        body,
    })
}

async fn write_response(stream: &mut TcpStream, status: u16, body: &Value) -> Result<()> {
    let reason = match status {
        200 => "OK",
        400 => "Bad Request",
        401 => "Unauthorized",
        403 => "Forbidden",
        404 => "Not Found",
        _ => "Internal Server Error",
    };
    let payload = serde_json::to_vec(body)?;
    let head = format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nAccess-Control-Allow-Origin: *\r\nConnection: close\r\n\r\n",
        payload.len()
    );
    stream.write_all(head.as_bytes()).await?;
    stream.write_all(&payload).await?;
    stream.flush().await?;
    Ok(())
}

async fn handle_conn(
    app: AppHandle,
    state: Arc<BridgeState>,
    mut stream: TcpStream,
    token: String,
) -> Result<()> {
    let req = match read_request(&mut stream).await {
        Ok(r) => r,
        Err(_) => {
            return write_response(&mut stream, 400, &json!({"error": "bad request"})).await;
        }
    };

    let authed = req
        .auth
        .as_deref()
        .and_then(|a| a.strip_prefix("Bearer "))
        .map(|t| t == token)
        .unwrap_or(false);
    if !authed {
        return write_response(&mut stream, 401, &json!({"error": "unauthorized"})).await;
    }

    // MCP (Streamable HTTP) lives on /mcp and manages its own JSON-RPC framing.
    if req.path == "/mcp" {
        if req.method == "POST" {
            return handle_mcp(&app, &state, &req.body, &mut stream).await;
        }
        // The optional GET server→client SSE stream is unsupported; clients
        // fall back gracefully when it returns 405.
        return write_empty(&mut stream, 405).await;
    }

    let (status, body) = route(&app, &state, &req).await;
    write_response(&mut stream, status, &body).await
}

async fn route(app: &AppHandle, state: &Arc<BridgeState>, req: &Request) -> (u16, Value) {
    match (req.method.as_str(), req.path.as_str()) {
        ("GET", "/health") => (
            200,
            json!({"ok": true, "name": "ghostftp-agent-bridge", "version": env!("CARGO_PKG_VERSION")}),
        ),
        ("GET", "/context") => op_context(app, state).await,
        ("GET", "/sessions") => handle_sessions(app, state).await,
        ("POST", "/exec") => handle_exec(app, state, &req.body).await,
        ("POST", "/exec_script") => handle_exec_script(app, state, &req.body).await,
        ("POST", "/job") => handle_job(app, state, &req.body).await,
        ("POST", "/jobs") => handle_jobs(app, state, &req.body).await,
        ("POST", "/write") => handle_write(app, state, &req.body).await,
        ("POST", "/list") => handle_list(app, state, &req.body).await,
        ("POST", "/read") => handle_read(app, state, &req.body).await,
        ("POST", "/download") => handle_download(app, state, &req.body).await,
        ("POST", "/delete") => handle_delete(app, state, &req.body).await,
        ("POST", "/rename") => handle_rename(app, state, &req.body).await,
        ("POST", "/mkdir") => handle_mkdir(app, state, &req.body).await,
        ("POST", "/upload") => handle_upload(app, state, &req.body).await,
        ("POST", "/upload_dir") => handle_upload_dir(app, state, &req.body).await,
        ("POST", "/sync") => handle_sync(app, state, &req.body).await,
        ("POST", "/diff") => handle_diff(app, state, &req.body).await,
        ("POST", "/search") => handle_search(app, state, &req.body).await,
        ("POST", "/read_batch") => handle_read_batch(app, state, &req.body).await,
        ("POST", "/glob") => handle_glob(app, state, &req.body).await,
        ("POST", "/tail") => handle_tail(app, state, &req.body).await,
        ("POST", "/info") => handle_info(app, state, &req.body).await,
        ("POST", "/transfer") => handle_transfer_status(app, &req.body).await,
        ("POST", "/history") => handle_history(app, state, &req.body).await,
        ("GET", "/commands") => (200, json!({ "commands": state.list_commands().await })),
        ("POST", "/run") => handle_run(app, state, &req.body).await,
        ("GET", "/skills") => (200, json!({ "skills": state.list_skills().await })),
        ("POST", "/skill_run") => handle_skill_run(app, state, &req.body).await,
        _ => (404, json!({"error": "not found"})),
    }
}

async fn handle_sessions(app: &AppHandle, state: &Arc<BridgeState>) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let enabled: Vec<String> = state.enabled.lock().await.iter().cloned().collect();
    let mut out = Vec::new();
    for id in enabled {
        if let Some(sess) = manager.get(&id).await {
            let p = sess.profile();
            out.push(json!({
                "id": id,
                "name": p.name,
                "host": p.host,
                "protocol": sess.protocol(),
                "canExec": matches!(&*sess, Session::Ssh(_) | Session::Agent(_)),
            }));
        }
    }
    (200, json!({ "sessions": out }))
}

fn body_str(v: &Value, key: &str) -> String {
    v.get(key)
        .and_then(|x| x.as_str())
        .unwrap_or("")
        .to_string()
}

fn parse_body(body: &[u8]) -> Result<Value, (u16, Value)> {
    serde_json::from_slice(body).map_err(|_| (400, json!({"error": "invalid JSON body"})))
}

async fn handle_exec(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let command = body_str(&parsed, "command");
    if session_id.is_empty() || command.is_empty() {
        return (400, json!({"error": "sessionId and command are required"}));
    }
    let dry_run = parsed
        .get("dryRun")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    let timeout_ms = parsed.get("timeoutMs").and_then(|v| v.as_u64());
    // A detached exec returns a job id at once instead of blocking (Plan 10 Phase 4).
    if parsed
        .get("detach")
        .and_then(|v| v.as_bool())
        .unwrap_or(false)
    {
        return op_exec_detached(app, state, &session_id, &command, dry_run).await;
    }
    exec_on(app, state, &session_id, &command, dry_run, timeout_ms).await
}

async fn handle_job(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let job_id = body_str(&parsed, "jobId");
    if session_id.is_empty() || job_id.is_empty() {
        return (400, json!({"error": "sessionId and jobId are required"}));
    }
    op_job(app, state, &session_id, &job_id).await
}

async fn handle_jobs(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    if session_id.is_empty() {
        return (400, json!({"error": "sessionId is required"}));
    }
    op_jobs(app, state, &session_id).await
}

/// `/exec_script` — run a whole local script (Plan 10 Phase 1). The body carries
/// the script as base64 (`script`) so any bytes survive the JSON/HTTP hop
/// untouched; `label` is a friendly name for the console/audit (e.g. the source
/// filename). Decodes, then runs it verbatim via `exec_script_on`.
async fn handle_exec_script(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    body: &[u8],
) -> (u16, Value) {
    use base64::Engine as _;
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let script_b64 = body_str(&parsed, "script");
    if session_id.is_empty() || script_b64.is_empty() {
        return (
            400,
            json!({"error": "sessionId and script (base64) are required"}),
        );
    }
    let Ok(bytes) = base64::engine::general_purpose::STANDARD.decode(script_b64.as_bytes()) else {
        return (400, json!({"error": "script must be valid base64"}));
    };
    let Ok(script) = String::from_utf8(bytes) else {
        return (400, json!({"error": "script bytes are not valid UTF-8"}));
    };
    let mut label = body_str(&parsed, "label");
    if label.is_empty() {
        label = "a script".to_string();
    }
    let dry_run = parsed
        .get("dryRun")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    let timeout_ms = parsed.get("timeoutMs").and_then(|v| v.as_u64());
    exec_script_on(
        app,
        state,
        &session_id,
        &script,
        &label,
        dry_run,
        timeout_ms,
    )
    .await
}

/// `/write` — drop text/bytes straight into a remote file (Plan 10 Phase 2). The
/// body carries the content as base64 (`content`) so any bytes survive the hop;
/// `overwrite` (default false) controls whether an existing file is replaced.
/// This is the direct alternative to staging a local file and `upload`-ing it —
/// no mangling-prone path, one round-trip.
async fn handle_write(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    use base64::Engine as _;
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    let content_b64 = body_str(&parsed, "content");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    let Ok(bytes) = base64::engine::general_purpose::STANDARD.decode(content_b64.as_bytes()) else {
        return (400, json!({"error": "content must be valid base64"}));
    };
    let overwrite = parsed
        .get("overwrite")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_write(app, state, &session_id, &path, &bytes, overwrite).await
}

async fn handle_delete(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    let recursive = parsed
        .get("recursive")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_delete(app, state, &session_id, &path, recursive).await
}

async fn handle_rename(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let from = body_str(&parsed, "from");
    let to = body_str(&parsed, "to");
    if session_id.is_empty() || from.is_empty() || to.is_empty() {
        return (400, json!({"error": "sessionId, from and to are required"}));
    }
    op_rename(app, state, &session_id, &from, &to).await
}

async fn handle_mkdir(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    op_mkdir(app, state, &session_id, &path).await
}

async fn handle_run(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let name = body_str(&parsed, "name");
    if session_id.is_empty() || name.is_empty() {
        return (400, json!({"error": "sessionId and name are required"}));
    }
    let dry_run = parsed
        .get("dryRun")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    let timeout_ms = parsed.get("timeoutMs").and_then(|v| v.as_u64());
    op_run_command(app, state, &session_id, &name, dry_run, timeout_ms).await
}

/// Extract a `{ "k": "v", ... }` object into a String map, coercing scalar values
/// (number/bool) to their string form. Shared by the `/skill_run` route and the
/// `ghostftp_run_skill` MCP tool.
fn params_from_value(v: Option<&Value>) -> HashMap<String, String> {
    v.and_then(|x| x.as_object())
        .map(|o| {
            o.iter()
                .filter_map(|(k, val)| {
                    let s = match val {
                        Value::String(s) => s.clone(),
                        Value::Number(n) => n.to_string(),
                        Value::Bool(b) => b.to_string(),
                        _ => return None,
                    };
                    Some((k.clone(), s))
                })
                .collect()
        })
        .unwrap_or_default()
}

/// Extract a JSON string array (e.g. a `targets` list). `None` means "omitted"
/// (fall back to the skill's default selector).
fn str_array(v: Option<&Value>) -> Option<Vec<String>> {
    v.and_then(|x| x.as_array()).map(|a| {
        a.iter()
            .filter_map(|x| x.as_str().map(String::from))
            .collect()
    })
}

async fn handle_skill_run(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let name = body_str(&parsed, "name");
    if name.is_empty() {
        return (400, json!({"error": "name is required"}));
    }
    let params = params_from_value(parsed.get("params"));
    let targets = str_array(parsed.get("targets"));
    let dry_run = parsed
        .get("dryRun")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_run_skill(app, state, &name, params, targets, dry_run).await
}

async fn handle_list(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    if session_id.is_empty() {
        return (400, json!({"error": "sessionId is required"}));
    }
    let path = match body_str(&parsed, "path") {
        p if p.is_empty() => ".".to_string(),
        p => p,
    };
    op_list_dir(app, state, &session_id, &path).await
}

async fn handle_read(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    op_read_file(app, state, &session_id, &path).await
}

async fn handle_download(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    let local_dir = parsed
        .get("localDir")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string());
    op_download(app, state, &session_id, &path, local_dir).await
}

async fn handle_upload(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let local_path = body_str(&parsed, "localPath");
    let remote_dir = body_str(&parsed, "remoteDir");
    if session_id.is_empty() || local_path.is_empty() || remote_dir.is_empty() {
        return (
            400,
            json!({"error": "sessionId, localPath and remoteDir are required"}),
        );
    }
    let overwrite = parsed
        .get("overwrite")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_upload(app, state, &session_id, &local_path, &remote_dir, overwrite).await
}

async fn handle_upload_dir(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let local_dir = body_str(&parsed, "localDir");
    let remote_dir = body_str(&parsed, "remoteDir");
    if session_id.is_empty() || local_dir.is_empty() || remote_dir.is_empty() {
        return (
            400,
            json!({"error": "sessionId, localDir and remoteDir are required"}),
        );
    }
    let overwrite = parsed
        .get("overwrite")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_upload_dir(app, state, &session_id, &local_dir, &remote_dir, overwrite).await
}

async fn handle_sync(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let local_dir = body_str(&parsed, "localDir");
    let remote_dir = body_str(&parsed, "remoteDir");
    if session_id.is_empty() || local_dir.is_empty() || remote_dir.is_empty() {
        return (
            400,
            json!({"error": "sessionId, localDir and remoteDir are required"}),
        );
    }
    let (direction, strategy) = match parse_sync_args(
        &body_str(&parsed, "direction"),
        &body_str(&parsed, "strategy"),
    ) {
        Ok(v) => v,
        Err(msg) => return (400, json!({ "error": msg })),
    };
    let dry_run = parsed
        .get("dryRun")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_sync(
        app,
        state,
        &session_id,
        &local_dir,
        &remote_dir,
        direction,
        strategy,
        dry_run,
    )
    .await
}

async fn handle_diff(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let path_a = body_str(&parsed, "pathA");
    let path_b = body_str(&parsed, "pathB");
    if path_a.is_empty() || path_b.is_empty() {
        return (400, json!({"error": "pathA and pathB are required"}));
    }
    // An omitted / empty session means the local filesystem.
    let side_a = parsed
        .get("sessionA")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty());
    let side_b = parsed
        .get("sessionB")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty());
    let hash = parsed
        .get("hash")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    op_diff(app, state, side_a, &path_a, side_b, &path_b, hash).await
}

async fn handle_search(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let pattern = body_str(&parsed, "query");
    if session_id.is_empty() || pattern.is_empty() {
        return (400, json!({"error": "sessionId and query are required"}));
    }
    let root = match body_str(&parsed, "path") {
        p if p.is_empty() => ".".to_string(),
        p => p,
    };
    let query = build_search_query(&parsed, pattern);
    op_search(app, state, &session_id, &root, &query).await
}

async fn handle_read_batch(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let paths = parsed
        .get("paths")
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|v| v.as_str().map(String::from))
                .collect::<Vec<_>>()
        })
        .unwrap_or_default();
    if session_id.is_empty() || paths.is_empty() {
        return (400, json!({"error": "sessionId and paths are required"}));
    }
    op_read_file_batch(app, state, &session_id, paths).await
}

async fn handle_glob(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let pattern = body_str(&parsed, "pattern");
    if session_id.is_empty() || pattern.is_empty() {
        return (400, json!({"error": "sessionId and pattern are required"}));
    }
    let root = match body_str(&parsed, "path") {
        p if p.is_empty() => ".".to_string(),
        p => p,
    };
    op_glob(app, state, &session_id, &root, &pattern).await
}

async fn handle_tail(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    let path = body_str(&parsed, "path");
    if session_id.is_empty() || path.is_empty() {
        return (400, json!({"error": "sessionId and path are required"}));
    }
    let lines = parsed.get("lines").and_then(|v| v.as_u64()).unwrap_or(50) as usize;
    op_tail(app, state, &session_id, &path, lines).await
}

async fn handle_info(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let session_id = body_str(&parsed, "sessionId");
    if session_id.is_empty() {
        return (400, json!({"error": "sessionId is required"}));
    }
    op_server_info(app, state, &session_id).await
}

async fn handle_transfer_status(app: &AppHandle, body: &[u8]) -> (u16, Value) {
    let parsed = match parse_body(body) {
        Ok(v) => v,
        Err(e) => return e,
    };
    let transfer_id = body_str(&parsed, "transferId");
    if transfer_id.is_empty() {
        return (400, json!({"error": "transferId is required"}));
    }
    op_transfer_status(app, &transfer_id).await
}

async fn handle_history(app: &AppHandle, state: &Arc<BridgeState>, body: &[u8]) -> (u16, Value) {
    // An empty body (or `{}`) returns the whole recent log.
    let parsed = if body.is_empty() {
        json!({})
    } else {
        match parse_body(body) {
            Ok(v) => v,
            Err(e) => return e,
        }
    };
    let session = parsed
        .get("sessionId")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty());
    let limit = parsed.get("limit").and_then(|v| v.as_u64()).unwrap_or(50) as usize;
    op_history(app, state, session, limit).await
}

// ---- Operation cores (shared by REST routes and MCP tools) ----

/// Run a non-interactive shell command on an SSH server or a Ghost FTP Agent machine.
/// `timeout_ms` optionally overrides the 60 s default (clamped to [1 s, 15 min]);
/// the 512 KiB output cap is fixed.
pub(crate) async fn exec_on(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    command: &str,
    dry_run: bool,
    timeout_ms: Option<u64>,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();

    // Resolve which execable backend this session is. Both SSH servers and
    // paired Ghost FTP Agent machines can run commands; other protocols cannot.
    let (session_name, is_agent) = if let Some(ssh) = manager.get_ssh(session_id).await {
        (ssh.profile.name.clone(), false)
    } else if let Some(agent) = manager.get_agent(session_id).await {
        (agent.profile.name.clone(), true)
    } else {
        return (
            400,
            json!({"error": "the requested connection can't run commands. Exec works on SSH/SFTP and Ghost FTP Agent connections; use list_dir/read/download/upload for other protocols."}),
        );
    };

    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!(
                    "connection '{}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.",
                    session_name
                )
            }),
        );
    }

    let policy = *state.policy.lock().await;
    let would_auto_approve =
        policy.allow_all || (policy.auto_safe_exec && is_read_only_command(command));
    if dry_run {
        return (
            200,
            json!({
                "wouldRun": command,
                "needsApproval": !would_auto_approve,
                "reason": if policy.allow_all {
                    "auto-approve all requests is on"
                } else if policy.auto_safe_exec && is_read_only_command(command) {
                    "command matches the safe read-only heuristic"
                } else {
                    "command will prompt for approval in Ghost FTP"
                },
            }),
        );
    }

    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &session_name,
        OpClass::Exec,
        "exec",
        command,
        Some(command),
    )
    .await
    {
        return resp;
    }

    let timeout = exec_timeout_from(timeout_ms);
    if is_agent {
        let Some(agent) = manager.get_agent(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        exec_core_agent(
            app,
            state,
            &agent,
            session_id,
            &session_name,
            command,
            command,
            timeout,
        )
        .await
    } else {
        let Some(ssh) = manager.get_ssh(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        exec_core(
            app,
            state,
            &ssh,
            session_id,
            &session_name,
            command,
            command,
            timeout,
        )
        .await
    }
}

/// Run a whole local script (read as raw bytes by the caller, so heredocs, nested
/// quotes, and newlines survive) on an SSH server or a paired agent — the answer
/// to the base64/heredoc gymnastics a multi-line command otherwise forces
/// (Plan 10 Phase 1). The script text runs the same way the tracked `exec` path
/// already runs a command: piped verbatim into the target's shell (`base64 -d |
/// sh` on SSH, `sh -c`/`powershell -Command` on an agent daemon), never spliced
/// into a command line. Gated as an Exec, but — like a Write — never auto-approved
/// by the safe-read-only heuristic (only allow-all), since a script is
/// multi-statement and can't be classified safe. `label` is the friendly name
/// shown in the console + audit (the script never dumps there).
async fn exec_script_on(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    script: &str,
    label: &str,
    dry_run: bool,
    timeout_ms: Option<u64>,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let (session_name, is_agent) = if let Some(ssh) = manager.get_ssh(session_id).await {
        (ssh.profile.name.clone(), false)
    } else if let Some(agent) = manager.get_agent(session_id).await {
        (agent.profile.name.clone(), true)
    } else {
        return (
            400,
            json!({"error": "the requested connection can't run scripts. Scripts run on SSH/SFTP and Ghost FTP Agent connections only."}),
        );
    };

    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!(
                    "connection '{}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.",
                    session_name
                )
            }),
        );
    }

    if dry_run {
        let policy = *state.policy.lock().await;
        return (
            200,
            json!({
                "wouldRun": label,
                "bytes": script.len(),
                "lines": script.lines().count(),
                "needsApproval": !policy.allow_all,
                "reason": if policy.allow_all {
                    "auto-approve all requests is on"
                } else {
                    "a script always prompts for approval in Ghost FTP (never auto-approved by the safe-exec heuristic)"
                },
            }),
        );
    }

    // Gate as an Exec, but pass `None` for the command so the safe-read-only
    // heuristic can't auto-approve a multi-statement script — only allow-all does.
    // The approval summary shows a header plus the script body (capped) so the
    // user reviews exactly what will run.
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &session_name,
        OpClass::Exec,
        "script",
        &script_approval_summary(label, script),
        None,
    )
    .await
    {
        return resp;
    }

    let timeout = exec_timeout_from(timeout_ms);
    if is_agent {
        let Some(agent) = manager.get_agent(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        exec_core_agent(
            app,
            state,
            &agent,
            session_id,
            &session_name,
            script,
            label,
            timeout,
        )
        .await
    } else {
        let Some(ssh) = manager.get_ssh(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        exec_core(
            app,
            state,
            &ssh,
            session_id,
            &session_name,
            script,
            label,
            timeout,
        )
        .await
    }
}

/// Approval-prompt text for a script run: a one-line header naming the script and
/// its size, then the body (capped at ~4 KiB so a huge script doesn't overflow
/// the prompt — the user still sees the head and the total size).
fn script_approval_summary(label: &str, script: &str) -> String {
    const CAP: usize = 4096;
    let header = format!(
        "Run {label} ({} bytes, {} lines):",
        script.len(),
        script.lines().count()
    );
    if script.len() <= CAP {
        format!("{header}\n{script}")
    } else {
        // Cut at the largest char boundary ≤ CAP so we never split a UTF-8 scalar.
        let mut cut = CAP;
        while cut > 0 && !script.is_char_boundary(cut) {
            cut -= 1;
        }
        format!(
            "{header}\n{}\n… [preview truncated; {} bytes total]",
            &script[..cut],
            script.len()
        )
    }
}

/// Exec on a Ghost FTP Agent machine. The daemon runs the command natively (and
/// enforces its own policy), so there's no PTY/pgid/streaming as with SSH — we
/// emit the same live-console events (`exec-start` → `output` → terminal `job`)
/// around a single request so the UI renders it identically, then log the audit
/// line.
#[allow(clippy::too_many_arguments)]
async fn exec_core_agent(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    agent: &Arc<crate::session::AgentSession>,
    session_id: &str,
    session_name: &str,
    command: &str,
    label: &str,
    timeout: Duration,
) -> (u16, Value) {
    let op_id = Uuid::new_v4().to_string();
    let started_at = now_millis();
    let _ = app.emit(
        "agent://exec-start",
        json!({
            "opId": op_id,
            "sessionId": session_id,
            "sessionName": session_name,
            "command": label,
        }),
    );

    let result = agent
        .exec(command, MAX_EXEC_BYTES, timeout.as_millis() as u64)
        .await;

    match result {
        Ok(out) => {
            if !out.stdout.is_empty() {
                let _ = app.emit(
                    "agent://output",
                    json!({ "opId": op_id, "stream": "stdout", "chunk": out.stdout }),
                );
            }
            if !out.stderr.is_empty() {
                let _ = app.emit(
                    "agent://output",
                    json!({ "opId": op_id, "stream": "stderr", "chunk": out.stderr }),
                );
            }
            let ok = !out.timed_out && out.exit_code.unwrap_or(0) == 0;
            let status = if out.timed_out {
                "timedout"
            } else if ok {
                "completed"
            } else {
                "failed"
            };
            // Finalize the live-console row.
            let _ = app.emit(
                "agent://job",
                json!({
                    "opId": op_id,
                    "sessionId": session_id,
                    "label": label,
                    "pgid": Value::Null,
                    "startedAt": started_at,
                    "status": status,
                    "exitCode": out.exit_code,
                }),
            );
            let mut detail = label.to_string();
            if out.truncated {
                detail.push_str("  [output truncated]");
            }
            if out.timed_out {
                detail.push_str("  [timed out]");
            }
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "exec".into(),
                        detail,
                        ok,
                        at: now_millis(),
                    },
                )
                .await;
            (
                200,
                json!({
                    "stdout": out.stdout,
                    "stderr": out.stderr,
                    "exitCode": out.exit_code,
                    "truncated": out.truncated,
                    "timedOut": out.timed_out,
                }),
            )
        }
        Err(e) => {
            let _ = app.emit(
                "agent://job",
                json!({
                    "opId": op_id,
                    "sessionId": session_id,
                    "label": label,
                    "pgid": Value::Null,
                    "startedAt": started_at,
                    "status": "failed",
                    "exitCode": Value::Null,
                }),
            );
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "error".into(),
                        detail: format!("{label} — {e}"),
                        ok: false,
                        at: now_millis(),
                    },
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

// ---- Detached / background jobs (Plan 10 Phase 4, SSH arm) ----
//
// A `--detach` exec returns a job id immediately and keeps running server-side,
// retiring the manual `nohup … & ; tail -f log` loop. SSH-only for now (pure
// convention, no protocol change): the command runs detached (`setsid`/`nohup`)
// into a per-job dir `~/.ghostftp/jobs/<id>/{cmd,out,err,exit,pid}`, and `job`/`jobs`
// read those files. The agent-target arm (ExecStart/Poll/Kill in the protocol)
// is the documented next step.

/// Root of the per-job spill on the server. `$HOME` is expanded remotely.
const JOB_DIR: &str = "$HOME/.ghostftp/jobs";

/// Shell to launch `cmd_b64` (base64 of the command) detached into a fresh job
/// dir. Base64 keeps the command opaque so no quoting can break the wrapper. The
/// launcher prunes job dirs older than 7 days first (best-effort), then returns
/// at once — the real work runs in a new session/`nohup` and records its own exit
/// code. Prints the job id on stdout.
fn job_launch_script(job_id: &str, cmd_b64: &str) -> String {
    format!(
        r#"d="{JOB_DIR}/{job_id}"; mkdir -p "$d" || exit 1
find "{JOB_DIR}" -maxdepth 1 -mindepth 1 -type d -mtime +7 -exec rm -rf {{}} + 2>/dev/null
printf %s '{cmd_b64}' | base64 -d > "$d/cmd" || exit 1
run='sh "$1/cmd" >"$1/out" 2>"$1/err"; printf %s "$?" > "$1/exit"'
if command -v setsid >/dev/null 2>&1; then
  setsid sh -c "$run" _ "$d" </dev/null >/dev/null 2>&1 &
else
  nohup sh -c "$run" _ "$d" </dev/null >/dev/null 2>&1 &
fi
printf %s "$!" > "$d/pid" 2>/dev/null
echo "{job_id}""#
    )
}

/// Shell to poll a job: prints `exit=<code>` (empty while running) and the
/// base64 of the (capped) out/err streams, so no marker can collide with content.
fn job_poll_script(job_id: &str) -> String {
    format!(
        r#"d="{JOB_DIR}/{job_id}"
if [ ! -d "$d" ]; then echo "nojob=1"; exit 0; fi
printf 'exit='; cat "$d/exit" 2>/dev/null; printf '\n'
printf 'out='; head -c 524288 "$d/out" 2>/dev/null | base64 | tr -d '\n'; printf '\n'
printf 'err='; head -c 524288 "$d/err" 2>/dev/null | base64 | tr -d '\n'; printf '\n'"#
    )
}

/// Shell to list all jobs: one `id<TAB>status<TAB>exitCode` line each.
fn job_list_script() -> String {
    format!(
        r#"jd="{JOB_DIR}"; [ -d "$jd" ] || exit 0
for d in "$jd"/*/; do
  [ -d "$d" ] || continue
  id=$(basename "$d")
  if [ -f "$d/exit" ]; then printf '%s\t%s\t%s\n' "$id" done "$(cat "$d/exit" 2>/dev/null)"; else printf '%s\t%s\t\n' "$id" running; fi
done"#
    )
}

/// Launch a detached job on an SSH server; returns its job id at once. Gated as
/// an Exec (the command runs), never auto-approved except by allow-all.
async fn op_exec_detached(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    command: &str,
    dry_run: bool,
) -> (u16, Value) {
    use base64::Engine as _;
    let manager = app.state::<AppState>().sessions.clone();
    // Ghost FTP Agent target: launch via the daemon's ExecStart (Plan 10 Phase 4).
    if let Some(agent) = manager.get_agent(session_id).await {
        return op_exec_detached_agent(app, state, &agent, session_id, command, dry_run).await;
    }
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (
            400,
            json!({"error": "detached exec needs an SSH/SFTP or Ghost FTP Agent connection."}),
        );
    };
    let session_name = ssh.profile.name.clone();
    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!("connection '{session_name}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.")
            }),
        );
    }
    if dry_run {
        let policy = *state.policy.lock().await;
        return (
            200,
            json!({
                "wouldRun": command,
                "detach": true,
                "needsApproval": !policy.allow_all,
            }),
        );
    }
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &session_name,
        OpClass::Exec,
        "exec",
        &format!("run detached (background job): {command}"),
        None,
    )
    .await
    {
        return resp;
    }
    let job_id = Uuid::new_v4().to_string();
    let script = job_launch_script(
        &job_id,
        &base64::engine::general_purpose::STANDARD.encode(command),
    );
    match ssh
        .exec_bounded(&script, 64 * 1024, Duration::from_secs(20), None)
        .await
    {
        Ok(out) if out.exit_code == Some(0) => {
            state
                .log(
                    app,
                    activity(
                        "exec",
                        session_id,
                        format!("detached job {job_id}: {command}"),
                        true,
                    ),
                )
                .await;
            (200, json!({ "jobId": job_id, "status": "started" }))
        }
        Ok(out) => (
            500,
            json!({"error": format!("failed to launch background job: {}", out.stderr.trim())}),
        ),
        Err(e) => (500, json!({"error": e.to_string()})),
    }
}

/// Launch a detached job on a paired Ghost FTP Agent (Plan 10 Phase 4, agent arm):
/// the daemon spawns it under the bridge-chosen job id and returns at once; poll
/// it with `/job`. Gated identically to the SSH arm.
async fn op_exec_detached_agent(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    agent: &Arc<crate::session::AgentSession>,
    session_id: &str,
    command: &str,
    dry_run: bool,
) -> (u16, Value) {
    let session_name = agent.profile.name.clone();
    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!("connection '{session_name}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.")
            }),
        );
    }
    if dry_run {
        let policy = *state.policy.lock().await;
        return (
            200,
            json!({
                "wouldRun": command,
                "detach": true,
                "needsApproval": !policy.allow_all,
            }),
        );
    }
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &session_name,
        OpClass::Exec,
        "exec",
        &format!("run detached (background job): {command}"),
        None,
    )
    .await
    {
        return resp;
    }
    let job_id = Uuid::new_v4().to_string();
    // 512 KiB per stream, matching the SSH poll cap.
    match agent.exec_start(&job_id, command, 512 * 1024).await {
        Ok(id) => {
            state
                .log(
                    app,
                    activity(
                        "exec",
                        session_id,
                        format!("detached job {id}: {command}"),
                        true,
                    ),
                )
                .await;
            (200, json!({ "jobId": id, "status": "started" }))
        }
        Err(e) => (
            500,
            json!({"error": format!(
                "failed to start background job on agent '{session_name}': {e}. The agent may predate Plan 10 (update ghostftp-agentd), or the connection dropped."
            )}),
        ),
    }
}

/// Parse the `job_poll_script` output into a status object.
fn parse_job_poll(text: &str) -> Value {
    use base64::Engine as _;
    let b64 = base64::engine::general_purpose::STANDARD;
    let mut nojob = false;
    let mut exit: Option<i64> = None;
    let mut running = true;
    let mut stdout = String::new();
    let mut stderr = String::new();
    for line in text.lines() {
        if let Some(v) = line.strip_prefix("nojob=") {
            nojob = v.trim() == "1";
        } else if let Some(v) = line.strip_prefix("exit=") {
            let v = v.trim();
            if !v.is_empty() {
                running = false;
                exit = v.parse().ok();
            }
        } else if let Some(v) = line.strip_prefix("out=") {
            stdout = b64
                .decode(v.trim())
                .ok()
                .map(|b| String::from_utf8_lossy(&b).to_string())
                .unwrap_or_default();
        } else if let Some(v) = line.strip_prefix("err=") {
            stderr = b64
                .decode(v.trim())
                .ok()
                .map(|b| String::from_utf8_lossy(&b).to_string())
                .unwrap_or_default();
        }
    }
    if nojob {
        return json!({"error": "no such job on that server (it may have been pruned)"});
    }
    json!({ "running": running, "exitCode": exit, "stdout": stdout, "stderr": stderr })
}

/// Poll a detached job's progress/output (SSH server or Ghost FTP Agent). Read-only.
async fn op_job(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    job_id: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    // Ghost FTP Agent target: poll via the daemon's ExecPoll (Plan 10 Phase 4).
    if let Some(agent) = manager.get_agent(session_id).await {
        let name = agent.profile.name.clone();
        if let Err(resp) = gate(
            app,
            state,
            session_id,
            &name,
            OpClass::Read,
            "job",
            &format!("read job {job_id}"),
            None,
        )
        .await
        {
            return resp;
        }
        return match agent.exec_poll(job_id).await {
            Ok(s) if s.not_found => (
                404,
                json!({"error": "no such job on that agent (it may have been pruned, or the daemon restarted)", "jobId": job_id}),
            ),
            Ok(s) => (
                200,
                json!({ "running": s.running, "exitCode": s.exit_code, "stdout": s.stdout, "stderr": s.stderr, "jobId": job_id }),
            ),
            Err(e) => (500, json!({"error": e.to_string()})),
        };
    }
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (
            400,
            json!({"error": "detached jobs need an SSH/SFTP or Ghost FTP Agent connection."}),
        );
    };
    let name = ssh.profile.name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "job",
        &format!("read job {job_id}"),
        None,
    )
    .await
    {
        return resp;
    }
    // A job id is a UUID we generated; reject anything else so it can't be used
    // to smuggle shell into the path.
    if !job_id
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '-')
    {
        return (400, json!({"error": "invalid job id"}));
    }
    match ssh
        .exec_bounded(
            &job_poll_script(job_id),
            1024 * 1024,
            Duration::from_secs(15),
            None,
        )
        .await
    {
        Ok(out) => {
            let mut body = parse_job_poll(&out.stdout);
            if let Some(obj) = body.as_object_mut() {
                obj.insert("jobId".into(), json!(job_id));
            }
            let status = if body.get("error").is_some() {
                404
            } else {
                200
            };
            (status, body)
        }
        Err(e) => (500, json!({"error": e.to_string()})),
    }
}

/// List detached jobs on a server (SSH only). Read-only.
async fn op_jobs(app: &AppHandle, state: &Arc<BridgeState>, session_id: &str) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (400, json!({"error": "jobs are SSH-only for now."}));
    };
    let name = ssh.profile.name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "jobs",
        "list background jobs",
        None,
    )
    .await
    {
        return resp;
    }
    match ssh
        .exec_bounded(
            &job_list_script(),
            256 * 1024,
            Duration::from_secs(15),
            None,
        )
        .await
    {
        Ok(out) => {
            let jobs: Vec<Value> = out
                .stdout
                .lines()
                .filter_map(|line| {
                    let mut it = line.split('\t');
                    let id = it.next()?.trim();
                    if id.is_empty() {
                        return None;
                    }
                    let status = it.next().unwrap_or("").trim();
                    let code = it.next().unwrap_or("").trim();
                    Some(json!({
                        "jobId": id,
                        "running": status == "running",
                        "exitCode": code.parse::<i64>().ok(),
                    }))
                })
                .collect();
            (200, json!({ "jobs": jobs }))
        }
        Err(e) => (500, json!({"error": e.to_string()})),
    }
}

/// Run a user-defined saved command by name. Pre-approved: it still requires the
/// session to be opted in, but it deliberately BYPASSES the approval gate
/// because the user wrote and vetted the exact string when they saved it (the
/// agent supplies only the name and can't author the command). Always SSH-only.
pub(crate) async fn op_run_command(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    name: &str,
    dry_run: bool,
    timeout_ms: Option<u64>,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (
            400,
            json!({"error": "the requested connection is not an SSH session. Saved commands can only run on SSH/SFTP connections."}),
        );
    };
    let Some(cmd) = state.find_command(name).await else {
        return (
            404,
            json!({"error": format!("no saved command named '{name}'")}),
        );
    };
    // Opt-in is still required (a saved command is pre-approved, not un-gated):
    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!(
                    "connection '{}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.",
                    ssh.profile.name
                )
            }),
        );
    }
    let session_name = ssh.profile.name.clone();
    let label = format!("[{}] {}", cmd.name, cmd.command);
    if dry_run {
        return (
            200,
            json!({
                "wouldRun": cmd.command,
                "needsApproval": false,
                "reason": "saved commands are pre-approved by the user",
            }),
        );
    }
    let timeout = exec_timeout_from(timeout_ms);
    exec_core(
        app,
        state,
        &ssh,
        session_id,
        &session_name,
        &cmd.command,
        &label,
        timeout,
    )
    .await
}

/// Shared exec body used by `exec_on` (after the approval gate) and
/// `op_run_command` (after the opt-in check only). Tags the run so streamed
/// output + the final audit line correlate in the live agent console, runs it
/// bounded/streamed, and logs the activity. `label` is what shows in the
/// console + audit log (the raw command for `exec`, "[name] command" for a
/// saved command); `command` is what actually executes. `timeout` is the
/// already-clamped run deadline (see `exec_timeout_from`).
#[allow(clippy::too_many_arguments)]
async fn exec_core(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    ssh: &crate::session::SshSession,
    session_id: &str,
    session_name: &str,
    command: &str,
    label: &str,
    timeout: Duration,
) -> (u16, Value) {
    let op_id = Uuid::new_v4().to_string();
    let _ = app.emit(
        "agent://exec-start",
        json!({
            "opId": op_id,
            "sessionId": session_id,
            "sessionName": session_name,
            "command": label,
        }),
    );
    let stream = ExecStream {
        app: app.clone(),
        op_id: op_id.clone(),
        label: label.to_string(),
    };

    // Sudo support: when the user opted in (allow_sudo) and this is a
    // password-auth session, answer sudo's prompt with the connection password
    // over a PTY. Otherwise the normal no-tty exec (clean stdout/stderr split).
    let sudo_pw = {
        let policy = *state.policy.lock().await;
        if policy.allow_sudo && command_uses_sudo(command) {
            connection_password(ssh)
        } else {
            None
        }
    };
    let result = match &sudo_pw {
        Some(pw) => {
            ssh.exec_sudo(command, pw, MAX_EXEC_BYTES, timeout, Some(&stream))
                .await
        }
        None => {
            ssh.exec_bounded(command, MAX_EXEC_BYTES, timeout, Some(&stream))
                .await
        }
    };

    match result {
        Ok(out) => {
            let ok = !out.timed_out && out.exit_code.unwrap_or(0) == 0;
            let mut detail = label.to_string();
            if out.truncated {
                detail.push_str("  [output truncated]");
            }
            if out.timed_out {
                detail.push_str(if out.killed {
                    "  [timed out — terminated]"
                } else {
                    "  [timed out]"
                });
            }
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "exec".into(),
                        detail,
                        ok,
                        at: now_millis(),
                    },
                )
                .await;
            (
                200,
                json!({
                    "stdout": out.stdout,
                    "stderr": out.stderr,
                    "exitCode": out.exit_code,
                    "truncated": out.truncated,
                    "timedOut": out.timed_out,
                    "killed": out.killed,
                }),
            )
        }
        Err(e) => {
            // Reuse op_id so the live console finalizes the same "running" entry
            // emitted by agent://exec-start (instead of leaving a stuck spinner
            // and appending a duplicate row).
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "error".into(),
                        detail: format!("{label} — {e}"),
                        ok: false,
                        at: now_millis(),
                    },
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

pub(crate) async fn op_list_dir(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    path: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "read",
        &format!("list {path}"),
        None,
    )
    .await
    {
        return resp;
    }
    let fs = crate::commands::fs_for_session(&sess);
    match fs.list_dir(path).await {
        Ok(entries) => {
            state
                .log(
                    app,
                    activity(
                        "read",
                        session_id,
                        format!("list {path} ({} items)", entries.len()),
                        true,
                    ),
                )
                .await;
            (200, json!({ "entries": entries }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("list {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Delete a remote file or directory. Every backend implements `RemoteFs`, so
/// this works on all of them — including FTP, where the alternative was asking
/// the user to open a file manager and do it by hand.
///
/// Destructive and irreversible: gated as a Write, and the approval summary
/// names the exact path (and says "and everything inside it" for a recursive
/// delete) so the user is agreeing to something specific.
pub(crate) async fn op_delete(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    path: &str,
    recursive: bool,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "delete",
        &format!(
            "DELETE {name}:{path}{}",
            if recursive {
                " and everything inside it"
            } else {
                ""
            }
        ),
        None,
    )
    .await
    {
        return resp;
    }
    let fs = crate::commands::fs_for_session(&sess);
    match fs.delete(path, recursive).await {
        Ok(()) => {
            state
                .log(
                    app,
                    activity("delete", session_id, format!("delete {path}"), true),
                )
                .await;
            (200, json!({ "path": path, "status": "deleted" }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("delete {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Rename or move a remote path. Also the way to quarantine something without
/// destroying it — `mv shell.php shell.php.quarantine` beats a delete when the
/// user may still want to look at the file.
pub(crate) async fn op_rename(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    from: &str,
    to: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if !crate::commands::fs_for_session(&sess)
        .capabilities()
        .can_rename
    {
        return (
            400,
            json!({"error": format!("{name} ({}) has no rename operation", sess.profile().protocol)}),
        );
    }
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "rename",
        &format!("rename {name}:{from} → {to}"),
        None,
    )
    .await
    {
        return resp;
    }
    let fs = crate::commands::fs_for_session(&sess);
    match fs.rename(from, to).await {
        Ok(()) => {
            state
                .log(
                    app,
                    activity("rename", session_id, format!("rename {from} → {to}"), true),
                )
                .await;
            (200, json!({ "from": from, "to": to, "status": "renamed" }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("rename {from} → {to} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Create a remote directory (one level — the parent must exist).
pub(crate) async fn op_mkdir(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    path: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "mkdir",
        &format!("create directory {name}:{path}"),
        None,
    )
    .await
    {
        return resp;
    }
    let fs = crate::commands::fs_for_session(&sess);
    match fs.create_dir(path).await {
        Ok(()) => {
            state
                .log(
                    app,
                    activity("mkdir", session_id, format!("mkdir {path}"), true),
                )
                .await;
            (200, json!({ "path": path, "status": "created" }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("mkdir {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Read a (text) file from a Ghost FTP Agent machine via the daemon, capped like the
/// SFTP path. The gate has already run in `op_read_file`.
async fn op_read_file_agent(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    agent: &Arc<crate::session::AgentSession>,
    session_id: &str,
    path: &str,
) -> (u16, Value) {
    use base64::Engine as _;
    use ghostftp_agent_proto::msg::{Request, Response};
    let resp = agent
        .request(Request::ReadFile {
            path: path.to_string(),
            max_bytes: MAX_READ_FILE as u64,
        })
        .await;
    match resp {
        Ok(Response::File {
            data,
            bytes,
            truncated,
        }) => {
            let decoded = base64::engine::general_purpose::STANDARD
                .decode(&data)
                .unwrap_or_default();
            let content = String::from_utf8_lossy(&decoded).to_string();
            state
                .log(
                    app,
                    activity(
                        "read",
                        session_id,
                        format!("read {path} ({bytes} bytes)"),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({ "content": content, "bytes": bytes, "truncated": truncated }),
            )
        }
        Ok(Response::Error { message, .. }) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("read {path} — {message}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": message}))
        }
        Ok(other) => (
            500,
            json!({"error": format!("unexpected reply: {other:?}")}),
        ),
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("read {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Ceiling on a fetched read over FTP/FTPS. Every other backend either reads a
/// range or drops the rest of the stream once it has enough, but FTP can't abort
/// a `RETR` mid-flight without desyncing the control connection — so the head of
/// a file costs the whole file. Past this we'd be pulling gigabytes to print
/// 256 KiB, which is what `download` is for.
const MAX_FTP_READ_FETCH: u64 = 16 * 1024 * 1024; // 16 MiB

/// Refuse a fetched read that would drag the whole file across a connection that
/// can't stop early (see [`MAX_FTP_READ_FETCH`]). A size lookup that fails —
/// some FTP servers refuse `SIZE` outside binary mode — is not fatal: an unknown
/// size reads as before.
async fn guard_fetch_size(sess: &Arc<Session>, path: &str) -> Result<()> {
    if !matches!(&**sess, Session::Ftp(_)) {
        return Ok(());
    }
    if let Ok(size) = crate::transfer::remote_size(sess, path).await {
        if size > MAX_FTP_READ_FETCH {
            anyhow::bail!(
                "{path} is {size} bytes — FTP can't read just the head of a file, so reading it \
                 would transfer the whole thing for at most {MAX_READ_FILE} bytes of output. \
                 Download it instead."
            );
        }
    }
    Ok(())
}

/// Read a text file from a backend with no in-place read op — FTP/FTPS, S3/Azure,
/// WebDAV, HTTP, Dropbox, OneDrive, Drive, Box, Shopify, HubSpot, Dynamics — by
/// pulling the first `MAX_READ_FILE` bytes over the wire (`preview::read_head`,
/// the same bounded-read the thumbnailer uses: it Ranges where the protocol
/// supports it and stops draining the stream once it has enough).
///
/// This used to be a flat 400 telling the agent to `download` instead. That was
/// busywork: the agent would start a transfer into the user's Downloads folder,
/// poll it, read the file off disk, and leave the copy behind — the same bytes
/// over the same connection, plus litter. Reading is a read, so it's gated as
/// one (nothing is written locally) and `read_file` now works on every protocol.
async fn op_read_file_fetch(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    sess: &Arc<Session>,
    session_id: &str,
    path: &str,
) -> (u16, Value) {
    let name = sess.profile().name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "read",
        &format!("read {path}"),
        None,
    )
    .await
    {
        return resp;
    }
    if let Err(e) = guard_fetch_size(sess, path).await {
        return (400, json!({"error": e.to_string()}));
    }
    // One byte past the cap, so a file that exactly fills it isn't mislabelled
    // as truncated.
    match crate::preview::read_head(Some(sess), path, MAX_READ_FILE as u64 + 1).await {
        Ok(mut buf) => {
            let truncated = buf.len() > MAX_READ_FILE;
            buf.truncate(MAX_READ_FILE);
            let bytes = buf.len();
            state
                .log(
                    app,
                    activity(
                        "read",
                        session_id,
                        format!("read {path} ({bytes} bytes)"),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({
                    "content": String::from_utf8_lossy(&buf).to_string(),
                    "bytes": bytes,
                    "truncated": truncated,
                }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("read {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

pub(crate) async fn op_read_file(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    path: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();

    // A Ghost FTP Agent machine reads via the daemon; SSH via SFTP. Handle the agent
    // case up front, then fall through to the SFTP path.
    if let Some(agent) = manager.get_agent(session_id).await {
        let name = agent.profile.name.clone();
        if let Err(resp) = gate(
            app,
            state,
            session_id,
            &name,
            OpClass::Read,
            "read",
            &format!("read {path}"),
            None,
        )
        .await
        {
            return resp;
        }
        return op_read_file_agent(app, state, &agent, session_id, path).await;
    }

    let Some(ssh) = manager.get_ssh(session_id).await else {
        // Every other protocol fetches the head of the file instead (below).
        let Some(sess) = manager.get(session_id).await else {
            return (
                400,
                json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
            );
        };
        return op_read_file_fetch(app, state, &sess, session_id, path).await;
    };
    let name = ssh.profile.name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "read",
        &format!("read {path}"),
        None,
    )
    .await
    {
        return resp;
    }

    // The whole open + capped read runs inside with_sftp so that, after a long
    // idle, a dead transport transparently reconnects + reopens the subsystem and
    // the read replays once (read-only — always safe to re-run).
    let read = ssh
        .with_sftp(|sftp_cell| async move {
            // Open under the lock, then read without holding it (the file handle
            // is independent of the SFTP guard — see transfer.rs).
            let mut file = {
                let sftp = sftp_cell.lock().await;
                sftp.open(path)
                    .await
                    .with_context(|| format!("open {path}"))?
            };
            let mut buf: Vec<u8> = Vec::new();
            let mut chunk = vec![0u8; 64 * 1024];
            let mut truncated = false;
            loop {
                let n = file.read(&mut chunk).await?;
                if n == 0 {
                    break;
                }
                buf.extend_from_slice(&chunk[..n]);
                if buf.len() >= MAX_READ_FILE {
                    buf.truncate(MAX_READ_FILE);
                    truncated = true;
                    break;
                }
            }
            let bytes = buf.len();
            let content = String::from_utf8_lossy(&buf).to_string();
            Ok((bytes, content, truncated))
        })
        .await;

    match read {
        Ok((bytes, content, truncated)) => {
            state
                .log(
                    app,
                    activity(
                        "read",
                        session_id,
                        format!("read {path} ({bytes} bytes)"),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({ "content": content, "bytes": bytes, "truncated": truncated }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("read {path} — {e}"), false),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

const MAX_READ_BATCH_TOTAL: usize = 1024 * 1024; // 1 MiB total across all files

async fn read_one_file(ssh: &Arc<crate::session::SshSession>, path: &str) -> Result<Value> {
    ssh.with_sftp(|sftp_cell| async move {
        let mut file = {
            let sftp = sftp_cell.lock().await;
            sftp.open(path)
                .await
                .with_context(|| format!("open {path}"))?
        };
        let mut buf: Vec<u8> = Vec::new();
        let mut chunk = vec![0u8; 64 * 1024];
        let mut truncated = false;
        loop {
            let n = file.read(&mut chunk).await?;
            if n == 0 {
                break;
            }
            buf.extend_from_slice(&chunk[..n]);
            if buf.len() >= MAX_READ_FILE {
                buf.truncate(MAX_READ_FILE);
                truncated = true;
                break;
            }
        }
        Ok(json!({
            "path": path,
            "content": String::from_utf8_lossy(&buf).to_string(),
            "bytes": buf.len(),
            "truncated": truncated,
        }))
    })
    .await
}

/// Read one file for the batch on a non-SSH session, shaped like
/// [`read_one_file`]. SSH keeps its own path because `with_sftp` replays the
/// read through a reconnect; everything else goes through the bounded head-read.
async fn read_one_file_any(sess: &Arc<Session>, path: &str) -> Result<Value> {
    guard_fetch_size(sess, path).await?;
    let mut buf = crate::preview::read_head(Some(sess), path, MAX_READ_FILE as u64 + 1).await?;
    let truncated = buf.len() > MAX_READ_FILE;
    buf.truncate(MAX_READ_FILE);
    Ok(json!({
        "path": path,
        "content": String::from_utf8_lossy(&buf).to_string(),
        "bytes": buf.len(),
        "truncated": truncated,
    }))
}

/// Read multiple text files in one call. Total output is capped to avoid
/// ballooning the response. Each file result includes its own truncated flag.
pub(crate) async fn op_read_file_batch(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    paths: Vec<String>,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if paths.is_empty() {
        return (400, json!({"error": "paths array is required"}));
    }
    if paths.len() > 50 {
        return (400, json!({"error": "at most 50 paths per batch"}));
    }
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "read_batch",
        &format!(
            "read {} files (e.g. {})",
            paths.len(),
            paths.first().cloned().unwrap_or_default()
        ),
        None,
    )
    .await
    {
        return resp;
    }

    let mut files = Vec::new();
    let mut total_bytes: usize = 0;
    for path in paths {
        let read = match &*sess {
            Session::Ssh(ssh) => read_one_file(ssh, &path).await,
            _ => read_one_file_any(&sess, &path).await,
        };
        match read {
            Ok(mut v) => {
                let bytes = v.get("bytes").and_then(|b| b.as_u64()).unwrap_or(0) as usize;
                total_bytes += bytes;
                if total_bytes > MAX_READ_BATCH_TOTAL {
                    v["truncated"] = json!(true);
                    v["content"] = json!("");
                    files.push(v);
                    break;
                }
                files.push(v);
            }
            Err(e) => {
                files.push(json!({
                    "path": path,
                    "error": e.to_string(),
                }));
            }
        }
    }

    state
        .log(
            app,
            activity(
                "read_batch",
                session_id,
                format!("read batch of {} files", files.len()),
                true,
            ),
        )
        .await;
    (200, json!({ "files": files }))
}

/// Find files/directories matching a glob-like pattern. SSH uses `find`;
/// SFTP falls back to recursive listing with simple `*`/`?` matching.
/// Returns matching paths relative to the root.
pub(crate) async fn op_glob(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    root: &str,
    pattern: &str,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (
            400,
            json!({"error": "glob is SSH-only (it runs `find` over the shell) — on a Ghost FTP Agent machine use exec with a native command instead"}),
        );
    };
    let name = ssh.profile.name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "glob",
        &format!("glob {root}/{pattern}"),
        None,
    )
    .await
    {
        return resp;
    }

    let cmd = format!(
        "find {} -maxdepth {} -name '{}' -type f 2>/dev/null",
        shell_quote(root),
        SEARCH_MAX_DEPTH,
        pattern.replace('\\', "\\\\").replace('\'', "'\"'\"'")
    );
    match ssh
        .exec_bounded(&cmd, 256 * 1024, Duration::from_secs(30), None)
        .await
    {
        Ok(out) => {
            let paths: Vec<String> = out
                .stdout
                .lines()
                .map(|s| s.trim().to_string())
                .filter(|s| !s.is_empty())
                .take(SEARCH_MAX_RESULTS)
                .collect();
            state
                .log(
                    app,
                    activity("glob", session_id, format!("glob {root}/{pattern}"), true),
                )
                .await;
            (200, json!({ "matches": paths }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("glob {root}/{pattern} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

const MAX_TAIL_BYTES: usize = 512 * 1024;
const TAIL_TIMEOUT: Duration = Duration::from_secs(30);

/// Stream the tail of a log file for a bounded time. Uses `tail -n <lines> -f`
/// and forwards chunks to the agent console via `agent://output` events, then
/// returns the collected output. The command is killed when the timeout elapses.
pub(crate) async fn op_tail(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    path: &str,
    lines: usize,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return (
            400,
            json!({"error": "tail is SSH-only — on a Ghost FTP Agent machine use exec with a native tail command instead"}),
        );
    };
    let name = ssh.profile.name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "tail",
        &format!("tail -n {lines} -f {path}"),
        None,
    )
    .await
    {
        return resp;
    }

    let op_id = Uuid::new_v4().to_string();
    let label = format!("tail -n {lines} -f {path}");
    let _ = app.emit(
        "agent://exec-start",
        json!({
            "opId": op_id,
            "sessionId": session_id,
            "sessionName": name,
            "command": label,
        }),
    );
    let stream = ExecStream {
        app: app.clone(),
        op_id: op_id.clone(),
        label: label.clone(),
    };

    let command = format!("tail -n {lines} -f {}", shell_quote(path));
    match ssh
        .exec_bounded(&command, MAX_TAIL_BYTES, TAIL_TIMEOUT, Some(&stream))
        .await
    {
        Ok(out) => {
            let stdout = out.stdout;
            let ok = !out.timed_out;
            let mut detail = label.clone();
            if out.truncated {
                detail.push_str("  [output truncated]");
            }
            if out.timed_out {
                detail.push_str("  [timed out after 30s]");
            }
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "tail".into(),
                        detail,
                        ok,
                        at: now_millis(),
                    },
                )
                .await;
            (
                200,
                json!({
                    "stdout": stdout,
                    "truncated": out.truncated,
                    "timedOut": out.timed_out,
                }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    ActivityEntry {
                        id: op_id,
                        session_id: session_id.to_string(),
                        kind: "error".into(),
                        detail: format!("{label} — {e}"),
                        ok: false,
                        at: now_millis(),
                    },
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

fn shell_quote(s: &str) -> String {
    format!("'{}'", s.replace('\'', "'\"'\"'"))
}

fn default_download_dir(app: &AppHandle) -> String {
    app.path()
        .download_dir()
        .or_else(|_| app.path().app_data_dir())
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_else(|_| ".".to_string())
}

async fn op_download(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    remote_path: &str,
    local_dir: Option<String>,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    let local_dir = local_dir.unwrap_or_else(|| default_download_dir(app));
    // Download is a *local write* (to an agent-supplied directory), so it's
    // classed Write — only auto-approved under allow-all, never under auto-read.
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "download",
        &format!("download {remote_path} → {local_dir}"),
        None,
    )
    .await
    {
        return resp;
    }
    let transfers = app.state::<AppState>().transfers.clone();
    match transfers
        .start_download(
            sess.clone(),
            remote_path.to_string(),
            local_dir.clone(),
            OverwritePolicy::Rename,
            app.clone(),
        )
        .await
    {
        Ok(id) => {
            state
                .log(
                    app,
                    activity(
                        "download",
                        session_id,
                        format!("download {remote_path} → {local_dir}"),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({ "transferId": id, "localDir": local_dir, "status": "started" }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("download {remote_path} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Split a remote path into (parent directory, file name). A bare name lives in
/// the session's working directory, which every backend spells `.`.
fn split_parent(path: &str) -> (String, String) {
    match path.trim_end_matches('/').rfind('/') {
        Some(0) => ("/".to_string(), path[1..].to_string()),
        Some(i) => (path[..i].to_string(), path[i + 1..].to_string()),
        None => (".".to_string(), path.to_string()),
    }
}

/// Refuse to write over something that's already there. Backends differ wildly
/// in how (and whether) they stat a single path, but every one of them lists a
/// directory — so ask the parent. A listing that fails is reported as such
/// rather than assumed empty: "I couldn't check" must not read as "it's free".
async fn ensure_absent(sess: &Arc<Session>, path: &str) -> Result<()> {
    let (dir, name) = split_parent(path);
    let fs = crate::commands::fs_for_session(sess);
    let entries = fs.list_dir(&dir).await.with_context(|| {
        format!("checking whether {path} already exists; pass overwrite to write without checking")
    })?;
    if entries.iter().any(|e| e.name == name) {
        anyhow::bail!("{path} already exists; pass overwrite to replace it");
    }
    Ok(())
}

/// `write` for backends with no direct write primitive: stage the bytes locally,
/// then upload them onto `remote_path` in place and clean the scratch file up.
async fn write_via_upload(
    sess: &Arc<Session>,
    remote_path: &str,
    bytes: &[u8],
    overwrite: bool,
) -> Result<()> {
    if !overwrite {
        ensure_absent(sess, remote_path).await?;
    }
    let (_, name) = split_parent(remote_path);
    let dir = std::env::temp_dir()
        .join("ghostftp-agent-writes")
        .join(Uuid::new_v4().to_string());
    std::fs::create_dir_all(&dir).context("create scratch dir for write")?;
    let local = dir.join(if name.is_empty() { "file".into() } else { name });
    let staged = std::fs::write(&local, bytes)
        .with_context(|| format!("stage {} bytes for write", bytes.len()));
    let result = match staged {
        Ok(()) => crate::editor::upload_from(sess, &local.to_string_lossy(), remote_path)
            .await
            .map(|_| ()),
        Err(e) => Err(e),
    };
    let _ = std::fs::remove_dir_all(&dir);
    result
}

/// Write `bytes` to `remote_path` on an SSH server (SFTP create), a paired
/// agent (`WriteChunk`), or any other backend (staged in-place upload).
/// Gated as a Write — never auto-approved except by
/// allow-all. Refuses to clobber an existing file unless `overwrite`. This is the
/// engine behind `/write`, `ghostftp_write` and `ghostftp-cli agent write` (Plan 10
/// Phase 2). Content is bounded by the bridge's 1 MiB request-body cap — for
/// large files use `upload`.
async fn op_write(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    remote_path: &str,
    bytes: &[u8],
    overwrite: bool,
) -> (u16, Value) {
    use base64::Engine as _;
    use ghostftp_agent_proto::msg::{Request, Response};
    use tokio::io::AsyncWriteExt;

    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let session_name = sess.profile().name.clone();
    let is_agent = matches!(&*sess, Session::Agent(_));
    let is_ssh = matches!(&*sess, Session::Ssh(_));

    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &session_name,
        OpClass::Write,
        "write",
        &format!(
            "Write {} → {session_name}:{remote_path} (overwrite: {})",
            human_bytes(bytes.len() as u64),
            if overwrite { "yes" } else { "no" }
        ),
        None,
    )
    .await
    {
        return resp;
    }

    let result: Result<()> = if is_agent {
        let Some(agent) = manager.get_agent(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        // Existence check (unless overwriting): a successful Stat means it's there.
        async {
            if !overwrite {
                if let Ok(Response::Stat { .. }) = agent
                    .request(Request::Stat {
                        path: remote_path.to_string(),
                    })
                    .await
                {
                    anyhow::bail!("{remote_path} already exists; pass overwrite to replace it");
                }
            }
            let data = base64::engine::general_purpose::STANDARD.encode(bytes);
            match agent
                .request(Request::WriteChunk {
                    path: remote_path.to_string(),
                    offset: 0,
                    data,
                    truncate: true,
                    done: true,
                })
                .await?
            {
                Response::Written { .. } => Ok(()),
                Response::Error { message, denied } => {
                    anyhow::bail!(if denied {
                        format!("denied: {message}")
                    } else {
                        message
                    })
                }
                other => anyhow::bail!("unexpected write reply: {other:?}"),
            }
        }
        .await
    } else if is_ssh {
        let Some(ssh) = manager.get_ssh(session_id).await else {
            return (400, json!({"error": "session went away"}));
        };
        async {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            if !overwrite && sftp.metadata(remote_path).await.is_ok() {
                anyhow::bail!("{remote_path} already exists; pass overwrite to replace it");
            }
            let mut file = sftp
                .create(remote_path)
                .await
                .with_context(|| format!("create {remote_path}"))?;
            file.write_all(bytes).await?;
            file.flush().await?;
            file.shutdown().await.ok();
            Ok(())
        }
        .await
    } else {
        // Every other backend: stage the bytes in a scratch file and push them
        // through the same in-place upload that edit-in-place saves use, so a
        // write lands ON the target path — no `_1` twin, which is what made
        // "replace this file" impossible over FTP without the desktop UI.
        write_via_upload(&sess, remote_path, bytes, overwrite).await
    };

    match result {
        Ok(()) => {
            state
                .log(
                    app,
                    activity(
                        "write",
                        session_id,
                        format!("write {} → {remote_path}", human_bytes(bytes.len() as u64)),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({ "path": remote_path, "bytes": bytes.len(), "status": "written" }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("write {remote_path} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Upload one local file into a remote directory. `overwrite` picks the policy:
/// off (the default) renames around an existing file — `shell.php` lands as
/// `shell_1.php` — which is the safe choice for "put this file there" but the
/// wrong one for "replace that file", where the rename silently leaves the
/// original in place. Say `overwrite` and the existing file is replaced.
async fn op_upload(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    local_path: &str,
    remote_dir: &str,
    overwrite: bool,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "upload",
        &format!(
            "upload {local_path} → {remote_dir}{}",
            if overwrite {
                " (replacing any file of the same name)"
            } else {
                ""
            }
        ),
        None,
    )
    .await
    {
        return resp;
    }
    let transfers = app.state::<AppState>().transfers.clone();
    match transfers
        .start_upload(
            sess.clone(),
            local_path.to_string(),
            remote_dir.to_string(),
            if overwrite {
                OverwritePolicy::Overwrite
            } else {
                OverwritePolicy::Rename
            },
            app.clone(),
        )
        .await
    {
        Ok(id) => {
            state
                .log(
                    app,
                    activity(
                        "upload",
                        session_id,
                        format!("upload {local_path} → {remote_dir}"),
                        true,
                    ),
                )
                .await;
            (200, json!({ "transferId": id, "status": "started" }))
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("upload {local_path} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Human-readable byte count for approval summaries ("risk obvious" — the user
/// should read "3.4 MB", not "3563520").
fn human_bytes(n: u64) -> String {
    if n < 1024 {
        format!("{n} B")
    } else if n < 1024 * 1024 {
        format!("{:.1} KB", n as f64 / 1024.0)
    } else if n < 1024 * 1024 * 1024 {
        format!("{:.1} MB", n as f64 / (1024.0 * 1024.0))
    } else {
        format!("{:.2} GB", n as f64 / (1024.0 * 1024.0 * 1024.0))
    }
}

/// Overwrite policy for the agent-facing directory upload: collisions are
/// renamed (`_1`, `_2`, …) unless the agent explicitly asked to overwrite.
fn upload_overwrite_policy(overwrite: bool) -> OverwritePolicy {
    if overwrite {
        OverwritePolicy::Overwrite
    } else {
        OverwritePolicy::Rename
    }
}

/// The one-shot approval summary for a whole-tree upload. Counts are computed
/// BEFORE gating so the user approves a concrete, sized operation.
fn upload_dir_summary(
    local_dir: &str,
    session_name: &str,
    remote_dir: &str,
    files: usize,
    bytes: u64,
    overwrite: bool,
) -> String {
    format!(
        "Upload directory {local_dir} → {session_name}:{remote_dir} ({files} files, {} total, overwrite: {})",
        human_bytes(bytes),
        if overwrite { "yes" } else { "no" },
    )
}

/// Count the files/bytes a directory upload would queue. Mirrors the walk in
/// `TransferManager::start_directory_upload` (skips symlinks and unreadable
/// entries) so the approval summary matches what actually uploads.
async fn count_local_tree(root: &std::path::Path) -> Result<(usize, u64)> {
    let mut dirs_to_visit: Vec<PathBuf> = vec![root.to_path_buf()];
    let mut files: usize = 0;
    let mut bytes: u64 = 0;
    while let Some(d) = dirs_to_visit.pop() {
        let mut rd = tokio::fs::read_dir(&d)
            .await
            .with_context(|| format!("read_dir {}", d.display()))?;
        while let Some(entry) = rd.next_entry().await? {
            let meta = match entry.metadata().await {
                Ok(m) => m,
                Err(_) => continue,
            };
            if meta.is_dir() {
                dirs_to_visit.push(entry.path());
            } else if meta.is_file() {
                files += 1;
                bytes = bytes.saturating_add(meta.len());
            }
        }
    }
    Ok((files, bytes))
}

/// Upload a whole local directory tree into a remote directory through the
/// transfer engine (one transfer per file, remote dirs created depth-first).
/// ONE approval covers the whole tree; the summary names the file/byte counts
/// and the overwrite mode so the risk is obvious up front.
async fn op_upload_dir(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    local_dir: &str,
    remote_dir: &str,
    overwrite: bool,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();

    let local_root = PathBuf::from(local_dir);
    match tokio::fs::metadata(&local_root).await {
        Ok(m) if m.is_dir() => {}
        Ok(_) => {
            return (
                400,
                json!({"error": format!("localDir {local_dir} is not a directory — use upload for a single file")}),
            )
        }
        Err(e) => {
            return (
                400,
                json!({"error": format!("localDir {local_dir} does not exist or is unreadable: {e}")}),
            )
        }
    }

    // Size the tree first so the single gate names exactly what's at stake.
    let (file_count, total_bytes) = match count_local_tree(&local_root).await {
        Ok(v) => v,
        Err(e) => return (500, json!({"error": format!("walking {local_dir}: {e}")})),
    };

    let summary = upload_dir_summary(
        local_dir,
        &name,
        remote_dir,
        file_count,
        total_bytes,
        overwrite,
    );
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "upload_dir",
        &summary,
        None,
    )
    .await
    {
        return resp;
    }

    // Same remote-root shape start_directory_upload computes internally:
    // the local directory is recreated INSIDE remoteDir.
    let root_name = local_root
        .file_name()
        .map(|n| n.to_string_lossy().into_owned())
        .unwrap_or_else(|| "upload".into());
    let remote_root = if remote_dir.ends_with('/') {
        format!("{remote_dir}{root_name}")
    } else {
        format!("{remote_dir}/{root_name}")
    };

    let transfers = app.state::<AppState>().transfers.clone();
    match transfers
        .start_directory_upload(
            sess.clone(),
            local_dir.to_string(),
            remote_dir.to_string(),
            upload_overwrite_policy(overwrite),
            app.clone(),
        )
        .await
    {
        Ok(ids) => {
            state
                .log(
                    app,
                    activity(
                        "upload_dir",
                        session_id,
                        format!(
                            "upload dir {local_dir} → {remote_root} ({} files, {})",
                            ids.len(),
                            human_bytes(total_bytes)
                        ),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({
                    "transferIds": ids,
                    "fileCount": ids.len(),
                    "totalBytes": total_bytes,
                    "remoteRoot": remote_root,
                    "status": "started",
                }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("upload dir {local_dir} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Parse the wire words for a sync request. Defaults (empty string) are
/// push + additive — the least destructive combination.
fn parse_sync_args(
    direction: &str,
    strategy: &str,
) -> Result<(SyncDirection, SyncStrategy), String> {
    let d = match direction {
        "" | "push" => SyncDirection::LocalToRemote,
        "pull" => SyncDirection::RemoteToLocal,
        other => {
            return Err(format!(
                "direction must be \"push\" or \"pull\" (got \"{other}\")"
            ))
        }
    };
    let s = match strategy {
        "" | "additive" => SyncStrategy::Additive,
        "mirror" => SyncStrategy::Mirror,
        other => {
            return Err(format!(
                "strategy must be \"additive\" or \"mirror\" (got \"{other}\")"
            ))
        }
    };
    Ok((d, s))
}

fn sync_direction_word(direction: SyncDirection) -> &'static str {
    match direction {
        SyncDirection::LocalToRemote => "push",
        SyncDirection::RemoteToLocal => "pull",
    }
}

fn sync_strategy_word(strategy: SyncStrategy) -> &'static str {
    match strategy {
        SyncStrategy::Additive => "additive",
        SyncStrategy::Mirror => "mirror",
    }
}

/// The one-shot approval summary for executing a sync plan. Mirror deletes are
/// the riskiest part, so the delete count is always spelled out for mirror —
/// and additive says so explicitly, so "no deletes" is a promise, not an
/// omission.
#[allow(clippy::too_many_arguments)]
fn sync_gate_summary(
    direction: SyncDirection,
    strategy: SyncStrategy,
    local_dir: &str,
    session_name: &str,
    remote_dir: &str,
    copies: usize,
    bytes: u64,
    deletes: usize,
) -> String {
    let tail = match strategy {
        SyncStrategy::Mirror => {
            format!(", delete {deletes} files on the destination (mirror)")
        }
        SyncStrategy::Additive => ", no deletes (additive)".to_string(),
    };
    format!(
        "Sync {} {local_dir} ↔ {session_name}:{remote_dir} — copy {copies} files ({}){tail}",
        sync_direction_word(direction),
        human_bytes(bytes),
    )
}

/// Cap on per-file plan entries returned by a dry run.
const SYNC_PLAN_MAX_ENTRIES: usize = 200;

/// One-way directory sync through the transfer engine, mirroring the UI's
/// plan → confirm → execute flow. A dry run only walks both trees, so it's
/// classed as a READ (auto-approvable under auto_read) and returns the capped
/// plan without changing anything. Executing plans first, then gates ONCE as a
/// Write with the copy/delete counts in the summary — OpClass::Write is never
/// auto-approved by auto_read/auto_safe_exec, so a mirror sync's deletes always
/// face the user unless they opted into allow-all. Copies overwrite the
/// destination file (OverwritePolicy::Overwrite), exactly like the UI's sync.
#[allow(clippy::too_many_arguments)]
async fn op_sync(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    local_dir: &str,
    remote_dir: &str,
    direction: SyncDirection,
    strategy: SyncStrategy,
    dry_run: bool,
) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();

    // Opt-in check before ANY I/O: even planning walks the remote tree.
    // (The gate below re-checks; this just refuses earlier.)
    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": format!(
                    "connection '{}' has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it.",
                    name
                )
            }),
        );
    }

    // For a push the local side is the source and must exist; for a pull it's
    // the destination (the planner tolerates a missing/empty side).
    if matches!(direction, SyncDirection::LocalToRemote) {
        match tokio::fs::metadata(local_dir).await {
            Ok(m) if m.is_dir() => {}
            _ => {
                return (
                    400,
                    json!({"error": format!("localDir {local_dir} does not exist or is not a directory")}),
                )
            }
        }
    }

    let dir_word = sync_direction_word(direction);
    let strategy_word = sync_strategy_word(strategy);

    if dry_run {
        // Gate the dry run as a read BEFORE walking anything, so the tree
        // listing itself is covered by the user's read policy.
        if let Err(resp) = gate(
            app,
            state,
            session_id,
            &name,
            OpClass::Read,
            "sync",
            &format!(
                "Plan sync {dir_word} {local_dir} ↔ {name}:{remote_dir} ({strategy_word}, dry run — no changes)"
            ),
            None,
        )
        .await
        {
            return resp;
        }
    }

    let local_fs: Box<dyn crate::remotefs::RemoteFs> = Box::new(crate::remotefs::local::LocalFs);
    let remote_fs = crate::commands::fs_for_session(&sess);
    let plan = match crate::sync::plan(
        local_fs.as_ref(),
        remote_fs.as_ref(),
        local_dir,
        remote_dir,
        direction,
        strategy,
    )
    .await
    {
        Ok(p) => p,
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("sync {dir_word} {local_dir} ↔ {remote_dir} — {e}"),
                        false,
                    ),
                )
                .await;
            return (500, json!({"error": e.to_string()}));
        }
    };

    let copy_count = plan.copies.len();
    let delete_count = plan.deletes.len();
    let total_bytes = plan.total_bytes;

    if dry_run {
        state
            .log(
                app,
                activity(
                    "sync",
                    session_id,
                    format!(
                        "sync dry-run {dir_word} {local_dir} ↔ {remote_dir} ({strategy_word}) — copy {copy_count} ({}), delete {delete_count}",
                        human_bytes(total_bytes)
                    ),
                    true,
                ),
            )
            .await;
        let copies: Vec<Value> = plan
            .copies
            .iter()
            .take(SYNC_PLAN_MAX_ENTRIES)
            .map(|c| json!({ "relative": c.relative, "size": c.size, "reason": c.reason }))
            .collect();
        let deletes: Vec<Value> = plan
            .deletes
            .iter()
            .take(SYNC_PLAN_MAX_ENTRIES)
            .map(|d| json!({ "relative": d.relative, "size": d.size }))
            .collect();
        return (
            200,
            json!({
                "dryRun": true,
                "direction": dir_word,
                "strategy": strategy_word,
                "localDir": plan.local_root,
                "remoteDir": plan.remote_root,
                "copyCount": copy_count,
                "deleteCount": delete_count,
                "totalBytes": total_bytes,
                "copies": copies,
                "deletes": deletes,
                "listTruncated": copy_count > SYNC_PLAN_MAX_ENTRIES || delete_count > SYNC_PLAN_MAX_ENTRIES,
            }),
        );
    }

    if copy_count == 0 && delete_count == 0 {
        // Nothing to do — don't raise an approval prompt for a no-op.
        state
            .log(
                app,
                activity(
                    "sync",
                    session_id,
                    format!("sync {dir_word} {local_dir} ↔ {remote_dir} — already in sync"),
                    true,
                ),
            )
            .await;
        return (
            200,
            json!({
                "transferIds": [],
                "copyCount": 0,
                "deleteCount": 0,
                "totalBytes": 0,
                "status": "in-sync",
            }),
        );
    }

    let summary = sync_gate_summary(
        direction,
        strategy,
        local_dir,
        &name,
        remote_dir,
        copy_count,
        total_bytes,
        delete_count,
    );
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Write,
        "sync",
        &summary,
        None,
    )
    .await
    {
        return resp;
    }

    let transfers = app.state::<AppState>().transfers.clone();
    match crate::commands::execute_sync_plan(sess.clone(), plan, &transfers, app).await {
        Ok(ids) => {
            state
                .log(
                    app,
                    activity(
                        "sync",
                        session_id,
                        format!(
                            "sync {dir_word} {local_dir} ↔ {remote_dir} ({strategy_word}) — {} copies ({}), {delete_count} deletes",
                            ids.len(),
                            human_bytes(total_bytes)
                        ),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({
                    "transferIds": ids,
                    "copyCount": copy_count,
                    "deleteCount": delete_count,
                    "totalBytes": total_bytes,
                    "status": "started",
                }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        session_id,
                        format!("sync {dir_word} {local_dir} ↔ {remote_dir} — {e}"),
                        false,
                    ),
                )
                .await;
            (500, json!({"error": e.to_string()}))
        }
    }
}

/// Cap on per-file diff entries returned to the agent. Differences beyond this
/// are elided (`listTruncated`); the summary counts always reflect the whole tree.
const DIFF_MAX_ENTRIES: usize = 500;

/// One resolved diff side: `None` is the local filesystem; `Some` is a connected,
/// opted-in server. Kept as `(id, name, session)` so the label, the gate, and the
/// hashing read all share one lookup.
type DiffSide = Option<(String, String, Arc<Session>)>;

/// Resolve one side of a diff. Absent or the literal `"local"` → the local
/// filesystem; anything else resolves to an enabled session (so a server that
/// hasn't granted agent access simply won't match — same rule as every other tool).
async fn resolve_diff_side(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_arg: Option<&str>,
) -> Result<DiffSide, String> {
    match session_arg {
        None => Ok(None),
        Some(s) if s.eq_ignore_ascii_case("local") => Ok(None),
        Some(s) => {
            let id = resolve_session(app, state, Some(s), SessionNeed::Any).await?;
            let sess = app
                .state::<AppState>()
                .sessions
                .get(&id)
                .await
                .ok_or_else(|| format!("session {id} went away"))?;
            let name = sess.profile().name.clone();
            Ok(Some((id, name, sess)))
        }
    }
}

fn diff_side_label(side: &DiffSide, path: &str) -> String {
    match side {
        None => format!("local:{path}"),
        Some((_, name, _)) => format!("{name}:{path}"),
    }
}

fn diff_side_fs(side: &DiffSide) -> Box<dyn crate::remotefs::RemoteFs> {
    match side {
        None => Box::new(crate::remotefs::local::LocalFs),
        Some((_, _, sess)) => crate::commands::fs_for_session(sess),
    }
}

/// Compare two directory trees across any two Ghost FTP backends — including
/// remote↔remote — and return the classified differences (Plan 6). A side is a
/// connected server (its session name/id) or the local filesystem (`session`
/// absent / "local"); at least one side must be a server so there's an opted-in
/// session to authorize against. Gated ONCE as a READ: the whole op only walks
/// (and, with `hash`, reads) — it never mutates. `hash` confirms same-size files
/// by content sha256 (server-side over SSH where possible). Same-class files are
/// summarized but omitted from `entries` to keep the response focused.
async fn op_diff(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    side_a_arg: Option<&str>,
    path_a: &str,
    side_b_arg: Option<&str>,
    path_b: &str,
    hash: bool,
) -> (u16, Value) {
    if path_a.is_empty() || path_b.is_empty() {
        return (400, json!({"error": "pathA and pathB are required"}));
    }
    let side_a = match resolve_diff_side(app, state, side_a_arg).await {
        Ok(v) => v,
        Err(msg) => return (400, json!({ "error": msg })),
    };
    let side_b = match resolve_diff_side(app, state, side_b_arg).await {
        Ok(v) => v,
        Err(msg) => return (400, json!({ "error": msg })),
    };

    // Need a server side to authorize against — a purely local↔local diff has no
    // session, and that's what `ghostftp-cli diff` is for.
    let gate_side = side_a.as_ref().or(side_b.as_ref());
    let Some((gate_id, gate_name, _)) = gate_side else {
        return (
            400,
            json!({"error": "ghostftp_diff needs at least one connected server side (a local↔local diff has no session to authorize); use the `ghostftp-cli diff` command for two local folders."}),
        );
    };
    let gate_id = gate_id.clone();
    let gate_name = gate_name.clone();

    let label_a = diff_side_label(&side_a, path_a);
    let label_b = diff_side_label(&side_b, path_b);
    let summary_text = format!(
        "Diff {label_a} ↔ {label_b}{}",
        if hash { " (hashing content)" } else { "" }
    );

    // Walking (and hashing) both trees is a read; gate before ANY I/O so the
    // listing itself is covered by the user's read policy.
    if let Err(resp) = gate(
        app,
        state,
        &gate_id,
        &gate_name,
        OpClass::Read,
        "diff",
        &summary_text,
        None,
    )
    .await
    {
        return resp;
    }

    let fs_a = diff_side_fs(&side_a);
    let fs_b = diff_side_fs(&side_b);
    let sess_a = side_a.as_ref().map(|(_, _, s)| s.as_ref());
    let sess_b = side_b.as_ref().map(|(_, _, s)| s.as_ref());

    let result = match crate::diff::diff(
        fs_a.as_ref(),
        path_a,
        sess_a,
        fs_b.as_ref(),
        path_b,
        sess_b,
        hash,
    )
    .await
    {
        Ok(r) => r,
        Err(e) => {
            state
                .log(
                    app,
                    activity(
                        "error",
                        &gate_id,
                        format!("diff {label_a} ↔ {label_b} — {e}"),
                        false,
                    ),
                )
                .await;
            return (500, json!({"error": format!("{e:#}")}));
        }
    };

    let s = &result.summary;
    state
        .log(
            app,
            activity(
                "diff",
                &gate_id,
                format!(
                    "diff {label_a} ↔ {label_b} — {} only in A, {} only in B, {} differ, {} same",
                    s.only_in_a, s.only_in_b, s.different, s.same
                ),
                true,
            ),
        )
        .await;

    // Only the actual differences go in `entries` (same-class files are noise for
    // an agent); the summary still counts them all.
    let diffs: Vec<&crate::diff::DiffEntry> = result
        .entries
        .iter()
        .filter(|e| e.class != crate::diff::DiffClass::Same)
        .collect();
    let list_truncated = diffs.len() > DIFF_MAX_ENTRIES;
    let entries: Vec<Value> = diffs
        .iter()
        .take(DIFF_MAX_ENTRIES)
        .filter_map(|e| serde_json::to_value(e).ok())
        .collect();

    (
        200,
        json!({
            "rootA": result.root_a,
            "rootB": result.root_b,
            "hashed": result.hashed,
            "summary": {
                "onlyInA": s.only_in_a,
                "onlyInB": s.only_in_b,
                "different": s.different,
                "same": s.same,
                "total": s.total,
            },
            "entries": entries,
            "listTruncated": list_truncated,
        }),
    )
}

const DEDUPE_MAX_GROUPS: usize = 200;

/// Find duplicate files in one tree on a connected server — the `name_1.ext`
/// copies Ghost FTP's rename-on-conflict policy creates, or (with `hash`) exact
/// content duplicates anywhere in the tree. Read-only: gated ONCE as a READ —
/// it walks (and, with `hash`, reads) but never mutates. Returns the groups
/// with a suggested `keep` index each; deleting the rest is the caller's
/// explicit follow-up (e.g. `ghostftp_exec` `rm`, or the GUI's dedupe panel).
async fn op_dedupe(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_arg: Option<&str>,
    path: &str,
    hash: bool,
) -> (u16, Value) {
    if path.is_empty() {
        return (400, json!({"error": "path is required"}));
    }
    let side = match resolve_diff_side(app, state, session_arg).await {
        Ok(v) => v,
        Err(msg) => return (400, json!({ "error": msg })),
    };
    let Some((gate_id, gate_name, sess)) = side else {
        return (
            400,
            json!({"error": "ghostftp_dedupe needs a connected server (a local-only scan has no session to authorize); use the `ghostftp-cli dedupe` command for local folders."}),
        );
    };
    let summary_text = format!(
        "Dedupe scan of {gate_name}:{path}{}",
        if hash { " (hashing content)" } else { "" }
    );
    if let Err(resp) = gate(
        app,
        state,
        &gate_id,
        &gate_name,
        OpClass::Read,
        "dedupe",
        &summary_text,
        None,
    )
    .await
    {
        return resp;
    }

    let fs = crate::commands::fs_for_session(&sess);
    let mode = if hash {
        crate::dedupe::DedupeMode::Hash
    } else {
        crate::dedupe::DedupeMode::Name
    };
    let result =
        match crate::dedupe::find_duplicates(fs.as_ref(), path, Some(sess.as_ref()), mode).await {
            Ok(r) => r,
            Err(e) => {
                state
                    .log(
                        app,
                        activity("error", &gate_id, format!("{summary_text} — {e}"), false),
                    )
                    .await;
                return (500, json!({"error": format!("{e:#}")}));
            }
        };

    let s = &result.summary;
    state
        .log(
            app,
            activity(
                "dedupe",
                &gate_id,
                format!(
                    "{summary_text} — {} groups, {} duplicates, {} B reclaimable",
                    s.groups, s.duplicate_files, s.wasted_bytes
                ),
                true,
            ),
        )
        .await;

    let groups_truncated = result.groups.len() > DEDUPE_MAX_GROUPS;
    let groups: Vec<Value> = result
        .groups
        .iter()
        .take(DEDUPE_MAX_GROUPS)
        .filter_map(|g| serde_json::to_value(g).ok())
        .collect();

    (
        200,
        json!({
            "root": result.root,
            "mode": result.mode,
            "summary": result.summary,
            "groups": groups,
            "groupsTruncated": groups_truncated,
            "note": "Read-only — nothing was deleted. To clean up, review each group and delete every file except the `keep` index (a suggestion: the unsuffixed name, else the oldest copy).",
        }),
    )
}

/// Build a [`crate::search::SearchQuery`] from a JSON body / MCP args object.
/// Shared by the HTTP `/search` handler, the `ghostftp_search` MCP dispatch, and the
/// agent tool router — all hand in a `serde_json::Value` object with the same
/// field names. The hit cap is clamped to `SEARCH_MAX_RESULTS` so an agent
/// response stays lean.
pub(crate) fn build_search_query(v: &Value, pattern: String) -> crate::search::SearchQuery {
    use crate::search::{SearchKind, SearchQuery, DEFAULT_MAX_FILE_BYTES};
    let regex = v.get("regex").and_then(|x| x.as_bool()).unwrap_or(false);
    // `--regex` (or explicit content) selects content grep; otherwise name search.
    let content = regex || v.get("content").and_then(|x| x.as_bool()).unwrap_or(false);
    let str_arr = |key: &str| {
        v.get(key)
            .and_then(|a| a.as_array())
            .map(|a| {
                a.iter()
                    .filter_map(|x| x.as_str().map(String::from))
                    .collect::<Vec<_>>()
            })
            .unwrap_or_default()
    };
    let max = v
        .get("maxResults")
        .and_then(|x| x.as_u64())
        .map(|n| (n as usize).clamp(1, SEARCH_MAX_RESULTS))
        .unwrap_or(SEARCH_MAX_RESULTS);
    SearchQuery {
        pattern,
        kind: if content {
            SearchKind::Content
        } else {
            SearchKind::Name
        },
        regex,
        case_sensitive: v
            .get("caseSensitive")
            .and_then(|x| x.as_bool())
            .unwrap_or(false),
        include_globs: str_arr("include"),
        exclude_globs: str_arr("exclude"),
        content_remote: v
            .get("contentRemote")
            .and_then(|x| x.as_bool())
            .unwrap_or(false),
        max_results: max,
        max_file_bytes: DEFAULT_MAX_FILE_BYTES,
    }
}

/// Search a connected server by file **name** or by **content** (Plan 7). Runs
/// through the shared search engine, so content grep uses the server-side
/// `rg`/`grep` fast path on SSH/agent and object stores name-match a flat key
/// listing. Gated ONCE as a READ (like `glob`/`tail`, it only walks/greps — it
/// never mutates).
pub(crate) async fn op_search(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
    root: &str,
    query: &crate::search::SearchQuery,
) -> (u16, Value) {
    use crate::search::SearchKind;
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let name = sess.profile().name.clone();
    let kind_label = match query.kind {
        SearchKind::Name => "name",
        SearchKind::Content => "content",
    };
    let summary = format!("{kind_label} search \"{}\" in {root}", query.pattern);
    if let Err(resp) = gate(
        app,
        state,
        session_id,
        &name,
        OpClass::Read,
        "search",
        &summary,
        None,
    )
    .await
    {
        return resp;
    }

    let fs = crate::commands::fs_for_session(&sess);
    match crate::search::search(fs.as_ref(), Some(sess.as_ref()), root, query).await {
        Ok(result) => {
            let matches: Vec<Value> = result
                .hits
                .iter()
                .filter_map(|h| serde_json::to_value(h).ok())
                .collect();
            state
                .log(
                    app,
                    activity(
                        "search",
                        session_id,
                        format!("{summary} ({} hits)", matches.len()),
                        true,
                    ),
                )
                .await;
            (
                200,
                json!({
                    "kind": kind_label,
                    "strategy": result.stats.strategy,
                    "truncated": result.stats.truncated,
                    "note": result.stats.note,
                    "matches": matches,
                }),
            )
        }
        Err(e) => {
            state
                .log(
                    app,
                    activity("error", session_id, format!("{summary} — {e}"), false),
                )
                .await;
            (400, json!({"error": format!("{e:#}")}))
        }
    }
}

/// Status of a transfer the agent previously started. No gate — the agent owns
/// the (random) transfer id, and this is local metadata about its own action.
async fn op_transfer_status(app: &AppHandle, transfer_id: &str) -> (u16, Value) {
    let transfers = app.state::<AppState>().transfers.clone();
    match transfers.snapshot(transfer_id).await {
        Some(t) => (
            200,
            json!({
                "transferId": t.id,
                "kind": t.kind,
                "status": t.status,
                "source": t.source,
                "destination": t.destination,
                "size": t.size,
                "transferred": t.transferred,
                "error": t.error,
            }),
        ),
        None => (
            404,
            json!({"error": "no transfer with that id (it may have been cleared)"}),
        ),
    }
}

/// Newest-first view of the agent's own activity log — the commands, reads,
/// transfers and denials that already ran through Ghost FTP. The whole log is
/// readable without per-op approval (it's the agent's own audit trail, already
/// scoped by the bearer token); when a `session` filter is given we require
/// that session to be opted-in, mirroring `op_server_info`.
async fn op_history(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_filter: Option<&str>,
    limit: usize,
) -> (u16, Value) {
    let limit = limit.clamp(1, MAX_ACTIVITY);
    let filter_id = if let Some(arg) = session_filter {
        match resolve_session(app, state, Some(arg), SessionNeed::Any).await {
            Ok(id) => {
                if !state.is_enabled(&id).await {
                    return (
                        403,
                        json!({
                            "error": "that connection has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it."
                        }),
                    );
                }
                Some(id)
            }
            Err(msg) => return (400, json!({ "error": msg })),
        }
    } else {
        None
    };

    let mut entries = state.recent_activity().await; // oldest-first
    entries.reverse(); // newest-first
    let out: Vec<Value> = entries
        .into_iter()
        .filter(|e| filter_id.as_deref().is_none_or(|id| e.session_id == id))
        .take(limit)
        .map(|e| {
            json!({
                "kind": e.kind,
                "detail": e.detail,
                "ok": e.ok,
                "at": e.at,
                "sessionId": e.session_id,
            })
        })
        .collect();
    (200, json!({ "history": out }))
}

/// Best-effort remote context for an SSH session. These are convenience values
/// for the agent; failures are silently ignored and returned as null.
async fn remote_context(app: &AppHandle, session_id: &str) -> Value {
    let manager = app.state::<AppState>().sessions.clone();
    // A Ghost FTP Agent machine already reported its OS/shell/home at connect — hand
    // those back directly so the AI writes commands for the right platform.
    if let Some(agent) = manager.get_agent(session_id).await {
        let info = &agent.system_info;
        return json!({
            "home": (!info.home_dir.is_empty()).then(|| info.home_dir.clone()),
            "cwd": null,
            "shell": (!info.shell.is_empty()).then(|| info.shell.clone()),
            "os": (!info.os.is_empty()).then(|| info.os.clone()),
        });
    }
    let Some(ssh) = manager.get_ssh(session_id).await else {
        return json!({ "home": null, "cwd": null, "shell": null, "os": null });
    };
    let home = ssh
        .exec_bounded("echo \"$HOME\"", 1024, Duration::from_secs(5), None)
        .await
        .ok()
        .filter(|o| o.exit_code == Some(0))
        .map(|o| o.stdout.trim().to_string())
        .filter(|s| !s.is_empty());
    let cwd = ssh
        .exec_bounded("pwd", 1024, Duration::from_secs(5), None)
        .await
        .ok()
        .filter(|o| o.exit_code == Some(0))
        .map(|o| o.stdout.trim().to_string())
        .filter(|s| !s.is_empty());
    let shell = ssh
        .exec_bounded("echo \"$SHELL\"", 1024, Duration::from_secs(5), None)
        .await
        .ok()
        .filter(|o| o.exit_code == Some(0))
        .map(|o| o.stdout.trim().to_string())
        .filter(|s| !s.is_empty());
    let os = ssh
        .exec_bounded("uname -s", 1024, Duration::from_secs(5), None)
        .await
        .ok()
        .filter(|o| o.exit_code == Some(0))
        .map(|o| o.stdout.trim().to_string())
        .filter(|s| !s.is_empty());
    json!({
        "home": home,
        "cwd": cwd,
        "shell": shell,
        "os": os,
    })
}

/// Context about a session — Ghost FTP-local metadata plus best-effort remote context
/// for SSH sessions. Gated on the per-session opt-in so we don't leak metadata
/// for sessions the user hasn't granted access to.
pub(crate) async fn op_server_info(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    session_id: &str,
) -> (u16, Value) {
    if !state.is_enabled(session_id).await {
        return (
            403,
            json!({
                "error": "that connection has not granted agent access. Ask the user to open Ghost FTP → Agent Bridge and toggle 'Allow agent access' for it."
            }),
        );
    }
    let manager = app.state::<AppState>().sessions.clone();
    let Some(sess) = manager.get(session_id).await else {
        return (
            400,
            json!({"error": "no enabled connection matches that id or name. Call ghostftp_context or ghostftp_list_sessions to see what's available, and make sure the connection has granted agent access."}),
        );
    };
    let p = sess.profile();
    let can_exec = matches!(&*sess, Session::Ssh(_) | Session::Agent(_));
    let remote = if can_exec {
        remote_context(app, session_id).await
    } else {
        json!({ "home": null, "cwd": null, "shell": null, "os": null })
    };
    (
        200,
        json!({
            "id": session_id,
            "name": p.name,
            "protocol": sess.protocol(),
            "host": p.host,
            "port": p.port,
            "username": p.username,
            "canExec": can_exec,
            "defaultRemotePath": p.default_remote_path,
            "remote": remote,
        }),
    )
}

/// Agent-facing overview: bridge state, current policy, enabled sessions, and
/// saved commands. This lets an agent discover what it can do in one call
/// instead of inferring it from multiple tools.
pub(crate) async fn op_context(app: &AppHandle, state: &Arc<BridgeState>) -> (u16, Value) {
    let manager = app.state::<AppState>().sessions.clone();
    let enabled: Vec<String> = state.enabled.lock().await.iter().cloned().collect();
    let mut sessions = Vec::new();
    for id in enabled {
        if let Some(sess) = manager.get(&id).await {
            let p = sess.profile();
            sessions.push(json!({
                "id": id,
                "name": p.name,
                "protocol": sess.protocol(),
                "host": p.host,
                "canExec": matches!(&*sess, Session::Ssh(_) | Session::Agent(_)),
            }));
        }
    }
    let policy = *state.policy.lock().await;
    let saved_commands: Vec<Value> = state
        .list_commands()
        .await
        .into_iter()
        .map(|c| {
            json!({
                "name": c.name,
                "description": c.description,
            })
        })
        .collect();
    let active_session = state.active_session_id.lock().await.clone();
    (
        200,
        json!({
            "bridgeRunning": state.running.lock().await.is_some(),
            "policy": {
                "allowAll": policy.allow_all,
                "autoRead": policy.auto_read,
                "autoSafeExec": policy.auto_safe_exec,
            },
            "activeSessionId": active_session,
            "sessions": sessions,
            "savedCommands": saved_commands,
        }),
    )
}

// ---- Skills runner (Plan 8) ----

/// How many targets a fleet run touches at once. Steps run in order *within* a
/// target; this bounds the fan-out *across* targets.
const SKILL_MAX_CONCURRENCY: usize = 4;
/// Per-stream cap on a step's stdout/stderr in the aggregated run result, so a
/// noisy fleet run doesn't balloon an MCP/CLI response (the live console still
/// shows the full, un-clipped output — this only trims the summary).
const SKILL_STEP_OUTPUT_CAP: usize = 16 * 1024;

/// Merge a Skill's declared defaults with the caller-provided values (provided
/// wins), then verify every required param resolved. The returned map is what
/// `${param}` placeholders substitute from.
fn build_param_map(
    skill: &Skill,
    provided: &HashMap<String, String>,
) -> Result<HashMap<String, String>, String> {
    let mut map: HashMap<String, String> = HashMap::new();
    for p in &skill.params {
        if let Some(d) = &p.default {
            map.insert(p.name.clone(), d.clone());
        }
    }
    for (k, v) in provided {
        map.insert(k.clone(), v.clone());
    }
    for p in &skill.params {
        if p.required && !map.contains_key(&p.name) {
            return Err(format!("missing required parameter '{}'", p.name));
        }
    }
    Ok(map)
}

/// Substitute `${name}` placeholders in a step's command template. Values are
/// inserted verbatim (not shell-escaped) — matching the raw-shell model of
/// `ghostftp_exec` and saved commands, where the author writes the shell. The dry-run
/// preview and the one confirm gate both show the fully resolved command, so any
/// interpolation is visible before it runs.
fn substitute(template: &str, params: &HashMap<String, String>) -> String {
    let mut out = template.to_string();
    for (k, v) in params {
        out = out.replace(&format!("${{{k}}}"), v);
    }
    out
}

/// One-line human summary of a Skill's default target selector (for tool
/// descriptions).
fn target_summary(sel: &TargetSelector) -> String {
    if sel.all {
        "Runs on ALL connected exec-capable servers by default.".to_string()
    } else if sel.sessions.is_empty() {
        "No default targets — pass `targets` to choose servers.".to_string()
    } else {
        format!("Default targets: {}.", sel.sessions.join(", "))
    }
}

/// Trim a stream to `max` bytes on a UTF-8 boundary, with a marker.
fn clip(s: &str, max: usize) -> String {
    if s.len() <= max {
        return s.to_string();
    }
    let mut end = max;
    while end > 0 && !s.is_char_boundary(end) {
        end -= 1;
    }
    format!("{}\n… [output clipped]", &s[..end])
}

/// Resolve a Skill's targets to concrete, enabled, exec-capable sessions.
/// `override_targets` (from the run call) wins over the skill's default selector;
/// the literal `"all"` (in either) expands to every enabled exec session. Returns
/// `(runnable, skipped)` — `skipped` names each requested target that couldn't
/// run and why, so a fan-out never silently drops a server.
async fn resolve_skill_targets(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    skill: &Skill,
    override_targets: Option<&[String]>,
) -> (Vec<(String, String, bool)>, Vec<(String, String)>) {
    let all = enabled_sessions(app, state).await; // (id, name, BackendKind)
    let mut runnable: Vec<(String, String, bool)> = Vec::new();
    let mut skipped: Vec<(String, String)> = Vec::new();

    let (use_all, explicit): (bool, Vec<String>) = match override_targets {
        Some(list) if !list.is_empty() => {
            if list.iter().any(|t| t.eq_ignore_ascii_case("all")) {
                (true, Vec::new())
            } else {
                (false, list.to_vec())
            }
        }
        _ => (skill.targets.all, skill.targets.sessions.clone()),
    };

    let classify = |kind: &BackendKind| -> Option<bool> {
        match kind {
            BackendKind::Ssh => Some(false),
            BackendKind::Agent => Some(true),
            BackendKind::Other => None,
        }
    };

    if use_all {
        for (id, name, kind) in &all {
            match classify(kind) {
                Some(is_agent) => runnable.push((id.clone(), name.clone(), is_agent)),
                None => skipped.push((name.clone(), "not exec-capable (read-only backend)".into())),
            }
        }
        return (runnable, skipped);
    }

    for t in explicit {
        match all
            .iter()
            .find(|(id, name, _)| *id == t || name.eq_ignore_ascii_case(&t))
        {
            Some((id, name, kind)) => match classify(kind) {
                Some(is_agent) => runnable.push((id.clone(), name.clone(), is_agent)),
                None => skipped.push((name.clone(), "not exec-capable (read-only backend)".into())),
            },
            None => skipped.push((
                t.clone(),
                "no connection with agent access matches this name/id".into(),
            )),
        }
    }
    // A name could resolve to the same session twice — dedupe by id.
    runnable.sort_by(|a, b| a.0.cmp(&b.0));
    runnable.dedup_by(|a, b| a.0 == b.0);
    (runnable, skipped)
}

/// Summarize one finished step into `(ok, json)`. `ok` folds exit code + timeout;
/// output streams are clipped for the aggregate (the live console keeps the full
/// text).
fn summarize_step(
    n: usize,
    label: &str,
    command: &str,
    status: u16,
    body: &Value,
) -> (bool, Value) {
    if status == 200 {
        let exit = body.get("exitCode").and_then(|v| v.as_i64());
        let timed_out = body
            .get("timedOut")
            .and_then(|v| v.as_bool())
            .unwrap_or(false);
        let ok = !timed_out && exit.unwrap_or(0) == 0;
        (
            ok,
            json!({
                "step": n,
                "name": label,
                "command": command,
                "ok": ok,
                "exitCode": exit,
                "stdout": clip(body.get("stdout").and_then(|v| v.as_str()).unwrap_or(""), SKILL_STEP_OUTPUT_CAP),
                "stderr": clip(body.get("stderr").and_then(|v| v.as_str()).unwrap_or(""), SKILL_STEP_OUTPUT_CAP),
                "truncated": body.get("truncated").and_then(|v| v.as_bool()).unwrap_or(false),
                "timedOut": timed_out,
            }),
        )
    } else {
        (
            false,
            json!({
                "step": n,
                "name": label,
                "command": command,
                "ok": false,
                "error": body.get("error").and_then(|v| v.as_str()).unwrap_or("error"),
            }),
        )
    }
}

/// Run a Skill's (already substituted) steps in order on one target, collecting a
/// per-step result. Each step reuses the exec path (`exec_core` / `exec_core_agent`)
/// so it streams to the Agent Console and lands in the audit log — the skill run
/// was already approved once, so no per-step gate. Stops the remaining steps on
/// the first failure when `stop_on_error`.
#[allow(clippy::too_many_arguments)]
async fn run_skill_on_target(
    app: AppHandle,
    state: Arc<BridgeState>,
    session_id: String,
    session_name: String,
    is_agent: bool,
    skill_name: String,
    steps: Vec<(String, String)>, // (label, resolved command)
    stop_on_error: bool,
    timeout: Duration,
) -> Value {
    let manager = app.state::<AppState>().sessions.clone();
    let mut step_results: Vec<Value> = Vec::new();
    let mut target_ok = true;
    for (i, (label, command)) in steps.iter().enumerate() {
        let n = i + 1;
        let console_label = format!("[{skill_name} · step {n}] {label}");
        let (status, body) = if is_agent {
            match manager.get_agent(&session_id).await {
                Some(agent) => {
                    exec_core_agent(
                        &app,
                        &state,
                        &agent,
                        &session_id,
                        &session_name,
                        command,
                        &console_label,
                        timeout,
                    )
                    .await
                }
                None => (500, json!({"error": "session went away"})),
            }
        } else {
            match manager.get_ssh(&session_id).await {
                Some(ssh) => {
                    exec_core(
                        &app,
                        &state,
                        &ssh,
                        &session_id,
                        &session_name,
                        command,
                        &console_label,
                        timeout,
                    )
                    .await
                }
                None => (500, json!({"error": "session went away"})),
            }
        };
        let (ok, sr) = summarize_step(n, label, command, status, &body);
        step_results.push(sr);
        if !ok {
            target_ok = false;
            if stop_on_error {
                break;
            }
        }
    }
    json!({
        "sessionId": session_id,
        "sessionName": session_name,
        "ok": target_ok,
        "steps": step_results,
    })
}

/// Run a Skill across its resolved targets (Plan 8). Validates params, resolves
/// targets (override wins over the skill's default), and either previews
/// (`dry_run`) or executes. A real run refuses a proposal (needs human approval),
/// gates ONCE over the whole fleet (only allow-all auto-approves), then fans out
/// with bounded concurrency and aggregates a per-target success/fail summary.
pub(crate) async fn op_run_skill(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    skill_ref: &str,
    params: HashMap<String, String>,
    target_override: Option<Vec<String>>,
    dry_run: bool,
) -> (u16, Value) {
    let Some(skill) = state.find_skill(skill_ref).await else {
        return (
            404,
            json!({"error": format!("no skill named '{skill_ref}'")}),
        );
    };

    let param_map = match build_param_map(&skill, &params) {
        Ok(m) => m,
        Err(msg) => return (400, json!({"error": msg})),
    };

    if skill.steps.is_empty() {
        return (400, json!({"error": "this skill has no steps"}));
    }

    let (runnable, skipped) =
        resolve_skill_targets(app, state, &skill, target_override.as_deref()).await;
    let skipped_json: Vec<Value> = skipped
        .iter()
        .map(|(t, r)| json!({"target": t, "reason": r}))
        .collect();

    // Resolve every step's command once (reused across targets).
    let steps: Vec<(String, String)> = skill
        .steps
        .iter()
        .map(|s| {
            let cmd = substitute(&s.command, &param_map);
            let label = if s.name.trim().is_empty() {
                cmd.clone()
            } else {
                s.name.clone()
            };
            (label, cmd)
        })
        .collect();

    // Dry run: pure string substitution — no server is contacted, so no gate and
    // no proposal block. Returns the resolved commands per target for preview.
    if dry_run {
        let auto = state.policy.lock().await.allow_all;
        let targets_json: Vec<Value> = runnable
            .iter()
            .map(|(id, name, _)| {
                json!({
                    "sessionId": id,
                    "sessionName": name,
                    "commands": steps.iter().map(|(_, c)| json!(c)).collect::<Vec<_>>(),
                })
            })
            .collect();
        return (
            200,
            json!({
                "dryRun": true,
                "skill": skill.name,
                "proposal": skill.status == SkillStatus::Proposed,
                "stepCount": steps.len(),
                "targets": targets_json,
                "skipped": skipped_json,
                "needsApproval": !auto,
            }),
        );
    }

    // A proposal is not runnable until a human approves it in the Skills panel.
    if skill.status == SkillStatus::Proposed {
        return (
            403,
            json!({"error": format!(
                "skill '{}' is a proposal awaiting human approval — approve it in Ghost FTP's Skills panel before running it",
                skill.name
            )}),
        );
    }

    if runnable.is_empty() {
        return (
            400,
            json!({
                "error": "no runnable targets for this skill — need at least one SSH or Ghost FTP Agent connection with agent access granted",
                "skipped": skipped_json,
            }),
        );
    }

    // One confirm gate covers the whole fleet run. A skill fans arbitrary
    // multi-step commands across many servers, so ONLY allow-all auto-approves —
    // the read-only safe-exec heuristic never applies to a whole skill.
    let target_names: Vec<&str> = runnable.iter().map(|(_, n, _)| n.as_str()).collect();
    let summary = format!(
        "Run skill \"{}\" on {} server{} ({}) — {} step{} each",
        skill.name,
        runnable.len(),
        if runnable.len() == 1 { "" } else { "s" },
        target_names.join(", "),
        steps.len(),
        if steps.len() == 1 { "" } else { "s" },
    );
    let auto = state.policy.lock().await.allow_all;
    if !auto {
        let (gate_id, gate_name, _) = &runnable[0];
        let approved = state
            .request_approval(app, gate_id, gate_name, "skill", &summary)
            .await;
        if !approved {
            state
                .log(app, activity("denied", gate_id, summary.clone(), false))
                .await;
            return (
                403,
                json!({"error": "the user denied or did not respond to the skill run approval in Ghost FTP. Ask before trying again."}),
            );
        }
    }

    // Fan out across targets with bounded concurrency.
    let timeout = EXEC_TIMEOUT;
    let concurrency = SKILL_MAX_CONCURRENCY.min(runnable.len().max(1));
    let sem = Arc::new(tokio::sync::Semaphore::new(concurrency));
    let mut set = tokio::task::JoinSet::new();
    for (id, name, is_agent) in runnable.clone() {
        let app = app.clone();
        let state = state.clone();
        let steps = steps.clone();
        let sem = sem.clone();
        let skill_name = skill.name.clone();
        let stop_on_error = skill.stop_on_error;
        set.spawn(async move {
            let _permit = sem.acquire().await;
            run_skill_on_target(
                app,
                state,
                id,
                name,
                is_agent,
                skill_name,
                steps,
                stop_on_error,
                timeout,
            )
            .await
        });
    }
    let mut results: Vec<Value> = Vec::new();
    while let Some(r) = set.join_next().await {
        if let Ok(v) = r {
            results.push(v);
        }
    }
    results.sort_by(|a, b| {
        a.get("sessionName")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .cmp(b.get("sessionName").and_then(|v| v.as_str()).unwrap_or(""))
    });
    let succeeded = results
        .iter()
        .filter(|r| r.get("ok").and_then(|v| v.as_bool()).unwrap_or(false))
        .count();
    let failed = results.len() - succeeded;

    state
        .log(
            app,
            activity(
                "skill",
                &runnable[0].0,
                format!(
                    "skill \"{}\" on {} server(s) — {} ok, {} failed",
                    skill.name,
                    results.len(),
                    succeeded,
                    failed
                ),
                failed == 0,
            ),
        )
        .await;

    (
        200,
        json!({
            "skill": skill.name,
            "status": "completed",
            "targetCount": results.len(),
            "succeeded": succeeded,
            "failed": failed,
            "results": results,
            "skipped": skipped_json,
        }),
    )
}

// ---- MCP (Model Context Protocol) over Streamable HTTP ----
//
// One stateless JSON-RPC endpoint. Claude Code connects with:
//   claude mcp add --transport http ghostftp http://127.0.0.1:<port>/mcp \
//     --header "Authorization: Bearer <token>"
// and auto-discovers the ghostftp_* tools below.

async fn handle_mcp(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    body: &[u8],
    stream: &mut TcpStream,
) -> Result<()> {
    let req: Value = match serde_json::from_slice(body) {
        Ok(v) => v,
        Err(_) => return write_jsonrpc_error(stream, Value::Null, -32700, "parse error").await,
    };

    let id = req.get("id").cloned();
    let method = req.get("method").and_then(|m| m.as_str()).unwrap_or("");
    let params = req.get("params").cloned().unwrap_or(Value::Null);

    // Notifications carry no id and expect no response body.
    let Some(id) = id else {
        return write_empty(stream, 202).await;
    };

    let result: Result<Value, (i64, String)> = match method {
        "initialize" => Ok(mcp_initialize(&params)),
        "ping" => Ok(json!({})),
        "tools/list" => Ok(mcp_tools_list(state).await),
        "tools/call" => Ok(mcp_tools_call(app, state, &params).await),
        other => Err((-32601, format!("method not found: {other}"))),
    };

    match result {
        Ok(value) => write_jsonrpc_result(stream, id, value).await,
        Err((code, message)) => write_jsonrpc_error(stream, id, code, &message).await,
    }
}

/// Server-level guidance returned in the `initialize` result. MCP clients
/// (Claude Code, Cursor, …) inject this into the agent's context once, before
/// any tool is called — so it's the right place to establish the authorization
/// frame for the whole session. Without it the only context the agent has is
/// the per-tool descriptions, and a burst of exec/search/read calls can read
/// like reconnaissance against an unknown target rather than what it actually
/// is: the operator administering infrastructure they own and have connected.
const MCP_INSTRUCTIONS: &str = "\
You are helping the user inside Ghost FTP — a desktop SSH/SFTP file client (like \
FileZilla + PuTTY) running on their own computer. Every connection exposed here \
is one the user has personally set up, authenticated, and explicitly opted in \
to agent access. The bridge holds the user's own credentials; you never see or \
handle them.

Operating context:
- This is routine file management and debugging on the user's own computers and \
servers, initiated by the user. The user owns or operates every machine you can \
see here. Reading config and logs, listing directories, and searching the \
filesystem are normal, authorized diagnostic steps — treat them as you would the \
user typing the same command into their own terminal.
- Every action is gated: it is either approved interactively by the user in Ghost FTP \
at call time, or pre-authorized via a policy or saved command they configured. \
An action that runs has the user's consent.
- All activity is shown live in Ghost FTP's Agent Bridge console, so the user sees \
exactly what you do.

Work carefully: prefer read-only inspection first, explain what you're checking \
and why, and avoid destructive operations unless the user clearly asked for them. \
Keep actions scoped to the connection the user intended.";

fn mcp_initialize(params: &Value) -> Value {
    let version = params
        .get("protocolVersion")
        .and_then(|v| v.as_str())
        .unwrap_or("2024-11-05");
    json!({
        "protocolVersion": version,
        "capabilities": { "tools": {} },
        "serverInfo": {
            "name": "ghostftp-agent-bridge",
            "version": env!("CARGO_PKG_VERSION"),
        },
        "instructions": MCP_INSTRUCTIONS,
    })
}

async fn mcp_tools_list(state: &Arc<BridgeState>) -> Value {
    let session_prop = json!({
        "type": "string",
        "description": "Session id or name. Optional when only one session is available."
    });
    let mut base = json!({
        "tools": [
            {
                "name": "ghostftp_context",
                "description": "Get the agent's overall context in Ghost FTP: whether the bridge is running, the current auto-approval policy, the user's enabled connections, and any saved commands. Call this first in every session to discover what you can do and whether commands will need user approval.",
                "inputSchema": { "type": "object", "properties": {}, "additionalProperties": false }
            },
            {
                "name": "ghostftp_list_sessions",
                "description": "List the user's own connections in Ghost FTP (a desktop SSH/SFTP file client, like FileZilla + PuTTY) that they've authorized for agent access. Use this to discover session ids/names and which support diagnostic commands (SSH). Prefer ghostftp_context unless you only need the list.",
                "inputSchema": { "type": "object", "properties": {}, "additionalProperties": false }
            },
            {
                "name": "ghostftp_server_info",
                "description": "Get context about one of the user's connected computers or servers: protocol, host, port, username, default remote path, whether it supports diagnostic commands (SSH), and best-effort remote context (home directory, current working directory, shell, OS) for SSH sessions.",

                "inputSchema": {
                    "type": "object",
                    "properties": { "session": session_prop },
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_exec",
                "description": "Run a diagnostic or status command on a connected computer in Ghost FTP — the user's own machine, which they have already authenticated to and authorized for agent access. Works on SSH servers and on Ghost FTP Agent machines (a paired Windows/Linux box running ghostftp-agentd); on a Ghost FTP Agent machine the command runs in the daemon's native shell (PowerShell on Windows, sh elsewhere), so write commands for that OS — check ghostftp_server_info for which. This is equivalent to the user typing the command into Ghost FTP's built-in terminal. The user approves each command in Ghost FTP (or has pre-approved this kind in their settings). Returns stdout, stderr and exit code; output is capped at 512 KiB. Set dryRun=true to preview whether the command would need approval without running it.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "command": { "type": "string", "description": "The command to run. Keep it non-interactive (no pagers/prompts). Prefer read-only status and diagnostic commands. `sudo` is supported: when the user has enabled it, Ghost FTP answers sudo's password prompt with their connection password — so just write `sudo <cmd>` normally; do NOT add `-S`, a password, or `echo <pw> |`." },
                        "dryRun": { "type": "boolean", "description": "If true, return a preview of what would run and whether it would need approval, without executing." },
                        "timeoutMs": { "type": "integer", "description": "Optional timeout in milliseconds for commands that legitimately run long (builds, backups). Default 60000; clamped to [1000, 900000]. The 512 KiB output cap is unchanged." },
                        "detach": { "type": "boolean", "description": "Launch the command as a BACKGROUND job and return a jobId immediately instead of waiting — for multi-minute work (backfills, migrations) that would blow the timeout. Poll it with ghostftp_job. Works on SSH servers and paired Ghost FTP Agent machines. On SSH the job keeps running server-side even across bridge restarts; on a Ghost FTP Agent it runs until the job finishes or the daemon restarts. A Ghost FTP Agent must run ghostftp-agentd from Plan 10 or newer." },
                        "session": session_prop
                    },
                    "required": ["command"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_job",
                "description": "Poll a background job started by ghostftp_exec with detach=true (on an SSH server or a paired Ghost FTP Agent). Returns whether it's still running, its exit code once finished, and its captured stdout/stderr (each capped at 512 KiB). Call repeatedly until running is false.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "jobId": { "type": "string", "description": "The jobId returned by a detached ghostftp_exec." },
                        "session": session_prop
                    },
                    "required": ["jobId"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_exec_script",
                "description": "Run a whole multi-line script on a connected computer in Ghost FTP (SSH server or paired Ghost FTP Agent machine) WITHOUT the base64/heredoc/quoting gymnastics a single-line `ghostftp_exec` forces. Pass the script's exact source in `script`; it runs verbatim in the target's native shell (sh on POSIX/SSH, sh or PowerShell on a Ghost FTP Agent — check ghostftp_server_info) so heredocs, nested quotes and newlines all survive. Use this instead of `ghostftp_exec` whenever the program spans multiple lines or contains quoting that is awkward to escape. Always prompts for approval (a script is never auto-approved by the safe-read-only heuristic). Returns stdout, stderr and exit code; output is capped at 512 KiB.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "script": { "type": "string", "description": "The full script source to run, verbatim (multi-line allowed). It is NOT re-parsed at a shell boundary, so write it exactly as you would a local .sh/.ps1 file." },
                        "label": { "type": "string", "description": "Optional friendly name shown in Ghost FTP's console + audit log (e.g. a filename or purpose). The script body itself is never dumped there." },
                        "dryRun": { "type": "boolean", "description": "If true, return a preview (byte/line count, whether it would need approval) without running." },
                        "timeoutMs": { "type": "integer", "description": "Optional timeout in milliseconds. Default 60000; clamped to [1000, 900000]." },
                        "session": session_prop
                    },
                    "required": ["script"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_list_dir",
                "description": "List a directory on a server the user has open in Ghost FTP (their own machine; SFTP, FTP or S3). Returns entries with name, path, kind, size and modified time.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote directory path. Defaults to \".\"." },
                        "session": session_prop
                    },
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_read_file",
                "description": "Read a text file on the user's connected server. Works on ANY protocol — SSH/SFTP and paired Ghost FTP Agent machines read it in place; FTP/FTPS, S3/Azure, WebDAV, Dropbox/OneDrive/Drive/Box and the SaaS backends fetch the head of the file over the wire, so there is no need to download first. Output is capped at 256 KiB; check the `truncated` flag.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote file path to read." },
                        "session": session_prop
                    },
                    "required": ["path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_search",
                "description": "Search a connected server by file NAME or by CONTENT (grep), recursively under a root path. Works on ANY protocol. NAME search (default) matches each entry's name — a pattern with '*'/'?' is a glob (e.g. '*.log'), otherwise a case-insensitive substring; it returns files AND directories. CONTENT search (set content=true, or regex=true) greps inside files: on SSH / Ghost FTP Agent servers it runs ripgrep/grep SERVER-SIDE (fast, no download) and returns matching lines with line numbers + previews; on object stores / FTP / WebDAV / cloud it must DOWNLOAD each file, so it's refused unless you set contentRemote=true. Use include/exclude name globs to scope which files are considered. Read-only. Results are capped (see truncated); the `strategy` field says which path ran (shell = server-side grep, generic = walk, objectFlat = bucket listing).",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "query": { "type": "string", "description": "What to look for: a name glob/substring, or (with content/regex) the grep pattern." },
                        "path": { "type": "string", "description": "Root directory to search under. Defaults to \".\"." },
                        "content": { "type": "boolean", "description": "Grep file CONTENTS instead of matching names. Default false." },
                        "regex": { "type": "boolean", "description": "Treat the content pattern as a regular expression (implies content). Default false = literal." },
                        "caseSensitive": { "type": "boolean", "description": "Case-sensitive matching. Default false." },
                        "include": { "type": "array", "items": { "type": "string" }, "description": "Only consider files whose name matches one of these globs (e.g. ['*.rs','*.toml'])." },
                        "exclude": { "type": "array", "items": { "type": "string" }, "description": "Skip files whose name matches any of these globs." },
                        "contentRemote": { "type": "boolean", "description": "Allow content search to DOWNLOAD every file on backends with no server-side grep (object stores, FTP, WebDAV, cloud). Default false." },
                        "session": session_prop
                    },
                    "required": ["query"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_read_files_batch",
                "description": "Read several text files from a connected server in one call. Works on ANY protocol. Each file is capped at 256 KiB and the total response is capped at 1 MiB. Returns an array of file objects; entries that fail to read include an `error` field instead of `content`.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "paths": { "type": "array", "items": { "type": "string" }, "description": "Remote file paths to read (max 50)." },
                        "session": session_prop
                    },
                    "required": ["paths"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_glob",
                "description": "Find files on a connected SSH server whose names match a shell glob pattern (e.g. '/var/log/nginx/*.log'), recursively under a root path. SSH-only — it runs `find` over the shell; on a Ghost FTP Agent machine use ghostftp_exec with a native command instead (e.g. Get-ChildItem on Windows). Bounded in depth and results.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "pattern": { "type": "string", "description": "Glob pattern to match against file names, e.g. '*.log' or 'error.*'." },
                        "path": { "type": "string", "description": "Root directory to search under. Defaults to \".\"." },
                        "session": session_prop
                    },
                    "required": ["pattern"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_tail",
                "description": "Stream the tail of a log file from a connected SSH server for up to 30 seconds. SSH-only. Output is forwarded live to Ghost FTP's Agent Console and returned when the stream ends (either the timeout or the connection closing). Use this to watch logs in real time.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote log file path to tail." },
                        "lines": { "type": "integer", "description": "Number of trailing lines to start with (default 50)." },
                        "session": session_prop
                    },
                    "required": ["path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_download",
                "description": "Download a file from the user's connected server to their own machine via Ghost FTP's transfer engine. Returns a transferId; the transfer also appears in Ghost FTP's transfer panel.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote file path to download." },
                        "localDir": { "type": "string", "description": "Local destination directory. Defaults to the user's Downloads folder." },
                        "session": session_prop
                    },
                    "required": ["path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_upload",
                "description": "Upload a local file to a directory on the user's connected server via Ghost FTP's transfer engine. Returns a transferId. By default a name collision is RENAMED (uploading shell.php next to an existing shell.php lands as shell_1.php, leaving the original untouched) — when the point is to REPLACE the remote file, set overwrite=true.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "localPath": { "type": "string", "description": "Path to the local file to upload." },
                        "remoteDir": { "type": "string", "description": "Remote destination directory." },
                        "overwrite": { "type": "boolean", "description": "Replace an existing remote file of the same name. Default false (renames to name_1)." },
                        "session": session_prop
                    },
                    "required": ["localPath", "remoteDir"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_delete",
                "description": "Delete a file or directory on the user's connected server. Works on ANY protocol (SSH/SFTP, FTP, S3/Azure, WebDAV, cloud drives, Ghost FTP Agent) — you do NOT need a shell, and you should never ask the user to delete a file by hand in a file manager. Directories need recursive=true. Destructive and irreversible: the user approves it in Ghost FTP (never auto-approved except by allow-all) and the prompt shows the exact path. When the file might still be wanted (suspected malware, a config you're replacing), prefer ghostftp_rename to move it aside instead.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote path to delete." },
                        "recursive": { "type": "boolean", "description": "Required to delete a directory: removes everything inside it. Default false." },
                        "session": session_prop
                    },
                    "required": ["path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_rename",
                "description": "Rename or move a file/directory on the user's connected server (same connection, server-side — no download/upload round trip). Works on any protocol that supports rename. Use it to move a suspicious file out of the web root, to stage a replacement, or to keep a backup before overwriting. Approved as a write in Ghost FTP.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "from": { "type": "string", "description": "Existing remote path." },
                        "to": { "type": "string", "description": "New remote path (including the new name)." },
                        "session": session_prop
                    },
                    "required": ["from", "to"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_mkdir",
                "description": "Create a directory on the user's connected server. One level only — the parent must already exist. Works on any protocol with directories. Approved as a write in Ghost FTP.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Remote directory path to create." },
                        "session": session_prop
                    },
                    "required": ["path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_write",
                "description": "Write text straight into a file on the user's connected server — ANY protocol, no local staging file, no upload step. Use this to drop a small debug script, a config snippet, or a one-file patch directly on the server, and to REPLACE a file's contents in place (SSH streams it via SFTP, a Ghost FTP Agent via a ranged write, FTP/S3/WebDAV/cloud via an in-place upload). By default it will NOT overwrite an existing file — set overwrite=true to replace. Prefer this over ghostftp_upload when replacing an existing remote file: upload renames around a collision, write replaces. Approved as a write in Ghost FTP (never auto-approved except by allow-all). Content is bounded by a ~1 MiB request cap; for larger files use ghostftp_upload. Returns the path and bytes written.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": { "type": "string", "description": "Absolute remote file path to write." },
                        "content": { "type": "string", "description": "The exact text to write into the file (verbatim; written as UTF-8)." },
                        "overwrite": { "type": "boolean", "description": "Replace the file if it already exists. Default false (a collision errors)." },
                        "session": session_prop
                    },
                    "required": ["path", "content"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_upload_dir",
                "description": "Upload a whole local directory tree into a directory on the user's connected server via Ghost FTP's transfer engine. The local directory is recreated INSIDE remoteDir (uploading /a/dist to /srv gives /srv/dist/…); remote subdirectories are created automatically and one transfer is queued per file. The user approves the WHOLE tree once — the prompt shows the file count, total size and overwrite mode. By default existing remote files are kept and colliding uploads are renamed (_1, _2, …); set overwrite=true to replace them. Returns transferIds plus counts; poll ghostftp_transfer_status to confirm completion.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "localDir": { "type": "string", "description": "Path to the local directory to upload. Must exist and be a directory." },
                        "remoteDir": { "type": "string", "description": "Remote destination directory the local directory is created inside." },
                        "overwrite": { "type": "boolean", "description": "Replace existing remote files instead of renaming the new ones. Default false." },
                        "session": session_prop
                    },
                    "required": ["localDir", "remoteDir"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_sync",
                "description": "One-way sync between a local directory and a directory on the user's connected server, through Ghost FTP's transfer engine. direction 'push' (default) copies local → remote; 'pull' copies remote → local. Only missing, newer or size-changed files are copied, and each copy OVERWRITES the destination file. strategy 'additive' (default) never deletes anything; 'mirror' ALSO DELETES destination files that don't exist on the source — destructive, so only use it when the user explicitly wants an exact mirror. ALWAYS run with dryRun=true first and show the user the plan: a dry run only lists both trees (a read — nothing changes) and returns the per-file plan plus copy/delete/byte totals. Executing asks the user to approve the WHOLE plan once, with those counts in the prompt. Returns transferIds (deletes are applied in-call); poll ghostftp_transfer_status for the copies.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "localDir": { "type": "string", "description": "Local directory (the source for push, the destination for pull)." },
                        "remoteDir": { "type": "string", "description": "Remote directory (the destination for push, the source for pull)." },
                        "direction": { "type": "string", "enum": ["push", "pull"], "description": "push = local → remote (default), pull = remote → local." },
                        "strategy": { "type": "string", "enum": ["additive", "mirror"], "description": "additive = copy only, never delete (default). mirror = also DELETE destination files missing from the source." },
                        "dryRun": { "type": "boolean", "description": "Plan only — return what would copy/delete without changing anything. Do this first." },
                        "session": session_prop
                    },
                    "required": ["localDir", "remoteDir"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_diff",
                "description": "Compare two directory trees and get back the classified differences — which files are only on side A, only on side B, or present on both but differing. Works across ANY two of the user's connected backends, including remote↔remote (staging vs prod, two servers, two buckets), which a local diff tool can't do. Each side is a connected server (its name/id) plus a path, or the local filesystem (omit the session, or pass \"local\"); at least one side must be a server. By default files are compared by SIZE (cheap, no download); set hash=true to also confirm same-size files by content sha256 (server-side over SSH where possible, otherwise it reads the bytes — slower, so use it when a size match isn't conclusive). Read-only: it never changes anything, so it's gated as a read. Returns a summary (onlyInA/onlyInB/different/same counts) and the differing entries (same files are counted but omitted; the list is capped, see listTruncated). Use it to answer 'what's different between these two trees?' and then act (e.g. ghostftp_sync the files that differ).",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "sessionA": { "type": "string", "description": "Side A connection (name or id). Omit, or pass \"local\", for the local filesystem." },
                        "pathA": { "type": "string", "description": "Directory path on side A." },
                        "sessionB": { "type": "string", "description": "Side B connection (name or id). Omit, or pass \"local\", for the local filesystem." },
                        "pathB": { "type": "string", "description": "Directory path on side B." },
                        "hash": { "type": "boolean", "description": "Confirm same-size files by content hash (sha256). Default false (size only)." }
                    },
                    "required": ["pathA", "pathB"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_dedupe",
                "description": "Find duplicate files in one directory tree on a connected server — including the `name_1.ext` copies Ghost FTP's rename-on-conflict policy creates when uploading/downloading over existing files. Two modes: by default it groups files in the same folder whose names collapse to the same stem once copy suffixes (`_1`, ` (1)`, ` - copy`) are stripped AND whose sizes match (cheap, metadata only — no download); with hash=true it instead content-hashes every same-size file in the tree (sha256, server-side over SSH where possible) and groups by digest — slower, but catches exact duplicates with unrelated names anywhere in the tree. Read-only and gated as a read: it never deletes anything. Returns a summary (groups, duplicateFiles, wastedBytes) and the groups; each group lists its files plus a suggested `keep` index (the unsuffixed name, else the oldest copy). To clean up, review the groups and delete the non-keeper files yourself — the user approves deletions separately, so always show them what you plan to remove first.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "session": { "type": "string", "description": "The connection (name or id) to scan." },
                        "path": { "type": "string", "description": "Directory path to scan for duplicates." },
                        "hash": { "type": "boolean", "description": "Group by content hash (sha256) instead of name+size. Default false." }
                    },
                    "required": ["session", "path"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_transfer_status",
                "description": "Check whether a transfer (started via ghostftp_download, ghostftp_upload or ghostftp_upload_dir) has finished. Returns status (queued|transferring|done|skipped|error|canceled), bytes transferred, and any error. Poll this after starting a transfer to confirm success.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "transferId": { "type": "string", "description": "A transferId returned by ghostftp_download, ghostftp_upload or ghostftp_upload_dir." }
                    },
                    "required": ["transferId"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_history",
                "description": "Review the recent Agent Bridge activity log — the commands, reads, transfers and denials that have already run through Ghost FTP on the user's servers, newest first. Use it to recall what you did earlier in this session or to confirm an action was approved. Optionally filter by `session`.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "session": session_prop,
                        "limit": { "type": "integer", "description": "Max entries to return (default 50, max 200)." }
                    },
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_list_commands",
                "description": "List the user's SAVED commands — named, pre-approved commands the user defined in Ghost FTP. Prefer running one of these by name (ghostftp_run_command) over composing a raw command: it runs with no approval prompt and exactly as the user vetted it. Returns each command's name, the exact command it runs, and a description.",

                "inputSchema": { "type": "object", "properties": {}, "additionalProperties": false }
            },
            {
                "name": "ghostftp_run_command",
                "description": "Run one of the user's SAVED commands (see ghostftp_list_commands) by name on a connected SSH server. The exact command string was written and pre-approved by the user, so this runs immediately with NO approval prompt. You supply only the name and the target connection — never the command text. Prefer this over ghostftp_exec whenever a saved command fits the task. Set dryRun=true to preview the command without running it.",

                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string", "description": "The saved command's name (see ghostftp_list_commands)." },
                        "timeoutMs": { "type": "integer", "description": "Optional timeout in milliseconds. Default 60000; clamped to [1000, 900000]." },
                        "session": session_prop
                    },
                    "required": ["name"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_list_skills",
                "description": "List the user's saved SKILLS — named, parameterized, multi-step workflows that fan shell commands across one or more of the user's connected servers (a Skill is the fleet-automation layer above a saved command). Returns each skill's name, description, parameters, step count, default targets, and status (approved = runnable; proposed = an AI-authored skill still awaiting the user's approval). Each APPROVED skill is also exposed as its own skill_<name> tool. Use this to discover what fleet workflows exist before composing raw commands.",
                "inputSchema": { "type": "object", "properties": {}, "additionalProperties": false }
            },
            {
                "name": "ghostftp_run_skill",
                "description": "Run one of the user's saved SKILLS by name across one or many connected servers. Provide `params` for the skill's declared parameters and, optionally, `targets` to override which servers it runs on (names/ids, or [\"all\"] for every exec-capable connection); omit `targets` to use the skill's configured default. ALWAYS run with dryRun=true first and show the user the resolved commands per target — a dry run substitutes params and lists what would run without contacting any server. A real run asks the user to approve the WHOLE fleet run once (unless they've enabled allow-all), then executes each step in order on every target, returning a per-target success/fail summary. A proposed (AI-authored) skill can be dry-run but must be human-approved in Ghost FTP before it will actually run.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string", "description": "The skill's name (see ghostftp_list_skills)." },
                        "params": { "type": "object", "description": "Values for the skill's declared parameters, as a { name: value } object.", "additionalProperties": { "type": "string" } },
                        "targets": { "type": "array", "items": { "type": "string" }, "description": "Override which servers to run on: connection names/ids, or [\"all\"] for every exec-capable connection. Omit to use the skill's default targets." },
                        "dryRun": { "type": "boolean", "description": "Preview the resolved commands per target without running anything. Do this first." }
                    },
                    "required": ["name"],
                    "additionalProperties": false
                }
            },
            {
                "name": "ghostftp_save_skill",
                "description": "Compose and save a new SKILL — a named, parameterized, multi-step shell workflow — for the user to run across their fleet. This is how you author fleet automations: describe the steps as shell command templates using ${paramName} placeholders, declare the parameters, and set default targets. The skill is saved as a PROPOSAL: it does NOT run until the user reviews and approves it in Ghost FTP's Skills panel (you cannot approve your own skill). After saving, tell the user a proposal is waiting for their approval. Keep skills linear (a simple ordered list of steps) and prefer safe, idempotent commands.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "name": { "type": "string", "description": "Short, unique skill name (used as its skill_<name> tool once approved)." },
                        "description": { "type": "string", "description": "What the skill does and when to use it." },
                        "params": {
                            "type": "array",
                            "description": "Declared parameters the steps interpolate via ${name}.",
                            "items": {
                                "type": "object",
                                "properties": {
                                    "name": { "type": "string" },
                                    "description": { "type": "string" },
                                    "required": { "type": "boolean" },
                                    "default": { "type": "string" }
                                },
                                "required": ["name"]
                            }
                        },
                        "steps": {
                            "type": "array",
                            "description": "Ordered shell steps. Each command may use ${param} placeholders.",
                            "items": {
                                "type": "object",
                                "properties": {
                                    "name": { "type": "string", "description": "Optional label for the step." },
                                    "command": { "type": "string", "description": "Shell command template to run on each target." }
                                },
                                "required": ["command"]
                            }
                        },
                        "targets": {
                            "type": "object",
                            "description": "Default servers to run on.",
                            "properties": {
                                "all": { "type": "boolean", "description": "Run on every exec-capable connection." },
                                "sessions": { "type": "array", "items": { "type": "string" }, "description": "Specific connection names/ids." }
                            }
                        },
                        "stopOnError": { "type": "boolean", "description": "Halt a target's remaining steps after the first failing step. Default true." }
                    },
                    "required": ["name", "steps"],
                    "additionalProperties": false
                }
            }
        ]
    });

    // Expose each APPROVED skill as its own `skill_<name>` tool so the agent can
    // invoke it directly (the "MCPs create skills" surface). Proposals stay
    // hidden until a human approves them.
    if let Some(arr) = base.get_mut("tools").and_then(|t| t.as_array_mut()) {
        for skill in state.list_skills().await {
            if skill.status == SkillStatus::Approved {
                arr.push(skill_tool_def(&skill));
            }
        }
    }
    base
}

/// MCP tool name for a skill: `skill_<slug>` (the plan's `skill:<name>` — colons
/// aren't allowed in MCP tool names, so non-alphanumerics collapse to `_`).
fn skill_tool_name(name: &str) -> String {
    let slug: String = name
        .trim()
        .chars()
        .map(|c| {
            if c.is_ascii_alphanumeric() {
                c.to_ascii_lowercase()
            } else {
                '_'
            }
        })
        .collect();
    format!("skill_{}", slug.trim_matches('_'))
}

/// Build the per-skill MCP tool definition: one property per declared parameter,
/// plus `targets` / `dryRun` overrides.
fn skill_tool_def(skill: &Skill) -> Value {
    let mut props = serde_json::Map::new();
    let mut required: Vec<Value> = Vec::new();
    for p in &skill.params {
        props.insert(
            p.name.clone(),
            json!({
                "type": "string",
                "description": if p.description.trim().is_empty() {
                    format!("Parameter {}", p.name)
                } else {
                    p.description.clone()
                }
            }),
        );
        if p.required {
            required.push(json!(p.name));
        }
    }
    props.insert(
        "targets".into(),
        json!({
            "type": "array",
            "items": { "type": "string" },
            "description": "Override which servers to run on (names/ids, or [\"all\"]). Omit to use the skill's default targets."
        }),
    );
    props.insert(
        "dryRun".into(),
        json!({
            "type": "boolean",
            "description": "Preview the resolved commands per target without running anything. Do this first."
        }),
    );
    let desc = format!(
        "Run the user's saved Skill \"{}\"{}. A Skill is a pre-authored, multi-step workflow that fans shell commands across the user's connected servers. {} It has {} step(s). Always dryRun=true first to preview, then run (the user approves the whole fleet run once).",
        skill.name,
        if skill.description.trim().is_empty() {
            String::new()
        } else {
            format!(" — {}", skill.description)
        },
        target_summary(&skill.targets),
        skill.steps.len(),
    );
    json!({
        "name": skill_tool_name(&skill.name),
        "description": desc,
        "inputSchema": {
            "type": "object",
            "properties": Value::Object(props),
            "required": required,
            "additionalProperties": false
        }
    })
}

fn arg_str(args: &Value, key: &str) -> Option<String> {
    args.get(key)
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
}

/// Wrap an op's `(status, json)` result as an MCP tool response.
fn mcp_wrap(res: (u16, Value)) -> Value {
    let (status, body) = res;
    if status == 200 {
        tool_text(serde_json::to_string_pretty(&body).unwrap_or_default())
    } else {
        tool_error(
            body.get("error")
                .and_then(|v| v.as_str())
                .unwrap_or("error"),
        )
    }
}

async fn mcp_tools_call(app: &AppHandle, state: &Arc<BridgeState>, params: &Value) -> Value {
    let name = params.get("name").and_then(|v| v.as_str()).unwrap_or("");
    let args = params
        .get("arguments")
        .cloned()
        .unwrap_or_else(|| json!({}));
    let session_arg = args.get("session").and_then(|v| v.as_str());

    // Tools that don't need a session.
    if name == "ghostftp_list_sessions" {
        let (_, body) = handle_sessions(app, state).await;
        return tool_text(serde_json::to_string_pretty(&body).unwrap_or_default());
    }
    if name == "ghostftp_list_commands" {
        return tool_text(
            serde_json::to_string_pretty(&json!({ "commands": state.list_commands().await }))
                .unwrap_or_default(),
        );
    }
    if name == "ghostftp_context" {
        let (_, body) = op_context(app, state).await;
        return tool_text(serde_json::to_string_pretty(&body).unwrap_or_default());
    }
    if name == "ghostftp_list_skills" {
        return tool_text(
            serde_json::to_string_pretty(&skills_overview(state).await).unwrap_or_default(),
        );
    }

    match name {
        "ghostftp_run_command" => {
            let Some(cmd_name) = arg_str(&args, "name") else {
                return tool_error("`name` is required");
            };
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            let timeout_ms = args.get("timeoutMs").and_then(|v| v.as_u64());
            // Saved commands run through the SSH exec path only.
            let session_id =
                match resolve_session(app, state, session_arg, SessionNeed::SshOnly).await {
                    Ok(id) => id,
                    Err(msg) => return tool_error(&msg),
                };
            let (status, body) =
                op_run_command(app, state, &session_id, &cmd_name, dry_run, timeout_ms).await;
            if status == 200 {
                if dry_run {
                    tool_text(serde_json::to_string_pretty(&body).unwrap_or_default())
                } else {
                    tool_text(format_exec_result(&body))
                }
            } else {
                tool_error(
                    body.get("error")
                        .and_then(|v| v.as_str())
                        .unwrap_or("error"),
                )
            }
        }
        "ghostftp_exec" => {
            let Some(command) = arg_str(&args, "command") else {
                return tool_error("`command` is required");
            };
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            let timeout_ms = args.get("timeoutMs").and_then(|v| v.as_u64());
            let detach = args
                .get("detach")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            // Exec works on SSH servers and paired Ghost FTP Agent machines alike.
            let session_id = match resolve_session(app, state, session_arg, SessionNeed::Exec).await
            {
                Ok(id) => id,
                Err(msg) => return tool_error(&msg),
            };
            let (status, body) = if detach {
                op_exec_detached(app, state, &session_id, &command, dry_run).await
            } else {
                exec_on(app, state, &session_id, &command, dry_run, timeout_ms).await
            };
            if status == 200 {
                if dry_run || detach {
                    tool_text(serde_json::to_string_pretty(&body).unwrap_or_default())
                } else {
                    tool_text(format_exec_result(&body))
                }
            } else {
                tool_error(
                    body.get("error")
                        .and_then(|v| v.as_str())
                        .unwrap_or("error"),
                )
            }
        }
        "ghostftp_job" => {
            let Some(job_id) = arg_str(&args, "jobId") else {
                return tool_error("`jobId` is required");
            };
            // SSH servers and Ghost FTP Agent machines both run detached jobs now.
            let session_id = match resolve_session(app, state, session_arg, SessionNeed::Exec).await
            {
                Ok(id) => id,
                Err(msg) => return tool_error(&msg),
            };
            mcp_wrap(op_job(app, state, &session_id, &job_id).await)
        }
        "ghostftp_exec_script" => {
            let Some(script) = arg_str(&args, "script") else {
                return tool_error("`script` is required");
            };
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            let timeout_ms = args.get("timeoutMs").and_then(|v| v.as_u64());
            let label = arg_str(&args, "label")
                .filter(|s| !s.trim().is_empty())
                .unwrap_or_else(|| "a script".to_string());
            let session_id = match resolve_session(app, state, session_arg, SessionNeed::Exec).await
            {
                Ok(id) => id,
                Err(msg) => return tool_error(&msg),
            };
            let (status, body) = exec_script_on(
                app,
                state,
                &session_id,
                &script,
                &label,
                dry_run,
                timeout_ms,
            )
            .await;
            if status == 200 {
                if dry_run {
                    tool_text(serde_json::to_string_pretty(&body).unwrap_or_default())
                } else {
                    tool_text(format_exec_result(&body))
                }
            } else {
                tool_error(
                    body.get("error")
                        .and_then(|v| v.as_str())
                        .unwrap_or("error"),
                )
            }
        }
        "ghostftp_server_info" => {
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_server_info(app, state, &id).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_list_dir" => {
            let path = arg_str(&args, "path").unwrap_or_else(|| ".".to_string());
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_list_dir(app, state, &id, &path).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_read_file" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            // read_file has both an SFTP and a Ghost FTP Agent daemon path.
            match resolve_session(app, state, session_arg, SessionNeed::Exec).await {
                Ok(id) => mcp_wrap(op_read_file(app, state, &id, &path).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_search" => {
            let Some(pattern) = arg_str(&args, "query") else {
                return tool_error("`query` is required");
            };
            let root = arg_str(&args, "path").unwrap_or_else(|| ".".to_string());
            let query = build_search_query(&args, pattern);
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_search(app, state, &id, &root, &query).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_read_files_batch" => {
            let paths = args
                .get("paths")
                .and_then(|v| v.as_array())
                .map(|arr| {
                    arr.iter()
                        .filter_map(|v| v.as_str().map(String::from))
                        .collect::<Vec<_>>()
                })
                .unwrap_or_default();
            if paths.is_empty() {
                return tool_error("`paths` is required");
            }
            // Batch reads go through the SFTP subsystem — SSH only.
            match resolve_session(app, state, session_arg, SessionNeed::SshOnly).await {
                Ok(id) => mcp_wrap(op_read_file_batch(app, state, &id, paths).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_glob" => {
            let Some(pattern) = arg_str(&args, "pattern") else {
                return tool_error("`pattern` is required");
            };
            let root = arg_str(&args, "path").unwrap_or_else(|| ".".to_string());
            // glob shells out to `find` — SSH only.
            match resolve_session(app, state, session_arg, SessionNeed::SshOnly).await {
                Ok(id) => mcp_wrap(op_glob(app, state, &id, &root, &pattern).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_tail" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            let lines = args.get("lines").and_then(|v| v.as_u64()).unwrap_or(50) as usize;
            // tail streams `tail -f` over the SSH shell — SSH only.
            match resolve_session(app, state, session_arg, SessionNeed::SshOnly).await {
                Ok(id) => mcp_wrap(op_tail(app, state, &id, &path, lines).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_download" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            let local_dir = arg_str(&args, "localDir");
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_download(app, state, &id, &path, local_dir).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_upload" => {
            let (Some(local_path), Some(remote_dir)) =
                (arg_str(&args, "localPath"), arg_str(&args, "remoteDir"))
            else {
                return tool_error("`localPath` and `remoteDir` are required");
            };
            let overwrite = args
                .get("overwrite")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => {
                    mcp_wrap(op_upload(app, state, &id, &local_path, &remote_dir, overwrite).await)
                }
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_delete" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            let recursive = args
                .get("recursive")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_delete(app, state, &id, &path, recursive).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_rename" => {
            let (Some(from), Some(to)) = (arg_str(&args, "from"), arg_str(&args, "to")) else {
                return tool_error("`from` and `to` are required");
            };
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_rename(app, state, &id, &from, &to).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_mkdir" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(op_mkdir(app, state, &id, &path).await),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_write" => {
            let (Some(path), Some(content)) = (arg_str(&args, "path"), arg_str(&args, "content"))
            else {
                return tool_error("`path` and `content` are required");
            };
            let overwrite = args
                .get("overwrite")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            // SSH writes via SFTP, a Ghost FTP Agent via WriteChunk, everything else
            // through a staged in-place upload — so any protocol qualifies.
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => {
                    mcp_wrap(op_write(app, state, &id, &path, content.as_bytes(), overwrite).await)
                }
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_upload_dir" => {
            let (Some(local_dir), Some(remote_dir)) =
                (arg_str(&args, "localDir"), arg_str(&args, "remoteDir"))
            else {
                return tool_error("`localDir` and `remoteDir` are required");
            };
            let overwrite = args
                .get("overwrite")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(
                    op_upload_dir(app, state, &id, &local_dir, &remote_dir, overwrite).await,
                ),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_sync" => {
            let (Some(local_dir), Some(remote_dir)) =
                (arg_str(&args, "localDir"), arg_str(&args, "remoteDir"))
            else {
                return tool_error("`localDir` and `remoteDir` are required");
            };
            let (direction, strategy) = match parse_sync_args(
                &arg_str(&args, "direction").unwrap_or_default(),
                &arg_str(&args, "strategy").unwrap_or_default(),
            ) {
                Ok(v) => v,
                Err(msg) => return tool_error(&msg),
            };
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            match resolve_session(app, state, session_arg, SessionNeed::Any).await {
                Ok(id) => mcp_wrap(
                    op_sync(
                        app,
                        state,
                        &id,
                        &local_dir,
                        &remote_dir,
                        direction,
                        strategy,
                        dry_run,
                    )
                    .await,
                ),
                Err(msg) => tool_error(&msg),
            }
        }
        "ghostftp_diff" => {
            let (Some(path_a), Some(path_b)) = (arg_str(&args, "pathA"), arg_str(&args, "pathB"))
            else {
                return tool_error("`pathA` and `pathB` are required");
            };
            // A missing / "local" session on a side means the local filesystem;
            // op_diff resolves each side and gates against a server side.
            let side_a = arg_str(&args, "sessionA");
            let side_b = arg_str(&args, "sessionB");
            let hash = args.get("hash").and_then(|v| v.as_bool()).unwrap_or(false);
            mcp_wrap(
                op_diff(
                    app,
                    state,
                    side_a.as_deref(),
                    &path_a,
                    side_b.as_deref(),
                    &path_b,
                    hash,
                )
                .await,
            )
        }
        "ghostftp_dedupe" => {
            let Some(path) = arg_str(&args, "path") else {
                return tool_error("`path` is required");
            };
            let hash = args.get("hash").and_then(|v| v.as_bool()).unwrap_or(false);
            mcp_wrap(op_dedupe(app, state, session_arg, &path, hash).await)
        }
        "ghostftp_transfer_status" => {
            let Some(transfer_id) = arg_str(&args, "transferId") else {
                return tool_error("`transferId` is required");
            };
            mcp_wrap(op_transfer_status(app, &transfer_id).await)
        }
        "ghostftp_history" => {
            let limit = args.get("limit").and_then(|v| v.as_u64()).unwrap_or(50) as usize;
            // session_arg is resolved + gated inside op_history only when present.
            mcp_wrap(op_history(app, state, session_arg, limit).await)
        }
        "ghostftp_run_skill" => {
            let Some(skill_name) = arg_str(&args, "name") else {
                return tool_error("`name` is required");
            };
            let params = params_from_value(args.get("params"));
            let targets = str_array(args.get("targets"));
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            mcp_wrap(op_run_skill(app, state, &skill_name, params, targets, dry_run).await)
        }
        "ghostftp_save_skill" => match serde_json::from_value::<Skill>(args.clone()) {
            Ok(def) => {
                if def.name.trim().is_empty() {
                    return tool_error("a skill `name` is required");
                }
                if def.steps.is_empty() || def.steps.iter().any(|s| s.command.trim().is_empty()) {
                    return tool_error("a skill needs at least one step with a non-empty command");
                }
                let saved = state.propose_skill(def).await;
                // Nudge the GUI to surface the new proposal.
                let _ = app.emit("bridge://skill-proposed", &saved);
                tool_text(format!(
                    "Saved skill \"{}\" as a PROPOSAL (id {}). It won't run until the user reviews and approves it in Ghost FTP's Skills panel (you can't approve your own skill). Ask the user to review and approve it; you can dry-run it in the meantime with ghostftp_run_skill (dryRun=true) to show what it would do.",
                    saved.name, saved.id
                ))
            }
            Err(e) => tool_error(&format!("invalid skill definition: {e}")),
        },
        // Per-skill tools: `skill_<slug>` invokes an approved skill directly.
        n if n.starts_with("skill_") => {
            let skills = state.list_skills().await;
            let Some(skill) = skills
                .iter()
                .find(|s| s.status == SkillStatus::Approved && skill_tool_name(&s.name) == n)
            else {
                return tool_error(&format!("unknown skill tool: {n}"));
            };
            // Pull declared param values out of the flat args object.
            let mut params: HashMap<String, String> = HashMap::new();
            for p in &skill.params {
                if let Some(v) = arg_str(&args, &p.name) {
                    params.insert(p.name.clone(), v);
                }
            }
            let targets = str_array(args.get("targets"));
            let dry_run = args
                .get("dryRun")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            mcp_wrap(op_run_skill(app, state, &skill.name, params, targets, dry_run).await)
        }
        other => tool_error(&format!("unknown tool: {other}")),
    }
}

/// Lean, agent-facing overview of the saved skills (name, description, params,
/// step count, default targets, status). Shared by `ghostftp_list_skills` (MCP) and
/// available to the REST/CLI surface via `/skills`.
async fn skills_overview(state: &Arc<BridgeState>) -> Value {
    let skills = state.list_skills().await;
    let out: Vec<Value> = skills
        .iter()
        .map(|s| {
            json!({
                "name": s.name,
                "description": s.description,
                "status": if s.status == SkillStatus::Approved { "approved" } else { "proposed" },
                "params": s.params.iter().map(|p| json!({
                    "name": p.name,
                    "description": p.description,
                    "required": p.required,
                    "default": p.default,
                })).collect::<Vec<_>>(),
                "stepCount": s.steps.len(),
                "targets": { "all": s.targets.all, "sessions": s.targets.sessions },
                "tool": if s.status == SkillStatus::Approved { Some(skill_tool_name(&s.name)) } else { None },
            })
        })
        .collect();
    json!({ "skills": out })
}

/// Backend class of an enabled session, for tool-capability routing.
#[derive(Clone, Copy, PartialEq, Eq)]
enum BackendKind {
    /// SSH/SFTP server — full shell + SFTP subsystem.
    Ssh,
    /// Paired Ghost FTP Agent machine — native exec + file reads via the daemon.
    Agent,
    /// FTP / object store — file operations only.
    Other,
}

/// What a tool needs from the target session.
#[derive(Clone, Copy, PartialEq, Eq)]
enum SessionNeed {
    /// Any protocol works (RemoteFs-based ops: list, search, transfers).
    Any,
    /// Runs commands / reads files natively: SSH servers and Ghost FTP Agent
    /// machines both qualify.
    Exec,
    /// Needs the SSH shell / SFTP subsystem specifically (glob's `find`,
    /// tail's streaming, batch SFTP reads, saved commands). Ghost FTP Agent
    /// machines do NOT qualify.
    SshOnly,
}

impl SessionNeed {
    fn satisfied_by(self, kind: BackendKind) -> bool {
        match self {
            SessionNeed::Any => true,
            SessionNeed::Exec => matches!(kind, BackendKind::Ssh | BackendKind::Agent),
            SessionNeed::SshOnly => matches!(kind, BackendKind::Ssh),
        }
    }
}

/// All enabled sessions as (id, name, backend kind).
async fn enabled_sessions(
    app: &AppHandle,
    state: &Arc<BridgeState>,
) -> Vec<(String, String, BackendKind)> {
    let manager = app.state::<AppState>().sessions.clone();
    let ids: Vec<String> = state.enabled.lock().await.iter().cloned().collect();
    let mut out = Vec::new();
    for id in ids {
        if let Some(sess) = manager.get(&id).await {
            let kind = match &*sess {
                Session::Ssh(_) => BackendKind::Ssh,
                Session::Agent(_) => BackendKind::Agent,
                _ => BackendKind::Other,
            };
            out.push((id, sess.profile().name.clone(), kind));
        }
    }
    out
}

/// Resolve the `session` argument to a concrete session id. `need` says what
/// the calling tool requires of the backend: `Exec` accepts SSH servers AND
/// paired Ghost FTP Agent machines (both run commands and read files natively);
/// `SshOnly` is for tools built on the SSH shell/SFTP subsystem. We resolve
/// against ALL enabled sessions first so a capability mismatch yields an
/// accurate "wrong kind of connection" error rather than a misleading
/// "no match".
async fn resolve_session(
    app: &AppHandle,
    state: &Arc<BridgeState>,
    arg: Option<&str>,
    need: SessionNeed,
) -> Result<String, String> {
    let all = enabled_sessions(app, state).await; // (id, name, kind)

    if let Some(a) = arg {
        return match all
            .iter()
            .find(|(id, name, _)| id == a || name.eq_ignore_ascii_case(a))
        {
            Some((id, _, kind)) => {
                if need.satisfied_by(*kind) {
                    Ok(id.clone())
                } else {
                    Err(match need {
                        SessionNeed::Exec => format!(
                            "session \"{a}\" can't run this tool — it needs an SSH or Ghost FTP Agent connection; use list_dir/search/download/upload for this server"
                        ),
                        _ if *kind == BackendKind::Agent => format!(
                            "session \"{a}\" is a Ghost FTP Agent machine — this tool is SSH-only; use ghostftp_exec with a native command there instead"
                        ),
                        _ => format!(
                            "session \"{a}\" is not an SSH session — this tool needs SSH/SFTP; use list_dir/download/upload for this server"
                        ),
                    })
                }
            }
            None => Err(format!("no enabled session matches \"{a}\"")),
        };
    }

    let candidates: Vec<&(String, String, BackendKind)> = all
        .iter()
        .filter(|(_, _, kind)| need.satisfied_by(*kind))
        .collect();
    match candidates.len() {
        0 if !all.is_empty() => Err(match need {
            SessionNeed::SshOnly => {
                "none of the enabled sessions are SSH — this tool is SSH-only; on a Ghost FTP Agent machine use ghostftp_exec instead".into()
            }
            _ => {
                "none of the enabled sessions can run commands — this needs an SSH or Ghost FTP Agent connection; use list_dir/download/upload instead".into()
            }
        }),
        0 => Err(
            "no server has granted agent access — ask the user to enable it in Ghost FTP's Agent Bridge panel"
                .into(),
        ),
        1 => Ok(candidates[0].0.clone()),
        _ => {
            let names: Vec<&str> = candidates.iter().map(|(_, n, _)| n.as_str()).collect();
            Err(format!(
                "multiple sessions are available ({}); pass the `session` argument",
                names.join(", ")
            ))
        }
    }
}

fn tool_text(text: String) -> Value {
    json!({ "content": [ { "type": "text", "text": text } ] })
}

fn tool_error(msg: &str) -> Value {
    json!({ "content": [ { "type": "text", "text": format!("Error: {msg}") } ], "isError": true })
}

fn format_exec_result(body: &Value) -> String {
    let stdout = body.get("stdout").and_then(|v| v.as_str()).unwrap_or("");
    let stderr = body.get("stderr").and_then(|v| v.as_str()).unwrap_or("");
    let code = body.get("exitCode").and_then(|v| v.as_i64());
    let truncated = body
        .get("truncated")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    let timed_out = body
        .get("timedOut")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);
    let mut out = format!(
        "exit code: {}\n",
        code.map(|c| c.to_string())
            .unwrap_or_else(|| "unknown".into())
    );
    if timed_out {
        out.push_str("NOTE: command timed out before finishing — output is partial and the exit code is unknown.\n");
    }
    if truncated {
        out.push_str("NOTE: output was truncated (exceeded the size cap).\n");
    }
    if !stdout.is_empty() {
        out.push_str("\n--- stdout ---\n");
        out.push_str(stdout);
    }
    if !stderr.is_empty() {
        out.push_str("\n--- stderr ---\n");
        out.push_str(stderr);
    }
    out
}

async fn write_raw(
    stream: &mut TcpStream,
    status: u16,
    content_type: &str,
    body: &[u8],
) -> Result<()> {
    let reason = match status {
        200 => "OK",
        202 => "Accepted",
        405 => "Method Not Allowed",
        _ => "OK",
    };
    let head = format!(
        "HTTP/1.1 {status} {reason}\r\nContent-Type: {content_type}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    );
    stream.write_all(head.as_bytes()).await?;
    if !body.is_empty() {
        stream.write_all(body).await?;
    }
    stream.flush().await?;
    Ok(())
}

async fn write_empty(stream: &mut TcpStream, status: u16) -> Result<()> {
    write_raw(stream, status, "application/json", b"").await
}

async fn write_jsonrpc_result(stream: &mut TcpStream, id: Value, result: Value) -> Result<()> {
    let env = json!({ "jsonrpc": "2.0", "id": id, "result": result });
    write_raw(stream, 200, "application/json", &serde_json::to_vec(&env)?).await
}

async fn write_jsonrpc_error(
    stream: &mut TcpStream,
    id: Value,
    code: i64,
    message: &str,
) -> Result<()> {
    let env = json!({ "jsonrpc": "2.0", "id": id, "error": { "code": code, "message": message } });
    write_raw(stream, 200, "application/json", &serde_json::to_vec(&env)?).await
}

#[cfg(test)]
mod tests {
    use super::*;

    // Plan 10 Phase 1: the CLI base64-encodes a script's raw bytes; the bridge
    // decodes them back. Prove that hop is byte-exact for a heredoc + nested
    // quotes + trailing-newline script (the content that breaks a command line).
    #[test]
    fn exec_script_base64_roundtrip_is_byte_exact() {
        use base64::Engine as _;
        let script = "#!/bin/sh\ncat <<'EOF'\nnested 'single' and \"double\" quotes\nEOF\n";
        let b64 = base64::engine::general_purpose::STANDARD.encode(script);
        let decoded = base64::engine::general_purpose::STANDARD
            .decode(b64.as_bytes())
            .unwrap();
        assert_eq!(String::from_utf8(decoded).unwrap(), script);
    }

    #[test]
    fn script_summary_has_header_and_survives_multibyte_truncation() {
        // Short script → header + full body.
        let s = script_approval_summary("script foo.sh", "echo hi\necho bye");
        assert!(
            s.starts_with("Run script foo.sh (16 bytes, 2 lines):"),
            "got: {s}"
        );
        assert!(s.contains("echo bye"));
        // Long script with a multibyte char straddling the 4 KiB cap must not
        // panic on a char-boundary split.
        let big = format!("{}é tail", "a".repeat(4095));
        let s = script_approval_summary("big", &big);
        assert!(s.contains("preview truncated"));
    }

    #[test]
    fn exec_timeout_defaults_and_clamps() {
        // Absent → the 60 s default.
        assert_eq!(exec_timeout_from(None), EXEC_TIMEOUT);
        // In range → taken verbatim.
        assert_eq!(exec_timeout_from(Some(5_000)), Duration::from_secs(5));
        assert_eq!(
            exec_timeout_from(Some(EXEC_TIMEOUT_MS_MAX)),
            Duration::from_millis(EXEC_TIMEOUT_MS_MAX)
        );
        // Below the floor → clamped up (a typo can't insta-kill commands).
        assert_eq!(
            exec_timeout_from(Some(1)),
            Duration::from_millis(EXEC_TIMEOUT_MS_MIN)
        );
        assert_eq!(
            exec_timeout_from(Some(0)),
            Duration::from_millis(EXEC_TIMEOUT_MS_MIN)
        );
        // Above the ceiling → clamped to 15 min.
        assert_eq!(
            exec_timeout_from(Some(3_600_000)),
            Duration::from_millis(EXEC_TIMEOUT_MS_MAX)
        );
    }

    #[test]
    fn human_bytes_scales() {
        assert_eq!(human_bytes(0), "0 B");
        assert_eq!(human_bytes(1023), "1023 B");
        assert_eq!(human_bytes(2 * 1024), "2.0 KB");
        assert_eq!(human_bytes(3 * 1024 * 1024 + 512 * 1024), "3.5 MB");
        assert_eq!(human_bytes(5 * 1024 * 1024 * 1024), "5.00 GB");
    }

    #[test]
    fn upload_dir_policy_defaults_to_rename() {
        assert_eq!(upload_overwrite_policy(false), OverwritePolicy::Rename);
        assert_eq!(upload_overwrite_policy(true), OverwritePolicy::Overwrite);
    }

    #[test]
    fn upload_dir_summary_names_counts_and_mode() {
        let s = upload_dir_summary("C:\\site\\dist", "prod", "/var/www", 12, 3_670_016, false);
        assert_eq!(
            s,
            "Upload directory C:\\site\\dist → prod:/var/www (12 files, 3.5 MB total, overwrite: no)"
        );
        let s = upload_dir_summary("./dist", "prod", "/var/www/", 1, 10, true);
        assert!(s.ends_with("(1 files, 10 B total, overwrite: yes)"));
    }

    #[test]
    fn sync_args_parse_with_defaults() {
        let (d, s) = parse_sync_args("", "").unwrap();
        assert!(matches!(d, SyncDirection::LocalToRemote));
        assert!(matches!(s, SyncStrategy::Additive));
        let (d, s) = parse_sync_args("pull", "mirror").unwrap();
        assert!(matches!(d, SyncDirection::RemoteToLocal));
        assert!(matches!(s, SyncStrategy::Mirror));
        assert!(parse_sync_args("sideways", "").is_err());
        assert!(parse_sync_args("", "destroy").is_err());
    }

    #[test]
    fn sync_summary_names_mirror_deletes() {
        let s = sync_gate_summary(
            SyncDirection::LocalToRemote,
            SyncStrategy::Mirror,
            "./dist",
            "prod",
            "/var/www/app",
            12,
            3_670_016,
            3,
        );
        assert_eq!(
            s,
            "Sync push ./dist ↔ prod:/var/www/app — copy 12 files (3.5 MB), delete 3 files on the destination (mirror)"
        );
    }

    #[test]
    fn sync_summary_promises_no_deletes_for_additive() {
        let s = sync_gate_summary(
            SyncDirection::RemoteToLocal,
            SyncStrategy::Additive,
            "C:\\backup",
            "prod",
            "/etc/nginx",
            2,
            2048,
            0,
        );
        assert_eq!(
            s,
            "Sync pull C:\\backup ↔ prod:/etc/nginx — copy 2 files (2.0 KB), no deletes (additive)"
        );
    }

    #[tokio::test]
    async fn count_local_tree_counts_nested_files() {
        let root = std::env::temp_dir().join(format!("ghostftp-bridge-test-{}", Uuid::new_v4()));
        let sub = root.join("a").join("b");
        std::fs::create_dir_all(&sub).unwrap();
        std::fs::write(root.join("one.txt"), b"12345").unwrap();
        std::fs::write(root.join("a").join("two.txt"), b"123").unwrap();
        std::fs::write(sub.join("three.txt"), b"1").unwrap();

        let (files, bytes) = count_local_tree(&root).await.unwrap();
        assert_eq!(files, 3);
        assert_eq!(bytes, 9);

        std::fs::remove_dir_all(&root).ok();
    }

    // ---- Skills (Plan 8) ----

    fn skill_with_params(params: Vec<SkillParam>) -> Skill {
        Skill {
            params,
            ..Default::default()
        }
    }

    #[test]
    fn param_map_defaults_and_overrides() {
        let skill = skill_with_params(vec![
            SkillParam {
                name: "service".into(),
                required: true,
                ..Default::default()
            },
            SkillParam {
                name: "signal".into(),
                default: Some("HUP".into()),
                ..Default::default()
            },
        ]);
        let mut provided = HashMap::new();
        provided.insert("service".to_string(), "nginx".to_string());
        let map = build_param_map(&skill, &provided).unwrap();
        assert_eq!(map.get("service").unwrap(), "nginx");
        // Declared default fills in when not provided.
        assert_eq!(map.get("signal").unwrap(), "HUP");
        // Provided wins over the declared default.
        provided.insert("signal".to_string(), "TERM".to_string());
        let map = build_param_map(&skill, &provided).unwrap();
        assert_eq!(map.get("signal").unwrap(), "TERM");
    }

    #[test]
    fn param_map_requires_required() {
        let skill = skill_with_params(vec![SkillParam {
            name: "service".into(),
            required: true,
            ..Default::default()
        }]);
        let err = build_param_map(&skill, &HashMap::new()).unwrap_err();
        assert!(err.contains("service"));
    }

    #[test]
    fn substitute_replaces_placeholders() {
        let mut params = HashMap::new();
        params.insert("service".to_string(), "nginx".to_string());
        params.insert("signal".to_string(), "HUP".to_string());
        assert_eq!(
            substitute("systemctl reload ${service} # ${signal}", &params),
            "systemctl reload nginx # HUP"
        );
        // An unresolved placeholder is left verbatim.
        assert_eq!(substitute("echo ${missing}", &params), "echo ${missing}");
    }

    #[test]
    fn migrate_command_seeds_single_step_approved_skill() {
        let cmd = SavedCommand {
            id: "c1".into(),
            name: "disk".into(),
            command: "df -h".into(),
            description: String::new(),
        };
        let skill = migrate_command_to_skill(&cmd);
        assert_eq!(skill.name, "disk");
        assert_eq!(skill.steps.len(), 1);
        assert_eq!(skill.steps[0].command, "df -h");
        assert_eq!(skill.status, SkillStatus::Approved);
        assert_eq!(skill.created_by, "user");
        assert!(!skill.id.is_empty());
    }

    #[test]
    fn skill_tool_name_sanitizes() {
        // Colons/spaces/dots collapse to underscores (MCP names allow no colon).
        assert_eq!(skill_tool_name("Restart Web"), "skill_restart_web");
        assert_eq!(skill_tool_name("rotate.logs"), "skill_rotate_logs");
        assert_eq!(skill_tool_name("  audit  "), "skill_audit");
    }

    #[test]
    fn clip_trims_on_char_boundary() {
        // Short strings pass through untouched.
        assert_eq!(clip("hello", 100), "hello");
        // A multi-byte char at the cut point isn't split.
        let s = "aé"; // 'a' = 1 byte, 'é' = 2 bytes
        let out = clip(s, 2);
        assert!(out.starts_with('a'));
        assert!(out.contains("clipped"));
    }

    fn one_step_skill(name: &str) -> Skill {
        Skill {
            name: name.into(),
            steps: vec![SkillStep {
                name: String::new(),
                command: "true".into(),
            }],
            ..Default::default()
        }
    }

    /// The safety crux: a hand-authored skill is born approved, but an
    /// AI-authored one is forced to a proposal it can't self-approve; only the
    /// (local-UI) approve path flips it to runnable.
    #[tokio::test]
    async fn skills_store_propose_approve_delete() {
        // No config_path (default) → persist() is a no-op, so this touches no disk.
        let state = BridgeState::default();

        // Hand-authored via the local UI path → approved.
        let list = state.upsert_skill(one_step_skill("restart")).await;
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].status, SkillStatus::Approved);
        let restart_id = list[0].id.clone();
        assert!(!restart_id.is_empty());

        // AI path: even if the definition claims approved/user, it's forced to a
        // proposal authored by "ai" with a fresh id.
        let sneaky = Skill {
            status: SkillStatus::Approved,
            created_by: "user".into(),
            id: "attacker-chosen".into(),
            ..one_step_skill("ai-skill")
        };
        let proposed = state.propose_skill(sneaky).await;
        assert_eq!(proposed.status, SkillStatus::Proposed);
        assert_eq!(proposed.created_by, "ai");
        assert_ne!(proposed.id, "attacker-chosen");

        // The human approve gate flips it to runnable.
        let found = state.find_skill("ai-skill").await.unwrap();
        assert_eq!(found.status, SkillStatus::Proposed);
        let list = state.approve_skill(&found.id).await;
        let ai = list.iter().find(|s| s.name == "ai-skill").unwrap();
        assert_eq!(ai.status, SkillStatus::Approved);

        // Delete removes exactly the targeted skill.
        let list = state.delete_skill(&restart_id).await;
        assert!(list.iter().all(|s| s.id != restart_id));
        assert_eq!(list.len(), 1);
    }
}
