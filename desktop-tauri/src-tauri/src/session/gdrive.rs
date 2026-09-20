use crate::oauth::{self, OAuthConfig, RefreshingToken};
use crate::profiles::ConnectionProfile;
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Mutex as StdMutex;
use std::time::Duration;
use uuid::Uuid;

/// Keychain service name for Google Drive tokens (keyed by profile id).
pub const GDRIVE_SERVICE: &str = "ghostftp-gdrive";

const DEFAULT_API_BASE: &str = "https://www.googleapis.com/drive/v3";
const DEFAULT_UPLOAD_BASE: &str = "https://www.googleapis.com/upload/drive/v3";
const DEFAULT_AUTH_URL: &str = "https://accounts.google.com/o/oauth2/v2/auth";
const DEFAULT_TOKEN_URL: &str = "https://oauth2.googleapis.com/token";

pub const FOLDER_MIME: &str = "application/vnd.google-apps.folder";

/// Google OAuth client id. Create a **Desktop app** OAuth client at
/// <https://console.cloud.google.com/apis/credentials>, enable the Drive API,
/// add `http://localhost:53682/` as an authorized redirect, and paste the client
/// id here or set `GHOSTFTP_GDRIVE_CLIENT_ID`. PKCE ⇒ no secret needed for a public
/// desktop client.
const GDRIVE_CLIENT_ID: &str = "";

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// Google Drive OAuth config. `access_type=offline` + `prompt=consent` are what
/// make Google return a refresh token.
pub fn gdrive_config() -> OAuthConfig {
    OAuthConfig {
        client_id: env_or("GHOSTFTP_GDRIVE_CLIENT_ID", GDRIVE_CLIENT_ID),
        auth_url: env_or("GHOSTFTP_GDRIVE_AUTH_URL", DEFAULT_AUTH_URL),
        token_url: env_or("GHOSTFTP_GDRIVE_TOKEN_URL", DEFAULT_TOKEN_URL),
        scopes: vec!["https://www.googleapis.com/auth/drive".into()],
        extra_auth_params: vec![
            ("access_type".into(), "offline".into()),
            ("prompt".into(), "consent".into()),
        ],
    }
}

/// A live Google Drive connection. Drive is **ID-addressed**, so this session
/// carries a path→file-id resolver cache; every RemoteFs op resolves the path to
/// an id first. The cache is cleared on any structural mutation to avoid stale ids.
pub struct GDriveSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    pub upload_base: String,
    token: RefreshingToken,
    /// Ghost FTP path → Drive file id. `/` maps to the special id `root`.
    cache: StdMutex<HashMap<String, String>>,
}

impl GDriveSession {
    pub async fn access_token(&self) -> Result<String> {
        self.token.access_token().await
    }

    pub async fn force_refresh(&self) -> Result<()> {
        self.token.force_refresh().await
    }

    pub fn clear_cache(&self) {
        self.cache.lock().unwrap().clear();
    }

    /// Warm the resolver cache with a known path→id (used while listing).
    pub fn cache_path(&self, ghostftp_path: &str, id: &str) {
        self.cache
            .lock()
            .unwrap()
            .insert(normalize(ghostftp_path), id.to_string());
    }

