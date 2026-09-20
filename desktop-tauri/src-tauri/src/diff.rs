//! Directory diff engine (see `docs/plans/6_directory-diff.md`).
//!
//! A Meld/Beyond-Compare for directory trees across **any** Ghost FTP backend —
//! including **remote↔remote** (staging vs prod, two servers, two buckets),
//! which no local diff tool can do. It reuses the shared [`crate::scan`] walk
//! (the same one folder-sync and disk usage lean on) to flatten both sides, then
//! classifies every path symmetrically:
//!
//! * [`DiffClass::OnlyInA`] / [`DiffClass::OnlyInB`] — present on one side only,
//! * [`DiffClass::Different`] — present on both but the copies differ,
//! * [`DiffClass::Same`] — present on both and (as far as we can tell) identical.
//!
//! By default "differ" is decided by **size** — cheap, no byte transfer, and
//! reliable across heterogeneous backends (an object store's upload time vs a
//! local mtime are not comparable, so mtime is surfaced for display but never
//! drives classification). Opt-in `--hash` mode confirms same-size pairs by
//! **content**: a server-side `sha256sum` over SSH when available (no transfer),
//! otherwise a streamed sha256 of the bytes. All backends produce the same
//! lowercase-hex sha256, so a hash computed on one side is comparable to the
//! other regardless of protocol.
//!
//! Three surfaces consume this one engine — `ghostftp-cli diff`, the `ghostftp_diff`
//! Agent-Bridge/MCP tool, and the GUI two-tree view — so the classification
//! lives here once. The pure [`classify`]/[`summarize`] halves are I/O-free and
//! unit-tested; [`hash_pass`] refines them for `--hash`; [`diff`] wires the two
//! walks together for the CLI and bridge (the GUI drives its own cancellable
//! walks and reuses the pure halves). Symlinks are skipped, exactly as the walk
//! already skips them.

use std::collections::BTreeSet;
use std::sync::Arc;

use anyhow::{bail, Context, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::remotefs::RemoteFs;
use crate::scan::{self, ScanEntry, ScanOptions, ScanTree};
use crate::session::{FtpSession, ObjectSession, Session, SshSession};

/// Where a path lives, and whether the two copies match. `A` and `B` are the
/// two sides passed to [`diff`], in order.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum DiffClass {
    /// Present on side A only.
    OnlyInA,
    /// Present on side B only.
    OnlyInB,
    /// Present on both sides but the copies differ (see [`DiffReason`]).
    Different,
    /// Present on both sides and, as far as we can tell, identical.
    Same,
}

/// Why a path present on both sides was classified [`DiffClass::Different`].
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum DiffReason {
    /// File sizes differ — definitively different, no hashing needed.
    Size,
    /// Content hashes differ (only ever set in `--hash` mode — authoritative).
    Content,
}

/// One path in the diff, with whichever side(s) it appears on. `aPath`/`aSize`/
/// `aModified` are populated when the file exists on side A (likewise B). A file
/// on both sides carries both.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DiffEntry {
    /// POSIX-relative path from each side's root (the key both sides share).
    pub relative: String,
    pub class: DiffClass,
    /// Set only when `class` is [`DiffClass::Different`].
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<DiffReason>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub a_path: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub a_size: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub a_modified: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub b_path: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub b_size: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub b_modified: Option<i64>,
    /// When `--hash` couldn't hash one side (unsupported backend or a read
    /// error), the pair keeps its size classification and this says why.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hash_error: Option<String>,
}

impl DiffEntry {
    fn only_in_a(rel: &str, e: &ScanEntry) -> Self {
        DiffEntry {
            relative: rel.to_string(),
            class: DiffClass::OnlyInA,
            reason: None,
            a_path: Some(e.absolute.clone()),
            a_size: Some(e.size),
            a_modified: modified_opt(e.modified),
            b_path: None,
            b_size: None,
            b_modified: None,
            hash_error: None,
        }
    }

