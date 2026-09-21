use std::sync::Arc;
use tauri::Manager;

// The CLI crate (ghostftp-cli/src/main.rs) imports from these via the `ghostftp_lib`
// crate, so they need to be `pub` rather than `mod`. None of them expose
// secrets directly — credentials live in profiles::ConnectionProfile, which
// the CLI deliberately redacts in `profiles show`.
mod agent_host;
pub mod backup;
pub mod bridge;
mod cli_updater;
pub mod commands;
pub mod credentials;
pub mod db;
pub mod dedupe;
mod deeplink;
pub mod diff;
mod diskscan;
mod editor;
pub mod error;
mod foldersync;
pub mod grant;
pub mod importers;
pub mod keys;
mod known_hosts;
pub mod oauth;
mod path_integration;
mod preview;
pub mod profiles;
pub mod remotefs;
pub mod scan;
pub mod scp;
pub mod search;
pub mod session;
pub mod sync;
mod terminal;
mod transfer;
mod virtualfs;

pub struct AppState {
    pub sessions: Arc<session::SessionManager>,
    pub ptys: Arc<terminal::PtyManager>,
    pub profiles: Arc<profiles::ProfileStore>,
    pub transfers: Arc<transfer::TransferManager>,
    pub editors: Arc<editor::EditManager>,
    pub bridge: Arc<bridge::BridgeState>,
    pub agent_host: Arc<agent_host::AgentHost>,
    /// CLI version-drift watcher (Plan 10 Phase 0c/0d) — keeps `ghostftp-cli` in step
    /// with the app per the user's `cliUpdate` preference.
    pub cli_updater: Arc<cli_updater::CliUpdater>,
    pub foldersync: Arc<foldersync::FolderSync>,
    /// On-demand virtual folders (Plan 9) — OneDrive-style placeholders. Owns
    /// the OS sync-root registrations; inert on non-Windows / non-`virtualfs`
    /// builds.
    pub virtualfs: Arc<virtualfs::VirtualFs>,
    /// Running disk-usage scans (Plan 4). Ephemeral — not persisted.
    pub diskscan: Arc<diskscan::ScanManager>,
    /// Running directory diffs (Plan 6). Ephemeral — not persisted.
    pub diff: Arc<diff::DiffManager>,
    /// Running duplicate scans. Ephemeral — not persisted.
    pub dedupe: Arc<dedupe::DedupeManager>,
    /// Running fleet searches (Plan 7). Ephemeral — not persisted.
    pub search: Arc<search::SearchManager>,
    /// Shared `ghostftp.db` — the per-connection index (sync_state today; scan/search
    /// caches later). See `db.rs`.
    pub db: Arc<db::Db>,
    /// Remote image thumbnail previews (Plan 13 Phase 1) — bounded reads, a
    /// per-connection concurrency cap, and an LRU disk cache. See `preview.rs`.
    pub preview: Arc<preview::PreviewManager>,
}

/// Build the pre-paint JS injected into the main window (Plan 12 Phase 2). It
/// reads every setting from `ghostftp.db`, exposes them on `window.__GHOSTFTP_SETTINGS__`
/// for the frontend store to seed from, and sets `data-theme` on `<html>` before
/// the first paint so there's no theme flash on reload.
fn build_settings_init_script(db: &db::Db) -> String {
    let mut map = db.settings_get_all().unwrap_or_default();
    // A fresh install has no rows yet — default the theme so the window still
    // paints dark (matching the frontend DEFAULTS), not the bare CSS default.
    map.entry("appTheme".to_string())
        .or_insert_with(|| "\"dark\"".to_string());

    // Build a JS object literal `{ "key": <rawJson>, ... }`. Each stored value is
    // already a JSON string; skip any that don't parse so one bad row can't break
    // the whole injection (which would leave the window unthemed).
    let mut pairs: Vec<String> = Vec::new();
    for (k, v) in &map {
        if serde_json::from_str::<serde_json::Value>(v).is_ok() {
            if let Ok(key) = serde_json::to_string(k) {
                pairs.push(format!("{key}:{v}"));
            }
        }
    }
    let obj = format!("{{{}}}", pairs.join(","));
    let qa_view = std::env::var("GHOSTFTP_QA_VIEW")
        .ok()
        .filter(|view| {
            matches!(
                view.as_str(),
                "main"
                    | "siteManager"
                    | "settings"
                    | "transferCenter"
                    | "about"
                    | "newConnection"
                    | "properties"
            )
        })
        .unwrap_or_default();
    let qa_view_json = serde_json::to_string(&qa_view).unwrap_or_else(|_| "\"\"".to_string());

    format!(
        "(function(){{try{{\
           var s={obj};\
           window.__GHOSTFTP_SETTINGS__=s;\
           window.__GHOSTFTP_QA_VIEW__={qa_view_json};\
           var el=document.documentElement;\
           if(el&&typeof s.appTheme==='string'){{el.setAttribute('data-theme',s.appTheme);}}\
         }}catch(e){{}}}})();"
    )
}

