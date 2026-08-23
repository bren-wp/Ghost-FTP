# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused desktop file-transfer client for **FTP, FTPS and SFTP** on Windows, Linux and macOS. It is built for shared hosting, website deployment and routine server file management while keeping credentials local and avoiding advertising, analytics SDKs and mandatory cloud services.

**Current release: 1.0.12**

[Download the latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Installation](docs/INSTALLATION.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## Why ByFTP

ByFTP focuses on the workflows that matter when managing websites and shared-hosting accounts:

- FTP, explicit FTPS, implicit FTPS and SFTP connections.
- Shared-hosting friendly `public_html` navigation and login-relative FTP paths.
- Passive FTP with MLSD → LIST compatibility fallback for older hosting servers.
- Local and remote file browsing, folder creation, rename, delete and CHMOD operations.
- Upload and download queues with pause, resume, cancel and retry.
- Safe overwrite staging with rollback/backup handling rather than direct destructive replacement.
- Protection against unsafe local paths, symlinks, junctions and reparse points.
- SFTP host-key verification and fingerprint pinning.
- No application telemetry, advertising SDKs or mandatory remote account system.

## Supported platforms

| Platform | Production package |
| --- | --- |
| Windows x64 | Portable EXE, Setup EXE and release ZIP |
| Windows x86 | Portable EXE, Setup EXE and release ZIP |
| Linux amd64 | DEB package |
| Linux arm64 | DEB package |
| Linux i386 | DEB package |
| macOS | Universal PKG |

Exact release assets are produced by the repository release workflow and verified before publication. See [GitHub releases](docs/GITHUB-RELEASES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Supported protocols

### FTP

Standard FTP support is intended primarily for compatibility with hosting environments that still require it. Credentials and content are not encrypted by the FTP protocol itself, so FTPS or SFTP should be preferred whenever the server supports them.

### FTPS

ByFTP supports both explicit and implicit FTPS. TLS is used for the FTP connection while retaining familiar shared-hosting FTP semantics.

### SFTP

SFTP support uses SSH host-key verification and fingerprint pinning. Private-key based authentication is supported, and secret handling is designed so passwords and passphrases do not need to be persisted in plaintext or exposed through generic command-line arguments.

## Shared-hosting workflow

ByFTP is designed to behave predictably on typical shared hosting:

1. Connect using the host, port and credentials supplied by the hosting provider.
2. Open the account's web root, commonly `public_html`.
3. Upload website files or folders through the transfer queue.
4. Use rename, delete, folder creation and CHMOD operations where the server allows them.
5. Keep FTP paths relative to the authenticated account namespace instead of accidentally switching to the server root.

For hosting-specific behavior and compatibility notes, see [Shared hosting](docs/SHARED-HOSTING.md).

## Safer transfers

The transfer layer is deliberately conservative around destructive operations.

- Uploads use temporary remote staging before the final commit/rename.
- Existing targets are revalidated before overwrite commit.
- Skip-existing decisions are rechecked against fresh remote state.
- Downloads are staged locally and validated before activation.
- Local symbolic links, junctions and Windows reparse points are rejected at security-sensitive boundaries.
- Cleanup failures that can leave uncertain remote state are surfaced instead of being silently treated as success.
- Connection/session generations prevent stale work from mutating a newer session.

The detailed threat model and invariants are documented in [Security](docs/SECURITY.md).

## Privacy

ByFTP is designed to work without a project-controlled runtime backend. The application does not require analytics, advertising or a mandatory ByFTP cloud account.

Network traffic is limited to the servers and protocol helpers required for the connection or transfer initiated by the user. Go telemetry is explicitly disabled in production build and CI workflows.

See [Privacy](docs/PRIVACY.md) for the complete policy and technical boundaries.

## Languages

English is the canonical source and fallback language. Runtime localization is available for:

- English
- Croatian
- German
- French
- Spanish
- Turkish
- Greek
- Portuguese
- Simplified Chinese
- Russian
- Hindi
- Japanese
- Italian
- Polish
- Dutch
- Czech
- Ukrainian
- Swedish

The Windows setup wizard asks for a language before installation and stores that choice as the initial application language. Language can later be changed from Settings.

## Installation and upgrades

### Windows

Use the Setup EXE for a normal per-user installation or the Portable EXE when an installed copy is not required. Windows builds are produced for x64 and x86.

The installer validates its embedded payload before modifying files or registry state and uses transactional rollback for failed upgrades. Existing ByFTP user data is preserved across the data-directory normalization introduced in 1.0.12.

### Linux

Install the DEB package matching the machine architecture. Production packages are built for amd64, arm64 and i386.

### macOS

Use the Universal PKG from the official release. The package contains a Universal application build for supported Intel/Apple Silicon environments.

Detailed instructions are in [Installation](docs/INSTALLATION.md).

## Build from source

ByFTP uses Go and intentionally keeps the production module graph minimal. The canonical module path is:

```text
github.com/bren-wp/by-ftp
```

The repository `VERSION` file is the **single production version source**. Runtime binaries, installers, platform packages, release notes and GitHub Package metadata must derive the release number from it.

### Windows production build

```powershell
go telemetry off
.\BUILD-WINDOWS.ps1
```

The Windows production build performs asset, localization, version, documentation, security, privacy and release audits before compiling x64 and x86 artifacts.

### Linux production build

```bash
go telemetry off
bash scripts/BUILD-LINUX.sh
```

### macOS production build

```bash
go telemetry off
bash scripts/BUILD-MACOS.sh
```

### Tests and audits

```bash
go test ./...
go test -race ./...
go vet ./...
python scripts/generate_brand_assets.py --check
python scripts/audit_localization.py
python scripts/audit_version.py
python scripts/audit_docs.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

See [Testing](docs/TESTING.md) for the complete verification matrix.

## Release integrity

The release pipeline is fail-closed and designed to prevent accidental publication of incomplete or mismatched artifacts.

- Production builds run with Go telemetry disabled.
- `VERSION` is checked for consistency before release.
- Already-published version lines are protected by the release-version guard.
- Windows, Linux and macOS platform jobs must succeed before publication.
- Windows ZIP bundles use an explicit file allowlist and `BUNDLE-SHA256.txt`.
- Public release staging must match the expected artifact set exactly.
- GitHub Release mutation is centralized in `scripts/publish_release.ps1`.
- SHA-256 manifests are generated for published assets.

See [Release verification](docs/RELEASE-VERIFICATION.md) and [Signing](docs/SIGNING.md).

## Repository structure

```text
cmd/
  byftp/          Desktop application entry point
  installer/      Windows installer
  uninstaller/    Windows uninstaller
internal/
  api/            Typed application/engine boundary
  appdata/        Canonical and legacy user-data resolution
  config/         Profiles and settings persistence
  desktop/        Desktop UI
  i18n/           Runtime localization catalogs
  localfs/        Local filesystem operations
  platform/       OS-specific primitives
  remote/         FTP/FTPS/SFTP implementations
  security/       Path, secret and filesystem safety helpers
  transfer/       Transfer queue and lifecycle
scripts/          Build, audit, verification and release tools
docs/             Project documentation
build/            Generated/static build resources
```

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
- [Build and verification tools](scripts/README.md)

## Contributing

Changes should preserve the project's security, privacy and release invariants. New canonical user-facing text is English-first and runtime UI strings belong in the localization system rather than being hard-coded into platform logic.

Before proposing a change, read [Contributing](docs/CONTRIBUTING.md) and run the relevant tests/audits listed above.

## Security reports

Do not publish passwords, private keys, production hostnames, customer data or other sensitive material in public issues. Follow the repository [security policy](.github/SECURITY.md) for security-sensitive reports.

## License

See [LICENSE](LICENSE). The license file is intentionally preserved as the authoritative legal attribution source.
