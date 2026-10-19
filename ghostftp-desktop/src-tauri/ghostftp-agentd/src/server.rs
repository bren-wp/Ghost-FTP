//! TCP listener, per-connection handshake, and the request loop.
//!
//! One listener serves both connection kinds — [`serve`] sniffs the first
//! handshake frame to tell them apart (no prelude bytes, no second port):
//!   * **paired** — `Noise_XX`, first frame is 32 bytes (ephemeral key, empty
//!     plaintext payload). Refuse any peer whose static key isn't pinned, then
//!     service requests under the daemon's policy.
//!   * **pairing** — `Noise_XXpsk3`, first frame is 48 bytes (the psk modifier
//!     AEAD-seals even the first, empty payload — 16-byte tag). Honoured only
//!     while a [pairing window](Daemon::open_pairing) is open; each controller
//!     that proves the code gets pinned, persisted, and served immediately.

use crate::config::{config_path, Config, PeerPermissions};
use crate::jobs::JobStore;
use crate::ops;
use anyhow::{Context, Result};
use ghostftp_agent_proto::{
    identity::Identity,
    msg::{Hello, Request, Response, PROTOCOL_VERSION},
    pairing, Auth, Role, SecureChannel,
};
use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::io::{AsyncRead, AsyncWrite};
use tokio::net::{TcpListener, TcpStream};
use tokio::sync::{Mutex, Notify, Semaphore};

/// First-frame length of a paired `Noise_XX` handshake.
const XX_FIRST_FRAME: u32 = 32;
/// First-frame length of a pairing `Noise_XXpsk3` handshake.
const XXPSK3_FIRST_FRAME: u32 = 48;
/// How long `dispatch` waits for a first frame before dropping a silent peer.
const SNIFF_TIMEOUT: Duration = Duration::from_secs(10);
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(15);
const IDLE_TIMEOUT: Duration = Duration::from_secs(120);
const MAX_CLIENT_CONNECTIONS: usize = 32;
const MAX_PAIR_ATTEMPTS: u8 = 6;

type PairedHook = Arc<dyn Fn(&str, &str) + Send + Sync>;

/// An open pairing window: while unexpired, pairing handshakes that prove
/// `code` are accepted and pinned.
pub struct PairingWindow {
    pub code: String,
    pub expires_at: Instant,
    pub attempts: u8,
}

/// Shared daemon state handed to each connection.
#[derive(Clone)]
pub struct Daemon {
    pub identity: Arc<Identity>,
    pub config: Arc<Mutex<Config>>,
    pub config_dir: PathBuf,
    /// The only time pairing handshakes are honoured. Shared so an embedding
    /// app (or the CLI's expiry timer) can open/close it while serving.
    pairing: Arc<Mutex<Option<PairingWindow>>>,
    /// Detached background jobs (Plan 10 Phase 4), shared across every
    /// connection so a job started on one channel is pollable after a re-dial.
    jobs: Arc<JobStore>,
    auth_changes: Arc<Notify>,
    /// Invoked the moment a controller is pinned: `(name, public_key_b64)`.
    on_paired: PairedHook,
}

impl Daemon {
    pub fn new(identity: Identity, config: Config, config_dir: PathBuf) -> Self {
        Self {
            identity: Arc::new(identity),
            config: Arc::new(Mutex::new(config)),
            config_dir,
            pairing: Arc::new(Mutex::new(None)),
            jobs: Arc::new(JobStore::new()),
            auth_changes: Arc::new(Notify::new()),
            on_paired: Arc::new(|_, _| {}),
        }
    }

    /// Abort active channels when a local owner changes permissions or revokes
    /// a paired controller. Other devices may reconnect with their own grants.
    pub fn invalidate_sessions(&self) {
        self.auth_changes.notify_waiters();
    }

    pub fn cancel_peer_jobs(&self, peer_key: &str) -> usize {
        self.jobs.kill_namespace(peer_key)
    }

    pub fn cancel_all_jobs(&self) -> usize {
        self.jobs.kill_all()
    }

