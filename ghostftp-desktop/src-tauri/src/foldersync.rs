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
use std::path::{Path, PathBuf};
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
    trigger: mpsc::UnboundedSender<()>,
    /// Kept alive so the filesystem watch stays registered.
    _watcher: Option<notify::RecommendedWatcher>,
    status: Arc<Mutex<RuntimeStatus>>,
    /// Session created for this background pair. Shared with stop_pair so an
    /// aborted reconcile cannot strand a hidden live connection.
    session_id: Arc<Mutex<Option<String>>>,
}

pub struct FolderSync {
    settings_path: PathBuf,
    settings: Mutex<Settings>,
    running: Mutex<HashMap<String, Running>>,
}

impl FolderSync {
    pub fn load(app: &AppHandle) -> Result<Self> {
        let dir = app
            .path()
            .app_data_dir()
            .context("resolving app_data_dir")?;
        std::fs::create_dir_all(&dir).ok();
        let settings_path = dir.join("foldersync.json");
        let settings: Settings = std::fs::read(&settings_path)
            .ok()
            .and_then(|b| serde_json::from_slice(&b).ok())
            .unwrap_or_default();
        Ok(Self {
            settings_path,
            settings: Mutex::new(settings),
            running: Mutex::new(HashMap::new()),
        })
    }

    fn persist_snapshot(&self, settings: &Settings) -> Result<()> {
        let bytes = serde_json::to_vec_pretty(settings)?;
        let parent = self
            .settings_path
            .parent()
            .context("folder sync settings path has no parent")?;
        std::fs::create_dir_all(parent).with_context(|| format!("create {}", parent.display()))?;
        let tmp = self.settings_path.with_extension("json.tmp");
        let backup = self.settings_path.with_extension("json.bak");
        std::fs::write(&tmp, bytes).with_context(|| format!("write {}", tmp.display()))?;

        // Windows does not reliably allow rename(tmp, existing_target). Move the
        // previous settings aside first, then commit the new snapshot and restore
        // the backup if that commit fails.
        let had_previous = self.settings_path.exists();
        if had_previous {
            let _ = std::fs::remove_file(&backup);
            std::fs::rename(&self.settings_path, &backup)
                .with_context(|| format!("backup {}", self.settings_path.display()))?;
        }
        if let Err(error) = std::fs::rename(&tmp, &self.settings_path) {
            let _ = std::fs::remove_file(&tmp);
            if had_previous {
                let _ = std::fs::rename(&backup, &self.settings_path);
            }
            return Err(error).with_context(|| format!("replace {}", self.settings_path.display()));
        }
        if had_previous {
            let _ = std::fs::remove_file(&backup);
        }
        Ok(())
    }

