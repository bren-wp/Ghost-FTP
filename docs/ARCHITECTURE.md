# Architecture

ByFTP has four intentionally separate runtime implementations that share product, security and release policy rather than process memory: the Go desktop client, native Android client, native iOS client and shared-hosting WEB application.

## Desktop core

Windows/Linux/macOS are implemented in Go. `cmd/byftp` starts the client, `cmd/installer` owns the Windows installation lifecycle, `internal/api` exposes typed engine operations, `internal/desktop` contains presentation, `internal/remote` owns FTP/FTPS/SFTP sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings/profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

ByFTP 1.8.0 no longer builds or embeds a dedicated Windows `Uninstall.exe`. The Setup payload is application-only and contains `ByFTP.exe` plus its integrity manifest. During a successful upgrade Setup may clean legacy uninstall files/registry entries from older installations, but no new uninstaller binary is generated, packaged or registered.

The shared desktop engine is intentionally stored only once. Linux-specific packaging lives under `linux/` and macOS-specific packaging lives under `macos/`; neither directory carries a copied fork of the Go protocol/transfer/security core. This keeps fixes to the desktop engine consistent across Windows, Linux and macOS.

The constrained Windows OpenSSH AskPass path remains a separate security boundary with trusted-parent/token validation, protected runtime secrets and environment cleanup.

## Linux packaging surface

`linux/` is the canonical Linux application-packaging directory:

- `linux/BUILD.sh` cross-builds the shared desktop runtime for amd64, arm64 and i386 and creates the DEB packages.
- `linux/byftp.desktop` is the desktop-entry source copied into every DEB.
- `linux/debian/control.in` is the version/architecture-templated package metadata source.
- `linux/README.md` documents the Linux build boundary.

CI and production releases invoke `linux/BUILD.sh` directly. No Linux production-build wrapper remains under `scripts/`.

## macOS packaging surface

`macos/` is the canonical macOS application-packaging directory:

- `macos/BUILD.sh` builds amd64/arm64 desktop binaries, creates the Universal binary/application bundle and builds the PKG.
- `macos/Info.plist.in` is the version-templated bundle metadata source.
- `macos/launcher.zsh` is the application launcher installed into the bundle.
- `macos/README.md` documents the macOS build/signing boundary.

CI and production releases invoke `macos/BUILD.sh` directly. No macOS production-build wrapper remains under `scripts/`.

## Android runtime

`android/` is a native Java/Android application. It does not embed the Go executable, start a localhost bridge or send credentials through a ByFTP service.

- `model/` contains validated connection configuration, remote entries and strict remote-path helpers.
- `remote/` contains FTP/FTPS and SFTP adapters behind the `RemoteClient` boundary.
- `MainActivity` owns Android UI/lifecycle orchestration and Storage Access Framework interaction.
- `res/` contains UI resources and network-security/backup/device-transfer policy.
- `android/app/src/test` contains unit/security/path/version regressions.

Commons Net provides FTP/FTPS primitives. SSHJ provides SFTP primitives. Android validates SFTP fingerprints as real 32-byte SHA-256 digests, rejects credential/control-character input and fails closed on traversal/noncanonical remote paths and names.

The 1.8.0 Android build baseline uses JDK 17, Gradle 9.7.1, Android Gradle Plugin 9.3.2, API 37 and Build Tools 36.0.0. The application version is derived from root `VERSION` so Android cannot silently diverge from the rest of the release.

## iOS runtime

`ios/` is a native SwiftUI application with a conventional Xcode project and shared scheme. It does not embed a WebView or hidden desktop process.

- `ConnectionConfig.swift` owns validated host/port/credential input and the intentionally limited protocol set.
- `RemoteModels.swift` owns canonical remote paths and shared-hosting login-root mapping.
- `SocketConnection.swift` wraps Network.framework with bounded async I/O.
- `FTPRemoteClient.swift` implements FTP and implicit FTPS, passive-mode safety, MLSD/LIST parsing and remote operations.
- `SessionStore.swift` owns generation-bound async state so stale operations cannot update a disconnected/newer session.
- `ConnectionView.swift`, `RemoteBrowserView.swift` and `ContentView.swift` keep presentation separate from transport/session logic.
- `ios/Tests/ModelTests.swift` supplies dependency-free input/path regressions.
- `ios/BUILD.sh` is the canonical iPhoneOS build entry point used by CI and production releases.

The current iOS runtime intentionally supports FTP and implicit FTPS only. Explicit FTPS and SFTP require separately audited native implementations before they can be exposed.

Network.framework supplies TCP/TLS. FTPS uses platform TLS verification and `PROT P`. EPSV is preferred; PASV fallback accepts only the advertised port and keeps the data connection on the user-selected host. The authenticated FTP `PWD` becomes the UI account root. The transport password copy is cleared after connect, pending connections are disconnectable and the UI disconnects when the app enters the background.

The iOS build injects `MARKETING_VERSION` from root `VERSION`, validates the resulting bundle version and packages a real unsigned arm64 iPhoneOS app into the release IPA/app ZIP contract.

## WEB runtime

`ByFTP WEB/` is a PHP/shared-hosting implementation with its own request/session runtime but the same product/security release policy.

- `app/Remote` implements FTP/SFTP transport boundaries and strict remote-path handling.
- `app/Security` owns authentication, rate limiting, host validation, encryption and security logging.
- `app/Storage` owns users, preferences, encrypted profiles and workspace durability.
- `tests/` contains PHP runtime regressions for configuration, authentication/rate limiting, user-registry and encrypted-profile recovery behavior.
- `scripts/audit_web.py` executes and statically verifies these fail-closed invariants in CI.

`ByFTP WEB/VERSION`, Composer metadata and the PWA cache namespace are bound to the same 1.8.0 release as the native applications.

## Build and packaging boundary

Root `VERSION` is the canonical application version source for Windows, Linux, macOS, Android and iOS; `ByFTP WEB/VERSION` must match it. The 1.8.0 reviewed desktop baseline uses Go 1.27.0, while Android uses Gradle 9.7.1 with AGP 9.3.2/JDK 17.

- Windows builds x64/x86 Setup and Portable binaries through root `BUILD-WINDOWS.ps1`; no uninstaller binary is produced.
- `linux/BUILD.sh` builds amd64/arm64/i386 DEBs.
- `macos/BUILD.sh` builds a Universal PKG.
- Android builds debug-signed and optimized unsigned APKs; `scripts/package_android.py` validates/stages them.
- `ios/BUILD.sh` builds a generic arm64 iPhoneOS `.app`; `scripts/package_ios.py` validates the bundle and creates an unsigned IPA plus unsigned app ZIP.
- WEB is validated by the central release-quality job rather than emitted as an operating-system binary.

`scripts/` contains shared audits, packaging, verification and release tooling rather than platform-specific production build wrappers. The obsolete historical source-sync workflow has also been removed from `.github/workflows/`.

Android production signing, Windows Authenticode, macOS Developer ID/notarization and Apple iOS signing remain external trust boundaries. The repository never fabricates a production publisher identity.

## Shared invariants

All runtimes use canonical versioning, contact only endpoints selected by the user, avoid application telemetry/advertising and participate in fail-closed CI/release gates. Platform-specific security controls remain native rather than being weakened to force a single shared implementation.
