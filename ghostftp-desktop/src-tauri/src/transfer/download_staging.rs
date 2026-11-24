//! Durable, exclusive download staging and lossless local promotion.
//! Includes guards common to FTP, FTPS, SFTP, HTTP, cloud and Agent delta.
use crate::db::RecoveryPhase;
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
        .read(true)
        .write(true)
        .create_new(true)
        .open(path)
        .await
        .map_err(DownloadStagingOpenFailure)
        .with_context(|| format!("exclusively create download staging {}", path.display()))
}

/// Ownership evidence is a live handle cloned only after exclusive creation.
/// Blocking FTP workers share this record with the async transfer runner.
#[derive(Clone, Default)]
pub(super) struct DownloadStageOwner {
    file: std::sync::Arc<std::sync::Mutex<Option<std::fs::File>>>,
    #[cfg(target_os = "linux")]
    parent: Option<std::sync::Arc<std::fs::File>>,
}

impl DownloadStageOwner {
    pub(super) fn for_parent(parent: &DownloadParentGuard) -> Self {
        #[cfg(not(target_os = "linux"))]
        let _ = parent;
        Self {
            #[cfg(target_os = "linux")]
            parent: Some(parent.handle.clone()),
            ..Self::default()
        }
    }

    pub(super) fn record_std(&self, file: &std::fs::File) -> Result<()> {
        let clone = file.try_clone()?;
        let mut owned = self
            .file
            .lock()
            .map_err(|_| anyhow::anyhow!("download stage ownership lock poisoned"))?;
        *owned = Some(clone);
        Ok(())
    }

    pub(super) async fn record(&self, file: &tokio::fs::File) -> Result<()> {
        let clone = file.try_clone().await?.into_std().await;
        self.record_std(&clone)
    }

    pub(super) fn create_std(&self, path: &Path) -> Result<std::fs::File> {
        #[cfg(target_os = "linux")]
        let file = if let Some(parent) = &self.parent {
            let name = path.file_name().context("staging has no basename")?;
            create_pinned_staging(parent, name)
        } else {
            std::fs::OpenOptions::new()
                .read(true)
                .write(true)
                .create_new(true)
                .open(path)
        };
        #[cfg(not(target_os = "linux"))]
        let file = std::fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create_new(true)
            .open(path);
        let file = file
            .map_err(DownloadStagingOpenFailure)
            .context("exclusively create owned download stage")?;
        self.record_std(&file)?;
        Ok(file)
    }

    pub(super) async fn create(&self, path: &Path) -> Result<tokio::fs::File> {
        let owner = self.clone();
        let path = path.to_path_buf();
        let file = tokio::task::spawn_blocking(move || owner.create_std(&path))
            .await
            .context("join owned staging creation")??;
        Ok(tokio::fs::File::from_std(file))
    }

    pub(super) async fn cleanup(&self, parent: &DownloadParentGuard, path: &Path) {
        let owned = self.file.lock().ok().and_then(|mut file| file.take());
        if let Some(file) = owned {
            let file = tokio::fs::File::from_std(file);
            parent.cleanup_verified(path, &file).await;
        }
    }
}

/// Open an existing resumable stage without truncation or following links.
/// Validation failures must not grant the caller cleanup ownership.
#[cfg(test)]
pub(super) async fn open_resumable_download_staging(path: &Path) -> Result<tokio::fs::File> {
    open_owned_resumable_download_staging(path, &DownloadStageOwner::default()).await
}

pub(super) async fn open_owned_resumable_download_staging(
    path: &Path,
    owner: &DownloadStageOwner,
) -> Result<tokio::fs::File> {
    async fn open(path: &Path, owner: &DownloadStageOwner) -> Result<tokio::fs::File> {
        let mut options = tokio::fs::OpenOptions::new();
        options.read(true).write(true);
        #[cfg(unix)]
        options.custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK);
        #[cfg(windows)]
        options.custom_flags(windows_sys::Win32::Storage::FileSystem::FILE_FLAG_OPEN_REPARSE_POINT);
        let (file, created) = match options.create_new(true).open(path).await {
            Ok(file) => (file, true),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
                (options.create_new(false).open(path).await?, false)
            }
            Err(error) => return Err(error.into()),
        };
        let metadata = file.metadata().await?;
        anyhow::ensure!(
            metadata.file_type().is_file(),
            "resumable staging is not a regular file"
        );
        #[cfg(unix)]
        {
            use std::os::unix::fs::MetadataExt;
            anyhow::ensure!(
                metadata.nlink() == 1,
                "resumable staging must not have additional hard links"
            );
        }
        #[cfg(windows)]
        {
            windows_staging_identity(&file)?;
        }
        if created {
            owner.record(&file).await?;
        }
        Ok(file)
    }

    open(path, owner).await.map_err(|error| {
        DownloadStagingOpenFailure(std::io::Error::other(error.to_string())).into()
    })
}

/// Best-effort cleanup must never remove a path for which exclusive
/// creation failed. On subsequent I/O errors, only regular staging files may
/// be unlinked, never a planted symlink or directory.
pub(super) async fn cleanup_failed_download_staging(path: &Path, error: &anyhow::Error) {
    if error.downcast_ref::<DownloadStagingOpenFailure>().is_some() {
        return;
    }
    cleanup_owned_regular_staging_file(path).await;
}

/// After exclusive creation, only remove a regular staging entry. A planted
/// symlink or directory must not be deleted during best-effort cleanup.
pub(super) async fn cleanup_owned_regular_staging_file(path: &Path) {
    if tokio::fs::symlink_metadata(path)
        .await
        .is_ok_and(|meta| meta.file_type().is_file())
    {
        let _ = tokio::fs::remove_file(path).await;
    }
}

/// The delta-upload planner runs on a blocking thread and needs a synchronous
/// writer. Use CREATE_NEW just like a downloaded staging file, so a failed
/// open cannot truncate a user's file or follow a planted symlink.
pub(super) fn create_new_delta_upload_patch(path: &Path) -> Result<std::fs::File> {
    std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(DownloadStagingOpenFailure)
        .with_context(|| format!("exclusively create Agent delta patch {}", path.display()))
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

/// Rename relative to a pinned directory handle so the kernel cannot resolve
/// the source and destination through different versions of a replaced parent.
#[cfg(target_os = "linux")]
fn rename_pinned_siblings(
    parent: &std::fs::File,
    from: &std::ffi::OsStr,
    to: &std::ffi::OsStr,
) -> Result<()> {
    use std::os::fd::AsRawFd;
    use std::os::unix::ffi::OsStrExt;

    let source = std::ffi::CString::new(from.as_bytes()).context("rename source contains NUL")?;
    let target = std::ffi::CString::new(to.as_bytes()).context("rename target contains NUL")?;
    // SAFETY: parent owns a live directory descriptor and both names are
    // NUL-terminated basenames supplied by Path::file_name.
    let result = unsafe {
        libc::renameat2(
            parent.as_raw_fd(),
            source.as_ptr(),
            parent.as_raw_fd(),
            target.as_ptr(),
            libc::RENAME_NOREPLACE,
        )
    };
    if result != 0 {
        return Err(std::io::Error::last_os_error()).context("rename pinned download siblings");
    }
    Ok(())
}

/// Exclusively create a regular child in the original pinned directory.
#[cfg(target_os = "linux")]
fn create_pinned_staging(
    parent: &std::fs::File,
    name: &std::ffi::OsStr,
) -> std::io::Result<std::fs::File> {
    use std::os::fd::{AsRawFd, FromRawFd};
    use std::os::unix::ffi::OsStrExt;

    let bytes = name.as_bytes();
    if bytes.is_empty() || bytes == b"." || bytes == b".." || bytes.contains(&b'/') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "staging requires a basename",
        ));
    }
    let name = std::ffi::CString::new(bytes)?;
    // SAFETY: parent owns a live descriptor and name is a NUL-terminated basename.
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            name.as_ptr(),
            libc::O_RDWR | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0o666 as libc::mode_t,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    // SAFETY: openat returned an independently owned descriptor.
    Ok(unsafe { std::fs::File::from_raw_fd(fd) })
}

/// Pin the parent once; the caller owns this descriptor until all operations finish.
#[cfg(target_os = "linux")]
fn pin_download_parent(parent: &Path) -> Result<std::fs::File> {
    use std::os::unix::fs::OpenOptionsExt;

    let directory = if parent.as_os_str().is_empty() {
        Path::new(".")
    } else {
        parent
    };
    std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_PATH | libc::O_DIRECTORY | libc::O_CLOEXEC)
        .open(directory)
        .context("pin common download parent directory")
}

// O_PATH handles cannot be fsynced. Open the same pinned directory via
// openat(".") rather than resolving its original, potentially replaced path.
#[cfg(target_os = "linux")]
fn sync_recovery_directory(parent: &std::fs::File) -> Result<()> {
    use std::os::fd::{AsRawFd, FromRawFd};

    // SAFETY: parent is live; the relative name is a constant C string.
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            c".".as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error()).context("open recovery directory for sync");
    }
    // SAFETY: successful openat transfers ownership of this descriptor.
    let directory = unsafe { std::fs::File::from_raw_fd(fd) };
    anyhow::ensure!(
        same_download_directory(parent, &directory)?,
        "recovery directory sync identity mismatch"
    );
    directory
        .sync_all()
        .context("persist recovery directory entries")
}

/// Hold the initially resolved directory for the lifetime of a download.
/// Existing path-based backend I/O still needs identity checks at boundaries.
pub(super) struct DownloadParentGuard {
    path: PathBuf,
    handle: std::sync::Arc<std::fs::File>,
    recovery_db: Option<std::sync::Arc<crate::db::Db>>,
}

// Positional reads preserve the writer's offset. Hash the entire observed
// length and reject common concurrent changes; this is not a filesystem lock.
#[cfg(any(target_os = "linux", windows))]
fn recovery_file_hash(file: &std::fs::File, identity: (u64, u64, u64)) -> Result<String> {
    use sha2::{Digest, Sha256};

    let before = file.metadata()?;
    anyhow::ensure!(
        recovery_file_identity(file)? == identity,
        "recovery content handle identity mismatch"
    );
    let mut hash = Sha256::new();
    let mut buffer = vec![0u8; 128 * 1024];
    let mut offset = 0u64;
    loop {
        let remaining = identity.2 - offset;
        let count = if remaining == 0 {
            1
        } else {
            remaining.min(buffer.len() as u64) as usize
        };
        #[cfg(target_os = "linux")]
        let read = {
            use std::os::unix::fs::FileExt;
            file.read_at(&mut buffer[..count], offset)
        };
        #[cfg(windows)]
        let read = {
            use std::os::windows::fs::FileExt;
            file.seek_read(&mut buffer[..count], offset)
        };
        let read = match read {
            Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
            other => other?,
        };
        if remaining == 0 {
            anyhow::ensure!(read == 0, "recovery file grew during hashing");
            break;
        }
        anyhow::ensure!(read != 0, "recovery file shrank during hashing");
        hash.update(&buffer[..read]);
        offset += read as u64;
    }
    let after = file.metadata()?;
    #[cfg(target_os = "linux")]
    let unchanged = {
        use std::os::unix::fs::MetadataExt;
        before.mtime() == after.mtime()
            && before.mtime_nsec() == after.mtime_nsec()
            && before.ctime() == after.ctime()
            && before.ctime_nsec() == after.ctime_nsec()
    };
    #[cfg(windows)]
    let unchanged = {
        use std::os::windows::fs::MetadataExt;
        before.creation_time() == after.creation_time()
            && before.last_write_time() == after.last_write_time()
    };
    anyhow::ensure!(
        unchanged && before.len() == after.len() && recovery_file_identity(file)? == identity,
        "recovery file changed during hashing"
    );
    Ok(format!("{:x}", hash.finalize()))
}

#[cfg(any(target_os = "linux", windows))]
fn recovery_file_identity(file: &std::fs::File) -> Result<(u64, u64, u64)> {
    let metadata = file.metadata()?;
    anyhow::ensure!(metadata.is_file(), "recovery handle is not a regular file");
    #[cfg(target_os = "linux")]
    {
        use std::os::unix::fs::MetadataExt;
        anyhow::ensure!(metadata.nlink() == 1, "recovery handle has hard links");
        Ok((metadata.dev(), metadata.ino(), metadata.len()))
    }
    #[cfg(windows)]
    {
        let (volume, index) = windows_staging_identity(file)?;
        Ok((u64::from(volume), index, metadata.len()))
    }
}

#[cfg(target_os = "linux")]
fn open_recovery_read(parent: &std::fs::File, name: &std::ffi::OsStr) -> Result<std::fs::File> {
    use std::os::fd::{AsRawFd, FromRawFd};
    use std::os::unix::ffi::OsStrExt;

    let bytes = name.as_bytes();
    anyhow::ensure!(
        !bytes.is_empty() && bytes != b"." && bytes != b".." && !bytes.contains(&b'/'),
        "recovery read requires a basename"
    );
    let name = std::ffi::CString::new(bytes)?;
    // SAFETY: the directory descriptor and NUL-terminated basename are live.
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            name.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error()).context("open pinned recovery content");
    }
    // SAFETY: successful openat returned a newly owned descriptor.
    let file = unsafe { std::fs::File::from_raw_fd(fd) };
    recovery_file_identity(&file)?;
    Ok(file)
}

#[cfg(windows)]
fn open_recovery_read(parent: &std::fs::File, name: &std::ffi::OsStr) -> Result<std::fs::File> {
    let file = windows_open_relative(parent, name, 0x81)?;
    recovery_file_identity(&file)?;
    Ok(file)
}

/// This is a durable recovery intent, not permission to mutate files after
/// restart. File identifiers may be reused after all handles are closed.
#[derive(serde::Serialize)]
struct RecoveryRecord<'a> {
    schema: u32,
    parent: &'a Path,
    stage: &'a Path,
    target: &'a Path,
    backup: &'a Path,
    parent_identity: (u64, u64),
    stage_identity: (u64, u64, u64),
    original_identity: (u64, u64, u64),
    stage_sha256: String,
    original_sha256: String,
}

/// Persisted paths are hints, not proof of ownership. Observations never
/// authorize rename, deletion or advancing the durable journal.
#[derive(serde::Deserialize)]
struct RecoveryInspectionRecord {
    schema: u32,
    parent: PathBuf,
    stage: PathBuf,
    target: PathBuf,
    backup: PathBuf,
    parent_identity: (u64, u64),
    stage_identity: (u64, u64, u64),
    original_identity: (u64, u64, u64),
    #[serde(default)]
    stage_sha256: String,
    #[serde(default)]
    original_sha256: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum EntryObservation {
    Missing,
    Original,
    Stage,
    Other,
    Unavailable,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum RecoveryObservation {
    OriginalPresent,
    BackupPresent,
    PublishedWithBackup,
    PublishedWithoutBackup,
    OriginalWithoutStage,
    BackupWithoutStage,
    Ambiguous,
    Unavailable,
}

fn classify_recovery(
    target: EntryObservation,
    stage: EntryObservation,
    backup: EntryObservation,
) -> RecoveryObservation {
    use EntryObservation::{Missing, Original, Stage, Unavailable};
    match (target, stage, backup) {
        (Unavailable, _, _) | (_, Unavailable, _) | (_, _, Unavailable) => {
            RecoveryObservation::Unavailable
        }
        (Original, Stage, Missing) => RecoveryObservation::OriginalPresent,
        (Missing, Stage, Original) => RecoveryObservation::BackupPresent,
        (Stage, Missing, Original) => RecoveryObservation::PublishedWithBackup,
        (Stage, Missing, Missing) => RecoveryObservation::PublishedWithoutBackup,
        (Original, Missing, Missing) => RecoveryObservation::OriginalWithoutStage,
        (Missing, Missing, Original) => RecoveryObservation::BackupWithoutStage,
        _ => RecoveryObservation::Ambiguous,
    }
}

fn observation_matches_phase(phase: &str, observed: RecoveryObservation) -> bool {
    use RecoveryObservation::*;
    matches!(
        (phase, observed),
        ("prepared" | "rolled_back", OriginalPresent)
            | ("backup_saved" | "rollback_pending", BackupPresent)
            | ("promoted" | "cleanup_pending", PublishedWithBackup)
            | ("cleanup_pending", PublishedWithoutBackup)
    )
}

impl RecoveryInspectionRecord {
    fn valid_layout(&self) -> bool {
        self.schema == 2
            && [&self.stage_sha256, &self.original_sha256]
                .iter()
                .all(|hash| hash.len() == 64 && hash.bytes().all(|b| b.is_ascii_hexdigit()))
            && self.parent.is_absolute()
            && self.stage.parent() == Some(self.parent.as_path())
            && self.target.parent() == Some(self.parent.as_path())
            && self.backup.parent() == Some(self.parent.as_path())
            && self.stage.file_name().is_some()
            && self.target.file_name().is_some()
            && self.backup.file_name().is_some()
            && self.stage != self.target
            && self.stage != self.backup
            && self.target != self.backup
            && (self.stage_identity.0, self.stage_identity.1)
                != (self.original_identity.0, self.original_identity.1)
            && !self
                .parent
                .components()
                .any(|component| matches!(component, std::path::Component::ParentDir))
    }

    // The destination key is derived from fields inside the authenticated
    // payload, never from the mutable SQLite reservation alone.
    fn matches_reservation(&self, key: Option<&str>) -> bool {
        self.valid_layout()
            && recovery_destination_key(self.parent_identity, &self.target)
                .is_ok_and(|expected| key == Some(expected.as_str()))
    }

    fn inspect_entry(&self, parent: &std::fs::File, path: &Path) -> EntryObservation {
        let Some(name) = path.file_name() else {
            return EntryObservation::Unavailable;
        };
        #[cfg(target_os = "linux")]
        let identity = match pinned_entry(parent, name) {
            Ok(None) => return EntryObservation::Missing,
            Ok(Some(stat)) if pinned_regular_file(&stat) && stat.st_nlink == 1 => {
                let Ok(size) = u64::try_from(stat.st_size) else {
                    return EntryObservation::Other;
                };
                (stat.st_dev, stat.st_ino, size)
            }
            Ok(Some(_)) => return EntryObservation::Other,
            Err(_) => return EntryObservation::Unavailable,
        };
        #[cfg(windows)]
        let identity = match windows_open_relative(parent, name, 0x80) {
            Ok(file) => {
                let Ok((volume, index)) = windows_staging_identity(&file) else {
                    return EntryObservation::Other;
                };
                let Ok(metadata) = file.metadata() else {
                    return EntryObservation::Unavailable;
                };
                if !metadata.is_file() {
                    return EntryObservation::Other;
                }
                (u64::from(volume), index, metadata.len())
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                return EntryObservation::Missing;
            }
            Err(_) => return EntryObservation::Unavailable,
        };
        #[cfg(not(any(target_os = "linux", windows)))]
        let identity = {
            let _ = (parent, name);
            (0u64, 0u64, 0u64)
        };
        #[cfg(any(target_os = "linux", windows))]
        {
            let (expected_hash, observation) = if identity == self.original_identity {
                (&self.original_sha256, EntryObservation::Original)
            } else if identity == self.stage_identity {
                (&self.stage_sha256, EntryObservation::Stage)
            } else {
                return EntryObservation::Other;
            };
            match open_recovery_read(parent, name)
                .and_then(|file| recovery_file_hash(&file, identity))
            {
                Ok(hash) if &hash == expected_hash => observation,
                Ok(_) => EntryObservation::Other,
                Err(_) => EntryObservation::Unavailable,
            }
        }
        #[cfg(not(any(target_os = "linux", windows)))]
        {
            let _ = identity;
            EntryObservation::Unavailable
        }
    }

    fn inspect(&self) -> RecoveryObservation {
        if !self.valid_layout() {
            return RecoveryObservation::Ambiguous;
        }
        // Avoid initiating access to UNC or device namespaces during startup.
        #[cfg(windows)]
        if !matches!(
            self.parent.components().next(),
            Some(std::path::Component::Prefix(prefix))
                if matches!(
                    prefix.kind(),
                    std::path::Prefix::Disk(_) | std::path::Prefix::VerbatimDisk(_)
                )
        ) {
            return RecoveryObservation::Unavailable;
        }
        let Ok(parent) = open_download_directory(&self.parent) else {
            return RecoveryObservation::Unavailable;
        };
        #[cfg(unix)]
        let identity = {
            use std::os::unix::fs::MetadataExt;
            let Ok(metadata) = parent.metadata() else {
                return RecoveryObservation::Unavailable;
            };
            (metadata.dev(), metadata.ino())
        };
        #[cfg(windows)]
        let identity = {
            let Ok((volume, index)) = windows_file_identity(&parent) else {
                return RecoveryObservation::Unavailable;
            };
            (u64::from(volume), index)
        };
        #[cfg(not(any(unix, windows)))]
        let identity = (0u64, 0u64);
        if identity != self.parent_identity {
            return RecoveryObservation::Ambiguous;
        }
        let inspect = || {
            (
                self.inspect_entry(&parent, &self.target),
                self.inspect_entry(&parent, &self.stage),
                self.inspect_entry(&parent, &self.backup),
            )
        };
        let first = inspect();
        if first != inspect() {
            return RecoveryObservation::Ambiguous;
        }
        classify_recovery(first.0, first.1, first.2)
    }
}

/// Startup reconciliation repairs authenticated metadata only; no filesystem mutations.
/// Full content reads are bounded; larger records require manual review.
/// Concurrent operations can invalidate observations.
pub(super) fn reconcile_pending_recovery(db: &crate::db::Db) -> Result<()> {
    let _lock = match db.recovery_lock() {
        Ok(lock) => lock,
        Err(_) => {
            tracing::warn!("recovery index lock unavailable; inspection deferred");
            return Ok(());
        }
    };
    let mut ledger = match GlobalRecoveryLedger::load(db, false) {
        Ok(ledger) => ledger,
        Err(_) => {
            tracing::warn!("global recovery index unavailable; manual review required");
            return Ok(());
        }
    };
    if ledger.reconcile_metadata(db).is_err() || ledger.check_rows(db).is_err() {
        tracing::warn!("recovery metadata reconciliation requires manual review");
        return Ok(());
    }
    if GlobalRecoveryLedger::collect_garbage(db).is_err() {
        tracing::warn!("verified recovery garbage collection deferred");
    }
    let rows = db.download_recovery_snapshot()?;
    let truncated = rows.len() > 128;
    let mut matching = 0usize;
    let mut divergent = 0usize;
    let mut uncertain = 0usize;
    let mut rollback_candidates = 0usize;
    let mut cleanup_candidates = 0usize;
    let mut remaining_hash_bytes = 64u64 * 1024 * 1024;
    for row in rows.into_iter().take(128) {
        let signed = serde_json::from_str::<AuthenticatedRecovery>(&row.payload)
            .ok()
            .filter(|record| {
                record
                    .authenticate(&row.id, &row.payload)
                    .is_ok_and(|phase| phase.as_str() == row.phase)
            });
        let Some(signed) = signed else {
            // Legacy, tampered and inaccessible records never authorize even
            // an automatic filesystem inspection of their stored path hints.
            uncertain += 1;
            continue;
        };
        let Ok(record) = serde_json::from_str::<RecoveryInspectionRecord>(&signed.payload) else {
            uncertain += 1;
            continue;
        };
        if !record.matches_reservation(row.destination_key.as_deref()) {
            // A missing, substituted or incompatible reservation is not
            // sufficient evidence to inspect the recorded filesystem paths.
            uncertain += 1;
            continue;
        }
        // Three names, two inspections: conservatively budget six full reads
        // of the larger file. Never substitute a sampled digest.
        let cost = record
            .stage_identity
            .2
            .max(record.original_identity.2)
            .saturating_mul(6);
        if cost > remaining_hash_bytes {
            uncertain += 1;
            continue;
        }
        remaining_hash_bytes -= cost;
        let observed = record.inspect();
        match recovery_decision(true, observed) {
            RecoveryDecision::RollbackCandidate => rollback_candidates += 1,
            RecoveryDecision::CleanupCandidate => cleanup_candidates += 1,
            _ => {}
        }
        if matches!(
            observed,
            RecoveryObservation::Ambiguous | RecoveryObservation::Unavailable
        ) {
            uncertain += 1;
        } else if observation_matches_phase(&row.phase, observed) {
            matching += 1;
        } else {
            divergent += 1;
        }
    }
    if matching + divergent + uncertain > 0 {
        tracing::warn!(
            matching,
            divergent,
            uncertain,
            rollback_candidates,
            cleanup_candidates,
            truncated,
            "download recovery observations require review"
        );
    }
    Ok(())
}

const RECOVERY_VAULT_SERVICE: &str = "com.ghostftp.download-recovery.v1";
const RECOVERY_ANCHOR_SERVICE: &str = "com.ghostftp.download-recovery-anchor.v1";
const RECOVERY_INDEX_SERVICE: &str = "com.ghostftp.download-recovery-index.v1";
const RECOVERY_INDEX_KEY_SERVICE: &str = "com.ghostftp.download-recovery-index-key.v1";
const RECOVERY_BLOB_SERVICE: &str = "com.ghostftp.download-recovery-blob.v2";
// A small authenticated manifest commits to immutable eight-entry shards.
// The overall bound matches the SQLite reconciliation snapshot capacity.
const RECOVERY_SHARD_SIZE: usize = 8;
const RECOVERY_INDEX_LIMIT: usize = 128;
const RECOVERY_BLOB_CHUNK: usize = 720;
const RECOVERY_BLOB_LIMIT: usize = 256 * 1024;
const RECOVERY_GC_SERVICE: &str = "com.ghostftp.download-recovery-gc.v1";
const RECOVERY_GC_KEY_SERVICE: &str = "com.ghostftp.download-recovery-gc-keys.v1";
const RECOVERY_GC_KEY_LIMIT: usize = 32;

type RecoveryGcKeys = std::collections::BTreeSet<Uuid>;

fn read_gc_keys(scope: &str, key: &[u8; 32]) -> Result<RecoveryGcKeys> {
    use base64::Engine;

    let account = format!("{scope}:keys");
    let Some(stored) = crate::credentials::vault_get_strict(RECOVERY_GC_KEY_SERVICE, &account)?
    else {
        return Ok(RecoveryGcKeys::new());
    };
    anyhow::ensure!(stored.len() <= 1200, "recovery key GC page too large");
    let signed: AuthenticatedRecovery = serde_json::from_str(&stored)?;
    signed.verify(&account, key)?;
    anyhow::ensure!(
        signed.key_id == Uuid::nil() && signed.phases.is_empty(),
        "invalid recovery key GC page"
    );
    let bytes = base64::engine::general_purpose::STANDARD.decode(&signed.payload)?;
    anyhow::ensure!(
        bytes.len() % 16 == 0 && bytes.len() <= RECOVERY_GC_KEY_LIMIT * 16,
        "invalid recovery key GC page size"
    );
    let mut keys = RecoveryGcKeys::new();
    for chunk in bytes.as_chunks::<16>().0 {
        let id = Uuid::from_slice(chunk)?;
        anyhow::ensure!(
            !id.is_nil() && keys.insert(id),
            "invalid recovery GC key ID"
        );
    }
    Ok(keys)
}

fn write_gc_keys(scope: &str, key: &[u8; 32], keys: &RecoveryGcKeys) -> Result<()> {
    use base64::Engine;

    anyhow::ensure!(
        keys.len() <= RECOVERY_GC_KEY_LIMIT,
        "recovery key GC queue full"
    );
    let bytes: Vec<u8> = keys
        .iter()
        .flat_map(|id| id.as_bytes().iter().copied())
        .collect();
    let account = format!("{scope}:keys");
    let payload = base64::engine::general_purpose::STANDARD.encode(bytes);
    let signed = AuthenticatedRecovery::seal(&account, payload, Uuid::nil(), key)?;
    let encoded = serde_json::to_string(&signed)?;
    anyhow::ensure!(encoded.len() <= 1200, "recovery key GC page too large");
    crate::credentials::vault_set(RECOVERY_GC_KEY_SERVICE, &account, &encoded)?;
    anyhow::ensure!(
        crate::credentials::vault_get_strict(RECOVERY_GC_KEY_SERVICE, &account)?.as_deref()
            == Some(encoded.as_str()),
        "recovery key GC readback mismatch"
    );
    Ok(())
}

fn register_recovery_gc_key(scope: &str, id: Uuid) -> Result<()> {
    anyhow::ensure!(!id.is_nil(), "invalid recovery key ID");
    let key = crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, scope)?
        .context("recovery index key unavailable")?;
    GlobalRecoveryLedger::with_key(&key, |key| {
        let mut keys = read_gc_keys(scope, key)?;
        if keys.insert(id) {
            write_gc_keys(scope, key, &keys)?;
        }
        recovery_fault_checkpoint("key_registered");
        Ok(())
    })
}
const RECOVERY_GC_PAGES: usize = 4;
const RECOVERY_GC_PAGE_SIZE: usize = 12;

type RecoveryGcPage = std::collections::BTreeMap<[u8; 32], usize>;

fn gc_page_account(scope: &str, page: usize) -> String {
    format!("{scope}:{page}")
}

fn read_gc_page(scope: &str, page: usize, key: &[u8; 32]) -> Result<RecoveryGcPage> {
    use base64::Engine;

    let account = gc_page_account(scope, page);
    let Some(stored) = crate::credentials::vault_get_strict(RECOVERY_GC_SERVICE, &account)? else {
        return Ok(RecoveryGcPage::new());
    };
    anyhow::ensure!(stored.len() <= 1200, "recovery GC page too large");
    let signed: AuthenticatedRecovery = serde_json::from_str(&stored)?;
    signed.verify(&account, key)?;
    anyhow::ensure!(
        signed.key_id == Uuid::nil() && signed.phases.is_empty(),
        "invalid recovery GC page"
    );
    let bytes = base64::engine::general_purpose::STANDARD.decode(&signed.payload)?;
    anyhow::ensure!(
        bytes.len() % 36 == 0 && bytes.len() <= RECOVERY_GC_PAGE_SIZE * 36,
        "invalid recovery GC page size"
    );
    let mut entries = RecoveryGcPage::new();
    for entry in bytes.as_chunks::<36>().0 {
        let digest: [u8; 32] = entry[..32].try_into()?;
        let length = u32::from_be_bytes(entry[32..].try_into()?) as usize;
        anyhow::ensure!(
            length > 0 && length <= RECOVERY_BLOB_LIMIT && entries.insert(digest, length).is_none(),
            "invalid recovery GC candidate"
        );
    }
    Ok(entries)
}

fn write_gc_page(scope: &str, page: usize, key: &[u8; 32], entries: &RecoveryGcPage) -> Result<()> {
    use base64::Engine;

    anyhow::ensure!(
        entries.len() <= RECOVERY_GC_PAGE_SIZE,
        "recovery GC page full"
    );
    let mut bytes = Vec::new();
    for (digest, length) in entries {
        bytes.extend_from_slice(digest);
        bytes.extend_from_slice(&u32::try_from(*length)?.to_be_bytes());
    }
    let account = gc_page_account(scope, page);
    let payload = base64::engine::general_purpose::STANDARD.encode(bytes);
    let signed = AuthenticatedRecovery::seal(&account, payload, Uuid::nil(), key)?;
    let encoded = serde_json::to_string(&signed)?;
    anyhow::ensure!(encoded.len() <= 1200, "recovery GC page too large");
    crate::credentials::vault_set(RECOVERY_GC_SERVICE, &account, &encoded)?;
    anyhow::ensure!(
        crate::credentials::vault_get_strict(RECOVERY_GC_SERVICE, &account)?.as_deref()
            == Some(encoded.as_str()),
        "recovery GC page readback mismatch"
    );
    Ok(())
}

// Register ownership before writing any chunk. A bounded authenticated queue
// retains interrupted writes and superseded objects, even after process death.
fn register_recovery_gc(scope: &str, digest: [u8; 32], length: usize) -> Result<()> {
    anyhow::ensure!(
        length > 0 && length <= RECOVERY_BLOB_LIMIT,
        "invalid GC length"
    );
    let key = crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, scope)?
        .context("recovery index key unavailable")?;
    GlobalRecoveryLedger::with_key(&key, |key| {
        let mut pages = Vec::new();
        for page in 0..RECOVERY_GC_PAGES {
            let entries = read_gc_page(scope, page, key)?;
            if let Some(existing) = entries.get(&digest) {
                anyhow::ensure!(*existing == length, "recovery GC length mismatch");
                return Ok(());
            }
            pages.push(entries);
        }
        let (page, entries) = pages
            .iter_mut()
            .enumerate()
            .find(|(_, entries)| entries.len() < RECOVERY_GC_PAGE_SIZE)
            .context("recovery GC queue full; reconcile before writing")?;
        entries.insert(digest, length);
        write_gc_page(scope, page, key, entries)?;
        recovery_fault_checkpoint("gc_registered");
        Ok(())
    })
}

