//! Ghost FTP Agent client session — the controller half of the Ghost FTP-to-Ghost FTP bridge.
//!
//! An `AgentSession` holds one encrypted [`SecureChannel`] to a paired
//! `ghostftp-agentd` and drives request/response over it. Requests are serialised
//! through a single channel, so every call takes the channel lock, sends, and
//! awaits its reply before releasing — the daemon answers in order. A dead
//! channel transparently re-dials and re-handshakes (verifying the pinned key)
//! so a browse after a long idle just works, mirroring `SshSession`'s reconnect.

use crate::profiles::ConnectionProfile;
use anyhow::{anyhow, bail, Context, Result};
use ghostftp_agent_proto::{
    identity::Identity,
    msg::{Hello, Request, Response, SystemInfo, PROTOCOL_VERSION},
    Auth, Role, SecureChannel,
};
use tokio::net::TcpStream;
use tokio::sync::Mutex;
use uuid::Uuid;

/// The controller's own long-lived static identity, shared by every agent
/// session (and the CLI). Stored alongside Ghost FTP's other data so a paired daemon
/// keeps recognising this machine across restarts.
fn controller_identity() -> Result<Identity> {
    let dir = dirs::data_dir()
        .context("no OS data dir for controller identity")?
        .join("ghostftp");
    Identity::load_or_create(&dir.join("agent-identity.json"))
}

/// Our public key (base64) — what a daemon pins for us at pairing time.
pub fn controller_public_key() -> Result<String> {
    Ok(controller_identity()?.public_b64().to_string())
}

/// Result of a successful pairing: what to persist + what to show the user.
pub struct PairOutcome {
    /// The daemon's static public key (base64) — persist into the profile.
    pub server_key: String,
    /// The daemon's fingerprint, for the confirmation toast.
    pub fingerprint: String,
    /// Facts about the machine we just paired with.
    pub system_info: SystemInfo,
}

/// Pair with a daemon at `host:port` using a one-time `code`. Completes a
/// pairing handshake, learns + returns the daemon's static key (the caller pins
/// it into the profile), and fetches system info so the UI can confirm which
/// machine was paired. Does not create a persistent session.
pub async fn agent_pair(host: &str, port: u16, code: &str) -> Result<PairOutcome> {
    if !ghostftp_agent_proto::is_valid_code(code) {
        bail!("pairing code must be 6 digits");
    }
    let identity = controller_identity()?;
    let sk = identity.private_bytes()?;
    let psk = ghostftp_agent_proto::psk_from_code(code);

    let stream = TcpStream::connect((host, port))
        .await
        .with_context(|| format!("connect to {host}:{port}"))?;
    let mut channel = SecureChannel::establish(stream, Role::Initiator, &sk, Auth::Pairing { psk })
        .await
        .context(
            "pairing handshake failed — check the 6-digit code, and that the machine is \
             showing a pairing code right now (run `ghostftp-agentd pair` on it, or open \
             Remote control in its Ghost FTP app)",
        )?;

    let server_key = channel.remote_static_b64();
    let fingerprint = ghostftp_agent_proto::fingerprint_of(channel.remote_static());

    channel
        .send(&Hello {
            protocol_version: PROTOCOL_VERSION,
            client_name: hostname(),
        })
        .await?;
    // The daemon acknowledges the pairing before we ask anything else.
    match channel.recv::<Response>().await? {
        Response::Ok => {}
        Response::Error { message, .. } => bail!("daemon rejected pairing: {message}"),
        other => bail!("unexpected pairing reply: {other:?}"),
    }
    // Grab system info while the channel is open so the UI can name the machine
    // it just paired with. Best-effort: both sides have already pinned by this
    // point, and old daemons (≤ v1.3) drop the pairing socket right after the
    // ack — a failure here must never turn a successful pairing into an error.
    let system_info = fetch_system_info(&mut channel).await.unwrap_or(SystemInfo {
        os: String::new(),
        hostname: host.to_string(),
        arch: String::new(),
        shell: String::new(),
        username: String::new(),
        home_dir: String::new(),
        agentd_version: String::new(),
    });

    Ok(PairOutcome {
        server_key,
        fingerprint,
        system_info,
    })
}

