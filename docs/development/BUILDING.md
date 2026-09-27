# Building Ghost FTP from Source

The production desktop application lives in `ghostftp-desktop/` and uses React, TypeScript, Tauri 2 and Rust. The Android application lives in `android/`. Tooling under `tools/` supports development/runtime/installer workflows but is not the authoritative production desktop GUI.

## Toolchain

CI currently uses:

- Node.js 22
- npm
- stable Rust with rustfmt and Clippy
- Go 1.23 for Go tooling checks
- Java 17 and Gradle 8.10.2 for Android
- Android platform/build-tools 35
- platform prerequisites required by Tauri 2

Use the committed lockfiles. Do not regenerate dependency graphs merely to make CI green unless a dependency change is intentional and reviewed.

## Version source of truth

Root `version.json` is authoritative. Run:

```bash
cd ghostftp-desktop
npm run version:check
```

The synchronization tooling checks package, Cargo, Tauri, Android, updater, website and release-note metadata.

## Desktop frontend checks

From `ghostftp-desktop/`:

```bash
npm ci
npm audit --audit-level=high
npm run check:i18n
npm run check:ui
npm run check
npm run build
```

## Rust checks

From `ghostftp-desktop/src-tauri/`:

```bash
cargo fmt --all -- --check
cargo metadata --locked --format-version 1 >/dev/null
cargo check --workspace --all-targets --locked
cargo test --workspace --exclude ghostftp-cli --locked
cargo test -p ghostftp-cli --no-default-features --lib --locked
cargo check -p ghostftp-cli --locked
cargo clippy --workspace --all-targets --locked -- -D warnings
```

The exact CI commands may be split between workflows, but `Cargo.lock` is committed and must remain usable with `--locked`.

## Go tooling checks

```bash
cd tools/ghostftp-runtime
go test ./...
go vet ./...

cd ../ghostftp-installer
mkdir -p payload
: > payload/GhostFTP.exe
go test ./...
go vet ./...
```

The empty installer payload above is a test fixture only; production installer/package builds supply real binaries through their packaging path.

## Native desktop packages

Windows:

```bash
cd ghostftp-desktop
npm run build:windows
```

Linux:

```bash
cd ghostftp-desktop
npm run build:linux
```

Windows production packaging uses NSIS. Linux packaging targets the native executable, AppImage, DEB and RPM formats configured by Tauri/workflows.

## Android

From the repository root:

```bash
bash android/scripts/check-android-contract.sh
gradle -p android lintDebug lintRelease lintPreview assembleDebug assembleRelease assemblePreview
```

CI verifies that the preview APK is signed/installable, that the release-check APK is unsigned, and that the installable preview uses the expected package id.

## Development workflow

1. Start from current `main`.
2. Keep `version.json` as the version source of truth.
3. Reuse existing Ghost FTP modules/components rather than creating parallel implementations.
4. Keep protocol behavior in native backends; UI must report real outcomes.
5. Add/update automated checks for changed behavior.
6. Run frontend, Rust and affected platform gates.
7. Verify responsive/focus/keyboard/dialog behavior for UI changes.
8. Update active documentation when behavior, workflow names or release state changes.
9. Open a PR and require the exact PR HEAD to pass the applicable release gates before merge.

See [../release/PROCESS.md](../release/PROCESS.md) and [../qa/README.md](../qa/README.md).