    fn only_in_b(rel: &str, e: &ScanEntry) -> Self {
        DiffEntry {
            relative: rel.to_string(),
            class: DiffClass::OnlyInB,
            reason: None,
            a_path: None,
            a_size: None,
            a_modified: None,
            b_path: Some(e.absolute.clone()),
            b_size: Some(e.size),
            b_modified: modified_opt(e.modified),
            hash_error: None,
        }
    }

    fn both(rel: &str, a: &ScanEntry, b: &ScanEntry, class: DiffClass, reason: Option<DiffReason>) -> Self {
        DiffEntry {
            relative: rel.to_string(),
            class,
            reason,
            a_path: Some(a.absolute.clone()),
            a_size: Some(a.size),
            a_modified: modified_opt(a.modified),
            b_path: Some(b.absolute.clone()),
            b_size: Some(b.size),
            b_modified: modified_opt(b.modified),
            hash_error: None,
        }
    }
}

/// `modified` is normalised to `0` by the walk when the backend didn't report an
/// mtime — hoist that to `None` so the UI can distinguish "no mtime" from epoch.
fn modified_opt(m: i64) -> Option<i64> {
    (m > 0).then_some(m)
}

/// Counts by class, plus the total number of distinct paths compared.
#[derive(Debug, Clone, Copy, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DiffSummary {
    pub only_in_a: usize,
    pub only_in_b: usize,
    pub different: usize,
    pub same: usize,
    pub total: usize,
}

/// The full result the three surfaces render. `entries` are sorted by relative
/// path (the walk keys a `BTreeMap`, so the union is naturally ordered).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DiffResult {
    pub root_a: String,
    pub root_b: String,
    /// Whether content hashing ran (`--hash`). When false, same-size files are
    /// reported [`DiffClass::Same`] even if their bytes differ.
    pub hashed: bool,
    pub summary: DiffSummary,
    pub entries: Vec<DiffEntry>,
}

/// Classify two already-walked trees. Pure — no I/O — so it's unit-testable and
/// the hashing pass ([`hash_pass`]) can refine its output in place. "Differ" is
/// size-only here; mtime is carried on the entries for display but never drives
/// the class (see the module docs for why).
pub fn classify(tree_a: &ScanTree, tree_b: &ScanTree) -> Vec<DiffEntry> {
    let keys: BTreeSet<&String> = tree_a.files.keys().chain(tree_b.files.keys()).collect();
    let mut entries = Vec::with_capacity(keys.len());
    for rel in keys {
        match (tree_a.files.get(rel), tree_b.files.get(rel)) {
            (Some(a), None) => entries.push(DiffEntry::only_in_a(rel, a)),
            (None, Some(b)) => entries.push(DiffEntry::only_in_b(rel, b)),
            (Some(a), Some(b)) => {
                let (class, reason) = if a.size == b.size {
                    (DiffClass::Same, None)
                } else {
                    (DiffClass::Different, Some(DiffReason::Size))
                };
                entries.push(DiffEntry::both(rel, a, b, class, reason));
            }
            (None, None) => {} // impossible: the key came from one of the maps
        }
    }
    entries
}

/// Fold entries into class counts.
pub fn summarize(entries: &[DiffEntry]) -> DiffSummary {
    let mut s = DiffSummary { total: entries.len(), ..Default::default() };
    for e in entries {
        match e.class {
            DiffClass::OnlyInA => s.only_in_a += 1,
            DiffClass::OnlyInB => s.only_in_b += 1,
            DiffClass::Different => s.different += 1,
            DiffClass::Same => s.same += 1,
        }
    }
    s
}

/// Refine same-size, both-present pairs by content hash (the `--hash` pass).
/// Files that differ in size are already [`DiffClass::Different`] and are left
/// alone (no transfer wasted proving what size already proved). For each
/// candidate we hash both sides — server-side over SSH where possible, else a
/// streamed sha256 — and set the class from whether the hashes match. A pair we
/// can't hash (unsupported backend, read error) keeps its size classification
/// and records `hash_error` so the surface can flag it. Sequential by design:
/// `--hash` is opt-in and the per-file cost is the point of the opt-in.
pub async fn hash_pass(
    entries: &mut [DiffEntry],
    tree_a: &ScanTree,
    tree_b: &ScanTree,
    session_a: Option<&Session>,
    session_b: Option<&Session>,
) {
    for e in entries.iter_mut() {
        // Only pairs the size check couldn't separate need a content check.
        let (Some(a), Some(b)) = (tree_a.files.get(&e.relative), tree_b.files.get(&e.relative)) else {
            continue;
        };
        if a.size != b.size {
            continue;
        }
        match hash_both(session_a, &a.absolute, session_b, &b.absolute).await {
            Ok((ha, hb)) => {
                if ha == hb {
                    e.class = DiffClass::Same;
                    e.reason = None;
                } else {
                    e.class = DiffClass::Different;
                    e.reason = Some(DiffReason::Content);
                }
            }
            Err(err) => e.hash_error = Some(err.to_string()),
        }
    }
}

