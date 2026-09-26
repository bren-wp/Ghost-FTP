//! Shared OAuth 2.0 infrastructure for the consumer-cloud backends (Plan 5).
//!
//! Implements the loopback + PKCE authorization-code flow (RFC 8252 / 7636) with
//! no external OAuth crate: a fixed-port local HTTP listener catches the redirect,
//! the code is exchanged for tokens against a configurable token endpoint, and
//! refresh is a second form POST. Dropbox is the first consumer; OneDrive / Drive
//! / Box reuse the same helper by supplying a different `OAuthConfig`.
//!
//! Tokens live in the OS keychain (`keyring`), never in the plaintext profile
//! JSON — refresh tokens are long-lived bearer credentials.

use anyhow::{anyhow, Context, Result};
use base64::Engine as _;
use serde::{Deserialize, Serialize};
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use url::Url;

/// Fixed loopback redirect port. Deterministic so the maintainer registers
/// exactly `http://localhost:53682/` as the app's redirect URI (the same port
/// rclone uses, for the same reason). See the provider app-setup notes.
pub const REDIRECT_PORT: u16 = 53682;

/// Everything provider-specific about an OAuth flow. Endpoints are fields (not
/// constants) so tests can point them at a local mock.
#[derive(Clone)]
pub struct OAuthConfig {
    pub client_id: String,
    pub auth_url: String,
    pub token_url: String,
    pub scopes: Vec<String>,
    /// Extra query params on the authorize URL (e.g. Dropbox's
    /// `token_access_type=offline` to get a refresh token).
    pub extra_auth_params: Vec<(String, String)>,
}

/// Tokens returned by an exchange/refresh. `expires_at` is an absolute Unix
/// timestamp (seconds); 0 means "unknown, treat as expired".
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct TokenSet {
    pub access_token: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    pub expires_at: i64,
}

#[derive(Deserialize)]
struct TokenResponse {
    access_token: String,
    #[serde(default)]
    refresh_token: Option<String>,
    #[serde(default)]
    expires_in: Option<i64>,
}

fn now_secs() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

/// A PKCE `code_verifier` (unreserved chars) and its S256 `code_challenge`.
pub fn pkce_pair() -> (String, String) {
    // 96 hex chars of CSPRNG entropy (uuid v4 is getrandom-backed) — well within
    // the 43–128 unreserved-character verifier range.
    let verifier = format!(
        "{}{}{}",
        uuid::Uuid::new_v4().simple(),
        uuid::Uuid::new_v4().simple(),
        uuid::Uuid::new_v4().simple()
    );
    let challenge = pkce_challenge(&verifier);
    (verifier, challenge)
}

/// S256 challenge: base64url-no-pad of SHA-256(verifier).
pub fn pkce_challenge(verifier: &str) -> String {
    use sha2::{Digest, Sha256};
    let digest = Sha256::digest(verifier.as_bytes());
    base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(digest)
}

fn redirect_uri() -> String {
    format!("http://localhost:{REDIRECT_PORT}/")
}

/// Build the provider authorize URL for a given challenge.
pub fn authorize_url(config: &OAuthConfig, challenge: &str) -> Result<Url> {
    let mut url = Url::parse(&config.auth_url).context("parse auth_url")?;
    {
        let mut q = url.query_pairs_mut();
        q.append_pair("client_id", &config.client_id);
        q.append_pair("response_type", "code");
        q.append_pair("redirect_uri", &redirect_uri());
        q.append_pair("code_challenge", challenge);
        q.append_pair("code_challenge_method", "S256");
        if !config.scopes.is_empty() {
            q.append_pair("scope", &config.scopes.join(" "));
        }
        for (k, v) in &config.extra_auth_params {
            q.append_pair(k, v);
        }
    }
    Ok(url)
}

