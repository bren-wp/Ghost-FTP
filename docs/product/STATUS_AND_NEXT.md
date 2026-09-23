# Ghost FTP — Project Status & Recommended Next Work

This document separates what Ghost FTP provides in the current **2.1.1 RC19** source from future product work. Historical release details belong in `docs/releases/`.

## Implemented in RC19

| Area | Current capability |
|---|---|
| Connections | FTP, explicit FTPS, SFTP, temporary connections, saved Sites, private-key paths, host-key/TLS verification |
| Application shell | One persistent native window for Files, Sites, Transfers, Sync & Backup, Settings and Help & About |
| File browser | Local/remote browsing, upload/download, folder operations, rename, delete, duplicate, hidden files, multiple views |
| File properties | SHA-256 plus supported chmod/permission and owner/group workflows |
| Transfers | Concurrent queue, pause/resume, retry, retry-all, cancel, priority movement, bandwidth control, conflict handling, scheduling, live progress-derived speed/ETA |
| Sync & analysis | Folder sync, directory comparison, duplicate detection and disk analysis |
| Productivity | Docked terminal, command palette, snippets, shortcuts and shell integration |
| Preferences | Themes, language, transfer limits, connection/security settings, notifications and advanced controls |
| Security/privacy | OS credential storage where supported, CSP, signed updater path, credential redaction, session-only terminal suggestion and notification history |
| Platforms | Windows portable + NSIS Setup; Linux binary, AppImage, DEB and RPM |
| Languages | English primary plus 13 additional selectable interface languages |
| Release QA | Quality + Rust gates, real FTP/explicit-FTPS/SFTP E2E, Windows/Linux native builds and Windows native 7-surface × 3-viewport evidence |

Full capability detail: [Implemented Features](FEATURES.md).

## RC19 hardening completed in source

- Preserves process-memory-only terminal suggestion history from RC18.
- Filters credential-looking shell commands before they can enter in-memory suggestion history.
- Makes notification-center history session-only.
- Redacts query credentials, assignments, sensitive CLI flags, Authorization/Bearer values, URL passwords, JSON credential fields and private-key blocks before user-facing diagnostic storage/display.
- Purges legacy terminal/notification history keys created by older candidates during startup migration.
- Fixes terminal listener/PTy startup disposal races and transfer listener/snapshot startup races.
- Keeps workspace-level error containment and the one-window navigation model.
- Keeps terminal PTY lifetime docked and deterministic with no secondary-window handoff residue.
- Keeps real progress-delta transfer telemetry and near-minimum native Windows QA coverage.
- Updates active repository documentation and release metadata consistently to RC19.

## Gates before stable / FINAL

A release candidate is not a stable/FINAL claim. Stable promotion still requires product-owner acceptance of:

1. Windows clean install, upgrade, reinstall and uninstall lifecycle.
2. Windows 10/11 installed + portable visual acceptance, including titlebar and snap behavior.
3. Required security failure-path review on target systems.
4. Production code-signing decision and verification.
5. Final accessibility/keyboard/high-contrast review.
6. Any broader protocol matrix beyond the automated FTP, explicit FTPS and SFTP release E2E suite.

## Recommended next improvements

- Per-profile reconnect/keep-alive policies and connection health/reconnect status.
- Per-profile bandwidth limits and transfer-history export.
- Verify-after-transfer checksums where both endpoints can calculate them.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile export/import and duplicate-profile detection.
- Windows snap-layout, screen-reader, high-contrast and touch-target acceptance.
- SBOM/provenance, secret scanning and reproducible-build documentation.
- Production signing and optional managed package repositories when distribution policy requires them.

## Repository rules

- Keep product-facing names Ghost FTP / GhostFTP.
- Keep framework-specific internal names only where technically required.
- Keep implemented features separate from planned work.
- Never seed fake servers, fake transfers, fake connection state or screenshot-backed runtime UI.
- Never label a build FINAL solely because it compiles or packages successfully.
