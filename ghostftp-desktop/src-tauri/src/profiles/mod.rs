use anyhow::{bail, Context, Result};
use serde::{Deserialize, Serialize};
use std::io::Write;
#[cfg(unix)]
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use tauri::{AppHandle, Manager};
use tokio::sync::Mutex;

#[derive(Clone, Serialize, Deserialize)]
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

// A ConnectionProfile is often included in structured error contexts.
// Never allow deriving Debug to print passwords, SSH key passphrases, or
// references to sensitive key material during diagnostics or panic reporting.
// Serde compatibility for historical migrations remains unchanged.
impl std::fmt::Debug for AuthMethod {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Password { .. } => formatter.write_str("Password { password: [REDACTED] }"),
            Self::Key { .. } => formatter.write_str("Key { path: [REDACTED], passphrase: [REDACTED] }"),
            Self::Agent => formatter.write_str("Agent"),
            Self::KeyRef { .. } => formatter.write_str("KeyRef { key_ref: [REDACTED] }"),
        }
    }
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
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
    // Connect automatically on app launch. Optional so existing profile JSON
    // files (written before this field existed) keep loading as `None`.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub auto_connect: Option<bool>,
    // SFTP-only reliability policy. Optional keeps existing profile JSON
    // compatible; transport defaults are applied at connect time.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub keep_alive_seconds: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reconnect_attempts: Option<u8>,
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
    // Preserve the last recoverable snapshot until the user explicitly
    // repairs storage; never silently replace malformed files with [].
    recovery_required: Option<String>,
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
        let mut recovery_required = None;
        let profiles: Vec<ConnectionProfile> = if path.exists() {
            let bytes =
                std::fs::read(&path).with_context(|| format!("reading {}", path.display()))?;
            match serde_json::from_slice(&bytes) {
                Ok(v) => v,
                Err(main_error) => {
                    // Keep corrupt bytes in a private, uniquely named snapshot,
                    // never reusing a recovery filename for the backup.
                    preserve_corrupt_copy(&path, &bytes)?;
                    recovery_required = Some(format!(
                        "profiles.json is damaged ({main_error}); profile changes are disabled until storage is repaired from its recovery copies"
                    ));
                    if backup.exists() {
                        let backup_bytes = std::fs::read(&backup)
                            .with_context(|| format!("reading {}", backup.display()))?;
                        match serde_json::from_slice(&backup_bytes) {
                            Ok(v) => v,
                            Err(_) => {
                                preserve_corrupt_copy(&backup, &backup_bytes)?;
                                Vec::new()
                            }
                        }
                    } else {
                        Vec::new()
                    }
                }
            }
        } else if backup.exists() {
            // An interrupted file promotion can leave only profiles.json.bak.
            // Expose recoverable data but do not delete the last valid copy.
            let bytes =
                std::fs::read(&backup).with_context(|| format!("reading {}", backup.display()))?;
            recovery_required = Some(
                "profiles.json is missing but profiles.json.bak exists; changes are disabled until the primary file is restored"
                    .to_string(),
            );
            match serde_json::from_slice(&bytes) {
                Ok(v) => v,
                Err(_) => {
                    preserve_corrupt_copy(&backup, &bytes)?;
                    Vec::new()
                }
            }
        } else {
            Vec::new()
        };
        Ok(Self {
            path,
            inner: Arc::new(Mutex::new(profiles)),
            recovery_required,
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
        let previous = g.clone();
        if let Some(existing) = g.iter_mut().find(|p| p.id == profile.id) {
            *existing = profile;
        } else {
            g.push(profile);
        }
        if let Err(error) = self.write(&g) {
            *g = previous;
            return Err(error);
        }
        Ok(())
    }

    /// Persist a manual rail order: each listed profile gets `sort_order` =
    /// its index. Ids not in the store are skipped; one write for the batch.
    pub async fn reorder(&self, ids: &[String]) -> Result<()> {
        let mut g = self.inner.lock().await;
        let previous = g.clone();
        for (i, id) in ids.iter().enumerate() {
            if let Some(p) = g.iter_mut().find(|p| &p.id == id) {
                p.sort_order = Some(i as u32);
            }
        }
        if let Err(error) = self.write(&g) {
            *g = previous;
            return Err(error);
        }
        Ok(())
    }

    pub async fn delete(&self, id: &str) -> Result<()> {
        let mut g = self.inner.lock().await;
        let previous = g.clone();
        g.retain(|p| p.id != id);
        if let Err(error) = self.write(&g) {
            *g = previous;
            return Err(error);
        }
        Ok(())
    }

    fn write(&self, profiles: &[ConnectionProfile]) -> Result<()> {
        if let Some(reason) = &self.recovery_required {
            bail!("{reason}");
        }
        // Defense in depth: no code path (CLI, import, reorder or future
        // command) may accidentally persist plaintext credentials. Callers
        // must migrate password/passphrase values into the OS keychain first.
        for profile in profiles {
            let unsafe_secret = match &profile.auth {
                AuthMethod::Password { password } => !password.is_empty(),
                AuthMethod::Key { passphrase, .. } => {
                    passphrase.as_deref().is_some_and(|s| !s.is_empty())
                }
                _ => false,
            };
            if unsafe_secret {
                bail!("refusing to write plaintext profile credentials; store secrets in the OS keychain first");
            }
        }
        let bytes = serde_json::to_vec_pretty(profiles)?;
        let parent = self.path.parent().context("profile path has no parent")?;
        // Never truncate a predictable temporary filename: a stale file or a
        // symlink must not redirect profile metadata writes outside app data.
        let tmp = parent.join(format!(".profiles.json.{}.tmp", uuid::Uuid::new_v4()));
        let backup = parent.join("profiles.json.bak");

        // Write + fsync a sibling temp file first, so a process/power failure
        // never leaves a half-written profiles.json.
        {
            let mut options = std::fs::OpenOptions::new();
            options.create_new(true).write(true);
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

/// Never overwrite a recovery file, and never place recovered credentials or
/// connection metadata in a world-readable default-mode file on Unix.
fn preserve_corrupt_copy(path: &Path, bytes: &[u8]) -> Result<PathBuf> {
    let parent = path
        .parent()
        .context("corrupt profile path has no parent")?;
    let source = path
        .file_name()
        .context("corrupt profile path has no filename")?
        .to_string_lossy();
    let recovery = parent.join(format!(
        ".{source}.ghostftp-corrupt-{}.recovery",
        uuid::Uuid::new_v4()
    ));
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    options.mode(0o600);
    let mut file = options
        .open(&recovery)
        .with_context(|| format!("creating private recovery copy {}", recovery.display()))?;
    file.write_all(bytes)
        .with_context(|| format!("writing recovery copy {}", recovery.display()))?;
    file.sync_all()
        .with_context(|| format!("syncing recovery copy {}", recovery.display()))?;
    #[cfg(unix)]
    std::fs::set_permissions(&recovery, std::fs::Permissions::from_mode(0o600))
        .with_context(|| format!("securing recovery copy {}", recovery.display()))?;
    Ok(recovery)
}

#[cfg(test)]
mod recovery_tests {
    use super::*;

    fn temp_profile_dir() -> PathBuf {
        let path = std::env::temp_dir().join(format!(
            "ghostftp-profile-integrity-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&path).expect("private test directory");
        path
    }

    fn recovery_files(dir: &Path) -> Vec<PathBuf> {
        std::fs::read_dir(dir)
            .expect("read test directory")
            .map(|entry| entry.expect("directory entry").path())
            .filter(|path| path.to_string_lossy().contains(".ghostftp-corrupt-"))
            .collect()
    }

    #[tokio::test]
    async fn corrupt_main_and_valid_backup_remain_readable_but_never_get_overwritten() {
        let dir = temp_profile_dir();
        let primary = dir.join("profiles.json");
        let backup = dir.join("profiles.json.bak");
        std::fs::write(&primary, b"{broken-primary").unwrap();
        std::fs::write(&backup, b"[]").unwrap();
        let store = ProfileStore::from_dir(&dir).expect("load backup read-only");
        assert!(store.list().await.unwrap().is_empty());
        assert!(store.delete("any").await.is_err());
        assert_eq!(std::fs::read(&backup).unwrap(), b"[]");
        assert_eq!(std::fs::read(&primary).unwrap(), b"{broken-primary");
        assert_eq!(recovery_files(&dir).len(), 1);
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[tokio::test]
    async fn two_corrupt_copies_are_preserved_separately_and_never_replaced() {
        let dir = temp_profile_dir();
        let primary = dir.join("profiles.json");
        let backup = dir.join("profiles.json.bak");
        std::fs::write(&primary, b"{broken-primary").unwrap();
        std::fs::write(&backup, b"{broken-backup").unwrap();
        let store = ProfileStore::from_dir(&dir).expect("open read-only");
        assert!(store.reorder(&[]).await.is_err());
        let copies = recovery_files(&dir);
        assert_eq!(copies.len(), 2);
        let bytes: Vec<Vec<u8>> = copies
            .iter()
            .map(|path| std::fs::read(path).unwrap())
            .collect();
        assert!(bytes.contains(&b"{broken-primary".to_vec()));
        assert!(bytes.contains(&b"{broken-backup".to_vec()));
        #[cfg(unix)]
        for path in &copies {
            assert_eq!(
                std::fs::metadata(path).unwrap().permissions().mode() & 0o777,
                0o600
            );
        }
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[tokio::test]
    async fn only_backup_after_interrupted_commit_is_loaded_read_only() {
        let dir = temp_profile_dir();
        let backup = dir.join("profiles.json.bak");
        std::fs::write(&backup, b"[]").unwrap();
        let store = ProfileStore::from_dir(&dir).unwrap();
        assert!(store.list().await.unwrap().is_empty());
        assert!(store.delete("any").await.is_err());
        assert_eq!(std::fs::read(&backup).unwrap(), b"[]");
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[tokio::test]
    async fn predictable_legacy_staging_file_is_not_overwritten() {
        let dir = temp_profile_dir();
        let legacy = dir.join("profiles.json.tmp");
        std::fs::write(&legacy, b"keep-existing-staging").unwrap();
        let store = ProfileStore::from_dir(&dir).unwrap();
        store.reorder(&[]).await.unwrap();
        assert_eq!(std::fs::read(&legacy).unwrap(), b"keep-existing-staging");
        assert_eq!(std::fs::read(dir.join("profiles.json")).unwrap(), b"[]");
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[test]
    fn auth_debug_and_profile_debug_never_expose_in_memory_secrets() {
        let password = AuthMethod::Password {
            password: "PRIVATE_PASSWORD_SENTINEL".to_string(),
        };
        let key = AuthMethod::Key {
            path: "/private/keys/hidden_ed25519".to_string(),
            passphrase: Some("PRIVATE_PASSPHRASE_SENTINEL".to_string()),
        };
        let key_ref = AuthMethod::KeyRef {
            key_ref: "grant-key:PRIVATE_KEY_REFERENCE".to_string(),
        };
        for method in [&password, &key, &key_ref] {
            let debug = format!("{method:?}");
            assert!(debug.contains("[REDACTED]"));
            for forbidden in [
                "PRIVATE_PASSWORD_SENTINEL",
                "PRIVATE_PASSPHRASE_SENTINEL",
                "/private/keys/hidden_ed25519",
                "PRIVATE_KEY_REFERENCE",
            ] {
                assert!(!debug.contains(forbidden), "Debug emitted a secret or key reference");
            }
        }
        assert_eq!(format!("{:?}", AuthMethod::Agent), "Agent");
        // The legacy serde representation must still deserialize; this is a
        // diagnostic fix, not a destructive profile schema migration.
        let legacy = r#"{"kind":"password","password":"PRIVATE_PASSWORD_SENTINEL"}"#;
        let restored: AuthMethod = serde_json::from_str(legacy).unwrap();
        assert!(!format!("{restored:?}").contains("PRIVATE_PASSWORD_SENTINEL"));
    }

    #[tokio::test]
    async fn profile_store_never_serializes_plaintext_credentials() {
        let dir = temp_profile_dir();
        let store = ProfileStore::from_dir(&dir).unwrap();
        let base = serde_json::json!({
            "id":"id-1","name":"Example","protocol":"sftp","host":"example.invalid",
            "port":22,"username":"user","auth":{"kind":"password","password":"SUPER_SECRET_TEST"}
        });
        let with_password: ConnectionProfile = serde_json::from_value(base.clone()).unwrap();
        let error = store.upsert(with_password).await.unwrap_err();
        assert!(error.to_string().contains("OS keychain"));
        assert!(!dir.join("profiles.json").exists());
        assert!(store.list().await.unwrap().is_empty());

        let mut with_key = base;
        with_key["auth"] = serde_json::json!({
            "kind":"key","path":"/id_ed25519","passphrase":"NEVER_STORE_PASSPHRASE"
        });
        let with_key: ConnectionProfile = serde_json::from_value(with_key).unwrap();
        assert!(store.upsert(with_key).await.is_err());
        assert!(!dir.join("profiles.json").exists());

        let mut safe = serde_json::json!({
            "id":"id-1","name":"Example","protocol":"sftp","host":"example.invalid",
            "port":22,"username":"user","auth":{"kind":"password","password":""}
        });
        store
            .upsert(serde_json::from_value(safe.take()).unwrap())
            .await
            .unwrap();
        let saved = std::fs::read_to_string(dir.join("profiles.json")).unwrap();
        assert!(!saved.contains("SUPER_SECRET_TEST"));
        assert!(!saved.contains("NEVER_STORE_PASSPHRASE"));
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[tokio::test]
    async fn healthy_store_still_persists_updates() {
        let dir = temp_profile_dir();
        let store = ProfileStore::from_dir(&dir).unwrap();
        store.reorder(&[]).await.unwrap();
        assert_eq!(std::fs::read(dir.join("profiles.json")).unwrap(), b"[]");
        drop(store);
        std::fs::remove_dir_all(dir).unwrap();
    }
}
