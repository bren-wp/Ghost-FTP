# Ghost FTP 0.95.38 — Windows/Linux SQLite synchronization integrity audit

## Threat model and confirmed source behavior

The native desktop app is local-first and uses SQLite `ghostftp.db` with WAL mode and a private file policy. SQLite's structured transactions are useful for protecting saved indexes and settings after unexpected process termination. SQLite does **not** encrypt contents. Saved account credentials use an OS keyring; no new hosted backend is required for local operation.

**Finding 1 (high):** `foldersync::reconcile_inner` previously substituted `HashMap::default()` when SQLite `load_sync_state` failed. Replacing the authoritative source history with a fictional empty state can cause incorrect or repeated sync decisions. Fixed by propagating the database error before planning or transferring.

**Finding 2 (critical for Mirror):** `scan::walk` tolerated unreadable directory listing errors by skipping those subtrees, so missing files could be mistaken for intentional deletions. Kept permissive legacy scanning for non-destructive file-size/search views. Added `walk_tree_strict` for both local and remote synchronization inventories: any unreadable subtree aborts the scan before `execute_sync_plan` receives a plan.

**Finding 3 (high):** `snapshot_index` wrote each row independently with best-effort upserts and deletes. Mid-update SQLite faults could expose partially advanced source history. Fixed with `Db::replace_sync_state_snapshot`: one transaction, prepared statements, integer range validation and all-or-nothing semantics for a single pair.

**Finding 4 (medium):** If the Mirror empty-source or delete-count safeguard withheld destructive work, the prior code still advanced `last_synced` and rewrote the source index. The updated path preserves existing history and the timestamp when safety warnings are present. Subsequent polls will still see the unexecuted difference.

## Regression matrix

| Scenario | Assertion |
| --- | --- |
| Source directory disappears or becomes unreadable | Strict scan errors instead of returning an empty, deletion-eligible tree |
| SQLite unavailable when reading sync history | Fail closed; do not construct a synchronization plan |
| 1,000s of source-file index changes | One SQLite transaction with prepared statements, not independently committed rows |
| SQLite write fails after row deletion | Transaction rollback preserves prior complete snapshot |
| Bad file size larger than signed 64-bit SQLite integer | Reject before the transactional deletion |
| Concurrently stored unrelated sync pair | Other pair's rows remain intact |
| Mirror delete guard refuses work | No replacement index or full-success timestamp |

## Follow-up boundaries

This code audit cannot prove all Windows/Linux user flows run without errors. Real-drive NAS races, physical power cuts, 100k+ tree performance benchmarks, FTP/NAT/IPv6 and cloud-vendor multipart fixtures still require instrumented tests. File operations should remain non-destructive by default, with explicit confirmations for Mirror and remote deletion. Product parity claims require native pixel-by-pixel UI review and reproducible throughput comparison against FileZilla.
