//! CLI updater (Plan 10 Phase 0c/0d) — keeps the standalone `ghostftp-cli` in step
//! with this app.
//!
//! `ghostftp-cli` and the desktop app ship as **separate downloads**, so after an
//! app update the on-PATH CLI can silently lag — advertising flags/commands it
//! doesn't have (the drift that produced the session's `unexpected argument
//! '--timeout-ms'`). This subsystem locates the installed CLI, compares its
//! version to the app's, and — per the user's `cliUpdate` preference — either
//! prompts (`ask`), updates silently (`auto`), or stays quiet (`off`). Shaped
//! like `agent_host.rs`: load / persist / auto_start_if_enabled / status, JSON
//! config under the app data dir, `"cli-updater://status"` events via `Emitter`.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::sync::Mutex;

use crate::AppState;

/// Ghost FTP release service base URL (matches `ghostftp-cli self-update`).
const RELEASE_BASE: &str = "https://ghostftp.com";

/// What to do when the installed `ghostftp-cli` is older than the app.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum CliUpdateMode {
    /// Surface a non-blocking prompt and let the user decide (default).
    #[default]
    Ask,
    /// Download + swap silently on launch, no prompt.
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

impl CliUpdater {
    pub fn load(app: &AppHandle) -> Result<Self> {
        let dir = app.path().app_data_dir().context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).ok();
        let settings_path = dir.join("cli-updater.json");
        let settings: Settings = std::fs::read(&settings_path)
            .ok()
            .and_then(|b| serde_json::from_slice(&b).ok())
            .unwrap_or_default();
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

    /// On launch: unless disabled, check the CLI and — when `auto` and stale —
    /// update it silently. Either way the resulting status is emitted so the
    /// frontend can raise the prompt (`ask`) or just reflect state.
    pub async fn auto_start_if_enabled(&self, app: AppHandle) {
        if self.settings.lock().await.mode == CliUpdateMode::Off {
            return;
        }
        let status = self.check().await;
        tracing::info!(
            installed = status.installed,
            cli_version = ?status.cli_version,
            app_version = %status.app_version,
            stale = status.stale,
            mode = ?status.mode,
            "cli-updater startup check"
        );
        let _ = app.emit("cli-updater://status", &status);
        if status.stale && status.mode == CliUpdateMode::Auto {
            let after = self.update(&app).await;
            let _ = app.emit("cli-updater://status", &after);
        }
    }

    /// Locate + version-check the installed CLI; store and return the status.
    /// Purely local (no network) so it's cheap to run on every launch.
    pub async fn check(&self) -> CliStatus {
        let mode = self.settings.lock().await.mode;
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

    /// Bring the CLI up to date: delegate to `ghostftp-cli self-update` when the
    /// binary exists (reusing its tested download+swap), otherwise download the
    /// matching asset into an app-owned `bin/` dir. Re-checks and returns status.
    pub async fn update(&self, app: &AppHandle) -> Result<CliStatus, String> {
        let located = locate_cli();
        let message = match &located {
            Some(path) => run_self_update(path).await,
            None => install_missing(app).await,
        };
        let mut status = self.check().await;
        status.message = Some(match &message {
            Ok(m) => m.clone(),
            Err(e) => format!("update failed: {e}"),
        });
        *self.last.lock().await = Some(status.clone());
        match message {
            Ok(_) => Ok(status),
            Err(e) => Err(e),
        }
    }

    async fn set_mode(&self, mode: CliUpdateMode) -> Result<()> {
        self.settings.lock().await.mode = mode;
        self.persist().await
    }
}

/// Run `ghostftp-cli --version` and pull the `X.Y.Z` out of `ghostftp-cli X.Y.Z`.
fn read_cli_version(path: &std::path::Path) -> Option<String> {
    let out = std::process::Command::new(path).arg("--version").output().ok()?;
    if !out.status.success() {
        return None;
    }
    let text = String::from_utf8_lossy(&out.stdout);
    text.split_whitespace().last().map(|s| s.trim().to_string())
}

/// Find an installed `ghostftp-cli`: a sidecar next to the app executable first
/// (the bundled copy, if any), then the first hit on PATH (`where`/`which`).
fn locate_cli() -> Option<PathBuf> {
    let exe_name = if cfg!(windows) { "ghostftp-cli.exe" } else { "ghostftp-cli" };
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
    let out = std::process::Command::new(finder).arg("ghostftp-cli").output().ok()?;
    if !out.status.success() {
        return None;
    }
    String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|l| l.trim())
        .find(|l| !l.is_empty())
        .map(PathBuf::from)
}

/// `<cli> self-update` — the CLI does the ghostftp.com download + in-place swap.
async fn run_self_update(path: &std::path::Path) -> Result<String, String> {
    let out = tokio::process::Command::new(path)
        .arg("self-update")
        .output()
        .await
        .map_err(|e| format!("run {} self-update: {e}", path.display()))?;
    let stdout = String::from_utf8_lossy(&out.stdout);
    let stderr = String::from_utf8_lossy(&out.stderr);
    if out.status.success() {
        // The CLI prints "Updated ghostftp-cli: vX → vY" (or "already up to date").
        Ok(stdout
            .lines()
            .find(|l| l.starts_with("Updated") || l.contains("up to date"))
            .unwrap_or("ghostftp-cli updated")
            .to_string())
    } else {
        Err(format!("ghostftp-cli self-update failed: {}", stderr.trim()))
    }
}

/// The release asset name matching this OS/arch (mirrors the release workflow).
fn target_asset_name() -> Result<&'static str, String> {
    Ok(match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => "ghostftp-cli-windows-x86_64.exe",
        ("linux", "x86_64") => "ghostftp-cli-linux-x86_64",
        (os, arch) => return Err(format!("no prebuilt ghostftp-cli for {os}/{arch}")),
    })
}

/// Download the matching release asset into an app-owned `bin/` directory when no
/// CLI is on PATH. Not added to PATH (that's OS-specific and intrusive) — the UI
/// reports the path so the user can wire it up.
async fn install_missing(app: &AppHandle) -> Result<String, String> {
    let asset = target_asset_name()?;
    let url = format!("{RELEASE_BASE}/downloads/{asset}");
    let bytes = reqwest::get(&url)
        .await
        .map_err(|e| format!("download {url}: {e}"))?
        .error_for_status()
        .map_err(|e| format!("download {url}: {e}"))?
        .bytes()
        .await
        .map_err(|e| format!("read download: {e}"))?;
    if bytes.len() < 200_000 {
        return Err(format!(
            "downloaded asset is only {} bytes — that doesn't look like ghostftp-cli",
            bytes.len()
        ));
    }
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("resolve app_data_dir: {e}"))?
        .join("bin");
    std::fs::create_dir_all(&dir).map_err(|e| format!("create {}: {e}", dir.display()))?;
    let name = if cfg!(windows) { "ghostftp-cli.exe" } else { "ghostftp-cli" };
    let dest = dir.join(name);
    std::fs::write(&dest, &bytes).map_err(|e| format!("write {}: {e}", dest.display()))?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = std::fs::set_permissions(&dest, std::fs::Permissions::from_mode(0o755));
    }
    Ok(format!("installed ghostftp-cli to {} (add it to your PATH)", dest.display()))
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
    // Flipping to Auto while stale should update right away.
    let mut status = state.cli_updater.check().await;
    if status.stale && mode == CliUpdateMode::Auto {
        status = state.cli_updater.update(&app).await?;
    }
    let _ = app.emit("cli-updater://status", &status);
    Ok(status)
}

#[cfg(test)]
mod tests {
    use super::{locate_cli, parse_semver, read_cli_version};

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
