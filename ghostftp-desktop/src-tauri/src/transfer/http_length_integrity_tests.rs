// Ghost FTP by Brendigo
use super::*;
use super::integrity::declared_plain_http_length;
use reqwest::header::{HeaderMap, HeaderValue, CONTENT_ENCODING, CONTENT_LENGTH};

#[test]
fn ftp_ftps_sftp_downloads_cannot_confirm_truncated_or_oversized_streams() {
    assert!(verify_committed_length("SFTP download", 4096, 2048, 2048).is_err());
    assert!(verify_committed_length("FTP download", 4096, 4097, 4097).is_err());
    assert!(verify_committed_length("FTPS download", 4096, 4096, 4095).is_err());
    assert!(verify_committed_length("SFTP download", 0, 0, 0).is_ok());
    assert!(verify_committed_length("FTPS download", 65537, 65537, 65537).is_ok());
}

#[test]
fn object_store_response_metadata_rejects_partial_and_extra_chunks() {
    // A missing or stale listing estimate must not let a truncated object
    // pass as complete: the GET's own size is the invariant.
    let expected = 4 * 1024 * 1024 + 3_u64;
    let first = checked_object_download_progress(expected, 0, 4 * 1024 * 1024)
        .expect("first object part fits");
    assert_eq!(first, 4 * 1024 * 1024);
    assert_eq!(
        checked_object_download_progress(expected, first, 3).unwrap(),
        expected
    );
    assert!(checked_object_download_progress(expected, first, 4).is_err());
    assert!(verify_committed_length("object", expected, first, first).is_err());
    assert!(verify_committed_length("object", expected, expected, first).is_err());
    assert!(verify_committed_length("object", expected, expected, expected).is_ok());
}

#[test]
fn zero_byte_objects_and_invalid_large_counters_fail_closed() {
    assert_eq!(checked_object_download_progress(0, 0, 0).unwrap(), 0);
    assert!(checked_object_download_progress(0, 0, 1).is_err());
    assert!(checked_object_download_progress(u64::MAX, u64::MAX, 1).is_err());
    assert!(verify_committed_length("object", 0, 0, 0).is_ok());
}

#[test]
fn known_size_is_required_but_an_unknown_estimate_never_fakes_zero_progress() {
    assert!(completed_transfer_size(100, 99).is_err());
    assert!(completed_transfer_size(100, 101).is_err());
    assert_eq!(completed_transfer_size(100, 100).unwrap(), 100);
    assert_eq!(
        completed_transfer_size(0, 128 * 1024 * 1024).unwrap(),
        128 * 1024 * 1024
    );
    assert!(completed_transfer_size(0, 0).is_ok());
    assert!(completed_transfer_size(u32::MAX as u64 + 10, u32::MAX as u64).is_err());
}

#[test]
fn remote_errors_never_persist_authorization_headers_or_signed_urls() {
    let error = anyhow::anyhow!(
        "server failure: Authorization: Bearer private_token \
https://alice:pass@cloud.example/file?x-amz-signature=secret"
    );
    let displayed = safe_transfer_failure(&error);
    assert!(!displayed.contains("private_token"));
    assert!(!displayed.contains("alice"));
    assert!(!displayed.contains("secret"));
    assert!(!displayed.contains("cloud.example"));
    let error = anyhow::anyhow!("401 unauthorized: token=private");
    assert!(safe_transfer_failure(&error).contains("Authentication"));
}

#[test]
fn truncated_and_extra_payloads_cannot_be_confirmed() {
    assert!(verify_plain_http_download_length(Some(100), 99, 99).is_err());
    assert!(verify_plain_http_download_length(Some(100), 101, 101).is_err());
    assert!(verify_plain_http_download_length(Some(100), 100, 99).is_err());
    assert!(verify_plain_http_download_length(Some(0), 0, 0).is_ok());
    assert!(verify_plain_http_download_length(Some(100), 100, 100).is_ok());
    assert!(verify_plain_http_download_length(None, 5, 5).is_ok());
}

