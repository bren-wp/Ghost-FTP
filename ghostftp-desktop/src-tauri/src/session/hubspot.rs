use crate::profiles::ConnectionProfile;
use crate::session::http_throttle::{send_retried, HttpThrottle};
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method, StatusCode};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::{Arc, Mutex as StdMutex};
use std::time::{Duration, Instant};
use uuid::Uuid;

/// Keychain purpose prefix for a profile's HubSpot credential
/// (`hubspot:{profile_id}` via `credentials.rs`) — a static private-app access
/// token (`pat-…`). Never stored in `profiles.json`.
pub const HUBSPOT_SERVICE: &str = "hubspot";

/// The fixed HubSpot API base (no `host` field on the profile — the portal
/// follows from the token). The mock overrides it via env.
const API_BASE: &str = "https://api.hubapi.com";

/// Pacing for the portal-wide quota (private apps get ~100 req / 10 s burst —
/// and every HubSpot surface shares that one quota, so one session paces
/// everything at ≈10 req/s).
const MIN_INTERVAL: Duration = Duration::from_millis(100);

/// How long a folder's metadata listing stays fresh (the `children` array is
/// the whole directory listing, so fetch once per folder and reuse briefly).
const METADATA_TTL: Duration = Duration::from_secs(30);

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// Keychain purpose under which a profile's HubSpot token is stored. The
/// profile editor only ever `set`/`has` this — the value never crosses IPC.
pub fn credential_key(profile_id: &str) -> String {
    format!("{HUBSPOT_SERVICE}:{profile_id}")
}

/// Extensions the Design Manager accepts on upload. Anything else is rejected
/// client-side, before the PUT (HubSpot answers 400 anyway — we fail earlier
/// and clearer).
const UPLOAD_EXTENSIONS: &[&str] = &[
    "css", "js", "json", "html", "txt", "md", "jpg", "jpeg", "png", "gif", "map", "svg", "ttf",
    "woff", "woff2", "zip",
];

/// Client-side upload whitelist check on a path's extension.
pub fn upload_allowed(path: &str) -> bool {
    let name = path.rsplit('/').next().unwrap_or(path);
    match name.rsplit_once('.') {
        Some((_, ext)) => UPLOAD_EXTENSIONS.contains(&ext.to_ascii_lowercase().as_str()),
        None => false,
    }
}

/// The user-actionable version of [`upload_allowed`], called before every PUT.
fn check_upload(path: &str) -> Result<()> {
    if upload_allowed(path) {
        return Ok(());
    }
    let name = path.rsplit('/').next().unwrap_or(path);
    let ext = name
        .rsplit_once('.')
        .map(|(_, e)| format!(".{e}"))
        .unwrap_or_else(|| "extensionless".to_string());
    Err(anyhow!(
        "HubSpot doesn't allow {ext} files in the Design Manager. Allowed: {}",
        UPLOAD_EXTENSIONS.join(" ")
    ))
}

/// One file/folder in the Design Manager tree, from
/// `GET /cms/v3/source-code/{env}/metadata/{path}`. Folders carry their
/// immediate children unpaginated — the tree is shallow.
#[derive(Debug, Clone)]
pub struct NodeMeta {
    pub name: String,
    pub folder: bool,
    pub size: u64,
    pub updated_at: Option<i64>,
    /// Immediate children when `folder`.
    pub children: Vec<NodeMeta>,
}

/// The default access level for File Manager uploads. Hardcoded: the profile
/// has no per-connection settings hook yet. `PUBLIC_NOT_INDEXABLE` serves the
/// asset to anyone with the URL but keeps it out of search indexes — the
/// safe middle ground for an agency pushing client assets in bulk.
pub const FILES_UPLOAD_ACCESS: &str = "PUBLIC_NOT_INDEXABLE";

/// One file or folder in the HubSpot File Manager (Files API v3), from the
/// `GET /files/v3/files/search` / `GET /files/v3/folders/search` paged
/// listings. Note the API's `name` field excludes the file extension — the
/// display name is derived from `path`, which always carries it.
#[derive(Debug, Clone)]
pub struct FileEntry {
    pub id: String,
    pub name: String,
    /// The File Manager path (`/library/cat.png`), `/`-prefixed.
    pub path: String,
    pub folder: bool,
    pub size: u64,
    /// `PUBLIC_INDEXABLE` / `PUBLIC_NOT_INDEXABLE` / `PRIVATE` (files only).
    pub access: String,
    /// CDN URL for public files (`defaultHostingUrl`).
    pub url: Option<String>,
    pub updated_at: Option<i64>,
}

/// One HubDB table, from `GET /cms/v3/hubdb/tables` (summary) or
/// `GET /cms/v3/hubdb/tables/{idOrName}/draft` (detail, with columns).
/// Surfaced as a virtual `{name}.csv` file under the `/hubdb/` root.
#[derive(Debug, Clone)]
pub struct HubDbTable {
    pub id: String,
    pub name: String,
    pub updated_at: Option<i64>,
    /// Column names in schema order (only the detail response carries them).
    pub columns: Vec<String>,
}

