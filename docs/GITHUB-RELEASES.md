# GitHub releases

ByFTP releases are built and published by `.github/workflows/release.yml`. The workflow is version-bound, serialized through the `byftp-release` concurrency group and will not publish until the complete quality/platform matrix passes.

**Current release target: 1.8.0**

## Trigger

The normal production path is a merge to `main` that changes root `VERSION`. A manual `workflow_dispatch` may also be used, but a supplied version must exactly match the canonical `VERSION` file.

The centralized publisher additionally verifies that the workflow commit is still the repository's current `main` commit immediately before public release mutation. A slower stale run therefore cannot publish an older source tree after `main` has advanced.

## Quality matrix

Publication depends on all of these jobs:

1. production quality: repository, WEB, localization, version, docs, security, privacy and release audits; Python release-tool regressions; Go tests/race/vet;
2. Windows x64/x86 Setup + Portable production build and bundle verification;
3. Linux amd64/arm64/i386 DEB build and metadata verification;
4. macOS Universal PKG build and structure verification;
5. Android JUnit/lint/debug APK/unsigned release APK build using AGP 9.3.2 and Gradle 9.7.1;
6. native arm64 iPhoneOS build plus unsigned IPA/app ZIP packaging;
7. centralized staging verification and publication.

ByFTP WEB does not invent a compiled web artifact; its source/runtime/security contract is enforced inside the quality job and its VERSION must match root VERSION.

## Windows 1.8.0 contract

Windows publishes only:

- `ByFTP-<version>-Portable-x64.exe`
- `ByFTP-<version>-Setup-x64.exe`
- `ByFTP-<version>-Windows-x64.zip`
- `ByFTP-<version>-Portable-x86.exe`
- `ByFTP-<version>-Setup-x86.exe`
- `ByFTP-<version>-Windows-x86.zip`

Starting with 1.8.0 there is no standalone `Uninstall.exe` source, build output, embedded payload or release asset. Setup embeds only `ByFTP.exe` and a schema-2 integrity manifest. Release metadata records `WINDOWS_UNINSTALLER=none`, and CI fails if an uninstall-named binary appears.

The verified Windows ZIPs contain public binaries, release notes/build metadata, root README/CHANGELOG/LICENSE and documentation. Bundle SHA-256 is checked before publication.

## Other platform assets

Linux publishes three DEBs: amd64, arm64 and i386. macOS publishes one Universal PKG. Android publishes a debug-signed APK and an optimized unsigned release APK. iOS publishes an unsigned arm64 IPA plus the corresponding unsigned `.app` ZIP.

The Android release APK and iOS artifacts are deliberately not described as store-signed production packages. Production Android/Apple signing material remains an external trust boundary.

## Public staging

Before publication, the workflow requires exactly 14 platform artifacts. It then generates:

- `SHA256.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

That produces 17 final public release files. Unexpected/missing platform artifacts cause publication to fail closed.

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

Always verify downloaded artifacts against `SHA256.txt`. See [Release verification](RELEASE-VERIFICATION.md) for the full platform and Windows app-only Setup contract.
