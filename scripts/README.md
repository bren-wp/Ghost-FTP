# ByFTP build and verification tools

This directory contains shared development/CI audit, packaging, release and verification utilities. Platform-specific production build entry points live with their applications under `linux/`, `macos/` and `ios/`; they are not duplicated here.

## Build and packaging tools

- `BUILD-LOCAL.sh` — local/offline cross-build smoke check for the shared desktop core.
- `make_payload.py` — creates the verified Windows installer payload.
- `pe_resources.py` — writes Windows PE icon and VERSIONINFO resources.
- `generate_brand_assets.py` — reproducibly generates and verifies PNG/ICO brand assets.
- `package_android.py` — validates debug/release APK structure and stages versioned Android release artifacts.
- `package_ios.py` — validates the native iOS `.app`, version/bundle identity, Mach-O executable and archive paths before staging versioned unsigned IPA/app artifacts.

Canonical production build entry points are [`BUILD-WINDOWS.ps1`](../BUILD-WINDOWS.ps1), [`linux/BUILD.sh`](../linux/BUILD.sh), [`macos/BUILD.sh`](../macos/BUILD.sh) and [`ios/BUILD.sh`](../ios/BUILD.sh).

## Audit tools

- `audit_localization.py` — verifies English-first localization, supported desktop catalogs and Windows startup fallback policy.
- `audit_version.py` — verifies the single `VERSION`, reviewed Go/Gradle toolchain pins and canonical platform build entry points.
- `audit_android.py` — verifies Android TLS/SSH, permissions, canonical names/login-root paths, lifecycle, credential lifetime, picker-state and version invariants.
- `audit_ios.py` — verifies native iOS project structure, transport/path hardening, pending-session/temp-file cleanup, privacy/lifecycle rules, Xcode version binding and unsigned IPA packaging contract.
- `audit_docs.py` — checks local documentation links, platform guides, the documentation index and version-neutral document titles.
- `audit_security.py` — protects filesystem, credential, transfer and session security invariants.
- `audit_privacy.py` — enforces privacy and network policy.
- `audit_release.py` — validates the production Windows/Linux/macOS/Android/iOS matrix, canonical platform build directories, obsolete-workflow removal and centralized publisher.
- `audit_release_version_guard.py` — prevents mutation of already-published version lines.

## Release and verification tools

- `verify_release.py` — validates Windows PE files and release security properties.
- `verify_bundle.py` — fail-closed validation of Windows release ZIP contents, paths and `BUNDLE-SHA256.txt`.
- `package_android.py` — fail-closed Android APK container/path validation and versioned staging.
- `package_ios.py` — fail-closed iOS app/IPA validation and versioned staging.
- `release_notes.py` — generates release notes from the exact matching `CHANGELOG.md` section.
- `publish_release.ps1` — centralized GitHub Release publication with tag/commit and remote asset integrity checks.
- `test_release_tools.py` — regression tests for release tooling.
- `test_package_android.py` — regression tests for Android APK staging/validation.
- `test_package_ios.py` — regression tests for iOS app/IPA staging, version checks and unsafe archive rejection.

## Production rules

1. `VERSION` is the only production version source.
2. Go telemetry must be disabled before production desktop builds.
3. Production Go builds run with `GOPROXY=off` and `GOSUMDB=off`.
4. Security, privacy, localization, documentation, Android, iOS and release audits must pass.
5. Linux packaging belongs under `linux/`, macOS packaging under `macos/`, and the iOS build entry point under `ios/`; obsolete platform wrappers under `scripts/` are rejected.
6. Windows bundles are checked against an explicit allowlist and SHA-256 manifest.
7. Android debug and unsigned release APKs must pass structural/path validation before staging.
8. iOS must compile as a native arm64 iPhoneOS app and its unsigned IPA/app artifacts must pass bundle/version/Mach-O/path validation before staging.
9. Public release staging must match the exact Windows/Linux/macOS/Android/iOS artifact allowlist.
10. GitHub Release publication is performed only through `publish_release.ps1` and final remote assets are re-verified.
11. Production code-signing identities are external secrets and must never be fabricated or committed.

## Documentation

See the [Linux guide](../linux/README.md), [macOS guide](../macos/README.md), [Android guide](../android/README.md), [iOS guide](../ios/README.md), [GitHub releases](../docs/GITHUB-RELEASES.md), [Release verification](../docs/RELEASE-VERIFICATION.md), [Testing](../docs/TESTING.md) and [Security](../docs/SECURITY.md).
