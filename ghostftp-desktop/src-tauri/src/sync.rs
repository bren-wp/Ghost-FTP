use crate::remotefs::{FileKind, RemoteFs};
use anyhow::Result;
use serde::{Deserialize, Serialize};

/// Direction of a one-way sync. Bidirectional is intentionally out of scope —
/// it needs conflict resolution UI that's a project of its own.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SyncDirection {
    LocalToRemote,
    RemoteToLocal,
}

/// How aggressive the sync is. Additive only copies missing/newer files;
/// Mirror also deletes files on the destination that don't exist on the
/// source. We default to Additive in the UI since Mirror is destructive.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SyncStrategy {
    Additive,
    Mirror,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SyncReason {
    Missing,     // not present on destination
    Newer,       // source mtime > destination mtime
    SizeChanged, // same name but bytes differ
    Edited,      // same size, but the sync_state index says the source changed
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncFile {
    pub relative: String,
    pub source_path: String,
    pub destination_path: String,
    pub size: u64,
    pub reason: SyncReason,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncDelete {
    pub relative: String,
    pub path: String,
    pub kind: FileKind,
    pub size: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncPlan {
    pub direction: SyncDirection,
    pub strategy: SyncStrategy,
    pub local_root: String,
    pub remote_root: String,
    pub copies: Vec<SyncFile>,
    pub deletes: Vec<SyncDelete>,
    pub total_bytes: u64,
}

fn join(base: &str, rel: &str) -> String {
    let base = base.trim_end_matches('/').trim_end_matches('\\');
    if base.is_empty() {
        return rel.to_string();
    }
    // For local Windows paths, mirror the backslash separator. Everywhere
    // else POSIX `/` is fine.
    if base.len() >= 2 && base.as_bytes()[1] == b':' {
        format!("{base}\\{}", rel.replace('/', "\\"))
    } else {
        format!("{base}/{rel}")
    }
}

/// Stateless one-shot diff (size + mtime), used by the manual Sync dialog and
/// `ghostftp-cli`. No persistent memory — a same-size in-place edit whose mtime the
/// destination doesn't reflect can be missed. Folder-sync uses [`plan_indexed`]
/// to close that gap.
pub async fn plan(
    local_fs: &dyn RemoteFs,
    remote_fs: &dyn RemoteFs,
    local_root: &str,
    remote_root: &str,
    direction: SyncDirection,
    strategy: SyncStrategy,
) -> Result<SyncPlan> {
    let (plan, _source_tree) = plan_indexed(
        local_fs,
        remote_fs,
        local_root,
        remote_root,
        direction,
        strategy,
        None,
        crate::remotefs::ChangeSignal::MtimeSize,
    )
    .await?;
    Ok(plan)
}

/// Diff with an optional `sync_state` index (the folder-sync path). When a file
/// looks up-to-date by the live size/mtime comparison, the index is consulted:
/// if the *current* source signal differs from what was recorded at last sync,
/// the file is re-copied as [`SyncReason::Edited`] — this catches same-size edits
/// the stateless diff misses. Returns the plan plus the walked **source** tree so
/// the caller can snapshot it back into the index without a second walk.
// The planner deliberately receives both roots, direction, strategy and index metadata explicitly.
#[allow(clippy::too_many_arguments)]
pub(crate) async fn plan_indexed(
    local_fs: &dyn RemoteFs,
    remote_fs: &dyn RemoteFs,
    local_root: &str,
    remote_root: &str,
    direction: SyncDirection,
    strategy: SyncStrategy,
    index: Option<&std::collections::HashMap<String, crate::db::SyncStateRow>>,
    source_signal: crate::remotefs::ChangeSignal,
) -> Result<(SyncPlan, crate::scan::ScanTree)> {
    let local_tree = crate::scan::walk_tree(local_fs, local_root).await?;
    let remote_tree = crate::scan::walk_tree(remote_fs, remote_root).await?;

    // Borrow both trees only for the diff, then move the source tree out to hand
    // back to the caller (moving while `source`/`dest` still borrow would fail).
    let (copies, deletes, total_bytes) = {
        let (source, dest, dest_root) = match direction {
            SyncDirection::LocalToRemote => (&local_tree, &remote_tree, remote_root),
            SyncDirection::RemoteToLocal => (&remote_tree, &local_tree, local_root),
        };

        let mut copies: Vec<SyncFile> = Vec::new();
        let mut total_bytes: u64 = 0;
        for (rel, src) in &source.files {
            let reason = match dest.files.get(rel) {
                None => Some(SyncReason::Missing),
                Some(d) if src.size != d.size => Some(SyncReason::SizeChanged),
                Some(d) if src.modified > d.modified && src.modified > 0 && d.modified > 0 => {
                    Some(SyncReason::Newer)
                }
                // Live diff says up-to-date. Ask the index whether the source has
                // changed since we last synced this file (same-size edit).
                Some(_) => index
                    .and_then(|ix| ix.get(rel))
                    .filter(|row| source_changed(source_signal, src, row))
                    .map(|_| SyncReason::Edited),
            };
            if let Some(reason) = reason {
                total_bytes = total_bytes.saturating_add(src.size);
                copies.push(SyncFile {
                    relative: rel.clone(),
                    source_path: src.absolute.clone(),
                    destination_path: join(dest_root, rel),
                    size: src.size,
                    reason,
                });
            }
        }

        let mut deletes: Vec<SyncDelete> = Vec::new();
        if matches!(strategy, SyncStrategy::Mirror) {
            for (rel, d) in &dest.files {
                if !source.files.contains_key(rel) {
                    deletes.push(SyncDelete {
                        relative: rel.clone(),
                        path: d.absolute.clone(),
                        kind: FileKind::File,
                        size: d.size,
                    });
                }
            }
        }
        (copies, deletes, total_bytes)
    };

    let source_tree = match direction {
        SyncDirection::LocalToRemote => local_tree,
        SyncDirection::RemoteToLocal => remote_tree,
    };

    Ok((
        SyncPlan {
            direction,
            strategy,
            local_root: local_root.to_string(),
            remote_root: remote_root.to_string(),
            copies,
            deletes,
            total_bytes,
        },
        source_tree,
    ))
}

/// True when the current source entry differs from what the index recorded at
/// last sync. For ETag backends this compares the opaque token (falling back to
/// size+mtime when either side lacks one); otherwise it's size+mtime.
fn source_changed(
    signal: crate::remotefs::ChangeSignal,
    cur: &crate::scan::ScanEntry,
    row: &crate::db::SyncStateRow,
) -> bool {
    use crate::remotefs::ChangeSignal;
    match signal {
        ChangeSignal::Etag => match (&cur.etag, &row.remote_signal) {
            (Some(a), Some(b)) => a != b,
            // Missing a token on either side → fall back to size+mtime.
            _ => cur.size != row.size || cur.modified != row.mtime,
        },
        ChangeSignal::MtimeSize | ChangeSignal::Hash => {
            cur.size != row.size || cur.modified != row.mtime
        }
    }
}

#[cfg(test)]
mod index_tests {
    use super::*;
    use crate::db::Db;
    use crate::remotefs::{local::LocalFs, ChangeSignal};
    use std::path::{Path, PathBuf};
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::time::{Duration, UNIX_EPOCH};

    static COUNTER: AtomicU32 = AtomicU32::new(0);

    fn scratch(tag: &str) -> PathBuf {
        let n = COUNTER.fetch_add(1, Ordering::Relaxed);
        let mut d = std::env::temp_dir();
        d.push(format!(
            "ghostftp_idx_test_{}_{tag}_{n}",
            std::process::id()
        ));
        let _ = std::fs::remove_dir_all(&d);
        std::fs::create_dir_all(&d).unwrap();
        d
    }

    /// Pin a file's mtime so the diff is deterministic (no sleeps). Requires
    /// write access to the handle, which is why we reopen it writable.
    fn set_mtime(p: &Path, secs: u64) {
        let f = std::fs::OpenOptions::new().write(true).open(p).unwrap();
        f.set_modified(UNIX_EPOCH + Duration::from_secs(secs))
            .unwrap();
    }

    async fn plan_stateless(src: &Path, dst: &Path) -> SyncPlan {
        let fs = LocalFs;
        plan(
            &fs,
            &fs,
            src.to_str().unwrap(),
            dst.to_str().unwrap(),
            SyncDirection::LocalToRemote,
            SyncStrategy::Additive,
        )
        .await
        .unwrap()
    }

    async fn plan_with(
        src: &Path,
        dst: &Path,
        index: &std::collections::HashMap<String, crate::db::SyncStateRow>,
    ) -> (SyncPlan, crate::scan::ScanTree) {
        let fs = LocalFs;
        plan_indexed(
            &fs,
            &fs,
            src.to_str().unwrap(),
            dst.to_str().unwrap(),
            SyncDirection::LocalToRemote,
            SyncStrategy::Additive,
            Some(index),
            ChangeSignal::MtimeSize,
        )
        .await
        .unwrap()
    }

    // The payoff, exercised end-to-end against a real SQLite Db and real files:
    // a same-size in-place edit that the stateless size+mtime diff misses (because
    // the destination's mtime is newer than the source's — the object-store
    // upload-time quirk), that the sync_state index catches, and that a seeded
    // index then declines to re-upload on the next pass (resume).
    #[test]
    fn index_catches_same_size_edit_and_resume_is_a_noop() {
        let rt = tokio::runtime::Builder::new_current_thread()
            .build()
            .unwrap();
        rt.block_on(async {
            let src = scratch("src");
            let dst = scratch("dst");
            let base = 1_000_000u64;

            // Both sides hold the same 5-byte file; the destination's mtime is
            // *newer* than the source's, so "src newer than dst" never fires.
            std::fs::write(src.join("a.txt"), "hello").unwrap();
            std::fs::write(dst.join("a.txt"), "hello").unwrap();
            set_mtime(&dst.join("a.txt"), base + 100);
            set_mtime(&src.join("a.txt"), base); // original source state

            let db = Db::open_in_memory().unwrap();
            db.upsert_sync_state("pair1", "a.txt", 5, base as i64, None, 0)
                .unwrap();

            // Edit in place: same byte count, mtime bumped but still < dest's.
            std::fs::write(src.join("a.txt"), "world").unwrap();
            set_mtime(&src.join("a.txt"), base + 1);

            // 1) Stateless diff misses it — proves the gap the index closes.
            let stateless = plan_stateless(&src, &dst).await;
            assert!(
                stateless.copies.is_empty(),
                "stateless size+mtime diff should miss a same-size edit under a newer dest mtime"
            );

            // 2) Index-aware diff catches it as an Edited copy.
            let index = db.load_sync_state("pair1").unwrap();
            let (indexed, source_tree) = plan_with(&src, &dst, &index).await;
            assert_eq!(indexed.copies.len(), 1, "index should flag the edit");
            assert_eq!(indexed.copies[0].relative, "a.txt");
            assert!(matches!(indexed.copies[0].reason, SyncReason::Edited));

            // 3) Snapshot the source back into the DB (what a real reconcile does)
            //    and re-plan: the index now matches reality → no re-upload (resume).
            for (rel, e) in &source_tree.files {
                db.upsert_sync_state("pair1", rel, e.size, e.modified, e.etag.as_deref(), 1)
                    .unwrap();
            }
            let index2 = db.load_sync_state("pair1").unwrap();
            let (resumed, _) = plan_with(&src, &dst, &index2).await;
            assert!(
                resumed.copies.is_empty(),
                "a seeded, up-to-date index must not re-upload unchanged files"
            );

            let _ = std::fs::remove_dir_all(&src);
            let _ = std::fs::remove_dir_all(&dst);
        });
    }

    // ETag backends compare the opaque token, not mtime: same size, same mtime,
    // different ETag ⇒ changed.
    #[test]
    fn etag_signal_detects_token_change() {
        let cur = crate::scan::ScanEntry {
            absolute: "/x".into(),
            size: 10,
            modified: 42,
            etag: Some("etag-B".into()),
        };
        let row = crate::db::SyncStateRow {
            size: 10,
            mtime: 42,
            remote_signal: Some("etag-A".into()),
            last_synced_ms: 0,
        };
        assert!(source_changed(ChangeSignal::Etag, &cur, &row));

        // Same token ⇒ unchanged, even if mtime drifted.
        let row_same = crate::db::SyncStateRow {
            remote_signal: Some("etag-B".into()),
            mtime: 999,
            ..row
        };
        assert!(!source_changed(ChangeSignal::Etag, &cur, &row_same));
    }
}
