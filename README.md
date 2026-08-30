# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused file-transfer client for **Windows, Linux, macOS, Android and iOS**. Desktop and Android support FTP, explicit FTPS, implicit FTPS and SFTP. The native iOS client supports FTP and implicit FTPS while preserving the same shared-hosting and fail-closed path principles.

**Current release: 1.6.0**

[Download the latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Linux](linux/README.md) · [macOS](macos/README.md) · [Android](android/README.md) · [iOS](ios/README.md) · [Installation](docs/INSTALLATION.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## Highlights

- Native desktop, Android and iOS applications; no mobile WebView wrapper.
- Repository-wide integrity validation now scans **every tracked Git file** for portable paths, case-insensitive collisions, accidental build/cache artifacts, symlinks, UTF-8/BOM issues, merge-conflict remnants, trailing whitespace and stale current-release markers.
- Shared-hosting friendly authenticated-account roots plus non-secret connection diagnostics derived from the first root listing the client already performs.
- Common hosting web roots are recognized in deterministic priority: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`.
- Diagnostics distinguish secure FTPS/SFTP from plain unencrypted FTP and report account-root/SFTP-home context without persisting or automatically opening the detected path.
- Remote browse, upload, download, create directory, rename and delete.
- Android and iOS mobile browsers include local filtering/search, deterministic directory-first sorting, direct **Go to path** navigation and multi-file upload.
- Android and iOS expose byte-level transfer progress for the active upload/download without changing the underlying FTP/FTPS/SFTP trust model.
- Multi-file uploads on both mobile platforms can be stopped safely **after the current file**; ByFTP deliberately avoids tearing down a socket in the middle of an FTP transaction just to implement a cancel button.
- Android uses a compact connected-state mobile layout with a touch-friendly Up / Refresh / Menu action surface and 48dp minimum controls.
- iOS uses native SwiftUI search, batch document import, a compact action menu and an iOS-16-compatible empty/filter state.
- Android can restore the last successful non-secret connection metadata from app-private preferences; password/passphrase values are structurally excluded and app backup/device transfer remains disabled.
- iOS can restore the last successful non-secret connection metadata from Keychain `WhenUnlockedThisDeviceOnly`; the persistent `ConnectionPreset` has no password field and is revalidated before use.
- Mobile password fields are cleared after connection attempts, and asynchronous persistence callbacks receive only pre-extracted non-secret metadata rather than credential-bearing connection configs.
- Fail-closed SFTP host-key verification where SFTP is available.
- FTPS platform trust and endpoint/hostname verification.
- Strict remote-path and remote-name validation instead of silently rewriting traversal, separators, edge whitespace or protocol control characters.
- Raw endpoint and credential fields are validated before unsafe normalization so CR/LF/NUL controls cannot disappear before protocol checks.
- Windows quick-connect and profile-save share one raw connection-input validator: host and username reach security checks verbatim, while port text is parsed strictly without trimming.
- Saved-profile persistence independently validates raw protocol, host, username and SFTP fingerprint input before storing it, including direct fingerprint updates.
- Saved remote, local and private-key paths are preserved verbatim by the persistence/API layer after validation instead of being silently trimmed into different filesystem identities.
- FTP/FTPS runtime configs strip SFTP-only private-key, passphrase and host-key fingerprint state before persistence in the active session or transfer identity.
- Desktop SFTP passphrases remain bound to the exact private-key identity, using Windows path semantics only on Windows and fail-closed exact path matching on other desktop platforms.
- Desktop profile matching, reconnect transfer identity and endpoint-scoped trust share one canonical DNS/IPv6 endpoint identity policy instead of duplicating host normalization.
- Desktop SFTP fingerprints are validated as canonical OpenSSH `SHA256:` values that decode to exactly one 32-byte SHA-256 digest for saved profiles, direct fingerprint updates and direct connections.
- Runtime connection and transfer settings share one validated fallback policy instead of duplicating timeout/retry/parallelism defaults.
- Session-scoped mobile credentials with no advertising, analytics SDK or mandatory ByFTP cloud account.
- One canonical `VERSION` drives Windows, Linux, macOS, Android, iOS and public release metadata.
- Linux packaging lives under `linux/`, macOS packaging under `macos/`, and the canonical iOS build entry point under `ios/`; platform build logic is not duplicated under `scripts/`.
- CI and production builds remain pinned to Go 1.27.0 and Gradle 9.7.0 for the 1.6.0 release line.

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

FTP is retained for compatibility and does **not** encrypt credentials or file contents. Prefer a secure protocol whenever the server supports one. Desktop, Android and iOS map FTP UI `/` to the authenticated account namespace instead of assuming an unrelated server filesystem root. Traversal, NULs and noncanonical path components are rejected before remote operations. Version 1.5.0 also labels plain FTP as unencrypted in shared-hosting diagnostics.

### FTPS

Desktop and Android support explicit and implicit FTPS. iOS supports implicit FTPS through Apple Network.framework. Android uses platform trust plus endpoint checking; iOS uses the platform TLS stack. Protected FTPS data channels use `PROT P`.

### SFTP

Desktop and Android support SFTP with host-key verification. Android validates the provided OpenSSH-style `SHA256:` fingerprint as a real 32-byte SHA-256 digest before SSHJ receives it. Desktop applies the same canonical digest requirement to saved profiles, direct fingerprint updates and direct-connect fingerprints, while saved key passphrases remain scoped to the selected account and private-key identity with platform-correct path comparison. iOS does not claim SFTP yet; adding it requires an audited native implementation rather than a permissive compatibility layer.

## Shared-hosting diagnostics

After a successful connection, ByFTP analyzes the root listing that was already required to establish the usable session. It does **not** scan ports, try alternate servers, fetch an external service or issue an extra diagnostic listing. The result can identify a common web-root directory and explain whether the current transport is encrypted.

A detected `public_html`/`httpdocs`/`htdocs`/`www`/`web`/`html` directory is advisory only. ByFTP does not automatically navigate to it and does not save it into the selected profile. Existing saved paths and user navigation remain authoritative. Diagnostics intentionally exclude credentials, private-key material, usernames, fingerprints, certificate material and server banners.

Common shared-hosting failures such as FTP data-channel errors, TLS/certificate failures, connection limits and quota errors continue through the safe localized user-error layer rather than exposing raw `curl`/server output.

See [Shared hosting](docs/SHARED-HOSTING.md).

## Android

The Android application under `android/` supports FTP, explicit/implicit FTPS and SFTP. It uses the Storage Access Framework instead of broad storage permissions, keeps passwords session-only, excludes app data from backup/device transfer, and cleans pending connection/file-picker state during lifecycle teardown.

Version 1.5.0 analyzes the existing first `list("/")` result and adds secure/plain transport plus detected web-root/account-home context to the connected summary. The diagnostic model has no network behavior and is not persisted. Version 1.4.0's transport-neutral byte progress and safe **Stop after current file** behavior remain intact.

Version 1.6.0 adds no weaker transport shortcut; Android source, resources and packaging are now also covered by the repository-wide tracked-file audit in addition to Android lint, JUnit and the dedicated mobile security audit.

See [ByFTP for Android](android/README.md).

## iOS

The native SwiftUI application lives under `ios/` with a normal Xcode project and shared scheme. It provides FTP and implicit FTPS, remote browsing, batch upload/download, mkdir, rename/delete, MLSD-to-LIST fallback, shared-hosting login-root mapping and system document/share integration.

Version 1.5.0 derives diagnostics from the existing initial listing and displays a native **Shared hosting** section. It explicitly states that a detected web root is not opened or saved automatically. The password-free Keychain preset, PASV-host redirect blocking, background disconnect, temporary-download cleanup, transfer progress and safe batch-stop behavior remain unchanged.

Version 1.6.0 extends release hygiene to every tracked iOS project, Swift, plist, scheme, documentation and packaging file without changing the reviewed protocol claims.

See [ByFTP for iOS](ios/README.md).

## Windows, Linux and macOS

The desktop application remains written in Go. The shared remote engine returns a non-secret `ConnectionDiagnostics` object from the same initial connection probe it already performed. Windows surfaces that information in status while preserving the existing profile/user-selected `remoteStart` path. Linux and macOS retain the shared Go protocol/security core and their existing package formats.

Windows retains transactional x64/x86 installers, portable packages, DPAPI-backed saved secrets, localized UI and hardened AskPass/process boundaries. Linux builds DEBs for amd64, arm64 and i386 from `linux/`. macOS builds a Universal Intel/Apple Silicon PKG from `macos/`.

Version 1.6.0 adds repository-wide source/package hygiene checks around all three desktop platforms, including case-insensitive path-collision detection so a source layout that is valid only on one filesystem cannot silently enter the release line.

## Shared-hosting workflow

1. Enter the server host, port and hosting-account username.
2. Prefer FTPS or SFTP where the selected platform/server supports it.
3. Connect and review the non-secret shared-hosting status.
4. Open the account web root, commonly `public_html`, when appropriate for that hosting account.
5. Upload/download files or use supported remote management operations while staying inside the authenticated account namespace.

## Security and privacy

ByFTP keeps transport, credential, transfer, diagnostics and filesystem checks fail-closed. Shared-hosting diagnostics are derived only from already loaded remote entries and cannot introduce a second connection/listing path. Security audits reject secret-bearing diagnostic fields and automatic path changes.

Android centralizes remote-name and file-list rules, validates fingerprints, shortens transport/UI password lifetime, persists only non-secret endpoint/trust metadata in app-private preferences excluded from backup, and measures transfer progress by wrapping local streams rather than weakening transport validation. iOS uses bounded Network.framework I/O, canonical names, account-root path mapping, PASV-host redirect blocking, pending-connect cleanup, temporary-download cleanup, background disconnect, a password-free Keychain preset with this-device-only accessibility and progress callbacks on the existing data sockets.

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
bash linux/BUILD.sh
```

macOS:

```bash
go telemetry off
bash macos/BUILD.sh
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
bash ios/BUILD.sh
```

The iOS script runs dependency-free Swift model/path/preset/diagnostic regressions, validates the Xcode project, builds a generic arm64 `iphoneos` Release application with code signing disabled, verifies the application bundle, and creates versioned unsigned IPA and app ZIP artifacts.

## Tests and audits

Core gates include:

```bash
go test ./...
go test -race ./...
go vet ./...
python scripts/generate_brand_assets.py --check
python scripts/audit_repository.py
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

`audit_repository.py` enumerates the checkout with `git ls-files` and validates every tracked path/file, not only files named by a feature-specific audit. Android separately runs JUnit including shared-hosting diagnostics, mobile filter/sort and transfer-stream byte-accounting regressions, plus `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and APK validation. iOS runs dependency-free model/path/preset/diagnostic regressions and a real arm64 iPhoneOS build. Windows includes diagnostic-status regressions alongside the x64/x86 production build. Linux amd64/arm64/i386 and macOS Universal builds remain required. See [Testing](docs/TESTING.md).

## Release integrity

The release lane is fail-closed:

- Every tracked repository path/file passes the repository-wide integrity audit before platform packaging.
- `VERSION` consistency and published-version immutability are checked before release publication.
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
ios/              Native SwiftUI iOS application, Xcode project, tests and build entry point
linux/            Linux build, desktop entry and DEB packaging metadata
macos/            macOS Universal build, app-bundle metadata and launcher
cmd/              Shared desktop app, installer and uninstaller entry points
internal/         Shared desktop engine, UI, protocols, security, persistence and transfers
scripts/          Shared audits, repository integrity, packaging, verification and release tools
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
- [Linux source and build guide](linux/README.md)
- [macOS source and build guide](macos/README.md)
- [Android source and build guide](android/README.md)
- [iOS source and build guide](ios/README.md)
- [Build and verification tools](scripts/README.md)

## Contributing and security reports

Changes must preserve security, privacy and release invariants. Mobile secrets must not be routed through hidden backend services or persisted without an explicitly reviewed design. New canonical user-facing text and documentation are English-first. Do not publish passwords, private keys, signing credentials, provisioning profiles, production hostnames or customer data in public issues. Follow the repository [security policy](.github/SECURITY.md).

## License

See [LICENSE](LICENSE). The license remains the authoritative legal attribution source.
