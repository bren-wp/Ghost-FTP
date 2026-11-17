//! Fresh, sibling-staged FTP/FTPS/SFTP uploads. Existing remote files must not
//! be truncated by a failed write before the completed staging file is verified.
//! A server may reject rename-over-existing; we fail closed, never delete the
//! previous version to force promotion.
use crate::remotefs::{self, RemoteFs};
use crate::session::Session;
use anyhow::{Context, Result};
use sha2::{Digest, Sha256};
use std::io::SeekFrom;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncSeekExt};
use uuid::Uuid;

#[cfg(unix)]
use std::os::unix::fs::MetadataExt;
#[cfg(windows)]
use std::os::windows::fs::MetadataExt;

/// File identity and last modification at the start of a staged upload.
/// This is a low-overhead guard against ordinary source edits/symlink swaps,
/// not a cryptographic guarantee against forged timestamps or live write races.
#[derive(Debug, Clone)]
pub(super) struct LocalUploadSnapshot {
    len: u64,
    modified: std::time::SystemTime,
    // Hash of bounded source windows, NOT a full-file integrity digest.
    sampled_sha256: [u8; 32],
    #[cfg(unix)]
    device: u64,
    #[cfg(unix)]
    inode: u64,
    #[cfg(windows)]
    created: u64,
}

impl LocalUploadSnapshot {
    pub(super) fn capture(metadata: &std::fs::Metadata) -> Result<Self> {
        anyhow::ensure!(
            metadata.file_type().is_file(),
            "staged upload source is not a regular file"
        );
        Ok(Self {
            len: metadata.len(),
            // Initialized by capture_local_upload_snapshot after opening the source
            // and validating the same file handle and pathname before/after reads.
            sampled_sha256: [0; 32],
            modified: metadata
                .modified()
                .context("read upload source modification time")?,
            #[cfg(unix)]
            device: metadata.dev(),
            #[cfg(unix)]
            inode: metadata.ino(),
            #[cfg(windows)]
            created: metadata.creation_time(),
        })
    }

    pub(super) fn verify(&self, metadata: &std::fs::Metadata) -> Result<()> {
        anyhow::ensure!(
            metadata.file_type().is_file(),
            "staged upload source changed into a non-file"
        );
        anyhow::ensure!(
            self.len == metadata.len()
                && self.modified
                    == metadata
                        .modified()
                        .context("recheck upload source modification time")?,
            "staged upload source changed during transfer"
        );
        #[cfg(unix)]
        anyhow::ensure!(
            self.device == metadata.dev() && self.inode == metadata.ino(),
            "staged upload source was replaced during transfer"
        );
        #[cfg(windows)]
        anyhow::ensure!(
            self.created == metadata.creation_time(),
            "staged upload source was replaced during transfer"
        );
        Ok(())
    }
}

/// Compare bounded windows (first, middle and last 64 KiB) without loading
/// multi-gigabyte uploads into memory. Small files are hashed in full. This
/// detects common same-size edits with restored timestamps, but cannot detect
/// changes exclusively outside the sampled windows; it is not end-to-end
/// integrity verification against the remote server.
async fn sampled_source_sha256(
    path: &std::path::Path,
    source: &LocalUploadSnapshot,
) -> Result<[u8; 32]> {
    const WINDOW: u64 = 64 * 1024;
    let mut file = tokio::fs::File::open(path)
        .await
        .with_context(|| format!("open upload source for fingerprint {}", path.display()))?;
    source.verify(
        &file
            .metadata()
            .await
            .context("stat opened upload fingerprint source")?,
    )?;

    let last = source.len.saturating_sub(WINDOW);
    let mut offsets = vec![0, last / 2, last];
    offsets.sort_unstable();
    offsets.dedup();
    let mut hasher = Sha256::new();
    hasher.update(source.len.to_le_bytes());
    let mut buffer = [0u8; WINDOW as usize];
    for offset in offsets {
        let count = (source.len - offset).min(WINDOW) as usize;
        file.seek(SeekFrom::Start(offset))
            .await
            .context("seek upload fingerprint window")?;
        file.read_exact(&mut buffer[..count])
            .await
            .context("read upload fingerprint window")?;
        hasher.update(offset.to_le_bytes());
        hasher.update(&buffer[..count]);
    }

    source.verify(
        &file
            .metadata()
            .await
            .context("recheck opened upload fingerprint source")?,
    )?;
    source.verify(
        &tokio::fs::symlink_metadata(path)
            .await
            .context("recheck upload fingerprint source pathname")?,
    )?;
    Ok(hasher.finalize().into())
}

