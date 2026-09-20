use crate::profiles::ConnectionProfile;
use crate::session::http_throttle::{send_retried, HttpThrottle};
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method, StatusCode};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Mutex as StdMutex;
use std::time::{Duration, Instant};
use uuid::Uuid;

/// Keychain purpose prefix for a profile's Shopify Admin credential
/// (`shopify:{profile_id}` via `credentials.rs`) — either a static Admin API
/// access token (`shpat_…`), or `client_id:client_secret` for a Dev Dashboard
/// app, exchanged on first use. Never stored in `profiles.json`.
pub const SHOPIFY_SERVICE: &str = "shopify";

/// Admin REST API version pinned for all theme/asset calls.
const API_VERSION: &str = "2025-01";

/// Pacing for the REST "leaky bucket" (≈2 req/s on standard shops).
const MIN_INTERVAL: Duration = Duration::from_millis(500);

/// How long a fetched `assets.json` listing stays fresh (it is unpaginated —
/// hundreds of keys — so fetch once per theme and reuse briefly).
const ASSETS_TTL: Duration = Duration::from_secs(30);

/// Client-credentials tokens last 24h; re-fetch this far ahead of expiry.
const TOKEN_MARGIN: Duration = Duration::from_secs(300);

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// Keychain purpose under which a profile's Shopify credential is stored. The
/// profile editor only ever `set`/`has` this — the value never crosses IPC.
pub fn credential_key(profile_id: &str) -> String {
    format!("{SHOPIFY_SERVICE}:{profile_id}")
}

/// A store theme as reported by `GET /themes.json`.
#[derive(Debug, Clone)]
pub struct ThemeInfo {
    pub id: u64,
    pub name: String,
    /// `main` (live), `unpublished`, or `demo`.
    pub role: String,
}

impl ThemeInfo {
    /// Directory name shown at the Ghost FTP root: the live theme carries a suffix
    /// so it's unmistakable before you edit it.
    pub fn display_name(&self) -> String {
        if self.role == "main" {
            format!("{} [main]", self.name)
        } else {
            self.name.clone()
        }
    }
}

/// One theme asset's metadata, from `GET /themes/{id}/assets.json`.
#[derive(Debug, Clone)]
pub struct AssetMeta {
    pub key: String,
    pub size: u64,
    pub updated_at: Option<i64>,
    pub content_type: Option<String>,
}

/// A live Shopify connection. Stateless HTTP like the other cloud backends —
/// every op is an independent request carrying an access token, paced by a
/// tiny throttle because Shopify rate-limits hard (429 + `Retry-After`).
pub struct ShopifySession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    token_url: String,
    /// Static token, or `client_id:client_secret` (see [`credential_key`]).
    secret: String,
    /// Client-credentials exchange cache: `(access_token, expires_at)`.
    token_cache: StdMutex<Option<(String, Instant)>>,
    /// Pacing: a token bucket of one refilled every 500 ms.
    throttle: HttpThrottle,
    /// Theme list, probed at connect (connect-time validation).
    themes: StdMutex<Vec<ThemeInfo>>,
    /// Asset listing per theme id, cached briefly.
    assets: StdMutex<HashMap<u64, (Instant, Vec<AssetMeta>)>>,
}

