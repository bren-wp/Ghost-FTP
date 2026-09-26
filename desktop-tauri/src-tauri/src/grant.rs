//! Access grants (`ghostftp://grant`) — the client half of the Ghost FTP Grant
//! Protocol v1 (`docs/grant-links.md`).
//!
//! An issuer (hosting panel, agency's a hosting panel, …) mints a redemption token;
//! the deep link carries only `{issuer, token}`. `fetch_grant_manifest` reads
//! the manifest so the consent dialog can show exactly what access is being
//! offered. Only after the user clicks Accept does `accept_grant` run the
//! exchange: generate a fresh ed25519 keypair in memory, upload the *public*
//! key, store the private key in the OS keychain (`grant-key:<profile-id>`,
//! never on disk), and import the granted servers as ordinary profiles.
//!
//! The private key never crosses the IPC boundary and never lands in
//! `profiles.json` — the imported profiles reference it via
//! `AuthMethod::KeyRef`.

use crate::error::{ErrorKind, GhostFTPError};
use crate::profiles::{AuthMethod, ConnectionProfile};
use crate::AppState;
use serde::{Deserialize, Serialize};
use std::time::Duration;
use tauri::State;
use uuid::Uuid;

/// Keychain purpose prefix for grant private keys (`grant-key:<profile-id>`).
pub const GRANT_KEY_PREFIX: &str = "grant-key:";

/// Supported manifest version (Ghost FTP Grant Protocol v1).
const SUPPORTED_VERSION: u32 = 1;
/// Spec limit: 1–64 connections per grant.
const MAX_CONNECTIONS: usize = 64;

// ---------- Wire types ----------

/// The manifest served at `{issuer}/.well-known/ghostftp-grant/{token}`. Issuers
/// emit snake_case per the spec; the frontend round-trips this type back into
/// `accept_grant`, so it also accepts/emit camelCase aliases.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GrantManifest {
    pub version: u32,
    /// Display name of the issuing org/service (not a URL).
    pub issuer: String,
    /// Display name of the grant.
    pub name: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub group: Option<String>,
    /// RFC 3339 expiry. Informational — enforcement is the issuer's job.
    #[serde(default, alias = "expires_at", skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
    pub auth: GrantAuth,
    pub connections: Vec<GrantConnection>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GrantAuth {
    /// Only `"key-install"` in v1.
    #[serde(rename = "type")]
    pub kind: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GrantConnection {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    /// Only `"sftp"` in v1.
    pub protocol: String,
    /// DNS name or IP — never a URL.
    pub host: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub port: Option<u16>,
    pub username: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub path: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub jump: Option<GrantJump>,
}

/// Optional bastion hop. The SAME uploaded key authenticates both hops.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GrantJump {
    pub host: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub port: Option<u16>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub username: Option<String>,
}

/// What `accept_grant` imported, and what the issuer reported as failed.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GrantImportResult {
    /// The rail folder the imported profiles landed in.
    pub group: String,
    pub imported: Vec<ConnectionProfile>,
    pub failed: Vec<GrantImportFailure>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GrantImportFailure {
    pub name: String,
    pub error: String,
}

/// The issuer's reply to the key upload: `{ "installed": [...], "failed":
/// [{name,error}] }`. An absent `installed` means every connection succeeded.
#[derive(Debug, Deserialize)]
struct KeyInstallResponse {
    #[serde(default)]
    installed: Option<Vec<String>>,
    #[serde(default)]
    failed: Vec<KeyInstallFailure>,
}

#[derive(Debug, Deserialize)]
struct KeyInstallFailure {
    name: String,
    #[serde(default)]
    error: String,
}

// ---------- Validation (pure, unit-testable) ----------

