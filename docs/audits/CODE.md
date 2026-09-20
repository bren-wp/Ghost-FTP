# Ghost FTP Code Audit — RC9

## Scope

RC9 continues the existing Ghost FTP production source. The production GUI remains the React + Tauri + Rust application in `desktop-tauri/`; compatibility Go hosts are retained only as developer/tooling surfaces and are not shipped as the production desktop UI.

## Corrected and hardened

- Removed the old save-connect-delete workaround for temporary Quick Connect sessions; ephemeral profiles remain memory-only.
- Persisted Site Manager favorites, bookmarks, tags, folders and last-used metadata.
- Re-applied persisted transfer concurrency, retry, throttle and delta-sync settings to the native engine at startup.
- Preferences Cancel restores the captured settings snapshot; Reset reapplies native defaults.
- Shell Integration is wired to the real per-user PATH integration path.
- Corrupt profile metadata and SQLite state are preserved/quarantined instead of crashing startup.
- Passwords and SSH passphrases remain outside profile JSON and use OS-protected credential storage where supported.
- File Properties uses real SHA-256 and permission operations where the active backend supports them; unsupported operations return explicit errors.
- Removed an unused Rust agent-service constant.
- Removed a redundant earlier dark-theme token block that was fully overridden by the canonical Ghost FTP visual-system block.
- Kept release-critical source paths stable while improving documentation and downloadable artifact naming to avoid breaking imports, build scripts or Tauri configuration.

## CI quality gates

The source audit runs:

- Go tests and `go vet` for runtime/installer tooling;
- JavaScript syntax checks for runtime, installer and website;
- `npm ci`, TypeScript typecheck and production Vite build;
- Rust `cargo fmt --check`, workspace check, tests and Clippy with warnings denied;
- legacy/demo-branding scan across production source surfaces.

## Naming policy

Runtime-critical paths, Rust crate/package names and Tauri identifiers are not renamed casually. Human-facing release artifacts use a clearer convention instead:

`GhostFTP-<Platform>-<Arch>-<Role>-v<Version>.<ext>`

This gives cleaner downloads without destabilizing source imports or installer/update identifiers.
