# Ghost FTP Build Status — 27 September 2026

## Authoritative current state

- **Active source:** Ghost FTP **0.17.0** development preview.
- **Latest published release:** Ghost FTP **0.16.0**.
- **Version source of truth:** `version.json`.
- **Desktop production source:** `ghostftp-desktop/`.
- **Android production source:** `android/`.
- **Website source:** `website/`.

The production desktop GUI is the native React + TypeScript + Tauri + Rust application. Go tooling under `tools/` is support/compatibility tooling and is not the authoritative end-user desktop GUI.

## Current CI / build organization

- Quality: `.github/workflows/ghostftp-quality.yml` — **Ghost FTP quality**
- Protocol E2E: `.github/workflows/ghostftp-protocol-e2e.yml` — **Ghost FTP protocol E2E**
- Native preview bundles: `.github/workflows/ghostftp-native-preview.yml` — **Ghost FTP native preview build**
- Native production build gate: `.github/workflows/ghostftp-build.yml` — **Ghost FTP native build**
- Android: `.github/workflows/ghostftp-android.yml` — **Ghost FTP Android**
- Windows hardening: `.github/workflows/validate-win-hardening.yml` — **Validate Windows hardening**
- Canonical release orchestration: `.github/workflows/ghostftp-preview-release.yml` — **Ghost FTP release**
- Version synchronization: `.github/workflows/version-sync.yml`

## Exact-head evidence

PR #37 head `d471fdd77c0d99b2ca2834a71eb528e04813b936` passed all six primary gates before merge:

- Ghost FTP quality
- Ghost FTP protocol E2E
- Ghost FTP native preview build
- Ghost FTP native build
- Ghost FTP Android
- Validate Windows hardening

That evidence applies to that exact source SHA. Future changes require fresh exact-head validation.

## Platform artifact scope

### Windows x64

- native portable executable;
- NSIS Setup executable;
- Windows archive;
- native-window QA evidence where produced by the build workflow.

### Linux x86-64

- native executable;
- AppImage;
- DEB;
- RPM;
- Linux archive.

### Android

- installable APK from the verified Android workflow artifact;
- package id `com.ghostftp.android.preview` for the installable preview artifact;
- signature verification via `apksigner`;
- separate unsigned release-check APK used for validation, not as the canonical end-user APK.

### Source/support assets

- full source archive;
- desktop source archive;
- Android source archive;
- website archive;
- update metadata archive;
- documentation archive;
- SHA-256 checksum file.

## Release truth

`0.17.0` is **not published** merely because the source version exists or CI compiles. The latest published release remains `0.16.0` until the canonical release workflow creates and verifies `v0.17.0`.

Stable/FINAL status is separate from publishing a development release and still requires the target-OS lifecycle, visual, security, accessibility and signing acceptance described in the QA/release documentation.
