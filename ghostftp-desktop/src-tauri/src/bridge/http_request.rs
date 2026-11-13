//! Strict bounded HTTP/1.x framing for the localhost Agent Bridge.
//! No intermediary-compatible header ambiguity, lossy decoding, or chunked bodies.
use super::{MAX_BODY, MAX_HEADERS};
use anyhow::{anyhow, Context, Result};
use tokio::io::AsyncReadExt;
use tokio::net::TcpStream;

pub(super) struct Request {
    pub(super) method: String,
    pub(super) path: String,
    pub(super) auth: Option<String>,
    pub(super) has_origin: bool,
    pub(super) body: Vec<u8>,
}

fn find_subslice(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack.windows(needle.len()).position(|w| w == needle)
}

pub(super) async fn read_request(stream: &mut TcpStream) -> Result<Request> {
    let mut buf = Vec::new();
    let mut tmp = [0u8; 4096];
    let header_end = loop {
        if let Some(pos) = find_subslice(&buf, b"\r\n\r\n") {
            if pos + 4 > MAX_HEADERS {
                return Err(anyhow!("request headers too large"));
            }
            break pos + 4;
        }
        if buf.len() > MAX_HEADERS {
            return Err(anyhow!("request headers too large"));
        }
        let n = stream.read(&mut tmp).await?;
        if n == 0 {
            return Err(anyhow!("connection closed before headers"));
        }
        buf.extend_from_slice(&tmp[..n]);
    };

    // Lossy decoding could turn malformed security-relevant header bytes into
    // different text. Local native callers must send valid UTF-8 headers.
    let header_text =
        std::str::from_utf8(&buf[..header_end]).context("invalid HTTP header encoding")?;
    let mut lines = header_text.split("\r\n");
    let request_line = lines.next().unwrap_or("");
    let mut parts = request_line.split_whitespace();
    let method = parts.next().unwrap_or("").to_string();
    let path = parts.next().unwrap_or("").to_string();
    let version = parts.next().unwrap_or("");
    anyhow::ensure!(
        parts.next().is_none()
            && matches!(version, "HTTP/1.0" | "HTTP/1.1")
            && !method.is_empty()
            && method.bytes().all(|ch| ch.is_ascii_uppercase())
            && path.starts_with('/'),
        "invalid HTTP request line"
    );

    let mut content_length: Option<usize> = None;
    let mut auth = None;
    let mut has_origin = false;
    for line in lines {
        if line.is_empty() {
            continue;
        }
        // Never interpret obs-fold or malformed fields as a second framing
        // protocol. Do not silently normalize whitespace before a field name.
        anyhow::ensure!(
            !line.starts_with(' ') && !line.starts_with('\t'),
            "folded HTTP headers are not supported"
        );
        let (k, v) = line.split_once(':').context("malformed HTTP header")?;
        anyhow::ensure!(
            !k.is_empty()
                && k.trim() == k
                && k.bytes().all(|ch| ch.is_ascii_graphic() && ch != b':'),
            "invalid HTTP header name"
        );
        anyhow::ensure!(
            v.bytes()
                .all(|ch| ch == b'\t' || (ch >= 0x20 && ch != 0x7f)),
            "invalid HTTP header value"
        );
        let key = k.to_ascii_lowercase();
        let val = v.trim();
        match key.as_str() {
            "content-length" => {
                if content_length.is_some() {
                    return Err(anyhow!("duplicate Content-Length"));
                }
                anyhow::ensure!(
                    !val.is_empty() && val.bytes().all(|digit| digit.is_ascii_digit()),
                    "invalid Content-Length"
                );
                content_length = Some(val.parse().context("invalid Content-Length")?);
            }
            "transfer-encoding" => {
                // Chunked requests are not implemented. Reject rather than
                // disagreeing with HTTP intermediaries on the body length.
                return Err(anyhow!("Transfer-Encoding is not supported"));
            }
            "origin" => has_origin = true,
            "authorization" => {
                if auth.is_some() {
                    return Err(anyhow!("duplicate Authorization"));
                }
                auth = Some(val.to_string());
            }
            _ => {}
        }
    }
    if method == "POST" && content_length.is_none() {
        return Err(anyhow!("POST requires Content-Length"));
    }
    let content_length = content_length.unwrap_or(0);
    if content_length > MAX_BODY {
        return Err(anyhow!("request body too large"));
    }

    let mut body = buf[header_end..].to_vec();
    while body.len() < content_length {
        let n = stream.read(&mut tmp).await?;
        if n == 0 {
            break;
        }
        body.extend_from_slice(&tmp[..n]);
    }
    if body.len() < content_length {
        return Err(anyhow!("truncated request body"));
    }
    body.truncate(content_length);

    Ok(Request {
        method,
        path,
        auth,
        has_origin,
        body,
    })
}