/// Validate the issuer base URL from the link. Must be HTTPS; plain HTTP is
/// accepted only for loopback hosts (local development). Returns the
/// normalised base URL (no trailing slash) for building endpoint URLs.
pub fn validate_issuer(issuer: &str) -> Result<String, GhostFTPError> {
    let url = url::Url::parse(issuer.trim())
        .map_err(|e| GhostFTPError::other(format!("invalid grant issuer URL: {e}")))?;
    match url.scheme() {
        "https" => {}
        "http" => {
            let loopback = match url.host() {
                Some(url::Host::Domain(d)) => d.eq_ignore_ascii_case("localhost"),
                Some(url::Host::Ipv4(ip)) => ip.is_loopback(),
                Some(url::Host::Ipv6(ip)) => ip.is_loopback(),
                None => false,
            };
            if !loopback {
                return Err(GhostFTPError::new(
                    ErrorKind::Unsupported,
                    "grant issuer must use HTTPS (plain HTTP is only allowed for localhost)",
                ));
            }
        }
        other => {
            return Err(GhostFTPError::new(
                ErrorKind::Unsupported,
                format!("grant issuer must be an HTTPS URL, got scheme '{other}'"),
            ));
        }
    }
    if url.host().is_none() {
        return Err(GhostFTPError::other("grant issuer URL has no host"));
    }
    Ok(url.as_str().trim_end_matches('/').to_string())
}

/// Validate a grant token: `[A-Za-z0-9_-]{16,128}` per the spec. The charset
/// is URL-path-safe, so a validated token can be concatenated into the
/// endpoint path without further encoding.
pub fn validate_token(token: &str) -> Result<(), GhostFTPError> {
    let ok = (16..=128).contains(&token.len())
        && token
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '-');
    if !ok {
        return Err(GhostFTPError::other(
            "invalid grant token (expected 16–128 of [A-Za-z0-9_-])",
        ));
    }
    Ok(())
}

/// Validate a fetched (or caller-supplied) manifest against the v1 spec.
pub fn validate_manifest(manifest: &GrantManifest) -> Result<(), GhostFTPError> {
    if manifest.version != SUPPORTED_VERSION {
        return Err(GhostFTPError::new(
            ErrorKind::Unsupported,
            format!(
                "unsupported grant manifest version {} (Ghost FTP speaks v{SUPPORTED_VERSION})",
                manifest.version
            ),
        ));
    }
    if manifest.auth.kind != "key-install" {
        return Err(GhostFTPError::new(
            ErrorKind::Unsupported,
            format!("unsupported grant auth type '{}'", manifest.auth.kind),
        ));
    }
    if manifest.connections.is_empty() {
        return Err(GhostFTPError::other("grant manifest lists no connections"));
    }
    if manifest.connections.len() > MAX_CONNECTIONS {
        return Err(GhostFTPError::other(format!(
            "grant manifest lists {} connections (max {MAX_CONNECTIONS})",
            manifest.connections.len()
        )));
    }
    for conn in &manifest.connections {
        if conn.protocol != "sftp" {
            return Err(GhostFTPError::new(
                ErrorKind::Unsupported,
                format!(
                    "unsupported connection protocol '{}' (only sftp in v1)",
                    conn.protocol
                ),
            ));
        }
        validate_host(&conn.host)?;
        if let Some(jump) = &conn.jump {
            validate_host(&jump.host)?;
        }
    }
    Ok(())
}

/// A manifest host must be a DNS name or IP — never a URL.
fn validate_host(host: &str) -> Result<(), GhostFTPError> {
    if host.trim().is_empty() {
        return Err(GhostFTPError::other("grant connection has an empty host"));
    }
    if host.contains("://") || host.contains('/') {
        return Err(GhostFTPError::other(format!(
            "grant connection host '{host}' looks like a URL, not a host"
        )));
    }
    Ok(())
}

// ---------- HTTP ----------

fn http_client() -> Result<reqwest::Client, GhostFTPError> {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(15))
        .build()
        .map_err(|e| GhostFTPError::other(format!("build HTTP client: {e}")))
}

fn http_err(context: &str, e: reqwest::Error) -> GhostFTPError {
    let kind = if e.is_timeout() {
        ErrorKind::Timeout
    } else if e.is_connect() {
        ErrorKind::Network
    } else {
        ErrorKind::Other
    };
    GhostFTPError::new(kind, format!("{context}: {e}"))
}

/// Extract the optional `{ "error": "…" }` body from an error response.
async fn error_body(resp: reqwest::Response) -> String {
    #[derive(Deserialize)]
    struct ErrBody {
        error: Option<String>,
    }
    resp.json::<ErrBody>()
        .await
        .ok()
        .and_then(|b| b.error)
        .unwrap_or_default()
}

// ---------- Commands ----------

