# Ghost FTP 0.99.7 — cross-platform dead-code audit

**Starting baseline:** v0.99.6 passed seven exact-source CI workflows. Its published audit enumerated **723 working-tree files**: 117 TS/TSX, 87 Rust, 14 production Kotlin, 17 Kotlin tests, 54 Android resources, 16 Swift, 9 Swift tests, 12 Go and 68 scripts. It found **zero orphaned source modules, Android resource candidates, unreferenced operational scripts and zero entrypoint blockers**. This is not proof of absence of dead functions.

**New review depth:** `tools/audits/full-dead-code-audit.mjs` now outputs advisory private/nonpublic function candidates for Kotlin (Android), Swift (macOS), Rust (shared Tauri desktop on Windows/Linux/macOS), TypeScript/React and Go (Windows installer/runtime). The exact SHA CI artifact `GhostFTP-Full-Repository-Dead-Code-Audit` contains counts and paths. No files are removed solely because textual references are absent: Android callbacks, dynamic resource lookups, Swift selectors, Rust macros, Go build tags, JS imports and historic migrations can reference code implicitly.

**Cross-platform required gates:** Android Gradle unit/lint/R8/API 36 emulator; macOS Swift unit/native bundle; shared Rust cargo check/test/Clippy and cross OS packaging; TypeScript check:dead/typecheck/build; Go vet and tests; release scripts integrity checks.

**Concrete defect fixed:** Android's private local file browser previously called `listFiles()` and sorted all entries synchronously. Storage failures or large folders can block UI or throw. New bounded DirectoryStream worker, canonical path isolation and stale callback generation prevent that class of failure.

**Remaining work:** Independently inspect every symbol finding and test physical target devices before deletion. This static audit does not certify all 16 Android screenshots, user-data migration, large-transfer integrity or absence of all runtime crashes.

## Identified unused helper

`tools/ghostftp-runtime/main.go: decodeURLPath` appeared only at its declaration across inspected Go source and tests, with no route/selector references. It was removed together with its otherwise-unused `net/url` import. This remains subject to `go test`, `go vet`, and Windows/Linux target compilation. The extended Rust inventory initially marked 447 nonpublic symbols, mostly `#[cfg(test)]` functions. Such test-only entrypoints are explicitly excluded from candidate counts in the updated scan; no Rust application functionality was removed.
