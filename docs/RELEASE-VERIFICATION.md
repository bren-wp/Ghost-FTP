# Ghost FTP release verification

The current maintained release is **0.0.5**.

## Published 0.0.5 release identity

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
TITLE=Ghost FTP 0.0.5
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
LATEST_ONLY_RELEASE_RETENTION=YES
```

The release source must be the exact current `main` commit that passed the complete release gate. The canonical public release contains **14 platform artifacts / 17 public files**.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

The independently validated Android APK, macOS development app and browser companion source are not among these 17 public Windows/Linux release files.

## Canonical release dispatch

The canonical branch namespace is `release/ghostftp-vX.Y.Z`; the 0.0.5 release branch is `release/ghostftp-v0.0.5`. It must point to exact fully verified current `main` and its version must match root `VERSION`.

Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. The branch trigger validates source/version equality, dispatches canonical `release.yml`, waits for the exact newly created release run to finish successfully, then dispatches and verifies retention.

A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Source verification

Before publication:

1. root `VERSION` must equal `0.0.5`;
2. the release branch must equal `release/ghostftp-v0.0.5` and exact current `main`;
3. all exact-head release-prep workflows triggered for the final candidate must be successful;
4. all required post-merge push workflows on the exact merge SHA must be successful;
5. authentic maintained runtime evidence must bind to the exact source revision;
6. the release workflow's quality, Windows and Linux jobs must succeed again from fresh source;
7. the public Windows Setup and Portable artifacts must pass trusted Authenticode verification;
8. Windows metadata must record the x64/x86/ARM64 embedded payload set and the ARM64 runtime-evidence boundary.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is trusted for exact-byte integrity only when its local hash matches the corresponding entry.

There is no public x32/x86/x64/ARM64 Windows alias set in 0.0.5. Only universal Setup and Portable executables are public; architecture-specific native payloads are internal verified staging inputs.

## Windows universal payload verification

The public Windows packaging contract is:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

`BUILD-WINDOWS-ARCH-STAGE.ps1` creates native Setup/Portable staging binaries for x64, x86 and ARM64. `scripts/verify_release.py` validates the architecture-specific PE machine/magic pair, GUI subsystem, required mitigation flags, `.rsrc`, product/version metadata, manifest architecture, telemetry-marker absence and distinct Setup/Portable identity. ARM64 uses machine `0xAA64`, PE32+ and `processorArchitecture="arm64"`.

`BUILD-WINDOWS.ps1` embeds all three native payload families into each public universal executable. At runtime the x86 bootstrap calls `GetNativeSystemInfo`, maps the native processor to `x64`, `x86` or `arm64`, reads only that embedded payload, stages it under Local AppData, verifies the staged bytes against the embedded SHA-256 identity and starts it without a network download.

Architecture-specific staging executables must never appear among public release assets. An unexpected `*-x64.exe`, `*-x86.exe`, `*-x32.exe` or `*-arm64.exe` is a release failure.

The ARM64 evidence statement is intentionally narrow. Current maintained Windows CI cross-builds and structurally verifies ARM64 artifacts and packaging/signing mechanics, but does not claim native ARM64 execution because the maintained Windows runner is not ARM64. Native ARM64 runtime evidence must only be claimed after a maintained ARM64 Windows runner/device executes that path successfully.

## Build metadata

`BUILD-METADATA.txt` binds release identity to source and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.5
RELEASE_TAG=ghostftp-v0.0.5
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.5
```

Android and macOS remain independently validated source/development platforms and are not silently added to the release platform/file count.

## Windows Authenticode

Official Windows publication requires trusted Authenticode. The production workflow **does not create a self-signed production identity** and has no unsigned-publication fallback.

The canonical `Publish Ghost FTP` workflow requires protected production signing credentials. If the PFX or its password is unavailable, publication fails before release creation. Both public Windows executables are checked with the Windows Authenticode API and must have a signer certificate with status `Valid`.

The native ARM64 client and setup payloads pass through the same signing path as x64/x86 staging payloads before embedding when production signing is configured. The outer universal Setup/Portable executables are then signed and verified as the two public Windows release files.

`scripts/verify_release.py` adds an independent fail-closed check: while running under the public release workflow it rejects unsigned Setup or Portable artifacts.

Local development and ordinary CI Windows builds may be unsigned. Those artifacts are test/development outputs and are not accepted as official public release evidence.

## Linux package verification

Debian/Ubuntu DEB and Fedora RPM metadata must match version, architecture, product URL and maintained dependencies. For each architecture the release workflow compares extracted package `ghostftp` bytes with the matching Portable archive. Native package-manager/runtime lifecycle is separately proven only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Android development verification

The Android APK workflow verifies source/security/lifecycle contracts, JVM regressions, lint, installable APK construction, APK identity verification and artifact upload. This proves the maintained development surface builds cleanly but does not add an APK to the 17-file public Windows/Linux release allow-list.

## macOS development verification

The macOS workflow builds and validates the universal native development app against the shared engine and platform-specific source contract. Development app success is not equivalent to a public notarized macOS release; macOS remains outside the current 17-file public release allow-list until a separate production signing/notarization/publication transaction is completed.

## Authentic runtime evidence

Exact-head UI evidence is immutable and source-bound. Maintained workflows capture real runtime surfaces and record exact source SHA, expected filenames, byte counts and SHA-256 hashes before emitting verified evidence. Mockups and generated approximations are not release evidence.

Current Windows authentic UI evidence runs on the maintained Windows runner architecture. It is valid Windows runtime evidence for that runner, but it is not native ARM64 runtime evidence and must not be represented as such.

## Remote release read-back

The publish workflow requires the remote GitHub Release asset set to match the exact 17-file allow-list immediately and after a delay. For 0.0.5 it requires `prerelease=false`. A local build alone is not release evidence.

## GitHub Packages read-back

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The exact version tag is verified after push. The package is a distribution bundle, not a supported runtime container.

## Latest-only retention verification

Only after the 0.0.5 release read-back succeeds does retention remove superseded public versions. Retention independently verifies current release/tag/main/package identity before destructive cleanup.

Completion requires the current release/tag to be `ghostftp-v0.0.5`, the current canonical release branch to remain, the exact 0.0.5 package version to remain, obsolete Ghost FTP release identities/package versions to be removed and `LATEST_ONLY_RELEASE_RETENTION=YES` to be emitted. `main` history is never rewritten.

## Verification commands

```bash
sha256sum -c SHA256.txt
```

For Windows, also require a valid Authenticode signature on both official executables; an unsigned file does not satisfy the current official public-release contract. Confirm that release metadata reports `WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64`, and do not infer native ARM64 runtime proof from cross-build verification alone.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
