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

use crate::config::{config_path, Config};
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
use tokio::sync::Mutex;

/// First-frame length of a paired `Noise_XX` handshake.
const XX_FIRST_FRAME: u32 = 32;
/// First-frame length of a pairing `Noise_XXpsk3` handshake.
const XXPSK3_FIRST_FRAME: u32 = 48;
/// How long `dispatch` waits for a first frame before dropping a silent peer.
const SNIFF_TIMEOUT: Duration = Duration::from_secs(10);

/// An open pairing window: while unexpired, pairing handshakes that prove
/// `code` are accepted and pinned.
pub struct PairingWindow {
    pub code: String,
    pub expires_at: Instant,
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
    /// Invoked the moment a controller is pinned: `(name, public_key_b64)`.
    on_paired: Arc<dyn Fn(&str, &str) + Send + Sync>,
}

impl Daemon {
    pub fn new(identity: Identity, config: Config, config_dir: PathBuf) -> Self {
        Self {
            identity: Arc::new(identity),
            config: Arc::new(Mutex::new(config)),
            config_dir,
            pairing: Arc::new(Mutex::new(None)),
            jobs: Arc::new(JobStore::new()),
            on_paired: Arc::new(|_, _| {}),
        }
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
    loop {
        let (stream, peer) = listener.accept().await.context("accept")?;
        let daemon = daemon.clone();
        tokio::spawn(async move {
            if let Err(e) = dispatch(stream, daemon).await {
                tracing::info!(%peer, "connection closed: {e:#}");
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
            let Some(code) = daemon.pairing_code().await else {
                // No window open: drop it. The PSK couldn't match anyway; a
                // fabricated failure would only leak timing about the window.
                anyhow::bail!("pairing attempt refused — no pairing window is open");
            };
            let (mut channel, name, key) = pair_handshake(stream, &daemon, &code).await?;
            (daemon.on_paired)(&name, &key);
            request_loop(&mut channel, &daemon, &name).await;
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
    let mut channel = SecureChannel::establish(
        stream,
        Role::Responder,
        &sk,
        Auth::Paired {
            expect_remote: None,
        },
    )
    .await
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
            anyhow::bail!("rejected unpinned peer {peer_key}");
        }
        cfg.peer_name(&peer_key)
            .unwrap_or("paired peer")
            .to_string()
    };

    // First message must be a Hello with a compatible protocol version.
    let hello: Hello = channel.recv().await.context("expected hello")?;
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

    request_loop(&mut channel, &daemon, &peer_name).await;
    Ok(())
}

/// Answer a peer's requests under the daemon's live policy until it hangs up.
/// Shared by the paired path and the post-pairing tail of a pairing connection.
async fn request_loop<S>(channel: &mut SecureChannel<S>, daemon: &Daemon, peer_name: &str)
where
    S: AsyncRead + AsyncWrite + Unpin,
{
    loop {
        let req: Request = match channel.recv().await {
            Ok(r) => r,
            Err(_) => break, // peer hung up
        };
        let policy = daemon.config.lock().await.policy;
        let summary = describe(&req);
        let resp = ops::handle(req, policy, &daemon.jobs).await;
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
    let mut channel = SecureChannel::establish(stream, Role::Responder, &sk, Auth::Pairing { psk })
        .await
        .context("pairing handshake (wrong code?)")?;

    let peer_key = channel.remote_static_b64();
    let hello: Hello = channel
        .recv()
        .await
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
    request_loop(&mut channel, &daemon, &name).await;
    Ok((name, key))
}

/// One-line human summary of a request for the audit log (never includes file
/// bodies or command output).
fn describe(req: &Request) -> String {
    match req {
        Request::Ping => "ping".into(),
        Request::SystemInfo => "system-info".into(),
        Request::ListDir { path } => format!("list {path}"),
        Request::Stat { path } => format!("stat {path}"),
        Request::ReadFile { path, .. } => format!("read {path}"),
        Request::ReadChunk { path, offset, .. } => format!("read-chunk {path}@{offset}"),
        Request::WriteChunk { path, offset, .. } => format!("write-chunk {path}@{offset}"),
        Request::Signature { path } => format!("signature {path}"),
        Request::DeltaAssemble { dest, recipe, .. } => {
            format!("delta-assemble {dest} ({} ops)", recipe.len())
        }
        Request::Delete { path, .. } => format!("delete {path}"),
        Request::CreateDir { path } => format!("mkdir {path}"),
        Request::Rename { from, to } => format!("rename {from} -> {to}"),
        Request::Chmod { path, mode } => format!("chmod {path} {mode:o}"),
        Request::Exec { command, .. } => format!("exec: {command}"),
        Request::ExecStart {
            job_id, command, ..
        } => format!("exec-start {job_id}: {command}"),
        Request::ExecPoll { job_id } => format!("exec-poll {job_id}"),
        Request::ExecKill { job_id } => format!("exec-kill {job_id}"),
    }
}

fn log_op(peer: &str, summary: &str, resp: &Response) {
    match resp {
        Response::Error {
            message,
            denied: true,
        } => {
            tracing::warn!(%peer, "DENIED {summary} — {message}")
        }
        Response::Error {
            message,
            denied: false,
        } => {
            tracing::warn!(%peer, "FAILED {summary} — {message}")
        }
        _ => tracing::info!(%peer, "{summary}"),
    }
}