/// One HubDB row: the stable row id plus its `values` map. The id rides the
/// synthesized CSV as the first column so a future write-back can round-trip
/// rows safely — a CSV import without row ids deletes rows missing from the
/// file (the row-deletion caveat that keeps Phase 3 read-only).
#[derive(Debug, Clone)]
pub struct HubDbRow {
    pub id: String,
    pub values: serde_json::Map<String, Value>,
}

/// Folder metadata cache: (environment, path) → (fetched at, listing).
type MetadataCache = StdMutex<HashMap<(String, String), (Instant, Arc<NodeMeta>)>>;

/// Files API listing cache: folder path ("" = root) → (fetched at, entries).
type FilesListingCache = StdMutex<HashMap<String, (Instant, Arc<Vec<FileEntry>>)>>;

/// HubDB table listing cache — the whole `/hubdb/` dir in one entry.
type HubDbCache = StdMutex<Option<(Instant, Arc<Vec<HubDbTable>>)>>;

/// Which HubSpot surfaces the private app's scopes unlock, detected at
/// connect: `content` → the two design roots, `files` → `/files/`, `hubdb` →
/// `/hubdb/`. A missing scope hides that root instead of failing connect.
#[derive(Clone, Copy, Default)]
pub struct Surfaces {
    pub design: bool,
    pub files: bool,
    pub hubdb: bool,
}

/// A live HubSpot connection. Stateless HTTP like the other cloud backends —
/// every op is an independent request carrying the private-app token, paced by
/// a tiny throttle because HubSpot rate-limits hard (429 + `Retry-After`).
pub struct HubSpotSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    /// Static private-app access token (`pat-…`), no OAuth dance.
    token: String,
    /// Pacing: a token bucket of one refilled every 100 ms.
    throttle: HttpThrottle,
    /// The connection label, resolved at connect (portal id when fetchable).
    label: String,
    /// Roots the app's scopes unlock, probed at connect.
    pub surfaces: Surfaces,
    /// Metadata per (environment, path), cached briefly.
    metadata: MetadataCache,
    /// File Manager folder listings, cached briefly (path ↔ id resolution
    /// rides these: a folder's listing carries its children's ids).
    files_listings: FilesListingCache,
    /// The HubDB table listing, cached briefly.
    hubdb_tables: HubDbCache,
}

impl HubSpotSession {
    /// Send a HubSpot API request (`Authorization: Bearer` auth), pacing every
    /// call, honoring `Retry-After` on 429, and backing off on 5xx. `path` may
    /// be an API path (`/cms/v3/…`) or an absolute URL.
    pub async fn send(
        &self,
        method: Method,
        path: &str,
        body: Option<&Value>,
    ) -> Result<reqwest::Response> {
        let url = if path.starts_with("http") {
            path.to_string()
        } else {
            format!("{}{path}", self.api_base)
        };
        let body = body.cloned();
        send_retried(&self.throttle, &format!("hubspot {path}"), move || {
            let url = url.clone();
            let method = method.clone();
            let body = body.clone();
            async move {
                let mut rb = self.client.request(method, &url).bearer_auth(&self.token);
                if let Some(b) = &body {
                    rb = rb.json(b);
                }
                Ok(rb)
            }
        })
        .await
    }

    /// Send one file's bytes as a multipart/form-data PUT/POST (Source Code
    /// API writes take a single `file` field). The form is rebuilt per attempt
    /// because reqwest forms aren't cloneable.
    async fn send_file(
        &self,
        method: Method,
        path: &str,
        file_name: &str,
        data: &[u8],
    ) -> Result<reqwest::Response> {
        self.send_multipart(method, path, Vec::new(), file_name, data)
            .await
    }

    /// Multipart/form-data request with a `file` field plus extra text fields
    /// (Files API uploads take `fileName` / `folderPath` / `options`).
    async fn send_multipart(
        &self,
        method: Method,
        path: &str,
        fields: Vec<(String, String)>,
        file_name: &str,
        data: &[u8],
    ) -> Result<reqwest::Response> {
        let url = format!("{}{path}", self.api_base);
        let file_name = file_name.to_string();
        let data = Arc::new(data.to_vec());
        send_retried(&self.throttle, &format!("hubspot {path}"), move || {
            let url = url.clone();
            let method = method.clone();
            let fields = fields.clone();
            let file_name = file_name.clone();
            let data = data.clone();
            async move {
                let part =
                    reqwest::multipart::Part::bytes(data.as_ref().clone()).file_name(file_name);
                let mut form = reqwest::multipart::Form::new().part("file", part);
                for (k, v) in fields {
                    form = form.text(k, v);
                }
                Ok(self
                    .client
                    .request(method, &url)
                    .bearer_auth(&self.token)
                    .multipart(form))
            }
        })
        .await
    }

    /// The connection label: the portal id when the account-details probe
    /// worked at connect, else a plain "HubSpot".
    pub async fn account_label(&self) -> Result<String> {
        Ok(self.label.clone())
    }

