//! Duplicate-file finder — surfaces the `name_1.ext` copies Ghost FTP's
//! rename-on-conflict policy creates (and any other exact duplicates) so they
//! can be reviewed and cleaned.
//!
//! One tree, any backend: the shared [`crate::scan`] walk flattens it, then one
//! of two groupings runs:
//!
//! * **Name mode** (default, cheap — metadata only, no byte transfer): files in
//!   the *same directory* whose names collapse to the same stem once the usual
//!   "copy" suffixes are stripped (`_1`/`_12` — Ghost FTP's rename policy, ` (1)`,
//!   ` - copy`) **and** whose sizes match are a duplicate group. This is the
//!   "Ghost FTP made a lot of `_1` files" cleanup view.
//! * **Hash mode** (`--hash`, opt-in): every same-size pair anywhere in the
//!   tree is content-hashed (server-side `sha256sum` over SSH where possible,
//!   else a streamed sha256 — the exact [`crate::diff::hash_path`] machinery the
//!   diff engine uses) and grouped by digest. Catches real duplicates with
//!   unrelated names, anywhere in the tree.
//!
//! Deletion is never implicit: the engine only *reports* groups plus a
//! suggested keeper (the unsuffixed name, else the oldest copy); each surface
//! (GUI panel, `ghostftp-cli dedupe --delete`, the `ghostftp_dedupe` bridge tool's
//! caller) decides which paths to delete and issues the deletes itself.

use std::collections::BTreeMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::sync::Mutex as StdMutex;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use anyhow::Result;
use serde::Serialize;
use tauri::{AppHandle, Emitter, State};
use tokio::sync::Mutex;
use uuid::Uuid;

use crate::remotefs::RemoteFs;
use crate::scan::{self, CancelToken, ScanOptions, ScanProgress, ScanTree};
use crate::session::Session;
use crate::AppState;

/// One file inside a duplicate group.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DedupeFile {
    pub path: String,
    pub size: u64,
    pub modified: i64,
}

/// How the groups were formed — by normalized name (cheap) or content hash.
#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum DedupeMode {
    Name,
    Hash,
}

/// A set of files believed identical. `keep` is the suggested survivor; the
/// rest are the deletable duplicates.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DedupeGroup {
    /// Display key: the normalized `dir/stem.ext` (name mode) or the short
    // hash (hash mode).
    pub key: String,
    pub size: u64,
    pub hash: Option<String>,
    pub files: Vec<DedupeFile>,
    /// Index into `files` of the suggested keeper.
    pub keep: usize,
}

#[derive(Debug, Clone, Default, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DedupeSummary {
    pub files_scanned: usize,
    pub groups: usize,
    /// Files that would be deleted keeping one per group.
    pub duplicate_files: usize,
    /// Bytes reclaimed keeping one per group.
    pub wasted_bytes: u64,
    /// Files that couldn't be hashed (hash mode only) — excluded from groups.
    pub hash_errors: usize,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DedupeResult {
    pub root: String,
    pub mode: DedupeMode,
    pub summary: DedupeSummary,
    pub groups: Vec<DedupeGroup>,
}

// ---------- Name normalization ----------

/// Split a name into (stem, extension-with-dot). `.gitignore`-style leading
/// dots are not extensions; the last dot past position 0 wins.
fn split_ext(name: &str) -> (&str, &str) {
    match name.rfind('.') {
        Some(i) if i > 0 => (&name[..i], &name[i..]),
        _ => (name, ""),
    }
}

/// Strip one round of "copy" suffixes from a stem: `_12` / ` (2)` numeric
/// tails (Ghost FTP's rename-on-conflict emits `_N`), and `- copy` / ` - copy`
/// (case-insensitive). Returns the shortened stem and whether anything was
/// stripped.
fn strip_copy_suffixes(stem: &str) -> (&str, bool) {
    let mut s = stem;
    let mut changed = false;
    loop {
        // ` (N)` paren tail.
        if let Some(open) = s.strip_suffix(')') {
            let t = open.trim_end_matches(|c: char| c.is_ascii_digit());
            if t.len() < open.len() {
                if let Some(b) = t.strip_suffix(" (") {
                    if !b.is_empty() {
                        s = b;
                        changed = true;
                        continue;
                    }
                }
            }
        }
        // Ghost FTP's `_N` tail.
        let t = s.trim_end_matches(|c: char| c.is_ascii_digit());
        if t.len() < s.len() {
            if let Some(b) = t.strip_suffix('_') {
                if !b.is_empty() {
                    s = b;
                    changed = true;
                    continue;
                }
            }
        }
        // "copy" tail, tolerating a trailing number: "… - copy 2".
        let t = s.trim_end_matches(|c: char| c.is_ascii_digit()).trim_end();
        let lower = t.to_ascii_lowercase();
        let mut stripped = false;
        for suf in [" - copy", "- copy"] {
            if lower.ends_with(suf) && t.len() > suf.len() {
                s = &t[..t.len() - suf.len()];
                changed = true;
                stripped = true;
                break;
            }
        }
        if !stripped {
            break;
        }
    }
    (s, changed)
}

