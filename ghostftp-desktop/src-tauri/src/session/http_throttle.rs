//! Shared pacing + retry for the throttled REST cloud backends (Shopify,
//! HubSpot): a token bucket of one refilled every `min_interval`, `Retry-After`
//! honored on 429, exponential backoff on 5xx. Extracted from
//! `session/shopify.rs` so every throttled backend paces identically — the
//! behavior here is byte-identical to what Shopify shipped with.

use anyhow::{Context, Result};
use reqwest::StatusCode;
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

/// Send an HTTP request through `throttle`, honoring `Retry-After` on 429 and
/// backing off on 5xx (up to 3 retries each). `build` must be re-runnable —
/// each attempt re-invokes it for a fresh `RequestBuilder` (auth headers,
/// bodies). `what` describes the call for the transport-error context
/// (`"shopify {path}"`-style).
pub async fn send_retried<F, Fut>(
    throttle: &HttpThrottle,
    what: &str,
    mut build: F,
) -> Result<reqwest::Response>
where
    F: FnMut() -> Fut,
    Fut: std::future::Future<Output = Result<reqwest::RequestBuilder>>,
{
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
        if status == StatusCode::TOO_MANY_REQUESTS && attempt < 3 {
            let wait = retry_after_delay(resp.headers().get("Retry-After"));
            tokio::time::sleep(wait).await;
            attempt += 1;
            continue;
        }
        if status.is_server_error() && attempt < 3 {
            tokio::time::sleep(backoff).await;
            backoff *= 2;
            attempt += 1;
            continue;
        }
        return Ok(resp);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reqwest::header::HeaderValue;

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
