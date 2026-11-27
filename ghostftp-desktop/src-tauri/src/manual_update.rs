//! Manual release discovery only; this module never downloads or executes installers.
use serde::Deserialize;
use std::time::Duration;

const CATALOG_URL: &str = "https://update.ghostftp.com/manual-version.json";
const MAX_RESPONSE: usize = 4096;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Catalog {
    version: String,
}

fn version_parts(value: &str) -> Option<[u32; 3]> {
    let mut parts = value.split('.');
    let mut result = [0; 3];
    for part in &mut result {
        let raw = parts.next()?;
        if raw.is_empty()
            || raw.len() > 10
            || !raw.bytes().all(|byte| byte.is_ascii_digit())
            || (raw.len() > 1 && raw.starts_with('0'))
        {
            return None;
        }
        *part = raw.parse().ok()?;
    }
    if parts.next().is_some() {
        return None;
    }
    Some(result)
}

fn newer_version(bytes: &[u8], current: &str) -> Result<Option<String>, String> {
    if bytes.len() > MAX_RESPONSE {
        return Err("Manual update response is too large".into());
    }
    let catalog: Catalog = serde_json::from_slice(bytes)
        .map_err(|_| "Invalid manual update response".to_string())?;
    let offered = version_parts(&catalog.version)
        .ok_or_else(|| "Invalid manual update version".to_string())?;
    let installed =
        version_parts(current).ok_or_else(|| "Invalid installed version".to_string())?;
    Ok((offered > installed).then_some(catalog.version))
}

#[tauri::command]
pub async fn check_manual_update() -> Result<Option<String>, String> {
    let client = reqwest::Client::builder()
        .https_only(true)
        .redirect(reqwest::redirect::Policy::none())
        .connect_timeout(Duration::from_secs(3))
        .timeout(Duration::from_secs(6))
        .build()
        .map_err(|_| "Manual update service is unavailable".to_string())?;
    let mut response = client
        .get(CATALOG_URL)
        .header(reqwest::header::ACCEPT, "application/json")
        .send()
        .await
        .map_err(|_| "Manual update service is unavailable".to_string())?;
    if response.status() == reqwest::StatusCode::NO_CONTENT {
        return Ok(None);
    }
    if response.status() != reqwest::StatusCode::OK
        || response
            .content_length()
            .is_some_and(|size| size > MAX_RESPONSE as u64)
    {
        return Err("Invalid manual update response".into());
    }
    let mut body = Vec::new();
    while let Some(chunk) = response
        .chunk()
        .await
        .map_err(|_| "Manual update service is unavailable".to_string())?
    {
        if chunk.len() > MAX_RESPONSE - body.len() {
            return Err("Manual update response is too large".into());
        }
        body.extend_from_slice(&chunk);
    }
    newer_version(&body, env!("CARGO_PKG_VERSION"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn compares_numeric_versions_without_downgrade() {
        assert_eq!(
            newer_version(br#"{"version":"0.99.100"}"#, "0.99.99").unwrap(),
            Some("0.99.100".into())
        );
        assert_eq!(
            newer_version(br#"{"version":"0.99.90"}"#, "0.99.90").unwrap(),
            None
        );
        assert_eq!(
            newer_version(br#"{"version":"0.99.89"}"#, "0.99.90").unwrap(),
            None
        );
    }

    #[test]
    fn rejects_bad_versions_and_unexpected_payload() {
        for bytes in [
            br#"{"version":"0.99.090"}"#.as_slice(),
            br#"{"version":"1.0.0-rc.1"}"#,
            br#"{"version":"4294967296.0.0"}"#,
            br#"{"version":"1.0.0","url":"https://evil.example/"}"#,
            br#"{"version":"1.0.0","version":"2.0.0"}"#,
            br#"{"version":123}"#,
            b"<html>not a catalogue</html>",
        ] {
            assert!(newer_version(bytes, "0.99.90").is_err());
        }
        assert!(newer_version(&vec![b' '; MAX_RESPONSE + 1], "0.99.90").is_err());
    }
}
