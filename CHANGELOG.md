# Changelog

## 1.1.1 — Android APK distribution and maintenance hardening

**Focus:** make Android APK artifacts part of the verified release contract, improve shared-hosting compatibility, clean Android UI state and finish English-first Windows startup fallbacks without weakening existing desktop gates.

- Added versioned Android debug and optimized unsigned release APK artifacts to CI and the production GitHub Release workflow.
- Added `scripts/package_android.py` to validate APK ZIP structure, required Android entries, path safety and deterministic versioned staging names.
- Added regression coverage for Android APK staging, malformed APKs, unsafe ZIP members and invalid semantic versions.
- Production release staging now requires 12 platform artifacts before shared metadata: six Windows packages, three Linux DEBs, one macOS PKG and two Android APKs.
- Android APKs are included in `SHA256.txt`, build metadata and the centralized fail-closed GitHub Release publisher.
- The debug APK is explicitly a debug-signed installable development/test build; the optimized release APK is deliberately unsigned and requires an external private production signing identity before production distribution.
- Android CI and release gates now run both `lintDebug` and `lintRelease`, plus `assembleDebug` and `assembleRelease`.
- Hardened Android FTP/FTPS shared-hosting paths so the UI root maps to the authenticated login working directory instead of forcing an unrelated server filesystem root.
- Added login-root path tests covering `public_html`, unavailable `PWD`, virtual-root servers and traversal/noncanonical path rejection.
- Cleared pending Android download-picker state on every download result, disconnect and Activity destruction so stale remote paths cannot survive cancelled or incomplete picker results.
- Simplified Android long-press file actions to use stable action indexes instead of comparing localized labels.
- Standardized remaining Windows startup and AskPass fallback messages on English and added localization audit coverage that rejects a return of the retired Croatian-only fallbacks.
- Expanded Android, localization and release audits so the new path, picker-state and APK-publication guarantees fail closed in CI.
- Updated README, Android documentation, installation, security, testing, release verification, GitHub Releases and release-note generation for the 1.1.1 behavior.

## 1.1.0 — Native Android client and mobile release gate

**Focus:** add a real native Android FTP/FTPS/SFTP client while keeping the existing desktop security, privacy and release contracts fail-closed.

- Added an isolated native Android application under `android/` rather than embedding a web view or hidden local service.
- Added FTP, explicit FTPS, implicit FTPS and SFTP connections for Android.
- Added passive/binary FTP behavior and FTPS private data-channel protection.
- Android FTPS now explicitly uses the Android/JVM platform trust manager plus endpoint/hostname certificate checking; permissive/custom trust managers are rejected by the mobile source audit.
- Made the expected OpenSSH-style `SHA256:` host-key fingerprint mandatory for Android SFTP and delegated verification to SSHJ's native fingerprint verifier.
- Added remote directory browsing/navigation, refresh, upload, download, create directory, rename and delete operations.
- Added Android Storage Access Framework upload/download so the application does not request broad filesystem/storage permission.
- Kept Android passwords session-only; the mobile client does not write passwords or SSH secrets to preferences, databases, files or a ByFTP backend.
- Added Android backup and device-transfer exclusion rules for root, file, database, shared-preference and external app-data domains.
- Disabled generic cleartext traffic for Android platform-aware networking while retaining explicit plain FTP only as a user-selected compatibility protocol.
- Hardened Android Activity destruction: pending/active remote clients are tracked and closed, executor work is interrupted and late main-thread callbacks are ignored.
- Removed dead Android UI/resources and enabled release resource shrinking alongside code minification.
- Deliberately deferred Android private-key import until Android Keystore-backed key handling and migration semantics are implemented and audited.
- Added Android connection/path/security/version tests plus static mobile security/privacy/lifecycle invariants.
- Added a dedicated Android CI job using JDK 17, Gradle 9.5.0, Android Gradle Plugin 9.3.0 and API 37.
- Android CI runs JUnit, lint with warnings treated as errors and debug APK compilation; lint reports and APKs are retained as validation evidence.
- Bound Android `versionName` and `versionCode` to the repository root `VERSION` source.
- Added Android source validation as a production release gate; a public production APK is intentionally withheld until a stable private signing identity exists.
- Updated root documentation, Android documentation, installation, architecture, security, privacy, testing and release documentation for the new platform.

## 1.0.13 — Release pipeline and packaging finalization

**Focus:** publish the completed English-first 1.0.x line with reproducible cross-platform assets, stricter release verification and verified Windows/Linux/macOS production packages.

- Expanded the primary README with protocol, shared-hosting, security, privacy, localization, build, release-integrity and repository-structure documentation.
- Standardized build and release-facing documentation on English while retaining all supported runtime languages.
- Made generated PNG/ICO verification deterministic across Windows, Linux and macOS by validating PNG structure, CRCs, dimensions, filters and decoded RGBA pixels instead of depending on zlib byte-for-byte compression output.
- Updated security, privacy and release audits to follow the current localized/runtime implementations without weakening the underlying fail-closed controls.
- Restored lifecycle and SFTP user-error regression coverage and modernized shared-hosting/UI regressions to validate i18n-backed behavior instead of obsolete hard-coded Croatian text.
- Fixed Windows production Go/Python executable discovery on GitHub-hosted runners and aligned the production build baseline with Go 1.26.5.
- Fixed the native Windows setup-language dialog font-height conversion so x64/x86 production builds compile correctly under Go 1.26.5.
- Standardized Linux DEB and macOS package build output/metadata on English-first release text.
- Public release notes are now generated in English from the exact matching changelog section.
- The release workflow builds and verifies Windows x64/x86 Setup, Portable and ZIP packages; Linux amd64/arm64/i386 DEB packages; macOS Universal PKG; SHA-256 checksums; release notes; build provenance metadata; and the `ByFTP.Windows` GitHub Package.

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
