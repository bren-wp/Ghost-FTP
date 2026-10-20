//! Continuous one-way folder sync — the "attach a local folder to a remote and
//! keep it mirrored" feature (see `docs/plans/2_continuous-folder-sync.md`).
//!
//! Each **pair** binds a local folder to a remote root on a saved connection.
//! While enabled, a background task re-reconciles the pair whenever the local
//! folder changes (a `notify` watcher) or a poll interval elapses (the backstop
//! for remote-side drift, and the only trigger for `RemoteToLocal`). Because
//! object stores, SFTP, FTP, and the Ghost FTP Agent have no change feed, remote
//! detection is polling by design.
//!
//! The reconciler is deliberately thin: it reuses the existing stateless
//! [`crate::sync::plan`] (files-only diff by size + mtime) and
//! [`crate::commands::execute_sync_plan`] (chunked transfers via
//! `TransferManager`). One-way sync needs no persistent per-file index —
//! the source side is authoritative, so `plan` + `Mirror` deletes fully
//! describe the work. (A persistent index is only needed for bidirectional
//! conflict detection and same-size-edit detection; both are future work.)
//!
//! Structure mirrors `agent_host.rs`: a `Mutex`-guarded settings blob persisted
//! as JSON under the app data dir, a map of running tasks, `load` /
//! `auto_start_if_enabled` / per-pair `start` / `stop` / `status`, and
//! `foldersync://changed` events so the UI can refresh.

use std::collections::HashMap;
use std::path::Path;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result};
use notify::{RecursiveMode, Watcher};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::sync::{mpsc, Mutex};

use crate::sync::{SyncDirection, SyncPlan, SyncStrategy};
use crate::transfer::TransferStatus;
use crate::AppState;

/// Backstop poll cadence when a pair doesn't specify one.
const DEFAULT_POLL_SECS: u64 = 60;
/// Coalesce a burst of local FS events into one reconcile.
const DEBOUNCE: Duration = Duration::from_millis(700);
/// Give up waiting on a single reconcile's transfers after this long (they keep
/// running in the TransferManager; we just stop blocking the pair's status on
/// them so the next tick can proceed).
const TRANSFER_WAIT_CAP: Duration = Duration::from_secs(30 * 60);
/// Delta-sync temp files older than this are crash litter (Plan 23 Phase 3) —
/// a live transfer's temps are minutes old at most.
const DELTA_TEMP_MAX_AGE: Duration = Duration::from_secs(24 * 60 * 60);

fn now_ms() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

// ---------- persisted config ----------

/// One configured sync pair. Persisted verbatim in `foldersync.json`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncPair {
    pub id: String,
    pub name: String,
    pub local_root: String,
    /// The saved connection this pair syncs against (`ConnectionProfile.id`).
    pub profile_id: String,
    pub remote_root: String,
    pub direction: SyncDirection,
    pub strategy: SyncStrategy,
    /// How the pair materializes files locally. `Mirror` (default) moves whole
    /// files eagerly, today's behavior. `OnDemand` registers OS placeholders
    /// that hydrate on open — Plan 9, Windows-only, driven by the `VirtualFs`
    /// subsystem instead of the reconcile loop below.
    #[serde(default)]
    pub mode: SyncMode,
    pub enabled: bool,
    #[serde(default = "default_poll_secs")]
    pub poll_interval_secs: u64,
    /// gitignore-style patterns; matching files are neither pushed nor (under
    /// Mirror) deleted. See [`is_excluded`].
    #[serde(default)]
    pub exclude: Vec<String>,
    /// Safety cap on how many destination files a single Mirror reconcile may
    /// delete. `0` disables the cap. Doesn't apply to Additive (no deletes).
    #[serde(default = "default_mirror_delete_cap")]
    pub mirror_delete_cap: u32,
}

fn default_poll_secs() -> u64 {
    DEFAULT_POLL_SECS
}

/// Default blast-radius cap for Mirror deletes — high enough for normal churn,
/// low enough to catch "the source folder vanished, delete everything remote."
const DEFAULT_MIRROR_DELETE_CAP: u32 = 100;

fn default_mirror_delete_cap() -> u32 {
    DEFAULT_MIRROR_DELETE_CAP
}

/// How a pair keeps local files in step with the remote.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SyncMode {
    /// Whole files, moved eagerly by the reconcile loop (Plan 2). The default.
    #[default]
    Mirror,
    /// OneDrive-style placeholders that hydrate on open, driven by the
    /// `VirtualFs` provider (Plan 9). Windows-only; degrades to inert elsewhere.
    OnDemand,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
struct Settings {
    pairs: Vec<SyncPair>,
}

// ---------- live status ----------

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum PairState {
    Idle,
    Scanning,
    Syncing,
    Error,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct RuntimeStatus {
    state: PairState,
    /// Files still transferring in the current reconcile.
    in_flight: u32,
    /// ms epoch of the last successful reconcile, if any.
    last_synced: Option<i64>,
    last_error: Option<String>,
}

impl Default for RuntimeStatus {
    fn default() -> Self {
        Self {
            state: PairState::Idle,
            in_flight: 0,
            last_synced: None,
            last_error: None,
        }
    }
}

/// A pair merged with its live runtime status — the shape the UI consumes.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PairView {
    #[serde(flatten)]
    pair: SyncPair,
    running: bool,
    state: PairState,
    in_flight: u32,
    last_synced: Option<i64>,
    last_error: Option<String>,
}

// ---------- running task handle ----------

struct Running {
    task: tauri::async_runtime::JoinHandle<()>,
    /// Manual/`notify` reconcile trigger. Dropping every clone ends the loop.
    trigger: mpsc::Sender<()>,
    /// Kept alive so the filesystem watch stays registered.
    _watcher: Option<notify::RecommendedWatcher>,
    status: Arc<Mutex<RuntimeStatus>>,
    /// Session created for this background pair. Shared with stop_pair so an
    /// aborted reconcile cannot strand a hidden live connection.
    session_id: Arc<Mutex<Option<String>>>,
}

pub struct FolderSync {
    db: Arc<crate::db::Db>,
    settings: Mutex<Settings>,
    running: Mutex<HashMap<String, Running>>,
}