#[test]
fn unexpected_partial_or_no_content_responses_never_commit_full_download() {
    use reqwest::header::CONTENT_RANGE;

    let mut headers = HeaderMap::new();
    headers.insert(CONTENT_LENGTH, HeaderValue::from_static("4"));
    assert_eq!(
        declared_full_http_download_length(reqwest::StatusCode::OK, &headers).unwrap(),
        Some(4)
    );
    for status in [
        reqwest::StatusCode::PARTIAL_CONTENT,
        reqwest::StatusCode::NO_CONTENT,
        reqwest::StatusCode::RESET_CONTENT,
        reqwest::StatusCode::NOT_MODIFIED,
        reqwest::StatusCode::RANGE_NOT_SATISFIABLE,
    ] {
        assert!(declared_full_http_download_length(status, &headers).is_err());
    }
    headers.insert(CONTENT_RANGE, HeaderValue::from_static("bytes 0-3/100"));
    assert!(declared_full_http_download_length(reqwest::StatusCode::OK, &headers).is_err());
}

#[test]
fn http_cloud_download_rejects_excess_before_disk_write_and_overflow() {
    assert_eq!(checked_http_download_progress(Some(6), 0, 4).unwrap(), 4);
    assert_eq!(checked_http_download_progress(Some(6), 4, 2).unwrap(), 6);
    assert!(checked_http_download_progress(Some(6), 4, 3).is_err());
    assert_eq!(checked_http_download_progress(None, 4, 3).unwrap(), 7);
    assert!(checked_http_download_progress(None, u64::MAX, 1).is_err());
    assert!(verify_plain_http_download_length(Some(6), 4, 4).is_err());
    assert!(verify_plain_http_download_length(Some(6), 6, 5).is_err());
}

#[tokio::test]
async fn unsolicited_http_range_response_is_rejected_before_body_download() {
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    let server = tokio::net::TcpListener::bind("127.0.0.1:0")
        .await
        .expect("bind loopback HTTP fixture");
    let url = format!("http://{}/download", server.local_addr().unwrap());
    let accept = tokio::spawn(async move {
        let (mut peer, _) = server.accept().await.expect("accept HTTP GET");
        let mut request = [0u8; 2048];
        let _ = peer.read(&mut request).await.expect("read HTTP GET");
        peer.write_all(
            b"HTTP/1.1 206 Partial Content\r\nContent-Range: bytes 0-3/100\r\nContent-Length: 4\r\nConnection: close\r\n\r\ndata",
        )
        .await
        .expect("return well-formed partial response");
    });
    let response = reqwest::Client::new()
        .get(&url)
        .send()
        .await
        .expect("make real loopback HTTP request");
    assert!(declared_full_http_download_length(response.status(), response.headers()).is_err());
    accept.await.expect("join HTTP fixture");
}

#[test]
fn encoded_or_invalid_length_headers_are_not_trusted() {
    let mut headers = HeaderMap::new();
    headers.insert(CONTENT_ENCODING, HeaderValue::from_static("gzip"));
    assert!(declared_plain_http_length(&headers).is_err());
    headers.insert(CONTENT_ENCODING, HeaderValue::from_static("identity"));
    headers.insert(CONTENT_LENGTH, HeaderValue::from_static("invalid"));
    assert!(declared_plain_http_length(&headers).is_err());
    headers.insert(CONTENT_LENGTH, HeaderValue::from_static("100"));
    assert_eq!(declared_plain_http_length(&headers).unwrap(), Some(100));
}

#[test]
fn ambiguous_duplicate_wire_metadata_fails_closed() {
    let mut headers = HeaderMap::new();
    headers.append(CONTENT_LENGTH, HeaderValue::from_static("10"));
    headers.append(CONTENT_LENGTH, HeaderValue::from_static("10"));
    assert!(declared_plain_http_length(&headers).is_err());
    assert!(declared_full_http_download_length(reqwest::StatusCode::OK, &headers).is_err());

    let mut headers = HeaderMap::new();
    headers.append(CONTENT_ENCODING, HeaderValue::from_static("identity"));
    headers.append(CONTENT_ENCODING, HeaderValue::from_static("identity"));
    assert!(declared_plain_http_length(&headers).is_err());
}
