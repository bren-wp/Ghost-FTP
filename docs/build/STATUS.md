# Ghost FTP Build Status — 24 September 2026

## Authoritative source status

**Ghost FTP 2.1.1 RC19 — native Windows/Linux release-candidate source. NOT FINAL.**

The authoritative desktop application is `ghostftp-desktop/`. Production end-user GUI releases come from the native React + TypeScript + Tauri + Rust path, not a localhost/browser-shell compatibility host.

## Build organization

- Desktop source: `ghostftp-desktop/`
- Quality workflow: `.github/workflows/ghostftp-quality.yml`
- Protocol E2E workflow: Ghost FTP protocol E2E
- Native build workflow: `.github/workflows/ghostftp-build.yml`
- Versioned release workflow: `.github/workflows/ghostftp-rc19-release.yml`
- Release approval marker: `.github/RC19_RELEASE_APPROVED` after the exact candidate is validated and merged

## RC19 quality gates

The exact RC19 candidate must pass:

- npm clean install;
- npm production dependency audit at high severity;
- i18n parity check;
- single-window/UI contract and privacy/security regression guards;
- TypeScript typecheck and production frontend build;
- Go tests and `go vet` for repository tooling;
- website and updater-script syntax/policy checks;
- Rust formatting;
- Rust workspace check;
- Rust workspace tests;
- Clippy with warnings denied;
- real FTP, explicit FTPS and SFTP roundtrip E2E;
- Windows x64 native bundle;
- Linux x86-64 native binary/AppImage/DEB/RPM;
- Windows native visual evidence for seven critical surfaces at canonical, compact and near-minimum viewports;
- RC19 source/documentation packages.

## RC19 stability/privacy delta

- Terminal suggestion history is memory-only and never written to WebView localStorage.
- Terminal listener registration is disposal-aware and cleans partial async startup.
- Transfer listeners start before the initial snapshot, and revision guards prevent stale snapshots from overwriting newer live events.
- Commands that look credential-bearing are excluded from suggestion history.
- Notification-center history is memory-only.
- User-facing diagnostic/toast text is centrally credential-redacted.
- Startup migration deletes legacy persisted terminal/notification history created by older candidates.
- The CLI executable replacement path remains fail-closed until signed package verification is available.

## Native package targets

### Windows x64

- portable Ghost FTP EXE;
- NSIS Setup EXE;
- Windows bundle ZIP;
- 21-image + 21-metadata native QA bundle.

### Linux x86-64

- native executable;
- AppImage;
- DEB;
- RPM;
- Linux bundle archive.

## Release truth

A successful build does not equal FINAL acceptance. RC19 publication uses an immutable version tag pointing at the exact source commit that passed the required workflows. Earlier release candidates remain unchanged.
