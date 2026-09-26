use crate::oauth::{self, OAuthConfig, RefreshingToken};
use crate::profiles::ConnectionProfile;
use crate::session::http_throttle::{send_retried, HttpThrottle};
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method, StatusCode};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::{Arc, Mutex as StdMutex};
use std::time::{Duration, Instant};
use uuid::Uuid;

/// Keychain purpose prefix for a profile's Dynamics 365 client-credentials
/// blob (`dynamics:{profile_id}` via `credentials.rs`), colon-joined as
/// `tenant_id:client_id:client_secret` (the Shopify `client_id:client_secret`
/// precedent, one field wider). Never stored in `profiles.json`.
pub const DYNAMICS_SERVICE: &str = "dynamics";

/// Keychain service for the delegated (interactive) OAuth `TokenSet`, keyed
/// by profile id — same shape as `ghostftp-onedrive`.
pub const DYNAMICS_TOKEN_SERVICE: &str = "ghostftp-dynamics";

/// Dataverse Web API version pinned for all calls.
const API_VERSION: &str = "v9.2";

/// Pacing for the Dataverse service-protection limits (~6,000 requests /
/// 5 min / user): one session paces everything at ≈10 req/s.
const MIN_INTERVAL: Duration = Duration::from_millis(100);

/// How long the whole web-resource listing stays fresh (one paged query is
/// the entire directory tree, so fetch once and reuse briefly).
const LIST_TTL: Duration = Duration::from_secs(30);

/// Client-credentials tokens last ~1h; re-fetch this far ahead of expiry.
const TOKEN_MARGIN: Duration = Duration::from_secs(300);

/// Dataverse refuses web resources over 5 MB by default (org-configurable) —
/// fail client-side, earlier and clearer.
pub const MAX_WRITE_BYTES: usize = 5 * 1024 * 1024;

/// Dynamics 365 (Microsoft) app client id for the delegated flow. Register an
/// app at <https://entra.microsoft.com> (App registrations), add a **Mobile &
/// desktop** redirect `http://localhost:53682/`, enable public-client flows,
/// and grant the delegated `user_impersonation` scope on **Dynamics CRM**.
/// Paste the Application (client) id here or set `GHOSTFTP_DYNAMICS_CLIENT_ID`.
/// PKCE ⇒ no secret.
const DYNAMICS_CLIENT_ID: &str = "";

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| default.to_string())
}

/// Keychain purpose under which a profile's Dynamics client-credentials blob
/// (`tenant_id:client_id:client_secret`) is stored. The profile editor only
/// ever `set`/`has` this — the value never crosses IPC.
pub fn credential_key(profile_id: &str) -> String {
    format!("{DYNAMICS_SERVICE}:{profile_id}")
}

/// The org host from a profile's `host` field, normalized
/// (`contoso.crm.dynamics.com` — regional variants like `crm4`/`crm5` are
/// just different hosts). Accepts a full URL too, since the connection
/// editor copy says "Environment URL".
pub fn org_host(profile: &ConnectionProfile) -> Result<String> {
    let host = profile
        .host
        .trim()
        .trim_start_matches("https://")
        .trim_start_matches("http://")
        .trim_end_matches('/')
        .to_string();
    if host.is_empty() {
        return Err(anyhow!(
            "Dynamics profile is missing the environment URL (contoso.crm.dynamics.com)"
        ));
    }
    Ok(host)
}

/// Delegated OAuth config: the existing loopback + PKCE machinery with
/// Microsoft Entra endpoints and the org's `user_impersonation` scope (the
/// `common` tenant works for any org). Env overrides exist for tests.
pub fn dynamics_config(host: &str) -> OAuthConfig {
    OAuthConfig {
        client_id: env_or("GHOSTFTP_DYNAMICS_CLIENT_ID", DYNAMICS_CLIENT_ID),
        auth_url: env_or(
            "GHOSTFTP_DYNAMICS_AUTH_URL",
            "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
        ),
        token_url: env_or(
            "GHOSTFTP_DYNAMICS_OAUTH_TOKEN_URL",
            "https://login.microsoftonline.com/common/oauth2/v2.0/token",
        ),
        scopes: vec![
            format!("https://{host}/user_impersonation"),
            "offline_access".into(),
        ],
        extra_auth_params: vec![],
    }
}