/// Read the single SQLite source of truth, migrating the old JSON snapshot
/// once. Keep the original file intact for disaster recovery and refuse to
/// silently reset corrupt configuration to an empty (dangerous) sync list.
fn load_settings(db: &crate::db::Db, dir: &Path) -> Result<Settings> {
    if let Some(payload) = db.folder_sync_config_get()? {
        return serde_json::from_str(&payload)
            .context("SQLite folder sync configuration is corrupt; existing pairs preserved");
    }
    // If a corrupt database was quarantined, importing the stale legacy
    // snapshot could reactivate a previously removed destructive Mirror pair.
    let marker = dir.join("foldersync.sqlite-migrated");
    match std::fs::symlink_metadata(&marker) {
        Ok(_) => anyhow::bail!(
            "SQLite sync configuration is missing after a prior migration; restore ghostftp.db from backup"
        ),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        Err(error) => return Err(error).context("stat sync migration marker"),
    }
    let legacy = dir.join("foldersync.json");
    let settings: Settings = match std::fs::read(&legacy) {
        Ok(bytes) => serde_json::from_slice(&bytes).with_context(|| {
            format!(
                "legacy folder sync configuration is corrupt: {}",
                legacy.display()
            )
        })?,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Settings::default(),
        Err(error) => return Err(error).with_context(|| format!("read {}", legacy.display())),
    };
    let payload = serde_json::to_string(&settings)?;
    db.folder_sync_config_set(&payload)?;
    // Marker has no secrets or user profile data. Never silently resurrect
    // old JSON pairs following a future complete SQLite database loss.
    use std::io::Write;
    #[cfg(unix)]
    use std::os::unix::fs::OpenOptionsExt;
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    options.mode(0o600);
    let mut file = options
        .open(&marker)
        .context("create sync migration marker")?;
    file.write_all(b"Ghost FTP SQLite sync migrated\n")?;
    file.sync_all().context("persist sync migration marker")?;
    Ok(settings)
}

impl FolderSync {
    pub fn load(app: &AppHandle, db: Arc<crate::db::Db>) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).with_context(|| format!("create {}", dir.display()))?;
        let settings = load_settings(&db, &dir)?;
        Ok(Self {
            db,
            settings: Mutex::new(settings),
            running: Mutex::new(HashMap::new()),
        })
    }

    fn persist_snapshot(&self, settings: &Settings) -> Result<()> {
        // All mutations are serialized by the pair-settings Mutex. The SQLite
        // UPSERT is atomic; on failure the in-memory caller rolls back.
        self.db
            .folder_sync_config_set(&serde_json::to_string(settings)?)
    }

    /// Bring up every pair the user left enabled.
    pub async fn auto_start_if_enabled(&self, app: AppHandle) {
        let pairs = self.settings.lock().await.pairs.clone();
        for p in pairs.into_iter().filter(|p| p.enabled) {
            if let Err(e) = self.start_pair(&app, &p).await {
                tracing::warn!("folder sync '{}' failed to auto-start: {e:#}", p.name);
            }
        }
        // Hand the on-demand pairs to the VirtualFs provider and reconcile
        // orphaned sync roots (registered but no longer configured).
        self.reconcile_virtualfs(&app).await;
    }

    /// The enabled on-demand pairs, as the provider's lighter [`OnDemandPair`].
    async fn enabled_ondemand_pairs(&self) -> Vec<crate::virtualfs::OnDemandPair> {
        self.settings
            .lock()
            .await
            .pairs
            .iter()
            .filter(|p| p.enabled && p.mode == SyncMode::OnDemand)
            .map(|p| crate::virtualfs::OnDemandPair {
                id: p.id.clone(),
                name: p.name.clone(),
                local_root: p.local_root.clone(),
                profile_id: p.profile_id.clone(),
                remote_root: p.remote_root.clone(),
            })
            .collect()
    }

    /// Push the current on-demand pair set into the VirtualFs subsystem, which
    /// registers/starts the new ones and unregisters orphaned roots.
    async fn reconcile_virtualfs(&self, app: &AppHandle) {
        let pairs = self.enabled_ondemand_pairs().await;
        let vfs = app.state::<AppState>().virtualfs.clone();
        vfs.reconcile(app, pairs).await;
    }

    /// Merge config with live status for the UI.
    pub async fn views(&self) -> Vec<PairView> {
        let pairs = self.settings.lock().await.pairs.clone();
        let running = self.running.lock().await;
        let mut out = Vec::with_capacity(pairs.len());
        for pair in pairs {
            let (is_running, rt) = match running.get(&pair.id) {
                Some(r) => (true, r.status.lock().await.clone()),
                None => (false, RuntimeStatus::default()),
            };
            out.push(PairView {
                running: is_running,
                state: rt.state,
                in_flight: rt.in_flight,
                last_synced: rt.last_synced,
                last_error: rt.last_error,
                pair,
            });
        }
        out
    }

    /// A mirror root must never resolve to a drive/filesystem root or a
    /// symbolic link. A missing directory can be configured for later, but
    /// it cannot be used for a destructive reconcile.
    fn validate_mirror_local_root(path: &Path, require_existing: bool) -> Result<()> {
        let metadata = match std::fs::symlink_metadata(path) {
            Ok(meta) => meta,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound && !require_existing => {
                return Ok(());
            }
            Err(error) => return Err(error).context("inspect mirror local root"),
        };
        anyhow::ensure!(
            metadata.is_dir() && !metadata.file_type().is_symlink(),
            "mirror sync requires a real local directory, not a symlink or file"
        );
        let canonical = std::fs::canonicalize(path).context("resolve mirror local root")?;
        anyhow::ensure!(
            canonical.parent().is_some()
                && canonical
                    .components()
                    .filter(|part| matches!(part, std::path::Component::Normal(_)))
                    .count()
                    > 0,
            "mirror sync refuses a filesystem or drive root"
        );
        Ok(())
    }

    fn validate_pair(pair: &SyncPair) -> Result<()> {
        if pair.name.trim().is_empty() {
            anyhow::bail!("sync pair name must not be empty");
        }
        if pair.profile_id.trim().is_empty() {
            anyhow::bail!("sync pair connection must not be empty");
        }
        let local = Path::new(pair.local_root.trim());
        if pair.local_root.trim().is_empty() || !local.is_absolute() {
            anyhow::bail!("sync local root must be an absolute path");
        }
        if pair.remote_root.trim().is_empty() {
            anyhow::bail!("sync remote root must not be empty");
        }
        if matches!(pair.strategy, SyncStrategy::Mirror) {
            let remote = pair.remote_root.trim().replace('\\', "/");
            // Dot segments are dangerous even when hidden inside a longer
            // path: '/uploads/..' resolves to root and a Mirror may delete
            // unrelated remote files. Treat both slash spellings the same.
            let segments: Vec<&str> = remote
                .split('/')
                .filter(|segment| !segment.is_empty())
                .collect();
            if segments.is_empty()
                || segments
                    .iter()
                    .any(|segment| matches!(*segment, "." | ".."))
            {
                anyhow::bail!("mirror sync refuses remote root aliases or traversal");
            }
            if local.parent().is_none()
                || local
                    .components()
                    .any(|component| component == std::path::Component::ParentDir)
            {
                anyhow::bail!("mirror sync refuses a local filesystem root or parent traversal");
            }
            Self::validate_mirror_local_root(local, false)?;
        }
        Ok(())
    }

    /// Add or replace a pair. If it's enabled, (re)start it.
    pub async fn upsert(&self, app: &AppHandle, mut pair: SyncPair) -> Result<()> {
        if pair.id.is_empty() {
            pair.id = uuid::Uuid::new_v4().to_string();
        }
        if pair.poll_interval_secs == 0 {
            pair.poll_interval_secs = DEFAULT_POLL_SECS;
        }
        Self::validate_pair(&pair)?;
        {
            let mut s = self.settings.lock().await;
            let previous = s.clone();
            match s.pairs.iter_mut().find(|p| p.id == pair.id) {
                Some(existing) => *existing = pair.clone(),
                None => s.pairs.push(pair.clone()),
            }
            if let Err(error) = self.persist_snapshot(&s) {
                *s = previous;
                return Err(error);
            }
        }
        self.stop_pair(app, &pair.id).await;
        if pair.enabled {
            self.start_pair(app, &pair).await?;
        }
        self.reconcile_virtualfs(app).await;
        Ok(())
    }

    pub async fn remove(&self, app: &AppHandle, id: &str) -> Result<()> {
        {
            let mut s = self.settings.lock().await;
            let previous = s.clone();
            s.pairs.retain(|p| p.id != id);
            if let Err(error) = self.persist_snapshot(&s) {
                *s = previous;
                return Err(error);
            }
        }
        self.stop_pair(app, id).await;
        // Unregister the OS sync root if this was an on-demand pair.
        self.reconcile_virtualfs(app).await;
        Ok(())
    }

    pub async fn set_enabled(&self, app: &AppHandle, id: &str, enabled: bool) -> Result<()> {
        let pair = {
            let mut s = self.settings.lock().await;
            let previous = s.clone();
            let p = s.pairs.iter_mut().find(|p| p.id == id);
            let pair = match p {
                Some(p) => {
                    p.enabled = enabled;
                    p.clone()
                }
                None => anyhow::bail!("no such sync pair"),
            };
            if let Err(error) = self.persist_snapshot(&s) {
                *s = previous;
                return Err(error);
            }
            pair
        };
        if enabled {
            self.stop_pair(app, id).await;
            self.start_pair(app, &pair).await?;
        } else {
            self.stop_pair(app, id).await;
        }
        self.reconcile_virtualfs(app).await;
        Ok(())
    }

    /// Fire an immediate reconcile for a running pair.
    pub async fn sync_now(&self, id: &str) -> Result<()> {
        let running = self.running.lock().await;
        match running.get(id) {
            Some(r) => {
                // Repeated manual requests coalesce into one pending reconcile.
                let _ = r.trigger.try_send(());
                Ok(())
            }
            None => anyhow::bail!("sync pair is not enabled"),
        }
    }

    async fn stop_pair(&self, app: &AppHandle, id: &str) {
        if let Some(r) = self.running.lock().await.remove(id) {
            r.task.abort();
            if let Some(session_id) = r.session_id.lock().await.take() {
                let state = app.state::<AppState>();
                if let Err(error) = state.sessions.disconnect(&session_id).await {
                    tracing::debug!(
                        "folder sync '{id}': background session cleanup failed: {error:#}"
                    );
                }
            }
            // Dropping `_watcher` unregisters the filesystem watch.
        }
    }

    async fn start_pair(&self, app: &AppHandle, pair: &SyncPair) -> Result<()> {
        // On-demand pairs are driven by the VirtualFs provider (placeholders +
        // hydration on access), not this eager reconcile loop. `reconcile_virtualfs`
        // stands them up; nothing to spawn here.
        if pair.mode == SyncMode::OnDemand {
            return Ok(());
        }

        let mut running = self.running.lock().await;
        if running.contains_key(&pair.id) {
            return Ok(());
        }

        // A one-slot wake-up prevents thousands of FS events exhausting RAM.
        let (trigger, mut trigger_rx) = mpsc::channel::<()>(1);
        let status = Arc::new(Mutex::new(RuntimeStatus::default()));

        // Watch the local folder for LocalToRemote so edits sync promptly.
        // RemoteToLocal relies on the poll backstop (no remote change feed).
        let watcher = if matches!(pair.direction, SyncDirection::LocalToRemote) {
            match make_watcher(&pair.local_root, trigger.clone()) {
                Ok(w) => Some(w),
                Err(e) => {
                    tracing::warn!("folder sync '{}': watcher unavailable: {e:#}", pair.name);
                    None
                }
            }
        } else {
            None
        };

        let task_app = app.clone();
        let task_pair = pair.clone();
        let task_status = status.clone();
        let session_id = Arc::new(Mutex::new(None));
        let task_session_id = session_id.clone();
        let poll = Duration::from_secs(pair.poll_interval_secs.max(1));
        let task = tauri::async_runtime::spawn(async move {
            // A live session for this pair, connected lazily and rebuilt on error.
            let mut session_id = task_session_id.lock().await.take();
            // Reconcile once immediately on start.
            let mut pending = true;
            loop {
                if !pending {
                    tokio::select! {
                        _ = tokio::time::sleep(poll) => {}
                        r = trigger_rx.recv() => {
                            if r.is_none() { break; } // all senders dropped → stop
                            // Debounce short bursts, but a continuous
                            // stream must not starve reconciliation forever.
                            // No more than five seconds of delay per burst.
                            let disconnected = tokio::time::timeout(
                                Duration::from_secs(5),
                                async {
                                    loop {
                                        tokio::select! {
                                            _ = tokio::time::sleep(DEBOUNCE) => return false,
                                            r2 = trigger_rx.recv() => {
                                                if r2.is_none() { return true; }
                                            }
                                        }
                                    }
                                },
                            )
                            .await
                            .unwrap_or(false);
                            if disconnected { return; }
                        }
                    }
                }
                pending = false;
                reconcile(&task_app, &task_pair, &task_status, &mut session_id).await;
                *task_session_id.lock().await = session_id.clone();
            }
        });

        running.insert(
            pair.id.clone(),
            Running {
                task,
                trigger,
                _watcher: watcher,
                status,
                session_id,
            },
        );
        Ok(())
    }
}