    /// Replace the paired-notification hook (builder-style). The CLI prints a
    /// ✓ line; the embedding app emits an event + toast.
    pub fn with_on_paired(mut self, f: impl Fn(&str, &str) + Send + Sync + 'static) -> Self {
        self.on_paired = Arc::new(f);
        self
    }

    /// Open (or replace) the pairing window: `code` is accepted for `ttl`.
    pub async fn open_pairing(&self, code: String, ttl: Duration) {
        *self.pairing.lock().await = Some(PairingWindow {
            code,
            expires_at: Instant::now() + ttl,
            attempts: 0,
        });
    }

    pub async fn close_pairing(&self) {
        *self.pairing.lock().await = None;
    }

    /// The active pairing code, if a window is open. Expired windows are
    /// cleared (and refused) here, so expiry needs no background timer.
    pub async fn pairing_code(&self) -> Option<String> {
        let mut slot = self.pairing.lock().await;
        match &*slot {
            Some(w) if w.expires_at > Instant::now() => Some(w.code.clone()),
            Some(_) => {
                *slot = None;
                None
            }
            None => None,
        }
    }

    async fn authorize_pairing_attempt(&self) -> Result<String> {
        let mut slot = self.pairing.lock().await;
        let Some(window) = slot.as_mut() else {
            anyhow::bail!("pairing unavailable");
        };
        if Instant::now() >= window.expires_at || window.attempts >= MAX_PAIR_ATTEMPTS {
            *slot = None;
            anyhow::bail!("pairing unavailable or attempt limit reached");
        }
        window.attempts += 1;
        Ok(window.code.clone())
    }

    /// Time left on the open pairing window, for countdown UIs.
    pub async fn pairing_remaining(&self) -> Option<Duration> {
        let slot = self.pairing.lock().await;
        slot.as_ref()
            .map(|w| w.expires_at.saturating_duration_since(Instant::now()))
            .filter(|d| !d.is_zero())
    }
}

/// Serve until the listener errors or the process exits: paired connections
/// always; pairing handshakes whenever a pairing window is open.
pub async fn serve(listener: TcpListener, daemon: Daemon) -> Result<()> {
    let capacity = Arc::new(Semaphore::new(MAX_CLIENT_CONNECTIONS));
    loop {
        let (stream, peer) = listener.accept().await.context("accept")?;
        let Ok(permit) = capacity.clone().try_acquire_owned() else {
            tracing::warn!(%peer, "remote agent connection limit reached");
            drop(stream);
            continue;
        };
        let daemon = daemon.clone();
        tokio::spawn(async move {
            let _permit = permit;
            if dispatch(stream, daemon).await.is_err() {
                // Don't echo handshake frames, keys, user paths or commands.
                tracing::debug!(%peer, "remote agent session ended");
            }
        });
    }
}

/// Sniff which handshake the client is opening and route it. `peek` doesn't
/// consume, so the handshake drivers still read the stream from byte 0.
async fn dispatch(stream: TcpStream, daemon: Daemon) -> Result<()> {
    let first = tokio::time::timeout(SNIFF_TIMEOUT, sniff_frame_len(&stream))
        .await
        .context("timed out waiting for a handshake")??;
    match first {
        XXPSK3_FIRST_FRAME => {
            let code = daemon.authorize_pairing_attempt().await?;
            let (mut channel, name, key) = pair_handshake(stream, &daemon, &code).await?;
            (daemon.on_paired)(&name, &key);
            request_loop(&mut channel, &daemon, &key, &name).await;
            Ok(())
        }
        XX_FIRST_FRAME => handle_paired(stream, daemon).await,
        other => anyhow::bail!("unrecognised first handshake frame ({other} bytes)"),
    }
}