/// One web resource row, from the paged
/// `GET /webresourceset?$select=name,webresourceid,webresourcetype,modifiedon,ismanaged`
/// — the whole "directory tree" (`name` is the path, prefixes are folders).
#[derive(Debug, Clone)]
pub struct WebResource {
    pub id: String,
    pub name: String,
    /// `webresourcetype`: 1 HTML, 2 CSS, 3 JS, 4 XML, 5 PNG, 6 JPG, 7 GIF,
    /// 8 XAP, 9 XSL, 10 ICO, 11 SVG, 12 RESX.
    pub res_type: i64,
    pub modified: Option<i64>,
    /// Managed-solution components can't be edited in place — listed
    /// read-only, mutations refused client-side.
    pub managed: bool,
}

/// The two ways to authenticate against an environment (see the plan):
/// client credentials (app registration + application user — the agency/
/// daemon pattern), or the delegated interactive sign-in.
enum Auth {
    /// Tenant + client id + client secret → v2.0 token endpoint (the tenant
    /// rides `token_url`), scope `{org-url}/.default`, cached with a 5-minute
    /// margin (same shape as Shopify's client-credentials exchange).
    ClientCredentials {
        client_id: String,
        client_secret: String,
        cache: StdMutex<Option<(String, Instant)>>,
    },
    /// Interactive Entra ID sign-in (loopback + PKCE); tokens refresh through
    /// the shared [`RefreshingToken`].
    Delegated(RefreshingToken),
}

/// A live Dynamics 365 / Dataverse connection. Stateless HTTP like the other
/// cloud backends — every op is an independent Web API request carrying a
/// bearer token, paced by a tiny throttle (429 + `Retry-After`, 5xx backoff).
pub struct DynamicsSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub api_base: String,
    /// Client-credentials token endpoint
    /// (`login.microsoftonline.com/{tenant}/oauth2/v2.0/token`).
    token_url: String,
    /// The org host (`contoso.crm.dynamics.com`) — token scopes derive from it.
    host: String,
    auth: Auth,
    /// Pacing: a token bucket of one refilled every 100 ms.
    throttle: HttpThrottle,
    /// The connection label, resolved at connect from `WhoAmI`.
    label: String,
    /// The whole web-resource listing, cached briefly.
    resources: StdMutex<Option<(Instant, Arc<Vec<WebResource>>)>>,
    /// Decoded content lengths by name (size needs a content fetch — cache it).
    sizes: StdMutex<HashMap<String, (Instant, u64)>>,
}

impl DynamicsSession {
    /// A valid access token: the delegated refreshing token, or a
    /// client-credentials exchange cached with a 5-minute margin.
    pub async fn token(&self) -> Result<String> {
        match &self.auth {
            Auth::Delegated(t) => t.access_token().await,
            Auth::ClientCredentials {
                client_id,
                client_secret,
                cache,
                ..
            } => {
                {
                    if let Some((tok, exp)) = &*cache.lock().unwrap() {
                        if Instant::now() + TOKEN_MARGIN < *exp {
                            return Ok(tok.clone());
                        }
                    }
                }
                let resp = self
                    .client
                    .post(&self.token_url)
                    .form(&[
                        ("client_id", client_id.as_str()),
                        ("client_secret", client_secret.as_str()),
                        ("grant_type", "client_credentials"),
                        ("scope", format!("https://{}/.default", self.host).as_str()),
                    ])
                    .send()
                    .await
                    .context("dynamics token exchange")?;
                let status = resp.status();
                let text = resp.text().await.unwrap_or_default();
                if !status.is_success() {
                    return Err(anyhow!(
                        "dynamics token exchange failed ({}): {text}. Check the tenant id, \
                         client id and client secret, and that an application user exists for \
                         the app registration in this environment.",
                        status.as_u16()
                    ));
                }
                let v: Value =
                    serde_json::from_str(&text).context("parse dynamics token response")?;
                let token = v
                    .get("access_token")
                    .and_then(|t| t.as_str())
                    .ok_or_else(|| anyhow!("dynamics token exchange returned no access_token"))?
                    .to_string();
                let expires_in = v
                    .get("expires_in")
                    .and_then(|e| e.as_u64())
                    .unwrap_or(3599);
                *cache.lock().unwrap() = Some((
                    token.clone(),
                    Instant::now() + Duration::from_secs(expires_in),
                ));
                Ok(token)
            }
        }
    }

    /// One throttled send (pacing + 429/5xx retry via the shared helper).
    /// `path` may be an API path (`/webresourceset…`) or an absolute URL (an
    /// `@odata.nextLink`).
    async fn send_once(
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
        send_retried(&self.throttle, &format!("dynamics {path}"), move || {
            let url = url.clone();
            let method = method.clone();
            let body = body.clone();
            async move {
                let token = self.token().await?;
                let mut rb = self.client.request(method, &url).bearer_auth(&token);
                if let Some(b) = &body {
                    rb = rb.json(b);
                }
                Ok(rb)
            }
        })
        .await
    }

