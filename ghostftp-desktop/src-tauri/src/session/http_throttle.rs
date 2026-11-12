//! Shared pacing + retry for the throttled REST cloud backends (Shopify,
//! HubSpot): a token bucket of one refilled every `min_interval`, `Retry-After`
//! honored on 429, exponential backoff on 5xx. Extracted from
//! `session/shopify.rs` so every throttled backend paces identically — the
//! behavior here is byte-identical to what Shopify shipped with.

use anyhow::{Context, Result};
use reqwest::{Method, StatusCode};
use std::sync::Mutex as StdMutex;
use std::time::{Duration, Instant};

/// Pacing state for one backend: the next instant a request may leave, plus
/// the refill interval. A token bucket of one refilled every `min_interval`.
pub struct HttpThrottle {
    next_request: StdMutex<Instant>,
    min_interval: Duration,
}

impl HttpThrottle {
    pub fn new(min_interval: Duration) -> Self {
        Self {
            next_request: StdMutex::new(Instant::now()),
            min_interval,
        }
    }

    /// Wait for this request's slot, then reserve the next one.
    pub async fn wait(&self) {
        let wait = {
            let mut next = self.next_request.lock().unwrap();
            let now = Instant::now();
            let at = now.max(*next);
            *next = at + self.min_interval;
            at.saturating_duration_since(now)
        };
        if !wait.is_zero() {
            tokio::time::sleep(wait).await;
        }
    }
}

/// Never feed an untrusted Retry-After value directly into
/// Duration::from_secs_f64: NaN, infinities and overflow can panic or stall
/// retries for years. The common backoff budget is bounded to one minute.
fn retry_after_delay(header: Option<&reqwest::header::HeaderValue>) -> Duration {
    let seconds = header
        .and_then(|value| value.to_str().ok())
        .and_then(|value| value.parse::<f64>().ok())
        .filter(|value| value.is_finite())
        .unwrap_or(2.0)
        .clamp(0.1, 60.0);
    Duration::from_secs_f64(seconds)
}

/// Fail closed for provider writes: a 429/5xx does not prove a POST/PUT/PATCH/
/// DELETE was not applied. Retries could create duplicate remote data.
pub(crate) fn can_retry_method(method: &Method) -> bool {
    *method == Method::GET || *method == Method::HEAD || *method == Method::OPTIONS
}

/// Send an HTTP request through `throttle`, honoring `Retry-After` on 429 and
/// backing off on 5xx (up to 3 retries each). `build` must be re-runnable —
/// each attempt re-invokes it for a fresh `RequestBuilder` (auth headers,
/// bodies). `what` describes the call for the transport-error context
/// (`"shopify {path}"`-style).
pub async fn send_retried<F, Fut>(
    throttle: &HttpThrottle,
    what: &str,
    method: Method,
    mut build: F,
) -> Result<reqwest::Response>
where
    F: FnMut() -> Fut,
    Fut: std::future::Future<Output = Result<reqwest::RequestBuilder>>,
{
    let replay_safe = can_retry_method(&method);
    let mut backoff = Duration::from_millis(500);
    let mut attempt = 0;
    loop {
        throttle.wait().await;
        let resp = build()
            .await?
            .send()
            .await
            .with_context(|| what.to_string())?;
        let status = resp.status();
        if replay_safe && status == StatusCode::TOO_MANY_REQUESTS && attempt < 3 {
            let wait = retry_after_delay(resp.headers().get("Retry-After"));
            tokio::time::sleep(wait).await;
            attempt += 1;
            continue;
        }
        if replay_safe && status.is_server_error() && attempt < 3 {
            tokio::time::sleep(backoff).await;
            backoff *= 2;
            attempt += 1;
            continue;
        }
        return Ok(resp);
    }
}

/// Resolve an API path or provider pagination URL without ever attaching a
/// bearer token to an attacker-controlled destination. Provider test fixtures
/// may use an explicitly configured loopback HTTP base; production endpoints
/// require HTTPS and cannot silently downgrade.
pub fn trusted_bearer_url(base: &str, target: &str) -> Result<reqwest::Url> {
    use anyhow::anyhow;
    let endpoint =
        reqwest::Url::parse(base).map_err(|_| anyhow!("invalid provider API base URL"))?;
    let trusted_loopback = endpoint.scheme() == "http"
        && matches!(endpoint.host_str(), Some("127.0.0.1" | "localhost" | "::1"));
    anyhow::ensure!(
        endpoint.scheme() == "https" || trusted_loopback,
        "provider API base must use HTTPS"
    );
    anyhow::ensure!(
        endpoint.username().is_empty()
            && endpoint.password().is_none()
            && endpoint.fragment().is_none(),
        "provider API base contains forbidden credentials or fragment"
    );
    let url = if target.starts_with('/') && !target.starts_with("//") {
        reqwest::Url::parse(&format!("{base}{target}"))
            .map_err(|_| anyhow!("invalid relative provider request path"))?
    } else {
        reqwest::Url::parse(target).map_err(|_| anyhow!("invalid absolute provider request URL"))?
    };
    anyhow::ensure!(
        url.origin() == endpoint.origin()
            && url.username().is_empty()
            && url.password().is_none()
            && url.fragment().is_none(),
        "provider response attempted an untrusted cross-origin or credentialed URL"
    );
    Ok(url)
}