/// Walk both sides, classify, and (optionally) content-hash. The convenience
/// entry point for `ghostftp-cli diff` and the `ghostftp_diff` bridge tool — both hold a
/// concrete [`Session`] per remote side (and pass `None` for a local side). The
/// GUI drives its own cancellable walks instead and reuses [`classify`] /
/// [`hash_pass`] / [`summarize`] directly.
#[allow(clippy::too_many_arguments)]
pub async fn diff(
    fs_a: &dyn RemoteFs,
    root_a: &str,
    session_a: Option<&Session>,
    fs_b: &dyn RemoteFs,
    root_b: &str,
    session_b: Option<&Session>,
    hash: bool,
) -> Result<DiffResult> {
    let opts = ScanOptions::default();
    let tree_a = scan::walk(fs_a, root_a, &opts, |_| {})
        .await
        .with_context(|| format!("walking side A ({root_a})"))?;
    let tree_b = scan::walk(fs_b, root_b, &opts, |_| {})
        .await
        .with_context(|| format!("walking side B ({root_b})"))?;

    let mut entries = classify(&tree_a, &tree_b);
    if hash {
        hash_pass(&mut entries, &tree_a, &tree_b, session_a, session_b).await;
    }
    let summary = summarize(&entries);
    Ok(DiffResult {
        root_a: root_a.to_string(),
        root_b: root_b.to_string(),
        hashed: hash,
        summary,
        entries,
    })
}

// ---------- Content hashing ----------

/// Hash both sides. Kept together so a future refactor could parallelise the two
/// reads; today it's two awaits.
async fn hash_both(
    session_a: Option<&Session>,
    path_a: &str,
    session_b: Option<&Session>,
    path_b: &str,
) -> Result<(String, String)> {
    let a = hash_path(session_a, path_a)
        .await
        .with_context(|| format!("hashing side A ({path_a})"))?;
    let b = hash_path(session_b, path_b)
        .await
        .with_context(|| format!("hashing side B ({path_b})"))?;
    Ok((a, b))
}

/// A lowercase-hex sha256 of a file's bytes. `session == None` means the local
/// filesystem. Backends without a byte-read path here (WebDAV/HTTP/cloud/agent)
/// return an error the caller records as `hash_error`, leaving the size-based
/// classification intact rather than failing the whole diff.
///
/// Shared with [`crate::dedupe`], whose hash mode hashes every same-size file
/// in one tree through this same path.
pub(crate) async fn hash_path(session: Option<&Session>, path: &str) -> Result<String> {
    match session {
        None => hash_local(path).await,
        Some(Session::Ssh(ssh)) => hash_ssh(ssh, path).await,
        Some(Session::Object(obj)) => hash_object(obj, path).await,
        Some(Session::Ftp(ftp)) => hash_ftp(ftp, path).await,
        Some(other) => bail!(
            "--hash is not supported for {} connections yet",
            other.protocol()
        ),
    }
}

/// POSIX single-quote a path so it survives spaces / shell metacharacters when
/// interpolated into a remote command.
fn sh_quote(s: &str) -> String {
    format!("'{}'", s.replace('\'', "'\\''"))
}

fn hex_encode(bytes: &[u8]) -> String {
    use std::fmt::Write as _;
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        let _ = write!(s, "{b:02x}");
    }
    s
}

