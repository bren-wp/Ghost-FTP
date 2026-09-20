//! In-app Ghost FTP Agent host — lets THIS machine be controlled from the user's
//! other Ghost FTPs with zero extra downloads: no separate daemon binary, no
//! terminal, no SmartScreen detour. Settings → Remote control drives it.
//!
//! It embeds the same `ghostftp-agentd` stack the headless binary runs, pointed at
//! the same identity/config directory — so pinned controllers, policy, and the
//! machine's key are shared, and the embedded host and a headless daemon are
//! interchangeable views of one agent (only one can hold the port at a time;
//! the loser gets a clear error).

use anyhow::{bail, Context, Result};
use ghostftp_agent_proto::identity::Identity;
use ghostftp_agentd::{config_path, discovery::Advertisement, identity_path, ops, Config, Daemon};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::net::TcpListener;
use tokio::sync::Mutex;

use crate::AppState;

/// How long a pairing window opened from Settings stays open.
const PAIRING_WINDOW: Duration = Duration::from_secs(10 * 60);

/// Persisted host settings (`agent-host.json` in the app data dir): whether
/// the host starts with the app, and on which port.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct HostSettings {
    pub enabled: bool,
    pub port: u16,
}

impl Default for HostSettings {
    fn default() -> Self {
        Self {
            enabled: false,
            port: 8722,
        }
    }
}

/// Live serving state; present only while the host is running.
struct Running {
    daemon: Daemon,
    port: u16,
    advert: Option<Advertisement>,
    accept_task: tauri::async_runtime::JoinHandle<()>,
}

pub struct AgentHost {
    settings_path: PathBuf,
    settings: Mutex<HostSettings>,
    running: Mutex<Option<Running>>,
    /// Bumped on every open/close so a stale window-expiry timer can tell it
    /// lost the race and must not touch newer state.
    window_seq: AtomicU64,
}