/// One live, paired connection to a `ghostftp-agentd`.
pub struct AgentSession {
    pub id: String,
    pub profile: ConnectionProfile,
    /// The pinned daemon static key (base64) this session must match on connect
    /// and every reconnect.
    server_key: String,
    channel: Mutex<Option<SecureChannel<TcpStream>>>,
    /// Reported by the daemon at connect; drives OS-aware UI + capabilities.
    pub system_info: SystemInfo,
}

impl AgentSession {
    /// Open a paired session from a profile. Requires the profile to carry the
    /// daemon's pinned key (i.e. it has been paired already).
    pub async fn connect(profile: ConnectionProfile) -> Result<Self> {
        let server_key = profile.agent_key.clone().ok_or_else(|| {
            anyhow!("this Ghost FTP Agent connection isn't paired yet — pair with a code first")
        })?;
        let mut channel = Self::dial(&profile, &server_key).await?;
        let system_info = fetch_system_info(&mut channel).await?;
        Ok(Self {
            id: Uuid::new_v4().to_string(),
            profile,
            server_key,
            channel: Mutex::new(Some(channel)),
            system_info,
        })
    }

    /// Establish + handshake a fresh channel, verifying the pinned key and
    /// sending the protocol Hello. Shared by connect and reconnect.
    async fn dial(
        profile: &ConnectionProfile,
        server_key: &str,
    ) -> Result<SecureChannel<TcpStream>> {
        let identity = controller_identity()?;
        let sk = identity.private_bytes()?;
        let expect = ghostftp_agent_proto::decode_public(server_key)?;
        let stream = TcpStream::connect((profile.host.as_str(), profile.port))
            .await
            .with_context(|| format!("connect to {}:{}", profile.host, profile.port))?;
        let mut channel = SecureChannel::establish(
            stream,
            Role::Initiator,
            &sk,
            Auth::Paired {
                expect_remote: Some(expect),
            },
        )
        .await
        .with_context(|| {
            format!(
                "secure handshake with {}:{} failed — if the machine is showing a pairing \
                 code, finish (or close) pairing there first; if its daemon was reinstalled \
                 or re-keyed, re-pair this connection",
                profile.host, profile.port
            )
        })?;
        channel
            .send(&Hello {
                protocol_version: PROTOCOL_VERSION,
                client_name: hostname(),
            })
            .await?;
        Ok(channel)
    }

    /// Send one request and await its response, re-dialing once if the channel
    /// has died. The whole exchange holds the channel lock so responses can't
    /// interleave across concurrent callers.
    pub async fn request(&self, req: Request) -> Result<Response> {
        let mut guard = self.channel.lock().await;

        // Fast path: use the live channel.
        if let Some(ch) = guard.as_mut() {
            match Self::exchange(ch, &req).await {
                Ok(resp) => return Ok(resp),
                Err(_) => {
                    // Transport hiccup — drop it and fall through to a reconnect.
                    *guard = None;
                }
            }
        }

        // Reconnect once and retry.
        let mut ch = Self::dial(&self.profile, &self.server_key)
            .await
            .context("reconnecting agent session")?;
        let resp = Self::exchange(&mut ch, &req).await?;
        *guard = Some(ch);
        Ok(resp)
    }

    async fn exchange(ch: &mut SecureChannel<TcpStream>, req: &Request) -> Result<Response> {
        ch.send(req).await?;
        ch.recv::<Response>().await
    }

    /// Close the channel (best-effort; dropping the stream is enough).
    pub async fn disconnect(&self) {
        *self.channel.lock().await = None;
    }

