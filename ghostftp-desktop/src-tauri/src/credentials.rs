use anyhow::{anyhow, Context, Result};

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

#[cfg(test)]
mod tests {
    use super::*;

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
