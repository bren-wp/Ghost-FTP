use crate::session::ftp::FtpTransferControl;
use crate::session::{
    BoxSession, DropboxSession, DynamicsSession, FtpSession, GDriveSession, HttpSession,
    HubSpotSession, ObjectSession, OneDriveSession, Session, ShopifySession, SshSession,
    WebdavSession,
};
use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet, VecDeque};
use std::future::Future;
use std::io::{Seek, SeekFrom, Write};
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, AtomicU64, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use tauri::{AppHandle, Emitter};
use tokio::io::{AsyncReadExt, AsyncSeekExt, AsyncWriteExt};
use tokio::sync::{watch, Mutex, OwnedSemaphorePermit, Semaphore};
use tokio::task::JoinHandle;
use uuid::Uuid;

#[derive(Debug, Clone, Copy, Deserialize, Default, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum OverwritePolicy {
    #[default]
    Overwrite,
    Skip,
    Rename,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TransferKind {
    Download,
    Upload,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TransferStatus {
    Queued,
    Transferring,
    Paused,
    Done,
    Skipped,
    Error,
    Canceled,
}

/// Marker error: a paused transfer was resumed. The active copy loop unwinds
/// so the runner can reopen the backend. Backends with proven ranged-resume
/// support continue from the last committed byte; all others restart safely.
#[derive(Debug)]
struct RestartFromPause;

impl std::fmt::Display for RestartFromPause {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("transfer resumed after pause")
    }
}

impl std::error::Error for RestartFromPause {}

/// Payload of the `transfer://queue` event: the FIFO of waiting transfer ids
/// plus the manager-level state the panel header renders.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct QueueState {
    pub waiting: Vec<String>,
    pub paused_all: bool,
    pub concurrency: usize,
    pub throttle_kbps: u64,
}

/// Global bandwidth cap shared by every active copy loop:
/// a token bucket refilling at `rate` bytes/sec (0 = unlimited). Because all
/// transfers draw from this one bucket, the cap is split across active
/// transfers rather than applied per transfer.
struct TokenBucket {
    inner: Mutex<BucketInner>,
    rate_bps: AtomicU64,
}

struct BucketInner {
    tokens: f64,
    // tokio's Instant (not std's) so `start_paused` tests drive refills.
    last: tokio::time::Instant,
}

impl TokenBucket {
    fn new() -> Self {
        Self {
            inner: Mutex::new(BucketInner {
                tokens: 0.0,
                last: tokio::time::Instant::now(),
            }),
            rate_bps: AtomicU64::new(0),
        }
    }

    fn rate_kbps(&self) -> u64 {
        self.rate_bps.load(Ordering::Relaxed) / 1024
    }

    fn set_rate_kbps(&self, kbps: u64) {
        self.rate_bps
            .store(kbps.saturating_mul(1024), Ordering::Relaxed);
    }

    /// Wait until `bytes` may flow under the cap. Drawn in tranches capped at
    /// one second's worth of rate (min 64 KiB) so a large chunk is charged in
    /// full while a 1 KiB file never waits a whole token window.
    async fn acquire(&self, bytes: u64) {
        let mut remaining = bytes as f64;
        while remaining > 0.0 {
            let rate = self.rate_bps.load(Ordering::Relaxed);
            if rate == 0 {
                return;
            }
            let wait = {
                let mut g = self.inner.lock().await;
                let now = tokio::time::Instant::now();
                let elapsed = now.duration_since(g.last).as_secs_f64();
                let rate_f = rate as f64;
                let cap = rate_f.max(64.0 * 1024.0);
                g.tokens = (g.tokens + elapsed * rate_f).min(cap);
                g.last = now;
                let tranche = remaining.min(cap);
                if g.tokens >= tranche {
                    g.tokens -= tranche;
                    remaining -= tranche;
                    continue;
                }
                Duration::from_secs_f64((tranche - g.tokens) / rate_f)
            };
            // Cap the sleep so a live rate change takes effect promptly.
            tokio::time::sleep(wait.min(Duration::from_millis(250))).await;
        }
    }
}

/// A pause gate shared by the scheduler (pause-all) and individual transfers
/// Watch-channel based so waiters never miss a wakeup.
#[derive(Debug, Clone)]
pub struct PauseGate {
    tx: watch::Sender<bool>,
    // Keeps the channel open: with zero receivers `send()` silently fails.
    _rx: watch::Receiver<bool>,
}