    /// Send a Web API request; a delegated token that 401s despite looking
    /// valid forces one refresh + retry (the OneDrive pattern).
    pub async fn send(
        &self,
        method: Method,
        path: &str,
        body: Option<&Value>,
    ) -> Result<reqwest::Response> {
        let resp = self.send_once(method.clone(), path, body).await?;
        if resp.status() == StatusCode::UNAUTHORIZED {
            if let Auth::Delegated(t) = &self.auth {
                t.force_refresh().await?;
                return self.send_once(method, path, body).await;
            }
        }
        Ok(resp)
    }

    /// `send` + status check + JSON parse (mirrors the other cloud sessions).
    pub async fn rpc(&self, method: Method, path: &str, body: Option<&Value>) -> Result<Value> {
        let resp = self.send(method, path, body).await?;
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        if !status.is_success() {
            return Err(anyhow!(
                "dynamics {path} failed ({}): {text}",
                status.as_u16()
            ));
        }
        if text.trim().is_empty() {
            return Ok(Value::Null);
        }
        serde_json::from_str(&text).with_context(|| format!("parse dynamics {path} response"))
    }

    /// The connection label: the org host plus the `WhoAmI` user.
    pub async fn account_label(&self) -> Result<String> {
        Ok(self.label.clone())
    }

    /// The whole web-resource listing (every page via `@odata.nextLink`),
    /// cached for ~30s — the flat list *is* the directory tree.
    pub async fn webresources(&self) -> Result<Vec<WebResource>> {
        {
            if let Some((fetched, list)) = self.resources.lock().unwrap().as_ref() {
                if fetched.elapsed() < LIST_TTL {
                    return Ok(list.as_ref().clone());
                }
            }
        }
        let mut out = Vec::new();
        let mut next: Option<String> = Some(
            "/webresourceset?$select=name,webresourceid,webresourcetype,modifiedon,ismanaged"
                .to_string(),
        );
        while let Some(path) = next.take() {
            let v = self.rpc(Method::GET, &path, None).await?;
            if let Some(arr) = v.get("value").and_then(|a| a.as_array()) {
                out.extend(arr.iter().filter_map(resource_from_json));
            }
            next = v
                .get("@odata.nextLink")
                .and_then(|l| l.as_str())
                .map(str::to_string);
        }
        *self.resources.lock().unwrap() = Some((Instant::now(), Arc::new(out.clone())));
        Ok(out)
    }

    /// Drop the cached listing + sizes after a mutation so the next list
    /// re-fetches.
    pub fn invalidate(&self) {
        *self.resources.lock().unwrap() = None;
        self.sizes.lock().unwrap().clear();
    }

    /// A fresh, authoritative lookup by exact `name` (`$filter=name eq '…'`)
    /// — the write path's create-vs-update branch. The listing cache would do
    /// for reads, but writes can't afford a stale miss (create on an existing
    /// name 409s).
    pub async fn resource_lookup(&self, name: &str) -> Result<Option<WebResource>> {
        let escaped = name.replace('\'', "''");
        let path = format!(
            "/webresourceset?$filter=name%20eq%20'{}'&$select=name,webresourceid,webresourcetype,modifiedon,ismanaged",
            urlenc(&escaped)
        );
        let v = self.rpc(Method::GET, &path, None).await?;
        Ok(v
            .get("value")
            .and_then(|a| a.as_array())
            .and_then(|a| a.first())
            .and_then(resource_from_json))
    }

    /// Read one web resource's bytes (`GET …({id})?$select=content` → base64
    /// `content`). Caches the decoded length under `name` for transfer
    /// planning (size needs a content fetch — there is no metadata size).
    pub async fn content_get(&self, id: &str, name: &str) -> Result<Vec<u8>> {
        use base64::Engine as _;
        let v = self
            .rpc(Method::GET, &format!("/webresourceset({id})?$select=content"), None)
            .await?;
        let b64 = v
            .get("content")
            .and_then(|c| c.as_str())
            .ok_or_else(|| anyhow!("dynamics {name}: no content in response"))?;
        let data = base64::engine::general_purpose::STANDARD
            .decode(b64)
            .with_context(|| format!("decode dynamics {name}"))?;
        self.sizes
            .lock()
            .unwrap()
            .insert(name.to_string(), (Instant::now(), data.len() as u64));
        Ok(data)
    }

    /// The cached decoded length for a name, when a recent read/write set it.
    pub fn cached_size(&self, name: &str) -> Option<u64> {
        self.sizes
            .lock()
            .unwrap()
            .get(name)
            .filter(|(fetched, _)| fetched.elapsed() < LIST_TTL)
            .map(|(_, size)| *size)
    }

