use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::ftp::validate_ftp_command_path;
use crate::session::FtpSession;
use anyhow::{Context, Result};
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

    /// A single read-only listing, with no automatic reconnect in the
    /// underlying blocking FTP stream. The outer list_dir decides whether
    /// an FTPS data-TLS failure permits one new authenticated session.
    async fn list_dir_once(&self, path: &str) -> Result<Vec<DirEntry>> {
        validate_ftp_command_path(if path.is_empty() { "." } else { path })?;
        let target = if path.is_empty() { "." } else { path }.to_string();
        let parent = path.to_string();
        self.session
            .with_stream(move |stream| {
                let listing = list_lines(stream, &target)?;
                // Never silently omit unknown server entries: a partial listing
                // could hide existing targets from sync and conflict checks.
                parse_directory_listing(&parent, listing)
            })
            .await
    }
}

/// A 522 from a protected FTPS data channel is a *TLS handshake failure*,
/// not an unsupported LIST option. A new authenticated control connection
/// may recover from a transient server-side data-socket race; no TLS
/// verification or PROT P requirement is bypassed, and the retry is bounded.
fn is_ftps_data_tls_failure(error: &anyhow::Error) -> bool {
    error.chain().any(|cause| {
        let message = cause.to_string();
        message.contains("522 SSL handshake failed") || message.contains("522 TLS handshake failed")
    })
}

fn join(base: &str, name: &str) -> String {
    if base.is_empty() || base == "/" {
        format!("/{}", name.trim_start_matches('/'))
    } else {
        format!(
            "{}/{}",
            base.trim_end_matches('/'),
            name.trim_start_matches('/')
        )
    }
}

/// IIS and Windows FTP servers also return MS-DOS-style LIST records.
/// Keep this parser strict: an ambiguous line is *not* proof that a target
/// is absent, and its size must never silently overflow.
fn dos_listing_entry(line: &str) -> Option<(String, FileKind, u64)> {
    fn token(input: &str) -> Option<(&str, &str)> {
        let input = input.trim_start_matches(char::is_whitespace);
        let end = input.find(char::is_whitespace)?;
        Some((&input[..end], &input[end..]))
    }
    let (date, rest) = token(line)?;
    let (time, rest) = token(rest)?;
    let (kind_or_size, rest) = token(rest)?;
    let name = rest
        .trim_start_matches(char::is_whitespace)
        .trim_end_matches('\r');
    if name.is_empty()
        || name
            .chars()
            .any(|ch| matches!(ch, '/' | '\\' | '\n' | '\r' | '\0'))
    {
        return None;
    }

    let mut date_parts = date.split(['-', '/']);
    let month = date_parts.next()?.parse::<u8>().ok()?;
    let day = date_parts.next()?.parse::<u8>().ok()?;
    let year_text = date_parts.next()?;
    if date_parts.next().is_some()
        || !matches!(year_text.len(), 2 | 4)
        || !year_text.bytes().all(|b| b.is_ascii_digit())
    {
        return None;
    }
    let year = year_text.parse::<u16>().ok()?;
    let full_year = if year_text.len() == 2 {
        if year < 70 {
            year + 2000
        } else {
            year + 1900
        }
    } else {
        year
    };
    let leap = full_year % 4 == 0 && (full_year % 100 != 0 || full_year % 400 == 0);
    let max_day = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if leap => 29,
        2 => 28,
        _ => return None,
    };
    if day == 0 || day > max_day {
        return None;
    }

    let upper_time = time.to_ascii_uppercase();
    let clock = upper_time
        .strip_suffix("AM")
        .or_else(|| upper_time.strip_suffix("PM"))?;
    let (hour, minute) = clock.split_once(':')?;
    if !(1..=2).contains(&hour.len())
        || minute.len() != 2
        || !hour.bytes().all(|b| b.is_ascii_digit())
        || !minute.bytes().all(|b| b.is_ascii_digit())
        || !(1..=12).contains(&hour.parse::<u8>().ok()?)
        || minute.parse::<u8>().ok()? > 59
    {
        return None;
    }

    let (kind, size) = if kind_or_size.eq_ignore_ascii_case("<DIR>") {
        (FileKind::Directory, 0)
    } else {
        let size = kind_or_size.parse::<u64>().ok()?;
        (FileKind::File, size)
    };
    Some((name.to_string(), kind, size))
}