async fn hash_reader<R: tokio::io::AsyncRead + Unpin>(mut reader: R) -> Result<String> {
    use tokio::io::AsyncReadExt;
    let mut hasher = Sha256::new();
    let mut buf = vec![0u8; 128 * 1024];
    loop {
        let n = reader.read(&mut buf).await?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
    }
    Ok(hex_encode(&hasher.finalize()))
}

async fn hash_local(path: &str) -> Result<String> {
    let file = tokio::fs::File::open(path)
        .await
        .with_context(|| format!("open {path}"))?;
    hash_reader(file).await
}

/// SSH: prefer a server-side `sha256sum` (one exec, no byte transfer). Fall back
/// to streaming the file over SFTP and hashing locally when the tool is missing
/// or its output doesn't parse.
async fn hash_ssh(ssh: &Arc<SshSession>, path: &str) -> Result<String> {
    let cmd = format!("sha256sum -b {}", sh_quote(path));
    if let Ok(out) = ssh.exec(&cmd).await {
        if out.exit_code == Some(0) {
            if let Some(tok) = out.stdout.split_whitespace().next() {
                if tok.len() == 64 && tok.bytes().all(|b| b.is_ascii_hexdigit()) {
                    return Ok(tok.to_ascii_lowercase());
                }
            }
        }
    }
    // Fallback: stream over SFTP.
    let cell = ssh.ensure_sftp().await?;
    let sftp = cell.lock().await;
    let remote = sftp
        .open(path)
        .await
        .with_context(|| format!("open {path} over SFTP"))?;
    hash_reader(remote).await
}

async fn hash_object(obj: &Arc<ObjectSession>, path: &str) -> Result<String> {
    use futures::StreamExt;
    let key = path.trim_start_matches('/');
    let p = object_store::path::Path::from(key);
    let get = obj
        .store
        .get(&p)
        .await
        .with_context(|| format!("get {key}"))?;
    let mut stream = get.into_stream();
    let mut hasher = Sha256::new();
    while let Some(chunk) = stream.next().await {
        hasher.update(&chunk?);
    }
    Ok(hex_encode(&hasher.finalize()))
}

/// FTP is a blocking API driven from a worker thread; hash into a `Write` sink
/// as the file streams so we never buffer it whole.
async fn hash_ftp(ftp: &Arc<FtpSession>, path: &str) -> Result<String> {
    struct HashSink(Sha256);
    impl std::io::Write for HashSink {
        fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
            self.0.update(buf);
            Ok(buf.len())
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }
    let path = path.to_string();
    ftp.with_stream(move |stream| {
        let mut sink = HashSink(Sha256::new());
        stream.retr_to_writer(&path, &mut sink)?;
        Ok(hex_encode(&sink.0.finalize()))
    })
    .await
}

// ---------- GUI diff runner (Plan 6 Phase 4) ----------
//
// The two-tree GUI view drives its own cancellable walks (so it can stream
// progress and stop mid-scan) and reuses the pure [`classify`] / [`hash_pass`] /
// [`summarize`] halves above. Shaped exactly like `diskscan::ScanManager`: an
// `Arc<DiffManager>` in `AppState`, one run per `Mutex<HashMap<id, …>>`, progress
// over `diff://…` events, cancel via a shared [`CancelToken`], the result fetched
// once on completion.

use std::collections::HashMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex as StdMutex;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use tauri::{AppHandle, Emitter, State};
use tokio::sync::Mutex;
use uuid::Uuid;

use crate::scan::{CancelToken, ScanProgress};
use crate::AppState;

#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum DiffRunState {
    Comparing,
    Done,
    Error,
    Canceled,
}

/// Which stage a running diff is in, for the progress banner.
#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum DiffPhase {
    WalkingA,
    WalkingB,
    Hashing,
}

/// A snapshot of a diff for the frontend: live counts while `Comparing`, the full
/// [`DiffResult`] once `Done`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiffSnapshot {
    pub id: String,
    pub session_a: String,
    pub path_a: String,
    pub session_b: String,
    pub path_b: String,
    pub hashed: bool,
    pub state: DiffRunState,
    pub phase: DiffPhase,
    pub files_a: usize,
    pub files_b: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<DiffResult>,
    pub started_at: i64,
}

