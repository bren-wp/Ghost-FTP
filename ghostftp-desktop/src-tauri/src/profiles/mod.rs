use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::io::Write;
#[cfg(unix)]
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tauri::{AppHandle, Manager};
use tokio::sync::Mutex;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
pub enum AuthMethod {
    Password {
        password: String,
    },
    Key {
        path: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        passphrase: Option<String>,
    },
    Agent,
    /// A reference to private-key material stored in the OS keychain (under
    /// `key_ref`, e.g. `grant-key:<profile-id>`), never on disk. The connect
    /// path resolves it via `credentials::get_secret` at connect time.
    #[serde(rename_all = "camelCase")]
    KeyRef {
        key_ref: String,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConnectionProfile {
    pub id: String,
    pub name: String,
    pub protocol: String, // "sftp" | "ftp" | "ftps" | "s3"
    pub host: String,
    pub port: u16,
    pub username: String,
    pub auth: AuthMethod,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_remote_path: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
    // Connect automatically on app launch. Optional so existing profile JSON
    // files (written before this field existed) keep loading as `None`.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub auto_connect: Option<bool>,
    // Object-store-specific fields. Unused for sftp/ftp protocols. We keep
    // them optional so existing profile JSON files keep loading.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bucket: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub region: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub endpoint: Option<String>,
    // Azure Blob specific. `account` is the Azure storage account name,
    // and `bucket` doubles as the container. For S3, both stay absent.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub account: Option<String>,
    // Ghost FTP Agent (protocol "ghostftp-agent") specific: the daemon's pinned static
    // public key (base64), learned at pairing time. Its presence means this
    // profile is paired; absence means the user still needs to pair with a code.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_key: Option<String>,
    // Rail organisation. `group` is a free-form folder name (absent = ungrouped);
    // `sort_order` is the manual drag-and-drop position. Profiles without one
    // sort after ordered ones, by protocol/name.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub favorite: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bookmarked: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_used: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sort_order: Option<u32>,
    // Custom bubble glyph for the rail: an emoji (any short string without a
    // colon) or a bundled Iconify key ("prefix:name"). Absent = the name
    // monogram. Optional so existing profile JSON files keep loading.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icon: Option<String>,
    // Single-hop bastion (ProxyJump). When `jump_host` is set, the SSH connect
    // path first connects there with the SAME auth material, then tunnels to
    // `host:port` over a direct-tcpip channel. `jump_port` defaults to 22,
    // `jump_username` defaults to `username`.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub jump_host: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub jump_port: Option<u16>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub jump_username: Option<String>,
}

// JSON metadata file in the app data dir. Passwords and private-key passphrases
// are stripped by the command layer into the OS credential store before writes.
pub struct ProfileStore {
    path: PathBuf,
    inner: Arc<Mutex<Vec<ConnectionProfile>>>,
}

impl ProfileStore {
    pub fn load_or_create(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        Self::from_dir(&dir)
    }

    /// Load (or initialise) the profile store from a known directory. Used by
    /// the CLI binary, which doesn't have a Tauri AppHandle. Resolves to the
    /// same on-disk path the GUI uses — see `default_data_dir`.
    pub fn from_dir(dir: &Path) -> Result<Self> {
        std::fs::create_dir_all(dir).with_context(|| format!("creating {}", dir.display()))?;
        #[cfg(unix)]
        {
            // Connection metadata contains server names and usernames even after
            // secrets are stripped. Keep the per-user app-data directory private.
            std::fs::set_permissions(dir, std::fs::Permissions::from_mode(0o700))
                .with_context(|| format!("securing {}", dir.display()))?;
        }
        let path = dir.join("profiles.json");
        let backup = dir.join("profiles.json.bak");
        let profiles: Vec<ConnectionProfile> = if path.exists() {
            let bytes =
                std::fs::read(&path).with_context(|| format!("reading {}", path.display()))?;
            match serde_json::from_slice(&bytes) {
                Ok(v) => v,
                Err(_main_err) => {
                    // A partial/corrupt profile write must not crash Ghost FTP or
                    // silently destroy the last known-good data. Prefer the backup
                    // and preserve the broken file for manual recovery/support.
                    if backup.exists() {
                        let backup_bytes = std::fs::read(&backup)
                            .with_context(|| format!("reading {}", backup.display()))?;
                        if let Ok(v) = serde_json::from_slice(&backup_bytes) {
                            preserve_corrupt_copy(&path, &bytes);
                            v
                        } else {
                            // Both copies are corrupt. Preserve both for support/recovery,
                            // then start with an empty store instead of crashing the entire
                            // application during startup. No corrupt data is silently deleted.
                            preserve_corrupt_copy(&path, &bytes);
                            preserve_corrupt_copy(&backup, &backup_bytes);
                            Vec::new()
                        }
                    } else {
                        // A malformed profiles.json must not make Ghost FTP unusable. Keep
                        // the broken file for recovery and continue with an empty profile
                        // list; the next successful write will create a new known-good file.
                        preserve_corrupt_copy(&path, &bytes);
                        Vec::new()
                    }
                }
            }
        } else {
            Vec::new()
        };
        Ok(Self {
            path,
            inner: Arc::new(Mutex::new(profiles)),
        })
    }

    pub async fn list(&self) -> Result<Vec<ConnectionProfile>> {
        Ok(self.inner.lock().await.clone())
    }

    pub async fn get(&self, id: &str) -> Result<Option<ConnectionProfile>> {
        Ok(self.inner.lock().await.iter().find(|p| p.id == id).cloned())
    }

    pub async fn upsert(&self, profile: ConnectionProfile) -> Result<()> {
        let mut g = self.inner.lock().await;
        if let Some(existing) = g.iter_mut().find(|p| p.id == profile.id) {
            *existing = profile;
        } else {
            g.push(profile);
        }
        self.write(&g)?;
        Ok(())
    }

    /// Persist a manual rail order: each listed profile gets `sort_order` =
    /// its index. Ids not in the store are skipped; one write for the batch.
    pub async fn reorder(&self, ids: &[String]) -> Result<()> {
        let mut g = self.inner.lock().await;
        for (i, id) in ids.iter().enumerate() {
            if let Some(p) = g.iter_mut().find(|p| &p.id == id) {
                p.sort_order = Some(i as u32);
            }
        }
        self.write(&g)?;
        Ok(())
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        let mut g = self.inner.lock().await;
        g.retain(|p| p.id != id);
        self.write(&g)?;
        Ok(())
    }

    fn write(&self, profiles: &[ConnectionProfile]) -> Result<()> {
        let bytes = serde_json::to_vec_pretty(profiles)?;
        let parent = self.path.parent().context("profile path has no parent")?;
        let tmp = parent.join("profiles.json.tmp");
        let backup = parent.join("profiles.json.bak");

        // Write + fsync a sibling temp file first, so a process/power failure
        // never leaves a half-written profiles.json.
        {
            let mut options = std::fs::OpenOptions::new();
            options.create(true).truncate(true).write(true);
            #[cfg(unix)]
            options.mode(0o600);
            let mut file = options
                .open(&tmp)
                .with_context(|| format!("opening {}", tmp.display()))?;
            #[cfg(unix)]
            std::fs::set_permissions(&tmp, std::fs::Permissions::from_mode(0o600))
                .with_context(|| format!("securing {}", tmp.display()))?;
            file.write_all(&bytes)
                .with_context(|| format!("writing {}", tmp.display()))?;
            file.sync_all()
                .with_context(|| format!("syncing {}", tmp.display()))?;
        }

        if self.path.exists() {
            let _ = std::fs::remove_file(&backup);
            std::fs::rename(&self.path, &backup)
                .with_context(|| format!("backing up {}", self.path.display()))?;
        }
        if let Err(err) = std::fs::rename(&tmp, &self.path) {
            // Best-effort rollback to the previous known-good store.
            if backup.exists() && !self.path.exists() {
                let _ = std::fs::rename(&backup, &self.path);
            }
            return Err(err).with_context(|| format!("committing {}", self.path.display()));
        }
        #[cfg(unix)]
        {
            std::fs::set_permissions(&self.path, std::fs::Permissions::from_mode(0o600))
                .with_context(|| format!("securing {}", self.path.display()))?;
            if backup.exists() {
                let _ = std::fs::set_permissions(&backup, std::fs::Permissions::from_mode(0o600));
            }
        }
        Ok(())
    }
}

fn preserve_corrupt_copy(path: &Path, bytes: &[u8]) {
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    if let Some(parent) = path.parent() {
        let recovery = parent.join(format!("profiles.corrupt.{stamp}.json"));
        let _ = std::fs::write(recovery, bytes);
    }
}