fn delete_recovery_blob(scope: &str, digest: &[u8; 32], length: usize) -> Result<()> {
    let length_account = recovery_blob_account(scope, digest, "length");
    if let Some(marker) =
        crate::credentials::vault_get_strict(RECOVERY_BLOB_SERVICE, &length_account)?
    {
        anyhow::ensure!(
            marker.parse::<usize>()? == length,
            "recovery GC length mismatch"
        );
        anyhow::ensure!(
            get_recovery_blob(scope, digest)?.len() == length,
            "recovery GC blob verification failed"
        );
        // Removing the length first marks a verified deletion in progress.
        // The authenticated queue retains the exact chunk count across crashes.
        crate::credentials::vault_delete_strict(RECOVERY_BLOB_SERVICE, &length_account)?;
        recovery_fault_checkpoint("gc_length_removed");
    }
    for index in 0..length.div_ceil(RECOVERY_BLOB_CHUNK) {
        let account = recovery_blob_account(scope, digest, &index.to_string());
        crate::credentials::vault_delete_strict(RECOVERY_BLOB_SERVICE, &account)?;
        recovery_fault_checkpoint("gc_chunk_removed");
    }
    Ok(())
}

#[cfg(not(test))]
#[inline]
fn recovery_fault_checkpoint(_point: &str) {}

#[cfg(test)]
fn recovery_fault_checkpoint(point: &str) {
    crate::credentials::recovery_test_checkpoint(point);
}

fn recovery_digest(bytes: &[u8]) -> [u8; 32] {
    use sha2::{Digest, Sha256};
    Sha256::digest(bytes).into()
}

fn recovery_blob_account(scope: &str, digest: &[u8; 32], part: &str) -> String {
    use base64::Engine;
    let digest = base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(digest);
    format!("{scope}:{digest}:{part}")
}

// Content-addressed immutable values are staged before publishing the root.
// Existing bytes must match exactly; a missing or corrupt chunk never permits
// reading a different generation. All credentials use persistent strict reads.
fn put_recovery_blob(scope: &str, bytes: &[u8]) -> Result<[u8; 32]> {
    use base64::Engine;

    anyhow::ensure!(
        !bytes.is_empty() && bytes.len() <= RECOVERY_BLOB_LIMIT,
        "invalid recovery blob size"
    );
    let digest = recovery_digest(bytes);
    register_recovery_gc(scope, digest, bytes.len())?;
    let put = |part: &str, value: &str| -> Result<()> {
        let account = recovery_blob_account(scope, &digest, part);
        if let Some(existing) =
            crate::credentials::vault_get_strict(RECOVERY_BLOB_SERVICE, &account)?
        {
            anyhow::ensure!(existing == value, "recovery blob collision");
            return Ok(());
        }
        crate::credentials::vault_set(RECOVERY_BLOB_SERVICE, &account, value)?;
        anyhow::ensure!(
            crate::credentials::vault_get_strict(RECOVERY_BLOB_SERVICE, &account)?.as_deref()
                == Some(value),
            "recovery blob readback mismatch"
        );
        Ok(())
    };
    for (index, chunk) in bytes.chunks(RECOVERY_BLOB_CHUNK).enumerate() {
        let encoded = base64::engine::general_purpose::STANDARD.encode(chunk);
        put(&index.to_string(), &encoded)?;
        recovery_fault_checkpoint("blob_chunk_written");
    }
    // Publish the length only after every chunk has been verified.
    put("length", &bytes.len().to_string())?;
    recovery_fault_checkpoint("blob_length_written");
    Ok(digest)
}

fn get_recovery_blob(scope: &str, digest: &[u8; 32]) -> Result<Vec<u8>> {
    use base64::Engine;

    let get = |part: &str| -> Result<String> {
        let account = recovery_blob_account(scope, digest, part);
        crate::credentials::vault_get_strict(RECOVERY_BLOB_SERVICE, &account)?
            .context("recovery blob part missing")
    };
    let length: usize = get("length")?.parse()?;
    anyhow::ensure!(
        length > 0 && length <= RECOVERY_BLOB_LIMIT,
        "invalid recovery blob size"
    );
    let mut bytes = Vec::with_capacity(length);
    for index in 0..length.div_ceil(RECOVERY_BLOB_CHUNK) {
        let encoded = get(&index.to_string())?;
        anyhow::ensure!(encoded.len() <= 960, "recovery blob chunk too large");
        let chunk = base64::engine::general_purpose::STANDARD.decode(encoded)?;
        anyhow::ensure!(
            chunk.len() == (length - bytes.len()).min(RECOVERY_BLOB_CHUNK),
            "invalid recovery blob chunk length"
        );
        bytes.extend_from_slice(&chunk);
    }
    anyhow::ensure!(
        bytes.len() == length && recovery_digest(&bytes) == *digest,
        "recovery blob digest mismatch"
    );
    Ok(bytes)
}

#[derive(Default)]
struct GlobalRecoveryLedger {
    generation: u64,
    entries: std::collections::BTreeMap<[u8; 32], [u8; 32]>,
    sharded: bool,
}

#[derive(Debug, Default, PartialEq, Eq)]
struct RecoveryIndexDifference {
    missing: usize,
    unindexed: usize,
    divergent: usize,
}

fn recovery_index_entry(
    id: &str,
    phase: &str,
    destination: Option<&str>,
    envelope: &str,
) -> Result<([u8; 32], [u8; 32])> {
    use sha2::{Digest, Sha256};

    let identity = Sha256::digest(id.as_bytes()).into();
    let bytes = serde_json::to_vec(&(id, phase, destination, envelope))?;
    Ok((identity, Sha256::digest(bytes).into()))
}

impl GlobalRecoveryLedger {
    #[cfg(test)]
    fn encode(&self) -> String {
        use base64::Engine;

        let mut bytes = self.generation.to_be_bytes().to_vec();
        for (id, proof) in &self.entries {
            bytes.extend_from_slice(id);
            bytes.extend_from_slice(proof);
        }
        base64::engine::general_purpose::STANDARD.encode(bytes)
    }

    fn decode(payload: &str) -> Result<Self> {
        use base64::Engine;

        anyhow::ensure!(payload.len() <= 704, "recovery index exceeds size limit");
        let bytes = base64::engine::general_purpose::STANDARD.decode(payload)?;
        anyhow::ensure!(
            bytes.len() >= 8 && (bytes.len() - 8) % 64 == 0,
            "invalid recovery index format"
        );
        let generation = u64::from_be_bytes(bytes[..8].try_into()?);
        let mut entries = std::collections::BTreeMap::new();
        for entry in bytes[8..].as_chunks::<64>().0 {
            let id: [u8; 32] = entry[..32].try_into()?;
            let proof: [u8; 32] = entry[32..].try_into()?;
            anyhow::ensure!(
                entries.insert(id, proof).is_none(),
                "duplicate recovery index ID"
            );
        }
        anyhow::ensure!(
            entries.len() <= RECOVERY_SHARD_SIZE,
            "legacy recovery index capacity exceeded"
        );
        Ok(Self {
            generation,
            entries,
            sharded: false,
        })
    }

    fn shard_bytes(&self) -> Vec<Vec<u8>> {
        let mut shards = Vec::new();
        for (index, (id, proof)) in self.entries.iter().enumerate() {
            if index % RECOVERY_SHARD_SIZE == 0 {
                shards.push(Vec::with_capacity(RECOVERY_SHARD_SIZE * 64));
            }
            let shard = shards.last_mut().expect("shard allocated");
            shard.extend_from_slice(id);
            shard.extend_from_slice(proof);
        }
        shards
    }

    fn manifest(generation: u64, shards: &[[u8; 32]]) -> Result<String> {
        use base64::Engine;
        anyhow::ensure!(
            shards.len() <= RECOVERY_INDEX_LIMIT / RECOVERY_SHARD_SIZE,
            "too many recovery shards"
        );
        let mut bytes = generation.to_be_bytes().to_vec();
        for digest in shards {
            bytes.extend_from_slice(digest);
        }
        Ok(format!(
            "v2:{}",
            base64::engine::general_purpose::STANDARD.encode(bytes)
        ))
    }

    fn parse_manifest(payload: &str) -> Result<(u64, Vec<[u8; 32]>)> {
        use base64::Engine;
        let encoded = payload
            .strip_prefix("v2:")
            .context("invalid recovery manifest version")?;
        anyhow::ensure!(encoded.len() <= 704, "recovery manifest too large");
        let bytes = base64::engine::general_purpose::STANDARD.decode(encoded)?;
        anyhow::ensure!(
            bytes.len() >= 8 && (bytes.len() - 8) % 32 == 0,
            "invalid recovery manifest"
        );
        let generation = u64::from_be_bytes(bytes[..8].try_into()?);
        let shards = bytes[8..].as_chunks::<32>().0.to_vec();
        anyhow::ensure!(
            shards.len() <= RECOVERY_INDEX_LIMIT / RECOVERY_SHARD_SIZE,
            "too many recovery shards"
        );
        Ok((generation, shards))
    }

    fn from_shards(generation: u64, shards: impl IntoIterator<Item = Vec<u8>>) -> Result<Self> {
        let mut ledger = Self {
            generation,
            sharded: true,
            ..Self::default()
        };
        let mut previous = None;
        for shard in shards {
            anyhow::ensure!(
                !shard.is_empty()
                    && shard.len() <= RECOVERY_SHARD_SIZE * 64
                    && shard.len() % 64 == 0,
                "invalid recovery shard"
            );
            for entry in shard.as_chunks::<64>().0 {
                let id: [u8; 32] = entry[..32].try_into()?;
                let proof: [u8; 32] = entry[32..].try_into()?;
                anyhow::ensure!(
                    previous.is_none_or(|last| last < id),
                    "unordered or duplicate recovery shard entry"
                );
                ledger.entries.insert(id, proof);
                previous = Some(id);
            }
        }
        anyhow::ensure!(
            ledger.entries.len() <= RECOVERY_INDEX_LIMIT,
            "recovery index capacity exceeded"
        );
        Ok(ledger)
    }

    fn store_record(
        db: &crate::db::Db,
        id: &str,
        phase: &str,
        destination: &str,
        envelope: &str,
    ) -> Result<[u8; 32]> {
        anyhow::ensure!(envelope.len() <= 65536, "recovery envelope too large");
        let bytes = serde_json::to_vec(&(id, phase, Some(destination), envelope))?;
        put_recovery_blob(&db.recovery_scope()?, &bytes)
    }

    fn differences(
        &self,
        rows: &[crate::db::DownloadRecoverySnapshot],
    ) -> Result<RecoveryIndexDifference> {
        let mut actual = std::collections::BTreeMap::new();
        for row in rows {
            let (id, proof) = recovery_index_entry(
                &row.id,
                &row.phase,
                row.destination_key.as_deref(),
                &row.payload,
            )?;
            anyhow::ensure!(actual.insert(id, proof).is_none(), "duplicate recovery row");
        }
        let mut difference = RecoveryIndexDifference::default();
        for (id, proof) in &self.entries {
            match actual.get(id) {
                None => difference.missing += 1,
                Some(current) if current != proof => difference.divergent += 1,
                Some(_) => {}
            }
        }
        difference.unindexed = actual
            .keys()
            .filter(|id| !self.entries.contains_key(*id))
            .count();
        Ok(difference)
    }

    // Only authenticated metadata is repaired. No recorded filesystem path
    // is opened or changed by this method. Caller owns the interprocess lock.
    fn reconcile_metadata(&mut self, db: &crate::db::Db) -> Result<()> {
        if !self.sharded {
            return self.check_rows(db);
        }
        let rows = db.download_recovery_snapshot()?;
        let difference = self.differences(&rows)?;
        anyhow::ensure!(
            difference.unindexed == 0,
            "unindexed SQLite recovery record requires review"
        );
        let scope = db.recovery_scope()?;
        for (identity, proof) in self.entries.clone() {
            let bytes = get_recovery_blob(&scope, &proof)?;
            let (id, phase, destination, envelope): (String, String, Option<String>, String) =
                serde_json::from_slice(&bytes)?;
            let destination = destination.context("redundant reservation missing")?;
            let (record_id, record_proof) =
                recovery_index_entry(&id, &phase, Some(&destination), &envelope)?;
            anyhow::ensure!(
                record_id == identity && record_proof == proof,
                "redundant recovery commitment mismatch"
            );
            let signed: AuthenticatedRecovery = serde_json::from_str(&envelope)?;
            let record: RecoveryInspectionRecord = serde_json::from_str(&signed.payload)?;
            anyhow::ensure!(
                record.matches_reservation(Some(&destination)),
                "redundant recovery reservation mismatch"
            );
            let phase = RecoveryPhase::parse(&phase).context("invalid redundant recovery phase")?;
            let row = rows.iter().find(|row| row.id == id);
            let exact = row.is_some_and(|row| {
                row.phase == phase.as_str()
                    && row.payload == envelope
                    && row.destination_key.as_deref() == Some(destination.as_str())
            });
            let previous = signed.previous_envelope(&id)?;
            if let Some(row) = row.filter(|_| !exact) {
                anyhow::ensure!(
                    previous.as_ref().is_some_and(|(from, old)| {
                        row.phase == from.as_str()
                            && row.payload == *old
                            && row.destination_key.as_deref() == Some(destination.as_str())
                    }),
                    "divergent recovery record is not the immediate signed predecessor"
                );
            }
            let anchor = signed.reconciliation_anchor(&id, &envelope, phase, previous.as_ref())?;
            if anchor == RecoveryAnchorState::Terminal {
                anyhow::ensure!(
                    phase == RecoveryPhase::CleanupPending && (row.is_none() || exact),
                    "unexpected terminal recovery record"
                );
                register_recovery_gc_key(&scope, signed.key_id)?;
                if exact {
                    db.download_recovery_finish_checked(&id, &destination, &envelope)?;
                }
                self.update(
                    db,
                    &id,
                    Some((phase.as_str(), &destination, &envelope)),
                    None,
                )?;
                // The authenticated key GC queue survives a crash after root retirement.
                continue;
            }
            if anchor == RecoveryAnchorState::NeedsAdvance {
                signed.store_anchor(&id, &envelope, false)?;
            }
            if row.is_none() {
                db.download_recovery_restore_missing(&id, phase, &destination, &envelope)?;
            } else if !exact {
                let (from, old) = previous.context("missing signed predecessor")?;
                db.download_recovery_transition_checked(
                    &id,
                    from,
                    phase,
                    &destination,
                    &old,
                    &envelope,
                )?;
            }
        }
        Ok(())
    }

    fn check_rows(&self, db: &crate::db::Db) -> Result<()> {
        let difference = self.differences(&db.download_recovery_snapshot()?)?;
        if difference != RecoveryIndexDifference::default() {
            tracing::warn!(
                missing = difference.missing,
                unindexed = difference.unindexed,
                divergent = difference.divergent,
                "global recovery index requires manual review"
            );
            anyhow::bail!("global recovery index does not match SQLite");
        }
        Ok(())
    }

    fn with_key<T>(encoded: &str, action: impl FnOnce(&[u8; 32]) -> Result<T>) -> Result<T> {
        use base64::Engine;

        let mut decoded = base64::engine::general_purpose::STANDARD.decode(encoded)?;
        let result = match <&[u8; 32]>::try_from(decoded.as_slice()) {
            Ok(key) => action(key),
            Err(_) => Err(anyhow::anyhow!("invalid recovery index key")),
        };
        decoded.fill(0);
        result
    }

