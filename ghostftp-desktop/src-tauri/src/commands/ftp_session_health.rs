//! FTP/FTPS liveness: a dropped idle socket should not make the user reconnect
//! manually or silently remove a live tab. One bounded reconnect per probe.
use crate::session::FtpSession;
use anyhow::{Context, Result};
use std::future::Future;

async fn probe_with_one_reconnect<P, R, PF, RF>(
    mut probe: P,
    reconnect: R,
) -> Result<()>
where
    P: FnMut() -> PF,
    R: FnOnce() -> RF,
    PF: Future<Output = Result<()>>,
    RF: Future<Output = Result<()>>,
{
    if probe().await.is_ok() {
        return Ok(());
    }
    reconnect()
        .await
        .context("FTP/FTPS reconnect after failed health probe")?;
    probe()
        .await
        .context("FTP/FTPS health probe still failed after reconnect")
}

pub(super) async fn check_ftp_session(session: &FtpSession) -> Result<()> {
    probe_with_one_reconnect(
        || session.with_stream(|stream| stream.noop()),
        || session.reconnect(),
    )
    .await
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[tokio::test]
    async fn healthy_session_never_reconnects() {
        let checks = AtomicUsize::new(0);
        let reconnects = AtomicUsize::new(0);
        probe_with_one_reconnect(
            || {
                checks.fetch_add(1, Ordering::SeqCst);
                async { Ok(()) }
            },
            || {
                reconnects.fetch_add(1, Ordering::SeqCst);
                async { Ok(()) }
            },
        )
        .await
        .unwrap();
        assert_eq!(checks.load(Ordering::SeqCst), 1);
        assert_eq!(reconnects.load(Ordering::SeqCst), 0);
    }

    #[tokio::test]
    async fn stale_idle_socket_is_probed_again_on_new_connection() {
        let checks = AtomicUsize::new(0);
        let reconnects = AtomicUsize::new(0);
        probe_with_one_reconnect(
            || {
                let attempt = checks.fetch_add(1, Ordering::SeqCst);
                async move {
                    if attempt == 0 {
                        anyhow::bail!("stale socket")
                    }
                    Ok(())
                }
            },
            || {
                reconnects.fetch_add(1, Ordering::SeqCst);
                async { Ok(()) }
            },
        )
        .await
        .unwrap();
        assert_eq!(checks.load(Ordering::SeqCst), 2);
        assert_eq!(reconnects.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn failed_reconnect_is_not_reported_as_healthy_or_retried_forever() {
        let checks = AtomicUsize::new(0);
        let reconnects = AtomicUsize::new(0);
        let result = probe_with_one_reconnect(
            || {
                checks.fetch_add(1, Ordering::SeqCst);
                async { anyhow::bail!("offline") }
            },
            || {
                reconnects.fetch_add(1, Ordering::SeqCst);
                async { anyhow::bail!("cannot reconnect") }
            },
        )
        .await;
        assert!(result.is_err());
        assert_eq!(checks.load(Ordering::SeqCst), 1);
        assert_eq!(reconnects.load(Ordering::SeqCst), 1);
    }
}
