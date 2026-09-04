# Ghost FTP

**Ghost FTP** is a privacy-focused FTP, FTPS and SFTP client for Windows, Linux, macOS, Android, iOS and the web. The project is designed for dependable day-to-day file transfer, shared-hosting workflows and environments where explicit security controls and reproducible releases matter.

Current Ghost FTP version: **1.0.0**

## Core capabilities

- FTP, explicit/implicit FTPS and SFTP workflows.
- Local and remote file browsing, upload, download, rename, delete and directory operations.
- Transfer state tracking with defensive staging, cleanup and rollback behavior.
- SFTP host-key fingerprint validation and strict connection-input validation.
- Protected profile secrets and platform-specific credential protection where supported.
- Windows desktop client and installer, Linux packages, universal macOS package, Android client, iOS client and shared-hosting web/PWA client.
- No application telemetry in production builds.

## Security and stability

Ghost FTP treats remote paths, local paths, temporary files and credentials as security boundaries. The repository includes regression coverage for traversal protection, SFTP fingerprint policy, transfer cleanup, staging/rollback behavior, configuration durability, runtime-secret handling, process lifecycle behavior and other failure modes.

The web application additionally uses CSRF protection, strict session handling, secure cookies, security headers, rate limiting and `noindex` directives. Production CI runs Go formatting checks, unit tests, the race detector, `go vet`, security/privacy audits, PHP syntax validation and native platform build checks.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Testing](docs/TESTING.md) for details.

## Releases

Ghost FTP releases use the tag namespace `ghostftp-vX.Y.Z`. The first release in the Ghost FTP product line is `ghostftp-v1.0.0`; subsequent patch versions are `ghostftp-v1.0.1`, `ghostftp-v1.0.2`, and so on.

Public release assets are intentionally simple:

| Platform | Public package |
| --- | --- |
| Windows x64 | `Ghost-FTP-X.Y.Z-Setup-x64.exe` |
| Windows x86 / 32-bit | `Ghost-FTP-X.Y.Z-Setup-x86.exe` |
| Windows x32 alias | `Ghost-FTP-X.Y.Z-Setup-x32.exe` |
| Linux | `Ghost-FTP-X.Y.Z-Linux-multiarch.zip` |
| macOS | `Ghost-FTP-X.Y.Z-macOS-Universal.pkg` |
| Android | `Ghost-FTP-X.Y.Z-Android.apk` |
| iOS | `Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa` |
| Web | `Ghost-FTP-X.Y.Z-Web.zip` |

`x32` and `x86` refer to the same 32-bit Windows architecture in this project. The x32 file is therefore a byte-identical compatibility alias of the x86 installer, not a third CPU architecture.

Every release also contains `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`. Verify the checksum before installing a downloaded package.

Android CI produces an installable debug-signed APK unless a private production signing identity is configured externally. iOS release artifacts are unsigned and require a valid Apple signing identity and provisioning profile before normal device/TestFlight/App Store distribution. The workflow never labels unsigned or debug-signed artifacts as store-signed production binaries.

See [GitHub Releases](docs/GITHUB-RELEASES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Signing](docs/SIGNING.md).

## Versioning policy

The **Ghost FTP** product line starts at `1.0.0`. Version changes use Semantic Versioning:

- patch fixes: `1.0.0` → `1.0.1` → `1.0.2`
- backward-compatible feature releases: `1.0.x` → `1.1.0`
- breaking changes: next major version

Historical ByFTP commits and tags remain in Git history for provenance. They are not rewritten. In particular, the historical `v1.0.0` tag is intentionally separate from the Ghost FTP tag `ghostftp-v1.0.0`.

## Compatibility identifiers

The public product name is **Ghost FTP**. Some internal source paths, package identifiers, migration keys, namespaces or build-stage filenames may still contain the legacy `ByFTP`/`byftp` identifier where changing it would unnecessarily break existing installations, saved profiles, application identities or migration/cleanup logic. These identifiers are compatibility implementation details and must not be used as new public branding.

New public documentation, UI, release titles and downloadable package names use **Ghost FTP**.

## Development

The Go core requires a modern Go toolchain. CI currently builds with Go 1.27.1 and production scripts enforce the repository's minimum supported toolchain.

```bash
go telemetry off
go test ./...
go test -race ./...
go vet ./...
```

Linux packages:

```bash
bash linux/BUILD.sh
```

macOS package:

```bash
bash macos/BUILD.sh
```

Windows packages are built with `BUILD-WINDOWS.ps1`. Android uses the Gradle project in `android/`, iOS uses the Xcode project in `ios/`, and the web client is packaged by `scripts/package_web.py`.

## Repository structure

- `cmd/` — Go application and installer entry points.
- `internal/` — transfer, remote protocol, configuration, security, desktop and platform logic.
- `android/` — Android application.
- `ios/` — iOS application.
- `linux/` — Debian package build.
- `macos/` — universal macOS package build.
- `ByFTP WEB/` — legacy-named source directory for the Ghost FTP web/PWA client; directory name is retained for compatibility during the rebrand.
- `scripts/` — build, audit, packaging and verification tooling.
- `docs/` — architecture, security, release and operator documentation.

## Support

Issues and feature requests: https://github.com/bren-wp/Ghost-FTP/issues

Repository: https://github.com/bren-wp/Ghost-FTP

## License

See [LICENSE](LICENSE).