    // Caller holds the application data directory's interprocess lock.
    fn load(db: &crate::db::Db, initialize: bool) -> Result<Self> {
        use base64::Engine;
        use rand::RngCore;

        let scope = db.recovery_scope()?;
        let key = crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, &scope)?;
        let stored = crate::credentials::vault_get_strict(RECOVERY_INDEX_SERVICE, &scope)?;
        match (key, stored) {
            (Some(key), Some(stored)) => {
                let mut ledger = Self::with_key(&key, |key| {
                    anyhow::ensure!(stored.len() <= 1200, "recovery index envelope too large");
                    let signed: AuthenticatedRecovery = serde_json::from_str(&stored)?;
                    signed.verify(&scope, key)?;
                    anyhow::ensure!(
                        signed.key_id == Uuid::nil() && signed.phases.is_empty(),
                        "invalid global recovery index envelope"
                    );
                    if signed.payload.starts_with("v2:") {
                        let (generation, digests) = Self::parse_manifest(&signed.payload)?;
                        let shards = digests
                            .iter()
                            .map(|digest| get_recovery_blob(&scope, digest))
                            .collect::<Result<Vec<_>>>()?;
                        let ledger = Self::from_shards(generation, shards.clone())?;
                        anyhow::ensure!(
                            ledger.shard_bytes() == shards,
                            "noncanonical recovery shards"
                        );
                        Ok(ledger)
                    } else {
                        Self::decode(&signed.payload)
                    }
                })?;
                db.recovery_vault_initialized(true)?;
                if initialize && !ledger.sharded {
                    ledger.check_rows(db)?;
                    for row in db.download_recovery_snapshot()? {
                        let destination = row
                            .destination_key
                            .as_deref()
                            .context("legacy reservation missing")?;
                        let signed: AuthenticatedRecovery = serde_json::from_str(&row.payload)?;
                        anyhow::ensure!(
                            signed.authenticate(&row.id, &row.payload)?.as_str() == row.phase,
                            "legacy recovery record is not authenticated"
                        );
                        Self::store_record(db, &row.id, &row.phase, destination, &row.payload)?;
                    }
                    ledger.sharded = true;
                    ledger.save(db)?;
                }
                Ok(ledger)
            }
            (None, None) => {
                anyhow::ensure!(
                    !db.recovery_vault_initialized(false)?,
                    "initialized recovery vault is missing; refusing reinitialization"
                );
                let ledger = Self {
                    sharded: true,
                    ..Self::default()
                };
                ledger.check_rows(db)?;
                if initialize {
                    // Persist the denial marker before creating any vault state.
                    // Interrupted bootstrap is intentionally manual-review only.
                    db.recovery_vault_initialized(true)?;
                    recovery_fault_checkpoint("vault_bootstrap_marked");
                    let mut key = [0u8; 32];
                    rand::rngs::OsRng.try_fill_bytes(&mut key)?;
                    let encoded = base64::engine::general_purpose::STANDARD.encode(key);
                    key.fill(0);
                    crate::credentials::vault_set(RECOVERY_INDEX_KEY_SERVICE, &scope, &encoded)?;
                    let readback =
                        crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, &scope)?;
                    anyhow::ensure!(
                        readback.as_deref() == Some(encoded.as_str()),
                        "recovery index key readback mismatch"
                    );
                    ledger.save(db)?;
                }
                Ok(ledger)
            }
            _ => anyhow::bail!("incomplete global recovery index; manual review required"),
        }
    }

    fn save(&self, db: &crate::db::Db) -> Result<()> {
        self.save_with_reuse(db, &std::collections::BTreeSet::new())
    }

    fn save_with_reuse(
        &self,
        db: &crate::db::Db,
        reusable: &std::collections::BTreeSet<[u8; 32]>,
    ) -> Result<()> {
        let scope = db.recovery_scope()?;
        let key = crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, &scope)?
            .context("recovery index key unavailable")?;
        Self::with_key(&key, |key| {
            let shards = self
                .shard_bytes()
                .iter()
                .map(|bytes| {
                    let digest = recovery_digest(bytes);
                    if reusable.contains(&digest) {
                        anyhow::ensure!(
                            get_recovery_blob(&scope, &digest)? == *bytes,
                            "reused recovery shard changed"
                        );
                        return Ok(digest);
                    }
                    put_recovery_blob(&scope, bytes)
                })
                .collect::<Result<Vec<_>>>()?;
            let manifest = Self::manifest(self.generation, &shards)?;
            let signed = AuthenticatedRecovery::seal(&scope, manifest, Uuid::nil(), key)?;
            let encoded = serde_json::to_string(&signed)?;
            anyhow::ensure!(encoded.len() <= 1200, "recovery index envelope too large");
            crate::credentials::vault_set(RECOVERY_INDEX_SERVICE, &scope, &encoded)?;
            recovery_fault_checkpoint("manifest_written");
            let readback = crate::credentials::vault_get_strict(RECOVERY_INDEX_SERVICE, &scope)?;
            anyhow::ensure!(
                readback.as_deref() == Some(encoded.as_str()),
                "recovery index readback mismatch"
            );
            Ok(())
        })
    }

    // Caller owns the interprocess lock. Always reload the authoritative
    // manifest before collecting; never trust the caller's cached ledger.
    fn collect_garbage(db: &crate::db::Db) -> Result<()> {
        let scope = db.recovery_scope()?;
        let Some(key) = crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, &scope)?
        else {
            return Ok(());
        };
        Self::with_key(&key, |key| {
            let pages = (0..RECOVERY_GC_PAGES)
                .map(|page| read_gc_page(&scope, page, key))
                .collect::<Result<Vec<_>>>()?;
            let keys = read_gc_keys(&scope, key)?;
            if pages.iter().all(RecoveryGcPage::is_empty) && keys.is_empty() {
                return Ok(());
            }
            let ledger = Self::load(db, false)?;
            if !ledger.sharded {
                // Interrupted v1 migration may have staged candidates. Do
                // not delete them until v2 is published and authenticated.
                return Ok(());
            }
            ledger.check_rows(db)?;
            let mut live = std::collections::BTreeSet::new();
            let mut live_keys = RecoveryGcKeys::new();
            for shard in ledger.shard_bytes() {
                live.insert(recovery_digest(&shard));
            }
            for row in db.download_recovery_snapshot()? {
                let signed: AuthenticatedRecovery = serde_json::from_str(&row.payload)?;
                anyhow::ensure!(
                    signed.authenticate(&row.id, &row.payload)?.as_str() == row.phase,
                    "recovery GC requires current authenticated rows"
                );
                live_keys.insert(signed.key_id);
                let (_, proof) = recovery_index_entry(
                    &row.id,
                    &row.phase,
                    row.destination_key.as_deref(),
                    &row.payload,
                )?;
                get_recovery_blob(&scope, &proof)?;
                live.insert(proof);
            }
            for (page, entries) in pages.iter().enumerate() {
                if entries.is_empty() {
                    continue;
                }
                for (digest, length) in entries {
                    if !live.contains(digest) {
                        delete_recovery_blob(&scope, digest, *length)?;
                    }
                }
                // Live candidates are forgotten, not deleted. A later
                // supersession registers their digest again before publication.
                write_gc_page(&scope, page, key, &RecoveryGcPage::new())?;
                recovery_fault_checkpoint("gc_page_cleared");
            }
            if !keys.is_empty() {
                for id in keys {
                    if !live_keys.contains(&id) {
                        let account = id.to_string();
                        // Revoke the authentication key before the anchor.
                        // An interrupted deletion cannot reauthorize a row.
                        crate::credentials::vault_delete_strict(RECOVERY_VAULT_SERVICE, &account)?;
                        recovery_fault_checkpoint("gc_key_removed");
                        crate::credentials::vault_delete_strict(RECOVERY_ANCHOR_SERVICE, &account)?;
                        recovery_fault_checkpoint("gc_anchor_removed");
                    }
                }
                write_gc_keys(&scope, key, &RecoveryGcKeys::new())?;
            }
            Ok(())
        })
    }

    fn update(
        &mut self,
        db: &crate::db::Db,
        id: &str,
        old: Option<(&str, &str, &str)>,
        next: Option<(&str, &str, &str)>,
    ) -> Result<()> {
        use sha2::{Digest, Sha256};

        let identity: [u8; 32] = Sha256::digest(id.as_bytes()).into();
        let expected = old
            .map(|(phase, destination, payload)| {
                recovery_index_entry(id, phase, Some(destination), payload).map(|(_, proof)| proof)
            })
            .transpose()?;
        anyhow::ensure!(
            self.entries.get(&identity).copied() == expected,
            "stale global recovery index entry"
        );
        let scope = db.recovery_scope()?;
        let previous_shards = if self.sharded {
            self.shard_bytes()
        } else {
            Vec::new()
        };
        if self.sharded {
            if let Some(proof) = expected {
                let bytes = get_recovery_blob(&scope, &proof)?;
                register_recovery_gc(&scope, proof, bytes.len())?;
            }
        }
        if let Some((phase, destination, payload)) = next {
            let (_, proof) = recovery_index_entry(id, phase, Some(destination), payload)?;
            anyhow::ensure!(
                expected.is_some() || self.entries.len() < RECOVERY_INDEX_LIMIT,
                "global recovery index capacity reached"
            );
            let stored = Self::store_record(db, id, phase, destination, payload)?;
            anyhow::ensure!(stored == proof, "redundant recovery record digest mismatch");
            self.entries.insert(identity, proof);
        } else {
            self.entries.remove(&identity);
        }
        let current: std::collections::BTreeSet<_> = self
            .shard_bytes()
            .iter()
            .map(|shard| recovery_digest(shard))
            .collect();
        let mut reusable = std::collections::BTreeSet::new();
        for shard in previous_shards {
            let digest = recovery_digest(&shard);
            if current.contains(&digest) {
                reusable.insert(digest);
            } else {
                // Register superseded root references before publication.
                register_recovery_gc(&scope, digest, shard.len())?;
            }
        }
        self.generation = self
            .generation
            .checked_add(1)
            .context("recovery index overflow")?;
        self.save_with_reuse(db, &reusable)
    }
}

/// Compact commitment to the exact SQLite envelope, including its phase chain.
/// A terminal commitment cannot match any live envelope.
fn recovery_anchor_payload(envelope: &str, terminal: bool) -> String {
    use sha2::{Digest, Sha256};

    let kind = if terminal { "terminal" } else { "active" };
    format!(
        "ghostftp-recovery-anchor-v1:{kind}:{:x}",
        Sha256::digest(envelope.as_bytes())
    )
}

/// Verify the independent vault anchor using the transaction authentication key.
fn verify_recovery_anchor(
    anchor: &AuthenticatedRecovery,
    id: &str,
    key_id: Uuid,
    envelope: &str,
    key: &[u8; 32],
) -> Result<()> {
    anchor.verify(id, key)?;
    anyhow::ensure!(
        anchor.key_id == key_id
            && anchor.phases.is_empty()
            && anchor.payload == recovery_anchor_payload(envelope, false),
        "recovery anchor does not match current envelope"
    );
    Ok(())
}

/// Authenticates immutable intent bytes and a chained sequence of phase
/// proofs. The SQLite phase must match the authenticated chain tip.
/// The independent vault anchor detects replay of a superseded SQL envelope.
/// Restoring both SQLite and the corresponding vault state remains out of scope.
#[derive(serde::Serialize, serde::Deserialize)]
struct AuthenticatedRecovery {
    key_id: Uuid,
    nonce: [u8; 12],
    tag: Vec<u8>,
    payload: String,
    #[serde(default)]
    phases: Vec<RecoveryPhaseProof>,
}

#[derive(serde::Serialize, serde::Deserialize)]
struct RecoveryPhaseProof {
    phase: String,
    sequence: u64,
    previous_tag: Vec<u8>,
    nonce: [u8; 12],
    tag: Vec<u8>,
}

#[derive(Debug, PartialEq, Eq)]
enum RecoveryAnchorState {
    Current,
    NeedsAdvance,
    Terminal,
}

impl AuthenticatedRecovery {
    fn seal(id: &str, payload: String, key_id: Uuid, key: &[u8; 32]) -> Result<Self> {
        use aes_gcm::aead::{Aead, KeyInit, Payload};
        use aes_gcm::{Aes256Gcm, Key, Nonce};
        use rand::RngCore;

        let mut nonce = [0u8; 12];
        rand::rngs::OsRng.try_fill_bytes(&mut nonce)?;
        let aad = serde_json::to_vec(&(RECOVERY_VAULT_SERVICE, id, key_id, &payload))?;
        let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(key));
        let tag = cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                Payload {
                    msg: &[],
                    aad: &aad,
                },
            )
            .map_err(|_| anyhow::anyhow!("recovery authentication failed"))?;
        Ok(Self {
            key_id,
            nonce,
            tag,
            payload,
            phases: Vec::new(),
        })
    }

    fn verify(&self, id: &str, key: &[u8; 32]) -> Result<()> {
        use aes_gcm::aead::{Aead, KeyInit, Payload};
        use aes_gcm::{Aes256Gcm, Key, Nonce};

        anyhow::ensure!(self.tag.len() == 16, "invalid recovery authentication tag");
        let aad = serde_json::to_vec(&(RECOVERY_VAULT_SERVICE, id, self.key_id, &self.payload))?;
        let cipher = Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(key));
        cipher
            .decrypt(
                Nonce::from_slice(&self.nonce),
                Payload {
                    msg: &self.tag,
                    aad: &aad,
                },
            )
            .map_err(|_| anyhow::anyhow!("recovery authentication failed"))?;
        Ok(())
    }

    fn phase_payload(&self, phase: &str, sequence: u64, previous: &[u8]) -> Result<String> {
        Ok(serde_json::to_string(&(
            "ghostftp-recovery-phase-v1",
            &self.tag,
            phase,
            sequence,
            previous,
        ))?)
    }

    fn append_phase(&mut self, id: &str, phase: RecoveryPhase, key: &[u8; 32]) -> Result<()> {
        let previous = self
            .phases
            .last()
            .map_or(self.tag.as_slice(), |proof| proof.tag.as_slice())
            .to_vec();
        let sequence = u64::try_from(self.phases.len())?;
        let payload = self.phase_payload(phase.as_str(), sequence, &previous)?;
        let signed = Self::seal(id, payload, self.key_id, key)?;
        self.phases.push(RecoveryPhaseProof {
            phase: phase.as_str().to_string(),
            sequence,
            previous_tag: previous,
            nonce: signed.nonce,
            tag: signed.tag,
        });
        Ok(())
    }

    fn verify_phases(&self, id: &str, key: &[u8; 32]) -> Result<RecoveryPhase> {
        self.verify(id, key)?;
        anyhow::ensure!(
            !self.phases.is_empty() && self.phases.len() <= 4,
            "missing or invalid recovery phase chain"
        );
        let mut previous_tag = self.tag.as_slice();
        let mut previous_phase: Option<RecoveryPhase> = None;
        for (index, proof) in self.phases.iter().enumerate() {
            let phase = RecoveryPhase::parse(&proof.phase).context("unknown recovery phase")?;
            anyhow::ensure!(
                proof.sequence == u64::try_from(index)?
                    && proof.previous_tag.as_slice() == previous_tag
                    && previous_phase.map_or(phase == RecoveryPhase::Prepared, |from| {
                        from.permits(phase)
                    }),
                "invalid recovery phase sequence"
            );
            let payload = self.phase_payload(&proof.phase, proof.sequence, previous_tag)?;
            Self {
                key_id: self.key_id,
                nonce: proof.nonce,
                tag: proof.tag.clone(),
                payload,
                phases: Vec::new(),
            }
            .verify(id, key)?;
            previous_tag = &proof.tag;
            previous_phase = Some(phase);
        }
        previous_phase.context("missing recovery phase")
    }

    fn advance_signed(
        &mut self,
        id: &str,
        from: RecoveryPhase,
        to: RecoveryPhase,
        key: &[u8; 32],
    ) -> Result<()> {
        anyhow::ensure!(
            self.verify_phases(id, key)? == from && from.permits(to),
            "invalid authenticated recovery transition"
        );
        self.append_phase(id, to, key)
    }

    fn create(scope: &str, id: &str, payload: String) -> Result<Self> {
        use base64::Engine;
        use rand::RngCore;

        let key_id = Uuid::new_v4();
        register_recovery_gc_key(scope, key_id)?;
        let account = key_id.to_string();
        let mut key = [0u8; 32];
        rand::rngs::OsRng.try_fill_bytes(&mut key)?;
        let result = (|| {
            let mut signed = Self::seal(id, payload, key_id, &key)?;
            signed.append_phase(id, RecoveryPhase::Prepared, &key)?;
            let encoded = base64::engine::general_purpose::STANDARD.encode(key);
            crate::credentials::vault_set(RECOVERY_VAULT_SERVICE, &account, &encoded)?;
            recovery_fault_checkpoint("transaction_key_written");
            let readback = crate::credentials::vault_get_strict(RECOVERY_VAULT_SERVICE, &account)?
                .context("recovery authentication key is missing after creation")?;
            anyhow::ensure!(readback == encoded, "recovery key readback mismatch");
            Ok(signed)
        })();
        key.fill(0);
        result
    }

    fn with_key<T>(key_id: Uuid, action: impl FnOnce(&[u8; 32]) -> Result<T>) -> Result<T> {
        use base64::Engine;

        let encoded =
            crate::credentials::vault_get_strict(RECOVERY_VAULT_SERVICE, &key_id.to_string())?
                .context("recovery authentication key is unavailable")?;
        let mut decoded = base64::engine::general_purpose::STANDARD.decode(encoded)?;
        let result = match <&[u8; 32]>::try_from(decoded.as_slice()) {
            Ok(key) => action(key),
            Err(_) => Err(anyhow::anyhow!(
                "invalid recovery authentication key length"
            )),
        };
        decoded.fill(0);
        result
    }

    fn authenticate(&self, id: &str, envelope: &str) -> Result<RecoveryPhase> {
        Self::with_key(self.key_id, |key| {
            let phase = self.verify_phases(id, key)?;
            let stored = crate::credentials::vault_get_strict(
                RECOVERY_ANCHOR_SERVICE,
                &self.key_id.to_string(),
            )?
            .context("recovery anchor is unavailable")?;
            let anchor: Self = serde_json::from_str(&stored)?;
            verify_recovery_anchor(&anchor, id, self.key_id, envelope, key)?;
            Ok(phase)
        })
    }

    fn previous_envelope(&self, id: &str) -> Result<Option<(RecoveryPhase, String)>> {
        Self::with_key(self.key_id, |key| {
            self.verify_phases(id, key)?;
            if self.phases.len() <= 1 {
                return Ok(None);
            }
            let mut previous: Self = serde_json::from_str(&serde_json::to_string(self)?)?;
            previous.phases.pop();
            let phase = previous.verify_phases(id, key)?;
            Ok(Some((phase, serde_json::to_string(&previous)?)))
        })
    }

    fn classify_anchor(
        &self,
        id: &str,
        envelope: &str,
        phase: RecoveryPhase,
        previous: Option<&(RecoveryPhase, String)>,
        anchor: Option<&Self>,
        key: &[u8; 32],
    ) -> Result<RecoveryAnchorState> {
        anyhow::ensure!(
            self.verify_phases(id, key)? == phase,
            "redundant signed phase mismatch"
        );
        let Some(anchor) = anchor else {
            anyhow::ensure!(
                phase == RecoveryPhase::Prepared,
                "missing non-initial recovery anchor"
            );
            return Ok(RecoveryAnchorState::NeedsAdvance);
        };
        anchor.verify(id, key)?;
        anyhow::ensure!(
            anchor.key_id == self.key_id && anchor.phases.is_empty(),
            "invalid recovery anchor"
        );
        if anchor.payload == recovery_anchor_payload(envelope, true) {
            anyhow::ensure!(
                phase == RecoveryPhase::CleanupPending,
                "invalid terminal recovery phase"
            );
            return Ok(RecoveryAnchorState::Terminal);
        }
        if verify_recovery_anchor(anchor, id, self.key_id, envelope, key).is_ok() {
            return Ok(RecoveryAnchorState::Current);
        }
        if let Some((from, old)) = previous {
            anyhow::ensure!(from.permits(phase), "invalid recovery predecessor");
            verify_recovery_anchor(anchor, id, self.key_id, old, key)?;
            return Ok(RecoveryAnchorState::NeedsAdvance);
        }
        anyhow::bail!("recovery anchor cannot be reconciled")
    }

    fn reconciliation_anchor(
        &self,
        id: &str,
        envelope: &str,
        phase: RecoveryPhase,
        previous: Option<&(RecoveryPhase, String)>,
    ) -> Result<RecoveryAnchorState> {
        Self::with_key(self.key_id, |key| {
            let stored = crate::credentials::vault_get_strict(
                RECOVERY_ANCHOR_SERVICE,
                &self.key_id.to_string(),
            )?;
            let anchor = stored
                .as_deref()
                .map(serde_json::from_str::<Self>)
                .transpose()?;
            self.classify_anchor(id, envelope, phase, previous, anchor.as_ref(), key)
        })
    }

    fn store_anchor(&self, id: &str, envelope: &str, terminal: bool) -> Result<()> {
        Self::with_key(self.key_id, |key| {
            self.verify_phases(id, key)?;
            let anchor = Self::seal(
                id,
                recovery_anchor_payload(envelope, terminal),
                self.key_id,
                key,
            )?;
            let encoded = serde_json::to_string(&anchor)?;
            let account = self.key_id.to_string();
            crate::credentials::vault_set(RECOVERY_ANCHOR_SERVICE, &account, &encoded)?;
            let readback = crate::credentials::vault_get_strict(RECOVERY_ANCHOR_SERVICE, &account)?;
            anyhow::ensure!(
                readback.as_deref() == Some(encoded.as_str()),
                "recovery anchor readback mismatch"
            );
            Ok(())
        })
    }

    fn advance(&mut self, id: &str, from: RecoveryPhase, to: RecoveryPhase) -> Result<()> {
        Self::with_key(self.key_id, |key| self.advance_signed(id, from, to, key))
    }
}

#[derive(Debug, PartialEq, Eq)]
enum RecoveryDecision {
    ManualReview,
    InspectOnly,
    RollbackCandidate,
    CleanupCandidate,
}

/// Candidates are advisory only: authentication of the journal does not
/// prove current file contents or exclude replay, identifier reuse or races.
fn recovery_decision(authenticated: bool, observed: RecoveryObservation) -> RecoveryDecision {
    if !authenticated {
        return RecoveryDecision::ManualReview;
    }
    match observed {
        RecoveryObservation::BackupPresent | RecoveryObservation::BackupWithoutStage => {
            RecoveryDecision::RollbackCandidate
        }
        RecoveryObservation::PublishedWithBackup => RecoveryDecision::CleanupCandidate,
        RecoveryObservation::OriginalPresent
        | RecoveryObservation::OriginalWithoutStage
        | RecoveryObservation::PublishedWithoutBackup => RecoveryDecision::InspectOnly,
        RecoveryObservation::Ambiguous | RecoveryObservation::Unavailable => {
            RecoveryDecision::ManualReview
        }
    }
}

struct RecoveryJournal {
    db: std::sync::Arc<crate::db::Db>,
    id: String,
    parent: PathBuf,
    stage: PathBuf,
    target: PathBuf,
    backup: PathBuf,
    auth_key: std::sync::Mutex<Option<Uuid>>,
    persisted: std::sync::Mutex<Option<(String, String)>>,
}

fn recovery_destination_key(parent_identity: (u64, u64), target: &Path) -> Result<String> {
    let leaf = target
        .file_name()
        .context("recovery target has no basename")?;
    // Case folding is conservative on Windows; aliases and filesystem-specific
    // Unicode case equivalence still need native validation.
    #[cfg(unix)]
    let name = {
        use std::os::unix::ffi::OsStrExt;
        leaf.as_bytes()
            .iter()
            .map(|b| format!("{b:02x}"))
            .collect::<String>()
    };
    #[cfg(windows)]
    let name = leaf
        .to_str()
        .context("unsupported recovery filename encoding")?
        .to_uppercase();
    #[cfg(not(any(unix, windows)))]
    let name = leaf.to_string_lossy().into_owned();
    Ok(format!(
        "{:016x}:{:016x}:{name}",
        parent_identity.0, parent_identity.1
    ))
}

impl RecoveryJournal {
    fn begin(
        &self,
        parent_identity: (u64, u64),
        stage_identity: (u64, u64, u64),
        original_identity: (u64, u64, u64),
        stage_sha256: String,
        original_sha256: String,
    ) -> Result<()> {
        let record = RecoveryRecord {
            schema: 2,
            parent: &self.parent,
            stage: &self.stage,
            target: &self.target,
            backup: &self.backup,
            parent_identity,
            stage_identity,
            original_identity,
            stage_sha256,
            original_sha256,
        };
        let destination_key = recovery_destination_key(parent_identity, &self.target)?;
        let payload = serde_json::to_string(&record)?;
        let mut ledger = GlobalRecoveryLedger::load(&self.db, true)?;
        ledger.check_rows(&self.db)?;
        GlobalRecoveryLedger::collect_garbage(&self.db)?;
        let signed = AuthenticatedRecovery::create(&self.db.recovery_scope()?, &self.id, payload)?;
        let stored = serde_json::to_string(&signed)?;
        // Index first, transaction anchor second, SQLite third. Any interrupted
        // write leaves a detectable mismatch before filesystem mutation.
        ledger.update(
            &self.db,
            &self.id,
            None,
            Some((RecoveryPhase::Prepared.as_str(), &destination_key, &stored)),
        )?;
        recovery_fault_checkpoint("prepared_manifest_committed");
        signed.store_anchor(&self.id, &stored, false)?;
        recovery_fault_checkpoint("prepared_anchor_written");
        *self.auth_key.lock().unwrap_or_else(|e| e.into_inner()) = Some(signed.key_id);
        let mut persisted = self
            .persisted
            .lock()
            .map_err(|_| anyhow::anyhow!("recovery journal lock poisoned"))?;
        self.db
            .download_recovery_begin(&self.id, &stored, &destination_key)
            .context("persist recovery intent before backup")?;
        *persisted = Some((destination_key, stored));
        recovery_fault_checkpoint("prepared_sql_committed");
        if GlobalRecoveryLedger::collect_garbage(&self.db).is_err() {
            tracing::warn!("verified recovery garbage collection deferred");
        }
        Ok(())
    }

    fn advance(&self, from: RecoveryPhase, to: RecoveryPhase) -> Result<()> {
        let mut persisted = self
            .persisted
            .lock()
            .map_err(|_| anyhow::anyhow!("recovery journal lock poisoned"))?;
        let (destination, previous) = persisted.as_mut().context("missing recovery intent")?;
        let auth_key = self
            .auth_key
            .lock()
            .map_err(|_| anyhow::anyhow!("recovery key lock poisoned"))?;
        self.db
            .download_recovery_check_current(&self.id, from, destination, previous)?;
        let mut ledger = GlobalRecoveryLedger::load(&self.db, false)?;
        ledger.check_rows(&self.db)?;
        let next = if let Some(key_id) = *auth_key {
            let mut signed: AuthenticatedRecovery = serde_json::from_str(previous)?;
            anyhow::ensure!(signed.key_id == key_id, "recovery key changed");
            anyhow::ensure!(
                signed.authenticate(&self.id, previous)? == from,
                "recovery anchor phase mismatch"
            );
            signed.advance(&self.id, from, to)?;
            let next = serde_json::to_string(&signed)?;
            // Vault first, SQL CAS second. A failed CAS leaves an intentional
            // mismatch for manual review; never move the anchor backward.
            ledger.update(
                &self.db,
                &self.id,
                Some((from.as_str(), destination, previous)),
                Some((to.as_str(), destination, &next)),
            )?;
            recovery_fault_checkpoint("phase_manifest_committed");
            signed.store_anchor(&self.id, &next, false)?;
            recovery_fault_checkpoint("phase_anchor_written");
            next
        } else {
            anyhow::bail!("unsigned recovery requires manual review");
        };
        self.db.download_recovery_transition_checked(
            &self.id,
            from,
            to,
            destination,
            previous,
            &next,
        )?;
        *previous = next;
        recovery_fault_checkpoint("phase_sql_committed");
        if GlobalRecoveryLedger::collect_garbage(&self.db).is_err() {
            tracing::warn!("verified recovery garbage collection deferred");
        }
        Ok(())
    }