/// The lightweight body streamed over `diff://progress`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct DiffProgressEvent {
    id: String,
    phase: DiffPhase,
    files_a: usize,
    files_b: usize,
}

enum DiffOutcome {
    Running,
    Done(DiffResult),
    Error(String),
    Canceled,
}

struct DiffRun {
    id: String,
    session_a: String,
    path_a: String,
    session_b: String,
    path_b: String,
    hashed: bool,
    phase: StdMutex<DiffPhase>,
    files_a: AtomicUsize,
    files_b: AtomicUsize,
    cancel: CancelToken,
    started_at: i64,
    outcome: StdMutex<DiffOutcome>,
}

impl DiffRun {
    fn snapshot(&self) -> DiffSnapshot {
        let (state, error, result) = match &*self.outcome.lock().unwrap() {
            DiffOutcome::Running => (DiffRunState::Comparing, None, None),
            DiffOutcome::Done(r) => (DiffRunState::Done, None, Some(r.clone())),
            DiffOutcome::Error(e) => (DiffRunState::Error, Some(e.clone()), None),
            DiffOutcome::Canceled => (DiffRunState::Canceled, None, None),
        };
        DiffSnapshot {
            id: self.id.clone(),
            session_a: self.session_a.clone(),
            path_a: self.path_a.clone(),
            session_b: self.session_b.clone(),
            path_b: self.path_b.clone(),
            hashed: self.hashed,
            state,
            phase: *self.phase.lock().unwrap(),
            files_a: self.files_a.load(Ordering::Relaxed),
            files_b: self.files_b.load(Ordering::Relaxed),
            error,
            result,
            started_at: self.started_at,
        }
    }

    fn set_phase(&self, p: DiffPhase) {
        *self.phase.lock().unwrap() = p;
    }
}

struct DiffHandle {
    info: Arc<DiffRun>,
    task: tauri::async_runtime::JoinHandle<()>,
}

pub struct DiffManager {
    runs: Mutex<HashMap<String, DiffHandle>>,
}

impl Default for DiffManager {
    fn default() -> Self {
        Self::new()
    }
}

fn now_ts() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

impl DiffManager {
    pub fn new() -> Self {
        Self { runs: Mutex::new(HashMap::new()) }
    }

    /// Kick off a diff of two sides. Each side is a `RemoteFs` (for the walk) and
    /// an optional live `Session` (for `--hash`; `None` = local). Returns the run
    /// id immediately; the work streams `diff://progress` and settles on one of
    /// `diff://done` / `diff://error` / `diff://canceled`.
    #[allow(clippy::too_many_arguments)]
    pub async fn start(
        self: &Arc<Self>,
        session_a: String,
        path_a: String,
        fs_a: Box<dyn RemoteFs>,
        sess_a: Option<Arc<Session>>,
        session_b: String,
        path_b: String,
        fs_b: Box<dyn RemoteFs>,
        sess_b: Option<Arc<Session>>,
        hash: bool,
        app: AppHandle,
    ) -> String {
        let id = Uuid::new_v4().to_string();
        let info = Arc::new(DiffRun {
            id: id.clone(),
            session_a,
            path_a: path_a.clone(),
            session_b,
            path_b: path_b.clone(),
            hashed: hash,
            phase: StdMutex::new(DiffPhase::WalkingA),
            files_a: AtomicUsize::new(0),
            files_b: AtomicUsize::new(0),
            cancel: CancelToken::new(),
            started_at: now_ts(),
            outcome: StdMutex::new(DiffOutcome::Running),
        });

        let task_info = Arc::clone(&info);
        let task = tauri::async_runtime::spawn(async move {
            run_diff(task_info, fs_a, sess_a, fs_b, sess_b, hash, app).await;
        });

        self.runs.lock().await.insert(id.clone(), DiffHandle { info, task });
        id
    }

    pub async fn snapshot(&self, id: &str) -> Option<DiffSnapshot> {
        self.runs.lock().await.get(id).map(|h| h.info.snapshot())
    }

    pub async fn cancel(&self, id: &str) {
        if let Some(h) = self.runs.lock().await.get(id) {
            h.info.cancel.cancel();
        }
    }

