use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::remotefs::dropbox::parse_iso8601;
use crate::session::gdrive::{basename, folder_mime_is, normalize, parent_of};
use crate::session::GDriveSession;
use anyhow::{Context, Result};
use async_trait::async_trait;
use percent_encoding::{utf8_percent_encode, NON_ALPHANUMERIC};
use reqwest::Method;
use std::sync::Arc;

/// RemoteFs over Google Drive v3. Drive is **ID-addressed**, so each op resolves
/// the Ghost FTP path to a file id via the session's resolver first, then acts on the
/// id. Real folders, rename/move via PATCH, no chmod. `md5Checksum` is the change
/// token for binary files (absent for folders / Google-native docs → mtime+size).
pub struct GDriveFs {
    session: Arc<GDriveSession>,
}

impl GDriveFs {
    pub fn new(session: Arc<GDriveSession>) -> Self {
        Self { session }
    }
}

fn urlenc(s: &str) -> String {
    utf8_percent_encode(s, NON_ALPHANUMERIC).to_string()
}

#[async_trait]
impl RemoteFs for GDriveFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let folder_id = self.session.folder_id(path).await?;
        let q = format!("'{}' in parents and trashed = false", folder_id);
        let mut out = Vec::new();
        let mut page_token: Option<String> = None;
        loop {
            let mut url = format!(
                "/files?q={}&fields=nextPageToken,files(id,name,mimeType,size,modifiedTime,md5Checksum)&pageSize=200",
                urlenc(&q)
            );
            if let Some(tok) = &page_token {
                url.push_str(&format!("&pageToken={}", urlenc(tok)));
            }
            let v = self.session.rpc(Method::GET, &url, None).await?;
            if let Some(files) = v.get("files").and_then(|f| f.as_array()) {
                for f in files {
                    if let Some((entry, id, is_dir)) = entry_from_json(f, path) {
                        // Warm the resolver cache for subfolders so navigating in
                        // doesn't re-query.
                        if is_dir {
                            self.session.cache_path(&entry.path, &id);
                        }
                        out.push(entry);
                    }
                }
            }
            match v.get("nextPageToken").and_then(|t| t.as_str()) {
                Some(tok) => page_token = Some(tok.to_string()),
                None => break,
            }
        }
        Ok(out)
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let (id, _) = self
            .session
            .resolve_item(from)
            .await?
            .with_context(|| format!("{from}: not found"))?;
        let from_parent = self.session.folder_id(&parent_of(&normalize(from))).await?;
        let to_parent = self.session.folder_id(&parent_of(&normalize(to))).await?;
        let new_name = basename(&normalize(to)).to_string();

        let mut url = format!("/files/{id}?fields=id");
        if from_parent != to_parent {
            url.push_str(&format!(
                "&addParents={}&removeParents={}",
                urlenc(&to_parent),
                urlenc(&from_parent)
            ));
        }
        let body = serde_json::json!({ "name": new_name });
        self.session
            .rpc(Method::PATCH, &url, Some(&body))
            .await
            .with_context(|| format!("move {from} -> {to}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn delete(&self, path: &str, _recursive: bool) -> Result<()> {
        let (id, _) = self
            .session
            .resolve_item(path)
            .await?
            .with_context(|| format!("{path}: not found"))?;
        // DELETE removes a folder and its contents permanently.
        self.session
            .rpc(Method::DELETE, &format!("/files/{id}"), None)
            .await
            .with_context(|| format!("delete {path}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let parent_id = self.session.folder_id(&parent_of(&normalize(path))).await?;
        let body = serde_json::json!({
            "name": basename(&normalize(path)),
            "mimeType": crate::session::gdrive::FOLDER_MIME,
            "parents": [parent_id],
        });
        self.session
            .rpc(Method::POST, "/files?fields=id", Some(&body))
            .await
            .with_context(|| format!("create dir {path}"))?;
        self.session.clear_cache();
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow::anyhow!("Google Drive has no POSIX permissions"))
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

/// Parse a Drive file resource into `(DirEntry, id, is_folder)`.
fn entry_from_json(f: &serde_json::Value, request_path: &str) -> Option<(DirEntry, String, bool)> {
    let id = f.get("id").and_then(|x| x.as_str())?.to_string();
    let name = f.get("name").and_then(|x| x.as_str())?.to_string();
    let is_folder = folder_mime_is(f.get("mimeType").and_then(|m| m.as_str()).unwrap_or(""));
    // Drive returns `size` as a decimal string, and only for binary files.
    let size = f
        .get("size")
        .and_then(|s| s.as_str())
        .and_then(|s| s.parse::<u64>().ok())
        .unwrap_or(0);
    let modified = f
        .get("modifiedTime")
        .and_then(|m| m.as_str())
        .and_then(parse_iso8601);
    let etag = f
        .get("md5Checksum")
        .and_then(|m| m.as_str())
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
    use crate::session::gdrive::escape_q;

    #[test]
    fn q_escaping() {
        assert_eq!(escape_q("plain"), "plain");
        assert_eq!(escape_q("O'Brien"), "O\\'Brien");
        assert_eq!(escape_q("a\\b"), "a\\\\b");
    }

    #[test]
    fn parses_drive_files() {
        let files = serde_json::json!([
            { "id": "f1", "name": "Photos", "mimeType": "application/vnd.google-apps.folder" },
            { "id": "f2", "name": "report.pdf", "mimeType": "application/pdf",
              "size": "2048", "modifiedTime": "2024-07-15T09:30:00Z", "md5Checksum": "abc123" }
        ]);
        let arr = files.as_array().unwrap();
        let parsed: Vec<_> = arr
            .iter()
            .filter_map(|f| entry_from_json(f, "/docs"))
            .collect();
        assert_eq!(parsed.len(), 2);

        let (dir, id, is_dir) = parsed.iter().find(|(e, _, _)| e.name == "Photos").unwrap();
        assert!(is_dir);
        assert_eq!(id, "f1");
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/docs/Photos");

        let (file, _, _) = parsed
            .iter()
            .find(|(e, _, _)| e.name == "report.pdf")
            .unwrap();
        assert_eq!(file.kind, FileKind::File);
        assert_eq!(file.size, 2048);
        assert_eq!(file.etag.as_deref(), Some("abc123"));
        assert_eq!(file.modified, Some(1_721_035_800));
    }

    /// End-to-end against the Drive mock (`tests/gdrive_mock.py`). Skipped unless
    /// GHOSTFTP_GDRIVE_MOCK_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored gdrive_roundtrip` after starting it.
    /// Exercises the ID resolver on a nested path plus token exchange, the
    /// 401→refresh retry, and list/create/upload/download/move/delete.
    #[tokio::test]
    #[ignore = "requires the Google Drive mock (GHOSTFTP_GDRIVE_MOCK_URL)"]
    async fn live_gdrive_roundtrip() {
        use crate::oauth::{self, TokenSet};
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::gdrive::{gdrive_config, gdrive_connect, GDRIVE_SERVICE};
        use std::time::{SystemTime, UNIX_EPOCH};

        let Ok(mock) = std::env::var("GHOSTFTP_GDRIVE_MOCK_URL") else {
            eprintln!("skip: GHOSTFTP_GDRIVE_MOCK_URL unset");
            return;
        };
        std::env::set_var("GHOSTFTP_GDRIVE_CLIENT_ID", "test-client");
        std::env::set_var("GHOSTFTP_GDRIVE_TOKEN_URL", format!("{mock}/token"));
        std::env::set_var("GHOSTFTP_GDRIVE_API_BASE", &mock);
        std::env::set_var("GHOSTFTP_GDRIVE_UPLOAD_BASE", &mock);

        let ex = oauth::exchange_code(&gdrive_config(), "authcode", "verifier")
            .await
            .expect("exchange");
        assert_eq!(ex["access_token"], "ACCESS1");

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let pid = "gdrive-mock-test";
        oauth::store_tokens(
            GDRIVE_SERVICE,
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
            name: "gd".into(),
            protocol: "gdrive".into(),
            host: "drive.google.com".into(),
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
        let session = Arc::new(gdrive_connect(&profile).await.expect("connect"));

        // First call uses STALE → 401 → refresh → succeeds.
        assert_eq!(
            session.account_label().await.expect("account"),
            "tester@example.invalid"
        );

        let fs = GDriveFs::new(session.clone());
        // Nested folders exercise the resolver descending two levels.
        fs.create_dir("/ghostftp-test").await.expect("mkdir1");
        fs.create_dir("/ghostftp-test/sub").await.expect("mkdir2");

        // Upload into the nested folder (multipart create, same as the transfer path).
        let parent = session
            .folder_id("/ghostftp-test/sub")
            .await
            .expect("resolve parent");
        let token = session.access_token().await.unwrap();
        let boundary = "ghostftpTESTboundary";
        let metaj = serde_json::json!({ "name": "hello.txt", "parents": [parent] });
        let mut body: Vec<u8> = Vec::new();
        body.extend_from_slice(format!("--{boundary}\r\n").as_bytes());
        body.extend_from_slice(b"Content-Type: application/json; charset=UTF-8\r\n\r\n");
        body.extend_from_slice(metaj.to_string().as_bytes());
        body.extend_from_slice(format!("\r\n--{boundary}\r\n").as_bytes());
        body.extend_from_slice(b"Content-Type: application/octet-stream\r\n\r\n");
        body.extend_from_slice(b"hi drive");
        body.extend_from_slice(format!("\r\n--{boundary}--\r\n").as_bytes());
        let put = session
            .client
            .post(format!(
                "{}/files?uploadType=multipart",
                session.upload_base
            ))
            .bearer_auth(&token)
            .header(
                reqwest::header::CONTENT_TYPE,
                format!("multipart/related; boundary={boundary}"),
            )
            .body(body)
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
        assert_eq!(hello.size, 8);
        assert!(hello.etag.is_some(), "md5 change token");

        // Download via resolver → id → alt=media.
        let (fid, _) = session
            .resolve_item("/ghostftp-test/sub/hello.txt")
            .await
            .expect("resolve")
            .expect("present");
        let dl = session
            .get_stream(&format!("/files/{fid}?alt=media"))
            .await
            .expect("dl")
            .text()
            .await
            .expect("body");
        assert_eq!(dl, "hi drive");

        // Move across folders (addParents/removeParents).
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

        oauth::delete_tokens(GDRIVE_SERVICE, pid);
        eprintln!("live_gdrive_roundtrip: OK");
    }
}
