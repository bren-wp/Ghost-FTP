//! Encrypted backup / restore (Plan 12 Phase 4).
//!
//! Ghost FTP holds real value on a machine: thirteen backends' worth of connection
//! profiles, their keychain secrets, and the `ghostftp.db` state. Moving to a new
//! machine shouldn't mean re-entering everything. This module packs it all into
//! a single password-protected container that's safe to email or drop on a USB
//! stick.
//!
//! ## Container format
//!
//! ```text
//! ┌───────────────────────────────────────────────────────────────┐
//! │ MAGIC   12 bytes  b"GHOSTFTPBAK\x01"                                 │
//! │ VERSION 1 byte   schema version (1)                            │  ── AEAD
//! │ SALT    16 bytes Argon2id salt                                 │   associated
//! │ NONCE   12 bytes AES-GCM nonce                                 │   data (AAD)
//! ├───────────────────────────────────────────────────────────────┤
//! │ CIPHERTEXT  AES-256-GCM( gzip(archive JSON) )  + 16-byte tag   │
//! └───────────────────────────────────────────────────────────────┘
//! ```
//!
//! The key is Argon2id(password, salt) with m=64 MiB, t=3, p=1. The whole
//! header is the AEAD's associated data, so a wrong password (or a tampered
//! header) fails the GCM tag cleanly — the archive never partially decrypts.
//!
//! ## Contents
//! - `profiles.json` — connection profiles (secrets already live in the keychain)
//! - `ghostftp.db` — a WAL-safe snapshot of the state DB (settings, sync index, …)
//! - `bridge.json`, `foldersync.json` — subsystem configs
//! - every keychain credential — the app's service keys (via the `ghostftp.db`
//!   manifest) plus each cloud profile's OAuth tokens, so the restored machine
//!   works without re-authorizing anything.
//!
//! ## Restore
//! Restore stages the files (writing `<name>.restore` next to each) and injects
//! the credentials straight into the keychain; [`apply_pending_restore`] swaps
//! the staged files in at the next startup, before anything opens them. The CLI
//! path applies immediately (no running app to coordinate with).

use std::collections::HashSet;
#[cfg(unix)]
use std::fs::OpenOptions;
use std::io::{Read, Write};
#[cfg(unix)]
use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};

use aes_gcm::aead::{Aead, KeyInit, Payload};
use aes_gcm::{Aes256Gcm, Key, Nonce};
use anyhow::{anyhow, bail, Context, Result};
use argon2::{Algorithm, Argon2, Params, Version};
use base64::Engine as _;
use rand::RngCore;
use serde::{Deserialize, Serialize};

use crate::db::Db;

const MAGIC: &[u8; 12] = b"GHOSTFTPBAK\x01";
const VERSION: u8 = 1;
const SALT_LEN: usize = 16;
const NONCE_LEN: usize = 12;
/// Argon2id memory cost in KiB (64 MiB).
const ARGON_MEM_KIB: u32 = 65_536;
const ARGON_TIME: u32 = 3;
const ARGON_LANES: u32 = 1;
const MAX_BACKUP_FILE_BYTES: u64 = 512 * 1024 * 1024;
const MAX_DECOMPRESSED_BYTES: usize = 768 * 1024 * 1024;
const MAX_CREDENTIALS: usize = 4_096;
const MAX_CREDENTIAL_SERVICE_BYTES: usize = 256;
const MAX_CREDENTIAL_ACCOUNT_BYTES: usize = 1_024;
const MIN_EXPORT_PASSWORD_BYTES: usize = 12;
const MAX_BACKUP_PASSWORD_BYTES: usize = 1_024;

/// The config files carried in a backup, in the app data dir. `ghostftp.db` is
/// handled separately (it needs a WAL-safe snapshot on export and staging on
/// restore).
const CONFIG_FILES: &[&str] = &["profiles.json", "bridge.json", "foldersync.json"];

/// What a backup contains — shown in the UI's "what's inside" summary and
/// returned after an import so the caller can report what was restored.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BackupSummary {
    pub profiles: usize,
    pub credentials: usize,
    pub has_bridge: bool,
    pub has_sync: bool,
    pub db_bytes: usize,
}

#[derive(Serialize, Deserialize)]
struct Cred {
    service: String,
    account: String,
    secret: String,
}

