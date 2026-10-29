use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::SshSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use russh_sftp::protocol::FileAttributes;
use std::collections::HashSet;
use std::sync::Arc;

pub struct SftpFs {
    session: Arc<SshSession>,
}

impl SftpFs {
    pub fn new(session: Arc<SshSession>) -> Self {
        Self { session }
    }
}

// Directory listings are used for overwrite/conflict and sync decisions.
// Reject incomplete or ambiguous names instead of advertising a false absence.
const MAX_SFTP_LIST_ENTRIES: usize = 100_000;
const MAX_SFTP_LIST_NAME_BYTES: usize = 16 * 1024 * 1024;

#[derive(Default)]
struct SftpListingBudget {
    entries: usize,
    name_bytes: usize,
}

impl SftpListingBudget {
    fn record(&mut self, name: &str) -> Result<()> {
        check_delete_name(name)?;
        anyhow::ensure!(
            self.entries < MAX_SFTP_LIST_ENTRIES,
            "SFTP listing refused: maximum entry count exceeded"
        );
        let bytes = self
            .name_bytes
            .checked_add(name.len())
            .context("SFTP listing name byte count overflow")?;
        anyhow::ensure!(
            bytes <= MAX_SFTP_LIST_NAME_BYTES,
            "SFTP listing refused: maximum filename byte count exceeded"
        );
        self.name_bytes = bytes;
        self.entries += 1;
        Ok(())
    }
}

const MAX_SFTP_DELETE_ENTRIES: usize = 25_000;
const MAX_SFTP_DELETE_DEPTH: usize = 64;
const MAX_SFTP_DELETE_PATH_BYTES: usize = 8 * 1024 * 1024;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum DeleteKind {
    File,
    Directory,
    Symlink,
}

fn delete_kind(attrs: &FileAttributes) -> Result<DeleteKind> {
    if attrs.is_symlink() {
        Ok(DeleteKind::Symlink)
    } else if attrs.is_dir() {
        Ok(DeleteKind::Directory)
    } else if attrs.is_regular() {
        Ok(DeleteKind::File)
    } else {
        anyhow::bail!("SFTP recursive delete refused: unknown entry type")
    }
}

fn check_delete_name(name: &str) -> Result<()> {
    anyhow::ensure!(
        !name.is_empty()
            && !matches!(name, "." | "..")
            && !name
                .chars()
                .any(|ch| matches!(ch, '/' | '\\' | '\0' | '\r' | '\n')),
        "SFTP recursive delete refused: unsafe entry name"
    );
    Ok(())
}

#[derive(Default)]
struct DeleteBudget {
    entries: usize,
    bytes: usize,
}

