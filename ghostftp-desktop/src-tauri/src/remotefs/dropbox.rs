use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::remotefs::webdav::days_from_civil;
use crate::session::DropboxSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use std::sync::Arc;

/// RemoteFs over the Dropbox API v2. Dropbox is path-addressed, so it slots
/// straight into the trait: real folders, rename via `move_v2`, no chmod. The
/// per-file `rev` is a reliable change token (`ChangeSignal::Etag`).
pub struct DropboxFs {
    session: Arc<DropboxSession>,
}

impl DropboxFs {
    pub fn new(session: Arc<DropboxSession>) -> Self {
        Self { session }
    }
}

/// Translate a Ghost FTP path (`/a/b`, or `/` for root) into a Dropbox API path
/// (`""` for root, otherwise `/a/b`).
pub fn dropbox_api_path(ghostftp_path: &str) -> String {
    let trimmed = ghostftp_path.trim().trim_matches('/');
    if trimmed.is_empty() || trimmed == "." {
        String::new()
    } else {
        format!("/{trimmed}")
    }
}

#[async_trait]
impl RemoteFs for DropboxFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let dbx = dropbox_api_path(path);
        let mut out = Vec::new();

        let mut resp = self
            .session
            .rpc(
                "/2/files/list_folder",
                serde_json::json!({ "path": dbx, "recursive": false }),
            )
            .await?;

        loop {
            if let Some(entries) = resp.get("entries").and_then(|e| e.as_array()) {
                for e in entries {
                    if let Some(entry) = entry_from_json(e) {
                        out.push(entry);
                    }
                }
            }
            let has_more = resp
                .get("has_more")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);
            if !has_more {
                break;
            }
            let cursor = resp
                .get("cursor")
                .and_then(|c| c.as_str())
                .unwrap_or("")
                .to_string();
            resp = self
                .session
                .rpc(
                    "/2/files/list_folder/continue",
                    serde_json::json!({ "cursor": cursor }),
                )
                .await?;
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        self.session
            .rpc(
                "/2/files/move_v2",
                serde_json::json!({
                    "from_path": dropbox_api_path(from),
                    "to_path": dropbox_api_path(to),
                    "autorename": false,
                }),
            )
            .await
            .with_context(|| format!("move {from} -> {to}"))?;
        Ok(())
    }

    async fn delete(&self, path: &str, _recursive: bool) -> Result<()> {
        // delete_v2 removes a folder and its contents; the flag needs no handling.
        self.session
            .rpc(
                "/2/files/delete_v2",
                serde_json::json!({ "path": dropbox_api_path(path) }),
            )
            .await
            .with_context(|| format!("delete {path}"))?;
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        self.session
            .rpc(
                "/2/files/create_folder_v2",
                serde_json::json!({ "path": dropbox_api_path(path), "autorename": false }),
            )
            .await
            .with_context(|| format!("create dir {path}"))?;
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow::anyhow!("Dropbox has no POSIX permissions"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            // Dropbox's per-file `rev` changes on every edit — a reliable token.
            change_signal: ChangeSignal::Etag,
        }
    }
}

