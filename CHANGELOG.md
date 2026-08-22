# Changelog

## 1.0.12 — English-first localization and production cleanup

**Focus:** make English the canonical product/repository language, add tested runtime localization, reduce duplicated user-facing text, and continue the Windows UI/stability cleanup without changing the FTP/FTPS/SFTP security model.

- English is the canonical/default runtime language and persisted locale fallback.
- Added runtime catalogs for Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi and Japanese.
- Added catalog parity and formatting-placeholder regression tests so missing or incompatible translations fail CI instead of silently mixing languages.
- Added a live Windows language selector that updates owner-drawn buttons, cues, protocol labels, list columns and localized transfer/file rendering without restarting.
- Windows layout reserves space for longer translated labels and keeps the language selector available on smaller laptop widths.
- Settings now persist the selected locale and migrate legacy files without a language field to English.
- End-user error mapping is centralized and locale-aware instead of duplicating Croatian-only protocol/tool messages at each call site.
- Repository README, release history and build/audit policy are being standardized on English as the canonical source while localized user documentation remains available separately.
- `v1.0.11` remains immutable; 1.0.12 is a new semantic release line.

## 1.0.11 — Responsive Windows UI and safer session state

- Made the Windows layout adapt to actual client area and compact connection fields on narrower windows.
- Rebalanced local, remote and transfer list columns according to available width.
- Preserved file and transfer selections across automatic refreshes when the selected items still exist.
- Temporarily disabled list redraw while repopulating to reduce flicker.
- Derived action-button state from real selection, connection and transfer status instead of allowing invalid clicks and reporting errors afterwards.
- Bound connection, retry and health-check callbacks to a connection generation so stale callbacks cannot modify a newer session.
- Blocked profile/settings actions while a connection attempt is active and validated endpoint fields before processing typed credentials.
- Prevented an old-session health check from disconnecting a newly reconnected session.
- Added honest partial-success accounting for remote multi-delete and CHMOD operations and refreshed the remote view whenever at least one mutation succeeded.
- Skipped symlinks for Windows CHMOD as defense in depth.
- Synced Pause/Resume UI state to the transfer manager's authoritative paused state.
- Added regression coverage for batch accounting, action-state derivation, selection preservation and session-generation ordering.

## 1.0.10 — External process lifecycle hardening

- Linux/macOS network/helper processes run in a dedicated process group so cancellation terminates descendants rather than only the direct process.
- Windows cancellation snapshots and recursively terminates descendants of curl/OpenSSH processes, including potential AskPass helpers.
- Kept Windows `CREATE_NO_WINDOW` behavior while preserving handles required for SSH AskPass.
- Applied the shared lifecycle configuration to curl, curl capability probes, ssh-keyscan, ssh-keygen and sftp.
- Added a cross-platform functional test that creates a real parent/child process tree and proves the descendant does not survive cancellation long enough to leave its marker.
- Added source-level regressions that prevent future remote process call sites from bypassing lifecycle protection.

## 1.0.9 — Windows installer transaction hardening

- Bound upgrade backup to the same opened filesystem object verified with `os.SameFile` instead of trusting a separate check-then-open path.
- Re-read and SHA-256 verified the same backup source handle to detect in-place changes during backup.
- Revalidated identity, metadata and content before activating an upgrade.
- Used no-replace activation for fresh installs so a newly appearing target is not overwritten.
- Tracked whether the installer actually activated its own target before rollback may delete or replace anything.
- Verified installed object identity/content before rollback and verified restored backup content afterwards.
- Added deterministic tests for path replacement, fresh-target races, pre-activation rollback and tampering before rollback.

## 1.0.8 — Fail-closed remote cleanup and transfer lifecycle

- Propagated FTP/FTPS and SFTP staging-cleanup failures instead of hiding them behind ordinary transfer errors.
- Added an explicit uncertain-remote-state error and blocked automatic retry while a previous temporary/rollback object may remain.
- Prevented cleanup uncertainty from being downgraded to skipped/cancelled status.
- Reported post-commit rollback cleanup failure instead of returning false full success or retrying the overwrite.
- Added defense-in-depth validation that a single-file remote target is a concrete file path in the queue and both network adapters.
- Cleaned local download staging after a final no-replace activation failure.
- Replaced per-timeout waiter goroutines with a shared worker-idle signal.