    /// Only a live, anchored transaction may authorize backup restoration
    /// or deletion. Verify both stores before reading any recorded content.
    #[cfg(any(target_os = "linux", windows))]
    fn verify_before_mutation(
        &self,
        parent: &std::fs::File,
        phase: RecoveryPhase,
        published: bool,
    ) -> Result<()> {
        let persisted = self
            .persisted
            .lock()
            .map_err(|_| anyhow::anyhow!("recovery journal lock poisoned"))?;
        let (destination, envelope) = persisted.as_ref().context("missing recovery intent")?;
        let signed: AuthenticatedRecovery =
            serde_json::from_str(envelope).context("unsigned recovery requires manual review")?;
        anyhow::ensure!(
            signed.authenticate(&self.id, envelope)? == phase,
            "recovery anchor phase mismatch"
        );
        self.db
            .download_recovery_check_current(&self.id, phase, destination, envelope)?;
        GlobalRecoveryLedger::load(&self.db, false)?.check_rows(&self.db)?;
        let record: RecoveryInspectionRecord = serde_json::from_str(&signed.payload)?;
        anyhow::ensure!(
            record.matches_reservation(Some(destination.as_str())),
            "recovery reservation mismatch"
        );
        #[cfg(target_os = "linux")]
        let parent_identity = {
            use std::os::unix::fs::MetadataExt;
            let metadata = parent.metadata()?;
            (metadata.dev(), metadata.ino())
        };
        #[cfg(windows)]
        let parent_identity = {
            let (volume, index) = windows_file_identity(parent)?;
            (u64::from(volume), index)
        };
        anyhow::ensure!(
            parent_identity == record.parent_identity,
            "recovery parent identity changed"
        );
        if published {
            anyhow::ensure!(
                record.inspect_entry(parent, &record.target) == EntryObservation::Stage,
                "published download content changed; backup retained"
            );
        }
        anyhow::ensure!(
            record.inspect_entry(parent, &record.backup) == EntryObservation::Original,
            "recovery backup content changed or is unavailable"
        );
        // Content reads may be long; do not act on an outdated SQL/vault
        // observation. These checks still do not lock external writers.
        anyhow::ensure!(
            signed.authenticate(&self.id, envelope)? == phase,
            "recovery anchor phase mismatch"
        );
        self.db
            .download_recovery_check_current(&self.id, phase, destination, envelope)?;
        GlobalRecoveryLedger::load(&self.db, false)?.check_rows(&self.db)?;
        Ok(())
    }

    fn finish(&self) {
        let Ok(persisted) = self.persisted.lock() else {
            return;
        };
        let Some((destination, payload)) = persisted.as_ref() else {
            return;
        };
        let Ok(mut ledger) = GlobalRecoveryLedger::load(&self.db, false) else {
            return;
        };
        if ledger.check_rows(&self.db).is_err() {
            return;
        }
        // Tombstone first: replay of a completed row must fail even if
        // credential deletion is interrupted or the SQL commit fails.
        let terminal = (|| -> Result<()> {
            let signed: AuthenticatedRecovery = serde_json::from_str(payload)?;
            anyhow::ensure!(
                signed.authenticate(&self.id, payload)? == RecoveryPhase::CleanupPending,
                "recovery anchor phase mismatch"
            );
            self.db.download_recovery_check_current(
                &self.id,
                RecoveryPhase::CleanupPending,
                destination,
                payload,
            )?;
            register_recovery_gc_key(&self.db.recovery_scope()?, signed.key_id)?;
            signed.store_anchor(&self.id, payload, true)
        })();
        if terminal.is_err() {
            tracing::warn!("completed recovery anchor could not be retired");
            return;
        }
        recovery_fault_checkpoint("terminal_anchor_written");
        if self
            .db
            .download_recovery_finish_checked(&self.id, destination, payload)
            .is_err()
        {
            tracing::warn!("completed download recovery intent could not be cleared");
            return;
        }
        recovery_fault_checkpoint("terminal_sql_removed");
        if ledger
            .update(
                &self.db,
                &self.id,
                Some((RecoveryPhase::CleanupPending.as_str(), destination, payload)),
                None,
            )
            .is_err()
        {
            tracing::warn!("completed transaction remains in global recovery index");
            return;
        }
        recovery_fault_checkpoint("terminal_manifest_written");
        if GlobalRecoveryLedger::collect_garbage(&self.db).is_err() {
            tracing::warn!("verified recovery garbage collection deferred");
        }
        // Key and anchor deletion is handled by the resumable collector.
        *self.auth_key.lock().unwrap_or_else(|e| e.into_inner()) = None;
    }
}

impl DownloadParentGuard {
    #[cfg(test)]
    pub(super) fn open(target: &Path) -> Result<Self> {
        Self::with_db(target, None)
    }

    pub(super) fn with_db(
        target: &Path,
        recovery_db: Option<std::sync::Arc<crate::db::Db>>,
    ) -> Result<Self> {
        let path = target.parent().context("download target has no parent")?;
        let path = if path.as_os_str().is_empty() {
            Path::new(".")
        } else {
            path
        };
        Ok(Self {
            path: path.to_path_buf(),
            handle: std::sync::Arc::new(open_download_directory(path)?),
            recovery_db,
        })
    }

    fn recovery_journal(
        &self,
        staging: &Path,
        target: &Path,
        id: &str,
    ) -> Result<Option<RecoveryJournal>> {
        let Some(db) = &self.recovery_db else {
            return Ok(None);
        };
        Ok(Some(RecoveryJournal {
            db: db.clone(),
            id: id.to_string(),
            parent: std::path::absolute(&self.path)?,
            stage: std::path::absolute(staging)?,
            target: std::path::absolute(target)?,
            backup: std::path::absolute(download_backup_path(target, id))?,
            auth_key: std::sync::Mutex::new(None),
            persisted: std::sync::Mutex::new(None),
        }))
    }

    fn check_recovery_destination(&self, target: &Path) -> Result<Option<std::fs::File>> {
        let Some(db) = &self.recovery_db else {
            return Ok(None);
        };
        let lock = db.recovery_lock()?;
        let mut ledger = GlobalRecoveryLedger::load(db, false)?;
        if ledger.check_rows(db).is_err() {
            ledger.reconcile_metadata(db)?;
            ledger.check_rows(db)?;
        }
        if GlobalRecoveryLedger::collect_garbage(db).is_err() {
            // An exact SQL row can still have a terminal anchor after a
            // crash. Retry authenticated metadata reconciliation first.
            ledger.reconcile_metadata(db)?;
            ledger.check_rows(db)?;
            GlobalRecoveryLedger::collect_garbage(db)?;
        }
        #[cfg(unix)]
        let identity = {
            use std::os::unix::fs::MetadataExt;
            let metadata = self.handle.metadata()?;
            (metadata.dev(), metadata.ino())
        };
        #[cfg(windows)]
        let identity = {
            let (volume, index) = windows_file_identity(&*self.handle)?;
            (u64::from(volume), index)
        };
        #[cfg(not(any(unix, windows)))]
        let identity = (0, 0);
        db.download_recovery_check_destination(&recovery_destination_key(identity, target)?)?;
        Ok(Some(lock))
    }

    pub(super) fn verify(&self) -> Result<()> {
        let current = open_download_directory(&self.path)?;
        anyhow::ensure!(
            same_download_directory(&self.handle, &current)?,
            "download parent directory changed during transfer"
        );
        Ok(())
    }

    /// Atomically publish a hash-verified delta without the backup/rename gap.
    pub(super) async fn promote_delta(
        &self,
        staging: &Path,
        target: &Path,
        verified: &tokio::fs::File,
    ) -> Result<()> {
        self.verify()?;
        let _recovery_lock = self.check_recovery_destination(target)?;
        #[cfg(target_os = "linux")]
        {
            anyhow::ensure!(
                staging.parent() == target.parent(),
                "delta staging must be a sibling of its target"
            );
            let stage = staging
                .file_name()
                .context("delta staging has no basename")?
                .to_owned();
            let target = target
                .file_name()
                .context("delta target has no basename")?
                .to_owned();
            let parent = self.handle.try_clone()?;
            let verified = verified.try_clone().await?.into_std().await;
            tokio::task::spawn_blocking(move || {
                let _lock = _recovery_lock;
                use std::os::fd::AsRawFd;
                use std::os::unix::ffi::OsStrExt;

                let expected = verified.metadata()?;
                let entry = pinned_entry(&parent, &stage)?.context("delta stage is missing")?;
                require_pinned_identity(&entry, &expected)?;
                match pinned_entry(&parent, &target)? {
                    None => rename_pinned_siblings(&parent, &stage, &target),
                    Some(existing) => {
                        anyhow::ensure!(
                            pinned_regular_file(&existing),
                            "delta download destination must be a real file"
                        );
                        let stage = std::ffi::CString::new(stage.as_bytes())?;
                        let target = std::ffi::CString::new(target.as_bytes())?;
                        // SAFETY: the directory handle is live and both names are basenames.
                        let result = unsafe {
                            libc::renameat(
                                parent.as_raw_fd(),
                                stage.as_ptr(),
                                parent.as_raw_fd(),
                                target.as_ptr(),
                            )
                        };
                        if result != 0 {
                            return Err(std::io::Error::last_os_error())
                                .context("atomically replace verified delta target");
                        }
                        Ok(())
                    }
                }
            })
            .await
            .context("join guarded delta promotion")?
        }
        #[cfg(windows)]
        {
            anyhow::ensure!(
                staging.parent() == target.parent(),
                "delta staging must be a sibling of its target"
            );
            let stage = staging.file_name().context("delta stage has no basename")?;
            let name = target.file_name().context("delta target has no basename")?;
            let delete = windows_verified_delete_access(&self.handle, stage, verified)?;
            let replace = match windows_open_relative(&self.handle, name, 0x80) {
                Ok(existing) => {
                    windows_staging_identity(&existing)?;
                    anyhow::ensure!(
                        existing.metadata()?.is_file(),
                        "delta download destination must be a real file"
                    );
                    true
                }
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => false,
                Err(error) => return Err(error).context("inspect relative delta destination"),
            };
            windows_rename_handle(&delete, &self.handle, name, replace)
        }
        #[cfg(not(any(target_os = "linux", windows)))]
        {
            match tokio::fs::symlink_metadata(target).await {
                Ok(metadata) => {
                    anyhow::ensure!(
                        metadata.is_file(),
                        "delta download destination must be a real file"
                    );
                    tokio::fs::rename(staging, target)
                        .await
                        .context("atomically replace verified delta target")
                }
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                    promote_staged_download_noreplace(staging, target).await
                }
                Err(error) => Err(error).context("inspect delta download destination"),
            }
        }
    }

    /// Best-effort cleanup of a stage whose actual handle is still open.
    /// A mismatch leaves the entry untouched rather than deleting unknown data.
    pub(super) async fn cleanup_verified(&self, staging: &Path, verified: &tokio::fs::File) {
        if self.verify().is_err() {
            return;
        }
        #[cfg(target_os = "linux")]
        {
            use std::os::fd::AsRawFd;
            use std::os::unix::ffi::OsStrExt;

            let Some(name) = staging.file_name() else {
                return;
            };
            let name = name.to_owned();
            let Ok(parent) = self.handle.try_clone() else {
                return;
            };
            let Ok(verified) = verified.try_clone().await else {
                return;
            };
            let verified = verified.into_std().await;
            let _ = tokio::task::spawn_blocking(move || -> Result<()> {
                let expected = verified.metadata()?;
                let entry = pinned_entry(&parent, &name)?.context("staging is missing")?;
                require_pinned_identity(&entry, &expected)?;
                let name = std::ffi::CString::new(name.as_bytes())?;
                // SAFETY: the directory descriptor is live and the name is a basename.
                let result = unsafe { libc::unlinkat(parent.as_raw_fd(), name.as_ptr(), 0) };
                if result != 0 {
                    return Err(std::io::Error::last_os_error().into());
                }
                Ok(())
            })
            .await;
        }
        #[cfg(windows)]
        {
            if let Some(name) = staging.file_name() {
                if let Ok(delete) = windows_verified_delete_access(&self.handle, name, verified) {
                    let _ = windows_delete_handle(&delete);
                }
            }
        }
        #[cfg(not(any(target_os = "linux", windows)))]
        let _ = (staging, verified);
    }

    pub(super) async fn promote(
        &self,
        staging: &Path,
        target: &Path,
        id: &str,
        allow_replace: bool,
        verified: &tokio::fs::File,
    ) -> Result<()> {
        self.verify()?;
        let _recovery_lock = self.check_recovery_destination(target)?;
        let journal = if allow_replace {
            self.recovery_journal(staging, target, id)?
        } else {
            None
        };
        #[cfg(target_os = "linux")]
        if staging.parent() == target.parent() {
            let staging = staging
                .file_name()
                .context("staging has no basename")?
                .to_owned();
            let target_name = target
                .file_name()
                .context("target has no basename")?
                .to_owned();
            let backup = download_backup_path(target, id);
            let backup = backup
                .file_name()
                .context("backup has no basename")?
                .to_owned();
            let pinned = self.handle.try_clone()?;
            let verified = verified.try_clone().await?.into_std().await;
            return tokio::task::spawn_blocking(move || {
                let _lock = _recovery_lock;
                let expected = verified.metadata()?;
                if allow_replace {
                    promote_pinned_download(
                        &pinned,
                        &staging,
                        &target_name,
                        &backup,
                        Some(&verified),
                        journal.as_ref(),
                    )
                } else {
                    let entry =
                        pinned_entry(&pinned, &staging)?.context("download staging is missing")?;
                    require_pinned_identity(&entry, &expected)?;
                    rename_pinned_siblings(&pinned, &staging, &target_name)
                }
            })
            .await
            .context("join guarded download promotion")?;
        }

        #[cfg(windows)]
        {
            let stage = staging.file_name().context("stage has no basename")?;
            let name = target.file_name().context("target has no basename")?;
            anyhow::ensure!(
                staging.parent() == target.parent(),
                "download staging must be a sibling of its target"
            );
            let delete = windows_verified_delete_access(&self.handle, stage, verified)?;
            if allow_replace {
                let backup = download_backup_path(target, id);
                let backup = backup.file_name().context("backup has no basename")?;
                let parent = self.handle.clone();
                let name = name.to_owned();
                let backup = backup.to_owned();
                tokio::task::spawn_blocking(move || {
                    let _lock = _recovery_lock;
                    windows_promote_with_backup(&parent, &delete, &name, &backup, journal.as_ref())
                })
                .await
                .context("join Windows recovery promotion")?
            } else {
                windows_rename_handle(&delete, &self.handle, name, false)
            }
        }
        #[cfg(not(windows))]
        if allow_replace {
            promote_staged_download(staging, target, id).await
        } else {
            promote_staged_download_noreplace(staging, target).await
        }
    }
}

fn open_download_directory(path: &Path) -> Result<std::fs::File> {
    #[cfg(target_os = "linux")]
    let file = pin_download_parent(path)?;
    #[cfg(windows)]
    let file = {
        use std::os::windows::fs::OpenOptionsExt;
        use windows_sys::Win32::Storage::FileSystem::{
            FILE_FLAG_BACKUP_SEMANTICS, FILE_READ_ATTRIBUTES, FILE_TRAVERSE,
        };

        std::fs::OpenOptions::new()
            .access_mode(FILE_READ_ATTRIBUTES | FILE_TRAVERSE)
            .custom_flags(FILE_FLAG_BACKUP_SEMANTICS)
            .open(path)
            .context("open download parent directory handle")?
    };
    #[cfg(not(any(target_os = "linux", windows)))]
    let file = std::fs::File::open(path)?;
    anyhow::ensure!(
        file.metadata()?.is_dir(),
        "download parent is not a directory"
    );
    Ok(file)
}

fn same_download_directory(a: &std::fs::File, b: &std::fs::File) -> Result<bool> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        let a = a.metadata()?;
        let b = b.metadata()?;
        Ok(a.dev() == b.dev() && a.ino() == b.ino())
    }
    #[cfg(windows)]
    {
        use std::os::windows::io::AsRawHandle;
        use windows_sys::Win32::Storage::FileSystem::{
            GetFileInformationByHandle, BY_HANDLE_FILE_INFORMATION,
        };

        fn identity(file: &std::fs::File) -> Result<(u32, u64)> {
            let mut info = std::mem::MaybeUninit::<BY_HANDLE_FILE_INFORMATION>::uninit();
            // SAFETY: the handle is live and the output buffer is writable.
            let ok = unsafe { GetFileInformationByHandle(file.as_raw_handle(), info.as_mut_ptr()) };
            if ok == 0 {
                return Err(std::io::Error::last_os_error()).context("query directory identity");
            }
            // SAFETY: a successful call initialized the information structure.
            let info = unsafe { info.assume_init() };
            let index = (u64::from(info.nFileIndexHigh) << 32) | u64::from(info.nFileIndexLow);
            Ok((info.dwVolumeSerialNumber, index))
        }

        Ok(identity(a)? == identity(b)?)
    }
    #[cfg(not(any(unix, windows)))]
    {
        let _ = (a, b);
        anyhow::bail!("download directory identity is unsupported on this platform")
    }
}

#[cfg(target_os = "linux")]
fn pinned_entry(parent: &std::fs::File, name: &std::ffi::OsStr) -> Result<Option<libc::stat>> {
    use std::os::fd::AsRawFd;
    use std::os::unix::ffi::OsStrExt;

    let name = std::ffi::CString::new(name.as_bytes()).context("entry name contains NUL")?;
    let mut stat = std::mem::MaybeUninit::<libc::stat>::uninit();
    // SAFETY: the parent descriptor is live and the output buffer is writable.
    let result = unsafe {
        libc::fstatat(
            parent.as_raw_fd(),
            name.as_ptr(),
            stat.as_mut_ptr(),
            libc::AT_SYMLINK_NOFOLLOW,
        )
    };
    if result == 0 {
        // SAFETY: a successful fstatat initialized the buffer.
        return Ok(Some(unsafe { stat.assume_init() }));
    }
    let error = std::io::Error::last_os_error();
    if error.kind() == std::io::ErrorKind::NotFound {
        return Ok(None);
    }
    Err(error).context("inspect pinned download entry")
}

#[cfg(target_os = "linux")]
fn pinned_regular_file(stat: &libc::stat) -> bool {
    stat.st_mode & libc::S_IFMT == libc::S_IFREG
}

#[cfg(target_os = "linux")]
fn pinned_same_file(a: &libc::stat, b: &libc::stat) -> bool {
    pinned_regular_file(a) && pinned_regular_file(b) && a.st_dev == b.st_dev && a.st_ino == b.st_ino
}

#[cfg(target_os = "linux")]
fn require_pinned_identity(entry: &libc::stat, expected: &std::fs::Metadata) -> Result<()> {
    use std::os::unix::fs::MetadataExt;

    anyhow::ensure!(
        pinned_regular_file(entry)
            && expected.is_file()
            && entry.st_nlink == 1
            && expected.nlink() == 1
            && entry.st_dev == expected.dev()
            && entry.st_ino == expected.ino()
            && u64::try_from(entry.st_size).ok() == Some(expected.len()),
        "verified staging identity changed before promotion"
    );
    Ok(())
}

/// Keep backup, promotion, rollback and backup cleanup in one directory.
/// The remaining same-directory entry substitution race requires additional
/// ownership/identity controls; pinning is not a complete TOCTOU solution.
#[cfg(target_os = "linux")]
fn promote_pinned_download(
    parent: &std::fs::File,
    staging: &std::ffi::OsStr,
    target: &std::ffi::OsStr,
    backup: &std::ffi::OsStr,
    verified: Option<&std::fs::File>,
    journal: Option<&RecoveryJournal>,
) -> Result<()> {
    use std::os::fd::AsRawFd;
    use std::os::unix::ffi::OsStrExt;

    let expected = verified.map(std::fs::File::metadata).transpose()?;
    let expected = expected.as_ref();
    let staged = pinned_entry(parent, staging)?.context("download staging is missing")?;
    anyhow::ensure!(
        pinned_regular_file(&staged) && staged.st_nlink == 1,
        "download promotion refused: staging target is not a regular file or has hard links"
    );
    if let Some(expected) = expected {
        require_pinned_identity(&staged, expected)?;
    }
    let original = pinned_entry(parent, target)?;
    if let Some(ref original) = original {
        anyhow::ensure!(
            pinned_regular_file(original),
            "download overwrite refused: existing destination is not a regular file"
        );
        if let Some(journal) = journal {
            use std::os::unix::fs::MetadataExt;
            let directory = parent.metadata()?;
            let stage_identity = (staged.st_dev, staged.st_ino, u64::try_from(staged.st_size)?);
            let original_identity = (
                original.st_dev,
                original.st_ino,
                u64::try_from(original.st_size)?,
            );
            let writer = verified.context("recovery requires verified staging handle")?;
            let stage_hash = match recovery_file_hash(writer, stage_identity) {
                Ok(hash) => hash,
                Err(error)
                    if error
                        .downcast_ref::<std::io::Error>()
                        .is_some_and(|error| error.raw_os_error() == Some(libc::EBADF)) =>
                {
                    // Some backends exclusively create write-only handles.
                    // Keep the writer live and bind the relative read to its identity.
                    let reader = open_recovery_read(parent, staging)?;
                    recovery_file_hash(&reader, stage_identity)?
                }
                Err(error) => return Err(error),
            };
            let original_reader = open_recovery_read(parent, target)?;
            let original_hash = recovery_file_hash(&original_reader, original_identity)?;
            // Persist the existing file data and initial namespace before
            // recording any intent that permits moving the original.
            writer.sync_all().context("persist recovery staging data")?;
            original_reader
                .sync_all()
                .context("persist original recovery data")?;
            sync_recovery_directory(parent)?;
            journal.begin(
                (directory.dev(), directory.ino()),
                stage_identity,
                original_identity,
                stage_hash,
                original_hash,
            )?;
        }
        rename_pinned_siblings(parent, target, backup)
            .context("preserve existing local download target without overwriting a backup")?;
        recovery_fault_checkpoint("backup_renamed");
        if let Some(journal) = journal {
            sync_recovery_directory(parent)?;
            recovery_fault_checkpoint("backup_directory_synced");
            journal.advance(RecoveryPhase::Prepared, RecoveryPhase::BackupSaved)?;
        }
    }

    let promotion = (|| -> Result<()> {
        #[cfg(test)]
        anyhow::ensure!(
            !crate::credentials::recovery_test_force_rollback(),
            "injected promotion failure"
        );
        let current = pinned_entry(parent, staging)?.context("download staging disappeared")?;
        anyhow::ensure!(
            pinned_same_file(&staged, &current) && current.st_nlink == 1,
            "download staging identity changed before promotion"
        );
        if let Some(expected) = expected {
            require_pinned_identity(&current, expected)?;
        }
        rename_pinned_siblings(parent, staging, target)
    })();
    if let Err(error) = promotion {
        if let Some(ref original) = original {
            let recoverable = pinned_entry(parent, backup).is_ok_and(|entry| {
                entry.is_some_and(|current| pinned_same_file(original, &current))
            });
            if !recoverable {
                return Err(error)
                    .context("download promotion failed; backup identity is uncertain");
            }
            if let Some(journal) = journal {
                journal.advance(RecoveryPhase::BackupSaved, RecoveryPhase::RollbackPending)?;
                journal.verify_before_mutation(parent, RecoveryPhase::RollbackPending, false)?;
            }
            return match rename_pinned_siblings(parent, backup, target) {
                Ok(()) => {
                    recovery_fault_checkpoint("rollback_restored");
                    if let Some(journal) = journal {
                        sync_recovery_directory(parent)?;
                        recovery_fault_checkpoint("rollback_directory_synced");
                        journal.advance(RecoveryPhase::RollbackPending, RecoveryPhase::RolledBack)?;
                    }
                    Err(error).context("download promotion failed; original target restored")
                }
                Err(restore) => Err(anyhow::anyhow!(
                    "download promotion failed; recovery backup {:?} remains in pinned directory: {error:#}; rollback: {restore:#}",
                    backup
                )),
            };
        }
        return Err(error);
    }

    recovery_fault_checkpoint("stage_published");
    if let Some(original) = original {
        if let Some(journal) = journal {
            if sync_recovery_directory(parent).is_err() {
                tracing::warn!(
                    "download published; directory durability uncertain; backup retained"
                );
                return Ok(());
            }
            recovery_fault_checkpoint("published_directory_synced");
            if journal
                .advance(RecoveryPhase::BackupSaved, RecoveryPhase::Promoted)
                .is_err()
                || journal
                    .advance(RecoveryPhase::Promoted, RecoveryPhase::CleanupPending)
                    .is_err()
            {
                // The new file is already published. Preserve the original
                // backup and recovery evidence if phase persistence fails.
                tracing::warn!("download published; recovery phase could not be persisted");
                return Ok(());
            }
        }
        // Never knowingly delete a different file substituted at the backup name.
        let owned = pinned_entry(parent, backup)
            .is_ok_and(|entry| entry.is_some_and(|current| pinned_same_file(&original, &current)));
        if owned {
            if let Some(journal) = journal {
                if journal
                    .verify_before_mutation(parent, RecoveryPhase::CleanupPending, true)
                    .is_err()
                {
                    tracing::warn!("download published; unverified recovery backup retained");
                    return Ok(());
                }
            }
            if let Ok(name) = std::ffi::CString::new(backup.as_bytes()) {
                // SAFETY: parent is live; name is a NUL-terminated sibling basename.
                let removed = unsafe { libc::unlinkat(parent.as_raw_fd(), name.as_ptr(), 0) };
                if removed == 0 {
                    recovery_fault_checkpoint("backup_removed");
                    if let Some(journal) = journal {
                        if sync_recovery_directory(parent).is_err() {
                            tracing::warn!(
                                "backup removal durability uncertain; recovery intent retained"
                            );
                            return Ok(());
                        }
                        recovery_fault_checkpoint("cleanup_directory_synced");
                        journal.finish();
                    }
                }
            }
        }
    }
    Ok(())
}

