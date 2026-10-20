use anyhow::{bail, Context, Result};
use russh::keys::{PublicKey, PublicKeyBase64};
use std::collections::BTreeSet;
#[cfg(unix)]
use std::fs::File;
use std::fs::OpenOptions;
use std::io::{Read, Write};
use std::path::{Path, PathBuf};

/// Path to `~/.ssh/known_hosts`. We use OpenSSH's standard location even on
/// Windows so users get the same file OpenSSH-compatible clients already use.
/// Matching is delegated to russh so both plaintext and OpenSSH hashed
/// (`|1|salt|hash`) host fields are honored.
pub fn known_hosts_path() -> Option<PathBuf> {
    let home = dirs::home_dir()?;
    Some(home.join(".ssh").join("known_hosts"))
}

/// Reject hostnames that could inject or override known_hosts entries.
/// OpenSSH host patterns use commas, wildcards and negations; a connection
/// target must be one literal host, not a pattern or another line.
fn validate_host_key_target(host: &str, port: u16) -> Result<()> {
    if port == 0
        || host.is_empty()
        || host.trim() != host
        || host.chars().any(|character| {
            character.is_control()
                || character.is_whitespace()
                || matches!(character, ',' | '*' | '?' | '!' | '|' | '#')
        })
    {
        bail!("invalid SSH host/port for known_hosts");
    }
    Ok(())
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
    validate_host_key_target(host, port)?;
    let Some(path) = known_hosts_path() else {
        return Ok(HostKeyStatus::Unknown);
    };

    let recorded_keys = russh::keys::known_hosts::known_host_keys_path(host, port, &path)
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

/// A link at known_hosts could redirect a trust decision into an unrelated
/// user file. On Unix, O_NOFOLLOW closes the final-component check/open race.
fn open_known_hosts_for_append(path: &Path) -> Result<std::fs::File> {
    match std::fs::symlink_metadata(path) {
        Ok(meta) => anyhow::ensure!(
            meta.file_type().is_file(),
            "known_hosts is not a regular file"
        ),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        Err(error) => return Err(error).context("inspect known_hosts before append"),
    }
    let mut options = OpenOptions::new();
    options.create(true).append(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC);
    }
    let file = options
        .open(path)
        .context("open known_hosts without following links")?;
    anyhow::ensure!(
        file.metadata()
            .context("inspect opened known_hosts")?
            .is_file(),
        "known_hosts is not a regular file"
    );
    Ok(file)
}

/// Append a host entry. Writes `host[:port] <type> <base64>` in OpenSSH's
/// non-standard-port form (`[host]:port`). Creates `~/.ssh` with `0700` on
/// unix if needed.
pub fn append(host: &str, port: u16, key: &PublicKey) -> Result<()> {
    validate_host_key_target(host, port)?;
    let path = known_hosts_path().context("could not resolve ~/.ssh/known_hosts")?;
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("creating known_hosts parent {}", parent.display()))?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700))
                .with_context(|| format!("securing known_hosts parent {}", parent.display()))?;
        }
    }
    let host_field = if port == 22 {
        host.to_string()
    } else {
        format!("[{host}]:{port}")
    };
    let line = format!(
        "{host_field} {} {}\n",
        key.algorithm().as_str(),
        key.public_key_base64()
    );

    let mut file = open_known_hosts_for_append(&path)?;
    file.write_all(line.as_bytes())
        .context("append trusted SSH host key")?;
    file.sync_all().context("sync SSH known_hosts update")?;
    Ok(())
}

// Refuse symlink and Unix hardlink aliases before mutating the host trust store.
// A metadata snapshot is also used to detect file replacement between reads.
fn inspect_known_hosts_for_replace(path: &Path) -> Result<std::fs::Metadata> {
    let metadata = std::fs::symlink_metadata(path)
        .with_context(|| format!("inspect SSH known_hosts {}", path.display()))?;
    anyhow::ensure!(
        metadata.file_type().is_file(),
        "SSH known_hosts replacement requires a regular non-symlink file"
    );
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        anyhow::ensure!(
            metadata.nlink() == 1,
            "SSH known_hosts replacement refuses multiply linked files"
        );
    }
    Ok(metadata)
}

