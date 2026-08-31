# ByFTP

![ByFTP — Secure File Transfer](docs/images/byftp-header.png)

ByFTP is a privacy-focused file-transfer suite for **Windows, Linux, macOS, Android, iOS and the web**. Native desktop and Android clients support FTP, explicit FTPS, implicit FTPS and SFTP. The native iOS client supports FTP and implicit FTPS. **ByFTP WEB** is a PHP shared-hosting PWA with FTP/FTPS and optional SFTP when the hosting environment provides `ext-ssh2`.

**Current release: 1.8.0**

[Latest release](https://github.com/bren-wp/by-ftp/releases/latest) · [Installation](docs/INSTALLATION.md) · [ByFTP WEB](ByFTP%20WEB/README.md) · [Security](docs/SECURITY.md) · [Release verification](docs/RELEASE-VERIFICATION.md)

## 1.8.0 highlights

- Synchronizes Windows, Linux, macOS, Android, iOS and ByFTP WEB on the single canonical `VERSION` value `1.8.0`.
- Removes the standalone Windows `Uninstall.exe` binary from source, build output and installer payload. Windows Setup now embeds only the verified `ByFTP.exe` payload plus its integrity manifest; upgrades best-effort clean the legacy uninstaller from older installations after the new application commit succeeds.
- Upgrades the Android build chain to Android Gradle Plugin **9.3.2** and Gradle **9.7.1**, while retaining Android API 37, build-tools 36.0.0 and the current Go **1.27.0** toolchain used by native desktop builds.
- Makes ByFTP WEB security state fail closed instead of silently rolling back from stale JSON backups: application policy, user registry, rate limits, encrypted connection profiles, preferences and legacy migration data now require explicit recovery when their primary generation is corrupt or missing.
- Makes web login/registration rate limiting atomic and orders login gates IP-first so a blocked source cannot consume arbitrary account budgets.
- Prevents stale FTP/SFTP credentials from being inherited when a saved endpoint/account/key identity changes.
- Requires a pinned SHA-256 host fingerprint before any ByFTP WEB SFTP client can be created, preserving server-identity verification for both password and private-key authentication.
- Makes web password writes and authentication completion generation-safe: concurrent requests that verified an older password hash cannot overwrite a newer password or publish a session after a parallel password change.
- Makes user deletion retryable and symlink-safe, setup/config recovery fail closed, and profile/preference deletion semantics resistant to stale backup resurrection.
- Preserves the complete release-quality matrix: tests, race detector, `go vet`, security/privacy/docs audits, Windows x64/x86, Linux amd64/arm64/i386, macOS Universal, Android APKs, iOS arm64 artifacts and ByFTP WEB runtime validation.

## 1.7.1 release integrity

- Carries forward the complete 1.7.0 native cleanup and `ByFTP WEB/` integration from the verified final `main` tree.
- Blocks stale release workflows before any tag or GitHub Release mutation: the publisher resolves the repository's current `main` SHA and requires it to equal the workflow's exact release commit.
- Repeats the current-main check immediately before release creation/editing so an older VERSION-triggered run cannot publish after a newer integration merge lands.
- Adds a permanent Python regression that requires the main-head guard to remain ahead of release lookup/mutation.
- Keeps root `VERSION`, ByFTP WEB `VERSION`, composer metadata and the PWA cache namespace synchronized at 1.7.1.

## 1.7.0 highlights

- Integrated the complete maintainable **`ByFTP WEB/`** source tree instead of shipping the browser client as an unrelated ZIP.
- Added a dedicated web release audit, PHP syntax/unit gates and JavaScript syntax gates.
- ByFTP WEB has multi-user accounts, isolated per-user workspaces, encrypted saved remote credentials, FTP/FTPS/SFTP profiles, upload/download, folder upload, editor, search, favorites, copy/move/duplicate, recursive delete/CHMOD, checksum, ZIP and a mobile-installable PWA.
- Web remote paths are fail-closed: traversal, dot components, duplicate separators, backslashes and protocol-control characters are rejected instead of rewritten.
- Web host/port/fingerprint input is validated in raw form. Values such as `22junk`, host edge whitespace and CR/LF cannot be silently normalized into valid connection data.
- Public/shared-hosting web installs block private, loopback and reserved targets by default; an administrator must explicitly enable private-host access.
- Web passwords/private-key/passphrase material is removed from active transport profile state after authentication; persistent profile secrets are encrypted with Sodium or AES-256-GCM/OpenSSL fallback.
- PWA caching is restricted to static assets. Navigation, authentication, API, download, preview, diagnostics and settings responses are never intentionally stored offline.
- Fixed Windows remote Create Folder/Rename handling so typed remote names reach the canonical validator verbatim rather than being trimmed first.
- Fixed Linux/macOS terminal input handling so host, username and private-key paths preserve raw edge whitespace until central validation; only terminal CR/LF line endings are removed.
- Fixed Android document-provider upload naming when `DISPLAY_NAME` is missing or null by using a deterministic URI-tail/`upload.bin` fallback before canonical name validation.
- Fixed iOS saved-connection replacement so Keychain uses update-or-add semantics and does not delete the last valid preset before a replacement is safely stored.
- Repository-wide release hygiene continues to scan every tracked file for portable paths, accidental generated output, UTF-8/BOM/NUL problems, trailing whitespace, merge remnants and version drift.

## Platform matrix

| Surface | Transport support | Release/build form |
| --- | --- | --- |
| Windows x64/x86 | FTP, explicit FTPS, implicit FTPS, SFTP | Portable EXE, app-only Setup EXE, verified ZIP; no standalone uninstaller |
| Linux amd64/arm64/i386 | FTP, explicit FTPS, implicit FTPS, SFTP | DEB |
| macOS Universal | FTP, explicit FTPS, implicit FTPS, SFTP | Universal PKG |
| Android 8.0+ | FTP, explicit FTPS, implicit FTPS, SFTP | Debug-signed APK, unsigned optimized release APK |
| iOS 16+ | FTP, implicit FTPS | Unsigned arm64 IPA and `.app` ZIP |
| ByFTP WEB | FTP/FTPS; SFTP with PHP `ext-ssh2` | Shared-hosting PHP/PWA source tree |

Android production signing and Apple production signing remain external trust boundaries. Debug/unsigned artifacts are never described as store-signed production packages.

## Security model

ByFTP treats paths, endpoint identity, credentials and release metadata as security boundaries rather than UI conveniences.

Remote path/name input is validated before protocol commands are sent. Saved profiles and direct-connect flows reject noncanonical endpoint input rather than silently trimming control characters or edge whitespace. SFTP host-key verification remains mandatory where the client exposes SFTP. Android and desktop use canonical SHA-256 host-key fingerprints; ByFTP WEB requires a pinned SHA-256 SFTP fingerprint before client creation; iOS does not claim SFTP until an audited native implementation exists.

Plain FTP remains available for compatibility but does not encrypt credentials or content. Prefer SFTP or FTPS where supported. Android FTPS uses platform trust and endpoint checking; iOS implicit FTPS uses Apple Network.framework. PHP `ext-ftp` does not expose the same peer-verification controls as SFTP, so ByFTP WEB recommends fingerprint-pinned SFTP when a cryptographically verified server identity is required.

Saved mobile connection presets intentionally exclude passwords/passphrases. Windows keeps saved secrets behind its existing platform credential boundary. ByFTP WEB encrypts saved remote credential material with an installation-specific 256-bit key and isolates profile/preference data per ByFTP user.

No official client includes an advertising SDK or requires a ByFTP telemetry backend. Connections target endpoints selected by the user, subject to platform safety policies.

## Shared hosting

Native clients map the visible FTP root to the authenticated account namespace and expose non-secret shared-hosting diagnostics derived from the first listing already needed for the session. Common web roots are recognized in this deterministic order: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Detection is advisory only: ByFTP never silently changes or saves the user's selected path because a common web-root name was found.

ByFTP WEB can run directly on ordinary PHP shared hosting. It needs PHP 8.1+, a writable `storage/` directory, `ext-ftp` for FTP/FTPS, Sodium or OpenSSL for encrypted credential storage, optional `ext-ssh2` for SFTP and optional `ext-zip` for ZIP operations. Production deployments should use HTTPS. See [Shared hosting](docs/SHARED-HOSTING.md) and [ByFTP WEB documentation](ByFTP%20WEB/README.md).

## Mobile behavior

Android and iOS include local filtering/search, deterministic directory-first sorting, direct path navigation and multi-file upload. Transfer progress is byte-based. Batch upload can stop safely after the currently active file rather than tearing down an FTP transaction mid-command.

Android uses the Storage Access Framework and does not request broad storage access. Application backup/device transfer is disabled for private app data. iOS uses security-scoped document access and clears active sessions when entering the background.

## Desktop behavior

The desktop core is written in Go and shared by Windows, Linux and macOS. Windows has the native graphical shell and the verified x64/x86 build pipeline. The 1.8.0 Windows Setup installs only `ByFTP.exe`; it does not install or register a standalone `Uninstall.exe`. Linux/macOS retain the shared transport/security core and canonical platform packaging under `linux/` and `macos/`.

The terminal client preserves raw host, account and filesystem identity input until central validation. Valid Unix paths are not altered by UI preprocessing.

## Repository and release integrity

Root `VERSION` is the single native release source. `ByFTP WEB/VERSION` must match it exactly. CI enforces:

- repository-wide tracked-file audit;
- localization/version/documentation/security/privacy/release audits;
- Go formatting, unit/integration tests, race detector and `go vet`;
- Windows x64/x86 production builds with an app-only Setup payload and an explicit no-uninstaller invariant;
- Linux amd64/arm64/i386 DEB builds;
- macOS Universal PKG build;
- Android JUnit, lintDebug, lintRelease, debug APK and unsigned release APK using AGP 9.3.2 / Gradle 9.7.1;
- real arm64 iPhoneOS build plus unsigned IPA/app ZIP validation;
- ByFTP WEB PHP syntax, JavaScript syntax, web unit tests and the dedicated web audit;
- exact-current-`main` verification immediately before GitHub Release mutation.

Only the active public development history from **1.3.0 onward** remains in the current changelog/documentation surface. Older Git history is not rewritten, but obsolete pre-1.3 product-version references are not allowed to re-enter active source or documentation.

## Languages

English is the canonical source and fallback language for the desktop runtime and repository documentation. The desktop application currently supports 18 runtime languages: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.

Android, iOS and ByFTP WEB remain English-first at the canonical source/documentation level while platform-specific UI localization is expanded only through reviewed, complete locale sets. New primary documentation and fallback/error text must be written in English first so every supported platform retains a deterministic fallback.

## Build from source

Use the canonical platform entry points:

```text
Windows: BUILD-WINDOWS.ps1
Linux:   linux/BUILD.sh
macOS:   macos/BUILD.sh
Android: Gradle project under android/ (AGP 9.3.2 / Gradle 9.7.1)
iOS:     ios/BUILD.sh
Web:     ByFTP WEB/ (PHP 8.1+, no build step required)
```

For ByFTP WEB verification:

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
```

## Documentation

Core documentation:
[Architecture](docs/ARCHITECTURE.md) · [Contributing](docs/CONTRIBUTING.md) · [GitHub releases](docs/GITHUB-RELEASES.md) · [Installation](docs/INSTALLATION.md) · [Privacy](docs/PRIVACY.md) · [Release verification](docs/RELEASE-VERIFICATION.md) · [Roadmap](docs/ROADMAP.md) · [Security](docs/SECURITY.md) · [Shared hosting](docs/SHARED-HOSTING.md) · [Signing](docs/SIGNING.md) · [Support](docs/SUPPORT.md) · [Testing](docs/TESTING.md) · [Third-party notices](docs/THIRD-PARTY-NOTICES.md).

Platform/source guides:
[Linux](linux/README.md) · [macOS](macos/README.md) · [Android](android/README.md) · [iOS](ios/README.md) · [ByFTP WEB](ByFTP%20WEB/README.md) · [Build and verification tooling](scripts/README.md) · [Documentation index](docs/README.md).