// The FTP library has already materialized the LIST response at this point.
// Bound the published inventory and parse work even when a server sends a
// pathological response; the transport-level allocation is a separate limit.
const MAX_FTP_LIST_ENTRIES: usize = 100_000;
const MAX_FTP_LIST_RAW_BYTES: usize = 32 * 1024 * 1024;
const MAX_FTP_LIST_NAME_BYTES: usize = 16 * 1024 * 1024;

fn validate_ftp_listing_budget(listing: &[String]) -> Result<()> {
    anyhow::ensure!(
        listing.len() <= MAX_FTP_LIST_ENTRIES,
        "FTP LIST refused: maximum entry count exceeded"
    );
    let mut bytes = 0usize;
    for line in listing {
        bytes = bytes
            .checked_add(line.len())
            .context("FTP LIST raw byte count overflow")?;
        anyhow::ensure!(
            bytes <= MAX_FTP_LIST_RAW_BYTES,
            "FTP LIST refused: maximum response byte count exceeded"
        );
    }
    Ok(())
}

fn parse_directory_listing(parent: &str, listing: Vec<String>) -> Result<Vec<DirEntry>> {
    validate_ftp_listing_budget(&listing)?;
    let mut entries = Vec::with_capacity(listing.len());
    let mut seen_names = std::collections::HashSet::new();
    let mut name_bytes = 0usize;
    for line in listing {
        if is_ignorable_listing_line(&line) {
            continue;
        }
        let entry = entry_from_listing(parent, &line).ok_or_else(|| {
            anyhow::anyhow!("FTP LIST includes an unsupported or ambiguous directory entry")
        })?;
        name_bytes = name_bytes
            .checked_add(entry.name.len())
            .context("FTP LIST filename byte count overflow")?;
        anyhow::ensure!(
            name_bytes <= MAX_FTP_LIST_NAME_BYTES,
            "FTP LIST refused: maximum filename byte count exceeded"
        );
        anyhow::ensure!(
            seen_names.insert(entry.name.clone()),
            "FTP LIST contains duplicate directory entry names"
        );
        entries.push(entry);
    }
    Ok(entries)
}

/// DOS LIST records have a numeric date token. Dispatch them to the strict
/// DOS parser *before* using the more tolerant Unix LIST implementation:
/// otherwise a malformed DOS time can be interpreted as a valid Unix row.
fn looks_like_dos_listing(line: &str) -> bool {
    line.split_whitespace().next().is_some_and(|date| {
        (date.contains('-') || date.contains('/'))
            && date
                .bytes()
                .all(|b| b.is_ascii_digit() || matches!(b, b'-' | b'/'))
    })
}

/// Reject non-Unix server prose before entering SuppaFTP's permissive
/// Unix LIST parser. These are the standard 10 permission/type positions,
/// optionally followed by an ACL/xattr marker.
fn looks_like_unix_listing(line: &str) -> bool {
    let Some(mode) = line.split_whitespace().next() else {
        return false;
    };
    let bytes = mode.as_bytes();
    if !matches!(bytes.len(), 10 | 11)
        || !matches!(bytes[0], b'-' | b'd' | b'l' | b'c' | b'b' | b'p' | b's')
        || !bytes[1..10]
            .iter()
            .all(|ch| matches!(*ch, b'r' | b'w' | b'x' | b'-' | b's' | b'S' | b't' | b'T'))
    {
        return false;
    }
    bytes.len() == 10 || matches!(bytes[10], b'+' | b'@' | b'.')
}

fn is_dot_listing(line: &str) -> bool {
    if looks_like_dos_listing(line) {
        return dos_listing_entry(line)
            .is_some_and(|(name, _, _)| matches!(name.as_str(), "." | ".."));
    }
    looks_like_unix_listing(line)
        && FtpFile::try_from(line.to_string())
            .map(|file| matches!(file.name(), "." | ".."))
            .unwrap_or(false)
}

fn is_ignorable_listing_line(line: &str) -> bool {
    let line = line.trim();
    is_dot_listing(line)
        || line.is_empty()
        || line
            .strip_prefix("total ")
            .is_some_and(|count| !count.is_empty() && count.bytes().all(|b| b.is_ascii_digit()))
}

