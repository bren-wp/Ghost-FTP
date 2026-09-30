# Changelog

## 0.20.1 — patch cycle — 30 September 2026

0.20.1 follows canonical 0.20.0. It is not considered published until the exact-source GitHub Release and required assets exist.

- Closed a desktop transfer worker-registration/finalization race where an immediately completed worker could finish before its `JoinHandle` was recorded, leaving stale completed task state in memory.
- Added Tokio regression coverage for the completed-before-registration interleaving and routed download, upload and manual-retry workers through the race-safe registration helper.
- Advanced desktop, Rust workspace, Android, compatibility-tool and updater metadata to 0.20.1 with Android `versionCode` 211030.
- Simplified Android release packaging so it no longer depends on private `ANDROID_RELEASE_*` secrets or a second optional signing branch.
- The Android gate now always verifies an intentionally unsigned production `com.ghostftp.android` APK plus a non-debuggable installable preview used for clean-install/reinstall/launch smoke; canonical release naming keeps the unsigned production artifact explicitly marked `.unsigned`.
- Reworked the Android shell into separate Files, Sites, Transfers, Settings and Help & About workspaces behind a persistent left navigation rail, with workspace state preserved across Activity recreation without persisting credentials.
- Hardened the shared Windows/Linux title-bar Disconnect action so asynchronous disconnect failures are caught locally, redacted through the normal error path and surfaced to the user instead of falling through to the global async handler.
- Centralized the Windows/Linux workspace resolver used by the app shell, primary sidebar and title bar, removing three independent copies of workspace-state mapping and adding a regression guard against reintroduction.
- Refreshed build, QA, product, versioning and release documentation for the patch cycle while keeping documentation imagery pinned to the latest published release until new release-proven captures can replace it.

See [docs/releases/0.20.1.md](docs/releases/0.20.1.md).

## 0.20.0 — release cycle — 30 September 2026

0.20.0 follows canonical 0.19.0; live publication status is determined by GitHub Releases.

- Added a durable, credential-free desktop transfer ledger in SQLite so interrupted queue/history state survives process restart and application updates.
- Recovered transfers now reconnect through the saved profile and OS credential store; cross-process retries restart safely from byte zero when source identity cannot be proven, preventing hybrid/corrupted files.
- Bounded terminal transfer history during runtime while preserving active transfers, avoiding unbounded in-memory and SQLite growth in long-running clients.
- Made FTP and explicit FTPS pause/resume byte-accurate, including cooperative ABOR cleanup, fresh authenticated control-session recovery after interrupted transfers, and verified resumed uploads with safe full-restart fallback when the remote final size is wrong.
- Expanded real FTP/explicit-FTPS protocol E2E coverage for pause/resume, cancellation, ABOR race recovery, control-channel isolation and I/O failure cleanup, plus real OpenSSH SFTP non-zero-offset resume primitive verification.
- Hardened Android transfers with staged upload/download replacement, lifecycle-aware cancellation for upload/download/delete/new-folder actions, and deterministic stream cleanup.
- Corrected Android SAF persisted-permission handling so the picker keeps persistable access while `takePersistableUriPermission` receives only valid READ/WRITE grant modes.
- Tightened Android stable-signing continuity so an existing stable APK must be downloadable, inspectable and signed by the same certificate; failures now block release rather than silently skipping the check.
- Kept Windows console-free helper hardening, Windows NSIS/MSI lifecycle smoke, Linux AppImage/DEB/RPM lifecycle smoke and exact-head release gates mandatory.
- Hardened release CI against transient Ubuntu APT stalls with bounded retries/timeouts and extended exact-SHA gate waiting so valid Windows/Linux/Android release candidates are not dropped by a premature orchestration timeout.
- Updated release/version metadata, documentation and updater templates for 0.20.0.

See [docs/releases/0.20.0.md](docs/releases/0.20.0.md).

## 0.19.0 — release cycle — 27 September 2026

0.19.0 follows canonical 0.18.0; live publication status is determined by GitHub Releases.

- Branded the canonical Windows NSIS Setup with Ghost FTP installer/uninstaller icons, header/sidebar artwork and the root EULA as the interactive licence page.
- Added real silent Setup install/uninstall smoke coverage to the canonical Windows native build.
- Added Linux AppImage/DEB/RPM lifecycle and package metadata checks.
- Updated Android FTP/FTPS and SFTP libraries to maintained releases.
- Switched native release codegen to size-focused `opt-level = "s"` while retaining LTO, one codegen unit, panic abort and stripped symbols.
- Explicitly disabled optional AppImage media-framework bundling because Ghost FTP does not require audio/video playback.
- Reduced CI native artifact staging to final package files instead of expanded bundle directories.
- Added artifact size budgets and build-summary size reporting.
- Fixed updater note synchronization and hardened version-sync against concurrent branch pushes.
- Kept README/documentation imagery pinned to the newest published release tag.
- Removed the obsolete website application source, website CI/release packaging and active website roadmap/docs.
- Added release-proven real native application screenshot gallery to the root README.
- Added Android emulator click-through instrumentation and desktop-aligned Files/Sites/Transfers/Settings/Help & About navigation.
- Made Android run on every `main` push so exact-SHA release orchestration always has a mobile gate.
- Removed a dead preview verifier that depended on a non-existent mock runtime.
- Updated EULA, commercial licence, privacy, security, support and third-party notices for the app-only distribution model.

See [docs/releases/0.19.0.md](docs/releases/0.19.0.md).

## 0.18.0 — published — 27 September 2026

0.18.0 was published after 0.17.0 and established the canonical release-image provenance/native-build flow.

- Added CI provenance validation that keeps README/documentation images byte-identical to the newest published Ghost FTP release tag.
- Consolidated duplicate Windows/Linux Tauri build paths into one canonical production native build.
- Kept Windows native-window screenshot QA in the canonical build and added its evidence archive to release packaging.
- Removed the obsolete duplicate native build workflow and stale RC-era workflow references.
- Removed hard-coded `v2.1.1-Preview` source package naming from the active build path.
- Fixed version synchronization so ignored release-note paths no longer break the bot commit.
- Made version synchronization refresh and commit the canonical Cargo.lock atomically with Cargo metadata changes.
- Kept exact-SHA quality, real protocol E2E, Android and Windows hardening around the canonical native build.
- Continued the no-version-only policy.

See [docs/releases/0.18.0.md](docs/releases/0.18.0.md).

## 0.17.0 — published — 27 September 2026

- Completed the canonical public-history migration.
- Hardened Windows background helper processes against unintended console flashes.
- Added Windows helper-process regression checks.
- Made Android validation run on every pull request to `main`.
- Allowed meaningful fixes to remain within the active development version without forcing a version-only bump.
- Published normalized Windows, Linux, Android, source, documentation and SHA-256 assets.

See [docs/releases/0.17.0.md](docs/releases/0.17.0.md).

## Older canonical history

Canonical release notes for `0.1.0` through `0.16.0` are retained under `docs/releases/`.

The older `2.1.1-rc.*` identifiers are historical aliases only. Their verified mapping/provenance is recorded in `docs/releases/version-map.json`; they are not the active version scheme.
