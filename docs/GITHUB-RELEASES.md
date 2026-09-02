# GitHub releases

ByFTP releases are built and published by `.github/workflows/release.yml`. The workflow is version-bound, serialized through the `byftp-release` concurrency group and will not publish until the complete quality/platform matrix passes.

**Current release target: 1.9.0**

## Trigger

The normal production path is a merge to `main` that changes root `VERSION`. A manual `workflow_dispatch` may also be used, but a supplied version must exactly match the canonical `VERSION` file.

The centralized publisher additionally verifies that the workflow commit is still the repository's current `main` commit immediately before public release mutation. A slower stale run therefore cannot publish an older source tree after `main` has advanced.

## Quality matrix

Publication depends on all of these jobs:

1. production quality: repository, WEB, localization, version, docs, security, privacy and release audits; Python regressions; Go 1.27.1 tests/race/vet; deterministic WEB release packaging;
2. Windows x64/x86 Setup + Portable production build and bundle verification;
3. Linux amd64/arm64/i386 DEB build and metadata verification;
4. macOS Universal PKG build and structure verification;
5. Android JUnit/lint/debug APK/unsigned release APK build using AGP 9.4.0 and Gradle 9.7.1;
6. native arm64 iPhoneOS build plus unsigned IPA/app ZIP packaging;
7. centralized public staging verification, SHA-256 generation and publication.

ByFTP WEB is validated in the quality job and now also produces the deployable `ByFTP-<version>-WEB-shared-hosting.zip` release artifact. The packager is deterministic and uses tracked production files only.

## Windows 1.9.0 contract

Windows publishes only:

- `ByFTP-<version>-Portable-x64.exe`
- `ByFTP-<version>-Setup-x64.exe`
- `ByFTP-<version>-Windows-x64.zip`
- `ByFTP-<version>-Portable-x86.exe`
- `ByFTP-<version>-Setup-x86.exe`
- `ByFTP-<version>-Windows-x86.zip`

There is no standalone `Uninstall.exe` source, build output, embedded payload or release asset. Setup embeds only `ByFTP.exe` and a schema-2 integrity manifest. Release metadata records `WINDOWS_UNINSTALLER=none`, and both build and staging gates fail if an uninstall-named public binary appears.

`scripts/package_windows_bundles.ps1` creates the verified Windows ZIPs from the already validated Setup/Portable executables, documentation and release metadata. This removes duplicated bundle logic from workflow YAML and keeps x64/x86 packaging under one testable implementation.

## Other platform assets

Linux publishes three DEBs: amd64, arm64 and i386. macOS publishes one Universal PKG. Android publishes a debug-signed APK and an optimized unsigned release APK. iOS publishes an unsigned arm64 IPA plus the corresponding unsigned `.app` ZIP. WEB publishes one verified shared-hosting PHP/PWA ZIP.

The Android release APK and iOS artifacts are deliberately not described as store-signed production packages. Production Android/Apple signing material remains an external trust boundary.

## Public staging

Before shared metadata, `scripts/prepare_release.ps1` requires exactly **15 platform artifacts**:

- Windows: 6
- Linux: 3
- macOS: 1
- Android: 2
- iOS: 2
- WEB: 1

It then generates:

- `SHA256.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

That produces exactly **18 final public release files**. Unexpected, missing, duplicated or uninstall-named platform artifacts cause publication to fail closed.

## Publication

`scripts/publish_release.ps1` is the only maintained GitHub Release mutation path. It:

- verifies the current `main` SHA still equals the exact release commit;
- verifies/creates the `v<version>` tag at that commit;
- creates or safely completes the GitHub Release;
- uploads only the expected verified assets;
- re-reads remote assets and verifies names, sizes and GitHub SHA-256 digests against local files;
- emits release verification markers only after the remote state matches the local release evidence.

The release workflow also packages the public Windows distribution as `ByFTP.Windows <version>` in GitHub Packages/NuGet. That package contains public Windows Setup/Portable/ZIP artifacts plus verified metadata; it does not contain a standalone uninstaller.

## Verification

Always verify downloaded artifacts against `SHA256.txt`. See [Release verification](RELEASE-VERIFICATION.md) for the full platform, WEB packaging and Windows app-only Setup contract.
