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
            let wait = resp
                .headers()
                .get("Retry-After")
                .and_then(|v| v.to_str().ok())
                .and_then(|s| s.parse::<f64>().ok())
                .unwrap_or(2.0);
            tokio::time::sleep(Duration::from_secs_f64(wait.max(0.1))).await;
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
