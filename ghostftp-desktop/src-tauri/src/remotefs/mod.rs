use async_trait::async_trait;
use serde::{Deserialize, Serialize};

pub mod agent;
pub mod boxdrive;
pub mod dropbox;
pub mod dynamics;
pub mod ftp;
pub mod gdrive;
pub mod http;
pub mod hubspot;
pub mod local;
pub mod object;
pub mod onedrive;
pub mod sftp;
pub mod shopify;
pub mod webdav;

/// Fail closed before any remote backend executes a destructive delete.
/// Empty/root targets and dot path segments can otherwise expand a recursive
/// delete far beyond the item the user selected.
pub(crate) fn validate_remote_delete_path(path: &str) -> anyhow::Result<()> {
    let parts: Vec<&str> = path
        .trim()
        .split('/')
        .filter(|segment| !segment.is_empty())
        .collect();
    if parts.is_empty() || parts.iter().any(|segment| matches!(*segment, "." | "..")) {
        anyhow::bail!("refusing unsafe remote delete path: {path:?}");
    }
    Ok(())
}

/// Defensive collision identity for remote endpoints. Case-insensitive servers
/// may resolve two different visible names to the same file. This conservative
/// comparison prevents known case-fold aliases from becoming silent overwrites.
/// Some filesystems apply more complex Unicode normalization, so this is not
/// an atomic create-if-absent guarantee.
pub(crate) fn remote_names_may_alias(existing: &str, requested: &str) -> bool {
    existing == requested || existing.to_lowercase() == requested.to_lowercase()
}

/// Resolve the complete parent inventory that must be checked before a
/// non-overwriting remote rename. Relative paths use ".", absolute-root
/// children use "/"; trailing separators cannot masquerade as file targets.
pub(crate) fn remote_destination_parent_name(path: &str) -> anyhow::Result<(&str, &str)> {
    validate_protocol_mutation_path(path)?;
    let (parent, name) = match path.rsplit_once('/') {
        Some(("", name)) => ("/", name),
        Some((parent, name)) => (parent, name),
        None => (".", path),
    };
    anyhow::ensure!(
        !name.is_empty() && name != "." && name != "..",
        "remote rename requires a concrete destination filename"
    );
    Ok((parent, name))
}

