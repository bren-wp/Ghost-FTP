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
#[derive(Clone, Serialize, Deserialize, PartialEq)]
pub struct TokenSet {
    pub access_token: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    pub expires_at: i64,
    /// Public OAuth application ID needed to refresh user-authorized tokens
    /// after restart; absent for existing keychain entries (backward-compatible).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub client_id: Option<String>,
}

// Token strings must never be accidentally printed by debugging, panic or trace logs.
impl std::fmt::Debug for TokenSet {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("TokenSet")
            .field("access_token", &"[REDACTED]")
            .field(
                "refresh_token",
                &self.refresh_token.as_ref().map(|_| "[REDACTED]"),
            )
            .field("expires_at", &self.expires_at)
            .field(
                "client_id",
                &self.client_id.as_ref().map(|_| "[configured]"),
            )
            .finish()
    }
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

/// Build the provider authorize URL for a given PKCE challenge and per-login
/// CSRF state. Never start an authorization flow with an empty state.
pub fn authorize_url(config: &OAuthConfig, challenge: &str, state: &str) -> Result<Url> {
    if state.is_empty() {
        return Err(anyhow!("OAuth state is required"));
    }
    // The browser and PKCE token exchange must never downgrade to a public
    // HTTP origin. Loopback HTTP remains supported for isolated mock tests.
    let mut url =
        crate::session::http_throttle::trusted_bearer_url(&config.auth_url, &config.auth_url)?;
    // Do not permit endpoint URLs to pre-seed conflicting security-critical
    // OAuth parameters ahead of the values generated for this login attempt.
    for (name, _) in url.query_pairs() {
        anyhow::ensure!(
            !matches!(
                name.as_ref(),
                "client_id"
                    | "response_type"
                    | "redirect_uri"
                    | "state"
                    | "code_challenge"
                    | "code_challenge_method"
            ),
            "OAuth authorization URL has a conflicting reserved parameter"
        );
    }
    {
        let mut q = url.query_pairs_mut();
        q.append_pair("client_id", &config.client_id);
        q.append_pair("response_type", "code");
        q.append_pair("redirect_uri", &redirect_uri());
        q.append_pair("code_challenge", challenge);
        q.append_pair("code_challenge_method", "S256");
        q.append_pair("state", state);
        if !config.scopes.is_empty() {
            q.append_pair("scope", &config.scopes.join(" "));
        }
        for (k, v) in &config.extra_auth_params {
            // Provider extras must not append duplicate security-critical keys.
            if matches!(
                k.as_str(),
                "client_id"
                    | "response_type"
                    | "redirect_uri"
                    | "state"
                    | "code_challenge"
                    | "code_challenge_method"
            ) {
                return Err(anyhow!("provider overrides a reserved OAuth parameter"));
            }
            q.append_pair(k, v);
        }
    }
    Ok(url)
}

/// Allow a user-supplied public OAuth client ID when this build has no
/// centrally registered ID. Reject URLs, tokens, whitespace and oversized
/// input: the ID is not a secret and is stored with tokens in the OS vault.
pub fn use_client_id(config: &mut OAuthConfig, candidate: Option<&str>) -> Result<()> {
    if let Some(candidate) = candidate {
        let id = candidate.trim();
        anyhow::ensure!(
            !id.is_empty()
                && id.len() <= 256
                && id
                    .bytes()
                    .all(|c| c.is_ascii_alphanumeric() || matches!(c, b'-' | b'_' | b'.' | b'~')),
            "OAuth application ID has invalid characters or length"
        );
        config.client_id = id.to_string();
    }
    anyhow::ensure!(
        !config.client_id.is_empty(),
        "This provider needs a registered OAuth application ID. Enter its public Client ID in Advanced setup, or use a build with a preconfigured ID."
    );
    Ok(())
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
    // Two independent v4 UUIDs provide 244 random bits, bound to this one
    // authorization attempt. This is distinct from the PKCE code verifier.
    let state = format!(
        "{}{}",
        uuid::Uuid::new_v4().simple(),
        uuid::Uuid::new_v4().simple()
    );
    let auth_url = authorize_url(config, &challenge, &state)?;

    // Bind the fixed loopback port *before* opening the browser so the redirect
    // can't race an unbound socket.
    let listener = tokio::net::TcpListener::bind(("127.0.0.1", REDIRECT_PORT))
        .await
        .with_context(|| format!("bind loopback :{REDIRECT_PORT} for OAuth redirect"))?;

    open_url(auth_url.as_str())?;

    // Wait (bounded) for the browser redirect and pull the ?code= out of it.
    let code = tokio::time::timeout(Duration::from_secs(300), accept_code(&listener, &state))
        .await
        .map_err(|_| anyhow!("timed out waiting for the authorization redirect"))??;

    let tokens_raw = exchange_code(config, &code, &verifier).await?;
    let mut tokens = token_set_from(&tokens_raw)?;
    // Persist only the public client ID alongside OS-keychain tokens. The
    // provider secret is never requested or sent over IPC.
    tokens.client_id = Some(config.client_id.clone());
    Ok((tokens, tokens_raw))
}

