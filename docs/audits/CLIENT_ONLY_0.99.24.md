# Ghost FTP 0.99.24 — client-only code/security audit scope and findings

**Status:** Review in progress; static inventory and CI tests are not equivalent to full manual production certification. This document records concrete code-level changes and remaining blockers without claiming an independent penetration test.

## Reachability and removed components

- Removed standalone `ghost-server/`, `ghost-demo/`, `ghost-cli/`, `ghostftp-desktop/src-tauri/ghostftp-cli/`, installer tests, service workflow and release package edges.
- Removed `path_integration.rs` and the client Settings shell-toggle IPC bridge rather than leaving an always-failing control.
- Removed sync-script generators referencing the deleted CLI and their visible Copy-script actions; built-in folder synchronization is preserved.
- Cargo workspace, Rust lock metadata, version propagation and release asset contracts were updated without changing the shared desktop FTP/FTPS/SFTP session engine.
- Live agent/pairing code is retained pending independent reachability and behavior verification; it must not advertise the removed CLI as an available dependency.

## Security test plan and verified source constraints

- Run `node --test tools/audits/full-security-code-audit.test.mjs` and `node tools/audits/full-security-code-audit.mjs` on exact HEAD. This detects selected TLS/SSH identity bypasses, secret logging, weak script flags and required publisher guards, but is not a penetration test.
- Run `node --test tools/audits/full-dead-code-audit.test.mjs` and `node tools/audits/full-dead-code-audit.mjs --json` plus TypeScript import-graph check, `cargo metadata --locked`, Rust tests, `clippy`, Kotlin lint, Swift XCTest, Go tests and supported-OS native build.
- Verify that retired names do not appear in active current package inventory, CI tasks or user-visible actions; preserve historical release notes for audit provenance.
- Test TLS certificate rejection and SFTP known-host mismatch, overwrite/rename/delete refusal, symlink traversal, backup/restore, authentication error sanitization and low-privilege file-write paths with real protocols, not mocks alone.

## Open production blockers

1. Native macOS SwiftUI FTPS and SFTP are still not accepted as implemented, according to the repository security audit and UI text. Cross-platform parity is not demonstrated.
2. Signed installer/update/notarization/Play production acceptance and data-preserving upgrade tests are not mandatory/verified. Keep release channel Preview until this changes.
3. Full 75-screen pixel parity, stress and soak tests, device-specific accessibility, multi-gigabyte transfer recovery and all cloud-provider credentials are not independently verified.
4. New client-only builds require a fresh exact-SHA all-gates CI pass and Windows/Linux installer smoke tests before any merge or public release.
