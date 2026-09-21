use super::{Capabilities, ChangeSignal, DirEntry, FileKind, RemoteFs};
use crate::remotefs::webdav::parse_http_date;
use crate::session::http::HttpMode;
use crate::session::HttpSession;
use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use percent_encoding::percent_decode_str;
use reqwest::header::{CONTENT_LENGTH, CONTENT_TYPE, ETAG, LAST_MODIFIED};
use reqwest::Method;
use std::collections::HashSet;
use std::sync::Arc;
use url::Url;

/// Read-only RemoteFs over a static HTTP(S) server. Lists via autoindex (nginx /
/// Apache HTML, or nginx JSON) and reads via `GET`; every mutation errors. In
/// single-file mode it surfaces one entry for a pasted direct URL. Change signal
/// is the file's `ETag`/`Last-Modified` (populated on read/HEAD, not in a listing).
pub struct HttpFs {
    session: Arc<HttpSession>,
}

impl HttpFs {
    pub fn new(session: Arc<HttpSession>) -> Self {
        Self { session }
    }

    /// HEAD a file URL for its size / last-modified / etag. Best-effort — a
    /// server that rejects HEAD just yields zeros.
    async fn head_meta(&self, url: &Url) -> (u64, Option<i64>, Option<String>) {
        match self.session.request(Method::HEAD, url.clone()).send().await {
            Ok(resp) if resp.status().is_success() => {
                let h = resp.headers();
                let size = h
                    .get(CONTENT_LENGTH)
                    .and_then(|v| v.to_str().ok())
                    .and_then(|s| s.parse::<u64>().ok())
                    .unwrap_or(0);
                let modified = h
                    .get(LAST_MODIFIED)
                    .and_then(|v| v.to_str().ok())
                    .and_then(parse_http_date);
                let etag = h
                    .get(ETAG)
                    .and_then(|v| v.to_str().ok())
                    .map(clean_etag)
                    .filter(|s| !s.is_empty());
                (size, modified, etag)
            }
            _ => (0, None, None),
        }
    }

    fn read_only(op: &str) -> anyhow::Error {
        anyhow!("HTTP source is read-only ({op} not supported)")
    }
}

#[async_trait]
impl RemoteFs for HttpFs {
    async fn list_dir(&self, path: &str) -> Result<Vec<DirEntry>> {
        if let HttpMode::DirectFile { name } = &self.session.mode {
            // Single-file connection: one entry, HEAD for its size.
            let url = self.session.url_for(path, false);
            let (size, modified, etag) = self.head_meta(&url).await;
            return Ok(vec![DirEntry {
                name: name.clone(),
                path: format!("/{name}"),
                kind: FileKind::File,
                size,
                modified,
                mode: None,
                etag,
            }]);
        }

        let url = self.session.url_for(path, true);
        let resp = self
            .session
            .request(Method::GET, url)
            .send()
            .await
            .with_context(|| format!("GET {path}"))?;
        let status = resp.status();
        if status == reqwest::StatusCode::NOT_FOUND {
            return Err(anyhow!("{path}: not found"));
        }
        if !status.is_success() {
            return Err(anyhow!("list {path} failed: HTTP {}", status.as_u16()));
        }
        let is_json = resp
            .headers()
            .get(CONTENT_TYPE)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_ascii_lowercase().contains("json"))
            .unwrap_or(false);
        let body = resp.text().await.context("read listing body")?;

        let entries = if is_json || body.trim_start().starts_with('[') {
            parse_nginx_json(&body, path)
        } else {
            parse_autoindex_html(&body, path)
        };
        Ok(entries)
    }

    async fn rename(&self, _from: &str, _to: &str) -> Result<()> {
        Err(Self::read_only("rename"))
    }

    async fn delete(&self, _path: &str, _recursive: bool) -> Result<()> {
        Err(Self::read_only("delete"))
    }

    async fn create_dir(&self, _path: &str) -> Result<()> {
        Err(Self::read_only("create dir"))
    }

    async fn chmod(&self, _path: &str, _mode: u32) -> Result<()> {
        Err(Self::read_only("chmod"))
    }

    fn capabilities(&self) -> Capabilities {
        Capabilities {
            can_chmod: false,
            can_symlink: false,
            can_rename: false,
            // Autoindex exposes real subdirectories; direct-file mode just has one.
            has_directories: true,
            has_shell: false,
            change_signal: ChangeSignal::Etag,
        }
    }
}