fn ensure_unchanged_known_hosts(
    expected: &std::fs::Metadata,
    actual: &std::fs::Metadata,
) -> Result<()> {
    anyhow::ensure!(
        expected.len() == actual.len() && expected.modified().ok() == actual.modified().ok(),
        "SSH known_hosts changed during replacement; retry the connection"
    );
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        anyhow::ensure!(
            expected.dev() == actual.dev() && expected.ino() == actual.ino(),
            "SSH known_hosts identity changed during replacement; retry the connection"
        );
    }
    Ok(())
}

/// Replace every existing entry that matches `host:port` with the newly
/// trusted key. The rewrite uses a sibling temporary file and a rollback
/// backup so Windows and Unix both recover the original file if the final
/// rename fails. Matching line numbers come from russh, so hashed host
/// fields are removed as safely as plaintext entries.
pub fn replace(host: &str, port: u16, key: &PublicKey) -> Result<()> {
    validate_host_key_target(host, port)?;
    let path = known_hosts_path().context("could not resolve ~/.ssh/known_hosts")?;
    let metadata_preflight = inspect_known_hosts_for_replace(&path)?;
    let mut options = OpenOptions::new();
    options.read(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC);
    }
    let mut source = options
        .open(&path)
        .with_context(|| format!("opening SSH known_hosts {}", path.display()))?;
    let metadata_before = source
        .metadata()
        .context("inspect opened SSH known_hosts")?;
    anyhow::ensure!(
        metadata_before.file_type().is_file(),
        "SSH known_hosts source is not a regular file"
    );
    ensure_unchanged_known_hosts(&metadata_preflight, &metadata_before)?;
    let mut contents = String::new();
    source
        .read_to_string(&mut contents)
        .context("reading SSH known_hosts safely")?;
    // Close the input handle before any Windows rename of this trust file.
    drop(source);
    let recorded_keys = russh::keys::known_hosts::known_host_keys_path(host, port, &path)
        .with_context(|| format!("reading host keys from {}", path.display()))?;

    if recorded_keys.is_empty() {
        anyhow::bail!("no existing host key entry to replace for {host}:{port}");
    }

    let matching_lines: BTreeSet<usize> = recorded_keys.into_iter().map(|(line, _)| line).collect();

    // Avoid overwriting a file that another OpenSSH-compatible client changed
    // while Ghost FTP was preparing the replacement.
    ensure_unchanged_known_hosts(&metadata_before, &inspect_known_hosts_for_replace(&path)?)?;

    let host_field = if port == 22 {
        host.to_string()
    } else {
        format!("[{host}]:{port}")
    };
    let replacement = format!(
        "{host_field} {} {}\n",
        key.algorithm().as_str(),
        key.public_key_base64()
    );
    let rewritten = rewrite_matching_lines(&contents, &matching_lines, &replacement)
        .context("matching known_hosts entry disappeared during replacement")?;

    replace_file_with_rollback(&path, rewritten.as_bytes(), &metadata_before)
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
    expected: &std::fs::Metadata,
) -> Result<()> {
    ensure_unchanged_known_hosts(expected, &inspect_known_hosts_for_replace(path)?)?;
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
        temp.set_permissions(expected.permissions())?;
        // Narrow the path-swap window before moving the original trust file.
        // Parent directory permissions still define our OS-level trust boundary.
        ensure_unchanged_known_hosts(expected, &inspect_known_hosts_for_replace(path)?)?;
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reject_newlines_controls_patterns_and_ambiguous_host_fields() {
        for host in [
            "",
            " server.test",
            "server.test ",
            "server\nattacker.test",
            "server\rattacker.test",
            "server\tattacker.test",
            "server.test,attacker.test",
            "*.test",
            "!trusted.test",
            "|1|injected|entry",
            "#comment",
        ] {
            assert!(
                validate_host_key_target(host, 22).is_err(),
                "accepted {host:?}"
            );
        }
        assert!(validate_host_key_target("example.test", 0).is_err());
    }

    #[test]
    fn permit_literal_ipv4_ipv6_dns_and_nondefault_ports() {
        for host in [
            "example.test",
            "127.0.0.1",
            "::1",
            "2001:db8::1234",
            "xn--bcher-kva.example",
        ] {
            assert!(validate_host_key_target(host, 22).is_ok());
            assert!(validate_host_key_target(host, 2222).is_ok());
        }
    }

    #[test]
    fn rewrite_matching_lines_replaces_once_and_preserves_unrelated_rows() {
        let contents = "alpha ssh-ed25519 old-a\nother ssh-rsa keep\nalpha ssh-rsa old-b\n";
        let matching = BTreeSet::from([1usize, 3usize]);
        let replacement = "alpha ssh-ed25519 new\n";

        let rewritten =
            rewrite_matching_lines(contents, &matching, replacement).expect("matching lines");

        assert_eq!(rewritten, "alpha ssh-ed25519 new\nother ssh-rsa keep\n");
    }

    #[cfg(unix)]
    #[test]
    fn known_hosts_append_rejects_symlink_without_modifying_target() {
        use std::os::unix::fs::symlink;
        let root = std::env::temp_dir().join(format!(
            "ghostftp-known-hosts-test-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let target = root.join("unrelated-secret");
        std::fs::write(&target, b"DO NOT TOUCH").unwrap();
        let link = root.join("known_hosts");
        symlink(&target, &link).unwrap();
        assert!(open_known_hosts_for_append(&link).is_err());
        assert_eq!(std::fs::read(&target).unwrap(), b"DO NOT TOUCH");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(unix)]
    #[test]
    fn replacement_refuses_symlink_without_modifying_target() {
        use std::os::unix::fs::symlink;
        let root = std::env::temp_dir().join(format!(
            "ghostftp-known-hosts-replace-link-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let original = root.join("original");
        let alias = root.join("known_hosts");
        std::fs::write(&original, b"TRUSTED ORIGINAL").unwrap();
        symlink(&original, &alias).unwrap();
        let expected = std::fs::metadata(&original).unwrap();
        assert!(inspect_known_hosts_for_replace(&alias).is_err());
        assert!(replace_file_with_rollback(&alias, b"BAD", &expected).is_err());
        assert_eq!(std::fs::read(&original).unwrap(), b"TRUSTED ORIGINAL");
        assert!(std::fs::symlink_metadata(&alias)
            .unwrap()
            .file_type()
            .is_symlink());
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(unix)]
    #[test]
    fn replacement_refuses_hardlinks_without_modifying_target() {
        let root = std::env::temp_dir().join(format!(
            "ghostftp-known-hosts-replace-hardlink-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let original = root.join("known_hosts");
        let alias = root.join("linked");
        std::fs::write(&original, b"ORIGINAL").unwrap();
        std::fs::hard_link(&original, &alias).unwrap();
        let expected = std::fs::metadata(&original).unwrap();
        assert!(inspect_known_hosts_for_replace(&original).is_err());
        assert!(replace_file_with_rollback(&original, b"BAD", &expected).is_err());
        assert_eq!(std::fs::read(&alias).unwrap(), b"ORIGINAL");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn replacement_refuses_changed_file_without_losing_trust_records() {
        let root = std::env::temp_dir().join(format!(
            "ghostftp-known-hosts-replace-change-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let path = root.join("known_hosts");
        std::fs::write(&path, b"FIRST").unwrap();
        let expected = inspect_known_hosts_for_replace(&path).unwrap();
        std::fs::write(&path, b"SECOND AND LONGER").unwrap();
        assert!(replace_file_with_rollback(&path, b"BAD", &expected).is_err());
        assert_eq!(std::fs::read(&path).unwrap(), b"SECOND AND LONGER");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn replacement_keeps_normal_file_working() {
        let root = std::env::temp_dir().join(format!(
            "ghostftp-known-hosts-replace-valid-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let path = root.join("known_hosts");
        std::fs::write(&path, b"ORIGINAL").unwrap();
        let expected = inspect_known_hosts_for_replace(&path).unwrap();
        replace_file_with_rollback(&path, b"VALID REPLACEMENT", &expected).unwrap();
        assert_eq!(std::fs::read(&path).unwrap(), b"VALID REPLACEMENT");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn rewrite_matching_lines_refuses_missing_match() {
        let matching = BTreeSet::from([4usize]);
        assert!(rewrite_matching_lines("one\ntwo\n", &matching, "new\n").is_none());
    }
}