/// Interpret only GET / callbacks with exactly one matching CSRF state.
/// Unrelated local HTTP requests cannot complete or cancel a login attempt.
#[derive(Debug, PartialEq, Eq)]
enum OAuthCallback {
    Ignore,
    Authorized(String),
    Denied,
}

fn parse_oauth_callback(request_line: &str, expected_state: &str) -> OAuthCallback {
    let mut parts = request_line.split_whitespace();
    if parts.next() != Some("GET") {
        return OAuthCallback::Ignore;
    }
    let Some(target) = parts.next() else {
        return OAuthCallback::Ignore;
    };
    if parts.next() != Some("HTTP/1.1") || parts.next().is_some() || !target.starts_with("/?") {
        return OAuthCallback::Ignore;
    }
    let Ok(url) = Url::parse(&format!("http://localhost:{REDIRECT_PORT}{target}")) else {
        return OAuthCallback::Ignore;
    };
    if url.path() != "/" || url.host_str() != Some("localhost") {
        return OAuthCallback::Ignore;
    }

    let mut received_state: Option<String> = None;
    let mut code: Option<String> = None;
    let mut error: Option<String> = None;
    for (key, value) in url.query_pairs() {
        match key.as_ref() {
            "state" => {
                if received_state.replace(value.into_owned()).is_some() {
                    return OAuthCallback::Ignore;
                }
            }
            "code" => {
                if code.replace(value.into_owned()).is_some() {
                    return OAuthCallback::Ignore;
                }
            }
            "error" => {
                let previous_error = error.replace(value.into_owned());
                if previous_error.is_some() {
                    return OAuthCallback::Ignore;
                }
            }
            _ => {}
        }
    }

    if received_state.as_deref() != Some(expected_state) {
        return OAuthCallback::Ignore;
    }
    match (code, error) {
        (Some(code), None) if !code.is_empty() => OAuthCallback::Authorized(code),
        (None, Some(error)) if !error.is_empty() => OAuthCallback::Denied,
        _ => OAuthCallback::Ignore,
    }
}

/// A wrong/missing state is ignored rather than reported as authentication
/// success or failure. The real provider callback may still arrive afterward.
async fn accept_code(listener: &tokio::net::TcpListener, expected_state: &str) -> Result<String> {
    loop {
        let (mut stream, _) = listener.accept().await.context("accept OAuth redirect")?;
        let mut buf = [0u8; 4096];
        let n = match tokio::time::timeout(Duration::from_secs(5), stream.read(&mut buf)).await {
            Ok(Ok(n)) => n,
            _ => continue, // a stalled or malformed local socket cannot lock up the flow
        };
        let Ok(request) = std::str::from_utf8(&buf[..n]) else {
            continue;
        };
        let callback = request
            .lines()
            .next()
            .map(|line| parse_oauth_callback(line, expected_state))
            .unwrap_or(OAuthCallback::Ignore);

        match callback {
            OAuthCallback::Ignore => {
                let _ = respond(&mut stream, "Authorization callback ignored.").await;
            }
            OAuthCallback::Denied => {
                let _ = respond(
                    &mut stream,
                    "Authorization failed. You can close this window and return to Ghost FTP.",
                )
                .await;
                // Never include untrusted provider error text in logs or UI.
                return Err(anyhow!("authorization was denied by the provider"));
            }
            OAuthCallback::Authorized(code) => {
                let _ = respond(
                    &mut stream,
                    "Ghost FTP is now connected. You can close this window.",
                )
                .await;
                return Ok(code);
            }
        }
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
    let target =
        crate::session::http_throttle::trusted_bearer_url(&config.token_url, &config.token_url)?;
    let client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::none())
        .connect_timeout(Duration::from_secs(10))
        .timeout(Duration::from_secs(30))
        .build()
        .context("initialize OAuth token client")?;
    let resp = client
        .post(target)
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
    if !status.is_success() {
        return Err(anyhow!(
            "token exchange failed with HTTP {}",
            status.as_u16()
        ));
    }
    crate::session::http_throttle::bounded_cloud_json(resp)
        .await
        .context("parse bounded token response")
}

