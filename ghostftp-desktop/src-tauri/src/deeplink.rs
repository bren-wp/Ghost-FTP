//! `ghostftp://` deep links — one-click "open in Ghost FTP" URLs a hosting panel like
//! a hosting panel can hand out next to a site ("Connect with Ghost FTP").
//!
//! A link **never auto-connects and never carries a password**: any web page
//! can invoke a registered protocol handler, so a link only ever opens a
//! prefilled, reviewable connection editor. The user sees exactly what they're
//! about to connect to and clicks Connect (or Pair) themselves.
//!
//! Grammar (see `docs/deep-links.md` for the full spec):
//!
//! ```text
//! ghostftp://connect?protocol=sftp&host=example.invalid&port=22&username=me&path=/var/www&name=My%20Site
//! ghostftp://pair?host=192.168.1.5&port=8722&code=428170&name=Basement%20NAS
//! ghostftp://terminal?name=My%20Site
//! ```
//!
//! `terminal` opens a standalone terminal window for an already-connected
//! server (matched by `name` or `host`); if the server isn't connected, the
//! UI falls back to the prefilled editor like `connect` does.

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};
use url::Url;

/// A parsed `ghostftp://` link, forwarded to the UI to prefill an editor. Every
/// field is optional so a partial link still does something sensible; the UI
/// fills the gaps with its own defaults.
#[derive(Debug, Clone, Default, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DeepLink {
    /// `"connect"` (open a server editor) or `"pair"` (open the Ghost FTP Agent
    /// pairing editor). Anything else is ignored by the UI.
    pub action: String,
    pub protocol: Option<String>,
    pub host: Option<String>,
    pub port: Option<u16>,
    pub username: Option<String>,
    pub path: Option<String>,
    pub name: Option<String>,
    /// Ghost FTP Agent pairing code (`ghostftp://pair` only). A one-time code, not a secret.
    pub code: Option<String>,
    // Object-store conveniences so a link can point straight at a bucket.
    pub bucket: Option<String>,
    pub region: Option<String>,
    pub endpoint: Option<String>,
    pub account: Option<String>,
    // Access grants (`ghostftp://grant` only): the issuing service's base URL and
    // the redemption token. Neither is a credential — the token only fetches a
    // manifest; the user consents before anything is imported.
    pub issuer: Option<String>,
    pub token: Option<String>,
}

/// Parse one `ghostftp://…` URL. Returns `None` for a non-`ghostftp` scheme or an
/// unparseable URL. The authority names the action (`ghostftp://connect?…`), and
/// everything else arrives as query parameters — so the site's own host lives
/// in `?host=`, not in the URL authority.
pub fn parse(raw: &str) -> Option<DeepLink> {
    let url = Url::parse(raw.trim()).ok()?;
    if url.scheme() != "ghostftp" {
        return None;
    }
    // `ghostftp://connect?…` → authority "connect". Fall back to the path for a
    // `ghostftp:connect` (no-authority) form so both parse.
    let action = match url.host_str() {
        Some(h) if !h.is_empty() => h.to_string(),
        _ => url.path().trim_start_matches('/').to_string(),
    };

    let mut dl = DeepLink {
        action: action.to_lowercase(),
        ..Default::default()
    };
    for (k, v) in url.query_pairs() {
        let v = v.to_string();
        if v.is_empty() {
            continue;
        }
        // Bound attacker-controlled protocol-handler input before it reaches
        // React state or later network flows. Deep links may be invoked by any
        // web page registered with the OS.
        if v.len() > 4096 {
            tracing::warn!("ignoring oversized '{k}' parameter in a ghostftp:// link");
            continue;
        }
        match k.as_ref() {
            "protocol" => dl.protocol = Some(v.to_lowercase()),
            "host" => dl.host = Some(v),
            "port" => dl.port = v.parse().ok(),
            "username" | "user" => dl.username = Some(v),
            "path" => dl.path = Some(v),
            "name" | "label" => dl.name = Some(v),
            "code" => dl.code = Some(v.chars().filter(|c| c.is_ascii_digit()).collect()),
            "bucket" => dl.bucket = Some(v),
            "region" => dl.region = Some(v),
            "endpoint" => dl.endpoint = Some(v),
            "account" => dl.account = Some(v),
            "issuer" => dl.issuer = Some(v),
            "token" => dl.token = Some(v),
            // Deliberately ignored — links must not carry credentials.
            "password" | "secret" | "passphrase" => {
                tracing::warn!("ignoring credential param '{k}' in a ghostftp:// link");
            }
            _ => {}
        }
    }
    if dl.action == "grant" {
        let issuer = dl.issuer.as_deref()?;
        let issuer_url = Url::parse(issuer).ok()?;
        if issuer_url.scheme() != "https"
            || issuer_url.host_str().is_none()
            || !issuer_url.username().is_empty()
            || issuer_url.password().is_some()
        {
            tracing::warn!("ignoring grant deep link with unsafe issuer");
            return None;
        }
        let token = dl.token.as_deref()?;
        if token.len() > 512
            || !token
                .chars()
                .all(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '-'))
        {
            tracing::warn!("ignoring grant deep link with invalid token");
            return None;
        }
    }
    Some(dl)
}