/// Build a child's Ghost FTP path from the directory being listed + a leaf name.
fn child_path(request_path: &str, name: &str) -> String {
    let base = request_path.trim_matches('/');
    if base.is_empty() {
        format!("/{name}")
    } else {
        format!("/{base}/{name}")
    }
}

/// Strip the surrounding quotes and any weak-validator `W/` prefix off an ETag.
fn clean_etag(raw: &str) -> String {
    raw.trim()
        .trim_start_matches("W/")
        .trim_matches('"')
        .to_string()
}

/// Parse an nginx/Apache autoindex HTML page into entries. Deliberately simple
/// and anchor-based (one entry per line, as both servers emit) so it survives the
/// `<pre>` and `<table>` variants; sizes are best-effort (real size comes from
/// Content-Length on download).
fn parse_autoindex_html(body: &str, request_path: &str) -> Vec<DirEntry> {
    let mut out = Vec::new();
    let mut seen = HashSet::new();
    for line in body.lines() {
        if let Some((href, tail)) = first_href(line) {
            if let Some(entry) = entry_from_href(&href, tail, request_path) {
                if seen.insert(entry.path.clone()) {
                    out.push(entry);
                }
            }
        }
    }
    out
}

/// Find the first quoted `href="…"` on a line, returning the href and the text
/// after the anchor's `</a>` (where nginx/Apache put the date + size columns).
fn first_href(line: &str) -> Option<(String, &str)> {
    let idx = line.find("href=")?;
    let after = &line[idx + 5..];
    let quote = match after.as_bytes().first()? {
        b'"' => '"',
        b'\'' => '\'',
        _ => return None, // unquoted href — skip
    };
    let rest = &after[1..];
    let end = rest.find(quote)?;
    let href = rest[..end].to_string();
    let after_href = &rest[end + 1..];
    let tail = match after_href.find("</a>") {
        Some(p) => &after_href[p + 4..],
        None => after_href,
    };
    Some((href, tail))
}

fn entry_from_href(href: &str, tail: &str, request_path: &str) -> Option<DirEntry> {
    // Drop query/fragment (Apache column-sort links become empty → skipped).
    let href = href.split(['?', '#']).next().unwrap_or(href);
    if href.is_empty() || href.contains("://") || href.starts_with('/') {
        return None; // external, absolute, or nav link
    }
    let is_dir = href.ends_with('/');
    let trimmed = href.trim_end_matches('/');
    let seg = trimmed.rsplit('/').next().unwrap_or(trimmed);
    if seg.is_empty() || seg == "." || seg == ".." {
        return None;
    }
    let name = percent_decode_str(seg).decode_utf8_lossy().into_owned();
    if name.eq_ignore_ascii_case("parent directory") {
        return None;
    }
    let size = if is_dir { 0 } else { size_from_tail(tail) };
    Some(DirEntry {
        name: name.clone(),
        path: child_path(request_path, &name),
        kind: if is_dir {
            FileKind::Directory
        } else {
            FileKind::File
        },
        size,
        modified: None,
        mode: None,
        etag: None,
    })
}

/// Best-effort size from the trailing column(s) of an autoindex row: the last
/// token that parses as a byte count (plain integer, or Apache's `1.2K`/`3M`).
fn size_from_tail(tail: &str) -> u64 {
    let text = strip_tags(tail);
    for tok in text.split_whitespace().rev() {
        if tok == "-" {
            return 0;
        }
        if let Some(n) = parse_size_token(tok) {
            return n;
        }
    }
    0
}

