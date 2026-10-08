# Ghost FTP — Project Status & Recommended Next Work

This document describes the current **0.30.13 development** source. Historical release details belong in `docs/releases/`.

Previous canonical release: **0.30.10**. Live publication state is determined from GitHub Releases.

## Implemented

| Area | Current capability |
|---|---|
| Desktop connections | FTP, explicit FTPS, SFTP, temporary connections, saved Sites, private-key paths, host-key/TLS verification |
| Desktop shell | One persistent native window for Files, Sites, Transfers, Sync & Backup, Settings and Help & About |
| Desktop file browser | Local/remote browsing, upload/download, folder operations, rename, delete, duplicate, hidden files and multiple views |
| Desktop file properties | SHA-256 plus supported chmod/permission and owner/group workflows |
| Desktop transfers | Concurrent queue, pause/resume, retry, retry-all, cancel, bandwidth control, conflicts, scheduling, live speed/ETA and credential-free restart recovery history |
| Sync & analysis | Folder sync, directory comparison, duplicate detection and disk analysis |
| Productivity | Docked terminal, command palette, snippets, shortcuts and shell integration |
| Preferences | Themes, language, transfer limits, security settings, notifications and advanced controls |
| Security/privacy | OS credential storage where supported, CSP, signed-updater path, credential redaction and no required telemetry |
| Android | Native FTP, explicit FTPS and SFTP connection/listing/download/upload/delete/new-folder actions plus Files/Sites/Transfers/Sync & Backup/Settings/Help & About workspace parity |
| Platforms | Windows portable + NSIS Setup; Linux binary/AppImage/DEB/RPM; Android APK; macOS ad-hoc-signed development Preview |
| Release QA | Quality, protocol E2E, canonical native build, Android, Windows hardening and macOS exact-head gates |
| Documentation provenance | Local README/docs images are verified against the latest published release tag |

Full capability detail: [FEATURES.md](FEATURES.md).

## 0.20.9 transfer-history and release-integrity hardening in source

- Adds a real desktop CSV transfer-history export through Transfer Center → state → IPC → Rust persisted transfer ledger.
- Neutralizes spreadsheet-formula prefixes in exported user/server-controlled path cells and excludes raw backend error text from the shareable CSV.
- Keeps stable GitHub package publication independent of private updater keys; without them the signed in-app updater bundle is omitted rather than faked.
- When updater signing is enabled, both signatures, verified `latest.json` and matching Update-Service package are required as an all-or-nothing set, and the embedded manifest must match byte-for-byte before tag mutation.
- Extends source reachability from TypeScript to Rust crate module graphs and requires operational CI/update/Android helper scripts to have a real workflow/package/runbook reference.
- Adds Android `ConnectionModel` JVM tests for cancellation, host/port validation, Unicode/IDN normalization, remote path traversal guards and root-delete rejection; the Android workflow runs them before APK packaging.
- Keeps Android production output explicitly unsigned and the debug-key-signed, non-debuggable package explicitly labeled as an installable preview.

## 0.20.8 cross-app action parity hardening in source

- Android Rename is protocol-backed for FTP, explicit FTPS and SFTP.
- Android Delete handles remote files and empty remote folders while refusing recursive folder deletion.
- Long-pressing a folder selects it for transfer actions without changing the current folder first.
- Android Settings now contains working Clear Activity, Reset Transfers, Reset Connection and Disconnect actions.
- Emulator smoke and the production contract enforce the new actions so dead UI cannot silently ship.
- Password/session secrecy rules remain unchanged across Activity recreation and disconnect.

## 0.20.7 dead-code and reachability hardening in source

- Enables TypeScript unused-local diagnostics across the desktop app and shared file-ui package.
- Adds an entrypoint/import-graph Quality gate for desktop TypeScript source files.
- Fails CI when a source file is not reachable from the production app or shared package entrypoint.
- Uses compiler/reachability evidence before deleting code, avoiding destructive “cleanup” based only on similar filenames.
- Keeps the 500 KiB JavaScript chunk budget and existing exact-SHA release requirements.
- First enforced audit result: 11 unreachable frontend files deleted, 21 compiler-reported unused imports/declarations removed, and 102 TypeScript files remain reachable from production entrypoints.

## 0.20.6 frontend bundle-size hardening in source

- Replaces the eager full Material Icon Theme manifest/glob with an explicit offline set for common code, document, media, archive and creative formats.
- Retains the existing Lucide fallback for unmatched files instead of shipping the entire upstream icon catalog.
- Emits file-browser, i18n and generated brand-icon data as dedicated production chunks.
- Enforces a 500 KiB maximum for every emitted production JavaScript chunk after Vite build.
- Keeps icon rendering local/offline and does not add runtime CDN or telemetry dependencies.

## 0.20.5 Android API and edge-to-edge hardening in source

- Replaces deprecated JSch String password handoff with the byte-array API and clears the temporary UTF-8 password buffer after the Session copies it.
- Removes direct deprecated status/navigation bar color assignments from the Android Activity.
- Applies Android 15 system-bar insets to the root layout so enforced edge-to-edge UI cannot obscure Ghost FTP controls.
- Adds API 35-specific theme resources without deprecated system-bar color attributes.
- Extends the Android production contract so deprecated password/system-bar APIs cannot be reintroduced silently.