    /// Metadata for `{env}:{path}` (file or folder), cached for ~30s — a
    /// folder's `children` array *is* the directory listing.
    pub async fn metadata(&self, env: &str, path: &str) -> Result<NodeMeta> {
        let key = (env.to_string(), path.to_string());
        {
            if let Some((fetched, node)) = self.metadata.lock().unwrap().get(&key) {
                if fetched.elapsed() < METADATA_TTL {
                    return Ok(node.as_ref().clone());
                }
            }
        }
        // The root's path parameter is "/" — but a single-encoded %2F is
        // rejected at HubSpot's edge (bare Jetty 404, before auth), and an
        // empty segment 404s the same way. Double-encoded %252F survives the
        // edge and resolves to the design root after the API's own decode.
        let api_path = if path.is_empty() {
            format!("/cms/v3/source-code/{env}/metadata/%252F")
        } else {
            format!("/cms/v3/source-code/{env}/metadata/{}", urlenc_path(path))
        };
        let resp = self.send(Method::GET, &api_path, None).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if status == StatusCode::NOT_FOUND {
            return Err(anyhow!("hubspot {env}/{path}: not found"));
        }
        if !status.is_success() {
            return Err(anyhow!(
                "hubspot {api_path} failed ({}): {text}",
                status.as_u16()
            ));
        }
        let v: Value =
            serde_json::from_str(&text).with_context(|| format!("parse hubspot {api_path}"))?;
        let node = node_from_json(&v)
            .ok_or_else(|| anyhow!("hubspot {env}/{path}: malformed metadata"))?;
        self.metadata
            .lock()
            .unwrap()
            .insert(key, (Instant::now(), Arc::new(node.clone())));
        Ok(node)
    }

    /// Drop cached metadata for an environment after a mutation so the next
    /// list re-fetches.
    pub fn invalidate_metadata(&self, env: &str) {
        self.metadata.lock().unwrap().retain(|(e, _), _| e != env);
    }

    /// Metadata stat for one path, or `None` when the portal 404s — the
    /// `exists()` primitive for transfer overwrite policies.
    pub async fn stat(&self, env: &str, path: &str) -> Option<NodeMeta> {
        self.metadata(env, path).await.ok()
    }