/// The grouping key for name mode: `dir/base-stem.ext` with copy suffixes
/// removed from the stem. Two distinct files in one directory can only share a
/// key if at least one of them carries a suffix — which is exactly the
/// duplicate shape we're after.
fn name_key(rel: &str) -> Option<String> {
    let (dir, name) = match rel.rfind('/') {
        Some(i) => (&rel[..i], &rel[i + 1..]),
        None => ("", rel),
    };
    if name.is_empty() {
        return None;
    }
    let (stem, ext) = split_ext(name);
    let (base, _) = strip_copy_suffixes(stem);
    if base.is_empty() {
        return None;
    }
    Some(if dir.is_empty() {
        format!("{base}{ext}")
    } else {
        format!("{dir}/{base}{ext}")
    })
}

fn file_name_of(path: &str) -> &str {
    path.rsplit(['/', '\\']).next().unwrap_or(path)
}

// ---------- Grouping ----------

/// The suggested keeper: prefer an unsuffixed name (the original Ghost FTP kept),
/// then the oldest copy, then the lexicographically smallest path.
pub fn pick_keep(files: &[DedupeFile]) -> usize {
    let mut best = 0usize;
    let mut best_score: Option<(bool, i64, &str)> = None;
    for (i, f) in files.iter().enumerate() {
        let (stem, _) = split_ext(file_name_of(&f.path));
        let (_, suffixed) = strip_copy_suffixes(stem);
        // Lower is better: unsuffixed (false) first, then oldest, then name.
        let score = (suffixed, f.modified, f.path.as_str());
        if best_score.map_or(true, |b| score < b) {
            best_score = Some(score);
            best = i;
        }
    }
    best
}

fn build_result(
    root: &str,
    mode: DedupeMode,
    files_scanned: usize,
    hash_errors: usize,
    mut groups: Vec<DedupeGroup>,
) -> DedupeResult {
    // Biggest reclaim first.
    groups.sort_by(|a, b| {
        (b.size * (b.files.len() - 1) as u64)
            .cmp(&(a.size * (a.files.len() - 1) as u64))
            .then_with(|| a.key.cmp(&b.key))
    });
    let duplicate_files = groups.iter().map(|g| g.files.len() - 1).sum();
    let wasted_bytes = groups
        .iter()
        .map(|g| g.size * (g.files.len() - 1) as u64)
        .sum();
    DedupeResult {
        root: root.to_string(),
        mode,
        summary: DedupeSummary {
            files_scanned,
            groups: groups.len(),
            duplicate_files,
            wasted_bytes,
            hash_errors,
        },
        groups,
    }
}

/// Name-mode grouping, pure: same directory + same normalized name + equal
/// size. No hashing, no byte transfer.
pub fn group_by_name(tree: &ScanTree, root: &str) -> DedupeResult {
    let mut buckets: BTreeMap<String, Vec<DedupeFile>> = BTreeMap::new();
    for (rel, e) in &tree.files {
        let Some(key) = name_key(rel) else { continue };
        buckets.entry(key).or_default().push(DedupeFile {
            path: e.absolute.clone(),
            size: e.size,
            modified: e.modified,
        });
    }
    let groups = buckets
        .into_iter()
        .filter_map(|(key, files)| {
            if files.len() < 2 || !files.iter().all(|f| f.size == files[0].size) {
                return None;
            }
            let keep = pick_keep(&files);
            Some(DedupeGroup { key, size: files[0].size, hash: None, files, keep })
        })
        .collect();
    build_result(root, DedupeMode::Name, tree.files.len(), 0, groups)
}

