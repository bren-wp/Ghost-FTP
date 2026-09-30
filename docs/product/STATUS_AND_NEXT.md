# Ghost FTP — Project Status & Recommended Next Work

This document describes the current **0.20.0 development** source. Historical release details belong in `docs/releases/`.

Previous canonical release: **0.19.0**. Live publication state is determined from GitHub Releases.

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
| Android | Native FTP, explicit FTPS and SFTP connection/listing/download/upload/delete/new-folder actions aligned to desktop Files/Sites/Transfers terminology |
| Platforms | Windows portable + NSIS Setup; Linux binary/AppImage/DEB/RPM; Android APK |
| Release QA | Quality, protocol E2E, canonical native build, Android and Windows hardening exact-head gates |
| Documentation provenance | Local README/docs images are verified against the latest published release tag |

Full capability detail: [FEATURES.md](FEATURES.md).

## 0.20.0 hardening in source

- Persists desktop transfer rows and retry descriptors without credentials so restart/update recovery retains user-visible history.
- Reconnects recovered retries through saved profiles and the OS credential store, while restarting from zero when cross-process file identity is not provable.
- Bounds completed/error/canceled/skipped transfer history during runtime without dropping active transfers.
- Verifies resumed FTP/explicit-FTPS uploads and safely restarts from zero if the server accepts the resume command but persists the wrong final size.
- Extends real protocol E2E coverage for cancellation, ABOR cleanup and post-error control-channel synchronization.
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
- Per-profile bandwidth limits and transfer-history export.
- Verify-after-transfer checksums where both endpoints support them.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile import/export and duplicate detection.
- Android stable signing-key continuity plan.
- Windows screen-reader/high-contrast/touch-target acceptance.
- Linux desktop integration acceptance.
- SBOM/provenance, secret scanning and reproducible-build verification.
- Optional managed package repositories when distribution policy requires them.
