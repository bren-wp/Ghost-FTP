use anyhow::{Context, Result};
use russh_keys::key::PublicKey;
use russh_keys::PublicKeyBase64;
use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;

/// Path to `~/.ssh/known_hosts`. We use OpenSSH's standard location even on
/// Windows so users get the same file OpenSSH-compatible clients already use.
/// Matching is delegated to russh-keys so both plaintext and OpenSSH hashed
/// (`|1|salt|hash`) host fields are honored.
pub fn known_hosts_path() -> Option<PathBuf> {
    let home = dirs::home_dir()?;
    Some(home.join(".ssh").join("known_hosts"))
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum HostKeyStatus {
    /// File doesn't exist, or no entries match the host:port — caller must prompt.
    Unknown,
    /// Host is recorded with this exact key.
    Match,
    /// Host is recorded but with a different key — caller should refuse.
    Mismatch { stored_fingerprint: String },
}

pub fn check(host: &str, port: u16, key: &PublicKey) -> HostKeyStatus {
    let Some(path) = known_hosts_path() else {
        return HostKeyStatus::Unknown;
    };

    let Ok(recorded_keys) = russh_keys::known_hosts::known_host_keys_path(host, port, &path) else {
        return HostKeyStatus::Unknown;
    };

    let presented_b64 = key.public_key_base64();
    let mut stored_fp: Option<String> = None;
    for (_, recorded) in recorded_keys {
        if recorded.public_key_base64() == presented_b64 {
            return HostKeyStatus::Match;
        }
        stored_fp = Some(fingerprint(&recorded));
    }

    match stored_fp {
        Some(fp) => HostKeyStatus::Mismatch {
            stored_fingerprint: fp,
        },
        None => HostKeyStatus::Unknown,
    }
}

/// Append a host entry. Writes `host[:port] <type> <base64>` in OpenSSH's
/// non-standard-port form (`[host]:port`). Creates `~/.ssh` with `0700` on
/// unix if needed.
pub fn append(host: &str, port: u16, key: &PublicKey) -> Result<()> {
    let path = known_hosts_path().context("could not resolve ~/.ssh/known_hosts")?;
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).ok();
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700));
        }
    }
    let host_field = if port == 22 {
        host.to_string()
    } else {
        format!("[{host}]:{port}")
    };
    let line = format!("{host_field} {} {}\n", key.name(), key.public_key_base64());

    let mut options = OpenOptions::new();
    options.create(true).append(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    let mut file = options
        .open(&path)
        .with_context(|| format!("opening {} for append", path.display()))?;
    file.write_all(line.as_bytes())?;
    Ok(())
}

/// Compute the SHA-256 fingerprint of a public key in OpenSSH form:
/// `SHA256:<base64-no-padding>`.
pub fn fingerprint(key: &PublicKey) -> String {
    use sha2::{Digest, Sha256};
    let blob = key.public_key_bytes();
    let mut hasher = Sha256::new();
    hasher.update(&blob);
    let digest = hasher.finalize();
    let b64 = base64_no_pad(&digest);
    format!("SHA256:{b64}")
}

fn base64_no_pad(bytes: &[u8]) -> String {
    use base64::Engine;
    base64::engine::general_purpose::STANDARD_NO_PAD.encode(bytes)
}
