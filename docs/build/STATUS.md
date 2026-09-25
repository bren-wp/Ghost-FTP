# Ghost FTP Build Status — 25 September 2026

## Authoritative source status

**Ghost FTP 2.1.1 RC23 — all-platform release candidate. NOT FINAL until all gates and release packaging pass.**

The authoritative desktop application is `ghostftp-desktop/`. Production end-user GUI releases come from the native React + TypeScript + Tauri + Rust path, not a localhost/browser-shell compatibility host.

The authoritative Android application is `android/`. Production Android release assets come from the native Android workflow and are pulled into the same GitHub Release as the Windows and Linux assets.

The authoritative public website source is `website/`. RC23 includes production routes for `/download/`, `/security/` and `/support/` plus the localized landing pages.

## Build organization

- Desktop source: `ghostftp-desktop/`
- Android source: `android/`
- Website source: `website/`
- Update manifest templates: `updates/`
- Quality workflow: `.github/workflows/ghostftp-quality.yml`
- Protocol E2E workflow: Ghost FTP protocol E2E
- Historical native build workflow: `.github/workflows/ghostftp-build.yml`
- RC23 native build workflow: `.github/workflows/ghostftp-native-rc23.yml`
- Android workflow: `.github/workflows/ghostftp-android.yml`
- Versioned release workflow: `.github/workflows/ghostftp-rc23-release.yml`

## RC23 quality gates

The exact RC23 candidate must pass:

- npm clean install;
- npm production dependency audit at high severity;
- i18n parity check;
- single-window/UI contract and privacy/security regression guards;
- TypeScript typecheck and production frontend build;
- Go tests and `go vet` for repository tooling;
- website syntax and markup policy checks;
- update-script syntax checks;
- Rust formatting;
- Rust workspace check;
- Rust workspace tests;
- Clippy with warnings denied;
- real FTP, explicit FTPS and SFTP roundtrip E2E;
- Windows x64 RC23 native bundle;
- Linux x86-64 RC23 native binary/AppImage/DEB/RPM;
- Android lint/build and APK artifact generation;
- RC23 source, website, update, documentation and checksum packages.

## RC23 platform scope

### Windows x64

- portable Ghost FTP EXE;
- NSIS Setup EXE;
- Windows bundle ZIP.

### Linux x86-64

- native executable;
- AppImage;
- DEB;
- RPM;
- Linux bundle archive.

### Android

- installable APK;
- FTP, explicit FTPS and SFTP listing/download/upload/delete/new-folder actions;
- strict SFTP host-key fingerprint verification;
- explicit FTPS protected data channel;
- guarded upload/delete UX;
- bounded activity log and lifecycle-safe UI updates.

### Website

- English homepage with Windows/Linux/Android RC23 copy;
- Croatian homepage with Windows/Linux/Android RC23 copy;
- `/download/` page;
- `/security/` page;
- `/support/` page;
- sitemap entries for production pages and localized landing pages.

## Release truth

A successful build does not equal FINAL acceptance. RC23 publication uses an immutable version tag pointing at the exact source commit that passed the required workflows. Earlier release candidates remain unchanged.

RC23 must not be described as issued until `v2.1.1-rc.23` exists as a GitHub pre-release and includes Windows, Linux, Android, website/source/documentation and SHA-256 checksum assets.