/// Parse one `list_folder` entry into a DirEntry, or `None` for a shape we don't
/// recognize (e.g. a deleted tombstone).
fn entry_from_json(e: &serde_json::Value) -> Option<DirEntry> {
    let tag = e.get(".tag").and_then(|t| t.as_str())?;
    let kind = match tag {
        "folder" => FileKind::Directory,
        "file" => FileKind::File,
        _ => return None, // "deleted" or unknown
    };
    let name = e.get("name").and_then(|n| n.as_str())?.to_string();
    // path_display is the properly-cased server path; fall back to path_lower.
    let path = e
        .get("path_display")
        .or_else(|| e.get("path_lower"))
        .and_then(|p| p.as_str())
        .map(|s| s.to_string())
        .unwrap_or_else(|| format!("/{name}"));
    let size = e.get("size").and_then(|s| s.as_u64()).unwrap_or(0);
    let modified = e
        .get("server_modified")
        .and_then(|m| m.as_str())
        .and_then(parse_iso8601);
    let etag = e.get("rev").and_then(|r| r.as_str()).map(|s| s.to_string());
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

/// Parse a fixed-form ISO-8601 UTC timestamp (`2024-07-15T09:30:00Z`, as Dropbox
/// and Graph emit) into a Unix timestamp. Dependency-free — the format is fixed.
/// Shared with the OneDrive backend.
pub(crate) fn parse_iso8601(s: &str) -> Option<i64> {
    let s = s.trim();
    let (date, rest) = s.split_once('T')?;
    let time = rest.trim_end_matches('Z');
    let d: Vec<&str> = date.split('-').collect();
    let t: Vec<&str> = time.split(':').collect();
    if d.len() != 3 || t.len() != 3 {
        return None;
    }
    let year: i64 = d[0].parse().ok()?;
    let month: i64 = d[1].parse().ok()?;
    let day: i64 = d[2].parse().ok()?;
    let hh: i64 = t[0].parse().ok()?;
    let mm: i64 = t[1].parse().ok()?;
    // Seconds may carry a fractional part (…:30.5Z); take the integer part.
    let ss: i64 = t[2].split('.').next()?.parse().ok()?;
    Some(days_from_civil(year, month, day) * 86400 + hh * 3600 + mm * 60 + ss)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn api_path_mapping() {
        assert_eq!(dropbox_api_path("/"), "");
        assert_eq!(dropbox_api_path(""), "");
        assert_eq!(dropbox_api_path("."), "");
        assert_eq!(dropbox_api_path("/docs"), "/docs");
        assert_eq!(dropbox_api_path("docs/sub/"), "/docs/sub");
    }

    #[test]
    fn iso8601_epoch() {
        assert_eq!(parse_iso8601("1970-01-01T00:00:00Z"), Some(0));
        assert_eq!(parse_iso8601("2024-07-15T09:30:00Z"), Some(1_721_035_800));
        assert_eq!(parse_iso8601("2024-07-15T09:30:00.5Z"), Some(1_721_035_800));
        assert_eq!(parse_iso8601("nope"), None);
    }

    #[test]
    fn parses_list_folder_entries() {
        let listing = serde_json::json!({
            "entries": [
                { ".tag": "folder", "name": "Photos",
                  "path_display": "/Photos", "path_lower": "/photos", "id": "id:1" },
                { ".tag": "file", "name": "report.pdf", "path_display": "/report.pdf",
                  "size": 2048, "server_modified": "2024-07-15T09:30:00Z",
                  "rev": "0158abc", "id": "id:2" },
                { ".tag": "deleted", "name": "gone.txt", "path_display": "/gone.txt" }
            ],
            "cursor": "AAA",
            "has_more": false
        });
        let entries: Vec<DirEntry> = listing["entries"]
            .as_array()
            .unwrap()
            .iter()
            .filter_map(entry_from_json)
            .collect();
        // The "deleted" tombstone is dropped.
        assert_eq!(entries.len(), 2);

        let dir = entries.iter().find(|e| e.name == "Photos").unwrap();
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/Photos");

        let file = entries.iter().find(|e| e.name == "report.pdf").unwrap();
        assert_eq!(file.kind, FileKind::File);
        assert_eq!(file.size, 2048);
        assert_eq!(file.etag.as_deref(), Some("0158abc"));
        assert_eq!(file.modified, Some(1_721_035_800));
    }

    /// End-to-end against the Dropbox mock (`tests/dropbox_mock.py`). Skipped
    /// unless GHOSTFTP_DROPBOX_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored dropbox_roundtrip` after starting the
    /// mock. Exercises the whole client path — the 401→refresh retry,
    /// get_current_account, list/create/upload/download/move/delete — through
    /// Ghost FTP's own DropboxSession/DropboxFs, no real account needed.
    #[tokio::test]
    #[ignore = "requires the Dropbox mock (GHOSTFTP_DROPBOX_MOCK_URL)"]
    async fn live_dropbox_roundtrip() {
        use crate::oauth::{self, TokenSet};
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::dropbox::{dropbox_config, dropbox_connect, DROPBOX_SERVICE};
        use std::time::{SystemTime, UNIX_EPOCH};

        let Ok(mock) = std::env::var("GHOSTFTP_DROPBOX_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_DROPBOX_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_DROPBOX_APP_KEY", "test-app-key");
        std::env::set_var("GHOSTFTP_DROPBOX_TOKEN_URL", format!("{mock}/oauth2/token"));
        std::env::set_var("GHOSTFTP_DROPBOX_API_BASE", &mock);
        std::env::set_var("GHOSTFTP_DROPBOX_CONTENT_BASE", &mock);

        // First: a real token exchange against the mock (the code path authorize
        // uses after the browser redirect).
        let exchanged = oauth::exchange_code(&dropbox_config(), "authcode", "verifier")
            .await
            .expect("exchange");
        assert_eq!(exchanged["access_token"], "ACCESS1");
        assert_eq!(exchanged["refresh_token"], "REFRESH1");

        // Seed a deliberately-stale-but-unexpired access token so the very first
        // API call 401s and drives the force-refresh retry path.
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let pid = "dropbox-mock-test";
        oauth::store_tokens(
            DROPBOX_SERVICE,
            pid,
            &TokenSet {
                access_token: "STALE".into(),
                refresh_token: Some("REFRESH1".into()),
                expires_at: now + 99_999,
            },
        )
        .expect("seed tokens");

        let profile = ConnectionProfile {
            icon: None,
            id: pid.into(),
            name: "dbx".into(),
            protocol: "dropbox".into(),
            host: "dropbox.com".into(),
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
        let session = Arc::new(dropbox_connect(&profile).await.expect("connect"));

        // First API call uses the STALE token → 401 → force_refresh → succeeds.
        let label = session.account_label().await.expect("account");
        assert_eq!(label, "tester@example.invalid");

        let fs = DropboxFs::new(session.clone());
        fs.create_dir("/ghostftp-test").await.expect("mkdir");

        // Upload through the content endpoint (same call the transfer manager makes).
        let token = session.access_token().await.unwrap();
        let arg = serde_json::json!({
            "path": "/ghostftp-test/hello.txt", "mode": "overwrite"
        })
        .to_string();
        let put = session
            .client
            .post(format!("{}/2/files/upload", session.content_base))
            .bearer_auth(&token)
            .header("Dropbox-API-Arg", &arg)
            .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
            .body("hello dropbox")
            .send()
            .await
            .expect("upload");
        assert!(put.status().is_success());

        // List sees it with the right size.
        let entries = fs.list_dir("/ghostftp-test").await.expect("list");
        let hello = entries
            .iter()
            .find(|e| e.name == "hello.txt")
            .expect("hello");
        assert_eq!(hello.kind, FileKind::File);
        assert_eq!(hello.size, 13);

        // Download round-trips the bytes.
        let dl = session
            .content_get(
                "/2/files/download",
                &serde_json::json!({"path": "/ghostftp-test/hello.txt"}).to_string(),
            )
            .await
            .expect("download")
            .text()
            .await
            .expect("body");
        assert_eq!(dl, "hello dropbox");

        // Move, then delete.
        fs.rename("/ghostftp-test/hello.txt", "/ghostftp-test/renamed.txt")
            .await
            .expect("move");
        let entries = fs.list_dir("/ghostftp-test").await.expect("list2");
        assert!(entries.iter().any(|e| e.name == "renamed.txt"));
        assert!(!entries.iter().any(|e| e.name == "hello.txt"));

        fs.delete("/ghostftp-test", true).await.expect("delete");
        let root = fs.list_dir("/").await.expect("list root");
        assert!(!root.iter().any(|e| e.name == "ghostftp-test"));

        oauth::delete_tokens(DROPBOX_SERVICE, pid);
        eprintln!("live_dropbox_roundtrip: OK");
    }
}
