# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused file-transfer client for **Windows, Linux, macOS, Android and iOS**. Desktop and Android support FTP, explicit FTPS, implicit FTPS and SFTP. The native iOS client supports FTP and implicit FTPS while preserving the same shared-hosting and fail-closed path principles.

**Current release: 1.2.1**

[Download the latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Android](android/README.md) · [iOS](ios/README.md) · [Installation](docs/INSTALLATION.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## Highlights

- Native desktop, Android and iOS applications; no mobile WebView wrapper.
- Shared-hosting friendly `public_html` navigation and authenticated-account FTP roots.
- Remote browse, upload, download, create directory, rename and delete.
- Fail-closed SFTP host-key verification where SFTP is available.
- FTPS platform trust and endpoint/hostname verification.
- Strict remote-path validation instead of silently rewriting traversal or unsafe separators.
- Raw mobile endpoint and credential fields are validated before trimming so CR/LF/NUL control characters cannot be normalized away before protocol checks.
- Session-scoped mobile credentials with no advertising, analytics SDK or mandatory ByFTP cloud account.
- One canonical `VERSION` drives Windows, Linux, macOS, Android, iOS and public release metadata.

## Supported platforms and release artifacts

| Platform | Official release artifacts |
| --- | --- |
| Windows x64 | Portable EXE, Setup EXE, verified ZIP |
| Windows x86 | Portable EXE, Setup EXE, verified ZIP |
| Linux amd64 | DEB |
| Linux arm64 | DEB |
| Linux i386 | DEB |
| macOS | Universal PKG |
| Android 8.0+ / API 26+ | Debug-signed APK, optimized unsigned release APK |
| iOS 16+ / arm64 | Unsigned IPA, unsigned `.app` ZIP |

Android and iOS release files deliberately separate reproducible build evidence from production signing. `ByFTP-<version>-Android-debug.apk` is installable for development/testing, while the Android release APK requires an external production signing identity. The iOS IPA and app ZIP contain the real arm64 iPhoneOS application but remain unsigned until a valid Apple identity and provisioning configuration is applied outside the repository.

## Protocol behavior

### FTP

FTP is retained for compatibility and does **not** encrypt credentials or file contents. Prefer a secure protocol whenever the server supports one. Desktop, Android and iOS map FTP UI `/` to the authenticated account namespace instead of assuming an unrelated server filesystem root. Traversal, NULs and noncanonical path components are rejected before remote operations.

### FTPS

Desktop and Android support explicit and implicit FTPS. iOS supports implicit FTPS through Apple Network.framework. Android uses platform trust plus endpoint checking; iOS uses the platform TLS stack. Protected FTPS data channels use `PROT P`.

### SFTP

Desktop and Android support SFTP with host-key verification. Android validates the provided OpenSSH-style `SHA256:` fingerprint as a real 32-byte SHA-256 digest before SSHJ receives it. iOS does not claim SFTP yet; adding it requires an audited native implementation rather than a permissive compatibility layer.

## Android

The Android application under `android/` supports FTP, explicit/implicit FTPS and SFTP. It uses the Storage Access Framework instead of broad storage permissions, keeps passwords session-only, excludes app data from backup/device transfer, and cleans pending connection/file-picker state during lifecycle teardown.

Version 1.2.1 additionally validates raw host, port, username, password and SFTP fingerprint input before whitespace normalization. Remote paths continue to reject backslashes, duplicate separators, dot/traversal components and noncanonical names, and SFTP fingerprints must decode to exactly 32 SHA-256 bytes.

See [ByFTP for Android](android/README.md).

## iOS

The native SwiftUI application lives under `ios/` with a normal Xcode project and shared scheme. It provides FTP and implicit FTPS, remote browsing, upload/download, mkdir, rename/delete, MLSD-to-LIST fallback, shared-hosting login-root mapping and system document/share integration.

The iOS transport ignores server-supplied PASV host redirects and reconnects data channels only to the host selected by the user. Version 1.2.1 validates raw host/port/credential input before trimming, the transport password copy is cleared after login, UI password state is cleared after a connection attempt and the app disconnects when entering the background.

See [ByFTP for iOS](ios/README.md).

## Windows, Linux and macOS

The desktop application remains written in Go. Windows retains transactional x64/x86 installers, portable packages, DPAPI-backed saved secrets, localized UI and hardened AskPass/process boundaries. Linux builds DEBs for amd64, arm64 and i386. macOS ships a Universal Intel/Apple Silicon PKG. Existing desktop gates remain mandatory for 1.2.1.

## Shared-hosting workflow

1. Enter the server host, port and hosting-account username.
2. Prefer FTPS or SFTP where the selected platform/server supports it.
3. Connect and open the account web root, commonly `public_html`.
4. Upload/download files or use supported remote management operations.
5. Keep operations inside the authenticated account namespace.

See [Shared hosting](docs/SHARED-HOSTING.md).

## Security and privacy

ByFTP keeps transport, credential, transfer and filesystem checks fail-closed. Desktop transfer code protects staging/activation and session generations. Android validates raw endpoint input, fingerprints and canonical remote paths. iOS uses bounded Network.framework I/O, raw-input validation, account-root path mapping, PASV-host redirect blocking, session-only credentials and background disconnect.

No mobile client includes analytics/advertising SDKs or requires a ByFTP backend. Connections target endpoints selected by the user. See [Security](docs/SECURITY.md) and [Privacy](docs/PRIVACY.md).

## Languages

The desktop runtime supports 18 languages with English as the canonical fallback: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish. Android and iOS remain English-first until complete reviewed locale sets are added.

## Installation

Use the artifact matching the operating system and architecture and verify it against `SHA256.txt`. Android production signing and iOS Apple signing remain external to this public repository; unsigned files are never described as store-signed software. See [Installation](docs/INSTALLATION.md).

## Build from source

`VERSION` is the single production version source for desktop binaries/packages, Android version metadata, iOS marketing version/build packaging, release notes and GitHub Release artifact names.

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

Android:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

iOS on macOS/Xcode:

```bash
bash scripts/BUILD-IOS.sh
```

The iOS script runs dependency-free Swift model/path regressions, validates the Xcode project, builds a generic arm64 `iphoneos` Release application with code signing disabled, verifies the application bundle, and creates versioned unsigned IPA and app ZIP artifacts.

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
python scripts/audit_ios.py
python scripts/audit_docs.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Android separately runs JUnit, `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and APK validation. iOS separately builds real arm64 iPhoneOS output and validates/archives `ByFTP.app`. Windows x64/x86, Linux amd64/arm64/i386 and macOS Universal builds remain required. See [Testing](docs/TESTING.md).

## Release integrity

The release lane is fail-closed:

- `VERSION` consistency and published-version immutability are checked first.
- Security, privacy, localization, documentation, Android, iOS and release-contract audits are mandatory.
- Windows, Linux, macOS, Android and iOS build jobs must all succeed.
- Android APKs and iOS IPA/app ZIP are structurally validated before staging.
- Public staging uses an exact platform-asset allowlist.
- `SHA256.txt` covers every public package plus release metadata.
- GitHub Release mutation stays centralized in `scripts/publish_release.ps1`, which re-reads the release and validates final asset identity.
- Production mobile signing remains external; unsigned/debug files are never labeled as production store-signed software.

See [GitHub releases](docs/GITHUB-RELEASES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Signing](docs/SIGNING.md).

## Repository structure

```text
android/          Native Android application and tests
ios/              Native SwiftUI iOS application and Xcode project
cmd/              Desktop app, installer and uninstaller entry points
internal/         Desktop engine, UI, protocols, security, persistence and transfers
scripts/          Build, audit, packaging, verification and release tools
docs/             Project documentation
build/            Canonical/static build resources
```

## Documentation

- [Documentation index](docs/README.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](docs/CONTRIBUTING.md)
- [GitHub releases](docs/GITHUB-RELEASES.md)
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
- [Android source and build guide](android/README.md)
- [iOS source and build guide](ios/README.md)
- [Build and verification tools](scripts/README.md)

## Contributing and security reports

Changes must preserve security, privacy and release invariants. Mobile secrets must not be routed through hidden backend services or persisted without an explicitly reviewed design. New canonical user-facing source text is English-first. Do not publish passwords, private keys, signing credentials, provisioning profiles, production hostnames or customer data in public issues. Follow the repository [security policy](.github/SECURITY.md).

## License

See [LICENSE](LICENSE). The license remains the authoritative legal attribution source.