/// Peek the 4-byte length prefix of the first frame without consuming it.
async fn sniff_frame_len(stream: &TcpStream) -> Result<u32> {
    let mut buf = [0u8; 4];
    loop {
        let n = stream.peek(&mut buf).await.context("peek handshake")?;
        if n >= 4 {
            return Ok(u32::from_be_bytes(buf));
        }
        if n == 0 {
            anyhow::bail!("peer closed before the handshake");
        }
        // Partial length prefix. peek returns immediately while bytes sit in
        // the buffer, so wait a beat for the rest instead of spinning.
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
}

/// Complete a paired handshake, verify the peer is pinned, then run its requests.
pub async fn handle_paired(stream: TcpStream, daemon: Daemon) -> Result<()> {
    let sk = daemon.identity.private_bytes()?;
    // We don't know *which* pinned peer is dialing until the handshake reveals
    // their static key, so accept any well-formed XX handshake, then authorise.
    let mut channel = tokio::time::timeout(
        HANDSHAKE_TIMEOUT,
        SecureChannel::establish(
            stream,
            Role::Responder,
            &sk,
            Auth::Paired {
                expect_remote: None,
            },
        ),
    )
    .await
    .context("paired handshake timeout")?
    .context("paired handshake")?;

    let peer_key = channel.remote_static_b64();
    let peer_name = {
        let cfg = daemon.config.lock().await;
        if !cfg.is_paired(&peer_key) {
            // Refuse an unpinned peer. It knows nothing of our filesystem; the
            // connection just closes.
            let _ = channel
                .send(&Response::denied(
                    "this controller is not paired with the daemon — run `ghostftp-agentd pair`",
                ))
                .await;
            anyhow::bail!("rejected unpinned peer");
        }
        cfg.peer_name(&peer_key)
            .unwrap_or("paired peer")
            .to_string()
    };

    // First message must be a Hello with a compatible protocol version.
    let hello: Hello = tokio::time::timeout(HANDSHAKE_TIMEOUT, channel.recv())
        .await
        .context("hello timeout")?
        .context("expected hello")?;
    if hello.protocol_version != PROTOCOL_VERSION {
        let _ = channel
            .send(&Response::error(format!(
                "protocol version mismatch: daemon speaks {PROTOCOL_VERSION}, controller speaks {}",
                hello.protocol_version
            )))
            .await;
        anyhow::bail!("protocol mismatch with {peer_name}");
    }
    tracing::info!(peer = %peer_name, client = %hello.client_name, "controller connected");

    request_loop(&mut channel, &daemon, &peer_key, &peer_name).await;
    Ok(())
}

/// Answer a peer's requests under the daemon's live policy until it hangs up.
/// Shared by the paired path and the post-pairing tail of a pairing connection.
/// Enforce per-controller privileges on EVERY request. This includes requests
/// from an already-established Noise session after its pin is revoked.
fn permitted(req: &Request, grant: PeerPermissions) -> bool {
    match req {
        Request::Ping => true,
        Request::SystemInfo => grant.read,
        Request::ListDir { .. }
        | Request::Stat { .. }
        | Request::ReadFile { .. }
        | Request::ReadChunk { .. }
        | Request::Signature { .. } => grant.read,
        Request::Delete { .. } => grant.delete,
        Request::WriteChunk { .. }
        | Request::DeltaAssemble { .. }
        | Request::CreateDir { .. }
        | Request::Rename { .. }
        | Request::Chmod { .. } => grant.write,
        Request::Exec { .. }
        | Request::ExecStart { .. }
        | Request::ExecPoll { .. }
        | Request::ExecKill { .. } => grant.terminal,
    }
}

fn namespace_job_request(req: Request, peer_key: &str) -> Request {
    // The native JobStore is daemon-wide, but job IDs must never cross a
    // controller boundary. A peer cannot poll/kill another peer's output.
    let namespace = |job_id: String| format!("{peer_key}:{job_id}");
    match req {
        Request::ExecStart {
            job_id,
            command,
            max_bytes,
        } => Request::ExecStart {
            job_id: namespace(job_id),
            command,
            max_bytes,
        },
        Request::ExecPoll { job_id } => Request::ExecPoll {
            job_id: namespace(job_id),
        },
        Request::ExecKill { job_id } => Request::ExecKill {
            job_id: namespace(job_id),
        },
        other => other,
    }
}

async fn request_loop<S>(
    channel: &mut SecureChannel<S>,
    daemon: &Daemon,
    peer_key: &str,
    peer_name: &str,
) where
    S: AsyncRead + AsyncWrite + Unpin,
{
    loop {
        let req: Request = tokio::select! {
            incoming = tokio::time::timeout(IDLE_TIMEOUT, channel.recv()) => match incoming {
                Ok(Ok(r)) => r,
                _ => break,
            },
            _ = daemon.auth_changes.notified() => break,
        };
        let (policy, permissions) = {
            let cfg = daemon.config.lock().await;
            (cfg.policy, cfg.peer_permissions(peer_key))
        };
        let Some(perms) = permissions else {
            tracing::warn!(peer = %peer_name, "revoked remote agent controller disconnected");
            break;
        };
        let summary = describe(&req);
        if !permitted(&req, perms) {
            let _ = channel
                .send(&Response::denied(
                    "remote operation not approved for this controller",
                ))
                .await;
            tracing::warn!(peer = %peer_name, "DENIED {summary}");
            continue;
        }
        let external_id = match &req {
            Request::ExecStart { job_id, .. } => Some(job_id.clone()),
            _ => None,
        };
        let mut resp = tokio::select! {
            result = ops::handle(namespace_job_request(req, peer_key), policy, &daemon.jobs) => result,
            _ = daemon.auth_changes.notified() => break,
        };
        if let (Some(id), Response::ExecStarted { job_id }) = (external_id, &mut resp) {
            *job_id = id;
        }
        log_op(peer_name, &summary, &resp);
        if channel.send(&resp).await.is_err() {
            break;
        }
    }
}

/// One pairing handshake: authenticate with the code's PSK, learn + pin the
/// controller's static key, persist, and ack. Returns the still-open channel
/// with `(client_name, public_key_b64)` so the caller can keep serving it —
/// dropping the socket here was the v1.3 bug that made every UI pairing report
/// failure (the controller's post-ack `SystemInfo` hit a closed connection).
pub async fn pair_handshake(
    stream: TcpStream,
    daemon: &Daemon,
    code: &str,
) -> Result<(SecureChannel<TcpStream>, String, String)> {
    let sk = daemon.identity.private_bytes()?;
    let psk = pairing::psk_from_code(code);
    let mut channel = tokio::time::timeout(
        HANDSHAKE_TIMEOUT,
        SecureChannel::establish(stream, Role::Responder, &sk, Auth::Pairing { psk }),
    )
    .await
    .context("pairing handshake timeout")?
    .context("pairing handshake rejected")?;

    let peer_key = channel.remote_static_b64();
    let hello: Hello = tokio::time::timeout(HANDSHAKE_TIMEOUT, channel.recv())
        .await
        .context("pairing hello timeout")?
        .context("expected hello during pairing")?;
    if hello.protocol_version != PROTOCOL_VERSION {
        let _ = channel
            .send(&Response::error(format!(
                "protocol version mismatch: daemon speaks {PROTOCOL_VERSION}, controller speaks {}",
                hello.protocol_version
            )))
            .await;
        anyhow::bail!("protocol mismatch while pairing {}", hello.client_name);
    }

    {
        let mut cfg = daemon.config.lock().await;
        cfg.upsert_peer(&hello.client_name, &peer_key);
        cfg.save(&config_path(&daemon.config_dir))
            .context("persist paired peer")?;
    }

    // Acknowledge so the controller knows it's pinned.
    channel.send(&Response::Ok).await.ok();
    tracing::info!(client = %hello.client_name, key = %peer_key, "paired new controller");
    Ok((channel, hello.client_name, peer_key))
}

/// Pair one connection, then serve it until the peer hangs up. Returns
/// `(client_name, public_key_b64)` once the controller disconnects. Composed
/// from [`pair_handshake`] + the request loop; `serve_pairing` uses the pieces
/// directly so it can notify at pin time.
pub async fn pair_connection(
    stream: TcpStream,
    daemon: Daemon,
    code: &str,
) -> Result<(String, String)> {
    let (mut channel, name, key) = pair_handshake(stream, &daemon, code).await?;
    request_loop(&mut channel, &daemon, &key, &name).await;
    Ok((name, key))
}

/// One-line human summary of a request for the audit log (never includes file
/// bodies or command output).
fn describe(req: &Request) -> &'static str {
    // Paths and remote shell commands may contain passwords or bearer tokens.
    match req {
        Request::Ping => "ping",
        Request::SystemInfo => "system-info",
        Request::ListDir { .. } => "list",
        Request::Stat { .. } => "stat",
        Request::ReadFile { .. } | Request::ReadChunk { .. } => "read",
        Request::WriteChunk { .. } | Request::DeltaAssemble { .. } => "write",
        Request::Signature { .. } => "signature",
        Request::Delete { .. } => "delete",
        Request::CreateDir { .. } => "mkdir",
        Request::Rename { .. } => "rename",
        Request::Chmod { .. } => "chmod",
        Request::Exec { .. } => "exec",
        Request::ExecStart { .. } => "exec-start",
        Request::ExecPoll { .. } => "exec-poll",
        Request::ExecKill { .. } => "exec-kill",
    }
}