/// Fetch and validate the grant manifest, so the consent dialog can show the
/// user exactly what access is being offered. No state is changed.
#[tauri::command]
pub async fn fetch_grant_manifest(
    issuer: String,
    token: String,
) -> Result<GrantManifest, GhostFTPError> {
    let base = validate_issuer(&issuer)?;
    validate_token(&token)?;
    let url = format!("{base}/.well-known/ghostftp-grant/{token}");

    let resp = http_client()?
        .get(&url)
        .send()
        .await
        .map_err(|e| http_err("fetch grant manifest", e))?;
    if !resp.status().is_success() {
        let status = resp.status();
        let detail = error_body(resp).await;
        return Err(grant_http_error("fetch grant manifest", status, &detail));
    }
    let manifest: GrantManifest = resp
        .json()
        .await
        .map_err(|e| GhostFTPError::other(format!("parse grant manifest: {e}")))?;
    validate_manifest(&manifest)?;
    Ok(manifest)
}

/// Run the grant exchange after the user consents: generate a fresh ed25519
/// keypair in memory, upload the public key, store the private key in the OS
/// keychain, and import one profile per installed connection. On total
/// failure (non-200 from the issuer) nothing is stored.
#[tauri::command]
pub async fn accept_grant(
    issuer: String,
    token: String,
    manifest: GrantManifest,
    state: State<'_, AppState>,
) -> Result<GrantImportResult, GhostFTPError> {
    // Never trust the caller — re-validate everything the dialog passed back.
    let base = validate_issuer(&issuer)?;
    validate_token(&token)?;
    validate_manifest(&manifest)?;

    // Keygen is CPU work — keep it off the async runtime.
    let (public_key, pem) = tokio::task::spawn_blocking(|| {
        crate::keys::generate_ed25519_in_memory("ghostftp-grant")
    })
    .await
    .map_err(|e| GhostFTPError::other(format!("grant keygen task: {e}")))?
    .map_err(GhostFTPError::from)?;

    // Upload ONLY the public key. The private half never leaves the machine.
    let url = format!("{base}/.well-known/ghostftp-grant/{token}/key");
    let resp = http_client()?
        .post(&url)
        .json(&serde_json::json!({ "public_key": public_key }))
        .send()
        .await
        .map_err(|e| http_err("upload grant key", e))?;
    if !resp.status().is_success() {
        let status = resp.status();
        let detail = error_body(resp).await;
        // Total failure — import nothing, store nothing.
        return Err(grant_http_error("upload grant key", status, &detail));
    }
    let reply: KeyInstallResponse = resp
        .json()
        .await
        .map_err(|e| GhostFTPError::other(format!("parse grant key response: {e}")))?;

    let group = manifest
        .group
        .clone()
        .unwrap_or_else(|| manifest.issuer.clone());

    // Which connections the issuer installed the key on. Absent = all.
    let installed: Vec<String> = reply.installed.unwrap_or_else(|| {
        manifest
            .connections
            .iter()
            .map(connection_label)
            .collect()
    });
    let mut failed: Vec<GrantImportFailure> = reply
        .failed
        .into_iter()
        .map(|f| GrantImportFailure {
            name: f.name,
            error: f.error,
        })
        .collect();

    let mut imported: Vec<ConnectionProfile> = Vec::new();
    for conn in &manifest.connections {
        let label = connection_label(conn);
        if !installed.iter().any(|n| n == &label) {
            if !failed.iter().any(|f| f.name == label) {
                failed.push(GrantImportFailure {
                    name: label,
                    error: "issuer did not install the key on this connection".into(),
                });
            }
            continue;
        }
        match import_connection(conn, &group, &pem, &state).await {
            Ok(profile) => imported.push(profile),
            Err(error) => {
                tracing::warn!(connection = %label, %error, "grant import failed");
                failed.push(GrantImportFailure { name: label, error });
            }
        }
    }

    tracing::info!(
        grant = %manifest.name,
        imported = imported.len(),
        failed = failed.len(),
        "grant accepted"
    );
    Ok(GrantImportResult {
        group,
        imported,
        failed,
    })
}

/// The display identity of a connection — what the issuer lists in
/// `installed`/`failed`, and the fallback profile name.
fn connection_label(conn: &GrantConnection) -> String {
    conn.name.clone().unwrap_or_else(|| conn.host.clone())
}

