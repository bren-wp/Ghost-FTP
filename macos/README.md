# ByFTP for macOS

`macos/` contains the canonical macOS packaging surface for the shared ByFTP desktop engine. The Go runtime remains shared under `cmd/` and `internal/`; macOS-specific bundle metadata, launcher behavior and Universal PKG build logic live here so the platform package does not duplicate the desktop core.

## Release target

The production macOS artifact is:

```text
dist/ByFTP-<version>-macOS-Universal.pkg
```

The package contains a Universal Intel/Apple Silicon ByFTP runtime and `/Applications/ByFTP.app`. The repository build is intentionally not Developer ID signed until a valid Apple signing identity is supplied outside the repository.

## Build

From the repository root on macOS:

```bash
go telemetry off
bash macos/BUILD.sh
```

The root `VERSION` file is the single release version source. `macos/Info.plist.in` is rendered with that version, `macos/launcher.zsh` is copied into the application bundle and the canonical `build/icon.png` is converted into the app icon set during the build.

`scripts/BUILD-MACOS.sh` is retained only as a compatibility wrapper and delegates to `macos/BUILD.sh`.

## Shared desktop core

FTP, FTPS, SFTP, transfer, persistence and security code are intentionally not copied into `macos/`. macOS builds the same reviewed desktop engine as Windows and Linux, while keeping platform packaging and launcher details isolated here.