#[derive(Serialize, Deserialize)]
struct Archive {
    created_ms: i64,
    #[serde(default)]
    profiles_json: Option<String>,
    #[serde(default)]
    bridge_json: Option<String>,
    #[serde(default)]
    foldersync_json: Option<String>,
    /// base64 of the `ghostftp.db` snapshot bytes.
    ghostftp_db_b64: String,
    #[serde(default)]
    credentials: Vec<Cred>,
}

impl Archive {
    fn summary(&self) -> BackupSummary {
        let profiles = self
            .profiles_json
            .as_deref()
            .and_then(|s| serde_json::from_str::<serde_json::Value>(s).ok())
            .and_then(|v| v.as_array().map(|a| a.len()))
            .unwrap_or(0);
        BackupSummary {
            profiles,
            credentials: self.credentials.len(),
            has_bridge: self.bridge_json.is_some(),
            has_sync: self.foldersync_json.is_some(),
            db_bytes: self.ghostftp_db_b64.len(),
        }
    }
}

struct RemoveOnDrop(PathBuf);

impl Drop for RemoveOnDrop {
    fn drop(&mut self) {
        let _ = std::fs::remove_file(&self.0);
    }
}

fn write_private_file(path: &Path, data: &[u8]) -> Result<()> {
    #[cfg(unix)]
    {
        let mut file = OpenOptions::new()
            .create(true)
            .write(true)
            .truncate(true)
            .mode(0o600)
            .open(path)
            .with_context(|| format!("open private file {}", path.display()))?;
        let mut permissions = file
            .metadata()
            .with_context(|| format!("read permissions for {}", path.display()))?
            .permissions();
        permissions.set_mode(0o600);
        file.set_permissions(permissions)
            .with_context(|| format!("set private permissions for {}", path.display()))?;
        file.write_all(data)
            .with_context(|| format!("write private file {}", path.display()))?;
        file.sync_all()
            .with_context(|| format!("sync private file {}", path.display()))?;
        Ok(())
    }

    #[cfg(not(unix))]
    {
        std::fs::write(path, data)
            .with_context(|| format!("write private file {}", path.display()))?;
        Ok(())
    }
}

fn credential_target_allowed(service: &str, account: &str) -> bool {
    let allowed_services = [
        crate::credentials::SERVICE,
        crate::session::dropbox::DROPBOX_SERVICE,
        crate::session::onedrive::ONEDRIVE_SERVICE,
        crate::session::gdrive::GDRIVE_SERVICE,
        crate::session::boxdrive::BOX_SERVICE,
    ];
    allowed_services.contains(&service)
        && !service.is_empty()
        && service.len() <= MAX_CREDENTIAL_SERVICE_BYTES
        && !account.is_empty()
        && account.len() <= MAX_CREDENTIAL_ACCOUNT_BYTES
        && !service.chars().any(char::is_control)
        && !account.chars().any(char::is_control)
}

fn validate_export_password(password: &str) -> Result<()> {
    let len = password.len();
    if len < MIN_EXPORT_PASSWORD_BYTES {
        bail!("backup password must be at least {MIN_EXPORT_PASSWORD_BYTES} bytes");
    }
    if len > MAX_BACKUP_PASSWORD_BYTES {
        bail!("backup password exceeds the maximum supported length");
    }
    Ok(())
}

fn validate_decrypt_password(password: &str) -> Result<()> {
    if password.is_empty() {
        bail!("a backup password is required");
    }
    if password.len() > MAX_BACKUP_PASSWORD_BYTES {
        bail!("backup password exceeds the maximum supported length");
    }
    Ok(())
}

fn wipe_string(value: &mut String) {
    let mut bytes = std::mem::take(value).into_bytes();
    bytes.fill(0);
}

fn clear_archive_memory(archive: &mut Archive) {
    if let Some(value) = archive.profiles_json.as_mut() {
        wipe_string(value);
    }
    if let Some(value) = archive.bridge_json.as_mut() {
        wipe_string(value);
    }
    if let Some(value) = archive.foldersync_json.as_mut() {
        wipe_string(value);
    }
    wipe_string(&mut archive.ghostftp_db_b64);
    for credential in &mut archive.credentials {
        wipe_string(&mut credential.secret);
    }
}

