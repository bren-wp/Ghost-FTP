//! Daemon configuration + pinned-peer store, persisted next to the identity.
//!
//! This is the *controlled machine's own* policy: what it will let a paired
//! controller do, regardless of what the controller asks. Reads are always
//! allowed once paired; exec and writes are gated here so the machine's owner
//! can hand out a read-only foothold, or lock the box down after the fact,
//! without unpairing.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

/// What a paired controller is permitted to do. Reads (list/stat/read/info) are
/// always allowed; these two gate the side-effecting classes.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Policy {
    /// Allow `Exec` (running shell commands). The headline capability.
    pub allow_exec: bool,
    /// Allow writes: `WriteChunk`, `Delete`, `CreateDir`, `Rename`, `Chmod`.
    pub allow_write: bool,
}

impl Default for Policy {
    fn default() -> Self {
        // Pairing (code + pin) is the consent gate, so a freshly-paired daemon is
        // useful out of the box. Lock it down with `--read-only` or by editing
        // the config; a read-only daemon still browses + reads + reports.
        Self { allow_exec: true, allow_write: true }
    }
}

/// A controller whose static key we've pinned at pairing time.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Peer {
    /// Human label (the controller's hostname at pairing time).
    pub name: String,
    /// base64 of the peer's X25519 static public key — the pin.
    pub public_key: String,
    /// Unix millis when this peer was paired.
    pub paired_at: i64,
}

/// On-disk daemon config (`agentd.json`).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct Config {
    pub policy: Policy,
    pub peers: Vec<Peer>,
}

impl Config {
    pub fn load(path: &Path) -> Result<Self> {
        if !path.exists() {
            return Ok(Self::default());
        }
        let bytes = std::fs::read(path).with_context(|| format!("read config {path:?}"))?;
        serde_json::from_slice(&bytes).with_context(|| format!("parse config {path:?}"))
    }

    pub fn save(&self, path: &Path) -> Result<()> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent).ok();
        }
        let bytes = serde_json::to_vec_pretty(self)?;
        std::fs::write(path, &bytes).with_context(|| format!("write config {path:?}"))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600));
        }
        Ok(())
    }

    /// True if `public_key` (base64) belongs to a pinned peer.
    pub fn is_paired(&self, public_key: &str) -> bool {
        self.peers.iter().any(|p| p.public_key == public_key)
    }

    pub fn peer_name(&self, public_key: &str) -> Option<&str> {
        self.peers
            .iter()
            .find(|p| p.public_key == public_key)
            .map(|p| p.name.as_str())
    }

    /// Add or refresh a pinned peer (idempotent on the key).
    pub fn upsert_peer(&mut self, name: &str, public_key: &str) {
        let paired_at = now_millis();
        if let Some(existing) = self.peers.iter_mut().find(|p| p.public_key == public_key) {
            existing.name = name.to_string();
            existing.paired_at = paired_at;
        } else {
            self.peers.push(Peer {
                name: name.to_string(),
                public_key: public_key.to_string(),
                paired_at,
            });
        }
    }
}

/// Resolve the daemon's config directory (`<data-dir>/ghostftp-agentd`), honouring an
/// explicit override. Created if missing.
pub fn config_dir(override_dir: Option<PathBuf>) -> Result<PathBuf> {
    let dir = match override_dir {
        Some(d) => d,
        None => dirs::data_dir()
            .context("no OS data dir")?
            .join("ghostftp-agentd"),
    };
    std::fs::create_dir_all(&dir).with_context(|| format!("create config dir {dir:?}"))?;
    Ok(dir)
}

pub fn identity_path(dir: &Path) -> PathBuf {
    dir.join("identity.json")
}

pub fn config_path(dir: &Path) -> PathBuf {
    dir.join("agentd.json")
}

pub fn now_millis() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}
