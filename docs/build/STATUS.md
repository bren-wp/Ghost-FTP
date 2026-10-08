# Ghost FTP Build Status — 6 October 2026

## Authoritative current state

- **Active source/release cycle:** Ghost FTP **0.30.14**.
- **Previous canonical release:** Ghost FTP **0.30.10**.
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
- macOS Preview: `.github/workflows/ghostftp-macos.yml` — **Ghost FTP macOS**
- Canonical release orchestration: `.github/workflows/ghostftp-release.yml` — **Ghost FTP release**
- Version/Cargo metadata synchronization: `.github/workflows/version-sync.yml`
- Dependency-manifest Cargo lock refresh: `.github/workflows/cargo-lock-refresh.yml`

The obsolete duplicate Windows/Linux native build workflow and website application surface have been removed. The canonical native build now supplies both release binaries and Windows native-window QA evidence. Android runs on every release-relevant PR and every `main` push and includes an emulator click-through smoke.

## 0.20.8 cross-app action parity hardening

Android now exposes real protocol-backed Rename for FTP, explicit FTPS and SFTP, safe deletion of files or empty folders, and working Settings actions instead of an informational-only Settings surface. Folder rows support long-press selection so folder rename/delete actions can target a directory without navigating into it.

The Android production contract and emulator smoke now require the shared action surface and Settings controls to remain wired and clickable. Non-secret rename state survives Activity recreation; passwords and authenticated sessions still do not.

## 0.20.7 dead-code and source-reachability hardening

Quality now treats unused TypeScript locals and unreachable frontend source files as release-blocking failures. The source-reachability check starts from the production desktop entrypoint and the shared file-ui package entrypoint, resolves project aliases/relative imports, and rejects TypeScript files outside that active graph.

The cleanup is intentionally evidence-driven: host/package utility copies that serve different boundaries are retained, while only demonstrably unreachable files or unused symbols are removed. The first enforced audit removed 11 unreachable frontend files and 21 compiler-reported unused imports/declarations; the resulting production graph reports 102 reachable TypeScript files.

## 0.20.6 frontend bundle-size hardening

Desktop production builds no longer eager-bundle the complete Material Icon Theme catalog. Ghost FTP ships an explicit offline subset for common file types and uses the existing Lucide fallback for unmatched files. File-browser, i18n and generated brand-icon data are emitted as dedicated chunks.

Every production desktop build now runs a post-build JavaScript budget check and fails if any emitted JS chunk exceeds 500 KiB.

## 0.20.5 Android API hardening

Android API 35+ now applies system-bar insets to the programmatic root view instead of relying on deprecated status/navigation bar color setters. The API 35 theme keeps the dark Ghost FTP system-bar appearance without deprecated color attributes. SFTP password handoff now uses JSch's byte-array API and clears the temporary UTF-8 buffer after the library copies it.

The Android production contract rejects the deprecated String password setter and direct system-bar color setters so this behavior cannot silently regress.

## 0.20.0 recovery and cross-platform hardening

The 0.20.0 cycle adds durable credential-free desktop transfer recovery, safe cross-process retry semantics, runtime-bounded transfer history, stricter FTP/FTPS resume verification, real FTP/explicit-FTPS pause/cancel/cleanup E2E coverage and real OpenSSH SFTP non-zero-offset resume primitive coverage. Android now uses lifecycle-aware cancellation through remote mutations, closes upload streams at the ownership boundary, preserves document-picker access with lint-safe SAF grant modes, and fails closed when stable signing continuity cannot be verified.

The same exact-source Windows/Linux native build, Android, Quality, Protocol E2E and Windows hardening gates remain release blockers.

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

- intentionally unsigned production APK named `GhostFTP-Android-v<version>.apk.unsigned`, verified as the production `com.ghostftp.android` package but not presented as directly installable;
- separate non-debuggable installable preview APK used only for clean-install, reinstall and launch validation;
- package identity and APK structure checks via Android build tooling;
- emulator click-through test and real emulator UI screenshot evidence in the Android CI artifact.

### Source/support assets

- full source archive;
- desktop source archive;
- Android source archive;
- update-service metadata/tooling archive;
- documentation archive;
- SHA-256 checksum file.

## Release truth

`0.20.4` is the active cycle after published `0.20.3`. Live publication status is determined by the canonical GitHub Release/tag, not by documentation wording or a successful compile.

Existing version tags are immutable. Stable/FINAL status is separate from publishing a pre-1.0 release and still requires the target-OS lifecycle, visual, security, accessibility and signing acceptance described in QA/release documentation.
