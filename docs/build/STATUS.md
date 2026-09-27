# Ghost FTP Build Status — 27 September 2026

## Authoritative current state

- **Active source/release cycle:** Ghost FTP **0.19.0**.
- **Latest published canonical release:** Ghost FTP **0.18.0**.
- **Live publication status:** GitHub Releases is authoritative and queried by CI.
- **Version source of truth:** `version.json`.
- **Desktop production source:** `ghostftp-desktop/`.
- **Android production source:** `android/`.

The production desktop GUI is the native React + TypeScript + Tauri + Rust application. Go tooling under `tools/` is support/compatibility tooling and is not the authoritative end-user desktop GUI.

## Current CI / build organization

- Quality: `.github/workflows/ghostftp-quality.yml` — **Ghost FTP quality**
- Protocol E2E: `.github/workflows/ghostftp-protocol-e2e.yml` — **Ghost FTP protocol E2E**
- Canonical Windows/Linux build + native-window QA: `.github/workflows/ghostftp-build.yml` — **Ghost FTP native build**
- Android: `.github/workflows/ghostftp-android.yml` — **Ghost FTP Android**
- Windows hardening: `.github/workflows/validate-win-hardening.yml` — **Validate Windows hardening**
- Canonical release orchestration: `.github/workflows/ghostftp-release.yml` — **Ghost FTP release**
- Version/Cargo metadata synchronization: `.github/workflows/version-sync.yml`
- Dependency-manifest Cargo lock refresh: `.github/workflows/cargo-lock-refresh.yml`

The obsolete duplicate Windows/Linux native build workflow has been removed. The canonical native build now supplies both release binaries and Windows native-window QA evidence.

## 0.19.0 packaging and size hardening

Windows Setup now uses the canonical Tauri NSIS package with Ghost FTP branding, installer/uninstaller icons, branded header/sidebar bitmaps and `EULA.txt` as the interactive licence page. The canonical native build silently installs and uninstalls that exact Setup package as a CI lifecycle smoke test.

Linux CI now verifies AppImage runtime metadata, DEB package metadata/install/remove behavior and RPM metadata before native artifacts are uploaded.

The published 0.18.0 size baseline is approximately:

- Windows portable executable: **22.93 MiB**
- Windows NSIS Setup: **7.60 MiB**
- Linux native executable: **25.73 MiB**
- Linux AppImage: **86.67 MiB**
- Linux DEB/RPM: **about 11.11 MiB each**

0.19.0 uses Cargo `opt-level = "s"` with LTO, one codegen unit, panic abort and stripped symbols. Optional AppImage media-framework bundling remains disabled. CI also enforces conservative upper size budgets and reports exact package sizes for every canonical native build. These budgets prevent accidental growth; they are not release-size targets.

## Documentation image provenance

Quality CI resolves the newest non-draft GitHub Release tag and verifies:

- all local images referenced by repository Markdown;
- all files under `docs/assets/screenshots/`;
- each current image blob equals the same path in the latest release tag.

This prevents unreleased screenshots from being presented as current release imagery.

## Required exact-head gates

Each release-relevant PR must pass:

- Ghost FTP quality
- Ghost FTP protocol E2E
- Ghost FTP native build
- Ghost FTP Android
- Validate Windows hardening

Evidence applies only to the exact tested source SHA.

## Platform artifact scope

### Windows x64

- native portable executable;
- NSIS Setup executable;
- Windows archive;
- native-window QA evidence archive.

### Linux x86-64

- native executable;
- AppImage;
- DEB;
- RPM;
- Linux archive.

### Android

- installable APK from the verified Android workflow artifact;
- package/signature checks via `apksigner`;
- separate unsigned release-check APK used for validation, not as the canonical end-user APK.

### Source/support assets

- full source archive;
- desktop source archive;
- Android source archive;
- update-service metadata/tooling archive;
- documentation archive;
- SHA-256 checksum file.

## Release truth

`0.19.0` is the active cycle after published `0.18.0`. Whether a version is currently published is determined by the canonical GitHub Release/tag, not by documentation wording or a successful compile.

Existing version tags are immutable. Stable/FINAL status is separate from publishing a pre-1.0 release and still requires the target-OS lifecycle, visual, security, accessibility and signing acceptance described in QA/release documentation.