    /// Read one file's bytes (`Accept: application/octet-stream`).
    pub async fn content_get(&self, env: &str, path: &str) -> Result<Vec<u8>> {
        let api_path = format!("/cms/v3/source-code/{env}/content/{}", urlenc_path(path));
        let resp = self.send(Method::GET, &api_path, None).await?;
        let status = resp.status();
        if status == StatusCode::NOT_FOUND {
            return Err(anyhow!("hubspot {env}/{path}: not found"));
        }
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("hubspot {api_path} failed ({code}): {text}"));
        }
        Ok(resp
            .bytes()
            .await
            .with_context(|| format!("read hubspot {env}/{path}"))?
            .to_vec())
    }

    /// Write one file (create and update are the same multipart PUT), after
    /// the client-side extension whitelist check.
    pub async fn content_put(&self, env: &str, path: &str, data: &[u8]) -> Result<()> {
        check_upload(path)?;
        let api_path = format!("/cms/v3/source-code/{env}/content/{}", urlenc_path(path));
        let name = path.rsplit('/').next().unwrap_or(path);
        let resp = self.send_file(Method::PUT, &api_path, name, data).await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!(
                "hubspot upload {env}/{path} failed ({code}): {text}"
            ));
        }
        self.invalidate_metadata(env);
        Ok(())
    }

    /// Delete one file. A 404 is not an error (idempotent cleanup after a
    /// best-effort rename).
    pub async fn content_delete(&self, env: &str, path: &str) -> Result<()> {
        let api_path = format!("/cms/v3/source-code/{env}/content/{}", urlenc_path(path));
        let resp = self.send(Method::DELETE, &api_path, None).await?;
        let status = resp.status();
        if !status.is_success() && status != StatusCode::NOT_FOUND {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!(
                "hubspot delete {env}/{path} failed ({code}): {text}"
            ));
        }
        self.invalidate_metadata(env);
        Ok(())
    }

    // ---- Files API v3 (the File Manager, `/files/` root) ----

    /// List one File Manager folder's immediate children (subfolders first,
    /// then files; all pages), cached for ~30s. `folder_path` is relative to
    /// the `/files/` root ("" = root). Errors when the folder doesn't exist.
    pub async fn files_list(&self, folder_path: &str) -> Result<Vec<FileEntry>> {
        let folder_path = folder_path.trim_matches('/');
        let id = self.files_resolve_folder_id(folder_path).await?;
        self.files_listing(folder_path, id.as_deref()).await
    }

    /// The id of a File Manager folder (None for the root, which has none),
    /// resolving segment-by-segment from the root so every ancestor's listing
    /// lands in the cache along the way.
    async fn files_resolve_folder_id(&self, folder_path: &str) -> Result<Option<String>> {
        if folder_path.is_empty() {
            return Ok(None);
        }
        let mut cur = String::new();
        let mut id: Option<String> = None;
        for seg in folder_path.split('/') {
            let entries = self.files_listing(&cur, id.as_deref()).await?;
            let child = entries
                .iter()
                .find(|e| e.folder && e.name == seg)
                .ok_or_else(|| anyhow!("hubspot /files/{folder_path}: folder not found"))?;
            id = Some(child.id.clone());
            cur = if cur.is_empty() {
                seg.to_string()
            } else {
                format!("{cur}/{seg}")
            };
        }
        Ok(id)
    }

    /// One folder's listing, from the cache when fresh, else paged API calls.
    /// The root (no folder id) lists with `parentFolderId` omitted — the real
    /// API then returns *every* file/folder in the portal, so root children
    /// are filtered client-side to depth-1 paths (documented in the plan:
    /// there is no root-scoped list parameter in v3).
    async fn files_listing(
        &self,
        folder_path: &str,
        folder_id: Option<&str>,
    ) -> Result<Vec<FileEntry>> {
        {
            if let Some((fetched, entries)) = self.files_listings.lock().unwrap().get(folder_path) {
                if fetched.elapsed() < METADATA_TTL {
                    return Ok(entries.as_ref().clone());
                }
            }
        }
        let mut entries = self.files_list_kind("folders", folder_id, true).await?;
        entries.extend(self.files_list_kind("files", folder_id, false).await?);
        if folder_path.is_empty() {
            entries.retain(|e| e.path.trim_matches('/').matches('/').count() == 0);
        }
        self.files_listings.lock().unwrap().insert(
            folder_path.to_string(),
            (Instant::now(), Arc::new(entries.clone())),
        );
        Ok(entries)
    }

    /// Page through one Files API search endpoint (`files` or `folders`),
    /// following the `paging.next.after` cursor.
    async fn files_list_kind(
        &self,
        kind: &str,
        parent_id: Option<&str>,
        folder: bool,
    ) -> Result<Vec<FileEntry>> {
        let mut out = Vec::new();
        let mut after: Option<String> = None;
        loop {
            // The plain list URLs 405 at HubSpot's edge — listing goes
            // through the GET search endpoints (same params + response).
            let mut api_path = format!("/files/v3/{kind}/search?limit=100");
            if let Some(id) = parent_id {
                api_path += &format!("&parentFolderId={id}");
            }
            if let Some(a) = &after {
                api_path += &format!("&after={}", urlenc_path(a));
            }
            let resp = self.send(Method::GET, &api_path, None).await?;
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            if !status.is_success() {
                return Err(anyhow!(
                    "hubspot {api_path} failed ({}): {text}",
                    status.as_u16()
                ));
            }
            let v: Value =
                serde_json::from_str(&text).with_context(|| format!("parse hubspot {api_path}"))?;
            if let Some(results) = v.get("results").and_then(|r| r.as_array()) {
                out.extend(
                    results
                        .iter()
                        .filter_map(|r| file_entry_from_json(r, folder)),
                );
            }
            match v
                .get("paging")
                .and_then(|p| p.get("next"))
                .and_then(|n| n.get("after"))
                .and_then(|a| a.as_str())
            {
                Some(a) => after = Some(a.to_string()),
                None => break,
            }
        }
        Ok(out)
    }

    /// Stat one File Manager path (file or folder), or `None` — the
    /// `exists()` primitive for the transfer overwrite policies.
    pub async fn files_stat(&self, path: &str) -> Option<FileEntry> {
        let path = path.trim_matches('/');
        if path.is_empty() {
            return None; // the files root — handled by the caller
        }
        let (parent, name) = match path.rsplit_once('/') {
            Some((p, n)) => (p, n),
            None => ("", path),
        };
        self.files_list(parent)
            .await
            .ok()?
            .into_iter()
            .find(|e| e.name == name)
    }

    /// Read one File Manager file's bytes. Public files come straight off
    /// their CDN URL (no bearer — a different host must not see the token,
    /// and CDN traffic doesn't touch the API quota); `PRIVATE` files go
    /// through `GET /files/v3/files/{id}/signed-url` first.
    pub async fn files_read(&self, path: &str) -> Result<Vec<u8>> {
        let entry = self
            .files_stat(path)
            .await
            .ok_or_else(|| anyhow!("hubspot /files/{path}: not found"))?;
        if entry.folder {
            return Err(anyhow!("hubspot /files/{path} is a folder, not a file"));
        }
        let url = if entry.access == "PRIVATE" {
            let api_path = format!("/files/v3/files/{}/signed-url", entry.id);
            let resp = self.send(Method::GET, &api_path, None).await?;
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            if !status.is_success() {
                return Err(anyhow!(
                    "hubspot {api_path} failed ({}): {text}",
                    status.as_u16()
                ));
            }
            let v: Value =
                serde_json::from_str(&text).with_context(|| format!("parse hubspot {api_path}"))?;
            v.get("url")
                .and_then(|u| u.as_str())
                .map(str::to_string)
                .ok_or_else(|| anyhow!("hubspot {api_path}: no signed url in response"))?
        } else {
            entry
                .url
                .clone()
                .ok_or_else(|| anyhow!("hubspot /files/{path}: no hosting url"))?
        };
        let resp = self
            .client
            .get(&url)
            .send()
            .await
            .with_context(|| format!("download hubspot /files/{path}"))?;
        let status = resp.status();
        if !status.is_success() {
            return Err(anyhow!(
                "hubspot cdn download /files/{path} failed ({})",
                status.as_u16()
            ));
        }
        Ok(resp
            .bytes()
            .await
            .with_context(|| format!("read hubspot /files/{path}"))?
            .to_vec())
    }

    /// Write one File Manager file: replace in place (`PUT …/{id}`) when the
    /// path already exists so references keep working, else upload
    /// (`POST /files/v3/files`). No extension whitelist here — unlike the
    /// Design Manager, the Files API accepts arbitrary types.
    pub async fn files_write(&self, path: &str, data: &[u8]) -> Result<()> {
        let path = path.trim_matches('/');
        let (parent, name) = match path.rsplit_once('/') {
            Some((p, n)) => (p, n),
            None => ("", path),
        };
        if name.is_empty() {
            return Err(anyhow!("hubspot /files/{path} is a folder, not a file"));
        }
        match self.files_stat(path).await {
            Some(e) if e.folder => {
                return Err(anyhow!("hubspot /files/{path} is a folder, not a file"))
            }
            Some(e) => {
                let api_path = format!("/files/v3/files/{}", e.id);
                let resp = self
                    .send_multipart(Method::PUT, &api_path, Vec::new(), name, data)
                    .await?;
                let status = resp.status();
                if !status.is_success() {
                    let code = status.as_u16();
                    let text = resp.text().await.unwrap_or_default();
                    return Err(anyhow!(
                        "hubspot replace /files/{path} failed ({code}): {text}"
                    ));
                }
            }
            None => {
                let options = serde_json::json!({"access": FILES_UPLOAD_ACCESS}).to_string();
                let fields = vec![
                    ("fileName".to_string(), name.to_string()),
                    (
                        "folderPath".to_string(),
                        format!("/{}", parent.trim_matches('/')),
                    ),
                    ("options".to_string(), options),
                ];
                let resp = self
                    .send_multipart(Method::POST, "/files/v3/files", fields, name, data)
                    .await?;
                let status = resp.status();
                if !status.is_success() {
                    let code = status.as_u16();
                    let text = resp.text().await.unwrap_or_default();
                    return Err(anyhow!(
                        "hubspot upload /files/{path} failed ({code}): {text}"
                    ));
                }
            }
        }
        self.invalidate_files();
        Ok(())
    }

    /// Rename a File Manager file in place (`PATCH …/files/{id}`). The API's
    /// `name` property excludes the extension, so the PATCH sends the stem.
    pub async fn files_rename_file(&self, id: &str, new_name: &str) -> Result<()> {
        let body = serde_json::json!({"name": files_name_stem(new_name)});
        let resp = self
            .send(Method::PATCH, &format!("/files/v3/files/{id}"), Some(&body))
            .await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("hubspot rename file {id} failed ({code}): {text}"));
        }
        self.invalidate_files();
        Ok(())
    }

    /// Rename a File Manager folder (`PATCH …/folders/{id}`). v3 folder
    /// updates are ASYNC: the response carries a task token whose status link
    /// we poll briefly, then give up and let the next listing refresh catch
    /// the result (documented best-effort).
    pub async fn files_rename_folder(&self, id: &str, new_name: &str) -> Result<()> {
        let body = serde_json::json!({"name": new_name});
        let resp = self
            .send(
                Method::PATCH,
                &format!("/files/v3/folders/{id}"),
                Some(&body),
            )
            .await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!(
                "hubspot rename folder {id} failed ({}): {text}",
                status.as_u16()
            ));
        }
        if let Ok(v) = serde_json::from_str::<Value>(&text) {
            if let Some(link) = v
                .get("links")
                .and_then(|l| l.get("status"))
                .and_then(|s| s.as_str())
            {
                self.poll_folder_task(link).await?;
            }
        }
        self.invalidate_files();
        Ok(())
    }

    /// Poll a folder-update task for ~2.5s. Still-pending is not an error —
    /// the listing cache is dropped either way, so the next list re-fetches.
    async fn poll_folder_task(&self, status_link: &str) -> Result<()> {
        for _ in 0..10 {
            let resp = self.send(Method::GET, status_link, None).await?;
            let v: Value = resp.json().await.unwrap_or_default();
            match v.get("status").and_then(|s| s.as_str()) {
                Some("COMPLETE") => return Ok(()),
                Some(s) if s == "FAILED" || s == "CANCELED" => {
                    return Err(anyhow!("hubspot folder update task {s}"))
                }
                _ => tokio::time::sleep(Duration::from_millis(250)).await,
            }
        }
        Ok(())
    }

    /// Create a File Manager folder (real folders here — the Files API has
    /// them, so no `.ghostftp-keep` placeholder is needed under `/files/`).
    pub async fn files_create_folder(&self, path: &str) -> Result<()> {
        let path = path.trim_matches('/');
        let (parent, name) = match path.rsplit_once('/') {
            Some((p, n)) => (p, n),
            None => ("", path),
        };
        if name.is_empty() {
            return Err(anyhow!("the /files root already exists"));
        }
        let body = serde_json::json!({
            "name": name,
            "parentFolderPath": format!("/{}", parent.trim_matches('/')),
        });
        let resp = self
            .send(Method::POST, "/files/v3/folders", Some(&body))
            .await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!(
                "hubspot create folder /files/{path} failed ({code}): {text}"
            ));
        }
        self.invalidate_files();
        Ok(())
    }

    /// Delete one File Manager file. A 404 is not an error (idempotent
    /// cleanup, mirroring `content_delete`).
    pub async fn files_delete_file(&self, id: &str) -> Result<()> {
        let resp = self
            .send(Method::DELETE, &format!("/files/v3/files/{id}"), None)
            .await?;
        let status = resp.status();
        if !status.is_success() && status != StatusCode::NOT_FOUND {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("hubspot delete file {id} failed ({code}): {text}"));
        }
        self.invalidate_files();
        Ok(())
    }

    /// Delete one (already empty) File Manager folder.
    pub async fn files_delete_folder(&self, id: &str) -> Result<()> {
        let resp = self
            .send(Method::DELETE, &format!("/files/v3/folders/{id}"), None)
            .await?;
        let status = resp.status();
        if !status.is_success() && status != StatusCode::NOT_FOUND {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!(
                "hubspot delete folder {id} failed ({code}): {text}"
            ));
        }
        self.invalidate_files();
        Ok(())
    }

    /// Drop cached File Manager listings after a mutation so the next list
    /// re-fetches.
    pub fn invalidate_files(&self) {
        self.files_listings.lock().unwrap().clear();
    }

    // ---- HubDB API v3 (the `/hubdb/` root, read-only) ----

    /// GET a JSON API path with the standard status/error handling.
    async fn get_json(&self, api_path: &str) -> Result<Value> {
        let resp = self.send(Method::GET, api_path, None).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!(
                "hubspot {api_path} failed ({}): {text}",
                status.as_u16()
            ));
        }
        serde_json::from_str(&text).with_context(|| format!("parse hubspot {api_path}"))
    }

    /// List all HubDB tables (paged), cached briefly. Read-only by design:
    /// write-back (CSV draft import or sparse row PATCH) ships only once
    /// row-id round-tripping is designed against the import semantics that
    /// DELETE rows missing from the file.
    pub async fn hubdb_list(&self) -> Result<Vec<HubDbTable>> {
        {
            if let Some((fetched, tables)) = self.hubdb_tables.lock().unwrap().as_ref() {
                if fetched.elapsed() < METADATA_TTL {
                    return Ok(tables.as_ref().clone());
                }
            }
        }
        let mut out = Vec::new();
        let mut after: Option<String> = None;
        loop {
            let mut api_path = "/cms/v3/hubdb/tables?limit=100".to_string();
            if let Some(a) = &after {
                api_path += &format!("&after={}", urlenc_path(a));
            }
            let v = self.get_json(&api_path).await?;
            if let Some(results) = v.get("results").and_then(|r| r.as_array()) {
                out.extend(results.iter().filter_map(hubdb_table_from_json));
            }
            match v
                .get("paging")
                .and_then(|p| p.get("next"))
                .and_then(|n| n.get("after"))
                .and_then(|a| a.as_str())
            {
                Some(a) => after = Some(a.to_string()),
                None => break,
            }
        }
        *self.hubdb_tables.lock().unwrap() = Some((Instant::now(), Arc::new(out.clone())));
        Ok(out)
    }

    /// Stat one table by its virtual file name (`pricing.csv`), or `None`.
    pub async fn hubdb_stat(&self, file_name: &str) -> Option<HubDbTable> {
        let name = file_name.strip_suffix(".csv")?;
        self.hubdb_list()
            .await
            .ok()?
            .into_iter()
            .find(|t| t.name == name)
    }

    /// Read one HubDB table as CSV bytes (see [`hubdb_to_csv`]). Uses the
    /// DRAFT endpoints (`…/draft`, `…/rows/draft`): the draft is the latest
    /// state of the table, including unpublished edits — consistent with
    /// Ghost FTP's draft-first labeling and the right basis for a future
    /// write-back. The published variants (`GET /tables/{id}/rows`) are what
    /// the live site renders; exposing them is a future option.
    pub async fn hubdb_read_csv(&self, file_name: &str) -> Result<Vec<u8>> {
        let name = file_name
            .strip_suffix(".csv")
            .filter(|n| !n.is_empty() && !n.contains('/'))
            .ok_or_else(|| anyhow!("hubspot /hubdb/{file_name}: not a table (flat .csv list)"))?;
        let detail = self
            .get_json(&format!("/cms/v3/hubdb/tables/{}/draft", urlenc_path(name)))
            .await
            .with_context(|| format!("hubspot /hubdb/{file_name}: table metadata"))?;
        let table = hubdb_table_from_json(&detail)
            .ok_or_else(|| anyhow!("hubspot /hubdb/{file_name}: malformed table metadata"))?;
        let mut rows = Vec::new();
        let mut after: Option<String> = None;
        loop {
            let mut api_path = format!(
                "/cms/v3/hubdb/tables/{}/rows/draft?limit=100",
                urlenc_path(name)
            );
            if let Some(a) = &after {
                api_path += &format!("&after={}", urlenc_path(a));
            }
            let v = self.get_json(&api_path).await?;
            if let Some(results) = v.get("results").and_then(|r| r.as_array()) {
                rows.extend(results.iter().filter_map(hubdb_row_from_json));
            }
            match v
                .get("paging")
                .and_then(|p| p.get("next"))
                .and_then(|n| n.get("after"))
                .and_then(|a| a.as_str())
            {
                Some(a) => after = Some(a.to_string()),
                None => break,
            }
        }
        Ok(hubdb_to_csv(&table.columns, &rows))
    }
}