    /// Test-only session wrapping an already-established channel (the transfer
    /// tests dial an in-process daemon themselves). The profile is a
    /// placeholder — nothing listens on it, so a dead channel fails instead of
    /// re-dialing, which is what tests want.
    #[cfg(test)]
    pub(crate) fn for_test(channel: SecureChannel<TcpStream>) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            profile: ConnectionProfile {
                id: "test".into(),
                name: "test".into(),
                protocol: "ghostftp-agent".into(),
                host: "127.0.0.1".into(),
                port: 1,
                username: String::new(),
                auth: crate::profiles::AuthMethod::Agent,
                default_remote_path: None,
                description: None,
                color: None,
                auto_connect: None,
                bucket: None,
                region: None,
                endpoint: None,
                account: None,
                agent_key: None,
                group: None,
                favorite: None,
                bookmarked: None,
                tags: None,
                last_used: None,
                sort_order: None,
                icon: None,
                jump_host: None,
                jump_port: None,
                jump_username: None,
            },
            server_key: String::new(),
            channel: Mutex::new(Some(channel)),
            system_info: SystemInfo {
                os: "test".into(),
                hostname: "test".into(),
                arch: "test".into(),
                shell: "sh".into(),
                username: "test".into(),
                home_dir: String::new(),
                agentd_version: "test".into(),
            },
        }
    }

    // ---- typed helpers used by AgentFs / transfer / bridge ----
    pub async fn exec(
        &self,
        command: &str,
        max_bytes: usize,
        timeout_ms: u64,
    ) -> Result<AgentExecOutput> {
        match self
            .request(Request::Exec {
                command: command.to_string(),
                timeout_ms,
                max_bytes: max_bytes as u64,
            })
            .await?
        {
            Response::Exec {
                stdout,
                stderr,
                exit_code,
                truncated,
                timed_out,
            } => Ok(AgentExecOutput {
                stdout,
                stderr,
                exit_code,
                truncated,
                timed_out,
            }),
            Response::Error { message, denied } => Err(anyhow!(if denied {
                format!("denied: {message}")
            } else {
                message
            })),
            other => Err(anyhow!("unexpected exec reply: {other:?}")),
        }
    }

    /// Launch a detached background job (Plan 10 Phase 4). The bridge owns the
    /// `job_id`; the daemon spawns the command and returns at once. Poll it with
    /// [`AgentSession::exec_poll`]. A pre-Plan-10 daemon doesn't know this op, so
    /// the request fails at the transport — the bridge phrases that as "update
    /// the agent".
    pub async fn exec_start(
        &self,
        job_id: &str,
        command: &str,
        max_bytes: usize,
    ) -> Result<String> {
        match self
            .request(Request::ExecStart {
                job_id: job_id.to_string(),
                command: command.to_string(),
                max_bytes: max_bytes as u64,
            })
            .await?
        {
            Response::ExecStarted { job_id } => Ok(job_id),
            Response::Error { message, denied } => Err(anyhow!(if denied {
                format!("denied: {message}")
            } else {
                message
            })),
            other => Err(anyhow!("unexpected exec-start reply: {other:?}")),
        }
    }

    /// Poll a detached job's captured output + status.
    pub async fn exec_poll(&self, job_id: &str) -> Result<AgentJobStatus> {
        match self
            .request(Request::ExecPoll {
                job_id: job_id.to_string(),
            })
            .await?
        {
            Response::ExecStatus {
                running,
                exit_code,
                stdout,
                stderr,
                truncated,
                not_found,
            } => Ok(AgentJobStatus {
                running,
                exit_code,
                stdout,
                stderr,
                truncated,
                not_found,
            }),
            Response::Error { message, denied } => Err(anyhow!(if denied {
                format!("denied: {message}")
            } else {
                message
            })),
            other => Err(anyhow!("unexpected exec-poll reply: {other:?}")),
        }
    }
}

/// Exec result from a Ghost FTP Agent daemon (mirrors the SSH `ExecOutput` shape the
/// bridge already returns, minus the pgid/kill machinery the daemon handles
/// itself via kill-on-drop).
pub struct AgentExecOutput {
    pub stdout: String,
    pub stderr: String,
    pub exit_code: Option<i32>,
    pub truncated: bool,
    pub timed_out: bool,
}