/// Build a recursive filesystem watcher. Queue capacity is one: editor saves
/// may emit thousands of rename/write events, but every burst needs one scan.
fn make_watcher(root: &str, trigger: mpsc::Sender<()>) -> Result<notify::RecommendedWatcher> {
    let mut watcher = notify::recommended_watcher(move |res: notify::Result<notify::Event>| {
        match res {
            Ok(ev) => {
                use notify::EventKind;
                if matches!(
                    ev.kind,
                    EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_)
                ) {
                    let _ = trigger.try_send(());
                }
            }
            Err(error) => {
                // Lost/overflowed platform watcher events require a full
                // rescan. Do not silently wait for the next minute-long poll.
                tracing::warn!("folder sync watcher needs rescan: {error}");
                let _ = trigger.try_send(());
            }
        }
    })?;
    watcher.watch(Path::new(root), RecursiveMode::Recursive)?;
    Ok(watcher)
}

/// One reconcile pass for a pair: (re)connect, plan, execute, wait, update status.
async fn reconcile(
    app: &AppHandle,
    pair: &SyncPair,
    status: &Arc<Mutex<RuntimeStatus>>,
    session_id: &mut Option<String>,
) {
    let state = app.state::<AppState>();
    set_state(app, status, PairState::Scanning, |_| {}).await;

    let result = reconcile_inner(app, &state, pair, session_id).await;
    match result {
        // `Some(warning)` = the reconcile completed, but the Mirror-delete guard
        // held some deletes back. Uploads still happened, so we report success
        // (Idle + last_synced) while surfacing the guard note in `last_error`.
        Ok(warning) => {
            // A delete safety guard may deliberately leave work unfinished.
            // Surface the warning and keep the last fully-synced timestamp.
            let complete = warning.is_none();
            set_state(app, status, PairState::Idle, |s| {
                s.in_flight = 0;
                if complete {
                    s.last_synced = Some(now_ms());
                }
                s.last_error = warning;
            })
            .await;
        }
        Err(e) => {
            // Drop the session so the next tick reconnects from scratch.
            *session_id = None;
            let msg = format!("{e:#}");
            tracing::warn!("folder sync '{}': {msg}", pair.name);
            set_state(app, status, PairState::Error, move |s| {
                s.in_flight = 0;
                s.last_error = Some(msg);
            })
            .await;
        }
    }
}