/// Open a HubSpot session from a profile whose private-app token is in the OS
/// keychain (`hubspot:{profile_id}`). Validates the token against account
/// details (the one endpoint every private-app token can read, so 401 means a
/// bad token — not a missing scope), then detects which roots the app's
/// scopes unlock: a scope the app lacks hides that root instead of failing
/// connect. Connect only fails when *no* surface is usable.
pub async fn hubspot_connect(profile: &ConnectionProfile) -> Result<HubSpotSession> {
    let token = crate::credentials::get_secret(&credential_key(&profile.id))?
        .filter(|s| !s.trim().is_empty())
        .ok_or_else(|| {
            anyhow!(
                "This HubSpot connection has no saved credential. Open it in the connection \
                 editor and paste a private-app access token (pat-…)."
            )
        })?;

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building HubSpot HTTP client")?;

    let mut session = HubSpotSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or("GHOSTFTP_HUBSPOT_API_BASE", API_BASE),
        token,
        throttle: HttpThrottle::new(MIN_INTERVAL),
        label: "HubSpot".to_string(),
        surfaces: Surfaces::default(),
        metadata: StdMutex::new(HashMap::new()),
        files_listings: StdMutex::new(HashMap::new()),
        hubdb_tables: StdMutex::new(None),
    };

    // Token validity + label in one call.
    let resp = session
        .send(Method::GET, "/account-info/v3/details", None)
        .await?;
    let status = resp.status();
    if status == StatusCode::UNAUTHORIZED {
        return Err(anyhow!(
            "HubSpot rejected the token (401). Check the private-app access token (pat-…) — \
             Settings → Integrations → Private Apps → your app → Show token."
        ));
    }
    if status.is_success() {
        if let Ok(v) = resp.json::<Value>().await {
            if let Some(pid) = v.get("portalId").and_then(|p| p.as_i64()) {
                session.label = format!("HubSpot portal {pid}");
            }
        }
    }

    // Surface detection: one scope per root. The design root's path parameter
    // is "%252F" — the double-encoded "/" (the plain `metadata/` form and
    // single-encoded %2F are rejected at HubSpot's edge with a bare 404).
    session.surfaces = Surfaces {
        design: probe_surface(&session, "/cms/v3/source-code/draft/metadata/%252F").await?,
        files: probe_surface(&session, "/files/v3/files/search?limit=1").await?,
        hubdb: probe_surface(&session, "/cms/v3/hubdb/tables?limit=1").await?,
    };
    if !session.surfaces.design && !session.surfaces.files && !session.surfaces.hubdb {
        return Err(anyhow!(
            "The token is valid, but the private app has none of the scopes Ghost FTP uses. Enable \
             at least one in the app's settings — `content` (Design Manager files), `files` \
             (File Manager), or `hubdb` (HubDB tables) — then reconnect."
        ));
    }
    Ok(session)
}

