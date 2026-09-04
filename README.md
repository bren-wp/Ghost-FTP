# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused file-transfer suite for **Windows, Linux, macOS, Android, iOS and the web**. Native desktop and Android clients support FTP, explicit FTPS, implicit FTPS and SFTP. The native iOS client supports FTP and implicit FTPS. **ByFTP WEB** is a PHP shared-hosting PWA with FTP/FTPS and optional SFTP when the hosting environment provides `ext-ssh2`.

**Current release: 1.9.1**

[Latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Installation](docs/INSTALLATION.md) · [ByFTP WEB](ByFTP%20WEB/README.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## 1.9.1 highlights

- Hardens private upload-source snapshot cleanup so failed filesystem removal remains retryable instead of losing the only cleanup target.
- Makes local download replacement report rollback-copy cleanup failures instead of silently leaving stale `.byftp-rollback-*` content after a committed replacement.
- Bounds ByFTP WEB JSON runtime state to 8 MiB for both reads and writes, using bounded stream reads before JSON decoding to reduce memory-exhaustion risk from abnormal state files.
- Fixes the WEB FTP raw LIST fallback so a regular filename containing ` -> ` is preserved; link-target normalization now applies only to actual Unix symlink entries.
- Adds regression coverage for snapshot cleanup retryability, local rollback cleanup, WEB JSON bounds and FTP LIST filename preservation.
- Strengthens release publication with an immediate complete remote asset/digest verification plus a delayed second GitHub readback after re-confirming the exact current `main` commit.
- Keeps canonical toolchains unchanged from 1.9.0: Go 1.27.1, Android Gradle Plugin 9.4.0, Gradle 9.7.1, JDK 17, Android API 37 and build-tools 36.0.0.

## 1.9.0 highlights

- Synchronizes Windows, Linux, macOS, Android, iOS and ByFTP WEB on canonical `VERSION` **1.9.0**.
- Updates the native desktop toolchain to **Go 1.27.1**, including the September 2026 compiler/runtime/library fixes.
- Updates Android to **Android Gradle Plugin 9.4.0** with **Gradle 9.7.1**, JDK 17, Android API 37 and build-tools 36.0.0.
- Publishes a verified deployable `ByFTP-1.9.0-WEB-shared-hosting.zip` built only from tracked production WEB files; runtime user/config/cache/backup data cannot enter the public package.
- Makes WEB ZIP extraction two-phase: every file entry is first materialized and validated locally, the real cumulative decompressed-byte limit is enforced, and only then may remote directories/uploads be mutated. A corrupt late ZIP entry therefore cannot cause earlier archive entries to be written first.
- Restricts WEB runtime diagnostics to administrators instead of every authenticated user.
- Removes the empty `internal/i18n/action_locale_de_fr.go` compilation unit and adds regression coverage so the dead file cannot silently return.
- Keeps the standalone Windows `Uninstall.exe` removed. Windows Setup uses the app-only schema-2 payload, while Setup/Portable x64/x86 all inherit the same canonical 1.9.0 version metadata.
- Refactors release packaging so Windows bundle creation and public release staging/SHA generation are centralized in dedicated scripts instead of duplicated workflow blocks.
- Expands the public release contract to **15 platform artifacts plus 3 shared metadata files = 18 public files**, including the WEB shared-hosting ZIP.
- Preserves the full release-quality matrix: repository/WEB/mobile/security/privacy/docs audits, Python regressions, Go tests/race/vet, Windows x64/x86, Linux amd64/arm64/i386, macOS Universal, Android APKs, iOS arm64 artifacts and WEB package verification.

## 1.8.0 highlights

- Synchronized Windows, Linux, macOS, Android, iOS and ByFTP WEB on the single canonical `VERSION` value `1.8.0`.
- Removed the standalone Windows `Uninstall.exe` binary from source, build output and installer payload. Windows Setup embeds only the verified `ByFTP.exe` payload plus its integrity manifest; upgrades best-effort clean the legacy uninstaller from older installations after the new application commit succeeds.
- Upgraded the Android build chain to Android Gradle Plugin 9.3.2 and Gradle 9.7.1 with Android API 37/build-tools 36.0.0 and Go 1.27.0 for the native desktop build.
- Hardened ByFTP WEB security state, authentication concurrency, rate limits, SFTP host-key pinning, encrypted profile binding and user/config recovery.

## Platform matrix

| Surface | Transport support | Release/build form |
| --- | --- | --- |
| Windows x64/x86 | FTP, explicit FTPS, implicit FTPS, SFTP | Portable EXE, app-only Setup EXE, verified ZIP; no standalone uninstaller |
| Linux amd64/arm64/i386 | FTP, explicit FTPS, implicit FTPS, SFTP | DEB |
| macOS Universal | FTP, explicit FTPS, implicit FTPS, SFTP | Universal PKG |
| Android 8.0+ | FTP, explicit FTPS, implicit FTPS, SFTP | Debug-signed APK, unsigned optimized release APK |
| iOS 16+ | FTP, implicit FTPS | Unsigned arm64 IPA and `.app` ZIP |
| ByFTP WEB | FTP/FTPS; SFTP with PHP `ext-ssh2` | Verified shared-hosting PHP/PWA ZIP plus maintained source tree |

Android production signing and Apple production signing remain external trust boundaries. Debug/unsigned artifacts are never described as store-signed production packages.

## Security model

ByFTP treats paths, endpoint identity, credentials and release metadata as security boundaries rather than UI conveniences.

Remote path/name input is validated before protocol commands are sent. Saved profiles and direct-connect flows reject noncanonical endpoint input rather than silently trimming control characters or edge whitespace. SFTP host-key verification remains mandatory where the client exposes SFTP. Android and desktop use canonical SHA-256 host-key fingerprints; ByFTP WEB requires a pinned SHA-256 SFTP fingerprint before client creation; iOS does not claim SFTP until an audited native implementation exists.

Plain FTP remains available for compatibility but does not encrypt credentials or content. Prefer SFTP or FTPS where supported. Android FTPS uses platform trust and endpoint checking; iOS implicit FTPS uses Apple Network.framework. PHP `ext-ftp` does not expose the same peer-verification controls as SFTP, so ByFTP WEB recommends fingerprint-pinned SFTP when a cryptographically verified server identity is required.

Saved mobile connection presets intentionally exclude passwords/passphrases. Windows keeps saved secrets behind its existing platform credential boundary. ByFTP WEB encrypts saved remote credential material with an installation-specific 256-bit key and isolates profile/preference data per ByFTP user.

No official client includes an advertising SDK or requires a ByFTP telemetry backend. Connections target endpoints selected by the user, subject to platform safety policies.

## Shared hosting

Native clients map the visible FTP root to the authenticated account namespace and expose non-secret shared-hosting diagnostics derived from the first listing already needed for the session. Common web roots are recognized in this deterministic order: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Detection is advisory only: ByFTP never silently changes or saves the user's selected path because a common web-root name was found.

ByFTP WEB can run directly on ordinary PHP shared hosting. It needs PHP 8.1+, a writable `storage/` directory, `ext-ftp` for FTP/FTPS, Sodium or OpenSSL for encrypted credential storage, optional `ext-ssh2` for SFTP and optional `ext-zip` for ZIP operations. Production deployments should use HTTPS. Release users can deploy the versioned `ByFTP-1.9.1-WEB-shared-hosting.zip`; its packaging process includes tracked production files only and excludes runtime data. See [Shared hosting](docs/SHARED-HOSTING.md) and [ByFTP WEB documentation](ByFTP%20WEB/README.md).

## Mobile behavior

Android and iOS include local filtering/search, deterministic directory-first sorting, direct path navigation and multi-file upload. Transfer progress is byte-based. Batch upload can stop safely after the currently active file rather than tearing down an FTP transaction mid-command.

Android uses the Storage Access Framework and does not request broad storage access. Application backup/device transfer is disabled for private app data. iOS uses security-scoped document access and clears active sessions when entering the background.

## Desktop behavior

The desktop core is written in Go and shared by Windows, Linux and macOS. Windows has the native graphical shell and the verified x64/x86 build pipeline. The 1.9.1 Windows Setup installs only `ByFTP.exe`; it does not install or register a standalone `Uninstall.exe`. Linux/macOS retain the shared transport/security core and canonical platform packaging under `linux/` and `macos/`.

The terminal client preserves raw host, account and filesystem identity input until central validation. Valid Unix paths are not altered by UI preprocessing.

## Repository and release integrity

Root `VERSION` is the single production version source. `ByFTP WEB/VERSION` must match it exactly. CI enforces:

- repository-wide tracked-file audit and current-version consistency;
- localization/documentation/security/privacy/release audits;
- Go formatting, unit/integration tests, race detector and `go vet` using Go 1.27.1;
- Windows x64/x86 production builds with app-only Setup payload and explicit no-uninstaller invariant;
- Linux amd64/arm64/i386 DEB builds;
- macOS Universal PKG build;
- Android JUnit, lintDebug, lintRelease, debug APK and unsigned release APK using AGP 9.4.0 / Gradle 9.7.1;
- real arm64 iPhoneOS build plus unsigned IPA/app ZIP validation;
- ByFTP WEB PHP/JavaScript/runtime tests plus deterministic tracked-source release ZIP packaging;
- exact public staging allowlist of 15 platform artifacts and three shared metadata files;
- exact-current-`main` verification immediately before GitHub Release mutation;
- immediate and delayed remote release asset/digest readback after publication.

Only the active public development history from **1.3.0 onward** remains in the maintained changelog/documentation surface. Older Git history is not rewritten.

## Languages

English is the canonical source and fallback language for the desktop runtime and repository documentation. The desktop application currently supports 18 runtime languages: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.

Android, iOS and ByFTP WEB remain English-first at the canonical source/documentation level while platform-specific UI localization is expanded only through reviewed, complete locale sets.

## Build from source

Use the canonical platform entry points:

```text
Windows: BUILD-WINDOWS.ps1
Linux:   linux/BUILD.sh
macOS:   macos/BUILD.sh
Android: Gradle project under android/ (AGP 9.4.0 / Gradle 9.7.1)
iOS:     ios/BUILD.sh
Web:     ByFTP WEB/ (PHP 8.1+) or python scripts/package_web.py
```

The current native build toolchain is Go 1.27.1. Android uses JDK 17, API 37 and build-tools 36.0.0.

For ByFTP WEB verification and packaging:

```bash
find 'ByFTP WEB' -name '*.php' -print0 | xargs -0 -n1 php -l
node --check 'ByFTP WEB/assets/js/api.js'
node --check 'ByFTP WEB/assets/js/app.js'
node --check 'ByFTP WEB/assets/js/pwa.js'
node --check 'ByFTP WEB/assets/js/settings.js'
node --check 'ByFTP WEB/assets/js/utils.js'
node --check 'ByFTP WEB/service-worker.js'
php 'ByFTP WEB/tests/unit.php'
python scripts/audit_web.py
python scripts/package_web.py --output-dir dist
```

## Documentation

Core documentation:
[Architecture](docs/ARCHITECTURE.md) · [Contributing](docs/CONTRIBUTING.md) · [GitHub releases](docs/GITHUB-RELEASES.md) · [Installation](docs/INSTALLATION.md) · [Privacy](docs/PRIVACY.md) · [Release verification](docs/RELEASE-VERIFICATION.md) · [Roadmap](docs/ROADMAP.md) · [Security](docs/SECURITY.md) · [Shared hosting](docs/SHARED-HOSTING.md) · [Signing](docs/SIGNING.md) · [Support](docs/SUPPORT.md) · [Testing](docs/TESTING.md) · [Third-party notices](docs/THIRD-PARTY-NOTICES.md).

Platform/source guides:
[Linux](linux/README.md) · [macOS](macos/README.md) · [Android](android/README.md) · [iOS](ios/README.md) · [ByFTP WEB](ByFTP%20WEB/README.md) · [Build and verification tooling](scripts/README.md) · [Documentation index](docs/README.md).
