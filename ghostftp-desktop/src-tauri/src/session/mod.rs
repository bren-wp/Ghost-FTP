pub mod agent;
pub mod boxdrive;
pub mod dropbox;
pub mod dynamics;
pub mod ftp;
pub mod gdrive;
pub mod http;
pub mod http_throttle;
pub mod hubspot;
pub mod object;
pub mod onedrive;
pub mod shopify;
pub mod webdav;
pub use agent::{agent_pair, AgentSession};
pub use boxdrive::{box_connect, BoxSession};
pub use dropbox::{dropbox_connect, DropboxSession};
pub use dynamics::{dynamics_connect, DynamicsSession};
pub use ftp::{ftp_connect, FtpSession};
pub use gdrive::{gdrive_connect, GDriveSession};
pub use http::{http_connect, HttpSession};
pub use hubspot::{hubspot_connect, HubSpotSession};
pub use object::{object_connect, ObjectSession};
pub use onedrive::{onedrive_connect, OneDriveSession};
pub use shopify::{shopify_connect, ShopifySession};
pub use webdav::{webdav_connect, WebdavSession};

use crate::known_hosts;
use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use russh::{client, Channel, ChannelMsg};
use russh_keys::key;
use russh_sftp::client::SftpSession;
use serde::Serialize;
use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tauri::{AppHandle, Emitter};
use tokio::sync::{oneshot, Mutex};
use uuid::Uuid;

// ---- Host-key verification plumbing ----
//
// The russh `Handler::check_server_key` callback runs on the connect path.
// When we encounter an unknown host we suspend that callback on a oneshot
// channel, emit `host://prompt` to the frontend, and resume when the user
// answers via the `respond_to_host_prompt` command.

#[derive(Debug, Clone, Copy, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum HostPromptKind {
    Unknown,
    Mismatch,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct HostPromptEvent {
    pub request_id: String,
    pub host: String,
    pub port: u16,
    pub key_type: String,
    pub fingerprint: String,
    pub stored_fingerprint: Option<String>,
    pub kind: HostPromptKind,
}

#[derive(Debug, Clone, Copy, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum HostDecision {
    /// Accept this session only. Do not persist to known_hosts.
    Accept,
    /// Accept and append to ~/.ssh/known_hosts.
    Trust,
    /// Refuse the connection.
    Reject,
}

#[derive(Default)]
pub struct HostPromptRegistry {
    pending: Mutex<HashMap<String, oneshot::Sender<HostDecision>>>,
}

impl HostPromptRegistry {
    pub fn new() -> Self {
        Self::default()
    }

    async fn register(&self) -> (String, oneshot::Receiver<HostDecision>) {
        let id = Uuid::new_v4().to_string();
        let (tx, rx) = oneshot::channel();
        self.pending.lock().await.insert(id.clone(), tx);
        (id, rx)
    }

    pub async fn resolve(&self, request_id: &str, decision: HostDecision) -> Result<()> {
        let tx = self
            .pending
            .lock()
            .await
            .remove(request_id)
            .ok_or_else(|| anyhow!("no pending host prompt {request_id}"))?;
        tx.send(decision)
            .map_err(|_| anyhow!("host prompt receiver dropped"))?;
        Ok(())
    }
}

// ---- Keyboard-interactive auth prompting ----
//
// A temp/expired password makes the server demand an immediate password change
// during *authentication* (via PAM, over the keyboard-interactive method),
// before any shell exists — so plain password auth just fails. We drive that
// exchange in `ssh_connect`: the prompts the server can't answer from the stored
// password (the "New password" / "Retype" rounds of a forced change) are
// surfaced to the user through this bridge, which mirrors the host-key prompt
// machinery above.

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AuthPromptField {
    pub prompt: String,
    pub echo: bool,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AuthPromptEvent {
    pub request_id: String,
    pub profile_id: String,
    pub host: String,
    pub name: String,
    pub instructions: String,
    pub prompts: Vec<AuthPromptField>,
}

#[derive(Default)]
pub struct AuthPromptRegistry {
    pending: Mutex<HashMap<String, oneshot::Sender<Option<Vec<String>>>>>,
}

impl AuthPromptRegistry {
    pub fn new() -> Self {
        Self::default()
    }

    async fn register(&self) -> (String, oneshot::Receiver<Option<Vec<String>>>) {
        let id = Uuid::new_v4().to_string();
        let (tx, rx) = oneshot::channel();
        self.pending.lock().await.insert(id.clone(), tx);
        (id, rx)
    }

    /// Resolve a pending prompt. `responses == None` means the user cancelled the
    /// dialog, which aborts authentication.
    pub async fn resolve(&self, request_id: &str, responses: Option<Vec<String>>) -> Result<()> {
        let tx = self
            .pending
            .lock()
            .await
            .remove(request_id)
            .ok_or_else(|| anyhow!("no pending auth prompt {request_id}"))?;
        tx.send(responses)
            .map_err(|_| anyhow!("auth prompt receiver dropped"))?;
        Ok(())
    }
}

/// Answers the server's keyboard-interactive prompts during auth. Implementations:
/// the GUI's `TauriAuthPrompter` (emits `auth://prompt` + awaits the user), and
/// `RejectAuthPrompter` (refuses — for silent reconnects and the CLI, where no
/// one is watching to type a new password).
#[async_trait]
pub trait AuthPrompter: Send + Sync {
    async fn prompt(
        &self,
        name: &str,
        instructions: &str,
        prompts: &[AuthPromptField],
    ) -> Result<Vec<String>>;

    /// Called once after an interactive exchange the user had to answer succeeds,
    /// so the GUI can offer to save a freshly-changed password. No-op otherwise.
    fn on_interactive_success(&self) {}
}

/// GUI auth prompter: emits `auth://prompt` and waits for `respond_to_auth_prompt`.
pub struct TauriAuthPrompter {
    app: AppHandle,
    registry: Arc<AuthPromptRegistry>,
    profile_id: String,
    host: String,
}

impl TauriAuthPrompter {
    pub fn new(
        app: AppHandle,
        registry: Arc<AuthPromptRegistry>,
        profile_id: String,
        host: String,
    ) -> Self {
        Self {
            app,
            registry,
            profile_id,
            host,
        }
    }
}

#[async_trait]
impl AuthPrompter for TauriAuthPrompter {
    async fn prompt(
        &self,
        name: &str,
        instructions: &str,
        prompts: &[AuthPromptField],
    ) -> Result<Vec<String>> {
        let (request_id, rx) = self.registry.register().await;
        let event = AuthPromptEvent {
            request_id,
            profile_id: self.profile_id.clone(),
            host: self.host.clone(),
            name: name.to_string(),
            instructions: instructions.to_string(),
            prompts: prompts.to_vec(),
        };
        self.app
            .emit("auth://prompt", event)
            .map_err(|_| anyhow!("failed to surface auth prompt"))?;
        match rx.await {
            Ok(Some(responses)) => Ok(responses),
            // Cancelled, or the window closed out from under the prompt.
            _ => Err(anyhow!("authentication cancelled")),
        }
    }

    fn on_interactive_success(&self) {
        // Signal that a password may have just been changed for this profile, so
        // the UI can offer to update the saved credential. Carries no secret.
        let _ = self.app.emit(
            "auth://changed",
            serde_json::json!({ "profileId": self.profile_id }),
        );
    }
}

/// Non-interactive prompter that refuses any prompt — used for silent reconnects
/// (no user to ask) and the CLI.
pub struct RejectAuthPrompter;

#[async_trait]
impl AuthPrompter for RejectAuthPrompter {
    async fn prompt(
        &self,
        _name: &str,
        _instructions: &str,
        _prompts: &[AuthPromptField],
    ) -> Result<Vec<String>> {
        Err(anyhow!(
            "this server requires interactive authentication, which isn't available here"
        ))
    }
}

/// Decides what to do with a server key on connect. Implementations: the
/// GUI's `TauriHostKeyVerifier` (emits a Tauri event + awaits the user's
/// click via oneshot), and the CLI's `CliHostKeyVerifier` (prompts on
/// stdin/stdout). A no-op `AutoTrustVerifier` exists for scripted scenarios
/// where the operator has explicitly opted out of prompting.
#[async_trait]
pub trait HostKeyVerifier: Send + Sync {
    async fn decide(
        &self,
        host: &str,
        port: u16,
        key_type: &str,
        fingerprint: &str,
        stored_fingerprint: Option<&str>,
        kind: HostPromptKind,
    ) -> Result<HostDecision, russh::Error>;
}

/// GUI verifier. Emits `host://prompt` and waits for the
/// `respond_to_host_prompt` command to fire a oneshot.
pub struct TauriHostKeyVerifier {
    app: AppHandle,
    prompts: Arc<HostPromptRegistry>,
}

impl TauriHostKeyVerifier {
    pub fn new(app: AppHandle, prompts: Arc<HostPromptRegistry>) -> Self {
        Self { app, prompts }
    }
}

#[async_trait]
impl HostKeyVerifier for TauriHostKeyVerifier {
    async fn decide(
        &self,
        host: &str,
        port: u16,
        key_type: &str,
        fingerprint: &str,
        stored_fingerprint: Option<&str>,
        kind: HostPromptKind,
    ) -> Result<HostDecision, russh::Error> {
        let (request_id, rx) = self.prompts.register().await;
        let event = HostPromptEvent {
            request_id,
            host: host.to_string(),
            port,
            key_type: key_type.to_string(),
            fingerprint: fingerprint.to_string(),
            stored_fingerprint: stored_fingerprint.map(|s| s.to_string()),
            kind,
        };
        if self.app.emit("host://prompt", event).is_err() {
            return Err(russh::Error::HUP);
        }
        rx.await.map_err(|_| russh::Error::HUP)
    }
}

/// A non-interactive host-key verifier used for transparent reconnects. On a
/// reconnect the server key is already in `known_hosts`, so `check_server_key`
/// short-circuits on `Match` *before* `decide()` is ever called; `decide()`
/// therefore only runs for an unknown or **changed** key — which a silent
/// background reconnect must refuse rather than auto-trust behind the user's
/// back. `reject_unknown` (the safe default) makes that explicit.
pub struct AutoTrustVerifier {
    reject_unknown: bool,
}

impl AutoTrustVerifier {
    /// Refuse any key not already trusted in `known_hosts`. Safe for silent
    /// reconnects: a matching key never reaches `decide()`; an unknown/changed
    /// one is rejected loudly instead of being trusted without the user.
    pub fn reject_unknown() -> Self {
        Self {
            reject_unknown: true,
        }
    }
}

#[async_trait]
impl HostKeyVerifier for AutoTrustVerifier {
    async fn decide(
        &self,
        _host: &str,
        _port: u16,
        _key_type: &str,
        _fingerprint: &str,
        _stored_fingerprint: Option<&str>,
        _kind: HostPromptKind,
    ) -> Result<HostDecision, russh::Error> {
        Ok(if self.reject_unknown {
            HostDecision::Reject
        } else {
            HostDecision::Accept
        })
    }
}

pub struct ClientHandler {
    host: String,
    port: u16,
    verifier: Arc<dyn HostKeyVerifier>,
}

#[async_trait]
impl client::Handler for ClientHandler {
    type Error = russh::Error;

    async fn check_server_key(
        &mut self,
        server_public_key: &key::PublicKey,
    ) -> Result<bool, Self::Error> {
        let status = known_hosts::check(&self.host, self.port, server_public_key);
        let fingerprint = known_hosts::fingerprint(server_public_key);
        let key_type = server_public_key.name().to_string();

        let (kind, stored) = match status {
            known_hosts::HostKeyStatus::Match => return Ok(true),
            known_hosts::HostKeyStatus::Unknown => (HostPromptKind::Unknown, None),
            known_hosts::HostKeyStatus::Mismatch { stored_fingerprint } => {
                (HostPromptKind::Mismatch, Some(stored_fingerprint))
            }
        };

        let decision = self
            .verifier
            .decide(
                &self.host,
                self.port,
                &key_type,
                &fingerprint,
                stored.as_deref(),
                kind,
            )
            .await?;

        match decision {
            HostDecision::Accept => Ok(true),
            HostDecision::Trust => {
                if let Err(e) = known_hosts::append(&self.host, self.port, server_public_key) {
                    tracing::warn!(?e, "failed to persist host key");
                }
                Ok(true)
            }
            HostDecision::Reject => Ok(false),
        }
    }
}

// ---- SshSession ----
//
// One TCP/SSH connection, shared between the SFTP browser and any number of
// terminal channels. This is the killer feature: single auth, two surfaces.
pub struct SshSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub handle: Mutex<client::Handle<ClientHandler>>,
    /// Bastion connection this session is tunneled through, when the profile
    /// has a `jump_host`. Held purely to keep the jump hop alive — dropping
    /// the handle would tear down the direct-tcpip channel the target
    /// session rides on. Replaced on every reconnect.
    jump_handle: Mutex<Option<client::Handle<ClientHandler>>>,
    /// Lazily-opened SFTP subsystem. Behind an `Option` so a reconnect can drop
    /// the stale channel and force the next caller to reopen it; handed out as
    /// an `Arc` so callers never borrow the session across an `.await`.
    sftp: Mutex<Option<Arc<Mutex<SftpSession>>>>,
    /// Serializes reconnects so a burst of failed ops triggers exactly one.
    reconnect_lock: Mutex<()>,
    /// Bumped on every successful reconnect; lets a waiter that lost the race
    /// notice another task already reconnected and skip doing it again.
    generation: AtomicU64,
    /// In-flight tracked commands, keyed by op_id, each holding the pgid needed
    /// to terminate it. Lets a long job be killed on demand or on disconnect even
    /// after the `exec_bounded` that launched it returned (e.g. on timeout).
    /// Terminal jobs are dropped once their `agent://job` event fires.
    jobs: Mutex<HashMap<String, JobHandle>>,
}