    /// Send a Drive request (bearer + one 401→refresh retry). `path` may be a
    /// Drive API path (`/files…`), an upload path, or an absolute URL.
    pub async fn send(
        &self,
        method: Method,
        url: &str,
        body: Option<&Value>,
    ) -> Result<reqwest::Response> {
        let full = if url.starts_with("http") {
            url.to_string()
        } else {
            format!("{}{url}", self.api_base)
        };
        let mut attempt = 0;
        loop {
            let token = self.token.access_token().await?;
            let mut rb = self.client.request(method.clone(), &full).bearer_auth(&token);
            if let Some(b) = body {
                rb = rb.json(b);
            }
            let resp = rb.send().await.with_context(|| format!("drive {url}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                attempt += 1;
                self.force_refresh().await?;
                continue;
            }
            return Ok(resp);
        }
    }

    pub async fn rpc(&self, method: Method, url: &str, body: Option<&Value>) -> Result<Value> {
        let resp = self.send(method, url, body).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!("drive {url} failed ({}): {text}", status.as_u16()));
        }
        if text.trim().is_empty() {
            return Ok(Value::Null);
        }
        serde_json::from_str(&text).with_context(|| format!("parse drive {url} response"))
    }

    pub async fn get_stream(&self, url: &str) -> Result<reqwest::Response> {
        let resp = self.send(Method::GET, url, None).await?;
        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("drive GET {url} failed ({code}): {text}"));
        }
        Ok(resp)
    }

    pub async fn account_label(&self) -> Result<String> {
        let v = self.rpc(Method::GET, "/about?fields=user", None).await?;
        let user = v.get("user");
        for key in ["emailAddress", "displayName"] {
            if let Some(s) = user
                .and_then(|u| u.get(key))
                .and_then(|x| x.as_str())
                .filter(|s| !s.is_empty())
            {
                return Ok(s.to_string());
            }
        }
        Ok("Google Drive".to_string())
    }

    /// Find a direct child of `parent_id` by name, returning `(id, is_folder)`.
    pub async fn find_child(&self, parent_id: &str, name: &str) -> Result<Option<(String, bool)>> {
        let q = format!(
            "'{}' in parents and name = '{}' and trashed = false",
            parent_id,
            escape_q(name)
        );
        let url = format!(
            "/files?q={}&fields=files(id,mimeType)&pageSize=2",
            urlencode(&q)
        );
        let v = self.rpc(Method::GET, &url, None).await?;
        let first = v
            .get("files")
            .and_then(|f| f.as_array())
            .and_then(|a| a.first());
        Ok(first.map(|f| {
            let id = f.get("id").and_then(|x| x.as_str()).unwrap_or("").to_string();
            let is_folder = f.get("mimeType").and_then(|m| m.as_str()) == Some(FOLDER_MIME);
            (id, is_folder)
        }))
    }

    /// Resolve a Ghost FTP directory path to a Drive folder id (`root` for `/`).
    pub async fn folder_id(&self, ghostftp_path: &str) -> Result<String> {
        let norm = normalize(ghostftp_path);
        if let Some(id) = self.cache.lock().unwrap().get(&norm).cloned() {
            return Ok(id);
        }
        if norm == "/" {
            return Ok("root".to_string());
        }
        let mut id = "root".to_string();
        let mut acc = String::new();
        for seg in norm.trim_matches('/').split('/') {
            acc = format!("{acc}/{seg}");
            let cached = self.cache.lock().unwrap().get(&acc).cloned();
            id = match cached {
                Some(c) => c,
                None => {
                    let (child_id, is_folder) = self
                        .find_child(&id, seg)
                        .await?
                        .ok_or_else(|| anyhow!("{ghostftp_path}: no such folder"))?;
                    if !is_folder {
                        return Err(anyhow!("{acc} is a file, not a folder"));
                    }
                    self.cache.lock().unwrap().insert(acc.clone(), child_id.clone());
                    child_id
                }
            };
        }
        Ok(id)
    }

    /// Resolve a Ghost FTP item path to `(id, is_folder)`, or `None` if missing.
    pub async fn resolve_item(&self, ghostftp_path: &str) -> Result<Option<(String, bool)>> {
        let norm = normalize(ghostftp_path);
        if norm == "/" {
            return Ok(Some(("root".to_string(), true)));
        }
        let parent = parent_of(&norm);
        let name = basename(&norm);
        let parent_id = self.folder_id(&parent).await?;
        self.find_child(&parent_id, name).await
    }

    pub async fn size(&self, ghostftp_path: &str) -> u64 {
        if let Ok(Some((id, _))) = self.resolve_item(ghostftp_path).await {
            if let Ok(v) = self
                .rpc(Method::GET, &format!("/files/{id}?fields=size"), None)
                .await
            {
                return v
                    .get("size")
                    .and_then(|s| s.as_str())
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(0);
            }
        }
        0
    }

    pub async fn exists(&self, ghostftp_path: &str) -> bool {
        matches!(self.resolve_item(ghostftp_path).await, Ok(Some(_)))
    }
}

pub async fn gdrive_connect(profile: &ConnectionProfile) -> Result<GDriveSession> {
    let tokens = oauth::load_tokens(GDRIVE_SERVICE, &profile.id)?.ok_or_else(|| {
        anyhow!(
            "This Google Drive connection isn't authorized yet. Open it in the connection \
             editor and click “Connect with Google Drive”."
        )
    })?;

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building Google Drive HTTP client")?;

    Ok(GDriveSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or("GHOSTFTP_GDRIVE_API_BASE", DEFAULT_API_BASE),
        upload_base: env_or("GHOSTFTP_GDRIVE_UPLOAD_BASE", DEFAULT_UPLOAD_BASE),
        token: RefreshingToken::new(GDRIVE_SERVICE, &profile.id, gdrive_config(), tokens),
        cache: StdMutex::new(HashMap::new()),
    })
}

// ---- path helpers ----

pub fn normalize(ghostftp: &str) -> String {
    let t = ghostftp.trim().trim_matches('/');
    if t.is_empty() || t == "." {
        "/".to_string()
    } else {
        format!("/{t}")
    }
}

pub fn basename(ghostftp: &str) -> &str {
    ghostftp.trim_matches('/').rsplit('/').next().unwrap_or("")
}

pub fn parent_of(ghostftp: &str) -> String {
    let t = ghostftp.trim_matches('/');
    match t.rsplit_once('/') {
        Some((p, _)) => format!("/{p}"),
        None => "/".to_string(),
    }
}

/// Escape a value for a Drive `q` string literal (backslash + single-quote).
pub fn escape_q(name: &str) -> String {
    name.replace('\\', "\\\\").replace('\'', "\\'")
}

/// Whether a Drive mimeType denotes a folder.
pub fn folder_mime_is(mime: &str) -> bool {
    mime == FOLDER_MIME
}

fn urlencode(s: &str) -> String {
    use percent_encoding::{utf8_percent_encode, NON_ALPHANUMERIC};
    utf8_percent_encode(s, NON_ALPHANUMERIC).to_string()
}
