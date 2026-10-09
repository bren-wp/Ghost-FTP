# Ghost FTP Code Audit — 0.30.21

## Scope

This audit covers the current Ghost FTP 0.30.21 source line: Windows/Linux desktop, Android, CLI, Agent/agentd/protocol crates, Go compatibility tools, updater/release tooling and shared file UI. Previous canonical release: **0.30.21**.

These findings build on the previously reviewed 0.30.15 code-audit baseline; the 0.30.16 changes are limited to live premium color surfaces and their CI contract. A complete new dead-code audit has not been claimed.

0.30.17 incremental review: Windows/Linux Help/About link destinations and brand copy, Android HTTPS allowlist, macOS resource links and transport-capability text changed; unrelated transfer/network implementations were not rewritten. The earlier 0.30.15/0.30.16 audit history below remains historical evidence, not an assertion of new audit coverage.

0.30.18 incremental review: inspected React Help/About and removed obsolete decorative SVG helpers; inspected native Android and SwiftUI macOS About views against the supplied concept family. This does not replace an independent full-project dead-code audit or certify pixel parity.

0.30.19 incremental review: examined the Files toolbar picker event routing and both FilePane refresh cycles, preserving existing transfer conflict handling. Added source-level regression checks. This is not a complete new dead-code audit or visual 1:1 acceptance.

## 0.30.20 full-source reachability audit (all four apps)

The source audit spans **86 desktop TypeScript/TSX app files, 18 shared file-UI modules, 85 Rust sources, 5 Kotlin source/test files, 15 Swift source/test files and 11 Go files**, plus workflow/release scripts (file inventory from the 0.30.19 git tree). It distinguishes static reachability, symbol-level warnings, runtime behavior and visual acceptance rather than misclassifying framework callbacks, string references or build scripts as dead code.

- **Windows/Linux:** `npm run check:dead` walks the real production TypeScript import graph, Rust crate module graphs and operational script references; `tsc --noEmit` rejects unused locals; Rust `cargo check --all-targets`, tests, clippy, formatting and production bundle checks validate actual compiled code. The removed background work is a now-visibility-gated clock timer, not an unverified module deletion.
- **Android:** production resource/locale contract, Kotlin private-symbol audit, Gradle lint and test, build and device/emulator click smoke. Privacy redaction patterns are now compiled once and applied **before** the short diagnostic UI limit, tested with a secret crossing the old boundary and an oversized untrusted message.
- **macOS:** declaration-only private Swift-symbol audit, Swift tests and release compiler, validated backup merge count invariant and transfer-history basename-only storage. The system still explicitly blocks unsupported FTPS/SFTP until real identity validation exists.
- **Shared CLI/Go:** Go test/vet and Rust source-graph checks. CI/download scripts receive shell/Node syntax checks; the one-shot README screenshot importer workflow/script were removed after their successful verified use; only the seven proven image assets and release-image SHA256 regression check remain.
- **Preservation:** no dynamic-dispatched function, schema migration, compatibility entrypoint, test fixture, generated source or optional platform path is deleted just because a grep suggests zero references. False positives must be inspected before deletion.

**Audit status:** source-level dead-module and declaration scans are defined and automated; an assertion that every possible runtime interaction and all 75 screenshots were manually accepted is **not** made. The exact-HEAD CI results and measured installed-build captures are the release authority.

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