/// FTP and SFTP mutation operands are paths, not arbitrary filenames. A
/// spelling with parent traversal, C0 controls or Windows path separators can
/// reach an unexpected remote target, especially on IIS/Windows servers.
/// Read-only navigation uses separate guards and can still list "/" or ".".
pub(crate) fn validate_protocol_mutation_path(path: &str) -> anyhow::Result<()> {
    validate_remote_delete_path(path)?;
    anyhow::ensure!(
        path.len() <= 4096 && !path.chars().any(|ch| ch.is_ascii_control() || ch == '\\'),
        "refusing unsafe FTP/SFTP mutation path"
    );
    Ok(())
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FileKind {
    File,
    Directory,
    Symlink,
    Other,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DirEntry {
    pub name: String,
    pub path: String,
    pub kind: FileKind,
    pub size: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub modified: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mode: Option<u32>,
    /// Backend-supplied opaque change token when it has one (object stores
    /// expose an ETag). Treat it as a change *token*, not a content hash —
    /// multipart-S3 ETags aren't MD5s. `None` for backends that rely on
    /// mtime+size instead. See [`ChangeSignal`].
    #[serde(skip_serializing_if = "Option::is_none")]
    pub etag: Option<String>,
}

/// The cheapest reliable "did this file change" a backend can offer. Consumed by
/// folder-sync's `sync_state` index (Plan 3), diff's `--hash` mode (Plan 6), and
/// declared per-backend so callers stop guessing from mtime.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ChangeSignal {
    /// POSIX-ish: compare size + mtime (SFTP, FTP, local, agent).
    MtimeSize,
    /// Object stores: compare the opaque ETag on `DirEntry.etag`.
    Etag,
    /// Neither is reliable; a content hash is required (no backend declares this
    /// today — reserved so a future backend can opt in without a schema change).
    Hash,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Capabilities {
    pub can_chmod: bool,
    pub can_symlink: bool,
    pub can_rename: bool,
    pub has_directories: bool,
    /// Backend runs shell commands (gates server-side archive + "open terminal
    /// here"). True for SSH; false for local/FTP/object stores.
    pub has_shell: bool,
    /// How this backend reports change — drives the sync index's staleness check.
    pub change_signal: ChangeSignal,
}

/// Unified filesystem abstraction. Each backend (Local, SFTP, S3, FTP, ...)
/// implements this trait. The UI queries `capabilities()` to know what actions
/// to expose per-backend.
#[async_trait]
pub trait RemoteFs: Send + Sync {
    async fn list_dir(&self, path: &str) -> anyhow::Result<Vec<DirEntry>>;
    async fn rename(&self, from: &str, to: &str) -> anyhow::Result<()>;
    /// Delete a file. For a directory, `recursive=true` walks children first.
    async fn delete(&self, path: &str, recursive: bool) -> anyhow::Result<()>;
    async fn create_dir(&self, path: &str) -> anyhow::Result<()>;
    async fn chmod(&self, path: &str, mode: u32) -> anyhow::Result<()>;
    fn capabilities(&self) -> Capabilities;
}

#[cfg(test)]
mod destructive_path_tests {
    use super::{
        remote_destination_parent_name, remote_names_may_alias, validate_protocol_mutation_path,
        validate_remote_delete_path,
    };

    #[test]
    fn rejects_root_and_dot_segments() {
        for path in ["", "/", "////", ".", "..", "/srv/../etc", "srv/./file"] {
            assert!(
                validate_remote_delete_path(path).is_err(),
                "{path:?} must be rejected"
            );
        }
        for path in ["file.txt", "/srv/file.txt", "folder/subfolder"] {
            assert!(
                validate_remote_delete_path(path).is_ok(),
                "{path:?} must be allowed"
            );
        }
    }

    #[test]
    fn cross_protocol_case_aliases_never_prove_a_remote_name_is_free() {
        for (existing, requested) in [
            ("REPORT.txt", "report.TXT"),
            (".ENV", ".env"),
            ("RÉSUMÉ.pdf", "résumé.pdf"),
        ] {
            assert!(remote_names_may_alias(existing, requested));
        }
        for (existing, requested) in [
            ("report.txt", "report.csv"),
            (".env.sample", ".env"),
            ("file-one", "file-two"),
        ] {
            assert!(!remote_names_may_alias(existing, requested));
        }
    }

    #[test]
    fn rename_destination_parent_is_normalized_without_discarding_root() {
        for (path, expected) in [
            ("report.txt", (".", "report.txt")),
            ("/report.txt", ("/", "report.txt")),
            ("/srv/archive/report.txt", ("/srv/archive", "report.txt")),
        ] {
            assert_eq!(remote_destination_parent_name(path).unwrap(), expected);
        }
        for path in ["", "/", "../escape", "folder/", "file\\bad", "file\n"] {
            assert!(remote_destination_parent_name(path).is_err());
        }
    }

    #[test]
    fn protocol_mutations_reject_traversal_controls_and_windows_separators() {
        for path in [
            "",
            " ",
            "/",
            ".",
            "..",
            "/safe/../secret",
            "safe/./config",
            "logs\\..\\private",
            "dir\\file",
            "safe\0tail",
            "safe\r\nDELE /",
            "safe\ttab",
            "safe\u{7f}del",
        ] {
            assert!(
                validate_protocol_mutation_path(path).is_err(),
                "unsafe FTP/SFTP mutation accepted: {path:?}"
            );
        }
        assert!(validate_protocol_mutation_path(&"x".repeat(4097)).is_err());

        for path in [
            "new folder",
            "folder/child",
            "/srv/site/index.html",
            "/home/računi/čćž 文件 2026.pdf",
            ".hidden",
        ] {
            assert!(
                validate_protocol_mutation_path(path).is_ok(),
                "valid FTP/SFTP mutation rejected: {path:?}"
            );
        }
    }
}
