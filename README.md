# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused FTP, FTPS and SFTP file-transfer client for Windows, Linux and macOS. It is designed for shared hosting, website deployment and day-to-day server file management without advertising, analytics SDKs or a mandatory cloud account.

**Current release: 1.0.12**

## Highlights

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Shared-hosting friendly `public_html` workflow, passive FTP and MLSD → LIST fallback.
- Bounded transfer queue with pause, resume, cancel and retry.
- Safe overwrite staging, backup/rollback and path/symlink protections.
- English is the canonical and fallback language.
- Runtime localization for English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.
- The Windows setup wizard asks for language before installation and persists that choice as the initial application language.
- No application telemetry, ads or fixed project runtime API.

## Build and install

Production builds read the version from the repository `VERSION` file. Do not maintain a second product version in scripts, documentation or package metadata.

See [Installation](docs/INSTALLATION.md), [Testing](docs/TESTING.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Documentation

- [Documentation index](docs/README.md)
- [Installation](docs/INSTALLATION.md)
- [Shared hosting](docs/SHARED-HOSTING.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Support](docs/SUPPORT.md)
- [Roadmap](docs/ROADMAP.md)
- [Contributing](docs/CONTRIBUTING.md)
- [GitHub releases](docs/GITHUB-RELEASES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Signing](docs/SIGNING.md)
- [Third-party notices](docs/THIRD-PARTY-NOTICES.md)

## License

See [LICENSE](LICENSE). The license file is intentionally preserved as the legal attribution source.
