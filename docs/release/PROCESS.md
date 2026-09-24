# Ghost FTP Release Process

This is the production release path for Ghost FTP release candidates. A successful compile is not equivalent to stable/FINAL acceptance.

## RC release flow

1. Start from current `main`.
2. Create a dedicated release-candidate branch.
3. Keep package, Cargo, Tauri, runtime, installer, updater and display versions consistent.
4. Update the changelog, release notes, active README/status documentation and artifact names.
5. Open a PR from the RC branch.
6. Require **Ghost FTP quality** on the exact PR HEAD.
7. Require **Ghost FTP protocol E2E** on the exact PR HEAD.
8. Require **Ghost FTP native build** on the exact PR HEAD.
9. Confirm Windows and Linux artifacts exist and inspect the Windows native QA evidence.
10. Merge only that tested source to `main`.
11. Add the version-specific approval marker containing the exact tested source SHA.
12. The version-specific release workflow verifies that SHA is an ancestor of `main`, rechecks all gates and packages from that exact source.
13. Publish an immutable prerelease tag and upload normalized native/source/documentation assets plus SHA-256 checksums.
14. Keep older tags/releases immutable.

## Current RC21 assets

Windows x64:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC21.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC21.exe`
- `GhostFTP-Windows-x64-v2.1.1-RC21.zip`
- `GhostFTP-Windows-x64-v2.1.1-RC21-Native-QA.zip`

Linux x86-64:

- `GhostFTP-Linux-x86_64-v2.1.1-RC21`
- `GhostFTP-Linux-x86_64-v2.1.1-RC21.AppImage`
- `GhostFTP-Linux-amd64-v2.1.1-RC21.deb`
- `GhostFTP-Linux-x86_64-v2.1.1-RC21.rpm`
- `GhostFTP-Linux-x86_64-v2.1.1-RC21.tar.gz`

Source/documentation:

- `GhostFTP-v2.1.1-RC21-Source.zip`
- `GhostFTP-v2.1.1-RC21-Desktop-Source.zip`
- `GhostFTP-v2.1.1-RC21-Website.zip`
- `GhostFTP-v2.1.1-RC21-Updates.zip`
- `GhostFTP-v2.1.1-RC21-Documentation.zip`
- `GhostFTP-v2.1.1-RC21-SHA256SUMS.txt`

## Windows QA evidence

The release workflow requires exactly seven critical surfaces in three viewport classes: canonical, compact and near-minimum. That is 21 PNG files plus 21 corresponding metadata TXT files. A blank/structureless, duplicate or incorrectly sized capture fails the native build gate.

## Stable / FINAL gates

Do not label an artifact FINAL until real evidence exists for the required installer lifecycle, target-OS visual acceptance, security failure paths, update verification, accessibility acceptance and production signing decision.

Blocked gates are BLOCKED, not PASS.

## Release integrity

- End-user GUI assets must come from `ghostftp-desktop/`.
- A browser-host/localhost wrapper is not a production release asset.
- Publication must fail if a required native platform artifact or QA set is absent.
- The approval marker must identify the exact previously tested source commit.
- Existing tags/releases are never moved merely to publish a newer RC.
- Public filenames use `GhostFTP`; product copy uses `Ghost FTP`.
