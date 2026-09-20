use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use reqwest::{Client, Method, RequestBuilder};
use std::time::Duration;
use url::Url;

/// How the WebDAV requests authenticate. Basic covers Nextcloud/ownCloud app
/// passwords, Hetzner Storage Box, and the generic long tail; Bearer covers
/// token servers (leave the username blank in the UI to select it).
pub enum WebdavAuth {
    Basic { user: String, password: String },
    Bearer { token: String },
    None,
}

/// A WebDAV connection. Unlike SSH/FTP there is no persistent socket — every op
/// is an independent HTTP request off a shared `reqwest::Client` (connection
/// pooling + keep-alive happen inside reqwest). The `base` URL already carries
/// the DAV root path (e.g. `/remote.php/dav/files/alice/`) and always ends in a
/// `/`, so `url_for` can just append the resource's Ghost FTP path.
pub struct WebdavSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub base: Url,
    pub auth: WebdavAuth,
}

impl WebdavSession {
    /// Apply the session's credentials to a request builder.
    pub fn authed(&self, rb: RequestBuilder) -> RequestBuilder {
        match &self.auth {
            WebdavAuth::Basic { user, password } => rb.basic_auth(user, Some(password)),
            WebdavAuth::Bearer { token } => rb.bearer_auth(token),
            WebdavAuth::None => rb,
        }
    }

    /// Absolute URL for a Ghost FTP path like `/dir/file.txt`. Each segment is
    /// percent-encoded by the `url` crate. `dir` appends a trailing slash so
    /// collections address correctly (some servers 301 otherwise).
    pub fn url_for(&self, path: &str, dir: bool) -> Url {
        let mut u = self.base.clone();
        let rel = path.trim_start_matches('/');
        {
            // `base` ends in `/`, i.e. a trailing empty segment — drop it before
            // appending so we don't get a doubled slash.
            let mut segs = u
                .path_segments_mut()
                .expect("webdav base is always a valid http(s) URL");
            segs.pop_if_empty();
            // Skip empty and "." segments so a "." default remote path (or a
            // doubled slash) resolves to the DAV root, not a bogus child.
            for part in rel.split('/').filter(|s| !s.is_empty() && *s != ".") {
                segs.push(part);
            }
        }
        if dir && !u.path().ends_with('/') {
            u.set_path(&format!("{}/", u.path()));
        }
        u
    }

    /// The decoded path component of the base URL (the DAV root), used to strip
    /// the prefix off PROPFIND `href`s. Always ends in `/`.
    pub fn base_path(&self) -> String {
        let p = self.base.path();
        if p.ends_with('/') {
            p.to_string()
        } else {
            format!("{p}/")
        }
    }

    pub fn request(&self, method: Method, url: Url) -> RequestBuilder {
        self.authed(self.client.request(method, url))
    }
}

/// Connect a WebDAV session: parse the server URL, resolve the auth style, and
/// do a cheap `PROPFIND` depth-0 on the root so bad credentials / wrong URLs
/// fail here instead of on the first browse.
pub async fn webdav_connect(profile: &ConnectionProfile) -> Result<WebdavSession> {
    let raw = profile
        .endpoint
        .as_ref()
        .filter(|s| !s.trim().is_empty())
        .ok_or_else(|| anyhow!("WebDAV profile is missing the server URL"))?
        .trim()
        .to_string();

    // Normalize: ensure a scheme and a trailing slash so `url_for` joins cleanly.
    let with_scheme = if raw.contains("://") {
        raw.clone()
    } else {
        format!("https://{raw}")
    };
    let with_slash = if with_scheme.ends_with('/') {
        with_scheme
    } else {
        format!("{with_scheme}/")
    };
    let base = Url::parse(&with_slash).with_context(|| format!("parse WebDAV URL {raw}"))?;
    if !matches!(base.scheme(), "http" | "https") {
        return Err(anyhow!(
            "WebDAV URL must be http or https, got {}",
            base.scheme()
        ));
    }

    let password = match &profile.auth {
        AuthMethod::Password { password } => password.clone(),
        _ => {
            return Err(anyhow!(
                "WebDAV needs password/token auth. Use a username + password (Basic) \
                 or leave the username blank and paste a bearer token."
            ))
        }
    };
    let auth = if !profile.username.is_empty() {
        WebdavAuth::Basic {
            user: profile.username.clone(),
            password,
        }
    } else if !password.is_empty() {
        WebdavAuth::Bearer { token: password }
    } else {
        WebdavAuth::None
    };

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(120))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building WebDAV HTTP client")?;

    let session = WebdavSession {
        id: uuid::Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        base,
        auth,
    };

    // Validate credentials + URL up front with a depth-0 PROPFIND on the root.
    let url = session.url_for("/", true);
    let resp = session
        .request(Method::from_bytes(b"PROPFIND").unwrap(), url)
        .header("Depth", "0")
        .header("Content-Type", "application/xml")
        .body(PROPFIND_BODY)
        .send()
        .await
        .with_context(|| format!("PROPFIND {}", session.base))?;
    let status = resp.status();
    if status == reqwest::StatusCode::UNAUTHORIZED || status == reqwest::StatusCode::FORBIDDEN {
        return Err(anyhow!(
            "WebDAV authentication failed ({}). Check the username and password/token.",
            status.as_u16()
        ));
    }
    if !(status.is_success() || status.as_u16() == 207) {
        return Err(anyhow!(
            "WebDAV server returned {} for {} — check the URL path.",
            status.as_u16(),
            session.base
        ));
    }

    Ok(session)
}

/// Minimal PROPFIND body requesting just the props we render/sync on. Sending a
/// specific set (rather than `allprop`) keeps responses small on big folders.
pub const PROPFIND_BODY: &str = r#"<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:resourcetype/>
    <D:getcontentlength/>
    <D:getlastmodified/>
    <D:getetag/>
  </D:prop>
</D:propfind>"#;