/// Handle every URL in an incoming open event: parse and forward the valid ones
/// to the frontend as `deep-link://open`. Invalid or credential-bearing links
/// are dropped with a log line rather than surfaced.
pub fn handle_urls(app: &AppHandle, urls: &[url::Url]) {
    for u in urls {
        match parse(u.as_str()) {
            Some(dl)
                if dl.action == "connect"
                    || dl.action == "pair"
                    || dl.action == "terminal"
                    || dl.action == "grant" =>
            {
                if let Err(e) = app.emit("deep-link://open", &dl) {
                    tracing::warn!("failed to forward deep link: {e}");
                }
                // Bring the window forward so the prefilled editor is visible.
                if let Some(win) = app.get_webview_window("main") {
                    let _ = win.set_focus();
                }
            }
            Some(dl) => tracing::info!(
                "ignoring ghostftp:// link with unknown action '{}'",
                dl.action
            ),
            None => tracing::info!("ignoring unparseable ghostftp:// link"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_connect_link() {
        let dl = parse(
            "ghostftp://connect?protocol=sftp&host=example.invalid&port=22&username=me&path=/var/www&name=My%20Site",
        )
        .unwrap();
        assert_eq!(dl.action, "connect");
        assert_eq!(dl.protocol.as_deref(), Some("sftp"));
        assert_eq!(dl.host.as_deref(), Some("example.invalid"));
        assert_eq!(dl.port, Some(22));
        assert_eq!(dl.username.as_deref(), Some("me"));
        assert_eq!(dl.path.as_deref(), Some("/var/www"));
        assert_eq!(dl.name.as_deref(), Some("My Site"));
    }

    #[test]
    fn parses_pair_link_and_sanitises_code() {
        let dl = parse("ghostftp://pair?host=192.168.1.5&port=8722&code=42-81-70").unwrap();
        assert_eq!(dl.action, "pair");
        assert_eq!(dl.host.as_deref(), Some("192.168.1.5"));
        assert_eq!(dl.port, Some(8722));
        assert_eq!(dl.code.as_deref(), Some("428170"));
    }

    #[test]
    fn drops_credentials() {
        let dl = parse("ghostftp://connect?host=h&password=hunter2&secret=x").unwrap();
        // No field on DeepLink can hold a credential; the params are ignored.
        assert_eq!(dl.host.as_deref(), Some("h"));
    }

    #[test]
    fn parses_terminal_link() {
        let dl = parse("ghostftp://terminal?name=My%20Site").unwrap();
        assert_eq!(dl.action, "terminal");
        assert_eq!(dl.name.as_deref(), Some("My Site"));
    }

    #[test]
    fn rejects_foreign_scheme() {
        assert!(parse("https://example.invalid").is_none());
        assert!(parse("ssh://example.invalid").is_none());
    }

    #[test]
    fn parses_grant_link() {
        let dl = parse(
            "ghostftp://grant?issuer=https%3A%2F%2Fpanel.agency.com&token=gr_9f2kQ7abcd1234&name=Client%20X",
        )
        .unwrap();
        assert_eq!(dl.action, "grant");
        assert_eq!(dl.issuer.as_deref(), Some("https://panel.agency.com"));
        assert_eq!(dl.token.as_deref(), Some("gr_9f2kQ7abcd1234"));
        assert_eq!(dl.name.as_deref(), Some("Client X"));
    }

    #[test]
    fn grant_link_keeps_token_and_issuer_separate() {
        let dl =
            parse("ghostftp://grant?issuer=https://panel.example&token=abcDEF_123-xyz456").unwrap();
        assert_eq!(dl.issuer.as_deref(), Some("https://panel.example"));
        assert_eq!(dl.token.as_deref(), Some("abcDEF_123-xyz456"));
        // Neither param leaks into the host/code fields of other actions.
        assert!(dl.host.is_none());
        assert!(dl.code.is_none());
    }

    #[test]
    fn rejects_insecure_grant_issuer() {
        assert!(parse("ghostftp://grant?issuer=http://panel.example&token=abc123").is_none());
    }

    #[test]
    fn rejects_grant_issuer_with_userinfo() {
        assert!(
            parse("ghostftp://grant?issuer=https://user:pass@panel.example&token=abc123").is_none()
        );
    }

    #[test]
    fn rejects_missing_or_malformed_grant_token() {
        assert!(parse("ghostftp://grant?issuer=https://panel.example").is_none());
        assert!(parse("ghostftp://grant?issuer=https://panel.example&token=abc%2F123").is_none());
    }

    #[test]
    fn rejects_oversized_grant_token() {
        let token = "a".repeat(513);
        let raw = format!("ghostftp://grant?issuer=https://panel.example&token={token}");
        assert!(parse(&raw).is_none());
    }

    #[test]
    fn grant_link_still_strips_credentials() {
        let dl = parse(
            "ghostftp://grant?issuer=https://panel.example&token=abcDEF_123-xyz456&password=hunter2&secret=x",
        )
        .unwrap();
        assert_eq!(dl.token.as_deref(), Some("abcDEF_123-xyz456"));
        // No field on DeepLink can hold the credential params; they're ignored.
        let json = serde_json::to_string(&dl).unwrap();
        assert!(!json.contains("hunter2"));
    }
}