/// Store the grant private key under `grant-key:<profile-id>` and upsert the
/// imported profile. On a profile-store failure the keychain entry is removed
/// again so no orphaned key material is left behind.
async fn import_connection(
    conn: &GrantConnection,
    group: &str,
    pem: &str,
    state: &State<'_, AppState>,
) -> Result<ConnectionProfile, String> {
    let id = Uuid::new_v4().to_string();
    let key_ref = format!("{GRANT_KEY_PREFIX}{id}");
    crate::credentials::set_secret(&key_ref, pem).map_err(|e| e.to_string())?;
    state
        .db
        .record_keychain(crate::credentials::SERVICE, &key_ref)
        .map_err(|e| e.to_string())?;

    let profile = ConnectionProfile {
        id,
        name: connection_label(conn),
        protocol: "sftp".into(),
        host: conn.host.clone(),
        port: conn.port.unwrap_or(22),
        username: conn.username.clone(),
        auth: AuthMethod::KeyRef {
            key_ref: key_ref.clone(),
        },
        default_remote_path: conn.path.clone(),
        color: None,
        auto_connect: None,
        bucket: None,
        region: None,
        endpoint: None,
        account: None,
        agent_key: None,
        group: Some(group.to_string()),
        favorite: None,
        bookmarked: None,
        tags: None,
        last_used: None,
        sort_order: None,
        icon: None,
        jump_host: conn.jump.as_ref().map(|j| j.host.clone()),
        jump_port: conn.jump.as_ref().and_then(|j| j.port),
        jump_username: conn.jump.as_ref().and_then(|j| j.username.clone()),
    };
    if let Err(e) = state.profiles.upsert(profile.clone()).await {
        crate::credentials::delete_secret(&key_ref);
        let _ = state
            .db
            .forget_keychain(crate::credentials::SERVICE, &key_ref);
        return Err(e.to_string());
    }
    Ok(profile)
}

