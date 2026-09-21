use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::session::webdav::PROPFIND_BODY;
use crate::session::WebdavSession;
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use percent_encoding::percent_decode_str;
use quick_xml::events::Event;
use quick_xml::Reader;
use reqwest::Method;
use std::sync::Arc;

/// RemoteFs over WebDAV (Nextcloud, ownCloud, Hetzner Storage Box, and the
/// generic long tail). Real directories, rename via `MOVE`, no chmod. Change
/// detection leans on the per-resource ETag (`getetag`); servers that omit it
/// degrade to the mtime+size the entries still carry.
pub struct WebdavFs {
    session: Arc<WebdavSession>,
}

impl WebdavFs {
    pub fn new(session: Arc<WebdavSession>) -> Self {
        Self { session }
    }

    fn propfind_method() -> Method {
        Method::from_bytes(b"PROPFIND").expect("PROPFIND is a valid method token")
    }
}

#[async_trait]
impl RemoteFs for WebdavFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        let url = self.session.url_for(path, true);
        let resp = self
            .session
            .request(Self::propfind_method(), url)
            .header("Depth", "1")
            .header("Content-Type", "application/xml")
            .body(PROPFIND_BODY)
            .send()
            .await
            .with_context(|| format!("PROPFIND {path}"))?;

        let status = resp.status();
        if status == reqwest::StatusCode::NOT_FOUND {
            return Err(anyhow!("{path}: not found"));
        }
        if !(status.is_success() || status.as_u16() == 207) {
            return Err(anyhow!("PROPFIND {path} failed: HTTP {}", status.as_u16()));
        }

        let body = resp.text().await.context("read PROPFIND body")?;
        Ok(parse_multistatus(&body, &self.session.base_path(), path))
    }

    async fn rename(&self, from: &str, to: &str) -> Result<()> {
        let src = self.session.url_for(from, false);
        let dst = self.session.url_for(to, false);
        let resp = self
            .session
            .request(Method::from_bytes(b"MOVE").unwrap(), src)
            .header("Destination", dst.as_str())
            // Don't silently clobber an existing target — mirrors SFTP/local rename.
            .header("Overwrite", "F")
            .send()
            .await
            .with_context(|| format!("MOVE {from} -> {to}"))?;
        if !resp.status().is_success() {
            return Err(anyhow!(
                "rename {from} -> {to} failed: HTTP {}",
                resp.status().as_u16()
            ));
        }
        Ok(())
    }

    async fn delete(&self, path: &str, _recursive: bool) -> Result<()> {
        // WebDAV DELETE on a collection removes the whole tree by spec, so the
        // `recursive` flag needs no separate handling here.
        let url = self.session.url_for(path, false);
        let resp = self
            .session
            .request(Method::DELETE, url)
            .send()
            .await
            .with_context(|| format!("DELETE {path}"))?;
        let status = resp.status();
        if !(status.is_success() || status == reqwest::StatusCode::NOT_FOUND) {
            return Err(anyhow!("delete {path} failed: HTTP {}", status.as_u16()));
        }
        Ok(())
    }

    async fn create_dir(&self, path: &str) -> Result<()> {
        let url = self.session.url_for(path, true);
        let resp = self
            .session
            .request(Method::from_bytes(b"MKCOL").unwrap(), url)
            .send()
            .await
            .with_context(|| format!("MKCOL {path}"))?;
        if !resp.status().is_success() {
            return Err(anyhow!(
                "create dir {path} failed: HTTP {}",
                resp.status().as_u16()
            ));
        }
        Ok(())
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(anyhow!("WebDAV has no POSIX permissions"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: true,
            has_directories: true,
            has_shell: false,
            // ETag is the reliable change token; entries also carry mtime+size
            // for servers that omit getetag.
            change_signal: ChangeSignal::Etag,
        }
    }
}