impl ShopifySession {
    /// A valid access token: the static token as-is, or a client-credentials
    /// exchange cached with a 5-minute margin (same shape as
    /// `oauth::RefreshingToken`, minus the loopback).
    pub async fn token(&self) -> Result<String> {
        let Some((client_id, client_secret)) = self.secret.split_once(':') else {
            return Ok(self.secret.clone());
        };
        {
            if let Some((tok, exp)) = &*self.token_cache.lock().unwrap() {
                if Instant::now() + TOKEN_MARGIN < *exp {
                    return Ok(tok.clone());
                }
            }
        }
        let body = serde_json::json!({
            "client_id": client_id,
            "client_secret": client_secret,
            "grant_type": "client_credentials",
        });
        let resp = self
            .client
            .post(&self.token_url)
            .json(&body)
            .send()
            .await
            .context("shopify token exchange")?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!(
                "shopify token exchange failed ({}): {text}",
                status.as_u16()
            ));
        }
        let v: Value = serde_json::from_str(&text).context("parse shopify token response")?;
        let token = v
            .get("access_token")
            .and_then(|t| t.as_str())
            .ok_or_else(|| anyhow!("shopify token exchange returned no access_token"))?
            .to_string();
        let expires_in = v
            .get("expires_in")
            .and_then(|e| e.as_u64())
            .unwrap_or(86_399);
        *self.token_cache.lock().unwrap() = Some((
            token.clone(),
            Instant::now() + Duration::from_secs(expires_in),
        ));
        Ok(token)
    }

    /// Hold requests to ≈2/s, honor `Retry-After` on 429, and back off on 5xx
    /// (via the shared [`send_retried`]). `path` may be an API path
    /// (`/themes.json…`) or an absolute URL.
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
        send_retried(&self.throttle, &format!("shopify {path}"), move || {
            let url = url.clone();
            let method = method.clone();
            let body = body.clone();
            async move {
                let token = self.token().await?;
                let mut rb = self
                    .client
                    .request(method, &url)
                    .header("X-Shopify-Access-Token", token);
                if let Some(b) = &body {
                    rb = rb.json(b);
                }
                Ok(rb)
            }
        })
        .await
    }

    /// `send` + status check + JSON parse (mirrors the other cloud sessions).
    pub async fn rpc(&self, method: Method, path: &str, body: Option<&Value>) -> Result<Value> {
        let resp = self.send(method, path, body).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!(
                "shopify {path} failed ({}): {text}",
                status.as_u16()
            ));
        }
        if text.trim().is_empty() {
            return Ok(Value::Null);
        }
        serde_json::from_str(&text).with_context(|| format!("parse shopify {path} response"))
    }

    /// The shop domain — the connection label.
    pub async fn account_label(&self) -> Result<String> {
        Ok(self.profile.host.clone())
    }

    /// Themes probed at connect, as virtual root directories.
    pub fn themes(&self) -> Vec<ThemeInfo> {
        self.themes.lock().unwrap().clone()
    }

    /// Look up a theme by its display name (`Dawn [main]` / `Dawn`).
    pub fn find_theme(&self, display_name: &str) -> Option<ThemeInfo> {
        self.themes
            .lock()
            .unwrap()
            .iter()
            .find(|t| t.display_name() == display_name)
            .cloned()
    }

    /// The flat asset listing for a theme, cached for ~30s (the API returns
    /// the whole thing unpaginated, so this *is* the directory tree).
    pub async fn assets(&self, theme_id: u64) -> Result<Vec<AssetMeta>> {
        {
            if let Some((fetched, list)) = self.assets.lock().unwrap().get(&theme_id) {
                if fetched.elapsed() < ASSETS_TTL {
                    return Ok(list.clone());
                }
            }
        }
        let v = self
            .rpc(
                Method::GET,
                &format!("/themes/{theme_id}/assets.json"),
                None,
            )
            .await?;
        let mut out = Vec::new();
        if let Some(arr) = v.get("assets").and_then(|a| a.as_array()) {
            for a in arr {
                if let Some(m) = asset_from_json(a) {
                    out.push(m);
                }
            }
        }
        self.assets
            .lock()
            .unwrap()
            .insert(theme_id, (Instant::now(), out.clone()));
        Ok(out)
    }

    /// Drop the cached listing after a mutation so the next list re-fetches.
    pub fn invalidate_assets(&self, theme_id: u64) {
        self.assets.lock().unwrap().remove(&theme_id);
    }

    /// Read one asset: text comes back as `value`, binary as base64
    /// `attachment`.
    pub async fn asset_get(&self, theme_id: u64, key: &str) -> Result<Vec<u8>> {
        use base64::Engine as _;
        let path = format!("/themes/{theme_id}/assets.json?asset[key]={}", urlenc(key));
        let v = self.rpc(Method::GET, &path, None).await?;
        let asset = v
            .get("asset")
            .ok_or_else(|| anyhow!("shopify asset {key}: empty response"))?;
        if let Some(text) = asset.get("value").and_then(|x| x.as_str()) {
            return Ok(text.as_bytes().to_vec());
        }
        let b64 = asset
            .get("attachment")
            .and_then(|x| x.as_str())
            .ok_or_else(|| anyhow!("shopify asset {key}: neither value nor attachment"))?;
        base64::engine::general_purpose::STANDARD
            .decode(b64)
            .with_context(|| format!("decode asset {key}"))
    }

    /// Write one asset (create and update are the same PUT). Text assets go as
    /// `value`, binary as base64 `attachment`.
    pub async fn asset_put(&self, theme_id: u64, key: &str, data: &[u8]) -> Result<()> {
        use base64::Engine as _;
        let text = crate::remotefs::shopify::is_text_key(key)
            .then(|| std::str::from_utf8(data).ok())
            .flatten();
        let asset = match text {
            Some(t) => serde_json::json!({ "key": key, "value": t }),
            None => serde_json::json!({
                "key": key,
                "attachment": base64::engine::general_purpose::STANDARD.encode(data),
            }),
        };
        let body = serde_json::json!({ "asset": asset });
        self.rpc(
            Method::PUT,
            &format!("/themes/{theme_id}/assets.json"),
            Some(&body),
        )
        .await?;
        self.invalidate_assets(theme_id);
        Ok(())
    }

    /// Delete one asset by key.
    pub async fn asset_delete(&self, theme_id: u64, key: &str) -> Result<()> {
        let path = format!("/themes/{theme_id}/assets.json?asset[key]={}", urlenc(key));
        self.rpc(Method::DELETE, &path, None).await?;
        self.invalidate_assets(theme_id);
        Ok(())
    }
}

