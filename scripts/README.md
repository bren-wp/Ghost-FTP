# ByFTP build and verification tools

This directory contains shared development/CI audit, packaging, release and verification utilities. Platform-specific production build entry points live with their applications under `linux/`, `macos/` and `ios/`; they are not duplicated here.

**Current release: 1.8.0**

## Build and packaging tools

- `BUILD-LOCAL.sh` — local/offline cross-build smoke check for the shared desktop core.
- `make_payload.py` — creates the Windows Setup **schema-2 app-only payload**. The archive contains exactly verified `ByFTP.exe` plus `manifest.json`; it no longer accepts or embeds a standalone uninstaller.
- `pe_resources.py` — writes Windows PE icon, manifest and VERSIONINFO resources for the only supported PE roles: `portable` and `setup`.
- `generate_brand_assets.py` — reproducibly generates and verifies PNG/ICO brand assets.
- `package_android.py` — validates debug/release APK structure and stages versioned Android release artifacts.
- `package_ios.py` — validates the native iOS `.app`, version/bundle identity, Mach-O executable and archive paths before staging versioned unsigned IPA/app artifacts.

Canonical production build entry points are [`BUILD-WINDOWS.ps1`](../BUILD-WINDOWS.ps1), [`linux/BUILD.sh`](../linux/BUILD.sh), [`macos/BUILD.sh`](../macos/BUILD.sh) and [`ios/BUILD.sh`](../ios/BUILD.sh). Windows 1.8.0 builds exactly Portable + Setup for x64/x86 and fails if an uninstall-named binary is produced.

## Reviewed toolchain

- Go 1.27.0 for the native desktop core and Windows/Linux/macOS builds.
- Android Gradle Plugin 9.3.2, Gradle 9.7.1, JDK 17, Android API 37 and Build Tools 36.0.0.
- Xcode/macOS runner for the native arm64 iPhoneOS build.
- PHP 8.1+ and Node syntax checks for ByFTP WEB verification.

The exact production pins are enforced by `audit_version.py` and the CI/release workflow definitions.

## Audit tools

- `audit_repository.py` — enumerates every tracked Git path/file and enforces portable paths, no committed build/cache output, strict UTF-8/text hygiene, no unresolved merge markers and canonical current-release metadata.
- `audit_localization.py` — verifies English-first localization, supported desktop catalogs and Windows startup fallback policy.
- `audit_version.py` — verifies the single `VERSION`, reviewed Go/Gradle toolchain pins, canonical platform build entry points, ByFTP WEB version/cache binding and repository-audit integration.
- `audit_web.py` — runs PHP/JavaScript syntax checks and ByFTP WEB runtime regressions for paths, users, recovery, rate limiting, encrypted profiles/preferences, SFTP host-key pinning and authentication concurrency.
- `audit_android.py` — verifies Android TLS/SSH, permissions, canonical names/login-root paths, lifecycle, credential lifetime, diagnostics, picker-state and version invariants.
- `audit_ios.py` — verifies native iOS project structure, transport/path hardening, diagnostics, pending-session/temp-file cleanup, privacy/lifecycle rules, Xcode version binding and unsigned IPA packaging contract.
- `audit_docs.py` — checks local documentation links, platform guides, the documentation index and version-neutral document titles.
- `audit_security.py` — protects filesystem, credential, transfer, session and shared-hosting diagnostic security invariants.
- `audit_privacy.py` — enforces privacy and network policy.
- `audit_release.py` — runs repository-wide/WEB integrity and validates the production Windows/Linux/macOS/Android/iOS matrix, canonical platform build directories, app-only Windows payload/no-uninstaller invariant, obsolete-workflow removal and centralized publisher.
- `audit_release_version_guard.py` — prevents mutation of already-published version lines.

## Regression and release tools

- `test_audit_repository.py` — unit coverage for repository path, symlink, text and current-version rules.
- `verify_release.py` — validates the Windows **Setup + Portable** PE pair for one architecture and emits `UNINSTALLER_BINARY=ABSENT`; it no longer accepts an uninstaller argument.
- `verify_bundle.py` — fail-closed validation of Windows release ZIP contents, paths and `BUNDLE-SHA256.txt`.
- `package_android.py` — fail-closed Android APK container/path validation and versioned staging.
- `package_ios.py` — fail-closed iOS app/IPA validation and versioned staging.
- `release_notes.py` — generates release notes from the exact matching `CHANGELOG.md` section.
- `publish_release.ps1` — centralized GitHub Release publication with current-`main` guard, tag/commit checks and remote asset integrity verification.
- `test_release_tools.py` — regression tests for release tooling.
- `test_package_android.py` — regression tests for Android APK staging/validation.
- `test_package_ios.py` — regression tests for iOS app/IPA staging, version checks and unsafe archive rejection.

## Production rules

1. `VERSION` is the only production version source; all maintained surfaces must resolve to the same release number.
2. Every tracked repository file must pass `audit_repository.py` through the release-integrity gate.
3. Go telemetry must be disabled before production desktop builds.
4. Production Go builds run with `GOPROXY=off`, `GOSUMDB=off` and local toolchain resolution.
5. Security, privacy, localization, documentation, WEB, Android, iOS, version and release audits must pass.
6. Linux packaging belongs under `linux/`, macOS packaging under `macos/`, and the iOS build entry point under `ios/`; obsolete platform wrappers under `scripts/` are rejected.
7. Windows Setup payload schema 2 contains only `ByFTP.exe` and its manifest. `cmd/uninstaller`, a generated `Uninstall.exe` and an uninstaller PE-resource role are forbidden by the 1.8.0 release contract.
8. Windows Setup/Portable x64/x86 binaries and ZIP bundles are checked against explicit allowlists and SHA-256 manifests.
9. Android debug and unsigned release APKs must pass JUnit, warning-as-error lint and structural/path validation before staging.
10. iOS must compile as a native arm64 iPhoneOS app and its unsigned IPA/app artifacts must pass bundle/version/Mach-O/path validation before staging.
11. Public release staging must match the exact Windows/Linux/macOS/Android/iOS artifact allowlist. ByFTP WEB is source-validated in the quality gate and does not invent a compiled binary package.
12. GitHub Release publication is performed only through `publish_release.ps1`; final remote assets are re-verified against the exact release commit and local SHA-256 evidence.
13. Production Android/Apple/Authenticode signing identities are external secrets and must never be fabricated or committed.

## Documentation

See the [Linux guide](../linux/README.md), [macOS guide](../macos/README.md), [Android guide](../android/README.md), [iOS guide](../ios/README.md), [ByFTP WEB guide](../ByFTP%20WEB/README.md), [GitHub releases](../docs/GITHUB-RELEASES.md), [Release verification](../docs/RELEASE-VERIFICATION.md), [Testing](../docs/TESTING.md) and [Security](../docs/SECURITY.md).