/// Which prop's text we're currently accumulating inside a `<response>`.
#[derive(PartialEq)]
enum Field {
    Href,
    ContentLength,
    LastModified,
    Etag,
}

/// Parse a WebDAV `multistatus` (207) body into directory entries, resolving each
/// `href` against the DAV root `base_path` and dropping the self-entry for
/// `request_path`. Namespace-prefix agnostic (matches on local element names) so
/// it handles Nextcloud (`d:`), Apache mod_dav (`D:`/`lp1:`), and bare servers.
fn parse_multistatus(xml: &str, base_path: &str, request_path: &str) -> Vec<DirEntry> {
    let request_norm = format!("/{}", request_path.trim_matches('/'));
    let request_norm = if request_norm == "/" {
        "/".to_string()
    } else {
        request_norm
    };

    let mut reader = Reader::from_str(xml);
    let mut out = Vec::new();

    // Per-response accumulators.
    let mut href: Option<String> = None;
    let mut is_collection = false;
    let mut size: u64 = 0;
    let mut etag: Option<String> = None;
    let mut modified: Option<i64> = None;

    let mut in_resourcetype = false;
    let mut capture: Option<Field> = None;
    let mut buf = String::new();

    loop {
        match reader.read_event() {
            Ok(Event::Start(e)) => {
                let local = e.local_name();
                match local.as_ref() {
                    b"response" => {
                        href = None;
                        is_collection = false;
                        size = 0;
                        etag = None;
                        modified = None;
                    }
                    b"href" => {
                        capture = Some(Field::Href);
                        buf.clear();
                    }
                    b"resourcetype" => in_resourcetype = true,
                    b"collection" if in_resourcetype => is_collection = true,
                    b"getcontentlength" => {
                        capture = Some(Field::ContentLength);
                        buf.clear();
                    }
                    b"getlastmodified" => {
                        capture = Some(Field::LastModified);
                        buf.clear();
                    }
                    b"getetag" => {
                        capture = Some(Field::Etag);
                        buf.clear();
                    }
                    _ => {}
                }
            }
            Ok(Event::Empty(e)) => {
                let local = e.local_name();
                if local.as_ref() == b"collection" && in_resourcetype {
                    is_collection = true;
                }
            }
            Ok(Event::Text(e)) => {
                if capture.is_some() {
                    if let Ok(t) = e.unescape() {
                        buf.push_str(&t);
                    }
                }
            }
            Ok(Event::CData(e)) => {
                if capture.is_some() {
                    buf.push_str(&String::from_utf8_lossy(&e));
                }
            }
            Ok(Event::End(e)) => {
                let local = e.local_name();
                match local.as_ref() {
                    b"href" => {
                        href = Some(buf.trim().to_string());
                        capture = None;
                    }
                    b"getcontentlength" => {
                        // Only overwrite on a real value: a 404 propstat block
                        // re-declares the prop empty and must not clobber the 200.
                        if let Ok(n) = buf.trim().parse::<u64>() {
                            size = n;
                        }
                        capture = None;
                    }
                    b"getlastmodified" => {
                        if let Some(ts) = parse_http_date(buf.trim()) {
                            modified = Some(ts);
                        }
                        capture = None;
                    }
                    b"getetag" => {
                        let clean = clean_etag(buf.trim());
                        if !clean.is_empty() {
                            etag = Some(clean);
                        }
                        capture = None;
                    }
                    b"resourcetype" => in_resourcetype = false,
                    b"response" => {
                        if let Some(h) = href.take() {
                            if let Some((entry_path, name)) = href_to_path(&h, base_path) {
                                if entry_path != request_norm {
                                    out.push(DirEntry {
                                        name,
                                        path: entry_path,
                                        kind: if is_collection {
                                            FileKind::Directory
                                        } else {
                                            FileKind::File
                                        },
                                        size,
                                        modified,
                                        mode: None,
                                        etag: etag.take(),
                                    });
                                }
                            }
                        }
                    }
                    _ => {}
                }
            }
            Ok(Event::Eof) => break,
            Err(_) => break,
            _ => {}
        }
    }

    out
}

