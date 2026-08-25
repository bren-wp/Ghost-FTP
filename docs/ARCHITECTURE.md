# Architecture

ByFTP has three intentionally separate runtime implementations that share product, security and release policy rather than process memory: the Go desktop client, native Android client and native iOS client.

## Desktop core

Windows/Linux/macOS are implemented in Go. `cmd/byftp` starts the client, `cmd/installer` and `cmd/uninstaller` own the Windows lifecycle, `internal/api` exposes typed engine operations, `internal/desktop` contains presentation, `internal/remote` owns FTP/FTPS/SFTP sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings/profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

The shared desktop engine is intentionally stored only once. Linux-specific packaging lives under `linux/` and macOS-specific packaging lives under `macos/`; neither directory carries a copied fork of the Go protocol/transfer/security core. This keeps fixes to the desktop engine consistent across Windows, Linux and macOS.

The constrained Windows OpenSSH AskPass path remains a separate security boundary with trusted-parent/token validation, protected runtime secrets and environment cleanup.

## Linux packaging surface

`linux/` is the canonical Linux application-packaging directory:

- `linux/BUILD.sh` cross-builds the shared desktop runtime for amd64, arm64 and i386 and creates the DEB packages.
- `linux/byftp.desktop` is the desktop-entry source copied into every DEB.
- `linux/debian/control.in` is the version/architecture-templated package metadata source.
- `linux/README.md` documents the Linux build boundary.

`scripts/BUILD-LINUX.sh` remains only as a compatibility wrapper that delegates to `linux/BUILD.sh`.

## macOS packaging surface

`macos/` is the canonical macOS application-packaging directory:

- `macos/BUILD.sh` builds amd64/arm64 desktop binaries, creates the Universal binary/application bundle and builds the PKG.
- `macos/Info.plist.in` is the version-templated bundle metadata source.
- `macos/launcher.zsh` is the application launcher installed into the bundle.
- `macos/README.md` documents the macOS build/signing boundary.

`scripts/BUILD-MACOS.sh` remains only as a compatibility wrapper that delegates to `macos/BUILD.sh`.

## Android runtime

`android/` is a native Java/Android application. It does not embed the Go executable, start a localhost bridge or send credentials through a ByFTP service.

- `model/` contains validated connection configuration, remote entries and strict remote-path helpers.
- `remote/` contains FTP/FTPS and SFTP adapters behind the `RemoteClient` boundary.
- `MainActivity` owns Android UI/lifecycle orchestration and Storage Access Framework interaction.
- `res/` contains UI resources and network-security/backup/device-transfer policy.
- `android/app/src/test` contains unit/security/path/version regressions.

Commons Net provides FTP/FTPS primitives. SSHJ provides SFTP primitives. Android validates SFTP fingerprints as real 32-byte SHA-256 digests, rejects credential/control-character input and fails closed on traversal/noncanonical remote paths and names.

## iOS runtime

`ios/` is a native SwiftUI application with a conventional Xcode project and shared scheme. It does not embed a WebView or hidden desktop process.

- `ConnectionConfig.swift` owns validated host/port/credential input and the intentionally limited first-release protocol set.
- `RemoteModels.swift` owns canonical remote paths and shared-hosting login-root mapping.
- `SocketConnection.swift` wraps Network.framework with bounded async I/O.
- `FTPRemoteClient.swift` implements FTP and implicit FTPS, passive-mode safety, MLSD/LIST parsing and remote operations.
- `SessionStore.swift` owns generation-bound async state so stale operations cannot update a disconnected/newer session.
- `ConnectionView.swift`, `RemoteBrowserView.swift` and `ContentView.swift` keep presentation separate from transport/session logic.
- `ios/Tests/ModelTests.swift` supplies dependency-free input/path regressions.

The first iOS release intentionally supports FTP and implicit FTPS only. Explicit FTPS and SFTP require separately audited native implementations before they can be exposed.

Network.framework supplies TCP/TLS. FTPS uses platform TLS verification and `PROT P`. EPSV is preferred; PASV fallback accepts only the advertised port and keeps the data connection on the user-selected host. The authenticated FTP `PWD` becomes the UI account root. The transport password copy is cleared after connect, pending connections are disconnectable and the UI disconnects when the app enters the background.

## Build and packaging boundary

Root `VERSION` is shared by all platforms.

- Windows builds x64/x86 Setup, Portable and verified ZIP bundles.
- `linux/BUILD.sh` builds amd64/arm64/i386 DEBs.
- `macos/BUILD.sh` builds a Universal PKG.
- Android builds debug-signed and optimized unsigned APKs; `scripts/package_android.py` validates/stages them.
- iOS builds a generic arm64 iPhoneOS `.app`; `scripts/package_ios.py` validates the bundle and creates an unsigned IPA plus unsigned app ZIP.

Android production signing and Apple iOS signing remain external trust boundaries. The repository never fabricates a production publisher identity.

## Shared invariants

All runtimes use canonical versioning, contact only endpoints selected by the user, avoid application telemetry/advertising and participate in fail-closed CI/release gates. Platform-specific security controls remain native rather than being weakened to force a single shared implementation.