pub(super) async fn capture_local_upload_snapshot(
    path: &std::path::Path,
) -> Result<LocalUploadSnapshot> {
    let metadata = tokio::fs::symlink_metadata(path)
        .await
        .with_context(|| format!("lstat staged upload source {}", path.display()))?;
    let mut source = LocalUploadSnapshot::capture(&metadata)?;
    source.sampled_sha256 = sampled_source_sha256(path, &source).await?;
    Ok(source)
}

pub(super) async fn verify_local_upload_snapshot(
    path: &std::path::Path,
    source: &LocalUploadSnapshot,
) -> Result<()> {
    let metadata = tokio::fs::symlink_metadata(path)
        .await
        .with_context(|| format!("recheck staged upload source {}", path.display()))?;
    source.verify(&metadata)?;
    anyhow::ensure!(
        sampled_source_sha256(path, source).await? == source.sampled_sha256,
        "staged upload source sampled content changed during transfer"
    );
    Ok(())
}

pub(super) fn fresh_remote_upload_staging_path(final_remote: &str, id: &str) -> Result<String> {
    // Do not embed an arbitrary user-supplied id or filename in an FTP command.
    let transfer_id = Uuid::parse_str(id).context("upload transfer id is not a UUID")?;
    let (parent, _) = remotefs::remote_destination_parent_name(final_remote)?;
    let staging_name = format!(
        ".ghostftp-upload-{}-{}.part",
        transfer_id.simple(),
        Uuid::new_v4().simple()
    );
    // "shared.dat" has parent "." in the remote inventory API. A mutation
    // pathname cannot contain a dot component; a bare sibling filename is
    // both safe and valid for FTP/SFTP operating in the current directory.
    let staging_path = if parent == "." {
        staging_name
    } else {
        super::join_remote(parent, &staging_name)
    };
    remotefs::validate_protocol_mutation_path(&staging_path)?;
    anyhow::ensure!(
        staging_path != final_remote,
        "upload staging must differ from destination"
    );
    Ok(staging_path)
}

/// FTP has no atomic exclusive STOR creation. A cryptographically random
/// filename plus strict preflight protects ordinary collision cases; SFTP
/// additionally uses CREATE|EXCLUDE when opening the staging file.
pub(super) async fn preflight_ftp_upload_staging(
    session: &Arc<crate::session::FtpSession>,
    staging: &str,
) -> Result<()> {
    anyhow::ensure!(
        !remotefs::ftp::path_exists_fail_closed(session, staging).await?,
        "temporary remote upload path is occupied"
    );
    Ok(())
}

/// FTP's successful local write count is not proof that the server stored
/// all bytes. Refuse remote promotion unless the server can also confirm SIZE.
/// If SIZE is unsupported, preserve the existing destination and surface an
/// error rather than silently accepting a possibly incomplete replacement.
pub(super) async fn verify_ftp_upload_staging_size(
    session: &Arc<crate::session::FtpSession>,
    staging: &str,
    expected: u64,
) -> Result<()> {
    let remote = staging.to_string();
    let stored = session
        .with_stream(move |stream| Ok(stream.size(&remote)? as u64))
        .await
        .context("FTP server must support SIZE for safe upload promotion")?;
    anyhow::ensure!(
        stored == expected,
        "FTP staged upload size mismatch: expected {expected} bytes, server stored {stored}"
    );
    Ok(())
}

