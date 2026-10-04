## 0.30.6 — transfer integrity, mobile UX and security hardening — 4 October 2026

- Remote overwrite and rename existence probes now fail closed across Dynamics, Shopify, HubSpot, OneDrive, Dropbox, FTP, Agent and object-backed paths instead of treating transport, permission or API failures as proof that a target is absent.
- Agent protocol responses distinguish confirmed not-found conditions from operational failures so destructive overwrite decisions propagate ambiguous errors safely.
- Android now uses a responsive left navigation drawer on compact screens, preserves the persistent navigation rail on larger screens and replaces popup confirmation dialogs with inline in-app confirmation UI.
- Android networking disables global cleartext traffic, explicitly warns when plain FTP is selected and removes unused keyboard-interactive SSH authentication.
- Ghost FTP Agent identity/fingerprint handling now rejects malformed or incorrectly sized X25519 keys and identity-file I/O failures; Unix identity files are enforced as owner-only `0600` instead of ignoring permission-hardening failures.
- Desktop official Support, Documentation, Privacy and EULA destinations are centralized and restricted to approved external URLs.
- Advertised desktop locale coverage now has enforced reference-UI parity across all registered languages, including completion dictionaries for previously incomplete locales.
- Production CI regressions found during the hardening cycle were fixed without weakening formatting, protocol E2E, Android, native build or Windows hardening gates.

See [docs/releases/0.30.6.md](docs/releases/0.30.6.md).

## 0.30.5 — Settings ordering and Android URI lifecycle hardening — 4 October 2026

- Serializes Windows/Linux Settings persistence so rapid preference changes cannot complete durable database writes out of user-action order.
- Prevents failed older preference writes from rolling the UI back over a newer action by tracking per-setting revisions and the last durable value.
- Serializes live transfer-engine IPC updates for concurrency, retries, throttle and delta sync so native runtime state cannot finish out of user-action order.
- Gives Reset to Defaults, notification permission and shell PATH integration one shared async mutation lock and busy state.
- Keeps Reset to Defaults locked through recovery and restores the previous DB snapshot, live transfer-engine settings and shell PATH state before returning failure.
- Prevents Preferences from closing while an OS-level Settings mutation is still running.
- Rejects Android picker URIs that cannot actually be opened for reading instead of presenting an unusable file as selected.
- Revalidates saved Android upload URIs after Activity recreation and clears stale grants fail-closed.
- Adds desktop interaction-contract and Android instrumentation/production-contract coverage for the new lifecycle guards.
- Keeps exact-head Quality, real FTP/FTPS/SFTP E2E, Windows/Linux native build, Android and Windows hardening gates mandatory.

See [docs/releases/0.30.5.md](docs/releases/0.30.5.md).

## 0.30.4 — cross-platform interaction serialization — 4 October 2026

- Serializes Agent Bridge master, endpoint, session-access and approval-policy mutations on Windows/Linux so rapid or conflicting clicks cannot race backend IPC state.
- Disables Bridge mutation controls while an operation is in flight and exposes endpoint Start/Stop busy state.
- Prevents Android Pick file, Reset Transfers and Reset Connection from mutating visible transfer context while a transfer/remote operation is active.
- Makes Android Settings Disconnect reflect the real authenticated-session state instead of remaining clickable while disconnected.
- Adds desktop interaction-contract and Android instrumentation/production-contract coverage for the new state guards.
- Keeps the exact-head Quality, real FTP/FTPS/SFTP E2E, Windows/Linux native build, Android and Windows hardening release gates mandatory.

See [docs/releases/0.30.4.md](docs/releases/0.30.4.md).

## 0.30.3 — Android interaction-state and dead-code hardening — 3 October 2026

- Fixes Android action availability so disconnected sessions no longer expose Refresh, Disconnect or remote file operations as actionable controls.
- Keeps the local Android file picker available before connection while guarding remote Upload until a session exists.
- Adds click-by-click instrumentation for disconnected and invalid-session enabled states.
- Serializes Windows/Linux Sync pair toggle, Sync now and Remove actions so duplicate/conflicting IPC mutations cannot race from rapid clicks.
- Adds an Android private Kotlin symbol audit so declaration-only private functions and fields fail the production contract.
- Keeps runtime npm high/critical findings release-blocking while machine-checking the temporary dev-only upstream `braces` advisory exception.
- Preserves the existing cross-platform exact-head quality, real protocol E2E, native build, Android and Windows hardening release gates.

