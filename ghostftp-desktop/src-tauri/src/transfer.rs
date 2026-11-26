// Ghost FTP by Brendigo
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
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex as StdMutex};
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

/// An FTP/SFTP server may have renamed the staged file even when the client
/// lost the final acknowledgment. Automatic replay could overwrite a newer
/// competing write. Leave the outcome uncertain until a user verifies it.
#[derive(Debug)]
struct UploadPromotionUnconfirmed(anyhow::Error);

impl std::fmt::Display for UploadPromotionUnconfirmed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("remote upload promotion could not be confirmed")
    }
}

impl std::error::Error for UploadPromotionUnconfirmed {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        Some(self.0.as_ref())
    }
}

/// Transfer priority retained across explicit retry and process recovery.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TransferPriority {
    Low,
    #[default]
    Normal,
    High,
}

// Weighted round-robin by admission, not by bytes or elapsed time. The
// scheduler skips empty tiers, so available slots are never left idle.
const QUEUE_TIER_CYCLE: [usize; 7] = [2, 2, 2, 2, 1, 1, 0];

fn next_queue_tier(cursor: usize, available: [bool; 3]) -> Option<(usize, usize)> {
    for offset in 0..QUEUE_TIER_CYCLE.len() {
        let index = (cursor + offset) % QUEUE_TIER_CYCLE.len();
        let tier = QUEUE_TIER_CYCLE[index];
        if available[tier] {
            return Some((tier, (index + 1) % QUEUE_TIER_CYCLE.len()));
        }
    }
    None
}

/// Payload of the `transfer://queue` event: the ordered waiting transfer ids
/// plus the manager-level state the panel header renders.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct QueueState {
    pub waiting: Vec<String>,
    pub priorities: HashMap<String, TransferPriority>,
    pub paused_all: bool,
    pub concurrency: usize,
    pub throttle_kbps: u64,
}

/// Reject source files that grew or shrank while a cloud transfer was
/// streaming. A successful HTTP/multipart response is not proof that all
/// bytes from the original file were actually uploaded.
fn require_exact_upload_length(expected: u64, actual: u64) -> Result<()> {
    anyhow::ensure!(
        expected == actual,
        "upload source changed during transfer (expected {expected} bytes, read {actual})"
    );
    Ok(())
}

/// Compare every byte of a previously committed transfer prefix before
/// resuming a seekable SFTP stream. File length alone cannot prove that the
/// source or destination still contains the same data after a pause.
/// This is O(prefix length) I/O but uses constant memory and fails closed.
async fn byte_equal_prefix<L, R>(local: &mut L, remote: &mut R, bytes: u64) -> Result<bool>
where
    L: tokio::io::AsyncRead + Unpin,
    R: tokio::io::AsyncRead + Unpin,
{
    let mut left = [0u8; 64 * 1024];
    let mut right = [0u8; 64 * 1024];
    let mut remaining = bytes;
    while remaining > 0 {
        let count = remaining.min(left.len() as u64) as usize;
        local
            .read_exact(&mut left[..count])
            .await
            .context("read local resume prefix")?;
        remote
            .read_exact(&mut right[..count])
            .await
            .context("read remote resume prefix")?;
        if left[..count] != right[..count] {
            return Ok(false);
        }
        remaining -= count as u64;
    }
    Ok(true)
}

/// Compare the complete committed prefix with the Ghost FTP Agent daemon before
/// resuming either an upload or a download. A changed file of identical length
/// must not silently yield a hybrid of two different revisions.
async fn agent_remote_prefix_matches(
    session: &Arc<crate::session::AgentSession>,
    path: &str,
    local_path: &Path,
    bytes: u64,
) -> Result<bool> {
    let mut local = tokio::fs::File::open(local_path)
        .await
        .context("open Agent resume source")?;
    agent_remote_prefix_matches_file(session, path, &mut local, bytes).await
}

async fn agent_remote_prefix_matches_file(
    session: &Arc<crate::session::AgentSession>,
    path: &str,
    local: &mut tokio::fs::File,
    bytes: u64,
) -> Result<bool> {
    use base64::Engine as _;
    use ghostftp_agent_proto::msg::{Request, Response};

    local.seek(SeekFrom::Start(0)).await?;
    let mut offset = 0u64;
    let mut buffer = [0u8; 64 * 1024];
    while offset < bytes {
        let request_len = (bytes - offset).min(buffer.len() as u64);
        let response = session
            .request(Request::ReadChunk {
                path: path.to_string(),
                offset,
                len: request_len,
            })
            .await
            .context("verify Agent resume prefix")?;
        let encoded = match response {
            Response::Chunk { data, .. } => data,
            Response::Error { .. } => anyhow::bail!("Agent refused resume-prefix verification"),
            _ => anyhow::bail!("unexpected Agent resume-prefix response"),
        };
        let chunk = base64::engine::general_purpose::STANDARD
            .decode(encoded)
            .context("decode Agent resume-prefix response")?;
        if chunk.is_empty() || chunk.len() as u64 > request_len {
            return Ok(false);
        }
        local
            .read_exact(&mut buffer[..chunk.len()])
            .await
            .context("read Agent local prefix")?;
        if buffer[..chunk.len()] != chunk {
            return Ok(false);
        }
        offset += chunk.len() as u64;
    }
    Ok(true)
}

/// Use only for providers that still require an in-memory request body.
/// Prevent an unexpected multi-GiB file or a file growing after stat from
/// crashing the desktop process. The 32 MiB bound is an explicit limitation,
/// not a claim of large-file support from those provider APIs.
const MAX_BUFFERED_CLOUD_UPLOAD: u64 = 32 * 1024 * 1024;

async fn read_cloud_upload_bounded(path: &Path, expected: u64) -> Result<Vec<u8>> {
    anyhow::ensure!(
        expected <= MAX_BUFFERED_CLOUD_UPLOAD,
        "This cloud adapter requires a resumable large-file upload for files above 32 MiB"
    );
    let file = tokio::fs::File::open(path)
        .await
        .with_context(|| format!("open {}", path.display()))?;
    let mut input = file.take(expected + 1);
    let mut data = Vec::with_capacity(expected as usize);
    input.read_to_end(&mut data).await?;
    require_exact_upload_length(expected, data.len() as u64)?;
    Ok(data)
}

/// Shared byte budget for copy loops that call the transfer checkpoint.
/// A one-second burst ceiling also applies at low configured rates.
struct TokenBucket {
    inner: StdMutex<BucketInner>,
}

struct BucketInner {
    tokens: f64,
    rate_bps: u64,
    // tokio's Instant (not std's) so `start_paused` tests drive refills.
    last: tokio::time::Instant,
}

impl BucketInner {
    fn refill(&mut self, now: tokio::time::Instant) {
        let elapsed = now.duration_since(self.last).as_secs_f64();
        let rate = self.rate_bps as f64;
        self.tokens = (self.tokens + elapsed * rate).min(rate);
        self.last = now;
    }
}

impl TokenBucket {
    fn new() -> Self {
        Self {
            inner: StdMutex::new(BucketInner {
                tokens: 0.0,
                rate_bps: 0,
                last: tokio::time::Instant::now(),
            }),
        }
    }

    fn lock(&self) -> std::sync::MutexGuard<'_, BucketInner> {
        self.inner
            .lock()
            .unwrap_or_else(|poison| poison.into_inner())
    }

    fn rate_kbps(&self) -> u64 {
        self.lock().rate_bps / 1024
    }

    fn set_rate_kbps(&self, kbps: u64) {
        let mut state = self.lock();
        // Settle time at the previous rate before applying a new limit.
        // Unlimited time must not accumulate credit at the next finite rate.
        state.refill(tokio::time::Instant::now());
        state.rate_bps = kbps.saturating_mul(1024);
        state.tokens = state.tokens.min(state.rate_bps as f64);
    }

    /// Shared accounting step for async and blocking protocol adapters.
    /// The lock is released before either caller sleeps.
    fn debit_step(&self, remaining: &mut f64) -> Duration {
        let mut state = self.lock();
        let rate = state.rate_bps as f64;
        if rate == 0.0 {
            *remaining = 0.0;
            return Duration::ZERO;
        }
        state.refill(tokio::time::Instant::now());
        let tranche = remaining.min(rate);
        if state.tokens >= tranche {
            state.tokens -= tranche;
            *remaining -= tranche;
            return Duration::ZERO;
        }
        Duration::from_secs_f64((tranche - state.tokens) / rate).min(Duration::from_millis(250))
    }

    async fn acquire(&self, bytes: u64) {
        let mut remaining = bytes as f64;
        while remaining > 0.0 {
            let wait = self.debit_step(&mut remaining);
            if !wait.is_zero() {
                tokio::time::sleep(wait).await;
            }
        }
    }

    /// Called only on FTP's blocking worker. Poll pause/cancel and live rate
    /// changes between bounded sleeps; never block a Tokio executor thread.
    fn acquire_ftp<F>(&self, bytes: u64, mut control: F) -> FtpTransferControl
    where
        F: FnMut() -> FtpTransferControl,
    {
        let mut remaining = bytes as f64;
        loop {
            let decision = control();
            if decision != FtpTransferControl::Continue || remaining <= 0.0 {
                return decision;
            }
            let wait = self.debit_step(&mut remaining);
            if !wait.is_zero() {
                std::thread::sleep(wait);
            }
        }
    }
}

/// One FTP attempt, using the same budget as asynchronous transfers.
struct FtpBudget {
    bucket: Arc<TokenBucket>,
    pause_all: PauseGate,
    pause: Option<PauseGate>,
    canceled: Arc<AtomicBool>,
    accounted: u64,
}

impl FtpBudget {
    fn control(&mut self, transferred: u64) -> FtpTransferControl {
        let bytes = transferred.saturating_sub(self.accounted);
        self.accounted = transferred;
        self.bucket.acquire_ftp(bytes, || {
            if self.canceled.load(Ordering::Acquire) {
                FtpTransferControl::Cancel
            } else if self.pause_all.is_paused()
                || self.pause.as_ref().is_some_and(|gate| gate.is_paused())
            {
                FtpTransferControl::Pause
            } else {
                FtpTransferControl::Continue
            }
        })
    }
}

/// Preserve the established full-upload recovery path while charging bytes
/// before handing them to SuppaFTP's writer. An interrupted read is an error,
/// never EOF, so a partially written remote staging file cannot look complete.
struct BudgetedFtpReader<R> {
    inner: R,
    budget: FtpBudget,
    stop: FtpTransferControl,
}