fn validate_archive(archive: &Archive) -> Result<()> {
    if archive.credentials.len() > MAX_CREDENTIALS {
        bail!("backup contains too many credential records");
    }
    let mut seen = HashSet::new();
    for credential in &archive.credentials {
        if !credential_target_allowed(&credential.service, &credential.account) {
            bail!("backup contains an unsupported credential target");
        }
        if !seen.insert((&credential.service, &credential.account)) {
            bail!("backup contains duplicate credential targets");
        }
    }
    // Metadata must not re-introduce secrets from an older or crafted archive.
    // Passwords and key passphrases belong exclusively in the encrypted
    // credential collection and are restored through the OS keychain.
    if let Some(ref json) = archive.profiles_json {
        let profiles: Vec<crate::profiles::ConnectionProfile> =
            serde_json::from_str(json).context("invalid profile metadata in backup")?;
        for profile in profiles {
            let plaintext = match profile.auth {
                crate::profiles::AuthMethod::Password { password } => !password.is_empty(),
                crate::profiles::AuthMethod::Key { passphrase, .. } => {
                    passphrase.as_deref().is_some_and(|s| !s.is_empty())
                }
                _ => false,
            };
            if plaintext {
                bail!("backup contains plaintext profile credentials");
            }
        }
    }
    Ok(())
}

impl Drop for Archive {
    fn drop(&mut self) {
        clear_archive_memory(self);
    }
}

trait RestoreVault {
    fn get(&mut self, service: &str, account: &str) -> Result<Option<String>>;
    fn put(&mut self, service: &str, account: &str, secret: &str) -> Result<()>;
    fn remove(&mut self, service: &str, account: &str) -> Result<()>;
}

struct OsVault;

impl RestoreVault for OsVault {
    fn get(&mut self, service: &str, account: &str) -> Result<Option<String>> {
        let entry = keyring::Entry::new(service, account).context("open OS credential vault")?;
        match entry.get_password() {
            Ok(secret) => Ok(Some(secret)),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(_) => bail!("OS credential vault is unavailable for reading"),
        }
    }

    fn put(&mut self, service: &str, account: &str, secret: &str) -> Result<()> {
        let entry = keyring::Entry::new(service, account).context("open OS credential vault")?;
        entry
            .set_password(secret)
            .map_err(|_| anyhow!("OS credential vault refused a secret write"))
    }

    fn remove(&mut self, service: &str, account: &str) -> Result<()> {
        let entry = keyring::Entry::new(service, account).context("open OS credential vault")?;
        match entry.delete_credential() {
            Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
            Err(_) => bail!("OS credential vault refused rollback"),
        }
    }
}

struct PreviousSecret(Option<String>);

impl Drop for PreviousSecret {
    fn drop(&mut self) {
        if let Some(secret) = self.0.as_mut() {
            wipe_string(secret);
        }
    }
}

/// Read every existing value before the first write. Never announce a
/// successful restore if the keychain is locked/unavailable or a write fails.
/// Revert touched records to their pre-restore values on partial failure.
fn restore_credentials_with_vault(
    credentials: &[Cred],
    vault: &mut impl RestoreVault,
) -> Result<()> {
    let mut previous = Vec::with_capacity(credentials.len());
    for c in credentials {
        previous.push(PreviousSecret(vault.get(&c.service, &c.account)?));
    }

    for (index, credential) in credentials.iter().enumerate() {
        if vault
            .put(&credential.service, &credential.account, &credential.secret)
            .is_err()
        {
            let mut rollback_failed = false;
            // Include the failing entry: some OS backends may have written
            // successfully before reporting an error.
            for restored in (0..=index).rev() {
                let c = &credentials[restored];
                let old = &previous[restored].0;
                let result = match old {
                    Some(secret) => vault.put(&c.service, &c.account, secret),
                    None => vault.remove(&c.service, &c.account),
                };
                if result.is_err() {
                    rollback_failed = true;
                }
            }
            if rollback_failed {
                bail!("OS vault restore failed and credential rollback was incomplete");
            }
            bail!("OS vault restore failed; original credentials were restored");
        }
    }

    Ok(())
}

// ---- Export ----

/// Build an encrypted backup of everything under `dir` (+ its keychain
/// credentials) and write it to `dest`.
pub fn export(dir: &Path, db: &Db, password: &str, dest: &Path) -> Result<BackupSummary> {
    export_with_vault(dir, db, password, dest, &mut OsVault)
}

