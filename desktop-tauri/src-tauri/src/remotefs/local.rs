use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use anyhow::{anyhow, Context};
use async_trait::async_trait;
use std::path::Path;

pub struct LocalFs;

#[async_trait]
impl RemoteFs for LocalFs {
    async fn list_dir(&self, path: &str) -> anyhow::Result<Vec<DirEntry>> {
        let p = Path::new(path).to_path_buf();
        let mut rd = tokio::fs::read_dir(&p)
            .await
            .with_context(|| format!("read_dir {}", p.display()))?;
        let mut out = Vec::new();
        while let Some(entry) = rd.next_entry().await? {
            let meta = match entry.metadata().await {
                Ok(m) => m,
                Err(_) => continue,
            };
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
            out.push(DirEntry {
                name: entry.file_name().to_string_lossy().into_owned(),
                path: entry.path().to_string_lossy().into_owned(),
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
        tokio::fs::rename(from, to)
            .await
            .with_context(|| format!("rename {from} -> {to}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> anyhow::Result<()> {
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