/// Use atomic no-replace operations. On Linux, sibling moves resolve the
/// common parent only once. Other moves retain the existing platform helper.
#[cfg(any(not(windows), test))]
async fn move_local_noreplace(from: &Path, to: &Path) -> Result<()> {
    let from = from.to_path_buf();
    let to = to.to_path_buf();
    tokio::task::spawn_blocking(move || {
        #[cfg(target_os = "linux")]
        if let (Some(parent), Some(other_parent), Some(source), Some(target)) =
            (from.parent(), to.parent(), from.file_name(), to.file_name())
        {
            if parent == other_parent {
                let pinned = pin_download_parent(parent)?;
                return rename_pinned_siblings(&pinned, source, target);
            }
        }
        let src = from.to_str().context("download path is not valid UTF-8")?;
        let dest = to
            .to_str()
            .context("download destination is not valid UTF-8")?;
        crate::remotefs::local::rename_local_noreplace(src, dest)
    })
    .await
    .context("join local no-replace download promotion")?
}

/// Open a single existing child of the pinned directory, never resolving
/// the full path again. NT native open is needed for a RootDirectory handle.
#[cfg(windows)]
fn windows_open_relative(
    parent: &std::fs::File,
    name: &std::ffi::OsStr,
    access: u32,
) -> std::io::Result<std::fs::File> {
    use std::ffi::c_void;
    use std::os::windows::ffi::OsStrExt;
    use std::os::windows::io::{AsRawHandle, FromRawHandle};
    use windows_sys::Win32::Foundation::HANDLE;

    #[repr(C)]
    struct UnicodeString {
        length: u16,
        maximum_length: u16,
        buffer: *mut u16,
    }
    #[repr(C)]
    struct ObjectAttributes {
        length: u32,
        root_directory: HANDLE,
        object_name: *mut UnicodeString,
        attributes: u32,
        security_descriptor: *mut c_void,
        security_quality_of_service: *mut c_void,
    }
    #[repr(C)]
    struct IoStatusBlock {
        status_or_pointer: usize,
        information: usize,
    }
    #[link(name = "ntdll")]
    extern "system" {
        fn NtOpenFile(
            file_handle: *mut HANDLE,
            desired_access: u32,
            object_attributes: *mut ObjectAttributes,
            io_status_block: *mut IoStatusBlock,
            share_access: u32,
            open_options: u32,
        ) -> i32;
        fn RtlNtStatusToDosError(status: i32) -> u32;
    }

    let mut wide: Vec<u16> = name.encode_wide().collect();
    if wide.is_empty()
        || name == "."
        || name == ".."
        || wide.iter().any(|&c| matches!(c, 0 | 47 | 58 | 92))
    {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "relative open requires a single filename",
        ));
    }
    let bytes = wide
        .len()
        .checked_mul(2)
        .and_then(|n| u16::try_from(n).ok())
        .ok_or_else(|| std::io::Error::other("relative filename too long"))?;
    let mut name = UnicodeString {
        length: bytes,
        maximum_length: bytes,
        buffer: wide.as_mut_ptr(),
    };
    let mut attributes = ObjectAttributes {
        length: std::mem::size_of::<ObjectAttributes>() as u32,
        root_directory: parent.as_raw_handle(),
        object_name: &mut name,
        attributes: 0x40, // OBJ_CASE_INSENSITIVE
        security_descriptor: std::ptr::null_mut(),
        security_quality_of_service: std::ptr::null_mut(),
    };
    let mut iosb = IoStatusBlock {
        status_or_pointer: 0,
        information: 0,
    };
    let mut handle: HANDLE = std::ptr::null_mut();
    // SAFETY: these repr(C) structures match the Windows x64 NT ABI. All
    // buffers and the root handle are live until the synchronous call returns.
    let status = unsafe {
        NtOpenFile(
            &mut handle,
            access | 0x0010_0000, // SYNCHRONIZE
            &mut attributes,
            &mut iosb,
            0x7,         // share read, write, delete
            0x0020_0060, // OPEN_REPARSE_POINT | NON_DIRECTORY | SYNCHRONOUS_IO_NONALERT
        )
    };
    if status < 0 {
        // SAFETY: conversion accepts any NTSTATUS value.
        let error = unsafe { RtlNtStatusToDosError(status) };
        return Err(std::io::Error::from_raw_os_error(error as i32));
    }
    if handle.is_null() {
        return Err(std::io::Error::other("relative open returned no handle"));
    }
    // SAFETY: successful NtOpenFile transfers ownership of this handle.
    Ok(unsafe { std::fs::File::from_raw_handle(handle) })
}

/// Preserve the original file by its own native handle. All rename targets
/// are resolved against the retained directory and cannot overwrite a collision.
#[cfg(windows)]
fn windows_promote_with_backup(
    parent: &std::fs::File,
    source: &std::fs::File,
    target: &std::ffi::OsStr,
    backup: &std::ffi::OsStr,
    journal: Option<&RecoveryJournal>,
) -> Result<()> {
    const DELETE_AND_ATTRIBUTES: u32 = 0x0001_0080;
    let access = if journal.is_some() {
        DELETE_AND_ATTRIBUTES | 1 // FILE_READ_DATA
    } else {
        DELETE_AND_ATTRIBUTES
    };
    let original = match windows_open_relative(parent, target, access) {
        Ok(file) => file,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            return windows_rename_handle(source, parent, target, false);
        }
        Err(error) => return Err(error).context("open existing destination relative to parent"),
    };
    windows_staging_identity(&original)?;
    anyhow::ensure!(
        original.metadata()?.is_file(),
        "destination is not a real file"
    );
    if let Some(journal) = journal {
        let (volume, index) = windows_file_identity(parent)?;
        let (source_volume, source_index) = windows_staging_identity(source)?;
        let (original_volume, original_index) = windows_staging_identity(&original)?;
        let stage_identity = (
            u64::from(source_volume),
            source_index,
            source.metadata()?.len(),
        );
        let original_identity = (
            u64::from(original_volume),
            original_index,
            original.metadata()?.len(),
        );
        let reader = windows_reopen_recovery_read(source)?;
        windows_flush_recovery_file(source)?;
        windows_flush_recovery_file(&original)?;
        journal.begin(
            (u64::from(volume), index),
            stage_identity,
            original_identity,
            recovery_file_hash(&reader, stage_identity)?,
            recovery_file_hash(&original, original_identity)?,
        )?;
    }
    windows_rename_handle(&original, parent, backup, false)
        .context("preserve original destination without replacing backup")?;
    recovery_fault_checkpoint("backup_renamed");
    if let Some(journal) = journal {
        windows_flush_recovery_file(&original)?;
        recovery_fault_checkpoint("backup_file_flushed");
        journal.advance(RecoveryPhase::Prepared, RecoveryPhase::BackupSaved)?;
    }
    let promotion = (|| -> Result<()> {
        #[cfg(test)]
        anyhow::ensure!(
            !crate::credentials::recovery_test_force_rollback(),
            "injected promotion failure"
        );
        windows_rename_handle(source, parent, target, false)
    })();
    if let Err(error) = promotion {
        if let Some(journal) = journal {
            journal.advance(RecoveryPhase::BackupSaved, RecoveryPhase::RollbackPending)?;
            journal.verify_before_mutation(parent, RecoveryPhase::RollbackPending, false)?;
        }
        return match windows_rename_handle(&original, parent, target, false) {
            Ok(()) => {
                recovery_fault_checkpoint("rollback_restored");
                if let Some(journal) = journal {
                    windows_flush_recovery_file(&original)?;
                    recovery_fault_checkpoint("rollback_file_flushed");
                    journal.advance(RecoveryPhase::RollbackPending, RecoveryPhase::RolledBack)?;
                }
                Err(error).context("promotion failed; original restored")
            }
            Err(rollback) => Err(anyhow::anyhow!(
                "promotion failed; recovery backup remains: {error:#}; rollback: {rollback:#}"
            )),
        };
    }
    recovery_fault_checkpoint("stage_published");
    if let Some(journal) = journal {
        if windows_flush_recovery_file(source).is_err() {
            tracing::warn!("download published; Windows file flush failed; backup retained");
            return Ok(());
        }
        recovery_fault_checkpoint("published_file_flushed");
        if journal
            .advance(RecoveryPhase::BackupSaved, RecoveryPhase::Promoted)
            .is_err()
            || journal
                .advance(RecoveryPhase::Promoted, RecoveryPhase::CleanupPending)
                .is_err()
        {
            tracing::warn!("download published; recovery phase could not be persisted");
            return Ok(());
        }
    }
    // The original handle now identifies the backup, not the new target.
    // Cleanup failure leaves the backup intact; publication already succeeded.
    if let Some(journal) = journal {
        if windows_flush_recovery_file(source).is_err() {
            tracing::warn!("download published; Windows file flush failed; backup retained");
            return Ok(());
        }
        if journal
            .verify_before_mutation(parent, RecoveryPhase::CleanupPending, true)
            .is_err()
        {
            tracing::warn!("download published; unverified recovery backup retained");
            return Ok(());
        }
    }
    let removed = windows_delete_handle(&original).is_ok();
    drop(original);
    if removed {
        recovery_fault_checkpoint("backup_removed");
        if let Some(journal) = journal {
            if windows_open_relative(parent, backup, 0x80)
                .is_err_and(|error| error.kind() == std::io::ErrorKind::NotFound)
            {
                if windows_flush_recovery_file(source).is_err() {
                    tracing::warn!("Windows cleanup flush uncertain; recovery intent retained");
                    return Ok(());
                }
                recovery_fault_checkpoint("cleanup_file_flushed");
                journal.finish();
            }
        }
    }
    Ok(())
}

/// Verify the directory entry relative to the retained parent and acquire
/// DELETE access to the original object, never to a replacement pathname.
#[cfg(windows)]
fn windows_verified_delete_access(
    parent: &std::fs::File,
    name: &std::ffi::OsStr,
    verified: &tokio::fs::File,
) -> Result<std::fs::File> {
    let entry = windows_open_relative(parent, name, 0x80)?;
    let expected = windows_staging_identity(verified)?;
    anyhow::ensure!(
        windows_staging_identity(&entry)? == expected,
        "verified staging identity changed before native operation"
    );
    let delete = windows_delete_access(verified)?;
    anyhow::ensure!(
        windows_staging_identity(&delete)? == expected && delete.metadata()?.is_file(),
        "verified staging identity changed when acquiring delete access"
    );
    Ok(delete)
}

#[cfg(windows)]
fn windows_file_identity(file: &impl std::os::windows::io::AsRawHandle) -> Result<(u32, u64)> {
    use windows_sys::Win32::Storage::FileSystem::{
        GetFileInformationByHandle, BY_HANDLE_FILE_INFORMATION,
    };
    let mut info = std::mem::MaybeUninit::<BY_HANDLE_FILE_INFORMATION>::uninit();
    // SAFETY: the file handle and writable output buffer remain live.
    let ok = unsafe { GetFileInformationByHandle(file.as_raw_handle(), info.as_mut_ptr()) };
    if ok == 0 {
        return Err(std::io::Error::last_os_error()).context("read recovery file identity");
    }
    // SAFETY: successful API call initialized the structure.
    let info = unsafe { info.assume_init() };
    Ok((
        info.dwVolumeSerialNumber,
        (u64::from(info.nFileIndexHigh) << 32) | u64::from(info.nFileIndexLow),
    ))
}

/// Read the same open object, not a path that could have been substituted.
#[cfg(windows)]
fn windows_reopen_recovery_read(file: &std::fs::File) -> Result<std::fs::File> {
    use std::os::windows::io::{AsRawHandle, FromRawHandle};
    use windows_sys::Win32::Foundation::INVALID_HANDLE_VALUE;
    use windows_sys::Win32::Storage::FileSystem::{
        ReOpenFile, FILE_FLAG_OPEN_REPARSE_POINT, FILE_READ_ATTRIBUTES, FILE_READ_DATA,
        FILE_SHARE_DELETE, FILE_SHARE_READ, FILE_SHARE_WRITE,
    };

    // SAFETY: the original handle is live; successful reopen owns a new handle.
    let handle = unsafe {
        ReOpenFile(
            file.as_raw_handle(),
            FILE_READ_DATA | FILE_READ_ATTRIBUTES,
            FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
            FILE_FLAG_OPEN_REPARSE_POINT,
        )
    };
    if handle == INVALID_HANDLE_VALUE {
        return Err(std::io::Error::last_os_error()).context("reopen recovery content");
    }
    // SAFETY: successful ReOpenFile returns an independently owned handle.
    Ok(unsafe { std::fs::File::from_raw_handle(handle) })
}

/// Request a write-capable handle to the same object and flush its buffers.
/// This is a file-level durability barrier, not a documented volume or
/// directory-entry power-loss guarantee. Never resolve the original pathname.
#[cfg(windows)]
fn windows_flush_recovery_file(file: &std::fs::File) -> Result<()> {
    use std::os::windows::io::{AsRawHandle, FromRawHandle};
    use windows_sys::Win32::Foundation::INVALID_HANDLE_VALUE;
    use windows_sys::Win32::Storage::FileSystem::{
        ReOpenFile, FILE_FLAG_OPEN_REPARSE_POINT, FILE_SHARE_DELETE, FILE_SHARE_READ,
        FILE_SHARE_WRITE,
    };

    const GENERIC_WRITE_ACCESS: u32 = 0x4000_0000;
    let identity = windows_staging_identity(file)?;
    // SAFETY: the original handle remains live; success returns an owned handle.
    let handle = unsafe {
        ReOpenFile(
            file.as_raw_handle(),
            GENERIC_WRITE_ACCESS | 0x80, // FILE_READ_ATTRIBUTES
            FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
            FILE_FLAG_OPEN_REPARSE_POINT,
        )
    };
    if handle == INVALID_HANDLE_VALUE {
        return Err(std::io::Error::last_os_error()).context("reopen recovery file for flush");
    }
    // SAFETY: successful ReOpenFile transfers ownership of the new handle.
    let writer = unsafe { std::fs::File::from_raw_handle(handle) };
    anyhow::ensure!(
        windows_staging_identity(&writer)? == identity,
        "recovery flush handle identity mismatch"
    );
    writer
        .sync_all()
        .context("flush verified Windows recovery file")?;
    anyhow::ensure!(
        windows_staging_identity(&writer)? == identity,
        "recovery flush identity changed"
    );
    Ok(())
}

/// Obtain DELETE access to the already opened object, not a new pathname.
#[cfg(windows)]
fn windows_delete_access(file: &tokio::fs::File) -> Result<std::fs::File> {
    use std::os::windows::io::{AsRawHandle, FromRawHandle};
    use windows_sys::Win32::Foundation::INVALID_HANDLE_VALUE;
    use windows_sys::Win32::Storage::FileSystem::{
        ReOpenFile, FILE_FLAG_OPEN_REPARSE_POINT, FILE_READ_ATTRIBUTES, FILE_SHARE_DELETE,
        FILE_SHARE_READ, FILE_SHARE_WRITE,
    };

    // DELETE is the standard access right required by rename and disposition.
    const DELETE_ACCESS: u32 = 0x0001_0000;
    // SAFETY: the original handle remains live; ReOpenFile returns a new owned handle.
    let handle = unsafe {
        ReOpenFile(
            file.as_raw_handle(),
            DELETE_ACCESS | FILE_READ_ATTRIBUTES,
            FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
            FILE_FLAG_OPEN_REPARSE_POINT,
        )
    };
    if handle == INVALID_HANDLE_VALUE {
        return Err(std::io::Error::last_os_error()).context("reopen verified object for delete");
    }
    // SAFETY: the successful call returned an independently owned handle.
    Ok(unsafe { std::fs::File::from_raw_handle(handle) })
}

/// Rename the open source object relative to an already opened destination directory.
#[cfg(windows)]
fn windows_rename_handle(
    source: &std::fs::File,
    parent: &std::fs::File,
    name: &std::ffi::OsStr,
    replace: bool,
) -> Result<()> {
    use std::ffi::c_void;
    use std::os::windows::ffi::OsStrExt;
    use std::os::windows::io::AsRawHandle;
    use windows_sys::Win32::Foundation::HANDLE;
    use windows_sys::Win32::Storage::FileSystem::FILE_RENAME_INFO;

    #[repr(C)]
    struct IoStatusBlock {
        status_or_pointer: usize,
        information: usize,
    }

    #[link(name = "ntdll")]
    extern "system" {
        fn NtSetInformationFile(
            file_handle: HANDLE,
            io_status_block: *mut IoStatusBlock,
            file_information: *mut c_void,
            length: u32,
            file_information_class: u32,
        ) -> i32;
        fn RtlNtStatusToDosError(status: i32) -> u32;
    }
    const FILE_RENAME_INFORMATION: u32 = 10;

    let wide: Vec<u16> = name.encode_wide().collect();
    anyhow::ensure!(
        !wide.is_empty()
            && name != "."
            && name != ".."
            && !wide.iter().any(|&c| matches!(c, 0 | 47 | 58 | 92)),
        "rename target must be a simple filename"
    );
    let bytes = wide.len().checked_mul(2).context("rename name too long")?;
    let offset = std::mem::offset_of!(FILE_RENAME_INFO, FileName);
    let len = offset
        .checked_add(bytes)
        .and_then(|n| n.checked_add(2))
        .context("rename buffer overflow")?
        .max(std::mem::size_of::<FILE_RENAME_INFO>());
    let len_u32 = u32::try_from(len)?;
    let name_len = u32::try_from(bytes)?;
    // usize storage is aligned for FILE_RENAME_INFO on supported Windows x64.
    let mut buffer = vec![0usize; len.div_ceil(std::mem::size_of::<usize>())];
    let info = buffer.as_mut_ptr().cast::<FILE_RENAME_INFO>();
    // SAFETY: the aligned buffer has room for the fixed header, UTF-16 name,
    // and zero terminator. Both file handles remain live throughout the call.
    let mut iosb = IoStatusBlock {
        status_or_pointer: 0,
        information: 0,
    };
    let status = unsafe {
        (*info).Anonymous.ReplaceIfExists = u8::from(replace);
        (*info).RootDirectory = parent.as_raw_handle();
        (*info).FileNameLength = name_len;
        std::ptr::copy_nonoverlapping(
            wide.as_ptr(),
            buffer.as_mut_ptr().cast::<u8>().add(offset).cast::<u16>(),
            wide.len(),
        );
        // The Win32 wrapper rejected the retained root handle with error 87.
        // Use the native FileRenameInformation class and the same pinned root.
        NtSetInformationFile(
            source.as_raw_handle(),
            &mut iosb,
            info.cast(),
            len_u32,
            FILE_RENAME_INFORMATION,
        )
    };
    if status < 0 {
        // SAFETY: this conversion accepts any NTSTATUS.
        let error = unsafe { RtlNtStatusToDosError(status) };
        return Err(std::io::Error::from_raw_os_error(error as i32))
            .context("rename verified Windows file handle");
    }
    anyhow::ensure!(status == 0, "Windows recovery rename did not complete");
    Ok(())
}

#[cfg(windows)]
fn windows_delete_handle(file: &std::fs::File) -> Result<()> {
    use std::os::windows::io::AsRawHandle;
    use windows_sys::Win32::Storage::FileSystem::{
        FileDispositionInfo, SetFileInformationByHandle, FILE_DISPOSITION_INFO,
    };

    let disposition = FILE_DISPOSITION_INFO { DeleteFile: 1 };
    // SAFETY: the source handle and fixed-size disposition buffer are live.
    let ok = unsafe {
        SetFileInformationByHandle(
            file.as_raw_handle(),
            FileDispositionInfo,
            (&disposition as *const FILE_DISPOSITION_INFO).cast(),
            std::mem::size_of::<FILE_DISPOSITION_INFO>() as u32,
        )
    };
    if ok == 0 {
        return Err(std::io::Error::last_os_error()).context("delete verified Windows file handle");
    }
    Ok(())
}

/// Windows file identity from an opened handle, not timestamps or path text.
/// The volume serial and file index identify the file on supported volumes.
#[cfg(windows)]
fn windows_staging_identity(file: &impl std::os::windows::io::AsRawHandle) -> Result<(u32, u64)> {
    use windows_sys::Win32::Storage::FileSystem::{
        GetFileInformationByHandle, BY_HANDLE_FILE_INFORMATION, FILE_ATTRIBUTE_REPARSE_POINT,
    };

    let mut info = std::mem::MaybeUninit::<BY_HANDLE_FILE_INFORMATION>::uninit();
    // SAFETY: file owns a live handle and the output buffer is writable.
    let result = unsafe { GetFileInformationByHandle(file.as_raw_handle(), info.as_mut_ptr()) };
    if result == 0 {
        return Err(std::io::Error::last_os_error())
            .context("query download staging file identity and link count");
    }
    // SAFETY: a successful call initialized the information structure.
    let info = unsafe { info.assume_init() };
    anyhow::ensure!(
        info.nNumberOfLinks == 1,
        "download staging must not have additional hard links"
    );
    anyhow::ensure!(
        info.dwFileAttributes & FILE_ATTRIBUTE_REPARSE_POINT == 0,
        "download staging must not be a reparse point"
    );
    let index = (u64::from(info.nFileIndexHigh) << 32) | u64::from(info.nFileIndexLow);
    Ok((info.dwVolumeSerialNumber, index))
}

/// The common staging path covers every non-Agent download backend: FTP,
/// FTPS, SFTP, S3, WebDAV, HTTP, Dropbox, OneDrive, Drive, Box and SaaS APIs.
/// A provider returning success is not proof that the bytes hit the local
/// temporary file. Do not replace a preexisting file unless the byte count
/// matches the actual staging file and it has been synced to persistent storage.
#[cfg(test)]
pub(super) async fn verify_staged_download_bytes(
    staging: &Path,
    transferred: u64,
) -> Result<tokio::fs::File> {
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
    let mut options = tokio::fs::OpenOptions::new();
    options.read(true).write(true);
    #[cfg(unix)]
    options.custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK);
    #[cfg(windows)]
    options.custom_flags(windows_sys::Win32::Storage::FileSystem::FILE_FLAG_OPEN_REPARSE_POINT);
    let file = options
        .open(staging)
        .await
        .context("open download staging for durability check")?;
    let opened = file
        .metadata()
        .await
        .context("inspect opened download staging")?;
    anyhow::ensure!(
        opened.file_type().is_file() && opened.len() == transferred,
        "opened download staging changed type or size"
    );
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        anyhow::ensure!(
            opened.dev() == metadata.dev() && opened.ino() == metadata.ino(),
            "download staging was replaced before durability check"
        );
        anyhow::ensure!(
            opened.nlink() == 1,
            "download staging must not have additional hard links"
        );
    }
    #[cfg(windows)]
    let windows_identity = {
        use std::os::windows::fs::MetadataExt;
        use windows_sys::Win32::Storage::FileSystem::FILE_ATTRIBUTE_REPARSE_POINT;
        let identity = windows_staging_identity(&file)?;
        anyhow::ensure!(
            opened.file_attributes() & FILE_ATTRIBUTE_REPARSE_POINT == 0,
            "download staging must not be a reparse point"
        );
        anyhow::ensure!(
            opened.creation_time() == metadata.creation_time()
                && opened.last_write_time() == metadata.last_write_time(),
            "download staging metadata changed before durability check"
        );
        identity
    };
    file.sync_all()
        .await
        .context("sync completed download staging before promotion")?;
    let current = tokio::fs::symlink_metadata(staging)
        .await
        .context("recheck synced download staging pathname")?;
    anyhow::ensure!(
        current.file_type().is_file() && current.len() == transferred,
        "download staging changed after durability check"
    );
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        anyhow::ensure!(
            current.dev() == opened.dev() && current.ino() == opened.ino() && current.nlink() == 1,
            "download staging identity changed after durability check"
        );
    }
    #[cfg(windows)]
    {
        use std::os::windows::fs::MetadataExt;
        use windows_sys::Win32::Storage::FileSystem::FILE_ATTRIBUTE_REPARSE_POINT;
        anyhow::ensure!(
            current.file_attributes() & FILE_ATTRIBUTE_REPARSE_POINT == 0
                && current.creation_time() == opened.creation_time()
                && current.last_write_time() == opened.last_write_time(),
            "download staging metadata changed after durability check"
        );
        let reopened = tokio::fs::OpenOptions::new()
            .read(true)
            .custom_flags(windows_sys::Win32::Storage::FileSystem::FILE_FLAG_OPEN_REPARSE_POINT)
            .open(staging)
            .await
            .context("reopen synced download staging for identity verification")?;
        let reopened_metadata = reopened.metadata().await?;
        anyhow::ensure!(
            reopened_metadata.file_type().is_file() && reopened_metadata.len() == transferred,
            "reopened download staging changed type or size"
        );
        anyhow::ensure!(
            windows_staging_identity(&file)? == windows_identity
                && windows_staging_identity(&reopened)? == windows_identity,
            "download staging file identity changed after durability check"
        );
    }
    Ok(file)
}

