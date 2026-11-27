//! Manual release discovery and credential-free update recovery journal.
use serde::{Deserialize, Serialize};
use std::time::Duration;

const CATALOG_URL: &str = "https://update.ghostftp.com/manual-version.json";
const MAX_RESPONSE: usize = 4096;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Catalog {
    version: String,
}

pub(crate) fn version_parts(value: &str) -> Option<[u32; 3]> {
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
    let catalog: Catalog =
        serde_json::from_slice(bytes).map_err(|_| "Invalid manual update response".to_string())?;
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

// Diagnostic intent only: the signed Tauri updater remains the trust boundary.
// Never persist installer URLs, credentials or executable paths in this record.
pub(crate) const UPDATE_ATTEMPT_KEY: &str = "nativeUpdateAttemptV1";

pub(crate) fn ensure_public_setting(key: &str) -> Result<(), String> {
    if key == UPDATE_ATTEMPT_KEY {
        return Err("Reserved native update state".into());
    }
    Ok(())
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct UpdateAttempt {
    source: String,
    target: String,
}

fn update_attempt(source: &str, target: &str) -> Result<UpdateAttempt, String> {
    let source_parts = version_parts(source).ok_or("Invalid installed version")?;
    let target_parts = version_parts(target).ok_or("Invalid update version")?;
    if target_parts <= source_parts {
        return Err("Update version must be newer than the installed version".into());
    }
    Ok(UpdateAttempt {
        source: source.into(),
        target: target.into(),
    })
}

pub(crate) fn record_attempt(db: &crate::db::Db, source: &str, target: &str) -> Result<(), String> {
    let attempt = update_attempt(source, target)?;
    let value = serde_json::to_string(&attempt).map_err(|_| "Cannot record update attempt")?;
    db.prepare_for_update(UPDATE_ATTEMPT_KEY, &value)
        .map_err(|_| "Cannot persist or checkpoint update recovery state".into())
}

fn recovery_notice(db: &crate::db::Db, current: &str) -> Result<Option<String>, String> {
    // Unlike the historical single-setting helper, this propagates DB read errors.
    let settings = db
        .settings_get_all()
        .map_err(|_| "Cannot read update recovery status")?;
    let Some(value) = settings.get(UPDATE_ATTEMPT_KEY) else {
        return Ok(None);
    };
    if value.len() > MAX_RESPONSE {
        return Err("Invalid update recovery record".into());
    }
    let attempt: UpdateAttempt =
        serde_json::from_str(value).map_err(|_| "Invalid update recovery record")?;
    let attempt = update_attempt(&attempt.source, &attempt.target)?;
    let installed = version_parts(current).ok_or("Invalid installed version")?;
    let target = version_parts(&attempt.target).ok_or("Invalid recorded update version")?;
    if installed >= target {
        // A version match alone is not evidence that the database survived.
        // Leave the journal intact; this is only a read-only SQLite health check.
        db.check_update_database_health()
            .map_err(|_| "Cannot verify database health after update")?;
        return Ok(None);
    }
    Ok(Some(format!(
        "A previous update targeted Ghost FTP {} but this installation is {}. Check your installation and backups before retrying.",
        attempt.target, current
    )))
}

#[tauri::command]
pub fn update_recovery_notice(
    state: tauri::State<'_, crate::AppState>,
) -> Result<Option<String>, String> {
    recovery_notice(&state.db, env!("CARGO_PKG_VERSION"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn native_update_record_is_not_a_public_setting() {
        assert!(ensure_public_setting(UPDATE_ATTEMPT_KEY).is_err());
        assert!(ensure_public_setting("appTheme").is_ok());
    }

    #[test]
    fn pending_restore_prevents_update_without_removing_staged_files() {
        let dir = std::env::temp_dir().join(format!("ghostftp-update-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        assert!(crate::backup::ensure_no_pending_restore(&dir).is_ok());
        let staged = dir.join("ghostftp.db.restore");
        std::fs::write(&staged, b"staged").unwrap();
        assert!(crate::backup::ensure_no_pending_restore(&dir).is_err());
        assert_eq!(std::fs::read(&staged).unwrap(), b"staged");
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[test]
    fn recovery_intent_survives_database_reopen_without_mutating_user_data() {
        let dir = std::env::temp_dir().join(format!("ghostftp-update-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        let path = dir.join("ghostftp.db");
        {
            let db = crate::db::Db::open(&path).unwrap();
            db.settings_set("testUserPreference", "\"dark\"").unwrap();
            assert!(recovery_notice(&db, "0.99.91").unwrap().is_none());
            assert!(record_attempt(&db, "0.99.91", "0.99.90").is_err());
            record_attempt(&db, "0.99.91", "0.99.92").unwrap();
        }
        {
            let db = crate::db::Db::open(&path).unwrap();
            assert!(recovery_notice(&db, "0.99.91").unwrap().is_some());
            assert!(recovery_notice(&db, "0.99.92").unwrap().is_none());
            assert!(db.check_update_database_health().is_ok());
            assert!(db.settings_get(UPDATE_ATTEMPT_KEY).unwrap().is_some());
            assert_eq!(
                db.settings_get("testUserPreference").unwrap().as_deref(),
                Some("\"dark\"")
            );
            db.settings_set(UPDATE_ATTEMPT_KEY, "invalid").unwrap();
            assert!(recovery_notice(&db, "0.99.91").is_err());
        }
        std::fs::remove_dir_all(dir).unwrap();
    }

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
