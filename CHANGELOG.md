# Changelog

The active product changelog intentionally starts at 1.3.0. Older development history remains available in Git history but is not part of the current maintained release documentation.

## 1.7.0 — Full cleanup, native hardening and ByFTP WEB integration

**Focus:** clean the maintained repository surface, fix platform-specific edge cases and make the hardened shared-hosting web client a first-class ByFTP source/release target.

- Added the complete `ByFTP WEB/` PHP/PWA source tree with the same canonical 1.7.0 release number as the native clients.
- Reworked the web application around raw-input validation, canonical remote paths, isolated multi-user workspaces, encrypted saved credentials, atomic JSON persistence, CSRF/session protections, no-index headers and static-only PWA caching.
- Added FTP/FTPS and optional SFTP web transports, including SFTP SHA-256 host-key verification, post-authentication secret cleanup and private/reserved-host blocking by default.
- Added web file-manager operations: browse, upload and folder upload, download, editor with optimistic ETag conflict protection, create/rename/delete, copy/move/duplicate, CHMOD, search, favorites, checksum, batch rename and bounded ZIP create/extract/download.
- Replaced the duplicated/missing PNG PWA icon path with the canonical SVG ByFTP mark and consolidated the web layout into maintainable responsive `app.css` plus a separate Brendigo-family visual layer.
- Added `scripts/audit_web.py` and web unit/syntax gates so the web client cannot enter a release as an unaudited source directory.
- Fixed Windows remote create/rename UI handling so remote names are validated verbatim and are never silently trimmed before backend validation.
- Fixed Linux/macOS terminal prompting so host, username and private-key paths preserve raw identity/path input while terminal CR/LF endings are removed separately.
- Fixed Android Storage Access Framework uploads when a document provider returns a null/blank display name by using a deterministic safe fallback followed by canonical remote-name validation.
- Fixed iOS Keychain connection-preset replacement to update the existing item before falling back to add, preserving the last known-good preset if replacement cannot be created.
- Added cross-platform maintenance regressions for all four native fixes and the web version/security contract.
- Kept full Windows/Linux/macOS/Android/iOS build validation mandatory and expanded the release contract with ByFTP WEB.

## 1.6.0 — Repository-wide integrity and release hygiene

**Focus:** make every tracked repository file part of a fail-closed release-quality contract.

- Added `scripts/audit_repository.py`, which enumerates the repository with `git ls-files` and validates every tracked path/file.
- Added checks for portable paths, case-insensitive collisions, Windows-reserved names, tracked symlinks, generated/cache output, UTF-8/BOM/NUL problems, trailing whitespace, missing final newlines and merge-conflict remnants.
- Bound explicit current-release markers to root `VERSION` and added regression tests for repository hygiene.
- Removed the obsolete Android connection string resource instead of suppressing the lint warning.
- Preserved shared-hosting diagnostics, transfer progress, safe batch stop, SFTP host-key verification, FTPS platform trust and external production-signing boundaries.

## 1.5.0 — Shared-hosting connection diagnostics

**Focus:** explain the authenticated hosting environment without scanners, hidden probes, credential exposure or automatic path changes.

- Added transport-neutral connection diagnostics derived from the initial root listing already used to validate a connection.
- Added deterministic common web-root detection: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`.
- Diagnostics expose only derived non-secret facts and never persist or automatically open a detected web root.
- Windows, Android and iOS surface platform-appropriate shared-hosting status without changing the selected start path.
- Added regressions for secure/plain transport reporting, web-root priority, account/home semantics and common safe error classes.

## 1.4.0 — Mobile transfer progress and safe batch control

**Focus:** make long mobile transfers observable and controllable without weakening transport trust or protocol state.

- Added byte-level Android upload/download progress through transport-neutral monitored streams.
- Added iOS Network.framework file-transfer progress callbacks and SwiftUI progress presentation.
- Added safe **Stop after current file** behavior to multi-file upload on both mobile platforms.
- Kept the active FTP/FTPS transaction intact rather than simulating instant cancellation by tearing down a socket mid-command.
- Added regressions for cumulative byte accounting and retained existing TLS/host-key verification boundaries.

## 1.3.0 — Mobile file-manager parity and secret-lifetime hardening

**Focus:** make the native Android/iOS clients practical daily file managers while shortening secret lifetime.

- Added local search/filtering, deterministic directory-first sorting, direct path navigation and multi-file upload on Android and iOS.
- Added compact touch-friendly action surfaces for phones and tablets.
- Added restoration of non-secret connection metadata while structurally excluding stored mobile passwords/passphrases.
- Android app-private saved connection state remains excluded from backup/device transfer; iOS uses Keychain `WhenUnlockedThisDeviceOnly`.
- Raw remote names are validated instead of trimmed into different names.
- Added mobile model/security regressions and preserved exact protocol claims: Android supports FTP/explicit FTPS/implicit FTPS/SFTP; iOS supports FTP/implicit FTPS.
