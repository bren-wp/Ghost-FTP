# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused **FTP, FTPS and SFTP** client for Windows, Linux, macOS and Android. It is designed for shared hosting, website deployment and routine server file management while keeping credentials local and avoiding advertising, analytics SDKs and mandatory cloud services.

**Current release: 1.1.1**

[Download the latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Android](android/README.md) · [Installation](docs/INSTALLATION.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## Highlights

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Shared-hosting friendly `public_html` navigation and login/account-relative FTP paths.
- Passive/binary FTP/FTPS with compatibility handling for hosting servers.
- Remote browse, upload, download, create directory, rename and delete operations.
- Desktop transfer queue with pause, resume, cancel and retry.
- Fail-closed SFTP host-key verification and SHA-256 fingerprint pinning.
- FTPS certificate-chain and endpoint/hostname verification.
- Native Android application — not a WebView wrapper.
- Android local file access through the Storage Access Framework without broad storage permissions.
- Session-only Android passwords and explicit cloud-backup/device-transfer exclusions.
- Verified Windows, Linux, macOS and Android release artifacts from one canonical `VERSION`.
- No application telemetry, advertising SDK or mandatory ByFTP cloud account.

## Supported platforms and release artifacts

| Platform | Official release artifacts |
| --- | --- |
| Windows x64 | Portable EXE, Setup EXE, verified ZIP |
| Windows x86 | Portable EXE, Setup EXE, verified ZIP |
| Linux amd64 | DEB |
| Linux arm64 | DEB |
| Linux i386 | DEB |
| macOS | Universal PKG |
| Android 8.0+ / API 26+ | Debug-signed APK and optimized unsigned release APK |

The Android artifacts deliberately distinguish build validity from production signing:

- `ByFTP-<version>-Android-debug.apk` is an **installable debug-signed development/test build**.
- `ByFTP-<version>-Android-release-unsigned.apk` is the **minified and resource-shrunk release build without a production signature**.

A production Android distribution must sign the release APK with a stable private identity managed outside this repository. ByFTP does not commit or fabricate a production signing key.

## Protocol behavior

### FTP

FTP is retained for compatibility with servers that require it and does **not** encrypt credentials or file contents. Prefer FTPS or SFTP whenever the server supports them.

Desktop and Android clients are designed around shared-hosting account namespaces. Android 1.1.1 records the server working directory after login and maps the UI `/` to that authenticated account root. If the server cannot report `PWD`, Android falls back to login-relative paths instead of forcing an unrelated filesystem `/`. Traversal and noncanonical path components are rejected before FTP operations are issued.

### FTPS

Explicit and implicit FTPS are supported. Android explicitly uses the platform trust manager, enables endpoint/hostname verification and protects the FTP data channel with `PROT P`. Project audits reject permissive/custom trust-manager patterns in ByFTP Android source.

### SFTP

SFTP requires host-key verification. The Android client requires the expected OpenSSH-style `SHA256:` host-key fingerprint before connecting and uses SSHJ's native fingerprint verifier.

Android currently supports SFTP password authentication. Private-key import remains deferred until Android Keystore-backed handling, import validation and migration semantics are implemented and audited.

## Android 1.1.1

The Android implementation is isolated from the Go desktop runtime so lifecycle, permissions, networking and packaging can be tested independently. It provides:

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Remote listing/navigation, refresh, upload, download, create directory, rename and delete.
- Login-root-aware FTP/FTPS paths for shared hosting.
- Mandatory SFTP SHA-256 host-key pinning.
- FTPS platform trust and hostname verification.
- Storage Access Framework upload/download without broad storage permission.
- Session-only credentials; no password or SSH-secret persistence.
- Cloud-backup and device-transfer exclusions.
- Connection cleanup on Activity destruction and protection against late callbacks.
- Immediate cleanup of pending download-picker state after any picker result, disconnect or Activity destruction.
- Stable action dispatch that does not depend on translated UI labels.

See [ByFTP for Android](android/README.md).

## Windows, Linux and macOS

The desktop implementation remains written in Go. Windows keeps the transactional installer, x64/x86 Setup/Portable packages, DPAPI-backed saved secrets, live 18-language UI and hardened AskPass/process boundaries. In 1.1.1 the remaining startup and AskPass fallback messages were standardized on English so they match the repository's English-first contract while runtime-localized UI remains unchanged.

Linux packages continue to be built for amd64, arm64 and i386. macOS continues to ship as a Universal Intel/Apple Silicon PKG. All desktop platforms remain part of the same gated test/build matrix.

## Shared-hosting workflow

1. Enter the server host, port and hosting-account username.
2. Prefer FTPS or SFTP where available.
3. Connect and open the account web root, commonly `public_html`.
4. Upload/download files or use supported remote management operations.
5. Keep operations inside the authenticated account namespace.

See [Shared hosting](docs/SHARED-HOSTING.md).

## Security and privacy

ByFTP keeps transport, credential, transfer and filesystem checks fail-closed. The desktop transfer engine uses staging/revalidation around destructive writes and protects against unsafe local links/reparse points and stale session generations. Android validates endpoint input, enforces SFTP host-key pinning, uses platform TLS trust for FTPS and rejects unsafe FTP account-path mappings.

Android does not persist connection passwords or SSH secrets, request unrestricted storage access, include analytics/advertising SDKs or depend on a ByFTP runtime backend. App data is excluded from cloud-backup/device-transfer flows. Connections are initiated only to endpoints selected by the user.

See [Security](docs/SECURITY.md) and [Privacy](docs/PRIVACY.md).

## Languages

The desktop runtime supports 18 languages with English as the canonical fallback: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.

Android remains English-first. Additional Android translations should be added only as complete reviewed resource sets. Windows startup/AskPass fallback text is audited to remain English-first rather than silently reintroducing hard-coded Croatian strings.

## Installation

Use the package matching the operating system and architecture. Verify downloads against `SHA256.txt` before redistribution.

For Android testing, install the versioned debug APK. The unsigned release APK is not a production-installable package until it has been signed with an external trusted identity. See [Installation](docs/INSTALLATION.md).

## Build from source

The canonical desktop module is:

```text
github.com/bren-wp/by-ftp
```

`VERSION` is the single production version source for desktop binaries/packages, Android `versionName`/`versionCode`, release notes and GitHub Release artifact names.

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

The Android module uses Java 17, Android Gradle Plugin 9.3.0, Gradle 9.5.0, compile/target SDK 37, Apache Commons Net 3.13.0 and SSHJ 0.40.0.

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

Android separately runs JUnit, `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and APK structural validation. Windows x64/x86, Linux amd64/arm64/i386 and macOS Universal builds remain required gates. See [Testing](docs/TESTING.md).

## Release integrity

The release lane is intentionally fail-closed:

- `VERSION` consistency and published-version immutability are checked before publication.
- Security, privacy, localization, documentation and release-contract audits are mandatory.
- Windows, Linux, macOS and Android builds must all succeed.
- Android debug and unsigned release APKs are structurally validated and staged with versioned names.
- Public staging uses an exact platform-asset allowlist.
- `SHA256.txt` covers every public package plus release metadata.
- GitHub Release mutation remains centralized in `scripts/publish_release.ps1`, which re-reads the release and validates final asset size/digest identity.
- Android production signing remains external; unsigned/debug artifacts are never labeled as production store-signed software.

See [GitHub releases](docs/GITHUB-RELEASES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Signing](docs/SIGNING.md).

## Repository structure

```text
android/          Native Android application, tests and mobile build configuration
cmd/              Desktop app, installer and uninstaller entry points
internal/         Desktop engine, UI, protocols, security, persistence and transfers
scripts/          Build, audit, packaging, verification and release tools
docs/             Project documentation
build/            Generated/static desktop build resources
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
- [Build and verification tools](scripts/README.md)

## Contributing and security reports

Changes must preserve security, privacy and release invariants. Android networking belongs under `android/`; mobile secrets must not be routed through hidden backend services or persisted without an explicitly reviewed security design. New canonical user-facing source text is English-first.

Do not publish passwords, private keys, production hostnames, customer data or other sensitive material in public issues. Follow the repository [security policy](.github/SECURITY.md).

## License

See [LICENSE](LICENSE). The license remains the authoritative legal attribution source.
