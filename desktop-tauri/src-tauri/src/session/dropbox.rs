use crate::oauth::{self, OAuthConfig, TokenSet};
use crate::profiles::ConnectionProfile;
use anyhow::{anyhow, Context, Result};
use reqwest::Client;
use std::time::Duration;
use tokio::sync::Mutex;
use uuid::Uuid;

/// Keychain service name under which Dropbox tokens are stored (keyed by profile id).
pub const DROPBOX_SERVICE: &str = "ghostftp-dropbox";

const DEFAULT_API_BASE: &str = "https://api.dropboxapi.com";
const DEFAULT_CONTENT_BASE: &str = "https://content.dropboxapi.com";
const DEFAULT_AUTH_URL: &str = "https://www.dropbox.com/oauth2/authorize";
const DEFAULT_TOKEN_URL: &str = "https://api.dropboxapi.com/oauth2/token";

/// Dropbox app key (client id). Register a **scoped** app at
/// <https://www.dropbox.com/developers/apps>, set its redirect URI to
/// `http://localhost:53682/`, enable the `account_info.read`,
/// `files.metadata.read`, `files.content.read`, and `files.content.write`
/// scopes, then paste the App key here (or set `GHOSTFTP_DROPBOX_APP_KEY`). PKCE
/// means no app secret is needed.
const DROPBOX_APP_KEY: &str = "";

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// The OAuth config for Dropbox. Endpoints fall back to env overrides so the
/// integration test can point the whole flow at a local mock.
pub fn dropbox_config() -> OAuthConfig {
    OAuthConfig {
        client_id: env_or("GHOSTFTP_DROPBOX_APP_KEY", DROPBOX_APP_KEY),
        auth_url: env_or("GHOSTFTP_DROPBOX_AUTH_URL", DEFAULT_AUTH_URL),
        token_url: env_or("GHOSTFTP_DROPBOX_TOKEN_URL", DEFAULT_TOKEN_URL),
        scopes: vec![
            "account_info.read".into(),
            "files.metadata.read".into(),
            "files.content.read".into(),
            "files.content.write".into(),
        ],
        // Offline access ⇒ Dropbox returns a long-lived refresh token.
        extra_auth_params: vec![("token_access_type".into(), "offline".into())],
    }
}

/// A live Dropbox connection. Stateless HTTP like the object/webdav backends —
/// every op is a request carrying a bearer access token, which is refreshed
/// (and re-persisted to the keychain) transparently when it nears expiry or a
/// request comes back 401.
pub struct DropboxSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    pub content_base: String,
    config: OAuthConfig,
    tokens: Mutex<TokenSet>,
}

impl DropboxSession {
    /// A valid access token, refreshing + persisting first if it's near expiry.
    pub async fn access_token(&self) -> Result<String> {
        let mut guard = self.tokens.lock().await;
        if oauth::is_expired(&guard) {
            let rt = guard
                .refresh_token
                .clone()
                .ok_or_else(|| anyhow!("no refresh token — re-authorize this Dropbox connection"))?;
            let fresh = oauth::refresh(&self.config, &rt).await?;
            *guard = fresh;
            oauth::store_tokens(DROPBOX_SERVICE, &self.profile.id, &guard)?;
        }
        Ok(guard.access_token.clone())
    }

    /// Force a refresh (used after a 401 on a token we thought was still valid).
    pub async fn force_refresh(&self) -> Result<()> {
        let mut guard = self.tokens.lock().await;
        let rt = guard
            .refresh_token
            .clone()
            .ok_or_else(|| anyhow!("no refresh token — re-authorize this Dropbox connection"))?;
        let fresh = oauth::refresh(&self.config, &rt).await?;
        *guard = fresh;
        oauth::store_tokens(DROPBOX_SERVICE, &self.profile.id, &guard)?;
        Ok(())
    }

