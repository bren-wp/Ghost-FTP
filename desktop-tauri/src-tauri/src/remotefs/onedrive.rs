use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::remotefs::dropbox::parse_iso8601;
use crate::session::OneDriveSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use percent_encoding::{utf8_percent_encode, AsciiSet, CONTROLS};
use reqwest::Method;
use std::sync::Arc;

/// Characters to percent-encode in a path segment: controls, space, and the URL
/// reserved/delimiter chars — but NOT the unreserved set (`. - _ ~`), so a normal
/// filename like `report.pdf` passes through unchanged.
const PATH_ENC: &AsciiSet = &CONTROLS
    .add(b' ')
    .add(b'"')
    .add(b'#')
    .add(b'%')
    .add(b'?')
    .add(b'/')
    .add(b'\\')
    .add(b'<')
    .add(b'>')
    .add(b'{')
    .add(b'}')
    .add(b'|')
    .add(b'^')
    .add(b'`');

/// RemoteFs over OneDrive via Microsoft Graph. Graph supports path addressing
/// (`/me/drive/root:/path:`), so it slots into the trait like Dropbox: real
/// folders, rename/move via PATCH, no chmod. The per-item `cTag` is the change
/// token.
pub struct OneDriveFs {
    session: Arc<OneDriveSession>,
}

impl OneDriveFs {
    pub fn new(session: Arc<OneDriveSession>) -> Self {
        Self { session }
    }
}

fn enc(seg: &str) -> String {
    utf8_percent_encode(seg, PATH_ENC).to_string()
}

/// Percent-encode a Ghost FTP path's segments, keeping the slashes.
fn enc_path(ghostftp: &str) -> String {
    ghostftp.trim_matches('/')
        .split('/')
        .filter(|s| !s.is_empty())
        .map(enc)
        .collect::<Vec<_>>()
        .join("/")
}

fn is_root(ghostftp: &str) -> bool {
    let p = ghostftp.trim().trim_matches('/');
    p.is_empty() || p == "."
}

/// Graph item reference for a path (addresses the item itself).
pub fn item_ref(ghostftp: &str) -> String {
    if is_root(ghostftp) {
        "/me/drive/root".to_string()
    } else {
        format!("/me/drive/root:/{}:", enc_path(ghostftp))
    }
}

/// Graph children reference for a directory path.
fn children_ref(ghostftp: &str) -> String {
    if is_root(ghostftp) {
        "/me/drive/root/children".to_string()
    } else {
        format!("/me/drive/root:/{}:/children", enc_path(ghostftp))
    }
}

/// Graph content reference for a file path (download/upload bytes).
pub fn content_ref(ghostftp: &str) -> String {
    format!("/me/drive/root:/{}:/content", enc_path(ghostftp))
}

fn basename(ghostftp: &str) -> &str {
    ghostftp.trim_matches('/').rsplit('/').next().unwrap_or("")
}

fn parent_of(ghostftp: &str) -> String {
    let t = ghostftp.trim_matches('/');
    match t.rsplit_once('/') {
        Some((p, _)) => p.to_string(),
        None => String::new(),
    }
}

/// Graph parentReference path for a Ghost FTP directory (`/drive/root:` at root).
fn parent_path_ref(parent_ghostftp: &str) -> String {
    if is_root(parent_ghostftp) {
        "/drive/root:".to_string()
    } else {
        format!("/drive/root:/{}", enc_path(parent_ghostftp))
    }
}

