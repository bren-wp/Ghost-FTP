# Ghost FTP Code Audit — RC10

## Scope

RC10 focuses on source organization, build reliability, naming consistency, visual-parity rules and removal of avoidable duplication without changing stable protocol behavior unnecessarily.

## Changes made

- Renamed product-facing top-level source directories to GhostFTP names:
  - `ghostftp-desktop/`
  - `ghostftp-web/`
  - `ghostftp-runtime/`
  - `ghostftp-installer/`
- Updated build, audit and release workflows to the new paths.
- Kept the nested `src-tauri/` folder because it is a framework build convention and renaming it would add avoidable build risk.
- Standardized release artifact naming.
- Removed an unused Rust service constant.
- Preserved platform-gated imports/constants to avoid warnings-as-errors failures.
- Preserved `cargo fmt --check`, workspace check/tests and Clippy `-D warnings`.
- Preserved TypeScript typecheck/build and Go test/vet checks.
- Removed duplicated New Connection/File Properties width overrides from the final visual-polish CSS block.
- Made both reference dialog heights explicit at the canonical viewport.
- Browser-host compatibility binaries remain excluded from end-user desktop releases.

## Existing hardening retained

- ephemeral Quick Connect sessions;
- persisted Site Manager metadata;
- configurable retry/transfer settings;
- complete Preferences cancel/reset semantics;
- OS-protected credential storage where supported;
- profile JSON recovery;
- SQLite recovery/quarantine;
- protocol-aware unsupported errors rather than fabricated data;
- SHA-256 and recursive chmod paths where supported;
- signed update configuration.

## Quality policy

Warnings in production Rust code are treated as errors by CI. Demo/legacy branding is rejected by source audit. Release publication requires both the quality audit and native desktop build to succeed for the same main commit.

## Known work that remains

See `docs/PROJECT-STATUS.md` and `docs/roadmap/ROADMAP.md`. The remaining blockers are primarily target-OS visual/installer acceptance and real-server FTP/FTPS/SFTP end-to-end verification, not claimed completed functionality.
