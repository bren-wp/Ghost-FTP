# ByFTP build and verification tools

This directory contains development/CI build, audit, packaging, release and verification utilities. They do not add runtime dependencies to the ByFTP application.

## Build and packaging tools

- `BUILD-LOCAL.sh` — local/offline cross-build smoke check.
- `BUILD-LINUX.sh` — production Linux DEB builds for amd64, arm64 and i386.
- `BUILD-MACOS.sh` — production macOS Universal application/package build.
- `BUILD-IOS.sh` — native iPhoneOS arm64 build, iOS model/path regressions, deterministic icon generation and unsigned IPA/app packaging.
- `make_payload.py` — creates the verified Windows installer payload.
- `pe_resources.py` — writes Windows PE icon and VERSIONINFO resources.
- `generate_brand_assets.py` — reproducibly generates and verifies PNG/ICO brand assets.
- `package_android.py` — validates debug/release APK structure and stages versioned Android release artifacts.
- `package_ios.py` — validates the native iOS `.app`, version/bundle identity, Mach-O executable and archive paths before staging versioned unsigned IPA/app artifacts.

The canonical Windows production build is [`BUILD-WINDOWS.ps1`](../BUILD-WINDOWS.ps1) in the repository root.

## Audit tools

- `audit_localization.py` — verifies English-first localization, supported desktop catalogs and Windows startup fallback policy.
- `audit_version.py` — verifies that `VERSION` is the single production version source across desktop, Android and iOS packaging.
- `audit_android.py` — verifies Android TLS/SSH, permissions, login-root paths, lifecycle, picker-state and version invariants.
- `audit_ios.py` — verifies native iOS project structure, transport/path hardening, privacy/lifecycle rules, Xcode version binding and unsigned IPA packaging contract.
- `audit_docs.py` — checks local documentation links, the documentation index and version-neutral document titles.
- `audit_security.py` — protects filesystem, credential, transfer and session security invariants.
- `audit_privacy.py` — enforces privacy and network policy.
- `audit_release.py` — validates the production Windows/Linux/macOS/Android/iOS matrix and centralized publisher.
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
5. Windows bundles are checked against an explicit allowlist and SHA-256 manifest.
6. Android debug and unsigned release APKs must pass structural/path validation before staging.
7. iOS must compile as a native arm64 iPhoneOS app and its unsigned IPA/app artifacts must pass bundle/version/Mach-O/path validation before staging.
8. Public release staging must match the exact Windows/Linux/macOS/Android/iOS artifact allowlist.
9. GitHub Release publication is performed only through `publish_release.ps1` and final remote assets are re-verified.
10. Production code-signing identities are external secrets and must never be fabricated or committed.

## Documentation

See [GitHub releases](../docs/GITHUB-RELEASES.md), [Release verification](../docs/RELEASE-VERIFICATION.md), [Testing](../docs/TESTING.md), [Security](../docs/SECURITY.md), the [Android guide](../android/README.md) and the [iOS guide](../ios/README.md).