See [docs/releases/0.30.3.md](docs/releases/0.30.3.md).

## 0.30.2 — cross-platform UX and transfer-integrity hardening — 3 October 2026

0.30.2 follows published 0.30.1 and packages the safety and polish completed after that canonical tag.

- Protects desktop overwrite downloads with sibling staging, backup-before-promotion and rollback so failed/canceled transfers cannot truncate an existing local destination.
- Extends staged local replacement to Ghost FTP Agent whole-file fallback while preserving the hash-verified delta path.
- Makes desktop SFTP Rename conflict probing and directory-upload metadata handling fail closed on ambiguous errors.
- Keeps Android FTP/FTPS/SFTP staged replacement fail closed with explicit existence classification, backup, promotion and rollback recovery.
- Adds a bounded Android instrumentation retry only for the confirmed Package Manager split-APK `Broken pipe` transport failure; real test failures stay blocking.
- Makes the Android connection form protocol-aware: FTP/FTPS/SFTP default ports follow 21/21/22 without overwriting custom ports, and SFTP fingerprint input appears only when relevant.
- Adds click-through instrumentation and production-contract coverage for protocol switching, custom-port preservation and SFTP security-field visibility.
- Compacts the shared Windows/Linux Files toolbar at minimum native widths so all contextual actions remain reachable without horizontal panning.
- Gives the active-site desktop control an explicit accessible name with the current site and action.
- Removes the unused Android `MAX_QUEUE_ROWS` alias and confirms no declaration-only private symbols remain in the audited Android Activity/connection controller, while mandatory desktop source-reachability and unused-local checks remain active.
- Synchronizes the 0.30.2 version/build metadata across desktop, Rust workspace, Android, Go compatibility tools and updater templates.
- Splits npm security validation into a zero-tolerance runtime audit plus a machine-checked dev-only exception for the unresolved upstream `braces` advisory, rejecting any unrelated or runtime high/critical finding.

See [docs/releases/0.30.2.md](docs/releases/0.30.2.md).

## 0.30.1 — staged-upload safety and version-line hardening — 2 October 2026

0.30.1 follows published 0.20.10 and starts a broader hardening cycle.

- Makes Android FTP/FTPS/SFTP staged upload replacement fail closed when an existing remote target cannot be safely preserved.
- Requires explicit remote-target existence verification before promotion instead of interpreting backup-rename failure as target absence.
- Refuses promotion when a target appears during an in-progress upload.
- Reports failed backup restoration and preserves the remote backup path for manual recovery instead of silently discarding recovery failure.
- Adds Android regression coverage and production-contract checks for staged replacement safety.
- Centralizes staged remote replacement across FTP/FTPS/SFTP so backup, promotion, rollback and temporary cleanup use one fail-closed transaction.
- Adds JVM regressions for existence-check denial, backup/promotion failures, rollback success/failure, late target races and SFTP `NO_SUCH_FILE` versus permission/protocol failures.
- Makes desktop SFTP Rename candidate probing fail closed so permission, connection and protocol errors can never be mistaken for an available destination.
- Adds Rust regressions proving only a positively classified missing path may be selected by the bounded Rename candidate search.
- Stops directory uploads from silently skipping local entries whose metadata cannot be read; the operation now fails closed with the affected source path.
- Aligns Agent Bridge directory-upload preflight with the same metadata guard so its approval summary cannot authorize a silently incomplete source tree.
- Stages desktop whole-file downloads into transfer-specific sibling files and promotes them only after successful completion, so failed/canceled overwrite downloads cannot truncate an existing local destination.
- Uses backup-before-promotion rollback for local replacement and routes the Ghost FTP Agent whole-file fallback through the same staged protection while preserving its existing hash-verified delta path.
- Allows intentional forward SemVer jumps only when the new version is strictly newer and `previousVersion` matches the exact published base.
- Adds Windows-hardening concurrency so superseded PR/ref runs are canceled instead of consuming runners.

See [docs/releases/0.30.1.md](docs/releases/0.30.1.md).

## 0.20.10 — protocol edge cases and conflict safety — 2 October 2026

0.20.10 follows published 0.20.9 with real protocol edge-case coverage and stricter fail-closed conflict handling.