/// True only when an error means the SSH transport itself is gone (the run-loop
/// task ended) rather than a normal command/SFTP failure. We reconnect *only*
/// on these — a spurious reconnect would swap the handle out from under live
/// PTY terminals and in-flight transfers.
fn is_transport_dead(err: &anyhow::Error) -> bool {
    matches!(
        err.downcast_ref::<russh::Error>(),
        Some(
            russh::Error::SendError
                | russh::Error::Disconnect
                | russh::Error::HUP
                | russh::Error::KeepaliveTimeout
                | russh::Error::InactivityTimeout
        )
    )
}

impl SshSession {
    /// Wrap an already-opened russh client handle and a profile into a
    /// freshly-uuid'd session. Used by `SessionManager` and the CLI binary
    /// (where there's no manager — the CLI builds one session per command
    /// and drops it on exit).
    pub fn new(profile: ConnectionProfile, handle: client::Handle<ClientHandler>) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            profile,
            handle: Mutex::new(handle),
            jump_handle: Mutex::new(None),
            sftp: Mutex::new(None),
            reconnect_lock: Mutex::new(()),
            generation: AtomicU64::new(0),
            jobs: Mutex::new(HashMap::new()),
        }
    }

    /// Lazily open the SFTP subsystem on first use, returning a shared handle to
    /// it. If the transport has died (e.g. the session sat idle past the
    /// keepalive budget, or the server rebooted), the stale channel is dropped
    /// and the connection re-established first — this is what keeps the file
    /// browser and transfers working after a long idle period instead of
    /// failing until the user manually toggles the connection off and on.
    pub async fn ensure_sftp(&self) -> Result<Arc<Mutex<SftpSession>>> {
        let dead = self.handle.lock().await.is_closed();
        if dead {
            *self.sftp.lock().await = None;
            let generation = self.generation.load(Ordering::Acquire);
            self.reconnect(generation).await?;
        }
        if let Some(existing) = self.sftp.lock().await.clone() {
            return Ok(existing);
        }
        // Reopen on a fresh channel, reconnecting once if the open itself trips
        // a dead transport (a race against the keepalive detector).
        self.with_reconnect(|| async move { self.open_sftp_channel().await })
            .await
    }

    /// Open a new SFTP channel and cache it. Done outside the cache lock so
    /// concurrent first-time callers don't serialize on the handshake; the
    /// double-open race is benign — the first writer to re-acquire the lock
    /// wins and later openers reuse its handle.
    async fn open_sftp_channel(&self) -> Result<Arc<Mutex<SftpSession>>> {
        let channel = {
            let h = self.handle.lock().await;
            h.channel_open_session().await?
        };
        channel.request_subsystem(true, "sftp").await?;
        let sftp = Arc::new(Mutex::new(SftpSession::new(channel.into_stream()).await?));
        let mut slot = self.sftp.lock().await;
        Ok(slot.get_or_insert(sftp).clone())
    }

    /// Open a fresh exec channel for `command`, reconnecting once if the
    /// transport is dead. Used by `exec_bounded` so the agent's first command
    /// after an idle hour transparently re-establishes the session.
    async fn open_exec_channel(&self, command: &str) -> Result<Channel<client::Msg>> {
        self.with_reconnect(|| async move {
            let channel = {
                let h = self.handle.lock().await;
                h.channel_open_session().await?
            };
            channel.exec(true, command.as_bytes()).await?;
            Ok(channel)
        })
        .await
    }

    /// Re-establish the SSH connection in place. Single-flight: callers
    /// serialize on `reconnect_lock`, and the `seen_generation` check means only
    /// the first one through actually reconnects — a task that lost the race
    /// sees the bumped generation and returns, then retries its op on the fresh
    /// handle. The session id never changes, so bridge grants and UI tabs keyed
    /// by it survive untouched.
    async fn reconnect(&self, seen_generation: u64) -> Result<()> {
        let _guard = self.reconnect_lock.lock().await;
        if self.generation.load(Ordering::Acquire) != seen_generation {
            return Ok(()); // someone else already reconnected while we waited
        }
        tracing::warn!(session = %self.id, host = %self.profile.host, "SSH transport dead — reconnecting");
        let verifier: Arc<dyn HostKeyVerifier> = Arc::new(AutoTrustVerifier::reject_unknown());
        // A silent reconnect has no user to answer prompts — refuse interactive
        // auth rather than hang. (An expired password on reconnect surfaces as a
        // failed reconnect, which the user resolves by reconnecting by hand.)
        let prompter: Arc<dyn AuthPrompter> = Arc::new(RejectAuthPrompter);
        let conn = ssh_connect(&self.profile, verifier, prompter)
            .await
            .context("reconnecting SSH session")?;
        *self.handle.lock().await = conn.handle;
        // Swap in the new bastion hop (if any); dropping the old handle tears
        // down the stale jump connection.
        *self.jump_handle.lock().await = conn.jump_handle;
        *self.sftp.lock().await = None; // stale channel — force reopen
        self.generation.fetch_add(1, Ordering::Release);
        tracing::info!(session = %self.id, "SSH reconnected");
        Ok(())
    }

    /// Run `op`; if it fails because the transport is dead, reconnect once and
    /// run it again. `op` may run twice, so it must be re-runnable — capture by
    /// shared/`Copy` reference (or clone per attempt), not move-once.
    async fn with_reconnect<T, F, Fut>(&self, op: F) -> Result<T>
    where
        F: Fn() -> Fut,
        Fut: std::future::Future<Output = Result<T>>,
    {
        let generation = self.generation.load(Ordering::Acquire);
        match op().await {
            Ok(v) => Ok(v),
            Err(e) => {
                let dead = is_transport_dead(&e) || self.handle.lock().await.is_closed();
                if dead {
                    self.reconnect(generation).await?;
                    op().await
                } else {
                    Err(e)
                }
            }
        }
    }

    /// Run an SFTP operation, transparently reconnecting + reopening the
    /// subsystem once if the transport died (e.g. the session sat idle past the
    /// keepalive budget). The closure receives a fresh handle to the SFTP session
    /// and MUST do its open + IO inside, because it can run twice. Reuses the
    /// single-flight `with_reconnect`/`ensure_sftp` machinery: on a retry,
    /// `reconnect()` has nulled the cached channel so `ensure_sftp()` reopens a
    /// fresh one. This is what makes the bridge's list/read/transfer paths (which
    /// operate on the cached SFTP handle, not just open it) survive a long idle.
    pub async fn with_sftp<T, F, Fut>(&self, op: F) -> Result<T>
    where
        F: Fn(Arc<Mutex<SftpSession>>) -> Fut,
        Fut: std::future::Future<Output = Result<T>>,
    {
        let op = &op;
        self.with_reconnect(move || async move {
            let cell = self.ensure_sftp().await?;
            op(cell).await
        })
        .await
    }

    /// Cheap proactive liveness probe: if the SSH transport is already dead,
    /// reconnect it now so the next op doesn't eat the first failure. Does NOT
    /// open SFTP (keeps exec-only sessions cheap; the `with_sftp` retry covers
    /// the subsystem). Best-effort — callers ignore the result.
    pub async fn ensure_alive(&self) -> Result<()> {
        if self.handle.lock().await.is_closed() {
            let generation = self.generation.load(Ordering::Acquire);
            self.reconnect(generation).await?;
        }
        Ok(())
    }

    /// Run a single non-interactive command on a fresh exec channel and collect
    /// its full stdout/stderr + exit code, unbounded. Thin wrapper over
    /// `exec_bounded` with no caps.
    pub async fn exec(&self, command: &str) -> Result<ExecOutput> {
        self.exec_bounded(command, usize::MAX, Duration::from_secs(86_400), None)
            .await
    }

    /// Whether the SFTP subsystem can be opened on this host. Hosts that disable
    /// it (busybox appliances, locked-down boxes) return `false`, which is the
    /// cue to fall back to SCP transfers (Plan 13 Phase 3). Best-effort: a
    /// transport hiccup also reads as unavailable.
    pub async fn sftp_available(&self) -> bool {
        self.ensure_sftp().await.is_ok()
    }

    /// Download `remote_path` over the SCP wire protocol, streaming its bytes into
    /// `out` and returning the parsed header. For hosts whose SFTP subsystem is
    /// disabled but whose shell still runs `scp` (Plan 13 Phase 3). See
    /// [`crate::scp`] for the wire details (unit-tested over an in-memory duplex).
    pub async fn scp_download_to<W>(
        &self,
        remote_path: &str,
        out: &mut W,
    ) -> Result<crate::scp::ScpHeader>
    where
        W: tokio::io::AsyncWrite + Unpin,
    {
        let channel = self
            .open_exec_channel(&crate::scp::source_command(remote_path))
            .await?;
        let mut stream = channel.into_stream();
        crate::scp::download_to(&mut stream, out).await
    }

    /// Upload `size` bytes from `data` to `remote_path` over the SCP wire
    /// protocol. `mode` is the POSIX mode to request for the created file.
    pub async fn scp_upload_from<R>(
        &self,
        remote_path: &str,
        mode: u32,
        size: u64,
        data: &mut R,
    ) -> Result<()>
    where
        R: tokio::io::AsyncRead + Unpin,
    {
        let name = remote_path
            .rsplit(['/', '\\'])
            .next()
            .unwrap_or(remote_path)
            .to_string();
        let channel = self
            .open_exec_channel(&crate::scp::sink_command(remote_path))
            .await?;
        let mut stream = channel.into_stream();
        let header = crate::scp::ScpHeader { mode, size, name };
        crate::scp::upload_from(&mut stream, &header, data).await
    }

    /// Like `exec`, but bounds total captured output to `max_bytes` and the
    /// whole run to `timeout` (so `tail -f`, `top`, or a huge dump can't hang
    /// the bridge or balloon memory). If `stream` is set, stdout/stderr chunks
    /// are emitted as `agent://output` events as they arrive, so Ghost FTP's UI can
    /// show the agent's command output live. The returned `ExecOutput` carries
    /// `truncated`/`timed_out` flags so the caller (and the agent) can tell the
    /// output was cut short or the command didn't finish.
    pub async fn exec_bounded(
        &self,
        command: &str,
        max_bytes: usize,
        timeout: Duration,
        stream: Option<&ExecStream>,
    ) -> Result<ExecOutput> {
        // Commands run on behalf of the agent (i.e. with a live `stream`) are
        // *tracked*: we wrap them so the remote shell prints its own pid on
        // stderr before running the real command. sshd runs each exec in a fresh
        // session, so that pid is the command's process-group leader, and
        // `kill -- -pid` reaps the whole tree (the command, any pipe stages,
        // anything it forks) if it overruns its deadline. Untracked calls (the
        // CLI's bare `exec`, the kill itself) run the command verbatim so they
        // can't recurse or pay for the wrapper. See `wrap_tracked`.
        let tracked = stream.is_some();
        let started_at = now_millis();
        let marker = format!("__GHOSTFTP_PGID_{}", Uuid::new_v4().simple());
        let launch = if tracked {
            wrap_tracked(command, &marker)
        } else {
            command.to_string()
        };
        let mut channel = self.open_exec_channel(&launch).await?;

        // Register the run as a live job so it can be killed on demand or on
        // disconnect; its pgid is filled in below once the wrapper reports it.
        if let Some(s) = stream {
            self.job_start(&s.app, &s.op_id, &s.label, started_at).await;
        }

        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut exit_code = None;
        let mut truncated = false;

        // Process-group id of the running command, parsed from the marker line
        // the wrapper writes to stderr. Stays `None` for untracked runs, or until
        // the marker is seen (it's the first thing the wrapper emits).
        let mut pgid: Option<i32> = None;
        let marker_bytes = marker.as_bytes();
        let mut marker_buf: Vec<u8> = Vec::new();
        let mut marker_done = !tracked;

        let collect = async {
            loop {
                let Some(msg) = channel.wait().await else {
                    break;
                };
                match msg {
                    ChannelMsg::Data { ref data } => {
                        let total = stdout.len() + stderr.len();
                        append_capped(&mut stdout, data, max_bytes, total, &mut truncated);
                        if let Some(s) = stream {
                            emit_chunk(s, "stdout", data);
                        }
                    }
                    ChannelMsg::ExtendedData { ref data, ext } => {
                        // ext == 1 is the conventional stderr stream.
                        let is_err = ext == 1;
                        // The wrapper's pgid marker is the first line on stderr.
                        // Buffer until its newline, parse out the pgid, and drop
                        // the line so it never reaches the caller/agent; forward
                        // any real stderr that trailed it in the same chunk.
                        if is_err && !marker_done {
                            marker_buf.extend_from_slice(data);
                            if let Some(nl) = marker_buf.iter().position(|&b| b == b'\n') {
                                pgid = parse_pgid(&marker_buf[..nl], marker_bytes);
                                let rest = marker_buf.split_off(nl + 1);
                                marker_done = true;
                                if let (Some(p), Some(s)) = (pgid, stream) {
                                    self.job_set_pgid(&s.app, &s.op_id, p).await;
                                }
                                if !rest.is_empty() {
                                    let total = stdout.len() + stderr.len();
                                    append_capped(
                                        &mut stderr,
                                        &rest,
                                        max_bytes,
                                        total,
                                        &mut truncated,
                                    );
                                    if let Some(s) = stream {
                                        emit_chunk(s, "stderr", &rest);
                                    }
                                }
                            }
                            if truncated {
                                break;
                            }
                            continue;
                        }
                        let total = stdout.len() + stderr.len();
                        if is_err {
                            append_capped(&mut stderr, data, max_bytes, total, &mut truncated);
                        } else {
                            append_capped(&mut stdout, data, max_bytes, total, &mut truncated);
                        }
                        if let Some(s) = stream {
                            emit_chunk(s, if is_err { "stderr" } else { "stdout" }, data);
                        }
                    }
                    ChannelMsg::ExitStatus { exit_status } => exit_code = Some(exit_status as i32),
                    ChannelMsg::Eof | ChannelMsg::Close => break,
                    _ => {}
                }
                if truncated {
                    break;
                }
            }
        };

        let timed_out = tokio::time::timeout(timeout, collect).await.is_err();

        // The old behaviour stopped here: dropping the channel left the remote
        // process running, because a no-pty exec gets no SIGHUP on channel close.
        // Now a tracked command that blew its deadline gets its whole process
        // group terminated, so it can't outlive the call (the failure mode that
        // let a stray `wp search-replace` grind for hours after the session ended).
        let mut killed = false;
        if timed_out {
            if let Some(pgid) = pgid {
                match self.kill_pgid(pgid).await {
                    Ok(()) => {
                        killed = true;
                        tracing::info!(session = %self.id, pgid, "terminated timed-out command group");
                    }
                    Err(e) => {
                        tracing::warn!(session = %self.id, pgid, "failed to kill timed-out command group: {e:#}");
                    }
                }
            }
        }

        // Finalize the job: emit its terminal `agent://job` and drop it from the
        // live registry. `job_finish` preserves a `Killed` state set by an
        // external `kill_job`, so an on-demand kill isn't relabelled a failure.
        if let Some(s) = stream {
            let status = if killed {
                JobStatus::Killed
            } else if timed_out {
                JobStatus::TimedOut
            } else if exit_code.unwrap_or(0) == 0 {
                JobStatus::Completed
            } else {
                JobStatus::Failed
            };
            self.job_finish(&s.app, &s.op_id, status, exit_code).await;
        }

        Ok(ExecOutput {
            stdout: String::from_utf8_lossy(&stdout).to_string(),
            stderr: String::from_utf8_lossy(&stderr).to_string(),
            exit_code,
            truncated,
            timed_out,
            killed,
        })
    }

    /// Terminate a process group started by a tracked `exec_bounded` run: SIGTERM
    /// the whole group (`-pgid`), give it a moment, then SIGKILL, with a
    /// single-process fallback for the unusual server where the shell wasn't a
    /// group leader. Runs **untracked** on a fresh short-lived channel — the
    /// channel that launched the original command is already gone by the time we
    /// need this (it timed out), and running untracked stops the kill from being
    /// wrapped/tracked itself. Best-effort. NOTE: this reaps the client process
    /// tree only; a server-side database statement the command kicked off (e.g. a
    /// long `UPDATE`) keeps running until cancelled at the database.
    pub async fn kill_pgid(&self, pgid: i32) -> Result<()> {
        let cmd = format!(
            "kill -TERM -{pgid} 2>/dev/null || kill -TERM {pgid} 2>/dev/null; \
             sleep 2; \
             kill -KILL -{pgid} 2>/dev/null || kill -KILL {pgid} 2>/dev/null; true"
        );
        // `Box::pin` breaks the exec_bounded → kill_pgid → exec_bounded async
        // recursion cycle (it only ever recurses one level: the kill runs
        // untracked, so it can't time out into another kill).
        Box::pin(self.exec_bounded(&cmd, 4096, Duration::from_secs(10), None))
            .await
            .map(|_| ())
    }

    /// Snapshot of this session's in-flight tracked jobs.
    pub async fn list_jobs(&self) -> Vec<JobHandle> {
        self.jobs.lock().await.values().cloned().collect()
    }

    /// Terminate a tracked job by op_id. Marks it `Killed` first (so the
    /// in-flight `exec_bounded` finalizes it as killed rather than a generic
    /// failure when its channel closes), then signals the process group. Returns
    /// `false` if there's no such live job; `true` once it's been marked
    /// (and signalled, if its pgid is known yet).
    pub async fn kill_job(&self, op_id: &str) -> Result<bool> {
        let pgid = {
            let mut map = self.jobs.lock().await;
            match map.get_mut(op_id) {
                Some(job) => {
                    job.status = JobStatus::Killed;
                    job.pgid
                }
                None => return Ok(false),
            }
        };
        if let Some(pgid) = pgid {
            self.kill_pgid(pgid).await?;
        }
        Ok(true)
    }

    /// Register a newly-launched tracked command as a `Running` job and emit it.
    async fn job_start(&self, app: &AppHandle, op_id: &str, label: &str, started_at: u64) {
        let job = JobHandle {
            op_id: op_id.to_string(),
            session_id: self.id.clone(),
            label: label.to_string(),
            pgid: None,
            started_at,
            status: JobStatus::Running,
            exit_code: None,
        };
        self.jobs
            .lock()
            .await
            .insert(op_id.to_string(), job.clone());
        emit_job(app, &job);
    }

    /// Record the process-group id parsed from the wrapper marker, and re-emit.
    async fn job_set_pgid(&self, app: &AppHandle, op_id: &str, pgid: i32) {
        let updated = {
            let mut map = self.jobs.lock().await;
            map.get_mut(op_id).map(|j| {
                j.pgid = Some(pgid);
                j.clone()
            })
        };
        if let Some(job) = updated {
            emit_job(app, &job);
        }
    }

    /// Move a job to a terminal state, emit it, and drop it from the live map.
    /// A `Killed` state set by an external `kill_job` is never downgraded.
    async fn job_finish(
        &self,
        app: &AppHandle,
        op_id: &str,
        status: JobStatus,
        exit_code: Option<i32>,
    ) {
        let finished = {
            let mut map = self.jobs.lock().await;
            map.remove(op_id).map(|mut job| {
                if job.status != JobStatus::Killed {
                    job.status = status;
                }
                job.exit_code = exit_code;
                job
            })
        };
        if let Some(job) = finished {
            emit_job(app, &job);
        }
    }

    /// Run a command that uses `sudo`, answering sudo's password prompt with
    /// `password`. A plain exec channel has no tty, so `sudo` either errors with
    /// "no tty present" or can't prompt — this opens a **PTY** so sudo prompts
    /// normally. We prime sudo's credential cache against a unique prompt marker
    /// (`sudo -p '<marker>' -v`) and type the password **only when that exact
    /// marker appears**, so a NOPASSWD server never receives it. After priming,
    /// the real command's own `sudo` calls reuse the cached timestamp (so it also
    /// works when sudo sits mid-pipeline). The password is written to the
    /// channel's stdin (the tty) and never embedded in the command string, so it
    /// never lands in logs, `ps`, or the captured output. PTY output is a single
    /// merged stream, so everything comes back as `stdout` with `stderr` empty.
    pub async fn exec_sudo(
        &self,
        command: &str,
        password: &str,
        max_bytes: usize,
        timeout: Duration,
        stream: Option<&ExecStream>,
    ) -> Result<ExecOutput> {
        let marker = format!("[ghostftp-sudo-{}]", Uuid::new_v4().simple());
        let pgid_marker = format!("__GHOSTFTP_SUDO_PGID_{}", Uuid::new_v4().simple());
        // The PTY shell is the foreground process-group leader on the SSH
        // session. Emit its pid before sudo starts so a timed-out privileged
        // command can be interrupted and, if needed, terminated from a fresh
        // channel instead of being abandoned on the server.
        let wrapped = format!(
            "printf '%s:%s\\n' '{pgid_marker}' \"\\x24\\x24\"; sudo -p '{marker}' -v && {command}"
        );
        let wrapped_ref = wrapped.as_str();

        // Open + request a PTY + exec, reconnecting once if the transport is dead
        // (mirrors open_exec_channel). The closure is re-runnable: it only
        // captures Copy references.
        let mut channel = self
            .with_reconnect(|| async move {
                let channel = {
                    let h = self.handle.lock().await;
                    h.channel_open_session().await?
                };
                channel
                    .request_pty(true, "xterm", 80, 24, 0, 0, &[])
                    .await?;
                channel.exec(true, wrapped_ref.as_bytes()).await?;
                Ok(channel)
            })
            .await?;

        let mut out = Vec::new();
        let mut exit_code = None;
        let mut truncated = false;
        let marker_bytes = marker.as_bytes();
        let pgid_marker_bytes = pgid_marker.as_bytes();
        let mut pgid_probe = Vec::new();
        let mut pgid: Option<i32> = None;
        let mut pgid_done = false;
        let mut markers_seen = 0usize; // prompts observed so far
        let mut pw_sent = 0usize; // password lines typed

        let collect = async {
            loop {
                let Some(msg) = channel.wait().await else {
                    break;
                };
                match msg {
                    ChannelMsg::Data { ref data } | ChannelMsg::ExtendedData { ref data, .. } => {
                        if !pgid_done {
                            pgid_probe.extend_from_slice(data);
                            if let Some(nl) = pgid_probe.iter().position(|&b| b == b'\n') {
                                pgid = parse_pgid(&pgid_probe[..nl], pgid_marker_bytes);
                                pgid_done = true;
                            } else if pgid_probe.len() > 1024 {
                                // The marker is emitted before sudo starts. If it has
                                // not appeared within 1 KiB, stop buffering raw PTY
                                // output and fall back to Ctrl-C-only timeout handling.
                                pgid_done = true;
                            }
                        }
                        let total = out.len();
                        append_capped(&mut out, data, max_bytes, total, &mut truncated);
                        if let Some(s) = stream {
                            emit_chunk(s, "stdout", data);
                        }
                        // Type the password each time a *new* sudo prompt appears.
                        // sudo retries up to 3x on a wrong password and then gives
                        // up (channel closes), so capping the sends avoids a hang.
                        let hits = count_occurrences(&out, marker_bytes);
                        while markers_seen < hits {
                            markers_seen += 1;
                            if pw_sent < 3 {
                                let mut line = password.as_bytes().to_vec();
                                line.push(b'\n');
                                if channel.data(&line[..]).await.is_err() {
                                    return;
                                }
                                pw_sent += 1;
                            }
                        }
                    }
                    ChannelMsg::ExitStatus { exit_status } => exit_code = Some(exit_status as i32),
                    ChannelMsg::Eof | ChannelMsg::Close => break,
                    _ => {}
                }
                if truncated {
                    break;
                }
            }
        };

        let timed_out = tokio::time::timeout(timeout, collect).await.is_err();
        let mut killed = false;

        if timed_out {
            // In a PTY, ETX (Ctrl-C) is handled by the terminal driver and sends
            // SIGINT to the foreground process group, including a sudo child.
            // Give the remote command a short grace period to exit cleanly.
            if channel.data(&[0x03u8][..]).await.is_ok() {
                let interrupted = async {
                    loop {
                        match channel.wait().await {
                            Some(ChannelMsg::ExitStatus { exit_status }) => {
                                exit_code = Some(exit_status as i32);
                            }
                            Some(ChannelMsg::Eof) | Some(ChannelMsg::Close) | None => break true,
                            Some(_) => {}
                        }
                    }
                };
                killed = tokio::time::timeout(Duration::from_secs(2), interrupted)
                    .await
                    .unwrap_or(false);
            }

            // If Ctrl-C did not close the command, terminate the tracked shell
            // process group from a fresh channel. This is best-effort because
            // unusually hardened sudo configurations can isolate descendants,
            // but it is strictly safer than abandoning the timed-out PTY.
            if !killed {
                if let Some(pgid) = pgid {
                    killed = self.kill_pgid(pgid).await.is_ok();
                }
            }

            let _ = channel.eof().await;
        }

        // Scrub Ghost FTP's private PTY markers so the caller/agent sees only
        // real command output.
        let mut text = String::from_utf8_lossy(&out).replace(&marker, "");
        if text.starts_with(&pgid_marker) {
            text = text
                .find('\n')
                .map(|nl| text[nl + 1..].to_string())
                .unwrap_or_default();
        }
        let text = text.trim_start_matches(['\r', '\n']).to_string();

        Ok(ExecOutput {
            stdout: text,
            stderr: String::new(),
            exit_code,
            truncated,
            timed_out,
            killed,
        })
    }
}

