use anyhow::{anyhow, Context, Result};
use std::sync::Mutex;

static VAULT_LOCK: Mutex<()> = Mutex::new(());

// Test binaries only. The production vault cannot be redirected by an
// environment variable, configuration file or IPC command.
#[cfg(test)]
struct RecoveryTestVault {
    root: std::path::PathBuf,
    checkpoint: Option<String>,
    hits: usize,
    native: bool,
}

#[cfg(test)]
std::thread_local! {
    static RECOVERY_TEST_VAULT: std::cell::RefCell<Option<RecoveryTestVault>> =
        const { std::cell::RefCell::new(None) };
}

#[cfg(test)]
pub(crate) fn configure_recovery_test_vault(root: std::path::PathBuf, checkpoint: Option<String>) {
    std::fs::create_dir_all(&root).unwrap();
    RECOVERY_TEST_VAULT.with(|state| {
        *state.borrow_mut() = Some(RecoveryTestVault {
            root,
            checkpoint,
            hits: 0,
            native: false,
        });
    });
}

#[cfg(test)]
pub(crate) fn configure_native_recovery_test_vault(
    root: std::path::PathBuf,
    checkpoint: Option<String>,
) {
    configure_recovery_test_vault(root, checkpoint);
    RECOVERY_TEST_VAULT.with(|state| {
        state.borrow_mut().as_mut().unwrap().native = true;
    });
}

#[cfg(test)]
pub(crate) fn recovery_test_force_rollback() -> bool {
    RECOVERY_TEST_VAULT.with(|state| {
        state.borrow().as_ref().is_some_and(|vault| {
            vault
                .checkpoint
                .as_deref()
                .is_some_and(|point| point.starts_with("rollback:"))
        })
    })
}

// Record only test-created recovery credentials before native writes. The
// registry survives a killed child and is never used by production code.
#[cfg(test)]
fn register_native_test_credential(service: &str, account: &str) -> Result<()> {
    use sha2::{Digest, Sha256};
    use std::io::Write;

    if !service.starts_with("com.ghostftp.download-recovery") {
        return Ok(());
    }
    RECOVERY_TEST_VAULT.with(|state| {
        let state = state.borrow();
        let Some(vault) = state.as_ref().filter(|vault| vault.native) else {
            return Ok(());
        };
        let registry = vault.root.join("native-accounts");
        std::fs::create_dir_all(&registry)?;
        let bytes = serde_json::to_vec(&(service, account))?;
        let digest = Sha256::digest(&bytes);
        let path = registry.join(format!("{digest:x}"));
        match std::fs::read(&path) {
            Ok(existing) => anyhow::ensure!(existing == bytes, "native test registry collision"),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                let mut file = std::fs::OpenOptions::new()
                    .create_new(true)
                    .write(true)
                    .open(path)?;
                file.write_all(&bytes)?;
                file.sync_all()?;
            }
            Err(error) => return Err(error.into()),
        }
        Ok(())
    })
}

#[cfg(test)]
pub(crate) fn cleanup_native_recovery_test_vault(root: &std::path::Path) -> Result<()> {
    let registry = root.join("native-accounts");
    if !registry.exists() {
        return Ok(());
    }
    for entry in std::fs::read_dir(registry)? {
        let entry = entry?;
        let (service, account): (String, String) =
            serde_json::from_slice(&std::fs::read(entry.path())?)?;
        anyhow::ensure!(
            service.starts_with("com.ghostftp.download-recovery"),
            "invalid native test credential service"
        );
        vault_delete_strict(&service, &account)?;
        std::fs::remove_file(entry.path())?;
    }
    Ok(())
}

#[cfg(test)]
pub(crate) fn clear_recovery_test_vault() {
    RECOVERY_TEST_VAULT.with(|state| *state.borrow_mut() = None);
}