/// Run the full interactive flow: start the loopback listener, open the browser,
/// wait for the redirect, and exchange the code. Returns the tokens plus the raw
/// token-endpoint JSON (providers stash extras there, e.g. Dropbox `account_id`).
pub async fn authorize_loopback(config: &OAuthConfig) -> Result<(TokenSet, serde_json::Value)> {
    if config.client_id.is_empty() {
        return Err(anyhow!(
            "no OAuth client id configured — register the app and set its key \
             (see the provider setup note)"
        ));
    }
    let (verifier, challenge) = pkce_pair();
    let auth_url = authorize_url(config, &challenge)?;

    // Bind the fixed loopback port *before* opening the browser so the redirect
    // can't race an unbound socket.
    let listener = tokio::net::TcpListener::bind(("127.0.0.1", REDIRECT_PORT))
        .await
        .with_context(|| format!("bind loopback :{REDIRECT_PORT} for OAuth redirect"))?;

    open_url(auth_url.as_str())?;

    // Wait (bounded) for the browser redirect and pull the ?code= out of it.
    let code = tokio::time::timeout(Duration::from_secs(300), accept_code(&listener))
        .await
        .map_err(|_| anyhow!("timed out waiting for the authorization redirect"))??;

    let tokens_raw = exchange_code(config, &code, &verifier).await?;
    let tokens = token_set_from(&tokens_raw)?;
    Ok((tokens, tokens_raw))
}

/// Accept a single redirect request, answer with a friendly page, and return the
/// `code` (or surface the `error`).
async fn accept_code(listener: &tokio::net::TcpListener) -> Result<String> {
    loop {
        let (mut stream, _) = listener.accept().await.context("accept OAuth redirect")?;
        let mut buf = [0u8; 4096];
        let n = stream.read(&mut buf).await.unwrap_or(0);
        let req = String::from_utf8_lossy(&buf[..n]);
        // First line: `GET /?code=... HTTP/1.1`
        let path = req
            .lines()
            .next()
            .and_then(|l| l.split_whitespace().nth(1))
            .unwrap_or("/");
        let parsed = Url::parse(&format!("http://localhost{path}")).ok();
        let (code, error) = parsed
            .as_ref()
            .map(|u| {
                let mut code = None;
                let mut error = None;
                for (k, v) in u.query_pairs() {
                    match k.as_ref() {
                        "code" => code = Some(v.into_owned()),
                        "error" => error = Some(v.into_owned()),
                        _ => {}
                    }
                }
                (code, error)
            })
            .unwrap_or((None, None));

        // Ignore stray hits (favicon, etc.) that carry neither code nor error.
        if code.is_none() && error.is_none() {
            let _ = respond(&mut stream, "Waiting for authorization…").await;
            continue;
        }

        let body = if error.is_some() {
            "Authorization failed. You can close this window and return to Ghost FTP."
        } else {
            "Ghost FTP is now connected. You can close this window."
        };
        let _ = respond(&mut stream, body).await;

        if let Some(e) = error {
            return Err(anyhow!("authorization was denied: {e}"));
        }
        return code.ok_or_else(|| anyhow!("redirect carried no authorization code"));
    }
}

async fn respond(stream: &mut tokio::net::TcpStream, body: &str) -> Result<()> {
    let html = format!(
        "<!doctype html><html><head><meta charset=utf-8><title>Ghost FTP</title></head>\
         <body style=\"font-family:system-ui;padding:3rem;text-align:center\">\
         <h2>{body}</h2></body></html>"
    );
    let resp = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n\
         Content-Length: {}\r\nConnection: close\r\n\r\n{}",
        html.len(),
        html
    );
    stream.write_all(resp.as_bytes()).await?;
    stream.flush().await?;
    Ok(())
}

/// Exchange an authorization code + PKCE verifier for tokens. Returns the raw
/// JSON so provider-specific fields survive.
pub async fn exchange_code(
    config: &OAuthConfig,
    code: &str,
    verifier: &str,
) -> Result<serde_json::Value> {
    let client = reqwest::Client::new();
    let resp = client
        .post(&config.token_url)
        .form(&[
            ("grant_type", "authorization_code"),
            ("code", code),
            ("code_verifier", verifier),
            ("client_id", &config.client_id),
            ("redirect_uri", &redirect_uri()),
        ])
        .send()
        .await
        .context("token exchange request")?;
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    if !status.is_success() {
        return Err(anyhow!("token exchange failed ({}): {text}", status.as_u16()));
    }
    serde_json::from_str(&text).context("parse token response")
}