/// Open a Shopify session from a profile whose credential is in the OS
/// keychain (`shopify:{profile_id}`). Probes `GET /themes.json` so a bad token
/// or wrong domain fails at connect, not on first browse.
pub async fn shopify_connect(profile: &ConnectionProfile) -> Result<ShopifySession> {
    let secret = crate::credentials::get_secret(&credential_key(&profile.id))?
        .filter(|s| !s.trim().is_empty())
        .ok_or_else(|| {
            anyhow!(
                "This Shopify connection has no saved credential. Open it in the connection \
                 editor and paste an Admin API access token or client credentials."
            )
        })?;

    let shop = profile.host.trim().trim_end_matches('/').to_string();
    if shop.is_empty() {
        return Err(anyhow!("Shopify profile is missing the store domain"));
    }

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building Shopify HTTP client")?;

    let session = ShopifySession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or(
            "GHOSTFTP_SHOPIFY_API_BASE",
            &format!("https://{shop}/admin/api/{API_VERSION}"),
        ),
        token_url: env_or(
            "GHOSTFTP_SHOPIFY_TOKEN_URL",
            &format!("https://{shop}/admin/oauth/access_token"),
        ),
        secret,
        token_cache: StdMutex::new(None),
        throttle: HttpThrottle::new(MIN_INTERVAL),
        themes: StdMutex::new(Vec::new()),
        assets: StdMutex::new(HashMap::new()),
    };

    // Connect-time validation with user-actionable errors.
    let resp = session.send(Method::GET, "/themes.json", None).await?;
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    match status {
        StatusCode::UNAUTHORIZED => {
            return Err(anyhow!(
                "Shopify rejected the credential (401). Check the access token / client \
                 credentials and that the app has the read_themes + write_themes scopes."
            ))
        }
        StatusCode::NOT_FOUND => {
            return Err(anyhow!(
                "Shopify store not found (404). Check the store domain ({shop})."
            ))
        }
        s if !s.is_success() => {
            return Err(anyhow!(
                "shopify /themes.json failed ({}): {text}",
                s.as_u16()
            ))
        }
        _ => {}
    }
    let v: Value = serde_json::from_str(&text).context("parse shopify /themes.json")?;
    let mut themes = Vec::new();
    if let Some(arr) = v.get("themes").and_then(|t| t.as_array()) {
        for t in arr {
            let (Some(id), Some(name)) = (
                t.get("id").and_then(|x| x.as_u64()),
                t.get("name").and_then(|x| x.as_str()),
            ) else {
                continue;
            };
            themes.push(ThemeInfo {
                id,
                name: name.to_string(),
                role: t
                    .get("role")
                    .and_then(|x| x.as_str())
                    .unwrap_or("")
                    .to_string(),
            });
        }
    }
    *session.themes.lock().unwrap() = themes;
    Ok(session)
}

/// Parse one asset object from `GET /themes/{id}/assets.json`.
pub fn asset_from_json(a: &Value) -> Option<AssetMeta> {
    let key = a.get("key").and_then(|x| x.as_str())?.to_string();
    Some(AssetMeta {
        key,
        size: a.get("size").and_then(|s| s.as_u64()).unwrap_or(0),
        updated_at: a
            .get("updated_at")
            .and_then(|x| x.as_str())
            .and_then(parse_time),
        content_type: a
            .get("content_type")
            .and_then(|x| x.as_str())
            .map(|s| s.to_string()),
    })
}

/// Parse Shopify's ISO-8601 timestamps (`2024-07-15T09:30:00Z`, or with a
/// `±HH:MM` offset). Dependency-free — rides the Dropbox parser for the base.
pub fn parse_time(s: &str) -> Option<i64> {
    let s = s.trim();
    let (base, offset_secs) = match s.len().checked_sub(6).map(|i| s.split_at(i)) {
        Some((base, off)) if off.starts_with('+') || off.starts_with('-') => {
            let sign: i64 = if off.starts_with('-') { -1 } else { 1 };
            let hh: i64 = off.get(1..3)?.parse().ok()?;
            let mm: i64 = off.get(4..6)?.parse().ok()?;
            (base, sign * (hh * 3600 + mm * 60))
        }
        _ => (s, 0),
    };
    let naive = crate::remotefs::dropbox::parse_iso8601(base)?;
    Some(naive - offset_secs)
}

fn urlenc(s: &str) -> String {
    use percent_encoding::{utf8_percent_encode, NON_ALPHANUMERIC};
    utf8_percent_encode(s, NON_ALPHANUMERIC).to_string()
}
