use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
#[cfg(not(unix))]
use anyhow::anyhow;
use anyhow::Context;
use async_trait::async_trait;
use std::path::{Component, Path};

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

#[async_trait]
impl RemoteFs for LocalFs {
    async fn list_dir(&self, path: &str) -> anyhow::Result<Vec<DirEntry>> {
        let p = Path::new(path).to_path_buf();
        let mut rd = tokio::fs::read_dir(&p)
            .await
            .with_context(|| format!("read_dir {}", p.display()))?;
        let mut out = Vec::new();
        while let Some(entry) = rd.next_entry().await? {
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
            let path = entry.path().into_os_string().into_string().map_err(|_| {
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
        tokio::fs::rename(from, to)
            .await
            .with_context(|| format!("rename {from} -> {to}"))?;
        Ok(())
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
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            Self::reject_filesystem_root(path, "chmod")?;
            // chmod follows symlinks on Unix. Explicitly refuse symbolic
            // links and non-file/directory nodes rather than silently changing
            // the target's permissions. This is a preflight check, not an
            // atomic substitute for descriptor-based no-follow permissions.
            let meta = tokio::fs::symlink_metadata(path)
                .await
                .with_context(|| format!("lstat chmod target {path}"))?;
            if !meta.is_file() && !meta.is_dir() {
                anyhow::bail!("chmod refuses symbolic links and special filesystem nodes");
            }
            let perms = std::fs::Permissions::from_mode(mode);
            tokio::fs::set_permissions(path, perms)
                .await
                .with_context(|| format!("chmod {path}"))?;
            Ok(())
        }
        #[cfg(not(unix))]
        {
            let _ = (path, mode);
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
            let path = std::env::temp_dir()
                .join(format!("ghostftp-localfs-{}", uuid::Uuid::new_v4()));
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
    fn destructive_actions_refuse_filesystem_roots_and_dot_aliases() {
        for path in ["", ".", "..", "folder/.."] {
            assert!(LocalFs::reject_filesystem_root(path, "delete").is_err(), "{path:?}");
        }

        #[cfg(unix)]
        for path in ["/", "//", "/tmp/..", "/tmp/../.", "/../../"] {
            assert!(LocalFs::reject_filesystem_root(path, "delete").is_err(), "{path:?}");
        }

        #[cfg(windows)]
        for path in [r"C:\\", r"C:\\Windows\\..", r"C:\\Windows\\System32\\..\\.."] {
            assert!(LocalFs::reject_filesystem_root(path, "rename").is_err(), "{path:?}");
        }

        for path in ["link/../file.txt", "folder/../other", "../file.txt"] {
            assert!(LocalFs::reject_filesystem_root(path, "rename").is_err(), "{path:?}");
        }

        assert!(LocalFs::reject_filesystem_root("safe.txt", "delete").is_ok());
        assert!(LocalFs::reject_filesystem_root("folder/child", "rename").is_ok());
    }

    #[tokio::test]
    async fn directory_listing_includes_files_and_directories() {
        let fixture = TestDirectory::new();
        std::fs::write(fixture.0.join("notes.txt"), b"hello").unwrap();
        std::fs::create_dir(fixture.0.join("assets")).unwrap();

        let entries = LocalFs.list_dir(&fixture.0.to_string_lossy()).await.unwrap();
        assert_eq!(entries.len(), 2);
        assert_eq!(
            entries.iter().find(|entry| entry.name == "notes.txt").unwrap().kind,
            FileKind::File
        );
        assert_eq!(
            entries.iter().find(|entry| entry.name == "assets").unwrap().kind,
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
        symlink(fixture.0.join("missing-target"), fixture.0.join("broken-link")).unwrap();

        let entries = LocalFs.list_dir(&fixture.0.to_string_lossy()).await.unwrap();
        assert_eq!(entries.len(), 3, "dangling entries must never silently disappear");
        for name in ["folder-link", "broken-link"] {
            let entry = entries.iter().find(|entry| entry.name == name).unwrap();
            assert_eq!(entry.kind, FileKind::Symlink, "{name} must not become a directory");
        }
        assert_eq!(
            entries.iter().find(|entry| entry.name == "real-folder").unwrap().kind,
            FileKind::Directory
        );

        // Deleting a symlink (even with recursive=true) must only unlink it.
        LocalFs.delete(&fixture.0.join("folder-link").to_string_lossy(), true)
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
        std::fs::write(fixture.0.join(opaque_name), b"keep original bytes").unwrap();

        // Returning a replacement-character string here would lose filename
        // identity and permit ambiguous rename/delete from the UI.
        let result = LocalFs.list_dir(&fixture.0.to_string_lossy()).await;
        assert!(result.is_err(), "non-UTF8 local paths must not be advertised as editable");
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
        assert!(LocalFs.chmod(&target.to_string_lossy(), 0o640).await.is_ok());
    }
}
