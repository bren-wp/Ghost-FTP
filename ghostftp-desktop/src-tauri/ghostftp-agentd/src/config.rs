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
        // A new installation is read-only until the owner explicitly grants
        // dangerous operations. Legacy persisted policy booleans are retained.
        Self {
            allow_exec: false,
            allow_write: false,
        }
    }
}

/// Least-privilege grants for one pinned controller. These privileges are
/// independent from the daemon-wide emergency restrictions in Policy.
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase", default)]
pub struct PeerPermissions {
    pub read: bool,
    pub write: bool,
    pub delete: bool,
    pub terminal: bool,
}
impl Default for PeerPermissions {
    fn default() -> Self {
        Self {
            read: true,
            write: false,
            delete: false,
            terminal: false,
        }
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
    /// None only on legacy on-disk peers. Load migrates their effective
    /// permissions once; new peers always receive explicit read-only grants.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub permissions: Option<PeerPermissions>,
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
        let mut cfg: Self =
            serde_json::from_slice(&bytes).with_context(|| format!("parse config {path:?}"))?;
        let mut migrated = false;
        for peer in &mut cfg.peers {
            if peer.permissions.is_none() {
                peer.permissions = Some(PeerPermissions {
                    read: true,
                    write: cfg.policy.allow_write,
                    delete: cfg.policy.allow_write,
                    terminal: cfg.policy.allow_exec,
                });
                migrated = true;
            }
        }
        if migrated {
            // Do not lose legacy grants when a later administrator edits the
            // global policy; migration must be persisted or refuse startup.
            cfg.save(path)
                .context("persist legacy peer permission migration")?;
        }
        Ok(cfg)
    }

    pub fn save(&self, path: &Path) -> Result<()> {
        use std::io::Write;
        let parent = path
            .parent()
            .context("agent config has no parent directory")?;
        std::fs::create_dir_all(parent).context("create agent config directory")?;
        let bytes = serde_json::to_vec_pretty(self)?;
        // Use a private, same-directory temporary file and atomic rename;
        // never truncate the only copy of pins or permissions on a crash.
        let tmp = path.with_extension(format!("json.{}.tmp", std::process::id()));
        let mut opts = std::fs::OpenOptions::new();
        opts.write(true).create_new(true);
        #[cfg(unix)]
        {
            use std::os::unix::fs::OpenOptionsExt;
            opts.mode(0o600);
        }
        let result = (|| -> Result<()> {
            let mut out = opts
                .open(&tmp)
                .context("create private agent config staging")?;
            out.write_all(&bytes).context("write staged agent config")?;
            out.sync_all().context("sync staged agent config")?;
            std::fs::rename(&tmp, path).context("atomically replace agent config")?;
            Ok(())
        })();
        if result.is_err() {
            let _ = std::fs::remove_file(&tmp);
        }
        result
    }

    /// True if `public_key` (base64) belongs to a pinned peer.
    pub fn is_paired(&self, public_key: &str) -> bool {
        self.peers.iter().any(|p| p.public_key == public_key)
    }

    pub fn peer_permissions(&self, public_key: &str) -> Option<PeerPermissions> {
        self.peers
            .iter()
            .find(|p| p.public_key == public_key)
            .map(|p| p.permissions.unwrap_or_default())
    }

    pub fn set_peer_permissions(&mut self, public_key: &str, permissions: PeerPermissions) -> bool {
        if let Some(peer) = self.peers.iter_mut().find(|p| p.public_key == public_key) {
            peer.permissions = Some(permissions);
            return true;
        }
        false
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
                permissions: Some(PeerPermissions::default()),
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

#[cfg(test)]
mod security_tests {
    use super::*;

    #[test]
    fn new_pairing_defaults_to_read_only_even_with_legacy_global_access() {
        let mut cfg = Config::default();
        assert!(!cfg.policy.allow_exec && !cfg.policy.allow_write);
        cfg.policy.allow_exec = true;
        cfg.policy.allow_write = true;
        cfg.upsert_peer("controller", "public-A");
        let grants = cfg.peer_permissions("public-A").unwrap();
        assert!(grants.read);
        assert!(!grants.write && !grants.delete && !grants.terminal);
    }

    #[test]
    fn legacy_migration_preserves_previous_grants_and_is_idempotent() {
        let path = std::env::temp_dir().join(format!(
            "ghostftp-agent-security-{}.json",
            std::process::id()
        ));
        std::fs::write(&path, r#"{"policy":{"allowExec":true,"allowWrite":true},"peers":[{"name":"old","publicKey":"legacy","pairedAt":1}]}"#).unwrap();
        let migrated = Config::load(&path).unwrap();
        assert_eq!(
            migrated.peer_permissions("legacy").unwrap(),
            PeerPermissions {
                read: true,
                write: true,
                delete: true,
                terminal: true
            }
        );
        let again = Config::load(&path).unwrap();
        assert!(again.peers[0].permissions.is_some());
        std::fs::remove_file(path).unwrap();
    }
}