- Adds zero-byte, Unicode remote-name and permission-denied FTP/FTPS/SFTP E2E coverage.
- Proves changed SFTP host keys are surfaced as mismatches, rejected by default and replaced only after explicit trust.
- Expands pause/resume coverage to a non-aligned multi-MiB payload spanning repeated chunk boundaries.
- Fixes remote conflict rename semantics for dotfiles and Windows-style paths.
- Prevents exhausted Rename candidate searches from falling back to a colliding existing path.

## 0.20.9 — transfer history and release/updater hardening — 1 October 2026

0.20.9 follows published 0.20.8 with a real transfer-history export and hardened release/updater publication.

- Adds end-to-end desktop CSV transfer-history export through Transfer Center, state, IPC and the persisted Rust transfer ledger.
- Neutralizes spreadsheet-formula prefixes in exported path cells and excludes raw backend error text from the shareable CSV.
- Stable GitHub packages may publish without private updater keys; unsigned in-app updater metadata is never generated.
- When updater signing is enabled, signatures, exact-version manifest and Update-Service package are required as an all-or-nothing set and re-verified in the final publish script before tag mutation.
- Extends source reachability to Rust crate module graphs and referenced CI/update/Android helper scripts.
- Expands operational Node/MJS and shell syntax checks.
- Adds Android JVM unit tests for connection input/path safety and cooperative cancellation, and makes `testDebugUnitTest` part of the Android gate.
- Refreshes active documentation from stale RC/current-cycle claims to the 0.20.9 / previous 0.20.8 line.
- Keeps Android production and installable-preview signing semantics explicitly separate.

See [docs/releases/0.20.9.md](docs/releases/0.20.9.md).

# Changelog

## 0.20.8 — cross-app action parity hardening — 1 October 2026

0.20.8 follows published 0.20.7 with Android action parity and stronger click-through enforcement.

- Adds remote Rename on Android for FTP, explicit FTPS and SFTP using the real protocol clients.
- Expands Android Delete to remove either a remote file or an empty remote folder; non-empty folders are never removed recursively.
- Adds long-press folder selection so directory rename/delete actions can target a folder without navigating into it first.
- Replaces the informational-only Android Settings workspace with working Clear Activity, Reset Transfers, Reset Connection and Disconnect controls.
- Persists the non-secret rename target across Activity recreation while continuing to clear passwords and authenticated session state.
- Expands instrumentation smoke coverage and the Android production contract so these actions cannot silently regress into dead UI.

See [docs/releases/0.20.8.md](docs/releases/0.20.8.md).

## 0.20.7 — dead-code and source-reachability hardening — 1 October 2026

0.20.7 follows published 0.20.6 with repository-wide TypeScript dead-code enforcement and source reachability checks.

- Enables TypeScript unused-local diagnostics for the desktop application and shared file-ui package.
- Adds a production Quality gate that walks desktop TypeScript entrypoints and fails on source files that are not reachable from the active application/package graph.
- Keeps cleanup evidence compiler- and entrypoint-driven so files are deleted only when they are demonstrably unused.
- Preserves the 0.20.6 production bundle budget and all exact-SHA release gates.

The audit removed 11 unreachable frontend files: `IconPicker.tsx`, `KeyboardSettings.tsx`, `ProfileEditor.tsx`, `PromptModal.tsx`, `RemoteControlSettings.tsx`, `ui/Badge.tsx`, `ui/Skeleton.tsx`, `ui/Tooltip.tsx`, `brandIconData.ts`, `brandIcons.tsx` and the legacy host `lib/fileIcons.tsx` shim.

TypeScript also identified 21 unused imports/declarations across active files. Those were removed from FilePane, Settings, Site Manager, Terminal and settingsStore, together with the now-unreferenced breadcrumb parser/helper chain.

See [docs/releases/0.20.7.md](docs/releases/0.20.7.md).

## 0.20.6 — frontend bundle-size hardening — 1 October 2026

0.20.6 follows published 0.20.5 with measurable desktop frontend bundle-size hardening.

- Replaced the eager full Material Icon Theme manifest and ~900-icon glob with a curated offline icon set for common development, document, media, archive and creative file types.
- Preserved Ghost FTP's existing Lucide fallback for file types outside the curated branded set.
- Split file-browser, i18n and generated brand-icon data into dedicated stable production chunks instead of keeping them in the main application chunk.
- Added a hard production JavaScript bundle budget: any emitted JS chunk above 500 KiB fails the desktop build instead of only producing a Vite warning.
- Kept all file icons local to the application; no runtime icon network requests were introduced.

