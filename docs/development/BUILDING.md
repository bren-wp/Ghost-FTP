# Building Ghost FTP from Source

The production desktop application is `ghostftp-desktop/` and uses React, TypeScript, Tauri and Rust. Tooling under `tools/` is for development support and is not the production desktop GUI.

## Toolchain

Use Node.js 22, npm, stable Rust with rustfmt/clippy, Go 1.23 for the Go tooling checks, and the platform prerequisites required by Tauri 2.

## Frontend checks

From `ghostftp-desktop/`:

```bash
npm ci
npm run check:i18n
npm run check
npm run build
```

## Rust checks

From `ghostftp-desktop/src-tauri/`:

```bash
cargo fmt --all -- --check
cargo check --workspace --all-targets
cargo test --workspace
cargo clippy --workspace --all-targets -- -D warnings
```

## Go tooling checks

```bash
cd tools/ghostftp-runtime
go test ./...
go vet ./...

cd ../ghostftp-installer
go test ./...
go vet ./...
```

## Native packages

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

Windows RC builds use NSIS. Linux builds target the native executable, AppImage, DEB and RPM formats defined by the Tauri configuration and workflow.

## Version consistency

Keep these release values aligned:

- `ghostftp-desktop/package.json`
- `ghostftp-desktop/package-lock.json`
- `ghostftp-desktop/src-tauri/Cargo.toml`
- `ghostftp-desktop/src-tauri/tauri.conf.json`

The release workflow validates these values before publication.

## Development workflow

1. Start from the current `main` source.
2. Reuse existing Ghost FTP modules and components rather than creating parallel implementations.
3. Keep protocol behavior in the native implementation; the React layer should report real backend outcomes.
4. Add or update automated checks for changed behavior.
5. Run the frontend and Rust gates above.
6. Verify responsive, focus, keyboard and dialog behavior for UI changes.
7. Update the relevant documentation and release status.
8. Use the native build workflow before merging release-candidate changes.

See [../release/PROCESS.md](../release/PROCESS.md) for release gates and [../qa/README.md](../qa/README.md) for acceptance evidence.
