//! Shared scan engine — the one place that walks a [`RemoteFs`] tree
//! (see `docs/plans/3_scan-index-foundation.md`).
//!
//! Extracted from `sync.rs::walk`. Folder-sync's reconciler, disk usage
//! (Plan 4), diff (Plan 6), and search (Plan 7) all call **this** instead of
//! rolling their own recursion. Latency — not CPU — dominates on SFTP/FTP, so
//! the generic walk lists many directories concurrently (bounded) rather than
//! one round-trip at a time.
//!
//! Two protocol fast paths named in the plan (an exec `du`/`find` for
//! SSH/agent, and an object-store flat listing for S3/Azure) will land with
//! their first consumer (Plan 4). The generic walk here always works and is the
//! fallback for all of them; the strategy hook is the [`ScanOptions`] the caller
//! passes.

use std::collections::{BTreeMap, VecDeque};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

use anyhow::Result;
use futures::stream::{FuturesUnordered, StreamExt};

use crate::remotefs::{FileKind, RemoteFs};

/// Default in-flight `list_dir` fan-out. High enough to hide per-request latency
/// on SFTP/FTP, low enough not to swamp a server with parallel connections.
pub const DEFAULT_CONCURRENCY: usize = 16;

/// A cheap, cloneable cooperative-cancel flag. Mirrors the abort handles the
/// `TransferManager` / `agent_host` use, without pulling in a runtime dep.
#[derive(Clone, Default)]
pub struct CancelToken(Arc<AtomicBool>);

impl CancelToken {
    pub fn new() -> Self {
        Self::default()
    }
    pub fn cancel(&self) {
        self.0.store(true, Ordering::Relaxed);
    }
    pub fn is_cancelled(&self) -> bool {
        self.0.load(Ordering::Relaxed)
    }
}

/// Progress emitted after each directory listing completes. Consumers (Plan 4's
/// disk-usage tab) turn this into `scan://…` events; the sync poller ignores it.
#[derive(Debug, Clone, Copy)]
pub struct ScanProgress {
    pub dirs_scanned: usize,
    pub files_found: usize,
    /// Running total of every file byte seen so far. Disk usage (Plan 4) shows
    /// this live ("1.2 GB found"); the sync poller ignores it.
    pub bytes_found: u64,
}

/// One file discovered by the walk. `modified` is normalised to `0` when the
/// backend didn't report an mtime (matching the old `sync.rs` walk). `etag` is
/// the backend's opaque change token when it has one (object stores).
#[derive(Debug, Clone)]
pub struct ScanEntry {
    pub absolute: String,
    pub size: u64,
    pub modified: i64,
    pub etag: Option<String>,
}

/// A flattened file tree keyed by relative path (POSIX `/`) from the scan root.
/// Directories are implicit — only files are tracked, exactly as sync needs.
#[derive(Debug, Default, Clone)]
pub struct ScanTree {
    pub files: BTreeMap<String, ScanEntry>,
}

/// Knobs for a walk. `Default` is what the sync poller wants: full concurrency,
/// never cancelled.
#[derive(Clone)]
pub struct ScanOptions {
    pub concurrency: usize,
    pub cancel: CancelToken,
}

impl Default for ScanOptions {
    fn default() -> Self {
        Self { concurrency: DEFAULT_CONCURRENCY, cancel: CancelToken::new() }
    }
}

/// Convenience wrapper for the common case (default options, no progress).
pub async fn walk_tree(fs: &dyn RemoteFs, root: &str) -> Result<ScanTree> {
    walk(fs, root, &ScanOptions::default(), |_| {}).await
}

/// Recursively list every file under `root`. Bounded-concurrency BFS: up to
/// `opts.concurrency` `list_dir` calls are in flight at once. Unreadable
/// directories are tolerated (skipped), symlinks are not followed — both
/// preserving the behaviour of the walk this replaced. `on_progress` fires once
/// per completed directory listing.
pub async fn walk<F: FnMut(ScanProgress)>(
    fs: &dyn RemoteFs,
    root: &str,
    opts: &ScanOptions,
    mut on_progress: F,
) -> Result<ScanTree> {
    let normalized_root = root.trim_end_matches('/').to_string();
    let limit = opts.concurrency.max(1);

    let mut tree = ScanTree::default();
    let mut queue: VecDeque<String> = VecDeque::new();
    queue.push_back(root.to_string());
    let mut inflight = FuturesUnordered::new();
    let mut dirs_scanned = 0usize;
    let mut bytes_found = 0u64;

    loop {
        if opts.cancel.is_cancelled() {
            break;
        }
        // Keep the pipeline full: top up in-flight listings from the queue.
        while inflight.len() < limit {
            match queue.pop_front() {
                Some(dir) => inflight.push(async move {
                    let res = fs.list_dir(&dir).await;
                    res
                }),
                None => break,
            }
        }
        // Empty queue *and* nothing in flight ⇒ the tree is fully walked.
        let Some(res) = inflight.next().await else {
            break;
        };
        dirs_scanned += 1;
        if let Ok(entries) = res {
            for entry in entries {
                match entry.kind {
                    FileKind::Directory => queue.push_back(entry.path.clone()),
                    FileKind::File => {
                        bytes_found += entry.size;
                        let rel = relative_of(&normalized_root, &entry.path);
                        tree.files.insert(
                            rel,
                            ScanEntry {
                                absolute: entry.path.clone(),
                                size: entry.size,
                                modified: entry.modified.unwrap_or(0),
                                etag: entry.etag.clone(),
                            },
                        );
                    }
                    _ => {} // skip symlinks/other — don't chase cycles
                }
            }
        }
        on_progress(ScanProgress { dirs_scanned, files_found: tree.files.len(), bytes_found });
    }

    Ok(tree)
}

/// The path of `full` relative to `root`, POSIX-normalised. Shared with `sync.rs`.
pub fn relative_of(root: &str, full: &str) -> String {
    let stripped = full.strip_prefix(root).unwrap_or(full);
    stripped
        .trim_start_matches(|c| c == '/' || c == '\\')
        .replace('\\', "/")
}
