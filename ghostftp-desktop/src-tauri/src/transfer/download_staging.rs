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
                .write(true)
                .create_new(true)
                .open(path)
        };
        #[cfg(not(target_os = "linux"))]
        let file = std::fs::OpenOptions::new()
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
            libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
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

/// Hold the initially resolved directory for the lifetime of a download.
/// Existing path-based backend I/O still needs identity checks at boundaries.
pub(super) struct DownloadParentGuard {
    path: PathBuf,
    handle: std::sync::Arc<std::fs::File>,
    recovery_db: Option<std::sync::Arc<crate::db::Db>>,
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
}

struct RecoveryJournal {
    db: std::sync::Arc<crate::db::Db>,
    id: String,
    parent: PathBuf,
    stage: PathBuf,
    target: PathBuf,
    backup: PathBuf,
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
    ) -> Result<()> {
        let record = RecoveryRecord {
            schema: 1,
            parent: &self.parent,
            stage: &self.stage,
            target: &self.target,
            backup: &self.backup,
            parent_identity,
            stage_identity,
            original_identity,
        };
        let destination_key = recovery_destination_key(parent_identity, &self.target)?;
        self.db
            .download_recovery_begin(
                &self.id,
                &serde_json::to_string(&record)?,
                &destination_key,
            )
            .context("persist recovery intent before backup")
    }

    fn advance(&self, from: RecoveryPhase, to: RecoveryPhase) -> Result<()> {
        self.db.download_recovery_transition(&self.id, from, to)
    }

    fn finish(&self) {
        if self.db.download_recovery_finish(&self.id).is_err() {
            tracing::warn!("completed download recovery intent could not be cleared");
        }
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
        }))
    }

    fn check_recovery_destination(&self, target: &Path) -> Result<()> {
        let Some(db) = &self.recovery_db else {
            return Ok(());
        };
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
        db.download_recovery_check_destination(&recovery_destination_key(identity, target)?)
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
        self.check_recovery_destination(target)?;
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
            return tokio::task::spawn_blocking(move || {
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
            .context("join guarded delta promotion")?;
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
        self.check_recovery_destination(target)?;
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
                let expected = verified.metadata()?;
                if allow_replace {
                    promote_pinned_download(
                        &pinned,
                        &staging,
                        &target_name,
                        &backup,
                        Some(&expected),
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
                windows_promote_with_backup(&self.handle, &delete, name, backup, journal.as_ref())
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
    expected: Option<&std::fs::Metadata>,
    journal: Option<&RecoveryJournal>,
) -> Result<()> {
    use std::os::fd::AsRawFd;
    use std::os::unix::ffi::OsStrExt;

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
            journal.begin(
                (directory.dev(), directory.ino()),
                (staged.st_dev, staged.st_ino, u64::try_from(staged.st_size)?),
                (original.st_dev, original.st_ino, u64::try_from(original.st_size)?),
            )?;
        }
        rename_pinned_siblings(parent, target, backup)
            .context("preserve existing local download target without overwriting a backup")?;
        if let Some(journal) = journal {
            journal.advance(RecoveryPhase::Prepared, RecoveryPhase::BackupSaved)?;
        }
    }

    let promotion = (|| -> Result<()> {
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
            }
            return match rename_pinned_siblings(parent, backup, target) {
                Ok(()) => {
                    if let Some(journal) = journal {
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

    if let Some(original) = original {
        if let Some(journal) = journal {
            if journal.advance(RecoveryPhase::BackupSaved, RecoveryPhase::Promoted).is_err()
                || journal.advance(RecoveryPhase::Promoted, RecoveryPhase::CleanupPending).is_err()
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
            if let Ok(name) = std::ffi::CString::new(backup.as_bytes()) {
                // SAFETY: parent is live; name is a NUL-terminated sibling basename.
                let removed = unsafe { libc::unlinkat(parent.as_raw_fd(), name.as_ptr(), 0) };
                if removed == 0 {
                    if let Some(journal) = journal {
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
    let original = match windows_open_relative(parent, target, DELETE_AND_ATTRIBUTES) {
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
        journal.begin(
            (u64::from(volume), index),
            (u64::from(source_volume), source_index, source.metadata()?.len()),
            (u64::from(original_volume), original_index, original.metadata()?.len()),
        )?;
    }
    windows_rename_handle(&original, parent, backup, false)
        .context("preserve original destination without replacing backup")?;
    if let Some(journal) = journal {
        journal.advance(RecoveryPhase::Prepared, RecoveryPhase::BackupSaved)?;
    }
    if let Err(error) = windows_rename_handle(source, parent, target, false) {
        if let Some(journal) = journal {
            journal.advance(RecoveryPhase::BackupSaved, RecoveryPhase::RollbackPending)?;
        }
        return match windows_rename_handle(&original, parent, target, false) {
            Ok(()) => {
                if let Some(journal) = journal {
                    journal.advance(RecoveryPhase::RollbackPending, RecoveryPhase::RolledBack)?;
                }
                Err(error).context("promotion failed; original restored")
            }
            Err(rollback) => Err(anyhow::anyhow!(
                "promotion failed; recovery backup remains: {error:#}; rollback: {rollback:#}"
            )),
        };
    }
    if let Some(journal) = journal {
        if journal.advance(RecoveryPhase::BackupSaved, RecoveryPhase::Promoted).is_err()
            || journal.advance(RecoveryPhase::Promoted, RecoveryPhase::CleanupPending).is_err()
        {
            tracing::warn!("download published; recovery phase could not be persisted");
            return Ok(());
        }
    }
    // The original handle now identifies the backup, not the new target.
    // Cleanup failure leaves the backup intact; publication already succeeded.
    let removed = windows_delete_handle(&original).is_ok();
    drop(original);
    if removed {
        if let Some(journal) = journal {
            if windows_open_relative(parent, backup, 0x80)
                .is_err_and(|error| error.kind() == std::io::ErrorKind::NotFound)
            {
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
    use std::os::windows::ffi::OsStrExt;
    use std::os::windows::io::AsRawHandle;
    use windows_sys::Win32::Storage::FileSystem::{
        FileRenameInfo, SetFileInformationByHandle, FILE_RENAME_INFO,
    };

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
    let ok = unsafe {
        (*info).Anonymous.ReplaceIfExists = u8::from(replace);
        (*info).RootDirectory = parent.as_raw_handle();
        (*info).FileNameLength = name_len;
        std::ptr::copy_nonoverlapping(
            wide.as_ptr(),
            buffer.as_mut_ptr().cast::<u8>().add(offset).cast::<u16>(),
            wide.len(),
        );
        SetFileInformationByHandle(source.as_raw_handle(), FileRenameInfo, info.cast(), len_u32)
    };
    if ok == 0 {
        return Err(std::io::Error::last_os_error()).context("rename verified Windows file handle");
    }
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
    options.write(true);
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
            promote_pinned_download(&pinned, &staging_name, &target_name, &backup_name, None, None)
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

        assert!(guard
            .promote(&stage, &target, "recovery", true, &verified)
            .await
            .is_err());
        let rows = db.download_recovery_list().unwrap();
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].0, "recovery");
        assert_eq!(db.download_recovery_phase("recovery").unwrap(), "prepared");
        let record: serde_json::Value = serde_json::from_str(&rows[0].1).unwrap();
        assert_eq!(record["schema"], 1);
        assert!(record["original_identity"].is_array());
        assert!(guard
            .promote(&stage, &target, "different-id", false, &verified)
            .await
            .is_err());
        assert!(guard.promote_delta(&stage, &target, &verified).await.is_err());
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
