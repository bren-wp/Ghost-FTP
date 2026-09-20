# Ghost FTP Code Audit — RC10

## Scope

RC10 continues the existing Ghost FTP production source. The authoritative desktop application now lives in `ghostftp-desktop/`. The Go-based `tools/ghostftp-runtime/` and `tools/ghostftp-installer/` trees are retained only as developer/compatibility tooling and are not shipped as the production desktop GUI.

## Corrected and hardened

- Removed the old save-connect-delete workaround for temporary Quick Connect sessions; ephemeral profiles remain memory-only.
- Persisted Site Manager favorites, bookmarks, tags, folders and last-used metadata.
- Re-applied transfer concurrency, retry, throttle and delta-sync settings to the native engine at startup.
- Preferences Cancel restores the captured settings snapshot; Reset reapplies Ghost FTP defaults.
- Shell integration is wired to the real per-user PATH integration path.
- Corrected the managed PATH comparison so historical `Ghost FTP` and `GhostFTP` app-directory aliases are treated as the same Ghost FTP-owned segment without stripping spaces from unrelated Windows paths.
- Corrupt profile metadata and SQLite state are preserved/quarantined instead of causing a startup crash.
- Passwords and SSH passphrases remain outside ordinary profile JSON and use OS-protected credential storage where supported.
- File Properties uses real SHA-256 and permission operations where the active backend supports them; unsupported operations return explicit errors.
- Removed an unused Rust agent-service constant.
- Removed a redundant dark-theme token block that was fully overridden by the canonical Ghost FTP visual-system block.
- Corrected Transfer Queue UX/state behavior around completed, skipped and canceled transfers.
- Completed transfers no longer expose a meaningless Cancel action.
- Added Retry All for failed transfers.
- Replaced dynamic inline-width transfer progress with semantic progress controls and centralized CSS.
- Validated transfer-throttle input before applying it.
- Exposed existing browser-layout, hidden-file, preview, download-folder and editor settings in Preferences.
- Website timer cleanup stops background interval work after completion and respects reduced-motion preferences.
- Removed the browser-host GUI from the production release path after the visible `127.0.0.1` chrome regression was reproduced.
- Reorganized source roots under GhostFTP-branded directories without changing migration-sensitive product identifiers.

## Repository naming hardening

Product-facing source roots are now:

- `ghostftp-desktop/`
- `tools/ghostftp-runtime/`
- `tools/ghostftp-installer/`
- `website/`
- `updates/`

Public artifacts follow:

`GhostFTP-<Platform>-<Arch>-<Role>-v<Version>.<ext>`

Internal framework-required names remain only where the build ecosystem consumes them directly.

## CI quality gates

`.github/workflows/ghostftp-quality.yml` runs:

- Go tests and `go vet` for GhostFTP runtime/installer tooling;
- JavaScript syntax checks for runtime, installer and website;
- website markup policy checks;
- `npm ci`, TypeScript typecheck and production Vite build;
- Rust formatting check;
- Rust workspace check;
- Rust workspace tests;
- Clippy with warnings denied;
- legacy/demo-branding scans across production source surfaces.

`.github/workflows/ghostftp-build.yml` separately produces the Windows and Linux production bundles used for release publication.

## Current quality truth

An earlier RC9 quality run exposed PATH-integration failures and later RC9 validation narrowed the remaining gate to two CLI Clippy findings. RC10 contains both corrections and still requires a fully successful current quality run before publication.

## Remaining engineering validation

Code-level CI does not replace target-OS acceptance. FINAL still requires real Windows visual/titlebar acceptance, installer lifecycle acceptance and real FTP/FTPS/SFTP integration tests.