#[cfg(test)]
fn test_recovery_path(service: &str, account: &str) -> Option<std::path::PathBuf> {
    use sha2::{Digest, Sha256};

    if !service.starts_with("com.ghostftp.download-recovery") {
        return None;
    }
    RECOVERY_TEST_VAULT.with(|state| {
        state
            .borrow()
            .as_ref()
            .filter(|vault| !vault.native)
            .map(|vault| {
                let digest = Sha256::digest(
                    serde_json::to_vec(&(service, account)).expect("serialize test vault account"),
                );
                vault.root.join(format!("{digest:x}"))
            })
    })
}

#[cfg(test)]
pub(crate) fn recovery_test_checkpoint(point: &str) {
    RECOVERY_TEST_VAULT.with(|state| {
        let mut state = state.borrow_mut();
        let Some(vault) = state.as_mut() else {
            return;
        };
        let Some(checkpoint) = vault.checkpoint.as_deref() else {
            return;
        };
        let selected = checkpoint.strip_prefix("rollback:").unwrap_or(checkpoint);
        let (name, occurrence) = selected.split_once('@').unwrap_or((selected, "1"));
        if name != point {
            return;
        }
        vault.hits += 1;
        if vault.hits == occurrence.parse::<usize>().unwrap() {
            std::fs::write(vault.root.join("checkpoint"), checkpoint).unwrap();
            // Parent kills this process while the real recovery lock is held.
            loop {
                std::thread::sleep(std::time::Duration::from_secs(1));
            }
        }
    });
}

#[cfg(target_os = "linux")]
mod linux_secret_service;

/// These identifiers have shipped in existing Windows Credential Manager
/// entries and Linux keyutils entries. NEVER change them during migration.
pub const SERVICE: &str = "com.ghostftp.credentials";