/// Runs one reconcile. `Ok(None)` = clean; `Ok(Some(msg))` = completed but the
/// Mirror-delete guard withheld deletes (a warning, not a failure); `Err` = the
/// reconcile failed outright.
async fn reconcile_inner(
    app: &AppHandle,
    state: &AppState,
    pair: &SyncPair,
    session_id: &mut Option<String>,
) -> Result<Option<String>> {
    // Recheck at execution time: a previously configured directory can turn
    // into a symlink or vanish before the next polling/watch reconcile.
    // Never plan Mirror deletions against a redirected or missing local root.
    if matches!(pair.strategy, SyncStrategy::Mirror) {
        FolderSync::validate_pair(pair)?;
        FolderSync::validate_mirror_local_root(Path::new(&pair.local_root), true)?;
    }
    // Delta-sync temp hygiene (Plan 23 Phase 3): a crash mid-delta can strand
    // `.ghostftp-patch-*` / `.ghostftp-new-*` / `.ghostftp-delta-*.tmp` files next to the
    // synced files. Best-effort sweep of day-old ones, once per reconcile,
    // LOCAL root only — remote-side temps are the daemon's own concern.
    {
        let root = pair.local_root.clone();
        let _ = tokio::task::spawn_blocking(move || sweep_delta_temps(Path::new(&root))).await;
    }

    // Ensure a live session for this pair's connection.
    let session = ensure_session(app, state, pair, session_id).await?;

    let local_fs: Box<dyn crate::remotefs::RemoteFs> = Box::new(crate::remotefs::local::LocalFs);
    let remote_fs = crate::commands::fs_for_session(&session);

    // The index tracks the *source* side (authoritative for one-way sync); its
    // change signal comes from that backend's capabilities.
    let source_signal = match pair.direction {
        SyncDirection::LocalToRemote => local_fs.capabilities().change_signal,
        SyncDirection::RemoteToLocal => remote_fs.capabilities().change_signal,
    };
    // Refuse to plan against an invented empty index when SQLite is locked,
    // damaged, or inaccessible. Especially important in Mirror mode: a false
    // empty history can convert a temporary I/O fault into destructive work.
    let index = state
        .db
        .load_sync_state(&pair.id)
        .with_context(|| format!("reading saved sync history for '{}'", pair.name))?;

    let (mut plan, source_tree) = crate::sync::plan_indexed(
        local_fs.as_ref(),
        remote_fs.as_ref(),
        &pair.local_root,
        &pair.remote_root,
        pair.direction,
        pair.strategy,
        Some(&index),
        source_signal,
    )
    .await
    .context("planning sync")?;

    // For Mirror, learn whether the source root currently has any entries
    // *before* trusting the diff's delete set (see `apply_safety`). Skip the
    // extra listing for Additive, which never deletes.
    let source_available = if matches!(pair.strategy, SyncStrategy::Mirror) {
        let (src_fs, src_root): (&dyn crate::remotefs::RemoteFs, &str) = match pair.direction {
            SyncDirection::LocalToRemote => (local_fs.as_ref(), pair.local_root.as_str()),
            SyncDirection::RemoteToLocal => (remote_fs.as_ref(), pair.remote_root.as_str()),
        };
        matches!(src_fs.list_dir(src_root).await, Ok(e) if !e.is_empty())
    } else {
        true
    };

    let warning = apply_safety(&mut plan, pair, source_available);

    if plan.copies.is_empty() && plan.deletes.is_empty() {
        // Nothing to transfer — but still snapshot the source so a first run
        // seeds the index (later same-size edits become detectable) and a delete
        // prunes its row.
        if warning.is_none() {
            snapshot_index(&state.db, &pair.id, &source_tree)?;
        }
        return Ok(warning);
    }

    let ids = crate::commands::execute_sync_plan(session, plan, &state.transfers, app)
        .await
        .context("executing sync")?;

    // Do not claim a pair was synchronized, or advance the persistent source
    // index, when any queued transfer failed, was canceled or timed out.
    wait_for_transfers(state, ids).await?;

    // Record what the source looks like now, so the next reconcile compares
    // against reality rather than re-uploading (resume) and catches in-place
    // edits the live size/mtime diff would miss.
    if warning.is_none() {
        snapshot_index(&state.db, &pair.id, &source_tree)?;
    }
    Ok(warning)
}

/// Store one consistent per-pair snapshot, or leave the previous SQLite rows
/// intact. No partial upsert/delete loops; a failed write must be visible to
/// the caller and must not advance the last successful synchronization time.
fn snapshot_index(
    db: &crate::db::Db,
    pair_id: &str,
    source_tree: &crate::scan::ScanTree,
) -> Result<()> {
    db.replace_sync_state_snapshot(pair_id, &source_tree.files, now_ms())
        .context("commit atomic sync history")
}

// Bound temporary-file cleanup work after crashes and watcher storms.
const DELTA_SWEEP_MAX_ENTRIES: usize = 100_000;
const DELTA_SWEEP_MAX_DEPTH: usize = 64;

/// Remove stale delta transfer temps inside a selected local sync tree.
/// Do not traverse symlinked directories or remove symlinked files. This
/// preflight is not descriptor-relative and does not close every TOCTOU race.
fn sweep_delta_temps(root: &Path) {
    let Ok(root_meta) = std::fs::symlink_metadata(root) else {
        return;
    };
    if !root_meta.is_dir() || root_meta.file_type().is_symlink() {
        return;
    }
    let mut directories = vec![(root.to_path_buf(), 0usize)];
    let mut visited = 0usize;
    while let Some((directory, depth)) = directories.pop() {
        let Ok(entries) = std::fs::read_dir(&directory) else {
            continue;
        };
        for entry in entries {
            if visited >= DELTA_SWEEP_MAX_ENTRIES {
                tracing::warn!("folder sync: delta temp sweep entry limit reached");
                return;
            }
            visited += 1;
            let Ok(entry) = entry else { continue };
            let path = entry.path();
            // DirEntry::metadata follows symlinks; symlink_metadata does not.
            let Ok(meta) = std::fs::symlink_metadata(&path) else {
                continue;
            };
            if meta.file_type().is_symlink() {
                continue;
            }
            if meta.is_dir() {
                if depth < DELTA_SWEEP_MAX_DEPTH {
                    directories.push((path, depth + 1));
                }
                continue;
            }
            if !meta.is_file() {
                continue;
            }
            let name = entry.file_name().to_string_lossy().into_owned();
            let is_delta_temp = name.contains(".ghostftp-patch-")
                || name.contains(".ghostftp-new-")
                || name.contains(".ghostftp-delta-");
            if !is_delta_temp {
                continue;
            }
            let stale = meta
                .modified()
                .ok()
                .and_then(|m| m.elapsed().ok())
                .map(|age| age > DELTA_TEMP_MAX_AGE)
                .unwrap_or(false);
            if stale {
                if let Err(e) = std::fs::remove_file(&path) {
                    tracing::debug!("folder sync: delta temp sweep {}: {e:#}", path.display());
                }
            }
        }
    }
}