/// Wrap a command so the remote shell prints its own pid (which, because sshd
/// runs each exec in a fresh session, is the command's process-group leader) to
/// stderr before running, so `exec_bounded` can later `kill -- -pid` the whole
/// tree. The real command is base64-encoded and piped into `sh`, so any quoting
/// or shell metacharacters in it can't break the wrapper. The marker line is
/// stripped from the captured output before the caller ever sees it. The marker
/// echo runs first and unconditionally, so the pid is reported even if the
/// command (or `base64`) then fails.
fn wrap_tracked(command: &str, marker: &str) -> String {
    use base64::Engine as _;
    let encoded = base64::engine::general_purpose::STANDARD.encode(command);
    format!("echo \"{marker}:$$\" 1>&2; echo {encoded} | base64 -d | sh")
}

/// Parse the `<marker>:<pid>` line that `wrap_tracked` emits into the pid.
fn parse_pgid(line: &[u8], marker: &[u8]) -> Option<i32> {
    let line = line.strip_suffix(b"\r").unwrap_or(line);
    let rest = line.strip_prefix(marker)?.strip_prefix(b":")?;
    std::str::from_utf8(rest).ok()?.trim().parse().ok()
}

/// Count non-overlapping occurrences of `needle` in `haystack`.
fn count_occurrences(haystack: &[u8], needle: &[u8]) -> usize {
    if needle.is_empty() || haystack.len() < needle.len() {
        return 0;
    }
    let mut count = 0;
    let mut i = 0;
    while i + needle.len() <= haystack.len() {
        if &haystack[i..i + needle.len()] == needle {
            count += 1;
            i += needle.len();
        } else {
            i += 1;
        }
    }
    count
}

