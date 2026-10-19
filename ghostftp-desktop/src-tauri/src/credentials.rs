use anyhow::{anyhow, Context, Result};
use std::sync::Mutex;

static VAULT_LOCK: Mutex<()> = Mutex::new(());


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
    let _guard = VAULT_LOCK.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
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
            _ => Err(anyhow!("Linux credential vault unavailable; unlock Secret Service")),
        }
    }
    #[cfg(not(target_os = "linux"))]
    {
        legacy_get(service, account)
    }
}

/// New Linux writes are persistent-only and must verify their destination.
/// No fallback to kernel keyutils: it loses data on reboot.
pub fn vault_set(service: &str, account: &str, value: &str) -> Result<()> {
    let _guard = VAULT_LOCK.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
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

pub fn vault_delete(service: &str, account: &str) -> Result<()> {
    let _guard = VAULT_LOCK.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
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
    ].contains(&service) {
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
    anyhow::ensure!(entries.len() <= 20_000, "Credential manifest exceeds audit limit");
    let mut missing = Vec::new();
    let mut verified = 0;
    let mut orphans = 0;
    for (service, account) in &entries {
        match read(service, account)? {
            Some(_) => {
                verified += 1;
                if profile_owner(service, account)
                    .is_some_and(|id| !saved_profiles.contains(id))
                {
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
        db.record_keychain(SERVICE, "profile-password:present").unwrap();
        db.record_keychain(SERVICE, "profile-password:deleted").unwrap();
        db.record_keychain(SERVICE, "profile-password:absent").unwrap();
        let mut profiles = std::collections::HashSet::new();
        profiles.insert("present".to_string());
        let result = repair_manifest_with(&db, &profiles, |_svc, key| match key {
            "profile-password:absent" => Ok(None),
            _ => Ok(Some("PRIVATE_SENTINEL".to_string())),
        }).unwrap();
        assert_eq!(result.checked, 3);
        assert_eq!(result.verified, 2);
        assert_eq!(result.missing_removed, 1);
        assert_eq!(result.orphaned_retained, 1);
        assert_eq!(db.list_keychain().unwrap().len(), 2);
    }

    #[test]
    fn locked_vault_does_not_prune_any_manifest_record() {
        let db = crate::db::Db::open_in_memory().unwrap();
        db.record_keychain(SERVICE, "profile-password:missing").unwrap();
        db.record_keychain(SERVICE, "profile-password:locked").unwrap();
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
            || { forget_calls.set(forget_calls.get() + 1); Ok(()) },
        );
        assert!(bad.is_err());
        assert_eq!(forget_calls.get(), 0);
        let good = delete_then_forget(
            || Ok(()),
            || { forget_calls.set(forget_calls.get() + 1); Ok(()) },
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
    #[ignore = "touches the OS keychain"]
    fn secret_roundtrip() {
        let purpose = "ghostftp-credentials-test-key";
        assert!(!has_secret(purpose));
        set_secret(purpose, "sk-test-123").unwrap();
        assert!(has_secret(purpose));
        assert_eq!(get_secret(purpose).unwrap().as_deref(), Some("sk-test-123"));
        // Empty value clears it.
        set_secret(purpose, "").unwrap();
        assert!(!has_secret(purpose));
        assert!(get_secret(purpose).unwrap().is_none());
    }
}
