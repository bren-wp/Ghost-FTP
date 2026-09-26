use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::FtpSession;
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use std::sync::Arc;
use suppaftp::list::File as FtpFile;

pub struct FtpFs {
    session: Arc<FtpSession>,
}

impl FtpFs {
    pub fn new(session: Arc<FtpSession>) -> Self {
        Self { session }
    }
}

fn join(base: &str, name: &str) -> String {
    if base.is_empty() || base == "/" {
        format!("/{}", name.trim_start_matches('/'))
    } else {
        format!("{}/{}", base.trim_end_matches('/'), name.trim_start_matches('/'))
    }
}

fn entry_from_listing(parent: &str, line: &str) -> Option<DirEntry> {
    // suppaftp::list::File parses Unix-style `ls -l` lines emitted by most FTP
    // daemons. Some servers (notably IIS in MS-DOS mode) use a different
    // format that won't parse here; users can switch to UNIX listing on the
    // server side or rely on coarse type detection in v0.5.
    let f = FtpFile::try_from(line.to_string()).ok()?;
    let name = f.name().to_string();
    if name == "." || name == ".." {
        return None;
    }
    let kind = if f.is_directory() {
        FileKind::Directory
    } else if f.is_symlink() {
        FileKind::Symlink
    } else if f.is_file() {
        FileKind::File
    } else {
        FileKind::Other
    };
    let path = join(parent, &name);
    let modified = f
        .modified()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .ok();
    Some(DirEntry {
        name,
        path,
        kind,
        size: f.size() as u64,
        modified,
        mode: None,
        etag: None,
    })
}

/// Issue a directory listing that includes dotfiles.
///
/// Plain `LIST` runs the server's `ls` without `-a`, so `.htaccess`, `.env` and
/// every other dotfile are simply absent from the listing — the file is there
/// and downloadable by name, but nothing that walks a directory can see it. That
/// silence is worse than an error: a tree walk (recursive delete, sync, search)
/// quietly skips them.
///
/// So ask for `LIST -a` and fall back to a bare `LIST` when the server rejects
/// the flag (IIS and a few others take the argument as a literal path). Servers
/// that already include dotfiles are unaffected — `-a` is a no-op there.
fn list_lines(
    stream: &mut crate::session::ftp::FtpStreamKind,
    target: &str,
) -> Result<Vec<String>> {
    match stream.list(Some(&format!("-a {target}"))) {
        Ok(lines) => Ok(lines),
        Err(_) => stream
            .list(Some(target))
            .with_context(|| format!("FTP LIST {target}")),
    }
}

#[async_trait]
impl RemoteFs for FtpFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let path_owned = path.to_string();
        let parent_for_entries = path_owned.clone();
        self.session
            .with_stream(move |stream| {
                let target = if path_owned.is_empty() { "." } else { &path_owned };
                let listing = list_lines(stream, target)?;
                let mut out = Vec::with_capacity(listing.len());
                for line in listing {
                    if let Some(entry) = entry_from_listing(&parent_for_entries, &line) {
                        out.push(entry);
                    }
                }
                Ok(out)
            })
            .await
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let from = from.to_string();
        let to = to.to_string();
        self.session
            .with_stream(move |stream| {
                stream
                    .rename(&from, &to)
                    .with_context(|| format!("FTP RNFR/RNTO {from} -> {to}"))?;
                Ok(())
            })
            .await
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        // The FTP protocol distinguishes file deletion (DELE) from directory
        // deletion (RMD), and has no native recursive variant. We probe the
        // type via a CWD trick: if we can change into it, it's a directory.
        let path = path.to_string();
        if recursive {
            // Recursive: walk children using a stack of (dir, listing-line)
            // pairs we collect by repeatedly issuing LIST.
            delete_recursive(self.session.clone(), path).await
        } else {
            self.session
                .with_stream(move |stream| {
                    // Try file delete first; if the server says it's a directory,
                    // fall back to RMD. Many servers return distinct codes.
                    match stream.rm(&path) {
                        Ok(()) => Ok(()),
                        Err(_) => stream
                            .rmdir(&path)
                            .with_context(|| format!("FTP RMD/DELE {path}")),
                    }
                })
                .await
        }
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let path = path.to_string();
        self.session
            .with_stream(move |stream| {
                stream
                    .mkdir(&path)
                    .with_context(|| format!("FTP MKD {path}"))?;
                Ok(())
            })
            .await
    }

    async fn chmod(&self, path: &str, mode: u32) -> Result<()> {
        // Standard FTP has no permission command. Most Unix FTP servers
        // accept `SITE CHMOD <oct> <path>` as a server extension; we try it
        // and surface the server's error if it isn't supported.
        let path = path.to_string();
        self.session
            .with_stream(move |stream| {
                let cmd = format!("SITE CHMOD {:o} {}", mode, path);
                stream
                    .site(&cmd)
                    .with_context(|| format!("FTP {cmd}"))?;
                Ok(())
            })
            .await
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: true, // best-effort via SITE CHMOD
            can_symlink: false,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            change_signal: ChangeSignal::MtimeSize,
        }
    }
}

/// Walk an FTP tree breadth-first and delete leaves then parents.
async fn delete_recursive(session: Arc<FtpSession>, root: String) -> Result<()> {
    // Collect a list of (full_path, is_dir) entries, deepest-last.
    let to_delete: Vec<(String, bool)> = session
        .with_stream({
            let root = root.clone();
            move |stream| {
                let mut stack = vec![root.clone()];
                let mut out = Vec::new();
                while let Some(d) = stack.pop() {
                    // `-a`, so a directory holding only dotfiles isn't reported
                    // as empty and left behind by the RMD pass below.
                    let listing = list_lines(stream, &d).unwrap_or_default();
                    for line in listing {
                        let Some(entry) = entry_from_listing(&d, &line) else {
                            continue;
                        };
                        match entry.kind {
                            FileKind::Directory => {
                                stack.push(entry.path.clone());
                                out.push((entry.path, true));
                            }
                            _ => out.push((entry.path, false)),
                        }
                    }
                }
                Ok::<_, anyhow::Error>(out)
            }
        })
        .await?;

    session
        .with_stream(move |stream| {
            for (p, is_dir) in to_delete.iter().rev() {
                let res = if *is_dir { stream.rmdir(p) } else { stream.rm(p) };
                res.with_context(|| format!("FTP delete {p}"))?;
            }
            stream
                .rmdir(&root)
                .with_context(|| format!("FTP RMD {root}"))?;
            Ok(())
        })
        .await
}

/// Helper to surface "unsupported" errors uniformly. Kept private to this
/// module; callers should rely on capabilities() advertising the truth.
#[allow(dead_code)]
fn unsupported(action: &str) -> anyhow::Error {
    anyhow!("FTP backend does not support {action}")
}