fn export_with_vault(
    dir: &Path,
    db: &Db,
    password: &str,
    dest: &Path,
    vault: &mut impl RestoreVault,
) -> Result<BackupSummary> {
    validate_export_password(password)?;

    // WAL-safe DB snapshot into a temp file. The guard removes it on every
    // return path, including snapshot/read/encryption failures.
    let tmp = dir.join("ghostftp.db.backup.tmp");
    let _tmp_guard = RemoveOnDrop(tmp.clone());
    db.snapshot_to(&tmp).context("snapshot ghostftp.db")?;
    let mut db_bytes = std::fs::read(&tmp).context("read ghostftp.db snapshot")?;

    let read_opt = |name: &str| -> Option<String> { std::fs::read_to_string(dir.join(name)).ok() };

    let mut archive = Archive {
        created_ms: crate::db::now_ms(),
        profiles_json: read_opt("profiles.json"),
        bridge_json: read_opt("bridge.json"),
        foldersync_json: read_opt("foldersync.json"),
        ghostftp_db_b64: base64::engine::general_purpose::STANDARD.encode(&db_bytes),
        credentials: collect_credentials(dir, db, vault)?,
    };
    validate_archive(&archive)?;
    let summary = archive.summary();

    let mut plaintext = serde_json::to_vec(&archive).context("serialize archive")?;
    db_bytes.fill(0);
    clear_archive_memory(&mut archive);
    let mut gz = gzip(&plaintext).context("gzip archive")?;

    // Fresh random salt + nonce for every export.
    let mut salt = [0u8; SALT_LEN];
    let mut nonce = [0u8; NONCE_LEN];
    rand::rngs::OsRng.fill_bytes(&mut salt);
    rand::rngs::OsRng.fill_bytes(&mut nonce);

    let header = build_header(&salt, &nonce);
    let mut key = derive_key(password, &salt)?;
    let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(&key));
    let encrypted = cipher.encrypt(
        Nonce::from_slice(&nonce),
        Payload {
            msg: &gz,
            aad: &header,
        },
    );
    drop(cipher);
    key.fill(0);
    plaintext.fill(0);
    gz.fill(0);
    let ciphertext = encrypted.map_err(|_| anyhow!("encryption failed"))?;

    let mut out = header;
    out.extend_from_slice(&ciphertext);
    write_private_file(dest, &out)
        .with_context(|| format!("write backup to {}", dest.display()))?;

    Ok(summary)
}

/// Inspect an existing backup without applying it — decrypts + returns the
/// summary (used by the UI to show "what's inside" before restoring).
pub fn inspect(password: &str, src: &Path) -> Result<BackupSummary> {
    Ok(decrypt(password, src)?.summary())
}

// ---- Import / restore ----

/// Decrypt `src` and restore it into `dir`. When `defer` is true (the GUI, whose
/// app has files open), config + DB files are *staged* as `<name>.restore` and
/// applied at the next startup by [`apply_pending_restore`]; credentials go
/// straight to the keychain. When false (the CLI, no running app), everything is
/// applied immediately.
pub fn import(dir: &Path, password: &str, src: &Path, defer: bool) -> Result<BackupSummary> {
    import_with_vault(dir, password, src, defer, &mut OsVault)
}

fn import_with_vault(
    dir: &Path,
    password: &str,
    src: &Path,
    defer: bool,
    vault: &mut impl RestoreVault,
) -> Result<BackupSummary> {
    let archive = decrypt(password, src)?;
    let summary = archive.summary();

    // Validate the DB payload before modifying credentials or staging anything.
    let mut db_bytes = base64::engine::general_purpose::STANDARD
        .decode(archive.ghostftp_db_b64.as_bytes())
        .context("decode ghostftp.db snapshot")?;

    // Restore through the OS vault only. If it is locked, unreadable or refuses
    // a write, no .restore files are staged and no success is reported.
    restore_credentials_with_vault(&archive.credentials, vault)
        .context("restore backup credentials")?;

    // Stage configuration only after vault writes have been confirmed.
    stage_opt(dir, "profiles.json", archive.profiles_json.as_deref())?;
    stage_opt(dir, "bridge.json", archive.bridge_json.as_deref())?;
    stage_opt(dir, "foldersync.json", archive.foldersync_json.as_deref())?;
    let staged = write_private_file(&dir.join("ghostftp.db.restore"), &db_bytes)
        .context("stage ghostftp.db");
    db_bytes.fill(0);
    staged?;

    if !defer {
        apply_pending_restore(dir);
    }
    Ok(summary)
}