/// Sink for live exec output — emits `agent://output` events tagged with `op_id`.
pub struct ExecStream {
    pub app: AppHandle,
    pub op_id: String,
    /// Friendly label for the run (the raw command, or "[name] cmd" for a saved
    /// command) — shown in the live console and the Jobs panel.
    pub label: String,
}

/// Lifecycle state of a tracked remote command. The registry (`SshSession.jobs`)
/// holds only *in-flight* (`Running`) jobs; reaching a terminal state emits an
/// `agent://job` event and drops the entry (completed history lives in the
/// bridge activity log).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum JobStatus {
    Running,
    Completed,
    /// Ran past its deadline but couldn't be terminated (no pgid / kill failed).
    TimedOut,
    /// Terminated by Ghost FTP — on timeout, on demand, or on disconnect.
    Killed,
    Failed,
}

/// A long-running command Ghost FTP launched on a session, carrying the process-group
/// id needed to terminate it on demand (`kill_job`) or on disconnect, even after
/// the originating `exec_bounded` call already returned (e.g. on timeout).
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct JobHandle {
    pub op_id: String,
    pub session_id: String,
    pub label: String,
    pub pgid: Option<i32>,
    pub started_at: u64,
    pub status: JobStatus,
    pub exit_code: Option<i32>,
}

