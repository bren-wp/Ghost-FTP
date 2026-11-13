//! Private per-profile credential vault and durable metadata-export helpers.
//! Never expose hydrated secrets across the Tauri command boundary.
use super::err;
use crate::error::GhostFTPError;
use crate::profiles::{AuthMethod, ConnectionProfile};
use crate::AppState;
use std::path::Path;
use uuid::Uuid;

pub(super) fn profile_password_key(id: &str) -> String {
    format!("profile-password:{id}")
}

pub(super) fn profile_key_passphrase_key(id: &str) -> String {
    format!("profile-key-passphrase:{id}")
}

/// Move profile secrets into the OS credential store before the profile is
/// written to profiles.json. The JSON retains only non-secret connection
/// metadata; empty password/passphrase fields are hydrated inside Rust at
/// connect time and are never returned to the frontend as saved values.
pub(super) fn protect_profile_secrets(
    profile: &mut ConnectionProfile,
    state: &AppState,
) -> Result<bool, String> {
    let mut changed = false;
    let profile_id = profile.id.clone();
    match &mut profile.auth {
        AuthMethod::Password { password } if !password.is_empty() => {
            let key = profile_password_key(&profile_id);
            // Record before writing. If the OS vault writes then reports an
            // error, the manifest must still know about that possible secret.
            state
                .db
                .record_keychain(crate::credentials::SERVICE, &key)
                .map_err(err)?;
            crate::credentials::set_secret(&key, password)
                .map_err(|_| "OS credential store rejected the profile password".to_string())?;
            password.clear();
            changed = true;
        }
        AuthMethod::Key { passphrase, .. } => {
            if let Some(value) = passphrase.as_ref().filter(|value| !value.is_empty()) {
                let key = profile_key_passphrase_key(&profile_id);
                state
                    .db
                    .record_keychain(crate::credentials::SERVICE, &key)
                    .map_err(err)?;
                crate::credentials::set_secret(&key, value)
                    .map_err(|_| "OS credential store rejected the key passphrase".to_string())?;
                *passphrase = None;
                changed = true;
            }
        }
        _ => {}
    }
    Ok(changed)
}

pub(super) fn hydrate_profile_secrets(profile: &mut ConnectionProfile) -> Result<(), GhostFTPError> {
    let profile_id = profile.id.clone();
    match &mut profile.auth {
        AuthMethod::Password { password } if password.is_empty() => {
            if let Some(value) = crate::credentials::get_secret(&profile_password_key(&profile_id))
                .map_err(GhostFTPError::from)?
            {
                *password = value;
            }
        }
        AuthMethod::Key { passphrase, .. } if passphrase.is_none() => {
            if let Some(value) =
                crate::credentials::get_secret(&profile_key_passphrase_key(&profile_id))
                    .map_err(GhostFTPError::from)?
            {
                *passphrase = Some(value);
            }
        }
        _ => {}
    }
    Ok(())
}

pub(super) fn delete_profile_secret(key: &str, state: &AppState) -> Result<(), String> {
    crate::credentials::delete_secret_recorded(key, &state.db)
        .map_err(|_| "Profile secret cleanup failed; check the OS vault and manifest".to_string())
}

/// Export server/user metadata through an exclusive, private staging file.
///
/// Never stream directly into an existing destination: disk-full errors could
/// otherwise truncate the user's last export. Reject symlink targets and keep
/// a rollback copy until the new file has been promoted on Windows or Unix.
pub(super) fn write_private_profile_export(destination: &Path, bytes: &[u8]) -> Result<(), String> {
    use std::io::Write;
    #[cfg(unix)]
    use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};

    let parent = destination
        .parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| Path::new("."));
    std::fs::create_dir_all(parent).map_err(err)?;
    let filename = destination
        .file_name()
        .ok_or_else(|| "export destination must identify a file".to_string())?
        .to_string_lossy();
    let temporary = parent.join(format!(".{filename}.ghostftp-{}.tmp", Uuid::new_v4()));
    let rollback = parent.join(format!(".{filename}.ghostftp-{}.previous", Uuid::new_v4()));

    let result = (|| -> Result<(), String> {
        let mut options = std::fs::OpenOptions::new();
        options.create_new(true).write(true);
        #[cfg(unix)]
        options.mode(0o600);
        let mut file = options.open(&temporary).map_err(err)?;
        file.write_all(bytes).map_err(err)?;
        file.sync_all().map_err(err)?;
        #[cfg(unix)]
        std::fs::set_permissions(&temporary, std::fs::Permissions::from_mode(0o600))
            .map_err(err)?;
        drop(file);

        let existed = match std::fs::symlink_metadata(destination) {
            Ok(metadata) if metadata.file_type().is_symlink() || !metadata.is_file() => {
                return Err("Profile exports cannot replace links or non-files.".to_string());
            }
            Ok(_) => true,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => false,
            Err(error) => return Err(err(error)),
        };
        if existed {
            std::fs::rename(destination, &rollback).map_err(err)?;
        }
        if let Err(install_error) = std::fs::rename(&temporary, destination) {
            if existed {
                if let Err(recovery_error) = std::fs::rename(&rollback, destination) {
                    return Err(format!(
                        "Profile export failed ({install_error}); original file remains at {}: {recovery_error}",
                        rollback.display()
                    ));
                }
            }
            return Err(err(install_error));
        }
        if existed {
            let _ = std::fs::remove_file(&rollback);
        }
        Ok(())
    })();
    if result.is_err() {
        let _ = std::fs::remove_file(&temporary);
    }
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn credential_vault_keys_keep_legacy_identity() {
        assert_eq!(profile_password_key("site-42"), "profile-password:site-42");
        assert_eq!(
            profile_key_passphrase_key("site-42"),
            "profile-key-passphrase:site-42"
        );
    }
}