#[cfg(any(not(windows), test))]
pub(super) async fn promote_staged_download(
    staging: &Path,
    final_path: &Path,
    id: &str,
) -> Result<()> {
    let backup = download_backup_path(final_path, id);
    #[cfg(target_os = "linux")]
    if staging.parent() == final_path.parent() && backup.parent() == final_path.parent() {
        let parent = final_path
            .parent()
            .context("download target has no parent")?;
        let staging_name = staging
            .file_name()
            .context("staging has no basename")?
            .to_owned();
        let target_name = final_path
            .file_name()
            .context("target has no basename")?
            .to_owned();
        let backup_name = backup
            .file_name()
            .context("backup has no basename")?
            .to_owned();
        let parent = parent.to_path_buf();
        return tokio::task::spawn_blocking(move || {
            let pinned = pin_download_parent(&parent)?;
            promote_pinned_download(
                &pinned,
                &staging_name,
                &target_name,
                &backup_name,
                None,
                None,
            )
        })
        .await
        .context("join pinned download promotion")?;
    }
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
#[cfg(any(not(windows), test))]
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
#[cfg(test)]
pub(super) async fn commit_verified_delta_output(
    staging: &Path,
    final_path: &Path,
    expected_bytes: u64,
) -> Result<()> {
    let guard = DownloadParentGuard::open(final_path)?;
    let verified = verify_staged_download_bytes(staging, expected_bytes).await?;
    guard.promote_delta(staging, final_path, &verified).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    fn recovery_protocol_survives_killed_process_at_durable_boundaries() {
        run_recovery_crash_matrix(false);
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "requires an unlocked native credential vault; run explicitly in native CI"]
    fn native_vault_recovery_survives_killed_process() {
        run_recovery_crash_matrix(true);
    }

    #[cfg(any(target_os = "linux", windows))]
    fn native_fixture_promote(
        db: &std::sync::Arc<crate::db::Db>,
        stage: &Path,
        target: &Path,
        id: &str,
    ) -> Result<()> {
        let backup = download_backup_path(target, id);
        let guard = DownloadParentGuard::with_db(target, Some(db.clone()))?;
        let _lock = guard.check_recovery_destination(target)?;
        let journal = guard.recovery_journal(stage, target, id)?.unwrap();
        #[cfg(target_os = "linux")]
        {
            let writer = std::fs::OpenOptions::new()
                .read(true)
                .write(true)
                .open(stage)?;
            promote_pinned_download(
                &guard.handle,
                stage.file_name().unwrap(),
                target.file_name().unwrap(),
                backup.file_name().unwrap(),
                Some(&writer),
                Some(&journal),
            )
        }
        #[cfg(windows)]
        {
            let writer =
                windows_open_relative(&guard.handle, stage.file_name().unwrap(), 0x0001_0081)?;
            windows_promote_with_backup(
                &guard.handle,
                &writer,
                target.file_name().unwrap(),
                backup.file_name().unwrap(),
                Some(&journal),
            )
        }
    }

    #[cfg(any(target_os = "linux", windows))]
    fn write_durable_fixture(path: &Path, bytes: &[u8]) -> Result<()> {
        use std::io::Write;

        let mut file = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(path)?;
        file.write_all(bytes)?;
        file.sync_all()?;
        Ok(())
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "requires a disposable native VM and explicit hard-reset harness"]
    fn vm_recovery_hard_reset_probe() {
        assert_eq!(
            std::env::var("GHOSTFTP_VM_TEST_OPT_IN").ok().as_deref(),
            Some("disposable-guest")
        );
        let root = PathBuf::from(std::env::var_os("GHOSTFTP_VM_RECOVERY_ROOT").unwrap());
        let mode = std::env::var("GHOSTFTP_VM_RECOVERY_MODE").unwrap();
        let point = std::env::var("GHOSTFTP_VM_RECOVERY_POINT").unwrap();
        assert!(root.is_absolute());
        let name = root.file_name().unwrap().to_str().unwrap();
        assert!(Uuid::parse_str(name.strip_prefix("ghostftp-vm-").unwrap()).is_ok());
        assert!(matches!(
            point.as_str(),
            "prepared_sql_committed"
                | "backup_renamed"
                | "stage_published"
                | "backup_removed"
                | "terminal_manifest_written"
                | "rollback:rollback_restored"
        ));
        if mode == "prepare" {
            // Never reuse an existing directory or overwrite a user's file.
            std::fs::create_dir(&root).unwrap();
            write_durable_fixture(&root.join("stage"), b"new").unwrap();
            write_durable_fixture(&root.join("target"), b"old").unwrap();
            write_durable_fixture(&root.join("unrelated"), b"keep").unwrap();
            write_durable_fixture(&root.join("vm-fixture"), point.as_bytes()).unwrap();
            #[cfg(target_os = "linux")]
            {
                std::fs::File::open(&root).unwrap().sync_all().unwrap();
                std::fs::File::open(root.parent().unwrap())
                    .unwrap()
                    .sync_all()
                    .unwrap();
            }
        } else {
            assert_eq!(
                std::fs::read(root.join("vm-fixture")).unwrap(),
                point.as_bytes()
            );
        }
        let root = root.canonicalize().unwrap();
        let stage = root.join("stage");
        let target = root.join("target");
        let backup = download_backup_path(&target, "vm");
        let checkpoint = (mode == "crash").then(|| point.clone());
        crate::credentials::configure_native_recovery_test_vault(root.join("vault"), checkpoint);
        if mode == "cleanup" {
            crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault")).unwrap();
            crate::credentials::clear_recovery_test_vault();
            // Retain guest evidence for the operator; never remove the root.
            return;
        }
        let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
        match mode.as_str() {
            "prepare" => {
                let _lock = db.recovery_lock().unwrap();
                GlobalRecoveryLedger::load(&db, true).unwrap();
                eprintln!("GHOSTFTP_VM_PREPARE_OK");
            }
            "crash" => {
                let outcome = native_fixture_promote(&db, &stage, &target, "vm");
                panic!("VM checkpoint not reached: {outcome:?}");
            }
            "verify" => {
                let _lock = db.recovery_lock().unwrap();
                let paths = [&stage, &target, &backup];
                let observe = || {
                    paths.map(|path| match std::fs::read(path) {
                        Ok(bytes) => Some(bytes),
                        Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
                        Err(error) => panic!("VM file observation failed: {error}"),
                    })
                };
                let before = observe();
                let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
                ledger.reconcile_metadata(&db).unwrap();
                ledger.check_rows(&db).unwrap();
                GlobalRecoveryLedger::collect_garbage(&db).unwrap();
                let rows = db.download_recovery_snapshot().unwrap();
                if point == "terminal_manifest_written" {
                    assert!(rows.is_empty());
                    assert_eq!(before[1].as_deref(), Some(b"new".as_slice()));
                    assert!(before[2].is_none(), "retired backup reappeared");
                } else {
                    assert_eq!(rows.len(), 1);
                    assert_eq!(rows[0].id, "vm");
                    let signed: AuthenticatedRecovery =
                        serde_json::from_str(&rows[0].payload).unwrap();
                    assert_eq!(
                        signed
                            .authenticate(&rows[0].id, &rows[0].payload)
                            .unwrap()
                            .as_str(),
                        rows[0].phase
                    );
                    let record: RecoveryInspectionRecord =
                        serde_json::from_str(&signed.payload).unwrap();
                    assert!(record.matches_reservation(rows[0].destination_key.as_deref()));
                    assert!(!matches!(
                        record.inspect(),
                        RecoveryObservation::Ambiguous | RecoveryObservation::Unavailable
                    ));
                }
                assert!(before
                    .iter()
                    .flatten()
                    .all(|bytes| bytes == b"old" || bytes == b"new"));
                assert!(before.iter().flatten().any(|bytes| bytes == b"new"));
                if !matches!(
                    point.as_str(),
                    "backup_removed" | "terminal_manifest_written"
                ) {
                    assert!(before.iter().flatten().any(|bytes| bytes == b"old"));
                }
                assert_eq!(before, observe(), "metadata recovery changed VM user files");
                assert_eq!(std::fs::read(root.join("unrelated")).unwrap(), b"keep");
                // Verify a second startup and keep all filesystem evidence.
                let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
                ledger.reconcile_metadata(&db).unwrap();
                ledger.check_rows(&db).unwrap();
                assert_eq!(before, observe());
                eprintln!("GHOSTFTP_VM_VERIFY_OK");
            }
            _ => panic!("invalid VM probe mode"),
        }
        crate::credentials::clear_recovery_test_vault();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "native full recovery benchmark; dedicated credential session required"]
    fn native_vault_end_to_end_benchmark() {
        run_native_vault_file_benchmark(&[1024 * 1024, 16 * 1024 * 1024], &std::env::temp_dir());
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "requires disposable disk with at least 10 GiB free and unlocked native vault"]
    fn native_vault_large_file_benchmark() {
        assert_eq!(
            std::env::var("GHOSTFTP_LARGE_RECOVERY_BENCH_OPT_IN")
                .ok()
                .as_deref(),
            Some("disposable-fixtures")
        );
        let parent = PathBuf::from(
            std::env::var_os("GHOSTFTP_LARGE_RECOVERY_BENCH_ROOT")
                .expect("dedicated benchmark directory required"),
        );
        assert!(parent.is_absolute() && parent.is_dir());
        run_native_vault_file_benchmark(
            &[
                100 * 1024 * 1024,
                1024 * 1024 * 1024,
                4 * 1024 * 1024 * 1024 + 1,
            ],
            &parent,
        );
    }

    #[cfg(any(target_os = "linux", windows))]
    fn write_benchmark_pattern(path: &Path, size: u64, byte: u8) -> Result<()> {
        use std::io::Write;

        let mut file = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(path)?;
        let buffer = vec![byte; 1024 * 1024];
        let mut remaining = size;
        while remaining > 0 {
            let count = remaining.min(buffer.len() as u64) as usize;
            file.write_all(&buffer[..count])?;
            remaining -= count as u64;
        }
        file.sync_all()?;
        Ok(())
    }

    #[cfg(any(target_os = "linux", windows))]
    fn verify_benchmark_pattern(path: &Path, size: u64, byte: u8) -> Result<()> {
        use std::io::Read;

        let mut file = std::fs::File::open(path)?;
        anyhow::ensure!(file.metadata()?.len() == size, "benchmark size mismatch");
        let mut buffer = vec![0; 1024 * 1024];
        let mut remaining = size;
        while remaining > 0 {
            let count = remaining.min(buffer.len() as u64) as usize;
            file.read_exact(&mut buffer[..count])?;
            anyhow::ensure!(
                buffer[..count].iter().all(|value| *value == byte),
                "benchmark content mismatch"
            );
            remaining -= count as u64;
        }
        let mut extra = [0];
        anyhow::ensure!(file.read(&mut extra)? == 0, "benchmark file grew");
        Ok(())
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    fn streaming_benchmark_fixture_detects_corruption_and_truncation() {
        let root = std::env::temp_dir().join(format!("ghostftp-pattern-{}", Uuid::new_v4()));
        std::fs::create_dir(&root).unwrap();
        let path = root.join("fixture");
        let size = 1024 * 1024 + 1;
        write_benchmark_pattern(&path, size, 0x79).unwrap();
        verify_benchmark_pattern(&path, size, 0x79).unwrap();
        assert!(verify_benchmark_pattern(&path, size, 0x31).is_err());
        std::fs::OpenOptions::new()
            .write(true)
            .open(&path)
            .unwrap()
            .set_len(size - 1)
            .unwrap();
        assert!(verify_benchmark_pattern(&path, size, 0x79).is_err());
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(any(target_os = "linux", windows))]
    fn run_native_vault_file_benchmark(sizes: &[u64], parent: &Path) {
        let root = parent.join(format!("ghostftp-e2e-bench-{}", Uuid::new_v4()));
        std::fs::create_dir(&root).unwrap();
        let root = root.canonicalize().unwrap();
        crate::credentials::configure_native_recovery_test_vault(root.join("vault"), None);
        let result = std::panic::catch_unwind(|| {
            let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
            for &size in sizes {
                let mut timings = Vec::new();
                for sample in 0..3 {
                    let id = format!("bench-{size}-{sample}");
                    let target = root.join(format!("{id}-target"));
                    let stage = root.join(format!("{id}-stage"));
                    write_benchmark_pattern(&target, size, 0x31).unwrap();
                    write_benchmark_pattern(&stage, size, 0x79).unwrap();
                    #[cfg(target_os = "linux")]
                    crate::credentials::start_linux_helper_profile();
                    let started = std::time::Instant::now();
                    native_fixture_promote(&db, &stage, &target, &id).unwrap();
                    timings.push(started.elapsed().as_micros());
                    #[cfg(target_os = "linux")]
                    crate::credentials::finish_linux_helper_profile();
                    verify_benchmark_pattern(&target, size, 0x79).unwrap();
                    assert!(!stage.exists());
                    assert!(!download_backup_path(&target, &id).exists());
                    let _lock = db.recovery_lock().unwrap();
                    let ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
                    ledger.check_rows(&db).unwrap();
                    assert!(db.download_recovery_snapshot().unwrap().is_empty());
                    GlobalRecoveryLedger::collect_garbage(&db).unwrap();
                    std::fs::remove_file(target).unwrap();
                }
                timings.sort_unstable();
                eprintln!(
                    "recovery_e2e_benchmark platform={} bytes={size} samples=3 median_us={} max_us={}",
                    std::env::consts::OS, timings[1], timings[2]
                );
            }
        });
        crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault")).unwrap();
        crate::credentials::clear_recovery_test_vault();
        std::fs::remove_dir_all(root).unwrap();
        if let Err(error) = result {
            std::panic::resume_unwind(error);
        }
    }

    #[cfg(any(target_os = "linux", windows))]
    fn run_recovery_crash_matrix(native: bool) {
        const ROOT: &str = "GHOSTFTP_RECOVERY_PROTOCOL_TEST_ROOT";
        const POINT: &str = "GHOSTFTP_RECOVERY_PROTOCOL_TEST_POINT";

        if let Some(root) = std::env::var_os(ROOT) {
            let root = PathBuf::from(root);
            let point = std::env::var(POINT).unwrap();
            if native {
                crate::credentials::configure_native_recovery_test_vault(
                    root.join("vault"),
                    Some(point),
                );
            } else {
                crate::credentials::configure_recovery_test_vault(root.join("vault"), Some(point));
            }
            let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
            let stage = root.join("stage");
            let target = root.join("target");
            let backup = download_backup_path(&target, "fault");
            let guard = DownloadParentGuard::with_db(&target, Some(db.clone())).unwrap();
            let _lock = db.recovery_lock().unwrap();
            let journal = guard
                .recovery_journal(&stage, &target, "fault")
                .unwrap()
                .unwrap();
            #[cfg(target_os = "linux")]
            {
                let writer = std::fs::File::open(&stage).unwrap();
                promote_pinned_download(
                    &guard.handle,
                    stage.file_name().unwrap(),
                    target.file_name().unwrap(),
                    backup.file_name().unwrap(),
                    Some(&writer),
                    Some(&journal),
                )
                .unwrap();
            }
            #[cfg(windows)]
            {
                let writer =
                    windows_open_relative(&guard.handle, stage.file_name().unwrap(), 0x0001_0081)
                        .unwrap();
                windows_promote_with_backup(
                    &guard.handle,
                    &writer,
                    target.file_name().unwrap(),
                    backup.file_name().unwrap(),
                    Some(&journal),
                )
                .unwrap();
            }
            panic!("selected recovery checkpoint was not reached");
        }

        let isolated_points = [
            "key_registered",
            "transaction_key_written",
            "gc_registered",
            "blob_chunk_written",
            "blob_chunk_written@2",
            "blob_length_written",
            "manifest_written",
            "prepared_manifest_committed",
            "prepared_anchor_written",
            "prepared_sql_committed",
            "gc_page_cleared",
            "backup_renamed",
            "phase_manifest_committed",
            "phase_anchor_written",
            "phase_sql_committed",
            "gc_length_removed",
            "gc_chunk_removed",
            "stage_published",
            "phase_manifest_committed@2",
            "phase_anchor_written@2",
            "phase_sql_committed@2",
            "phase_manifest_committed@3",
            "phase_anchor_written@3",
            "phase_sql_committed@3",
            "backup_removed",
            "terminal_anchor_written",
            "terminal_sql_removed",
            "terminal_manifest_written",
            "gc_key_removed",
            "gc_anchor_removed",
            "rollback:phase_manifest_committed@2",
            "rollback:phase_anchor_written@2",
            "rollback:phase_sql_committed@2",
            "rollback:rollback_restored",
            "rollback:phase_manifest_committed@3",
            "rollback:phase_anchor_written@3",
            "rollback:phase_sql_committed@3",
            #[cfg(target_os = "linux")]
            "backup_directory_synced",
            #[cfg(target_os = "linux")]
            "published_directory_synced",
            #[cfg(target_os = "linux")]
            "cleanup_directory_synced",
            #[cfg(target_os = "linux")]
            "rollback:rollback_directory_synced",
            #[cfg(windows)]
            "backup_file_flushed",
            #[cfg(windows)]
            "published_file_flushed",
            #[cfg(windows)]
            "cleanup_file_flushed",
            #[cfg(windows)]
            "rollback:rollback_file_flushed",
        ];
        // The native smoke matrix is bounded independently of the exhaustive
        // isolated-backend matrix because Linux invokes a credential helper.
        let native_points = [
            "transaction_key_written",
            "prepared_manifest_committed",
            "phase_sql_committed",
            "terminal_anchor_written",
            "gc_key_removed",
            "rollback:phase_anchor_written@2",
            "rollback:rollback_restored",
            "rollback:phase_sql_committed@3",
        ];
        let points: &[&str] = if native {
            &native_points
        } else {
            &isolated_points
        };
        for &point in points {
            let started = std::time::Instant::now();
            let root = std::env::temp_dir().join(format!("ghostftp-crash-{}", Uuid::new_v4()));
            std::fs::create_dir_all(&root).unwrap();
            let root = root.canonicalize().unwrap();
            let stage = root.join("stage");
            let target = root.join("target");
            let backup = download_backup_path(&target, "fault");
            std::fs::write(&stage, b"new").unwrap();
            std::fs::write(&target, b"old").unwrap();
            std::fs::write(root.join("unrelated"), b"keep").unwrap();
            if native {
                crate::credentials::configure_native_recovery_test_vault(root.join("vault"), None);
            } else {
                crate::credentials::configure_recovery_test_vault(root.join("vault"), None);
            }
            {
                let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
                let _lock = db.recovery_lock().unwrap();
                GlobalRecoveryLedger::load(&db, true).unwrap();
            }
            let mut command = std::process::Command::new(std::env::current_exe().unwrap());
            if native {
                command
                    .arg("native_vault_recovery_survives_killed_process")
                    .arg("--ignored");
            } else {
                command.arg("recovery_protocol_survives_killed_process_at_durable_boundaries");
            }
            let mut child = command.env(ROOT, &root).env(POINT, point).spawn().unwrap();
            let mut reached = false;
            let attempts = if native { 2400 } else { 200 };
            for _ in 0..attempts {
                if std::fs::read_to_string(root.join("vault/checkpoint"))
                    .is_ok_and(|value| value == point)
                {
                    reached = true;
                    break;
                }
                if child.try_wait().unwrap().is_some() {
                    break;
                }
                std::thread::sleep(std::time::Duration::from_millis(50));
            }
            let killed = child.kill().is_ok();
            let status = child.wait().unwrap();
            assert!(reached && killed && !status.success(), "checkpoint {point}");
            let before = [&stage, &target, &backup].map(|path| std::fs::read(path).ok());
            {
                let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
                let _lock = db.recovery_lock().unwrap();
                let scope = db.recovery_scope().unwrap();
                let encoded =
                    crate::credentials::vault_get_strict(RECOVERY_INDEX_KEY_SERVICE, &scope)
                        .unwrap()
                        .unwrap();
                let (candidates, key_candidates) =
                    GlobalRecoveryLedger::with_key(&encoded, |key| {
                        let mut candidates = RecoveryGcPage::new();
                        for page in 0..RECOVERY_GC_PAGES {
                            candidates.extend(read_gc_page(&scope, page, key)?);
                        }
                        Ok((candidates, read_gc_keys(&scope, key)?))
                    })
                    .unwrap();
                let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
                ledger.reconcile_metadata(&db).unwrap();
                ledger.check_rows(&db).unwrap();
                GlobalRecoveryLedger::collect_garbage(&db).unwrap();
                // Recovery can be retried without changing the outcome.
                let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
                ledger.reconcile_metadata(&db).unwrap();
                GlobalRecoveryLedger::collect_garbage(&db).unwrap();
                ledger.check_rows(&db).unwrap();
                let mut live: std::collections::BTreeSet<_> =
                    ledger.entries.values().copied().collect();
                live.extend(
                    ledger
                        .shard_bytes()
                        .iter()
                        .map(|bytes| recovery_digest(bytes)),
                );
                for (digest, length) in candidates {
                    if live.contains(&digest) {
                        get_recovery_blob(&scope, &digest).unwrap();
                        continue;
                    }
                    let length_account = recovery_blob_account(&scope, &digest, "length");
                    assert!(crate::credentials::vault_get_strict(
                        RECOVERY_BLOB_SERVICE,
                        &length_account,
                    )
                    .unwrap()
                    .is_none());
                    for part in 0..length.div_ceil(RECOVERY_BLOB_CHUNK) {
                        let account = recovery_blob_account(&scope, &digest, &part.to_string());
                        assert!(crate::credentials::vault_get_strict(
                            RECOVERY_BLOB_SERVICE,
                            &account
                        )
                        .unwrap()
                        .is_none());
                    }
                }
                let live_keys: RecoveryGcKeys = db
                    .download_recovery_snapshot()
                    .unwrap()
                    .iter()
                    .map(|row| {
                        serde_json::from_str::<AuthenticatedRecovery>(&row.payload)
                            .unwrap()
                            .key_id
                    })
                    .collect();
                for id in key_candidates {
                    if !live_keys.contains(&id) {
                        for service in [RECOVERY_VAULT_SERVICE, RECOVERY_ANCHOR_SERVICE] {
                            assert!(
                                crate::credentials::vault_get_strict(service, &id.to_string())
                                    .unwrap()
                                    .is_none()
                            );
                        }
                    }
                }
                GlobalRecoveryLedger::with_key(&encoded, |key| {
                    for page in 0..RECOVERY_GC_PAGES {
                        assert!(read_gc_page(&scope, page, key)?.is_empty());
                    }
                    assert!(read_gc_keys(&scope, key)?.is_empty());
                    Ok(())
                })
                .unwrap();
                assert_eq!(
                    before,
                    [&stage, &target, &backup].map(|path| std::fs::read(path).ok()),
                    "metadata recovery changed user files at {point}"
                );
                let early = matches!(
                    point,
                    "key_registered"
                        | "transaction_key_written"
                        | "gc_registered"
                        | "blob_chunk_written"
                        | "blob_chunk_written@2"
                        | "blob_length_written"
                );
                let terminal = matches!(
                    point,
                    "terminal_anchor_written"
                        | "terminal_sql_removed"
                        | "terminal_manifest_written"
                        | "gc_key_removed"
                        | "gc_anchor_removed"
                );
                let rows = db.download_recovery_snapshot().unwrap();
                assert_eq!(rows.len(), if early || terminal { 0 } else { 1 }, "{point}");
                if point.starts_with("rollback:") {
                    let row = rows.first().unwrap();
                    let expected = if point.ends_with("@3") {
                        "rolled_back"
                    } else {
                        "rollback_pending"
                    };
                    assert_eq!(row.phase, expected, "{point}");
                    if point.ends_with("@3")
                        || point.ends_with("rollback_restored")
                        || point.ends_with("rollback_directory_synced")
                        || point.ends_with("rollback_file_flushed")
                    {
                        assert_eq!(std::fs::read(&target).unwrap(), b"old");
                        assert!(!backup.exists());
                    } else {
                        assert_eq!(std::fs::read(&backup).unwrap(), b"old");
                        assert!(!target.exists());
                    }
                    assert_eq!(std::fs::read(&stage).unwrap(), b"new");
                }
                if let Some(row) = rows.first() {
                    let (_, digest) = recovery_index_entry(
                        &row.id,
                        &row.phase,
                        row.destination_key.as_deref(),
                        &row.payload,
                    )
                    .unwrap();
                    get_recovery_blob(&db.recovery_scope().unwrap(), &digest).unwrap();
                }
            }
            assert_eq!(std::fs::read(root.join("unrelated")).unwrap(), b"keep");
            assert!(
                std::fs::read(&stage).is_ok_and(|bytes| bytes == b"new")
                    || std::fs::read(&target).is_ok_and(|bytes| bytes == b"new"),
                "new bytes lost at {point}"
            );
            if !matches!(
                point,
                "backup_removed"
                    | "cleanup_directory_synced"
                    | "cleanup_file_flushed"
                    | "terminal_anchor_written"
                    | "terminal_sql_removed"
                    | "terminal_manifest_written"
                    | "gc_key_removed"
                    | "gc_anchor_removed"
            ) {
                assert!(
                    std::fs::read(&target).is_ok_and(|bytes| bytes == b"old")
                        || std::fs::read(&backup).is_ok_and(|bytes| bytes == b"old"),
                    "original bytes lost at {point}"
                );
            }
            if native {
                crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault"))
                    .unwrap();
            }
            crate::credentials::clear_recovery_test_vault();
            std::fs::remove_dir_all(root).unwrap();
            eprintln!(
                "recovery checkpoint {point} (native={native}): {:?}",
                started.elapsed()
            );
        }
    }

    // These probes use the production OS credential adapter and explicit
    // CI-owned test directories. The service restart is orchestrated outside
    // the test process, so a new process verifies the persisted root.
    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "requires an isolated native vault session"]
    fn native_vault_restart_probe() {
        let root = PathBuf::from(std::env::var_os("GHOSTFTP_NATIVE_RESTART_ROOT").unwrap());
        let mode = std::env::var("GHOSTFTP_NATIVE_RESTART_MODE").unwrap();
        let root = root.canonicalize().unwrap();
        crate::credentials::configure_native_recovery_test_vault(root.join("vault"), None);
        if mode == "cleanup" {
            crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault")).unwrap();
            crate::credentials::clear_recovery_test_vault();
            return;
        }
        let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
        let _lock = db.recovery_lock().unwrap();
        if mode == "write" {
            let target = root.join("target");
            let guard = DownloadParentGuard::with_db(&target, Some(db.clone())).unwrap();
            let journal = guard
                .recovery_journal(&root.join("stage"), &target, "restart")
                .unwrap()
                .unwrap();
            // Metadata-only fixture: no file recovery is authorized or run.
            journal
                .begin((1, 2), (1, 3, 3), (1, 4, 3), "a".repeat(64), "b".repeat(64))
                .unwrap();
            journal
                .advance(RecoveryPhase::Prepared, RecoveryPhase::BackupSaved)
                .unwrap();
        } else {
            assert_eq!(mode, "read");
            let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
            ledger.reconcile_metadata(&db).unwrap();
            ledger.check_rows(&db).unwrap();
            let rows = db.download_recovery_snapshot().unwrap();
            assert_eq!(rows.len(), 1);
            assert_eq!(rows[0].id, "restart");
            assert_eq!(rows[0].phase, "backup_saved");
            let signed: AuthenticatedRecovery = serde_json::from_str(&rows[0].payload).unwrap();
            assert_eq!(
                signed.authenticate(&rows[0].id, &rows[0].payload).unwrap(),
                RecoveryPhase::BackupSaved
            );
            GlobalRecoveryLedger::collect_garbage(&db).unwrap();
        }
        crate::credentials::clear_recovery_test_vault();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "128 real prepared recovery journals; dedicated native vault session required"]
    fn native_vault_active_recovery_benchmark() {
        assert_eq!(
            std::env::var("GHOSTFTP_ACTIVE_RECOVERY_BENCH_OPT_IN")
                .ok()
                .as_deref(),
            Some("disposable-fixtures")
        );
        let root = std::env::temp_dir().join(format!("ghostftp-active-bench-{}", Uuid::new_v4()));
        std::fs::create_dir(&root).unwrap();
        let root = root.canonicalize().unwrap();
        crate::credentials::configure_native_recovery_test_vault(root.join("vault"), None);
        let result = std::panic::catch_unwind(|| {
            let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
            let mut admissions = Vec::new();
            for index in 0..128 {
                let id = format!("active-{index}");
                let target = root.join(format!("{id}-target"));
                let stage = root.join(format!("{id}-stage"));
                write_durable_fixture(&target, b"old").unwrap();
                write_durable_fixture(&stage, b"new").unwrap();
                let guard = DownloadParentGuard::with_db(&target, Some(db.clone())).unwrap();
                let journal = guard
                    .recovery_journal(&stage, &target, &id)
                    .unwrap()
                    .unwrap();
                let original = std::fs::File::open(&target).unwrap();
                let staged = std::fs::File::open(&stage).unwrap();
                let original_identity = recovery_file_identity(&original).unwrap();
                let stage_identity = recovery_file_identity(&staged).unwrap();
                #[cfg(target_os = "linux")]
                let parent_identity = {
                    use std::os::unix::fs::MetadataExt;
                    let metadata = guard.handle.metadata().unwrap();
                    (metadata.dev(), metadata.ino())
                };
                #[cfg(windows)]
                let parent_identity = {
                    let (volume, index) = windows_file_identity(&*guard.handle).unwrap();
                    (u64::from(volume), index)
                };
                let stage_hash = recovery_file_hash(&staged, stage_identity).unwrap();
                let original_hash = recovery_file_hash(&original, original_identity).unwrap();
                let started = std::time::Instant::now();
                let _lock = guard.check_recovery_destination(&target).unwrap();
                journal
                    .begin(
                        parent_identity,
                        stage_identity,
                        original_identity,
                        stage_hash,
                        original_hash,
                    )
                    .unwrap();
                admissions.push(started.elapsed().as_micros());
            }

            // Reopen the database to observe 128 persisted rows at once.
            drop(db);
            let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
            let _lock = db.recovery_lock().unwrap();
            let started = std::time::Instant::now();
            let mut ledger = GlobalRecoveryLedger::load(&db, false).unwrap();
            ledger.reconcile_metadata(&db).unwrap();
            ledger.check_rows(&db).unwrap();
            GlobalRecoveryLedger::collect_garbage(&db).unwrap();
            let rows = db.download_recovery_snapshot().unwrap();
            assert_eq!(rows.len(), 128);
            let reconcile_us = started.elapsed().as_micros();
            let mut inspections = Vec::new();
            let mut ids = std::collections::BTreeSet::new();
            for row in rows {
                let started = std::time::Instant::now();
                assert!(ids.insert(row.id.clone()));
                assert_eq!(row.phase, "prepared");
                let signed: AuthenticatedRecovery = serde_json::from_str(&row.payload).unwrap();
                assert_eq!(
                    signed.authenticate(&row.id, &row.payload).unwrap(),
                    RecoveryPhase::Prepared
                );
                let record: RecoveryInspectionRecord =
                    serde_json::from_str(&signed.payload).unwrap();
                assert!(record.matches_reservation(row.destination_key.as_deref()));
                assert_eq!(record.inspect(), RecoveryObservation::OriginalPresent);
                assert_eq!(std::fs::read(&record.target).unwrap(), b"old");
                assert_eq!(std::fs::read(&record.stage).unwrap(), b"new");
                assert!(!record.backup.exists());
                inspections.push(started.elapsed().as_micros());
            }
            assert_eq!(ids.len(), 128);
            ledger.check_rows(&db).unwrap();
            assert_eq!(db.download_recovery_snapshot().unwrap().len(), 128);
            for (phase, samples) in [
                ("admission", &mut admissions),
                ("inspection", &mut inspections),
            ] {
                samples.sort_unstable();
                let count = samples.len();
                eprintln!(
                    "recovery_active_benchmark platform={} phase={phase} samples={count} median_us={} p95_us={} p99_us={} max_us={} reconcile_us={reconcile_us}",
                    std::env::consts::OS,
                    samples[count / 2],
                    samples[(count * 95).div_ceil(100) - 1],
                    samples[(count * 99).div_ceil(100) - 1],
                    samples[count - 1]
                );
            }
        });
        crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault")).unwrap();
        crate::credentials::clear_recovery_test_vault();
        std::fs::remove_dir_all(root).unwrap();
        if let Err(error) = result {
            std::panic::resume_unwind(error);
        }
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    #[ignore = "native vault manifest benchmark; run on dedicated test session"]
    fn native_vault_manifest_benchmark() {
        for count in [1usize, 8, 32, 128] {
            let root = std::env::temp_dir().join(format!("ghostftp-bench-{}", Uuid::new_v4()));
            std::fs::create_dir_all(&root).unwrap();
            crate::credentials::configure_native_recovery_test_vault(root.join("vault"), None);
            let result = std::panic::catch_unwind(|| {
                let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
                let _lock = db.recovery_lock().unwrap();
                let mut ledger = GlobalRecoveryLedger::load(&db, true).unwrap();
                // Synthetic commitments measure authenticated manifest I/O,
                // not the full record, filesystem or GC workload.
                for i in 0..count {
                    let bytes = i.to_be_bytes();
                    ledger.entries.insert(
                        recovery_digest(&bytes),
                        recovery_digest(&[&bytes[..], b"proof"].concat()),
                    );
                }
                let started = std::time::Instant::now();
                ledger.save(&db).unwrap();
                let write_us = started.elapsed().as_micros();
                let mut reads = Vec::new();
                for _ in 0..5 {
                    let started = std::time::Instant::now();
                    let loaded = GlobalRecoveryLedger::load(&db, false).unwrap();
                    reads.push(started.elapsed().as_micros());
                    assert_eq!(loaded.entries, ledger.entries);
                }
                reads.sort_unstable();
                eprintln!(
                    "recovery_manifest_benchmark platform={} entries={count} write_us={write_us} read_median_us={} read_max_us={}",
                    std::env::consts::OS, reads[2], reads[4]
                );
            });
            crate::credentials::cleanup_native_recovery_test_vault(&root.join("vault")).unwrap();
            crate::credentials::clear_recovery_test_vault();
            std::fs::remove_dir_all(root).unwrap();
            if let Err(error) = result {
                std::panic::resume_unwind(error);
            }
        }
    }

    #[test]
    fn initialized_vault_cannot_silently_bootstrap_after_complete_loss() {
        let root = std::env::temp_dir().join(format!("ghostftp-vault-loss-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        crate::credentials::configure_recovery_test_vault(root.join("vault"), None);
        {
            let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
            let _lock = db.recovery_lock().unwrap();
            GlobalRecoveryLedger::load(&db, true).unwrap();
            let scope = db.recovery_scope().unwrap();
            assert!(db.recovery_vault_initialized(false).unwrap());
            crate::credentials::vault_delete_strict(RECOVERY_INDEX_SERVICE, &scope).unwrap();
            crate::credentials::vault_delete_strict(RECOVERY_INDEX_KEY_SERVICE, &scope).unwrap();
            assert!(GlobalRecoveryLedger::load(&db, false).is_err());
            assert!(GlobalRecoveryLedger::load(&db, true).is_err());
            assert!(
                crate::credentials::vault_get_strict(RECOVERY_INDEX_SERVICE, &scope)
                    .unwrap()
                    .is_none()
            );
        }
        crate::credentials::clear_recovery_test_vault();
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn recovery_gc_rejects_tampered_candidates_without_deleting_objects() {
        let root = std::env::temp_dir().join(format!("ghostftp-gc-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        crate::credentials::configure_recovery_test_vault(root.join("vault"), None);
        {
            let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
            let _lock = db.recovery_lock().unwrap();
            GlobalRecoveryLedger::load(&db, true).unwrap();
            let scope = db.recovery_scope().unwrap();
            let bytes = b"unreachable registered object";
            let digest = put_recovery_blob(&scope, bytes).unwrap();
            let account = gc_page_account(&scope, 0);
            let stored = crate::credentials::vault_get_strict(RECOVERY_GC_SERVICE, &account)
                .unwrap()
                .unwrap();
            let mut signed: AuthenticatedRecovery = serde_json::from_str(&stored).unwrap();
            signed.tag[0] ^= 1;
            crate::credentials::vault_set(
                RECOVERY_GC_SERVICE,
                &account,
                &serde_json::to_string(&signed).unwrap(),
            )
            .unwrap();
            assert!(GlobalRecoveryLedger::collect_garbage(&db).is_err());
            assert_eq!(get_recovery_blob(&scope, &digest).unwrap(), bytes);
            crate::credentials::vault_set(RECOVERY_GC_SERVICE, &account, &stored).unwrap();
            crate::credentials::vault_set(RECOVERY_BLOB_SERVICE, "foreign-account", "keep")
                .unwrap();
            GlobalRecoveryLedger::collect_garbage(&db).unwrap();
            assert!(get_recovery_blob(&scope, &digest).is_err());
            assert_eq!(
                crate::credentials::vault_get_strict(RECOVERY_BLOB_SERVICE, "foreign-account")
                    .unwrap()
                    .as_deref(),
                Some("keep")
            );
        }
        crate::credentials::clear_recovery_test_vault();
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn authenticated_recovery_rejects_modified_payload_id_key_and_tag() {
        let key = [0x51; 32];
        let mut record = AuthenticatedRecovery::seal(
            "transfer-a",
            "original intent".to_string(),
            Uuid::new_v4(),
            &key,
        )
        .unwrap();
        assert!(record.verify("transfer-a", &key).is_ok());
        assert!(record.verify("transfer-b", &key).is_err());
        assert!(record.verify("transfer-a", &[0x52; 32]).is_err());

        record.payload.push('!');
        assert!(record.verify("transfer-a", &key).is_err());
        record.payload.pop();
        assert!(record.verify("transfer-a", &key).is_ok());
        record.tag[0] ^= 1;
        assert!(record.verify("transfer-a", &key).is_err());
        record.tag[0] ^= 1;
        record.nonce[0] ^= 1;
        assert!(record.verify("transfer-a", &key).is_err());
        record.nonce[0] ^= 1;
        record.key_id = Uuid::new_v4();
        assert!(record.verify("transfer-a", &key).is_err());
    }

    #[test]
    fn global_index_detects_missing_unindexed_and_replayed_rows() {
        use crate::db::DownloadRecoverySnapshot;

        let row = |id: &str, phase: &str, payload: &str| DownloadRecoverySnapshot {
            id: id.into(),
            phase: phase.into(),
            payload: payload.into(),
            destination_key: Some("destination".into()),
        };
        let mut ledger = GlobalRecoveryLedger::default();
        let (id, proof) =
            recovery_index_entry("id", "backup_saved", Some("destination"), "current").unwrap();
        ledger.entries.insert(id, proof);
        let ledger = GlobalRecoveryLedger::decode(&ledger.encode()).unwrap();
        assert_eq!(
            ledger
                .differences(&[row("id", "backup_saved", "current")])
                .unwrap(),
            RecoveryIndexDifference::default()
        );
        assert_eq!(ledger.differences(&[]).unwrap().missing, 1);
        assert_eq!(
            ledger
                .differences(&[row("id", "prepared", "old")])
                .unwrap()
                .divergent,
            1
        );
        assert_eq!(
            ledger
                .differences(&[row("other", "prepared", "other")])
                .unwrap(),
            RecoveryIndexDifference {
                missing: 1,
                unindexed: 1,
                divergent: 0,
            }
        );
        assert!(GlobalRecoveryLedger::decode("invalid").is_err());
    }

    #[test]
    fn global_index_authenticated_envelope_rejects_tampering_and_wrong_scope() {
        let key = [0x42; 32];
        let mut ledger = GlobalRecoveryLedger::default();
        for index in 0..RECOVERY_SHARD_SIZE {
            let (id, proof) = recovery_index_entry(
                &index.to_string(),
                "prepared",
                Some("destination"),
                "payload",
            )
            .unwrap();
            ledger.entries.insert(id, proof);
        }
        let mut signed =
            AuthenticatedRecovery::seal("scope", ledger.encode(), Uuid::nil(), &key).unwrap();
        assert!(serde_json::to_string(&signed).unwrap().len() <= 1200);
        assert!(signed.verify("scope", &key).is_ok());
        assert!(signed.verify("other", &key).is_err());
        assert_eq!(
            GlobalRecoveryLedger::decode(&signed.payload)
                .unwrap()
                .entries
                .len(),
            RECOVERY_SHARD_SIZE
        );
        signed.payload.push('A');
        assert!(signed.verify("scope", &key).is_err());
    }

    #[test]
    fn sharded_index_round_trips_more_than_eight_entries() {
        let mut ledger = GlobalRecoveryLedger {
            generation: 42,
            ..GlobalRecoveryLedger::default()
        };
        for index in 0..RECOVERY_INDEX_LIMIT {
            let (id, proof) = recovery_index_entry(
                &index.to_string(),
                "prepared",
                Some("destination"),
                "payload",
            )
            .unwrap();
            ledger.entries.insert(id, proof);
        }
        let shards = ledger.shard_bytes();
        assert_eq!(shards.len(), 16);
        let digests: Vec<_> = shards.iter().map(|bytes| recovery_digest(bytes)).collect();
        let manifest = GlobalRecoveryLedger::manifest(ledger.generation, &digests).unwrap();
        let (generation, decoded) = GlobalRecoveryLedger::parse_manifest(&manifest).unwrap();
        assert_eq!(decoded, digests);
        let restored = GlobalRecoveryLedger::from_shards(generation, shards.clone()).unwrap();
        assert_eq!(restored.entries, ledger.entries);
        assert_eq!(restored.generation, 42);
        let mut duplicated = shards;
        duplicated[1] = duplicated[0].clone();
        assert!(GlobalRecoveryLedger::from_shards(generation, duplicated).is_err());
        let signed = AuthenticatedRecovery::seal("scope", manifest, Uuid::nil(), &[7; 32]).unwrap();
        assert!(serde_json::to_string(&signed).unwrap().len() <= 1200);
    }

    #[test]
    fn recovery_anchor_reconciliation_advances_only_one_authenticated_phase() {
        let key = [3; 32];
        let key_id = Uuid::new_v4();
        let mut signed = AuthenticatedRecovery::seal("id", "intent".into(), key_id, &key).unwrap();
        signed
            .append_phase("id", RecoveryPhase::Prepared, &key)
            .unwrap();
        let old = serde_json::to_string(&signed).unwrap();
        let anchor =
            AuthenticatedRecovery::seal("id", recovery_anchor_payload(&old, false), key_id, &key)
                .unwrap();
        signed
            .advance_signed(
                "id",
                RecoveryPhase::Prepared,
                RecoveryPhase::BackupSaved,
                &key,
            )
            .unwrap();
        let current = serde_json::to_string(&signed).unwrap();
        let previous = (RecoveryPhase::Prepared, old);
        assert_eq!(
            signed
                .classify_anchor(
                    "id",
                    &current,
                    RecoveryPhase::BackupSaved,
                    Some(&previous),
                    Some(&anchor),
                    &key,
                )
                .unwrap(),
            RecoveryAnchorState::NeedsAdvance
        );
        assert!(signed
            .classify_anchor(
                "id",
                &current,
                RecoveryPhase::BackupSaved,
                None,
                Some(&anchor),
                &key,
            )
            .is_err());
        assert!(signed
            .classify_anchor("id", &current, RecoveryPhase::BackupSaved, None, None, &key)
            .is_err());
    }

    #[test]
    fn independent_anchor_rejects_old_envelope_and_terminal_replay() {
        let key = [0x23; 32];
        let key_id = Uuid::new_v4();
        let mut record = AuthenticatedRecovery::seal("id", "intent".into(), key_id, &key).unwrap();
        record
            .append_phase("id", RecoveryPhase::Prepared, &key)
            .unwrap();
        let old = serde_json::to_string(&record).unwrap();
        record
            .advance_signed(
                "id",
                RecoveryPhase::Prepared,
                RecoveryPhase::BackupSaved,
                &key,
            )
            .unwrap();
        let current = serde_json::to_string(&record).unwrap();
        let anchor = AuthenticatedRecovery::seal(
            "id",
            recovery_anchor_payload(&current, false),
            key_id,
            &key,
        )
        .unwrap();
        assert!(verify_recovery_anchor(&anchor, "id", key_id, &current, &key).is_ok());
        assert!(verify_recovery_anchor(&anchor, "id", key_id, &old, &key).is_err());
        assert!(verify_recovery_anchor(&anchor, "other", key_id, &current, &key).is_err());
        let terminal = AuthenticatedRecovery::seal(
            "id",
            recovery_anchor_payload(&current, true),
            key_id,
            &key,
        )
        .unwrap();
        assert!(verify_recovery_anchor(&terminal, "id", key_id, &current, &key).is_err());
    }

    #[test]
    fn phase_chain_rejects_mutation_reordering_and_cross_transaction_reuse() {
        let key = [0x51; 32];
        let mut record =
            AuthenticatedRecovery::seal("id", "intent".into(), Uuid::new_v4(), &key).unwrap();
        assert!(record.verify_phases("id", &key).is_err());
        record
            .append_phase("id", RecoveryPhase::Prepared, &key)
            .unwrap();
        assert!(record
            .advance_signed("id", RecoveryPhase::Prepared, RecoveryPhase::Promoted, &key,)
            .is_err());
        record
            .advance_signed(
                "id",
                RecoveryPhase::Prepared,
                RecoveryPhase::BackupSaved,
                &key,
            )
            .unwrap();
        record
            .advance_signed(
                "id",
                RecoveryPhase::BackupSaved,
                RecoveryPhase::Promoted,
                &key,
            )
            .unwrap();
        record
            .advance_signed(
                "id",
                RecoveryPhase::Promoted,
                RecoveryPhase::CleanupPending,
                &key,
            )
            .unwrap();
        assert_eq!(
            record.verify_phases("id", &key).unwrap(),
            RecoveryPhase::CleanupPending
        );
        assert!(record.verify_phases("other", &key).is_err());
        record.phases[1].phase = "promoted".into();
        assert!(record.verify_phases("id", &key).is_err());
        record.phases[1].phase = "backup_saved".into();
        record.phases.swap(1, 2);
        assert!(record.verify_phases("id", &key).is_err());
        record.phases.swap(1, 2);
        record.phases[3].tag[0] ^= 1;
        assert!(record.verify_phases("id", &key).is_err());
        record.phases[3].tag[0] ^= 1;
        record.phases.pop();
        // A valid earlier prefix remains cryptographically valid: the SQL
        // phase comparison rejects partial rollback, not whole-store replay.
        assert_eq!(
            record.verify_phases("id", &key).unwrap(),
            RecoveryPhase::Promoted
        );
    }

    #[test]
    fn recovery_decisions_never_authorize_untrusted_observations() {
        use RecoveryObservation::*;

        for observed in [
            OriginalPresent,
            BackupPresent,
            PublishedWithBackup,
            PublishedWithoutBackup,
            OriginalWithoutStage,
            BackupWithoutStage,
            Ambiguous,
            Unavailable,
        ] {
            assert_eq!(
                recovery_decision(false, observed),
                RecoveryDecision::ManualReview
            );
        }
        assert_eq!(
            recovery_decision(true, BackupPresent),
            RecoveryDecision::RollbackCandidate
        );
        assert_eq!(
            recovery_decision(true, PublishedWithBackup),
            RecoveryDecision::CleanupCandidate
        );
        assert_eq!(
            recovery_decision(true, PublishedWithoutBackup),
            RecoveryDecision::InspectOnly
        );
        assert_eq!(
            recovery_decision(true, Ambiguous),
            RecoveryDecision::ManualReview
        );
    }

    #[test]
    fn recovery_classifies_layout_independently_of_last_recorded_phase() {
        use EntryObservation::{Missing, Original, Other, Stage, Unavailable};
        use RecoveryObservation::{
            Ambiguous, BackupPresent, BackupWithoutStage, OriginalPresent, OriginalWithoutStage,
            PublishedWithBackup, PublishedWithoutBackup,
        };
        for (target, stage, backup, expected) in [
            (Original, Stage, Missing, OriginalPresent),
            (Missing, Stage, Original, BackupPresent),
            (Stage, Missing, Original, PublishedWithBackup),
            (Stage, Missing, Missing, PublishedWithoutBackup),
            (Original, Missing, Missing, OriginalWithoutStage),
            (Missing, Missing, Original, BackupWithoutStage),
            (Other, Stage, Original, Ambiguous),
            (
                Missing,
                Unavailable,
                Original,
                RecoveryObservation::Unavailable,
            ),
        ] {
            assert_eq!(classify_recovery(target, stage, backup), expected);
        }
        assert!(observation_matches_phase("prepared", OriginalPresent));
        assert!(!observation_matches_phase("prepared", BackupPresent));
        assert!(!observation_matches_phase(
            "backup_saved",
            PublishedWithBackup
        ));
        assert!(observation_matches_phase("rollback_pending", BackupPresent));
        assert!(observation_matches_phase("rolled_back", OriginalPresent));
        assert!(observation_matches_phase(
            "cleanup_pending",
            PublishedWithoutBackup
        ));
        assert!(!observation_matches_phase(
            "legacy_unknown",
            OriginalPresent
        ));
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    fn reconciliation_observes_backup_promotion_and_substitution_without_mutation() {
        fn identity(path: &Path) -> (u64, u64, u64) {
            let file = std::fs::File::open(path).unwrap();
            let metadata = file.metadata().unwrap();
            #[cfg(target_os = "linux")]
            {
                use std::os::unix::fs::MetadataExt;
                (metadata.dev(), metadata.ino(), metadata.len())
            }
            #[cfg(windows)]
            {
                let (volume, index) = windows_file_identity(&file).unwrap();
                (u64::from(volume), index, metadata.len())
            }
        }

        let root = std::env::temp_dir().join(format!("ghostftp-reconcile-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        let target = root.join("target");
        let stage = root.join("stage");
        let backup = root.join("backup");
        std::fs::write(&target, b"old").unwrap();
        std::fs::write(&stage, b"new").unwrap();
        let parent = open_download_directory(&root).unwrap();
        #[cfg(target_os = "linux")]
        let parent_identity = {
            use std::os::unix::fs::MetadataExt;
            let metadata = parent.metadata().unwrap();
            (metadata.dev(), metadata.ino())
        };
        #[cfg(windows)]
        let parent_identity = {
            let (volume, index) = windows_file_identity(&parent).unwrap();
            (u64::from(volume), index)
        };
        drop(parent);
        let record = RecoveryInspectionRecord {
            schema: 2,
            parent: root.clone(),
            stage: stage.clone(),
            target: target.clone(),
            backup: backup.clone(),
            parent_identity,
            stage_identity: identity(&stage),
            original_identity: identity(&target),
            stage_sha256: recovery_file_hash(
                &std::fs::File::open(&stage).unwrap(),
                identity(&stage),
            )
            .unwrap(),
            original_sha256: recovery_file_hash(
                &std::fs::File::open(&target).unwrap(),
                identity(&target),
            )
            .unwrap(),
        };
        let reservation = recovery_destination_key(parent_identity, &target).unwrap();
        assert!(record.matches_reservation(Some(&reservation)));
        assert!(!record.matches_reservation(None));
        assert!(!record.matches_reservation(Some("substituted")));
        let other_reservation = recovery_destination_key(parent_identity, &backup).unwrap();
        assert!(!record.matches_reservation(Some(&other_reservation)));
        assert_eq!(record.inspect(), RecoveryObservation::OriginalPresent);
        std::fs::write(&stage, b"bad").unwrap();
        assert_eq!(record.inspect(), RecoveryObservation::Ambiguous);
        std::fs::write(&stage, b"new").unwrap();
        std::fs::rename(&target, &backup).unwrap();
        assert_eq!(record.inspect(), RecoveryObservation::BackupPresent);
        std::fs::write(&backup, b"bad").unwrap();
        assert_eq!(record.inspect(), RecoveryObservation::Ambiguous);
        std::fs::write(&backup, b"old").unwrap();
        assert!(!target.exists());
        assert_eq!(std::fs::read(&backup).unwrap(), b"old");
        std::fs::rename(&stage, &target).unwrap();
        assert_eq!(record.inspect(), RecoveryObservation::PublishedWithBackup);
        assert_eq!(std::fs::read(&target).unwrap(), b"new");
        assert_eq!(std::fs::read(&backup).unwrap(), b"old");
        std::fs::remove_file(&backup).unwrap();
        assert_eq!(
            record.inspect(),
            RecoveryObservation::PublishedWithoutBackup
        );
        std::fs::rename(&target, root.join("moved")).unwrap();
        std::fs::write(&target, b"bad").unwrap();
        assert_eq!(record.inspect(), RecoveryObservation::Ambiguous);
        assert_eq!(std::fs::read(&target).unwrap(), b"bad");
        assert_eq!(std::fs::read(root.join("moved")).unwrap(), b"new");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[tokio::test]
    async fn failed_stage_cleanup_requires_exclusive_creation_evidence() {
        let root = std::env::temp_dir().join(format!("ghostftp-owned-stage-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let target = root.join("target");
        let stage = root.join("stage");
        let parent = DownloadParentGuard::open(&target).unwrap();
        let owner = DownloadStageOwner::default();
        tokio::fs::write(&stage, b"unowned").await.unwrap();
        assert!(owner.create(&stage).await.is_err());
        owner.cleanup(&parent, &stage).await;
        assert_eq!(tokio::fs::read(&stage).await.unwrap(), b"unowned");
        tokio::fs::remove_file(&stage).await.unwrap();

        let file = owner.create(&stage).await.unwrap();
        drop(file);
        owner.cleanup(&parent, &stage).await;
        assert!(!stage.exists());
        drop(parent);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[tokio::test]
    async fn overwrite_recovery_intent_preserves_conflicting_backup() {
        let root = std::env::temp_dir().join(format!("ghostftp-recovery-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let db = std::sync::Arc::new(crate::db::Db::open(&root.join("ghostftp.db")).unwrap());
        let stage = root.join("stage");
        let target = root.join("target");
        let backup = download_backup_path(&target, "recovery");
        let guard = DownloadParentGuard::with_db(&target, Some(db.clone())).unwrap();
        tokio::fs::write(&stage, b"new").await.unwrap();
        tokio::fs::write(&target, b"old").await.unwrap();
        tokio::fs::write(&backup, b"keep").await.unwrap();
        let verified = verify_staged_download_bytes(&stage, 3).await.unwrap();
        let vault_ready = {
            let _lock = db.recovery_lock().unwrap();
            GlobalRecoveryLedger::load(&db, true).is_ok()
        };

        assert!(guard
            .promote(&stage, &target, "recovery", true, &verified)
            .await
            .is_err());
        let rows = db.download_recovery_list().unwrap();
        if rows.is_empty() {
            // A headless or locked vault must now refuse journal-backed
            // overwrite before moving the original. Do not claim collision QA.
            assert!(!vault_ready || GlobalRecoveryLedger::load(&db, false).is_err());
            assert_eq!(tokio::fs::read(&target).await.unwrap(), b"old");
            assert_eq!(tokio::fs::read(&backup).await.unwrap(), b"keep");
            assert_eq!(tokio::fs::read(&stage).await.unwrap(), b"new");
            drop(verified);
            drop(guard);
            drop(db);
            tokio::fs::remove_dir_all(root).await.unwrap();
            return;
        }
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].0, "recovery");
        assert_eq!(db.download_recovery_phase("recovery").unwrap(), "prepared");
        let payload = serde_json::from_str::<AuthenticatedRecovery>(&rows[0].1)
            .map(|signed| signed.payload)
            .unwrap_or_else(|_| rows[0].1.clone());
        let record: serde_json::Value = serde_json::from_str(&payload).unwrap();
        assert_eq!(record["schema"], 2);
        assert_eq!(record["stage_sha256"].as_str().unwrap().len(), 64);
        assert_eq!(record["original_sha256"].as_str().unwrap().len(), 64);
        assert_ne!(record["stage_sha256"], record["original_sha256"]);
        assert!(record["original_identity"].is_array());
        assert!(guard
            .promote(&stage, &target, "different-id", false, &verified)
            .await
            .is_err());
        assert!(guard
            .promote_delta(&stage, &target, &verified)
            .await
            .is_err());
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"old");
        assert_eq!(tokio::fs::read(&backup).await.unwrap(), b"keep");
        assert_eq!(tokio::fs::read(&stage).await.unwrap(), b"new");
        drop(verified);
        drop(guard);
        drop(db);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn owned_creation_uses_initial_directory_after_path_replacement() {
        let root = std::env::temp_dir().join(format!("ghostftp-openat-{}", Uuid::new_v4()));
        let parent = root.join("parent");
        let moved = root.join("moved");
        std::fs::create_dir_all(&parent).unwrap();
        let guard = DownloadParentGuard::open(&parent.join("target")).unwrap();
        let owner = DownloadStageOwner::for_parent(&guard);
        std::fs::rename(&parent, &moved).unwrap();
        std::fs::create_dir(&parent).unwrap();
        std::fs::write(parent.join("stage"), b"keep").unwrap();

        let mut file = owner.create_std(&parent.join("stage")).unwrap();
        std::io::Write::write_all(&mut file, b"new").unwrap();
        assert_eq!(std::fs::read(parent.join("stage")).unwrap(), b"keep");
        assert_eq!(std::fs::read(moved.join("stage")).unwrap(), b"new");
        assert!(owner.create_std(&parent.join("stage")).is_err());
        drop(file);
        drop(owner);
        drop(guard);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(windows)]
    #[test]
    fn native_windows_promotion_failure_restores_original() {
        use std::ffi::OsStr;

        let root = std::env::temp_dir().join(format!("ghostftp-win-rollback-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        std::fs::write(root.join("stage"), b"new").unwrap();
        std::fs::write(root.join("target"), b"old").unwrap();
        let parent = open_download_directory(&root).unwrap();
        // This file has no DELETE right: the second rename must fail after
        // backing up the original, forcing rollback through its live handle.
        let source = windows_open_relative(&parent, OsStr::new("stage"), 0x80).unwrap();
        let error = windows_promote_with_backup(
            &parent,
            &source,
            OsStr::new("target"),
            OsStr::new("backup"),
            None,
        )
        .unwrap_err();
        assert!(format!("{error:#}").contains("original restored"));
        assert_eq!(std::fs::read(root.join("target")).unwrap(), b"old");
        assert_eq!(std::fs::read(root.join("stage")).unwrap(), b"new");
        assert!(!root.join("backup").exists());
        drop(source);
        drop(parent);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(windows)]
    #[tokio::test]
    async fn native_windows_overwrite_preserves_existing_backup() {
        let root = std::env::temp_dir().join(format!("ghostftp-win-backup-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let stage = root.join("stage");
        let target = root.join("target");
        let backup = download_backup_path(&target, "native");
        let guard = DownloadParentGuard::open(&target).unwrap();
        tokio::fs::write(&stage, b"new").await.unwrap();
        tokio::fs::write(&target, b"old").await.unwrap();
        tokio::fs::write(&backup, b"recovery").await.unwrap();
        let verified = verify_staged_download_bytes(&stage, 3).await.unwrap();
        assert!(guard
            .promote(&stage, &target, "native", true, &verified)
            .await
            .is_err());
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"old");
        assert_eq!(tokio::fs::read(&backup).await.unwrap(), b"recovery");
        tokio::fs::remove_file(&backup).await.unwrap();
        guard
            .promote(&stage, &target, "native", true, &verified)
            .await
            .unwrap();
        drop(verified);
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"new");
        assert!(!backup.exists());
        drop(guard);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(windows)]
    #[tokio::test]
    async fn native_windows_no_replace_and_disposition_preserve_other_files() {
        let root = std::env::temp_dir().join(format!("ghostftp-win-handle-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let staging = root.join("stage");
        let target = root.join("target");
        let guard = DownloadParentGuard::open(&target).unwrap();
        tokio::fs::write(&staging, b"good").await.unwrap();
        tokio::fs::write(&target, b"keep").await.unwrap();
        let verified = verify_staged_download_bytes(&staging, 4).await.unwrap();

        assert!(guard
            .promote(&staging, &target, "collision", false, &verified)
            .await
            .is_err());
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"keep");
        assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"good");
        guard.cleanup_verified(&staging, &verified).await;
        drop(verified);
        assert!(!staging.exists());
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"keep");

        tokio::fs::write(&staging, b"new").await.unwrap();
        let destination = root.join("new-target");
        let verified = verify_staged_download_bytes(&staging, 3).await.unwrap();
        guard
            .promote(&staging, &destination, "new", false, &verified)
            .await
            .unwrap();
        drop(verified);
        assert!(!staging.exists());
        assert_eq!(tokio::fs::read(&destination).await.unwrap(), b"new");
        drop(guard);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(target_os = "linux")]
    #[tokio::test]
    async fn verified_stage_substitution_preserves_original_destination() {
        let root = std::env::temp_dir().join(format!("ghostftp-verified-stage-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let target = root.join("target");
        let staging = root.join("stage");
        let moved = root.join("verified");
        tokio::fs::write(&target, b"original").await.unwrap();
        tokio::fs::write(&staging, b"good").await.unwrap();
        let guard = DownloadParentGuard::open(&target).unwrap();
        let verified = verify_staged_download_bytes(&staging, 4).await.unwrap();
        tokio::fs::rename(&staging, &moved).await.unwrap();
        tokio::fs::write(&staging, b"fake").await.unwrap();

        assert!(guard
            .promote(&staging, &target, "test", true, &verified)
            .await
            .is_err());
        guard.cleanup_verified(&staging, &verified).await;
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"original");
        assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"fake");
        assert_eq!(tokio::fs::read(&moved).await.unwrap(), b"good");
        assert!(!download_backup_path(&target, "test").exists());
        drop(verified);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[tokio::test]
    async fn delta_promotion_rejects_replaced_verified_output() {
        let root = std::env::temp_dir().join(format!("ghostftp-delta-identity-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let target = root.join("target");
        let staging = root.join("stage");
        let moved = root.join("verified");
        tokio::fs::write(&target, b"original").await.unwrap();
        tokio::fs::write(&staging, b"good").await.unwrap();
        let guard = DownloadParentGuard::open(&target).unwrap();
        let verified = verify_staged_download_bytes(&staging, 4).await.unwrap();
        tokio::fs::rename(&staging, &moved).await.unwrap();
        tokio::fs::write(&staging, b"fake").await.unwrap();

        assert!(guard
            .promote_delta(&staging, &target, &verified)
            .await
            .is_err());
        guard.cleanup_verified(&staging, &verified).await;
        assert_eq!(tokio::fs::read(&target).await.unwrap(), b"original");
        assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"fake");
        assert_eq!(tokio::fs::read(&moved).await.unwrap(), b"good");
        drop(verified);
        drop(guard);
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(any(unix, windows))]
    #[tokio::test]
    async fn resumable_staging_refuses_hard_links_before_truncation() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-resume-hardlink-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let original = root.join("original");
        let staging = root.join("stage");
        tokio::fs::write(&original, b"valuable").await.unwrap();
        tokio::fs::hard_link(&original, &staging).await.unwrap();

        let error = open_resumable_download_staging(&staging).await.unwrap_err();
        cleanup_failed_download_staging(&staging, &error).await;
        assert_eq!(tokio::fs::read(&original).await.unwrap(), b"valuable");
        assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"valuable");
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn resumable_staging_refuses_symlink_before_truncation() {
        let root = std::env::temp_dir().join(format!("ghostftp-resume-link-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let original = root.join("original");
        let staging = root.join("stage");
        tokio::fs::write(&original, b"valuable").await.unwrap();
        std::os::unix::fs::symlink(&original, &staging).unwrap();

        let error = open_resumable_download_staging(&staging).await.unwrap_err();
        cleanup_failed_download_staging(&staging, &error).await;
        assert_eq!(tokio::fs::read(&original).await.unwrap(), b"valuable");
        assert!(tokio::fs::symlink_metadata(&staging)
            .await
            .unwrap()
            .file_type()
            .is_symlink());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[test]
    fn agent_delta_patch_cannot_clobber_preexisting_output() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-delta-upload-excl-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        let patch = root.join("user-owned.patch");
        std::fs::write(&patch, b"valuable existing data").unwrap();
        let error = create_new_delta_upload_patch(&patch).unwrap_err();
        assert!(error.downcast_ref::<DownloadStagingOpenFailure>().is_some());
        assert_eq!(std::fs::read(&patch).unwrap(), b"valuable existing data");
        let free = root.join("new.patch");
        let file = create_new_delta_upload_patch(&free).unwrap();
        drop(file);
        assert!(std::fs::metadata(&free).unwrap().is_file());
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(unix)]
    #[test]
    fn agent_delta_patch_cannot_follow_or_remove_symlink() {
        use std::os::unix::fs::symlink;
        let root =
            std::env::temp_dir().join(format!("ghostftp-delta-upload-link-{}", Uuid::new_v4()));
        std::fs::create_dir_all(&root).unwrap();
        let valuable = root.join("valuable");
        let patch = root.join("planted.patch");
        std::fs::write(&valuable, b"keep my bytes").unwrap();
        symlink(&valuable, &patch).unwrap();
        assert!(create_new_delta_upload_patch(&patch).is_err());
        assert!(std::fs::symlink_metadata(&patch)
            .unwrap()
            .file_type()
            .is_symlink());
        assert_eq!(std::fs::read(&valuable).unwrap(), b"keep my bytes");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[tokio::test]
    async fn agent_delta_patch_cleanup_removes_only_an_owned_regular_file() {
        let root = std::env::temp_dir().join(format!("ghostftp-delta-cleanup-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let patch = root.join("planned.patch");
        let mut file = create_new_delta_upload_patch(&patch).unwrap();
        use std::io::Write as _;
        file.write_all(b"partial patch").unwrap();
        drop(file);
        cleanup_owned_regular_staging_file(&patch).await;
        assert!(tokio::fs::symlink_metadata(&patch).await.is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn agent_delta_patch_cleanup_never_unlinks_a_substituted_symlink() {
        use std::os::unix::fs::symlink;
        let root =
            std::env::temp_dir().join(format!("ghostftp-delta-cleanup-link-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let valuable = root.join("valuable.bin");
        let patch = root.join("planned.patch");
        tokio::fs::write(&valuable, b"keep this file")
            .await
            .unwrap();
        symlink(&valuable, &patch).unwrap();
        cleanup_owned_regular_staging_file(&patch).await;
        assert!(tokio::fs::symlink_metadata(&patch)
            .await
            .unwrap()
            .file_type()
            .is_symlink());
        assert_eq!(tokio::fs::read(&valuable).await.unwrap(), b"keep this file");
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[tokio::test]
    async fn failed_exclusive_staging_open_never_deletes_existing_file() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-staging-ownership-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let path = root.join("original.part");
        tokio::fs::write(&path, b"keep original").await.unwrap();
        let error = create_new_download_staging(&path).await.unwrap_err();
        assert!(error.downcast_ref::<DownloadStagingOpenFailure>().is_some());
        cleanup_failed_download_staging(&path, &error).await;
        assert_eq!(tokio::fs::read(&path).await.unwrap(), b"keep original");
        let owned = root.join("owned.part");
        let mut file = create_new_download_staging(&owned).await.unwrap();
        use tokio::io::AsyncWriteExt as _;
        file.write_all(b"incomplete transfer").await.unwrap();
        drop(file);
        cleanup_failed_download_staging(&owned, &anyhow::anyhow!("network read failed")).await;
        assert!(tokio::fs::symlink_metadata(&owned).await.is_err());
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn failed_exclusive_staging_open_keeps_preplanted_symlink() {
        use std::os::unix::fs::symlink;
        let root = std::env::temp_dir().join(format!("ghostftp-staging-link-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let valuable = root.join("valuable.txt");
        let planted = root.join("planted.part");
        tokio::fs::write(&valuable, b"not transferred")
            .await
            .unwrap();
        symlink(&valuable, &planted).unwrap();
        let error = create_new_download_staging(&planted).await.unwrap_err();
        cleanup_failed_download_staging(&planted, &error).await;
        assert!(tokio::fs::symlink_metadata(&planted)
            .await
            .unwrap()
            .file_type()
            .is_symlink());
        assert_eq!(
            tokio::fs::read(&valuable).await.unwrap(),
            b"not transferred"
        );
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(windows)]
    #[tokio::test]
    async fn windows_staging_identity_distinguishes_equal_length_files() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-staging-identity-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let first = root.join("first.part");
        let second = root.join("second.part");
        tokio::fs::write(&first, b"same").await.unwrap();
        tokio::fs::write(&second, b"same").await.unwrap();
        let a = tokio::fs::File::open(&first).await.unwrap();
        let reopened = tokio::fs::File::open(&first).await.unwrap();
        let b = tokio::fs::File::open(&second).await.unwrap();
        assert_eq!(
            windows_staging_identity(&a).unwrap(),
            windows_staging_identity(&reopened).unwrap()
        );
        assert_ne!(
            windows_staging_identity(&a).unwrap(),
            windows_staging_identity(&b).unwrap()
        );
        drop((a, reopened, b));
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(any(unix, windows))]
    #[tokio::test]
    async fn hard_linked_staging_must_not_replace_existing_destination() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-staging-hardlink-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let original = root.join("existing.bin");
        let unrelated = root.join("unrelated.bin");
        let staging = root.join("staged.part");
        tokio::fs::write(&original, b"original").await.unwrap();
        tokio::fs::write(&unrelated, b"unrelated").await.unwrap();
        tokio::fs::hard_link(&unrelated, &staging).await.unwrap();

        assert!(commit_verified_delta_output(&staging, &original, 9)
            .await
            .is_err());
        assert_eq!(tokio::fs::read(&original).await.unwrap(), b"original");
        assert_eq!(tokio::fs::read(&unrelated).await.unwrap(), b"unrelated");
        assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"unrelated");
        tokio::fs::remove_dir_all(root).await.unwrap();
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn pinned_sibling_rename_does_not_follow_replaced_parent() {
        use std::ffi::OsStr;

        let root = std::env::temp_dir().join(format!("ghostftp-pinned-parent-{}", Uuid::new_v4()));
        let parent = root.join("parent");
        let moved = root.join("moved");
        std::fs::create_dir_all(&parent).unwrap();
        std::fs::write(parent.join("stage.part"), b"verified").unwrap();
        let pinned = std::fs::File::open(&parent).unwrap();
        std::fs::rename(&parent, &moved).unwrap();
        std::fs::create_dir(&parent).unwrap();
        std::fs::write(parent.join("stage.part"), b"unrelated").unwrap();

        rename_pinned_siblings(&pinned, OsStr::new("stage.part"), OsStr::new("final.bin")).unwrap();
        assert_eq!(std::fs::read(moved.join("final.bin")).unwrap(), b"verified");
        assert_eq!(
            std::fs::read(parent.join("stage.part")).unwrap(),
            b"unrelated"
        );
        assert!(!parent.join("final.bin").exists());
        std::fs::write(moved.join("stage.part"), b"second").unwrap();
        assert!(
            rename_pinned_siblings(&pinned, OsStr::new("stage.part"), OsStr::new("final.bin"))
                .is_err()
        );
        assert_eq!(std::fs::read(moved.join("final.bin")).unwrap(), b"verified");
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(any(target_os = "linux", windows))]
    #[test]
    fn download_parent_guard_rejects_replaced_directory() {
        let root = std::env::temp_dir().join(format!("ghostftp-parent-guard-{}", Uuid::new_v4()));
        let parent = root.join("parent");
        let moved = root.join("moved");
        std::fs::create_dir_all(&parent).unwrap();
        let guard = DownloadParentGuard::open(&parent.join("target.bin")).unwrap();
        guard.verify().unwrap();
        std::fs::rename(&parent, &moved).unwrap();
        std::fs::create_dir(&parent).unwrap();
        assert!(guard.verify().is_err());
        std::fs::remove_dir(&parent).unwrap();
        std::fs::rename(&moved, &parent).unwrap();
        guard.verify().unwrap();
        drop(guard);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn pinned_promotion_preserves_replacement_directory_and_backup() {
        use std::ffi::OsStr;

        let root = std::env::temp_dir().join(format!("ghostftp-pinned-promote-{}", Uuid::new_v4()));
        let parent = root.join("parent");
        let moved = root.join("moved");
        std::fs::create_dir_all(&parent).unwrap();
        std::fs::write(parent.join("stage.part"), b"new").unwrap();
        std::fs::write(parent.join("final.bin"), b"old").unwrap();
        let pinned = pin_download_parent(&parent).unwrap();
        std::fs::rename(&parent, &moved).unwrap();
        std::fs::create_dir(&parent).unwrap();
        std::fs::write(parent.join("final.bin"), b"unrelated").unwrap();
        std::fs::write(parent.join("backup"), b"keep").unwrap();

        promote_pinned_download(
            &pinned,
            OsStr::new("stage.part"),
            OsStr::new("final.bin"),
            OsStr::new("backup"),
            None,
            None,
        )
        .unwrap();

        assert_eq!(std::fs::read(moved.join("final.bin")).unwrap(), b"new");
        assert!(!moved.join("backup").exists());
        assert!(!moved.join("stage.part").exists());
        assert_eq!(
            std::fs::read(parent.join("final.bin")).unwrap(),
            b"unrelated"
        );
        assert_eq!(std::fs::read(parent.join("backup")).unwrap(), b"keep");
        std::fs::remove_dir_all(root).unwrap();
    }

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
        assert!(commit_verified_delta_output(&staging, &final_path, 4)
            .await
            .is_err());
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

    #[tokio::test]
    async fn delta_output_commit_creates_missing_destination_without_replace() {
        let root = std::env::temp_dir().join(format!("ghostftp-delta-new-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let destination = root.join("new.bin");
        let staged = root.join("staged.part");
        tokio::fs::write(&staged, b"verified").await.unwrap();
        commit_verified_delta_output(&staged, &destination, 8)
            .await
            .unwrap();
        assert_eq!(tokio::fs::read(&destination).await.unwrap(), b"verified");
        assert!(tokio::fs::symlink_metadata(&staged).await.is_err());
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
