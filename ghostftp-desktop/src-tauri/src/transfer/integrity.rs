use super::*;

/// Some HTTP/cloud backends cannot obtain a size before transfer and store 0.
/// In that case use the actual byte count; never overwrite real progress with
/// a guessed size. When size was known, a short/oversized transfer is an error.
pub(super) fn completed_transfer_size(expected: u64, actual: u64) -> Result<u64> {
    if expected != 0 {
        anyhow::ensure!(
            actual == expected,
            "transfer finished with {actual} of {expected} expected bytes"
        );
    }
    Ok(actual)
}

/// Object-store GET metadata comes from the response being streamed, not an
/// earlier directory listing. Guard each part before writing: a broken or
/// malicious backend must not consume unlimited disk after exceeding its own
/// declared object length. Zero-length objects are verified normally.
pub(super) fn checked_object_download_progress(expected: u64, written: u64, next_bytes: usize) -> Result<u64> {
    let next_bytes = u64::try_from(next_bytes).context("object chunk length overflow")?;
    let next = written
        .checked_add(next_bytes)
        .context("object download byte counter overflow")?;
    anyhow::ensure!(
        next <= expected,
        "object download exceeded the declared object size"
    );
    Ok(next)
}

/// Full-file HTTP and cloud downloads do not send a Range header. An
/// unsolicited 206 has a valid *partial* Content-Length, which must not be
/// treated as the entire file. Other 2xx statuses (for example 204) do not
/// provide a complete representation either. Check before creating staging.
pub(super) fn declared_full_http_download_length(
    status: reqwest::StatusCode,
    headers: &reqwest::header::HeaderMap,
) -> Result<Option<u64>> {
    anyhow::ensure!(
        status == reqwest::StatusCode::OK,
        "download requires HTTP 200 complete representation, received HTTP {}",
        status.as_u16()
    );
    anyhow::ensure!(
        !headers.contains_key(reqwest::header::CONTENT_RANGE),
        "download returned unexpected Content-Range without a Range request"
    );
    declared_plain_http_length(headers)
}

/// Reject oversized HTTP/cloud payloads *before* writing them to the staging
/// file, rather than merely noticing at EOF after consuming disk space.
pub(super) fn checked_http_download_progress(
    expected: Option<u64>,
    written: u64,
    next_bytes: usize,
) -> Result<u64> {
    let next = written
        .checked_add(u64::try_from(next_bytes).context("HTTP chunk length overflow")?)
        .context("HTTP download byte counter overflow")?;
    if let Some(expected) = expected {
        anyhow::ensure!(
            next <= expected,
            "HTTP download exceeds its declared Content-Length"
        );
    }
    Ok(next)
}

/// The declared wire size is meaningful only for identity-encoded HTTP.
pub(super) fn declared_plain_http_length(headers: &reqwest::header::HeaderMap) -> Result<Option<u64>> {
    use reqwest::header::{CONTENT_ENCODING, CONTENT_LENGTH};
    let mut encodings = headers.get_all(CONTENT_ENCODING).iter();
    if let Some(encoding) = encodings.next() {
        anyhow::ensure!(
            encodings.next().is_none(),
            "HTTP transfer returned ambiguous Content-Encoding headers"
        );
        anyhow::ensure!(
            encoding
                .to_str()
                .is_ok_and(|s| s.eq_ignore_ascii_case("identity")),
            "HTTP transfer returned an encoded body whose source size cannot be verified"
        );
    }
    let mut lengths = headers.get_all(CONTENT_LENGTH).iter();
    let wire_length = lengths.next();
    anyhow::ensure!(
        lengths.next().is_none(),
        "HTTP transfer returned ambiguous Content-Length headers"
    );
    wire_length
        .map(|value| {
            value
                .to_str()
                .context("invalid HTTP Content-Length header")?
                .parse::<u64>()
                .context("invalid HTTP Content-Length value")
        })
        .transpose()
}

/// A missing declared length remains unverified at the remote level; this
/// guard only proves the local file persisted the bytes received.
pub(super) fn verify_plain_http_download_length(
    expected: Option<u64>,
    transferred: u64,
    on_disk: u64,
) -> Result<()> {
    anyhow::ensure!(
        on_disk == transferred,
        "HTTP download staging length differs from received bytes"
    );
    if let Some(expected) = expected {
        anyhow::ensure!(
            transferred == expected,
            "HTTP download length mismatch: incomplete or oversized payload"
        );
    }
    Ok(())
}