    /// Create one web resource (`POST /webresourceset`), returning its id —
    /// parsed from the `OData-EntityId` header, with a name-lookup fallback.
    pub async fn resource_create(&self, name: &str, res_type: i64, data: &[u8]) -> Result<String> {
        use base64::Engine as _;
        let body = serde_json::json!({
            "name": name,
            "displayname": name,
            "webresourcetype": res_type,
            "content": base64::engine::general_purpose::STANDARD.encode(data),
        });
        let resp = self.send(Method::POST, "/webresourceset", Some(&body)).await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("dynamics create {name} failed ({code}): {text}"));
        }
        let id = resp
            .headers()
            .get("OData-EntityId")
            .and_then(|v| v.to_str().ok())
            .and_then(entity_id_guid);
        self.sizes
            .lock()
            .unwrap()
            .insert(name.to_string(), (Instant::now(), data.len() as u64));
        match id {
            Some(id) => Ok(id),
            None => Ok(self
                .resource_lookup(name)
                .await?
                .ok_or_else(|| anyhow!("dynamics create {name}: no id in response"))?
                .id),
        }
    }

    /// Update one web resource's content (`PATCH /webresourceset({id})`).
    pub async fn resource_update(&self, id: &str, name: &str, data: &[u8]) -> Result<()> {
        use base64::Engine as _;
        let body = serde_json::json!({
            "content": base64::engine::general_purpose::STANDARD.encode(data),
        });
        let resp = self
            .send(Method::PATCH, &format!("/webresourceset({id})"), Some(&body))
            .await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("dynamics update {name} failed ({code}): {text}"));
        }
        self.sizes
            .lock()
            .unwrap()
            .insert(name.to_string(), (Instant::now(), data.len() as u64));
        Ok(())
    }

    /// Delete one web resource row. A 404 is not an error (idempotent cleanup
    /// after a best-effort rename); managed/dependency errors surface verbatim.
    pub async fn resource_delete(&self, id: &str, name: &str) -> Result<()> {
        let resp = self
            .send(Method::DELETE, &format!("/webresourceset({id})"), None)
            .await?;
        let status = resp.status();
        if !status.is_success() && status != StatusCode::NOT_FOUND {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("dynamics delete {name} failed ({code}): {text}"));
        }
        Ok(())
    }

    /// Publish web resources (`POST /PublishXml`) so a write goes live —
    /// save = deployed, matching the connect-dialog copy. Writes treat a
    /// publish failure as fatal; deletes call it best-effort (a deleted row
    /// has nothing to publish and real Dataverse deletes take effect
    /// immediately — the call only keeps the audit trail honest).
    pub async fn publish(&self, ids: &[String]) -> Result<()> {
        let body = serde_json::json!({ "ParameterXml": publish_xml(ids) });
        let resp = self.send(Method::POST, "/PublishXml", Some(&body)).await?;
        let status = resp.status();
        if !status.is_success() {
            let code = status.as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("dynamics PublishXml failed ({code}): {text}"));
        }
        Ok(())
    }
}

/// The `PublishXml` payload: one `<webresource>{guid}</webresource>` per id.
pub fn publish_xml(ids: &[String]) -> String {
    let mut xml = String::from("<importexportxml><webresources>");
    for id in ids {
        xml.push_str(&format!("<webresource>{id}</webresource>"));
    }
    xml.push_str("</webresources></importexportxml>");
    xml
}

/// Parse one web resource row from the `value` arrays.
pub fn resource_from_json(v: &Value) -> Option<WebResource> {
    Some(WebResource {
        id: v.get("webresourceid")?.as_str()?.to_string(),
        name: v.get("name")?.as_str()?.to_string(),
        res_type: v.get("webresourcetype")?.as_i64()?,
        modified: v
            .get("modifiedon")
            .and_then(|m| m.as_str())
            .and_then(crate::session::shopify::parse_time),
        managed: v
            .get("ismanaged")
            .and_then(|m| m.as_bool())
            .unwrap_or(false),
    })
}

/// Extract the guid from an `OData-EntityId` header
/// (`https://{org}/api/data/v9.2/webresourceset({guid})`).
fn entity_id_guid(url: &str) -> Option<String> {
    let start = url.rfind('(')? + 1;
    let end = url.rfind(')')?;
    (end > start).then(|| url[start..end].to_string())
}