fn emit_job(app: &AppHandle, job: &JobHandle) {
    let _ = app.emit("agent://job", job);
}

fn now_millis() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

fn emit_chunk(s: &ExecStream, stream: &str, data: &[u8]) {
    let _ = s.app.emit(
        "agent://output",
        serde_json::json!({
            "opId": s.op_id,
            "stream": stream,
            "chunk": String::from_utf8_lossy(data),
        }),
    );
}

/// Append `data` to `buf`, but never let `buf+other` exceed `max_bytes`. Sets
/// `truncated` if anything was dropped.
fn append_capped(
    buf: &mut Vec<u8>,
    data: &[u8],
    max_bytes: usize,
    total: usize,
    truncated: &mut bool,
) {
    if total >= max_bytes {
        *truncated = true;
        return;
    }
    let take = (max_bytes - total).min(data.len());
    buf.extend_from_slice(&data[..take]);
    if take < data.len() {
        *truncated = true;
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ExecOutput {
    pub stdout: String,
    pub stderr: String,
    pub exit_code: Option<i32>,
    pub truncated: bool,
    pub timed_out: bool,
    /// True when a tracked command ran past its deadline and Ghost FTP terminated its
    /// remote process group (vs `timed_out` alone, which used to mean "we gave
    /// up but it's probably still running"). See `exec_bounded`/`kill_pgid`.
    pub killed: bool,
}

/// Open a Session for any supported protocol given a profile and a verifier.
/// Used by the CLI and (transitively) the GUI's `SessionManager`. Doesn't
/// register the session anywhere — the caller owns its lifetime.
pub async fn open_session(
    profile: &ConnectionProfile,
    verifier: Arc<dyn HostKeyVerifier>,
) -> Result<Session> {
    match profile.protocol.as_str() {
        "sftp" | "ssh" | "" => {
            // The CLI has no interactive prompt UI, so forced-change/KBI servers
            // aren't supported here — refuse rather than hang.
            let prompter: Arc<dyn AuthPrompter> = Arc::new(RejectAuthPrompter);
            let conn = ssh_connect(profile, verifier, prompter).await?;
            let ssh = Arc::new(SshSession::new(profile.clone(), conn.handle));
            *ssh.jump_handle.lock().await = conn.jump_handle;
            Ok(Session::Ssh(ssh))
        }
        "ftp" | "ftps" => {
            let ftp = ftp_connect(profile).await?;
            Ok(Session::Ftp(Arc::new(ftp)))
        }
        "s3" | "azure" | "gcs" => {
            let obj = object_connect(profile).await?;
            Ok(Session::Object(Arc::new(obj)))
        }
        "webdav" => {
            let dav = webdav_connect(profile).await?;
            Ok(Session::Webdav(Arc::new(dav)))
        }
        "http" | "https" => {
            let http = http_connect(profile).await?;
            Ok(Session::Http(Arc::new(http)))
        }
        "dropbox" => {
            let dbx = dropbox_connect(profile).await?;
            Ok(Session::Dropbox(Arc::new(dbx)))
        }
        "onedrive" => {
            let od = onedrive_connect(profile).await?;
            Ok(Session::OneDrive(Arc::new(od)))
        }
        "gdrive" => {
            let gd = gdrive_connect(profile).await?;
            Ok(Session::GDrive(Arc::new(gd)))
        }
        "box" => {
            let bx = box_connect(profile).await?;
            Ok(Session::Box(Arc::new(bx)))
        }
        "shopify" => {
            let sh = shopify_connect(profile).await?;
            Ok(Session::Shopify(Arc::new(sh)))
        }
        "hubspot" => {
            let hs = hubspot_connect(profile).await?;
            Ok(Session::HubSpot(Arc::new(hs)))
        }
        "dynamics" => {
            let dynm = dynamics_connect(profile).await?;
            Ok(Session::Dynamics(Arc::new(dynm)))
        }
        other => Err(anyhow!("unsupported protocol: {other}")),
    }
}

// ---- Connect ----

#[cfg(unix)]
async fn open_agent() -> Result<russh_keys::agent::client::AgentClient<tokio::net::UnixStream>> {
    russh_keys::agent::client::AgentClient::connect_env()
        .await
        .context("connecting to ssh-agent via $SSH_AUTH_SOCK")
}

#[cfg(windows)]
async fn open_agent(
) -> Result<russh_keys::agent::client::AgentClient<tokio::net::windows::named_pipe::NamedPipeClient>>
{
    // First try OpenSSH-for-Windows (shipped with Win10/11) which listens
    // on a fixed pipe when the 'OpenSSH Authentication Agent' service runs.
    const OPENSSH_PIPE: &str = r"\\.\pipe\openssh-ssh-agent";
    if let Ok(stream) = tokio::net::windows::named_pipe::ClientOptions::new().open(OPENSSH_PIPE) {
        return Ok(russh_keys::agent::client::AgentClient::connect(stream));
    }

    // Fall back to PuTTY Pageant 0.78+, which uses pipes like
    // `\\.\pipe\pageant.<user>.<random>`. The random suffix is per-launch,
    // so we enumerate `\\.\pipe\` and try each pageant.* entry. Older
    // Pageant (file-mapping IPC) is not supported.
    if let Some(pipe) = find_pageant_pipe() {
        if let Ok(stream) = tokio::net::windows::named_pipe::ClientOptions::new().open(&pipe) {
            return Ok(russh_keys::agent::client::AgentClient::connect(stream));
        }
    }

    Err(anyhow!(
        "no ssh-agent found on Windows. Tried OpenSSH ({OPENSSH_PIPE}) and Pageant pipes. \
         Start the 'OpenSSH Authentication Agent' service or launch Pageant (PuTTY 0.78+)."
    ))
}

#[cfg(windows)]
fn find_pageant_pipe() -> Option<std::ffi::OsString> {
    // Listing `\\.\pipe\` returns the names of every named pipe on the system.
    // The OS exposes them as a directory since Windows 7 / 10.
    let entries = std::fs::read_dir(r"\\.\pipe\").ok()?;
    for entry in entries.flatten() {
        let name = entry.file_name();
        let s = name.to_string_lossy();
        if s.starts_with("pageant.") {
            // The full pipe path is `\\.\pipe\<name>`.
            let mut p = std::ffi::OsString::from(r"\\.\pipe\");
            p.push(&name);
            return Some(p);
        }
    }
    None
}

async fn authenticate_with_agent(
    session: &mut client::Handle<ClientHandler>,
    username: &str,
) -> Result<bool> {
    let mut agent = open_agent().await?;

    let identities = agent
        .request_identities()
        .await
        .context("listing agent identities")?;
    if identities.is_empty() {
        return Err(anyhow!(
            "ssh-agent has no identities loaded (run `ssh-add`)"
        ));
    }
    for id in identities {
        let (next_agent, authed) = session.authenticate_future(username, id, agent).await;
        agent = next_agent;
        match authed {
            Ok(true) => return Ok(true),
            Ok(false) => continue,
            Err(e) => {
                tracing::debug!(?e, "agent identity failed, trying next");
                continue;
            }
        }
    }
    Ok(false)
}

/// An established SSH connection to a profile's target host. When the profile
/// has a bastion (`jump_host`), the target session is tunneled through it and
/// `jump_handle` keeps that hop alive — dropping it would kill the
/// direct-tcpip channel the target session rides on.
pub struct SshConnection {
    pub handle: client::Handle<ClientHandler>,
    pub jump_handle: Option<client::Handle<ClientHandler>>,
}

pub async fn ssh_connect(
    profile: &ConnectionProfile,
    verifier: Arc<dyn HostKeyVerifier>,
    prompter: Arc<dyn AuthPrompter>,
) -> Result<SshConnection> {
    let config = Arc::new(client::Config {
        // Keep idle sessions alive and detect a genuinely dead peer (laptop
        // slept, NAT/firewall dropped the socket, server rebooted) in ~60s:
        // russh sends a keepalive after `keepalive_interval` of server silence
        // and tears the connection down after `keepalive_max` unanswered ones.
        keepalive_interval: Some(Duration::from_secs(15)),
        keepalive_max: 3,
        // MUST stay None. A finite inactivity_timeout makes russh drop a session
        // that has merely been idle that long even when it's perfectly alive —
        // that was the "after an hour, every command fails" bug. Liveness is the
        // keepalive's job now, not a wall-clock cap.
        inactivity_timeout: None,
        ..Default::default()
    });

    // Single-hop ProxyJump: connect to the bastion with the SAME auth material
    // (grant spec: the issuer installs the uploaded public key on both hops),
    // open a direct-tcpip channel to the target, and run the target's SSH
    // handshake over that channel.
    if let Some(jump_host) = profile.jump_host.clone() {
        let jump_port = profile.jump_port.unwrap_or(22);
        let jump_username = profile
            .jump_username
            .clone()
            .unwrap_or_else(|| profile.username.clone());
        let jump_handler = ClientHandler {
            host: jump_host.clone(),
            port: jump_port,
            verifier: verifier.clone(),
        };
        let mut jump = client::connect(
            config.clone(),
            (jump_host.as_str(), jump_port),
            jump_handler,
        )
        .await
        .with_context(|| format!("connect to jump host {jump_host}:{jump_port}"))?;
        authenticate(&mut jump, profile, &jump_username, &prompter)
            .await
            .with_context(|| format!("authenticate on jump host {jump_host}"))?;

        let channel = jump
            .channel_open_direct_tcpip(profile.host.clone(), profile.port as u32, "127.0.0.1", 0)
            .await
            .with_context(|| {
                format!(
                    "open tunnel to {}:{} via {jump_host}",
                    profile.host, profile.port
                )
            })?;
        let handler = ClientHandler {
            host: profile.host.clone(),
            port: profile.port,
            verifier,
        };
        let mut session = client::connect_stream(config, channel.into_stream(), handler)
            .await
            .with_context(|| {
                format!(
                    "connect to {}:{} via jump host {jump_host}",
                    profile.host, profile.port
                )
            })?;
        authenticate(&mut session, profile, &profile.username, &prompter).await?;
        return Ok(SshConnection {
            handle: session,
            jump_handle: Some(jump),
        });
    }

    let addr = (profile.host.as_str(), profile.port);
    let handler = ClientHandler {
        host: profile.host.clone(),
        port: profile.port,
        verifier,
    };
    let mut session = client::connect(config, addr, handler)
        .await
        .with_context(|| format!("connect to {}:{}", profile.host, profile.port))?;
    authenticate(&mut session, profile, &profile.username, &prompter).await?;
    Ok(SshConnection {
        handle: session,
        jump_handle: None,
    })
}

/// Authenticate `session` with the profile's auth material, as `username`.
/// Shared by the direct and bastion (ProxyJump) paths — both hops authenticate
/// with the same material.
async fn authenticate(
    session: &mut client::Handle<ClientHandler>,
    profile: &ConnectionProfile,
    username: &str,
    prompter: &Arc<dyn AuthPrompter>,
) -> Result<()> {
    let authed = match &profile.auth {
        AuthMethod::Password { password } => {
            let ok = session
                .authenticate_password(username, password)
                .await
                .context("password auth")?;
            if ok {
                true
            } else {
                // Plain password failed. The server may only offer the
                // keyboard-interactive method, or it's demanding an immediate
                // change for an expired/temp password. Drive that exchange.
                keyboard_interactive_auth(session, username, Some(password.as_str()), prompter)
                    .await?
            }
        }
        AuthMethod::Key { path, passphrase } => {
            // Expand a leading `~` so a stored `~/.ssh/id_ed25519` (and the
            // absolute paths the in-app key generator writes) both load —
            // load_secret_key does no tilde expansion of its own.
            let resolved = crate::keys::expand_tilde(path);
            let key = russh_keys::load_secret_key(&resolved, passphrase.as_deref())
                .with_context(|| format!("load key {}", resolved.display()))?;
            session
                .authenticate_publickey(username, Arc::new(key))
                .await
                .context("publickey auth")?
        }
        AuthMethod::KeyRef { key_ref } => {
            // Key material lives only in the OS keychain (access grants) —
            // never on disk. Fetch the PEM and decode it straight into a
            // russh keypair.
            let pem = crate::credentials::get_secret(key_ref)
                .with_context(|| format!("read key {key_ref} from keychain"))?
                .ok_or_else(|| {
                    anyhow!(
                        "keychain entry {key_ref} is missing — the grant key may have been deleted"
                    )
                })?;
            let key = russh_keys::decode_secret_key(&pem, None)
                .with_context(|| format!("decode key {key_ref}"))?;
            session
                .authenticate_publickey(username, Arc::new(key))
                .await
                .context("publickey auth")?
        }
        AuthMethod::Agent => authenticate_with_agent(session, username).await?,
    };

    if !authed {
        return Err(anyhow!("Authentication failed"));
    }
    Ok(())
}

/// Drive a keyboard-interactive exchange. Auto-answers the ordinary login
/// password prompt from the stored password (so a server that only offers
/// keyboard-interactive still connects without bothering the user), and routes
/// every other round — the "New password"/"Retype" steps of a forced change —
/// to `prompter`. Returns whether authentication succeeded.
async fn keyboard_interactive_auth(
    session: &mut client::Handle<ClientHandler>,
    username: &str,
    stored_password: Option<&str>,
    prompter: &Arc<dyn AuthPrompter>,
) -> Result<bool> {
    use russh::client::KeyboardInteractiveAuthResponse as Kbi;

    let mut response = session
        .authenticate_keyboard_interactive_start(username, None)
        .await
        .context("starting keyboard-interactive auth")?;
    let mut auto_password_used = false;
    let mut interactive_used = false;

    loop {
        match response {
            Kbi::Success => {
                if interactive_used {
                    prompter.on_interactive_success();
                }
                return Ok(true);
            }
            Kbi::Failure => return Ok(false),
            Kbi::InfoRequest {
                name,
                instructions,
                prompts,
            } => {
                let fields: Vec<AuthPromptField> = prompts
                    .iter()
                    .map(|p| AuthPromptField {
                        prompt: p.prompt.clone(),
                        echo: p.echo,
                    })
                    .collect();

                let answers = if !auto_password_used && is_initial_password_request(&fields) {
                    if let Some(password) = stored_password {
                        auto_password_used = true;
                        vec![password.to_string()]
                    } else {
                        interactive_used = true;
                        prompter.prompt(&name, &instructions, &fields).await?
                    }
                } else if fields.is_empty() {
                    // Some servers send an empty info request (just a banner).
                    Vec::new()
                } else {
                    interactive_used = true;
                    prompter.prompt(&name, &instructions, &fields).await?
                };

                response = session
                    .authenticate_keyboard_interactive_respond(answers)
                    .await
                    .context("answering keyboard-interactive prompt")?;
            }
        }
    }
}

/// True when an info request looks like the ordinary "enter your password" login
/// step — a single hidden field whose label mentions "password" but not a *new*
/// or *retype* password — i.e. the one we can safely auto-answer from the stored
/// credential rather than asking the user.
fn is_initial_password_request(fields: &[AuthPromptField]) -> bool {
    if fields.len() != 1 || fields[0].echo {
        return false;
    }
    let p = fields[0].prompt.to_lowercase();
    p.contains("password")
        && !p.contains("new")
        && !p.contains("retype")
        && !p.contains("again")
        && !p.contains("confirm")
}

// ---- Session enum + SessionManager ----
//
// SessionManager keeps a single map of remote sessions regardless of
// protocol. Most commands speak through the `RemoteFs` trait so they don't
// care which variant they're talking to; commands that DO care (terminal
// open, host-key prompts) match explicitly.

pub enum Session {
    Ssh(Arc<SshSession>),
    Ftp(Arc<FtpSession>),
    Object(Arc<ObjectSession>),
    Webdav(Arc<WebdavSession>),
    Http(Arc<HttpSession>),
    Dropbox(Arc<DropboxSession>),
    OneDrive(Arc<OneDriveSession>),
    GDrive(Arc<GDriveSession>),
    Box(Arc<BoxSession>),
    Shopify(Arc<ShopifySession>),
    HubSpot(Arc<HubSpotSession>),
    Dynamics(Arc<DynamicsSession>),
    Agent(Arc<AgentSession>),
}

impl Session {
    pub fn profile(&self) -> &ConnectionProfile {
        match self {
            Self::Ssh(s) => &s.profile,
            Self::Ftp(s) => &s.profile,
            Self::Object(s) => &s.profile,
            Self::Webdav(s) => &s.profile,
            Self::Http(s) => &s.profile,
            Self::Dropbox(s) => &s.profile,
            Self::OneDrive(s) => &s.profile,
            Self::GDrive(s) => &s.profile,
            Self::Box(s) => &s.profile,
            Self::Shopify(s) => &s.profile,
            Self::HubSpot(s) => &s.profile,
            Self::Dynamics(s) => &s.profile,
            Self::Agent(s) => &s.profile,
        }
    }

    pub fn protocol(&self) -> &str {
        match self {
            Self::Ssh(_) => "sftp",
            Self::Ftp(_) => "ftp",
            Self::Object(s) => s.profile.protocol.as_str(),
            Self::Webdav(_) => "webdav",
            Self::Http(_) => "http",
            Self::Dropbox(_) => "dropbox",
            Self::OneDrive(_) => "onedrive",
            Self::GDrive(_) => "gdrive",
            Self::Box(_) => "box",
            Self::Shopify(_) => "shopify",
            Self::HubSpot(_) => "hubspot",
            Self::Dynamics(_) => "dynamics",
            Self::Agent(_) => "ghostftp-agent",
        }
    }
}

pub struct SessionManager {
    sessions: Mutex<HashMap<String, Arc<Session>>>,
    pub prompts: Arc<HostPromptRegistry>,
    pub auth_prompts: Arc<AuthPromptRegistry>,
}

impl Default for SessionManager {
    fn default() -> Self {
        Self::new()
    }
}

impl SessionManager {
    pub fn new() -> Self {
        Self {
            sessions: Mutex::new(HashMap::new()),
            prompts: Arc::new(HostPromptRegistry::new()),
            auth_prompts: Arc::new(AuthPromptRegistry::new()),
        }
    }

    pub async fn connect(&self, profile: ConnectionProfile, app: AppHandle) -> Result<String> {
        let verifier: Arc<dyn HostKeyVerifier> =
            Arc::new(TauriHostKeyVerifier::new(app.clone(), self.prompts.clone()));
        let prompter: Arc<dyn AuthPrompter> = Arc::new(TauriAuthPrompter::new(
            app.clone(),
            self.auth_prompts.clone(),
            profile.id.clone(),
            profile.host.clone(),
        ));
        self.connect_with_verifier(profile, app, verifier, prompter)
            .await
    }

    /// Used by the CLI binary, which plugs in its own stdin-based verifier
    /// instead of the Tauri event one. The `app` parameter is still required
    /// because the FTP/S3 connect paths don't use it, but the SSH path
    /// embeds the AppHandle elsewhere via `_ = app`.
    pub async fn connect_with_verifier(
        &self,
        profile: ConnectionProfile,
        app: AppHandle,
        verifier: Arc<dyn HostKeyVerifier>,
        prompter: Arc<dyn AuthPrompter>,
    ) -> Result<String> {
        if profile.id.trim().is_empty() {
            anyhow::bail!("connection profile id must not be empty");
        }
        if profile.host.trim().is_empty() {
            anyhow::bail!("connection host must not be empty");
        }
        if profile.port == 0 {
            anyhow::bail!("connection port must be between 1 and 65535");
        }
        // Validate and dispatch from one canonical protocol value. Imported
        // profiles can contain mixed case or surrounding whitespace; treating
        // validation and transport selection differently is both confusing and
        // an easy way for malformed profiles to bypass protocol-specific checks.
        let protocol = profile.protocol.trim().to_ascii_lowercase();
        if matches!(protocol.as_str(), "sftp" | "ssh" | "ftp" | "ftps" | "")
            && profile.username.trim().is_empty()
        {
            anyhow::bail!("connection username must not be empty");
        }
        if let AuthMethod::Key { path, .. } = &profile.auth {
            if path.trim().is_empty() {
                anyhow::bail!("SSH private key path must not be empty");
            }
        }
        let _ = app;
        let session = match protocol.as_str() {
            "sftp" | "ssh" | "" => {
                let conn = ssh_connect(&profile, verifier, prompter).await?;
                let id = Uuid::new_v4().to_string();
                let ssh = Arc::new(SshSession {
                    id: id.clone(),
                    profile,
                    handle: Mutex::new(conn.handle),
                    jump_handle: Mutex::new(conn.jump_handle),
                    sftp: Mutex::new(None),
                    reconnect_lock: Mutex::new(()),
                    generation: AtomicU64::new(0),
                    jobs: Mutex::new(HashMap::new()),
                });
                (id, Session::Ssh(ssh))
            }
            "ftp" | "ftps" => {
                let ftp = ftp_connect(&profile).await?;
                let id = ftp.id.clone();
                (id, Session::Ftp(Arc::new(ftp)))
            }
            "s3" | "azure" | "gcs" => {
                let _ = app; // Object stores have no per-connect UI prompts.
                let obj = object_connect(&profile).await?;
                let id = obj.id.clone();
                (id, Session::Object(Arc::new(obj)))
            }
            "webdav" => {
                let _ = app; // WebDAV auth is carried in each request, no UI prompt.
                let dav = webdav_connect(&profile).await?;
                let id = dav.id.clone();
                (id, Session::Webdav(Arc::new(dav)))
            }
            "http" | "https" => {
                let _ = app; // Read-only HTTP source; no per-connect UI prompt.
                let http = http_connect(&profile).await?;
                let id = http.id.clone();
                (id, Session::Http(Arc::new(http)))
            }
            "dropbox" => {
                let _ = app; // OAuth tokens loaded from the keychain; no prompt.
                let dbx = dropbox_connect(&profile).await?;
                let id = dbx.id.clone();
                (id, Session::Dropbox(Arc::new(dbx)))
            }
            "onedrive" => {
                let _ = app; // OAuth tokens loaded from the keychain; no prompt.
                let od = onedrive_connect(&profile).await?;
                let id = od.id.clone();
                (id, Session::OneDrive(Arc::new(od)))
            }
            "gdrive" => {
                let _ = app; // OAuth tokens loaded from the keychain; no prompt.
                let gd = gdrive_connect(&profile).await?;
                let id = gd.id.clone();
                (id, Session::GDrive(Arc::new(gd)))
            }
            "box" => {
                let _ = app; // OAuth tokens loaded from the keychain; no prompt.
                let bx = box_connect(&profile).await?;
                let id = bx.id.clone();
                (id, Session::Box(Arc::new(bx)))
            }
            "shopify" => {
                let _ = app; // Credential loaded from the keychain; no prompt.
                let sh = shopify_connect(&profile).await?;
                let id = sh.id.clone();
                (id, Session::Shopify(Arc::new(sh)))
            }
            "hubspot" => {
                let _ = app; // Credential loaded from the keychain; no prompt.
                let hs = hubspot_connect(&profile).await?;
                let id = hs.id.clone();
                (id, Session::HubSpot(Arc::new(hs)))
            }
            "dynamics" => {
                let _ = app; // Credential loaded from the keychain; no prompt.
                let dynm = dynamics_connect(&profile).await?;
                let id = dynm.id.clone();
                (id, Session::Dynamics(Arc::new(dynm)))
            }
            "ghostftp-agent" => {
                let agent = AgentSession::connect(profile).await?;
                let id = agent.id.clone();
                (id, Session::Agent(Arc::new(agent)))
            }
            other => return Err(anyhow!("unsupported protocol: {other}")),
        };
        let (id, sess) = session;
        self.sessions
            .lock()
            .await
            .insert(id.clone(), Arc::new(sess));
        Ok(id)
    }

    pub async fn get(&self, id: &str) -> Option<Arc<Session>> {
        self.sessions.lock().await.get(id).cloned()
    }

    /// Convenience accessor when the caller specifically needs an SSH session
    /// (e.g. opening a PTY). Returns None if the session is FTP or missing.
    pub async fn get_ssh(&self, id: &str) -> Option<Arc<SshSession>> {
        match self.sessions.lock().await.get(id) {
            Some(s) => match &**s {
                Session::Ssh(ssh) => Some(ssh.clone()),
                _ => None,
            },
            None => None,
        }
    }

    /// Convenience accessor for a Ghost FTP Agent session (e.g. the bridge routing an
    /// exec to a paired daemon). None if the session is another protocol/missing.
    pub async fn get_agent(&self, id: &str) -> Option<Arc<AgentSession>> {
        match self.sessions.lock().await.get(id) {
            Some(s) => match &**s {
                Session::Agent(a) => Some(a.clone()),
                _ => None,
            },
            None => None,
        }
    }

    pub async fn disconnect(&self, id: &str) -> Result<()> {
        let s = self.sessions.lock().await.remove(id);
        if let Some(s) = s {
            match &*s {
                Session::Ssh(ssh) => {
                    // Reap any still-running tracked commands *before* tearing the
                    // transport down, while the channel is still usable — so a
                    // long background job can't outlive the session (the failure
                    // that let a stray migration run for hours). Best-effort.
                    for job in ssh.list_jobs().await {
                        let _ = ssh.kill_job(&job.op_id).await;
                    }
                    let h = ssh.handle.lock().await;
                    let _ = h
                        .disconnect(russh::Disconnect::ByApplication, "bye", "en")
                        .await;
                    drop(h);
                    // Close the bastion hop too, if this session was tunneled.
                    if let Some(jump) = ssh.jump_handle.lock().await.take() {
                        let _ = jump
                            .disconnect(russh::Disconnect::ByApplication, "bye", "en")
                            .await;
                    }
                }
                Session::Ftp(ftp) => {
                    let _ = ftp
                        .with_stream(|s| {
                            s.quit();
                            Ok(())
                        })
                        .await;
                }
                Session::Object(_) => {
                    // Object stores are stateless HTTP — nothing to close.
                }
                Session::Webdav(_) => {
                    // WebDAV is stateless HTTP too — reqwest's pool drops itself.
                }
                Session::Http(_) => {
                    // Read-only HTTP is stateless — nothing to close.
                }
                Session::Dropbox(_) => {
                    // Dropbox is stateless HTTP — nothing to close.
                }
                Session::OneDrive(_) => {
                    // OneDrive/Graph is stateless HTTP — nothing to close.
                }
                Session::GDrive(_) => {
                    // Google Drive is stateless HTTP — nothing to close.
                }
                Session::Box(_) => {
                    // Box is stateless HTTP — nothing to close.
                }
                Session::Shopify(_) => {
                    // Shopify is stateless HTTP — nothing to close.
                }
                Session::HubSpot(_) => {
                    // HubSpot is stateless HTTP — nothing to close.
                }
                Session::Dynamics(_) => {
                    // Dynamics/Dataverse is stateless HTTP — nothing to close.
                }
                Session::Agent(agent) => {
                    agent.disconnect().await;
                }
            }
        }
        Ok(())
    }
}