/// Refresh an access token from a refresh token. Providers usually return only a
/// new access token, so the caller keeps the existing refresh token.
pub async fn refresh(config: &OAuthConfig, refresh_token: &str) -> Result<TokenSet> {
    let client = reqwest::Client::new();
    let resp = client
        .post(&config.token_url)
        .form(&[
            ("grant_type", "refresh_token"),
            ("refresh_token", refresh_token),
            ("client_id", &config.client_id),
        ])
        .send()
        .await
        .context("token refresh request")?;
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    if !status.is_success() {
        return Err(anyhow!("token refresh failed ({}): {text}", status.as_u16()));
    }
    let raw: serde_json::Value = serde_json::from_str(&text).context("parse refresh response")?;
    let mut set = token_set_from(&raw)?;
    // Carry forward the refresh token if the response omitted one.
    if set.refresh_token.is_none() {
        set.refresh_token = Some(refresh_token.to_string());
    }
    Ok(set)
}

fn token_set_from(raw: &serde_json::Value) -> Result<TokenSet> {
    let parsed: TokenResponse =
        serde_json::from_value(raw.clone()).context("token response missing access_token")?;
    let expires_at = parsed
        .expires_in
        .map(|s| now_secs() + s.max(0))
        .unwrap_or(0);
    Ok(TokenSet {
        access_token: parsed.access_token,
        refresh_token: parsed.refresh_token,
        expires_at,
    })
}

/// True when a token is expired or within 60s of it (or its expiry is unknown).
pub fn is_expired(tokens: &TokenSet) -> bool {
    tokens.expires_at == 0 || tokens.expires_at <= now_secs() + 60
}

fn open_url(url: &str) -> Result<()> {
    #[cfg(target_os = "windows")]
    let mut cmd = {
        let mut c = std::process::Command::new("rundll32");
        c.args(["url.dll,FileProtocolHandler", url]);
        c
    };
    #[cfg(unix)]
    let mut cmd = {
        let mut c = std::process::Command::new("xdg-open");
        c.arg(url);
        c
    };
    cmd.spawn().context("open the system browser")?;
    Ok(())
}

// ---- Token storage (OS keychain) ----

/// Persist a provider's tokens for a profile in the OS keychain.
pub fn store_tokens(service: &str, profile_id: &str, tokens: &TokenSet) -> Result<()> {
    let entry = keyring::Entry::new(service, profile_id).context("open keychain entry")?;
    let json = serde_json::to_string(tokens).context("serialize tokens")?;
    entry.set_password(&json).context("write tokens to keychain")?;
    Ok(())
}