impl<R: std::io::Read> std::io::Read for BudgetedFtpReader<R> {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        self.stop = self.budget.control(self.budget.accounted);
        if self.stop == FtpTransferControl::Continue {
            let count = self.inner.read(buf)?;
            self.stop = self
                .budget
                .control(self.budget.accounted.saturating_add(count as u64));
            if self.stop == FtpTransferControl::Continue {
                return Ok(count);
            }
        }
        Err(std::io::Error::other("FTP recovery upload interrupted"))
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
    /// Auto-retry round in progress, for the panel's
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
        /// An explicit Overwrite policy, never inferred from a stale preflight.
        allow_replace: bool,
    },
    Upload {
        session: Arc<Session>,
        local: PathBuf,
        final_remote: String,
        /// Only an explicitly selected Overwrite allows replacing an existing file.
        allow_replace: bool,
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
        /// Legacy saved rows lack a policy: default to non-overwrite to
        /// preserve local files rather than silently authorizing replacement.
        #[serde(default)]
        allow_replace: bool,
    },
    Upload {
        profile_id: String,
        local: PathBuf,
        final_remote: String,
        /// Legacy retry records must never infer overwrite permission.
        #[serde(default)]
        allow_replace: bool,
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
                allow_replace,
                ..
            } => RetryInfo::Download {
                session,
                remote_path: remote_path.clone(),
                final_path: final_path.clone(),
                allow_replace: *allow_replace,
            },
            Self::Upload {
                local,
                final_remote,
                allow_replace,
                ..
            } => RetryInfo::Upload {
                session,
                local: local.clone(),
                final_remote: final_remote.clone(),
                allow_replace: *allow_replace,
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
    #[serde(default)]
    priority: TransferPriority,
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

/// A transfer slot can retire one *in-use* permit on release when the user
/// lowers the limit. Unlike a spawned acquire_many, this debt can be cancelled
/// immediately when the limit is raised again, without stale shrink tasks.
struct TransferSlot {
    permit: Option<OwnedSemaphorePermit>,
    state: Arc<StdMutex<ConcurrencyState>>,
}

#[derive(Default)]
struct ConcurrencyState {
    pending_shrink: usize,
}

impl Drop for TransferSlot {
    fn drop(&mut self) {
        let mut state = self
            .state
            .lock()
            .unwrap_or_else(|poison| poison.into_inner());
        if state.pending_shrink > 0 {
            state.pending_shrink -= 1;
            if let Some(permit) = self.permit.take() {
                permit.forget();
            }
        }
    }
}

pub struct TransferManager {
    transfers: Mutex<HashMap<String, Transfer>>,
    tasks: Mutex<HashMap<String, JoinHandle<()>>>,
    /// FIFO within each priority tier for transfer ids waiting for a slot. An id is popped only
    /// once it is at the front, the pause-all gate is open, and a concurrency
    /// permit is available — until then it waits as `Queued`.
    waiting: Mutex<VecDeque<String>>,
    // Non-default priorities for retained transfer rows, including retry history.
    // Queue mutations hold waiting; ledger writes hold ledger_serial. No guard across await.
    queue_priorities: StdMutex<HashMap<String, TransferPriority>>,
    // Read and advance only under the waiting lock, on successful admission.
    queue_cursor: AtomicUsize,
    semaphore: Arc<Semaphore>,
    concurrency: AtomicUsize,
    concurrency_state: Arc<StdMutex<ConcurrencyState>>,
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
    /// Order mutation and SQLite persistence as one logical operation. Without
    /// this gate an older progress snapshot can be persisted after a terminal
    /// Done/Canceled record, making crash recovery falsely show active work.
    ledger_serial: Mutex<()>,
    /// Prevent simultaneous Retry actions from aborting each other's workers
    /// or enqueueing the same transfer twice.
    retry_serial: Mutex<()>,
    /// Throttle progress persistence to avoid turning a fast transfer into a
    /// high-frequency SQLite writer. State transitions still flush immediately.
    persisted_at: Mutex<HashMap<String, Instant>>,
    /// Bumped on every queue change so admission waiters re-check their turn.
    queue_gen: watch::Sender<u64>,
    /// Global bandwidth cap every copy loop draws from per chunk.
    bucket: Arc<TokenBucket>,
    /// Maximum automatic retries for transient network/timeout failures.
    max_auto_retries: AtomicUsize,
    /// Delta-sync master switch: the `deltaSync` setting,
    /// live-adjustable like the concurrency bound. `GHOSTFTP_DELTA=0` still
    /// force-disables regardless of this flag.
    delta_enabled: AtomicBool,
}

fn is_terminal_transfer_status(status: TransferStatus) -> bool {
    matches!(
        status,
        TransferStatus::Done
            | TransferStatus::Skipped
            | TransferStatus::Error
            | TransferStatus::Canceled
    )
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

/// A completed transfer is not proven by EOF or a success response alone.
/// Prevent truncation/oversized streams from being marked Done and promoted.
fn verify_committed_length(
    label: &str,
    expected: u64,
    transferred: u64,
    on_destination: u64,
) -> Result<()> {
    anyhow::ensure!(
        expected == transferred && transferred == on_destination,
        "{label} length mismatch: expected {expected}, received {transferred}, destination {on_destination}"
    );
    Ok(())
}

/// No remote-controlled diagnostics are persisted in the transfer ledger or
/// emitted to the UI. A server can echo OAuth tokens, signed download URLs,
/// Authorization headers or local file paths in an otherwise useful error.
fn safe_transfer_failure_message(message: &str) -> &'static str {
    use crate::error::ErrorKind;
    match crate::error::classify_message(message) {
        ErrorKind::Auth => "Authentication failed. Verify the connection credentials.",
        ErrorKind::Permission => "Permission denied by the destination or server.",
        ErrorKind::NotFound => "Source or destination was not found.",
        ErrorKind::Network => "Network connection was interrupted. Check the server and retry.",
        ErrorKind::Timeout => "Server timed out. Check the connection and retry.",
        ErrorKind::Unsupported => "This transfer operation is unsupported by the server.",
        ErrorKind::Conflict => "File conflict detected. Review the overwrite policy.",
        ErrorKind::InvalidInput => "Invalid input: check the file path or transfer request.",
        ErrorKind::Other => "Transfer failed. Check the server and retry.",
    }
}

fn safe_transfer_failure(error: &anyhow::Error) -> &'static str {
    if error.downcast_ref::<UploadPromotionUnconfirmed>().is_some() {
        return "Remote upload completion is uncertain. Verify the destination before retrying.";
    }
    safe_transfer_failure_message(&error.to_string())
}

// Preserve the intentionally actionable own-process recovery status verbatim.
// Unlike a remote error, this is a fixed internal literal with no secrets.
const INTERRUPTED_TRANSFER_MESSAGE: &str = "Transfer was interrupted by the previous Ghost FTP process. Retry will reconnect the saved profile and restart safely from byte zero because cross-process file identity was not verified.";

mod integrity;
use self::integrity::{
    checked_http_download_progress, checked_object_download_progress, completed_transfer_size,
    declared_full_http_download_length, verify_plain_http_download_length,
};

mod download_staging;
mod remote_upload_staging;
use download_staging::{
    cleanup_failed_download_staging, cleanup_owned_regular_staging_file,
    create_new_delta_upload_patch, create_new_download_staging, download_staging_path,
    fresh_download_staging_path, local_path_exists_fail_closed, local_path_exists_sync_fail_closed,
    open_owned_resumable_download_staging, DownloadParentGuard, DownloadStageOwner,
};
#[cfg(test)]
use download_staging::{
    download_backup_path, promote_staged_download, promote_staged_download_noreplace,
    verify_staged_download_bytes,
};
use remote_upload_staging::{
    capture_local_upload_snapshot, fresh_remote_upload_staging_path, preflight_ftp_upload_staging,
    promote_remote_upload, verify_ftp_upload_staging_size, verify_local_upload_snapshot,
    LocalUploadSnapshot,
};

mod remote_directory;
use remote_directory::ensure_upload_directory;

mod path_safety;
pub(crate) use path_safety::require_directory_upload_metadata;
use path_safety::{
    checked_remote_download_relative, ensure_download_subdirectory, inspect_directory_upload_entry,
    safe_local_download_leaf, safe_upload_name, validate_directory_download_budget,
    validate_directory_upload_budget,
};

const MAX_RENAME_CANDIDATES: usize = 999;

