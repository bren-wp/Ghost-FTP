# Ghost FTP — Project Status & Recommended Next Work

This document separates **what Ghost FTP provides now** from **recommended future product work**. It intentionally avoids unnecessary implementation detail.

## Implemented and available in RC11

| Area | Implemented |
|---|---|
| Connections | FTP, FTPS, SFTP, Quick Connect, saved profiles, private-key paths, host-key/TLS verification |
| Site Manager | Favorites, folders, tags, bookmarks, recent-server metadata |
| File browser | Dual panes, upload/download, folder operations, rename, delete, duplicate, hidden files, multiple views |
| File properties | SHA-256, supported chmod/permission workflows, owner/group display where available |
| Transfers | Concurrent queue, pause/resume, retry, Retry All, cancel, bandwidth control, conflict handling |
| Sync & analysis | Folder sync, directory compare, duplicate detection, disk analysis |
| Productivity | Integrated terminal, command palette, snippets, shortcuts, shell integration |
| Preferences | Themes, density, previews, transfer limits, download folder, editor, notifications, privacy controls |
| Security | OS credential storage where supported, CSP, signed update path, secret separation from profile JSON |
| Platforms | Windows portable/Setup; Linux binary/AppImage/DEB/RPM build targets |
| Languages | 14-language selector coverage |

Full details: [Implemented Features](FEATURES.md).

## RC11 engineering work completed

- Fixed standalone-view stacking so Ghost FTP application chrome can no longer cover About, Preferences, Site Manager, Transfer Center or New Connection surfaces.
- Fixed top-menu popovers so they float above the content instead of participating in normal layout flow.
- Added real FTP/explicit-FTPS/SFTP E2E acceptance and binary-safe FTP/FTPS transfer mode.
- Added isolated real backend connection probes for Quick Connect and Site Manager.
- Transfer Center now uses real queue data and live transferred-byte sampling instead of simulated activity.

- Repository reorganized into a clear product layout: `ghostftp-desktop/`, `website/`, `updates/`, `tools/` and `docs/`.
- Public release naming standardized by product, platform, architecture, role and version.
- Main README rebuilt as a marketing/product landing page with local Ghost FTP logo, icons and visual references.
- Visual reference assets renamed with GhostFTP-first filenames.
- Obsolete browser-host production path excluded from end-user releases.
- Transfer Queue terminal-state actions corrected.
- Retry All added for failed transfers.
- Semantic transfer progress and reduced-motion handling added.
- Preferences exposes previously hidden browser/transfer settings.
- Website expanded as the English-first product site with local Ghost FTP branding, screenshots and download/security/support content.
- Corrupt profile/database recovery paths hardened.
- PATH integration migration logic corrected so historical `Ghost FTP` and `GhostFTP` app directory aliases do not duplicate the managed entry.
- Preview/stable update channels, manifest schema and validation tooling are grouped under `updates/`.
- Rust, Go, TypeScript, website and update-tool checks remain CI-gated.

## Release gates still required before FINAL

These are not marked complete without real target-OS evidence:

1. Windows 10/11 native screenshot comparison for Main, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About.
2. Confirmation that only the Ghost FTP custom titlebar is visible in installed and portable Windows builds.
3. Real FTP upload/download/rename/delete/resume acceptance.
4. Real FTPS certificate and failure-path acceptance.
5. Real SFTP password/private-key and host-key acceptance.
6. Windows clean install, upgrade, reinstall and uninstall lifecycle QA.
7. Final responsive acceptance at all documented viewport sizes.
8. Production code-signing decision and signing validation.

## Recommended high-value improvements

### Transfers

- Persistent scheduled transfers.
- Per-transfer priority.
- Per-profile bandwidth limits.
- Rolling transfer-speed calculation.
- Optional verify-after-transfer checksum.
- Transfer-history export.

### Connections

- Per-profile reconnect/keep-alive policy.
- Connection-health indicator.
- Visible reconnect countdown.
- SOCKS/HTTP proxy workflows if demanded by users.

### File workflow

- Batch rename.
- Configurable double-click behavior.
- Safe-delete/trash integration for local files.
- Remote edit conflict detection.
- Better archive preview/extract workflows.

### Site Manager

- Encrypted selected-profile export/import.
- Duplicate profile detection.
- Profile templates.
- Richer search by host, username, tag and note.
- Optional profile color/icon.

### UI / UX

- Persisted column widths.
- Optional compact toolbar labels.
- User-configurable panel visibility.
- Full keyboard navigation audit.
- Screen-reader audit.
- High-contrast acceptance.
- Windows 11 snap-layout acceptance.

### Security and release engineering

- Production Windows code signing.
- SBOM generation.
- Dependency vulnerability scanning.
- Secret scanning.
- Reproducible-build documentation.
- Signed Linux repository metadata if package repositories are introduced.

## Recommended repository rules

- Keep product-facing paths and artifacts GhostFTP-branded.
- Keep framework-required internal names only where technically necessary.
- Keep implemented features separate from planned work.
- Never publish a compatibility/browser-host executable as the production desktop GUI.
- Never call a build FINAL until its required release evidence exists.
