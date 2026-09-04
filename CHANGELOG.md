# Changelog

## 1.0.3 - 2026-09-05

- Required a canonical pinned SHA-256 SFTP host fingerprint before Web profiles can be persisted, while retaining the connection-boundary pin requirement as defense in depth.
- Centralized the Web editor/new-file 4 MiB content limit and required complete local staging writes before atomic remote promotion.
- Hardened SFTP temporary key handling with fail-closed Unix `0600` permissions before key material is written and exact write-length checks.
- Added SFTP upload size verification against remote metadata when available so staging fails closed on incomplete transfers.
- Made destructive Web batch deletion use a two-pass preflight so every item and type is validated before the first remote delete.
- Made batch rename reject malformed rows and duplicate source paths before any remote mutation.
- Removed duplicate archive-processing code commentary and expanded regression coverage for the new fail-closed boundaries.
- Bound documentation current-release markers and the 10-artifact/13-file release contract to the canonical VERSION audit.

## 1.0.2 - 2026-09-04

- Bounded the non-Windows in-memory runtime-secret store and made capacity exhaustion fail closed instead of allowing unbounded growth.
- Added runtime-secret regression coverage for copy isolation, capacity limits, cleanup and capacity reuse.
- Hardened Web credential envelope parsing with strict driver validation, maximum encoded size, explicit truncation checks and authenticated-tamper rejection tests.
- Removed a stale hard-coded release number from Composer metadata and added a runtime metadata regression test so human-readable package descriptions cannot drift from canonical versioning.
- Advanced the web PWA cache namespace and all canonical version surfaces to 1.0.2.
- Refreshed root documentation to describe the current Ghost FTP security, release and package contract without version-specific metadata drift.

## 1.0.1 - 2026-09-04

- Completed the hard-cut Ghost FTP identity across tracked paths, namespaces and runtime identifiers.
- Standardized Android, iOS, Go, Windows installer and web technical identities on GhostFTP/Ghost FTP.
- Added a fail-closed repository brand audit.
- Added Windows portable x64/x86 artifacts and GitHub Packages publication under `GhostFTP`.
- Verified the complete multi-platform release and registry readback pipeline.

## 1.0.0 - 2026-09-04

- Established **Ghost FTP** as the canonical public product identity across Windows, Linux, macOS, Android, iOS and Web/PWA surfaces.
- Started the Ghost FTP semantic-version line at **1.0.0** with sequential patch releases.
- Introduced namespaced release tags (`ghostftp-vX.Y.Z`) so current releases never collide with historical generic tags.
- Standardized Linux packaging as `ghost-ftp` with the `ghostftp` executable and Ghost FTP desktop entry.
- Established the multi-platform Release contract and SHA-256/build-metadata verification model.
- Preserved strict path validation, transfer staging/rollback, SFTP host-key verification, encrypted profile secrets, rate limiting, session hardening and release provenance controls.
- Kept mobile and desktop signing status explicit rather than representing unsigned/debug-signed artifacts as store-signed packages.

## Historical provenance

Git tags and commits created before the current Ghost FTP release sequence remain immutable for repository provenance and reproducibility. Current product releases exclusively use the `ghostftp-vX.Y.Z` namespace.