/// One surface probe at connect: available on 2xx, hidden on 403 (missing
/// scope) or any other non-success, fatal on 401 (bad token — the surface
/// probes run after account details, so this shouldn't happen).
async fn probe_surface(session: &HubSpotSession, path: &str) -> Result<bool> {
    let resp = session.send(Method::GET, path, None).await?;
    let status = resp.status();
    if status == StatusCode::UNAUTHORIZED {
        return Err(anyhow!(
            "HubSpot rejected the token (401). Check the private-app access token (pat-…)."
        ));
    }
    Ok(status.is_success())
}

/// Parse one metadata object from the Source Code API's JSON. A folder's
/// children come as an array of metadata objects.
pub fn node_from_json(v: &Value) -> Option<NodeMeta> {
    let name = v.get("name").and_then(|x| x.as_str())?.to_string();
    let folder = v.get("folder").and_then(|x| x.as_bool()).unwrap_or(false);
    let mut children = Vec::new();
    if let Some(arr) = v.get("children").and_then(|c| c.as_array()) {
        for c in arr {
            if let Some(n) = node_from_json(c) {
                children.push(n);
            }
        }
    }
    Some(NodeMeta {
        name,
        folder,
        size: v.get("size").and_then(|s| s.as_u64()).unwrap_or(0),
        updated_at: v.get("updatedAt").and_then(parse_timestamp),
        children,
    })
}

