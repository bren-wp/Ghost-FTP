use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
#[cfg(not(unix))]
use anyhow::anyhow;
use anyhow::Context;
use async_trait::async_trait;
use std::path::{Component, Path};

// Avoid unbounded allocation when a local mount, shared folder or generated
// build directory contains an unexpectedly large number of files. Refuse to
// publish partial lists: sync/conflict logic must never infer that omitted
// paths do not exist.
const LOCAL_DIRECTORY_MAX_ENTRIES: usize = 100_000;
const LOCAL_DIRECTORY_MAX_NAME_BYTES: usize = 16 * 1024 * 1024;

fn check_local_inventory_budget(entries: usize, name_bytes: usize) -> anyhow::Result<()> {
    anyhow::ensure!(
        entries <= LOCAL_DIRECTORY_MAX_ENTRIES,
        "local directory inventory limit exceeded: too many entries"
    );
    anyhow::ensure!(
        name_bytes <= LOCAL_DIRECTORY_MAX_NAME_BYTES,
        "local directory inventory limit exceeded: filename budget exhausted"
    );
    Ok(())
}

pub struct LocalFs;

impl LocalFs {
    fn reject_filesystem_root(path: &str, operation: &str) -> anyhow::Result<()> {
        let candidate = Path::new(path);
        if path.trim().is_empty() {
            anyhow::bail!("{operation} path must not be empty");
        }
        // Parent components are unsafe for destructive operations even
        // when their lexical depth seems harmless. If an earlier component
        // is a symlink, "link/.." resolves relative to the link target,
        // potentially outside the selected tree or even to the root.
        let mut named_components = 0usize;
        for component in candidate.components() {
            match component {
                Component::Normal(_) => named_components += 1,
                Component::ParentDir => {
                    anyhow::bail!("{operation} refuses parent traversal in a local path");
                }
                _ => {}
            }
        }
        if named_components == 0 {
            anyhow::bail!("{operation} refuses a local filesystem root or empty target");
        }
        Ok(())
    }
}

/// Never overwrite an existing local target during a rename. Both Windows
/// and Linux use an atomic no-replace OS operation, including for symlinks.
pub(crate) fn rename_local_noreplace(from: &str, to: &str) -> anyhow::Result<()> {
    #[cfg(target_os = "linux")]
    {
        let source =
            std::ffi::CString::new(from.as_bytes()).context("rename source contains NUL")?;
        let destination =
            std::ffi::CString::new(to.as_bytes()).context("rename target contains NUL")?;
        let status = unsafe {
            libc::renameat2(
                libc::AT_FDCWD,
                source.as_ptr(),
                libc::AT_FDCWD,
                destination.as_ptr(),
                libc::RENAME_NOREPLACE,
            )
        };
        if status != 0 {
            return Err(std::io::Error::last_os_error())
                .with_context(|| format!("rename without replacing {from} -> {to}"));
        }
        Ok(())
    }

    #[cfg(windows)]
    {
        use std::os::windows::ffi::OsStrExt;
        let source: Vec<u16> = std::ffi::OsStr::new(from)
            .encode_wide()
            .chain(std::iter::once(0))
            .collect();
        let destination: Vec<u16> = std::ffi::OsStr::new(to)
            .encode_wide()
            .chain(std::iter::once(0))
            .collect();
        let ok = unsafe {
            windows_sys::Win32::Storage::FileSystem::MoveFileW(
                source.as_ptr(),
                destination.as_ptr(),
            )
        };
        if ok == 0 {
            return Err(std::io::Error::last_os_error())
                .with_context(|| format!("rename without replacing {from} -> {to}"));
        }
        Ok(())
    }

    #[cfg(target_os = "macos")]
    {
        // Darwin renamex_np(RENAME_EXCL) is atomic: no TOCTOU window between
        // testing the destination and moving the source, even for symlinks.
        // Never use std::fs::rename on macOS: it silently replaces targets.
        let source =
            std::ffi::CString::new(from.as_bytes()).context("rename source contains NUL")?;
        let destination =
            std::ffi::CString::new(to.as_bytes()).context("rename destination contains NUL")?;
        let status =
            unsafe { libc::renamex_np(source.as_ptr(), destination.as_ptr(), libc::RENAME_EXCL) };
        if status != 0 {
            return Err(std::io::Error::last_os_error())
                .with_context(|| format!("exclusive macOS rename {from} -> {to}"));
        }
        Ok(())
    }

    #[cfg(not(any(target_os = "linux", target_os = "macos", windows)))]
    {
        // No supported atomic create-if-absent rename contract. Fail closed
        // instead of a racy metadata-preflight followed by an overwriting move.
        anyhow::bail!("atomic no-overwrite rename is unsupported on this platform")
    }
}

