use crate::oauth::{self, OAuthConfig, RefreshingToken};
use crate::profiles::ConnectionProfile;
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method};
use serde_json::Value;
use std::time::Duration;
use uuid::Uuid;

/// Keychain service name for OneDrive tokens (keyed by profile id).
pub const ONEDRIVE_SERVICE: &str = "ghostftp-onedrive";

const DEFAULT_GRAPH_BASE: &str = "https://graph.microsoft.com/v1.0";
const DEFAULT_AUTH_URL: &str =
    "https://login.microsoftonline.com/common/oauth2/v2.0/authorize";
const DEFAULT_TOKEN_URL: &str = "https://login.microsoftonline.com/common/oauth2/v2.0/token";

/// OneDrive (Microsoft) app client id. Register an app at
/// <https://entra.microsoft.com> (App registrations), add a **Mobile & desktop**
/// redirect `http://localhost:53682/`, enable public-client flows, and grant the
/// delegated `Files.ReadWrite`, `offline_access`, and `User.Read` scopes. Paste
/// the Application (client) id here or set `GHOSTFTP_ONEDRIVE_CLIENT_ID`. PKCE ⇒ no
/// secret.
const ONEDRIVE_CLIENT_ID: &str = "";

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// OneDrive OAuth config. Endpoints fall back to env overrides for the mock test.
pub fn onedrive_config() -> OAuthConfig {
    OAuthConfig {
        client_id: env_or("GHOSTFTP_ONEDRIVE_CLIENT_ID", ONEDRIVE_CLIENT_ID),
        auth_url: env_or("GHOSTFTP_ONEDRIVE_AUTH_URL", DEFAULT_AUTH_URL),
        token_url: env_or("GHOSTFTP_ONEDRIVE_TOKEN_URL", DEFAULT_TOKEN_URL),
        scopes: vec![
            "Files.ReadWrite".into(),
            "offline_access".into(),
            "User.Read".into(),
        ],
        extra_auth_params: vec![],
    }
}

/// A live OneDrive connection over Microsoft Graph. Stateless HTTP; the embedded
/// [`RefreshingToken`] keeps the bearer token fresh.
pub struct OneDriveSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub graph_base: String,
    token: RefreshingToken,
}

impl OneDriveSession {
    pub async fn access_token(&self) -> Result<String> {
        self.token.access_token().await
    }

    pub async fn force_refresh(&self) -> Result<()> {
        self.token.force_refresh().await
    }

    /// Send a Graph request (bearer + one 401→refresh retry), returning the raw
    /// response so callers can stream downloads or parse JSON.
    pub async fn send(
        &self,
        method: Method,
        path: &str,
        body: Option<&Value>,
    ) -> Result<reqwest::Response> {
        // A `@odata.nextLink` is already absolute; item paths are relative.
        let url = if path.starts_with("http") {
            path.to_string()
        } else {
            format!("{}{path}", self.graph_base)
        };
        let mut attempt = 0;
        loop {
            let token = self.token.access_token().await?;
            let mut rb = self.client.request(method.clone(), &url).bearer_auth(&token);
            if let Some(b) = body {
                rb = rb.json(b);
            }
            let resp = rb.send().await.with_context(|| format!("graph {path}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                attempt += 1;
                self.force_refresh().await?;
                continue;
            }
            return Ok(resp);
        }
    }

    /// Send a Graph request and parse the JSON response, erroring on non-2xx.
    pub async fn rpc(&self, method: Method, path: &str, body: Option<&Value>) -> Result<Value> {
        let resp = self.send(method, path, body).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!("graph {path} failed ({}): {text}", status.as_u16()));
        }
        if text.trim().is_empty() {
            return Ok(Value::Null);
        }
        serde_json::from_str(&text).with_context(|| format!("parse graph {path} response"))
    }

    /// A streaming GET (download), erroring on non-2xx. `path` may be a Graph
    /// item path (`/me/…`) or an absolute URL (a `@odata.nextLink`).
    pub async fn get_stream(&self, path: &str) -> Result<reqwest::Response> {
        let resp = self.send(Method::GET, path, None).await?;
        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("graph GET {path} failed ({code}): {text}"));
        }
        Ok(resp)
    }

    /// The account's email/UPN (falling back to display name) for the label.
    pub async fn account_label(&self) -> Result<String> {
        let v = self.rpc(Method::GET, "/me", None).await?;
        for key in ["userPrincipalName", "mail", "displayName"] {
            if let Some(s) = v.get(key).and_then(|x| x.as_str()).filter(|s| !s.is_empty()) {
                return Ok(s.to_string());
            }
        }
        Ok("OneDrive".to_string())
    }

    /// Whether the item at a Graph item-ref exists.
    pub async fn exists(&self, item_ref: &str) -> bool {
        self.rpc(Method::GET, item_ref, None).await.is_ok()
    }

    /// Size of the item at a Graph item-ref, or 0.
    pub async fn size(&self, item_ref: &str) -> u64 {
        self.rpc(Method::GET, item_ref, None)
            .await
            .ok()
            .and_then(|v| v.get("size").and_then(|s| s.as_u64()))
            .unwrap_or(0)
    }
}

pub async fn onedrive_connect(profile: &ConnectionProfile) -> Result<OneDriveSession> {
    let tokens = oauth::load_tokens(ONEDRIVE_SERVICE, &profile.id)?.ok_or_else(|| {
        anyhow!(
            "This OneDrive connection isn't authorized yet. Open it in the connection \
             editor and click “Connect with OneDrive”."
        )
    })?;

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building OneDrive HTTP client")?;

    Ok(OneDriveSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        graph_base: env_or("GHOSTFTP_ONEDRIVE_GRAPH_BASE", DEFAULT_GRAPH_BASE),
        token: RefreshingToken::new(ONEDRIVE_SERVICE, &profile.id, onedrive_config(), tokens),
    })
}