fn entry_from_listing(parent: &str, line: &str) -> Option<DirEntry> {
    // FTP LIST is not standardized. Parse Unix and IIS/MS-DOS records, but
    // never invent missing metadata: DOS listings do not prove an mtime.
    if looks_like_dos_listing(line) {
        let (name, kind, size) = dos_listing_entry(line)?;
        if matches!(name.as_str(), "." | "..") {
            return None;
        }
        return Some(DirEntry {
            path: join(parent, &name),
            name,
            kind,
            size,
            modified: None,
            mode: None,
            etag: None,
        });
    }
    if !looks_like_unix_listing(line) {
        return None;
    }
    let f = FtpFile::try_from(line.to_string()).ok()?;
    let name = f.name().to_string();
    if name.is_empty()
        || matches!(name.as_str(), "." | "..")
        || name
            .chars()
            .any(|ch| matches!(ch, '/' | '\\' | '\n' | '\r' | '\0'))
    {
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
fn list_lines_with_dot_visibility(
    stream: &mut crate::session::ftp::FtpStreamKind,
    target: &str,
) -> Result<(Vec<String>, bool)> {
    validate_ftp_command_path(target)?;
    match stream.list(Some(&format!("-a {target}"))) {
        Ok(lines) => Ok((lines, true)),
        // A failed TLS data handshake invalidates the control/data state.
        // Issuing another LIST on it loses the original error and cannot
        // safely recover. Only the async caller can replace that session.
        Err(error) if is_ftps_data_tls_failure(&error) => {
            Err(error).with_context(|| format!("FTPS LIST -a {target} data TLS handshake"))
        }
        // Retain compatibility with FTP servers that interpret -a as a
        // literal path or reject that non-standard listing option.
        Err(_) => stream
            .list(Some(target))
            .with_context(|| format!("FTP LIST {target}"))
            .map(|lines| (lines, false)),
    }
}

fn list_lines(
    stream: &mut crate::session::ftp::FtpStreamKind,
    target: &str,
) -> Result<Vec<String>> {
    list_lines_with_dot_visibility(stream, target).map(|(lines, _)| lines)
}

/// Prove whether an FTP path exists without turning protocol/listing failures
/// into absence. A successful parent LIST is the proof source. When a server
/// rejects `LIST -a`, absence of a dotfile cannot be proven and therefore
/// fails closed instead of risking overwrite of a hidden target.
pub async fn path_exists_fail_closed(session: &Arc<FtpSession>, path: &str) -> Result<bool> {
    validate_ftp_command_path(path)?;
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() || trimmed == "/" {
        return Ok(true);
    }
    let (parent, name) = match trimmed.rsplit_once('/') {
        Some(("", name)) => ("/".to_string(), name.to_string()),
        Some((parent, name)) => (parent.to_string(), name.to_string()),
        None => (".".to_string(), trimmed.to_string()),
    };
    if name.is_empty() || matches!(name.as_str(), "." | "..") {
        anyhow::bail!("FTP existence check requires a concrete target path: {path}");
    }

    session
        .with_stream(move |stream| {
            let (listing, includes_dotfiles) =
                list_lines_with_dot_visibility(stream, &parent)
                    .with_context(|| format!("FTP existence LIST {parent}"))?;
            // Never short-circuit on the first matching name: later malformed
            // or duplicate entries make an overwrite/conflict decision unsafe.
            let entries = parse_directory_listing(&parent, listing)
                .with_context(|| format!("FTP existence inventory {parent}"))?;
            if entries.iter().any(|entry| entry.name == name) {
                return Ok(true);
            }
            if name.starts_with('.') && !includes_dotfiles {
                anyhow::bail!(
                    "FTP existence check for hidden target {name} is ambiguous because the server rejected LIST -a"
                );
            }
            Ok(false)
        })
        .await
}

#[async_trait]
impl RemoteFs for FtpFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        match self.list_dir_once(path).await {
            Ok(entries) => Ok(entries),
            Err(first_error)
                if self.session.profile.protocol.eq_ignore_ascii_case("ftps")
                    && is_ftps_data_tls_failure(&first_error) =>
            {
                // Listing is idempotent. Retire the failed TLS control
                // session and allow exactly ONE retry with a fresh TLS
                // handshake, unchanged certificate validation and PROT P.
                self.session
                    .reconnect()
                    .await
                    .context("re-authenticate FTPS after data-channel TLS 522")?;
                self.list_dir_once(path)
                    .await
                    .context("FTPS LIST failed after one fresh TLS session retry")
            }
            Err(error) => Err(error),
        }
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        super::validate_protocol_mutation_path(from)?;
        super::validate_protocol_mutation_path(to)?;
        validate_ftp_command_path(from)?;
        validate_ftp_command_path(to)?;
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
        validate_ftp_command_path(path)?;
        super::validate_protocol_mutation_path(path)?;
        // FTP distinguishes file deletion (DELE) from directory deletion
        // (RMD) and has no native recursive delete command. Recursive removal
        // therefore performs a guarded bottom-up traversal before removing root.
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
        super::validate_protocol_mutation_path(path)?;
        validate_ftp_command_path(path)?;
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
        super::validate_protocol_mutation_path(path)?;
        validate_ftp_command_path(path)?;
        // Standard FTP has no permission command. Most Unix FTP servers
        // accept `SITE CHMOD <oct> <path>` as a server extension; we try it
        // and surface the server's error if it isn't supported.
        let path = path.to_string();
        self.session
            .with_stream(move |stream| {
                let cmd = format!("SITE CHMOD {:o} {}", mode, path);
                stream.site(&cmd).with_context(|| format!("FTP {cmd}"))?;
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

/// Never start deleting a tree when a server refused the LIST -a
/// inventory request. A bare LIST can look successful while hiding dotfiles.
fn require_complete_listing_for_recursive_delete(includes_dotfiles: bool) -> Result<()> {
    anyhow::ensure!(
        includes_dotfiles,
        "FTP recursive delete refused: LIST -a unsupported; hidden entries may be omitted"
    );
    Ok(())
}

// Defensive inventory limits apply before the first DELE/RMD. A buggy or
// adversarial server can return an unbounded directory chain or duplicate
// entries; neither condition must make the desktop hang or exhaust memory.
const FTP_RECURSIVE_DELETE_MAX_ENTRIES: usize = 25_000;
const FTP_RECURSIVE_DELETE_MAX_DEPTH: usize = 64;
const FTP_RECURSIVE_DELETE_MAX_PATH_BYTES: usize = 8 * 1024 * 1024;

/// Inventory every child before issuing a destructive command. Reuse the
/// strict LIST parser, reject duplicate paths, and bound traversal resources.
/// The callback makes the safety gate independently testable without an FTP
/// daemon; the live adapter supplies protected FTP/FTPS LIST responses.
fn collect_recursive_delete_targets(
    root: &str,
    mut list_directory: impl FnMut(&str) -> Result<(Vec<String>, bool)>,
) -> Result<Vec<(String, bool)>> {
    let mut stack = vec![(root.to_string(), 0usize)];
    let mut visited = std::collections::HashSet::new();
    visited.insert(root.to_string());
    let mut out = Vec::new();
    let mut total_path_bytes = root.len();

    while let Some((directory, depth)) = stack.pop() {
        let (listing, includes_dotfiles) = list_directory(&directory)
            .with_context(|| format!("FTP recursive LIST {directory}"))?;
        require_complete_listing_for_recursive_delete(includes_dotfiles)?;
        anyhow::ensure!(
            listing.len() <= FTP_RECURSIVE_DELETE_MAX_ENTRIES - out.len(),
            "FTP recursive delete refused: maximum entry count exceeded"
        );
        for entry in parse_directory_listing(&directory, listing)? {
            anyhow::ensure!(
                out.len() < FTP_RECURSIVE_DELETE_MAX_ENTRIES,
                "FTP recursive delete refused: maximum entry count exceeded"
            );
            anyhow::ensure!(
                visited.insert(entry.path.clone()),
                "FTP recursive delete refused: duplicate directory entry path"
            );
            total_path_bytes = total_path_bytes
                .checked_add(entry.path.len())
                .context("FTP recursive delete path size overflow")?;
            anyhow::ensure!(
                total_path_bytes <= FTP_RECURSIVE_DELETE_MAX_PATH_BYTES,
                "FTP recursive delete refused: path inventory byte limit exceeded"
            );
            let is_directory = entry.kind == FileKind::Directory;
            if is_directory {
                anyhow::ensure!(
                    depth < FTP_RECURSIVE_DELETE_MAX_DEPTH,
                    "FTP recursive delete refused: maximum directory depth exceeded"
                );
                stack.push((entry.path.clone(), depth + 1));
            }
            out.push((entry.path, is_directory));
        }
    }
    Ok(out)
}

/// Walk the tree without deleting any part of it until the complete bounded
/// inventory succeeds. Existing reverse-order removal is preserved.
async fn delete_recursive(session: Arc<FtpSession>, root: String) -> Result<()> {
    let to_delete = session
        .with_stream({
            let root = root.clone();
            move |stream| {
                collect_recursive_delete_targets(&root, |directory| {
                    list_lines_with_dot_visibility(stream, directory)
                })
            }
        })
        .await?;

    session
        .with_stream(move |stream| {
            for (p, is_dir) in to_delete.iter().rev() {
                let res = if *is_dir {
                    stream.rmdir(p)
                } else {
                    stream.rm(p)
                };
                res.with_context(|| format!("FTP delete {p}"))?;
            }
            stream
                .rmdir(&root)
                .with_context(|| format!("FTP RMD {root}"))?;
            Ok(())
        })
        .await
}

#[cfg(test)]
mod ftp_command_path_regressions {
    use super::validate_ftp_command_path;

    #[test]
    fn rejects_protocol_controls_and_empty_targets_before_ftp_commands() {
        for path in [
            "",
            " ",
            "\n",
            "/safe\r\nDELE /unrelated",
            "folder\rfile",
            "folder\nfile",
            "folder\0file",
            "folder\tfile",
            "folder\u{7f}file",
        ] {
            assert!(
                validate_ftp_command_path(path).is_err(),
                "unsafe FTP command path accepted: {path:?}"
            );
        }
    }

    #[test]
    fn preserves_ordinary_unicode_spaces_and_listing_root_aliases() {
        for path in [
            ".",
            "/",
            "/invoices/report čćž 文件 2026.pdf",
            "folder name/child",
        ] {
            assert!(
                validate_ftp_command_path(path).is_ok(),
                "safe FTP command path rejected: {path:?}"
            );
        }
    }
}

#[cfg(test)]
mod ftp_recursive_delete_safety_tests {
    use super::{
        collect_recursive_delete_targets, require_complete_listing_for_recursive_delete,
        FTP_RECURSIVE_DELETE_MAX_ENTRIES,
    };

    fn file(name: &str) -> String {
        format!("04-27-25  09:09AM       42 {name}")
    }

    fn directory(name: &str) -> String {
        format!("04-27-25  09:09AM       <DIR> {name}")
    }

    #[test]
    fn nested_files_and_hidden_entries_are_planned_bottom_up() {
        let planned = collect_recursive_delete_targets("/root", |path| {
            Ok((
                match path {
                    "/root" => vec![directory("sub"), file(".env")],
                    "/root/sub" => vec![file("inside.txt")],
                    _ => panic!("unexpected FTP traversal: {path}"),
                },
                true,
            ))
        })
        .unwrap();
        assert_eq!(
            planned,
            vec![
                ("/root/sub".to_string(), true),
                ("/root/.env".to_string(), false),
                ("/root/sub/inside.txt".to_string(), false),
            ]
        );
        let deletion_order: Vec<&str> = planned
            .iter()
            .rev()
            .map(|(path, _)| path.as_str())
            .collect();
        assert_eq!(
            deletion_order,
            vec!["/root/sub/inside.txt", "/root/.env", "/root/sub"]
        );
    }

    #[test]
    fn malformed_or_duplicate_server_entries_abort_inventory() {
        for lines in [
            vec![file("good.txt"), "malformed server LIST reply".to_string()],
            vec![file("duplicate"), file("duplicate")],
        ] {
            assert!(
                collect_recursive_delete_targets("/root", |_| Ok((lines.clone(), true))).is_err()
            );
        }
    }

    #[test]
    fn unbounded_directory_chains_stop_before_any_delete() {
        let mut calls = 0usize;
        let error = collect_recursive_delete_targets("/root", |_| {
            calls += 1;
            Ok((vec![directory("child")], true))
        })
        .unwrap_err();
        assert!(error.to_string().contains("maximum directory depth"));
        assert!(calls <= 65, "directory traversal was not bounded");
    }

    #[test]
    fn oversized_server_inventories_are_rejected() {
        let lines: Vec<String> = (0..=FTP_RECURSIVE_DELETE_MAX_ENTRIES)
            .map(|i| file(&format!("entry-{i}")))
            .collect();
        let error =
            collect_recursive_delete_targets("/root", |_| Ok((lines.clone(), true))).unwrap_err();
        assert!(error.to_string().contains("maximum entry count"));
    }

    #[test]
    fn unsupported_dotfile_listing_aborts_before_inventory_publication() {
        let error =
            collect_recursive_delete_targets("/root", |_| Ok((vec![file("visible.txt")], false)))
                .unwrap_err();
        assert!(error.to_string().contains("hidden entries may be omitted"));
    }

    #[test]
    fn recursive_delete_requires_hidden_entry_listing_before_deleting_anything() {
        assert!(require_complete_listing_for_recursive_delete(true).is_ok());
        let error = require_complete_listing_for_recursive_delete(false)
            .expect_err("bare LIST is not a complete destructive inventory");
        assert!(error.to_string().contains("hidden entries may be omitted"));
    }
}

#[cfg(test)]
mod ftps_handshake_regression {
    use super::is_ftps_data_tls_failure;
    use anyhow::anyhow;

    #[test]
    fn only_explicit_data_tls_522_is_reconnect_eligible() {
        assert!(is_ftps_data_tls_failure(&anyhow!(
            "Invalid response: [0] 522 SSL handshake failed."
        )));
        assert!(is_ftps_data_tls_failure(&anyhow!(
            "FTPS LIST: 522 TLS handshake failed."
        )));
        for error in [
            "530 Not logged in",
            "550 Permission denied",
            "522 Unknown data network protocol",
            "Certificate verify failed",
            "FTP connect timeout",
        ] {
            assert!(!is_ftps_data_tls_failure(&anyhow!("{error}")));
        }
    }
}

#[cfg(test)]
mod ftp_list_compatibility_tests {
    use super::FileKind;
    use super::{
        dos_listing_entry, entry_from_listing, is_dot_listing, is_ignorable_listing_line,
        parse_directory_listing, MAX_FTP_LIST_ENTRIES, MAX_FTP_LIST_NAME_BYTES,
        MAX_FTP_LIST_RAW_BYTES,
    };

    #[test]
    fn iis_dos_files_and_directories_preserve_long_unicode_names() {
        let file = entry_from_listing(
            "/incoming",
            "05-13-25  04:17PM                427 report čćž 文件 with spaces.txt",
        )
        .expect("IIS file");
        assert_eq!(file.name, "report čćž 文件 with spaces.txt");
        assert_eq!(file.path, "/incoming/report čćž 文件 with spaces.txt");
        assert!(matches!(file.kind, FileKind::File));
        assert_eq!(file.size, 427);
        assert_eq!(file.modified, None, "DOS timestamps must not be guessed");
        let directory = entry_from_listing(
            "/incoming",
            "04-27-00  09:09PM       <DIR>          Folder Name",
        )
        .expect("IIS directory");
        assert!(matches!(directory.kind, FileKind::Directory));
        assert_eq!(directory.path, "/incoming/Folder Name");
        assert_eq!(directory.size, 0);
    }

    #[test]
    fn iis_dos_dot_and_total_rows_are_not_files() {
        assert!(is_dot_listing("04-27-00  09:09PM       <DIR>          ."));
        assert!(is_dot_listing("04-27-00  09:09PM       <DIR>          .."));
        assert!(is_ignorable_listing_line("total 15"));
        assert!(is_ignorable_listing_line(""));
        assert!(!is_ignorable_listing_line("total private notes"));
        assert!(entry_from_listing("/", "04-27-00  09:09PM       <DIR>          ..").is_none());
    }

    #[test]
    fn invalid_dos_records_fail_closed_instead_of_hiding_files() {
        for line in [
            "13-13-25  04:17PM                427 file.txt",
            "02-29-25  04:17PM                427 file.txt",
            "04-27-25  13:17PM                427 file.txt",
            "04-27-25  09:99AM                427 file.txt",
            "04-27-25  09:09AM                -1 file.txt",
            "04-27-25  09:09AM                18446744073709551616 huge",
            "04-27-25  09:09AM                42 ../escape.txt",
            "04-27-25  09:09AM                42 folder\\escape.txt",
            "04-27-25  09:09AM                42",
            "04-27-25  09:09AM                42 bad\nname",
        ] {
            assert!(
                dos_listing_entry(line).is_none(),
                "accepted malformed IIS record: {line:?}"
            );
            assert!(
                entry_from_listing("/root", line).is_none(),
                "malformed IIS row was accepted by the permissive Unix parser: {line:?}"
            );
        }
        assert!(dos_listing_entry("02-29-24  12:01AM       0 leap-day").is_some());
        assert!(dos_listing_entry("01/31/2025  12:01PM       1 slash-date").is_some());
    }

    #[test]
    fn unknown_list_format_refuses_partial_directory_results() {
        let valid = "04-27-25  09:09AM       42 good.txt".to_string();
        let unknown = "LIST malformed entry".to_string();
        let rows = vec![valid.clone(), unknown];
        assert!(parse_directory_listing("/root", rows).is_err());
        let rows = vec!["total 1".to_string(), valid];
        let entries = parse_directory_listing("/root", rows).unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].path, "/root/good.txt");
    }

    #[test]
    fn duplicate_names_fail_closed_before_any_partial_inventory_is_returned() {
        let rows = vec![
            "04-27-25  09:09AM       42 collision.txt".to_string(),
            "04-27-25  09:10AM       <DIR> collision.txt".to_string(),
        ];
        let error = parse_directory_listing("/root", rows).unwrap_err();
        assert!(error
            .to_string()
            .contains("duplicate directory entry names"));
    }

    #[test]
    fn oversized_ftp_listing_entry_count_fails_before_parsing() {
        let rows = vec!["04-27-25  09:09AM       42 item".to_string(); MAX_FTP_LIST_ENTRIES + 1];
        let error = parse_directory_listing("/root", rows).unwrap_err();
        assert!(error.to_string().contains("maximum entry count"));
    }

    #[test]
    fn oversized_ftp_listing_byte_budget_fails_closed() {
        let rows = vec!["x".repeat(MAX_FTP_LIST_RAW_BYTES + 1)];
        let error = parse_directory_listing("/root", rows).unwrap_err();
        assert!(error.to_string().contains("maximum response byte count"));
    }

    #[test]
    fn oversized_ftp_filename_budget_fails_closed() {
        let huge_name = "n".repeat(MAX_FTP_LIST_NAME_BYTES + 1);
        let rows = vec![format!("04-27-25  09:09AM       42 {huge_name}")];
        let error = parse_directory_listing("/root", rows).unwrap_err();
        assert!(error.to_string().contains("maximum filename byte count"));
    }

    #[test]
    fn repeated_names_are_detected_across_unix_and_dos_listing_formats() {
        let rows = vec![
            "-rw-r--r-- 1 ftp ftp 1 Apr 12 12:00 same.txt".to_string(),
            "04-27-25  09:09AM       1 same.txt".to_string(),
        ];
        assert!(parse_directory_listing("/root", rows).is_err());
    }

    #[test]
    fn unix_listing_still_parses_and_dos_listing_stays_non_destructive() {
        let unix = "-rw-r--r-- 1 ftp ftp 5 Apr 12 12:00 ordinary.txt";
        let parsed = entry_from_listing("/root", unix).expect("Unix LIST still supported");
        assert_eq!(parsed.name, "ordinary.txt");
        assert_eq!(parsed.size, 5);
        assert!(matches!(parsed.kind, FileKind::File));
        assert!(entry_from_listing("/root", "LIST server returned malformed data").is_none());
        assert!(
            entry_from_listing("/root", "drwxr-xr-x 1 ftp ftp 0 Apr 12 12:00 Folder").is_some()
        );
        assert!(
            entry_from_listing("/root", "-rw-r--r-- 1 ftp ftp 5 Apr 12 12:00 ../escape").is_none()
        );
        assert!(
            entry_from_listing("/root", "-rw-r--r-- 1 ftp ftp 5 Apr 12 12:00 bad\\name").is_none()
        );
    }
}