fn parse_size_token(tok: &str) -> Option<u64> {
    if let Ok(n) = tok.parse::<u64>() {
        return Some(n);
    }
    parse_human_size(tok)
}

/// Parse Apache's human-readable sizes (`1.2K`, `3.4M`, `12G`) into bytes.
fn parse_human_size(tok: &str) -> Option<u64> {
    let t = tok.trim();
    let last = t.chars().last()?;
    let mult: f64 = match last.to_ascii_uppercase() {
        'K' => 1024.0,
        'M' => 1024.0 * 1024.0,
        'G' => 1024.0 * 1024.0 * 1024.0,
        'T' => 1024.0f64.powi(4),
        _ => return None,
    };
    let num: f64 = t[..t.len() - 1].trim().parse().ok()?;
    Some((num * mult) as u64)
}

fn strip_tags(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    let mut in_tag = false;
    for c in s.chars() {
        match c {
            // Emit a space at each tag boundary so adjacent table cells
            // (`…09:30</td><td>1.2K…`) don't fuse into one token.
            '<' => {
                in_tag = true;
                out.push(' ');
            }
            '>' => in_tag = false,
            _ if !in_tag => out.push(c),
            _ => {}
        }
    }
    out
}

#[derive(serde::Deserialize)]
struct JsonItem {
    name: String,
    #[serde(rename = "type")]
    typ: Option<String>,
    size: Option<u64>,
    mtime: Option<String>,
}

