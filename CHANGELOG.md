# Changelog

## 1.0.1 - 2026-09-04

- Completed the hard-cut Ghost FTP rebrand across tracked paths, namespaces and runtime identifiers.
- Removed compatibility identifiers tied to the retired product identity.
- Renamed Android, iOS, Go, Windows installer and web technical identities to GhostFTP/Ghost FTP.
- Added a fail-closed repository brand audit.
- Prepared Windows portable artifacts and GitHub Packages publication under GhostFTP.

All notable Ghost FTP changes are documented here. Ghost FTP uses semantic versioning in the `1.0.x` line beginning with the rebrand release.

## 1.0.0 - 2026-09-04

- Rebranded the public product from GhostFTP to **Ghost FTP** across the core application identity, Android, iOS, macOS, Linux and the web/PWA surface.
- Restarted the Ghost FTP product version line at **1.0.0**. Future patch releases increment sequentially as `1.0.1`, `1.0.2`, and so on.
- Introduced namespaced Ghost FTP Git release tags (`ghostftp-v1.0.0`, `ghostftp-v1.0.1`, …) so historical GhostFTP tags are never rewritten or moved.
- Reworked Linux packaging as `ghost-ftp` with the `ghostftp` executable and a Ghost FTP desktop entry.
- Simplified public Releases to clear per-platform packages: Windows x64/x86/x32 alias installers, one Linux multi-architecture archive, one universal macOS package, one Android APK, one iOS IPA and one web archive.
- Kept cryptographic release verification through SHA-256 checksums and explicit build metadata.
- Preserved existing security controls for path validation, transfer staging/rollback, SFTP host-key verification, encrypted profile secrets, rate limiting, session hardening and release provenance.
- Kept signing status explicit: unsigned or debug-signed mobile artifacts are never presented as store-signed production packages.
- Updated repository and support links to `bren-wp/Ghost-FTP`.

## Pre-1.0.0 history

Releases and tags from the former GhostFTP product line remain in Git history for provenance and reproducibility. They are not part of the Ghost FTP semantic-version sequence and are not rewritten. In particular, the historical `v1.0.0` tag remains immutable; Ghost FTP uses `ghostftp-v1.0.0` for its new 1.0.0 release.
