# Changelog

## 2.1.1-rc.9 — 20 September 2026

- Reorganized repository documentation under `docs/` and removed duplicate root copies while preserving stable runtime source paths.
- Rewrote the main README as a product/marketing landing page with Ghost FTP branding and visual reference screenshots.
- Added the approved reference imagery to `docs/assets/screenshots/` for README and pixel QA use only.
- Removed obsolete RC7/RC8 release workflows and consolidated publication into a single native RC9 workflow.
- Improved release artifact naming with platform, architecture, role and version encoded consistently.
- Added stronger CI checks for frontend, Go and Rust workspace quality, including formatting, tests and Clippy.
- Removed an unused Rust agent-service constant and redundant default dark-theme token block.
- Aligned the default Electric Blue and Ice White tokens more closely with the approved Ghost FTP palette.
- Preserved the canonical 1290×852 1:1 shell geometry and adaptive smaller-window rules.
- RC9 remains a pre-release pending final native Windows titlebar/pixel, installer lifecycle and real FTP/FTPS/SFTP acceptance evidence.


## 2.1.1-rc.8 — 20 September 2026

- Removed browser-shell compatibility executables from the end-user release path after the visible 127.0.0.1 Chromium/Edge bar regression was reproduced.
- Production GUI releases now come only from the native Tauri/WebView pipeline.
- Fixed encrypted-backup magic-header length compilation error in the Rust backend.
- Fixed Windows RC packaging by using NSIS for prerelease builds instead of invalid MSI prerelease metadata.
- Native build pipeline now emits Windows native portable EXE + NSIS setup and Linux native binary + DEB/RPM/AppImage artifacts.
- Public release publishing is gated on a successful native Windows/Linux build.
- Kept the frameless custom Ghost FTP window, 1290×852 reference geometry and adaptive smaller-window rules.


## 2.1.1-rc.7 — 20 September 2026

- Reworked Site Manager, Preferences, Transfer Center and About into reference-style full application surfaces instead of generic modal cards.
- Added reusable frameless Ghost FTP titlebar/menu chrome and tightened adaptive behavior for smaller windows without removing critical actions.
- Matched New Connection and File Properties closer to the approved reference geometry.
- Added real SHA-256 checksum commands for local and SSH/SFTP paths and recursive chmod where supported.
- Rebuilt File Properties with General/Checksums tabs, chmod matrix, numeric mode, duplicate and open-containing-folder actions.
- Brought compatibility runtime surfaces closer to the same Windows/Linux visual structure used by the native source.
- Rebuilt Windows x64 and Linux x86-64 compatibility artifacts from RC7 source and refreshed package metadata.

## 2.1.1-rc.4 — 20 September 2026

- Hardened native startup against corrupted profile metadata and SQLite state; broken stores are preserved for recovery instead of causing a startup panic.
- Kept the frameless 1290×852 reference window and 480×600 minimum adaptive viewport contract.
- Revalidated compatibility runtime security boundaries, local filesystem operations, state recovery, branding scan and package metadata.
- Regenerated Windows/Linux compatibility artifacts and release checksums from the updated source tree.


## 2.1.1 RC3 — 20 September 2026

- Added true ephemeral Quick Connect sessions in the native Rust/Tauri source.
- Added split FTP/FTPS/SFTP Quick Connect protocol selection in the reference titlebar.
- Persisted Site Manager favorites, bookmarks, tags, folders and last-used metadata.
- Made native transfer retry count configurable and re-applied persisted engine settings at startup.
- Corrected Preferences Cancel/Reset semantics and wired Shell Integration to the real per-user PATH integration.
- Centralized React release metadata in the RC3 line; current RC7 metadata is 2.1.1-rc.7 / build 2026.09.20.6.
- Reached 185-key parity across every advertised non-English native dictionary.
- Removed inline language-switch JavaScript from all 14 localized website pages.
- Rebuilt Windows/Linux compatibility fallback artifacts and the Debian fallback package.
- Re-ran local runtime security/filesystem QA and Go test/vet checks.

## 2.1.0 — 2026-09-19 source/release-candidate hardening

- Continued the existing Ghost FTP v9 source tree rather than replacing it.
- Preserved the native React/Tauri/Rust protocol engine and verified the Tauri main window is frameless with custom decorations disabled.
- Reduced the native minimum window width to 480 px and added compact CSS breakpoints down to the required narrow desktop sizes without intentionally removing toolbar actions.
- Hardened native profile persistence with staged writes, a last-known-good backup and corrupt-file preservation.
- Reworked About/Updates source so it does not claim "up to date" before a real updater check and reports the current platform.
- Rebuilt the Go compatibility Windows/Linux hosts after removing fake connection-success and simulated transfer-progress behavior; fallback networking is now explicitly TCP reachability only.
- Added custom titlebar drag support to the Windows compatibility host and setup source.
- Hardened setup ordering so future steps cannot be clicked, embedded the full commercial EULA, added custom installation-folder selection/validation, improved upgrade rollback and corrected publisher metadata.
- Added/updated release, security, privacy, installation, uninstall, support and QA documentation.

This package is **not labelled FINAL** because a native Tauri build, Windows titlebar screenshot verification, native protocol end-to-end tests, full 14-language parity and native AppImage/RPM production packages could not all be executed in this environment.

### RC7 parity and delivery pass
- Matched the compatibility New Connection surface to the measured 752×628 reference envelope.
- Added the reference-scale Site Manager identity block and refined full-window secondary surfaces.
- Added local Ghost FTP mountain artwork to the About hero without using a reference screenshot as runtime UI.
- Executed automated render QA at all nine required viewport sizes with no whole-window horizontal overflow and no page errors.
- Rebuilt Windows x64 and Linux x86-64 compatibility artifacts from RC7 source.
- Prepared repository bootstrap automation for the official `bren-wp/Ghost-FTP-Premium` GitHub repository so the current source can be reconstructed and committed by GitHub Actions without relying on this sandbox's blocked outbound Git transport.
