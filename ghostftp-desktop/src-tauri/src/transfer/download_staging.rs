//! Durable, exclusive download staging and lossless local promotion.
//! Includes guards common to FTP, FTPS, SFTP, HTTP, cloud and Agent delta.
use anyhow::{Context, Result};
use std::path::{Path, PathBuf};
use uuid::Uuid;

pub(super) fn download_staging_path(final_path: &Path, id: &str) -> PathBuf {
    let parent = final_path.parent().unwrap_or(Path::new("."));
    parent.join(format!(".ghostftp-download-{id}.part"))
}

/// Whole-file backends cannot verify a resumed prefix. Allocate a fresh,
/// unpredictable sibling file per attempt so retries and canceled transfers
/// never reuse stale or partially written bytes.
pub(super) fn fresh_download_staging_path(final_path: &Path, id: &str) -> PathBuf {
    download_staging_path(final_path, &format!("{id}-{}", Uuid::new_v4().simple()))
}

/// An exclusive-open failure does *not* confer ownership of a staging
/// pathname. Preserve the cause through protocol/reconnect error contexts so
/// the runner never deletes a pre-existing user file/symlink on failure.
#[derive(Debug)]
pub(super) struct DownloadStagingOpenFailure(pub(super) std::io::Error);

impl std::fmt::Display for DownloadStagingOpenFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "exclusive download staging open failed: {}", self.0)
    }
}

impl std::error::Error for DownloadStagingOpenFailure {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        Some(&self.0)
    }
}

/// Exclusively create the staging file at the point of the actual write.
/// File::create follows a pre-planted symlink and truncates an existing file
/// before promotion can check its type. O_EXCL/CREATE_NEW rejects both.
pub(super) async fn create_new_download_staging(path: &Path) -> Result<tokio::fs::File> {
    tokio::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .await
        .map_err(DownloadStagingOpenFailure)
        .with_context(|| format!("exclusively create download staging {}", path.display()))
}

pub(super) fn download_backup_path(final_path: &Path, id: &str) -> PathBuf {
    let parent = final_path.parent().unwrap_or(Path::new("."));
    parent.join(format!(".ghostftp-download-{id}.backup"))
}

/// Never follow a final-component symbolic link when deciding whether a
/// download target is occupied. Path::exists and fs::metadata both report
/// a dangling symlink as absent, which can silently turn Skip/Rename into
/// replacement of an unrelated filesystem entry.
pub(super) async fn local_path_exists_fail_closed(path: &Path) -> Result<bool> {
    match tokio::fs::symlink_metadata(path).await {
        Ok(_) => Ok(true),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(false),
        Err(error) => {
            Err(error).with_context(|| format!("lstat local download target {}", path.display()))
        }
    }
}

pub(super) fn local_path_exists_sync_fail_closed(path: &Path) -> Result<bool> {
    match std::fs::symlink_metadata(path) {
        Ok(_) => Ok(true),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(false),
        Err(error) => {
            Err(error).with_context(|| format!("lstat local rename target {}", path.display()))
        }
    }
}

/// Use the same atomic no-replace Windows/Linux operation as local File Manager
/// rename. A preexisting (or concurrently created) backup/target must never
/// be overwritten by a staged download promotion.
async fn move_local_noreplace(from: &Path, to: &Path) -> Result<()> {
    let from = from.to_path_buf();
    let to = to.to_path_buf();
    tokio::task::spawn_blocking(move || {
        let src = from.to_str().context("download path is not valid UTF-8")?;
        let dest = to
            .to_str()
            .context("download destination is not valid UTF-8")?;
        crate::remotefs::local::rename_local_noreplace(src, dest)
    })
    .await
    .context("join local no-replace download promotion")?
}

/// The common staging path covers every non-Agent download backend: FTP,
/// FTPS, SFTP, S3, WebDAV, HTTP, Dropbox, OneDrive, Drive, Box and SaaS APIs.
/// A provider returning success is not proof that the bytes hit the local
/// temporary file. Do not replace a preexisting file unless the byte count
/// matches the actual staging file and it has been synced to persistent storage.
pub(super) async fn verify_staged_download_bytes(staging: &Path, transferred: u64) -> Result<()> {
    let metadata = tokio::fs::symlink_metadata(staging)
        .await
        .context("inspect completed download staging file")?;
    anyhow::ensure!(
        metadata.file_type().is_file(),
        "download staging is not a regular file"
    );
    anyhow::ensure!(
        metadata.len() == transferred,
        "download staging size mismatch: counted {transferred} bytes, stored {}",
        metadata.len()
    );
    let file = tokio::fs::OpenOptions::new()
        .write(true)
        .open(staging)
        .await
        .context("open download staging for durability check")?;
    file.sync_all()
        .await
        .context("sync completed download staging before promotion")?;
    Ok(())
}