impl PauseGate {
    fn new() -> Self {
        let (tx, _rx) = watch::channel(false);
        Self { tx, _rx }
    }
    fn is_paused(&self) -> bool {
        *self.tx.borrow()
    }
    fn set(&self, paused: bool) {
        let _ = self.tx.send(paused);
    }
    /// Park until the gate opens. Returns immediately if already open.
    async fn wait_open(&self) {
        let mut rx = self.tx.subscribe();
        while *rx.borrow_and_update() {
            if rx.changed().await.is_err() {
                break;
            }
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Transfer {
    pub id: String,
    pub kind: TransferKind,
    pub source: String,
    pub destination: String,
    pub size: u64,
    pub transferred: u64,
    pub status: TransferStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    /// Auto-retry round in progress (1- or 2-of-2), for the panel's
    /// "retrying in Ns (attempt N/total)" state.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub retry_attempt: Option<u32>,
    /// Delta-sync accounting, present when this transfer ran as a block-level
    /// delta instead of a whole-file copy: how many bytes actually crossed the
    /// wire vs. how many were reused from the basis.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub delta: Option<DeltaStats>,
    pub started_at: i64,
}

/// Delta-sync outcome attached to a finished [`Transfer`] (Agent backend only).
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DeltaStats {
    /// Literal bytes that crossed the wire.
    pub sent: u64,
    /// Bytes reused from the basis (never transferred).
    pub reused: u64,
}

/// Everything needed to re-run a failed/canceled transfer with its already
/// policy-resolved destination (overwrite/skip/rename was applied at enqueue
/// time) — used by manual retry.
#[derive(Clone)]
enum RetryInfo {
    Download {
        session: Arc<Session>,
        remote_path: String,
        final_path: PathBuf,
    },
    Upload {
        session: Arc<Session>,
        local: PathBuf,
        final_remote: String,
    },
}

/// Credential-free recovery descriptor persisted beside each transfer row.
/// The profile id reconnects through the existing ProfileStore/OS credential
/// path only when the user explicitly retries after a restart.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
enum PersistedRetryInfo {
    Download {
        profile_id: String,
        remote_path: String,
        final_path: PathBuf,
    },
    Upload {
        profile_id: String,
        local: PathBuf,
        final_remote: String,
    },
}

impl PersistedRetryInfo {
    fn profile_id(&self) -> &str {
        match self {
            Self::Download { profile_id, .. } | Self::Upload { profile_id, .. } => profile_id,
        }
    }

    fn with_session(&self, session: Arc<Session>) -> RetryInfo {
        match self {
            Self::Download {
                remote_path,
                final_path,
                ..
            } => RetryInfo::Download {
                session,
                remote_path: remote_path.clone(),
                final_path: final_path.clone(),
            },
            Self::Upload {
                local,
                final_remote,
                ..
            } => RetryInfo::Upload {
                session,
                local: local.clone(),
                final_remote: final_remote.clone(),
            },
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct PersistedTransferEnvelope {
    schema: u32,
    transfer: Transfer,
    retry: PersistedRetryInfo,
}

const TRANSFER_LEDGER_SCHEMA: u32 = 1;
const TRANSFER_LEDGER_KEEP: usize = 500;
const TRANSFER_LEDGER_PROGRESS_INTERVAL: Duration = Duration::from_secs(1);

/// Default bound on concurrently running transfers; the rest wait
/// in the FIFO as `Queued`. Overridden by the `transferConcurrency` setting.
const DEFAULT_CONCURRENCY: usize = 3;

/// Default number of automatic retries for transient network/timeout failures.
/// The user can change this live from Preferences; manual Retry remains separate.
const DEFAULT_MAX_AUTO_RETRIES: usize = 3;

pub struct TransferManager {
    transfers: Mutex<HashMap<String, Transfer>>,
    tasks: Mutex<HashMap<String, JoinHandle<()>>>,
    /// FIFO of transfer ids waiting for a slot. An id is popped only
    /// once it is at the front, the pause-all gate is open, and a concurrency
    /// permit is available — until then it waits as `Queued`.
    waiting: Mutex<VecDeque<String>>,
    semaphore: Arc<Semaphore>,
    concurrency: AtomicUsize,
    /// Manager-level pause gate. Admission checks it; transfer checkpoints do too.
    pause_all: PauseGate,
    /// Per-transfer pause gates, created at enqueue time.
    pauses: Mutex<HashMap<String, PauseGate>>,
    /// Cooperative cancellation flags. These matter most for FTP/FTPS where
    /// spawn_blocking cannot be force-aborted once a data stream is active.
    cancels: Mutex<HashMap<String, Arc<AtomicBool>>>,
    /// Original resolved inputs per transfer, for manual retry.
    retry: Mutex<HashMap<String, RetryInfo>>,
    /// Credential-free retry descriptors survive process restarts. They contain
    /// only profile ids and resolved paths; secrets remain in the OS keychain.
    persisted_retry: Mutex<HashMap<String, PersistedRetryInfo>>,
    /// Rows restored from a previous process require a live-session reattach
    /// before retry. Their historical byte count is display-only: without a
    /// persisted source identity/change signal, cross-process retry restarts at
    /// byte zero to avoid combining an old prefix with changed content.
    recovered_retry: Mutex<HashSet<String>>,
    /// Optional shared SQLite store. Unit tests that don't exercise persistence
    /// use `new()`; the desktop app uses `with_db()`.
    db: Option<Arc<crate::db::Db>>,
    /// Throttle progress persistence to avoid turning a fast transfer into a
    /// high-frequency SQLite writer. State transitions still flush immediately.
    persisted_at: Mutex<HashMap<String, Instant>>,
    /// Bumped on every queue change so admission waiters re-check their turn.
    queue_gen: watch::Sender<u64>,
    /// Global bandwidth cap every copy loop draws from per chunk.
    bucket: TokenBucket,
    /// Maximum automatic retries for transient network/timeout failures.
    max_auto_retries: AtomicUsize,
    /// Delta-sync master switch: the `deltaSync` setting,
    /// live-adjustable like the concurrency bound. `GHOSTFTP_DELTA=0` still
    /// force-disables regardless of this flag.
    delta_enabled: AtomicBool,
}

fn now_ts() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

/// Join a remote directory and a file name using the separator the directory
/// already speaks: backslash for a Windows-style path (`C:\srv`, `\\host\share`),
/// forward slash everywhere else. An existing trailing separator is reused
/// rather than doubled.
fn join_remote(dir: &str, name: &str) -> String {
    if dir.is_empty() {
        return name.to_string();
    }
    if dir.ends_with('/') || dir.ends_with('\\') {
        return format!("{dir}{name}");
    }
    let windows_style = dir.contains('\\') && !dir.contains('/');
    format!("{dir}{}{name}", if windows_style { '\\' } else { '/' })
}

fn basename(path: &str) -> String {
    path.rsplit(['/', '\\']).next().unwrap_or(path).to_string()
}

fn download_staging_path(final_path: &Path, id: &str) -> PathBuf {
    let parent = final_path.parent().unwrap_or(Path::new("."));
    parent.join(format!(".ghostftp-download-{id}.part"))
}

fn download_backup_path(final_path: &Path, id: &str) -> PathBuf {
    let parent = final_path.parent().unwrap_or(Path::new("."));
    parent.join(format!(".ghostftp-download-{id}.backup"))
}

async fn local_path_exists_fail_closed(path: &Path) -> Result<bool> {
    match tokio::fs::metadata(path).await {
        Ok(_) => Ok(true),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(false),
        Err(error) => {
            Err(error).with_context(|| format!("stat local download target {}", path.display()))
        }
    }
}

async fn promote_staged_download(staging: &Path, final_path: &Path, id: &str) -> Result<()> {
    let backup = download_backup_path(final_path, id);
    let mut backup_created = false;

    if local_path_exists_fail_closed(final_path).await? {
        let _ = tokio::fs::remove_file(&backup).await;
        tokio::fs::rename(final_path, &backup)
            .await
            .with_context(|| {
                format!(
                    "preserve existing local download target {} before replacement",
                    final_path.display()
                )
            })?;
        backup_created = true;
    }

    if let Err(error) = tokio::fs::rename(staging, final_path).await {
        if backup_created {
            match tokio::fs::rename(&backup, final_path).await {
                Ok(()) => {
                    return Err(error).with_context(|| {
                        format!(
                            "promote staged download over {}; original target was restored",
                            final_path.display()
                        )
                    });
                }
                Err(restore_error) => {
                    return Err(anyhow::anyhow!(
                        "failed to promote staged download over {}; original target could not be restored automatically and remains at {}: {}; restore error: {}",
                        final_path.display(),
                        backup.display(),
                        error,
                        restore_error
                    ));
                }
            }
        }
        return Err(error)
            .with_context(|| format!("promote staged download to {}", final_path.display()));
    }

    if backup_created {
        let _ = tokio::fs::remove_file(&backup).await;
    }
    Ok(())
}

pub(crate) fn require_directory_upload_metadata(
    path: &Path,
    metadata: std::io::Result<std::fs::Metadata>,
) -> Result<std::fs::Metadata> {
    metadata.with_context(|| {
        format!(
            "stat directory upload entry {}; refusing to continue with an incomplete source tree",
            path.display()
        )
    })
}

const MAX_RENAME_CANDIDATES: usize = 999;

/// Append _1, _2, … to the stem until a free local path is found. Never fall
/// back to the colliding original path: Rename must not silently become Overwrite.
fn resolve_local_rename(path: &Path) -> Result<PathBuf> {
    if !path.exists() {
        return Ok(path.to_path_buf());
    }
    let parent = path.parent().unwrap_or(Path::new("."));
    let stem = path
        .file_stem()
        .map(|s| s.to_string_lossy().into_owned())
        .unwrap_or_default();
    let ext = path
        .extension()
        .map(|e| format!(".{}", e.to_string_lossy()))
        .unwrap_or_default();
    for i in 1..=MAX_RENAME_CANDIDATES {
        let candidate = parent.join(format!("{stem}_{i}{ext}"));
        if !candidate.exists() {
            return Ok(candidate);
        }
    }
    anyhow::bail!(
        "no free rename target after {MAX_RENAME_CANDIDATES} candidates for {}",
        path.display()
    )
}

/// Resolve Rename without ever treating an ambiguous remote probe failure as
/// proof that a candidate is free. The probe returns `Ok(false)` only for a
/// protocol-proven absent path; permission, timeout, connection and other
/// errors propagate fail-closed instead of degrading Rename into Overwrite.
async fn resolve_remote_rename_with_probe<F, Fut, E>(
    path: &str,
    mut probe_exists: F,
) -> Result<String>
where
    F: FnMut(String) -> Fut,
    Fut: Future<Output = std::result::Result<bool, E>>,
    E: std::fmt::Display,
{
    let original = path.to_string();
    let original_exists = probe_exists(original.clone()).await.map_err(|error| {
        anyhow::anyhow!("unable to verify remote rename target {original}: {error}")
    })?;
    if !original_exists {
        return Ok(original);
    }

    for i in 1..=MAX_RENAME_CANDIDATES {
        let candidate = remote_rename_candidate(path, i);
        let exists = probe_exists(candidate.clone()).await.map_err(|error| {
            anyhow::anyhow!("unable to verify remote rename candidate {candidate}: {error}")
        })?;
        if !exists {
            return Ok(candidate);
        }
    }

    anyhow::bail!(
        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {path}"
    )
}

async fn resolve_remote_rename(
    sftp: &russh_sftp::client::SftpSession,
    path: &str,
) -> Result<String> {
    resolve_remote_rename_with_probe(path, |candidate| sftp.try_exists(candidate)).await
}

impl TransferManager {
    pub fn new() -> Self {
        Self::from_db(None)
    }

    pub fn with_db(db: Arc<crate::db::Db>) -> Self {
        Self::from_db(Some(db))
    }

    fn from_db(db: Option<Arc<crate::db::Db>>) -> Self {
        let mut transfers = HashMap::new();
        let mut persisted_retry = HashMap::new();
        let mut recovered_retry = HashSet::new();

        if let Some(store) = db.as_ref() {
            if let Ok(rows) = store.transfer_ledger_list() {
                let mut terminal_kept = 0usize;
                for (row_id, payload, _) in rows {
                    let Ok(mut envelope) =
                        serde_json::from_str::<PersistedTransferEnvelope>(&payload)
                    else {
                        let _ = store.transfer_ledger_delete(&row_id);
                        continue;
                    };
                    if envelope.schema != TRANSFER_LEDGER_SCHEMA || envelope.transfer.id != row_id {
                        let _ = store.transfer_ledger_delete(&row_id);
                        continue;
                    }

                    let was_active = matches!(
                        envelope.transfer.status,
                        TransferStatus::Queued
                            | TransferStatus::Transferring
                            | TransferStatus::Paused
                    );

                    // Always retain rows that were active when the previous
                    // process stopped, even if they are older than the history
                    // budget. Only terminal history is bounded.
                    if !was_active {
                        if terminal_kept >= TRANSFER_LEDGER_KEEP {
                            let _ = store.transfer_ledger_delete(&row_id);
                            continue;
                        }
                        terminal_kept += 1;
                    }

                    let needs_session_reattach = was_active
                        || matches!(
                            envelope.transfer.status,
                            TransferStatus::Error | TransferStatus::Canceled
                        );

                    if was_active {
                        envelope.transfer.status = TransferStatus::Error;
                        envelope.transfer.retry_attempt = None;
                        envelope.transfer.error = Some(
                            "Transfer was interrupted by the previous Ghost FTP process. Retry will reconnect the saved profile and restart safely from byte zero because cross-process file identity was not verified."
                                .to_string(),
                        );
                        if let Ok(normalized) = serde_json::to_string(&envelope) {
                            let _ = store.transfer_ledger_upsert(&row_id, &normalized);
                        }
                    }

                    if needs_session_reattach {
                        recovered_retry.insert(row_id.clone());
                    }
                    persisted_retry.insert(row_id.clone(), envelope.retry);
                    transfers.insert(row_id, envelope.transfer);
                }
            }
        }

        Self {
            transfers: Mutex::new(transfers),
            tasks: Mutex::new(HashMap::new()),
            waiting: Mutex::new(VecDeque::new()),
            semaphore: Arc::new(Semaphore::new(DEFAULT_CONCURRENCY)),
            concurrency: AtomicUsize::new(DEFAULT_CONCURRENCY),
            pause_all: PauseGate::new(),
            pauses: Mutex::new(HashMap::new()),
            cancels: Mutex::new(HashMap::new()),
            retry: Mutex::new(HashMap::new()),
            persisted_retry: Mutex::new(persisted_retry),
            recovered_retry: Mutex::new(recovered_retry),
            db,
            persisted_at: Mutex::new(HashMap::new()),
            queue_gen: watch::channel(0).0,
            bucket: TokenBucket::new(),
            max_auto_retries: AtomicUsize::new(DEFAULT_MAX_AUTO_RETRIES),
            delta_enabled: AtomicBool::new(true),
        }
    }

    pub async fn list(&self) -> Vec<Transfer> {
        let mut v: Vec<Transfer> = self.transfers.lock().await.values().cloned().collect();
        v.sort_by_key(|t| t.started_at);
        v
    }

    async fn get(&self, id: &str) -> Option<Transfer> {
        self.transfers.lock().await.get(id).cloned()
    }

    /// Public snapshot of a single transfer by id (used by the Agent Bridge so
    /// an agent can poll whether a download/upload it started has finished).
    pub async fn snapshot(&self, id: &str) -> Option<Transfer> {
        self.transfers.lock().await.get(id).cloned()
    }

    async fn update<F: FnOnce(&mut Transfer)>(&self, id: &str, f: F) {
        let snapshot = {
            let mut transfers = self.transfers.lock().await;
            transfers.get_mut(id).map(|transfer| {
                f(transfer);
                transfer.clone()
            })
        };
        if let Some(transfer) = snapshot {
            let force = transfer.status != TransferStatus::Transferring
                || transfer.transferred >= transfer.size;
            self.persist_snapshot(transfer, force).await;
        }
    }

    async fn insert(&self, t: Transfer) {
        self.transfers.lock().await.insert(t.id.clone(), t.clone());
        self.persist_snapshot(t, true).await;
    }

    /// Spawn and register a transfer worker while holding the task-map lock.
    ///
    /// Tokio may poll a spawned future immediately. Taking this lock before
    /// `tokio::spawn` guarantees a very fast worker that reaches `finalize()`
    /// blocks on the same map until its JoinHandle is registered. Once this
    /// method releases the lock, `finalize()` can deterministically remove the
    /// registered handle; there is no completion-before-registration window.
    async fn spawn_registered<F>(&self, id: String, future: F)
    where
        F: Future<Output = ()> + Send + 'static,
    {
        let mut tasks = self.tasks.lock().await;
        let task = tokio::spawn(future);
        tasks.insert(id, task);
    }

    async fn persist_snapshot(&self, transfer: Transfer, force: bool) {
        let Some(db) = self.db.as_ref() else {
            return;
        };
        let retry = self.persisted_retry.lock().await.get(&transfer.id).cloned();
        let Some(retry) = retry else {
            return;
        };

        if !force {
            let now = Instant::now();
            let mut persisted_at = self.persisted_at.lock().await;
            if persisted_at
                .get(&transfer.id)
                .is_some_and(|last| now.duration_since(*last) < TRANSFER_LEDGER_PROGRESS_INTERVAL)
            {
                return;
            }
            persisted_at.insert(transfer.id.clone(), now);
        } else {
            self.persisted_at
                .lock()
                .await
                .insert(transfer.id.clone(), Instant::now());
        }

        let envelope = PersistedTransferEnvelope {
            schema: TRANSFER_LEDGER_SCHEMA,
            transfer,
            retry,
        };
        if let Ok(payload) = serde_json::to_string(&envelope) {
            if db
                .transfer_ledger_upsert(&envelope.transfer.id, &payload)
                .is_ok()
                && matches!(
                    envelope.transfer.status,
                    TransferStatus::Done
                        | TransferStatus::Skipped
                        | TransferStatus::Error
                        | TransferStatus::Canceled
                )
            {
                self.prune_terminal_history().await;
            }
        }
    }

    /// Keep at most TRANSFER_LEDGER_KEEP terminal rows while preserving every
    /// active row. This bounds both SQLite and the in-memory transfer panel for
    /// long-running clients instead of waiting for the next application start.
    async fn prune_terminal_history(&self) {
        let stale = {
            let mut transfers = self.transfers.lock().await;
            let mut terminal: Vec<(String, i64)> = transfers
                .values()
                .filter(|transfer| {
                    matches!(
                        transfer.status,
                        TransferStatus::Done
                            | TransferStatus::Skipped
                            | TransferStatus::Error
                            | TransferStatus::Canceled
                    )
                })
                .map(|transfer| (transfer.id.clone(), transfer.started_at))
                .collect();

            terminal.sort_by(|(left_id, left_started), (right_id, right_started)| {
                right_started
                    .cmp(left_started)
                    .then_with(|| right_id.cmp(left_id))
            });

            let stale: Vec<String> = terminal
                .into_iter()
                .skip(TRANSFER_LEDGER_KEEP)
                .map(|(id, _)| id)
                .collect();
            for id in &stale {
                transfers.remove(id);
            }
            stale
        };

        if stale.is_empty() {
            return;
        }

        {
            let mut retry = self.retry.lock().await;
            let mut persisted_retry = self.persisted_retry.lock().await;
            let mut recovered_retry = self.recovered_retry.lock().await;
            let mut pauses = self.pauses.lock().await;
            let mut cancels = self.cancels.lock().await;
            let mut persisted_at = self.persisted_at.lock().await;
            for id in &stale {
                retry.remove(id);
                persisted_retry.remove(id);
                recovered_retry.remove(id);
                pauses.remove(id);
                cancels.remove(id);
                persisted_at.remove(id);
            }
        }

        if let Some(db) = self.db.as_ref() {
            for id in stale {
                let _ = db.transfer_ledger_delete(&id);
            }
        }
    }

    /// Profile needed to retry a row restored from a previous process.
    pub async fn recovery_profile_id(&self, id: &str) -> Option<String> {
        if !self.recovered_retry.lock().await.contains(id) {
            return None;
        }
        self.persisted_retry
            .lock()
            .await
            .get(id)
            .map(|retry| retry.profile_id().to_string())
    }

    /// Bind a freshly reconnected live session to a persisted retry descriptor.
    /// The profile id must match so a recovered transfer can never be redirected
    /// to a different saved site by mistake.
    pub async fn attach_recovered_session(&self, id: &str, session: Arc<Session>) -> Result<()> {
        let descriptor = self
            .persisted_retry
            .lock()
            .await
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("transfer {id} has no recovery metadata"))?;
        if descriptor.profile_id() != session.profile().id {
            anyhow::bail!(
                "recovery profile mismatch for transfer {id}: expected {}, got {}",
                descriptor.profile_id(),
                session.profile().id
            );
        }
        self.retry
            .lock()
            .await
            .insert(id.to_string(), descriptor.with_session(session));
        Ok(())
    }

    // ---------- Queue scheduling ----------

    fn build_queue_state(&self, waiting: &VecDeque<String>) -> QueueState {
        QueueState {
            waiting: waiting.iter().cloned().collect(),
            paused_all: self.pause_all.is_paused(),
            concurrency: self.concurrency.load(Ordering::Relaxed),
            throttle_kbps: self.bucket.rate_kbps(),
        }
    }

    /// Current queue snapshot for the panel's initial load.
    pub async fn queue_state(&self) -> QueueState {
        let w = self.waiting.lock().await;
        self.build_queue_state(&w)
    }

    /// Emit `transfer://queue` and bump the generation so admission waiters
    /// re-check whether it is their turn.
    async fn bump_queue(&self, app: &AppHandle) {
        let state = {
            let w = self.waiting.lock().await;
            self.build_queue_state(&w)
        };
        self.queue_gen.send_modify(|g| *g += 1);
        let _ = app.emit("transfer://queue", &state);
    }

    /// Wait until this transfer is at the front of the FIFO with the pause-all
    /// gate open, take a concurrency permit, and pop it from the queue.
    /// Returns `None` if the id left the queue (canceled while waiting).
    /// The permit is returned so the caller holds it for the transfer's life.
    async fn admit(&self, id: &str) -> Option<OwnedSemaphorePermit> {
        let mut rx = self.queue_gen.subscribe();
        loop {
            {
                let w = self.waiting.lock().await;
                if !w.iter().any(|x| x == id) {
                    return None;
                }
                if !self.is_my_turn(&w, id).await {
                    drop(w);
                    if rx.changed().await.is_err() {
                        return None;
                    }
                    continue;
                }
            }
            // My turn: take a permit, then pop.
            let permit = self.semaphore.clone().acquire_owned().await.ok()?;
            let mut w = self.waiting.lock().await;
            if self.is_my_turn(&w, id).await {
                if let Some(pos) = w.iter().position(|x| x == id) {
                    w.remove(pos);
                }
                return Some(permit);
            }
            // Lost a race (pause engaged mid-acquire): release, re-evaluate.
            drop(permit);
            if !w.iter().any(|x| x == id) {
                return None;
            }
            drop(w);
        }
    }

    /// Is `id` the first waiting transfer allowed to run? Strict FIFO except
    /// that per-transfer-paused rows are skipped (a paused row must not
    /// head-of-line block the queue); pause-all blocks everyone. Caller must
    /// hold the `waiting` lock; lock order is waiting → pauses.
    async fn is_my_turn(&self, w: &VecDeque<String>, id: &str) -> bool {
        if self.pause_all.is_paused() {
            return false;
        }
        let pauses = self.pauses.lock().await;
        let first_open = w
            .iter()
            .position(|x| pauses.get(x).is_none_or(|g| !g.is_paused()));
        first_open == w.iter().position(|x| x == id)
    }

    /// Reorder a waiting transfer (active transfers are untouched).
    pub async fn move_in_queue(&self, id: &str, up: bool, app: &AppHandle) -> Result<()> {
        {
            let mut w = self.waiting.lock().await;
            let Some(pos) = w.iter().position(|x| x == id) else {
                anyhow::bail!("transfer {id} is not waiting in the queue");
            };
            let swap_with = if up {
                pos.checked_sub(1)
            } else if pos + 1 < w.len() {
                Some(pos + 1)
            } else {
                None
            };
            if let Some(other) = swap_with {
                w.swap(pos, other);
            }
        }
        self.bump_queue(app).await;
        Ok(())
    }

    /// Pause admission of new transfers (running ones keep going until Phase
    /// 2's chunk checkpoints let them park too).
    pub async fn pause_all(&self, app: &AppHandle) {
        self.pause_all.set(true);
        self.bump_queue(app).await;
    }

    pub async fn resume_all(&self, app: &AppHandle) {
        self.pause_all.set(false);
        self.bump_queue(app).await;
    }

    pub fn is_paused_all(&self) -> bool {
        self.pause_all.is_paused()
    }

    /// Chunk-boundary checkpoint shared by every copy loop. Draws
    /// `bytes` from the global bandwidth bucket, then — when the
    /// transfer (or the whole manager) is paused — parks until resumed and
    /// returns `RestartFromPause` so the runner re-runs the file from byte 0.
    async fn checkpoint(&self, id: &str, bytes: u64) -> Result<()> {
        self.bucket.acquire(bytes).await;
        let gate = self.pauses.lock().await.get(id).cloned();
        let parked = self.pause_all.is_paused() || gate.as_ref().is_some_and(|g| g.is_paused());
        if !parked {
            return Ok(());
        }
        // Park until BOTH gates are open (resume requires both).
        loop {
            self.pause_all.wait_open().await;
            if let Some(g) = &gate {
                g.wait_open().await;
            }
            if !self.pause_all.is_paused() && gate.as_ref().is_none_or(|g| !g.is_paused()) {
                break;
            }
        }
        Err(RestartFromPause.into())
    }

    /// Pause a queued or transferring transfer. A running one parks at the
    /// next chunk boundary; a queued one is skipped by admission until resumed.
    pub async fn pause(&self, id: &str, app: &AppHandle) -> Result<()> {
        match self.get(id).await.map(|t| t.status) {
            Some(TransferStatus::Transferring) | Some(TransferStatus::Queued) => {}
            _ => anyhow::bail!("transfer {id} is not running or queued"),
        }
        let gate = self
            .pauses
            .lock()
            .await
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("transfer {id} not found"))?;
        gate.set(true);
        self.update(id, |t| t.status = TransferStatus::Paused).await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://updated", &t);
        }
        Ok(())
    }

    /// Resume a paused transfer. Backends with byte-range support retain the
    /// committed offset; unsupported backends are reset by the runner before
    /// the next attempt. A queued transfer simply re-enters FIFO admission.
    pub async fn resume(&self, id: &str, app: &AppHandle) -> Result<()> {
        if self.get(id).await.map(|t| t.status) != Some(TransferStatus::Paused) {
            anyhow::bail!("transfer {id} is not paused");
        }
        let still_queued = self.waiting.lock().await.iter().any(|x| x == id);
        let gate = self
            .pauses
            .lock()
            .await
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("transfer {id} not found"))?;
        self.update(id, |t| {
            t.status = if still_queued {
                TransferStatus::Queued
            } else {
                TransferStatus::Transferring
            };
        })
        .await;
        gate.set(false);
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://updated", &t);
        }
        // Wake admission waiters: a queued row may have become runnable.
        self.bump_queue(app).await;
        Ok(())
    }

    /// Re-enqueue a failed or canceled transfer with its original (already
    /// policy-resolved) source/destination. Same id — the panel row resets
    /// in place.
    pub async fn retry(self: &Arc<Self>, id: &str, app: &AppHandle) -> Result<()> {
        match self.get(id).await.map(|t| t.status) {
            Some(TransferStatus::Error) | Some(TransferStatus::Canceled) => {}
            _ => anyhow::bail!("only failed or canceled transfers can be retried"),
        }
        let info = self
            .retry
            .lock()
            .await
            .get(id)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("transfer {id} cannot be retried"))?;
        if let Some(h) = self.tasks.lock().await.remove(id) {
            h.abort();
        }
        // A cancel-while-paused leaves the gate closed — reopen it. A row
        // restored from SQLite has no in-memory gate yet, so recreate one.
        {
            let mut pauses = self.pauses.lock().await;
            pauses
                .entry(id.to_string())
                .or_insert_with(PauseGate::new)
                .set(false);
        }
        // Never reuse the canceled Arc: a previous FTP spawn_blocking worker
        // may still own it while finishing ABOR cleanup. A fresh token prevents
        // a fast Cancel -> Retry from resurrecting that old worker.
        self.cancels
            .lock()
            .await
            .insert(id.to_string(), Arc::new(AtomicBool::new(false)));

        // A recovered byte count is not proof that the source object is still
        // the same object after a process restart. Until the ledger persists a
        // backend-specific immutable identity/change signal, always restart the
        // first recovered retry from zero rather than risk a hybrid file.
        self.recovered_retry.lock().await.remove(id);
        self.update(id, |t| {
            t.status = TransferStatus::Queued;
            t.transferred = 0;
            t.error = None;
            t.retry_attempt = None;
        })
        .await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://updated", &t);
        }
        self.waiting.lock().await.push_back(id.to_string());
        self.bump_queue(app).await;
        let mgr = Arc::clone(self);
        let id_for_task = id.to_string();
        let app_for_task = app.clone();
        match info {
            RetryInfo::Download {
                session,
                remote_path,
                final_path,
            } => {
                self.spawn_registered(
                    id.to_string(),
                    run_download_task(
                        mgr,
                        id_for_task,
                        session,
                        remote_path,
                        final_path,
                        app_for_task,
                    ),
                )
                .await;
            }
            RetryInfo::Upload {
                session,
                local,
                final_remote,
            } => {
                self.spawn_registered(
                    id.to_string(),
                    run_upload_task(mgr, id_for_task, session, local, final_remote, app_for_task),
                )
                .await;
            }
        }
        Ok(())
    }

    /// Live-adjust the global bandwidth cap (KiB/s, 0 = unlimited). Takes
    /// effect on the next chunk of every active transfer.
    pub fn set_throttle_kbps(&self, kbps: u64) {
        self.bucket.set_rate_kbps(kbps);
    }

    /// Live-adjust the concurrency bound. Growing adds permits at once;
    /// shrinking forgets permits as running transfers release them, so
    /// in-flight transfers are never killed to satisfy the new bound.
    pub fn set_concurrency(&self, n: usize) {
        let n = n.clamp(1, 32);
        let old = self.concurrency.swap(n, Ordering::Relaxed);
        if n > old {
            self.semaphore.add_permits(n - old);
        } else if n < old {
            let sem = Arc::clone(&self.semaphore);
            tokio::spawn(async move {
                if let Ok(p) = sem.acquire_many((old - n) as u32).await {
                    p.forget();
                }
            });
        }
    }

    /// Live-adjust the delta-sync switch (the `deltaSync` setting). Takes effect on the next transfer decision; `GHOSTFTP_DELTA=0`
    /// still force-disables regardless.
    pub fn set_max_auto_retries(&self, attempts: usize) {
        self.max_auto_retries
            .store(attempts.min(8), Ordering::Relaxed);
    }

    pub fn set_delta_enabled(&self, enabled: bool) {
        self.delta_enabled.store(enabled, Ordering::Relaxed);
    }

    /// Delta-sync master switch: the `deltaSync` setting (default on), with
    /// `GHOSTFTP_DELTA=0` as a force-off escape hatch. Read once per transfer
    /// decision.
    fn delta_enabled(&self) -> bool {
        std::env::var("GHOSTFTP_DELTA").ok().as_deref() != Some("0")
            && self.delta_enabled.load(Ordering::Relaxed)
    }

    pub async fn cancel(&self, id: &str, app: &AppHandle) -> Result<()> {
        // Set the cooperative flag before aborting the async wrapper. A
        // spawn_blocking FTP/FTPS data loop keeps running after JoinHandle::abort,
        // so it must observe this flag and issue ABOR itself.
        if let Some(flag) = self.cancels.lock().await.get(id) {
            flag.store(true, Ordering::Release);
        }
        {
            let mut w = self.waiting.lock().await;
            if let Some(pos) = w.iter().position(|x| x == id) {
                w.remove(pos);
            }
        }
        if let Some(h) = self.tasks.lock().await.remove(id) {
            h.abort();
        }
        self.update(id, |t| {
            if matches!(
                t.status,
                TransferStatus::Transferring | TransferStatus::Queued | TransferStatus::Paused
            ) {
                t.status = TransferStatus::Canceled;
            }
        })
        .await;
        // There is no `transfer://canceled` event — `updated` carries the row.
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://updated", &t);
        }
        self.bump_queue(app).await;
        Ok(())
    }

    pub async fn start_download(
        self: &Arc<Self>,
        session: Arc<Session>,
        remote_path: String,
        local_dir: String,
        policy: OverwritePolicy,
        app: AppHandle,
    ) -> Result<String> {
        let file_name = basename(&remote_path);
        if file_name.is_empty() || matches!(file_name.as_str(), "." | "..") {
            anyhow::bail!("download path does not identify a file: {remote_path}");
        }
        let local_root = PathBuf::from(&local_dir);
        tokio::fs::create_dir_all(&local_root)
            .await
            .with_context(|| format!("create download directory {}", local_root.display()))?;
        let size = remote_size(&session, &remote_path).await.unwrap_or(0);

        let initial = local_root.join(file_name);
        let (final_path, skip) = match policy {
            OverwritePolicy::Overwrite => (initial, false),
            OverwritePolicy::Skip => {
                let exists = initial.exists();
                (initial, exists)
            }
            OverwritePolicy::Rename => (resolve_local_rename(&initial)?, false),
        };

        let id = Uuid::new_v4().to_string();
        let transfer = Transfer {
            id: id.clone(),
            kind: TransferKind::Download,
            source: remote_path.clone(),
            destination: final_path.to_string_lossy().into_owned(),
            size,
            transferred: 0,
            status: if skip {
                TransferStatus::Skipped
            } else {
                TransferStatus::Queued
            },
            error: None,
            retry_attempt: None,
            delta: None,
            started_at: now_ts(),
        };
        self.persisted_retry.lock().await.insert(
            id.clone(),
            PersistedRetryInfo::Download {
                profile_id: session.profile().id.clone(),
                remote_path: remote_path.clone(),
                final_path: final_path.clone(),
            },
        );
        self.insert(transfer.clone()).await;
        let _ = app.emit("transfer://added", &transfer);

        if skip {
            let _ = app.emit("transfer://done", &transfer);
            return Ok(id);
        }

        self.retry.lock().await.insert(
            id.clone(),
            RetryInfo::Download {
                session: Arc::clone(&session),
                remote_path: remote_path.clone(),
                final_path: final_path.clone(),
            },
        );
        self.waiting.lock().await.push_back(id.clone());
        self.pauses
            .lock()
            .await
            .insert(id.clone(), PauseGate::new());
        self.cancels
            .lock()
            .await
            .insert(id.clone(), Arc::new(AtomicBool::new(false)));
        self.bump_queue(&app).await;

        let mgr = Arc::clone(self);
        let id_for_task = id.clone();
        self.spawn_registered(
            id.clone(),
            run_download_task(mgr, id_for_task, session, remote_path, final_path, app),
        )
        .await;
        Ok(id)
    }

    /// Stream a download from a Ghost FTP Agent daemon by ranged `ReadChunk`s. The
    /// daemon caps each chunk; we advance the offset until it reports EOF.
    /// `app` is optional so tests can drive the loop without an `AppHandle`
    /// (a test binary that links tauri's mock runtime pulls an unmanifested
    /// `TaskDialogIndirect` import and fails to launch on Windows); `None`
    /// just skips the progress events.
    async fn agent_download_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        remote_path: &str,
        local_path: &Path,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        use base64::Engine as _;
        use ghostftp_agent_proto::msg::{Request, Response};
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path).await.ok().map(|m| m.len());
        let mut offset =
            if requested_offset > 0 && local_len.is_some_and(|len| len >= requested_offset) {
                requested_offset
            } else {
                0
            };
        if offset != requested_offset {
            self.update(id, |t| t.transferred = offset).await;
        }
        let mut local_file = tokio::fs::OpenOptions::new()
            .create(true)
            .write(true)
            .truncate(offset == 0)
            .open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        if offset > 0 {
            local_file.set_len(offset).await?;
            local_file.seek(SeekFrom::Start(offset)).await?;
        }
        let mut last_emit = Instant::now();
        loop {
            let resp = session
                .request(Request::ReadChunk {
                    path: remote_path.to_string(),
                    offset,
                    len: 0,
                })
                .await?;
            let (data_b64, eof) = match resp {
                Response::Chunk { data, eof } => (data, eof),
                Response::Error { message, .. } => {
                    return Err(anyhow::anyhow!("download {remote_path}: {message}"))
                }
                other => {
                    return Err(anyhow::anyhow!(
                        "download {remote_path}: unexpected {other:?}"
                    ))
                }
            };
            let bytes = base64::engine::general_purpose::STANDARD
                .decode(&data_b64)
                .context("decode chunk")?;
            self.checkpoint(id, bytes.len() as u64).await?;
            if !bytes.is_empty() {
                local_file.write_all(&bytes).await?;
                offset += bytes.len() as u64;
                self.update(id, |t| t.transferred = offset).await;
                if last_emit.elapsed() > Duration::from_millis(100) {
                    if let Some(app) = app {
                        if let Some(t) = self.get(id).await {
                            let _ = app.emit("transfer://progress", &t);
                        }
                    }
                    last_emit = Instant::now();
                }
            }
            if eof {
                break;
            }
        }
        local_file.flush().await?;
        self.update(id, |t| t.transferred = offset).await;
        Ok(())
    }

    async fn run_ssh_download(
        &self,
        id: &str,
        session: Arc<SshSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path).await.ok().map(|m| m.len());
        let resume_offset =
            if requested_offset > 0 && local_len.is_some_and(|len| len >= requested_offset) {
                requested_offset
            } else {
                0
            };
        if resume_offset != requested_offset {
            self.update(id, |t| t.transferred = resume_offset).await;
        }

        let sftp_cell = session.ensure_sftp().await?;
        let mut remote_file = {
            let sftp = sftp_cell.lock().await;
            sftp.open(remote_path)
                .await
                .with_context(|| format!("open remote {remote_path}"))?
        };
        if resume_offset > 0 {
            remote_file.seek(SeekFrom::Start(resume_offset)).await?;
        }

        let mut local_file = tokio::fs::OpenOptions::new()
            .create(true)
            .write(true)
            .truncate(resume_offset == 0)
            .open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        if resume_offset > 0 {
            local_file.set_len(resume_offset).await?;
            local_file.seek(SeekFrom::Start(resume_offset)).await?;
        }

        let mut buf = vec![0u8; 64 * 1024];
        let mut transferred: u64 = resume_offset;
        let mut last_emit = Instant::now();
        loop {
            let n = remote_file.read(&mut buf).await?;
            if n == 0 {
                break;
            }
            self.checkpoint(id, n as u64).await?;
            local_file.write_all(&buf[..n]).await?;
            transferred += n as u64;
            self.update(id, |t| t.transferred = transferred).await;
            if last_emit.elapsed() > Duration::from_millis(100) {
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        local_file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    pub async fn start_upload(
        self: &Arc<Self>,
        session: Arc<Session>,
        local_path: String,
        remote_dir: String,
        policy: OverwritePolicy,
        app: AppHandle,
    ) -> Result<String> {
        let local = PathBuf::from(&local_path);
        let metadata = tokio::fs::metadata(&local)
            .await
            .with_context(|| format!("stat {}", local.display()))?;
        if !metadata.is_file() {
            anyhow::bail!("upload source is not a regular file: {}", local.display());
        }
        let file_name = basename(&local_path);
        if file_name.is_empty() || matches!(file_name.as_str(), "." | "..") {
            anyhow::bail!("upload path does not identify a file: {}", local.display());
        }
        let size = metadata.len();

        // Join with the separator the destination already uses, and don't add a
        // second one. A Windows agent target spelled `C:\srv\` used to produce
        // `C:\srv\/file` — Win32 tolerates it, but it shows up in every error
        // message and audit line, and the mixed form trips path comparisons.
        let initial_remote = join_remote(&remote_dir, &file_name);

        let (final_remote, skip) = remote_resolve(&session, &initial_remote, policy).await?;

        let id = Uuid::new_v4().to_string();
        let transfer = Transfer {
            id: id.clone(),
            kind: TransferKind::Upload,
            source: local_path.clone(),
            destination: final_remote.clone(),
            size,
            transferred: 0,
            status: if skip {
                TransferStatus::Skipped
            } else {
                TransferStatus::Queued
            },
            error: None,
            retry_attempt: None,
            delta: None,
            started_at: now_ts(),
        };
        self.persisted_retry.lock().await.insert(
            id.clone(),
            PersistedRetryInfo::Upload {
                profile_id: session.profile().id.clone(),
                local: local.clone(),
                final_remote: final_remote.clone(),
            },
        );
        self.insert(transfer.clone()).await;
        let _ = app.emit("transfer://added", &transfer);

        if skip {
            let _ = app.emit("transfer://done", &transfer);
            return Ok(id);
        }

        self.retry.lock().await.insert(
            id.clone(),
            RetryInfo::Upload {
                session: Arc::clone(&session),
                local: local.clone(),
                final_remote: final_remote.clone(),
            },
        );
        self.waiting.lock().await.push_back(id.clone());
        self.pauses
            .lock()
            .await
            .insert(id.clone(), PauseGate::new());
        self.cancels
            .lock()
            .await
            .insert(id.clone(), Arc::new(AtomicBool::new(false)));
        self.bump_queue(&app).await;

        let mgr = Arc::clone(self);
        let id_for_task = id.clone();
        self.spawn_registered(
            id.clone(),
            run_upload_task(mgr, id_for_task, session, local, final_remote, app),
        )
        .await;
        Ok(id)
    }

    /// Stream an upload to a Ghost FTP Agent daemon by ranged `WriteChunk`s. The first
    /// chunk truncates/creates the file; subsequent chunks append at the offset.
    /// `app: None` (tests) skips the progress events — see
    /// [`Self::agent_download_core`].
    async fn agent_upload_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        local_path: &Path,
        remote_path: &str,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        use base64::Engine as _;
        use ghostftp_agent_proto::msg::{Request, Response};
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let remote_len = if requested_offset > 0 {
            agent_stat(session, remote_path).await.0
        } else {
            0
        };
        let mut offset = if requested_offset > 0
            && local_len >= requested_offset
            && remote_len == requested_offset
        {
            requested_offset
        } else {
            0
        };
        if offset != requested_offset {
            self.update(id, |t| t.transferred = offset).await;
        }

        let mut local_file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        if offset > 0 {
            local_file.seek(SeekFrom::Start(offset)).await?;
        }
        // 128 KiB plaintext keeps each request comfortably under the daemon's
        // per-chunk cap while amortising the round-trip.
        let mut buf = vec![0u8; 128 * 1024];
        let mut first = offset == 0;
        let mut last_emit = Instant::now();
        loop {
            let n = local_file.read(&mut buf).await?;
            if n == 0 {
                break;
            }
            self.checkpoint(id, n as u64).await?;
            let data = base64::engine::general_purpose::STANDARD.encode(&buf[..n]);
            let resp = session
                .request(Request::WriteChunk {
                    path: remote_path.to_string(),
                    offset,
                    data,
                    truncate: first,
                    done: false,
                })
                .await?;
            match resp {
                Response::Written { .. } => {}
                Response::Error { message, .. } => {
                    return Err(anyhow::anyhow!("upload {remote_path}: {message}"))
                }
                other => {
                    return Err(anyhow::anyhow!(
                        "upload {remote_path}: unexpected {other:?}"
                    ))
                }
            }
            offset += n as u64;
            first = false;
            self.update(id, |t| t.transferred = offset).await;
            if last_emit.elapsed() > Duration::from_millis(100) {
                if let Some(app) = app {
                    if let Some(t) = self.get(id).await {
                        let _ = app.emit("transfer://progress", &t);
                    }
                }
                last_emit = Instant::now();
            }
        }
        self.update(id, |t| t.transferred = offset).await;
        Ok(())
    }

    /// Agent upload entry point (delta sync): attempt a block-level
    /// delta when the switch is on, a remote basis exists, and the file is big
    /// enough — ANY delta error logs and falls back to the whole-file upload.
    async fn run_agent_upload_with_delta(
        &self,
        id: &str,
        session: Arc<crate::session::AgentSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.agent_upload_with_delta_core(id, &session, local_path, remote_path, Some(app))
            .await
    }

    /// Gate + fallback logic behind [`Self::run_agent_upload_with_delta`];
    /// `app: None` (tests) skips progress events.
    async fn agent_upload_with_delta_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        local_path: &Path,
        remote_path: &str,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        // Resume an interrupted whole-file transfer from its committed prefix.
        // Delta mode compares complete old/new files and must not reinterpret
        // a partial destination as a valid delta basis.
        if self.get(id).await.is_some_and(|t| t.transferred > 0) {
            return self
                .agent_upload_core(id, session, local_path, remote_path, app)
                .await;
        }

        let size = tokio::fs::metadata(local_path)
            .await
            .map(|m| m.len())
            .unwrap_or(0);
        let (_basis_size, basis_exists) = agent_stat(session, remote_path).await;
        if self.delta_enabled()
            && ghostftp_agent_proto::delta::should_attempt_delta(size, basis_exists, true)
        {
            match self
                .agent_delta_upload_core(id, session, local_path, remote_path, app)
                .await
            {
                Ok(()) => return Ok(()),
                Err(e) => {
                    tracing::warn!("delta upload fell back to whole-file copy: {e:#}");
                    self.update(id, |t| t.transferred = 0).await;
                }
            }
        }
        self.agent_upload_core(id, session, local_path, remote_path, app)
            .await
    }

    /// Upload to a Ghost FTP Agent daemon as a block-level delta: fetch the remote
    /// (old) file's chunk signature, plan locally, upload only the unmatched
    /// bytes as a patch temp via ordinary `WriteChunk`s, then ask the daemon to
    /// reassemble + hash-verify + atomically rename over the destination. Any
    /// error leaves the destination untouched; the caller falls back to a
    /// whole-file upload.
    async fn agent_delta_upload_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        local_path: &Path,
        remote_path: &str,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        use base64::Engine as _;
        use ghostftp_agent_proto::delta;
        use ghostftp_agent_proto::msg::{Request, Response};
        let started = Instant::now();

        // 1. The remote (old) file's chunk signature. A pre-delta daemon fails
        // this request, which sends the caller down the whole-file path.
        let remote_sig = match session
            .request(Request::Signature {
                path: remote_path.to_string(),
            })
            .await?
        {
            Response::Signature {
                size,
                min,
                avg,
                max,
                chunks,
                whole_hash,
            } => delta::FileSignature {
                size,
                min,
                avg,
                max,
                chunks,
                whole_hash,
            },
            Response::Error { message, .. } => {
                anyhow::bail!("signature {remote_path}: {message}")
            }
            other => anyhow::bail!("signature {remote_path}: unexpected {other:?}"),
        };
        anyhow::ensure!(
            delta::params_match(&remote_sig),
            "daemon chunk params differ from ours — no delta"
        );

        // 2. Plan the delta locally (CPU-bound → blocking thread); the patch of
        // literal bytes lands in a temp next to the source file.
        let local_dir = local_path.parent().unwrap_or(Path::new("."));
        let local_patch = local_dir.join(format!(".ghostftp-patch-{}", Uuid::new_v4()));
        let (local_owned, patch_owned) = (local_path.to_path_buf(), local_patch.clone());
        let plan = match tokio::task::spawn_blocking(move || {
            let patch = std::fs::File::create(&patch_owned)
                .with_context(|| format!("create {}", patch_owned.display()))?;
            delta::plan_delta(&remote_sig, &local_owned, patch)
        })
        .await
        {
            Ok(Ok(plan)) => plan,
            Ok(Err(e)) => {
                let _ = tokio::fs::remove_file(&local_patch).await;
                return Err(e.context("plan delta"));
            }
            Err(e) => {
                let _ = tokio::fs::remove_file(&local_patch).await;
                return Err(anyhow::anyhow!("plan delta task: {e}"));
            }
        };
        let size = plan.literal_bytes + plan.reused_bytes;
        if !delta::delta_worthwhile(&plan, size) {
            let _ = tokio::fs::remove_file(&local_patch).await;
            anyhow::bail!(
                "delta barely saves anything ({} of {size} bytes would still cross the wire)",
                plan.literal_bytes
            );
        }

        // 3.+4. Upload the patch (same WriteChunk style as a whole-file upload),
        // then ask the daemon to assemble. Any failure → best-effort remote
        // delete of the patch, local temp cleanup, and the caller falls back.
        let remote_patch = format!("{remote_path}.ghostftp-patch-{}", Uuid::new_v4());
        let result: Result<()> = async {
            let mut patch_file = tokio::fs::File::open(&local_patch)
                .await
                .with_context(|| format!("open {}", local_patch.display()))?;
            let mut buf = vec![0u8; 128 * 1024];
            let mut offset: u64 = 0;
            let mut first = true;
            let mut last_emit = Instant::now();
            loop {
                let n = patch_file.read(&mut buf).await?;
                if n == 0 {
                    break;
                }
                self.checkpoint(id, n as u64).await?;
                let data = base64::engine::general_purpose::STANDARD.encode(&buf[..n]);
                let resp = session
                    .request(Request::WriteChunk {
                        path: remote_patch.clone(),
                        offset,
                        data,
                        truncate: first,
                        done: false,
                    })
                    .await?;
                match resp {
                    Response::Written { .. } => {}
                    Response::Error { message, .. } => {
                        anyhow::bail!("upload patch {remote_patch}: {message}")
                    }
                    other => anyhow::bail!("upload patch {remote_patch}: unexpected {other:?}"),
                }
                offset += n as u64;
                first = false;
                if last_emit.elapsed() > Duration::from_millis(100) {
                    self.update(id, |t| t.transferred = offset).await;
                    if let Some(app) = app {
                        if let Some(t) = self.get(id).await {
                            let _ = app.emit("transfer://progress", &t);
                        }
                    }
                    last_emit = Instant::now();
                }
            }
            // Charge the reused bytes too so pause gates and the throttle
            // bucket see the same totals a whole-file copy would.
            self.checkpoint(id, plan.reused_bytes).await?;

            let resp = session
                .request(Request::DeltaAssemble {
                    basis: Some(remote_path.to_string()),
                    patch: remote_patch.clone(),
                    recipe: plan.recipe.clone(),
                    dest: remote_path.to_string(),
                    expected_hash: plan.whole_hash.clone(),
                })
                .await?;
            match resp {
                Response::DeltaDone { .. } => Ok(()),
                Response::Error { message, .. } => {
                    anyhow::bail!("delta assemble {remote_path}: {message}")
                }
                other => anyhow::bail!("delta assemble {remote_path}: unexpected {other:?}"),
            }
        }
        .await;
        if result.is_err() {
            let _ = session
                .request(Request::Delete {
                    path: remote_patch.clone(),
                    recursive: false,
                })
                .await;
        }
        let _ = tokio::fs::remove_file(&local_patch).await;
        result?;

        tracing::info!(
            "delta upload {remote_path}: {} bytes, sent {} literal, reused {} in {:?}",
            size,
            plan.literal_bytes,
            plan.reused_bytes,
            started.elapsed()
        );
        self.update(id, |t| {
            t.transferred = t.size;
            t.delta = Some(DeltaStats {
                sent: plan.literal_bytes,
                reused: plan.reused_bytes,
            });
        })
        .await;
        if let Some(app) = app {
            if let Some(t) = self.get(id).await {
                let _ = app.emit("transfer://progress", &t);
            }
        }
        Ok(())
    }

    async fn run_agent_full_download_staged(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        remote_path: &str,
        local_path: &Path,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        let staging = download_staging_path(local_path, id);
        let result = self
            .agent_download_core(id, session, remote_path, &staging, app)
            .await;
        match result {
            Ok(()) => promote_staged_download(&staging, local_path, id).await,
            Err(error) => {
                if error.downcast_ref::<RestartFromPause>().is_none() {
                    let _ = tokio::fs::remove_file(&staging).await;
                }
                Err(error)
            }
        }
    }

    /// Agent download entry point (delta sync): mirror of
    /// [`Self::run_agent_upload_with_delta`].
    async fn run_agent_download_with_delta(
        &self,
        id: &str,
        session: Arc<crate::session::AgentSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.agent_download_with_delta_core(id, &session, remote_path, local_path, Some(app))
            .await
    }

    /// Gate + fallback logic behind [`Self::run_agent_download_with_delta`];
    /// `app: None` (tests) skips progress events.
    async fn agent_download_with_delta_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        remote_path: &str,
        local_path: &Path,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        // Continue an interrupted whole-file download from the verified local
        // prefix. Delta mode is reserved for complete prior-file bases.
        if self.get(id).await.is_some_and(|t| t.transferred > 0) {
            return self
                .run_agent_full_download_staged(id, session, remote_path, local_path, app)
                .await;
        }

        let (size, remote_exists) = agent_stat(session, remote_path).await;
        let basis_exists = tokio::fs::metadata(local_path).await.is_ok();
        if remote_exists
            && self.delta_enabled()
            && ghostftp_agent_proto::delta::should_attempt_delta(size, basis_exists, true)
        {
            match self
                .agent_delta_download_core(id, session, remote_path, local_path, app)
                .await
            {
                Ok(()) => return Ok(()),
                Err(e) => {
                    tracing::warn!("delta download fell back to whole-file copy: {e:#}");
                    self.update(id, |t| t.transferred = 0).await;
                }
            }
        }
        self.run_agent_full_download_staged(id, session, remote_path, local_path, app)
            .await
    }

    /// Download from a Ghost FTP Agent daemon as a block-level delta: fetch the
    /// remote (new) file's signature, chunk the local basis locally, download
    /// only the unmatched ranges via ordinary `ReadChunk`s, reassemble into a
    /// same-directory temp, hash-verify, and rename over the destination. Any
    /// error leaves the old local file untouched; the caller falls back to a
    /// whole-file download.
    async fn agent_delta_download_core(
        &self,
        id: &str,
        session: &Arc<crate::session::AgentSession>,
        remote_path: &str,
        local_path: &Path,
        app: Option<&AppHandle>,
    ) -> Result<()> {
        use base64::Engine as _;
        use ghostftp_agent_proto::delta;
        use ghostftp_agent_proto::msg::{Request, Response};
        let started = Instant::now();

        // 1. The remote (new) file's chunk signature.
        let target_sig = match session
            .request(Request::Signature {
                path: remote_path.to_string(),
            })
            .await?
        {
            Response::Signature {
                size,
                min,
                avg,
                max,
                chunks,
                whole_hash,
            } => delta::FileSignature {
                size,
                min,
                avg,
                max,
                chunks,
                whole_hash,
            },
            Response::Error { message, .. } => {
                anyhow::bail!("signature {remote_path}: {message}")
            }
            other => anyhow::bail!("signature {remote_path}: unexpected {other:?}"),
        };
        anyhow::ensure!(
            delta::params_match(&target_sig),
            "daemon chunk params differ from ours — no delta"
        );

        // 2. Chunk the local basis (the old file). A missing basis is fine —
        // an empty signature makes every remote chunk a literal.
        let mut has_basis = false;
        let basis_sig = match tokio::fs::metadata(local_path).await {
            Ok(_) => {
                let basis_owned = local_path.to_path_buf();
                match tokio::task::spawn_blocking(move || delta::signature_of_file(&basis_owned))
                    .await
                {
                    Ok(Ok(sig)) => {
                        has_basis = true;
                        sig
                    }
                    Ok(Err(e)) => return Err(e.context("signature of local basis")),
                    Err(e) => return Err(anyhow::anyhow!("basis signature task: {e}")),
                }
            }
            Err(_) => delta::FileSignature {
                size: 0,
                min: delta::CHUNK_MIN,
                avg: delta::CHUNK_AVG,
                max: delta::CHUNK_MAX,
                chunks: Vec::new(),
                whole_hash: String::new(),
            },
        };

        // 3. Match the remote chunks against the local basis.
        let (plan, needed) = delta::plan_download(&basis_sig, &target_sig)?;
        let size = target_sig.size;
        if !delta::delta_worthwhile(&plan, size) {
            anyhow::bail!(
                "delta barely saves anything ({} of {size} bytes would still cross the wire)",
                plan.literal_bytes
            );
        }

        // 4. Fetch only the missing ranges into a local patch temp, then 5.
        // reassemble locally and rename over the destination. Any failure →
        // delete both temps; the old local file is untouched.
        let local_dir = local_path.parent().unwrap_or(Path::new("."));
        let patch_path = local_dir.join(format!(".ghostftp-patch-{}", Uuid::new_v4()));
        let out_path = local_dir.join(format!(".ghostftp-delta-{}.tmp", Uuid::new_v4()));
        let result: Result<()> = async {
            let mut patch_file = tokio::fs::File::create(&patch_path)
                .await
                .with_context(|| format!("create {}", patch_path.display()))?;
            let mut fetched: u64 = 0;
            let mut last_emit = Instant::now();
            for &(range_off, range_len) in &needed {
                let mut got: u64 = 0;
                while got < range_len {
                    let resp = session
                        .request(Request::ReadChunk {
                            path: remote_path.to_string(),
                            offset: range_off + got,
                            len: range_len - got,
                        })
                        .await?;
                    let data_b64 = match resp {
                        Response::Chunk { data, .. } => data,
                        Response::Error { message, .. } => {
                            anyhow::bail!("download {remote_path}: {message}")
                        }
                        other => {
                            anyhow::bail!("download {remote_path}: unexpected {other:?}")
                        }
                    };
                    let bytes = base64::engine::general_purpose::STANDARD
                        .decode(&data_b64)
                        .context("decode chunk")?;
                    if bytes.is_empty() {
                        anyhow::bail!("download {remote_path}: short read (file changed?)");
                    }
                    self.checkpoint(id, bytes.len() as u64).await?;
                    patch_file.write_all(&bytes).await?;
                    got += bytes.len() as u64;
                    fetched += bytes.len() as u64;
                    if last_emit.elapsed() > Duration::from_millis(100) {
                        self.update(id, |t| t.transferred = fetched).await;
                        if let Some(app) = app {
                            if let Some(t) = self.get(id).await {
                                let _ = app.emit("transfer://progress", &t);
                            }
                        }
                        last_emit = Instant::now();
                    }
                }
            }
            patch_file.flush().await?;
            drop(patch_file);
            // Charge the reused bytes too (same accounting as a full copy).
            self.checkpoint(id, plan.reused_bytes).await?;

            // 5. Reassemble + hash-verify into the temp, then atomically
            // rename over the destination.
            let (basis_owned, patch_owned, out_owned, expected) = (
                has_basis.then(|| local_path.to_path_buf()),
                patch_path.clone(),
                out_path.clone(),
                plan.whole_hash.clone(),
            );
            let recipe = plan.recipe.clone();
            tokio::task::spawn_blocking(move || {
                delta::apply_delta(
                    basis_owned.as_deref(),
                    &patch_owned,
                    &recipe,
                    &out_owned,
                    &expected,
                )
            })
            .await
            .map_err(|e| anyhow::anyhow!("apply delta task: {e}"))?
            .context("apply delta")?;
            tokio::fs::rename(&out_path, local_path)
                .await
                .with_context(|| format!("rename delta result over {}", local_path.display()))?;
            Ok(())
        }
        .await;
        let _ = tokio::fs::remove_file(&patch_path).await;
        if result.is_err() {
            let _ = tokio::fs::remove_file(&out_path).await;
        }
        result?;

        tracing::info!(
            "delta download {remote_path}: {} bytes, fetched {} literal, reused {} in {:?}",
            size,
            plan.literal_bytes,
            plan.reused_bytes,
            started.elapsed()
        );
        self.update(id, |t| {
            t.transferred = t.size;
            t.delta = Some(DeltaStats {
                sent: plan.literal_bytes,
                reused: plan.reused_bytes,
            });
        })
        .await;
        if let Some(app) = app {
            if let Some(t) = self.get(id).await {
                let _ = app.emit("transfer://progress", &t);
            }
        }
        Ok(())
    }

    async fn run_ssh_upload(
        &self,
        id: &str,
        session: Arc<SshSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();

        let sftp_cell = session.ensure_sftp().await?;
        let mut resume_offset = requested_offset.min(local_len);
        let mut remote_file = {
            let sftp = sftp_cell.lock().await;
            let remote_len = if resume_offset > 0 {
                sftp.metadata(remote_path)
                    .await
                    .ok()
                    .and_then(|metadata| metadata.size)
                    .unwrap_or(0)
            } else {
                0
            };
            if resume_offset > 0 && remote_len == resume_offset {
                sftp.open_with_flags(remote_path, russh_sftp::protocol::OpenFlags::WRITE)
                    .await
                    .with_context(|| format!("open remote {remote_path} for resume"))?
            } else {
                resume_offset = 0;
                sftp.create(remote_path)
                    .await
                    .with_context(|| format!("create remote {remote_path}"))?
            }
        };
        if resume_offset != requested_offset {
            self.update(id, |t| t.transferred = resume_offset).await;
        }
        if resume_offset > 0 {
            remote_file.seek(SeekFrom::Start(resume_offset)).await?;
        }

        let mut local_file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        if resume_offset > 0 {
            local_file.seek(SeekFrom::Start(resume_offset)).await?;
        }

        let mut buf = vec![0u8; 64 * 1024];
        let mut transferred: u64 = resume_offset;
        let mut last_emit = Instant::now();
        loop {
            let n = local_file.read(&mut buf).await?;
            if n == 0 {
                break;
            }
            self.checkpoint(id, n as u64).await?;
            remote_file.write_all(&buf[..n]).await?;
            transferred += n as u64;
            self.update(id, |t| t.transferred = transferred).await;
            if last_emit.elapsed() > Duration::from_millis(100) {
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        remote_file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }
}

impl TransferManager {
    /// Walk a remote directory tree and queue a transfer per file. Uses the
    /// RemoteFs trait so it works for both SFTP and FTP.
    pub async fn start_directory_download(
        self: &Arc<Self>,
        session: Arc<Session>,
        remote_root: String,
        local_dir: String,
        policy: OverwritePolicy,
        app: AppHandle,
    ) -> Result<Vec<String>> {
        let root_name = basename(&remote_root);
        if root_name.is_empty() || matches!(root_name.as_str(), "." | "..") {
            anyhow::bail!("directory download requires a named remote directory");
        }
        let local_root = PathBuf::from(&local_dir).join(&root_name);
        tokio::fs::create_dir_all(&local_root)
            .await
            .with_context(|| format!("mkdir -p {}", local_root.display()))?;

        let fs = fs_for_session(&session);

        let mut dirs_to_visit: Vec<String> = vec![remote_root.clone()];
        let mut files: Vec<(String, PathBuf)> = Vec::new();

        while let Some(d) = dirs_to_visit.pop() {
            let entries = fs
                .list_dir(&d)
                .await
                .with_context(|| format!("read_dir {d}"))?;
            for entry in entries {
                let remote_child = entry.path.clone();
                let rel = remote_child
                    .strip_prefix(&remote_root)
                    .unwrap_or(&remote_child)
                    .trim_start_matches('/');
                let local_child = local_root.join(rel);
                match entry.kind {
                    crate::remotefs::FileKind::Directory => {
                        tokio::fs::create_dir_all(&local_child).await.ok();
                        dirs_to_visit.push(remote_child);
                    }
                    crate::remotefs::FileKind::File => {
                        files.push((remote_child, local_child));
                    }
                    _ => {}
                }
            }
        }

        let mut ids = Vec::with_capacity(files.len());
        for (remote_path, local_path) in files {
            let parent_dir = local_path
                .parent()
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_else(|| ".".into());
            let id = self
                .start_download(
                    Arc::clone(&session),
                    remote_path,
                    parent_dir,
                    policy,
                    app.clone(),
                )
                .await?;
            ids.push(id);
        }
        Ok(ids)
    }

    pub async fn start_directory_upload(
        self: &Arc<Self>,
        session: Arc<Session>,
        local_root: String,
        remote_dir: String,
        policy: OverwritePolicy,
        app: AppHandle,
    ) -> Result<Vec<String>> {
        let local_root_path = PathBuf::from(&local_root);
        let metadata = tokio::fs::metadata(&local_root_path)
            .await
            .with_context(|| format!("stat {}", local_root_path.display()))?;
        if !metadata.is_dir() {
            anyhow::bail!(
                "directory upload source is not a directory: {}",
                local_root_path.display()
            );
        }
        let root_name = local_root_path
            .file_name()
            .map(|n| n.to_string_lossy().into_owned())
            .filter(|name| !name.is_empty() && !matches!(name.as_str(), "." | ".."))
            .ok_or_else(|| anyhow::anyhow!("directory upload requires a named local directory"))?;
        let remote_root = join_remote(&remote_dir, &root_name);

        let fs = fs_for_session(&session);

        // The selected upload root may already exist. Directory creation is
        // therefore best-effort here; file creation below still surfaces a
        // concrete permission/path error if the destination is unusable.
        let _ = fs.create_dir(&remote_root).await;

        let mut dirs_to_visit: Vec<PathBuf> = vec![local_root_path.clone()];
        let mut files: Vec<(PathBuf, String)> = Vec::new();
        let mut subdirs: Vec<String> = Vec::new();
        while let Some(d) = dirs_to_visit.pop() {
            let mut rd = tokio::fs::read_dir(&d)
                .await
                .with_context(|| format!("read_dir {}", d.display()))?;
            while let Some(entry) = rd.next_entry().await? {
                let p = entry.path();
                let meta = require_directory_upload_metadata(&p, entry.metadata().await)?;
                let rel = p
                    .strip_prefix(&local_root_path)
                    .unwrap_or(&p)
                    .to_string_lossy()
                    .replace('\\', "/");
                let remote_child = format!("{remote_root}/{rel}");
                if meta.is_dir() {
                    subdirs.push(remote_child);
                    dirs_to_visit.push(p);
                } else if meta.is_file() {
                    files.push((p, remote_child));
                }
            }
        }

        subdirs.sort_by_key(|s| s.matches('/').count());
        for sd in subdirs {
            // Preserve compatibility with FTP/SFTP servers that report
            // "already exists" as an error. Missing directories will still be
            // diagnosed precisely when the child transfer is started.
            let _ = fs.create_dir(&sd).await;
        }

        let mut ids = Vec::with_capacity(files.len());
        for (local_path, remote_path) in files {
            let parent = remote_path
                .rsplit_once('/')
                .map(|(p, _)| p.to_string())
                .unwrap_or_else(|| remote_root.clone());
            let id = self
                .start_upload(
                    Arc::clone(&session),
                    local_path.to_string_lossy().into_owned(),
                    parent,
                    policy,
                    app.clone(),
                )
                .await?;
            ids.push(id);
        }
        Ok(ids)
    }

    async fn run_ftp_download(
        &self,
        id: &str,
        session: Arc<FtpSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        self.checkpoint(id, 0).await?;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path).await.ok().map(|m| m.len());
        let candidate_offset =
            if requested_offset > 0 && local_len.is_some_and(|len| len >= requested_offset) {
                requested_offset
            } else {
                0
            };
        let pause = self.pauses.lock().await.get(id).cloned();
        let pause_all = self.pause_all.clone();
        let canceled = self
            .cancels
            .lock()
            .await
            .get(id)
            .cloned()
            .unwrap_or_else(|| Arc::new(AtomicBool::new(false)));
        let final_path = local_path.to_path_buf();
        let path = remote_path.to_string();

        let outcome = match session
            .with_stream(move |stream| {
                let remote_len = stream.size(&path).ok().map(|size| size as u64);
                let offset = if candidate_offset > 0
                    && remote_len.is_some_and(|len| len >= candidate_offset)
                {
                    candidate_offset
                } else {
                    0
                };

                let mut file = std::fs::OpenOptions::new()
                    .create(true)
                    .write(true)
                    .truncate(offset == 0)
                    .open(&final_path)
                    .with_context(|| format!("open {}", final_path.display()))?;
                if offset > 0 {
                    file.set_len(offset)?;
                    file.seek(SeekFrom::Start(offset))?;
                }
                let mut writer = std::io::BufWriter::new(file);
                let outcome = stream.retr_resumable(&path, offset, &mut writer, |_| {
                    if canceled.load(Ordering::Acquire) {
                        FtpTransferControl::Cancel
                    } else if pause_all.is_paused()
                        || pause.as_ref().is_some_and(|gate| gate.is_paused())
                    {
                        FtpTransferControl::Pause
                    } else {
                        FtpTransferControl::Continue
                    }
                })?;
                writer.flush().context("flush FTP download destination")?;
                Ok(outcome)
            })
            .await
        {
            Ok(outcome) => outcome,
            Err(error) => {
                return Err(match session.reconnect().await {
                    Ok(()) => error,
                    Err(reconnect_error) => anyhow::anyhow!(
                        "{error:#}; additionally failed to reconnect FTP session after download failure: {reconnect_error:#}"
                    ),
                });
            }
        };

        self.update(id, |t| t.transferred = outcome.transferred)
            .await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://progress", &t);
        }

        match outcome.control {
            FtpTransferControl::Continue => Ok(()),
            FtpTransferControl::Pause => {
                self.checkpoint(id, 0).await?;
                session
                    .reconnect()
                    .await
                    .context("reconnect FTP session after paused download")?;
                Err(RestartFromPause.into())
            }
            FtpTransferControl::Cancel => {
                session
                    .reconnect()
                    .await
                    .context("reconnect FTP session after canceled download")?;
                anyhow::bail!("transfer canceled")
            }
        }
    }

    async fn run_ftp_upload(
        &self,
        id: &str,
        session: Arc<FtpSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        self.checkpoint(id, 0).await?;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let candidate_offset = requested_offset.min(local_len);
        let pause = self.pauses.lock().await.get(id).cloned();
        let pause_all = self.pause_all.clone();
        let canceled = self
            .cancels
            .lock()
            .await
            .get(id)
            .cloned()
            .unwrap_or_else(|| Arc::new(AtomicBool::new(false)));
        let local = local_path.to_path_buf();
        let remote = remote_path.to_string();

        let retry_pause = pause.clone();
        let retry_pause_all = pause_all.clone();
        let retry_canceled = canceled.clone();
        let retry_local = local.clone();
        let retry_remote = remote.clone();

        let attempt = session
            .with_stream(move |stream| {
                // Resume only when the remote file is exactly the committed
                // prefix. Otherwise overwrite from zero.
                let remote_len = if candidate_offset > 0 {
                    stream.size(&remote).ok().map(|size| size as u64)
                } else {
                    None
                };
                let offset = if remote_len == Some(candidate_offset) {
                    candidate_offset
                } else {
                    0
                };

                let mut file = std::fs::File::open(&local)
                    .with_context(|| format!("open {}", local.display()))?;
                if offset > 0 {
                    file.seek(SeekFrom::Start(offset))?;
                }
                let mut reader = std::io::BufReader::new(file);
                let outcome = stream.stor_resumable(&remote, offset, &mut reader, |_| {
                    if canceled.load(Ordering::Acquire) {
                        FtpTransferControl::Cancel
                    } else if pause_all.is_paused()
                        || pause.as_ref().is_some_and(|gate| gate.is_paused())
                    {
                        FtpTransferControl::Pause
                    } else {
                        FtpTransferControl::Continue
                    }
                })?;

                // Some FTP/FTPS servers can acknowledge a resumed transfer but
                // persist fewer bytes than the client wrote. A resumed upload
                // is complete only when SIZE confirms the exact local length.
                // If SIZE itself becomes unavailable after a resumed attempt,
                // also distrust the session and retry on a fresh connection.
                let needs_fresh_restart = if outcome.control == FtpTransferControl::Continue {
                    match stream.size(&remote).ok().map(|size| size as u64) {
                        Some(size) => size != local_len,
                        None => offset > 0,
                    }
                } else {
                    false
                };

                Ok((outcome, needs_fresh_restart))
            })
            .await;

        let (mut outcome, needs_fresh_restart) = match attempt {
            Ok(result) => result,
            Err(error) => {
                return Err(match session.reconnect().await {
                    Ok(()) => error,
                    Err(reconnect_error) => anyhow::anyhow!(
                        "{error:#}; additionally failed to reconnect FTP session after upload failure: {reconnect_error:#}"
                    ),
                });
            }
        };

        if needs_fresh_restart {
            session
                .reconnect()
                .await
                .context("reconnect FTP session before safe upload restart")?;

            outcome = session
                .with_stream(move |stream| {
                    if retry_canceled.load(Ordering::Acquire) {
                        return Ok(crate::session::ftp::FtpTransferOutcome {
                            transferred: 0,
                            control: FtpTransferControl::Cancel,
                        });
                    }
                    if retry_pause_all.is_paused()
                        || retry_pause
                            .as_ref()
                            .is_some_and(|gate| gate.is_paused())
                    {
                        return Ok(crate::session::ftp::FtpTransferOutcome {
                            transferred: 0,
                            control: FtpTransferControl::Pause,
                        });
                    }

                    // The bounded writer already proved unreliable for this
                    // server/session combination. Recovery deliberately uses
                    // SuppaFTP's established full-upload path on a fresh
                    // connection, then verifies both bytes sent and remote SIZE.
                    let restart_file = std::fs::File::open(&retry_local)
                        .with_context(|| format!("reopen {}", retry_local.display()))?;
                    let mut restart_reader = std::io::BufReader::new(restart_file);
                    let written = stream.put_from_reader(&retry_remote, &mut restart_reader)?;
                    if written != local_len {
                        anyhow::bail!(
                            "FTP recovery upload wrote {} bytes, expected {}",
                            written,
                            local_len
                        );
                    }
                    if let Ok(size) = stream.size(&retry_remote) {
                        if size as u64 != local_len {
                            anyhow::bail!(
                                "FTP upload verification failed after reconnect: remote size {} != local size {}",
                                size,
                                local_len
                            );
                        }
                    }

                    Ok(crate::session::ftp::FtpTransferOutcome {
                        transferred: local_len,
                        control: FtpTransferControl::Continue,
                    })
                })
                .await?;
        }

        self.update(id, |t| t.transferred = outcome.transferred)
            .await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://progress", &t);
        }

        match outcome.control {
            FtpTransferControl::Continue => Ok(()),
            FtpTransferControl::Pause => {
                self.checkpoint(id, 0).await?;
                session
                    .reconnect()
                    .await
                    .context("reconnect FTP session after paused upload")?;
                Err(RestartFromPause.into())
            }
            FtpTransferControl::Cancel => {
                session
                    .reconnect()
                    .await
                    .context("reconnect FTP session after canceled upload")?;
                anyhow::bail!("transfer canceled")
            }
        }
    }

    async fn run_object_download(
        &self,
        id: &str,
        session: Arc<ObjectSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;
        use tokio::io::AsyncWriteExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let key = remote_path.trim_start_matches('/');
        let p = object_store::path::Path::from(key);
        let get = session
            .store
            .get(&p)
            .await
            .with_context(|| format!("s3 get {key}"))?;

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = get.into_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("s3 chunk for {key}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    async fn run_object_upload(
        &self,
        id: &str,
        session: Arc<ObjectSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        use tokio::io::AsyncReadExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let key = remote_path.trim_start_matches('/');
        let p = object_store::path::Path::from(key);

        // For files under ~16 MB we put in one shot; larger files go through
        // a multipart upload so we get streaming + parallelism without buffering
        // the whole body in memory.
        let meta = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?;
        let size = meta.len();
        let mut file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;

        if size <= 16 * 1024 * 1024 {
            self.checkpoint(id, size).await?;
            let mut buf = Vec::with_capacity(size as usize);
            file.read_to_end(&mut buf).await?;
            session
                .store
                .put(&p, bytes::Bytes::from(buf).into())
                .await
                .with_context(|| format!("s3 put {key}"))?;
            self.update(id, |t| t.transferred = size).await;
            if let Some(t) = self.get(id).await {
                let _ = app.emit("transfer://progress", &t);
            }
            return Ok(());
        }

        // Multipart path. object_store wants a `MultipartUpload` for which we
        // push parts and call complete() at the end.
        let mut upload = session
            .store
            .put_multipart(&p)
            .await
            .with_context(|| format!("s3 begin multipart {key}"))?;

        const PART: usize = 8 * 1024 * 1024;
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        let mut buf = vec![0u8; PART];
        loop {
            let mut filled = 0;
            while filled < buf.len() {
                let n = file.read(&mut buf[filled..]).await?;
                if n == 0 {
                    break;
                }
                filled += n;
            }
            if filled == 0 {
                break;
            }
            self.checkpoint(id, filled as u64).await?;
            let chunk = bytes::Bytes::copy_from_slice(&buf[..filled]);
            upload
                .put_part(chunk.into())
                .await
                .with_context(|| format!("s3 put_part {key}"))?;
            transferred += filled as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
            if filled < buf.len() {
                break;
            }
        }
        upload
            .complete()
            .await
            .with_context(|| format!("s3 complete multipart {key}"))?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Stream a WebDAV download: a single ranged-capable `GET`, written to the
    /// local file chunk by chunk so progress is real (not 0→size at the end).
    async fn run_webdav_download(
        &self,
        id: &str,
        session: Arc<WebdavSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let url = session.url_for(remote_path, false);
        let resp = session
            .request(reqwest::Method::GET, url)
            .send()
            .await
            .with_context(|| format!("GET {remote_path}"))?;
        if !resp.status().is_success() {
            return Err(anyhow::anyhow!(
                "download {remote_path} failed: HTTP {}",
                resp.status().as_u16()
            ));
        }

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("webdav chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Upload via WebDAV `PUT`, streaming the file body straight off disk (no
    /// full-file buffering) with an explicit Content-Length so servers accept it.
    async fn run_webdav_upload(
        &self,
        id: &str,
        session: Arc<WebdavSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        use tokio_util::io::ReaderStream;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // WebDAV PUT reports at completion, like the FTP path.

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        let body = reqwest::Body::wrap_stream(ReaderStream::new(file));
        self.checkpoint(id, size).await?;

        let url = session.url_for(remote_path, false);
        let resp = session
            .request(reqwest::Method::PUT, url)
            .header(reqwest::header::CONTENT_LENGTH, size)
            .body(body)
            .send()
            .await
            .with_context(|| format!("PUT {remote_path}"))?;
        if !resp.status().is_success() {
            return Err(anyhow::anyhow!(
                "upload {remote_path} failed: HTTP {}",
                resp.status().as_u16()
            ));
        }
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Stream a read-only HTTP download: a single `GET`, written chunk by chunk.
    async fn run_http_download(
        &self,
        id: &str,
        session: Arc<HttpSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let url = session.url_for(remote_path, false);
        let resp = session
            .request(reqwest::Method::GET, url)
            .send()
            .await
            .with_context(|| format!("GET {remote_path}"))?;
        if !resp.status().is_success() {
            return Err(anyhow::anyhow!(
                "download {remote_path} failed: HTTP {}",
                resp.status().as_u16()
            ));
        }

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("http chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Stream a Dropbox download: POST `/2/files/download` (path in the
    /// `Dropbox-API-Arg` header), written chunk by chunk.
    async fn run_dropbox_download(
        &self,
        id: &str,
        session: Arc<DropboxSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let dbx = crate::remotefs::dropbox::dropbox_api_path(remote_path);
        let arg = serde_json::json!({ "path": dbx }).to_string();
        let resp = session.content_get("/2/files/download", &arg).await?;

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("dropbox chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Upload a file to Dropbox via `/2/files/upload` (overwrite mode). Simple
    /// single-shot upload; Dropbox caps that at 150 MB, so larger files are
    /// refused with a clear message (chunked upload_session is a follow-up).
    async fn run_dropbox_upload(
        &self,
        id: &str,
        session: Arc<DropboxSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        use tokio_util::io::ReaderStream;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // Dropbox reports at completion, like the FTP/WebDAV paths.

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        const SIMPLE_UPLOAD_MAX: u64 = 150 * 1024 * 1024;
        if size > SIMPLE_UPLOAD_MAX {
            return Err(anyhow::anyhow!(
                "{} exceeds Dropbox's 150 MB single-request upload limit \
                 (chunked upload not yet implemented)",
                local_path.display()
            ));
        }

        let dbx = crate::remotefs::dropbox::dropbox_api_path(remote_path);
        let arg = serde_json::json!({
            "path": dbx, "mode": "overwrite", "autorename": false, "mute": true
        })
        .to_string();
        let url = format!("{}/2/files/upload", session.content_base);

        // Proactive refresh covers the common case; on a hard 401 we refresh and
        // retry once, re-opening the file for a fresh streamed body.
        self.checkpoint(id, size).await?;
        let mut attempt = 0;
        loop {
            let token = session.access_token().await?;
            let file = tokio::fs::File::open(local_path)
                .await
                .with_context(|| format!("open {}", local_path.display()))?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(file));
            let resp = session
                .client
                .post(&url)
                .bearer_auth(&token)
                .header("Dropbox-API-Arg", &arg)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(body)
                .send()
                .await
                .with_context(|| format!("PUT {remote_path}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                attempt += 1;
                session.force_refresh().await?;
                continue;
            }
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "upload {remote_path} failed ({code}): {text}"
                ));
            }
            break;
        }
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Download a Shopify theme asset. The Assets API answers with the whole
    /// file as JSON (`value`/base64 `attachment`), so there is nothing to
    /// stream — read it, write it out, report at completion like the FTP path.
    async fn run_shopify_download(
        &self,
        id: &str,
        session: Arc<ShopifySession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::shopify::read_asset(&session, remote_path).await?;
        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(())
    }

    /// Upload a file as a Shopify theme asset (create and update are the same
    /// PUT; theme files are small, so a single-shot write is the whole story).
    async fn run_shopify_upload(
        &self,
        id: &str,
        session: Arc<ShopifySession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // Shopify reports at completion, like the Dropbox path.

        let data = tokio::fs::read(local_path)
            .await
            .with_context(|| format!("read {}", local_path.display()))?;
        let size = data.len() as u64;
        self.checkpoint(id, size).await?;
        crate::remotefs::shopify::write_asset(&session, remote_path, &data).await?;
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Download a HubSpot Design Manager file. The Source Code API answers
    /// with the whole file as an octet-stream, so there is nothing to stream —
    /// read it, write it out, report at completion like the Shopify path.
    async fn run_hubspot_download(
        &self,
        id: &str,
        session: Arc<HubSpotSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::hubspot::read_file(&session, remote_path).await?;
        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(())
    }

    /// Upload a file to the HubSpot Design Manager (create and update are the
    /// same multipart PUT; theme files are small, so a single-shot write is
    /// the whole story).
    async fn run_hubspot_upload(
        &self,
        id: &str,
        session: Arc<HubSpotSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // HubSpot reports at completion, like the Shopify path.

        let data = tokio::fs::read(local_path)
            .await
            .with_context(|| format!("read {}", local_path.display()))?;
        let size = data.len() as u64;
        self.checkpoint(id, size).await?;
        crate::remotefs::hubspot::write_file(&session, remote_path, &data).await?;
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Download a Dataverse web resource. The Web API answers with the whole
    /// file as base64 `content`, so there is nothing to stream — read it,
    /// write it out, report at completion like the HubSpot path.
    async fn run_dynamics_download(
        &self,
        id: &str,
        session: Arc<DynamicsSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::dynamics::read_file(&session, remote_path).await?;
        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(())
    }

    /// Upload a file as a Dataverse web resource (create or update by name
    /// lookup, then publish — save = deployed).
    async fn run_dynamics_upload(
        &self,
        id: &str,
        session: Arc<DynamicsSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // Dynamics reports at completion, like the HubSpot path.

        let data = tokio::fs::read(local_path)
            .await
            .with_context(|| format!("read {}", local_path.display()))?;
        let size = data.len() as u64;
        self.checkpoint(id, size).await?;
        crate::remotefs::dynamics::write_file(&session, remote_path, &data).await?;
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Stream a OneDrive download: GET the item's `/content` (Graph 302s to a
    /// pre-authorized URL, which reqwest follows), written chunk by chunk.
    async fn run_onedrive_download(
        &self,
        id: &str,
        session: Arc<OneDriveSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let content = crate::remotefs::onedrive::content_ref(remote_path);
        let resp = session.get_stream(&content).await?;

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("onedrive chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Upload to OneDrive: a single `PUT …/content` for small files, or a
    /// chunked upload session for larger ones (Graph caps simple PUT at 4 MB).
    async fn run_onedrive_upload(
        &self,
        id: &str,
        session: Arc<OneDriveSession>,
        local_path: &Path,
        remote_path: &str,
        app: &AppHandle,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        // Graph's simple-upload ceiling is 4 MB; overridable for tests.
        let simple_max: u64 = std::env::var("GHOSTFTP_ONEDRIVE_SIMPLE_MAX")
            .ok()
            .and_then(|s| s.parse().ok())
            .unwrap_or(4 * 1024 * 1024);

        if size <= simple_max {
            self.checkpoint(id, size).await?;
            self.onedrive_simple_upload(id, &session, local_path, remote_path)
                .await?;
        } else {
            self.onedrive_session_upload(id, &session, local_path, remote_path, size, app)
                .await?;
        }
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    async fn onedrive_simple_upload(
        &self,
        _id: &str,
        session: &Arc<OneDriveSession>,
        local_path: &Path,
        remote_path: &str,
    ) -> Result<()> {
        use tokio_util::io::ReaderStream;
        let content = crate::remotefs::onedrive::content_ref(remote_path);
        let url = format!("{}{content}", session.graph_base);
        let mut attempt = 0;
        loop {
            let token = session.access_token().await?;
            let file = tokio::fs::File::open(local_path)
                .await
                .with_context(|| format!("open {}", local_path.display()))?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(file));
            let resp = session
                .client
                .put(&url)
                .bearer_auth(&token)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(body)
                .send()
                .await
                .with_context(|| format!("PUT {remote_path}"))?;
            if resp.status() == reqwest::StatusCode::UNAUTHORIZED && attempt == 0 {
                attempt += 1;
                session.force_refresh().await?;
                continue;
            }
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "upload {remote_path} failed ({code}): {text}"
                ));
            }
            return Ok(());
        }
    }

    async fn onedrive_session_upload(
        &self,
        id: &str,
        session: &Arc<OneDriveSession>,
        local_path: &Path,
        remote_path: &str,
        size: u64,
        app: &AppHandle,
    ) -> Result<()> {
        // Create the upload session.
        let item = crate::remotefs::onedrive::item_ref(remote_path);
        let create = format!("{item}/createUploadSession");
        let body = serde_json::json!({
            "item": { "@microsoft.graph.conflictBehavior": "replace" }
        });
        let sess = session
            .rpc(reqwest::Method::POST, &create, Some(&body))
            .await?;
        let upload_url = sess
            .get("uploadUrl")
            .and_then(|u| u.as_str())
            .ok_or_else(|| anyhow::anyhow!("createUploadSession returned no uploadUrl"))?
            .to_string();

        // Chunks must be a multiple of 320 KiB (except the last). ~6 MiB.
        const CHUNK: usize = 320 * 1024 * 20;
        let mut file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        let mut buf = vec![0u8; CHUNK];
        let mut offset: u64 = 0;
        let mut last_emit = Instant::now();
        while offset < size {
            let mut filled = 0;
            while filled < buf.len() {
                let n = file.read(&mut buf[filled..]).await?;
                if n == 0 {
                    break;
                }
                filled += n;
            }
            if filled == 0 {
                break;
            }
            self.checkpoint(id, filled as u64).await?;
            let start = offset;
            let end = offset + filled as u64 - 1;
            let range = format!("bytes {start}-{end}/{size}");
            let resp = session
                .client
                .put(&upload_url)
                .header(reqwest::header::CONTENT_LENGTH, filled as u64)
                .header("Content-Range", range)
                .body(bytes::Bytes::copy_from_slice(&buf[..filled]))
                .send()
                .await
                .with_context(|| format!("upload chunk for {remote_path}"))?;
            if !resp.status().is_success() {
                let code = resp.status().as_u16();
                let text = resp.text().await.unwrap_or_default();
                return Err(anyhow::anyhow!(
                    "upload {remote_path} chunk failed ({code}): {text}"
                ));
            }
            offset += filled as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = offset).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        Ok(())
    }

    /// Stream a Google Drive download: resolve the path to a file id, then GET
    /// `/files/{id}?alt=media`.
    async fn run_gdrive_download(
        &self,
        id: &str,
        session: Arc<GDriveSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let (file_id, _) = session
            .resolve_item(remote_path)
            .await?
            .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
        let resp = session
            .get_stream(&format!("/files/{file_id}?alt=media"))
            .await?;

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("drive chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Upload to Google Drive: update the existing file's media if a same-named
    /// child exists, else create a new file via a multipart/related request.
    async fn run_gdrive_upload(
        &self,
        id: &str,
        session: Arc<GDriveSession>,
        local_path: &Path,
        remote_path: &str,
        _app: &AppHandle,
    ) -> Result<()> {
        use crate::session::gdrive::{basename, normalize, parent_of};

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let norm = normalize(remote_path);
        let name = basename(&norm).to_string();
        let parent_id = session.folder_id(&parent_of(&norm)).await?;
        let existing = session.find_child(&parent_id, &name).await?;
        let bytes = tokio::fs::read(local_path)
            .await
            .with_context(|| format!("read {}", local_path.display()))?;
        let token = session.access_token().await?;
        self.checkpoint(id, size).await?;

        let resp = if let Some((file_id, _)) = existing {
            // Update the existing file's content in place.
            session
                .client
                .patch(format!(
                    "{}/files/{file_id}?uploadType=media",
                    session.upload_base
                ))
                .bearer_auth(&token)
                .header(reqwest::header::CONTENT_TYPE, "application/octet-stream")
                .body(bytes)
                .send()
                .await
                .with_context(|| format!("update {remote_path}"))?
        } else {
            // Create a new file: multipart/related metadata + media.
            let boundary = format!("ghostftp{}", Uuid::new_v4().simple());
            let meta = serde_json::json!({ "name": name, "parents": [parent_id] });
            let mut body: Vec<u8> = Vec::with_capacity(bytes.len() + 256);
            body.extend_from_slice(format!("--{boundary}\r\n").as_bytes());
            body.extend_from_slice(b"Content-Type: application/json; charset=UTF-8\r\n\r\n");
            body.extend_from_slice(meta.to_string().as_bytes());
            body.extend_from_slice(format!("\r\n--{boundary}\r\n").as_bytes());
            body.extend_from_slice(b"Content-Type: application/octet-stream\r\n\r\n");
            body.extend_from_slice(&bytes);
            body.extend_from_slice(format!("\r\n--{boundary}--\r\n").as_bytes());
            session
                .client
                .post(format!(
                    "{}/files?uploadType=multipart&fields=id",
                    session.upload_base
                ))
                .bearer_auth(&token)
                .header(
                    reqwest::header::CONTENT_TYPE,
                    format!("multipart/related; boundary={boundary}"),
                )
                .body(body)
                .send()
                .await
                .with_context(|| format!("create {remote_path}"))?
        };

        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow::anyhow!(
                "upload {remote_path} failed ({code}): {text}"
            ));
        }
        session.clear_cache();
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }

    /// Stream a Box download: resolve the path to a file id, GET `/files/{id}/content`.
    async fn run_box_download(
        &self,
        id: &str,
        session: Arc<BoxSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
    ) -> Result<()> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let (file_id, _) = session
            .resolve_item(remote_path)
            .await?
            .ok_or_else(|| anyhow::anyhow!("{remote_path}: not found"))?;
        let resp = session
            .get_stream(&format!("/files/{file_id}/content"))
            .await?;

        let mut file = tokio::fs::File::create(local_path)
            .await
            .with_context(|| format!("create {}", local_path.display()))?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("box chunk for {remote_path}"))?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred += chunk.len() as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(())
    }

    /// Upload to Box via multipart/form-data: a new file (`/files/content` with
    /// attributes) or a new version of an existing one (`/files/{id}/content`).
    async fn run_box_upload(
        &self,
        id: &str,
        session: Arc<BoxSession>,
        local_path: &Path,
        remote_path: &str,
        _app: &AppHandle,
    ) -> Result<()> {
        use crate::session::boxdrive::{basename, normalize, parent_of};

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let norm = normalize(remote_path);
        let name = basename(&norm).to_string();
        let parent_id = session.folder_id(&parent_of(&norm)).await?;
        let existing = session.find_child(&parent_id, &name).await?;
        let bytes = tokio::fs::read(local_path)
            .await
            .with_context(|| format!("read {}", local_path.display()))?;
        let token = session.access_token().await?;
        self.checkpoint(id, size).await?;

        let file_part = reqwest::multipart::Part::bytes(bytes).file_name(name.clone());
        let (url, form) = match existing {
            Some((file_id, false)) => (
                format!("{}/files/{file_id}/content", session.upload_base),
                reqwest::multipart::Form::new().part("file", file_part),
            ),
            _ => {
                let attrs = serde_json::json!({ "name": name, "parent": { "id": parent_id } });
                (
                    format!("{}/files/content", session.upload_base),
                    reqwest::multipart::Form::new()
                        .text("attributes", attrs.to_string())
                        .part("file", file_part),
                )
            }
        };
        let resp = session
            .client
            .post(&url)
            .bearer_auth(&token)
            .multipart(form)
            .send()
            .await
            .with_context(|| format!("upload {remote_path}"))?;
        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow::anyhow!(
                "upload {remote_path} failed ({code}): {text}"
            ));
        }
        session.clear_cache();
        self.update(id, |t| t.transferred = size).await;
        Ok(())
    }
}

/// Build a RemoteFs handle for the right backend.
fn fs_for_session(session: &Arc<Session>) -> Box<dyn crate::remotefs::RemoteFs> {
    match &**session {
        Session::Ssh(ssh) => Box::new(crate::remotefs::sftp::SftpFs::new(ssh.clone())),
        Session::Ftp(ftp) => Box::new(crate::remotefs::ftp::FtpFs::new(ftp.clone())),
        Session::Object(obj) => Box::new(crate::remotefs::object::ObjectFs::new(obj.clone())),
        Session::Webdav(dav) => Box::new(crate::remotefs::webdav::WebdavFs::new(dav.clone())),
        Session::Http(http) => Box::new(crate::remotefs::http::HttpFs::new(http.clone())),
        Session::Dropbox(dbx) => Box::new(crate::remotefs::dropbox::DropboxFs::new(dbx.clone())),
        Session::OneDrive(od) => Box::new(crate::remotefs::onedrive::OneDriveFs::new(od.clone())),
        Session::GDrive(gd) => Box::new(crate::remotefs::gdrive::GDriveFs::new(gd.clone())),
        Session::Box(bx) => Box::new(crate::remotefs::boxdrive::BoxFs::new(bx.clone())),
        Session::Shopify(sh) => Box::new(crate::remotefs::shopify::ShopifyFs::new(sh.clone())),
        Session::HubSpot(hs) => Box::new(crate::remotefs::hubspot::HubSpotFs::new(hs.clone())),
        Session::Dynamics(dynm) => {
            Box::new(crate::remotefs::dynamics::DynamicsFs::new(dynm.clone()))
        }
        Session::Agent(agent) => Box::new(crate::remotefs::agent::AgentFs::new(agent.clone())),
    }
}

/// HEAD a WebDAV resource, returning its size (from Content-Length) and whether
/// it exists. Best-effort: a server that rejects HEAD reports (0, false).
async fn webdav_head(session: &Arc<WebdavSession>, path: &str) -> (u64, bool) {
    let url = session.url_for(path, false);
    match session.request(reqwest::Method::HEAD, url).send().await {
        Ok(resp) if resp.status().is_success() => {
            let size = resp
                .headers()
                .get(reqwest::header::CONTENT_LENGTH)
                .and_then(|v| v.to_str().ok())
                .and_then(|s| s.parse::<u64>().ok())
                .unwrap_or(0);
            (size, true)
        }
        _ => (0, false),
    }
}

/// HEAD an HTTP-source file for its size. Best-effort (0 on any failure).
async fn http_size(session: &Arc<HttpSession>, path: &str) -> u64 {
    let url = session.url_for(path, false);
    match session.request(reqwest::Method::HEAD, url).send().await {
        Ok(resp) if resp.status().is_success() => resp
            .headers()
            .get(reqwest::header::CONTENT_LENGTH)
            .and_then(|v| v.to_str().ok())
            .and_then(|s| s.parse::<u64>().ok())
            .unwrap_or(0),
        _ => 0,
    }
}

/// Delta sync exists only for the Ghost FTP Agent backend: it's the one
/// remote where we run code, so a chunk signature + server-side reassemble is
/// possible. Every other backend arm dispatches to a plain whole-file copy.
/// The dispatch matches below already route only `Session::Agent` to the
/// `*_with_delta` entry points — this helper pins that contract (and the test
/// at the bottom of the file exercises it).
fn supports_delta(session: &Session) -> bool {
    matches!(session, Session::Agent(_))
}

/// Byte-range pause/resume is enabled only where Ghost FTP can prove exact
/// offset semantics today. SFTP exposes seekable file handles and Agent uses
/// explicit ReadChunk/WriteChunk offsets. Other backends retain the safe
/// restart-from-zero fallback until equivalent tests exist.
fn supports_byte_resume(session: &Session) -> bool {
    matches!(
        session,
        Session::Ssh(_) | Session::Ftp(_) | Session::Agent(_)
    )
}

/// Stat a path on a Ghost FTP Agent daemon, returning its size and whether it exists.
async fn agent_stat(session: &Arc<crate::session::AgentSession>, path: &str) -> (u64, bool) {
    use ghostftp_agent_proto::msg::{Request, Response};
    match session
        .request(Request::Stat {
            path: path.to_string(),
        })
        .await
    {
        Ok(Response::Stat { entry }) => (entry.size, true),
        _ => (0, false),
    }
}

/// Lookup the size of a remote file using each backend's native API.
pub(crate) async fn remote_size(session: &Arc<Session>, path: &str) -> Result<u64> {
    match &**session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            Ok(sftp
                .metadata(path)
                .await
                .with_context(|| format!("stat {path}"))?
                .size
                .unwrap_or(0))
        }
        Session::Ftp(ftp) => {
            let path = path.to_string();
            let sz = ftp.with_stream(move |s| s.size(&path)).await?;
            Ok(sz as u64)
        }
        Session::Object(obj) => {
            let key = path.trim_start_matches('/').to_string();
            let p = object_store::path::Path::from(key.as_str());
            let meta = obj
                .store
                .head(&p)
                .await
                .with_context(|| format!("object head {key}"))?;
            Ok(meta.size as u64)
        }
        Session::Webdav(dav) => Ok(webdav_head(dav, path).await.0),
        Session::Http(http) => Ok(http_size(http, path).await),
        Session::Dropbox(dbx) => Ok(dbx
            .size(&crate::remotefs::dropbox::dropbox_api_path(path))
            .await),
        Session::OneDrive(od) => Ok(od.size(&crate::remotefs::onedrive::item_ref(path)).await),
        Session::GDrive(gd) => Ok(gd.size(path).await),
        Session::Box(bx) => Ok(bx.size(path).await),
        Session::Shopify(sh) => Ok(crate::remotefs::shopify::asset_size(sh, path).await),
        Session::HubSpot(hs) => Ok(crate::remotefs::hubspot::file_size(hs, path).await),
        Session::Dynamics(dynm) => Ok(crate::remotefs::dynamics::file_size(dynm, path).await),
        Session::Agent(agent) => Ok(agent_stat(agent, path).await.0),
    }
}

/// Apply the overwrite policy on the remote side; returns (final_path, skip).
async fn remote_resolve(
    session: &Arc<Session>,
    initial_remote: &str,
    policy: OverwritePolicy,
) -> Result<(String, bool)> {
    match &**session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => {
                    let exists = sftp.metadata(initial_remote).await.is_ok();
                    (initial_remote.to_string(), exists)
                }
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename(&sftp, initial_remote).await?;
                    (renamed, false)
                }
            })
        }
        Session::Ftp(ftp) => {
            let probe = initial_remote.to_string();
            let exists = ftp.with_stream(move |s| Ok(s.size(&probe).is_ok())).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let session = ftp.clone();
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        let probe = candidate.clone();
                        let found = session
                            .with_stream(move |s| Ok(s.size(&probe).is_ok()))
                            .await?;
                        if !found {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Object(obj) => {
            let key = initial_remote.trim_start_matches('/').to_string();
            let probe = object_store::path::Path::from(key.as_str());
            let exists = obj.store.head(&probe).await.is_ok();
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        let key = candidate.trim_start_matches('/');
                        let p = object_store::path::Path::from(key);
                        if obj.store.head(&p).await.is_err() {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Webdav(dav) => {
            let (_, exists) = webdav_head(dav, initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !webdav_head(dav, &candidate).await.1 {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Http(_) => {
            // Read-only: no upload will actually run (run_http_upload errors), so
            // resolution is a no-op that just echoes the target back.
            Ok((initial_remote.to_string(), false))
        }
        Session::Dropbox(dbx) => {
            let dbx_path = crate::remotefs::dropbox::dropbox_api_path(initial_remote);
            let exists = dbx.exists(&dbx_path).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        let p = crate::remotefs::dropbox::dropbox_api_path(&candidate);
                        if !dbx.exists(&p).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::OneDrive(od) => {
            let exists = od
                .exists(&crate::remotefs::onedrive::item_ref(initial_remote))
                .await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !od
                            .exists(&crate::remotefs::onedrive::item_ref(&candidate))
                            .await
                        {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::GDrive(gd) => {
            let exists = gd.exists(initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !gd.exists(&candidate).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Box(bx) => {
            let exists = bx.exists(initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !bx.exists(&candidate).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Shopify(sh) => {
            let exists = crate::remotefs::shopify::asset_exists(sh, initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !crate::remotefs::shopify::asset_exists(sh, &candidate).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::HubSpot(hs) => {
            let exists = crate::remotefs::hubspot::file_exists(hs, initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !crate::remotefs::hubspot::file_exists(hs, &candidate).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Dynamics(dynm) => {
            let exists = crate::remotefs::dynamics::file_exists(dynm, initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !crate::remotefs::dynamics::file_exists(dynm, &candidate).await {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
        Session::Agent(agent) => {
            let (_, exists) = agent_stat(agent, initial_remote).await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    for i in 1..=MAX_RENAME_CANDIDATES {
                        let candidate = remote_rename_candidate(initial_remote, i);
                        if !agent_stat(agent, &candidate).await.1 {
                            return Ok((candidate, false));
                        }
                    }
                    anyhow::bail!(
                        "no free remote rename target after {MAX_RENAME_CANDIDATES} candidates for {initial_remote}"
                    )
                }
            })
        }
    }
}
fn split_ext(path: &str) -> (&str, &str) {
    let basename_start = path.rfind(['/', '\\']).map_or(0, |index| index + 1);
    let basename = &path[basename_start..];

    match basename.rfind('.') {
        // A leading dot is part of a hidden filename, not an extension.
        Some(relative_dot) if relative_dot > 0 => {
            let dot = basename_start + relative_dot;
            (&path[..dot], &path[dot..])
        }
        _ => (path, ""),
    }
}

fn remote_rename_candidate(path: &str, index: usize) -> String {
    let (stem, ext) = split_ext(path);
    format!("{stem}_{index}{ext}")
}

/// Transient = worth an auto-retry (structured IPC error kinds):
/// network and timeout failures; auth/permission/not-found never retry.
fn is_transient(e: &anyhow::Error) -> bool {
    matches!(
        crate::error::classify_message(&format!("{e:#}")),
        crate::error::ErrorKind::Network | crate::error::ErrorKind::Timeout
    )
}

/// Re-enter the global FIFO after retry backoff without monopolizing a
/// concurrency slot. Keeping retries in the same queue as fresh work prevents
/// an offline endpoint from repeatedly jumping ahead of already-waiting sites.
///
/// A transfer can be paused while it is sleeping or while it is waiting for
/// readmission. In that case it stays queued until resume opens its gate; it
/// must never silently lose its worker task.
async fn readmit_after_retry_backoff(
    mgr: &Arc<TransferManager>,
    id: &str,
    app: &AppHandle,
) -> Option<OwnedSemaphorePermit> {
    loop {
        match mgr.get(id).await.map(|t| t.status) {
            Some(TransferStatus::Transferring)
            | Some(TransferStatus::Queued)
            | Some(TransferStatus::Paused) => {}
            _ => return None,
        }

        {
            let mut waiting = mgr.waiting.lock().await;
            if !waiting.iter().any(|queued| queued == id) {
                waiting.push_back(id.to_string());
            }
        }
        mgr.bump_queue(app).await;

        let permit = mgr.admit(id).await?;

        let runnable = {
            let mut transfers = mgr.transfers.lock().await;
            let transfer = transfers.get_mut(id)?;
            match transfer.status {
                TransferStatus::Paused => false,
                TransferStatus::Queued | TransferStatus::Transferring => {
                    transfer.status = TransferStatus::Transferring;
                    transfer.error = None;
                    true
                }
                _ => return None,
            }
        };

        if runnable {
            if let Some(t) = mgr.get(id).await {
                let _ = app.emit("transfer://updated", &t);
            }
            return Some(permit);
        }

        // Pause raced with admission after the id was popped from the waiting queue.
        // Release the slot and loop so the id is queued again behind work that
        // was already waiting. resume() will wake admission through queue_gen.
        drop(permit);
    }
}

/// Shared download runner: admission → run loop → finalize → wake
/// the queue. The loop re-runs the file from byte 0 after a resume-from-pause
/// (Phase 2) and auto-retries transient errors with 5s/20s backoff.
/// Retry backoff releases the concurrency permit so an unreachable server
/// cannot reduce throughput for unrelated queued transfers. The transfer must
/// reacquire a permit before its next network attempt.
async fn run_download_task(
    mgr: Arc<TransferManager>,
    id: String,
    session: Arc<Session>,
    remote_path: String,
    final_path: PathBuf,
    app: AppHandle,
) {
    let Some(initial_permit) = mgr.admit(&id).await else {
        return;
    };
    let mut permit = Some(initial_permit);
    let mut auto_retries = 0u32;
    let max_auto_retries = mgr.max_auto_retries.load(Ordering::Relaxed) as u32;
    // Agent delta-download uses the existing destination as its block-level
    // basis and already assembles into its own verified temporary file. Every
    // other backend writes a whole-file stream, so route it through a sibling
    // staging file and promote only after the complete backend operation succeeds.
    let staged_download = !matches!(&*session, Session::Agent(_));
    let working_path = if staged_download {
        download_staging_path(&final_path, &id)
    } else {
        final_path.clone()
    };
    let res = loop {
        let attempt =
            dispatch_download(&mgr, &id, &session, &remote_path, &working_path, &app).await;
        match attempt {
            Err(e) if e.downcast_ref::<RestartFromPause>().is_some() => {
                if !supports_byte_resume(&session) {
                    mgr.update(&id, |t| t.transferred = 0).await;
                }
                continue;
            }
            Err(e) if auto_retries < max_auto_retries && is_transient(&e) => {
                auto_retries += 1;
                let delay = match auto_retries {
                    1 => 5,
                    2 => 20,
                    _ => 60,
                };
                mgr.update(&id, |t| {
                    t.transferred = 0;
                    t.retry_attempt = Some(auto_retries);
                    t.error = Some(format!(
                        "retrying in {delay}s (attempt {}/{})",
                        auto_retries + 1,
                        max_auto_retries + 1
                    ));
                })
                .await;
                if let Some(t) = mgr.get(&id).await {
                    let _ = app.emit("transfer://updated", &t);
                }
                // A retry that is only waiting must not occupy one of the
                // global transfer slots. This keeps healthy sites moving when
                // another endpoint is offline or timing out.
                drop(permit.take());
                tokio::time::sleep(Duration::from_secs(delay)).await;
                // Re-enter through the same FIFO as every other transfer.
                // This also keeps a transfer alive if the user pauses it during
                // backoff: admission waits for resume instead of abandoning the
                // worker and leaving a resumable-looking but inert row behind.
                let Some(next_permit) = readmit_after_retry_backoff(&mgr, &id, &app).await else {
                    return;
                };
                permit = Some(next_permit);
                continue;
            }
            other => break other,
        }
    };
    let res = if staged_download {
        match res {
            Ok(()) => promote_staged_download(&working_path, &final_path, &id).await,
            Err(error) => {
                let _ = tokio::fs::remove_file(&working_path).await;
                Err(error)
            }
        }
    } else {
        res
    };
    finalize(&mgr, &id, &app, res).await;
    mgr.bump_queue(&app).await;
}

/// Upload twin of `run_download_task`.
async fn run_upload_task(
    mgr: Arc<TransferManager>,
    id: String,
    session: Arc<Session>,
    local: PathBuf,
    final_remote: String,
    app: AppHandle,
) {
    let Some(initial_permit) = mgr.admit(&id).await else {
        return;
    };
    let mut permit = Some(initial_permit);
    let mut auto_retries = 0u32;
    let max_auto_retries = mgr.max_auto_retries.load(Ordering::Relaxed) as u32;
    let res = loop {
        let attempt = dispatch_upload(&mgr, &id, &session, &local, &final_remote, &app).await;
        match attempt {
            Err(e) if e.downcast_ref::<RestartFromPause>().is_some() => {
                if !supports_byte_resume(&session) {
                    mgr.update(&id, |t| t.transferred = 0).await;
                }
                continue;
            }
            Err(e) if auto_retries < max_auto_retries && is_transient(&e) => {
                auto_retries += 1;
                let delay = match auto_retries {
                    1 => 5,
                    2 => 20,
                    _ => 60,
                };
                mgr.update(&id, |t| {
                    t.transferred = 0;
                    t.retry_attempt = Some(auto_retries);
                    t.error = Some(format!(
                        "retrying in {delay}s (attempt {}/{})",
                        auto_retries + 1,
                        max_auto_retries + 1
                    ));
                })
                .await;
                if let Some(t) = mgr.get(&id).await {
                    let _ = app.emit("transfer://updated", &t);
                }
                // A retry that is only waiting must not occupy one of the
                // global transfer slots. This keeps healthy sites moving when
                // another endpoint is offline or timing out.
                drop(permit.take());
                tokio::time::sleep(Duration::from_secs(delay)).await;
                // Re-enter through the same FIFO as every other transfer.
                // This also keeps a transfer alive if the user pauses it during
                // backoff: admission waits for resume instead of abandoning the
                // worker and leaving a resumable-looking but inert row behind.
                let Some(next_permit) = readmit_after_retry_backoff(&mgr, &id, &app).await else {
                    return;
                };
                permit = Some(next_permit);
                continue;
            }
            other => break other,
        }
    };
    finalize(&mgr, &id, &app, res).await;
    mgr.bump_queue(&app).await;
}

/// Backend dispatch for a single-file download. Extracted so the runner (and
/// manual retry) can re-invoke it.
async fn dispatch_download(
    mgr: &Arc<TransferManager>,
    id: &str,
    session: &Arc<Session>,
    remote_path: &str,
    final_path: &Path,
    app: &AppHandle,
) -> Result<()> {
    match &**session {
        Session::Ssh(ssh) => {
            mgr.run_ssh_download(id, ssh.clone(), remote_path, final_path, app)
                .await
        }
        Session::Ftp(ftp) => {
            mgr.run_ftp_download(id, ftp.clone(), remote_path, final_path, app)
                .await
        }
        Session::Object(obj) => {
            mgr.run_object_download(id, obj.clone(), remote_path, final_path, app)
                .await
        }
        Session::Webdav(dav) => {
            mgr.run_webdav_download(id, dav.clone(), remote_path, final_path, app)
                .await
        }
        Session::Http(http) => {
            mgr.run_http_download(id, http.clone(), remote_path, final_path, app)
                .await
        }
        Session::Dropbox(dbx) => {
            mgr.run_dropbox_download(id, dbx.clone(), remote_path, final_path, app)
                .await
        }
        Session::OneDrive(od) => {
            mgr.run_onedrive_download(id, od.clone(), remote_path, final_path, app)
                .await
        }
        Session::GDrive(gd) => {
            mgr.run_gdrive_download(id, gd.clone(), remote_path, final_path, app)
                .await
        }
        Session::Box(bx) => {
            mgr.run_box_download(id, bx.clone(), remote_path, final_path, app)
                .await
        }
        Session::Shopify(sh) => {
            mgr.run_shopify_download(id, sh.clone(), remote_path, final_path, app)
                .await
        }
        Session::HubSpot(hs) => {
            mgr.run_hubspot_download(id, hs.clone(), remote_path, final_path, app)
                .await
        }
        Session::Dynamics(dynm) => {
            mgr.run_dynamics_download(id, dynm.clone(), remote_path, final_path, app)
                .await
        }
        Session::Agent(agent) => {
            debug_assert!(supports_delta(session));
            mgr.run_agent_download_with_delta(id, agent.clone(), remote_path, final_path, app)
                .await
        }
    }
}

/// Backend dispatch for a single-file upload.
async fn dispatch_upload(
    mgr: &Arc<TransferManager>,
    id: &str,
    session: &Arc<Session>,
    local: &Path,
    final_remote: &str,
    app: &AppHandle,
) -> Result<()> {
    match &**session {
        Session::Ssh(ssh) => {
            mgr.run_ssh_upload(id, ssh.clone(), local, final_remote, app)
                .await
        }
        Session::Ftp(ftp) => {
            mgr.run_ftp_upload(id, ftp.clone(), local, final_remote, app)
                .await
        }
        Session::Object(obj) => {
            mgr.run_object_upload(id, obj.clone(), local, final_remote, app)
                .await
        }
        Session::Webdav(dav) => {
            mgr.run_webdav_upload(id, dav.clone(), local, final_remote, app)
                .await
        }
        Session::Http(_) => Err(anyhow::anyhow!(
            "HTTP source is read-only — upload not supported"
        )),
        Session::Dropbox(dbx) => {
            mgr.run_dropbox_upload(id, dbx.clone(), local, final_remote, app)
                .await
        }
        Session::OneDrive(od) => {
            mgr.run_onedrive_upload(id, od.clone(), local, final_remote, app)
                .await
        }
        Session::GDrive(gd) => {
            mgr.run_gdrive_upload(id, gd.clone(), local, final_remote, app)
                .await
        }
        Session::Box(bx) => {
            mgr.run_box_upload(id, bx.clone(), local, final_remote, app)
                .await
        }
        Session::Shopify(sh) => {
            mgr.run_shopify_upload(id, sh.clone(), local, final_remote, app)
                .await
        }
        Session::HubSpot(hs) => {
            mgr.run_hubspot_upload(id, hs.clone(), local, final_remote, app)
                .await
        }
        Session::Dynamics(dynm) => {
            mgr.run_dynamics_upload(id, dynm.clone(), local, final_remote, app)
                .await
        }
        Session::Agent(agent) => {
            debug_assert!(supports_delta(session));
            mgr.run_agent_upload_with_delta(id, agent.clone(), local, final_remote, app)
                .await
        }
    }
}

async fn finalize(mgr: &Arc<TransferManager>, id: &str, app: &AppHandle, result: Result<()>) {
    // Cancel updates the row immediately. Cooperative blocking transports (FTP)
    // may finish ABOR cleanup after that; never let their late result overwrite
    // the user-visible Canceled state.
    if mgr.get(id).await.map(|t| t.status) == Some(TransferStatus::Canceled) {
        mgr.tasks.lock().await.remove(id);
        return;
    }
    match result {
        Ok(()) => {
            mgr.update(id, |t| {
                t.status = TransferStatus::Done;
                t.transferred = t.size;
            })
            .await;
            if let Some(t) = mgr.get(id).await {
                let _ = app.emit("transfer://done", &t);
            }
        }
        Err(e) => {
            mgr.update(id, |t| {
                t.status = TransferStatus::Error;
                t.error = Some(e.to_string());
            })
            .await;
            if let Some(t) = mgr.get(id).await {
                let _ = app.emit("transfer://error", &t);
            }
        }
    }
    mgr.tasks.lock().await.remove(id);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn spawn_registered_makes_handle_visible_before_worker_cleanup() {
        let mgr = Arc::new(TransferManager::new());
        let worker_mgr = Arc::clone(&mgr);
        let id = "immediate-worker".to_string();
        let worker_id = id.clone();
        let (tx, rx) = tokio::sync::oneshot::channel();

        mgr.spawn_registered(id, async move {
            let registered = worker_mgr.tasks.lock().await.remove(&worker_id).is_some();
            let _ = tx.send(registered);
        })
        .await;

        assert_eq!(
            rx.await,
            Ok(true),
            "worker cleanup must observe its registered JoinHandle even when the worker runs immediately"
        );
    }

    #[tokio::test]
    async fn staged_download_promotion_replaces_only_after_complete_temp_exists() {
        let root = std::env::temp_dir().join(format!("ghostftp-stage-test-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let final_path = root.join("file.bin");
        let staging = download_staging_path(&final_path, "t1");
        tokio::fs::write(&final_path, b"old").await.unwrap();
        tokio::fs::write(&staging, b"new-complete").await.unwrap();

        promote_staged_download(&staging, &final_path, "t1")
            .await
            .unwrap();

        assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"new-complete");
        assert!(!staging.exists());
        assert!(!download_backup_path(&final_path, "t1").exists());
        let _ = tokio::fs::remove_dir_all(&root).await;
    }

    #[tokio::test]
    async fn staged_download_failure_before_promotion_leaves_original_untouched() {
        let root = std::env::temp_dir().join(format!("ghostftp-stage-test-{}", Uuid::new_v4()));
        tokio::fs::create_dir_all(&root).await.unwrap();
        let final_path = root.join("file.bin");
        let staging = download_staging_path(&final_path, "t2");
        tokio::fs::write(&final_path, b"old-safe").await.unwrap();
        tokio::fs::write(&staging, b"partial").await.unwrap();

        // Simulate a failed/canceled backend before promotion: only the staging
        // file is discarded by the runner; the visible destination is untouched.
        tokio::fs::remove_file(&staging).await.unwrap();

        assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"old-safe");
        assert!(!staging.exists());
        let _ = tokio::fs::remove_dir_all(&root).await;
    }

    #[tokio::test]
    async fn staged_download_paths_are_sibling_scoped_and_transfer_specific() {
        let final_path = PathBuf::from("/tmp/downloads/report.bin");
        assert_eq!(
            download_staging_path(&final_path, "abc"),
            PathBuf::from("/tmp/downloads/.ghostftp-download-abc.part")
        );
        assert_eq!(
            download_backup_path(&final_path, "abc"),
            PathBuf::from("/tmp/downloads/.ghostftp-download-abc.backup")
        );
    }

    // ---------- TokenBucket ----------

    #[tokio::test(start_paused = true)]
    async fn token_bucket_caps_throughput() {
        let bucket = TokenBucket::new();
        bucket.set_rate_kbps(512); // 512 KiB/s
        let start = tokio::time::Instant::now();
        // 1 MiB at 512 KiB/s must take ~2s of (virtual) time, charged in full.
        bucket.acquire(1024 * 1024).await;
        let elapsed = start.elapsed();
        assert!(
            elapsed >= Duration::from_millis(1900),
            "too fast: {elapsed:?}"
        );
        assert!(
            elapsed <= Duration::from_millis(2600),
            "too slow: {elapsed:?}"
        );
    }

    #[tokio::test(start_paused = true)]
    async fn token_bucket_unlimited_is_instant() {
        let bucket = TokenBucket::new(); // rate 0 = unlimited
        let start = tokio::time::Instant::now();
        bucket.acquire(10 * 1024 * 1024).await;
        assert_eq!(start.elapsed(), Duration::ZERO);
    }

    #[tokio::test(start_paused = true)]
    async fn token_bucket_tiny_file_skips_full_window() {
        let bucket = TokenBucket::new();
        bucket.set_rate_kbps(64); // 64 KiB/s
        let start = tokio::time::Instant::now();
        bucket.acquire(1024).await; // 1 KiB → ~16ms, not a whole window
        assert!(start.elapsed() < Duration::from_millis(100));
    }

    // ---------- PauseGate + checkpoint ----------

    #[tokio::test]
    async fn pause_gate_parks_until_opened() {
        let gate = PauseGate::new();
        gate.set(true);
        let g2 = gate.clone();
        let handle = tokio::spawn(async move { g2.wait_open().await });
        for _ in 0..10 {
            tokio::task::yield_now().await;
        }
        assert!(!handle.is_finished());
        gate.set(false);
        handle.await.unwrap();
    }

    #[tokio::test]
    async fn checkpoint_parks_then_signals_restart() {
        let mgr = Arc::new(TransferManager::new());
        mgr.pauses
            .lock()
            .await
            .insert("t1".into(), PauseGate::new());
        // Not paused → passes straight through.
        mgr.checkpoint("t1", 128).await.unwrap();

        mgr.pauses.lock().await.get("t1").unwrap().set(true);
        let m2 = Arc::clone(&mgr);
        let handle = tokio::spawn(async move { m2.checkpoint("t1", 128).await });
        // The parked task cannot finish before the gate opens (Ok needs an
        // open gate, Err needs the park loop to break) — so a finished handle
        // here is impossible regardless of scheduling.
        tokio::time::sleep(Duration::from_millis(50)).await;
        assert!(!handle.is_finished());
        mgr.pauses.lock().await.get("t1").unwrap().set(false);
        let err = handle.await.unwrap().unwrap_err();
        assert!(err.downcast_ref::<RestartFromPause>().is_some());
    }

    // ---------- Durable restart recovery ----------

    #[tokio::test]
    async fn persisted_failed_transfer_can_reattach_after_restart() {
        let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
        let envelope = PersistedTransferEnvelope {
            schema: TRANSFER_LEDGER_SCHEMA,
            transfer: Transfer {
                id: "failed-before-restart".to_string(),
                kind: TransferKind::Upload,
                source: "/tmp/source.bin".to_string(),
                destination: "/remote/source.bin".to_string(),
                size: 4096,
                transferred: 2048,
                status: TransferStatus::Error,
                error: Some("connection reset by peer".to_string()),
                retry_attempt: Some(2),
                delta: None,
                started_at: 10,
            },
            retry: PersistedRetryInfo::Upload {
                profile_id: "profile-retry".to_string(),
                local: PathBuf::from("/tmp/source.bin"),
                final_remote: "/remote/source.bin".to_string(),
            },
        };
        db.transfer_ledger_upsert(
            "failed-before-restart",
            &serde_json::to_string(&envelope).unwrap(),
        )
        .unwrap();

        let recovered = TransferManager::with_db(db);
        assert_eq!(
            recovered
                .recovery_profile_id("failed-before-restart")
                .await
                .as_deref(),
            Some("profile-retry")
        );
        let row = recovered.snapshot("failed-before-restart").await.unwrap();
        assert_eq!(row.status, TransferStatus::Error);
        assert_eq!(row.transferred, 2048);
        assert_eq!(row.error.as_deref(), Some("connection reset by peer"));
    }

    #[tokio::test]
    async fn durable_recovery_keeps_progress_for_display_but_requires_safe_restart() {
        let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
        let id = "recover-me";

        {
            let mgr = TransferManager::with_db(db.clone());
            mgr.persisted_retry.lock().await.insert(
                id.to_string(),
                PersistedRetryInfo::Download {
                    profile_id: "profile-1".to_string(),
                    remote_path: "/srv/archive.bin".to_string(),
                    final_path: PathBuf::from("/tmp/archive.bin"),
                },
            );
            mgr.insert(Transfer {
                id: id.to_string(),
                kind: TransferKind::Download,
                source: "/srv/archive.bin".to_string(),
                destination: "/tmp/archive.bin".to_string(),
                size: 1024 * 1024,
                transferred: 256 * 1024,
                status: TransferStatus::Transferring,
                error: None,
                retry_attempt: Some(1),
                delta: None,
                started_at: 123,
            })
            .await;

            let rows = db.transfer_ledger_list().unwrap();
            assert_eq!(rows.len(), 1);
            assert_eq!(rows[0].0, id);
            assert!(rows[0].1.contains("profile-1"));
            assert!(!rows[0].1.contains("password"));
            assert!(!rows[0].1.contains("passphrase"));
            assert!(!rows[0].1.contains("token"));
        }

        // A fresh manager simulates the next application process after a crash
        // or update. The row must not pretend to still be actively transferring.
        let recovered = TransferManager::with_db(db);
        let row = recovered.snapshot(id).await.unwrap();
        assert_eq!(row.status, TransferStatus::Error);
        assert_eq!(row.transferred, 256 * 1024);
        assert_eq!(row.retry_attempt, None);
        assert!(row
            .error
            .as_deref()
            .is_some_and(|message| message.contains("interrupted")));
        assert_eq!(
            recovered.recovery_profile_id(id).await.as_deref(),
            Some("profile-1")
        );
        assert!(recovered.recovered_retry.lock().await.contains(id));
        assert!(row
            .error
            .as_deref()
            .is_some_and(|message| message.contains("restart safely from byte zero")));
    }

    #[tokio::test]
    async fn terminal_history_is_pruned_during_runtime_without_dropping_active_rows() {
        let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
        let mgr = TransferManager::with_db(db.clone());

        mgr.persisted_retry.lock().await.insert(
            "active".to_string(),
            PersistedRetryInfo::Upload {
                profile_id: "p".to_string(),
                local: PathBuf::from("/tmp/active"),
                final_remote: "/active".to_string(),
            },
        );
        mgr.insert(Transfer {
            id: "active".to_string(),
            kind: TransferKind::Upload,
            source: "/tmp/active".to_string(),
            destination: "/active".to_string(),
            size: 10,
            transferred: 1,
            status: TransferStatus::Transferring,
            error: None,
            retry_attempt: None,
            delta: None,
            started_at: 1,
        })
        .await;

        for index in 0..=TRANSFER_LEDGER_KEEP {
            let id = format!("done-{index:04}");
            mgr.persisted_retry.lock().await.insert(
                id.clone(),
                PersistedRetryInfo::Download {
                    profile_id: "p".to_string(),
                    remote_path: format!("/remote/{index}"),
                    final_path: PathBuf::from(format!("/tmp/{index}")),
                },
            );
            mgr.insert(Transfer {
                id,
                kind: TransferKind::Download,
                source: format!("/remote/{index}"),
                destination: format!("/tmp/{index}"),
                size: 1,
                transferred: 1,
                status: TransferStatus::Done,
                error: None,
                retry_attempt: None,
                delta: None,
                started_at: index as i64 + 10,
            })
            .await;
        }

        let list = mgr.list().await;
        assert!(list.iter().any(|transfer| transfer.id == "active"));
        assert_eq!(
            list.iter()
                .filter(|transfer| {
                    matches!(
                        transfer.status,
                        TransferStatus::Done
                            | TransferStatus::Skipped
                            | TransferStatus::Error
                            | TransferStatus::Canceled
                    )
                })
                .count(),
            TRANSFER_LEDGER_KEEP
        );
        assert_eq!(
            db.transfer_ledger_list().unwrap().len(),
            TRANSFER_LEDGER_KEEP + 1
        );
    }

    // ---------- FIFO admission ----------

    #[tokio::test]
    async fn fifo_skips_paused_and_pause_all_blocks_everyone() {
        let mgr = TransferManager::new();
        {
            let mut w = mgr.waiting.lock().await;
            w.push_back("a".to_string());
            w.push_back("b".to_string());
            w.push_back("c".to_string());
        }
        mgr.pauses.lock().await.insert("a".into(), PauseGate::new());
        mgr.pauses.lock().await.insert("b".into(), PauseGate::new());
        {
            let w = mgr.waiting.lock().await;
            assert!(mgr.is_my_turn(&w, "a").await);
            assert!(!mgr.is_my_turn(&w, "b").await);
        }
        // A paused front row never head-of-line blocks the queue.
        mgr.pauses.lock().await.get("a").unwrap().set(true);
        {
            let w = mgr.waiting.lock().await;
            assert!(!mgr.is_my_turn(&w, "a").await);
            assert!(mgr.is_my_turn(&w, "b").await);
            assert!(!mgr.is_my_turn(&w, "c").await);
        }
        // Pause-all blocks everyone, runnable or not.
        mgr.pause_all.set(true);
        {
            let w = mgr.waiting.lock().await;
            assert!(!mgr.is_my_turn(&w, "b").await);
        }
    }

    #[tokio::test(start_paused = true)]
    async fn concurrency_grows_and_shrinks() {
        let mgr = TransferManager::new();
        assert_eq!(mgr.semaphore.available_permits(), DEFAULT_CONCURRENCY);
        mgr.set_concurrency(5);
        assert_eq!(mgr.semaphore.available_permits(), 5);
        mgr.set_concurrency(2);
        for _ in 0..10 {
            tokio::task::yield_now().await;
        }
        assert_eq!(mgr.semaphore.available_permits(), 2);
    }

    // ---------- Retry classification ----------

    #[test]
    fn transient_errors_retry_permanent_ones_dont() {
        assert!(is_transient(&anyhow::anyhow!("connection reset by peer")));
        assert!(is_transient(&anyhow::anyhow!("operation timed out")));
        assert!(!is_transient(&anyhow::anyhow!("authentication failed")));
        assert!(!is_transient(&anyhow::anyhow!(
            "Permission denied (os error 13)"
        )));
        assert!(!is_transient(&anyhow::anyhow!("No such file or directory")));
    }

    #[test]
    fn directory_upload_metadata_failure_is_not_silently_skipped() {
        let path = PathBuf::from("/source/private.txt");
        let error = require_directory_upload_metadata(
            &path,
            Err(std::io::Error::new(
                std::io::ErrorKind::PermissionDenied,
                "permission denied",
            )),
        )
        .unwrap_err();

        let message = format!("{error:#}");
        assert!(message.contains("stat directory upload entry"));
        assert!(message.contains("private.txt"));
        assert!(message.contains("permission denied"));
        assert!(message.contains("incomplete source tree"));
    }

    #[test]
    fn remote_join_follows_the_destination_style() {
        // POSIX destinations keep forward slashes, with or without a trailing one.
        assert_eq!(join_remote("/var/www", "a.txt"), "/var/www/a.txt");
        assert_eq!(join_remote("/var/www/", "a.txt"), "/var/www/a.txt");
        // A Windows agent target keeps backslashes instead of going mixed.
        assert_eq!(join_remote(r"C:\srv", "a.txt"), r"C:\srv\a.txt");
        assert_eq!(join_remote(r"C:\srv\", "a.txt"), r"C:\srv\a.txt");
        // Forward-slashed Windows paths are already unambiguous — leave them be.
        assert_eq!(join_remote("C:/srv", "a.txt"), "C:/srv/a.txt");
        assert_eq!(join_remote("", "a.txt"), "a.txt");
    }

    #[test]
    fn remote_rename_candidates_respect_hidden_files_and_path_style() {
        assert_eq!(
            remote_rename_candidate("/srv/archive.tar.gz", 1),
            "/srv/archive.tar_1.gz"
        );
        assert_eq!(remote_rename_candidate("/srv/.env", 1), "/srv/.env_1");
        assert_eq!(remote_rename_candidate(".env", 2), ".env_2");
        assert_eq!(
            remote_rename_candidate(r"C:\dir.v1\file.txt", 3),
            r"C:\dir.v1\file_3.txt"
        );
        assert_eq!(
            remote_rename_candidate(r"C:\dir.v1\file", 4),
            r"C:\dir.v1\file_4"
        );
    }

    #[tokio::test]
    async fn remote_rename_returns_original_only_after_proven_absence() {
        let mut probes = Vec::new();
        let resolved = resolve_remote_rename_with_probe("/srv/file.txt", |candidate| {
            probes.push(candidate);
            std::future::ready(Ok::<bool, std::io::Error>(false))
        })
        .await
        .unwrap();

        assert_eq!(resolved, "/srv/file.txt");
        assert_eq!(probes, vec!["/srv/file.txt"]);
    }

    #[tokio::test]
    async fn remote_rename_skips_collisions_until_a_proven_free_candidate() {
        let mut probes = Vec::new();
        let resolved = resolve_remote_rename_with_probe("/srv/file.txt", |candidate| {
            let exists = matches!(
                candidate.as_str(),
                "/srv/file.txt" | "/srv/file_1.txt" | "/srv/file_2.txt"
            );
            probes.push(candidate);
            std::future::ready(Ok::<bool, std::io::Error>(exists))
        })
        .await
        .unwrap();

        assert_eq!(resolved, "/srv/file_3.txt");
        assert_eq!(
            probes,
            vec![
                "/srv/file.txt",
                "/srv/file_1.txt",
                "/srv/file_2.txt",
                "/srv/file_3.txt"
            ]
        );
    }

    #[tokio::test]
    async fn remote_rename_probe_failure_is_fail_closed() {
        let mut probes = 0usize;
        let error = resolve_remote_rename_with_probe("/srv/file.txt", |_candidate| {
            probes += 1;
            std::future::ready(Err::<bool, std::io::Error>(std::io::Error::new(
                std::io::ErrorKind::PermissionDenied,
                "permission denied",
            )))
        })
        .await
        .unwrap_err();

        assert_eq!(probes, 1);
        assert!(
            error
                .to_string()
                .contains("unable to verify remote rename target"),
            "unexpected probe error: {error:#}"
        );
    }

    #[tokio::test]
    async fn remote_rename_candidate_probe_failure_is_fail_closed() {
        let mut probes = 0usize;
        let error = resolve_remote_rename_with_probe("/srv/file.txt", |_candidate| {
            probes += 1;
            let result = if probes == 1 {
                Ok(true)
            } else {
                Err(std::io::Error::new(
                    std::io::ErrorKind::ConnectionReset,
                    "connection reset",
                ))
            };
            std::future::ready(result)
        })
        .await
        .unwrap_err();

        assert_eq!(probes, 2);
        assert!(
            error
                .to_string()
                .contains("unable to verify remote rename candidate /srv/file_1.txt"),
            "unexpected candidate probe error: {error:#}"
        );
    }

    #[test]
    fn local_rename_exhaustion_fails_instead_of_overwriting() {
        let dir = std::env::temp_dir().join(format!(
            "ghostftp-rename-exhaustion-{}-{}",
            std::process::id(),
            Uuid::new_v4().simple()
        ));
        std::fs::create_dir_all(&dir).unwrap();

        let original = dir.join("collision.txt");
        std::fs::write(&original, b"original").unwrap();
        for i in 1..=MAX_RENAME_CANDIDATES {
            std::fs::write(dir.join(format!("collision_{i}.txt")), b"occupied").unwrap();
        }

        let error = resolve_local_rename(&original).unwrap_err();
        assert!(
            error.to_string().contains("no free rename target"),
            "unexpected exhaustion error: {error:#}"
        );
        assert_eq!(std::fs::read(&original).unwrap(), b"original");

        std::fs::remove_dir_all(dir).unwrap();
    }

    // ---------- Delta sync transfer paths ----------

    /// Deterministic pseudo-random bytes (xorshift64*), so tests need no
    /// fixtures or extra dev-deps.
    fn det_bytes(seed: u64, n: usize) -> Vec<u8> {
        let mut x = seed;
        let mut out = Vec::with_capacity(n);
        while out.len() < n {
            x ^= x >> 12;
            x ^= x << 25;
            x ^= x >> 27;
            out.extend_from_slice(&x.wrapping_mul(0x2545_F491_4F6C_DD1D).to_le_bytes());
        }
        out.truncate(n);
        out
    }

    fn test_dirs(tag: &str) -> PathBuf {
        static COUNTER: AtomicUsize = AtomicUsize::new(0);
        let dir = std::env::temp_dir().join(format!(
            "ghostftp-transfer-delta-{tag}-{}-{}",
            std::process::id(),
            COUNTER.fetch_add(1, Ordering::SeqCst)
        ));
        std::fs::create_dir_all(dir.join("local")).unwrap();
        std::fs::create_dir_all(dir.join("remote")).unwrap();
        dir
    }

    /// An AgentSession wired over localhost TCP to an in-process daemon
    /// request loop (`ghostftp_agentd::ops::handle`, writes allowed). With
    /// `reject_signature`, `Signature` requests error out — simulating a
    /// pre-delta-sync daemon for the fallback tests.
    async fn delta_test_session(reject_signature: bool) -> Arc<crate::session::AgentSession> {
        use ghostftp_agent_proto::identity::Identity;
        use ghostftp_agent_proto::msg::{Hello, Request, Response, PROTOCOL_VERSION};
        use ghostftp_agent_proto::{Auth, Role, SecureChannel};

        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        let daemon_id = Identity::generate().unwrap();
        let daemon_pub = daemon_id.public_bytes().unwrap();
        tokio::spawn(async move {
            loop {
                let Ok((stream, _)) = listener.accept().await else {
                    break;
                };
                let sk = match daemon_id.private_bytes() {
                    Ok(sk) => sk,
                    Err(_) => break,
                };
                tokio::spawn(async move {
                    let jobs = ghostftp_agentd::jobs::JobStore::new();
                    let policy = ghostftp_agentd::Policy {
                        allow_exec: false,
                        allow_write: true,
                    };
                    let mut ch = SecureChannel::establish(
                        stream,
                        Role::Responder,
                        &sk,
                        Auth::Paired {
                            expect_remote: None,
                        },
                    )
                    .await?;
                    let _hello: Hello = ch.recv().await?;
                    loop {
                        let req: Request = match ch.recv().await {
                            Ok(r) => r,
                            Err(_) => break,
                        };
                        let resp = if reject_signature && matches!(req, Request::Signature { .. }) {
                            Response::error("unknown op")
                        } else {
                            ghostftp_agentd::ops::handle(req, policy, &jobs).await
                        };
                        if ch.send(&resp).await.is_err() {
                            break;
                        }
                    }
                    Ok::<(), anyhow::Error>(())
                });
            }
        });

        let ctrl_id = Identity::generate().unwrap();
        let stream = tokio::net::TcpStream::connect(addr).await.unwrap();
        let mut ch = SecureChannel::establish(
            stream,
            Role::Initiator,
            &ctrl_id.private_bytes().unwrap(),
            Auth::Paired {
                expect_remote: Some(daemon_pub),
            },
        )
        .await
        .unwrap();
        ch.send(&Hello {
            protocol_version: PROTOCOL_VERSION,
            client_name: "transfer-test".into(),
        })
        .await
        .unwrap();
        Arc::new(crate::session::AgentSession::for_test(ch))
    }

    /// A manager with one transfer row registered, so the delta arms' update /
    /// checkpoint / emit calls have a row to work on.
    async fn delta_test_manager(
        id: &str,
        kind: TransferKind,
        source: &str,
        destination: &str,
        size: u64,
    ) -> TransferManager {
        let mgr = TransferManager::new();
        mgr.insert(Transfer {
            id: id.to_string(),
            kind,
            source: source.to_string(),
            destination: destination.to_string(),
            size,
            transferred: 0,
            status: TransferStatus::Queued,
            error: None,
            retry_attempt: None,
            delta: None,
            started_at: 0,
        })
        .await;
        mgr
    }

    // ---------- Delta sync setting switch + cross-backend gate ----------

    /// The `deltaSync` setting drives the switch; `GHOSTFTP_DELTA=0` force-off
    /// wins over it either way.
    #[test]
    fn delta_switch_follows_the_setting() {
        let mgr = TransferManager::new();
        assert!(mgr.delta_enabled(), "default on");
        mgr.set_delta_enabled(false);
        assert!(!mgr.delta_enabled());
        mgr.set_delta_enabled(true);
        let env_off = std::env::var("GHOSTFTP_DELTA").ok().as_deref() == Some("0");
        assert_eq!(mgr.delta_enabled(), !env_off);
    }

    fn delta_gate_profile(protocol: &str) -> crate::profiles::ConnectionProfile {
        crate::profiles::ConnectionProfile {
            id: "t".into(),
            name: "t".into(),
            protocol: protocol.into(),
            host: "127.0.0.1".into(),
            port: 22,
            username: "u".into(),
            auth: crate::profiles::AuthMethod::Agent,
            default_remote_path: None,
            description: None,
            color: None,
            auto_connect: None,
            bucket: None,
            region: None,
            endpoint: None,
            account: None,
            agent_key: None,
            group: None,
            favorite: None,
            bookmarked: None,
            tags: None,
            last_used: None,
            sort_order: None,
            icon: None,
            jump_host: None,
            jump_port: None,
            jump_username: None,
        }
    }

    /// Cross-backend contract: `supports_delta` is true ONLY for
    /// `Session::Agent` — every other backend must take the whole-file path.
    /// One table row per variant constructible without a live connection;
    /// Ssh/Ftp (live sockets) and the OAuth/API-token cloud sessions (private
    /// connect-time state) can't be built in a unit test — for those the
    /// exhaustive `matches!` in `supports_delta` is the compile-time
    /// guarantee, and dispatch routes them to plain `run_*` arms.
    #[tokio::test]
    async fn supports_delta_only_for_agent_sessions() {
        use crate::session::http::{HttpAuth, HttpMode};
        use crate::session::webdav::WebdavAuth;
        use crate::session::{HttpSession, ObjectSession, WebdavSession};

        let client = reqwest::Client::new();
        let base = url::Url::parse("https://example.invalid/dav/").unwrap();
        let table: Vec<(&str, Session, bool)> = vec![
            (
                "webdav",
                Session::Webdav(Arc::new(WebdavSession {
                    id: "t".into(),
                    profile: delta_gate_profile("webdav"),
                    client: client.clone(),
                    base: base.clone(),
                    auth: WebdavAuth::None,
                })),
                false,
            ),
            (
                "http",
                Session::Http(Arc::new(HttpSession {
                    id: "t".into(),
                    profile: delta_gate_profile("http"),
                    client,
                    base,
                    auth: HttpAuth::None,
                    mode: HttpMode::Listing,
                })),
                false,
            ),
            (
                "object",
                Session::Object(Arc::new(ObjectSession {
                    id: "t".into(),
                    profile: delta_gate_profile("s3"),
                    container: "b".into(),
                    store: Arc::new(object_store::memory::InMemory::new()),
                })),
                false,
            ),
            (
                "agent",
                Session::Agent(delta_test_session(false).await),
                true,
            ),
        ];
        for (name, session, expected) in &table {
            assert_eq!(supports_delta(session), *expected, "{name} backend");
        }
    }

    #[tokio::test]
    async fn agent_download_resumes_from_committed_offset() {
        let dir = test_dirs("resume-down");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let content = det_bytes(0xA11CE, 768 * 1024);
        let offset = 192 * 1024;

        std::fs::write(&remote, &content).unwrap();
        std::fs::write(&local, &content[..offset]).unwrap();

        let session = delta_test_session(false).await;
        let mgr = delta_test_manager(
            "resume-down",
            TransferKind::Download,
            &remote_s,
            &local.to_string_lossy(),
            content.len() as u64,
        )
        .await;
        mgr.update("resume-down", |t| t.transferred = offset as u64)
            .await;

        // Drive the same delta-aware entry point production dispatch uses. A
        // non-zero committed offset must bypass delta and continue ReadChunk
        // requests from the verified local prefix.
        mgr.agent_download_with_delta_core("resume-down", &session, &remote_s, &local, None)
            .await
            .unwrap();

        assert_eq!(std::fs::read(&local).unwrap(), content);
        assert_eq!(
            mgr.get("resume-down").await.unwrap().transferred,
            content.len() as u64
        );
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[tokio::test]
    async fn agent_upload_resumes_from_committed_offset() {
        let dir = test_dirs("resume-up");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let content = det_bytes(0xBEE5, 768 * 1024);
        let offset = 192 * 1024;

        std::fs::write(&local, &content).unwrap();
        std::fs::write(&remote, &content[..offset]).unwrap();

        let session = delta_test_session(false).await;
        let mgr = delta_test_manager(
            "resume-up",
            TransferKind::Upload,
            &local.to_string_lossy(),
            &remote_s,
            content.len() as u64,
        )
        .await;
        mgr.update("resume-up", |t| t.transferred = offset as u64)
            .await;

        // Production dispatch is delta-aware, but an interrupted whole-file
        // upload must keep its exact prefix and continue WriteChunk at offset.
        mgr.agent_upload_with_delta_core("resume-up", &session, &local, &remote_s, None)
            .await
            .unwrap();

        assert_eq!(std::fs::read(&remote).unwrap(), content);
        assert_eq!(
            mgr.get("resume-up").await.unwrap().transferred,
            content.len() as u64
        );
        let _ = std::fs::remove_dir_all(&dir);
    }

    /// Delta upload: 20 MiB up whole-file, mutate 1 KiB, up again — the second
    /// run must reassemble a byte-equal remote file with < 10% of the bytes
    /// crossing the wire (the patch is the ONLY WriteChunk traffic).
    #[tokio::test]
    async fn delta_upload_reuses_unchanged_blocks() {
        let dir = test_dirs("up");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let size = 20 * 1024 * 1024;
        std::fs::write(&local, det_bytes(0xAAAA, size)).unwrap();

        let session = delta_test_session(false).await;
        // First upload: no remote basis → whole-file.
        let mgr = delta_test_manager(
            "t",
            TransferKind::Upload,
            &local.to_string_lossy(),
            &remote_s,
            size as u64,
        )
        .await;
        mgr.agent_upload_with_delta_core("t", &session, &local, &remote_s, None)
            .await
            .unwrap();
        assert_eq!(
            std::fs::read(&remote).unwrap(),
            std::fs::read(&local).unwrap()
        );
        assert!(
            mgr.get("t").await.unwrap().delta.is_none(),
            "no basis → whole-file"
        );

        // Mutate 1 KiB in the middle and upload again → delta.
        let mut content = std::fs::read(&local).unwrap();
        content[10 * 1024 * 1024..10 * 1024 * 1024 + 1024]
            .copy_from_slice(&det_bytes(0xBBBB, 1024));
        std::fs::write(&local, &content).unwrap();
        mgr.update("t", |t| t.transferred = 0).await;
        mgr.agent_upload_with_delta_core("t", &session, &local, &remote_s, None)
            .await
            .unwrap();

        assert_eq!(std::fs::read(&remote).unwrap(), content);
        let t = mgr.get("t").await.unwrap();
        let stats = t.delta.expect("second upload must run as a delta");
        assert!(
            stats.sent * 10 < size as u64,
            "1 KiB edit sent {} bytes (>= 10% of {size})",
            stats.sent
        );
        assert_eq!(stats.sent + stats.reused, size as u64);
        // No temp litter on either side.
        for side in ["local", "remote"] {
            let litter: Vec<_> = std::fs::read_dir(dir.join(side))
                .unwrap()
                .filter_map(|e| e.ok())
                .filter(|e| {
                    let n = e.file_name().to_string_lossy().into_owned();
                    n.contains(".ghostftp-patch-")
                        || n.contains(".ghostftp-new-")
                        || n.contains(".ghostftp-delta-")
                })
                .collect();
            assert!(litter.is_empty(), "{side} litter: {litter:?}");
        }
        let _ = std::fs::remove_dir_all(&dir);
    }

    /// Delta download: same shape as the upload test, other direction.
    #[tokio::test]
    async fn delta_download_reuses_unchanged_blocks() {
        let dir = test_dirs("down");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let size = 20 * 1024 * 1024;
        std::fs::write(&remote, det_bytes(0xCCCC, size)).unwrap();

        let session = delta_test_session(false).await;
        // First download: no local basis → whole-file.
        let mgr = delta_test_manager(
            "t",
            TransferKind::Download,
            &remote_s,
            &local.to_string_lossy(),
            size as u64,
        )
        .await;
        mgr.agent_download_with_delta_core("t", &session, &remote_s, &local, None)
            .await
            .unwrap();
        assert_eq!(
            std::fs::read(&local).unwrap(),
            std::fs::read(&remote).unwrap()
        );
        assert!(mgr.get("t").await.unwrap().delta.is_none());

        // Mutate 1 KiB remotely and download again → delta.
        let mut content = std::fs::read(&remote).unwrap();
        content[5 * 1024 * 1024..5 * 1024 * 1024 + 1024].copy_from_slice(&det_bytes(0xDDDD, 1024));
        std::fs::write(&remote, &content).unwrap();
        mgr.update("t", |t| t.transferred = 0).await;
        mgr.agent_download_with_delta_core("t", &session, &remote_s, &local, None)
            .await
            .unwrap();

        assert_eq!(std::fs::read(&local).unwrap(), content);
        let t = mgr.get("t").await.unwrap();
        let stats = t.delta.expect("second download must run as a delta");
        assert!(
            stats.sent * 10 < size as u64,
            "1 KiB edit fetched {} bytes (>= 10% of {size})",
            stats.sent
        );
        assert_eq!(stats.sent + stats.reused, size as u64);
        let _ = std::fs::remove_dir_all(&dir);
    }

    /// Old daemon (Signature unsupported) ⇒ the delta attempt fails and the
    /// wrapper silently completes a correct whole-file copy, both directions.
    #[tokio::test]
    async fn delta_falls_back_when_daemon_lacks_signature() {
        let dir = test_dirs("old");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let size = 20 * 1024 * 1024;
        std::fs::write(&remote, det_bytes(0xEEEE, size)).unwrap();
        std::fs::write(&local, std::fs::read(&remote).unwrap()).unwrap();

        let session = delta_test_session(true).await; // Signature → error
                                                      // Upload direction: remote basis exists and file is big enough, so the
                                                      // delta is attempted, fails at Signature, and falls back.
        let mut content = std::fs::read(&local).unwrap();
        content[1024..2048].copy_from_slice(&det_bytes(0xFFFF, 1024));
        std::fs::write(&local, &content).unwrap();
        let mgr = delta_test_manager(
            "u",
            TransferKind::Upload,
            &local.to_string_lossy(),
            &remote_s,
            size as u64,
        )
        .await;
        mgr.agent_upload_with_delta_core("u", &session, &local, &remote_s, None)
            .await
            .unwrap();
        assert_eq!(std::fs::read(&remote).unwrap(), content);
        assert!(
            mgr.get("u").await.unwrap().delta.is_none(),
            "fallback → no delta stats"
        );

        // Download direction.
        content[4096..5120].copy_from_slice(&det_bytes(0x1234, 1024));
        std::fs::write(&remote, &content).unwrap();
        let mgr = delta_test_manager(
            "d",
            TransferKind::Download,
            &remote_s,
            &local.to_string_lossy(),
            size as u64,
        )
        .await;
        mgr.agent_download_with_delta_core("d", &session, &remote_s, &local, None)
            .await
            .unwrap();
        assert_eq!(std::fs::read(&local).unwrap(), content);
        assert!(mgr.get("d").await.unwrap().delta.is_none());
        let _ = std::fs::remove_dir_all(&dir);
    }

    /// ≥ 60% literal ⇒ the worthwhile heuristic aborts the delta before any
    /// patch byte is sent and the wrapper falls back to a correct whole-file
    /// copy.
    #[tokio::test]
    async fn delta_falls_back_when_mostly_changed() {
        let dir = test_dirs("churn");
        let local = dir.join("local/file.bin");
        let remote = dir.join("remote/file.bin");
        let remote_s = remote.to_string_lossy().into_owned();
        let size = 20 * 1024 * 1024;
        std::fs::write(&remote, det_bytes(0x7777, size)).unwrap();
        // Same size, ~completely different content ⇒ ~100% literal.
        std::fs::write(&local, det_bytes(0x8888, size)).unwrap();

        let session = delta_test_session(false).await;
        let mgr = delta_test_manager(
            "u",
            TransferKind::Upload,
            &local.to_string_lossy(),
            &remote_s,
            size as u64,
        )
        .await;
        mgr.agent_upload_with_delta_core("u", &session, &local, &remote_s, None)
            .await
            .unwrap();
        assert_eq!(
            std::fs::read(&remote).unwrap(),
            std::fs::read(&local).unwrap()
        );
        assert!(
            mgr.get("u").await.unwrap().delta.is_none(),
            "≥60% literal → whole-file"
        );

        // Download direction, same setup reversed.
        std::fs::write(&remote, det_bytes(0x9999, size)).unwrap();
        let mgr = delta_test_manager(
            "d",
            TransferKind::Download,
            &remote_s,
            &local.to_string_lossy(),
            size as u64,
        )
        .await;
        mgr.agent_download_with_delta_core("d", &session, &remote_s, &local, None)
            .await
            .unwrap();
        assert_eq!(
            std::fs::read(&local).unwrap(),
            std::fs::read(&remote).unwrap()
        );
        assert!(mgr.get("d").await.unwrap().delta.is_none());
        let _ = std::fs::remove_dir_all(&dir);
    }
}