See [docs/releases/0.20.6.md](docs/releases/0.20.6.md).

## 0.20.5 — Android API and edge-to-edge hardening — 1 October 2026

0.20.5 follows published 0.20.4 with Android 15 system-bar compatibility and SFTP credential API hardening.

- Replaced the deprecated JSch String password setter with the byte-array API and zeroed the temporary UTF-8 password buffer immediately after JSch copied it.
- Removed direct deprecated Android status/navigation bar color assignments from the Activity.
- Added Android 15 system-bar inset handling so the programmatic root layout remains clear of enforced edge-to-edge system UI on API 35+.
- Added an API 35 theme override that keeps Ghost FTP's dark system-bar icon contract without deprecated status/navigation bar color attributes.
- Added Android production-contract checks that reject reintroduction of deprecated SFTP password and system-bar APIs.
- Preserved the existing FTP/FTPS/SFTP behavior, Android emulator click-through smoke, clean install/reinstall checks and exact-SHA release gates.

See [docs/releases/0.20.5.md](docs/releases/0.20.5.md).

## 0.20.4 — CI runtime and build-config hardening — 1 October 2026

0.20.4 follows published 0.20.3 with CI runtime modernization and build-configuration cleanup.

- Updated first-party GitHub Actions used by the release-relevant workflows to their current supported major trains: checkout/setup-node/setup-go v7, setup-java v6, setup-android v4, Gradle Actions v6 and upload-artifact v7.
- Disabled setup-go dependency caching in the two helper-tool workflows because the helper modules intentionally have no `go.sum`, eliminating the recurring false cache warning.
- Replaced Vite config `__dirname` usage with ESM-safe `fileURLToPath(import.meta.url)` directory resolution so Vite 8 native config loading no longer reports the compatibility warning.
- Added a CI/runtime regression contract that rejects stale action majors, re-enabled unsupported Go cache behavior and reintroduction of Vite `__dirname`.
- Preserved the existing exact-SHA Quality, protocol E2E, native build, Android and Windows hardening release gates.

See [docs/releases/0.20.4.md](docs/releases/0.20.4.md).

## 0.20.3 — file-action reliability hardening — 1 October 2026

0.20.3 follows published 0.20.2 with shared file-browser interaction hardening and release-documentation alignment.

- Serialized shared file-browser confirmation and prompt actions so asynchronous operations cannot be submitted twice while still running.
- Kept rename, new-folder, chmod and delete dialogs open when the underlying filesystem operation fails, allowing a safe retry instead of presenting a premature close as success.
- Added generic in-dialog failure feedback without exposing raw backend errors; detailed errors continue through the existing file-pane error path.
- Removed redundant prompt focus logic and kept focus trapping/restoration under the shared dialog hook.
- Added interaction-contract guards for shared file-ui async action locking and failure propagation.
- Corrected Android release documentation to match the current unsigned production APK plus separate installable-preview validation policy.

See [docs/releases/0.20.3.md](docs/releases/0.20.3.md).

## 0.20.2 — UI and launch-smoke hardening — 1 October 2026

0.20.2 follows published 0.20.1 and carries the post-0.20.1 Android/desktop polish that was merged after the 0.20.1 tag.

- Reworked Android into exclusive Files, Sites, Transfers, Settings and Help & About workspaces behind a persistent left navigation rail instead of one long stacked screen.
- Preserved only non-secret Android workspace state across Activity recreation while keeping passwords and authenticated sessions out of saved state.
- Fixed Android workspace navigation so the selected workspace is scrolled into the visible content area, including the Transfers viewport case caught by emulator instrumentation.
- Made the Android launcher smoke verify the real package-specific RESUMED Activity state and live PID instead of depending on unstable human-readable `am start -W` output formatting.
- Kept Android production contract, lint/APK build, click-through instrumentation, clean install, reinstall and launch proof as release blockers.
- Hardened the shared Windows/Linux Disconnect action so async disconnect errors are surfaced through the normal redacted error path.
- Centralized Windows/Linux workspace resolution for the app shell, primary sidebar and title bar to remove duplicated navigation-state mapping.
- Retained the 0.20.1 transfer worker-registration race fix and exact-SHA Quality, protocol E2E, native build, Android and Windows hardening gates.

See [docs/releases/0.20.2.md](docs/releases/0.20.2.md).

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