    async fn persist(&self) -> Result<()> {
        let settings = self.settings.lock().await.clone();
        self.persist_snapshot(&settings)
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
            if remote == "/" || remote == "." || remote == ".." {
                anyhow::bail!("mirror sync refuses a remote root path");
            }
            if local.parent().is_none() {
                anyhow::bail!("mirror sync refuses a local filesystem root");
            }
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
            self.stop_pair(id).await;
        }
        self.reconcile_virtualfs(app).await;
        Ok(())
    }

    /// Fire an immediate reconcile for a running pair.
    pub async fn sync_now(&self, id: &str) -> Result<()> {
        let running = self.running.lock().await;
        match running.get(id) {
            Some(r) => {
                let _ = r.trigger.send(());
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

        let (trigger, mut trigger_rx) = mpsc::unbounded_channel::<()>();
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
                            // debounce the burst
                            loop {
                                tokio::select! {
                                    _ = tokio::time::sleep(DEBOUNCE) => break,
                                    r2 = trigger_rx.recv() => { if r2.is_none() { return; } }
                                }
                            }
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

/// Build a recursive filesystem watcher that pings `trigger` on any change.
fn make_watcher(
    root: &str,
    trigger: mpsc::UnboundedSender<()>,
) -> Result<notify::RecommendedWatcher> {
    let mut watcher = notify::recommended_watcher(move |res: notify::Result<notify::Event>| {
        if let Ok(ev) = res {
            use notify::EventKind;
            if matches!(
                ev.kind,
                EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_)
            ) {
                let _ = trigger.send(());
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
            set_state(app, status, PairState::Idle, |s| {
                s.in_flight = 0;
                s.last_synced = Some(now_ms());
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
    let index = state.db.load_sync_state(&pair.id).unwrap_or_else(|e| {
        tracing::warn!("folder sync '{}': reading sync_state: {e:#}", pair.name);
        Default::default()
    });

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
        snapshot_index(&state.db, &pair.id, &source_tree, &index);
        return Ok(warning);
    }

    let ids = crate::commands::execute_sync_plan(session, plan, &state.transfers, app)
        .await
        .context("executing sync")?;

    wait_for_transfers(state, ids).await;

    // Record what the source looks like now, so the next reconcile compares
    // against reality rather than re-uploading (resume) and catches in-place
    // edits the live size/mtime diff would miss.
    snapshot_index(&state.db, &pair.id, &source_tree, &index);
    Ok(warning)
}

/// Persist the current source tree into `sync_state`: upsert a row per file and
/// prune rows for files that have disappeared from the source. Best-effort — a
/// DB hiccup here only costs the optimization, never correctness (the live diff
/// still runs every reconcile).
fn snapshot_index(
    db: &crate::db::Db,
    pair_id: &str,
    source_tree: &crate::scan::ScanTree,
    before: &std::collections::HashMap<String, crate::db::SyncStateRow>,
) {
    let now = now_ms();
    for (rel, e) in &source_tree.files {
        if let Err(err) =
            db.upsert_sync_state(pair_id, rel, e.size, e.modified, e.etag.as_deref(), now)
        {
            tracing::warn!("folder sync: sync_state upsert '{rel}': {err:#}");
        }
    }
    for rel in before.keys() {
        if !source_tree.files.contains_key(rel) {
            let _ = db.delete_sync_state(pair_id, rel);
        }
    }
}

/// Best-effort recursive sweep of delta-sync temp files (`*.ghostftp-patch-*`,
/// `*.ghostftp-new-*`, `*.ghostftp-delta-*.tmp`) older than [`DELTA_TEMP_MAX_AGE`]
/// under the LOCAL sync root (Plan 23 Phase 3). The matcher is filename-only
/// and the prefixes are delta-specific, so a normal user file can never
/// match; every error is ignored — a locked or racing temp simply survives to
/// the next reconcile.
fn sweep_delta_temps(root: &Path) {
    let Ok(entries) = std::fs::read_dir(root) else {
        return;
    };
    for entry in entries.flatten() {
        let path = entry.path();
        let Ok(meta) = entry.metadata() else {
            continue;
        };
        if meta.is_dir() {
            sweep_delta_temps(&path);
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

/// Block until every queued transfer reaches a terminal state (or the cap).
async fn wait_for_transfers(state: &AppState, ids: Vec<String>) {
    if ids.is_empty() {
        return;
    }
    let start = std::time::Instant::now();
    loop {
        let mut remaining = 0u32;
        for id in &ids {
            if let Some(t) = state.transfers.snapshot(id).await {
                match t.status {
                    TransferStatus::Queued
                    | TransferStatus::Transferring
                    | TransferStatus::Paused => remaining += 1,
                    _ => {}
                }
            }
        }
        if remaining == 0 || start.elapsed() > TRANSFER_WAIT_CAP {
            return;
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
    if !pair.exclude.is_empty() {
        plan.copies
            .retain(|c| !is_excluded(&c.relative, &pair.exclude));
        plan.deletes
            .retain(|d| !is_excluded(&d.relative, &pair.exclude));
        plan.total_bytes = plan.copies.iter().map(|c| c.size).sum();
    }

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

/// Glob match with `*` (any run within a segment), `**` (any run incl. `/`),
/// and `?` (one non-`/` char). Both sides are POSIX strings.
fn glob_match(pattern: &str, text: &str) -> bool {
    glob_rec(pattern.as_bytes(), text.as_bytes())
}

fn glob_rec(p: &[u8], t: &[u8]) -> bool {
    if p.is_empty() {
        return t.is_empty();
    }
    match p[0] {
        b'*' if p.len() >= 2 && p[1] == b'*' => {
            // `**` — optionally followed by `/`; matches any run including `/`.
            let rest = if p.len() >= 3 && p[2] == b'/' {
                &p[3..]
            } else {
                &p[2..]
            };
            if glob_rec(rest, t) {
                return true;
            }
            (0..t.len()).any(|i| glob_rec(rest, &t[i + 1..]))
        }
        b'*' => {
            // `*` — any run of non-`/` chars.
            if glob_rec(&p[1..], t) {
                return true;
            }
            let mut i = 0;
            while i < t.len() && t[i] != b'/' {
                if glob_rec(&p[1..], &t[i + 1..]) {
                    return true;
                }
                i += 1;
            }
            false
        }
        b'?' => !t.is_empty() && t[0] != b'/' && glob_rec(&p[1..], &t[1..]),
        c => !t.is_empty() && t[0] == c && glob_rec(&p[1..], &t[1..]),
    }
}

#[cfg(test)]
mod tests {
    use super::{apply_safety, is_excluded, sweep_delta_temps, SyncMode, SyncPair};
    use crate::sync::{self, SyncDirection, SyncStrategy};
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU32, Ordering};

    fn ex(rel: &str, pats: &[&str]) -> bool {
        let pats: Vec<String> = pats.iter().map(|s| s.to_string()).collect();
        is_excluded(rel, &pats)
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
