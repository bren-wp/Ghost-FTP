//! Disk Usage Explorer scan engine (see `docs/plans/4_disk-usage-explorer.md`).
//!
//! A WinDirStat/WizTree-style "what's eating the disk?" analysis, but over **any**
//! backend — because every `RemoteFs` already knows how to walk. A [`ScanManager`]
//! (shaped like `TransferManager` / `agent_host`: an `Arc` in `AppState`, running
//! scans in a `Mutex<HashMap<scanId, …>>`) recursively sizes a directory and
//! aggregates the totals into a tree the frontend renders as a treemap + a
//! size-ranked list.
//!
//! Phase 1 is the **generic walk** — [`crate::scan::walk`], the one shared scan
//! engine, summed up the tree. It works on every backend and is the fallback for
//! the protocol fast paths (shell `du` for SSH/agent, object-store flat listing)
//! that layer on in Phase 3 via [`ScanStrategy`].
//!
//! Progress streams over `diskscan://…` events (mirrors `transfer://…`); cancel
//! flips a shared [`CancelToken`]. The aggregated tree is fetched once, on
//! completion, via `diskscan_tree`.

use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex as StdMutex};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result};
use serde::Serialize;
use tauri::{AppHandle, Emitter};
use tokio::sync::Mutex;
use uuid::Uuid;

use crate::remotefs::{FileKind, RemoteFs};
use crate::scan::{self, CancelToken, ScanOptions, ScanProgress};
use crate::session::{AgentSession, ObjectSession, Session, SshSession};

/// Which strategy produced a scan. Phase 1 only ever reports `Generic`; the
/// exec (`Shell`) and object-store (`ObjectFlat`) fast paths land in Phase 3 and
/// set this so the UI can show which path fired (and why the generic fallback,
/// if exec was denied).
#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum ScanStrategy {
    /// Recursive `RemoteFs::list_dir` walk — always available.
    Generic,
    /// One `du -ab` / `find -printf` over the SSH exec channel or a Ghost FTP Agent.
    Shell,
    /// A single flat object listing under the prefix (S3/Azure).
    ObjectFlat,
}

#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum ScanState {
    Scanning,
    Done,
    Error,
    Canceled,
}

/// One node in the aggregated size tree. `size` is the *total* bytes under this
/// node (own bytes for a file; sum of descendants for a directory). Directories
/// carry `children`; files have none. Children are sorted largest-first so the
/// treemap and ranked list render the "biggest offenders" without re-sorting.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DuNode {
    pub name: String,
    pub path: String,
    pub kind: FileKind,
    pub size: u64,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub children: Vec<DuNode>,
}

/// A snapshot of a scan for the frontend: live progress while `Scanning`, the
/// aggregated `tree` once `Done`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ScanSnapshot {
    pub id: String,
    pub session_id: String,
    pub root: String,
    pub state: ScanState,
    pub strategy: ScanStrategy,
    pub dirs_scanned: usize,
    pub files_found: usize,
    pub total_bytes: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    /// A short explanation shown next to the strategy badge — e.g. why the fast
    /// path fell back to the generic walk ("exec disabled — used walk").
    #[serde(skip_serializing_if = "Option::is_none")]
    pub note: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tree: Option<DuNode>,
    pub started_at: i64,
}

/// The lightweight progress event body streamed over `diskscan://progress`.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct ProgressEvent {
    id: String,
    dirs_scanned: usize,
    files_found: usize,
    total_bytes: u64,
    strategy: ScanStrategy,
}

/// Shared live state for one scan. The task mutates the atomics as it walks; the
/// query commands read them — no lock is held across the walk.
struct ScanInfo {
    id: String,
    session_id: String,
    root: String,
    strategy: StdMutex<ScanStrategy>,
    note: StdMutex<Option<String>>,
    cancel: CancelToken,
    dirs: AtomicUsize,
    files: AtomicUsize,
    bytes: AtomicU64,
    started_at: i64,
    /// The terminal result — `None` until the scan settles.
    result: StdMutex<RunResult>,
}

enum RunResult {
    Running,
    Done(DuNode),
    Error(String),
    Canceled,
}

