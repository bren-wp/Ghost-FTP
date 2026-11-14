# Scoped security and persistence audit — Ghost FTP 0.99.63

**Baseline:** v0.99.62 Preview, merged commit `4e690353c0cf355f722e1be21de1f0540dff4354`.  
**Audited source:** `transfer.rs`, `transfer/path_safety.rs`, `transfer/remote_directory.rs`, `transfer/download_staging.rs`, `profiles/mod.rs`, and static audit contracts.  
**Evidence:** source-level review, targeted Rust tests and repository CI (exact-head results required). This is not a full manual penetration test or a guarantee that every provider/backend has been exercised with production credentials.

| Priority | Confirmed risk | Remediation |
| --- | --- | --- |
| P1 | Directory upload relied on `DirEntry::metadata`, which follows symlinks on supported platforms, allowing an out-of-tree file or directory to masquerade as ordinary upload content. | `inspect_directory_upload_entry` uses `tokio::fs::symlink_metadata` and rejects symlinks/special files before any remote mutation. Test covers linked file and linked directory. |
| P1 | Recursive upload discarded all remote `create_dir` errors, including authentication, permission and network failures; it could queue transfers into inaccessible destinations. | `ensure_upload_directory` checks listability after `mkdir` failure, and aborts planning if directory existence cannot be verified. |
| P1 | A failed exclusive staging open must not grant path ownership; unconditional error cleanup could remove a user's pre-existing staging path. | `cleanup_failed_download_staging` recognizes `DownloadStagingOpenFailure`, skips cleanup of unowned paths and only attempts regular-file cleanup; collision and symlink tests. |
| P1 | An interrupted `profiles.json` promotion may leave the latest intact copy in `.profiles.json.<id>.previous`, while startup loads an older backup or apparently empty list. | `interrupted_profile_predecessor` finds a unique intact predecessor, loads it read-only for manual recovery and refuses unsafe symlinks/multiple ambiguous snapshots. |
| P1 | Desktop startup did not invoke `connectionsStore.loadProfiles()`; the React state defaulted to an empty list even with valid saved connection metadata on disk. | `App.tsx` now loads durable profiles during startup, displays an error on failure and uses existing request-serial safeguards to discard stale overlapping reloads. Added a source contract regression. |
| P2 | Profile persistence after close was not directly asserted in one regression. | Tests recreate `ProfileStore` from the same data directory and verify saved connection identity, settings, ordering, and rejection of plaintext credential writes. |

### Remaining verification and limitations

- Tests assert profile metadata persistence; actual reloading of OS-vault passwords and OAuth tokens must be validated in installed Windows/Linux builds under a normal user account. No credentials are embedded in the fixture or release artifacts.
- A malicious local process with write access to ancestor directories can still create TOCTOU races. Fully resolving these requires OS handle-relative filesystem operations, explicit ownership handles and adversarial fault-injection tests.
- Remote mkdir/list APIs do not universally distinguish an existing directory from a server behaving inconsistently; real-provider acceptance remains required.
- Crash/power-loss fault injection, very large transfers, interrupted multipart uploads, profile backup restoration UI, installed-machine auto-update and all cloud backend credentials require separate acceptance.
- Static dead-code scanner is heuristic; do not automatically delete dynamic Tauri command/API handlers based on its candidates.