/// Turn a PROPFIND `href` into a Ghost FTP path (`/dir/file`) + display name relative
/// to the DAV root. Returns `None` for hrefs that aren't under the base (skipped
/// rather than mis-rendered).
fn href_to_path(href: &str, base_path: &str) -> Option<(String, String)> {
    // href may be a full URL or a server-absolute path.
    let path_part = if let Some(idx) = href.find("://") {
        let rest = &href[idx + 3..];
        match rest.find('/') {
            Some(slash) => &rest[slash..],
            None => "/",
        }
    } else {
        href
    };

    let decoded_href = percent_decode_str(path_part)
        .decode_utf8_lossy()
        .into_owned();
    let decoded_base = percent_decode_str(base_path)
        .decode_utf8_lossy()
        .into_owned();

    let hp = decoded_href.trim_end_matches('/');
    let bp = decoded_base.trim_end_matches('/');

    let rel = if hp == bp {
        "" // the collection itself
    } else {
        hp.strip_prefix(bp)?.trim_start_matches('/')
    };

    let entry_path = if rel.is_empty() {
        "/".to_string()
    } else {
        format!("/{rel}")
    };
    let name = rel.rsplit('/').next().unwrap_or(rel).to_string();
    Some((entry_path, name))
}

/// Strip the surrounding quotes and any weak-validator `W/` prefix off an ETag.
fn clean_etag(raw: &str) -> String {
    raw.trim()
        .trim_start_matches("W/")
        .trim_matches('"')
        .to_string()
}

/// Parse an RFC 1123 HTTP-date (always GMT for `getlastmodified`) into a Unix
/// timestamp. Deliberately dependency-free — the format is fixed. Shared with the
/// HTTP autoindex backend (`Last-Modified` headers, nginx-JSON `mtime`).
pub(crate) fn parse_http_date(s: &str) -> Option<i64> {
    // "Mon, 15 Jul 2024 12:34:56 GMT"
    let parts: Vec<&str> = s.split_whitespace().collect();
    if parts.len() < 5 {
        return None;
    }
    let day: i64 = parts[1].parse().ok()?;
    let month = month_num(parts[2])?;
    let year: i64 = parts[3].parse().ok()?;
    let time: Vec<&str> = parts[4].split(':').collect();
    if time.len() != 3 {
        return None;
    }
    let hh: i64 = time[0].parse().ok()?;
    let mm: i64 = time[1].parse().ok()?;
    let ss: i64 = time[2].parse().ok()?;
    Some(days_from_civil(year, month, day) * 86400 + hh * 3600 + mm * 60 + ss)
}

fn month_num(m: &str) -> Option<i64> {
    Some(match m {
        "Jan" => 1,
        "Feb" => 2,
        "Mar" => 3,
        "Apr" => 4,
        "May" => 5,
        "Jun" => 6,
        "Jul" => 7,
        "Aug" => 8,
        "Sep" => 9,
        "Oct" => 10,
        "Nov" => 11,
        "Dec" => 12,
        _ => return None,
    })
}

/// Days since the Unix epoch for a civil (proleptic Gregorian) date. Howard
/// Hinnant's `days_from_civil` algorithm. Shared with the Dropbox backend's
/// ISO-8601 timestamp parser.
pub(crate) fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = (if y >= 0 { y } else { y - 399 }) / 400;
    let yoe = y - era * 400; // [0, 399]
    let doy = (153 * (if m > 2 { m - 3 } else { m + 9 }) + 2) / 5 + d - 1; // [0, 365]
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy; // [0, 146096]
    era * 146097 + doe - 719468
}

#[cfg(test)]
mod tests {
    use super::*;