#[tauri::command]
fn open_external_url(url: String) -> Result<(), String> {
    let parsed = url::Url::parse(&url).map_err(|_| "Invalid external URL".to_string())?;
    let host = parsed.host_str().unwrap_or_default().to_ascii_lowercase();
    if parsed.scheme() != "https" || !(host == "ghostftp.com" || host.ends_with(".ghostftp.com")) {
        return Err("Ghost FTP only opens approved ghostftp.com HTTPS links".to_string());
    }

    #[cfg(windows)]
    let result = std::process::Command::new("rundll32.exe")
        .arg("url.dll,FileProtocolHandler")
        .arg(parsed.as_str())
        .spawn();

    #[cfg(target_os = "linux")]
    let result = std::process::Command::new("xdg-open")
        .arg(parsed.as_str())
        .spawn();

    #[cfg(target_os = "macos")]
    let result = std::process::Command::new("open")
        .arg(parsed.as_str())
        .spawn();

    #[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
    return Err("Opening external links is unsupported on this platform".to_string());

    #[cfg(any(windows, target_os = "linux", target_os = "macos"))]
    result
        .map(|_| ())
        .map_err(|error| format!("Could not open the default browser: {error}"))
}

fn open_db_resilient(path: &std::path::Path) -> anyhow::Result<db::Db> {
    match db::Db::open(path) {
        Ok(db) => Ok(db),
        Err(first) => {
            // Preserve a broken SQLite store and its sidecars instead of panicking
            // at startup. Ghost FTP can recreate indexes/settings; the quarantined
            // files remain available for support or manual recovery.
            let stamp = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_secs())
                .unwrap_or(0);
            for suffix in ["", "-wal", "-shm"] {
                let candidate = std::path::PathBuf::from(format!("{}{}", path.display(), suffix));
                if candidate.exists() {
                    let quarantine = std::path::PathBuf::from(format!(
                        "{}.corrupt.{}{}",
                        path.display(),
                        stamp,
                        suffix
                    ));
                    let _ = std::fs::rename(&candidate, quarantine);
                }
            }
            db::Db::open(path).map_err(|second| {
                anyhow::anyhow!(
                    "ghostftp.db recovery failed after initial error: {first:#}; retry: {second:#}"
                )
            })
        }
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "ghostftp_lib=info,warn".into()),
        )
        .init();

    tauri::Builder::default()
        // single-instance MUST be the first plugin: on Windows/Linux a
        // `ghostftp://` link launches a second process, and this forwards its argv
        // (which carries the URL) to the running instance and focuses it,
        // instead of opening a duplicate window. The `deep-link` feature makes
        // the forwarded URL fire the same on_open_url handler below.
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            use tauri::Manager;
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.set_focus();
            }
        }))
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .setup(|app| {
            let handle = app.handle().clone();

            // Apply a pending encrypted-backup restore (Plan 12 Phase 4) before
            // anything opens profiles.json / ghostftp.db, so a restore staged by a
            // previous run swaps in atomically on this launch.
            if let Ok(data_dir) = handle.path().app_data_dir() {
                std::fs::create_dir_all(&data_dir).ok();
                backup::apply_pending_restore(&data_dir);
            }

            let profile_store = Arc::new(profiles::ProfileStore::load_or_create(&handle)?);
            let db = {
                let dir = handle.path().app_data_dir()?;
                std::fs::create_dir_all(&dir)?;
                Arc::new(open_db_resilient(&dir.join("ghostftp.db"))?)
            };
            // Remote thumbnail cache lives under the app data dir alongside ghostftp.db
            // (the codebase keeps everything under app_data_dir; there's no
            // app_cache_dir convention here).
            let preview = {
                let dir = handle.path().app_data_dir()?;
                Arc::new(preview::PreviewManager::new(dir.join("thumbnails")))
            };

            // Create the main window ourselves (it's no longer in
            // tauri.conf.json) so we can inject a pre-paint script that reads the
            // theme from ghostftp.db and sets `data-theme` on <html> before the first
            // paint — killing the dark→light flash structurally, not by timing
            // luck (Plan 12 Phase 2). The script also stashes the full settings
            // snapshot on `window.__GHOSTFTP_SETTINGS__` so the frontend store seeds
            // itself synchronously instead of an async round-trip.
            {
                let init_script = build_settings_init_script(&db);
                let window_builder =
                    tauri::WebviewWindowBuilder::new(app, "main", tauri::WebviewUrl::default())
                        .title("Ghost FTP")
                        .inner_size(1290.0, 852.0)
                        .min_inner_size(480.0, 600.0)
                        .decorations(false)
                        .resizable(true)
                        .shadow(true)
                        .initialization_script(&init_script);

                #[cfg(windows)]
                let mut window_builder = window_builder;

                // Native Windows screenshot QA must capture the real WebView2
                // compositor, not a screenshot fixture. Apply the QA-only
                // browser arguments programmatically because elevated hosted
                // runners may ignore WebView2 environment/policy overrides.
                // These variables are used only with an allow-listed QA view.
                #[cfg(windows)]
                if std::env::var("GHOSTFTP_QA_VIEW").ok().is_some_and(|view| {
                    matches!(
                        view.as_str(),
                        "main"
                            | "siteManager"
                            | "settings"
                            | "transferCenter"
                            | "about"
                            | "newConnection"
                            | "properties"
                    )
                }) {
                    if let Ok(args) = std::env::var("GHOSTFTP_QA_BROWSER_ARGS") {
                        if !args.trim().is_empty() {
                            window_builder = window_builder.additional_browser_args(&args);
                        }
                    }
                    if let Ok(data_dir) = std::env::var("GHOSTFTP_QA_WEBVIEW_DATA") {
                        if !data_dir.trim().is_empty() {
                            window_builder =
                                window_builder.data_directory(std::path::PathBuf::from(data_dir));
                        }
                    }
                }

                window_builder.build()?;
            }

            let state = AppState {
                sessions: Arc::new(session::SessionManager::new()),
                ptys: Arc::new(terminal::PtyManager::new()),
                profiles: profile_store,
                transfers: Arc::new(transfer::TransferManager::new()),
                editors: Arc::new(editor::EditManager::new()),
                bridge: Arc::new(bridge::BridgeState::load_or_create(&handle).unwrap_or_default()),
                agent_host: Arc::new(agent_host::AgentHost::load(&handle)?),
                cli_updater: Arc::new(cli_updater::CliUpdater::load(&handle)?),
                foldersync: Arc::new(foldersync::FolderSync::load(&handle)?),
                virtualfs: Arc::new(virtualfs::VirtualFs::load(&handle)?),
                diskscan: Arc::new(diskscan::ScanManager::new()),
                diff: Arc::new(diff::DiffManager::new()),
                dedupe: Arc::new(dedupe::DedupeManager::new()),
                search: Arc::new(search::SearchManager::new()),
                db,
                preview,
            };
            app.manage(state);

            // Apply persisted transfer-queue settings (Plan 17, Plan 23). The
            // frontend writes `transferConcurrency`/`transferThrottleKbps`/
            // `deltaSync` via the settings table; the manager starts from
            // those values.
            {
                let st = app.state::<AppState>();
                if let Ok(Some(raw)) = st.db.settings_get("transferConcurrency") {
                    if let Ok(n) = serde_json::from_str::<usize>(&raw) {
                        st.transfers.set_concurrency(n);
                    }
                }
                if let Ok(Some(raw)) = st.db.settings_get("transferThrottleKbps") {
                    if let Ok(kbps) = serde_json::from_str::<u64>(&raw) {
                        st.transfers.set_throttle_kbps(kbps);
                    }
                }
                // Plan 23: the `deltaSync` toggle (default on). Absent row →
                // the manager's own default already has it enabled.
                if let Ok(Some(raw)) = st.db.settings_get("deltaSync") {
                    if let Ok(on) = serde_json::from_str::<bool>(&raw) {
                        st.transfers.set_delta_enabled(on);
                    }
                }
            }

            // Bring the Agent Bridge back up if the user left its master switch
            // on, so the `ghostftp-cli agent …` path keeps working across restarts.
            // Spawned off the async runtime so the sync setup() returns at once.
            let bridge = app.state::<AppState>().bridge.clone();
            let bridge_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                bridge.auto_start_if_enabled(bridge_handle).await;
            });

            // Likewise the Remote-control host (this machine as a Ghost FTP Agent).
            let host = app.state::<AppState>().agent_host.clone();
            let host_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                host.auto_start_if_enabled(host_handle).await;
            });

            // Check whether the installed ghostftp-cli lags the app (Plan 10 0c/0d);
            // updates silently when the user chose `auto`, else emits status so
            // the status-bar pill can prompt.
            let cli_updater = app.state::<AppState>().cli_updater.clone();
            let cli_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                cli_updater.auto_start_if_enabled(cli_handle).await;
            });

            // Restart any folder-sync pairs the user left enabled.
            let foldersync = app.state::<AppState>().foldersync.clone();
            let foldersync_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                foldersync.auto_start_if_enabled(foldersync_handle).await;
            });

            // ghostftp:// deep links. `on_open_url` covers the app-already-running
            // case (Linux always; Windows/Linux via single-instance forwarding).
            {
                use tauri_plugin_deep_link::DeepLinkExt;
                let dl_handle = app.handle().clone();
                app.deep_link().on_open_url(move |event| {
                    deeplink::handle_urls(&dl_handle, event.urls().as_slice());
                });
                // Cold start: the OS may have launched us WITH the URL already.
                if let Ok(Some(urls)) = app.deep_link().get_current() {
                    deeplink::handle_urls(&app.handle().clone(), urls.as_slice());
                }
                // On dev/Linux the scheme must be registered at runtime; on a
                // packaged build the installer does it. Best-effort.
                #[cfg(any(windows, target_os = "linux"))]
                {
                    let _ = app.deep_link().register_all();
                }
            }

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            open_external_url,
            commands::list_profiles,
            commands::save_profile,
            commands::reorder_profiles,
            commands::duplicate_profile,
            commands::delete_profile,
            commands::ssh_key_defaults,
            commands::generate_ssh_key,
            commands::ssh_public_key_for,
            commands::test_profile_connection,
            commands::test_ephemeral_connection,
            commands::connect,
            commands::connect_ephemeral,
            commands::disconnect,
            commands::discover_agents,
            commands::agent_public_key,
            commands::pair_agent,
            commands::dropbox_authorize,
            commands::onedrive_authorize,
            commands::gdrive_authorize,
            commands::box_authorize,
            commands::dynamics_authorize,
            agent_host::agent_host_status,
            agent_host::agent_host_set_enabled,
            agent_host::agent_host_open_pairing,
            agent_host::agent_host_close_pairing,
            agent_host::agent_host_set_policy,
            agent_host::agent_host_revoke_peer,
            cli_updater::cli_updater_status,
            cli_updater::cli_updater_check,
            cli_updater::cli_updater_update,
            cli_updater::cli_updater_set_mode,
            path_integration::path_status,
            path_integration::path_add,
            path_integration::path_remove,
            foldersync::foldersync_list,
            foldersync::foldersync_upsert,
            foldersync::foldersync_remove,
            foldersync::foldersync_set_enabled,
            foldersync::foldersync_sync_now,
            virtualfs::virtualfs_supported,
            virtualfs::virtualfs_status,
            virtualfs::virtualfs_free_up_space,
            diskscan::diskscan_start,
            diskscan::diskscan_status,
            diskscan::diskscan_tree,
            diskscan::diskscan_cancel,
            diskscan::diskscan_forget,
            diff::diff_start,
            diff::diff_status,
            diff::diff_result,
            diff::diff_cancel,
            diff::diff_forget,
            dedupe::dedupe_start,
            dedupe::dedupe_status,
            dedupe::dedupe_result,
            dedupe::dedupe_cancel,
            dedupe::dedupe_forget,
            dedupe::dedupe_delete,
            search::search_start,
            search::search_status,
            search::search_result,
            search::search_cancel,
            search::search_forget,
            commands::list_agent_jobs,
            commands::kill_agent_job,
            commands::respond_to_host_prompt,
            commands::respond_to_auth_prompt,
            commands::importer_default_paths,
            commands::import_openssh,
            commands::import_filezilla,
            commands::import_putty,
            commands::save_imported_profiles,
            commands::sync_plan,
            commands::sync_execute,
            commands::start_edit,
            commands::stop_edit,
            commands::list_directory,
            commands::capabilities,
            commands::read_file_preview,
            preview::preview_thumbnail,
            commands::open_terminal,
            commands::terminal_write,
            commands::terminal_resize,
            commands::close_terminal,
            commands::snippet_list,
            commands::snippet_save,
            commands::snippet_delete,
            commands::snippet_run,
            commands::start_download,
            commands::start_upload,
            commands::start_directory_download,
            commands::start_directory_upload,
            commands::cancel_transfer,
            commands::list_transfers,
            commands::transfer_move,
            commands::transfer_pause,
            commands::transfer_resume,
            commands::transfer_retry,
            commands::transfer_pause_all,
            commands::transfer_resume_all,
            commands::transfer_set_concurrency,
            commands::transfer_set_max_retries,
            commands::transfer_set_throttle,
            commands::transfer_set_delta_sync,
            commands::transfer_queue_state,
            commands::rename_path,
            commands::delete_path,
            commands::create_directory,
            commands::chmod_path,
            commands::chmod_path_recursive,
            commands::checksum_path,
            commands::duplicate_path,
            commands::start_archive_download,
            commands::bridge_start,
            commands::bridge_stop,
            commands::bridge_set_enabled,
            commands::bridge_status,
            commands::bridge_set_session_access,
            commands::bridge_set_policy,
            commands::bridge_set_active_session,
            commands::bridge_register_mcp,
            commands::set_api_key,
            commands::api_key_status,
            commands::settings_get_all,
            commands::settings_set,
            commands::settings_delete,
            commands::settings_set_all,
            grant::fetch_grant_manifest,
            grant::accept_grant,
            commands::backup_export,
            commands::backup_inspect,
            commands::backup_import,
            commands::respond_to_bridge_approval,
            commands::bridge_activity,
            commands::bridge_clear_activity,
            commands::bridge_list_commands,
            commands::bridge_save_command,
            commands::bridge_delete_command,
            commands::bridge_list_skills,
            commands::bridge_save_skill,
            commands::bridge_delete_skill,
            commands::bridge_approve_skill,
            commands::bridge_run_skill,
            commands::export_agent_log,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[cfg(test)]