/// Status of a detached background job on a Ghost FTP Agent daemon (Plan 10 Phase 4),
/// shaped like the SSH job-poll body so the bridge maps both the same way.
pub struct AgentJobStatus {
    pub running: bool,
    pub exit_code: Option<i32>,
    pub stdout: String,
    pub stderr: String,
    pub truncated: bool,
    /// The id was unknown (never started, pruned, or the daemon restarted).
    pub not_found: bool,
}

async fn fetch_system_info(channel: &mut SecureChannel<TcpStream>) -> Result<SystemInfo> {
    channel.send(&Request::SystemInfo).await?;
    match channel.recv::<Response>().await? {
        Response::SystemInfo(info) => Ok(info),
        Response::Error { message, .. } => bail!("daemon error: {message}"),
        other => bail!("unexpected reply to system-info: {other:?}"),
    }
}

/// This controller machine's hostname, sent to the daemon for its audit log.
fn hostname() -> String {
    gethostname_string()
}

#[cfg(windows)]
fn gethostname_string() -> String {
    std::env::var("COMPUTERNAME").unwrap_or_else(|_| "ghostftp".into())
}

#[cfg(not(windows))]
fn gethostname_string() -> String {
    std::env::var("HOSTNAME")
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| "ghostftp".into())
}

/// Discovery: browse the LAN for `ghostftp-agentd` daemons over mDNS.
pub mod discovery {
    use ghostftp_agent_proto::msg::SERVICE_TYPE;
    use serde::Serialize;
    use std::time::Duration;

    /// A daemon found on the network.
    #[derive(Debug, Clone, Serialize)]
    #[serde(rename_all = "camelCase")]
    pub struct Discovered {
        pub hostname: String,
        pub host: String, // resolved IP to connect to
        pub port: u16,
        pub fingerprint: String,
        pub os: String,
        pub version: String,
        /// Whether the daemon reports an open pairing window right now.
        /// `None` — daemon too old to say.
        pub pairable: Option<bool>,
    }

    /// Browse for `timeout` and return the daemons seen. Best-effort: an mDNS
    /// failure (no multicast, locked-down network) yields an empty list, not an
    /// error — the user can always add a machine by IP.
    pub async fn browse(timeout: Duration) -> Vec<Discovered> {
        let Ok(daemon) = mdns_sd::ServiceDaemon::new() else {
            return Vec::new();
        };
        let Ok(receiver) = daemon.browse(SERVICE_TYPE) else {
            return Vec::new();
        };
        let mut found: std::collections::HashMap<String, Discovered> =
            std::collections::HashMap::new();
        let deadline = tokio::time::Instant::now() + timeout;
        loop {
            let remaining = deadline.saturating_duration_since(tokio::time::Instant::now());
            if remaining.is_zero() {
                break;
            }
            match tokio::time::timeout(remaining, async { receiver.recv_async().await }).await {
                Ok(Ok(mdns_sd::ServiceEvent::ServiceResolved(info))) => {
                    let addr = info
                        .get_addresses()
                        .iter()
                        .next()
                        .map(|a| a.to_string())
                        .unwrap_or_default();
                    if addr.is_empty() {
                        continue;
                    }
                    let get = |k: &str| info.get_property_val_str(k).unwrap_or("").to_string();
                    let d = Discovered {
                        hostname: get("host"),
                        host: addr,
                        port: info.get_port(),
                        fingerprint: get("fp"),
                        os: get("os"),
                        version: get("v"),
                        pairable: info.get_property_val_str("pairable").map(|v| v == "1"),
                    };
                    found.insert(info.get_fullname().to_string(), d);
                }
                Ok(Ok(_)) => continue,
                _ => break,
            }
        }
        let _ = daemon.shutdown();
        found.into_values().collect()
    }
}