#[async_trait]
impl RemoteFs for LocalFs {
    async fn list_dir(&self, path: &str) -> anyhow::Result<Vec<DirEntry>> {
        let p = Path::new(path).to_path_buf();
        let mut rd = tokio::fs::read_dir(&p)
            .await
            .with_context(|| format!("read_dir {}", p.display()))?;
        let mut out = Vec::new();
        let mut total_name_bytes = 0usize;
        while let Some(entry) = rd.next_entry().await? {
            check_local_inventory_budget(out.len() + 1, total_name_bytes)?;
            // Tokio DirEntry::metadata() already uses symlink metadata
            // (it does not follow links). Do not silently skip an entry if
            // reading that metadata fails: comparisons or sync would then
            // act on an incomplete local inventory.
            let meta = entry
                .metadata()
                .await
                .with_context(|| format!("stat directory entry {}", entry.path().display()))?;
            let kind = if meta.is_dir() {
                FileKind::Directory
            } else if meta.file_type().is_symlink() {
                FileKind::Symlink
            } else if meta.is_file() {
                FileKind::File
            } else {
                FileKind::Other
            };
            let modified = meta
                .modified()
                .ok()
                .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
                .map(|d| d.as_secs() as i64);
            // Tauri's current DirEntry DTO uses UTF-8 strings. Lossy
            // conversion would collapse distinct Unix byte filenames into
            // the same U+FFFD display name and could route file actions to
            // the wrong object. Fail closed until raw OS paths have an
            // explicitly lossless UI/IPC representation.
            let name = entry.file_name().into_string().map_err(|_| {
                anyhow::anyhow!("local entry has a filename that cannot be represented as UTF-8")
            })?;
            total_name_bytes = total_name_bytes
                .checked_add(name.len())
                .context("local directory inventory filename budget overflow")?;
            check_local_inventory_budget(out.len() + 1, total_name_bytes)?;
            let path =
                entry.path().into_os_string().into_string().map_err(|_| {
                    anyhow::anyhow!("local entry path cannot be represented as UTF-8")
                })?;
            out.push(DirEntry {
                name,
                path,
                kind,
                size: meta.len(),
                modified,
                mode: None,
                etag: None,
            });
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> anyhow::Result<()> {
        Self::reject_filesystem_root(from, "rename")?;
        Self::reject_filesystem_root(to, "rename")?;
        let from = from.to_owned();
        let to = to.to_owned();
        tokio::task::spawn_blocking(move || rename_local_noreplace(&from, &to))
            .await
            .context("join local no-replace rename task")?
    }

    async fn delete(&self, path: &str, recursive: bool) -> anyhow::Result<()> {
        Self::reject_filesystem_root(path, "delete")?;
        let meta = tokio::fs::symlink_metadata(path)
            .await
            .with_context(|| format!("stat {path}"))?;
        if meta.is_dir() {
            if recursive {
                tokio::fs::remove_dir_all(path).await?;
            } else {
                tokio::fs::remove_dir(path).await?;
            }
        } else {
            tokio::fs::remove_file(path).await?;
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> anyhow::Result<()> {
        tokio::fs::create_dir(path)
            .await
            .with_context(|| format!("create_dir {path}"))?;
        Ok(())
    }

    async fn chmod(&self, path: &str, mode: u32) -> anyhow::Result<()> {
        Self::reject_filesystem_root(path, "chmod")?;
        #[cfg(target_os = "linux")]
        {
            use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
            let path = path.to_owned();
            // O_NOFOLLOW prevents final-component symlinks; fchmod works on
            // the opened inode, so a later symlink swap cannot redirect it.
            tokio::task::spawn_blocking(move || -> anyhow::Result<()> {
                let file = std::fs::OpenOptions::new()
                    .read(true)
                    .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
                    .open(&path)
                    .with_context(|| format!("open no-follow chmod target {path}"))?;
                let meta = file
                    .metadata()
                    .with_context(|| format!("fstat chmod target {path}"))?;
                anyhow::ensure!(
                    meta.is_file() || meta.is_dir(),
                    "chmod refuses symbolic links and special filesystem nodes"
                );
                file.set_permissions(std::fs::Permissions::from_mode(mode))
                    .with_context(|| format!("fchmod local target {path}"))?;
                Ok(())
            })
            .await
            .context("join no-follow local chmod task")?
        }
        #[cfg(all(unix, not(target_os = "linux")))]
        {
            use std::os::unix::fs::PermissionsExt;
            // Legacy non-Linux preflight does not eliminate a path race.
            let meta = tokio::fs::symlink_metadata(path)
                .await
                .with_context(|| format!("lstat chmod target {path}"))?;
            anyhow::ensure!(
                meta.is_file() || meta.is_dir(),
                "chmod refuses symbolic links and special filesystem nodes"
            );
            tokio::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
                .await
                .with_context(|| format!("chmod {path}"))?;
            Ok(())
        }
        #[cfg(not(unix))]
        {
            let _ = mode;
            Err(anyhow!("chmod is not supported on this platform"))
        }
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: cfg!(unix),
            can_symlink: true,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            change_signal: ChangeSignal::MtimeSize,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    struct TestDirectory(PathBuf);

    impl TestDirectory {
        fn new() -> Self {
            let path =
                std::env::temp_dir().join(format!("ghostftp-localfs-{}", uuid::Uuid::new_v4()));
            std::fs::create_dir(&path).expect("create isolated localfs fixture");
            Self(path)
        }
    }

    impl Drop for TestDirectory {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.0);
        }
    }

    #[test]
    fn bounded_inventory_rejects_oversized_directories_without_partial_results() {
        assert!(check_local_inventory_budget(0, 0).is_ok());
        assert!(check_local_inventory_budget(
            LOCAL_DIRECTORY_MAX_ENTRIES,
            LOCAL_DIRECTORY_MAX_NAME_BYTES
        )
        .is_ok());
        assert!(check_local_inventory_budget(LOCAL_DIRECTORY_MAX_ENTRIES + 1, 0).is_err());
        assert!(check_local_inventory_budget(1, LOCAL_DIRECTORY_MAX_NAME_BYTES + 1).is_err());
    }

    #[tokio::test]
    async fn local_listing_preserves_normal_files_and_symlinks() {
        let fixture = TestDirectory::new();
        std::fs::write(fixture.0.join("report čćž.txt"), b"contents").unwrap();
        std::fs::create_dir(fixture.0.join("subfolder")).unwrap();
        let files = LocalFs.list_dir(fixture.0.to_str().unwrap()).await.unwrap();
        assert_eq!(files.len(), 2);
        assert!(files.iter().any(|entry| entry.name == "report čćž.txt"));
        assert!(files.iter().any(|entry| entry.name == "subfolder"));
    }

    #[test]
    fn destructive_actions_refuse_filesystem_roots_and_dot_aliases() {
        for path in ["", ".", "..", "folder/.."] {
            assert!(
                LocalFs::reject_filesystem_root(path, "delete").is_err(),
                "{path:?}"
            );
        }

        #[cfg(unix)]
        for path in ["/", "//", "/tmp/..", "/tmp/../.", "/../../"] {
            assert!(
                LocalFs::reject_filesystem_root(path, "delete").is_err(),
                "{path:?}"
            );
        }

        #[cfg(windows)]
        for path in [
            r"C:\\",
            r"C:\\Windows\\..",
            r"C:\\Windows\\System32\\..\\..",
        ] {
            assert!(
                LocalFs::reject_filesystem_root(path, "rename").is_err(),
                "{path:?}"
            );
        }

        for path in ["link/../file.txt", "folder/../other", "../file.txt"] {
            assert!(
                LocalFs::reject_filesystem_root(path, "rename").is_err(),
                "{path:?}"
            );
        }

        assert!(LocalFs::reject_filesystem_root("safe.txt", "delete").is_ok());
        assert!(LocalFs::reject_filesystem_root("folder/child", "rename").is_ok());
    }

    #[tokio::test]
    async fn rename_never_replaces_an_existing_file() {
        let fixture = TestDirectory::new();
        let source = fixture.0.join("source.txt");
        let occupied = fixture.0.join("existing.txt");
        let available = fixture.0.join("renamed.txt");
        std::fs::write(&source, b"source preserved").unwrap();
        std::fs::write(&occupied, b"target preserved").unwrap();

        assert!(LocalFs
            .rename(&source.to_string_lossy(), &occupied.to_string_lossy())
            .await
            .is_err());
        assert_eq!(std::fs::read(&source).unwrap(), b"source preserved");
        assert_eq!(std::fs::read(&occupied).unwrap(), b"target preserved");

        LocalFs
            .rename(&source.to_string_lossy(), &available.to_string_lossy())
            .await
            .expect("free target must be renameable");
        assert!(!source.exists());
        assert_eq!(std::fs::read(&available).unwrap(), b"source preserved");
    }

    #[cfg(target_os = "macos")]
    #[test]
    fn macos_exclusive_rename_refuses_existing_directory_without_mutating_either() {
        let fixture = TestDirectory::new();
        let source = fixture.0.join("source-dir");
        let destination = fixture.0.join("existing-dir");
        std::fs::create_dir(&source).unwrap();
        std::fs::create_dir(&destination).unwrap();
        std::fs::write(source.join("source-data"), b"source").unwrap();
        std::fs::write(destination.join("target-data"), b"target").unwrap();

        assert!(
            rename_local_noreplace(source.to_str().unwrap(), destination.to_str().unwrap())
                .is_err()
        );
        assert_eq!(
            std::fs::read(source.join("source-data")).unwrap(),
            b"source"
        );
        assert_eq!(
            std::fs::read(destination.join("target-data")).unwrap(),
            b"target"
        );
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn rename_never_overwrites_a_dangling_symlink() {
        use std::os::unix::fs::symlink;
        let fixture = TestDirectory::new();
        let source = fixture.0.join("source.txt");
        let link = fixture.0.join("dangling");
        std::fs::write(&source, b"keep source").unwrap();
        symlink(fixture.0.join("missing"), &link).unwrap();

        assert!(LocalFs
            .rename(&source.to_string_lossy(), &link.to_string_lossy())
            .await
            .is_err());
        assert_eq!(std::fs::read(&source).unwrap(), b"keep source");
        assert!(std::fs::symlink_metadata(&link)
            .unwrap()
            .file_type()
            .is_symlink());
    }

    #[tokio::test]
    async fn directory_listing_includes_files_and_directories() {
        let fixture = TestDirectory::new();
        std::fs::write(fixture.0.join("notes.txt"), b"hello").unwrap();
        std::fs::create_dir(fixture.0.join("assets")).unwrap();

        let entries = LocalFs
            .list_dir(&fixture.0.to_string_lossy())
            .await
            .unwrap();
        assert_eq!(entries.len(), 2);
        assert_eq!(
            entries
                .iter()
                .find(|entry| entry.name == "notes.txt")
                .unwrap()
                .kind,
            FileKind::File
        );
        assert_eq!(
            entries
                .iter()
                .find(|entry| entry.name == "assets")
                .unwrap()
                .kind,
            FileKind::Directory
        );
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn directory_listing_keeps_real_and_dangling_symlinks_visible() {
        use std::os::unix::fs::symlink;

        let fixture = TestDirectory::new();
        let target = fixture.0.join("real-folder");
        std::fs::create_dir(&target).unwrap();
        std::fs::write(target.join("keep.txt"), b"not deleted").unwrap();
        symlink(&target, fixture.0.join("folder-link")).unwrap();
        symlink(
            fixture.0.join("missing-target"),
            fixture.0.join("broken-link"),
        )
        .unwrap();

        let entries = LocalFs
            .list_dir(&fixture.0.to_string_lossy())
            .await
            .unwrap();
        assert_eq!(
            entries.len(),
            3,
            "dangling entries must never silently disappear"
        );
        for name in ["folder-link", "broken-link"] {
            let entry = entries.iter().find(|entry| entry.name == name).unwrap();
            assert_eq!(
                entry.kind,
                FileKind::Symlink,
                "{name} must not become a directory"
            );
        }
        assert_eq!(
            entries
                .iter()
                .find(|entry| entry.name == "real-folder")
                .unwrap()
                .kind,
            FileKind::Directory
        );

        // Deleting a symlink (even with recursive=true) must only unlink it.
        LocalFs
            .delete(&fixture.0.join("folder-link").to_string_lossy(), true)
            .await
            .unwrap();
        assert!(target.join("keep.txt").is_file());
        assert!(!fixture.0.join("folder-link").exists());
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn invalid_utf8_filename_never_becomes_a_lossy_file_action_target() {
        use std::ffi::OsString;
        use std::os::unix::ffi::OsStringExt;

        let fixture = TestDirectory::new();
        let opaque_name = OsString::from_vec(vec![b'f', 0xff, b'o']);
        let opaque_path = fixture.0.join(opaque_name);
        let create_result = std::fs::write(&opaque_path, b"keep original bytes");

        // macOS filesystems reject malformed UTF-8 names with EILSEQ before
        // LocalFs ever sees an entry. Rejecting that name is already the safe
        // result; do not treat the host filesystem's refusal as a test failure.
        // Linux filesystems may accept these bytes, so the stricter LocalFs
        // listing rejection below must still run on Linux.
        #[cfg(target_os = "macos")]
        if let Err(error) = &create_result {
            assert_eq!(
                error.raw_os_error(),
                Some(libc::EILSEQ),
                "macOS must fail closed for malformed local filenames: {error}"
            );
            assert!(!opaque_path.exists());
            return;
        }
        create_result.expect("create the malformed filename for the LocalFs validation");

        // Returning a replacement-character string here would lose filename
        // identity and permit ambiguous rename/delete from the UI.
        let result = LocalFs.list_dir(&fixture.0.to_string_lossy()).await;
        assert!(
            result.is_err(),
            "non-UTF8 local paths must not be advertised as editable"
        );
    }

    #[cfg(unix)]
    #[tokio::test]
    async fn chmod_rejects_links_and_preserves_link_targets() {
        use std::os::unix::fs::{symlink, PermissionsExt};

        let fixture = TestDirectory::new();
        let target = fixture.0.join("confidential.txt");
        let link = fixture.0.join("confidential-link");
        std::fs::write(&target, b"private").unwrap();
        std::fs::set_permissions(&target, std::fs::Permissions::from_mode(0o600)).unwrap();
        symlink(&target, &link).unwrap();

        assert!(LocalFs.chmod(&link.to_string_lossy(), 0o777).await.is_err());
        assert_eq!(
            std::fs::metadata(&target).unwrap().permissions().mode() & 0o777,
            0o600
        );
        assert!(LocalFs.chmod("/", 0o777).await.is_err());
        assert!(LocalFs
            .chmod(&target.to_string_lossy(), 0o640)
            .await
            .is_ok());
    }
}