    // A Nextcloud-style multistatus: `d:` prefix, full-path hrefs, quoted etags,
    // one collection (the requested dir, dropped as self) + a subdir + a file.
    const NEXTCLOUD: &str = r#"<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/remote.php/dav/files/alice/docs/</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype><d:collection/></d:resourcetype>
        <d:getlastmodified>Mon, 15 Jul 2024 12:34:56 GMT</d:getlastmodified>
        <d:getetag>"abc123"</d:getetag>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/remote.php/dav/files/alice/docs/sub/</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype><d:collection/></d:resourcetype>
        <d:getetag>"dir-etag"</d:getetag>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/remote.php/dav/files/alice/docs/report%20final.pdf</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype/>
        <d:getcontentlength>2048</d:getcontentlength>
        <d:getlastmodified>Tue, 16 Jul 2024 09:00:00 GMT</d:getlastmodified>
        <d:getetag>W/"weak-etag"</d:getetag>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
    <d:propstat>
      <d:prop><d:getcontentlength/></d:prop>
      <d:status>HTTP/1.1 404 Not Found</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>"#;

    #[test]
    fn parses_nextcloud_listing() {
        let entries = parse_multistatus(NEXTCLOUD, "/remote.php/dav/files/alice/", "/docs");
        // Self-entry (docs/) dropped; a subdir + a file remain.
        assert_eq!(entries.len(), 2);

        let sub = entries.iter().find(|e| e.name == "sub").expect("sub dir");
        assert_eq!(sub.kind, FileKind::Directory);
        assert_eq!(sub.path, "/docs/sub");
        assert_eq!(sub.etag.as_deref(), Some("dir-etag"));

        // Percent-encoded name is decoded; 404-block empty length doesn't clobber.
        let file = entries
            .iter()
            .find(|e| e.name == "report final.pdf")
            .expect("pdf file");
        assert_eq!(file.kind, FileKind::File);
        assert_eq!(file.path, "/docs/report final.pdf");
        assert_eq!(file.size, 2048);
        assert_eq!(file.etag.as_deref(), Some("weak-etag"));
        assert!(file.modified.is_some());
    }

    // Apache mod_dav style: `D:` prefix, path hrefs, no etags, sizes present.
    const APACHE: &str = r#"<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response>
<D:href>/share/</D:href>
<D:propstat><D:prop><D:resourcetype><D:collection/></D:resourcetype></D:prop>
<D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>
<D:response>
<D:href>/share/notes.txt</D:href>
<D:propstat><D:prop>
<D:resourcetype/>
<D:getcontentlength>17</D:getcontentlength>
<D:getlastmodified>Wed, 01 Jan 2020 00:00:00 GMT</D:getlastmodified>
</D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>
</D:multistatus>"#;

    #[test]
    fn parses_apache_listing_without_etags() {
        let entries = parse_multistatus(APACHE, "/share/", "/");
        assert_eq!(entries.len(), 1);
        let f = &entries[0];
        assert_eq!(f.name, "notes.txt");
        assert_eq!(f.path, "/notes.txt");
        assert_eq!(f.size, 17);
        assert!(f.etag.is_none());
        assert_eq!(f.modified, Some(1_577_836_800)); // 2020-01-01T00:00:00Z
    }