    /// POST an RPC endpoint (`/2/…`) with a JSON body, retrying once on 401 after
    /// a forced refresh. Returns the parsed JSON response.
    pub async fn rpc(&self, endpoint: &str, body: serde_json::Value) -> Result<serde_json::Value> {
        let url = format!("{}{endpoint}", self.api_base);
        for attempt in 0..2 {
            let token = self.access_token().await?;
            let resp = self
                .client
                .post(&url)
                .bearer_auth(&token)
                .json(&body)
                .send()
                .await
                .with_context(|| format!("dropbox {endpoint}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                self.force_refresh().await?;
                continue;
            }
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            if !status.is_success() {
                return Err(anyhow!(
                    "dropbox {endpoint} failed ({}): {text}",
                    status.as_u16()
                ));
            }
            // No-arg endpoints (get_current_account) can return an empty body.
            if text.trim().is_empty() {
                return Ok(serde_json::Value::Null);
            }
            return serde_json::from_str(&text)
                .with_context(|| format!("parse dropbox {endpoint} response"));
        }
        unreachable!("rpc loop always returns within 2 attempts")
    }

    /// A content-API GET (download): POST with the arg in the `Dropbox-API-Arg`
    /// header, retrying once on 401. Returns the streaming success response.
    pub async fn content_get(&self, endpoint: &str, arg: &str) -> Result<reqwest::Response> {
        let url = format!("{}{endpoint}", self.content_base);
        for attempt in 0..2 {
            let token = self.access_token().await?;
            let resp = self
                .client
                .post(&url)
                .bearer_auth(&token)
                .header("Dropbox-API-Arg", arg)
                .send()
                .await
                .with_context(|| format!("dropbox {endpoint}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                self.force_refresh().await?;
                continue;
            }
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow!("dropbox {endpoint} failed ({code}): {text}"));
            }
            return Ok(resp);
        }
        unreachable!("content_get loop always returns within 2 attempts")
    }

    /// The account's email (falling back to display name) for the connection label.
    pub async fn account_label(&self) -> Result<String> {
        let v = self
            .rpc("/2/users/get_current_account", serde_json::Value::Null)
            .await?;
        let email = v.get("email").and_then(|x| x.as_str()).unwrap_or("");
        if !email.is_empty() {
            return Ok(email.to_string());
        }
        let name = v
            .get("name")
            .and_then(|n| n.get("display_name"))
            .and_then(|x| x.as_str())
            .unwrap_or("Dropbox");
        Ok(name.to_string())
    }

    /// Whether a path exists (via get_metadata).
    pub async fn exists(&self, dbx_path: &str) -> bool {
        self.rpc(
            "/2/files/get_metadata",
            serde_json::json!({ "path": dbx_path }),
        )
        .await
        .is_ok()
    }

    /// The size of a file at `dbx_path`, or 0 if unknown.
    pub async fn size(&self, dbx_path: &str) -> u64 {
        self.rpc(
            "/2/files/get_metadata",
            serde_json::json!({ "path": dbx_path }),
        )
        .await
        .ok()
        .and_then(|v| v.get("size").and_then(|s| s.as_u64()))
        .unwrap_or(0)
    }
}

/// Open a Dropbox session from a profile whose tokens were stored at authorize
/// time. Errors clearly if the connection was never authorized.
pub async fn dropbox_connect(profile: &ConnectionProfile) -> Result<DropboxSession> {
    let tokens = oauth::load_tokens(DROPBOX_SERVICE, &profile.id)?.ok_or_else(|| {
        anyhow!(
            "This Dropbox connection isn't authorized yet. Open it in the connection \
             editor and click “Connect with Dropbox”."
        )
    })?;

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building Dropbox HTTP client")?;

    Ok(DropboxSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or("GHOSTFTP_DROPBOX_API_BASE", DEFAULT_API_BASE),
        content_base: env_or("GHOSTFTP_DROPBOX_CONTENT_BASE", DEFAULT_CONTENT_BASE),
        config: dropbox_config(),
        tokens: Mutex::new(tokens),
    })
}