impl ScanInfo {
    fn snapshot(&self) -> ScanSnapshot {
        let strategy = *self.strategy.lock().unwrap();
        let note = self.note.lock().unwrap().clone();
        let (state, error, tree) = match &*self.result.lock().unwrap() {
            RunResult::Running => (ScanState::Scanning, None, None),
            RunResult::Done(tree) => (ScanState::Done, None, Some(tree.clone())),
            RunResult::Error(e) => (ScanState::Error, Some(e.clone()), None),
            RunResult::Canceled => (ScanState::Canceled, None, None),
        };
        ScanSnapshot {
            id: self.id.clone(),
            session_id: self.session_id.clone(),
            root: self.root.clone(),
            state,
            strategy,
            dirs_scanned: self.dirs.load(Ordering::Relaxed),
            files_found: self.files.load(Ordering::Relaxed),
            total_bytes: self.bytes.load(Ordering::Relaxed),
            error,
            note,
            tree,
            started_at: self.started_at,
        }
    }

    fn set_strategy(&self, s: ScanStrategy) {
        *self.strategy.lock().unwrap() = s;
    }

    fn set_note(&self, n: impl Into<String>) {
        *self.note.lock().unwrap() = Some(n.into());
    }
}

struct ScanHandle {
    info: Arc<ScanInfo>,
    task: tauri::async_runtime::JoinHandle<()>,
}

pub struct ScanManager {
    scans: Mutex<HashMap<String, ScanHandle>>,
}

fn now_ts() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

impl ScanManager {
    pub fn new() -> Self {
        Self {
            scans: Mutex::new(HashMap::new()),
        }
    }

    /// Kick off a scan of `root` on `fs`. `session` (absent for the local FS)
    /// enables the protocol fast paths — a shell `find` over SSH/agent, or a flat
    /// object listing. Returns the scan id immediately; the work runs in a spawned
    /// task streaming `diskscan://progress` and, on settle, one of
    /// `diskscan://done` / `diskscan://error` / `diskscan://canceled`.
    pub async fn start(
        self: &Arc<Self>,
        session_id: String,
        root: String,
        session: Option<Arc<Session>>,
        fs: Box<dyn RemoteFs>,
        app: AppHandle,
    ) -> String {
        let id = Uuid::new_v4().to_string();
        let info = Arc::new(ScanInfo {
            id: id.clone(),
            session_id,
            root: root.clone(),
            strategy: StdMutex::new(ScanStrategy::Generic),
            note: StdMutex::new(None),
            cancel: CancelToken::new(),
            dirs: AtomicUsize::new(0),
            files: AtomicUsize::new(0),
            bytes: AtomicU64::new(0),
            started_at: now_ts(),
            result: StdMutex::new(RunResult::Running),
        });

        let task_info = Arc::clone(&info);
        let mgr = Arc::clone(self);
        let task = tauri::async_runtime::spawn(async move {
            run_scan(task_info, root, session, fs, app).await;
            // Leave the handle in the map so `diskscan_tree`/`_status` can fetch
            // the settled result; the frontend evicts it when the tab closes.
            let _ = mgr;
        });

        self.scans
            .lock()
            .await
            .insert(id.clone(), ScanHandle { info, task });
        id
    }

    pub async fn snapshot(&self, id: &str) -> Option<ScanSnapshot> {
        self.scans.lock().await.get(id).map(|h| h.info.snapshot())
    }

    /// Flip the cancel flag; the walk stops at its next tick and reports Canceled.
    pub async fn cancel(&self, id: &str) {
        if let Some(h) = self.scans.lock().await.get(id) {
            h.info.cancel.cancel();
        }
    }

    /// Drop a finished (or abandoned) scan — the tab was closed. Aborts the task
    /// if it's somehow still running.
    pub async fn forget(&self, id: &str) {
        if let Some(h) = self.scans.lock().await.remove(id) {
            h.info.cancel.cancel();
            h.task.abort();
        }
    }
}

