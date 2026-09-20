# Ghost FTP Build Status — 20 September 2026

## Authoritative status

**Ghost FTP 2.1.1 RC9 — native Tauri release-candidate hardening. NOT FINAL.**

Development continued from the existing full source tree. No new replacement project was created and no screenshot-runtime implementation was introduced.

## Build environment truth

- GitHub Actions now provides the authoritative native build environment for Windows and Linux.
- Node/npm, Rust/Cargo and Tauri platform prerequisites are installed by `.github/workflows/native-build.yml`.
- Frontend TypeScript/Vite compilation is required before every native bundle step.
- Native RC9 bundle targets are Windows NSIS + portable native EXE and Linux DEB/RPM/AppImage + native executable. MSI is intentionally omitted for prerelease versions because WiX/MSI rejects the `rc.9` prerelease identifier.
- The Go `runtime/` and `installer/` trees remain developer/compatibility tooling only and are **not** the production GUI release path.
- Browser-shell compatibility executables were removed from the public RC9 release after QA showed a visible `127.0.0.1` Chromium/Edge app bar that cannot satisfy the 1:1 frameless requirement.

## RC9 source improvements

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

Production-facing Windows/Linux executables come from the native Tauri build. Compatibility browser-host artifacts are not published as end-user RC9 binaries. The RC9 source is prepared for a fresh native Windows/Linux CI build and GitHub pre-release. Publication remains gated on a successful current-main native build plus a successful source audit.

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
- Linux fallback reports version `2.1.1-rc.9`: PASS.
- Windows outputs identify as PE32+ x64 GUI executables: PASS.
- Linux output identifies as stripped static ELF x86-64: PASS.

## Native RC9 build evidence

- Windows native Tauri application build: PENDING current RC9 CI.
- Windows NSIS setup bundle: PENDING current RC9 CI.
- Windows native portable `ghostftp.exe`: PENDING current RC9 CI.
- Linux native Tauri application build: PENDING current RC9 CI.
- Linux DEB bundle: PENDING current RC9 CI.
- Linux RPM bundle: PENDING current RC9 CI.
- Linux AppImage bundle: PENDING current RC9 CI.
- Native Windows/Linux workflow run: PENDING current RC9 CI.
- GitHub RC9 pre-release publication: PENDING current RC9 CI.

## Gates still blocking FINAL

FINAL still requires actual FTP/FTPS/SFTP end-to-end connection and transfer tests against test servers, a Windows 10/11 frameless-titlebar screenshot confirming the absence of any extra browser/OS title strip, Windows install/upgrade/uninstall acceptance tests, and final pixel/responsive comparison at every required reference size. These are not marked passed without evidence.
