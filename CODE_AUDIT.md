# Ghost FTP Code Audit — RC3

## Corrected in this pass

- Removed the save-connect-delete workaround for non-persistent Quick Connect. Native Rust now accepts an in-memory `ConnectionProfile` through `connect_ephemeral`; the frontend owns that profile only while its session is live.
- Added proper lifecycle cleanup for ephemeral profiles when their session disconnects and kept them out of Site Manager saved-site lists.
- Converted the titlebar Quick Connect button into a real split protocol control instead of only opening a modal.
- Added persisted Site Manager metadata: favorite, bookmark, tags, folder and last-used timestamp.
- Refreshed persisted metadata after successful saved-profile connections.
- Made transfer auto-retry count a live backend setting and re-applied all transfer-engine settings at startup.
- Preferences X/Cancel now follows cancel semantics and restores a captured settings snapshot.
- Replaced the cosmetic Shell Integration toggle with the existing native PATH integration implementation.
- Centralized RC3 release metadata for version/build/date/site in the React source.
- Removed inline JavaScript from localized web language selectors.

## Structural checks

A branding scan of runtime source found no Faro identifiers. `example.com` occurrences in the native tree are confined to comments/unit-test fixtures and are not seeded production profiles. Reference PNGs are stored as QA/design evidence only and are not loaded as application UI surfaces.

The complete native TypeScript/Rust build cannot be executed in this environment because the dependency/toolchain prerequisites are absent. RC3 therefore remains a source release candidate rather than a claimed native production build.

## RC4 verification — 20 September 2026

- `go test ./...` and `go vet ./...` pass for both compatibility runtime and installer.
- Runtime JavaScript parses successfully with `node --check`.
- Production-source scan found no `Faro`/`faro` identifiers and no `example.com`, demo user, Production Server, Staging Server, Design Assets, Cloud Server or Alex placeholder strings in the runtime/native frontend source surfaces.
- Native startup panic paths for corrupted profile JSON and SQLite initialization were reduced by recovery/quarantine logic.