/// Return a live `Arc<Session>` for the pair, reusing the cached session id when
/// it's still valid, otherwise connecting the profile fresh.
async fn ensure_session(
    app: &AppHandle,
    state: &AppState,
    pair: &SyncPair,
    session_id: &mut Option<String>,
) -> Result<Arc<crate::session::Session>> {
    if let Some(id) = session_id.as_deref() {
        if let Some(s) = state.sessions.get(id).await {
            return Ok(s);
        }
    }
    let profile = state
        .profiles
        .get(&pair.profile_id)
        .await
        .context("loading the sync target's connection")?
        .ok_or_else(|| anyhow::anyhow!("connection for this sync pair no longer exists"))?;
    let id = state
        .sessions
        .connect(profile, app.clone())
        .await
        .context("connecting to the sync target")?;
    let session = state
        .sessions
        .get(&id)
        .await
        .ok_or_else(|| anyhow::anyhow!("session vanished right after connect"))?;
    *session_id = Some(id);
    Ok(session)
}

/// True only if every transfer actually completed (or was deliberately
/// skipped under the active overwrite policy). A missing ledger record is NOT
/// success: otherwise the indexed sync claims unchanged data was uploaded.
fn transfer_batch_complete(statuses: &[Option<TransferStatus>]) -> Result<bool> {
    let mut pending = false;
    for status in statuses {
        match status {
            Some(TransferStatus::Done) => {}
            Some(
                TransferStatus::Queued | TransferStatus::Transferring | TransferStatus::Paused,
            ) => {
                pending = true;
            }
            // A Skipped upload is not proof that the destination contains the
            // source bytes. Do not commit the new source index until a future
            // reconcile can verify the destination or the user resolves it.
            Some(TransferStatus::Skipped) => {
                anyhow::bail!(
                    "sync transfer was skipped without verification; source index was not advanced"
                );
            }
            Some(TransferStatus::Error | TransferStatus::Canceled) | None => {
                anyhow::bail!("sync has a failed, canceled or missing transfer; source index was not advanced");
            }
        }
    }
    Ok(!pending)
}

/// Wait for all queued work to succeed. Timeout and missing rows are errors,
/// not successful reconciliations; preserve old index so the next watcher or
/// poll retries instead of silently losing changed files.
async fn wait_for_transfers(state: &AppState, ids: Vec<String>) -> Result<()> {
    if ids.is_empty() {
        return Ok(());
    }
    let start = std::time::Instant::now();
    loop {
        let mut statuses = Vec::with_capacity(ids.len());
        for id in &ids {
            statuses.push(state.transfers.snapshot(id).await.map(|t| t.status));
        }
        if transfer_batch_complete(&statuses)? {
            return Ok(());
        }
        if start.elapsed() > TRANSFER_WAIT_CAP {
            anyhow::bail!("sync transfer wait timed out; source index was not advanced");
        }
        tokio::time::sleep(Duration::from_millis(300)).await;
    }
}

/// Update status, run a mutator, and emit a change event so the UI refreshes.
async fn set_state(
    app: &AppHandle,
    status: &Arc<Mutex<RuntimeStatus>>,
    state: PairState,
    mutate: impl FnOnce(&mut RuntimeStatus),
) {
    {
        let mut s = status.lock().await;
        s.state = state;
        mutate(&mut s);
    }
    let _ = app.emit("foldersync://changed", ());
}

// ---------- safety: exclude patterns + Mirror-delete guard ----------

/// Apply exclude patterns and the Mirror-delete guard to a freshly-planned
/// `SyncPlan`, in place. Returns `Some(warning)` when the guard withheld
/// deletes (the uploads still stand — only the destructive half is gated).
///
/// `source_available` is whether the source root currently lists any entries.
/// An empty or unreadable source is the classic footgun — an unmounted drive,
/// a deleted folder, a typo'd path — where the diff sees "nothing on the
/// source" and a naive Mirror would wipe the entire destination. We refuse.
fn apply_safety(plan: &mut SyncPlan, pair: &SyncPair, source_available: bool) -> Option<String> {
    // Excludes: drop matching files from BOTH copies and deletes, so an excluded
    // path is neither uploaded nor (under Mirror) deleted on the far side.
    // Internal partial downloads and delta reconstruction files must never
    // be published to a live server or targeted for Mirror deletion. They
    // can appear while a transfer writes into a watched local directory.
    plan.copies.retain(|c| {
        !is_internal_transfer_temp(&c.relative) && !is_excluded(&c.relative, &pair.exclude)
    });
    plan.deletes.retain(|d| {
        !is_internal_transfer_temp(&d.relative) && !is_excluded(&d.relative, &pair.exclude)
    });
    plan.total_bytes = plan.copies.iter().map(|c| c.size).sum();

    if !matches!(pair.strategy, SyncStrategy::Mirror) || plan.deletes.is_empty() {
        return None;
    }

    if !source_available {
        let n = plan.deletes.len();
        plan.deletes.clear();
        return Some(format!(
            "Mirror guard: skipped deleting {n} destination file(s) — the source folder is empty or unreadable (is it mounted?)."
        ));
    }

    if pair.mirror_delete_cap != 0 && plan.deletes.len() as u32 > pair.mirror_delete_cap {
        let n = plan.deletes.len();
        plan.deletes.clear();
        return Some(format!(
            "Mirror guard: skipped deleting {n} destination file(s) — over the {}-file safety cap. Raise the cap if this is intended.",
            pair.mirror_delete_cap
        ));
    }

    None
}

fn is_internal_transfer_temp(relative: &str) -> bool {
    relative.split('/').any(|part| {
        (part.starts_with(".ghostftp-download-")
            && (part.ends_with(".part") || part.ends_with(".backup")))
            || part.starts_with(".ghostftp-patch-")
            || part.starts_with(".ghostftp-new-")
            || (part.starts_with(".ghostftp-delta-") && part.ends_with(".tmp"))
    })
}

// ---------- exclude patterns ----------