/// Parse one entry from the Files API's paged `results` arrays (files or
/// folders). v3 ids may come back as numbers or strings; `createdAt` /
/// `updatedAt` are ISO-8601 strings (handled by [`parse_timestamp`]). The
/// display name is the basename of `path` because a file's `name` field
/// excludes its extension.
pub fn file_entry_from_json(v: &Value, folder: bool) -> Option<FileEntry> {
    let id = v.get("id").and_then(|x| {
        x.as_str()
            .map(str::to_string)
            .or_else(|| x.as_i64().map(|n| n.to_string()))
    })?;
    let path = v.get("path").and_then(|p| p.as_str())?.to_string();
    let name = path.trim_matches('/').rsplit('/').next()?.to_string();
    Some(FileEntry {
        id,
        name,
        path,
        folder,
        size: v.get("size").and_then(|s| s.as_u64()).unwrap_or(0),
        access: v
            .get("access")
            .and_then(|a| a.as_str())
            .unwrap_or("PUBLIC_INDEXABLE")
            .to_string(),
        url: v
            .get("defaultHostingUrl")
            .and_then(|u| u.as_str())
            .or_else(|| v.get("url").and_then(|u| u.as_str()))
            .map(str::to_string),
        updated_at: v.get("updatedAt").and_then(parse_timestamp),
    })
}

