use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::remotefs::dropbox::parse_iso8601;
use crate::session::boxdrive::{basename, normalize, parent_of};
use crate::session::BoxSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use reqwest::Method;
use std::sync::Arc;

/// RemoteFs over the Box API v2. Box is ID-addressed like Google Drive, so it
/// reuses the same resolver pattern (path→id via listing). Real folders,
/// rename/move via PUT, no chmod. `sha1` is the change token for files.
pub struct BoxFs {
    session: Arc<BoxSession>,
}

impl BoxFs {
    pub fn new(session: Arc<BoxSession>) -> Self {
        Self { session }
    }
}

#[async_trait]
impl RemoteFs for BoxFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let folder_id = self.session.folder_id(path).await?;
        let items = self.session.folder_items(&folder_id).await?;
        let mut out = Vec::with_capacity(items.len());
        for item in &items {
            if let Some((entry, id, is_dir)) = entry_from_json(item, path) {
                if is_dir {
                    self.session.cache_path(&entry.path, &id);
                }
                out.push(entry);
            }
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let (id, is_folder) = self
            .session
            .resolve_item(from)
            .await?
            .with_context(|| format!("{from}: not found"))?;
        let to_parent_id = self.session.folder_id(&parent_of(&normalize(to))).await?;
        let new_name = basename(&normalize(to)).to_string();
        let endpoint = if is_folder {
            format!("/folders/{id}")
        } else {
            format!("/files/{id}")
        };
        let body = serde_json::json!({ "name": new_name, "parent": { "id": to_parent_id } });
        self.session
            .rpc(Method::PUT, &endpoint, Some(&body))
            .await
            .with_context(|| format!("move {from} -> {to}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn delete(&self, path: &str, _recursive: bool) -> Result<()> {
        let (id, is_folder) = self
            .session
            .resolve_item(path)
            .await?
            .with_context(|| format!("{path}: not found"))?;
        let endpoint = if is_folder {
            format!("/folders/{id}?recursive=true")
        } else {
            format!("/files/{id}")
        };
        self.session
            .rpc(Method::DELETE, &endpoint, None)
            .await
            .with_context(|| format!("delete {path}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let parent_id = self.session.folder_id(&parent_of(&normalize(path))).await?;
        let body = serde_json::json!({
            "name": basename(&normalize(path)),
            "parent": { "id": parent_id },
        });
        self.session
            .rpc(Method::POST, "/folders", Some(&body))
            .await
            .with_context(|| format!("create dir {path}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow::anyhow!("Box has no POSIX permissions"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            change_signal: ChangeSignal::Etag,
        }
    }
}

/// Parse a Box item into `(DirEntry, id, is_folder)`.
fn entry_from_json(
    item: &serde_json::Value,
    request_path: &str,
) -> Option<(DirEntry, String, bool)> {
    let id = item.get("id").and_then(|x| x.as_str())?.to_string();
    let name = item.get("name").and_then(|x| x.as_str())?.to_string();
    let is_folder = item.get("type").and_then(|t| t.as_str()) == Some("folder");
    let size = item.get("size").and_then(|s| s.as_u64()).unwrap_or(0);
    // Box `modified_at` is RFC3339 with an offset; take the first 19 chars
    // (YYYY-MM-DDTHH:MM:SS) — the offset is dropped (minor display-mtime skew).
    let modified = item
        .get("modified_at")
        .and_then(|m| m.as_str())
        .map(|s| &s[..s.len().min(19)])
        .and_then(parse_iso8601);
    let etag = item
        .get("sha1")
        .and_then(|s| s.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string());
    let base = request_path.trim_matches('/');
    let path = if base.is_empty() {
        format!("/{name}")
    } else {
        format!("/{base}/{name}")
    };
    Some((
        DirEntry {
            name,
            path,
            kind: if is_folder {
                FileKind::Directory
            } else {
                FileKind::File
            },
            size,
            modified,
            mode: None,
            etag,
        },
        id,
        is_folder,
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_box_items() {
        let items = serde_json::json!([
            { "type": "folder", "id": "22222", "name": "Photos" },
            { "type": "file", "id": "33333", "name": "report.pdf", "size": 2048,
              "modified_at": "2024-07-15T09:30:00-07:00", "sha1": "deadbeef" }
        ]);
        let arr = items.as_array().unwrap();
        let parsed: Vec<_> = arr
            .iter()
            .filter_map(|i| entry_from_json(i, "/docs"))
            .collect();
        assert_eq!(parsed.len(), 2);

        let (dir, id, is_dir) = parsed.iter().find(|(e, _, _)| e.name == "Photos").unwrap();
        assert!(is_dir);
        assert_eq!(id, "22222");
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/docs/Photos");

        let (file, _, _) = parsed
            .iter()
            .find(|(e, _, _)| e.name == "report.pdf")
            .unwrap();
        assert_eq!(file.kind, FileKind::File);
        assert_eq!(file.size, 2048);
        assert_eq!(file.etag.as_deref(), Some("deadbeef"));
        assert_eq!(file.modified, Some(1_721_035_800)); // offset dropped
    }

    /// End-to-end against the Box mock (`tests/box_mock.py`). Skipped unless
    /// GHOSTFTP_BOX_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored box_roundtrip` after starting it.
    /// Exercises the ID resolver on a nested path plus token exchange, the
    /// 401→refresh retry, and list/create/upload/download/move/delete.
    #[tokio::test]
    #[ignore = "requires the Box mock (GHOSTFTP_BOX_MOCK_URL)"]
    async fn live_box_roundtrip() {
        use crate::oauth::{self, TokenSet};
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::boxdrive::{box_config, box_connect, BOX_SERVICE};
        use std::time::{SystemTime, UNIX_EPOCH};

        let Ok(mock) = std::env::var("GHOSTFTP_BOX_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_BOX_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_BOX_CLIENT_ID", "test-client");
        std::env::set_var("GHOSTFTP_BOX_TOKEN_URL", format!("{mock}/token"));
        std::env::set_var("GHOSTFTP_BOX_API_BASE", &mock);
        std::env::set_var("GHOSTFTP_BOX_UPLOAD_BASE", &mock);

        let ex = oauth::exchange_code(&box_config(), "authcode", "verifier")
            .await
            .expect("exchange");
        assert_eq!(ex["access_token"], "ACCESS1");

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let pid = "box-mock-test";
        oauth::store_tokens(
            BOX_SERVICE,
            pid,
            &TokenSet {
                access_token: "STALE".into(),
                refresh_token: Some("REFRESH1".into()),
                expires_at: now + 99_999,
            },
        )
        .expect("seed");

        let profile = ConnectionProfile {
            icon: None,
            id: pid.into(),
            name: "bx".into(),
            protocol: "box".into(),
            host: "box.com".into(),
            port: 443,
            username: String::new(),
            auth: AuthMethod::Password {
                password: String::new(),
            },
            default_remote_path: None,
            color: None,
            auto_connect: None,
            bucket: None,
            region: None,
            endpoint: None,
            account: None,
            agent_key: None,
            group: None,
            favorite: None,
            bookmarked: None,
            tags: None,
            last_used: None,
            sort_order: None,
            jump_host: None,
            jump_port: None,
            jump_username: None,
        };
        let session = Arc::new(box_connect(&profile).await.expect("connect"));

        // First call uses STALE → 401 → refresh → succeeds.
        assert_eq!(
            session.account_label().await.expect("account"),
            "tester@example.invalid"
        );

        let fs = BoxFs::new(session.clone());
        fs.create_dir("/ghostftp-test").await.expect("mkdir1");
        fs.create_dir("/ghostftp-test/sub").await.expect("mkdir2");

        // Upload into the nested folder (multipart create, as the transfer path does).
        let parent = session
            .folder_id("/ghostftp-test/sub")
            .await
            .expect("resolve parent");
        let token = session.access_token().await.unwrap();
        let attrs = serde_json::json!({ "name": "hello.txt", "parent": { "id": parent } });
        let part = reqwest::multipart::Part::bytes(b"hi box".to_vec()).file_name("hello.txt");
        let form = reqwest::multipart::Form::new()
            .text("attributes", attrs.to_string())
            .part("file", part);
        let put = session
            .client
            .post(format!("{}/files/content", session.upload_base))
            .bearer_auth(&token)
            .multipart(form)
            .send()
            .await
            .expect("upload");
        assert!(put.status().is_success(), "upload {}", put.status());
        session.clear_cache();

        let entries = fs.list_dir("/ghostftp-test/sub").await.expect("list");
        let hello = entries
            .iter()
            .find(|e| e.name == "hello.txt")
            .expect("hello");
        assert_eq!(hello.kind, FileKind::File);
        assert_eq!(hello.size, 6);
        assert!(hello.etag.is_some(), "sha1 change token");

        // Download via resolver → id → /files/{id}/content.
        let (fid, _) = session
            .resolve_item("/ghostftp-test/sub/hello.txt")
            .await
            .expect("resolve")
            .expect("present");
        let dl = session
            .get_stream(&format!("/files/{fid}/content"))
            .await
            .expect("dl")
            .text()
            .await
            .expect("body");
        assert_eq!(dl, "hi box");

        // Move across folders.
        fs.rename("/ghostftp-test/sub/hello.txt", "/ghostftp-test/renamed.txt")
            .await
            .expect("move");
        assert!(fs
            .list_dir("/ghostftp-test")
            .await
            .expect("l1")
            .iter()
            .any(|e| e.name == "renamed.txt"));
        assert!(!fs
            .list_dir("/ghostftp-test/sub")
            .await
            .expect("l2")
            .iter()
            .any(|e| e.name == "hello.txt"));

        fs.delete("/ghostftp-test", true).await.expect("delete");
        assert!(!fs
            .list_dir("/")
            .await
            .expect("root")
            .iter()
            .any(|e| e.name == "ghostftp-test"));

        oauth::delete_tokens(BOX_SERVICE, pid);
        eprintln!("live_box_roundtrip: OK");
    }
}
