# Ghost FTP — Project Status & Recommended Next Work

This document describes the current **0.17.0 development** source. Historical release details belong in `docs/releases/`.

Latest published release at this point: **0.16.0**.

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
| Android security | SHA-256 SFTP host-key verification, strict checking, FTPS protected data channel, timeouts and cleanup |
| Website | Landing/localized pages plus download, security, support and sitemap routes |
| Platforms | Windows portable + NSIS Setup; Linux binary/AppImage/DEB/RPM; Android APK |
| Release QA | Quality, protocol E2E, native preview, native build, Android and Windows hardening exact-head gates |

Full capability detail: [FEATURES.md](FEATURES.md).

## 0.17.0 hardening completed in source

- Canonical `0.x` release history is the active public version model.
- `version.json` is the version/build source of truth.
- Cargo dependency resolution is committed and checked with `--locked`.
- Windows background helper processes use no-console hardening where required.
- PATH detection avoids spawning `where.exe` merely to render status.
- Android validation runs on every PR to `main`.
- Version progression checks allow meaningful fixes to stay within the active development version and still reject invalid jumps.
- Release publication is exact-SHA gated and sequential.
- Active documentation is synchronized to canonical versions/workflow names.

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

## Repository rules

- Product-facing names: Ghost FTP / GhostFTP.
- Framework-specific internal names only where technically required.
- Implemented features and planned work remain clearly separated.
- No fake servers, fake transfers or fake connection state in production UI.
- Never label a build FINAL solely because it compiles/packages.