fn legacy_get(service: &str, account: &str) -> Result<Option<String>> {
    let entry = keyring::Entry::new(service, account).context("open OS credential entry")?;
    match entry.get_password() {
        Ok(secret) => Ok(Some(secret)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(_) => Err(anyhow!("OS credential vault could not read secret")),
    }
}

fn legacy_delete(service: &str, account: &str) -> Result<()> {
    let entry = keyring::Entry::new(service, account).context("open OS credential entry")?;
    checked_delete_result(entry.delete_credential())
}

pub fn vault_get(service: &str, account: &str) -> Result<Option<String>> {
    let _guard = VAULT_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    #[cfg(target_os = "linux")]
    {
        let primary = linux_secret_service::lookup(service, account);
        if let Ok(Some(secret)) = &primary {
            return Ok(Some(secret.clone()));
        }
        // Existing profiles and OAuth tokens use Linux kernel keyutils.
        // Never silently discard them when Secret Service is not yet ready.
        let legacy = legacy_get(service, account);
        match (primary, legacy) {
            (Ok(None), Ok(Some(old))) => {
                // Idempotent migration, with read-back validation inside store.
                // A locked Secret Service must not make an otherwise usable
                // legacy credential unavailable during gradual deployment.
                let _ = linux_secret_service::store(service, account, &old);
                Ok(Some(old))
            }
            (Err(_), Ok(Some(old))) => Ok(Some(old)),
            (Ok(None), Ok(None)) => Ok(None),
            // Never treat an unavailable vault as an empty vault.
            _ => Err(anyhow!(
                "Linux credential vault unavailable; unlock Secret Service"
            )),
        }
    }
    #[cfg(not(target_os = "linux"))]
    {
        legacy_get(service, account)
    }
}

/// Recovery anchors must never fall back to volatile legacy keyutils.
pub(crate) fn vault_get_strict(service: &str, account: &str) -> Result<Option<String>> {
    #[cfg(test)]
    if let Some(path) = test_recovery_path(service, account) {
        return match std::fs::read_to_string(path) {
            Ok(value) => Ok(Some(value)),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(None),
            Err(error) => Err(error.into()),
        };
    }
    let _guard = VAULT_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    #[cfg(target_os = "linux")]
    {
        linux_secret_service::lookup(service, account)
    }
    #[cfg(not(target_os = "linux"))]
    {
        legacy_get(service, account)
    }
}

/// New Linux writes are persistent-only and must verify their destination.
/// No fallback to kernel keyutils: it loses data on reboot.
pub fn vault_set(service: &str, account: &str, value: &str) -> Result<()> {
    #[cfg(test)]
    register_native_test_credential(service, account)?;
    #[cfg(test)]
    if let Some(path) = test_recovery_path(service, account) {
        use std::io::Write;
        let temporary = path.with_extension("tmp");
        let mut file = std::fs::File::create(&temporary)?;
        file.write_all(value.as_bytes())?;
        file.sync_all()?;
        drop(file);
        std::fs::rename(temporary, path)?;
        return Ok(());
    }
    let _guard = VAULT_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    #[cfg(target_os = "linux")]
    {
        linux_secret_service::store(service, account, value)
    }
    #[cfg(not(target_os = "linux"))]
    {
        let entry = keyring::Entry::new(service, account).context("open OS credential entry")?;
        entry
            .set_password(value)
            .map_err(|_| anyhow!("OS credential store refused secret write"))
    }
}

/// Idempotent persistent-only removal, with readback under one vault lock.
/// Never fall back to a volatile credential when deciding whether it is gone.
pub(crate) fn vault_delete_strict(service: &str, account: &str) -> Result<()> {
    #[cfg(test)]
    if let Some(path) = test_recovery_path(service, account) {
        return match std::fs::remove_file(path) {
            Ok(()) => Ok(()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(error) => Err(error.into()),
        };
    }
    let _guard = VAULT_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    #[cfg(target_os = "linux")]
    {
        if linux_secret_service::lookup(service, account)?.is_some() {
            linux_secret_service::remove(service, account)?;
        }
        anyhow::ensure!(
            linux_secret_service::lookup(service, account)?.is_none(),
            "persistent credential still exists after removal"
        );
    }
    #[cfg(not(target_os = "linux"))]
    {
        legacy_delete(service, account)?;
        anyhow::ensure!(
            legacy_get(service, account)?.is_none(),
            "persistent credential still exists after removal"
        );
    }
    Ok(())
}

pub fn vault_delete(service: &str, account: &str) -> Result<()> {
    let _guard = VAULT_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    #[cfg(target_os = "linux")]
    {
        // Preserve legacy keyutils if Secret Service is locked or refuses the
        // deletion. Once verified, clear the old value as well; a stale old
        // token must not reappear on the next credential lookup.
        linux_secret_service::remove(service, account)?;
        legacy_delete(service, account)
    }
    #[cfg(not(target_os = "linux"))]
    {
        legacy_delete(service, account)
    }
}

/// A service credential is *never* copied into profile JSON or the settings DB.
pub fn set_secret(purpose: &str, value: &str) -> Result<()> {
    if value.is_empty() {
        return vault_delete(SERVICE, purpose);
    }
    vault_set(SERVICE, purpose, value)
}

pub fn get_secret(purpose: &str) -> Result<Option<String>> {
    vault_get(SERVICE, purpose)
}

fn checked_delete_result(outcome: std::result::Result<(), keyring::Error>) -> Result<()> {
    match outcome {
        Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
        Err(_) => Err(anyhow!("OS credential store refused to remove the secret")),
    }
}

pub fn delete_secret_checked(purpose: &str) -> Result<()> {
    vault_delete(SERVICE, purpose)
}

/// Serial order is important: never forget an entry from SQLite unless the
/// vault has conclusively deleted it. Fault-injection tests verify this.
fn delete_then_forget<V, M>(vault_delete: V, forget_manifest: M) -> Result<()>
where
    V: FnOnce() -> Result<()>,
    M: FnOnce() -> Result<()>,
{
    vault_delete()?;
    forget_manifest()
}

pub fn delete_secret_recorded(purpose: &str, db: &crate::db::Db) -> Result<()> {
    delete_then_forget(
        || delete_secret_checked(purpose),
        || db.forget_keychain(SERVICE, purpose),
    )
}

pub fn secret_exists_checked(purpose: &str) -> Result<bool> {
    get_secret(purpose).map(|value| value.is_some())
}

/// Legacy compatibility API; new security-sensitive callers use the checked
/// variant so that unavailable vaults are never reported as successful.
pub fn delete_secret(purpose: &str) {
    let _ = delete_secret_checked(purpose);
}

pub fn has_secret(purpose: &str) -> bool {
    matches!(secret_exists_checked(purpose), Ok(true))
}

/// Non-destructive credential manifest reconciliation, never returning
/// service names, account IDs or secrets through the Tauri IPC response.
#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ManifestRepairSummary {
    pub checked: usize,
    pub verified: usize,
    pub missing_removed: usize,
    pub orphaned_retained: usize,
}

fn profile_owner<'a>(service: &str, account: &'a str) -> Option<&'a str> {
    if service == SERVICE {
        for prefix in [
            "profile-password:",
            "profile-key-passphrase:",
            "grant-key:",
            "shopify:",
            "hubspot:",
            "dynamics:",
        ] {
            if let Some(id) = account.strip_prefix(prefix) {
                return Some(id);
            }
        }
    } else if [
        crate::session::dropbox::DROPBOX_SERVICE,
        crate::session::onedrive::ONEDRIVE_SERVICE,
        crate::session::gdrive::GDRIVE_SERVICE,
        crate::session::boxdrive::BOX_SERVICE,
        crate::session::dynamics::DYNAMICS_TOKEN_SERVICE,
    ]
    .contains(&service)
    {
        return Some(account);
    }
    None
}

