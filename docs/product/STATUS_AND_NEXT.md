# Ghost FTP — Project Status & Recommended Next Work

This document describes the current **0.19.0 development** source. Historical release details belong in `docs/releases/`.

Latest published canonical release: **0.18.0**. Live publication state is determined from GitHub Releases.

## Implemented

| Area | Current capability |
|---|---|
| Desktop connections | FTP, explicit FTPS, SFTP, temporary connections, saved Sites, private-key paths, host-key/TLS verification |
| Desktop shell | One persistent native window for Files, Sites, Transfers, Sync & Backup, Settings and Help & About |
| Desktop file browser | Local/remote browsing, upload/download, folder operations, rename, delete, duplicate, hidden files and multiple views |
| Desktop file properties | SHA-256 plus supported chmod/permission and owner/group workflows |
| Desktop transfers | Concurrent queue, pause/resume, retry, retry-all, cancel, bandwidth control, conflicts, scheduling and live speed/ETA |
| Sync & analysis | Folder sync, directory comparison, duplicate detection and disk analysis |
| Productivity | Docked terminal, command palette, snippets, shortcuts and shell integration |
| Preferences | Themes, language, transfer limits, security settings, notifications and advanced controls |
| Security/privacy | OS credential storage where supported, CSP, signed-updater path, credential redaction and no required telemetry |
| Android | Native FTP, explicit FTPS and SFTP listing/download/upload/delete/new-folder actions |
| Website | Landing/localized pages plus download, security, support and sitemap routes |
| Platforms | Windows portable + NSIS Setup; Linux binary/AppImage/DEB/RPM; Android APK |
| Release QA | Quality, protocol E2E, canonical native build, Android and Windows hardening exact-head gates |
| Documentation provenance | Local README/docs images are verified against the latest published release tag |

Full capability detail: [FEATURES.md](FEATURES.md).

## 0.19.0 hardening in source

- Uses one authoritative Windows/Linux production build instead of compiling the same native bundles twice.
- Keeps Windows native-window QA inside that canonical build and packages QA evidence with releases.
- Verifies documentation image blobs against the newest published release.
- Makes central version synchronization refresh Cargo.lock atomically with Cargo metadata.
- Removes ignored release-note staging from version-sync.
- Removes stale legacy preview source-package naming from the active build path.
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