impl AgentHost {
    /// Load persisted settings (missing file → defaults, host off).
    pub fn load(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).ok();
        let settings_path = dir.join("agent-host.json");
        let settings: HostSettings = std::fs::read(&settings_path)
            .ok()
            .and_then(|b| serde_json::from_slice(&b).ok())
            .unwrap_or_default();
        Ok(Self {
            settings_path,
            settings: Mutex::new(settings),
            running: Mutex::new(None),
            window_seq: AtomicU64::new(0),
        })
    }

    /// Bring the host up on launch if the user left it enabled.
    pub async fn auto_start_if_enabled(&self, app: AppHandle) {
        let enabled = self.settings.lock().await.enabled;
        if enabled {
            if let Err(e) = self.start(&app).await {
                tracing::warn!("agent host failed to auto-start: {e:#}");
            }
        }
    }

    async fn persist(&self) -> Result<()> {
        let settings = self.settings.lock().await.clone();
        std::fs::write(&self.settings_path, serde_json::to_vec_pretty(&settings)?)
            .with_context(|| format!("write {}", self.settings_path.display()))?;
        Ok(())
    }

    /// The daemon's config dir — shared with a headless ghostftp-agentd.
    fn agent_dir() -> Result<PathBuf> {
        ghostftp_agentd::config_dir(None)
    }

    async fn start(&self, app: &AppHandle) -> Result<()> {
        let mut running = self.running.lock().await;
        if running.is_some() {
            return Ok(());
        }
        let port = self.settings.lock().await.port;
        let dir = Self::agent_dir()?;
        let identity = Identity::load_or_create(&identity_path(&dir))?;
        let config = Config::load(&config_path(&dir))?;
        let fingerprint = identity.fingerprint();
        let info = ops::system_info();

        let listener = match TcpListener::bind(("0.0.0.0", port)).await {
            Ok(l) => l,
            Err(e) if e.kind() == std::io::ErrorKind::AddrInUse => bail!(
                "port {port} is already in use — is a headless ghostftp-agentd running on this \
                 machine? Stop it (or change the port) and try again."
            ),
            Err(e) => return Err(e).with_context(|| format!("bind 0.0.0.0:{port}")),
        };

        let event_app = app.clone();
        let daemon = Daemon::new(identity, config, dir).with_on_paired(move |name, key| {
            let _ = event_app.emit(
                "agent-host://paired",
                serde_json::json!({ "name": name, "key": key }),
            );
        });

        let advert = match Advertisement::publish(port, &fingerprint, &info.os, false) {
            Ok(ad) => Some(ad),
            Err(e) => {
                tracing::warn!("agent host mDNS unavailable: {e:#} — reachable by IP only");
                None
            }
        };

        let serve_daemon = daemon.clone();
        let accept_task = tauri::async_runtime::spawn(async move {
            if let Err(e) = ghostftp_agentd::serve(listener, serve_daemon).await {
                tracing::warn!("agent host stopped serving: {e:#}");
            }
        });

        *running = Some(Running {
            daemon,
            port,
            advert,
            accept_task,
        });
        Ok(())
    }

    async fn stop(&self) {
        if let Some(r) = self.running.lock().await.take() {
            r.accept_task.abort();
            // Dropping `advert` unregisters the mDNS record.
        }
        self.window_seq.fetch_add(1, Ordering::SeqCst);
    }

    /// Snapshot for the Settings UI. Reads the daemon's live config when
    /// running, disk otherwise, so revokes/pairings are always current.
    pub async fn status(&self) -> Result<HostStatus> {
        let settings = self.settings.lock().await.clone();
        let dir = Self::agent_dir()?;
        let identity = Identity::load_or_create(&identity_path(&dir))?;
        let info = ops::system_info();

        let running = self.running.lock().await;
        let (is_running, port, config, pairing) = match &*running {
            Some(r) => {
                let cfg = r.daemon.config.lock().await.clone();
                let pairing = match (
                    r.daemon.pairing_code().await,
                    r.daemon.pairing_remaining().await,
                ) {
                    (Some(code), Some(left)) => Some(PairingView {
                        code,
                        remaining_secs: left.as_secs(),
                    }),
                    _ => None,
                };
                (true, r.port, cfg, pairing)
            }
            None => (
                false,
                settings.port,
                Config::load(&config_path(&dir))?,
                None,
            ),
        };

        Ok(HostStatus {
            enabled: settings.enabled,
            running: is_running,
            port,
            hostname: info.hostname,
            os: info.os,
            fingerprint: identity.fingerprint(),
            allow_exec: config.policy.allow_exec,
            allow_write: config.policy.allow_write,
            peers: config
                .peers
                .iter()
                .map(|p| PeerView {
                    name: p.name.clone(),
                    public_key: p.public_key.clone(),
                    fingerprint: ghostftp_agent_proto::decode_public(&p.public_key)
                        .map(|k| ghostftp_agent_proto::fingerprint_of(&k))
                        .unwrap_or_else(|_| "?".into()),
                    paired_at: p.paired_at,
                })
                .collect(),
            pairing,
        })
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PairingView {
    pub code: String,
    pub remaining_secs: u64,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerView {
    pub name: String,
    pub public_key: String,
    pub fingerprint: String,
    pub paired_at: i64,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct HostStatus {
    pub enabled: bool,
    pub running: bool,
    pub port: u16,
    pub hostname: String,
    pub os: String,
    pub fingerprint: String,
    pub allow_exec: bool,
    pub allow_write: bool,
    pub peers: Vec<PeerView>,
    pub pairing: Option<PairingView>,
}

fn err<E: std::fmt::Display>(e: E) -> String {
    e.to_string()
}

// ---------- Tauri commands (Settings → Remote control) ----------

#[tauri::command]
pub async fn agent_host_status(state: State<'_, AppState>) -> Result<HostStatus, String> {
    state.agent_host.status().await.map_err(err)
}

#[tauri::command]
pub async fn agent_host_set_enabled(
    enabled: bool,
    port: Option<u16>,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<HostStatus, String> {
    let host = &state.agent_host;
    {
        let mut s = host.settings.lock().await;
        s.enabled = enabled;
        if let Some(p) = port {
            s.port = p;
        }
    }
    host.persist().await.map_err(err)?;
    if enabled {
        host.start(&app).await.map_err(err)?;
    } else {
        host.stop().await;
    }
    host.status().await.map_err(err)
}

/// Open a 10-minute pairing window and return the fresh code. The window also
/// closes when the dialog does (`agent_host_close_pairing`) or on expiry.
#[tauri::command]
pub async fn agent_host_open_pairing(
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<HostStatus, String> {
    let host = &state.agent_host;
    let code = ghostftp_agent_proto::generate_code();
    {
        let mut running = host.running.lock().await;
        let Some(r) = running.as_mut() else {
            return Err("turn Remote control on first".into());
        };
        r.daemon.open_pairing(code, PAIRING_WINDOW).await;
        if let Some(ad) = r.advert.as_mut() {
            let _ = ad.set_pairable(true);
        }
    }
    // Drop the advertised flag when this window expires — unless a newer
    // window (or a stop) superseded it in the meantime.
    let seq = host.window_seq.fetch_add(1, Ordering::SeqCst) + 1;
    let timer_app = app.clone();
    tauri::async_runtime::spawn(async move {
        tokio::time::sleep(PAIRING_WINDOW).await;
        let state = timer_app.state::<AppState>();
        let host = &state.agent_host;
        if host.window_seq.load(Ordering::SeqCst) != seq {
            return;
        }
        let mut running = host.running.lock().await;
        if let Some(r) = running.as_mut() {
            r.daemon.close_pairing().await;
            if let Some(ad) = r.advert.as_mut() {
                let _ = ad.set_pairable(false);
            }
        }
        let _ = timer_app.emit("agent-host://window-closed", ());
    });
    host.status().await.map_err(err)
}

#[tauri::command]
pub async fn agent_host_close_pairing(state: State<'_, AppState>) -> Result<HostStatus, String> {
    let host = &state.agent_host;
    host.window_seq.fetch_add(1, Ordering::SeqCst);
    {
        let mut running = host.running.lock().await;
        if let Some(r) = running.as_mut() {
            r.daemon.close_pairing().await;
            if let Some(ad) = r.advert.as_mut() {
                let _ = ad.set_pairable(false);
            }
        }
    }
    host.status().await.map_err(err)
}

/// Update the machine's own policy — what paired controllers may do here.
/// Applies live to the running daemon and persists to the shared config.
#[tauri::command]
pub async fn agent_host_set_policy(
    allow_exec: bool,
    allow_write: bool,
    state: State<'_, AppState>,
) -> Result<HostStatus, String> {
    let host = &state.agent_host;
    let dir = AgentHost::agent_dir().map_err(err)?;
    let running = host.running.lock().await;
    match &*running {
        Some(r) => {
            let mut cfg = r.daemon.config.lock().await;
            cfg.policy.allow_exec = allow_exec;
            cfg.policy.allow_write = allow_write;
            cfg.save(&config_path(&dir)).map_err(err)?;
        }
        None => {
            let mut cfg = Config::load(&config_path(&dir)).map_err(err)?;
            cfg.policy.allow_exec = allow_exec;
            cfg.policy.allow_write = allow_write;
            cfg.save(&config_path(&dir)).map_err(err)?;
        }
    }
    drop(running);
    host.status().await.map_err(err)
}

/// Remove a pinned controller. Its next connection attempt is refused; an
/// already-open session lasts until it disconnects.
#[tauri::command]
pub async fn agent_host_revoke_peer(
    public_key: String,
    state: State<'_, AppState>,
) -> Result<HostStatus, String> {
    let host = &state.agent_host;
    let dir = AgentHost::agent_dir().map_err(err)?;
    let running = host.running.lock().await;
    match &*running {
        Some(r) => {
            let mut cfg = r.daemon.config.lock().await;
            cfg.peers.retain(|p| p.public_key != public_key);
            cfg.save(&config_path(&dir)).map_err(err)?;
        }
        None => {
            let mut cfg = Config::load(&config_path(&dir)).map_err(err)?;
            cfg.peers.retain(|p| p.public_key != public_key);
            cfg.save(&config_path(&dir)).map_err(err)?;
        }
    }
    drop(running);
    host.status().await.map_err(err)
}