/// Hash-mode grouping: bucket by size (0-byte files skipped — they're all
/// "equal" and never worth cleaning), hash every file in a multi-file bucket,
/// group by digest. Files that fail to hash are counted and left out.
pub async fn group_by_hash(
    tree: &ScanTree,
    root: &str,
    session: Option<&Session>,
    cancel: &CancelToken,
    mut on_hashed: impl FnMut(usize),
) -> DedupeResult {
    let mut by_size: BTreeMap<u64, Vec<&crate::scan::ScanEntry>> = BTreeMap::new();
    for e in tree.files.values() {
        if e.size > 0 {
            by_size.entry(e.size).or_default().push(e);
        }
    }

    let mut by_hash: BTreeMap<String, Vec<DedupeFile>> = BTreeMap::new();
    let mut hash_errors = 0usize;
    let mut hashed = 0usize;
    'outer: for (size, entries) in by_size.iter().filter(|(_, v)| v.len() >= 2) {
        for e in entries {
            if cancel.is_cancelled() {
                break 'outer;
            }
            match crate::diff::hash_path(session, &e.absolute).await {
                Ok(h) => by_hash.entry(h).or_default().push(DedupeFile {
                    path: e.absolute.clone(),
                    size: *size,
                    modified: e.modified,
                }),
                Err(_) => hash_errors += 1,
            }
            hashed += 1;
            on_hashed(hashed);
        }
    }

    let groups = by_hash
        .into_iter()
        .filter_map(|(h, files)| {
            if files.len() < 2 {
                return None;
            }
            let keep = pick_keep(&files);
            let key = format!("{}…", &h[..12]);
            Some(DedupeGroup { key, size: files[0].size, hash: Some(h), files, keep })
        })
        .collect();
    build_result(root, DedupeMode::Hash, tree.files.len(), hash_errors, groups)
}

/// Walk `root` and group its duplicates. The one-shot entry point for
/// `ghostftp-cli dedupe` and the `ghostftp_dedupe` bridge tool; the GUI drives its own
/// cancellable run via [`DedupeManager`].
pub async fn find_duplicates(
    fs: &dyn RemoteFs,
    root: &str,
    session: Option<&Session>,
    mode: DedupeMode,
) -> Result<DedupeResult> {
    let tree = scan::walk_tree(fs, root).await?;
    Ok(match mode {
        DedupeMode::Name => group_by_name(&tree, root),
        DedupeMode::Hash => group_by_hash(&tree, root, session, &CancelToken::new(), |_| {}).await,
    })
}

/// The deletable paths across all groups (everything but each group's keeper).
pub fn duplicate_paths(result: &DedupeResult) -> Vec<String> {
    result
        .groups
        .iter()
        .flat_map(|g| {
            g.files
                .iter()
                .enumerate()
                .filter(|(i, _)| *i != g.keep)
                .map(|(_, f)| f.path.clone())
        })
        .collect()
}

// ---------- GUI runner ----------
//
// Shaped exactly like `diff::DiffManager`: an `Arc<DedupeManager>` in
// `AppState`, one run per id, progress over `dedupe://…` events, cancel via a
// shared `CancelToken`, the result fetched once on completion.

#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum DedupeRunState {
    Scanning,
    Done,
    Error,
    Canceled,
}