/// Scan body: pick the fastest available strategy, produce a flat file tree, then
/// aggregate + record the result. The fast paths (shell `find`, object listing)
/// fall back to the generic walk on any failure so a scan never hard-fails on a
/// missing tool or a denied exec.
async fn run_scan(
    info: Arc<ScanInfo>,
    root: String,
    session: Option<Arc<Session>>,
    fs: Box<dyn RemoteFs>,
    app: AppHandle,
) {
    let flat: Result<scan::ScanTree, anyhow::Error> = match session.as_deref() {
        // Object stores: one flat listing under the prefix returns every key + size.
        Some(Session::Object(obj)) => {
            info.set_strategy(ScanStrategy::ObjectFlat);
            emit_progress(&info, &app);
            scan_object(obj, &root, &info, &app).await
        }
        // SSH: a single `find … -printf` beats thousands of SFTP round-trips.
        Some(Session::Ssh(ssh)) => {
            info.set_strategy(ScanStrategy::Shell);
            emit_progress(&info, &app);
            match scan_shell_ssh(ssh, &root, &info, &app).await {
                Ok(t) => Ok(t),
                Err(e) => fallback_walk(&info, &fs, &root, &app, &e).await,
            }
        }
        // Ghost FTP Agent: same idea, gated by the daemon's allowExec policy.
        Some(Session::Agent(agent)) => {
            info.set_strategy(ScanStrategy::Shell);
            emit_progress(&info, &app);
            match scan_shell_agent(agent, &root, &info, &app).await {
                Ok(t) => Ok(t),
                Err(e) => fallback_walk(&info, &fs, &root, &app, &e).await,
            }
        }
        // Local FS, FTP, or no shell: the generic walk is the only option.
        _ => generic_walk(&info, &fs, &root, &app).await,
    };

    // Cancellation wins even if a strategy returned a partial Ok tree.
    if info.cancel.is_cancelled() {
        *info.result.lock().unwrap() = RunResult::Canceled;
        let _ = app.emit("diskscan://canceled", info.snapshot());
        return;
    }

    match flat {
        Ok(tree) => {
            let node = build_tree(&root, &tree);
            info.bytes.store(node.size, Ordering::Relaxed);
            info.files.store(tree.files.len(), Ordering::Relaxed);
            *info.result.lock().unwrap() = RunResult::Done(node);
            let _ = app.emit("diskscan://done", info.snapshot());
        }
        Err(e) => {
            *info.result.lock().unwrap() = RunResult::Error(format!("{e:#}"));
            let _ = app.emit("diskscan://error", info.snapshot());
        }
    }
}

fn emit_progress(info: &ScanInfo, app: &AppHandle) {
    let _ = app.emit(
        "diskscan://progress",
        ProgressEvent {
            id: info.id.clone(),
            dirs_scanned: info.dirs.load(Ordering::Relaxed),
            files_found: info.files.load(Ordering::Relaxed),
            total_bytes: info.bytes.load(Ordering::Relaxed),
            strategy: *info.strategy.lock().unwrap(),
        },
    );
}

/// The generic bounded-concurrency `RemoteFs` walk, streaming progress counts.
async fn generic_walk(
    info: &Arc<ScanInfo>,
    fs: &Box<dyn RemoteFs>,
    root: &str,
    app: &AppHandle,
) -> Result<scan::ScanTree, anyhow::Error> {
    let opts = ScanOptions {
        concurrency: scan::DEFAULT_CONCURRENCY,
        cancel: info.cancel.clone(),
    };
    let mut last_emit = Instant::now();
    let ev_info = Arc::clone(info);
    let ev_app = app.clone();
    let on_progress = move |p: ScanProgress| {
        ev_info.dirs.store(p.dirs_scanned, Ordering::Relaxed);
        ev_info.files.store(p.files_found, Ordering::Relaxed);
        ev_info.bytes.store(p.bytes_found, Ordering::Relaxed);
        if last_emit.elapsed() >= Duration::from_millis(100) {
            emit_progress(&ev_info, &ev_app);
            last_emit = Instant::now();
        }
    };
    scan::walk(fs.as_ref(), root, &opts, on_progress).await
}

/// Reset to the generic walk after a fast path failed, recording why.
async fn fallback_walk(
    info: &Arc<ScanInfo>,
    fs: &Box<dyn RemoteFs>,
    root: &str,
    app: &AppHandle,
    reason: &anyhow::Error,
) -> Result<scan::ScanTree, anyhow::Error> {
    let msg = reason.to_string();
    let short = msg.lines().next().unwrap_or(&msg);
    let note = if short.contains("denied") {
        "exec disabled — used walk".to_string()
    } else {
        format!("fast path unavailable — used walk ({short})")
    };
    tracing::info!("disk-usage fast path fell back: {msg}");
    info.set_strategy(ScanStrategy::Generic);
    info.set_note(note);
    emit_progress(info, app);
    generic_walk(info, fs, root, app).await
}

