# GitHub releases

ByFTP releases are produced by the repository release workflow. The workflow reads root `VERSION`; no platform maintains a second current-version constant.

## Gated matrix

Publication waits for all of these independent jobs:

- production quality/security/privacy/release audits and Go unit/race/vet checks,
- Windows x64/x86 production builds,
- Linux amd64/arm64/i386 DEB builds,
- macOS Universal PKG build,
- Android JUnit, debug/release lint, debug/release APK compilation and structural APK validation,
- iOS model/path regressions, native iPhoneOS arm64 Xcode build and structural unsigned IPA/app-bundle validation.

## Android artifacts

Starting with 1.1.1, Android is part of the public artifact contract rather than source-validation only:

- `ByFTP-<version>-Android-debug.apk` is debug-signed and installable for development/testing.
- `ByFTP-<version>-Android-release-unsigned.apk` is optimized/minified but intentionally unsigned.

The workflow must never present either artifact as a production store-signed package. Production Android distribution still requires a stable private signing identity managed outside the repository.

## iOS artifacts

Starting with 1.2.0, iOS is an independent native release platform under `ios/` and part of the public artifact contract:

- `ByFTP-<version>-iOS-arm64-unsigned.ipa` contains the validated arm64 iPhoneOS application under `Payload/ByFTP.app`.
- `ByFTP-<version>-iOS-arm64-unsigned-app.zip` contains the same validated unsigned `.app` bundle for inspection and external signing workflows.

These artifacts prove source/build/package integrity only. They are not App Store, TestFlight, ad-hoc or enterprise distributions until a legitimate Apple signing identity and provisioning profile are applied outside this repository and separately verified.

## Public asset contract

The 1.2.0+ workflow stages exactly 14 platform artifacts before shared metadata:

- six Windows packages,
- three Linux DEBs,
- one macOS PKG,
- two Android APKs,
- two iOS unsigned arm64 artifacts.

It then adds `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, producing 17 public release files in total.

## Publication integrity

The publish job downloads only the named platform artifacts and compares them with an exact allowlist before generating release metadata. `SHA256.txt` covers all public platform packages and shared metadata. `scripts/publish_release.ps1` is the only GitHub Release mutation path and validates the release tag/commit plus final remote asset sizes/digests.

Published semantic releases are treated as immutable. Corrections require a new version rather than silently replacing artifacts under an existing release tag.