## 0.20.4 CI runtime and build-config hardening in source

- Release-relevant workflows now use current supported major trains for first-party GitHub Actions instead of deprecated Node 20 action runtimes.
- Go setup caching is disabled for the two small helper-tool workflows because those modules intentionally have no `go.sum`.
- Vite configuration resolves its directory through ESM-safe `fileURLToPath(import.meta.url)` instead of `__dirname`.
- A dedicated CI/runtime contract prevents stale action majors, unsupported Go cache assumptions and Vite `__dirname` from being reintroduced.

## 0.20.3 file-action reliability hardening in source

- Shared file-browser prompt and confirmation actions now await asynchronous rename, new-folder, chmod and delete operations before closing.
- In-flight actions disable repeat submission and prevent backdrop/Escape cancellation until the operation settles.
- Failed filesystem mutations keep the dialog available for retry while the existing file-pane error surface retains the detailed backend-safe message.
- Shared interaction contract checks prevent regressions back to fire-and-forget modal operations.
- Android release documentation now matches the unsigned production artifact and separate installable-preview validation policy.

## 0.20.2 UI and launch hardening in source

- Keeps Android Files, Sites, Transfers, Settings and Help & About as exclusive workspaces behind the persistent left navigation rail.
- Fixes workspace navigation so the selected Android workspace is brought into the visible content area after a rail action.
- Makes Android post-instrumentation launch verification state-based: command exit status, package-specific RESUMED Activity and live PID instead of brittle `am start -W` text parsing.
- Retains the shared Windows/Linux Disconnect rejection handler and centralized workspace resolver introduced after the 0.20.1 tag.

## 0.20.1 patch hardening in source

- Closes a desktop transfer-worker registration/finalization race that could retain an already-completed `JoinHandle` when a very fast transfer finished before task-map registration.
- Adds a regression test for the completed-before-registration interleaving.
- The historical 0.20.1 Android signing experiment is superseded by the current explicit policy: production package verification remains unsigned unless a persistent production key is deliberately introduced, while the separately signed CI package is labeled preview.

## 0.20.0 hardening retained

- Persists desktop transfer rows and retry descriptors without credentials so restart/update recovery retains user-visible history.
- Reconnects recovered retries through saved profiles and the OS credential store, while restarting from zero when cross-process file identity is not provable.
- Bounds completed/error/canceled/skipped transfer history during runtime without dropping active transfers.
- Verifies resumed FTP/explicit-FTPS uploads and safely restarts from zero if the server accepts the resume command but persists the wrong final size.
- Extends real FTP/explicit-FTPS E2E coverage for cancellation, ABOR cleanup and post-error control-channel synchronization, while real OpenSSH SFTP E2E verifies non-zero-offset resume primitives.
- Hardens Android staged transfers, lifecycle cancellation, stream cleanup, SAF persistence and release-signing continuity.

## 0.19.0 hardening in source

- Uses one authoritative Windows/Linux production build instead of compiling the same native bundles twice.
- Brands the canonical Windows NSIS Setup with Ghost FTP artwork and interactive EULA acceptance.
- Smoke-tests the exact Windows Setup install/uninstall lifecycle produced by the release build.
- Verifies Linux AppImage metadata plus DEB install/remove and RPM metadata in CI.
- Refreshes Android FTP/FTPS and SFTP libraries to maintained releases.
- Aligns Android navigation to Files, Sites, Transfers, Settings and Help & About and validates core actions in an emulator click-through smoke.
- Removes the obsolete website application, website CI/release archive and website roadmap surface.
- Uses size-focused native release codegen and keeps optional AppImage media bundling disabled.
- Uploads only final native package files from CI and enforces artifact-size budgets.
- Keeps Windows native-window QA inside the canonical build and packages QA evidence with releases.
- Verifies documentation image blobs against the newest published release.
- Makes central version synchronization refresh Cargo.lock atomically with Cargo metadata.
- Fixes updater note synchronization and rebases bot metadata commits before push to avoid branch races.
- Keeps existing tag immutability and exact-SHA release gating.

## Gates before stable / FINAL

Publishing a development release is not the same as a stable/FINAL claim. Stable promotion still requires product-owner acceptance of:

1. Windows clean install, upgrade, reinstall and uninstall lifecycle.
2. Windows 10/11 installed + portable visual acceptance including titlebar/snap behavior.
3. Linux package install/update/remove lifecycle across target distributions.
4. Android install/upgrade/storage/protocol acceptance on target devices.
5. Required security failure-path review on target systems.
6. Production code-signing decision and verification.
7. Final accessibility/keyboard/high-contrast review.
8. Broader protocol failure matrix beyond the automated release E2E suite.

## Recommended next improvements

- Per-profile reconnect/keep-alive policies and connection-health state.
- Per-profile bandwidth limits.
- Verify-after-transfer checksums where both endpoints support them.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile import/export and duplicate detection.
- Android stable signing-key continuity plan.
- Windows screen-reader/high-contrast/touch-target acceptance.
- Linux desktop integration acceptance.
- SBOM/provenance, secret scanning and reproducible-build verification.
- Optional managed package repositories when distribution policy requires them.