const MAX_CLOUD_METADATA_BYTES: usize = 16 * 1024 * 1024;

/// A bounded JSON reader shared by cloud API adapters. No provider response
/// bytes or signed URLs are included in failure diagnostics.
pub async fn bounded_cloud_json(mut response: reqwest::Response) -> Result<serde_json::Value> {
    anyhow::ensure!(
        response
            .content_length()
            .is_none_or(|size| size <= MAX_CLOUD_METADATA_BYTES as u64),
        "cloud metadata response exceeds size limit"
    );
    let mut bytes = Vec::new();
    while let Some(chunk) = response
        .chunk()
        .await
        .context("stream cloud metadata response")?
    {
        anyhow::ensure!(
            chunk.len() <= MAX_CLOUD_METADATA_BYTES.saturating_sub(bytes.len()),
            "cloud metadata response exceeds size limit"
        );
        bytes.extend_from_slice(&chunk);
    }
    if bytes.iter().all(u8::is_ascii_whitespace) {
        return Ok(serde_json::Value::Null);
    }
    serde_json::from_slice(&bytes).context("invalid cloud metadata JSON")
}

#[cfg(test)]
mod tests {
    use super::*;
    use reqwest::header::HeaderValue;

    #[test]
    fn cloud_bearer_urls_never_leave_configured_origin_or_downgrade() {
        let base = "https://graph.microsoft.com/v1.0";
        for good in [
            "/me/drive",
            "https://graph.microsoft.com/v1.0/me/drive?skiptoken=abc",
        ] {
            assert!(trusted_bearer_url(base, good).is_ok(), "{good}");
        }
        for bad in [
            "http://graph.microsoft.com/v1.0",
            "https://graph.microsoft.com.evil.test/v1.0",
            "https://evil.test/token",
            "https://user:password@graph.microsoft.com/v1.0",
            "https://graph.microsoft.com/v1.0#fragment",
            "//evil.test/path",
            "ftp://graph.microsoft.com/v1.0",
        ] {
            assert!(trusted_bearer_url(base, bad).is_err(), "{bad}");
        }
        assert!(trusted_bearer_url("http://127.0.0.1:9911/api", "/items").is_ok());
        assert!(
            trusted_bearer_url("http://127.0.0.1:9911/api", "http://127.0.0.1:9912/api").is_err()
        );
        assert!(trusted_bearer_url("http://remote.example/api", "/items").is_err());
    }

    #[test]
    fn only_safe_read_methods_are_automatically_retried() {
        for method in [Method::GET, Method::HEAD, Method::OPTIONS] {
            assert!(can_retry_method(&method));
        }
        for method in [Method::POST, Method::PUT, Method::PATCH, Method::DELETE] {
            assert!(!can_retry_method(&method));
        }
    }

    #[tokio::test]
    async fn provider_post_receiving_503_is_never_resent() {
        use tokio::io::{AsyncReadExt, AsyncWriteExt};
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
            .await
            .expect("bind loopback HTTP fixture");
        let address = listener.local_addr().expect("fixture address");
        let server = tokio::spawn(async move {
            let (mut stream, _) = listener.accept().await.expect("accept request");
            // Consume the full HTTP request head before responding. Closing
            // a socket with unread request data causes WSAECONNRESET on Windows,
            // which tests TCP teardown instead of retry/no-retry semantics.
            let mut request = Vec::with_capacity(512);
            let mut byte = [0u8; 1];
            for _ in 0..2048 {
                stream
                    .read_exact(&mut byte)
                    .await
                    .expect("read HTTP request header");
                request.push(byte[0]);
                if request.ends_with(b"\r\n\r\n") {
                    break;
                }
            }
            assert!(request.ends_with(b"\r\n\r\n"), "complete HTTP headers");
            stream
                .write_all(b"HTTP/1.1 503 Service Unavailable\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
                .await
                .expect("send provider failure");
            stream.shutdown().await.expect("finish provider response");
        });
        let url = format!("http://{address}/write");
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(2))
            .build()
            .unwrap();
        let throttle = HttpThrottle::new(Duration::ZERO);
        let response = send_retried(&throttle, "test write", Method::POST, || {
            let client = client.clone();
            let url = url.clone();
            async move { Ok(client.post(url)) }
        })
        .await
        .expect("do not retry post after response");
        assert_eq!(response.status(), StatusCode::SERVICE_UNAVAILABLE);
        server.await.unwrap();
    }

    #[test]
    fn server_retry_after_is_finite_and_bounded_without_panicking() {
        assert_eq!(retry_after_delay(None), Duration::from_secs(2));
        assert_eq!(
            retry_after_delay(Some(&HeaderValue::from_static("0"))),
            Duration::from_millis(100)
        );
        assert_eq!(
            retry_after_delay(Some(&HeaderValue::from_static("infinity"))),
            Duration::from_secs(2)
        );
        assert_eq!(
            retry_after_delay(Some(&HeaderValue::from_static("NaN"))),
            Duration::from_secs(2)
        );
        assert_eq!(
            retry_after_delay(Some(&HeaderValue::from_static("1e300"))),
            Duration::from_secs(60)
        );
        assert_eq!(
            retry_after_delay(Some(&HeaderValue::from_static("-1000"))),
            Duration::from_millis(100)
        );
    }
}