fn log_op(peer: &str, summary: &str, resp: &Response) {
    let outcome = match resp {
        Response::Error { denied: true, .. } => "denied",
        Response::Error { denied: false, .. } => "failed",
        _ => "completed",
    };
    tracing::info!(peer = %peer, operation = %summary, status = outcome, "agent request");
}

#[cfg(test)]
mod security_tests {
    use super::*;

    #[test]
    fn new_peers_cannot_write_delete_or_use_terminal() {
        let grant = PeerPermissions::default();
        assert!(permitted(&Request::Ping, grant));
        assert!(permitted(&Request::ListDir { path: "/".into() }, grant));
        assert!(!permitted(
            &Request::Delete {
                path: "/tmp/x".into(),
                recursive: false
            },
            grant
        ));
        assert!(!permitted(
            &Request::CreateDir {
                path: "/tmp/x".into()
            },
            grant
        ));
        assert!(!permitted(
            &Request::Exec {
                command: "echo PRIVATE_SENTINEL".into(),
                timeout_ms: 1000,
                max_bytes: 10,
            },
            grant
        ));
        assert_eq!(
            describe(&Request::Exec {
                command: "echo PRIVATE_SENTINEL".into(),
                timeout_ms: 1000,
                max_bytes: 10,
            }),
            "exec"
        );
        assert_eq!(
            describe(&Request::ReadFile {
                path: "/home/SECRET_SENTINEL".into(),
                max_bytes: 10,
            }),
            "read"
        );
    }

