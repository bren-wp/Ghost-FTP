//! Service-credential storage in the OS keychain (Plan 12 Phase 1).
//!
//! The rule is strict: **credentials never cross the IPC boundary as values**.
//! The frontend can *write* a secret (one-way `set`) and *ask whether one
//! exists* (`has`), but never *read* it back. Rust fetches the value from the
//! keychain at the moment of use.
//!
//! This mirrors the OAuth-token path in `oauth.rs` — same `keyring` crate, same
//! native backends — but for non-profile service keys. Everything lives under
//! one service name, keyed by *purpose*.
//!
//! Because Windows DPAPI and Linux Keychain can't *enumerate* entries, every
//! write is also recorded in a small `ghostftp.db` manifest (`db::record_keychain`)
//! so the encrypted backup (Phase 4) knows which entries to pull.

use anyhow::{anyhow, Context, Result};

/// The single keychain service that holds Ghost FTP's own service credentials.
/// Reverse-DNS so it's unambiguous in Credential Manager / Keychain Access.
pub const SERVICE: &str = "com.ghostftp.credentials";

/// Write (or overwrite) a secret for `purpose`. An empty value clears it.
pub fn set_secret(purpose: &str, value: &str) -> Result<()> {
    if value.is_empty() {
        delete_secret(purpose);
        return Ok(());
    }
    let entry = keyring::Entry::new(SERVICE, purpose).context("open keychain entry")?;
    entry
        .set_password(value)
        .context("write secret to keychain")?;
    Ok(())
}

/// Read a secret for `purpose`, or `None` if it was never set. Rust-only — this
/// is deliberately never exposed as a Tauri command.
pub fn get_secret(purpose: &str) -> Result<Option<String>> {
    let entry = keyring::Entry::new(SERVICE, purpose).context("open keychain entry")?;
    match entry.get_password() {
        Ok(v) => Ok(Some(v)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(e) => Err(anyhow!("read secret from keychain: {e}")),
    }
}

/// Remove a secret (best-effort — a missing entry is not an error).
pub fn delete_secret(purpose: &str) {
    if let Ok(entry) = keyring::Entry::new(SERVICE, purpose) {
        let _ = entry.delete_credential();
    }
}

/// Whether a secret exists for `purpose`. This is the only credential state the
/// frontend gets to see — enough to render a Set/••••/Clear affordance.
pub fn has_secret(purpose: &str) -> bool {
    matches!(get_secret(purpose), Ok(Some(_)))
}

#[cfg(test)]
mod tests {
    use super::*;

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