    #[test]
    fn full_url_hrefs_resolve() {
        let xml = r#"<multistatus xmlns="DAV:">
<response><href>https://dav.example.invalid/dav/photo.jpg</href>
<propstat><prop><resourcetype/><getcontentlength>9</getcontentlength></prop>
<status>HTTP/1.1 200 OK</status></propstat></response></multistatus>"#;
        let entries = parse_multistatus(xml, "/dav/", "/");
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].name, "photo.jpg");
        assert_eq!(entries[0].path, "/photo.jpg");
    }

    #[test]
    fn http_date_epoch() {
        assert_eq!(parse_http_date("Thu, 01 Jan 1970 00:00:00 GMT"), Some(0));
        assert_eq!(
            parse_http_date("Mon, 15 Jul 2024 12:34:56 GMT"),
            Some(1_721_046_896)
        );
        assert_eq!(parse_http_date("garbage"), None);
    }

    #[test]
    fn etag_cleaning() {
        assert_eq!(clean_etag(r#""abc""#), "abc");
        assert_eq!(clean_etag(r#"W/"weak""#), "weak");
        assert_eq!(clean_etag(""), "");
    }

    /// End-to-end against a real WebDAV server. Skipped unless GHOSTFTP_WEBDAV_URL is
    /// set; run with `cargo test -p ghostftp -- --ignored webdav` after starting one
    /// (e.g. wsgidav). Exercises connect → mkcol → put → list → get → move →
    /// delete through Ghost FTP's own WebdavFs/WebdavSession, not just the parser.
    #[tokio::test]
    #[ignore = "requires a live WebDAV server (GHOSTFTP_WEBDAV_URL/USER/PASS)"]
    async fn live_webdav_roundtrip() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::webdav::webdav_connect;
        use reqwest::Method;

        let Ok(url) = std::env::var("GHOSTFTP_WEBDAV_URL") else {
            eprintln!("skip: GHOSTFTP_WEBDAV_URL unset");
            return;
        };
        let user = std::env::var("GHOSTFTP_WEBDAV_USER").unwrap_or_default();
        let pass = std::env::var("GHOSTFTP_WEBDAV_PASS").unwrap_or_default();

        let profile = ConnectionProfile {
            icon: None,
            id: "test".into(),
            name: "webdav-test".into(),
            protocol: "webdav".into(),
            host: "127.0.0.1".into(),
            port: 443,
            username: user,
            auth: AuthMethod::Password { password: pass },
            default_remote_path: None,
            description: None,
            color: None,
            auto_connect: None,
            bucket: None,
            region: None,
            endpoint: Some(url),
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

        let sess = Arc::new(webdav_connect(&profile).await.expect("connect"));
        let fs = WebdavFs::new(sess.clone());

        // Clean slate.
        let _ = fs.delete("/ghostftp-test", true).await;
        fs.create_dir("/ghostftp-test").await.expect("mkcol");

        // PUT a file through the session (same path the transfer manager uses).
        let put = sess
            .request(Method::PUT, sess.url_for("/ghostftp-test/hello.txt", false))
            .body("hi there")
            .send()
            .await
            .expect("put send");
        assert!(put.status().is_success(), "PUT status {}", put.status());

        // LIST sees it, with the right kind + size.
        let entries = fs.list_dir("/ghostftp-test").await.expect("list");
        let hello = entries
            .iter()
            .find(|e| e.name == "hello.txt")
            .expect("hello.txt listed");
        assert_eq!(hello.kind, FileKind::File);
        assert_eq!(hello.size, 8);
        assert_eq!(hello.path, "/ghostftp-test/hello.txt");

        // GET round-trips the bytes.
        let got = sess
            .request(Method::GET, sess.url_for("/ghostftp-test/hello.txt", false))
            .send()
            .await
            .expect("get send")
            .text()
            .await
            .expect("get body");
        assert_eq!(got, "hi there");

        // MOVE (rename) then confirm the swap.
        fs.rename("/ghostftp-test/hello.txt", "/ghostftp-test/renamed.txt")
            .await
            .expect("move");
        let entries = fs
            .list_dir("/ghostftp-test")
            .await
            .expect("list after move");
        assert!(entries.iter().any(|e| e.name == "renamed.txt"));
        assert!(!entries.iter().any(|e| e.name == "hello.txt"));

        // DELETE the file, then the collection.
        fs.delete("/ghostftp-test/renamed.txt", false)
            .await
            .expect("delete file");
        fs.delete("/ghostftp-test", true).await.expect("delete dir");
        let root = fs.list_dir("/").await.expect("list root");
        assert!(!root.iter().any(|e| e.name == "ghostftp-test"));

        eprintln!("live_webdav_roundtrip: OK");
    }
}