/// gitignore-lite matcher. A relative path (POSIX, '/'-separated) is excluded
/// when any pattern matches. Rules:
/// - blank lines and `#` comments are ignored;
/// - a trailing `/` is stripped (dir patterns match the same as file patterns);
/// - a pattern with no `/` (e.g. `node_modules`, `*.log`, `.DS_Store`) matches
///   any single path segment at any depth;
/// - a pattern containing `/` (or a leading `/`) is matched against the whole
///   relative path, anchored at the root, and also excludes everything beneath
///   a matched directory.
///
/// Wildcards: `*` matches within a segment, `**` crosses `/`, `?` is one
/// non-`/` char.
fn is_excluded(rel: &str, patterns: &[String]) -> bool {
    let rel = rel.trim_start_matches('/');
    for raw in patterns {
        let pat = raw.trim();
        if pat.is_empty() || pat.starts_with('#') {
            continue;
        }
        let anchored = pat.starts_with('/');
        let pat = pat.trim_start_matches('/').trim_end_matches('/');
        if pat.is_empty() {
            continue;
        }
        if anchored || pat.contains('/') {
            // Whole-path match, plus descendants of a matched directory.
            if glob_match(pat, rel) || glob_match(&format!("{pat}/**"), rel) {
                return true;
            }
        } else {
            // Basename-style: match against any single path segment.
            if rel.split('/').any(|seg| glob_match(pat, seg)) {
                return true;
            }
        }
    }
    false
}

/// Linear-memory NFA matcher. Previous recursive backtracking could cost
/// exponential CPU on overlapping '*'/'**' patterns in automatic folder sync.
/// Byte-level '?' and path-segment semantics match the old implementation.
fn glob_match(pattern: &str, text: &str) -> bool {
    #[derive(Clone, Copy)]
    enum Token {
        Literal(u8),
        One,
        SegmentStar,
        AnyStar,
    }

    fn epsilon_closure(active: &mut [bool], tokens: &[Token]) {
        for (index, token) in tokens.iter().enumerate() {
            if active[index] && matches!(token, Token::SegmentStar | Token::AnyStar) {
                active[index + 1] = true;
            }
        }
    }

    let pattern = pattern.as_bytes();
    let mut tokens = Vec::with_capacity(pattern.len());
    let mut i = 0;
    while i < pattern.len() {
        match pattern[i] {
            b'*' if i + 1 < pattern.len() && pattern[i + 1] == b'*' => {
                tokens.push(Token::AnyStar);
                i += 2;
                if i < pattern.len() && pattern[i] == b'/' {
                    i += 1;
                }
            }
            b'*' => {
                tokens.push(Token::SegmentStar);
                i += 1;
            }
            b'?' => {
                tokens.push(Token::One);
                i += 1;
            }
            literal => {
                tokens.push(Token::Literal(literal));
                i += 1;
            }
        }
    }

    let mut active = vec![false; tokens.len() + 1];
    let mut next = vec![false; tokens.len() + 1];
    active[0] = true;
    epsilon_closure(&mut active, &tokens);
    for &byte in text.as_bytes() {
        next.fill(false);
        for (index, token) in tokens.iter().enumerate() {
            if !active[index] {
                continue;
            }
            match token {
                Token::Literal(c) if *c == byte => next[index + 1] = true,
                Token::One if byte != b'/' => next[index + 1] = true,
                Token::SegmentStar if byte != b'/' => next[index] = true,
                Token::AnyStar => next[index] = true,
                _ => {}
            }
        }
        epsilon_closure(&mut next, &tokens);
        std::mem::swap(&mut active, &mut next);
    }
    active[tokens.len()]
}