// ---------- Protocol fast paths ----------

/// POSIX single-quote a path so `find` survives spaces / shell metacharacters.
fn sh_quote(s: &str) -> String {
    format!("'{}'", s.replace('\'', "'\\''"))
}

/// `find <root> -type f -printf '%s\t%p\n'` — one line per regular file, its
/// byte size and absolute path, tab-separated. Symlinks aren't `-type f`, so
/// they're skipped exactly as the generic walk skips them.
fn find_command(root: &str) -> String {
    format!("find {} -type f -printf '%s\\t%p\\n'", sh_quote(root))
}

/// Parse `find -printf '%s\t%p\n'` output into a flat file tree.
fn parse_find(root: &str, out: &str) -> scan::ScanTree {
    let normalized_root = root.trim_end_matches('/');
    let mut tree = scan::ScanTree::default();
    for line in out.lines() {
        let line = line.trim_end_matches('\r');
        if line.is_empty() {
            continue;
        }
        let Some((size_str, path)) = line.split_once('\t') else {
            continue;
        };
        let Ok(size) = size_str.trim().parse::<u64>() else {
            continue;
        };
        let rel = scan::relative_of(normalized_root, path);
        if rel.is_empty() {
            continue;
        }
        tree.files.insert(
            rel,
            scan::ScanEntry {
                absolute: path.to_string(),
                size,
                modified: 0,
                etag: None,
            },
        );
    }
    tree
}

/// Sum the flat file sizes into `info` and emit a progress tick.
fn record_flat(info: &ScanInfo, tree: &scan::ScanTree, app: &AppHandle) {
    let bytes: u64 = tree.files.values().map(|e| e.size).sum();
    info.files.store(tree.files.len(), Ordering::Relaxed);
    info.bytes.store(bytes, Ordering::Relaxed);
    emit_progress(info, app);
}

async fn scan_shell_ssh(
    ssh: &Arc<SshSession>,
    root: &str,
    info: &ScanInfo,
    app: &AppHandle,
) -> Result<scan::ScanTree> {
    let out = ssh
        .exec(&find_command(root))
        .await
        .context("run find over SSH")?;
    let tree = parse_find(root, &out.stdout);
    // `find` exits non-zero when *some* subdir was unreadable, yet still lists the
    // rest — accept a non-empty result. Only a truly empty non-zero run is a real
    // failure (e.g. `find` missing), which falls back to the walk.
    if tree.files.is_empty() && out.exit_code != Some(0) {
        let err = out.stderr.trim();
        anyhow::bail!(
            "find failed (exit {:?}): {}",
            out.exit_code,
            if err.is_empty() { "no output" } else { err }
        );
    }
    record_flat(info, &tree, app);
    Ok(tree)
}

async fn scan_shell_agent(
    agent: &Arc<AgentSession>,
    root: &str,
    info: &ScanInfo,
    app: &AppHandle,
) -> Result<scan::ScanTree> {
    // Cap the captured output; if `find` overruns it the tree would be partial,
    // so we bail to the (correct) generic walk instead.
    const MAX: usize = 64 * 1024 * 1024;
    let out = agent.exec(&find_command(root), MAX, 120_000).await?;
    if out.timed_out {
        anyhow::bail!("find timed out");
    }
    if out.truncated {
        anyhow::bail!("find output exceeded the cap");
    }
    let tree = parse_find(root, &out.stdout);
    if tree.files.is_empty() && out.exit_code != Some(0) {
        let err = out.stderr.trim();
        anyhow::bail!(
            "find failed (exit {:?}): {}",
            out.exit_code,
            if err.is_empty() { "no output" } else { err }
        );
    }
    record_flat(info, &tree, app);
    Ok(tree)
}