mod init_script_tests {
    use super::*;

    #[test]
    fn injects_theme_and_snapshot() {
        let db = db::Db::open_in_memory().unwrap();
        db.settings_set("appTheme", "\"nord\"").unwrap();
        db.settings_set("terminalFontSize", "15").unwrap();
        let script = build_settings_init_script(&db);
        assert!(script.contains("window.__GHOSTFTP_SETTINGS__"));
        assert!(script.contains("setAttribute('data-theme'"));
        assert!(script.contains("\"appTheme\":\"nord\""));
        assert!(script.contains("\"terminalFontSize\":15"));
    }

    #[test]
    fn defaults_theme_to_dark_on_empty_db() {
        let db = db::Db::open_in_memory().unwrap();
        let script = build_settings_init_script(&db);
        assert!(script.contains("\"appTheme\":\"dark\""));
    }

    #[test]
    fn skips_corrupt_rows_without_breaking() {
        let db = db::Db::open_in_memory().unwrap();
        // A value that isn't valid JSON must be dropped, not emitted raw (which
        // would break the whole object literal and leave the window unthemed).
        db.settings_set("appTheme", "\"dracula\"").unwrap();
        db.settings_set("broken", "not json").unwrap();
        let script = build_settings_init_script(&db);
        assert!(script.contains("\"appTheme\":\"dracula\""));
        assert!(!script.contains("not json"));
    }
}