#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum DedupePhase {
    Walking,
    Hashing,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DedupeSnapshot {
    pub id: String,
    pub session_id: String,
    pub path: String,
    pub mode: DedupeMode,
    pub state: DedupeRunState,
    pub phase: DedupePhase,
    pub files_found: usize,
    pub hashed: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<DedupeResult>,
    pub started_at: i64,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct DedupeProgressEvent {
    id: String,
    phase: DedupePhase,
    files_found: usize,
    hashed: usize,
}

enum DedupeOutcome {
    Running,
    Done(DedupeResult),
    Error(String),
    Canceled,
}

struct DedupeRun {
    id: String,
    session_id: String,
    path: String,
    mode: DedupeMode,
    phase: StdMutex<DedupePhase>,
    files_found: AtomicUsize,
    hashed: AtomicUsize,
    cancel: CancelToken,
    started_at: i64,
    outcome: StdMutex<DedupeOutcome>,
}

impl DedupeRun {
    fn snapshot(&self) -> DedupeSnapshot {
        let (state, error, result) = match &*self.outcome.lock().unwrap() {
            DedupeOutcome::Running => (DedupeRunState::Scanning, None, None),
            DedupeOutcome::Done(r) => (DedupeRunState::Done, None, Some(r.clone())),
            DedupeOutcome::Error(e) => (DedupeRunState::Error, Some(e.clone()), None),
            DedupeOutcome::Canceled => (DedupeRunState::Canceled, None, None),
        };
        DedupeSnapshot {
            id: self.id.clone(),
            session_id: self.session_id.clone(),
            path: self.path.clone(),
            mode: self.mode,
            state,
            phase: *self.phase.lock().unwrap(),
            files_found: self.files_found.load(Ordering::Relaxed),
            hashed: self.hashed.load(Ordering::Relaxed),
            error,
            result,
            started_at: self.started_at,
        }
    }

    fn set_phase(&self, p: DedupePhase) {
        *self.phase.lock().unwrap() = p;
    }
}

struct DedupeHandle {
    info: Arc<DedupeRun>,
    task: tauri::async_runtime::JoinHandle<()>,
}

pub struct DedupeManager {
    runs: Mutex<std::collections::HashMap<String, DedupeHandle>>,
}

impl Default for DedupeManager {
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

impl DedupeManager {
    pub fn new() -> Self {
        Self { runs: Mutex::new(std::collections::HashMap::new()) }
    }

    /// Kick off a duplicate scan of one side. Returns the run id immediately;
    /// progress streams over `dedupe://progress` and the run settles on one of
    /// `dedupe://done` / `dedupe://error` / `dedupe://canceled`.
    pub async fn start(
        self: &Arc<Self>,
        session_id: String,
        path: String,
        mode: DedupeMode,
        fs: Box<dyn RemoteFs>,
        sess: Option<Arc<Session>>,
        app: AppHandle,
    ) -> String {
        let id = Uuid::new_v4().to_string();
        let info = Arc::new(DedupeRun {
            id: id.clone(),
            session_id,
            path,
            mode,
            phase: StdMutex::new(DedupePhase::Walking),
            files_found: AtomicUsize::new(0),
            hashed: AtomicUsize::new(0),
            cancel: CancelToken::new(),
            started_at: now_ts(),
            outcome: StdMutex::new(DedupeOutcome::Running),
        });

        let task_info = Arc::clone(&info);
        let task = tauri::async_runtime::spawn(async move {
            run_dedupe(task_info, fs, sess, app).await;
        });

        self.runs.lock().await.insert(id.clone(), DedupeHandle { info, task });
        id
    }

    pub async fn snapshot(&self, id: &str) -> Option<DedupeSnapshot> {
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

fn emit_progress(info: &DedupeRun, app: &AppHandle) {
    let _ = app.emit(
        "dedupe://progress",
        DedupeProgressEvent {
            id: info.id.clone(),
            phase: *info.phase.lock().unwrap(),
            files_found: info.files_found.load(Ordering::Relaxed),
            hashed: info.hashed.load(Ordering::Relaxed),
        },
    );
}

async fn run_dedupe(
    info: Arc<DedupeRun>,
    fs: Box<dyn RemoteFs>,
    sess: Option<Arc<Session>>,
    app: AppHandle,
) {
    info.set_phase(DedupePhase::Walking);
    emit_progress(&info, &app);

    let opts = ScanOptions {
        concurrency: scan::DEFAULT_CONCURRENCY,
        cancel: info.cancel.clone(),
    };
    let mut last_emit = Instant::now();
    let ev_info = Arc::clone(&info);
    let ev_app = app.clone();
    let on_progress = move |p: ScanProgress| {
        ev_info.files_found.store(p.files_found, Ordering::Relaxed);
        if last_emit.elapsed() >= Duration::from_millis(100) {
            emit_progress(&ev_info, &ev_app);
            last_emit = Instant::now();
        }
    };

    let tree = match scan::walk(fs.as_ref(), &info.path, &opts, on_progress).await {
        Ok(t) => t,
        Err(e) => {
            *info.outcome.lock().unwrap() =
                DedupeOutcome::Error(format!("walking {}: {e:#}", info.path));
            let _ = app.emit("dedupe://error", info.snapshot());
            return;
        }
    };
    if info.cancel.is_cancelled() {
        *info.outcome.lock().unwrap() = DedupeOutcome::Canceled;
        let _ = app.emit("dedupe://canceled", info.snapshot());
        return;
    }

    let result = match info.mode {
        DedupeMode::Name => group_by_name(&tree, &info.path),
        DedupeMode::Hash => {
            info.set_phase(DedupePhase::Hashing);
            emit_progress(&info, &app);
            let h_info = Arc::clone(&info);
            let h_app = app.clone();
            let mut last_hash_emit = Instant::now();
            let r = group_by_hash(&tree, &info.path, sess.as_deref(), &info.cancel, move |n| {
                h_info.hashed.store(n, Ordering::Relaxed);
                if last_hash_emit.elapsed() >= Duration::from_millis(100) {
                    emit_progress(&h_info, &h_app);
                    last_hash_emit = Instant::now();
                }
            })
            .await;
            if info.cancel.is_cancelled() {
                *info.outcome.lock().unwrap() = DedupeOutcome::Canceled;
                let _ = app.emit("dedupe://canceled", info.snapshot());
                return;
            }
            r
        }
    };

    *info.outcome.lock().unwrap() = DedupeOutcome::Done(result);
    let _ = app.emit("dedupe://done", info.snapshot());
}

// ---------- Tauri commands ----------

/// Start a duplicate scan. `session_id` is a session id or the local sentinel.
/// Returns the run id; the frontend listens for `dedupe://…` and fetches the
/// result on `done`.
#[tauri::command]
pub async fn dedupe_start(
    session_id: String,
    path: String,
    hash: bool,
    app: AppHandle,
    state: State<'_, AppState>,
) -> Result<String, String> {
    let (fs, sess) = crate::diff::resolve_diff_side(&session_id, &state).await?;
    let mode = if hash { DedupeMode::Hash } else { DedupeMode::Name };
    let mgr = Arc::clone(&state.dedupe);
    Ok(mgr.start(session_id, path, mode, fs, sess, app).await)
}

#[tauri::command]
pub async fn dedupe_status(
    dedupe_id: String,
    state: State<'_, AppState>,
) -> Result<DedupeSnapshot, String> {
    state
        .dedupe
        .snapshot(&dedupe_id)
        .await
        .ok_or_else(|| format!("dedupe {dedupe_id} not found"))
}

/// Full snapshot including the `result` (present once the scan is done).
#[tauri::command]
pub async fn dedupe_result(
    dedupe_id: String,
    state: State<'_, AppState>,
) -> Result<DedupeSnapshot, String> {
    state
        .dedupe
        .snapshot(&dedupe_id)
        .await
        .ok_or_else(|| format!("dedupe {dedupe_id} not found"))
}

#[tauri::command]
pub async fn dedupe_cancel(dedupe_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.dedupe.cancel(&dedupe_id).await;
    Ok(())
}

#[tauri::command]
pub async fn dedupe_forget(dedupe_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state.dedupe.forget(&dedupe_id).await;
    Ok(())
}

/// Delete an explicit list of paths on one side (the GUI's checked duplicates).
/// Returns the per-path errors; an empty list means everything deleted.
#[tauri::command]
pub async fn dedupe_delete(
    session_id: String,
    paths: Vec<String>,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    let (fs, _) = crate::diff::resolve_diff_side(&session_id, &state).await?;
    let mut errors = Vec::new();
    for p in paths {
        if let Err(e) = fs.delete(&p, false).await {
            errors.push(format!("{p}: {e}"));
        }
    }
    Ok(errors)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::scan::ScanEntry;

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
    fn strips_copy_suffixes() {
        assert_eq!(strip_copy_suffixes("photo_1"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo_12"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo (1)"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo (12)"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo - copy"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo - Copy"), ("photo", true));
        // Chained: "photo - copy 2" → digits → "photo - copy" → "photo".
        assert_eq!(strip_copy_suffixes("photo - copy 2"), ("photo", true));
        assert_eq!(strip_copy_suffixes("photo - copy_2"), ("photo", true));
        // Not suffixes:
        assert_eq!(strip_copy_suffixes("photo"), ("photo", false));
        assert_eq!(strip_copy_suffixes("photo_v2"), ("photo_v2", false)); // version, not a copy
        assert_eq!(strip_copy_suffixes("2024_report"), ("2024_report", false));
        assert_eq!(strip_copy_suffixes("my photo"), ("my photo", false));
        assert_eq!(strip_copy_suffixes("_1"), ("_1", false)); // never to empty
    }

    #[test]
    fn name_key_normalizes() {
        assert_eq!(name_key("img/logo_1.png").as_deref(), Some("img/logo.png"));
        assert_eq!(name_key("logo (2).png").as_deref(), Some("logo.png"));
        assert_eq!(name_key("a/b/c.txt").as_deref(), Some("a/b/c.txt"));
        assert_eq!(name_key(".gitignore").as_deref(), Some(".gitignore"));
    }

    #[test]
    fn name_mode_groups_same_dir_equal_size() {
        let t = tree(&[
            ("img/logo.png", entry("/r/img/logo.png", 100, 100)),
            ("img/logo_1.png", entry("/r/img/logo_1.png", 100, 200)),
            ("img/logo_2.png", entry("/r/img/logo_2.png", 100, 300)),
            // Same name shape but a different size → not a group member.
            ("img/other.png", entry("/r/img/other.png", 100, 100)),
            ("img/other_1.png", entry("/r/img/other_1.png", 999, 100)),
            // Different directory → different group scope.
            ("docs/logo_1.png", entry("/r/docs/logo_1.png", 100, 100)),
            ("solo.txt", entry("/r/solo.txt", 5, 0)),
        ]);
        let r = group_by_name(&t, "/r");
        assert_eq!(r.groups.len(), 1);
        let g = &r.groups[0];
        assert_eq!(g.key, "img/logo.png");
        assert_eq!(g.files.len(), 3);
        // Keeper is the unsuffixed original.
        assert_eq!(g.files[g.keep].path, "/r/img/logo.png");
        assert_eq!(r.summary.duplicate_files, 2);
        assert_eq!(r.summary.wasted_bytes, 200);
        assert_eq!(r.summary.files_scanned, 7);
    }

    #[test]
    fn pick_keep_prefers_unsuffixed_then_oldest() {
        let files = vec![
            DedupeFile { path: "/r/a_1.txt".into(), size: 1, modified: 50 },
            DedupeFile { path: "/r/a.txt".into(), size: 1, modified: 999 },
        ];
        assert_eq!(pick_keep(&files), 1); // unsuffixed beats older
        let all_suffixed = vec![
            DedupeFile { path: "/r/a_2.txt".into(), size: 1, modified: 60 },
            DedupeFile { path: "/r/a_1.txt".into(), size: 1, modified: 50 },
        ];
        assert_eq!(pick_keep(&all_suffixed), 1); // oldest wins
    }

    #[test]
    fn duplicate_paths_excludes_keepers() {
        let t = tree(&[
            ("a.txt", entry("/r/a.txt", 10, 1)),
            ("a_1.txt", entry("/r/a_1.txt", 10, 2)),
        ]);
        let r = group_by_name(&t, "/r");
        assert_eq!(duplicate_paths(&r), vec!["/r/a_1.txt".to_string()]);
    }

    // Hash mode end-to-end against LocalFs: same-size files with equal content
    // group together regardless of name; a same-size impostor does not.
    #[tokio::test]
    async fn hash_mode_groups_by_content() {
        let base = std::env::temp_dir().join(format!("ghostftp_dedupe_{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(base.join("sub")).unwrap();
        std::fs::write(base.join("a.txt"), "hello").unwrap();
        std::fs::write(base.join("sub/rename-me.txt"), "hello").unwrap(); // same bytes, other name/dir
        std::fs::write(base.join("b.txt"), "world").unwrap(); // same size, other content
        std::fs::write(base.join("empty.txt"), "").unwrap(); // 0-byte: skipped
        std::fs::write(base.join("empty_1.txt"), "").unwrap();

        let tree = scan::walk_tree(&crate::remotefs::local::LocalFs, base.to_str().unwrap())
            .await
            .unwrap();
        let r = group_by_hash(&tree, base.to_str().unwrap(), None, &CancelToken::new(), |_| {}).await;

        assert_eq!(r.groups.len(), 1);
        let g = &r.groups[0];
        assert_eq!(g.files.len(), 2);
        assert!(g.hash.as_deref().map_or(false, |h| h.len() == 64));
        assert_eq!(r.summary.hash_errors, 0);
        assert_eq!(r.summary.wasted_bytes, 5);

        std::fs::remove_dir_all(&base).ok();
    }
}