    /// Drop a finished (or abandoned) run — the view was closed.
    pub async fn forget(&self, id: &str) {
        if let Some(h) = self.runs.lock().await.remove(id) {
            h.info.cancel.cancel();
            h.task.abort();
        }
    }
}

fn emit_diff_progress(info: &DiffRun, app: &AppHandle) {
    let _ = app.emit(
        "diff://progress",
        DiffProgressEvent {
            id: info.id.clone(),
            phase: *info.phase.lock().unwrap(),
            files_a: info.files_a.load(Ordering::Relaxed),
            files_b: info.files_b.load(Ordering::Relaxed),
        },
    );
}

/// Walk `root`, streaming file counts into `counter` and throttled progress
/// events. Shares the shape of `diskscan::generic_walk`.
async fn walk_side(
    info: &Arc<DiffRun>,
    fs: &dyn RemoteFs,
    root: &str,
    counter: fn(&DiffRun) -> &AtomicUsize,
    app: &AppHandle,
) -> Result<crate::scan::ScanTree> {
    let opts = ScanOptions {
        concurrency: crate::scan::DEFAULT_CONCURRENCY,
        cancel: info.cancel.clone(),
    };
    let mut last_emit = Instant::now();
    let ev_info = Arc::clone(info);
    let ev_app = app.clone();
    let on_progress = move |p: ScanProgress| {
        counter(&ev_info).store(p.files_found, Ordering::Relaxed);
        if last_emit.elapsed() >= Duration::from_millis(100) {
            emit_diff_progress(&ev_info, &ev_app);
            last_emit = Instant::now();
        }
    };
    scan::walk(fs, root, &opts, on_progress).await
}

async fn run_diff(
    info: Arc<DiffRun>,
    fs_a: Box<dyn RemoteFs>,
    sess_a: Option<Arc<Session>>,
    fs_b: Box<dyn RemoteFs>,
    sess_b: Option<Arc<Session>>,
    hash: bool,
    app: AppHandle,
) {
    // Walk both sides (A then B), each cancellable.
    info.set_phase(DiffPhase::WalkingA);
    emit_diff_progress(&info, &app);
    let tree_a = match walk_side(&info, fs_a.as_ref(), &info.path_a, |r| &r.files_a, &app).await {
        Ok(t) => t,
        Err(e) => return settle_error(&info, &app, format!("walking side A: {e:#}")),
    };
    if info.cancel.is_cancelled() {
        return settle_canceled(&info, &app);
    }

    info.set_phase(DiffPhase::WalkingB);
    emit_diff_progress(&info, &app);
    let tree_b = match walk_side(&info, fs_b.as_ref(), &info.path_b, |r| &r.files_b, &app).await {
        Ok(t) => t,
        Err(e) => return settle_error(&info, &app, format!("walking side B: {e:#}")),
    };
    if info.cancel.is_cancelled() {
        return settle_canceled(&info, &app);
    }

    let mut entries = classify(&tree_a, &tree_b);
    if hash {
        info.set_phase(DiffPhase::Hashing);
        emit_diff_progress(&info, &app);
        hash_pass(&mut entries, &tree_a, &tree_b, sess_a.as_deref(), sess_b.as_deref()).await;
        if info.cancel.is_cancelled() {
            return settle_canceled(&info, &app);
        }
    }

    let summary = summarize(&entries);
    let result = DiffResult {
        root_a: info.path_a.clone(),
        root_b: info.path_b.clone(),
        hashed: hash,
        summary,
        entries,
    };
    *info.outcome.lock().unwrap() = DiffOutcome::Done(result);
    let _ = app.emit("diff://done", info.snapshot());
}

fn settle_error(info: &Arc<DiffRun>, app: &AppHandle, msg: String) {
    *info.outcome.lock().unwrap() = DiffOutcome::Error(msg);
    let _ = app.emit("diff://error", info.snapshot());
}

fn settle_canceled(info: &Arc<DiffRun>, app: &AppHandle) {
    *info.outcome.lock().unwrap() = DiffOutcome::Canceled;
    let _ = app.emit("diff://canceled", info.snapshot());
}