/// Map an issuer error status to a structured error with the optional detail.
fn grant_http_error(context: &str, status: reqwest::StatusCode, detail: &str) -> GhostFTPError {
    let suffix = if detail.is_empty() {
        String::new()
    } else {
        format!(": {detail}")
    };
    match status.as_u16() {
        404 => GhostFTPError::new(
            ErrorKind::NotFound,
            format!("grant token unknown, already redeemed, or expired{suffix}"),
        ),
        410 => GhostFTPError::new(ErrorKind::Auth, format!("grant was revoked{suffix}")),
        429 => GhostFTPError::new(
            ErrorKind::Other,
            format!("issuer rate-limited the grant request — try again later{suffix}"),
        ),
        _ => GhostFTPError::other(format!("{context} failed (HTTP {status}){suffix}")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn good_manifest() -> GrantManifest {
        GrantManifest {
            version: 1,
            issuer: "Hosting Panel · panel.example.invalid".into(),
            name: "Client X — 2 servers".into(),
            group: Some("Agency / Client X".into()),
            expires_at: Some("2026-08-07T00:00:00Z".into()),
            auth: GrantAuth {
                kind: "key-install".into(),
            },
            connections: vec![
                GrantConnection {
                    name: Some("web-1".into()),
                    protocol: "sftp".into(),
                    host: "10.0.0.11".into(),
                    port: Some(22),
                    username: "deploy".into(),
                    path: Some("/var/www".into()),
                    jump: Some(GrantJump {
                        host: "bastion.agency.com".into(),
                        port: None,
                        username: Some("ghostftp-grant".into()),
                    }),
                },
                GrantConnection {
                    name: None,
                    protocol: "sftp".into(),
                    host: "10.0.0.12".into(),
                    port: None,
                    username: "deploy".into(),
                    path: None,
                    jump: None,
                },
            ],
        }
    }

    // ---- validate_token ----

    #[test]
    fn token_charset_and_length() {
        assert!(validate_token("gr_9f2kQ7abcd-1234").is_ok());
        assert!(validate_token(&"a".repeat(16)).is_ok());
        assert!(validate_token(&"a".repeat(128)).is_ok());
        assert!(validate_token(&"a".repeat(15)).is_err());
        assert!(validate_token(&"a".repeat(129)).is_err());
        assert!(validate_token("short with spaces!!").is_err());
        assert!(validate_token("token/with/slashes1").is_err());
        assert!(validate_token("").is_err());
    }

    // ---- validate_issuer ----

    #[test]
    fn issuer_requires_https() {
        assert!(validate_issuer("https://panel.agency.com").is_ok());
        assert!(validate_issuer("https://panel.agency.com/prefix/").is_ok());
        assert!(validate_issuer("http://panel.agency.com").is_err());
        assert!(validate_issuer("ftp://panel.agency.com").is_err());
        assert!(validate_issuer("not a url").is_err());
    }

    #[test]
    fn issuer_allows_loopback_http() {
        assert!(validate_issuer("http://localhost:8080").is_ok());
        assert!(validate_issuer("http://127.0.0.1:9321").is_ok());
        assert!(validate_issuer("http://[::1]:8080").is_ok());
        // …but not lookalikes.
        assert!(validate_issuer("http://localhost.evil.com").is_err());
        assert!(validate_issuer("http://192.168.1.5").is_err());
    }

    #[test]
    fn issuer_normalises_trailing_slash() {
        let base = validate_issuer("https://panel.agency.com/").unwrap();
        assert_eq!(base, "https://panel.agency.com");
    }

    // ---- validate_manifest ----

    #[test]
    fn good_manifest_passes() {
        assert!(validate_manifest(&good_manifest()).is_ok());
    }

    #[test]
    fn rejects_wrong_version() {
        let mut m = good_manifest();
        m.version = 2;
        let e = validate_manifest(&m).unwrap_err();
        assert_eq!(e.kind, ErrorKind::Unsupported);
    }

    #[test]
    fn rejects_wrong_auth_type() {
        let mut m = good_manifest();
        m.auth.kind = "password".into();
        assert_eq!(validate_manifest(&m).unwrap_err().kind, ErrorKind::Unsupported);
    }

    #[test]
    fn rejects_bad_connection_counts() {
        let mut m = good_manifest();
        m.connections.clear();
        assert!(validate_manifest(&m).is_err());

        let one = good_manifest().connections.into_iter().next().unwrap();
        let mut m = good_manifest();
        m.connections = vec![one; 65];
        assert!(validate_manifest(&m).is_err());

        let one = good_manifest().connections.into_iter().next().unwrap();
        let mut m = good_manifest();
        m.connections = vec![one; 64];
        assert!(validate_manifest(&m).is_ok());
    }

    #[test]
    fn rejects_non_sftp_protocol() {
        let mut m = good_manifest();
        m.connections[0].protocol = "ftp".into();
        assert_eq!(validate_manifest(&m).unwrap_err().kind, ErrorKind::Unsupported);
    }

    #[test]
    fn rejects_url_hosts() {
        for bad in ["https://10.0.0.11", "10.0.0.11/evil", ""] {
            let mut m = good_manifest();
            m.connections[0].host = bad.into();
            assert!(validate_manifest(&m).is_err(), "host {bad:?} must fail");
        }
        // Same rules for the jump host.
        let mut m = good_manifest();
        m.connections[0].jump.as_mut().unwrap().host = "https://bastion".into();
        assert!(validate_manifest(&m).is_err());
    }

    // ---- serde round-trip ----

    #[test]
    fn manifest_parses_spec_json() {
        // The issuer's snake_case wire form, verbatim from the spec.
        let json = r#"{
            "version": 1,
            "issuer": "Hosting Panel · panel.example.invalid",
            "name": "Client X — 1 server",
            "group": "Agency / Client X",
            "expires_at": "2026-08-07T00:00:00Z",
            "auth": { "type": "key-install" },
            "connections": [{
                "name": "web-1",
                "protocol": "sftp",
                "host": "10.0.0.11",
                "port": 22,
                "username": "deploy",
                "path": "/var/www",
                "jump": { "host": "bastion.agency.com", "port": 22, "username": "ghostftp-grant" }
            }]
        }"#;
        let m: GrantManifest = serde_json::from_str(json).expect("parse spec manifest");
        assert_eq!(m.expires_at.as_deref(), Some("2026-08-07T00:00:00Z"));
        assert_eq!(m.auth.kind, "key-install");
        assert!(validate_manifest(&m).is_ok());
        // …and it round-trips back through camelCase (what the frontend sends).
        let m2: GrantManifest = serde_json::from_str(&serde_json::to_string(&m).unwrap()).unwrap();
        assert!(validate_manifest(&m2).is_ok());
    }
}
