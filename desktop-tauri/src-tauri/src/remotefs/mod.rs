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