pub(super) async fn promote_staged_download(
    staging: &Path,
    final_path: &Path,
    id: &str,
) -> Result<()> {
    let backup = download_backup_path(final_path, id);
    let mut backup_created = false;

    let staging_meta = tokio::fs::symlink_metadata(staging)
        .await
        .with_context(|| format!("lstat completed download staging {}", staging.display()))?;
    anyhow::ensure!(
        staging_meta.file_type().is_file(),
        "download promotion refused: staging target is not a regular file"
    );

    if local_path_exists_fail_closed(final_path).await? {
        let target_meta = tokio::fs::symlink_metadata(final_path)
            .await
            .with_context(|| format!("lstat download target {}", final_path.display()))?;
        anyhow::ensure!(
            target_meta.file_type().is_file(),
            "download overwrite refused: existing destination is not a regular file"
        );
        // A previous interrupted promotion may have left a backup. Never
        // remove it blindly: preserve the only copy of a user's original.
        move_local_noreplace(final_path, &backup)
            .await
            .with_context(|| {
                format!(
                    "preserve existing local download target {} without overwriting a backup",
                    final_path.display()
                )
            })?;
        backup_created = true;
    }

    if let Err(error) = move_local_noreplace(staging, final_path).await {
        if backup_created {
            match move_local_noreplace(&backup, final_path).await {
                Ok(()) => {
                    return Err(error).with_context(|| {
                        format!(
                            "promote staged download over {}; original target was restored",
                            final_path.display()
                        )
                    });
                }
                Err(restore_error) => {
                    return Err(anyhow::anyhow!(
                        "failed to promote staged download over {}; original target could not be restored automatically and remains at {}: {}; restore error: {}",
                        final_path.display(),
                        backup.display(),
                        error,
                        restore_error
                    ));
                }
            }
        }
        return Err(error)
            .with_context(|| format!("promote staged download to {}", final_path.display()));
    }

    if backup_created {
        let _ = tokio::fs::remove_file(&backup).await;
    }
    Ok(())
}

/// Finish a Skip/Rename download without racing another process or queued
/// transfer which created the destination after the initial existence check.
/// This is an atomic move-if-absent, not a check-then-rename sequence.
pub(super) async fn promote_staged_download_noreplace(
    staging: &Path,
    final_path: &Path,
) -> Result<()> {
    let metadata = tokio::fs::symlink_metadata(staging)
        .await
        .context("inspect non-overwrite staging file")?;
    anyhow::ensure!(
        metadata.file_type().is_file(),
        "non-overwrite promotion requires a regular staging file"
    );
    move_local_noreplace(staging, final_path)
        .await
        .with_context(|| {
            format!(
                "download target already occupied or cannot be created without replacing {}",
                final_path.display()
            )
        })
}

/// Delta downloads have already passed a whole-file BLAKE3 check.
/// Verify actual staged bytes and durability before making the new version
/// visible. Existing *regular* destinations are replaced by one filesystem
/// rename, so a crash/concurrent reader never observes the missing-file gap
/// inherent in move-to-backup followed by move-from-staging.
/// For absent destinations use an atomic no-replace move to prevent races.
pub(super) async fn commit_verified_delta_output(
    staging: &Path,
    final_path: &Path,
    expected_bytes: u64,
) -> Result<()> {
    verify_staged_download_bytes(staging, expected_bytes).await?;
    match tokio::fs::symlink_metadata(final_path).await {
        Ok(metadata) => {
            anyhow::ensure!(
                metadata.file_type().is_file(),
                "delta download destination must be a real file"
            );
            tokio::fs::rename(staging, final_path)
                .await
                .with_context(|| format!("atomically replace verified delta target {}", final_path.display()))
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            promote_staged_download_noreplace(staging, final_path).await
        }
        Err(error) => Err(error).with_context(|| {
            format!("inspect delta download destination {}", final_path.display())
        }),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn delta_output_commit_requires_exact_staged_bytes() {
        let root = std::env::temp_dir().join(format!("ghostftp-delta-safe-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let final_path = root.join("existing.bin");
        let staging = root.join("delta.part");
        tokio::fs::write(&final_path, b"original data")
            .await
            .unwrap();
        tokio::fs::write(&staging, b"new").await.unwrap();
        assert!(
            commit_verified_delta_output(&staging, &final_path, 4)
                .await
                .is_err()
        );
        assert_eq!(
            tokio::fs::read(&final_path).await.unwrap(),
            b"original data"
        );
        commit_verified_delta_output(&staging, &final_path, 3)
            .await
            .unwrap();
        assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"new");
        assert!(tokio::fs::symlink_metadata(&staging).await.is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn delta_output_commit_rejects_symlink_destination() {
        use std::os::unix::fs::symlink;
        let root = std::env::temp_dir().join(format!("ghostftp-delta-link-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let protected = root.join("protected.bin");
        let target = root.join("target.bin");
        let staging = root.join("delta.part");
        tokio::fs::write(&protected, b"not overwritten")
            .await
            .unwrap();
        tokio::fs::write(&staging, b"new").await.unwrap();
        symlink(&protected, &target).unwrap();
        assert!(commit_verified_delta_output(&staging, &target, 3)
            .await
            .is_err());
        assert_eq!(
            tokio::fs::read(&protected).await.unwrap(),
            b"not overwritten"
        );
        assert!(tokio::fs::symlink_metadata(&target)
            .await
            .unwrap()
            .file_type()
            .is_symlink());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }
}