/// Refresh an access token from a refresh token. Providers usually return only a
/// new access token, so the caller keeps the existing refresh token.
pub async fn refresh(config: &OAuthConfig, refresh_token: &str) -> Result<TokenSet> {
    let target =
        crate::session::http_throttle::trusted_bearer_url(&config.token_url, &config.token_url)?;
    let client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::none())
        .connect_timeout(Duration::from_secs(10))
        .timeout(Duration::from_secs(30))
        .build()
        .context("initialize OAuth token client")?;
    let resp = client
        .post(target)
        .form(&[
            ("grant_type", "refresh_token"),
            ("refresh_token", refresh_token),
            ("client_id", &config.client_id),
        ])
        .send()
        .await
        .context("token refresh request")?;
    let status = resp.status();
    if !status.is_success() {
        return Err(anyhow!(
            "token refresh failed with HTTP {}",
            status.as_u16()
        ));
    }
    let raw = crate::session::http_throttle::bounded_cloud_json(resp)
        .await
        .context("parse bounded token refresh response")?;
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
        client_id: None,
    })
}

/// True when a token is expired or within 60s of it (or its expiry is unknown).
pub fn is_expired(tokens: &TokenSet) -> bool {
    tokens.expires_at == 0 || tokens.expires_at <= now_secs() + 60
}

