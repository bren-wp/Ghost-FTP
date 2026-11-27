//! The WebView can request a version, not an installer, URL, key or signature.
//! Only the native signed updater may execute a package.
use crate::{manual_update, AppState};
use serde::Serialize;
use std::time::Duration;
use tauri::{ipc::Channel, AppHandle, Manager, State};
use tauri_plugin_updater::{Update, UpdaterExt};

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateOffer {
    version: String,
    current_version: String,
    body: Option<String>,
}

#[derive(Serialize)]
pub struct UpdateFailure {
    phase: &'static str,
    message: &'static str,
}

impl UpdateFailure {
    fn new(phase: &'static str, message: &'static str) -> Self {
        Self { phase, message }
    }
}

#[derive(Clone, Serialize)]
pub struct DownloadProgress {
    downloaded: u64,
    total: Option<u64>,
}

fn newer_target(current: &str, target: &str) -> bool {
    match (
        manual_update::version_parts(current),
        manual_update::version_parts(target),
    ) {
        (Some(current), Some(target)) => target > current,
        _ => false,
    }
}

async fn signed_offer(app: &AppHandle, timeout: Duration) -> Result<Option<Update>, UpdateFailure> {
    let updater = app
        .updater_builder()
        .timeout(timeout)
        .configure_client(|builder| builder.https_only(true))
        .build()
        .map_err(|_| UpdateFailure::new("check", "Update service is unavailable"))?;
    let offered = updater
        .check()
        .await
        .map_err(|_| UpdateFailure::new("check", "Update check failed"))?;
    match offered {
        Some(update)
            if newer_target(env!("CARGO_PKG_VERSION"), &update.version)
                && update.download_url.scheme() == "https" =>
        {
            Ok(Some(update))
        }
        Some(_) => Err(UpdateFailure::new("check", "Invalid update offer")),
        None => Ok(None),
    }
}

#[tauri::command]
pub async fn check_signed_update(app: AppHandle) -> Result<Option<UpdateOffer>, UpdateFailure> {
    Ok(signed_offer(&app, Duration::from_secs(15))
        .await?
        .map(|update| UpdateOffer {
            version: update.version,
            current_version: update.current_version,
            body: update.body,
        }))
}

// No network download or installation can be started concurrently through IPC.
static INSTALL_SERIAL: tokio::sync::Mutex<()> = tokio::sync::Mutex::const_new(());

#[tauri::command]
pub async fn install_signed_update(
    app: AppHandle,
    state: State<'_, AppState>,
    target: String,
    progress: Channel<DownloadProgress>,
) -> Result<(), UpdateFailure> {
    let current = env!("CARGO_PKG_VERSION");
    if !newer_target(current, &target) {
        return Err(UpdateFailure::new("check", "Invalid update version"));
    }
    let _serial = INSTALL_SERIAL
        .try_lock()
        .map_err(|_| UpdateFailure::new("prepare", "Another update is in progress"))?;
    // The target selected by the user must still match a fresh native offer.
    let update = signed_offer(&app, Duration::from_secs(300))
        .await?
        .filter(|update| update.version == target)
        .ok_or_else(|| UpdateFailure::new("check", "Update changed; check again"))?;
    let mut downloaded = 0u64;
    let bytes = update
        .download(
            |chunk, total| {
                downloaded = downloaded.saturating_add(chunk as u64);
                let _ = progress.send(DownloadProgress { downloaded, total });
            },
            || {},
        )
        .await
        .map_err(|_| UpdateFailure::new("download", "Could not download or verify update"))?;
    // Downloads and signature failures leave transfer admission untouched.
    // Recheck every native guard immediately before the installer boundary.
    let guard = state
        .transfers
        .begin_update_maintenance()
        .await
        .map_err(|_| UpdateFailure::new("prepare", "Finish transfers and remote edits"))?;
    state
        .foldersync
        .ensure_update_idle()
        .await
        .map_err(|_| UpdateFailure::new("prepare", "Disable folder synchronization"))?;
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|_| UpdateFailure::new("prepare", "Cannot access recovery state"))?;
    crate::backup::ensure_no_pending_restore(&dir)
        .map_err(|_| UpdateFailure::new("prepare", "Complete the pending backup restore"))?;
    manual_update::record_attempt(&state.db, current, &target)
        .map_err(|_| UpdateFailure::new("prepare", "Cannot save recovery state"))?;
    // Once installation can start, retain admission until process exit.
    // The plugin verifies the signed bytes again at install; do not bypass it.
    state.transfers.retain_update_maintenance(guard).await;
    tokio::task::spawn_blocking(move || update.install(bytes))
        .await
        .map_err(|_| UpdateFailure::new("install", "Installation status is uncertain"))?
        .map_err(|_| UpdateFailure::new("install", "Installation status is uncertain"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn native_target_rejects_downgrade_and_noncanonical_versions() {
        assert!(newer_target("0.99.91", "0.99.100"));
        for target in ["0.99.91", "0.99.90", "0.99.092", "1.0.0-rc.1", "x"] {
            assert!(!newer_target("0.99.91", target));
        }
    }
}