/// Swap any staged restore files into place. Called once at startup, *before*
/// anything opens `profiles.json` / `ghostftp.db`, so a restore applies atomically
/// on the next launch. A no-op when there's nothing staged.
pub fn apply_pending_restore(dir: &Path) {
    for name in CONFIG_FILES {
        let staged = dir.join(format!("{name}.restore"));
        if staged.exists() {
            let _ = std::fs::rename(&staged, dir.join(name));
        }
    }
    let staged_db = dir.join("ghostftp.db.restore");
    if staged_db.exists() {
        // Drop the live DB and its WAL sidecars before swapping the snapshot in.
        let _ = std::fs::remove_file(dir.join("ghostftp.db"));
        let _ = std::fs::remove_file(dir.join("ghostftp.db-wal"));
        let _ = std::fs::remove_file(dir.join("ghostftp.db-shm"));
        let _ = std::fs::rename(&staged_db, dir.join("ghostftp.db"));
    }
}

fn stage_opt(dir: &Path, name: &str, content: Option<&str>) -> Result<()> {
    let staged = dir.join(format!("{name}.restore"));
    match content {
        Some(c) => {
            write_private_file(&staged, c.as_bytes()).with_context(|| format!("stage {name}"))?
        }
        // Nothing to restore for this file — clear any leftover staging.
        None => {
            let _ = std::fs::remove_file(&staged);
        }
    }
    Ok(())
}

// ---- Crypto helpers ----

fn build_header(salt: &[u8; SALT_LEN], nonce: &[u8; NONCE_LEN]) -> Vec<u8> {
    let mut h = Vec::with_capacity(MAGIC.len() + 1 + SALT_LEN + NONCE_LEN);
    h.extend_from_slice(MAGIC);
    h.push(VERSION);
    h.extend_from_slice(salt);
    h.extend_from_slice(nonce);
    h
}

fn derive_key(password: &str, salt: &[u8]) -> Result<[u8; 32]> {
    let params = Params::new(ARGON_MEM_KIB, ARGON_TIME, ARGON_LANES, Some(32))
        .map_err(|e| anyhow!("argon2 params: {e}"))?;
    let argon = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    let mut key = [0u8; 32];
    argon
        .hash_password_into(password.as_bytes(), salt, &mut key)
        .map_err(|e| anyhow!("argon2 key derivation: {e}"))?;
    Ok(key)
}

fn decrypt(password: &str, src: &Path) -> Result<Archive> {
    validate_decrypt_password(password)?;
    let metadata =
        std::fs::metadata(src).with_context(|| format!("inspect backup {}", src.display()))?;
    if metadata.len() > MAX_BACKUP_FILE_BYTES {
        bail!("Ghost FTP backup exceeds the maximum supported size");
    }
    let bytes = std::fs::read(src).with_context(|| format!("read backup {}", src.display()))?;
    let header_len = MAGIC.len() + 1 + SALT_LEN + NONCE_LEN;
    if bytes.len() < header_len + 16 {
        bail!("not a Ghost FTP backup file (too short)");
    }
    if &bytes[..MAGIC.len()] != MAGIC {
        bail!("not a Ghost FTP backup file (bad magic)");
    }
    let version = bytes[MAGIC.len()];
    if version != VERSION {
        bail!("unsupported backup version {version}");
    }
    let salt = &bytes[MAGIC.len() + 1..MAGIC.len() + 1 + SALT_LEN];
    let nonce = &bytes[MAGIC.len() + 1 + SALT_LEN..header_len];
    let header = &bytes[..header_len];
    let ciphertext = &bytes[header_len..];

    let mut key = derive_key(password, salt)?;
    let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(&key));
    let decrypted = cipher.decrypt(
        Nonce::from_slice(nonce),
        Payload {
            msg: ciphertext,
            aad: header,
        },
    );
    drop(cipher);
    key.fill(0);
    let mut gz = decrypted.map_err(|_| anyhow!("wrong password, or the backup is corrupt"))?;
    let mut plaintext = gunzip_bounded(&gz).context("gunzip archive")?;
    gz.fill(0);
    let parsed = serde_json::from_slice::<Archive>(&plaintext).context("parse archive");
    plaintext.fill(0);
    let archive = parsed?;
    validate_archive(&archive)?;
    Ok(archive)
}