async fn scan_object(
    obj: &Arc<ObjectSession>,
    root: &str,
    info: &ScanInfo,
    app: &AppHandle,
) -> Result<scan::ScanTree> {
    use futures::StreamExt;
    use object_store::path::Path as ObjPath;
    use object_store::ObjectStore;

    let prefix_raw = root.trim().trim_matches('/');
    let prefix = if prefix_raw.is_empty() || prefix_raw == "." {
        String::new()
    } else {
        prefix_raw.to_string()
    };
    let prefix_path = if prefix.is_empty() {
        None
    } else {
        Some(ObjPath::from(prefix.as_str()))
    };

    let mut stream = obj.store.list(prefix_path.as_ref());
    let mut tree = scan::ScanTree::default();
    let mut bytes = 0u64;
    let mut last_emit = Instant::now();
    while let Some(meta) = stream.next().await {
        if info.cancel.is_cancelled() {
            break;
        }
        let meta = meta.context("list objects")?;
        let key = meta.location.as_ref().to_string();
        let rel = if prefix.is_empty() {
            key.clone()
        } else {
            key.strip_prefix(&prefix)
                .unwrap_or(&key)
                .trim_start_matches('/')
                .to_string()
        };
        if rel.is_empty() {
            continue; // a marker object for the prefix itself
        }
        let size = meta.size as u64;
        bytes += size;
        tree.files.insert(
            rel,
            scan::ScanEntry {
                absolute: format!("/{key}"),
                size,
                modified: meta.last_modified.timestamp(),
                etag: meta.e_tag.clone(),
            },
        );
        info.files.store(tree.files.len(), Ordering::Relaxed);
        info.bytes.store(bytes, Ordering::Relaxed);
        if last_emit.elapsed() >= Duration::from_millis(100) {
            emit_progress(info, app);
            last_emit = Instant::now();
        }
    }
    emit_progress(info, app);
    Ok(tree)
}

/// The basename of a POSIX-ish path, tolerating a trailing slash. Empty → "/".
fn basename(path: &str) -> String {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        return "/".to_string();
    }
    trimmed
        .rsplit(|c| c == '/' || c == '\\')
        .next()
        .unwrap_or(trimmed)
        .to_string()
}

/// Intermediate builder: a directory-or-file with children keyed by segment name
/// and a running aggregate `size`. Converted to a sorted [`DuNode`] at the end.
#[derive(Default)]
struct Build {
    size: u64,
    is_dir: bool,
    children: HashMap<String, Build>,
}

/// The scan root as a uniformly forward-slashed path. Child segments are already
/// POSIX-normalised (`scan::relative_of` replaces `\`), so normalising the root
/// prefix here makes *every* DuNode path below it fully `/`-separated instead of
/// the mixed `C:\Users\x/Projects` the raw local path would produce. A trailing
/// separator is trimmed except on a bare root — POSIX `/` or a Windows drive root
/// `C:/` — which must keep it (a bare `C:` is the drive's CWD, not its root).
fn normalize_root(root: &str) -> String {
    let trimmed = root.replace('\\', "/");
    let trimmed = trimmed.trim_end_matches('/');
    if trimmed.is_empty() {
        "/".to_string()
    } else if trimmed.ends_with(':') {
        format!("{trimmed}/")
    } else {
        trimmed.to_string()
    }
}

/// Fold the flat file list into a nested, size-aggregated tree.
fn build_tree(root: &str, tree: &scan::ScanTree) -> DuNode {
    let mut builder = Build {
        is_dir: true,
        ..Default::default()
    };
    for (rel, entry) in &tree.files {
        let segments: Vec<&str> = rel.split('/').filter(|s| !s.is_empty()).collect();
        if segments.is_empty() {
            continue;
        }
        builder.size += entry.size;
        insert_path(&mut builder, &segments, entry.size);
    }
    convert(
        basename(root),
        normalize_root(root),
        FileKind::Directory,
        builder,
    )
}

fn insert_path(node: &mut Build, segments: &[&str], size: u64) {
    let (first, rest) = segments.split_first().expect("non-empty segments");
    let child = node.children.entry((*first).to_string()).or_default();
    child.size += size;
    if rest.is_empty() {
        // Leaf file (unless a directory of the same name already claimed it).
    } else {
        child.is_dir = true;
        insert_path(child, rest, size);
    }
}

