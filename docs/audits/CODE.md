# Ghost FTP Code Audit — 0.30.16

## Scope

This audit covers the current Ghost FTP 0.30.16 source line: Windows/Linux desktop, Android, CLI, Agent/agentd/protocol crates, Go compatibility tools, updater/release tooling and shared file UI. Previous canonical release: **0.30.15**.

These findings build on the previously reviewed 0.30.15 code-audit baseline; the 0.30.16 changes are limited to live premium color surfaces and their CI contract. A complete new dead-code audit has not been claimed.

## Dead-code and reachability evidence

- TypeScript keeps `noUnusedLocals` plus production entrypoint/import-graph reachability.
- Rust keeps `cargo fmt`, locked metadata, all-target checks, tests and Clippy with warnings denied.
- 0.20.9 adds a Rust crate module-graph guard so orphan `.rs` files that are never compiled cannot silently remain in `src/`.
- CI/update/Android MJS and shell entrypoints must have a real workflow, package, runbook or helper reference.
- Operational MJS and shell files receive syntax checks in Quality.
- Android remains Gradle-source-set compiled and gated by lint/build, the Android production contract, emulator instrumentation and declaration-only private Kotlin symbol audit.
- macOS now has a declaration-only private Swift symbol audit in addition to Swift unit tests and release build verification.
- Cross-platform shell parity is release-blocking: Windows/Linux, Android and macOS must retain New connection plus Files, Sites, Transfers, Sync & Backup, Settings and Help & About.
- Go runtime/installer compatibility tools remain covered by `go test` and `go vet`.
- CSS remains exercised through the production frontend build and UI contract checks rather than being deleted by selector-name heuristics.

No file is deleted merely because it looks old. Removal requires compiler/linter, module/import graph, workflow/build reference or test evidence proving it is outside runtime/build/test/release/migration/compatibility paths.

## 0.20.9 functional changes

- Transfer history export is a real end-to-end path: Transfer Center → Zustand state → Tauri IPC → Rust persisted transfer ledger.
- Exported CSV omits raw backend error strings and neutralizes spreadsheet-formula prefixes in user/server-controlled cells.
- Stable package publication remains valid without updater-signing secrets; in that mode the in-app updater bundle is deliberately omitted.
- When signed updater artifacts are present, the workflow and publication script require the signatures, exact-version manifest and Update-Service package as an all-or-nothing verified set before tag mutation.

## Remaining refactor work

Large hotspots such as `bridge.rs`, `transfer.rs`, CLI `main.rs`, `commands.rs`, `session/mod.rs`, `i18n.ts`, `styles.css`, Android `MainActivity.kt`, `AgentBridge.tsx` and shared `FilePane.tsx` remain candidates for incremental, test-first decomposition. They are not considered dead solely because of size.

Status: current source has stronger cross-language reachability and action wiring guards; exact-SHA CI remains authoritative for acceptance.
