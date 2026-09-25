# Ghost FTP — Project Status & Recommended Next Work

This document separates what Ghost FTP provides in the current **2.1.1 RC23** source from future product work. Historical release details belong in `docs/releases/`.

## Implemented in RC23

| Area | Current capability |
|---|---|
| Desktop connections | FTP, explicit FTPS, SFTP, temporary connections, saved Sites, private-key paths, host-key/TLS verification |
| Desktop application shell | One persistent native window for Files, Sites, Transfers, Sync & Backup, Settings and Help & About |
| Desktop file browser | Local/remote browsing, upload/download, folder operations, rename, delete, duplicate, hidden files, multiple views |
| Desktop file properties | SHA-256 plus supported chmod/permission and owner/group workflows |
| Desktop transfers | Concurrent queue, pause/resume, retry, retry-all, cancel, priority movement, bandwidth control, conflict handling, scheduling, live progress-derived speed/ETA |
| Desktop sync & analysis | Folder sync, directory comparison, duplicate detection and disk analysis |
| Desktop productivity | Docked terminal, command palette, snippets, shortcuts and shell integration |
| Preferences | Themes, language, transfer limits, connection/security settings, notifications and advanced controls |
| Security/privacy | OS credential storage where supported, CSP, signed updater path, credential redaction, session-only terminal suggestion and notification history |
| Android app | Native FTP, explicit FTPS and SFTP listing, download, upload, delete and new-folder actions with guarded transfer UX |
| Android security | SFTP SHA-256 host-key fingerprint verification, strict host-key checking, FTPS protected data channel, timeouts and cleanup |
| Website | Public landing page, localized landing pages, `/download/`, `/security/`, `/support/` and sitemap entries |
| Platforms | Windows portable + NSIS Setup; Linux binary, AppImage, DEB and RPM; Android APK |
| Languages | English primary plus additional selectable interface languages |
| Release QA | Quality + Rust gates, real FTP/explicit-FTPS/SFTP E2E, Windows/Linux RC23 native builds, Android APK build and RC23 release packaging |

Full capability detail: [Implemented Features](FEATURES.md).

## RC23 hardening completed in source

- Updates all release metadata to `2.1.1-rc.23` across desktop, Rust packages, Android, updater templates and documentation.
- Adds RC23-specific native Windows/Linux build workflow and release workflow.
- Keeps the historical native build workflow separate from the RC23 release path.
- Adds production website routes for download, security and support.
- Adds sitemap entries for the new production routes.
- Keeps Android real file actions: list, download, upload, delete and create remote folder.
- Locks Android FTP/FTPS timeouts, passive binary transfers, login validation and cleanup.
- Locks explicit FTPS `PBSZ 0` and protected data channel `PROT P`.
- Locks Android SFTP strict host-key checking, SHA-256 fingerprint verification, connection timeouts and cleanup.
- Keeps Android guarded upload/delete confirmations, bounded activity log and lifecycle-safe UI updates.
- Keeps connection-attempt deduplication and correct multi-connect busy-state tracking.
- Keeps recurring transfer schedule start-date and stale-target safety.
- Keeps credential redaction for notification, updater and diagnostic text.
- Keeps process-memory-only terminal suggestion history and notification history.
- Keeps startup migration cleanup for legacy persisted terminal/notification history.
- Keeps terminal listener/PTy startup disposal race protection and transfer listener/snapshot startup race protection.
- Keeps workspace-level error containment and the one-window navigation model.
- Keeps real progress-delta transfer telemetry and native release gates.

## Gates before stable / FINAL

A release candidate is not a stable/FINAL claim. Stable promotion still requires product-owner acceptance of:

1. Windows clean install, upgrade, reinstall and uninstall lifecycle.
2. Windows 10/11 installed + portable visual acceptance, including titlebar and snap behavior.
3. Linux package install/update/remove lifecycle across target distributions.
4. Android install, upgrade, storage-access and protocol acceptance on target devices.
5. Required security failure-path review on target systems.
6. Production code-signing decision and verification.
7. Final accessibility/keyboard/high-contrast review.
8. Any broader protocol matrix beyond the automated FTP, explicit FTPS and SFTP release E2E suite.

## Recommended next improvements

- Per-profile reconnect/keep-alive policies and connection health/reconnect status.
- Per-profile bandwidth limits and transfer-history export.
- Verify-after-transfer checksums where both endpoints can calculate them.
- Batch rename and remote-edit conflict detection.
- Encrypted selected-profile export/import and duplicate-profile detection.
- Android stable signing-key continuity plan before broad mobile distribution.
- Windows snap-layout, screen-reader, high-contrast and touch-target acceptance.
- Linux desktop integration acceptance on target distributions.
- SBOM/provenance, secret scanning and reproducible-build documentation.
- Production signing and optional managed package repositories when distribution policy requires them.

## Repository rules

- Keep product-facing names Ghost FTP / GhostFTP.
- Keep framework-specific internal names only where technically required.
- Keep implemented features separate from planned work.
- Never seed fake servers, fake transfers, fake connection state or screenshot-backed runtime UI.
- Never label a build FINAL solely because it compiles or packages successfully.
