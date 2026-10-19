# Ghost FTP — full repository dead-code audit (19 October 2026)

**Baseline:** `main` at SHA `516a427a71c02b5416b4b3ca66b7157e9d2ec870`, released as [v0.95.46](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.95.46). This cross-platform work belongs to **one PR**, on `audit/full-repository-dead-code-v09547`, and preserves the Brendigo identity, user data and installed-app compatibility. A successful static scan alone does **not** establish that every unused code path has been found.

## Verified inventory and scope

The baseline Git tree contained **668 tracked files** (including source, assets, docs and release metadata). Identified production source areas include 66 Rust files under `ghostftp-desktop/src-tauri/src/`, 97 files in the primary desktop `src/`, 12 production Kotlin files, 12 Android JVM test files, 16 production Swift files and 9 Swift test files. The inventory also includes four Rust workspace crates, shared `packages/file-ui`, Android resources and localization, Go installer/runtime, shell/Python/Node scripts, update service and GitHub Actions.

## Executable checks

| Platform / surface | Authoritative test gates | Dead-code audit boundary |
| --- | --- | --- |
| Windows/Linux Rust and Tauri | `cargo fmt --all -- --check`, `cargo check/test --workspace --locked`, Clippy, Windows hardening, native Windows/Linux bundles | Production module graph, crate cfg features, IPC registration, OS FFI, database migrations, credential backends and updater assets must remain reachable and compatible |
| React and TypeScript | `npm run check:dead`, `npm run check`, `npm run check:ui`, `npm run build` | Production import graph plus strict compiler analysis; dynamic imports, generated entrypoints and CSS references require separate examination |
| Android Kotlin, resources and native app | `android/scripts/check-android-contract.sh`, Gradle unit tests, lint and R8-enabled shrinker, emulator instrumentation | Manifest activities, JNI/reflection, runtime-loaded resources, OS callbacks, locales and ProGuard keep rules prevent deleting code based on grep counts |
| macOS Swift and SwiftUI | `swift test`, release SwiftPM build, packaging contract, macOS native workflow | `@main`, protocol conformances, selectors, SwiftUI previews, registered commands and file-system observers have implicit entrypoints |
| Go installer and compatibility runtime | Platform-specific Go builds/tests and existing Go project checks | `main.go`, `_windows.go` vs `_other.go`, build tags, installers and embedded assets are platform entrypoints |
| Operational and release scripts | Workflow references, syntax checks, version/tag policy and release asset contracts | Scripts used only during manual recovery, signing, artifact packaging or historical migration must not be automatically deleted |

The new `tools/audits/full-dead-code-audit.mjs` traverses source files without following symlink directories, reports conservative Kotlin/Swift source-module, Android resource and scripts candidates, and blocks removal of known Android, Swift, Rust, React and Go application entrypoints. `tools/audits/full-dead-code-audit.test.mjs` includes synthetic positive/negative cases, and GitHub Actions publishes the exact-SHA JSON candidate inventory.

## Decisions

1. **No unproven files removed:** no production source file has been proved dead across all relevant OS targets, CI/release, migrations, configured features and dynamic call sites. Removing by a single search or import-graph result would be unsafe.
2. **Source-only candidates are advisory:** a module without a textual cross-file type reference may still contain top-level functions, extension methods, OS callbacks or Swift/Kotlin reflection. Android resources may be retrieved dynamically and be needed by the launcher, translations or in-app tests.
3. **Keep migration and security code:** SQLite forward-only migrations, profile JSON upgrades, Argon2id backup compatibility, Windows Credential Manager, Linux kernel-keyutils references pending verified Secret Service migration, SSH host keys, updater/signing and password redaction are not candidates for speculative removal.
4. **Do not replace runtime acceptance with CI:** real cloud logins, massive transfers, >4 GiB SHA-256 verified tests, resource lifetime profiling and actual target-device UI parity remain separate QA requirements.

## Deletion acceptance checklist

Before deleting a candidate, demonstrate its declaration and every production/test/package/runtime reference, prove unreachability on Windows **and** Linux build configurations or Android/macOS equivalents, verify installed-user migrations and feature flags, remove only the proven path, add focused regression coverage, and require every PR gate green on the *final* SHA. Any unresolved candidate stays documented; “no obvious reference” is not “dead”.

## Remaining limitations

The inventory cannot prove whole-program semantic reachability, dynamic reflection, OS callback use, symbol-level dead code in every Swift/Kotlin/Go package, false-positive-free resource cleanup, real system keyring acceptance, power-loss recovery or manual release tooling behavior. The CI-produced JSON is the review queue, not evidence for automatic deletion. No performance, storage or binary-size savings are claimed without reproducible measurements.