/// Percent-encode a query-string value (names may carry spaces/unicode).
fn urlenc(s: &str) -> String {
    use percent_encoding::{utf8_percent_encode, NON_ALPHANUMERIC};
    utf8_percent_encode(s, NON_ALPHANUMERIC).to_string()
}

/// Open a Dynamics session from a profile. Client credentials (the primary
/// mode): the OS keychain (`dynamics:{profile_id}`) holds the colon-joined
/// blob `tenant_id:client_id:client_secret` (the Shopify `client_id:secret`
/// precedent, one field wider). Delegated: an OAuth `TokenSet` under
/// `ghostftp-dynamics` from the "Sign in with Microsoft" flow. Probes
/// `GET /WhoAmI` so a bad credential or missing application user fails at
/// connect, not on first browse.
pub async fn dynamics_connect(profile: &ConnectionProfile) -> Result<DynamicsSession> {
    let host = org_host(profile)?;

    let (auth, tenant) = match crate::credentials::get_secret(&credential_key(&profile.id))?
        .filter(|s| !s.trim().is_empty())
    {
        // Client credentials (app registration + application user). The blob
        // splits on the first two colons only, so a secret containing ':'
        // still parses.
        Some(blob) => {
            let mut parts = blob.splitn(3, ':');
            let (tenant, client_id, client_secret) = match (
                parts.next(),
                parts.next(),
                parts.next(),
            ) {
                (Some(t), Some(c), Some(s))
                    if !t.trim().is_empty() && !c.trim().is_empty() && !s.is_empty() =>
                {
                    (t.trim().to_string(), c.trim().to_string(), s.to_string())
                }
                _ => {
                    return Err(anyhow!(
                        "This Dynamics connection's saved credential is malformed. Open it in \
                         the connection editor and fill in the tenant id, client id and client \
                         secret (stored as tenant:client_id:client_secret)."
                    ))
                }
            };
            (
                Auth::ClientCredentials {
                    client_id,
                    client_secret,
                    cache: StdMutex::new(None),
                },
                tenant,
            )
        }
        // Delegated (interactive "Sign in with Microsoft").
        None => {
            let tokens = oauth::load_tokens(DYNAMICS_TOKEN_SERVICE, &profile.id)?.ok_or_else(|| {
                anyhow!(
                    "This Dynamics connection has no saved credential. Open it in the \
                     connection editor and either fill in the client credentials (tenant id, \
                     client id, client secret) or click “Sign in with Microsoft” (delegated \
                     mode)."
                )
            })?;
            (
                Auth::Delegated(RefreshingToken::new(
                    DYNAMICS_TOKEN_SERVICE,
                    &profile.id,
                    dynamics_config(&host),
                    tokens,
                )),
                // The delegated token endpoint is tenant-agnostic (`common`).
                "common".to_string(),
            )
        }
    };

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(300))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building Dynamics HTTP client")?;

    let mut session = DynamicsSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        api_base: env_or(
            "GHOSTFTP_DYNAMICS_API_BASE",
            &format!("https://{host}/api/data/{API_VERSION}"),
        ),
        token_url: env_or(
            "GHOSTFTP_DYNAMICS_TOKEN_URL",
            &format!("https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token"),
        ),
        host: host.clone(),
        auth,
        throttle: HttpThrottle::new(MIN_INTERVAL),
        label: host.clone(),
        resources: StdMutex::new(None),
        sizes: StdMutex::new(HashMap::new()),
    };

    // Connect-time validation with user-actionable errors.
    let resp = session.send(Method::GET, "/WhoAmI", None).await?;
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    match status {
        StatusCode::UNAUTHORIZED => {
            return Err(anyhow!(
                "Dataverse rejected the credential (401). For client-credentials mode, check \
                 that an application user exists for the app registration in this environment \
                 (Power Platform admin center → Settings → Users → App users) with a security \
                 role that can read and write web resources."
            ))
        }
        StatusCode::FORBIDDEN => {
            return Err(anyhow!(
                "Dataverse denied access (403). The application user's security role needs \
                 read/write on web resources (e.g. System Customizer)."
            ))
        }
        StatusCode::NOT_FOUND => {
            return Err(anyhow!(
                "Dataverse environment not found (404). Check the environment URL ({host})."
            ))
        }
        s if !s.is_success() => {
            return Err(anyhow!("dynamics WhoAmI failed ({}): {text}", s.as_u16()))
        }
        _ => {}
    }
    // Label: org host + the WhoAmI user id (the API returns guids only).
    if let Ok(v) = serde_json::from_str::<Value>(&text) {
        if let Some(uid) = v.get("UserId").and_then(|u| u.as_str()) {
            session.label = format!("{host} · user {uid}");
        }
    }
    Ok(session)
}