/// The Files API's `name` property excludes the extension (the `path` keeps
/// it), so a rename PATCH sends the stem and the portal keeps the current
/// extension.
pub fn files_name_stem(name: &str) -> &str {
    match name.rsplit_once('.') {
        Some((stem, _)) if !stem.is_empty() => stem,
        _ => name,
    }
}

/// HubSpot timestamps are epoch milliseconds; tolerate ISO-8601 strings too
/// (riding Shopify's parser).
pub fn parse_timestamp(v: &Value) -> Option<i64> {
    if let Some(ms) = v.as_i64() {
        // Values past ~year 5138 in seconds must be milliseconds.
        return Some(if ms.abs() > 100_000_000_000 {
            ms / 1000
        } else {
            ms
        });
    }
    v.as_str().and_then(crate::session::shopify::parse_time)
}

/// Parse one HubDB table summary/detail. v3 ids may come back as numbers or
/// strings; `columns` (schema order) only appears on the detail response.
pub fn hubdb_table_from_json(v: &Value) -> Option<HubDbTable> {
    let id = v.get("id").and_then(|x| {
        x.as_str()
            .map(str::to_string)
            .or_else(|| x.as_i64().map(|n| n.to_string()))
    })?;
    let name = v.get("name").and_then(|n| n.as_str())?.to_string();
    let columns = v
        .get("columns")
        .and_then(|c| c.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|c| c.get("name").and_then(|n| n.as_str()).map(str::to_string))
                .collect()
        })
        .unwrap_or_default();
    Some(HubDbTable {
        id,
        name,
        updated_at: v.get("updatedAt").and_then(parse_timestamp),
        columns,
    })
}

/// Parse one HubDB row (`{id, values}`), tolerating numeric ids.
pub fn hubdb_row_from_json(v: &Value) -> Option<HubDbRow> {
    let id = v.get("id").and_then(|x| {
        x.as_str()
            .map(str::to_string)
            .or_else(|| x.as_i64().map(|n| n.to_string()))
    })?;
    let values = v
        .get("values")
        .and_then(|x| x.as_object())
        .cloned()
        .unwrap_or_default();
    Some(HubDbRow { id, values })
}

/// Serialize a HubDB table to CSV bytes: an `id` column first (the stable
/// HubDB row id — required for any safe future write-back), then one column
/// per table column in schema order, values from each row's `values` map.
/// RFC-4180 quoting (a cell is quoted when it contains `,`, `"`, `\n` or
/// `\r`; embedded quotes double), `\r\n` line endings, UTF-8. `null`/missing
/// values render empty; numbers and booleans render plain; structured values
/// (arrays/objects — e.g. multi-select columns) render as compact JSON.
pub fn hubdb_to_csv(columns: &[String], rows: &[HubDbRow]) -> Vec<u8> {
    let mut out = String::new();
    let mut header: Vec<&str> = Vec::with_capacity(columns.len() + 1);
    header.push("id");
    header.extend(columns.iter().map(String::as_str));
    csv_row(&mut out, &header);
    for row in rows {
        let mut cells: Vec<String> = Vec::with_capacity(columns.len() + 1);
        cells.push(row.id.clone());
        for col in columns {
            cells.push(match row.values.get(col) {
                None | Some(Value::Null) => String::new(),
                Some(Value::String(s)) => s.clone(),
                Some(v) => v.to_string(), // numbers/bools plain; structured → JSON
            });
        }
        csv_row(&mut out, &cells);
    }
    out.into_bytes()
}

/// One CSV record: cells joined with commas, RFC-4180 quoted, `\r\n` end.
fn csv_row<S: AsRef<str>>(out: &mut String, cells: &[S]) {
    for (i, cell) in cells.iter().enumerate() {
        if i > 0 {
            out.push(',');
        }
        let v = cell.as_ref();
        if v.contains([',', '"', '\n', '\r']) {
            out.push('"');
            out.push_str(&v.replace('"', "\"\""));
            out.push('"');
        } else {
            out.push_str(v);
        }
    }
    out.push_str("\r\n");
}

/// Percent-encode each path segment (HubSpot paths may carry spaces and
/// unicode), preserving the `/` separators.
fn urlenc_path(path: &str) -> String {
    use percent_encoding::{utf8_percent_encode, NON_ALPHANUMERIC};
    path.split('/')
        .map(|seg| utf8_percent_encode(seg, NON_ALPHANUMERIC).to_string())
        .collect::<Vec<_>>()
        .join("/")
}