/// Append _1, _2, … to the stem until a free local path is found. Never fall
/// back to the colliding original path: Rename must not silently become Overwrite.
fn resolve_local_rename(path: &Path) -> Result<PathBuf> {
    if !local_path_exists_sync_fail_closed(path)? {
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
        if !local_path_exists_sync_fail_closed(&candidate)? {
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

/// A malicious or faulty Agent must never send more delta patch bytes than
/// the requested range, or silently end a nonempty range with zero bytes.
/// A successful Agent patch upload must acknowledge *all* bytes in the
/// current request. Missing or over-counted acknowledgements cause a safe
/// fallback instead of reporting successful transfer of incomplete data.
/// A daemon reply alone does not prove that a delta update reached disk.
/// Revalidate its reused-byte and reconstructed-size accounting against a fresh
/// stat before marking the upload complete.
fn verify_agent_delta_upload_completion(
    expected_total: u64,
    expected_reused: u64,
    reported_reused: u64,
    reported_written: u64,
    remote_size: u64,
) -> Result<()> {
    anyhow::ensure!(
        reported_reused == expected_reused,
        "Agent delta upload reused-byte acknowledgement mismatch"
    );
    verify_committed_length(
        "Agent delta upload",
        expected_total,
        reported_written,
        remote_size,
    )
}

fn verify_agent_delta_patch_ack(expected: usize, acknowledged: u64) -> Result<()> {
    let expected = u64::try_from(expected).context("Agent delta patch length overflow")?;
    anyhow::ensure!(
        acknowledged == expected,
        "Agent delta patch acknowledged an incomplete or oversized chunk"
    );
    Ok(())
}

fn checked_agent_delta_chunk_length(remaining: u64, next_bytes: usize) -> Result<u64> {
    let count = u64::try_from(next_bytes).context("Agent delta chunk length overflow")?;
    anyhow::ensure!(
        count > 0 && count <= remaining,
        "Agent delta chunk is empty or exceeds the requested range"
    );
    Ok(count)
}

/// Run bounded read-only recovery diagnostics off the desktop startup thread.
pub(crate) fn reconcile_download_recovery(db: &crate::db::Db) {
    if download_staging::reconcile_pending_recovery(db).is_err() {
        tracing::warn!("download recovery reconciliation could not be completed");
    }
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
        let mut queue_priorities = HashMap::new();
        let mut recovered_retry = HashSet::new();

        if let Some(store) = db.as_ref() {
            // A recovery intent is evidence, never authorization to delete or
            // restore a path after the process has lost its original handles.
            match store.download_recovery_list() {
                Ok(rows) if !rows.is_empty() => {
                    tracing::warn!(count = rows.len(), "unresolved download recovery intents");
                }
                Err(_) => tracing::warn!("download recovery intents could not be loaded"),
                _ => {}
            }
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

                    // Historical versions persisted untrusted backend errors
                    // verbatim. Redact them on the first load, persist the safe
                    // replacement and do not repeat that write on every start.
                    // Keep our own fixed crash-recovery instruction unchanged.
                    let mut old_diagnostic_rewritten = false;
                    if let Some(original) = envelope.transfer.error.as_deref() {
                        if original != INTERRUPTED_TRANSFER_MESSAGE {
                            let safe = safe_transfer_failure_message(original);
                            if original != safe {
                                envelope.transfer.error = Some(safe.to_string());
                                old_diagnostic_rewritten = true;
                            }
                        }
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
                        envelope.transfer.error = Some(INTERRUPTED_TRANSFER_MESSAGE.to_string());
                    }
                    if was_active || old_diagnostic_rewritten {
                        if let Ok(normalized) = serde_json::to_string(&envelope) {
                            let _ = store.transfer_ledger_upsert(&row_id, &normalized);
                        }
                    }

                    if needs_session_reattach {
                        recovered_retry.insert(row_id.clone());
                    }
                    if envelope.priority != TransferPriority::Normal {
                        queue_priorities.insert(row_id.clone(), envelope.priority);
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
            queue_priorities: StdMutex::new(queue_priorities),
            queue_cursor: AtomicUsize::new(0),
            semaphore: Arc::new(Semaphore::new(DEFAULT_CONCURRENCY)),
            concurrency: AtomicUsize::new(DEFAULT_CONCURRENCY),
            concurrency_state: Arc::new(StdMutex::new(ConcurrencyState::default())),
            pause_all: PauseGate::new(),
            pauses: Mutex::new(HashMap::new()),
            cancels: Mutex::new(HashMap::new()),
            retry: Mutex::new(HashMap::new()),
            persisted_retry: Mutex::new(persisted_retry),
            recovered_retry: Mutex::new(recovered_retry),
            db,
            ledger_serial: Mutex::new(()),
            retry_serial: Mutex::new(()),
            persisted_at: Mutex::new(HashMap::new()),
            queue_gen: watch::channel(0).0,
            bucket: Arc::new(TokenBucket::new()),
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

    /// Progress and control updates can only modify an active row. A late
    /// transport callback must never turn a terminal record back into a live
    /// transfer, or replace its final byte count. Manual Retry has an explicit
    /// terminal-to-Queued transition below.
    async fn update<F: FnOnce(&mut Transfer)>(&self, id: &str, f: F) {
        let _ledger = self.ledger_serial.lock().await;
        let snapshot = {
            let mut transfers = self.transfers.lock().await;
            transfers.get_mut(id).and_then(|transfer| {
                if is_terminal_transfer_status(transfer.status) {
                    return None;
                }
                f(transfer);
                Some(transfer.clone())
            })
        };
        if let Some(transfer) = snapshot {
            let force = transfer.status != TransferStatus::Transferring
                || transfer.transferred >= transfer.size;
            self.persist_snapshot(transfer, force).await;
        }
    }

    /// Only explicit user Retry may requeue a canceled/failed transfer. The
    /// status check and reset are atomic with respect to other state updates.
    async fn reset_for_retry(&self, id: &str) -> Result<()> {
        let _ledger = self.ledger_serial.lock().await;
        let snapshot = {
            let mut transfers = self.transfers.lock().await;
            let transfer = transfers
                .get_mut(id)
                .ok_or_else(|| anyhow::anyhow!("transfer {id} not found"))?;
            anyhow::ensure!(
                matches!(
                    transfer.status,
                    TransferStatus::Error | TransferStatus::Canceled
                ),
                "only failed or canceled transfers can be retried"
            );
            transfer.status = TransferStatus::Queued;
            transfer.transferred = 0;
            transfer.error = None;
            transfer.retry_attempt = None;
            transfer.clone()
        };
        self.persist_snapshot(snapshot, true).await;
        Ok(())
    }

    /// Commit cancellation to the same serialized state/ledger pipeline as
    /// finish_transfer. Otherwise a late Done can land between the cooperative
    /// flag and the separate Canceled status write.
    async fn mark_canceled(&self, id: &str) -> Option<Transfer> {
        let _ledger = self.ledger_serial.lock().await;
        let token = self.cancels.lock().await.get(id).cloned();
        let snapshot = {
            let mut transfers = self.transfers.lock().await;
            let transfer = transfers.get_mut(id)?;
            if !matches!(
                transfer.status,
                TransferStatus::Queued | TransferStatus::Transferring | TransferStatus::Paused
            ) {
                return None;
            }
            if let Some(flag) = token {
                flag.store(true, Ordering::Release);
            }
            transfer.status = TransferStatus::Canceled;
            transfer.clone()
        };
        self.persist_snapshot(snapshot.clone(), true).await;
        Some(snapshot)
    }

    /// A successful backend return can still contain an incomplete stream.
    /// Choose and persist the terminal status while holding one transition lock
    /// so a concurrent Cancel cannot be overwritten by a late completion.
    async fn finish_transfer(&self, id: &str, result: Result<()>) -> Option<Transfer> {
        let _ledger = self.ledger_serial.lock().await;
        let snapshot = {
            let mut transfers = self.transfers.lock().await;
            let transfer = transfers.get_mut(id)?;
            if is_terminal_transfer_status(transfer.status) {
                return None;
            }
            let completion = match result {
                Ok(()) => completed_transfer_size(transfer.size, transfer.transferred).map(|_| ()),
                Err(error) => Err(error),
            };
            match completion {
                Ok(()) => {
                    transfer.status = TransferStatus::Done;
                    if transfer.size == 0 {
                        transfer.size = transfer.transferred;
                    }
                }
                Err(error) => {
                    transfer.status = TransferStatus::Error;
                    transfer.error = Some(safe_transfer_failure(&error).to_string());
                }
            }
            transfer.clone()
        };
        self.persist_snapshot(snapshot.clone(), true).await;
        Some(snapshot)
    }

    async fn insert(&self, t: Transfer) {
        let _ledger = self.ledger_serial.lock().await;
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

        let priority = self
            .priority_overrides()
            .get(&transfer.id)
            .copied()
            .unwrap_or_default();
        let envelope = PersistedTransferEnvelope {
            schema: TRANSFER_LEDGER_SCHEMA,
            transfer,
            retry,
            priority,
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
                self.priority_overrides().remove(id);
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

    fn priority_overrides(&self) -> std::sync::MutexGuard<'_, HashMap<String, TransferPriority>> {
        self.queue_priorities
            .lock()
            .unwrap_or_else(|poison| poison.into_inner())
    }

    async fn build_queue_state(&self, waiting: &VecDeque<String>) -> QueueState {
        let pauses = self.pauses.lock().await;
        let overrides = self.priority_overrides();
        let priorities: HashMap<_, _> = waiting
            .iter()
            .map(|id| (id.clone(), overrides.get(id).copied().unwrap_or_default()))
            .collect();
        let mut tiers: [VecDeque<String>; 3] = std::array::from_fn(|_| VecDeque::new());
        let mut parked = Vec::new();
        for id in waiting {
            if pauses.get(id).is_some_and(|gate| gate.is_paused()) {
                parked.push(id.clone());
            } else {
                tiers[priorities[id] as usize].push_back(id.clone());
            }
        }
        let mut ordered = Vec::with_capacity(waiting.len());
        let mut cursor = self.queue_cursor.load(Ordering::Relaxed);
        while let Some((tier, next)) =
            next_queue_tier(cursor, std::array::from_fn(|i| !tiers[i].is_empty()))
        {
            if let Some(id) = tiers[tier].pop_front() {
                ordered.push(id);
            }
            cursor = next;
        }
        ordered.extend(parked);
        QueueState {
            waiting: ordered,
            priorities,
            paused_all: self.pause_all.is_paused(),
            concurrency: self.concurrency.load(Ordering::Relaxed),
            throttle_kbps: self.bucket.rate_kbps(),
        }
    }

    /// Current queue snapshot for the panel's initial load.
    pub async fn queue_state(&self) -> QueueState {
        let w = self.waiting.lock().await;
        self.build_queue_state(&w).await
    }

    /// Emit `transfer://queue` and bump the generation so admission waiters
    /// re-check whether it is their turn.
    async fn bump_queue(&self, app: &AppHandle) {
        let state = {
            let w = self.waiting.lock().await;
            self.build_queue_state(&w).await
        };
        self.queue_gen.send_modify(|g| *g += 1);
        let _ = app.emit("transfer://queue", &state);
    }

    /// Wait until this transfer is selected by the weighted scheduler with the pause-all
    /// gate open, take a concurrency permit, and pop it from the queue.
    /// Returns `None` if the id left the queue (canceled while waiting).
    /// The permit is returned so the caller holds it for the transfer's life.
    async fn reserve_slot(&self) -> Option<TransferSlot> {
        let permit = self.semaphore.clone().acquire_owned().await.ok()?;
        Some(TransferSlot {
            permit: Some(permit),
            state: Arc::clone(&self.concurrency_state),
        })
    }

    async fn admit(&self, id: &str) -> Option<TransferSlot> {
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
            // A pause, cancel or reorder may change the head while all
            // slots are occupied. Drop the pending semaphore acquisition and
            // re-evaluate the FIFO on every queue generation change.
            let permit = tokio::select! {
                result = self.reserve_slot() => result?,
                changed = rx.changed() => {
                    changed.ok()?;
                    continue;
                }
            };
            let mut w = self.waiting.lock().await;
            if let Some((_, next_cursor)) = self
                .next_runnable(&w)
                .await
                .filter(|(next, _)| next.as_str() == id)
            {
                self.queue_cursor.store(next_cursor, Ordering::Relaxed);
                if let Some(pos) = w.iter().position(|x| x == id) {
                    w.remove(pos);
                }
                // The next waiter can use another free slot immediately,
                // rather than waiting for this transfer to finish.
                self.queue_gen.send_modify(|g| *g += 1);
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

    /// Select one runnable FIFO head per tier, then apply the weighted cycle.
    /// Paused rows never head-of-line block the queue; pause-all blocks everyone.
    /// Caller holds waiting; lock order is waiting → pauses → priority overrides.
    async fn next_runnable<'a>(&self, w: &'a VecDeque<String>) -> Option<(&'a String, usize)> {
        if self.pause_all.is_paused() {
            return None;
        }
        let pauses = self.pauses.lock().await;
        let priorities = self.priority_overrides();
        let mut heads = [None; 3];
        for id in w {
            if pauses.get(id).is_some_and(|gate| gate.is_paused()) {
                continue;
            }
            let tier = priorities.get(id).copied().unwrap_or_default() as usize;
            if heads[tier].is_none() {
                heads[tier] = Some(id);
            }
        }
        let cursor = self.queue_cursor.load(Ordering::Relaxed);
        let (tier, next) = next_queue_tier(cursor, std::array::from_fn(|i| heads[i].is_some()))?;
        heads[tier].map(|id| (id, next))
    }

    async fn is_my_turn(&self, w: &VecDeque<String>, id: &str) -> bool {
        self.next_runnable(w)
            .await
            .is_some_and(|(next, _)| next == id)
    }

    /// Reorder a waiting transfer (active transfers are untouched).
    pub async fn move_in_queue(&self, id: &str, up: bool, app: &AppHandle) -> Result<()> {
        {
            let mut w = self.waiting.lock().await;
            let Some(pos) = w.iter().position(|x| x == id) else {
                anyhow::bail!("transfer {id} is not waiting in the queue");
            };
            let priorities = self.priority_overrides();
            let priority = priorities.get(id).copied().unwrap_or_default();
            let same_tier =
                |index: usize| priorities.get(&w[index]).copied().unwrap_or_default() == priority;
            let swap_with = if up {
                (0..pos).rev().find(|&index| same_tier(index))
            } else {
                (pos + 1..w.len()).find(|&index| same_tier(index))
            };
            if let Some(other) = swap_with {
                w.swap(pos, other);
            }
        }
        self.bump_queue(app).await;
        Ok(())
    }

    /// Assign a queued row to a priority tier. Moving between tiers puts it
    /// behind existing work of the new tier; assigning the same tier is a no-op.
    async fn change_waiting_priority(&self, id: &str, priority: TransferPriority) -> Result<()> {
        // Serialize with progress/terminal writes so a stale ledger snapshot
        // cannot overwrite a newly selected priority.
        let _ledger = self.ledger_serial.lock().await;
        let mut waiting = self.waiting.lock().await;
        let pos = waiting
            .iter()
            .position(|queued| queued == id)
            .ok_or_else(|| anyhow::anyhow!("transfer {id} is not waiting in the queue"))?;
        let previous = self
            .priority_overrides()
            .get(id)
            .copied()
            .unwrap_or_default();
        if previous == priority {
            return Ok(());
        }
        // Commit first. A failed SQLite write must not produce an apparently
        // successful priority change in the live queue.
        if let Some(db) = &self.db {
            let transfer = self
                .get(id)
                .await
                .ok_or_else(|| anyhow::anyhow!("transfer {id} not found"))?;
            anyhow::ensure!(
                !is_terminal_transfer_status(transfer.status),
                "terminal transfer cannot change queue priority"
            );
            let retry = self
                .persisted_retry
                .lock()
                .await
                .get(id)
                .cloned()
                .ok_or_else(|| anyhow::anyhow!("transfer {id} has no recovery descriptor"))?;
            let envelope = PersistedTransferEnvelope {
                schema: TRANSFER_LEDGER_SCHEMA,
                transfer,
                retry,
                priority,
            };
            let payload = serde_json::to_string(&envelope)?;
            db.transfer_ledger_upsert(id, &payload)
                .context("persist transfer priority")?;
        }
        let mut overrides = self.priority_overrides();
        if priority == TransferPriority::Normal {
            overrides.remove(id);
        } else {
            overrides.insert(id.to_string(), priority);
        }
        if let Some(queued) = waiting.remove(pos) {
            waiting.push_back(queued);
        }
        Ok(())
    }

    pub async fn set_queue_priority(
        &self,
        id: &str,
        priority: TransferPriority,
        app: &AppHandle,
    ) -> Result<()> {
        self.change_waiting_priority(id, priority).await?;
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
        self.bump_queue(app).await;
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
        let _retry = self.retry_serial.lock().await;
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
        self.reset_for_retry(id).await?;
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
                allow_replace,
            } => {
                self.spawn_registered(
                    id.to_string(),
                    run_download_task(
                        mgr,
                        id_for_task,
                        session,
                        remote_path,
                        final_path,
                        allow_replace,
                        app_for_task,
                    ),
                )
                .await;
            }
            RetryInfo::Upload {
                session,
                local,
                final_remote,
                allow_replace,
            } => {
                self.spawn_registered(
                    id.to_string(),
                    run_upload_task(
                        mgr,
                        id_for_task,
                        session,
                        local,
                        final_remote,
                        allow_replace,
                        app_for_task,
                    ),
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

    /// Resize the global pool without leaving stale asynchronous shrink tasks.
    /// Retire free permits immediately; occupied slots are retired on Drop.
    /// A subsequent increase cancels outstanding retirement debt first.
    pub fn set_concurrency(&self, n: usize) {
        let n = n.clamp(1, 32);
        let mut state = self
            .concurrency_state
            .lock()
            .unwrap_or_else(|poison| poison.into_inner());
        let old = self.concurrency.swap(n, Ordering::Relaxed);
        if n < old {
            let retired = self.semaphore.forget_permits(old - n);
            state.pending_shrink += old - n - retired;
        } else if n > old {
            let increase = n - old;
            let cancelled = increase.min(state.pending_shrink);
            state.pending_shrink -= cancelled;
            if increase > cancelled {
                self.semaphore.add_permits(increase - cancelled);
            }
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
        // Cancel and manual Retry must not race to abort/re-register the same
        // worker. finish_transfer uses the ledger lock to resolve Cancel vs
        // Done deterministically before either becomes terminal.
        let _retry = self.retry_serial.lock().await;
        let Some(canceled) = self.mark_canceled(id).await else {
            // An already-completed transfer cannot retroactively be canceled.
            return Ok(());
        };
        {
            let mut waiting = self.waiting.lock().await;
            if let Some(pos) = waiting.iter().position(|queued| queued == id) {
                waiting.remove(pos);
            }
        }
        // FTP spawn_blocking may continue ABOR cleanup after async abort; it
        // already sees the cancellation flag published by mark_canceled.
        if let Some(handle) = self.tasks.lock().await.remove(id) {
            handle.abort();
        }
        let _ = app.emit("transfer://updated", &canceled);
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
        anyhow::ensure!(
            safe_local_download_leaf(&file_name),
            "download path contains an unsafe local filename"
        );
        let local_root = PathBuf::from(&local_dir);
        tokio::fs::create_dir_all(&local_root)
            .await
            .with_context(|| format!("create download directory {}", local_root.display()))?;
        let size = remote_size(&session, &remote_path).await.unwrap_or(0);

        let initial = local_root.join(file_name);
        let (final_path, skip) = match policy {
            OverwritePolicy::Overwrite => (initial, false),
            OverwritePolicy::Skip => {
                let exists = local_path_exists_fail_closed(&initial).await?;
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
                allow_replace: policy == OverwritePolicy::Overwrite,
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
                allow_replace: policy == OverwritePolicy::Overwrite,
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
            run_download_task(
                mgr,
                id_for_task,
                session,
                remote_path,
                final_path,
                policy == OverwritePolicy::Overwrite,
                app,
            ),
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        use base64::Engine as _;
        use ghostftp_agent_proto::msg::{Request, Response};
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let (expected_remote_len, remote_exists) = agent_stat(session, remote_path).await?;
        anyhow::ensure!(remote_exists, "Agent download source does not exist");
        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let mut local_file = open_owned_resumable_download_staging(local_path, owner).await?;
        let local_len = local_file.metadata().await?.len();
        let mut offset = if requested_offset > 0 && local_len >= requested_offset {
            requested_offset
        } else {
            0
        };
        if offset > 0
            && (expected_remote_len < offset
                || !agent_remote_prefix_matches_file(session, remote_path, &mut local_file, offset)
                    .await?)
        {
            offset = 0;
        }
        if offset != requested_offset {
            self.update(id, |t| t.transferred = offset).await;
        }
        local_file.set_len(offset).await?;
        local_file.seek(SeekFrom::Start(offset)).await?;
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
            anyhow::ensure!(
                !bytes.is_empty() || eof,
                "Agent download returned an empty non-final chunk"
            );
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
        local_file
            .sync_all()
            .await
            .context("sync Agent download staging file")?;
        let on_disk = local_file
            .metadata()
            .await
            .context("stat Agent download staging file")?
            .len();
        verify_committed_length("Agent download", expected_remote_len, offset, on_disk)?;
        let (after_len, still_exists) = agent_stat(session, remote_path).await?;
        anyhow::ensure!(
            still_exists && after_len == expected_remote_len,
            "Agent download source changed size during transfer"
        );
        self.update(id, |t| t.transferred = offset).await;
        Ok(local_file)
    }

    async fn run_ssh_download(
        &self,
        id: &str,
        session: Arc<SshSession>,
        remote_path: &str,
        local_path: &Path,
        app: &AppHandle,
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let mut prior = open_owned_resumable_download_staging(local_path, owner).await?;
        let local_len = prior.metadata().await?.len();
        let resume_offset = if requested_offset > 0 && local_len >= requested_offset {
            requested_offset
        } else {
            0
        };
        let sftp_cell = session.ensure_sftp().await?;
        // SFTP servers expose source file size. Read it before the stream
        // rather than trusting EOF: a broken connection or shortened response
        // must never promote a partial .part as a successful download.
        let expected_remote_len = {
            let sftp = sftp_cell.lock().await;
            sftp.metadata(remote_path)
                .await
                .context("stat SFTP download source")?
                .size
                .context("SFTP download source did not report its byte length")?
        };
        // A stale resume point beyond the new source is never trustworthy.
        let mut resume_offset = if resume_offset <= expected_remote_len {
            resume_offset
        } else {
            0
        };
        let mut remote_file = {
            let sftp = sftp_cell.lock().await;
            sftp.open(remote_path)
                .await
                .with_context(|| format!("open remote {remote_path}"))?
        };
        if resume_offset > 0
            && !byte_equal_prefix(&mut prior, &mut remote_file, resume_offset).await?
        {
            resume_offset = 0;
        }
        if resume_offset != requested_offset {
            self.update(id, |t| t.transferred = resume_offset).await;
        }
        remote_file.seek(SeekFrom::Start(resume_offset)).await?;

        let mut local_file = prior;
        local_file.set_len(resume_offset).await?;
        local_file.seek(SeekFrom::Start(resume_offset)).await?;

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
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        local_file.flush().await?;
        local_file
            .sync_all()
            .await
            .context("sync SFTP download staging file")?;
        let on_disk = local_file
            .metadata()
            .await
            .context("stat completed SFTP download staging")?
            .len();
        verify_committed_length("SFTP download", expected_remote_len, transferred, on_disk)?;
        drop(remote_file);
        let remote_after = {
            let sftp = sftp_cell.lock().await;
            sftp.metadata(remote_path)
                .await
                .context("stat SFTP source after download")?
                .size
                .context("SFTP source length unavailable after download")?
        };
        anyhow::ensure!(
            remote_after == expected_remote_len,
            "SFTP source changed size during download"
        );
        self.update(id, |t| t.transferred = transferred).await;
        Ok(local_file)
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
        // Native file pickers can return a symlink even when the UI has no
        // symlink-transfer action. Never dereference an arbitrary link as an
        // ordinary uploaded file (or follow a dangling/out-of-tree target).
        let metadata = tokio::fs::symlink_metadata(&local)
            .await
            .with_context(|| format!("lstat upload source {}", local.display()))?;
        if !metadata.file_type().is_file() {
            anyhow::bail!("upload source is not a regular file: {}", local.display());
        }
        // OS path components preserve Linux filenames that contain a literal
        // backslash. Such names cannot safely map to a remote path, so reject
        // them instead of silently uploading under a different basename.
        let file_name = local
            .file_name()
            .and_then(|name| name.to_str())
            .filter(|name| safe_upload_name(name))
            .ok_or_else(|| {
                anyhow::anyhow!("upload source has an unsafe filename: {}", local.display())
            })?;
        let size = metadata.len();

        // Join with the separator the destination already uses, and don't add a
        // second one. A Windows agent target spelled `C:\srv\` used to produce
        // `C:\srv\/file` — Win32 tolerates it, but it shows up in every error
        // message and audit line, and the mixed form trips path comparisons.
        let initial_remote = join_remote(&remote_dir, file_name);

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
                allow_replace: policy == OverwritePolicy::Overwrite,
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
                allow_replace: policy == OverwritePolicy::Overwrite,
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
            run_upload_task(
                mgr,
                id_for_task,
                session,
                local,
                final_remote,
                policy == OverwritePolicy::Overwrite,
                app,
            ),
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
            agent_stat(session, remote_path).await?.0
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
        if offset > 0
            && !agent_remote_prefix_matches(session, remote_path, local_path, offset).await?
        {
            offset = 0;
        }
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
        // An empty source has no data chunks, but it still must create or
        // truncate the destination. Otherwise a zero-byte upload could be
        // marked Done without changing the previous remote file.
        if local_len == 0 {
            let response = session
                .request(Request::WriteChunk {
                    path: remote_path.to_string(),
                    offset: 0,
                    data: String::new(),
                    truncate: true,
                    done: true,
                })
                .await?;
            match response {
                Response::Written { bytes: 0 } => {}
                Response::Error { .. } => anyhow::bail!("Agent refused empty upload"),
                _ => anyhow::bail!("Agent did not acknowledge an empty upload"),
            }
        }
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
                Response::Written { bytes } => {
                    anyhow::ensure!(
                        bytes == n as u64,
                        "Agent upload acknowledged an incomplete chunk"
                    );
                }
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
        let source_after = local_file
            .metadata()
            .await
            .context("stat Agent upload source after transfer")?
            .len();
        verify_committed_length("Agent upload source", local_len, offset, source_after)?;
        let (remote_after, exists) = agent_stat(session, remote_path).await?;
        anyhow::ensure!(exists, "Agent upload destination not found after transfer");
        verify_committed_length("Agent upload", local_len, offset, remote_after)?;
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
        let (_basis_size, basis_exists) = agent_stat(session, remote_path).await?;
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
            let patch = create_new_delta_upload_patch(&patch_owned)?;
            // Distinguish failures *after* exclusive creation from failures to
            // create the file. Only the former grant us cleanup ownership.
            Ok::<_, anyhow::Error>(delta::plan_delta(&remote_sig, &local_owned, patch))
        })
        .await
        {
            Ok(Ok(Ok(plan))) => plan,
            Ok(Ok(Err(e))) => {
                cleanup_failed_download_staging(&local_patch, &e).await;
                return Err(e.context("plan delta"));
            }
            Ok(Err(e)) => return Err(e.context("create delta upload patch")),
            Err(e) => {
                // A failed join cannot prove which file the worker created;
                // never unlink an unowned or concurrently replaced pathname.
                return Err(anyhow::anyhow!("plan delta task: {e}"));
            }
        };
        let size = plan.literal_bytes + plan.reused_bytes;
        if !delta::delta_worthwhile(&plan, size) {
            cleanup_owned_regular_staging_file(&local_patch).await;
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
                    Response::Written { bytes } => {
                        verify_agent_delta_patch_ack(n, bytes)?;
                    }
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
                Response::DeltaDone {
                    bytes_reused,
                    bytes_written,
                } => {
                    let (remote_size, exists) = agent_stat(session, remote_path).await?;
                    anyhow::ensure!(
                        exists,
                        "Agent delta upload destination missing after assembly"
                    );
                    verify_agent_delta_upload_completion(
                        size,
                        plan.reused_bytes,
                        bytes_reused,
                        bytes_written,
                        remote_size,
                    )
                }
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
        cleanup_owned_regular_staging_file(&local_patch).await;
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
        allow_replace: bool,
    ) -> Result<()> {
        let parent_guard = DownloadParentGuard::with_db(local_path, self.db.clone())?;
        let staging = download_staging_path(local_path, id);
        let owner = DownloadStageOwner::for_parent(&parent_guard);
        let result = self
            .agent_download_core(id, session, remote_path, &staging, app, &owner)
            .await;
        parent_guard.verify()?;
        match result {
            Ok(verified) => {
                let result = parent_guard
                    .promote(&staging, local_path, id, allow_replace, &verified)
                    .await;
                // A Skip/Rename collision must never strand an Agent staging
                // file, nor touch the destination owned by another process.
                if !allow_replace && result.is_err() {
                    parent_guard.cleanup_verified(&staging, &verified).await;
                }
                result
            }
            Err(error) => {
                if error.downcast_ref::<RestartFromPause>().is_none() {
                    owner.cleanup(&parent_guard, &staging).await;
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
        let parent_guard = DownloadParentGuard::with_db(local_path, self.db.clone())?;
        // Continue an interrupted whole-file download from the verified local
        // prefix. Delta mode is reserved for complete prior-file bases.
        if self.get(id).await.is_some_and(|t| t.transferred > 0) {
            return self
                .run_agent_full_download_staged(id, session, remote_path, local_path, app, true)
                .await;
        }

        let (size, remote_exists) = agent_stat(session, remote_path).await?;
        parent_guard.verify()?;
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
        parent_guard.verify()?;
        self.run_agent_full_download_staged(id, session, remote_path, local_path, app, true)
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
        let parent_guard = DownloadParentGuard::with_db(local_path, self.db.clone())?;

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
        parent_guard.verify()?;
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
        // A path is ours only AFTER exclusive creation succeeded. Never clean
        // up an unexpected pre-existing pathname, including a symlink.
        let mut patch_handle = None;
        let mut output_handle = None;
        let result: Result<()> = async {
            parent_guard.verify()?;
            let mut patch_file = create_new_download_staging(&patch_path).await?;
            patch_handle = Some(patch_file.try_clone().await?);
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
                    let next_len = checked_agent_delta_chunk_length(range_len - got, bytes.len())?;
                    self.checkpoint(id, next_len).await?;
                    patch_file.write_all(&bytes).await?;
                    got += next_len;
                    fetched = fetched
                        .checked_add(next_len)
                        .context("delta download fetched-byte counter overflow")?;
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
            parent_guard.verify()?;
            let output = tokio::task::spawn_blocking(move || {
                delta::apply_delta_open(
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
            let output = output_handle.insert(tokio::fs::File::from_std(output));
            verify_download_writer(output, size).await?;
            parent_guard
                .promote_delta(&out_path, local_path, output)
                .await
                .with_context(|| {
                    format!("safely promote delta result to {}", local_path.display())
                })?;
            Ok(())
        }
        .await;
        if let Some(patch) = patch_handle {
            parent_guard.cleanup_verified(&patch_path, &patch).await;
        }
        if result.is_err() {
            if let Some(output) = output_handle {
                parent_guard.cleanup_verified(&out_path, &output).await;
            }
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
        source_snapshot: &LocalUploadSnapshot,
    ) -> Result<()> {
        // Every call uses a *new* exclusively created staging pathname.
        // Progress from a previous attempt cannot be a valid resume offset.
        self.update(id, |t| {
            t.status = TransferStatus::Transferring;
            t.transferred = 0;
        })
        .await;

        let local_len = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let mut local_file = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        source_snapshot.verify(&local_file.metadata().await?)?;
        let sftp_cell = session.ensure_sftp().await?;
        let mut remote_file = {
            let sftp = sftp_cell.lock().await;
            use russh_sftp::protocol::OpenFlags;
            sftp.open_with_flags(
                remote_path,
                OpenFlags::WRITE | OpenFlags::CREATE | OpenFlags::EXCLUDE,
            )
            .await
            .with_context(|| format!("exclusively create remote upload staging {remote_path}"))?
        };

        let mut buf = vec![0u8; 64 * 1024];
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        loop {
            let n = local_file.read(&mut buf).await?;
            if n == 0 {
                break;
            }
            self.checkpoint(id, n as u64).await?;
            remote_file.write_all(&buf[..n]).await?;
            transferred += n as u64;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        remote_file.flush().await?;
        let source_after = local_file
            .metadata()
            .await
            .context("stat SFTP upload source after transfer")?;
        source_snapshot.verify(&source_after)?;
        verify_committed_length(
            "SFTP upload source",
            local_len,
            transferred,
            source_after.len(),
        )?;
        drop(remote_file);
        let remote_size = {
            let sftp = sftp_cell.lock().await;
            sftp.metadata(remote_path)
                .await
                .context("stat completed SFTP upload")?
                .size
                .context("SFTP upload destination did not report its byte length")?
        };
        verify_committed_length("SFTP upload", local_len, transferred, remote_size)?;
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
        ensure_download_subdirectory(&local_root, &local_root).await?;

        let fs = fs_for_session(&session);

        let mut dirs_to_visit: Vec<(String, usize)> = vec![(remote_root.clone(), 0)];
        let mut visited = HashSet::new();
        let mut files: Vec<(String, PathBuf)> = Vec::new();
        let mut entry_count = 0usize;
        let mut total_path_bytes = 0usize;

        while let Some((d, depth)) = dirs_to_visit.pop() {
            anyhow::ensure!(
                visited.insert(d.clone()),
                "server returned a cyclic or duplicated directory listing"
            );
            let entries = fs
                .list_dir(&d)
                .await
                .with_context(|| format!("read_dir {d}"))?;
            for entry in entries {
                let remote_child = entry.path.clone();
                let rel = checked_remote_download_relative(&remote_root, &remote_child)?;
                let local_child = local_root.join(rel);
                entry_count = entry_count
                    .checked_add(1)
                    .context("download entry count overflow")?;
                total_path_bytes = total_path_bytes
                    .checked_add(remote_child.len())
                    .context("download inventory path length overflow")?;
                validate_directory_download_budget(entry_count, depth, total_path_bytes)?;
                match entry.kind {
                    crate::remotefs::FileKind::Directory => {
                        validate_directory_download_budget(
                            entry_count,
                            depth + 1,
                            total_path_bytes,
                        )?;
                        ensure_download_subdirectory(&local_root, &local_child).await?;
                        dirs_to_visit.push((remote_child, depth + 1));
                    }
                    crate::remotefs::FileKind::File => {
                        if let Some(parent) = local_child.parent() {
                            ensure_download_subdirectory(&local_root, parent).await?;
                        }
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
        let metadata = tokio::fs::symlink_metadata(&local_root_path)
            .await
            .with_context(|| format!("lstat {}", local_root_path.display()))?;
        if !metadata.file_type().is_dir() {
            anyhow::bail!(
                "directory upload source is not a real directory: {}",
                local_root_path.display()
            );
        }
        let root_name = local_root_path
            .file_name()
            .and_then(|n| n.to_str())
            .filter(|name| safe_upload_name(name))
            .ok_or_else(|| {
                anyhow::anyhow!("directory upload requires a safe UTF-8 directory name")
            })?;
        let remote_root = join_remote(&remote_dir, root_name);

        let fs = fs_for_session(&session);

        // Inventory the whole local source before making any server-side
        // mutation. A symlink or unsupported entry must not create a remote
        // folder containing only a misleading subset of the source tree.
        let mut dirs_to_visit: Vec<(PathBuf, usize)> = vec![(local_root_path.clone(), 0)];
        let mut entry_count = 0usize;
        let mut total_path_bytes = 0usize;
        let mut files: Vec<(PathBuf, String)> = Vec::new();
        let mut subdirs: Vec<String> = Vec::new();
        while let Some((d, depth)) = dirs_to_visit.pop() {
            let mut rd = tokio::fs::read_dir(&d)
                .await
                .with_context(|| format!("read_dir {}", d.display()))?;
            while let Some(entry) = rd.next_entry().await? {
                let p = entry.path();
                let meta = inspect_directory_upload_entry(&p).await?;
                let name = entry.file_name().into_string().map_err(|_| {
                    anyhow::anyhow!("directory upload refuses filename not representable as UTF-8")
                })?;
                anyhow::ensure!(
                    safe_upload_name(&name),
                    "directory upload refused: unsafe entry name"
                );
                let relative = p
                    .strip_prefix(&local_root_path)
                    .context("directory upload entry escaped its selected root")?;
                let rel = relative
                    .to_str()
                    .context("directory upload refuses non-UTF-8 relative paths")?
                    .replace('\\', "/");
                let remote_child = format!("{remote_root}/{rel}");
                entry_count = entry_count
                    .checked_add(1)
                    .context("directory upload entry count overflow")?;
                total_path_bytes = total_path_bytes
                    .checked_add(remote_child.len())
                    .context("directory upload inventory size overflow")?;
                validate_directory_upload_budget(entry_count, depth, total_path_bytes)?;
                if meta.file_type().is_dir() {
                    validate_directory_upload_budget(entry_count, depth + 1, total_path_bytes)?;
                    subdirs.push(remote_child);
                    dirs_to_visit.push((p, depth + 1));
                } else if meta.file_type().is_file() {
                    files.push((p, remote_child));
                } else {
                    anyhow::bail!(
                        "directory upload refused: symbolic link or special entry {}",
                        p.display()
                    );
                }
            }
        }

        // Only after every local entry has passed preflight may we mutate
        // the remote tree. Preserve "already exists" compatibility, but never
        // swallow a permission, authentication or network failure.
        ensure_upload_directory(fs.as_ref(), &remote_root).await?;
        subdirs.sort_by_key(|s| s.matches('/').count());
        for sd in subdirs {
            ensure_upload_directory(fs.as_ref(), &sd).await?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        self.checkpoint(id, 0).await?;

        // FTP SIZE/REST cannot prove a prefix is unchanged. Restart RETR at
        // byte zero rather than combine a stale .part with a new remote file.
        self.update(id, |t| t.transferred = 0).await;
        let candidate_offset = 0;
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

        let owner = owner.clone();
        let bucket = Arc::clone(&self.bucket);
        let (outcome, file) = match session
            .with_stream(move |stream| {
                let remote_len = stream.size(&path).ok().map(|size| size as u64);
                let offset = if candidate_offset > 0
                    && remote_len.is_some_and(|len| len >= candidate_offset)
                {
                    candidate_offset
                } else {
                    0
                };

                // FTP cannot authenticate a previously downloaded prefix.
                anyhow::ensure!(offset == 0, "FTP resume requires a verified prefix");
                let file = owner.create_std(&final_path)?;
                let mut writer = std::io::BufWriter::new(file);
                let mut budget = FtpBudget {
                    bucket,
                    pause_all,
                    pause,
                    canceled,
                    accounted: offset,
                };
                let outcome = stream.retr_resumable(&path, offset, &mut writer, |bytes| {
                    budget.control(bytes)
                })?;
                writer.flush().context("flush FTP download destination")?;
                writer
                    .get_ref()
                    .sync_all()
                    .context("sync FTP download staging file")?;
                if outcome.control == FtpTransferControl::Continue {
                    let persisted = writer
                        .get_ref()
                        .metadata()
                        .context("stat completed FTP download staging")?
                        .len();
                    if let Some(expected) = remote_len {
                        verify_committed_length(
                            "FTP download",
                            expected,
                            outcome.transferred,
                            persisted,
                        )?;
                    }
                    // When SIZE is supported, also reject a source that was
                    // replaced/truncated or extended during RETR.
                    if let Ok(after) = stream.size(&path) {
                        verify_committed_length(
                            "FTP download",
                            after as u64,
                            outcome.transferred,
                            persisted,
                        )?;
                    }
                    anyhow::ensure!(
                        persisted == outcome.transferred,
                        "FTP download persisted fewer bytes than received"
                    );
                }
                Ok((
                    outcome,
                    writer.into_inner().context("finish FTP staging writer")?,
                ))
            })
            .await
        {
            Ok(outcome) => outcome,
            Err(error) => {
                return Err(match session.reconnect().await {
                    Ok(()) => error,
                    Err(reconnect_error) => error.context(format!(
                        "additionally failed to reconnect FTP session after download failure: {reconnect_error:#}"
                    )),
                });
            }
        };

        self.update(id, |t| t.transferred = outcome.transferred)
            .await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://progress", &t);
        }

        match outcome.control {
            FtpTransferControl::Continue => Ok(tokio::fs::File::from_std(file)),
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
        source_snapshot: &LocalUploadSnapshot,
    ) -> Result<()> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        self.checkpoint(id, 0).await?;

        let requested_offset = self.get(id).await.map(|t| t.transferred).unwrap_or(0);
        let local_len = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        // FTP SIZE/APPE verifies length only, not prefix identity. A partial
        // remote file may be a different revision with identical length.
        // Safely replace from zero instead of appending to unverified bytes.
        let candidate_offset = 0;
        if requested_offset > 0 {
            self.update(id, |t| t.transferred = 0).await;
        }
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

        let bucket = Arc::clone(&self.bucket);
        let retry_bucket = Arc::clone(&self.bucket);
        let retry_pause = pause.clone();
        let retry_pause_all = pause_all.clone();
        let retry_canceled = canceled.clone();
        let retry_local = local.clone();
        let retry_remote = remote.clone();
        let first_source_snapshot = source_snapshot.clone();
        let retry_source_snapshot = source_snapshot.clone();

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
                first_source_snapshot.verify(&file.metadata()?)?;
                if offset > 0 {
                    file.seek(SeekFrom::Start(offset))?;
                }
                let mut reader = std::io::BufReader::new(file);
                let mut budget = FtpBudget {
                    bucket,
                    pause_all,
                    pause,
                    canceled,
                    accounted: offset,
                };
                let outcome = stream.stor_resumable(&remote, offset, &mut reader, |bytes| {
                    budget.control(bytes)
                })?;
                first_source_snapshot.verify(&reader.get_ref().metadata()?)?;

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

            let restart = session
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
                    retry_source_snapshot.verify(&restart_file.metadata()?)?;
                    let mut restart_reader = BudgetedFtpReader {
                        inner: std::io::BufReader::new(restart_file),
                        budget: FtpBudget {
                            bucket: retry_bucket,
                            pause_all: retry_pause_all,
                            pause: retry_pause,
                            canceled: retry_canceled,
                            accounted: 0,
                        },
                        stop: FtpTransferControl::Continue,
                    };
                    let written = match stream.put_from_reader(&retry_remote, &mut restart_reader) {
                        Ok(written) => written,
                        Err(_) if restart_reader.stop != FtpTransferControl::Continue => {
                            return Ok(crate::session::ftp::FtpTransferOutcome {
                                transferred: 0,
                                control: restart_reader.stop,
                            });
                        }
                        Err(error) => return Err(error),
                    };
                    retry_source_snapshot.verify(&restart_reader.inner.get_ref().metadata()?)?;
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
                .await;
            outcome = match restart {
                Ok(outcome) => outcome,
                Err(error) => {
                    return Err(match session.reconnect().await {
                        Ok(()) => error,
                        Err(reconnect_error) => error.context(format!(
                            "additionally failed to reconnect FTP session after recovery upload failure: {reconnect_error:#}"
                        )),
                    });
                }
            };
        }

        self.update(id, |t| t.transferred = outcome.transferred)
            .await;
        if let Some(t) = self.get(id).await {
            let _ = app.emit("transfer://progress", &t);
        }

        match outcome.control {
            FtpTransferControl::Continue => verify_committed_length(
                "FTP upload",
                local_len,
                outcome.transferred,
                outcome.transferred,
            ),
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
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

        // The object response's own metadata is authoritative for this GET;
        // the earlier queue estimate may be stale or unavailable.
        let expected_size = u64::try_from(get.meta.size)
            .context("object storage reported an invalid download length")?;
        let mut file = owner.create(local_path).await?;
        let mut stream = get.into_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("s3 chunk for {key}"))?;
            let next = checked_object_download_progress(expected_size, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync object download staging file")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed object download staging")?
            .len();
        verify_committed_length(
            "object storage download",
            expected_size,
            transferred,
            on_disk,
        )?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
            (&mut file).take(size + 1).read_to_end(&mut buf).await?;
            require_exact_upload_length(size, buf.len() as u64)?;
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
        let result: Result<()> = async {
            while transferred < size {
                let expected_part = ((size - transferred).min(PART as u64)) as usize;
                let mut filled = 0;
                while filled < expected_part {
                    let n = file.read(&mut buf[filled..expected_part]).await?;
                    if n == 0 {
                        break;
                    }
                    filled += n;
                }
                // Never complete a shorter-than-advertised multipart object
                // after an upload source was truncated or became unreadable.
                require_exact_upload_length(expected_part as u64, filled as u64)?;
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
            }
            let on_disk = file.metadata().await?.len();
            require_exact_upload_length(size, on_disk)?;
            upload
                .complete()
                .await
                .with_context(|| format!("s3 complete multipart {key}"))?;
            Ok(())
        }
        .await;
        if let Err(error) = result {
            // A failed or paused upload must not leave billable orphaned parts
            // behind when the object-store service permits abort.
            let _ = upload.abort().await;
            return Err(error);
        }
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let url = session.url_for(remote_path, false);
        let resp = session
            .request(reqwest::Method::GET, url)
            .header(reqwest::header::ACCEPT_ENCODING, "identity")
            .send()
            .await
            .with_context(|| format!("GET {remote_path}"))?;
        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("webdav chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync HTTP download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed HTTP download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let url = session.url_for(remote_path, false);
        let resp = session
            .request(reqwest::Method::GET, url)
            .header(reqwest::header::ACCEPT_ENCODING, "identity")
            .send()
            .await
            .with_context(|| format!("GET {remote_path}"))?;
        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("http chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync HTTP download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed HTTP download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let dbx = crate::remotefs::dropbox::dropbox_api_path(remote_path);
        let arg = serde_json::json!({ "path": dbx }).to_string();
        let resp = session.content_get("/2/files/download", &arg).await?;

        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("dropbox chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync cloud download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed cloud download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::shopify::read_asset(&session, remote_path).await?;
        let mut file = owner.create(local_path).await?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(file)
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

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let data = read_cloud_upload_bounded(local_path, size).await?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::hubspot::read_file(&session, remote_path).await?;
        let mut file = owner.create(local_path).await?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(file)
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

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let data = read_cloud_upload_bounded(local_path, size).await?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;
        let _ = app; // single-shot API: progress is reported at completion.

        self.checkpoint(id, 0).await?;
        let data = crate::remotefs::dynamics::read_file(&session, remote_path).await?;
        let mut file = owner.create(local_path).await?;
        file.write_all(&data).await?;
        file.flush().await?;
        self.update(id, |t| t.transferred = data.len() as u64).await;
        Ok(file)
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

        let size = tokio::fs::metadata(local_path)
            .await
            .with_context(|| format!("stat {}", local_path.display()))?
            .len();
        let data = read_cloud_upload_bounded(local_path, size).await?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
        use futures::StreamExt;

        self.update(id, |t| t.status = TransferStatus::Transferring)
            .await;

        let content = crate::remotefs::onedrive::content_ref(remote_path);
        let resp = session.get_stream(&content).await?;

        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("onedrive chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync cloud download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed cloud download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
            let expected_part = ((size - offset).min(CHUNK as u64)) as usize;
            let mut filled = 0;
            while filled < expected_part {
                let n = file.read(&mut buf[filled..expected_part]).await?;
                if n == 0 {
                    break;
                }
                filled += n;
            }
            // Sending a short non-final Content-Range could silently produce
            // an incomplete remote item after the local file was truncated.
            require_exact_upload_length(expected_part as u64, filled as u64)?;
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
        require_exact_upload_length(size, file.metadata().await?.len())?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
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

        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("drive chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync cloud download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed cloud download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
        let bytes = read_cloud_upload_bounded(local_path, size).await?;
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
        owner: &DownloadStageOwner,
    ) -> Result<tokio::fs::File> {
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

        let expected_length = declared_full_http_download_length(resp.status(), resp.headers())?;

        let mut file = owner.create(local_path).await?;
        let mut stream = resp.bytes_stream();
        let mut transferred: u64 = 0;
        let mut last_emit = Instant::now();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.with_context(|| format!("box chunk for {remote_path}"))?;
            let next = checked_http_download_progress(expected_length, transferred, chunk.len())?;
            self.checkpoint(id, chunk.len() as u64).await?;
            file.write_all(&chunk).await?;
            transferred = next;
            if last_emit.elapsed() > Duration::from_millis(100) {
                self.update(id, |t| t.transferred = transferred).await;
                if let Some(t) = self.get(id).await {
                    let _ = app.emit("transfer://progress", &t);
                }
                last_emit = Instant::now();
            }
        }
        file.flush().await?;
        file.sync_all()
            .await
            .context("sync cloud download staging to disk")?;
        let on_disk = file
            .metadata()
            .await
            .context("stat completed cloud download staging")?
            .len();
        verify_plain_http_download_length(expected_length, transferred, on_disk)?;
        self.update(id, |t| t.transferred = transferred).await;
        Ok(file)
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
        // Box multipart supports a streaming body. Do not allocate the entire
        // local file (potentially >4 GiB) into process memory.
        let input = tokio::fs::File::open(local_path)
            .await
            .with_context(|| format!("open {}", local_path.display()))?;
        let body = reqwest::Body::wrap_stream(tokio_util::io::ReaderStream::new(input));
        let token = session.access_token().await?;
        self.checkpoint(id, size).await?;

        let file_part =
            reqwest::multipart::Part::stream_with_length(body, size).file_name(name.clone());
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

/// HEAD a WebDAV resource, returning its size and whether it exists.
///
/// Only an explicit HTTP 404 proves absence. Authentication, permission,
/// network and protocol failures propagate so overwrite decisions fail closed.
async fn webdav_head(session: &Arc<WebdavSession>, path: &str) -> Result<(u64, bool)> {
    let url = session.url_for(path, false);
    let resp = session
        .request(reqwest::Method::HEAD, url)
        .send()
        .await
        .with_context(|| format!("verify WebDAV target existence for {path}"))?;
    if resp.status() == reqwest::StatusCode::NOT_FOUND {
        return Ok((0, false));
    }
    if !resp.status().is_success() {
        anyhow::bail!(
            "verify WebDAV target existence for {path} failed: HTTP {}",
            resp.status().as_u16()
        );
    }
    let size = resp
        .headers()
        .get(reqwest::header::CONTENT_LENGTH)
        .and_then(|v| v.to_str().ok())
        .and_then(|s| s.parse::<u64>().ok())
        .unwrap_or(0);
    Ok((size, true))
}

async fn object_exists_fail_closed(
    store: &Arc<dyn object_store::ObjectStore>,
    path: &object_store::path::Path,
) -> Result<bool> {
    match store.head(path).await {
        Ok(_) => Ok(true),
        Err(object_store::Error::NotFound { .. }) => Ok(false),
        Err(error) => {
            Err(error).with_context(|| format!("verify object target existence for {path}"))
        }
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

/// Byte-range pause/resume is enabled only when full prefix bytes can be
/// compared before continuation. SFTP uses seekable handles and Agent exposes
/// ranged reads. FTP/FTPS have length-only REST/APPE semantics here, so restart
/// from zero until verified prefix support can be implemented.
fn supports_byte_resume(session: &Session) -> bool {
    matches!(session, Session::Ssh(_) | Session::Agent(_))
}

/// Stat a path on a Ghost FTP Agent daemon.
///
/// Only the daemon's structured OS-level not-found signal proves absence.
/// Policy, permission, network and generic operational failures propagate.
async fn agent_stat(
    session: &Arc<crate::session::AgentSession>,
    path: &str,
) -> Result<(u64, bool)> {
    use ghostftp_agent_proto::msg::{Request, Response};
    match session
        .request(Request::Stat {
            path: path.to_string(),
        })
        .await?
    {
        Response::Stat { entry } => Ok((entry.size, true)),
        Response::Error {
            not_found: true, ..
        } => Ok((0, false)),
        Response::Error {
            message, denied, ..
        } => {
            if denied {
                anyhow::bail!("agent stat {path}: denied by remote policy — {message}")
            }
            anyhow::bail!("agent stat {path}: {message}")
        }
        other => anyhow::bail!("agent stat {path}: unexpected reply {other:?}"),
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
        Session::Webdav(dav) => Ok(webdav_head(dav, path).await?.0),
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
        Session::Agent(agent) => Ok(agent_stat(agent, path).await?.0),
    }
}

/// Apply the overwrite policy on the remote side; returns (final_path, skip).
async fn remote_resolve(
    session: &Arc<Session>,
    initial_remote: &str,
    policy: OverwritePolicy,
) -> Result<(String, bool)> {
    // Overwrite intentionally does not need an existence round-trip. Skipping
    // the probe improves latency and avoids turning a harmless create/replace
    // into an extra failure point.
    if policy == OverwritePolicy::Overwrite {
        return Ok((initial_remote.to_string(), false));
    }

    match &**session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => {
                    let exists = sftp.try_exists(initial_remote.to_string()).await?;
                    (initial_remote.to_string(), exists)
                }
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename(&sftp, initial_remote).await?;
                    (renamed, false)
                }
            })
        }
        Session::Ftp(ftp) => {
            let exists = crate::remotefs::ftp::path_exists_fail_closed(ftp, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = ftp.clone();
                        async move {
                            crate::remotefs::ftp::path_exists_fail_closed(&session, &candidate)
                                .await
                        }
                    })
                    .await?;
                    (renamed, false)
                }
            })
        }
        Session::Object(obj) => {
            let key = initial_remote.trim_start_matches('/').to_string();
            let probe = object_store::path::Path::from(key.as_str());
            let exists = object_exists_fail_closed(&obj.store, &probe).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let store = obj.store.clone();
                    let renamed =
                        resolve_remote_rename_with_probe(initial_remote, move |candidate| {
                            let store = store.clone();
                            async move {
                                let key = candidate.trim_start_matches('/');
                                let path = object_store::path::Path::from(key);
                                object_exists_fail_closed(&store, &path).await
                            }
                        })
                        .await?;
                    (renamed, false)
                }
            })
        }
        Session::Webdav(dav) => {
            let (_, exists) = webdav_head(dav, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = dav.clone();
                        async move {
                            webdav_head(&session, &candidate)
                                .await
                                .map(|(_, exists)| exists)
                        }
                    })
                    .await?;
                    (renamed, false)
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
            let exists = dbx.try_exists(&dbx_path).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = dbx.clone();
                        async move {
                            let path = crate::remotefs::dropbox::dropbox_api_path(&candidate);
                            session.try_exists(&path).await
                        }
                    })
                    .await?;
                    (renamed, false)
                }
            })
        }
        Session::OneDrive(od) => {
            let item_ref = crate::remotefs::onedrive::item_ref(initial_remote);
            let exists = od.try_exists(&item_ref).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = od.clone();
                        async move {
                            let item_ref = crate::remotefs::onedrive::item_ref(&candidate);
                            session.try_exists(&item_ref).await
                        }
                    })
                    .await?;
                    (renamed, false)
                }
            })
        }
        Session::GDrive(gd) => {
            let exists = gd.resolve_item(initial_remote).await?.is_some();
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = gd.clone();
                        async move {
                            session
                                .resolve_item(&candidate)
                                .await
                                .map(|item| item.is_some())
                        }
                    })
                    .await?;
                    (renamed, false)
                }
            })
        }
        Session::Box(bx) => {
            let exists = bx.resolve_item(initial_remote).await?.is_some();
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = bx.clone();
                        async move {
                            session
                                .resolve_item(&candidate)
                                .await
                                .map(|item| item.is_some())
                        }
                    })
                    .await?;
                    (renamed, false)
                }
            })
        }
        Session::Shopify(sh) => {
            let exists = crate::remotefs::shopify::asset_exists(sh, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed =
                        resolve_remote_rename_with_probe(initial_remote, |candidate| {
                            let session = sh.clone();
                            async move {
                                crate::remotefs::shopify::asset_exists(&session, &candidate).await
                            }
                        })
                        .await?;
                    (renamed, false)
                }
            })
        }
        Session::HubSpot(hs) => {
            let exists = crate::remotefs::hubspot::file_exists(hs, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed =
                        resolve_remote_rename_with_probe(initial_remote, |candidate| {
                            let session = hs.clone();
                            async move {
                                crate::remotefs::hubspot::file_exists(&session, &candidate).await
                            }
                        })
                        .await?;
                    (renamed, false)
                }
            })
        }
        Session::Dynamics(dynm) => {
            let exists = crate::remotefs::dynamics::file_exists(dynm, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed =
                        resolve_remote_rename_with_probe(initial_remote, |candidate| {
                            let session = dynm.clone();
                            async move {
                                crate::remotefs::dynamics::file_exists(&session, &candidate).await
                            }
                        })
                        .await?;
                    (renamed, false)
                }
            })
        }
        Session::Agent(agent) => {
            let (_, exists) = agent_stat(agent, initial_remote).await?;
            Ok(match policy {
                OverwritePolicy::Overwrite => (initial_remote.to_string(), false),
                OverwritePolicy::Skip => (initial_remote.to_string(), exists),
                OverwritePolicy::Rename if !exists => (initial_remote.to_string(), false),
                OverwritePolicy::Rename => {
                    let renamed = resolve_remote_rename_with_probe(initial_remote, |candidate| {
                        let session = agent.clone();
                        async move {
                            agent_stat(&session, &candidate)
                                .await
                                .map(|(_, exists)| exists)
                        }
                    })
                    .await?;
                    (renamed, false)
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

fn is_safe_staged_upload_retry(error: &anyhow::Error) -> bool {
    error.downcast_ref::<UploadPromotionUnconfirmed>().is_none() && is_transient(error)
}

/// GET/HEAD downloads are safe to retry after bounded HTTP rate limits and
/// gateway outages. Mutating cloud POST/PUT operations deliberately use the
/// original narrower retry classifier: an ambiguous 5xx response might follow
/// a successful create and replaying it could duplicate user files.
fn is_transient_download(error: &anyhow::Error) -> bool {
    if is_transient(error) {
        return true;
    }
    let message = format!("{error:#}").to_ascii_lowercase();
    [
        "http 408", "http 425", "http 429", "http 500", "http 502", "http 503", "http 504",
    ]
    .iter()
    .any(|status| message.contains(status))
}

/// Bounded exponential backoff with jitter prevents transfers on a failed
/// endpoint from retrying in lockstep. The entropy is supplied separately so
/// the complete delay range can be tested without a timer or random source.
fn retry_delay_seconds(attempt: u32, entropy: u64) -> u64 {
    let exponent = attempt.saturating_sub(1).min(3);
    let ceiling = (8_u64 << exponent).min(60);
    let floor = ceiling * 2 / 3;
    floor + entropy % (ceiling - floor + 1)
}

fn next_retry_delay_seconds(attempt: u32) -> u64 {
    retry_delay_seconds(attempt, Uuid::new_v4().as_u128() as u64)
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
) -> Option<TransferSlot> {
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
        mgr.bump_queue(app).await;

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
/// (Phase 2) and auto-retries transient errors with bounded jittered backoff.
/// Retry backoff releases the concurrency permit so an unreachable server
/// cannot reduce throughput for unrelated queued transfers. The transfer must
/// reacquire a permit before its next network attempt.
async fn run_download_task(
    mgr: Arc<TransferManager>,
    id: String,
    session: Arc<Session>,
    remote_path: String,
    final_path: PathBuf,
    allow_replace: bool,
    app: AppHandle,
) {
    let Some(initial_permit) = mgr.admit(&id).await else {
        return;
    };
    mgr.bump_queue(&app).await;
    let mut permit = Some(initial_permit);
    let mut auto_retries = 0u32;
    let max_auto_retries = mgr.max_auto_retries.load(Ordering::Relaxed) as u32;
    // Agent delta-download uses the existing destination as its block-level
    // basis and already assembles into its own verified temporary file. Every
    // other backend writes a whole-file stream, so route it through a sibling
    // staging file and promote only after the complete backend operation succeeds.
    let staged_download = !matches!(&*session, Session::Agent(_));
    let parent_guard = match DownloadParentGuard::with_db(&final_path, mgr.db.clone()) {
        Ok(guard) => guard,
        Err(error) => {
            finalize(&mgr, &id, &app, Err(error)).await;
            mgr.bump_queue(&app).await;
            return;
        }
    };
    if let Some(db) = &mgr.db {
        let recovery = db.download_recovery_list().and_then(|rows| {
            anyhow::ensure!(
                !rows.iter().any(|(pending, _)| pending == &id),
                "download has an unresolved recovery intent"
            );
            Ok(())
        });
        if let Err(error) = recovery {
            finalize(&mgr, &id, &app, Err(error)).await;
            mgr.bump_queue(&app).await;
            return;
        }
    }
    let byte_resume = supports_byte_resume(&session);
    // Only verified-prefix resumable SFTP needs to retain a staging name
    // across pauses. Whole-file adapters use a new exclusive file each retry.
    let mut working_path = if staged_download {
        download_staging_path(&final_path, &id)
    } else {
        final_path.clone()
    };
    let mut owner = DownloadStageOwner::for_parent(&parent_guard);
    let res = loop {
        if let Err(error) = parent_guard.verify() {
            finalize(&mgr, &id, &app, Err(error)).await;
            mgr.bump_queue(&app).await;
            return;
        }
        if staged_download && !byte_resume {
            working_path = fresh_download_staging_path(&final_path, &id);
            owner = DownloadStageOwner::for_parent(&parent_guard);
        }
        let attempt = dispatch_download(
            &mgr,
            &id,
            &session,
            &remote_path,
            DownloadAttempt {
                final_path: &working_path,
                allow_replace,
                owner: &owner,
            },
            &app,
        )
        .await;
        if let Err(error) = parent_guard.verify() {
            // Never clean up through a replaced parent pathname.
            finalize(&mgr, &id, &app, Err(error)).await;
            mgr.bump_queue(&app).await;
            return;
        }
        if staged_download && !byte_resume && attempt.is_err() {
            // Only the handle recorded after exclusive creation can grant cleanup.
            owner.cleanup(&parent_guard, &working_path).await;
        }
        match attempt {
            Err(e) if e.downcast_ref::<RestartFromPause>().is_some() => {
                if !supports_byte_resume(&session) {
                    mgr.update(&id, |t| t.transferred = 0).await;
                }
                continue;
            }
            Err(e) if auto_retries < max_auto_retries && is_transient_download(&e) => {
                auto_retries += 1;
                let delay = next_retry_delay_seconds(auto_retries);
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
            Ok(Some(writer)) => {
                let checked = match mgr.get(&id).await {
                    Some(record) => verify_download_writer(&writer, record.transferred).await,
                    None => Err(anyhow::anyhow!("completed download record disappeared")),
                };
                match checked {
                    Ok(()) => {
                        parent_guard
                            .promote(&working_path, &final_path, &id, allow_replace, &writer)
                            .await
                    }
                    // A failed durability/identity check cannot establish
                    // ownership of the current pathname. Preserve it.
                    Err(error) => Err(error),
                }
            }
            Ok(None) => Err(anyhow::anyhow!(
                "download adapter did not return its staging writer"
            )),
            Err(error) => {
                owner.cleanup(&parent_guard, &working_path).await;
                Err(error)
            }
        }
    } else {
        res.map(|_| ())
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
    allow_replace: bool,
    app: AppHandle,
) {
    let Some(initial_permit) = mgr.admit(&id).await else {
        return;
    };
    mgr.bump_queue(&app).await;
    let mut permit = Some(initial_permit);
    let mut auto_retries = 0u32;
    let max_auto_retries = mgr.max_auto_retries.load(Ordering::Relaxed) as u32;
    // A network Retry is one logical upload: keep the original source
    // identity rather than silently switching to another file revision on
    // the next fresh staging attempt. Explicit manual Retry is a new run.
    let source_snapshot = if matches!(&*session, Session::Ftp(_) | Session::Ssh(_)) {
        match capture_local_upload_snapshot(&local).await {
            Ok(snapshot) => Some(snapshot),
            Err(error) => {
                finalize(&mgr, &id, &app, Err(error)).await;
                mgr.bump_queue(&app).await;
                return;
            }
        }
    } else {
        None
    };
    let res = loop {
        let request = UploadAttempt {
            local: &local,
            final_remote: &final_remote,
            allow_replace,
            source_snapshot: source_snapshot.as_ref(),
        };
        let attempt = dispatch_upload(&mgr, &id, &session, request, &app).await;
        match attempt {
            Err(e) if e.downcast_ref::<RestartFromPause>().is_some() => {
                // Agent alone has a verified upload resume prefix. FTP and
                // SFTP retries start on fresh remote staging paths.
                if !matches!(&*session, Session::Agent(_)) {
                    mgr.update(&id, |t| t.transferred = 0).await;
                }
                continue;
            }
            // A network failure after a cloud or Agent mutation may mean that
            // the server committed the write but the acknowledgment was lost.
            // Only fresh-staged FTP/SFTP whole-file uploads can auto-replay.
            Err(e)
                if auto_retries < max_auto_retries
                    && matches!(&*session, Session::Ftp(_) | Session::Ssh(_))
                    && is_safe_staged_upload_retry(&e) =>
            {
                auto_retries += 1;
                let delay = next_retry_delay_seconds(auto_retries);
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

/// Verify the bytes on the original writer, without reopening its pathname.
/// The parent guard checks the live writer's identity against the destination
/// directory entry immediately before promotion.
async fn verify_download_writer(file: &tokio::fs::File, transferred: u64) -> Result<()> {
    let metadata = file.metadata().await?;
    anyhow::ensure!(
        metadata.is_file() && metadata.len() == transferred,
        "download writer type or byte count changed"
    );
    file.sync_all()
        .await
        .context("sync original download writer")?;
    let synced = file.metadata().await?;
    anyhow::ensure!(
        synced.is_file() && synced.len() == transferred,
        "download writer changed during synchronization"
    );
    Ok(())
}

/// Keep the destination, overwrite intent and staging ownership together.
struct DownloadAttempt<'a> {
    final_path: &'a Path,
    allow_replace: bool,
    owner: &'a DownloadStageOwner,
}

/// Backend dispatch for a single-file download. Extracted so the runner (and
/// manual retry) can re-invoke it.
async fn dispatch_download(
    mgr: &Arc<TransferManager>,
    id: &str,
    session: &Arc<Session>,
    remote_path: &str,
    request: DownloadAttempt<'_>,
    app: &AppHandle,
) -> Result<Option<tokio::fs::File>> {
    let DownloadAttempt {
        final_path,
        allow_replace,
        owner,
    } = request;
    match &**session {
        Session::Ssh(ssh) => mgr
            .run_ssh_download(id, ssh.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Ftp(ftp) => mgr
            .run_ftp_download(id, ftp.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Object(obj) => mgr
            .run_object_download(id, obj.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Webdav(dav) => mgr
            .run_webdav_download(id, dav.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Http(http) => mgr
            .run_http_download(id, http.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Dropbox(dbx) => mgr
            .run_dropbox_download(id, dbx.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::OneDrive(od) => mgr
            .run_onedrive_download(id, od.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::GDrive(gd) => mgr
            .run_gdrive_download(id, gd.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Box(bx) => mgr
            .run_box_download(id, bx.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Shopify(sh) => mgr
            .run_shopify_download(id, sh.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::HubSpot(hs) => mgr
            .run_hubspot_download(id, hs.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Dynamics(dynm) => mgr
            .run_dynamics_download(id, dynm.clone(), remote_path, final_path, app, owner)
            .await
            .map(Some),
        Session::Agent(agent) => {
            debug_assert!(supports_delta(session));
            if allow_replace {
                mgr.run_agent_download_with_delta(id, agent.clone(), remote_path, final_path, app)
                    .await
                    .map(|()| None)
            } else {
                // Agent delta mode reconstructs/updates an existing destination.
                // For Skip/Rename use a staged full copy + atomic no-replace.
                mgr.run_agent_full_download_staged(
                    id,
                    agent,
                    remote_path,
                    final_path,
                    Some(app),
                    false,
                )
                .await
                .map(|()| None)
            }
        }
    }
}

/// A pause or Cancel that reached the transfer manager before remote
/// promotion must never publish a fully staged file as a successful upload.
async fn ensure_upload_promotion_allowed(mgr: &Arc<TransferManager>, id: &str) -> Result<()> {
    let token = mgr.cancels.lock().await.get(id).cloned();
    anyhow::ensure!(
        token.is_some_and(|flag| !flag.load(Ordering::Acquire)),
        "upload canceled before remote promotion"
    );
    let active = mgr
        .get(id)
        .await
        .is_some_and(|t| t.status == TransferStatus::Transferring);
    anyhow::ensure!(active, "upload is no longer transferring");
    let paused = mgr
        .pauses
        .lock()
        .await
        .get(id)
        .is_some_and(|gate| gate.is_paused());
    anyhow::ensure!(
        !paused && !mgr.pause_all.is_paused(),
        "upload paused before remote promotion"
    );
    Ok(())
}

/// Keep the resolved upload target, explicit overwrite intent and stable
/// source identity together across every automatic retry attempt.
struct UploadAttempt<'a> {
    local: &'a Path,
    final_remote: &'a str,
    allow_replace: bool,
    source_snapshot: Option<&'a LocalUploadSnapshot>,
}

/// Backend dispatch for a single-file upload.
async fn dispatch_upload(
    mgr: &Arc<TransferManager>,
    id: &str,
    session: &Arc<Session>,
    request: UploadAttempt<'_>,
    app: &AppHandle,
) -> Result<()> {
    let UploadAttempt {
        local,
        final_remote,
        allow_replace,
        source_snapshot,
    } = request;
    match &**session {
        Session::Ssh(ssh) => {
            let source_snapshot = source_snapshot.context("SFTP source identity not captured")?;
            verify_local_upload_snapshot(local, source_snapshot).await?;
            let staging = fresh_remote_upload_staging_path(final_remote, id)?;
            // SFTP CREATE|EXCLUDE prevents truncation even if a competing
            // writer guesses the staging name after our planner runs.
            mgr.run_ssh_upload(id, ssh.clone(), local, &staging, app, source_snapshot)
                .await?;
            mgr.checkpoint(id, 0).await?;
            verify_local_upload_snapshot(local, source_snapshot).await?;
            ensure_upload_promotion_allowed(mgr, id).await?;
            promote_remote_upload(session, &staging, final_remote, allow_replace)
                .await
                .map_err(|source| anyhow::Error::new(UploadPromotionUnconfirmed(source)))
        }
        Session::Ftp(ftp) => {
            let source_snapshot = source_snapshot.context("FTP source identity not captured")?;
            verify_local_upload_snapshot(local, source_snapshot).await?;
            let staging = fresh_remote_upload_staging_path(final_remote, id)?;
            preflight_ftp_upload_staging(ftp, &staging).await?;
            mgr.run_ftp_upload(id, ftp.clone(), local, &staging, app, source_snapshot)
                .await?;
            mgr.checkpoint(id, 0).await?;
            verify_local_upload_snapshot(local, source_snapshot).await?;
            let expected = tokio::fs::metadata(local)
                .await
                .context("stat uploaded FTP source")?
                .len();
            verify_ftp_upload_staging_size(ftp, &staging, expected).await?;
            ensure_upload_promotion_allowed(mgr, id).await?;
            promote_remote_upload(session, &staging, final_remote, allow_replace)
                .await
                .map_err(|source| anyhow::Error::new(UploadPromotionUnconfirmed(source)))
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

/// Terminal transition, persistence and emitted event agree even when Cancel
/// arrives between the network acknowledgment and finalization.
async fn finalize(mgr: &Arc<TransferManager>, id: &str, app: &AppHandle, result: Result<()>) {
    if let Some(transfer) = mgr.finish_transfer(id, result).await {
        match transfer.status {
            TransferStatus::Done => {
                let _ = app.emit("transfer://done", &transfer);
            }
            TransferStatus::Error => {
                let _ = app.emit("transfer://error", &transfer);
            }
            _ => unreachable!("finish_transfer only returns a terminal completion"),
        }
    }
    mgr.tasks.lock().await.remove(id);
}

#[cfg(test)]
mod http_length_integrity_tests;
#[cfg(test)]
mod tests;