fn gzip(data: &[u8]) -> Result<Vec<u8>> {
    let mut enc = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    enc.write_all(data)?;
    Ok(enc.finish()?)
}

fn gunzip_bounded(data: &[u8]) -> Result<Vec<u8>> {
    let dec = flate2::read::GzDecoder::new(data);
    let mut limited = dec.take(MAX_DECOMPRESSED_BYTES as u64 + 1);
    let mut out = Vec::new();
    limited.read_to_end(&mut out)?;
    if out.len() > MAX_DECOMPRESSED_BYTES {
        out.fill(0);
        bail!("Ghost FTP backup expands beyond the maximum supported size");
    }
    Ok(out)
}

// ---- Credential enumeration ----

/// Pull every keychain secret a restore needs: the app's own service keys (from
/// the `ghostftp.db` manifest) plus each profile's OAuth tokens across the known
/// cloud providers. Best-effort — a missing entry is simply skipped.
fn collect_credentials(dir: &Path, db: &Db, vault: &mut impl RestoreVault) -> Result<Vec<Cred>> {
    let mut seen: HashSet<(String, String)> = HashSet::new();
    let mut out: Vec<Cred> = Vec::new();

    let mut take = |service: &str, account: &str, required: bool| -> Result<()> {
        if !credential_target_allowed(service, account) {
            bail!("backup credential inventory contains an invalid target");
        }
        if !seen.insert((service.to_string(), account.to_string())) {
            return Ok(());
        }
        match vault.get(service, account)? {
            Some(secret) => out.push(Cred {
                service: service.to_string(),
                account: account.to_string(),
                secret,
            }),
            None if required => {
                bail!("OS vault is missing a credential recorded in the backup manifest")
            }
            None => {}
        }
        Ok(())
    };

    // A damaged/unreadable manifest is not proof that no credentials exist.
    for (service, account) in db.list_keychain().context("read backup credential manifest")? {
        take(&service, &account, true)?;
    }

    // OAuth tokens are optional for profiles without that provider. Absence is
    // normal; a vault read failure is not and must abort the entire backup.
    match std::fs::read(dir.join("profiles.json")) {
        Ok(bytes) => {
            let profiles: Vec<crate::profiles::ConnectionProfile> =
                serde_json::from_slice(&bytes).context("read valid backup profile metadata")?;
            for profile in profiles {
                let service = match profile.protocol.as_str() {
                    "dropbox" => Some(crate::session::dropbox::DROPBOX_SERVICE),
                    "onedrive" => Some(crate::session::onedrive::ONEDRIVE_SERVICE),
                    "gdrive" => Some(crate::session::gdrive::GDRIVE_SERVICE),
                    "box" => Some(crate::session::boxdrive::BOX_SERVICE),
                    _ => None,
                };
                if let Some(service) = service {
                    take(service, &profile.id, false)?;
                }
            }
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
        Err(e) => return Err(e).context("read backup profile metadata"),
    }
    Ok(out)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn write(dir: &Path, name: &str, content: &str) {
        std::fs::write(dir.join(name), content).unwrap();
    }

    #[test]
    fn round_trips_export_import() {
        let src_dir = std::env::temp_dir().join(format!("ghostftp_bak_src_{}", std::process::id()));
        let dst_dir = std::env::temp_dir().join(format!("ghostftp_bak_dst_{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&src_dir);
        let _ = std::fs::remove_dir_all(&dst_dir);
        std::fs::create_dir_all(&src_dir).unwrap();
        std::fs::create_dir_all(&dst_dir).unwrap();

        // Seed a source app-data dir with configs + a real ghostftp.db.
        write(
            &src_dir,
            "profiles.json",
            r#"[{"id":"p1","name":"Box A","protocol":"sftp","host":"h","port":22,"username":"u","auth":{"kind":"agent"}}]"#,
        );
        write(&src_dir, "bridge.json", r#"{"enabled":true}"#);
        write(&src_dir, "foldersync.json", r#"{"pairs":[]}"#);
        {
            let db = Db::open(&src_dir.join("ghostftp.db")).unwrap();
            db.settings_set("appTheme", "\"nord\"").unwrap();
            db.upsert_snippet("s1", "List", "ls -la", None).unwrap();
        }

        let backup =
            std::env::temp_dir().join(format!("ghostftp_bak_{}.ghostftpbak", std::process::id()));
        let _ = std::fs::remove_file(&backup);
        let db = Db::open(&src_dir.join("ghostftp.db")).unwrap();
        let summary = export(&src_dir, &db, "correct horse", &backup).unwrap();
        assert_eq!(summary.profiles, 1);
        assert!(!src_dir.join("ghostftp.db.backup.tmp").exists());
        assert!(summary.has_bridge && summary.has_sync);
        assert!(summary.db_bytes > 0);
        drop(db);

        // Wrong password must fail cleanly (GCM tag mismatch).
        assert!(import(&dst_dir, "wrong password", &backup, false).is_err());
        // Nothing should have been written on the failed import.
        assert!(!dst_dir.join("profiles.json").exists());

        // Correct password restores everything into the fresh dir.
        let restored = import(&dst_dir, "correct horse", &backup, false).unwrap();
        assert_eq!(restored.profiles, 1);
        assert_eq!(
            std::fs::read_to_string(dst_dir.join("profiles.json")).unwrap(),
            std::fs::read_to_string(src_dir.join("profiles.json")).unwrap()
        );
        assert_eq!(
            std::fs::read_to_string(dst_dir.join("bridge.json")).unwrap(),
            r#"{"enabled":true}"#
        );
        // The restored DB carries the seeded settings + snippet.
        let rdb = Db::open(&dst_dir.join("ghostftp.db")).unwrap();
        assert_eq!(
            rdb.settings_get("appTheme").unwrap().as_deref(),
            Some("\"nord\"")
        );
        assert_eq!(rdb.list_snippets().unwrap().len(), 1);

        drop(rdb);
        let _ = std::fs::remove_dir_all(&src_dir);
        let _ = std::fs::remove_dir_all(&dst_dir);
        let _ = std::fs::remove_file(&backup);
    }

    #[derive(Default)]
    struct TestVault {
        values: std::collections::HashMap<(String, String), String>,
        fail_read: bool,
        fail_write_at: Option<usize>,
        write_count: usize,
    }

    impl RestoreVault for TestVault {
        fn get(&mut self, service: &str, account: &str) -> Result<Option<String>> {
            if self.fail_read {
                bail!("simulated locked OS vault");
            }
            Ok(self.values.get(&(service.to_string(), account.to_string())).cloned())
        }

        fn put(&mut self, service: &str, account: &str, secret: &str) -> Result<()> {
            self.write_count += 1;
            if self.fail_write_at == Some(self.write_count) {
                bail!("simulated OS vault write fault");
            }
            self.values.insert((service.to_string(), account.to_string()), secret.to_string());
            Ok(())
        }

        fn remove(&mut self, service: &str, account: &str) -> Result<()> {
            self.values.remove(&(service.to_string(), account.to_string()));
            Ok(())
        }
    }

    fn test_credential(account: &str, secret: &str) -> Cred {
        Cred {
            service: crate::credentials::SERVICE.to_string(),
            account: account.to_string(),
            secret: secret.to_string(),
        }
    }

    #[test]
    fn locked_vault_fails_closed_without_mutating_credentials() {
        let mut vault = TestVault {
            fail_read: true,
            ..Default::default()
        };
        let credentials = vec![test_credential("one", "replacement")];
        assert!(restore_credentials_with_vault(&credentials, &mut vault).is_err());
        assert_eq!(vault.write_count, 0);
        assert!(vault.values.is_empty());
    }

    #[test]
    fn partial_vault_restore_rolls_back_existing_and_new_values() {
        let service = crate::credentials::SERVICE.to_string();
        let mut vault = TestVault {
            fail_write_at: Some(2),
            ..Default::default()
        };
        vault.values.insert((service.clone(), "one".to_string()), "original".to_string());
        let credentials = vec![
            test_credential("one", "replacement"),
            test_credential("two", "new-secret"),
        ];
        assert!(restore_credentials_with_vault(&credentials, &mut vault).is_err());
        assert_eq!(
            vault.values.get(&(service.clone(), "one".to_string())).map(String::as_str),
            Some("original")
        );
        assert!(!vault.values.contains_key(&(service, "two".to_string())));
    }

    #[test]
    fn malformed_and_legacy_plaintext_profile_backups_are_rejected() {
        let json = r#"[{"id":"legacy","name":"Legacy","protocol":"sftp","host":"host","port":22,"username":"user","auth":{"kind":"password","password":"must-not-persist"}}]"#;
        let archive = Archive {
            created_ms: 0,
            profiles_json: Some(json.to_string()),
            bridge_json: None,
            foldersync_json: None,
            ghostftp_db_b64: String::new(),
            credentials: Vec::new(),
        };
        assert!(validate_archive(&archive).is_err());
    }

    #[test]
    fn backup_export_aborts_on_missing_manifest_secret_or_locked_vault() {
        let db = Db::open_in_memory().unwrap();
        db.record_keychain(crate::credentials::SERVICE, "account").unwrap();
        let dir = std::env::temp_dir().join(format!("ghostftp-vault-read-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir_all(&dir).unwrap();
        let mut missing = TestVault::default();
        assert!(collect_credentials(&dir, &db, &mut missing).is_err());
        let mut locked = TestVault {
            fail_read: true,
            ..Default::default()
        };
        assert!(collect_credentials(&dir, &db, &mut locked).is_err());
        let _ = std::fs::remove_dir_all(dir);
    }

    #[test]
    fn failed_vault_import_does_not_stage_or_replace_profile_or_db() {
        let src_dir = std::env::temp_dir().join(format!("ghostftp-vault-src-{}", uuid::Uuid::new_v4()));
        let dst_dir = std::env::temp_dir().join(format!("ghostftp-vault-dst-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir_all(&src_dir).unwrap();
        std::fs::create_dir_all(&dst_dir).unwrap();
        write(&src_dir, "profiles.json", "[]");
        write(&dst_dir, "profiles.json", "old-original");
        let db = Db::open(&src_dir.join("ghostftp.db")).unwrap();
        db.record_keychain(crate::credentials::SERVICE, "account").unwrap();

        let backup = src_dir.join("test.ghostftpbak");
        let mut source_vault = TestVault::default();
        source_vault.values.insert(
            (crate::credentials::SERVICE.to_string(), "account".to_string()),
            "from-backup".to_string(),
        );
        export_with_vault(&src_dir, &db, "correct horse", &backup, &mut source_vault).unwrap();

        let mut locked = TestVault {
            fail_read: true,
            ..Default::default()
        };
        assert!(import_with_vault(&dst_dir, "correct horse", &backup, false, &mut locked).is_err());
        assert_eq!(std::fs::read_to_string(dst_dir.join("profiles.json")).unwrap(), "old-original");
        for staged in ["profiles.json.restore", "bridge.json.restore", "foldersync.json.restore", "ghostftp.db.restore"] {
            assert!(!dst_dir.join(staged).exists(), "unexpected staged restore: {staged}");
        }
        drop(db);
        let _ = std::fs::remove_dir_all(src_dir);
        let _ = std::fs::remove_dir_all(dst_dir);
    }

    #[test]
    fn rejects_unknown_credential_service() {
        let archive = Archive {
            created_ms: 0,
            profiles_json: None,
            bridge_json: None,
            foldersync_json: None,
            ghostftp_db_b64: String::new(),
            credentials: vec![Cred {
                service: "com.example.other-app".to_string(),
                account: "user".to_string(),
                secret: "secret".to_string(),
            }],
        };

        assert!(validate_archive(&archive).is_err());
    }

    #[test]
    fn private_file_writer_replaces_existing_content() {
        let path = std::env::temp_dir().join(format!(
            "ghostftp-private-write-{}-{}.tmp",
            std::process::id(),
            crate::db::now_ms()
        ));
        std::fs::write(&path, b"old data that must disappear").unwrap();
        write_private_file(&path, b"new").unwrap();
        assert_eq!(std::fs::read(&path).unwrap(), b"new");
        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn export_password_policy_rejects_weak_or_pathological_inputs() {
        assert!(validate_export_password("short").is_err());
        assert!(validate_export_password("correct horse").is_ok());
        assert!(validate_export_password(&"x".repeat(MAX_BACKUP_PASSWORD_BYTES + 1)).is_err());
        assert!(validate_decrypt_password("x").is_ok());
        assert!(validate_decrypt_password("").is_err());
    }

    #[test]
    fn rejects_non_backup_and_bad_magic() {
        let f = std::env::temp_dir().join(format!("ghostftp_notbak_{}.bin", std::process::id()));
        std::fs::write(
            &f,
            b"this is not a ghostftp backup at all, just some bytes here",
        )
        .unwrap();
        assert!(inspect("pw", &f).is_err());
        let _ = std::fs::remove_file(&f);
    }
}
