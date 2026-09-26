use anyhow::{Context, Result};
use russh_keys::key::PublicKey;
use russh_keys::PublicKeyBase64;
use std::collections::BTreeSet;
use std::fs::{File, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};

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

pub fn check(host: &str, port: u16, key: &PublicKey) -> Result<HostKeyStatus> {
    let Some(path) = known_hosts_path() else {
        return Ok(HostKeyStatus::Unknown);
    };

    let recorded_keys = russh_keys::known_host_keys_path(host, port, &path)
        .with_context(|| format!("reading host keys from {}", path.display()))?;

    let presented_b64 = key.public_key_base64();
    let mut stored_fp: Option<String> = None;
    for (_, recorded) in recorded_keys {
        if recorded.public_key_base64() == presented_b64 {
            return Ok(HostKeyStatus::Match);
        }
        stored_fp = Some(fingerprint(&recorded));
    }

    Ok(match stored_fp {
        Some(fp) => HostKeyStatus::Mismatch {
            stored_fingerprint: fp,
        },
        None => HostKeyStatus::Unknown,
    })
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

/// Replace every existing entry that matches `host:port` with the newly
/// trusted key. The rewrite uses a sibling temporary file and a rollback
/// backup so Windows and Unix both recover the original file if the final
/// rename fails. Matching line numbers come from russh-keys, so hashed host
/// fields are removed as safely as plaintext entries.
pub fn replace(host: &str, port: u16, key: &PublicKey) -> Result<()> {
    let path = known_hosts_path().context("could not resolve ~/.ssh/known_hosts")?;
    let metadata_before = std::fs::metadata(&path)
        .with_context(|| format!("reading metadata for {}", path.display()))?;
    let contents = std::fs::read_to_string(&path)
        .with_context(|| format!("reading {}", path.display()))?;
    let recorded_keys = russh_keys::known_host_keys_path(host, port, &path)
        .with_context(|| format!("reading host keys from {}", path.display()))?;

    if recorded_keys.is_empty() {
        anyhow::bail!("no existing host key entry to replace for {host}:{port}");
    }

    let matching_lines: BTreeSet<usize> =
        recorded_keys.into_iter().map(|(line, _)| line).collect();

    // Avoid overwriting a file that another OpenSSH-compatible client changed
    // while Ghost FTP was preparing the replacement.
    let metadata_after = std::fs::metadata(&path)
        .with_context(|| format!("rechecking metadata for {}", path.display()))?;
    if metadata_before.len() != metadata_after.len()
        || metadata_before.modified().ok() != metadata_after.modified().ok()
    {
        anyhow::bail!("known_hosts changed while replacing {host}:{port}; retry the connection");
    }

    let host_field = if port == 22 {
        host.to_string()
    } else {
        format!("[{host}]:{port}")
    };
    let replacement = format!("{host_field} {} {}\n", key.name(), key.public_key_base64());
    let rewritten = rewrite_matching_lines(&contents, &matching_lines, &replacement)
        .context("matching known_hosts entry disappeared during replacement")?;

    replace_file_with_rollback(&path, rewritten.as_bytes(), metadata_before.permissions())
}

fn rewrite_matching_lines(
    contents: &str,
    matching_lines: &BTreeSet<usize>,
    replacement: &str,
) -> Option<String> {
    let mut rewritten = String::with_capacity(contents.len() + replacement.len());
    let mut replaced = false;

    for (index, line) in contents.split_inclusive('\n').enumerate() {
        let line_number = index + 1;
        if matching_lines.contains(&line_number) {
            if !replaced {
                rewritten.push_str(replacement);
                replaced = true;
            }
            continue;
        }
        rewritten.push_str(line);
    }

    replaced.then_some(rewritten)
}

fn replace_file_with_rollback(
    path: &Path,
    contents: &[u8],
    permissions: std::fs::Permissions,
) -> Result<()> {
    let parent = path.parent().context("known_hosts path has no parent")?;
    let file_name = path
        .file_name()
        .context("known_hosts path has no file name")?
        .to_string_lossy();

    let temp_path = parent.join(format!(
        ".{file_name}.ghostftp-{}.new",
        uuid::Uuid::new_v4()
    ));
    let backup_path = parent.join(format!(
        ".{file_name}.ghostftp-{}.previous",
        uuid::Uuid::new_v4()
    ));

    let mut options = OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    let mut temp = options
        .open(&temp_path)
        .with_context(|| format!("creating {}", temp_path.display()))?;

    if let Err(error) = (|| -> Result<()> {
        temp.write_all(contents)?;
        temp.sync_all()?;
        std::fs::set_permissions(&temp_path, permissions)?;
        std::fs::rename(path, &backup_path).with_context(|| {
            format!(
                "moving {} to rollback backup {}",
                path.display(),
                backup_path.display()
            )
        })?;

        if let Err(rename_error) = std::fs::rename(&temp_path, path) {
            let restore_error = std::fs::rename(&backup_path, path).err();
            let _ = std::fs::remove_file(&temp_path);
            if let Some(restore_error) = restore_error {
                anyhow::bail!(
                    "installing new known_hosts failed ({rename_error}); restoring the original also failed ({restore_error})"
                );
            }
            return Err(rename_error).context("installing replacement known_hosts");
        }

        if let Err(cleanup_error) = std::fs::remove_file(&backup_path) {
            tracing::warn!(
                ?cleanup_error,
                path = %backup_path.display(),
                "failed to remove known_hosts rollback backup"
            );
        }

        #[cfg(unix)]
        File::open(parent)
            .and_then(|directory| directory.sync_all())
            .with_context(|| format!("syncing {}", parent.display()))?;

        Ok(())
    })() {
        let _ = std::fs::remove_file(&temp_path);
        return Err(error);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rewrite_matching_lines_replaces_once_and_preserves_unrelated_rows() {
        let contents = "alpha ssh-ed25519 old-a\nother ssh-rsa keep\nalpha ssh-rsa old-b\n";
        let matching = BTreeSet::from([1usize, 3usize]);
        let replacement = "alpha ssh-ed25519 new\n";

        let rewritten =
            rewrite_matching_lines(contents, &matching, replacement).expect("matching lines");

        assert_eq!(
            rewritten,
            "alpha ssh-ed25519 new\nother ssh-rsa keep\n"
        );
    }

    #[test]
    fn rewrite_matching_lines_refuses_missing_match() {
        let matching = BTreeSet::from([4usize]);
        assert!(rewrite_matching_lines("one\ntwo\n", &matching, "new\n").is_none());
    }
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