/// Two-phase: verify all vault entries first, then prune ONLY definitively
/// missing ones. Locked vaults cannot result in destructive manifest cleanup.
/// Orphans with actual secrets are deliberately retained for manual recovery.
pub fn repair_manifest_with<F>(
    db: &crate::db::Db,
    saved_profiles: &std::collections::HashSet<String>,
    mut read: F,
) -> Result<ManifestRepairSummary>
where
    F: FnMut(&str, &str) -> Result<Option<String>>,
{
    let entries = db.list_keychain()?;
    anyhow::ensure!(
        entries.len() <= 20_000,
        "Credential manifest exceeds audit limit"
    );
    let mut missing = Vec::new();
    let mut verified = 0;
    let mut orphans = 0;
    for (service, account) in &entries {
        match read(service, account)? {
            Some(_) => {
                verified += 1;
                if profile_owner(service, account).is_some_and(|id| !saved_profiles.contains(id)) {
                    orphans += 1;
                }
            }
            None => missing.push((service.clone(), account.clone())),
        }
    }
    for (service, account) in &missing {
        db.forget_keychain(service, account)?;
    }
    Ok(ManifestRepairSummary {
        checked: entries.len(),
        verified,
        missing_removed: missing.len(),
        orphaned_retained: orphans,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn orphan_audit_retains_existing_secrets_and_prunes_only_proven_absent() {
        let db = crate::db::Db::open_in_memory().unwrap();
        db.record_keychain(SERVICE, "profile-password:present")
            .unwrap();
        db.record_keychain(SERVICE, "profile-password:deleted")
            .unwrap();
        db.record_keychain(SERVICE, "profile-password:absent")
            .unwrap();
        let mut profiles = std::collections::HashSet::new();
        profiles.insert("present".to_string());
        let result = repair_manifest_with(&db, &profiles, |_svc, key| match key {
            "profile-password:absent" => Ok(None),
            _ => Ok(Some("PRIVATE_SENTINEL".to_string())),
        })
        .unwrap();
        assert_eq!(result.checked, 3);
        assert_eq!(result.verified, 2);
        assert_eq!(result.missing_removed, 1);
        assert_eq!(result.orphaned_retained, 1);
        assert_eq!(db.list_keychain().unwrap().len(), 2);
    }

    #[test]
    fn locked_vault_does_not_prune_any_manifest_record() {
        let db = crate::db::Db::open_in_memory().unwrap();
        db.record_keychain(SERVICE, "profile-password:missing")
            .unwrap();
        db.record_keychain(SERVICE, "profile-password:locked")
            .unwrap();
        let known = std::collections::HashSet::new();
        let result = repair_manifest_with(&db, &known, |_svc, key| {
            if key.ends_with(":locked") {
                Err(anyhow!("simulated locked store"))
            } else {
                Ok(None)
            }
        });
        assert!(result.is_err());
        assert_eq!(db.list_keychain().unwrap().len(), 2);
    }

    #[test]
    fn injected_vault_failure_never_forgets_sqlite_manifest() {
        use std::cell::Cell;
        let forget_calls = Cell::new(0);
        let bad = delete_then_forget(
            || Err(anyhow!("simulated locked vault: PRIVATE_SENTINEL")),
            || {
                forget_calls.set(forget_calls.get() + 1);
                Ok(())
            },
        );
        assert!(bad.is_err());
        assert_eq!(forget_calls.get(), 0);
        let good = delete_then_forget(
            || Ok(()),
            || {
                forget_calls.set(forget_calls.get() + 1);
                Ok(())
            },
        );
        assert!(good.is_ok());
        assert_eq!(forget_calls.get(), 1);
        let db_error = delete_then_forget(|| Ok(()), || Err(anyhow!("SQLite write failed")));
        assert!(db_error.is_err());
    }

    #[test]
    fn clearing_missing_entry_is_idempotent_but_locked_vault_fails_closed() {
        assert!(checked_delete_result(Ok(())).is_ok());
        assert!(checked_delete_result(Err(keyring::Error::NoEntry)).is_ok());
        let locked = keyring::Error::Invalid(
            "delete".to_string(),
            "simulated unavailable keyring".to_string(),
        );
        let error = checked_delete_result(Err(locked)).unwrap_err();
        assert_eq!(
            error.to_string(),
            "OS credential store refused to remove the secret"
        );
        assert!(!error.to_string().contains("simulated unavailable"));
    }

    /// Round-trips a secret through the real OS keychain. `#[ignore]`d because it
    /// touches the credential store (unavailable on headless CI); run locally
    /// with `cargo test -p ghostftp -- --ignored credentials`.
    #[test]
    #[ignore = "requires an unlocked real Windows Credential Manager or Linux Secret Service"]
    fn secret_roundtrip() {
        let purpose = format!("ghostftp-live-test-{}", uuid::Uuid::new_v4());
        assert!(!secret_exists_checked(&purpose).unwrap());
        set_secret(&purpose, "PRIVATE_SENTINEL_WIPE_ON_DELETE").unwrap();
        assert!(secret_exists_checked(&purpose).unwrap());
        assert_eq!(
            get_secret(&purpose).unwrap().as_deref(),
            Some("PRIVATE_SENTINEL_WIPE_ON_DELETE")
        );
        // Confirm read-after-write before deletion (actual storage provider).
        delete_secret_checked(&purpose).unwrap();
        assert!(!secret_exists_checked(&purpose).unwrap());
    }

    #[test]
    #[ignore = "requires an unlocked real OS vault and tests same-process concurrency"]
    fn live_vault_concurrent_writes_are_serialized() {
        let purpose = format!("ghostftp-concurrency-{}", uuid::Uuid::new_v4());
        let mut workers = Vec::new();
        for index in 0..6 {
            let key = purpose.clone();
            workers.push(std::thread::spawn(move || {
                set_secret(&key, &format!("UNIQUE_SENTINEL_{index}")).unwrap();
                assert!(get_secret(&key).unwrap().is_some());
            }));
        }
        for worker in workers {
            worker.join().unwrap();
        }
        let final_value = get_secret(&purpose).unwrap().unwrap();
        assert!(final_value.starts_with("UNIQUE_SENTINEL_"));
        delete_secret_checked(&purpose).unwrap();
        assert!(get_secret(&purpose).unwrap().is_none());
    }
}