/// Parse nginx's `autoindex_format json;` listing.
fn parse_nginx_json(body: &str, request_path: &str) -> Vec<DirEntry> {
    let items: Vec<JsonItem> = serde_json::from_str(body).unwrap_or_default();
    items
        .into_iter()
        .filter_map(|it| {
            if it.name.is_empty() || it.name == "." || it.name == ".." {
                return None;
            }
            let is_dir = it.typ.as_deref() == Some("directory");
            Some(DirEntry {
                path: child_path(request_path, &it.name),
                name: it.name,
                kind: if is_dir {
                    FileKind::Directory
                } else {
                    FileKind::File
                },
                size: it.size.unwrap_or(0),
                modified: it.mtime.as_deref().and_then(parse_http_date),
                mode: None,
                etag: None,
            })
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    const NGINX_HTML: &str = r#"<html>
<head><title>Index of /pub/</title></head>
<body bgcolor="white">
<h1>Index of /pub/</h1><hr><pre><a href="../">../</a>
<a href="images/">images/</a>                                        14-Jul-2024 10:00       -
<a href="notes.txt">notes.txt</a>                                    15-Jul-2024 09:30      42
<a href="archive%20v2.zip">archive v2.zip</a>                        16-Jul-2024 12:00  1048576
</pre><hr></body>
</html>"#;

    #[test]
    fn parses_nginx_autoindex() {
        let e = parse_autoindex_html(NGINX_HTML, "/pub");
        assert_eq!(e.len(), 3, "got {e:?}");

        let dir = e.iter().find(|x| x.name == "images").unwrap();
        assert_eq!(dir.kind, FileKind::Directory);
        assert_eq!(dir.path, "/pub/images");

        let notes = e.iter().find(|x| x.name == "notes.txt").unwrap();
        assert_eq!(notes.kind, FileKind::File);
        assert_eq!(notes.size, 42);

        let zip = e.iter().find(|x| x.name == "archive v2.zip").unwrap();
        assert_eq!(zip.size, 1_048_576);
        assert_eq!(zip.path, "/pub/archive v2.zip");
    }

    const APACHE_HTML: &str = r#"<html><body>
<h1>Index of /files</h1>
<table>
<tr><th><a href="?C=N;O=D">Name</a></th><th>Last modified</th><th>Size</th></tr>
<tr><td><a href="/files/">Parent Directory</a></td><td>&nbsp;</td><td>-</td></tr>
<tr><td><a href="docs/">docs/</a></td><td>2024-07-14 10:00</td><td>-</td></tr>
<tr><td><a href="readme.md">readme.md</a></td><td>2024-07-15 09:30</td><td>1.2K</td></tr>
<tr><td><a href="big.iso">big.iso</a></td><td>2024-07-16 12:00</td><td>2.0G</td></tr>
</table>
</body></html>"#;

    #[test]
    fn parses_apache_autoindex() {
        let e = parse_autoindex_html(APACHE_HTML, "/files");
        // Sort link + Parent Directory dropped; docs + readme.md + big.iso remain.
        assert_eq!(e.len(), 3, "got {e:?}");
        assert!(e
            .iter()
            .any(|x| x.name == "docs" && x.kind == FileKind::Directory));
        let readme = e.iter().find(|x| x.name == "readme.md").unwrap();
        assert_eq!(readme.size, (1.2 * 1024.0) as u64);
        let iso = e.iter().find(|x| x.name == "big.iso").unwrap();
        assert_eq!(iso.size, 2 * 1024 * 1024 * 1024);
    }

    const NGINX_JSON: &str = r#"[
{ "name":"images", "type":"directory", "mtime":"Sun, 14 Jul 2024 10:00:00 GMT" },
{ "name":"notes.txt", "type":"file", "mtime":"Mon, 15 Jul 2024 09:30:00 GMT", "size":42 },
{ "name":".", "type":"directory" }
]"#;

    #[test]
    fn parses_nginx_json() {
        let e = parse_nginx_json(NGINX_JSON, "/pub");
        assert_eq!(e.len(), 2); // "." dropped
        let dir = e.iter().find(|x| x.name == "images").unwrap();
        assert_eq!(dir.kind, FileKind::Directory);
        assert!(dir.modified.is_some());
        let notes = e.iter().find(|x| x.name == "notes.txt").unwrap();
        assert_eq!(notes.size, 42);
        assert_eq!(notes.path, "/pub/notes.txt");
    }

    #[test]
    fn human_sizes() {
        assert_eq!(parse_human_size("1.2K"), Some(1228));
        assert_eq!(parse_human_size("3M"), Some(3 * 1024 * 1024));
        assert_eq!(parse_size_token("512"), Some(512));
        assert_eq!(parse_size_token("12:00"), None);
    }

    /// End-to-end against a real static file server (e.g. `python -m http.server`
    /// autoindex). Skipped unless GHOSTFTP_HTTP_URL is set; run with
    /// `cargo test -p ghostftp -- --ignored http_source`.
    #[tokio::test]
    #[ignore = "requires a live static HTTP server (GHOSTFTP_HTTP_URL)"]
    async fn live_http_source() {
        use crate::profiles::{AuthMethod, ConnectionProfile};
        use crate::session::http::http_connect;

        let Ok(url) = std::env::var("GHOSTFTP_HTTP_URL") else {
            eprintln!("skip: GHOSTFTP_HTTP_URL unset");
            return;
        };
        let profile = ConnectionProfile {
            icon: None,
            id: "t".into(),
            name: "http".into(),
            protocol: "http".into(),
            host: "127.0.0.1".into(),
            port: 443,
            username: String::new(),
            auth: AuthMethod::Password {
                password: String::new(),
            },
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
        let sess = Arc::new(http_connect(&profile).await.expect("connect"));
        let fs = HttpFs::new(sess.clone());
        let entries = fs.list_dir("/").await.expect("list root");
        eprintln!("live_http_source: {} entries", entries.len());
        assert!(!entries.is_empty());

        // Mutations must be refused.
        assert!(fs.delete("/anything", false).await.is_err());
        assert!(fs.create_dir("/nope").await.is_err());

        // GET a listed file back through the session (the transfer read path).
        if let Some(file) = entries.iter().find(|e| e.kind == FileKind::File) {
            let body = sess
                .request(Method::GET, sess.url_for(&file.path, false))
                .send()
                .await
                .expect("get send")
                .text()
                .await
                .expect("get body");
            eprintln!("fetched {} ({} bytes)", file.name, body.len());
            assert!(!body.is_empty());
        }
        eprintln!("live_http_source: OK");
    }
}
