//! CLI version monitor for the standalone `ghostftp-cli`.
//!
//! The desktop app may detect version drift, but executable replacement is
//! intentionally disabled until CLI packages have a cryptographically verified
//! update manifest. Version checks remain local/read-only and never install code.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::sync::Mutex;

use crate::AppState;

/// What to do when the installed `ghostftp-cli` is older than the app.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum CliUpdateMode {
    /// Surface a non-blocking prompt and let the user decide (default).
    #[default]
    Ask,
    /// Legacy persisted value. Treated as Ask until verified CLI updates exist.
    Auto,
    /// Never check.
    Off,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
struct Settings {
    mode: CliUpdateMode,
}

pub struct CliUpdater {
    settings_path: PathBuf,
    settings: Mutex<Settings>,
    /// Last computed status, so the Settings UI and the status-bar pill can read
    /// it without re-running a check.
    last: Mutex<Option<CliStatus>>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CliStatus {
    pub mode: CliUpdateMode,
    /// Whether a `ghostftp-cli` binary was located at all.
    pub installed: bool,
    pub cli_path: Option<String>,
    pub cli_version: Option<String>,
    pub app_version: String,
    /// installed && cli_version < app_version.
    pub stale: bool,
    /// A one-line result of the last action (update/install), for the UI.
    pub message: Option<String>,
}

fn safe_mode(mode: CliUpdateMode) -> CliUpdateMode {
    match mode {
        CliUpdateMode::Auto => CliUpdateMode::Ask,
        other => other,
    }
}

impl CliUpdater {
    pub fn load(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).ok();
        let settings_path = dir.join("cli-updater.json");
        let mut settings: Settings = std::fs::read(&settings_path)
            .ok()
            .and_then(|b| serde_json::from_slice(&b).ok())
            .unwrap_or_default();
        settings.mode = safe_mode(settings.mode);
        Ok(Self {
            settings_path,
            settings: Mutex::new(settings),
            last: Mutex::new(None),
        })
    }

    async fn persist(&self) -> Result<()> {
        let s = self.settings.lock().await.clone();
        std::fs::write(&self.settings_path, serde_json::to_vec_pretty(&s)?)
            .with_context(|| format!("write {}", self.settings_path.display()))?;
        Ok(())
    }

    /// On launch, only detect local CLI version drift. Native-code replacement
    /// remains disabled until the CLI distribution has verifiable signatures.
    pub async fn auto_start_if_enabled(&self, app: AppHandle) {
        if safe_mode(self.settings.lock().await.mode) == CliUpdateMode::Off {
            return;
        }
        let status = self.check().await;
        tracing::info!(
            installed = status.installed,
            cli_version = ?status.cli_version,
            app_version = %status.app_version,
            stale = status.stale,
            mode = ?status.mode,
            "cli version check"
        );
        let _ = app.emit("cli-updater://status", &status);
    }

    /// Locate + version-check the installed CLI; store and return the status.
    /// Purely local (no network) so it's cheap to run on every launch.
    pub async fn check(&self) -> CliStatus {
        let mode = safe_mode(self.settings.lock().await.mode);
        let app_version = env!("CARGO_PKG_VERSION").to_string();
        let cli_path = locate_cli();
        let cli_version = cli_path.as_ref().and_then(|p| read_cli_version(p));
        let stale = match (&cli_version, parse_semver(&app_version)) {
            (Some(cv), Some(app)) => parse_semver(cv).map(|c| c < app).unwrap_or(false),
            _ => false,
        };
        let status = CliStatus {
            mode,
            installed: cli_path.is_some(),
            cli_path: cli_path.map(|p| p.to_string_lossy().to_string()),
            cli_version,
            app_version,
            stale,
            message: None,
        };
        *self.last.lock().await = Some(status.clone());
        status
    }

    /// Executable replacement stays fail-closed until CLI release artifacts
    /// have a cryptographically verifiable manifest/signature.
    pub async fn update(&self, _app: &AppHandle) -> Result<CliStatus, String> {
        Err("Automatic ghostftp-cli installation and updates are disabled until signed package verification is available.".to_string())
    }

    async fn set_mode(&self, mode: CliUpdateMode) -> Result<()> {
        self.settings.lock().await.mode = safe_mode(mode);
        self.persist().await
    }
}

/// Run `ghostftp-cli --version` and pull the `X.Y.Z` out of `ghostftp-cli X.Y.Z`.
fn read_cli_version(path: &std::path::Path) -> Option<String> {
    let out = std::process::Command::new(path)
        .arg("--version")
        .output()
        .ok()?;
    if !out.status.success() {
        return None;
    }
    let text = String::from_utf8_lossy(&out.stdout);
    text.split_whitespace().last().map(|s| s.trim().to_string())
}

