use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use percent_encoding::percent_decode_str;
use reqwest::{Client, Method, RequestBuilder};
use std::time::Duration;
use url::Url;

/// Optional HTTP Basic auth for static servers sitting behind a login.
pub enum HttpAuth {
    Basic { user: String, password: String },
    None,
}

/// Whether the connection browses an autoindex listing or points straight at a
/// single file. Decided at connect from the URL shape (trailing slash / a
/// filename with an extension), so "paste a direct release-artifact URL" works
/// without a listing at all.
pub enum HttpMode {
    /// Autoindex listing under `base` (which always ends in `/`).
    Listing,
    /// The URL is one file; `name` is its display name.
    DirectFile { name: String },
}

/// A read-only HTTP(S) source. Like WebDAV, every op is an independent request
/// off a shared `reqwest::Client`; unlike WebDAV there are no mutations — this
/// backend only lists (autoindex) and reads (GET).
pub struct HttpSession {
    pub id: String,
    pub profile: ConnectionProfile,
    pub client: Client,
    pub base: Url,
    pub auth: HttpAuth,
    pub mode: HttpMode,
}

impl HttpSession {
    pub fn authed(&self, rb: RequestBuilder) -> RequestBuilder {
        match &self.auth {
            HttpAuth::Basic { user, password } => rb.basic_auth(user, Some(password)),
            HttpAuth::None => rb,
        }
    }

    pub fn request(&self, method: Method, url: Url) -> RequestBuilder {
        self.authed(self.client.request(method, url))
    }

    /// Absolute URL for a Ghost FTP path. In `DirectFile` mode every path resolves to
    /// the single file `base`; in `Listing` mode segments are appended (and
    /// percent-encoded) under the base directory.
    pub fn url_for(&self, path: &str, dir: bool) -> Url {
        if matches!(self.mode, HttpMode::DirectFile { .. }) {
            return self.base.clone();
        }
        let mut u = self.base.clone();
        let rel = path.trim_start_matches('/');
        {
            let mut segs = u
                .path_segments_mut()
                .expect("http base is always a valid http(s) URL");
            segs.pop_if_empty();
            for part in rel.split('/').filter(|s| !s.is_empty() && *s != ".") {
                segs.push(part);
            }
        }
        if dir && !u.path().ends_with('/') {
            u.set_path(&format!("{}/", u.path()));
        }
        u
    }
}

/// Connect a read-only HTTP source: parse the URL, decide listing vs single-file
/// mode, and probe reachability + auth with a cheap `HEAD` (falling back to
/// proceeding when a server rejects HEAD).
pub async fn http_connect(profile: &ConnectionProfile) -> Result<HttpSession> {
    let raw = profile
        .endpoint
        .as_ref()
        .filter(|s| !s.trim().is_empty())
        .ok_or_else(|| anyhow!("HTTP profile is missing the URL"))?
        .trim()
        .to_string();

    let with_scheme = if raw.contains("://") {
        raw.clone()
    } else {
        format!("https://{raw}")
    };
    let parsed = Url::parse(&with_scheme).with_context(|| format!("parse HTTP URL {raw}"))?;
    if !matches!(parsed.scheme(), "http" | "https") {
        return Err(anyhow!(
            "HTTP URL must be http or https, got {}",
            parsed.scheme()
        ));
    }

    // Mode: trailing slash → listing; a final segment with an extension → single
    // file; otherwise treat as a directory and add the slash.
    let ends_slash = parsed.path().ends_with('/');
    let last = parsed
        .path()
        .rsplit('/')
        .find(|s| !s.is_empty())
        .unwrap_or("")
        .to_string();
    let (base, mode) = if ends_slash {
        (parsed, HttpMode::Listing)
    } else if last.contains('.') {
        let name = percent_decode_str(&last).decode_utf8_lossy().into_owned();
        (parsed, HttpMode::DirectFile { name })
    } else {
        let u = Url::parse(&format!("{with_scheme}/"))
            .with_context(|| format!("parse HTTP URL {raw}"))?;
        (u, HttpMode::Listing)
    };

    let auth = if !profile.username.is_empty() {
        let password = match &profile.auth {
            AuthMethod::Password { password } => password.clone(),
            _ => String::new(),
        };
        HttpAuth::Basic {
            user: profile.username.clone(),
            password,
        }
    } else {
        HttpAuth::None
    };

    let client = Client::builder()
        .connect_timeout(Duration::from_secs(20))
        .timeout(Duration::from_secs(120))
        .user_agent(concat!("Ghost FTP/", env!("CARGO_PKG_VERSION")))
        .build()
        .context("building HTTP client")?;

    let session = HttpSession {
        id: uuid::Uuid::new_v4().to_string(),
        profile: profile.clone(),
        client,
        base,
        auth,
        mode,
    };

    // Reachability + auth probe. HEAD is cheap; a 405 (HEAD unsupported) is not a
    // failure — just proceed.
    let resp = session
        .request(Method::HEAD, session.base.clone())
        .send()
        .await
        .with_context(|| format!("HEAD {}", session.base))?;
    let status = resp.status();
    if status == reqwest::StatusCode::UNAUTHORIZED || status == reqwest::StatusCode::FORBIDDEN {
        return Err(anyhow!(
            "HTTP source requires authentication ({}). Add a username + password.",
            status.as_u16()
        ));
    }
    if !(status.is_success()
        || status.is_redirection()
        || status == reqwest::StatusCode::METHOD_NOT_ALLOWED)
    {
        return Err(anyhow!(
            "HTTP source returned {} for {} — check the URL.",
            status.as_u16(),
            session.base
        ));
    }

    Ok(session)
}