/// Load a profile's tokens, or `None` if it was never authorized.
pub fn load_tokens(service: &str, profile_id: &str) -> Result<Option<TokenSet>> {
    let entry = keyring::Entry::new(service, profile_id).context("open keychain entry")?;
    match entry.get_password() {
        Ok(json) => Ok(Some(serde_json::from_str(&json).context("parse stored tokens")?)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(e) => Err(anyhow!("read tokens from keychain: {e}")),
    }
}

/// Remove a profile's tokens (best-effort — a missing entry is not an error).
pub fn delete_tokens(service: &str, profile_id: &str) {
    if let Ok(entry) = keyring::Entry::new(service, profile_id) {
        let _ = entry.delete_credential();
    }
}

/// Shared token holder for the OAuth cloud backends: hands out a valid access
/// token, refreshing (proactively on near-expiry, or on a forced 401 retry) and
/// re-persisting to the keychain. Each provider session embeds one so the
/// refresh logic lives in exactly one place.
pub struct RefreshingToken {
    service: String,
    profile_id: String,
    config: OAuthConfig,
    tokens: tokio::sync::Mutex<TokenSet>,
}

impl RefreshingToken {
    pub fn new(service: &str, profile_id: &str, config: OAuthConfig, tokens: TokenSet) -> Self {
        Self {
            service: service.to_string(),
            profile_id: profile_id.to_string(),
            config,
            tokens: tokio::sync::Mutex::new(tokens),
        }
    }

    /// A valid access token, refreshing + persisting first if it's near expiry.
    pub async fn access_token(&self) -> Result<String> {
        let mut guard = self.tokens.lock().await;
        if is_expired(&guard) {
            self.refresh_locked(&mut guard).await?;
        }
        Ok(guard.access_token.clone())
    }

    /// Force a refresh (used after a 401 on a token we thought was still valid).
    pub async fn force_refresh(&self) -> Result<()> {
        let mut guard = self.tokens.lock().await;
        self.refresh_locked(&mut guard).await
    }

    async fn refresh_locked(&self, guard: &mut TokenSet) -> Result<()> {
        let rt = guard
            .refresh_token
            .clone()
            .ok_or_else(|| anyhow!("no refresh token — re-authorize this connection"))?;
        let fresh = refresh(&self.config, &rt).await?;
        *guard = fresh;
        store_tokens(&self.service, &self.profile_id, guard)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pkce_challenge_is_s256_base64url() {
        // RFC 7636 Appendix B reference vector.
        let verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
        assert_eq!(
            pkce_challenge(verifier),
            "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
        );
    }

    #[test]
    fn pkce_pair_lengths() {
        let (v, c) = pkce_pair();
        assert_eq!(v.len(), 96);
        assert!(!c.is_empty());
        assert_eq!(pkce_challenge(&v), c);
    }

    #[test]
    fn authorize_url_has_pkce_and_redirect() {
        let cfg = OAuthConfig {
            client_id: "abc123".into(),
            auth_url: "https://example.invalid/oauth2/authorize".into(),
            token_url: "https://example.invalid/oauth2/token".into(),
            scopes: vec!["files.read".into(), "files.write".into()],
            extra_auth_params: vec![("token_access_type".into(), "offline".into())],
        };
        let url = authorize_url(&cfg, "CHAL").unwrap();
        let q: std::collections::HashMap<_, _> = url.query_pairs().into_owned().collect();
        assert_eq!(q["client_id"], "abc123");
        assert_eq!(q["response_type"], "code");
        assert_eq!(q["code_challenge"], "CHAL");
        assert_eq!(q["code_challenge_method"], "S256");
        assert_eq!(q["scope"], "files.read files.write");
        assert_eq!(q["token_access_type"], "offline");
        assert_eq!(q["redirect_uri"], "http://localhost:53682/");
    }

    #[test]
    fn token_set_computes_expiry() {
        let raw = serde_json::json!({
            "access_token": "at", "refresh_token": "rt", "expires_in": 3600
        });
        let set = token_set_from(&raw).unwrap();
        assert_eq!(set.access_token, "at");
        assert_eq!(set.refresh_token.as_deref(), Some("rt"));
        assert!(set.expires_at > now_secs() + 3000);
        assert!(!is_expired(&set));

        let no_exp = token_set_from(&serde_json::json!({"access_token":"x"})).unwrap();
        assert_eq!(no_exp.expires_at, 0);
        assert!(is_expired(&no_exp)); // unknown expiry ⇒ treat as expired
    }

    /// Round-trips tokens through the real OS keychain. `#[ignore]`d because it
    /// touches the credential store (unavailable on headless CI); run locally
    /// with `cargo test -p ghostftp -- --ignored keychain`.
    #[test]
    #[ignore = "touches the OS keychain"]
    fn keychain_roundtrip() {
        let svc = "ghostftp-oauth-test";
        let id = "roundtrip-profile";
        let tokens = TokenSet {
            access_token: "AT".into(),
            refresh_token: Some("RT".into()),
            expires_at: 1_234_567_890,
        };
        store_tokens(svc, id, &tokens).expect("store");
        let loaded = load_tokens(svc, id).expect("load").expect("present");
        assert_eq!(loaded, tokens);
        delete_tokens(svc, id);
        assert!(load_tokens(svc, id).expect("load after delete").is_none());
    }
}