/// Promote only a fully verified upload. Non-overwrite policy additionally
/// checks that no competing writer occupied the chosen final name since
/// enqueue. FTP cannot make this last check atomic with RNTO; disclose that.
pub(super) async fn promote_remote_upload(
    session: &Arc<Session>,
    staging: &str,
    final_remote: &str,
    allow_replace: bool,
) -> Result<()> {
    remotefs::validate_protocol_mutation_path(staging)?;
    remotefs::validate_protocol_mutation_path(final_remote)?;
    match &**session {
        Session::Ssh(ssh) => {
            if !allow_replace {
                // The SftpFs rename path performs a fail-closed *directory
                // inventory*: try_exists alone may follow a dangling symlink
                // and falsely report that its destination is free.
                return remotefs::sftp::SftpFs::new(ssh.clone())
                    .rename(staging, final_remote)
                    .await
                    .context("SFTP staged upload target is occupied or cannot be safely listed");
            }
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            sftp.rename(staging, final_remote)
                .await
                .context("SFTP server could not safely promote verified upload")?;
            Ok(())
        }
        Session::Ftp(ftp) => {
            if !allow_replace {
                anyhow::ensure!(
                    !remotefs::ftp::path_exists_fail_closed(ftp, final_remote).await?,
                    "FTP upload destination became occupied before promotion"
                );
            }
            let source = staging.to_string();
            let target = final_remote.to_string();
            ftp.with_stream(move |stream| {
                stream
                    .rename(&source, &target)
                    .context("FTP server could not safely promote verified upload")?;
                Ok(())
            })
            .await
        }
        _ => anyhow::bail!("staged remote upload requires FTP, FTPS or SFTP"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn sampled_fingerprint_handles_empty_single_byte_and_window_boundaries() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-sampled-boundaries-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();

        for (index, length) in [0usize, 1, 64 * 1024, 64 * 1024 + 1]
            .into_iter()
            .enumerate()
        {
            let path = root.join(format!("source-{index}.bin"));
            let mut bytes = vec![0x5Au8; length];
            tokio::fs::write(&path, &bytes).await.unwrap();
            let snapshot = capture_local_upload_snapshot(&path).await.unwrap();
            assert_eq!(snapshot.len, length as u64);
            verify_local_upload_snapshot(&path, &snapshot)
                .await
                .unwrap();

            if !bytes.is_empty() {
                bytes[length - 1] ^= 0xFF;
                tokio::fs::write(&path, &bytes).await.unwrap();
                std::fs::File::options()
                    .write(true)
                    .open(&path)
                    .unwrap()
                    .set_modified(snapshot.modified)
                    .unwrap();
                assert!(verify_local_upload_snapshot(&path, &snapshot)
                    .await
                    .is_err());
            }
        }

        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[test]
    fn staged_upload_uses_an_unpredictable_sibling_not_the_user_target() {
        let id = Uuid::new_v4().to_string();
        let a = fresh_remote_upload_staging_path("/sites/public/file.txt", &id).unwrap();
        let b = fresh_remote_upload_staging_path("/sites/public/file.txt", &id).unwrap();
        assert!(a.starts_with("/sites/public/.ghostftp-upload-"));
        assert!(a.ends_with(".part"));
        assert_ne!(a, b);
        assert_ne!(a, "/sites/public/file.txt");
        assert!(!a.contains("file.txt"));
    }

    #[test]
    fn staging_refuses_unsafe_names_and_traversal_before_remote_io() {
        let id = Uuid::new_v4().to_string();
        for target in ["/a/../file", "/a/unsafe\\name", "/a/control\nname", "/"] {
            assert!(fresh_remote_upload_staging_path(target, &id).is_err());
        }
        assert!(fresh_remote_upload_staging_path("/safe/file", "../unsafe").is_err());
    }

    #[test]
    fn concurrent_transfers_cannot_share_the_same_staging_name() {
        let id = Uuid::new_v4().to_string();
        let names: std::collections::HashSet<_> = (0..128)
            .map(|_| fresh_remote_upload_staging_path("shared.dat", &id).unwrap())
            .collect();
        assert_eq!(names.len(), 128);
        assert!(names.iter().all(|name| !name.starts_with("./")));
    }

    #[tokio::test]
    async fn source_changed_during_upload_must_not_be_promoted() {
        let root = std::env::temp_dir().join(format!("ghostftp-source-guard-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let source = root.join("source.bin");
        tokio::fs::write(&source, b"before").await.unwrap();
        let snapshot = capture_local_upload_snapshot(&source).await.unwrap();
        verify_local_upload_snapshot(&source, &snapshot)
            .await
            .unwrap();
        tokio::fs::write(&source, b"changed-and-longer")
            .await
            .unwrap();
        assert!(verify_local_upload_snapshot(&source, &snapshot)
            .await
            .is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[tokio::test]
    async fn detects_same_size_edit_even_when_mtime_is_restored() {
        let root = std::env::temp_dir().join(format!("ghostftp-sampled-source-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let path = root.join("source.bin");
        tokio::fs::write(&path, b"original-content").await.unwrap();
        let original = capture_local_upload_snapshot(&path).await.unwrap();
        tokio::fs::write(&path, b"modified-content").await.unwrap();
        std::fs::File::options()
            .write(true)
            .open(&path)
            .unwrap()
            .set_modified(original.modified)
            .unwrap();
        // The legacy metadata guard alone accepts this same-file, same-length
        // rewrite when the original modification timestamp is restored.
        original.verify(&std::fs::metadata(&path).unwrap()).unwrap();
        assert!(verify_local_upload_snapshot(&path, &original)
            .await
            .is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(target_os = "linux")]
    #[tokio::test]
    async fn sparse_source_larger_than_four_gib_uses_bounded_fingerprint_io() {
        use std::io::{Seek, Write};
        let root = std::env::temp_dir().join(format!("ghostftp-sparse-upload-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let path = root.join("large.bin");
        let len = u32::MAX as u64 + 17;
        let file = std::fs::File::create(&path).unwrap();
        file.set_len(len).unwrap();
        drop(file);
        let snapshot = capture_local_upload_snapshot(&path).await.unwrap();
        assert_eq!(snapshot.len, len);
        verify_local_upload_snapshot(&path, &snapshot)
            .await
            .unwrap();

        let mut file = std::fs::OpenOptions::new().write(true).open(&path).unwrap();
        file.seek(SeekFrom::Start(len - 1)).unwrap();
        file.write_all(b"X").unwrap();
        file.set_modified(snapshot.modified).unwrap();
        drop(file);
        snapshot.verify(&std::fs::metadata(&path).unwrap()).unwrap();
        assert!(verify_local_upload_snapshot(&path, &snapshot)
            .await
            .is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn same_length_source_replacement_or_symlink_is_not_accepted() {
        let root = std::env::temp_dir().join(format!("ghostftp-source-swap-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let source = root.join("source.bin");
        let replacement = root.join("replacement.bin");
        tokio::fs::write(&source, b"original").await.unwrap();
        tokio::fs::write(&replacement, b"replaced").await.unwrap();
        let snapshot = capture_local_upload_snapshot(&source).await.unwrap();
        tokio::fs::remove_file(&source).await.unwrap();
        tokio::fs::rename(&replacement, &source).await.unwrap();
        assert!(verify_local_upload_snapshot(&source, &snapshot)
            .await
            .is_err());
        tokio::fs::remove_file(&source).await.unwrap();
        std::os::unix::fs::symlink(&root, &source).unwrap();
        assert!(capture_local_upload_snapshot(&source).await.is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }
}