/// Resolve a session id to a `RemoteFs` + optional live `Session` (the local
/// sentinel yields `LocalFs` and no session, so `--hash` reads the local disk).
/// Shared with [`crate::dedupe`], which resolves its single side the same way.
pub(crate) async fn resolve_diff_side(
    session_id: &str,
    state: &AppState,
) -> Result<(Box<dyn RemoteFs>, Option<Arc<Session>>), String> {
    if session_id == crate::commands::LOCAL_SESSION {
        return Ok((Box::new(crate::remotefs::local::LocalFs), None));
    }
    let sess = state
        .sessions
        .get(session_id)
        .await
        .ok_or_else(|| format!("session {session_id} not found"))?;
    Ok((crate::commands::fs_for_session(&sess), Some(sess)))
}

/// Start a two-tree diff. `session_a`/`session_b` are session ids (or the local
/// sentinel). Returns the run id; the frontend then listens for `diff://…` and
/// fetches the result on `done`.
#[tauri::command]
#[allow(clippy::too_many_arguments)]
pub async fn diff_start(
    session_a: String,
    path_a: String,
    session_b: String,
    path_b: String,
    hash: bool,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<String, String> {
    let (fs_a, sess_a) = resolve_diff_side(&session_a, &state).await?;
    let (fs_b, sess_b) = resolve_diff_side(&session_b, &state).await?;
    let mgr = Arc::clone(&state.diff);
    Ok(mgr
        .start(
            session_a, path_a, fs_a, sess_a, session_b, path_b, fs_b, sess_b, hash, app,
        )
        .await)
}

#[tauri::command]
pub async fn diff_status(diff_id: String, state: State<'_, AppState>) -> Result<DiffSnapshot, String> {
    state
        .diff
        .snapshot(&diff_id)
        .await
        .ok_or_else(|| format!("diff {diff_id} not found"))
}

/// Full snapshot including the `result` (present once the diff is done).
#[tauri::command]
pub async fn diff_result(diff_id: String, state: State<'_, AppState>) -> Result<DiffSnapshot, String> {
    state
        .diff
        .snapshot(&diff_id)
        .await
        .ok_or_else(|| format!("diff {diff_id} not found"))
}

#[tauri::command]
pub async fn diff_cancel(diff_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.diff.cancel(&diff_id).await;
    Ok(())
}

#[tauri::command]
pub async fn diff_forget(diff_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.diff.forget(&diff_id).await;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::scan::{ScanEntry, ScanTree};

    fn entry(abs: &str, size: u64, modified: i64) -> ScanEntry {
        ScanEntry { absolute: abs.into(), size, modified, etag: None }
    }

    fn tree(files: &[(&str, ScanEntry)]) -> ScanTree {
        let mut t = ScanTree::default();
        for (rel, e) in files {
            t.files.insert((*rel).to_string(), e.clone());
        }
        t
    }

    #[test]
    fn classifies_presence_and_size() {
        let a = tree(&[
            ("only_a.txt", entry("/a/only_a.txt", 10, 100)),
            ("both_same.txt", entry("/a/both_same.txt", 20, 100)),
            ("both_diff.txt", entry("/a/both_diff.txt", 30, 100)),
        ]);
        let b = tree(&[
            ("only_b.txt", entry("/b/only_b.txt", 5, 200)),
            ("both_same.txt", entry("/b/both_same.txt", 20, 999)), // same size, mtime differs
            ("both_diff.txt", entry("/b/both_diff.txt", 31, 200)), // size differs
        ]);

        let entries = classify(&a, &b);
        let by_rel = |rel: &str| entries.iter().find(|e| e.relative == rel).unwrap().clone();

        // Sorted by relative path.
        assert_eq!(
            entries.iter().map(|e| e.relative.as_str()).collect::<Vec<_>>(),
            vec!["both_diff.txt", "both_same.txt", "only_a.txt", "only_b.txt"]
        );

        assert_eq!(by_rel("only_a.txt").class, DiffClass::OnlyInA);
        assert_eq!(by_rel("only_b.txt").class, DiffClass::OnlyInB);
        // Same size ⇒ Same even though mtime differs (size-only default).
        assert_eq!(by_rel("both_same.txt").class, DiffClass::Same);
        assert_eq!(by_rel("both_same.txt").reason, None);
        // Different size ⇒ Different(Size).
        let d = by_rel("both_diff.txt");
        assert_eq!(d.class, DiffClass::Different);
        assert_eq!(d.reason, Some(DiffReason::Size));
        // Both sides carry their metadata.
        assert_eq!(d.a_size, Some(30));
        assert_eq!(d.b_size, Some(31));

        let s = summarize(&entries);
        assert_eq!(s.only_in_a, 1);
        assert_eq!(s.only_in_b, 1);
        assert_eq!(s.different, 1);
        assert_eq!(s.same, 1);
        assert_eq!(s.total, 4);
    }

    #[test]
    fn identical_trees_are_all_same() {
        let a = tree(&[
            ("x.txt", entry("/a/x.txt", 1, 0)),
            ("d/y.txt", entry("/a/d/y.txt", 2, 0)),
        ]);
        let b = tree(&[
            ("x.txt", entry("/b/x.txt", 1, 0)),
            ("d/y.txt", entry("/b/d/y.txt", 2, 0)),
        ]);
        let entries = classify(&a, &b);
        assert!(entries.iter().all(|e| e.class == DiffClass::Same));
        let s = summarize(&entries);
        assert_eq!(s.same, 2);
        assert_eq!(s.different + s.only_in_a + s.only_in_b, 0);
    }

    #[test]
    fn modified_zero_is_hidden() {
        let a = tree(&[("x.txt", entry("/a/x.txt", 1, 0))]);
        let b = tree(&[("y.txt", entry("/b/y.txt", 1, 1700))]);
        let entries = classify(&a, &b);
        let ea = entries.iter().find(|e| e.relative == "x.txt").unwrap();
        let eb = entries.iter().find(|e| e.relative == "y.txt").unwrap();
        assert_eq!(ea.a_modified, None); // 0 → None
        assert_eq!(eb.b_modified, Some(1700));
    }

    #[test]
    fn hex_encode_is_lowercase_padded() {
        assert_eq!(hex_encode(&[0x00, 0x0f, 0xff, 0xa0]), "000fffa0");
    }

    // The `--hash` pass over real local files: two same-size files whose bytes
    // differ (a same-size edit the size check reports as Same) become
    // Different(Content); flipping one to match makes it Same again. This is the
    // payoff of `--hash` — catching what size alone can't — exercised end-to-end
    // against LocalFs with no session.
    #[tokio::test]
    async fn hash_pass_catches_same_size_content_edit() {
        let base = std::env::temp_dir().join(format!("ghostftp_diff_hash_{}", std::process::id()));
        let da = base.join("a");
        let db = base.join("b");
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&da).unwrap();
        std::fs::create_dir_all(&db).unwrap();
        // Same length (5 bytes), different content.
        std::fs::write(da.join("f.txt"), "hello").unwrap();
        std::fs::write(db.join("f.txt"), "world").unwrap();

        let ta = scan::walk_tree(&crate::remotefs::local::LocalFs, da.to_str().unwrap())
            .await
            .unwrap();
        let tb = scan::walk_tree(&crate::remotefs::local::LocalFs, db.to_str().unwrap())
            .await
            .unwrap();

        // Size-only classification can't tell them apart.
        let mut entries = classify(&ta, &tb);
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].class, DiffClass::Same);

        // The hash pass does.
        hash_pass(&mut entries, &ta, &tb, None, None).await;
        assert_eq!(entries[0].class, DiffClass::Different);
        assert_eq!(entries[0].reason, Some(DiffReason::Content));
        assert!(entries[0].hash_error.is_none());

        // Make side B identical → Same again.
        std::fs::write(db.join("f.txt"), "hello").unwrap();
        let tb2 = scan::walk_tree(&crate::remotefs::local::LocalFs, db.to_str().unwrap())
            .await
            .unwrap();
        let mut entries2 = classify(&ta, &tb2);
        hash_pass(&mut entries2, &ta, &tb2, None, None).await;
        assert_eq!(entries2[0].class, DiffClass::Same);

        std::fs::remove_dir_all(&base).ok();
    }
}
