use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::ObjectSession;
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use futures::StreamExt;
use object_store::path::Path as ObjPath;
use object_store::ObjectStore;
use std::sync::Arc;

/// RemoteFs implementation for any object_store-backed session (S3, R2, B2,
/// Azure Blob, …). Semantics are uniform: no real directories, no chmod,
/// rename = copy + delete (object_store handles that for us).
pub struct ObjectFs {
    session: Arc<ObjectSession>,
}

impl ObjectFs {
    pub fn new(session: Arc<ObjectSession>) -> Self {
        Self { session }
    }
}

fn normalize_prefix(raw: &str) -> String {
    let trimmed = raw.trim().trim_matches('/');
    if trimmed.is_empty() || trimmed == "." {
        String::new()
    } else {
        trimmed.to_string()
    }
}

fn entry_for_object(
    key: &str,
    size: u64,
    modified_secs: Option<i64>,
    etag: Option<String>,
) -> DirEntry {
    let name = key.rsplit('/').next().unwrap_or(key).to_string();
    DirEntry {
        name,
        path: format!("/{key}"),
        kind: FileKind::File,
        size,
        modified: modified_secs,
        mode: None,
        etag,
    }
}

fn entry_for_prefix(prefix: &str) -> DirEntry {
    let trimmed = prefix.trim_end_matches('/');
    let name = trimmed.rsplit('/').next().unwrap_or(trimmed).to_string();
    DirEntry {
        name,
        path: format!("/{trimmed}"),
        kind: FileKind::Directory,
        size: 0,
        modified: None,
        mode: None,
        etag: None,
    }
}

#[async_trait]
impl RemoteFs for ObjectFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let prefix = normalize_prefix(path);
        let prefix_path = if prefix.is_empty() {
            None
        } else {
            Some(ObjPath::from(prefix.as_str()))
        };

        let listing = self
            .session
            .store
            .list_with_delimiter(prefix_path.as_ref())
            .await
            .with_context(|| format!("list {}/{prefix}", self.session.container))?;

        let mut out =
            Vec::with_capacity(listing.objects.len() + listing.common_prefixes.len());
        for cp in listing.common_prefixes {
            out.push(entry_for_prefix(cp.as_ref()));
        }
        for obj in listing.objects {
            let modified = obj.last_modified.timestamp();
            out.push(entry_for_object(
                obj.location.as_ref(),
                obj.size as u64,
                Some(modified),
                obj.e_tag.clone(),
            ));
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let src = ObjPath::from(normalize_prefix(from));
        let dst = ObjPath::from(normalize_prefix(to));
        self.session
            .store
            .rename(&src, &dst)
            .await
            .with_context(|| format!("object rename {src} -> {dst}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, recursive: bool) -> Result<()> {
        let key = normalize_prefix(path);
        let target = ObjPath::from(key.as_str());
        let is_object = self.session.store.head(&target).await.is_ok();

        if is_object {
            self.session
                .store
                .delete(&target)
                .await
                .with_context(|| format!("delete {target}"))?;
            return Ok(());
        }
        if !recursive {
            return Err(anyhow!(
                "{target} is a prefix; pass recursive=true to remove all objects under it"
            ));
        }
        let mut stream = self.session.store.list(Some(&target));
        while let Some(meta) = stream.next().await {
            let meta = meta.with_context(|| format!("list under {target}"))?;
            self.session
                .store
                .delete(&meta.location)
                .await
                .with_context(|| format!("delete {}", meta.location))?;
        }
        Ok(())
    }

    async fn create_dir(&self, _path: &str) -> Result<()> {
        // Object stores have no folders. Accepted as a no-op so the UI's
        // mkdir flow doesn't error; the prefix appears once the first
        // object lands in it.
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow!("object stores have no POSIX permissions"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: true,
            has_directories: false,
            has_shell: false,
            // Object stores expose an ETag per object — an opaque change token
            // (not necessarily an MD5 for multipart uploads). See ChangeSignal.
            change_signal: ChangeSignal::Etag,
        }
    }
}
