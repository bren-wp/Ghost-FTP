# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused **FTP, FTPS and SFTP** client for Windows, Linux, macOS and Android. It is designed for shared hosting, website deployment and routine server file management while keeping credentials local and avoiding advertising, analytics SDKs and mandatory cloud services.

**Current release: 1.1.0**

[Download the latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Android source](android/README.md) · [Installation](docs/INSTALLATION.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## Highlights

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Shared-hosting friendly `public_html` navigation and login-relative FTP paths.
- Passive FTP/FTPS with compatibility handling for hosting servers.
- Remote browse, upload, download, create directory, rename and delete operations.
- Desktop transfer queue with pause, resume, cancel and retry.
- Fail-closed SFTP host-key verification and SHA-256 fingerprint pinning.
- FTPS certificate-chain and endpoint/hostname verification.
- Native Android application under `android/` — not a WebView wrapper.
- Android uploads/downloads through the Storage Access Framework without broad storage permissions.
- Session-only Android passwords; no Android credential database or plaintext secret persistence.
- Android app data excluded from cloud-backup and device-transfer extraction rules.
- No application telemetry, advertising SDK or mandatory ByFTP cloud account.

## Supported platforms

| Platform | Build / distribution |
| --- | --- |
| Windows x64 | Portable EXE, Setup EXE and verified release ZIP |
| Windows x86 | Portable EXE, Setup EXE and verified release ZIP |
| Linux amd64 | DEB package |
| Linux arm64 | DEB package |
| Linux i386 | DEB package |
| macOS | Universal PKG |
| Android 8.0+ (API 26+) | Native source in `android/`; CI-tested debug APK evidence |

Windows, Linux and macOS public packages are produced by the gated release workflow. Android source is a required release-quality gate starting with 1.1.0. A public Android production APK is intentionally not published until a stable private Android signing identity is configured outside the repository. The project never commits or fabricates a production signing key.

## Protocols

### FTP

FTP remains available for compatibility with servers that require it. FTP does **not** encrypt credentials or file content. Prefer FTPS or SFTP whenever the server supports them.

### FTPS

ByFTP supports explicit and implicit FTPS. On Android, the FTPS adapter explicitly selects the platform trust manager, enables endpoint/hostname checking and protects the FTP data channel with `PROT P`. ByFTP Android source is audited to reject custom trust-all TLS implementations.

### SFTP

SFTP uses SSH host-key verification. Desktop builds preserve the established host-key pinning and credential-hardening model. The Android client requires the expected OpenSSH-style `SHA256:` host-key fingerprint before connecting and uses SSHJ's native fingerprint verifier.

Android 1.1.0 supports SFTP password authentication. Private-key import is deliberately deferred until Android Keystore-backed handling, import validation and migration semantics are designed and audited.

## Shared-hosting workflow

A typical workflow is:

1. Enter the host, port and hosting-account username.
2. Prefer FTPS or SFTP where available.
3. Open the account web root, commonly `public_html`.
4. Upload or download files and perform supported remote management operations.
5. Keep paths inside the authenticated account namespace.

See [Shared hosting](docs/SHARED-HOSTING.md).

## Android 1.1.0

The Android application is isolated from the Go desktop runtime so mobile lifecycle, permissions and networking can be tested independently.

The initial native Android client provides:

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Remote directory listing and navigation.
- Upload/download using Android document providers.
- Create directory, rename and delete operations.
- Mandatory SFTP SHA-256 host-key pinning.
- FTPS platform certificate trust plus hostname verification.
- No broad storage permission.
- No password or SSH-secret persistence.
- No analytics, advertising SDK or ByFTP runtime backend.
- Explicit cloud-backup/device-transfer exclusions.
- Connection cleanup when the Activity is destroyed and protection against late UI callbacks.

See [ByFTP for Android](android/README.md).

## Security and safer transfers

The desktop transfer engine is deliberately conservative around destructive operations: uploads use staging, destinations are revalidated before commit, cleanup uncertainty is surfaced, local filesystem boundaries reject unsafe links/reparse points and stale session generations cannot mutate newer connections.

Android uses a smaller native protocol boundary appropriate to the mobile lifecycle. It validates host/port input, requires SFTP SHA-256 host-key pinning, explicitly uses platform TLS trust for FTPS, enables endpoint checking, uses passive/binary FTP mode and obtains local files through Android document providers instead of requesting unrestricted storage access.

Passwords must never be logged. Android passwords are held only for the active in-memory connection configuration and are not written to preferences, databases, files or a project backend.

Android generic cleartext traffic is disabled for platform-aware network stacks. Application data is excluded from Android backup/device-transfer extraction rules.

See [Security](docs/SECURITY.md).

## Privacy

ByFTP has no project-controlled runtime API, analytics SDK, advertising SDK or mandatory account service. Connections are initiated only to hosts selected by the user.

Desktop saved-state behavior remains platform-specific and documented. Android 1.1.0 does not persist connection passwords or SSH secrets, does not request broad storage access and excludes app data from cloud-backup/device-transfer flows.

See [Privacy](docs/PRIVACY.md).

## Languages

The established desktop runtime supports 18 languages with English as the canonical fallback: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.

The initial Android module is English-first. Additional Android resource translations should be added only as complete, reviewed resource sets so the mobile app does not ship partially translated screens.

## Installation

### Windows

Use Setup for a normal per-user installation or Portable when installation is not required. Windows builds are produced for x64 and x86. The installer validates its payload and preserves transactional rollback behavior during upgrades.

### Linux

Install the DEB matching the machine architecture: amd64, arm64 or i386.

### macOS

Use the Universal PKG from the official release.

### Android

The Android app is located entirely under `android/`. Version 1.1.0 requires Android 8.0 (API 26) or newer and targets API 37. CI builds a debug APK as verification evidence. A signed public production APK is not yet distributed because release signing must use a stable private key that is never committed to this repository.

See [Android source documentation](android/README.md) and [Installation](docs/INSTALLATION.md).

## Build from source

### Desktop core

ByFTP's desktop implementation uses Go. The canonical module path is:

```text
github.com/bren-wp/by-ftp
```

`VERSION` is the single production version source. Desktop binaries/packages and Android `versionName`/`versionCode` derive from it.

Windows:

```powershell
go telemetry off
.\BUILD-WINDOWS.ps1
```

Linux:

```bash
go telemetry off
bash scripts/BUILD-LINUX.sh
```

macOS:

```bash
go telemetry off
bash scripts/BUILD-MACOS.sh
```

### Android

The Android module uses Java 17, Android Gradle Plugin 9.3.0, Gradle 9.5.0, compile/target SDK 37, Apache Commons Net 3.13.0 and SSHJ 0.40.0.

From the repository root with JDK 17, Gradle 9.5.0 and Android SDK 37 installed:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon
```

Release builds enable code minification and resource shrinking.

## Tests and audits

Core gates include:

```bash
go test ./...
go test -race ./...
go vet ./...
python scripts/generate_brand_assets.py --check
python scripts/audit_localization.py
python scripts/audit_version.py
python scripts/audit_android.py
python scripts/audit_docs.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Android additionally runs JUnit, Android lint with warnings treated as errors, APK compilation and dedicated mobile security/privacy/version/lifecycle audits in a separate CI job. See [Testing](docs/TESTING.md).

## Release integrity

The release lane is fail-closed:

- `VERSION` consistency is checked before publication.
- Published semantic versions cannot be silently reused.
- Security, privacy, documentation and release-contract audits are mandatory.
- Windows, Linux, macOS **and Android source validation** must pass before publication.
- Android validation includes unit tests, lint and APK compilation.
- Android debug APKs are CI evidence and are not represented as production-signed packages.
- Public desktop release staging uses an exact allowlist and SHA-256 checksums.
- GitHub Release mutation remains centralized in `scripts/publish_release.ps1`.
- Android production distribution remains separate until a real private signing identity is configured.

See [GitHub releases](docs/GITHUB-RELEASES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Signing](docs/SIGNING.md).

## Repository structure

```text
android/          Native Android application, tests and mobile build configuration
cmd/
  byftp/          Desktop application entry point
  installer/      Windows installer
  uninstaller/    Windows uninstaller
internal/
  api/            Typed desktop application/engine boundary
  appdata/        Canonical and legacy user-data resolution
  config/         Profiles and settings persistence
  desktop/        Desktop UI
  i18n/           Desktop runtime localization catalogs
  localfs/        Local filesystem operations
  platform/       OS-specific primitives
  remote/         Desktop FTP/FTPS/SFTP implementations
  security/       Desktop path, secret and filesystem safety helpers
  transfer/       Desktop transfer queue and lifecycle
scripts/          Build, audit, verification and release tools
docs/             Project documentation
build/            Generated/static desktop build resources
```

## Documentation

- [Documentation index](docs/README.md)
- [Android](android/README.md)
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

Changes must preserve security, privacy and release invariants. Android networking code belongs under `android/`; do not route mobile secrets through the desktop process model or introduce hidden backend dependencies. New canonical user-facing text is English-first.

## Security reports

Do not publish passwords, private keys, production hostnames, customer data or other sensitive material in public issues. Follow the repository [security policy](.github/SECURITY.md).

## License

See [LICENSE](LICENSE). The license remains the authoritative legal attribution source.
