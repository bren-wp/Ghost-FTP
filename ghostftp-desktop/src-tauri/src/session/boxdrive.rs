use crate::oauth::{self, OAuthConfig, RefreshingToken};
use crate::profiles::ConnectionProfile;
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Mutex as StdMutex;
use std::time::Duration;
use uuid::Uuid;

/// Keychain service name for Box tokens (keyed by profile id).
pub const BOX_SERVICE: &str = "ghostftp-box";

const DEFAULT_API_BASE: &str = "https://api.box.com/2.0";
const DEFAULT_UPLOAD_BASE: &str = "https://upload.box.com/api/2.0";
const DEFAULT_AUTH_URL: &str = "https://account.box.com/api/oauth2/authorize";
const DEFAULT_TOKEN_URL: &str = "https://api.box.com/oauth2/token";

/// Box root folder id.
pub const ROOT_ID: &str = "0";

/// Box app client id. Create a **Custom App → OAuth 2.0 (User Auth)** at
/// <https://app.box.com/developers/console>, add `http://localhost:53682/` as a
/// redirect, enable "write all files" scope, and paste the Client ID here or set
/// `GHOSTFTP_BOX_CLIENT_ID`. (Box also issues a client secret; PKCE is used so it's
/// not embedded — leave it out of Ghost FTP.)
// Public provider application ID injected during Windows/Linux compilation.
// Runtime GHOSTFTP_* overrides still support local developer builds.
const BOX_CLIENT_ID: &str = match option_env!("GHOSTFTP_BOX_CLIENT_ID") {
    Some(id) => id,
    None => "",
};

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

pub fn box_config() -> OAuthConfig {
    OAuthConfig {
        client_id: env_or("GHOSTFTP_BOX_CLIENT_ID", BOX_CLIENT_ID),
        auth_url: env_or("GHOSTFTP_BOX_AUTH_URL", DEFAULT_AUTH_URL),
        token_url: env_or("GHOSTFTP_BOX_TOKEN_URL", DEFAULT_TOKEN_URL),
        // Box scopes are configured on the app, not requested per-flow.
        scopes: vec![],
        extra_auth_params: vec![],
    }
}

/// A live Box connection. Box is **ID-addressed** like Google Drive, so this
/// session carries a path→id resolver cache (cleared on structural mutations).
pub struct BoxSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    pub upload_base: String,
    token: RefreshingToken,
    cache: StdMutex<HashMap<String, String>>,
}

impl BoxSession {
    pub async fn access_token(&self) -> Result<String> {
        self.token.access_token().await
    }
    pub async fn force_refresh(&self) -> Result<()> {
        self.token.force_refresh().await
    }
    pub fn clear_cache(&self) {
        self.cache
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .clear();
    }
    pub fn cache_path(&self, ghostftp_path: &str, id: &str) {
        self.cache
            .lock()
            .unwrap()
            .insert(normalize(ghostftp_path), id.to_string());
    }

    pub async fn send(
        &self,
        method: Method,
        url: &str,
        body: Option<&Value>,
    ) -> Result<reqwest::Response> {
        // Provider pagination links are untrusted: do not attach an OAuth
        // bearer token to cross-origin, credentialed or downgraded URLs.
        let full = crate::session::http_throttle::trusted_bearer_url(&self.api_base, url)?;
        let mut attempt = 0;
        loop {
            let token = self.token.access_token().await?;
            let mut rb = self
                .client
                .request(method.clone(), full.as_str())
                .bearer_auth(&token);
            if let Some(b) = body {
                rb = rb.json(b);
            }
            let resp = rb
                .send()
                .await
                .map_err(|_| anyhow!("Box API transport error"))?;
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
        if !status.is_success() {
            return Err(box_http_error("request", status));
        }
        crate::session::http_throttle::bounded_cloud_json(resp).await
    }

    pub async fn get_stream(&self, url: &str) -> Result<reqwest::Response> {
        let resp = self.send(Method::GET, url, None).await?;
        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            return Err(anyhow!("Box download failed (HTTP {code})"));
        }
        Ok(resp)
    }

    pub async fn account_label(&self) -> Result<String> {
        let v = self.rpc(Method::GET, "/users/me", None).await?;
        for key in ["login", "name"] {
            if let Some(s) = v
                .get(key)
                .and_then(|x| x.as_str())
                .filter(|s| !s.is_empty())
            {
                return Ok(s.to_string());
            }
        }
        Ok("Box".to_string())
    }

    /// All entries in a folder (paged), each `{type,id,name,size,modified_at,sha1}`.
    pub async fn folder_items(&self, folder_id: &str) -> Result<Vec<Value>> {
        let mut out = Vec::new();
        let mut offset = 0u64;
        loop {
            let url = format!(
                "/folders/{folder_id}/items?fields=id,name,type,size,modified_at,sha1&limit=1000&offset={offset}"
            );
            let v = self.rpc(Method::GET, &url, None).await?;
            let entries = v
                .get("entries")
                .and_then(|e| e.as_array())
                .ok_or_else(|| anyhow!("Box folder listing has no valid entries array"))?;
            let total = v
                .get("total_count")
                .and_then(|t| t.as_u64())
                .unwrap_or(offset.saturating_add(entries.len() as u64));
            let next_offset = checked_box_page(offset, entries.len(), total)?;
            out.extend(entries.iter().cloned());
            offset = next_offset;
            if offset >= total {
                break;
            }
        }
        Ok(out)
    }