#[cfg(test)]
mod tests {
    use super::{
        apply_safety, glob_match, is_excluded, is_internal_transfer_temp, load_settings,
        sweep_delta_temps, transfer_batch_complete, SyncMode, SyncPair,
    };
    use crate::sync::{self, SyncDirection, SyncStrategy};
    use crate::transfer::TransferStatus;
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU32, Ordering};

    fn ex(rel: &str, pats: &[&str]) -> bool {
        let pats: Vec<String> = pats.iter().map(|s| s.to_string()).collect();
        is_excluded(rel, &pats)
    }

    #[test]
    fn continuous_sync_does_not_advance_index_for_failed_or_missing_work() {
        assert!(transfer_batch_complete(&[]).unwrap());
        assert!(transfer_batch_complete(&[Some(TransferStatus::Done)]).unwrap());
        // Skip is terminal for a user action, but it does not validate the
        // destination and must not be committed as confirmed source state.
        assert!(transfer_batch_complete(&[
            Some(TransferStatus::Done),
            Some(TransferStatus::Skipped)
        ])
        .is_err());
        assert!(!transfer_batch_complete(&[
            Some(TransferStatus::Done),
            Some(TransferStatus::Transferring)
        ])
        .unwrap());
        for bad in [
            Some(TransferStatus::Error),
            Some(TransferStatus::Canceled),
            None,
        ] {
            assert!(transfer_batch_complete(&[Some(TransferStatus::Done), bad]).is_err());
        }
    }

    #[test]
    fn continuous_sync_ignores_its_own_incomplete_transfer_files() {
        for name in [
            ".ghostftp-download-abc.part",
            "nested/.ghostftp-download-abc.backup",
            ".ghostftp-patch-123",
            "nested/.ghostftp-new-abcdef",
            "nested/.ghostftp-delta-random.tmp",
        ] {
            assert!(is_internal_transfer_temp(name), "{name}");
        }
        for normal in [
            ".ghostftp-project.json",
            "code.php",
            "nested/.ghostftp-download-example.txt",
        ] {
            assert!(!is_internal_transfer_temp(normal), "{normal}");
        }
    }

    #[test]
    fn sqlite_import_preserves_legacy_json_and_refuses_malformed_configuration() {
        let root = std::env::temp_dir().join(format!(
            "ghostftp-sqlite-sync-test-{}",
            uuid::Uuid::new_v4()
        ));
        std::fs::create_dir(&root).unwrap();
        let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
        let legacy = root.join("foldersync.json");
        let fixture = r#"{"pairs":[]}"#;
        std::fs::write(&legacy, fixture).unwrap();
        assert!(load_settings(&db, &root).unwrap().pairs.is_empty());
        assert_eq!(std::fs::read_to_string(&legacy).unwrap(), fixture);
        assert!(root.join("foldersync.sqlite-migrated").is_file());
        assert_eq!(
            db.folder_sync_config_get().unwrap().as_deref(),
            Some(fixture)
        );

        std::fs::write(&legacy, b"{ broken snapshot").unwrap();
        // SQLite is authoritative after migration, so a stale or damaged
        // legacy backup must never override it.
        assert!(load_settings(&db, &root).is_ok());

        db.folder_sync_config_set("{ invalid db").unwrap();
        assert!(load_settings(&db, &root).is_err());
        drop(db);
        std::fs::remove_file(root.join("ghostftp.db")).unwrap();
        let blank_db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
        assert!(load_settings(&blank_db, &root).is_err());
        drop(blank_db);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn malformed_legacy_sync_file_is_never_silently_replaced_with_empty_pairs() {
        let root =
            std::env::temp_dir().join(format!("ghostftp-malformed-sync-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&root).unwrap();
        std::fs::write(root.join("foldersync.json"), b"{ invalid").unwrap();
        let db = crate::db::Db::open(&root.join("ghostftp.db")).unwrap();
        assert!(load_settings(&db, &root).is_err());
        assert!(db.folder_sync_config_get().unwrap().is_none());
        drop(db);
        std::fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn basename_patterns_match_at_any_depth() {
        assert!(ex("node_modules/react/index.js", &["node_modules"]));
        assert!(ex(".git/config", &[".git"]));
        assert!(ex("a/b/c.log", &["*.log"]));
        assert!(ex(".DS_Store", &[".DS_Store"]));
        assert!(!ex("src/main.rs", &["*.log", "node_modules"]));
    }

    #[test]
    fn path_patterns_are_anchored_and_cover_descendants() {
        assert!(ex("dist/app.js", &["/dist"])); // anchored dir + descendants
        assert!(ex("build/x/y.o", &["build/**"]));
        assert!(ex("build/app.js", &["build"])); // path-anchored dir name
        assert!(!ex("src/dist/app.js", &["/dist"])); // anchored, not any-depth
        assert!(ex("a/b/temp.tmp", &["a/**/*.tmp"]));
    }

    #[test]
    fn blanks_and_comments_are_ignored() {
        assert!(!ex("src/main.rs", &["", "  ", "# a comment"]));
    }

    #[test]
    fn adversarial_globs_complete_without_exponential_recursion() {
        let pattern = format!("{}Z", "*a".repeat(90));
        let value = "a".repeat(90);
        assert!(!glob_match(&pattern, &value));
        assert!(glob_match(&pattern, &format!("{value}Z")));
        assert!(glob_match(&"**/".repeat(512), "deep/folder/file.txt"));
        assert!(glob_match("**/report*.pdf", "računi/report čćž.pdf"));
        assert!(!glob_match("files/*.txt", "files/nested/notes.txt"));
        assert!(glob_match("files/?otes.txt", "files/notes.txt"));
    }

    #[test]
    fn mirror_sync_never_accepts_remote_or_local_parent_traversal() {
        use super::FolderSync;
        let mut candidate = pair(&[], 100);
        candidate.local_root = std::env::temp_dir()
            .join("ghostftp-mirror-test")
            .to_string_lossy()
            .into_owned();
        candidate.remote_root = "/safe-folder".into();
        candidate.profile_id = "test-profile".into();
        assert!(FolderSync::validate_pair(&candidate).is_ok());

        for remote in [
            "/",
            "//",
            "/safe/..",
            "/safe/./files",
            "safe\\..\\",
            "../escape",
        ] {
            candidate.remote_root = remote.into();
            assert!(
                FolderSync::validate_pair(&candidate).is_err(),
                "unsafe mirror remote root was accepted: {remote:?}"
            );
        }
        candidate.remote_root = "/safe-folder".into();
        candidate.local_root = std::env::temp_dir()
            .join("ghostftp-mirror-test")
            .join("..")
            .join("other-folder")
            .to_string_lossy()
            .into_owned();
        assert!(FolderSync::validate_pair(&candidate).is_err());
    }

    #[cfg(unix)]
    #[test]
    fn mirror_sync_rejects_local_symlink_and_filesystem_root_alias() {
        use super::FolderSync;
        use std::os::unix::fs::symlink;
        let dir =
            std::env::temp_dir().join(format!("ghostftp-mirror-root-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir(&dir).unwrap();
        let mut candidate = pair(&[], 100);
        candidate.profile_id = "test-profile".into();
        candidate.remote_root = "/safe-folder".into();
        candidate.strategy = SyncStrategy::Mirror;
        let root_link = dir.join("root-link");
        symlink("/", &root_link).unwrap();
        candidate.local_root = root_link.to_string_lossy().into_owned();
        assert!(FolderSync::validate_pair(&candidate).is_err());
        candidate.local_root = "/".into();
        assert!(FolderSync::validate_pair(&candidate).is_err());
        candidate.local_root = dir.join("missing").to_string_lossy().into_owned();
        assert!(FolderSync::validate_pair(&candidate).is_ok());
        assert!(FolderSync::validate_mirror_local_root(
            std::path::Path::new(&candidate.local_root),
            true
        )
        .is_err());
        candidate.local_root = dir.to_string_lossy().into_owned();
        assert!(FolderSync::validate_pair(&candidate).is_ok());
        assert!(FolderSync::validate_mirror_local_root(&dir, true).is_ok());
        std::fs::remove_dir_all(dir).unwrap();
    }

    #[test]
    fn bursty_sync_notifications_use_constant_queue_capacity() {
        let (tx, mut rx) = tokio::sync::mpsc::channel::<()>(1);
        assert!(tx.try_send(()).is_ok());
        for _ in 0..50_000 {
            assert!(matches!(
                tx.try_send(()),
                Err(tokio::sync::mpsc::error::TrySendError::Full(()))
            ));
        }
        assert!(rx.try_recv().is_ok());
        assert!(tx.try_send(()).is_ok());
    }

    // ----- on-disk exercise of the real planner + guard -----

    static COUNTER: AtomicU32 = AtomicU32::new(0);

    fn scratch(tag: &str) -> PathBuf {
        let n = COUNTER.fetch_add(1, Ordering::Relaxed);
        let mut d = std::env::temp_dir();
        d.push(format!("ghostftp_fs_test_{}_{tag}_{n}", std::process::id()));
        let _ = std::fs::remove_dir_all(&d);
        std::fs::create_dir_all(&d).unwrap();
        d
    }

    fn write(root: &std::path::Path, rel: &str, body: &str) {
        let p = root.join(rel);
        std::fs::create_dir_all(p.parent().unwrap()).unwrap();
        std::fs::write(p, body).unwrap();
    }

    /// Delta-temp sweep (Plan 23 Phase 3): stale `*.ghostftp-patch-*` /
    /// `*.ghostftp-new-*` / `*.ghostftp-delta-*.tmp` files are removed (even nested);
    /// fresh temps and normal files are untouched.
    #[test]
    fn delta_temp_sweep_removes_only_stale_temps() {
        use std::time::{Duration, SystemTime};

        let root = scratch("sweep");
        write(&root, "sub/.ghostftp-patch-abc", "x");
        write(&root, "file.bin.ghostftp-new-123-4", "x");
        write(&root, ".ghostftp-delta-xyz.tmp", "x");
        write(&root, "fresh.ghostftp-patch-def", "x");
        write(&root, "normal.txt", "x");

        // Age the three stale temps past the 24 h cutoff.
        let old = SystemTime::now() - Duration::from_secs(48 * 60 * 60);
        for rel in [
            "sub/.ghostftp-patch-abc",
            "file.bin.ghostftp-new-123-4",
            ".ghostftp-delta-xyz.tmp",
        ] {
            std::fs::File::options()
                .write(true)
                .open(root.join(rel))
                .unwrap()
                .set_modified(old)
                .unwrap();
        }

        sweep_delta_temps(&root);

        assert!(
            !root.join("sub/.ghostftp-patch-abc").exists(),
            "stale nested patch swept"
        );
        assert!(
            !root.join("file.bin.ghostftp-new-123-4").exists(),
            "stale assemble temp swept"
        );
        assert!(
            !root.join(".ghostftp-delta-xyz.tmp").exists(),
            "stale download temp swept"
        );
        assert!(
            root.join("fresh.ghostftp-patch-def").exists(),
            "fresh temp survives"
        );
        assert!(root.join("normal.txt").exists(), "normal file survives");

        let _ = std::fs::remove_dir_all(&root);
    }

    #[cfg(unix)]
    #[test]
    fn delta_sweep_does_not_follow_symlinks_outside_sync_root() {
        use std::os::unix::fs::symlink;
        use std::time::{Duration, SystemTime};

        let root = scratch("sweep_symlink_root");
        let outside = scratch("sweep_symlink_outside");
        write(&outside, ".ghostftp-patch-secret", "keep");
        write(&outside, "nested/.ghostftp-delta-secret.tmp", "keep too");
        write(&root, ".ghostftp-patch-stale", "remove");
        let old = SystemTime::now() - Duration::from_secs(48 * 60 * 60);
        for path in [
            root.join(".ghostftp-patch-stale"),
            outside.join(".ghostftp-patch-secret"),
            outside.join("nested/.ghostftp-delta-secret.tmp"),
        ] {
            std::fs::File::options()
                .write(true)
                .open(path)
                .unwrap()
                .set_modified(old)
                .unwrap();
        }
        symlink(&outside, root.join("linked_folder")).unwrap();
        symlink(
            outside.join(".ghostftp-patch-secret"),
            root.join(".ghostftp-patch-linked"),
        )
        .unwrap();
        sweep_delta_temps(&root);
        assert!(!root.join(".ghostftp-patch-stale").exists());
        assert!(root.join("linked_folder").is_symlink());
        assert!(root.join(".ghostftp-patch-linked").is_symlink());
        assert_eq!(
            std::fs::read(outside.join(".ghostftp-patch-secret")).unwrap(),
            b"keep"
        );
        assert!(outside.join("nested/.ghostftp-delta-secret.tmp").exists());
        sweep_delta_temps(&root.join("linked_folder"));
        assert!(outside.join(".ghostftp-patch-secret").exists());
        let _ = std::fs::remove_dir_all(&root);
        let _ = std::fs::remove_dir_all(&outside);
    }

    fn pair(exclude: &[&str], cap: u32) -> SyncPair {
        SyncPair {
            id: "t".into(),
            name: "t".into(),
            local_root: String::new(),
            profile_id: String::new(),
            remote_root: String::new(),
            direction: SyncDirection::LocalToRemote,
            strategy: SyncStrategy::Mirror,
            mode: SyncMode::Mirror,
            enabled: true,
            poll_interval_secs: 60,
            exclude: exclude.iter().map(|s| s.to_string()).collect(),
            mirror_delete_cap: cap,
        }
    }

    async fn plan_dirs(src: &std::path::Path, dst: &std::path::Path) -> sync::SyncPlan {
        let fs = crate::remotefs::local::LocalFs;
        sync::plan(
            &fs,
            &fs,
            src.to_str().unwrap(),
            dst.to_str().unwrap(),
            SyncDirection::LocalToRemote,
            SyncStrategy::Mirror,
        )
        .await
        .unwrap()
    }

    #[test]
    fn excludes_and_guard_over_real_files() {
        let rt = tokio::runtime::Builder::new_current_thread()
            .build()
            .unwrap();
        rt.block_on(async {
            let src = scratch("src");
            let dst = scratch("dst");

            // Source tree: a normal file, an excluded dir, an excluded ext,
            // and a nested keeper.
            write(&src, "keep.txt", "hi");
            write(&src, "sub/b.txt", "b");
            write(&src, "node_modules/pkg/index.js", "x");
            write(&src, "debug.log", "log");

            // Destination has extras (Mirror deletes) — including one under an
            // excluded dir that must survive.
            write(&dst, "extra1.txt", "1");
            write(&dst, "extra2.txt", "2");
            write(&dst, "node_modules/pkg/old.js", "old");

            let excl = &["node_modules", "*.log"];

            // 1) Excludes filter copies + protect excluded deletes.
            let mut p = plan_dirs(&src, &dst).await;
            let w = apply_safety(&mut p, &pair(excl, 100), true);
            assert!(w.is_none(), "under cap, available source → no warning");
            let copies: Vec<&str> = p.copies.iter().map(|c| c.relative.as_str()).collect();
            assert!(copies.contains(&"sub/b.txt"));
            assert!(!copies.iter().any(|r| r.contains("node_modules")));
            assert!(!copies.iter().any(|r| r.ends_with(".log")));
            let dels: Vec<&str> = p.deletes.iter().map(|d| d.relative.as_str()).collect();
            assert!(dels.contains(&"extra1.txt") && dels.contains(&"extra2.txt"));
            assert!(
                !dels.iter().any(|r| r.contains("node_modules")),
                "excluded destination files are never mirror-deleted"
            );

            // 2) Delete cap trips → deletes withheld, warning surfaced.
            let mut p = plan_dirs(&src, &dst).await;
            let w = apply_safety(&mut p, &pair(excl, 1), true);
            assert!(w.unwrap().contains("safety cap"));
            assert!(p.deletes.is_empty(), "over-cap deletes are withheld");
            assert!(!p.copies.is_empty(), "uploads still proceed past the guard");

            // 3) Empty/unreadable source → never wipe the destination.
            let mut p = plan_dirs(&src, &dst).await;
            let w = apply_safety(&mut p, &pair(excl, 100), false);
            assert!(w.unwrap().contains("empty or unreadable"));
            assert!(p.deletes.is_empty());

            let _ = std::fs::remove_dir_all(&src);
            let _ = std::fs::remove_dir_all(&dst);
        });
    }
}

// ---------- Tauri commands ----------

fn err<E: std::fmt::Display>(e: E) -> String {
    e.to_string()
}

#[tauri::command]
pub async fn foldersync_list(state: State<'_, AppState>) -> Result<Vec<PairView>, String> {
    Ok(state.foldersync.views().await)
}

#[tauri::command]
pub async fn foldersync_upsert(
    pair: SyncPair,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<Vec<PairView>, String> {
    state.foldersync.upsert(&app, pair).await.map_err(err)?;
    Ok(state.foldersync.views().await)
}

#[tauri::command]
pub async fn foldersync_remove(
    id: String,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<Vec<PairView>, String> {
    state.foldersync.remove(&app, &id).await.map_err(err)?;
    // Drop the pair's index rows so a future pair reusing the id starts clean.
    let _ = state.db.clear_pair(&id);
    Ok(state.foldersync.views().await)
}

#[tauri::command]
pub async fn foldersync_set_enabled(
    id: String,
    enabled: bool,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<Vec<PairView>, String> {
    state
        .foldersync
        .set_enabled(&app, &id, enabled)
        .await
        .map_err(err)?;
    Ok(state.foldersync.views().await)
}

#[tauri::command]
pub async fn foldersync_sync_now(
    id: String,
    state: State<'_, AppState>,
) -> Result<Vec<PairView>, String> {
    state.foldersync.sync_now(&id).await.map_err(err)?;
    Ok(state.foldersync.views().await)
}
