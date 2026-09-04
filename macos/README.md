# ByFTP for macOS

`macos/` contains the canonical macOS packaging surface for the shared ByFTP desktop engine. The Go runtime remains shared under `cmd/` and `internal/`; macOS-specific bundle metadata, launcher behavior and Universal PKG build logic live here so the platform package does not duplicate the desktop core.

**Current release: 1.9.1**

## Release target

The production macOS artifact is:

```text
dist/ByFTP-<version>-macOS-Universal.pkg
```

The package contains a Universal Intel/Apple Silicon ByFTP runtime and `/Applications/ByFTP.app`. Release 1.9.1 reads the same root `VERSION` used by Windows, Linux, Android, iOS and ByFTP WEB. The repository build is intentionally not Developer ID signed until a valid Apple signing identity is supplied outside the repository.

## Build

From the repository root on macOS with the reviewed **Go 1.27.1** toolchain:

```bash
go telemetry off
bash macos/BUILD.sh
```

The root `VERSION` file is the single release version source. `macos/Info.plist.in` is rendered with that version, `macos/launcher.zsh` is copied into the application bundle and the canonical `build/icon.png` is converted into the app icon set during the build.

CI and the production release workflow invoke `macos/BUILD.sh` directly, execute shared Go tests/`go vet`, build the Universal package and verify its package structure. There is no duplicate macOS production-build wrapper under `scripts/`.

## Shared desktop core

FTP, FTPS, SFTP, transfer, persistence, shared-hosting diagnostics and security code are intentionally not copied into `macos/`. macOS builds the same reviewed desktop engine as Windows and Linux, while keeping platform packaging and launcher details isolated here. The 1.9.1 release uses Go 1.27.1 and the same canonical transfer-cleanup, security and release-integrity invariants as the other desktop targets.