    #[test]
    fn per_controller_job_namespaces_are_disjoint_even_when_job_ids_match() {
        let a = namespace_job_request(
            Request::ExecPoll {
                job_id: "shared".into(),
            },
            "peerA",
        );
        let b = namespace_job_request(
            Request::ExecPoll {
                job_id: "shared".into(),
            },
            "peerB",
        );
        match (a, b) {
            (Request::ExecPoll { job_id: x }, Request::ExecPoll { job_id: y }) => {
                assert_ne!(x, y);
                assert!(x.starts_with("peerA:"));
                assert!(y.starts_with("peerB:"));
            }
            _ => panic!("unexpected request"),
        }
    }

    #[tokio::test]
    async fn six_attempts_per_pairing_window_then_fail_closed() {
        let identity = Identity::generate().unwrap();
        let path = std::env::temp_dir().join(format!("agent-limit-{}", std::process::id()));
        let daemon = Daemon::new(identity, Config::default(), path);
        daemon
            .open_pairing("123456".to_string(), Duration::from_secs(30))
            .await;
        for _ in 0..MAX_PAIR_ATTEMPTS {
            assert_eq!(daemon.authorize_pairing_attempt().await.unwrap(), "123456");
        }
        assert!(daemon.authorize_pairing_attempt().await.is_err());
        assert!(daemon.pairing_code().await.is_none());
        daemon
            .open_pairing("654321".into(), Duration::from_secs(30))
            .await;
        assert_eq!(daemon.authorize_pairing_attempt().await.unwrap(), "654321");
    }
}