## 1.0.7 — SFTP RSA/SHA-2 and stricter transfer boundaries

- Stopped forcing scanned RSA host keys into `HostKeyAlgorithms ssh-rsa`; the same pinned RSA public key can use modern RSA/SHA-2 signatures negotiated by OpenSSH.
- Kept Ed25519/ECDSA host-key constraints and SHA-256 fingerprint pinning.
- Rejected a visible remote upload staging entry when it is a directory or symlink.
- Preserved compatibility with legacy FTP LIST servers that hide dotfiles.
- Made Linux/macOS use native `curl` rather than preferring `curl.exe`.
- Required single-file upload/download requests to target a concrete remote file; directory-tree transfer remains a separate operation.

## 1.0.6 — Runtime and connection lifecycle cleanup

- Removed redundant raw-session prechecks before queueing/retrying transfers.
- Removed the raw remote `Session()` getter so new code uses operation-scoped session access.
- Normalized nil contexts across remote connection/disconnection and worker-wait paths.
- Centralized FTP/SFTP probe-path behavior.
- Deduplicated transfer ID normalization/validation.
- Switched read-only transfer-manager operations to `RWMutex` read locking.
- Centralized typed engine panic recovery and added lifecycle regressions.

## 1.0.5 — Shared-hosting compatibility

- Made raw FTP control commands use the login/home-relative namespace consistently with listing and upload/download.
- Avoided unnecessary data-channel transfers after control-only FTP operations such as mkdir/rename/delete/CHMOD.
- Added MLSD → LIST fallback and remembered the working listing mode for the session.
- Added shared-hosting process-smoke tests for email-style usernames and `public_html` semantics.
- Improved user-facing handling of common FTP 530, 421 and 425/426 failures.

## 1.0.4 — Transfer generation binding

- Captured transfer generation before connection-identity lookup and revalidated it before queue mutation.
- Prevented a stale connection identity from being combined with a newer transfer generation during reserve/retry races.
- Kept connection identity lookup outside the transfer-manager lock to avoid lock-order coupling.
- Added deterministic generation-race tests.

## 1.0.3 — Stable local upload snapshot

- Opened and identity-checked the upload source, copied it byte-for-byte into a private random temporary snapshot and passed only that snapshot to the network child process.
- SHA-256 verified source stability while creating the snapshot and verified snapshot identity/content after the network read.
- Blocked path replacement and same-size/same-mtime tampering before remote commit.
- Used no-follow cleanup for the private snapshot directory.

## 1.0.2 — Remote commit revalidation

- Re-listed the destination directory after upload staging and immediately before the final commit/rename.
- Blocked directory/symlink target replacement and respected a fresh SkipExisting decision.
- Used the fresh target snapshot for overwrite backup/rollback decisions.
- Failed closed and cleaned staging when final revalidation failed.
- Shared the same revalidation helper between FTP/FTPS and SFTP.

## 1.0.1 — Filesystem and SFTP hardening

- Replaced non-atomic destination-check/rename fallbacks with no-replace platform primitives where available.
- Hardened recursive local deletion around stable directory handles and identity revalidation.
- Added a size limit and stable private snapshot for SFTP private keys.
- Removed unsafe shared startup cleanup of SFTP temporary artifacts on Unix and moved Windows cleanup behind redirect-directory validation.

## 1.0.0 — Stable production baseline

- Established `VERSION` as the single semantic version source for runtime binaries, platform packages, GitHub Release and `ByFTP.Windows` package metadata.
- Hardened host validation, including bracketed IPv6 handling.
- Kept fail-closed SFTP host-key pinning and endpoint-bound saved credentials.
- Kept Windows DPAPI profile secrets and fail-closed unsupported Unix SFTP password/passphrase modes.
- Preserved bounded transfer queue/event history, connection-identity retry protection and reference-counted remote session close behavior.
- Preserved path traversal, symlink/junction/reparse validation and external process environment/output restrictions.
- Required Windows x64/x86, Linux amd64/arm64/i386 and macOS Universal production build gates.
- Kept release publication serialized, allowlisted and digest-verified.

### Pre-1.0 development

Before the stable 1.0.0 line, the project used internal development versioning during architecture, packaging and security-hardening work. The detailed pre-1.0 history remains available in Git history; this changelog tracks the public stable semantic line from 1.0.0 onward.