/// Find an installed `ghostftp-cli`: a sidecar next to the app executable first
/// (the bundled copy, if any), then the first hit on PATH (`where`/`which`).
fn locate_cli() -> Option<PathBuf> {
    let exe_name = if cfg!(windows) {
        "ghostftp-cli.exe"
    } else {
        "ghostftp-cli"
    };
    // 1) Sidecar next to the running app.
    if let Ok(app_exe) = std::env::current_exe() {
        if let Some(dir) = app_exe.parent() {
            let sidecar = dir.join(exe_name);
            if sidecar.is_file() {
                return Some(sidecar);
            }
        }
    }
    // 2) On PATH.
    let finder = if cfg!(windows) { "where" } else { "which" };
    let out = std::process::Command::new(finder)
        .arg("ghostftp-cli")
        .output()
        .ok()?;
    if !out.status.success() {
        return None;
    }
    String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|l| l.trim())
        .find(|l| !l.is_empty())
        .map(PathBuf::from)
}

/// Parse `MAJOR.MINOR.PATCH` (ignoring pre-release/build metadata) into a
/// comparable tuple; None if it isn't semver-shaped.
fn parse_semver(s: &str) -> Option<(u64, u64, u64)> {
    let core = s.trim().split(['-', '+']).next().unwrap_or(s);
    let mut it = core.split('.');
    let major = it.next()?.parse().ok()?;
    let minor = it.next().unwrap_or("0").parse().ok()?;
    let patch = it.next().unwrap_or("0").parse().ok()?;
    Some((major, minor, patch))
}

fn err<E: std::fmt::Display>(e: E) -> String {
    e.to_string()
}

// ---------- Tauri commands (Settings → ghostftp-cli) ----------

#[tauri::command]
pub async fn cli_updater_status(state: State<'_, AppState>) -> Result<CliStatus, String> {
    // Prefer a fresh check so the UI reflects reality when it opens.
    Ok(state.cli_updater.check().await)
}

#[tauri::command]
pub async fn cli_updater_check(
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<CliStatus, String> {
    let status = state.cli_updater.check().await;
    let _ = app.emit("cli-updater://status", &status);
    Ok(status)
}

#[tauri::command]
pub async fn cli_updater_update(
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<CliStatus, String> {
    let status = state.cli_updater.update(&app).await?;
    let _ = app.emit("cli-updater://status", &status);
    Ok(status)
}

#[tauri::command]
pub async fn cli_updater_set_mode(
    mode: CliUpdateMode,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<CliStatus, String> {
    state.cli_updater.set_mode(mode).await.map_err(err)?;
    let status = state.cli_updater.check().await;
    let _ = app.emit("cli-updater://status", &status);
    Ok(status)
}

#[cfg(test)]
mod tests {
    use super::{locate_cli, parse_semver, read_cli_version, safe_mode, CliUpdateMode};

    #[test]
    fn automatic_native_code_update_mode_is_fail_closed() {
        assert_eq!(safe_mode(CliUpdateMode::Auto), CliUpdateMode::Ask);
        assert_eq!(safe_mode(CliUpdateMode::Ask), CliUpdateMode::Ask);
        assert_eq!(safe_mode(CliUpdateMode::Off), CliUpdateMode::Off);
    }

    #[test]
    fn stale_compare() {
        assert!(parse_semver("1.3.19") < parse_semver("1.3.21"));
        assert!(!(parse_semver("1.3.21") < parse_semver("1.3.21")));
        assert_eq!(parse_semver("1.4.0-rc1"), Some((1, 4, 0)));
    }

    /// Exercises the REAL locate + version path against whatever `ghostftp-cli` is on
    /// this machine's PATH. Ignored by default (CI has no CLI installed); run with
    /// `cargo test -p ghostftp locate_installed_cli -- --ignored --nocapture` on a box
    /// where ghostftp-cli is installed to observe the drift the app surfaces.
    #[test]
    #[ignore]
    fn locate_installed_cli() {
        let path = locate_cli().expect("ghostftp-cli not found on PATH");
        eprintln!("located ghostftp-cli at {}", path.display());
        let ver = read_cli_version(&path).expect("could not read --version");
        eprintln!("ghostftp-cli version = {ver}");
        assert!(parse_semver(&ver).is_some(), "version wasn't semver-shaped");
    }
}