fn open_url(url: &str) -> Result<()> {
    #[cfg(target_os = "windows")]
    let mut cmd = {
        let mut command = std::process::Command::new("rundll32");
        crate::windows_process::hide_console(&mut command);
        command.args(["url.dll,FileProtocolHandler", url]);
        command
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
    let json = serde_json::to_string(tokens).context("serialize tokens")?;
    crate::credentials::vault_set(service, profile_id, &json)
        .context("write tokens to OS credential store")
}

/// Load a profile's tokens, or `None` if it was never authorized.
pub fn load_tokens(service: &str, profile_id: &str) -> Result<Option<TokenSet>> {
    match crate::credentials::vault_get(service, profile_id)? {
        Some(json) => Ok(Some(
            serde_json::from_str(&json).context("parse stored tokens")?,
        )),
        None => Ok(None),
    }
}

/// Do not announce token revocation when an OS vault is locked. Retain the
/// old entry if deletion fails; the caller must surface the failure.
pub fn delete_tokens_checked(service: &str, profile_id: &str) -> Result<()> {
    crate::credentials::vault_delete(service, profile_id)
        .context("OS credential store refused OAuth token deletion")
}

/// Retained for existing call sites that cannot return Result. New security
/// sensitive delete paths must use delete_tokens_checked instead.
pub fn delete_tokens(service: &str, profile_id: &str) {
    let _ = delete_tokens_checked(service, profile_id);
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
    pub fn new(service: &str, profile_id: &str, mut config: OAuthConfig, tokens: TokenSet) -> Self {
        if let Some(client_id) = tokens.client_id.as_deref() {
            config.client_id = client_id.to_string();
        }
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
        let mut fresh = refresh(&self.config, &rt).await?;
        fresh.client_id = guard.client_id.clone();
        *guard = fresh;
        store_tokens(&self.service, &self.profile_id, guard)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oauth_authorization_refuses_preseeded_state_and_insecure_endpoints() {
        let mut config = OAuthConfig {
            client_id: "public-client".into(),
            auth_url: "https://login.example.test/oauth/authorize?state=forged".into(),
            token_url: "https://login.example.test/oauth/token".into(),
            scopes: vec![],
            extra_auth_params: vec![],
        };
        assert!(authorize_url(&config, "challenge", "genuine").is_err());
        config.auth_url = "http://login.example.test/oauth/authorize".into();
        assert!(authorize_url(&config, "challenge", "genuine").is_err());
        config.auth_url = "https://login.example.test/oauth/authorize".into();
        assert!(authorize_url(&config, "challenge", "genuine").is_ok());
    }

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
        let url = authorize_url(&cfg, "CHAL", "CSRF").unwrap();
        let q: std::collections::HashMap<_, _> = url.query_pairs().into_owned().collect();
        assert_eq!(q["client_id"], "abc123");
        assert_eq!(q["response_type"], "code");
        assert_eq!(q["code_challenge"], "CHAL");
        assert_eq!(q["code_challenge_method"], "S256");
        assert_eq!(q["state"], "CSRF");
        assert_eq!(q["scope"], "files.read files.write");
        assert_eq!(q["token_access_type"], "offline");
        assert_eq!(q["redirect_uri"], "http://localhost:53682/");
    }

    #[test]
    fn oauth_callback_requires_exact_state_method_and_single_code() {
        assert_eq!(
            parse_oauth_callback("GET /?state=NONCE&code=valid HTTP/1.1", "NONCE"),
            OAuthCallback::Authorized("valid".to_string())
        );
        for request in [
            "GET /?code=attacker HTTP/1.1",
            "GET /?state=wrong&code=attacker HTTP/1.1",
            "GET /other?state=NONCE&code=attacker HTTP/1.1",
            "POST /?state=NONCE&code=attacker HTTP/1.1",
            "GET /?state=NONCE&state=NONCE&code=attacker HTTP/1.1",
            "GET /?state=NONCE&code=one&code=two HTTP/1.1",
            "GET /?state=NONCE&code=one&error=denied HTTP/1.1",
            "GET /?state=NONCE&code= HTTP/1.1",
            "GET /?state=NONCE&code=ok HTTP/1.0",
        ] {
            assert_eq!(
                parse_oauth_callback(request, "NONCE"),
                OAuthCallback::Ignore,
                "accepted hostile OAuth callback: {request}"
            );
        }
        assert_eq!(
            parse_oauth_callback("GET /?state=NONCE&error=access_denied HTTP/1.1", "NONCE"),
            OAuthCallback::Denied
        );
        assert_eq!(
            parse_oauth_callback("GET /?state=wrong&error=access_denied HTTP/1.1", "NONCE"),
            OAuthCallback::Ignore
        );
    }

    #[test]
    fn provider_extra_parameters_cannot_override_security_settings() {
        let mut config = OAuthConfig {
            client_id: "client".into(),
            auth_url: "https://example.invalid/authorize".into(),
            token_url: "https://example.invalid/token".into(),
            scopes: vec![],
            extra_auth_params: vec![("state".into(), "attacker".into())],
        };
        assert!(authorize_url(&config, "PKCE", "NONCE").is_err());
        config.extra_auth_params = vec![("code_challenge".into(), "attacker".into())];
        assert!(authorize_url(&config, "PKCE", "NONCE").is_err());
        config.extra_auth_params.clear();
        assert!(authorize_url(&config, "PKCE", "").is_err());
    }

    #[test]
    fn oauth_token_debug_output_never_contains_bearer_credentials() {
        let tokens = TokenSet {
            access_token: "unique-sensitive-access-token".into(),
            refresh_token: Some("unique-sensitive-refresh-token".into()),
            expires_at: 123,
            client_id: Some("public-client-id".into()),
        };
        let debug = format!("{tokens:?}");
        assert!(!debug.contains("unique-sensitive-access-token"));
        assert!(!debug.contains("unique-sensitive-refresh-token"));
        assert!(debug.contains("[REDACTED]"));
    }

    #[test]
    fn user_supplied_client_id_is_validated_and_stored_without_changing_existing_tokens() {
        let mut config = OAuthConfig {
            client_id: String::new(),
            auth_url: "https://example.invalid/authorize".into(),
            token_url: "https://example.invalid/token".into(),
            scopes: vec![],
            extra_auth_params: vec![],
        };
        assert!(use_client_id(&mut config, None).is_err());
        for invalid in [
            "",
            "https://example.com",
            "key with spaces",
            "a/b",
            "x?token=secret",
        ] {
            assert!(use_client_id(&mut config, Some(invalid)).is_err());
        }
        use_client_id(&mut config, Some(" app-123.apps.googleusercontent.com ")).unwrap();
        assert_eq!(config.client_id, "app-123.apps.googleusercontent.com");

        // Old OS-vault JSON must still deserialize after the optional field.
        let old = r#"{"access_token":"AT","refresh_token":"RT","expires_at":123}"#;
        let old: TokenSet = serde_json::from_str(old).unwrap();
        assert_eq!(old.client_id, None);
        let new = TokenSet {
            client_id: Some(config.client_id.clone()),
            ..old
        };
        assert_eq!(
            serde_json::from_str::<TokenSet>(&serde_json::to_string(&new).unwrap()).unwrap(),
            new
        );
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
            client_id: Some("test-app".into()),
        };
        store_tokens(svc, id, &tokens).expect("store");
        let loaded = load_tokens(svc, id).expect("load").expect("present");
        assert_eq!(loaded, tokens);
        delete_tokens(svc, id);
        assert!(load_tokens(svc, id).expect("load after delete").is_none());
    }
}