#[async_trait]
impl RemoteFs for OneDriveFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let mut out = Vec::new();
        let mut next = children_ref(path);
        loop {
            let page = self.session.rpc(Method::GET, &next, None).await?;
            if let Some(items) = page.get("value").and_then(|v| v.as_array()) {
                for item in items {
                    if let Some(e) = entry_from_json(item, path) {
                        out.push(e);
                    }
                }
            }
            match page.get("@odata.nextLink").and_then(|v| v.as_str()) {
                Some(url) => next = url.to_string(),
                None => break,
            }
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let body = serde_json::json!({
            "name": basename(to),
            "parentReference": { "path": parent_path_ref(&parent_of(to)) },
        });
        self.session
            .rpc(Method::PATCH, &item_ref(from), Some(&body))
            .await
            .with_context(|| format!("move {from} -> {to}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, _recursive: bool) -> Result<()> {
        // Graph DELETE removes a folder and its contents.
        self.session
            .rpc(Method::DELETE, &item_ref(path), None)
            .await
            .with_context(|| format!("delete {path}"))?;
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let parent = parent_of(path);
        let body = serde_json::json!({
            "name": basename(path),
            "folder": {},
            "@microsoft.graph.conflictBehavior": "fail",
        });
        self.session
            .rpc(Method::POST, &children_ref(&parent), Some(&body))
            .await
            .with_context(|| format!("create dir {path}"))?;
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow::anyhow!("OneDrive has no POSIX permissions"))
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

/// Parse a Graph driveItem into a DirEntry (path built under `request_path`).
fn entry_from_json(item: &serde_json::Value, request_path: &str) -> Option<DirEntry> {
    let name = item.get("name").and_then(|n| n.as_str())?.to_string();
    let kind = if item.get("folder").is_some() {
        FileKind::Directory
    } else {
        FileKind::File
    };
    let size = item.get("size").and_then(|s| s.as_u64()).unwrap_or(0);
    let modified = item
        .get("lastModifiedDateTime")
        .and_then(|m| m.as_str())
        .and_then(parse_iso8601);
    // cTag tracks content changes; fall back to eTag.
    let etag = item
        .get("cTag")
        .or_else(|| item.get("eTag"))
        .and_then(|t| t.as_str())
        .map(|s| s.to_string());
    let base = request_path.trim_matches('/');
    let path = if base.is_empty() {
        format!("/{name}")
    } else {
        format!("/{base}/{name}")
    };
    Some(DirEntry {
        name,
        path,
        kind,
        size,
        modified,
        mode: None,
        etag,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn graph_path_refs() {
        assert_eq!(item_ref("/"), "/me/drive/root");
        assert_eq!(item_ref("/a/b"), "/me/drive/root:/a/b:");
        assert_eq!(children_ref("/"), "/me/drive/root/children");
        assert_eq!(children_ref("/docs"), "/me/drive/root:/docs:/children");
        assert_eq!(content_ref("/a/f.txt"), "/me/drive/root:/a/f.txt:/content");
        // Spaces get percent-encoded.
        assert_eq!(item_ref("/my docs"), "/me/drive/root:/my%20docs:");
        assert_eq!(parent_path_ref(""), "/drive/root:");
        assert_eq!(parent_path_ref("/a"), "/drive/root:/a");
    }

    #[test]
    fn parses_graph_children() {
        let page = serde_json::json!({
            "value": [
                { "name": "Photos", "folder": { "childCount": 3 }, "size": 0,
                  "lastModifiedDateTime": "2024-07-14T10:00:00Z", "cTag": "ctag1", "id": "1" },
                { "name": "report.pdf", "file": { "mimeType": "application/pdf" },
                  "size": 2048, "lastModifiedDateTime": "2024-07-15T09:30:00Z",
                  "cTag": "ctag2", "eTag": "etag2", "id": "2" }
            ]
        });
        let items = page["value"].as_array().unwrap();
        let entries: Vec<DirEntry> = items
            .iter()
            .filter_map(|i| entry_from_json(i, "/docs"))
            .collect();
        assert_eq!(entries.len(), 2);
        let dir = entries.iter().find(|e| e.name == "Photos").unwrap();
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/docs/Photos");
        let file = entries.iter().find(|e| e.name == "report.pdf").unwrap();
        assert_eq!(file.kind, FileKind::File);
        assert_eq!(file.size, 2048);
        assert_eq!(file.etag.as_deref(), Some("ctag2"));
        assert_eq!(file.modified, Some(1_721_035_800));
    }

    /// End-to-end against the Graph mock (`tests/onedrive_mock.py`). Skipped
    /// unless GHOSTFTP_ONEDRIVE_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored onedrive_roundtrip` after starting it.
    /// Drives the whole client path — token exchange, 401→refresh retry, /me,
    /// list/create/upload/download/move/delete — through Ghost FTP's own code.
    #[tokio::test]
    #[ignore = "requires the OneDrive mock (GHOSTFTP_ONEDRIVE_MOCK_URL)"]
    async fn live_onedrive_roundtrip() {
        use crate::oauth::{self, TokenSet};
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::onedrive::{onedrive_config, onedrive_connect, ONEDRIVE_SERVICE};
        use std::time::{SystemTime, UNIX_EPOCH};

        let Ok(mock) = std::env::var("GHOSTFTP_ONEDRIVE_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_ONEDRIVE_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_ONEDRIVE_CLIENT_ID", "test-client");
        std::env::set_var("GHOSTFTP_ONEDRIVE_TOKEN_URL", format!("{mock}/oauth2/token"));
        std::env::set_var("GHOSTFTP_ONEDRIVE_GRAPH_BASE", &mock);

        // Real token exchange against the mock.
        let ex = oauth::exchange_code(&onedrive_config(), "authcode", "verifier")
            .await
            .expect("exchange");
        assert_eq!(ex["access_token"], "ACCESS1");

        // Seed a stale-but-unexpired token so the first call 401s → force refresh.
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as i64;
        let pid = "onedrive-mock-test";
        oauth::store_tokens(
            ONEDRIVE_SERVICE,
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
            name: "od".into(),
            protocol: "onedrive".into(),
            host: "onedrive.com".into(),
            port: 443,
            username: String::new(),
            auth: AuthMethod::Password { password: String::new() },
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
        let session = Arc::new(onedrive_connect(&profile).await.expect("connect"));

        // First API call uses STALE → 401 → force_refresh → succeeds.
        assert_eq!(session.account_label().await.expect("account"), "tester@example.com");

        let fs = OneDriveFs::new(session.clone());
        fs.create_dir("/ghostftp-test").await.expect("mkdir");

        // Simple upload (PUT …/content), then list/download/move/delete.
        let token = session.access_token().await.unwrap();
        let put = session
            .client
            .put(format!("{}{}", session.graph_base, content_ref("/ghostftp-test/hello.txt")))
            .bearer_auth(&token)
            .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
            .body("hi onedrive")
            .send()
            .await
            .expect("upload");
        assert!(put.status().is_success(), "upload {}", put.status());

        let entries = fs.list_dir("/ghostftp-test").await.expect("list");
        let hello = entries.iter().find(|e| e.name == "hello.txt").expect("hello");
        assert_eq!(hello.kind, FileKind::File);
        assert_eq!(hello.size, 11);

        let dl = session
            .get_stream(&content_ref("/ghostftp-test/hello.txt"))
            .await
            .expect("download")
            .text()
            .await
            .expect("body");
        assert_eq!(dl, "hi onedrive");

        fs.rename("/ghostftp-test/hello.txt", "/ghostftp-test/renamed.txt")
            .await
            .expect("move");
        let entries = fs.list_dir("/ghostftp-test").await.expect("list2");
        assert!(entries.iter().any(|e| e.name == "renamed.txt"));
        assert!(!entries.iter().any(|e| e.name == "hello.txt"));

        fs.delete("/ghostftp-test", true).await.expect("delete");
        assert!(!fs
            .list_dir("/")
            .await
            .expect("root")
            .iter()
            .any(|e| e.name == "ghostftp-test"));

        oauth::delete_tokens(ONEDRIVE_SERVICE, pid);
        eprintln!("live_onedrive_roundtrip: OK");
    }
}