    /// Find a direct child by name, returning `(id, is_folder)`.
    pub async fn find_child(&self, parent_id: &str, name: &str) -> Result<Option<(String, bool)>> {
        for e in self.folder_items(parent_id).await? {
            if e.get("name").and_then(|n| n.as_str()) == Some(name) {
                let id = e
                    .get("id")
                    .and_then(|x| x.as_str())
                    .unwrap_or("")
                    .to_string();
                let is_folder = e.get("type").and_then(|t| t.as_str()) == Some("folder");
                return Ok(Some((id, is_folder)));
            }
        }
        Ok(None)
    }

    pub async fn folder_id(&self, ghostftp_path: &str) -> Result<String> {
        let norm = normalize(ghostftp_path);
        if let Some(id) = self.cache.lock().unwrap().get(&norm).cloned() {
            return Ok(id);
        }
        if norm == "/" {
            return Ok(ROOT_ID.to_string());
        }
        let mut id = ROOT_ID.to_string();
        let mut acc = String::new();
        for seg in norm.trim_matches('/').split('/') {
            acc = format!("{acc}/{seg}");
            let cached = self.cache.lock().unwrap().get(&acc).cloned();
            id = match cached {
                Some(c) => c,
                None => {
                    let (cid, is_folder) = self
                        .find_child(&id, seg)
                        .await?
                        .ok_or_else(|| anyhow!("{ghostftp_path}: no such folder"))?;
                    if !is_folder {
                        return Err(anyhow!("{acc} is a file, not a folder"));
                    }
                    self.cache.lock().unwrap().insert(acc.clone(), cid.clone());
                    cid
                }
            };
        }
        Ok(id)
    }

    pub async fn resolve_item(&self, ghostftp_path: &str) -> Result<Option<(String, bool)>> {
        let norm = normalize(ghostftp_path);
        if norm == "/" {
            return Ok(Some((ROOT_ID.to_string(), true)));
        }
        let parent_id = self.folder_id(&parent_of(&norm)).await?;
        self.find_child(&parent_id, basename(&norm)).await
    }

    pub async fn size(&self, ghostftp_path: &str) -> u64 {
        if let Ok(Some((id, is_folder))) = self.resolve_item(ghostftp_path).await {
            if !is_folder {
                if let Ok(v) = self
                    .rpc(Method::GET, &format!("/files/{id}?fields=size"), None)
                    .await
                {
                    return v.get("size").and_then(|s| s.as_u64()).unwrap_or(0);
                }
            }
        }
        0
    }
}

/// Never include response bodies, signed URLs or bearer tokens in diagnostics.
/// HTTP status is sufficient for a user-actionable retry/auth error.
fn box_http_error(operation: &str, status: reqwest::StatusCode) -> anyhow::Error {
    anyhow!("Box API {operation} failed (HTTP {})", status.as_u16())
}

/// A Box folder uses offset pagination, and a hostile or broken provider must
/// not keep a desktop session allocating indefinitely. Bound both page size
/// and the total returned inventory, including inconsistent total_count values.
const BOX_PAGE_SIZE: usize = 1_000;
const MAX_BOX_FOLDER_ITEMS: u64 = 100_000;

fn checked_box_page(offset: u64, page_len: usize, total: u64) -> Result<u64> {
    anyhow::ensure!(
        page_len <= BOX_PAGE_SIZE,
        "Box folder page exceeds requested size"
    );
    let next = offset
        .checked_add(page_len as u64)
        .context("Box folder page offset overflow")?;
    anyhow::ensure!(
        next <= MAX_BOX_FOLDER_ITEMS && total <= MAX_BOX_FOLDER_ITEMS,
        "Box folder exceeds the safe listing limit of 100,000 entries"
    );
    anyhow::ensure!(
        next >= total || page_len > 0,
        "Box folder listing ended before the declared total"
    );
    Ok(next)
}

pub async fn box_connect(profile: &ConnectionProfile) -> Result<BoxSession> {
    let tokens = oauth::load_tokens(BOX_SERVICE, &profile.id)?.ok_or_else(|| {
        anyhow!(
            "This Box connection isn't authorized yet. Open it in the connection editor \
             and click “Connect with Box”."
        )
    })?;

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building Box HTTP client")?;

    Ok(BoxSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or("GHOSTFTP_BOX_API_BASE", DEFAULT_API_BASE),
        upload_base: env_or("GHOSTFTP_BOX_UPLOAD_BASE", DEFAULT_UPLOAD_BASE),
        token: RefreshingToken::new(BOX_SERVICE, &profile.id, box_config(), tokens),
        cache: StdMutex::new(HashMap::new()),
    })
}

// ---- path helpers (shared shape with the Drive resolver) ----

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

#[cfg(test)]
mod pagination_safety_tests {
    use super::*;

    #[test]
    fn box_page_accepts_valid_boundary_and_empty_folder() {
        assert_eq!(checked_box_page(0, 0, 0).unwrap(), 0);
        assert_eq!(checked_box_page(0, 1000, 100_000).unwrap(), 1000);
        assert_eq!(checked_box_page(99_000, 1000, 100_000).unwrap(), 100_000);
    }

    #[test]
    fn box_page_rejects_unbounded_or_incomplete_provider_listing() {
        assert!(checked_box_page(0, 1001, 1001).is_err());
        assert!(checked_box_page(0, 1000, 100_001).is_err());
        assert!(checked_box_page(99_999, 2, 100_000).is_err());
        assert!(checked_box_page(u64::MAX, 1, u64::MAX).is_err());
        assert!(checked_box_page(1000, 0, 2000).is_err());
    }

    #[test]
    fn box_error_diagnostics_never_include_remote_body_or_bearer_url() {
        let message = box_http_error("request", reqwest::StatusCode::FORBIDDEN).to_string();
        assert_eq!(message, "Box API request failed (HTTP 403)");
        assert!(!message.contains("PRIVATE_TOKEN"));
    }
}
