# Ghost FTP Release Process

This document describes the production release path. A successful compile is not equivalent to FINAL acceptance.

## RC release flow

1. Audit the current `main` branch and confirm the intended version.
2. Keep package, Cargo and Tauri version references consistent.
3. Update changelog, release notes, README download names and documentation.
4. Open a release-hardening PR rather than building from an unrelated source copy.
5. Require the `Ghost FTP quality` workflow to pass.
6. Require native Windows and Linux jobs in `Ghost FTP native build` to pass.
7. Inspect generated artifacts and Windows native-window QA evidence when available.
8. Merge only the tested source into `main`.
9. The main native build produces release binaries.
10. `Ghost FTP RC9 release` publishes only after a successful main native build and a successful quality run for the current production source.
11. Generate/upload SHA-256 checksums with the release assets.
12. Retain older releases.

## Required RC9 assets

Windows x64:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC9.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC9.exe`
- Windows bundle ZIP

Linux x86-64:

- native executable
- AppImage
- DEB
- RPM
- Linux bundle archive

Source/documentation:

- Full Source ZIP
- Desktop Source ZIP
- Website ZIP
- Updates ZIP
- Documentation ZIP
- SHA256SUMS file

## FINAL gates

Do not label an artifact FINAL until the QA documents contain real evidence for:

- Windows native titlebar/pixel acceptance;
- required responsive sizes;
- FTP/FTPS/SFTP end-to-end behavior;
- installer clean install/upgrade/reinstall/uninstall;
- update verification;
- required security failure paths;
- production signing decision.

Blocked gates must be recorded as BLOCKED, not PASS.

## Release integrity

- End-user GUI assets must come from `ghostftp-desktop/`.
- A browser-host/localhost wrapper is not a production release asset.
- Release publication must fail if required native platform bundles are absent.
- Existing tag/release history must not be deleted merely to publish a new RC.
- Public filenames use `GhostFTP`; product copy uses `Ghost FTP`.
