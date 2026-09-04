# Ghost FTP

**Ghost FTP** is a privacy-focused FTP, FTPS and SFTP client for Windows, Linux, macOS, Android, iOS and the web. It is designed for dependable day-to-day file transfer, shared-hosting workflows, explicit security controls and reproducible releases.

Current Ghost FTP version: **1.0.1**

## Identity

The public product and UI name is **Ghost FTP**. The canonical technical identifier is **GhostFTP** where a space is not valid or practical, including executable names, package IDs, application identifiers and source-directory names.

The current codebase uses the Ghost FTP identity directly. New code must not introduce a retired product name, namespace, path, package ID or release filename. CI enforces this with a repository-wide fail-closed brand audit.

Canonical project locations:

- Website: https://ghostftp.com
- Repository: https://github.com/bren-wp/Ghost-FTP
- Issues: https://github.com/bren-wp/Ghost-FTP/issues

## Core capabilities

- FTP, explicit/implicit FTPS and SFTP workflows.
- Local and remote file browsing, upload, download, rename, delete and directory operations.
- Transfer-state tracking with defensive staging, cleanup and rollback behavior.
- SFTP host-key fingerprint validation and strict connection-input validation.
- Protected profile secrets and platform-specific credential protection where supported.
- Windows desktop client and installer, Linux packages, universal macOS package, Android client, iOS client and shared-hosting web/PWA client.
- No application telemetry in production builds.

## Security and stability

Ghost FTP treats remote paths, local paths, temporary files and credentials as security boundaries. Regression coverage includes traversal protection, SFTP fingerprint policy, transfer cleanup, staging and rollback, configuration durability, runtime-secret handling, process lifecycle behavior and connection-state failure modes.

The web application additionally uses CSRF protection, strict session handling, secure cookies, security headers, rate limiting and `noindex` directives. Production CI runs formatting checks, Go unit tests, the race detector, `go vet`, security/privacy/repository/release audits, PHP syntax checks and native platform builds.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Testing](docs/TESTING.md).

## Shared hosting

Ghost FTP supports common shared-hosting FTP/FTPS/SFTP layouts without silently changing the user's remote location. Initial directory diagnostics can recognize conventional web roots such as `public_html`, `httpdocs`, `htdocs`, `www`, `web` and `html`, but detected paths remain informational: Ghost FTP does not automatically navigate to or persist a derived web root.

Usernames such as `account@domain` are supported. FTP directory listings retain an MLSD-to-LIST fallback for hosts with older server configurations. Passive connection behavior and the security differences between plain FTP, FTPS and SFTP are documented in [Shared hosting](docs/SHARED-HOSTING.md).

## Releases

Ghost FTP releases use the tag namespace `ghostftp-vX.Y.Z`. The product line starts at `ghostftp-v1.0.0`; patch releases continue as `ghostftp-v1.0.1`, `ghostftp-v1.0.2`, and so on.

Version 1.0.1 uses the following public platform artifacts:

| Platform | Public package |
| --- | --- |
| Windows x64 installer | `Ghost-FTP-X.Y.Z-Setup-x64.exe` |
| Windows x86 installer | `Ghost-FTP-X.Y.Z-Setup-x86.exe` |
| Windows x32 alias | `Ghost-FTP-X.Y.Z-Setup-x32.exe` |
| Windows x64 portable | `Ghost-FTP-X.Y.Z-Portable-x64.exe` |
| Windows x86 portable | `Ghost-FTP-X.Y.Z-Portable-x86.exe` |
| Linux | `Ghost-FTP-X.Y.Z-Linux-multiarch.zip` |
| macOS | `Ghost-FTP-X.Y.Z-macOS-Universal.pkg` |
| Android | `Ghost-FTP-X.Y.Z-Android.apk` |
| iOS | `Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa` |
| Web | `Ghost-FTP-X.Y.Z-Web.zip` |

`x32` and `x86` refer to the same 32-bit Windows architecture in this project. The x32 setup file is a byte-identical alias of the x86 installer, not a third CPU architecture.

Every release also contains:

- `SHA256.txt` — checksums for release files.
- `RELEASE-NOTES.txt` — generated release notes.
- `BUILD-METADATA.txt` — build, commit, platform and signing metadata.

The release therefore contains **10 platform artifacts and 13 public files total**.

## GitHub Packages

Windows portable binaries are also published as the GitHub Packages NuGet package **`GhostFTP`**. Package versions follow the same canonical `VERSION` file as GitHub Releases. The release workflow performs a package-registry readback after publication so a release cannot silently claim package publication without verifying the version is visible.

The retired package identity is not used by the current source or release pipeline.

## Signing status

The release workflow never fabricates publisher identities or signing status:

- Windows Authenticode signing requires a valid external code-signing identity.
- macOS Developer ID signing/notarization requires valid Apple credentials.
- Android CI currently produces an installable debug-signed APK unless production signing credentials are configured externally.
- iOS produces an unsigned arm64 IPA that requires external Apple signing/provisioning for normal device, TestFlight or App Store distribution.

Always verify `SHA256.txt` before installing a downloaded package. See [Signing](docs/SIGNING.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Versioning policy

Ghost FTP starts at **1.0.0** and follows Semantic Versioning:

- patch fixes: `1.0.0` → `1.0.1` → `1.0.2`
- backward-compatible feature releases: `1.0.x` → `1.1.0`
- breaking changes: next major version

Published historical Git tags remain immutable for repository provenance. Ghost FTP releases use the dedicated `ghostftp-vX.Y.Z` namespace so current release tags never collide with older generic `vX.Y.Z` tags.

## Platform identities

Current canonical technical identities include:

- Go module: `github.com/bren-wp/Ghost-FTP`
- Go application entry point: `cmd/ghostftp`
- Windows executable: `GhostFTP.exe`
- Android application ID / namespace: `com.ghostftp.client`
- iOS project/app: `GhostFTP`
- iOS bundle ID: `com.ghostftp.client`
- macOS bundle ID: `io.github.bren-wp.ghostftp`
- Linux package: `ghost-ftp`
- Web source: `GhostFTP WEB/`
- GitHub Packages ID: `GhostFTP`

## Languages

English is the default runtime language. Ghost FTP currently includes localization catalogs for English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.

Language selection is persisted in application settings. New canonical user-facing text is maintained English-first and translated through the localization system.

## Documentation

Core documentation:

- [Documentation index](docs/README.md)
- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Signing](docs/SIGNING.md)
- [Shared hosting](docs/SHARED-HOSTING.md)
- [Roadmap](docs/ROADMAP.md)
- [Third-party notices](docs/THIRD-PARTY-NOTICES.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Support](docs/SUPPORT.md)

Platform documentation:

- [Linux](linux/README.md)
- [macOS](macos/README.md)
- [Android](android/README.md)
- [iOS](ios/README.md)
- [Web/PWA](GhostFTP%20WEB/README.md)

## Development

CI builds with Go 1.27.1. The repository pins and audits its production toolchain.

```bash
go telemetry off
go test ./...
go test -race ./...
go vet ./...
```

Platform build entry points:

```bash
bash linux/BUILD.sh
bash macos/BUILD.sh
bash ios/BUILD.sh
```

Windows packages are built with `BUILD-WINDOWS.ps1`. Android uses the Gradle project under `android/`. The web client is packaged by `scripts/package_web.py`.

## Repository structure

- `cmd/` — Go application and installer entry points.
- `internal/` — transfer, protocol, configuration, security, desktop and platform logic.
- `android/` — Android application.
- `ios/` — iOS application.
- `linux/` — Debian package build.
- `macos/` — universal macOS package build.
- `GhostFTP WEB/` — Ghost FTP web/PWA client.
- `scripts/` — build, audit, packaging and verification tooling.
- `docs/` — architecture, security, release and operator documentation.

## Support

Issues and feature requests: https://github.com/bren-wp/Ghost-FTP/issues

Repository: https://github.com/bren-wp/Ghost-FTP

## License

See [LICENSE](LICENSE).
