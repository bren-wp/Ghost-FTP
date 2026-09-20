# Ghost FTP Build Status — 20 September 2026

## Authoritative status

**Ghost FTP 2.1.1 RC7 — visual-parity/native-source hardening + newly rebuilt compatibility fallback packages. NOT FINAL.**

Development continued from the existing full source tree. No new replacement project was created and no screenshot-runtime implementation was introduced.

## Environment truth

- Node: v22.16.0
- npm: 10.9.2
- Go: 1.23.2
- TypeScript compiler: 5.8.3
- Rust/Cargo: **not installed**
- Complete `node_modules`: **not present** and external package fetch is unavailable
- `dpkg-deb`: available
- RPM/AppImage native packaging tools: **not available**
- Connected Windows test host: **offline** during this pass

Because Rust/Cargo and the complete frontend dependency tree are unavailable, this environment cannot truthfully compile or execute the native Tauri/Rust application. The native source is retained as the production implementation; the runnable Go binaries are explicitly compatibility fallbacks.

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

## Rebuilt compatibility artifacts

- `release/rc7/GhostFTP-Portable-2.1.1-RC7.exe` — Windows x64 PE GUI fallback.
- `release/rc7/GhostFTP-Setup-2.1.1-RC7.exe` — Windows x64 PE GUI fallback installer.
- `release/rc7/GhostFTP-linux-x86_64-2.1.1-RC7` — static Linux x86_64 fallback.
- `release/rc7/GhostFTP-linux-x86_64-2.1.1-RC7.tar.gz` — portable Linux fallback bundle.
- `release/rc7/ghostftp_2.1.1~rc7_amd64.deb` — Debian fallback package.

The fallback runtime reports `kind=compatibility-fallback` and `nativeProtocols=false`; it is not advertised as the production FTP/FTPS/SFTP engine.

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

FINAL still requires a machine with the native toolchain and target OS coverage to execute: native Tauri Windows/Linux compilation; frontend dependency build; actual FTP/FTPS/SFTP connection and transfer tests; Windows Setup/Portable/native frameless-titlebar screenshots; Windows 10/11 install/upgrade/uninstall tests; native DEB/RPM/AppImage generation; and screenshot/pixel comparison at every required responsive size. These are not marked passed without evidence.