impl DeleteBudget {
    fn record(&mut self, name: &str, depth: usize) -> Result<()> {
        anyhow::ensure!(
            depth <= MAX_SFTP_DELETE_DEPTH,
            "SFTP recursive delete depth limit exceeded"
        );
        anyhow::ensure!(
            self.entries < MAX_SFTP_DELETE_ENTRIES,
            "SFTP recursive delete entry limit exceeded"
        );
        let bytes = self
            .bytes
            .checked_add(name.len())
            .context("SFTP delete path size overflow")?;
        anyhow::ensure!(
            bytes <= MAX_SFTP_DELETE_PATH_BYTES,
            "SFTP recursive delete path size limit exceeded"
        );
        self.bytes = bytes;
        self.entries += 1;
        Ok(())
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
                let mut budget = SftpListingBudget::default();
                let mut seen = HashSet::new();
                for entry in entries {
                    let name = entry.file_name();
                    if name == "." || name == ".." {
                        continue;
                    }
                    budget.record(&name)?;
                    anyhow::ensure!(
                        seen.insert(name.clone()),
                        "SFTP listing refused: duplicate filename"
                    );
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
        super::validate_protocol_mutation_path(from)?;
        super::validate_protocol_mutation_path(to)?;
        // Some SFTP servers replace destination contents on RENAME. Refuse
        // a known collision without executing the mutating request. Do not
        // interpret listing failures as proof of an empty directory.
        let (parent, target_name) = super::remote_destination_parent_name(to)?;
        let entries = self
            .list_dir(parent)
            .await
            .with_context(|| format!("SFTP rename destination inventory {parent}"))?;
        anyhow::ensure!(
            !entries
                .iter()
                .any(|entry| super::remote_names_may_alias(&entry.name, target_name)),
            "SFTP rename refused: destination already exists"
        );
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        sftp.rename(from, to)
            .await
            .with_context(|| format!("sftp rename {from} -> {to}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        super::validate_protocol_mutation_path(path)?;
        anyhow::ensure!(
            !path.chars().any(|ch| matches!(ch, '\0' | '\r' | '\n')),
            "SFTP delete refused: unsafe target path"
        );
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;

        // LSTAT instead of STAT: never traverse a symlink to another tree.
        let root = delete_kind(
            &sftp
                .symlink_metadata(path)
                .await
                .context("SFTP lstat root")?,
        )?;
        if root != DeleteKind::Directory {
            return sftp
                .remove_file(path)
                .await
                .with_context(|| format!("sftp unlink {path}"));
        }
        if !recursive {
            return sftp
                .remove_dir(path)
                .await
                .with_context(|| format!("sftp rmdir {path}"));
        }

        // Inventory every descendant before the FIRST destructive request.
        let mut budget = DeleteBudget::default();
        budget.record(path, 0)?;
        let mut seen = HashSet::from([path.to_string()]);
        let mut pending = vec![(path.to_string(), 0usize)];
        let mut targets = vec![(path.to_string(), DeleteKind::Directory)];
        while let Some((directory, depth)) = pending.pop() {
            let entries = sftp
                .read_dir(&directory)
                .await
                .with_context(|| format!("sftp list {directory}"))?;
            for entry in entries {
                let name = entry.file_name();
                if matches!(name.as_str(), "." | "..") {
                    continue;
                }
                check_delete_name(&name)?;
                let child = format!("{}/{}", directory.trim_end_matches('/'), name);
                anyhow::ensure!(
                    seen.insert(child.clone()),
                    "SFTP recursive delete duplicate path"
                );
                let child_depth = depth + 1;
                budget.record(&child, child_depth)?;
                let kind = delete_kind(
                    &sftp
                        .symlink_metadata(&child)
                        .await
                        .with_context(|| format!("sftp lstat {child}"))?,
                )?;
                if kind == DeleteKind::Directory {
                    pending.push((child.clone(), child_depth));
                }
                targets.push((child, kind));
            }
        }

        // Children follow their parents in this inventory, so reverse order
        // removes children first. Detect type swaps before each removal.
        // SFTP does not provide an atomic multi-path delete transaction.
        for (target, expected) in targets.into_iter().rev() {
            let actual = delete_kind(
                &sftp
                    .symlink_metadata(&target)
                    .await
                    .with_context(|| format!("sftp recheck lstat {target}"))?,
            )?;
            anyhow::ensure!(
                actual == expected,
                "SFTP delete stopped: entry changed type"
            );
            if expected == DeleteKind::Directory {
                sftp.remove_dir(&target)
                    .await
                    .with_context(|| format!("sftp rmdir {target}"))?;
            } else {
                sftp.remove_file(&target)
                    .await
                    .with_context(|| format!("sftp unlink {target}"))?;
            }
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        super::validate_protocol_mutation_path(path)?;
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        sftp.create_dir(path)
            .await
            .with_context(|| format!("sftp mkdir {path}"))?;
        Ok(())
    }

    async fn chmod(&self, path: &str, mode: u32) -> Result<()> {
        super::validate_protocol_mutation_path(path)?;
        let sftp_cell = self.session.ensure_sftp().await?;
        let sftp = sftp_cell.lock().await;
        let attrs = FileAttributes {
            permissions: Some(mode),
            ..Default::default()
        };
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

#[cfg(test)]
mod recursive_delete_safety_tests {
    use super::*;

    #[test]
    fn name_validation_blocks_remote_path_escape() {
        for name in [
            "",
            ".",
            "..",
            "a/b",
            "a\\b",
            "bad\0name",
            "bad\nname",
            "bad\rname",
        ] {
            assert!(check_delete_name(name).is_err(), "{name:?}");
        }
        for name in ["normal.txt", ".hidden", "čćž.txt", "with spaces"] {
            assert!(check_delete_name(name).is_ok(), "{name:?}");
        }
    }

    #[test]
    fn directory_inventory_rejects_unsafe_names_and_oversized_listings() {
        let mut budget = SftpListingBudget::default();
        for name in ["document.txt", ".hidden", "čćž.txt", "spaced name"] {
            budget.record(name).unwrap();
        }
        let before = budget.entries;
        for name in [
            "",
            ".",
            "..",
            "dir/file",
            "dir\\\\file",
            "bad\\0name",
            "bad\\nname",
        ] {
            assert!(budget.record(name).is_err(), "{name:?}");
        }
        assert_eq!(budget.entries, before);
        let mut entries = SftpListingBudget {
            entries: MAX_SFTP_LIST_ENTRIES - 1,
            name_bytes: 0,
        };
        entries.record("one").unwrap();
        assert!(entries.record("extra").is_err());
        let mut bytes = SftpListingBudget {
            entries: 0,
            name_bytes: MAX_SFTP_LIST_NAME_BYTES - 4,
        };
        bytes.record("four").unwrap();
        assert!(bytes.record("extra").is_err());
        assert_eq!(bytes.entries, 1);
    }

    #[test]
    fn inventory_limits_fail_without_changing_counters() {
        let mut budget = DeleteBudget::default();
        budget.record("root", 0).unwrap();
        budget.record("last", MAX_SFTP_DELETE_DEPTH).unwrap();
        assert!(budget.record("deep", MAX_SFTP_DELETE_DEPTH + 1).is_err());
        assert_eq!(budget.entries, 2);

        let mut budget = DeleteBudget {
            entries: MAX_SFTP_DELETE_ENTRIES - 1,
            bytes: 0,
        };
        budget.record("last", 1).unwrap();
        assert!(budget.record("extra", 1).is_err());

        let mut budget = DeleteBudget {
            entries: 0,
            bytes: MAX_SFTP_DELETE_PATH_BYTES - 4,
        };
        budget.record("four", 1).unwrap();
        assert!(budget.record("extra", 1).is_err());
    }
}
