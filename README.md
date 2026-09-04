# Ghost FTP

**Ghost FTP** is a privacy-focused FTP, FTPS and SFTP client for Windows, Linux, macOS, Android, iOS and the web. The project prioritizes predictable file-transfer behavior, explicit trust boundaries, reproducible releases and zero application telemetry.

Current Ghost FTP version: **1.0.2**

## Identity

The public product and UI name is **Ghost FTP**. The canonical technical identifier is **GhostFTP** where spaces are not valid or practical, including executable names, package IDs, application identifiers and source-directory names. New source must not introduce retired product names or package identities; CI enforces this repository-wide.

Canonical locations:

- Website: https://ghostftp.com
- Repository: https://github.com/bren-wp/Ghost-FTP
- Issues: https://github.com/bren-wp/Ghost-FTP/issues

## Capabilities

- FTP, explicit/implicit FTPS and SFTP connections.
- Local and remote browsing, upload, download, rename, delete and directory operations.
- Defensive transfer staging, cleanup and rollback behavior.
- SFTP host-key fingerprint verification and strict host/path/input validation.
- Protected profile credentials and platform-specific credential protection where supported.
- Windows desktop/setup and portable builds, Linux packages, universal macOS package, Android client, iOS client and shared-hosting web/PWA client.
- No production application telemetry.

## Security and stability

Remote paths, local paths, temporary files, credentials and release provenance are treated as security boundaries. Regression coverage includes traversal rejection, host-key policy, transfer cleanup, configuration durability, process lifecycle, runtime-secret handling and fail-closed recovery behavior.

Ghost FTP 1.0.2 additionally bounds the in-memory Unix runtime-secret store so abandoned secret handles cannot grow without limit. The web credential envelope parser now rejects unsupported formats, oversized inputs and truncated authenticated-encryption payloads before decryption. Web runtime tests also enforce that human-readable Composer metadata cannot silently retain an obsolete release number.

The web application uses CSRF protection, strict sessions, secure cookies, security headers, rate limiting and `noindex` directives. Production CI runs Go formatting, unit tests, the race detector, `go vet`, repository/version/documentation/release/security/privacy/web audits, PHP/JavaScript validation and native platform builds.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Testing](docs/TESTING.md).

## Releases

Ghost FTP releases use `ghostftp-vX.Y.Z`. The current product line starts at `ghostftp-v1.0.0` and advances sequentially through patch releases such as `ghostftp-v1.0.1` and `ghostftp-v1.0.2`.

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

`x32` and `x86` are the same 32-bit Windows architecture in this release contract; the x32 setup is a byte-identical alias of the x86 installer. Every release also contains `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **10 platform artifacts and 13 public files total**.

Windows portable binaries are additionally published as the GitHub Packages NuGet package **`GhostFTP`**. Package and release versions both derive from the repository `VERSION` file, and publication performs registry/release readback before a workflow can succeed.

## Signing status

Release automation never fabricates publisher identities:

- Windows Authenticode requires an external code-signing identity.
- macOS Developer ID signing/notarization requires Apple credentials.
- Android CI currently produces an installable debug-signed APK unless production signing is configured externally.
- iOS produces an unsigned arm64 IPA requiring external Apple signing/provisioning for normal distribution.

Always verify `SHA256.txt` before installation. See [Signing](docs/SIGNING.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Platform identities

- Go module: `github.com/bren-wp/Ghost-FTP`
- Go application: `cmd/ghostftp`
- Windows executable: `GhostFTP.exe`
- Android application ID: `com.ghostftp.client`
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

- [Documentation index](docs/README.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](docs/CONTRIBUTING.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [Installation](docs/INSTALLATION.md)
- [Privacy](docs/PRIVACY.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Roadmap](docs/ROADMAP.md)
- [Security](docs/SECURITY.md)
- [Shared hosting](docs/SHARED-HOSTING.md)
- [Signing](docs/SIGNING.md)
- [Support](docs/SUPPORT.md)
- [Testing](docs/TESTING.md)
- [Third-party notices](docs/THIRD-PARTY-NOTICES.md)

Platform documentation:

- [Linux](linux/README.md)
- [macOS](macos/README.md)
- [Android](android/README.md)
- [iOS](ios/README.md)
- [Web/PWA](GhostFTP%20WEB/README.md)

## Development

CI uses Go 1.27.1 and disables Go telemetry. The baseline local checks are:

```bash
go telemetry off
go test ./...
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_web.py
```

Canonical platform build entry points are `BUILD-WINDOWS.ps1`, `linux/BUILD.sh`, `macos/BUILD.sh`, `ios/BUILD.sh`, the Android Gradle project and `scripts/package_web.py`.

## License

See [LICENSE](LICENSE).
