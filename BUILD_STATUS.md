# Ghost FTP Build Status — 20 September 2026

## Authoritative status

**Ghost FTP 2.1.1 RC7 — native Tauri release-candidate hardening. NOT FINAL.**

Development continued from the existing full source tree. No new replacement project was created and no screenshot-runtime implementation was introduced.

## Build environment truth

- GitHub Actions now provides the authoritative native build environment for Windows and Linux.
- Node/npm, Rust/Cargo and Tauri platform prerequisites are installed by `.github/workflows/native-build.yml`.
- Frontend TypeScript/Vite compilation is required before every native bundle step.
- Native bundle targets remain Windows NSIS/MSI and Linux DEB/RPM/AppImage.
- The Go `runtime/` and `installer/` trees remain developer/compatibility tooling only and are **not** the production GUI release path.
- Browser-shell compatibility executables were removed from the public RC7 release after QA showed a visible `127.0.0.1` Chromium/Edge app bar that cannot satisfy the 1:1 frameless requirement.

## RC7 source improvements

- Converted Site Manager, Preferences, Transfer Center and About into real full-window Ghost FTP application surfaces with shared custom window chrome instead of generic centered overlays.
- Tightened reference geometry around the 1290×852 canonical desktop frame and added local-scroll adaptive rules instead of hiding critical controls at narrow sizes.
- Rebuilt New Connection around the reference 752×628 modal envelope and File Properties around the reference 530×770 envelope.
- Added real SHA-256 checksum and recursive chmod IPC paths for local and SSH/SFTP backends, with explicit unsupported errors for protocols that cannot provide those operations.
- File Properties now exposes General/Checksums, numeric chmod, owner/group display without fabricated values, recursive permissions, preserve-timestamp and synchronization controls, Open Containing Folder and Duplicate.
- Mirrored the same full-window Site Manager/Preferences/Transfer/About structure in the compatibility host for closer design parity during non-native QA.

- Corrupted `profiles.json` and backup metadata are preserved as recovery copies; Ghost FTP continues with an empty profile store instead of crashing startup.
- `ghostftp.db` startup failure now quarantines the database plus WAL/SHM sidecars and retries with a clean database instead of panicking.
- Real ephemeral Quick Connect command: temporary sessions no longer save then delete a profile.
- Quick Connect split protocol selector for FTP/FTPS/SFTP with protocol-default ports.
- Site Manager persistence for favorites, bookmarks, tags, folders and last-used metadata.
- Recent Servers is now based on persisted successful connection time rather than only the current process session.
- Persisted transfer concurrency, retry count, throttle and delta-sync settings are re-applied to the native transfer engine at startup.
- Transfer retry count is live-configurable instead of a hard-coded constant.
- Preferences Cancel restores the complete settings snapshot; Reset re-applies native engine defaults.
- Shell integration uses the real per-user Ghost FTP PATH integration command instead of a cosmetic toggle.
- Native profile passwords/private-key passphrases remain OS-keychain-backed and are not persisted in profile JSON.
- Native Tauri window remains frameless with `decorations(false)` and the 1290×852 reference geometry.
- All 13 non-English native dictionaries now match the same 185-key canonical set used by Croatian coverage.
- Website language switcher no longer uses inline JavaScript; 14 localized pages remain `notranslate` and responsive.

## Public release artifact policy

Production-facing Windows/Linux executables must come from the native Tauri build. Compatibility browser-host artifacts are not published as end-user RC7 binaries. The RC7 release is kept as a pre-release until native Windows/Linux bundles and the remaining OS acceptance gates pass.

## Executed QA in this pass

- `go test ./...` — runtime: PASS.
- `go vet ./...` — runtime: PASS.
- `go test ./...` — installer: PASS.
- `go vet ./...` — installer: PASS.
- Runtime JavaScript syntax: PASS.
- Website JavaScript syntax: PASS.
- 14 website entry pages metadata/notranslate/language selector audit: PASS.
- Native locale key parity: PASS — HR/DE/FR/ES/IT/PT/NL/PL/SL/SR/BS/MK/SQ each expose 185 keys.
- Fallback runtime API: PASS for authorization rejection, mkdir, listing, SHA-256, duplicate and root-delete protection.
- Linux fallback reports version `2.1.1-rc.7`: PASS.
- Windows outputs identify as PE32+ x64 GUI executables: PASS.
- Linux output identifies as stripped static ELF x86-64: PASS.

## Gates still blocking FINAL

FINAL still requires the native GitHub Actions Windows/Linux build to complete successfully, followed by actual FTP/FTPS/SFTP connection and transfer tests, Windows frameless-titlebar screenshots, Windows 10/11 install/upgrade/uninstall tests, native package verification and screenshot/pixel comparison at every required responsive size. These are not marked passed without evidence.
