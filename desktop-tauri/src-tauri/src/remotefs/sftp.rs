use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::SshSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use russh_sftp::protocol::FileAttributes;
use std::sync::Arc;

pub struct SftpFs {
    session: Arc<SshSession>,
}

impl SftpFs {
    pub fn new(session: Arc<SshSession>) -> Self {
        Self { session }
    }
}

#[async_trait]
impl RemoteFs for SftpFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        // Routed through with_sftp so a directory listing transparently
        // reconnects + reopens the subsystem after a long idle (read-only, so
        // re-running it on a reconnect is always safe).
        self.session
            .with_sftp(|sftp_cell| async move {
                let sftp = sftp_cell.lock().await;

                let entries = sftp
                    .read_dir(path)
                    .await
                    .with_context(|| format!("sftp read_dir {path}"))?;

                let base = if path == "/" {
                    String::new()
                } else {
                    path.trim_end_matches('/').to_string()
                };

                let mut out = Vec::new();
                for entry in entries {
                    let name = entry.file_name();
                    if name == "." || name == ".." {
                        continue;
                    }
                    let attrs = entry.metadata();
                    let kind = if attrs.is_dir() {
                        FileKind::Directory
                    } else if attrs.is_symlink() {
                        FileKind::Symlink
                    } else if attrs.is_regular() {
                        FileKind::File
                    } else {
                        FileKind::Other
                    };
                    let full_path = format!("{base}/{name}");
                    out.push(DirEntry {
                        name,
                        path: full_path,
                        kind,
                        size: attrs.size.unwrap_or(0),
                        modified: attrs.mtime.map(|t| t as i64),
                        mode: attrs.permissions,
                        etag: None,
                    });
                }
                Ok(out)
            })
            .await
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        sftp.rename(from, to)
            .await
            .with_context(|| format!("sftp rename {from} -> {to}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;

        // Probe what we're dealing with.
        let meta = sftp
            .metadata(path)
            .await
            .with_context(|| format!("stat {path}"))?;

        if meta.is_dir() {
            if recursive {
                // Bottom-up walk so we delete leaves first.
                let mut stack: Vec<String> = vec![path.to_string()];
                let mut to_delete: Vec<(String, bool)> = Vec::new(); // (path, is_dir)
                while let Some(d) = stack.pop() {
                    let entries = sftp.read_dir(&d).await?;
                    for e in entries {
                        let n = e.file_name();
                        if n == "." || n == ".." {
                            continue;
                        }
                        let child = format!("{}/{}", d.trim_end_matches('/'), n);
                        let attrs = e.metadata();
                        if attrs.is_dir() {
                            stack.push(child.clone());
                            to_delete.push((child, true));
                        } else {
                            to_delete.push((child, false));
                        }
                    }
                }
                // Files first, then directories (in reverse for deepest-first).
                for (p, is_dir) in to_delete.iter().rev() {
                    if *is_dir {
                        sftp.remove_dir(p).await?;
                    } else {
                        sftp.remove_file(p).await?;
                    }
                }
                sftp.remove_dir(path).await?;
            } else {
                sftp.remove_dir(path).await?;
            }
        } else {
            sftp.remove_file(path).await?;
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        sftp.create_dir(path)
            .await
            .with_context(|| format!("sftp mkdir {path}"))?;
        Ok(())
    }

    async fn chmod(&self, path: &str, mode: u32) -> Result<()> {
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        let mut attrs = FileAttributes::default();
        attrs.permissions = Some(mode);
        sftp.set_metadata(path, attrs)
            .await
            .with_context(|| format!("sftp chmod {path}"))?;
        Ok(())
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: true,
            can_symlink: true,
            can_rename: true,
            has_directories: true,
            has_shell: true,
            change_signal: ChangeSignal::MtimeSize,
        }
    }
}