fn convert(name: String, path: String, kind: FileKind, build: Build) -> DuNode {
    let mut children: Vec<DuNode> = build
        .children
        .into_iter()
        .map(|(seg, child)| {
            // Roots already end in their separator (POSIX `/`, drive root `C:/`);
            // everything else needs one inserted before the segment.
            let child_path = if path.ends_with('/') {
                format!("{path}{seg}")
            } else {
                format!("{path}/{seg}")
            };
            let child_kind = if child.is_dir {
                FileKind::Directory
            } else {
                FileKind::File
            };
            convert(seg, child_path, child_kind, child)
        })
        .collect();
    // Largest-first: the biggest offenders lead in both the treemap and the list.
    children.sort_by(|a, b| b.size.cmp(&a.size).then_with(|| a.name.cmp(&b.name)));
    DuNode {
        name,
        path,
        kind,
        size: build.size,
        children,
    }
}

// ---------- Tauri commands ----------

use tauri::State;

/// Start a disk-usage scan of `path` on `session_id`. Returns the scan id; the
/// frontend then listens for `diskscan://…` and fetches the tree on `done`.
#[tauri::command]
pub async fn diskscan_start(
    session_id: String,
    path: String,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<String, String> {
    // The live Session (absent for the local FS) unlocks the protocol fast paths;
    // `fs` is always the generic-walk fallback.
    let session = if session_id == crate::commands::LOCAL_SESSION {
        None
    } else {
        Some(
            state
                .sessions
                .get(&session_id)
                .await
                .ok_or_else(|| format!("session {session_id} not found"))?,
        )
    };
    let fs = crate::commands::fs_for_public(&session_id, &state).await?;
    let mgr = Arc::clone(&state.diskscan);
    Ok(mgr.start(session_id, path, session, fs, app).await)
}

#[tauri::command]
pub async fn diskscan_status(
    scan_id: String,
    state: State<'_, AppState>,
) -> Result<ScanSnapshot, String> {
    state
        .diskscan
        .snapshot(&scan_id)
        .await
        .ok_or_else(|| format!("scan {scan_id} not found"))
}

/// Full snapshot including the aggregated `tree` (present once the scan is done).
#[tauri::command]
pub async fn diskscan_tree(
    scan_id: String,
    state: State<'_, AppState>,
) -> Result<ScanSnapshot, String> {
    state
        .diskscan
        .snapshot(&scan_id)
        .await
        .ok_or_else(|| format!("scan {scan_id} not found"))
}

#[tauri::command]
pub async fn diskscan_cancel(scan_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.diskscan.cancel(&scan_id).await;
    Ok(())
}

#[tauri::command]
pub async fn diskscan_forget(scan_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.diskscan.forget(&scan_id).await;
    Ok(())
}

use crate::AppState;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::scan::{ScanEntry, ScanTree};

    fn entry(size: u64) -> ScanEntry {
        ScanEntry {
            absolute: String::new(),
            size,
            modified: 0,
            etag: None,
        }
    }

    #[test]
    fn aggregates_sizes_up_the_tree() {
        let mut tree = ScanTree::default();
        tree.files.insert("a/b.txt".into(), entry(10));
        tree.files.insert("a/c.txt".into(), entry(20));
        tree.files.insert("d.txt".into(), entry(5));

        let root = build_tree("/data", &tree);
        assert_eq!(root.size, 35);
        assert_eq!(root.path, "/data");
        // Children sorted largest-first: dir `a` (30) before file `d.txt` (5).
        assert_eq!(root.children.len(), 2);
        assert_eq!(root.children[0].name, "a");
        assert_eq!(root.children[0].size, 30);
        assert_eq!(root.children[0].kind, FileKind::Directory);
        assert_eq!(root.children[0].path, "/data/a");
        assert_eq!(root.children[1].name, "d.txt");
        assert_eq!(root.children[1].size, 5);
        assert_eq!(root.children[1].kind, FileKind::File);

        // The dir's own children are sorted too: c.txt (20) before b.txt (10).
        let a = &root.children[0];
        assert_eq!(a.children[0].name, "c.txt");
        assert_eq!(a.children[0].size, 20);
        assert_eq!(a.children[1].name, "b.txt");
    }

    #[test]
    fn empty_tree_yields_empty_root() {
        let tree = ScanTree::default();
        let root = build_tree("/x/", &tree);
        assert_eq!(root.size, 0);
        assert!(root.children.is_empty());
        assert_eq!(root.path, "/x");
    }

    #[test]
    fn root_slash_child_paths() {
        let mut tree = ScanTree::default();
        tree.files.insert("etc/hosts".into(), entry(1));
        let root = build_tree("/", &tree);
        assert_eq!(root.path, "/");
        assert_eq!(root.children[0].path, "/etc");
        assert_eq!(root.children[0].children[0].path, "/etc/hosts");
    }

    #[test]
    fn normalizes_windows_root_prefix_to_forward_slashes() {
        let mut tree = ScanTree::default();
        // `scan::relative_of` already hands us `/`-separated relatives; only the
        // root prefix (as the local Windows backend reports it) has backslashes.
        tree.files
            .insert("Projects/Ghost FTP/x.txt".into(), entry(3));
        let root = build_tree("C:\\Users\\User\\Documents", &tree);
        assert_eq!(root.path, "C:/Users/User/Documents");
        assert_eq!(root.children[0].path, "C:/Users/User/Documents/Projects");
        assert_eq!(
            root.children[0].children[0].children[0].path,
            "C:/Users/User/Documents/Projects/Ghost FTP/x.txt"
        );
    }

    #[test]
    fn windows_drive_root_stays_listable() {
        let mut tree = ScanTree::default();
        tree.files.insert("Windows/notepad.exe".into(), entry(1));
        // A bare `C:` is the drive's working dir, not its root — keep the slash.
        let root = build_tree("C:\\", &tree);
        assert_eq!(root.path, "C:/");
        assert_eq!(root.children[0].path, "C:/Windows");
        assert_eq!(root.children[0].children[0].path, "C:/Windows/notepad.exe");
    }

    #[test]
    fn parse_find_output_matches_the_walk_shape() {
        // What `find <root> -type f -printf '%s\t%p\n'` prints — including a
        // non-zero-noise CRLF line and an unparseable line that must be skipped.
        let out = "10\t/data/a/b.txt\r\n20\t/data/a/c.txt\n5\t/data/d.txt\ngarbage line\n";
        let tree = parse_find("/data", out);
        assert_eq!(tree.files.len(), 3);
        assert_eq!(tree.files["a/b.txt"].size, 10);
        assert_eq!(tree.files["a/c.txt"].size, 20);
        assert_eq!(tree.files["d.txt"].size, 5);
        // Aggregates identically to the generic walk.
        let root = build_tree("/data", &tree);
        assert_eq!(root.size, 35);
        assert_eq!(root.children[0].name, "a");
        assert_eq!(root.children[0].size, 30);
    }

    // A real end-to-end run of the "always works" generic strategy: walk an
    // actual temp directory over LocalFs and fold it into the size tree. This is
    // the runtime observation the whole feature leans on (every backend falls
    // back to it), exercised without the Tauri/event plumbing.
    #[tokio::test]
    async fn walks_a_real_directory_into_a_size_tree() {
        use crate::remotefs::local::LocalFs;
        let base =
            std::env::temp_dir().join(format!("ghostftp_diskscan_walk_{}", std::process::id()));
        let sub = base.join("sub");
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&sub).unwrap();
        std::fs::write(base.join("top.bin"), vec![0u8; 100]).unwrap();
        std::fs::write(sub.join("inner.bin"), vec![0u8; 250]).unwrap();

        let root_str = base.to_string_lossy().to_string();
        let tree = crate::scan::walk_tree(&LocalFs, &root_str).await.unwrap();
        let node = build_tree(&root_str, &tree);

        assert_eq!(node.size, 350);
        assert_eq!(node.kind, FileKind::Directory);
        // Largest-first: the 250-byte "sub" dir leads the 100-byte file.
        assert_eq!(node.children[0].name, "sub");
        assert_eq!(node.children[0].size, 250);
        assert_eq!(node.children[0].kind, FileKind::Directory);
        assert_eq!(node.children[1].name, "top.bin");
        assert_eq!(node.children[1].size, 100);

        std::fs::remove_dir_all(&base).ok();
    }
}
