# Ghost FTP release verification

The current maintained release is **0.0.6**.

## Published 0.0.6 release identity

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
TITLE=Ghost FTP 0.0.6
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
LATEST_ONLY_RELEASE_RETENTION=YES
```

The release source must be the exact current `main` commit that passed the complete release gate.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
Ghost-FTP-0.0.6-Linux-Debian-arm64.deb
Ghost-FTP-0.0.6-Linux-Debian-i386.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-i386.tar.gz
```

Android and browser helpers:

```text
Ghost-FTP-0.0.6-Android.apk
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

macOS remains outside the public 21-file release until real Developer ID signing and Apple notarization succeed.

## Canonical release dispatch

The canonical branch namespace is `release/ghostftp-vX.Y.Z`; the 0.0.6 release branch is `release/ghostftp-v0.0.6`. It must point to exact fully verified current `main` and its version must match root `VERSION`.

Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. The branch trigger validates source/version equality, dispatches canonical `release.yml`, waits for the exact new release run to finish successfully, then dispatches and verifies retention.

A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Source verification

Before publication:

1. root `VERSION` must equal `0.0.6`;
2. `release/ghostftp-v0.0.6` must equal exact current `main`;
3. every exact-head release-prep workflow for the final candidate must be successful;
4. every required post-merge push workflow on the exact merge SHA must be successful;
5. authentic Windows/Linux/Android runtime evidence must bind to that exact source revision;
6. release quality, Windows, Linux, Android and browser jobs must succeed again from fresh source;
7. official Windows Setup/Portable must pass trusted Authenticode verification;
8. the Android APK must pass production signing verification and signer-fingerprint validation;
9. the public release must contain exactly the canonical 21-file set.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is accepted for exact-byte integrity only when its local hash matches the corresponding manifest entry.

```bash
sha256sum -c SHA256.txt
```

The canonical workflow also performs remote release digest readback against the exact locally assembled bundle.

## Windows verification

The public Windows packaging contract is:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Architecture-specific staging executables are internal verified inputs and must not appear among public assets. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` means current CI cross-builds and structurally verifies the ARM64 payload/signing/packaging path but does not claim native Windows ARM64 runtime execution.

Official Windows publication has no unsigned fallback. The protected production PFX/password must be available and both outer public EXEs must report a valid trusted Authenticode signature.

## Android production-signing verification

The public APK is:

```text
Ghost-FTP-0.0.6-Android.apk
```

The canonical release job requires:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_SIGNER_SHA256
```

The workflow builds the unsigned release APK, signs it with the protected publisher keystore, runs `apksigner verify --verbose --print-certs`, normalizes the signer certificate SHA-256 digest and requires exact equality with `GHOSTFTP_ANDROID_SIGNER_SHA256`. The production workflow must not generate its own replacement publisher identity.

The ordinary development artifact `Ghost-FTP-Android-dev.apk` is not accepted as the public APK. Its ephemeral CI signing identity proves only the mechanics of the signing pipeline.

Android SFTP remains hidden/unsupported until strict maintained host-key verification exists. A valid APK signature does not alter that protocol boundary.

## Browser-helper verification

The canonical browser job verifies source/permission contracts and builds deterministic ZIPs for Chrome, Edge and Firefox. Each archive must be non-empty and pass ZIP integrity validation. Browser publication does not add a supported browser-to-desktop URI/native-messaging handoff, network relay or automatic update service.

## Linux verification

Debian/Ubuntu DEB and Fedora RPM metadata must match version, architecture, product URL and maintained dependencies. For each architecture, release CI compares extracted package `ghostftp` bytes with the matching Portable archive. Native package-manager/runtime lifecycle is separately proven only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Build metadata

`BUILD-METADATA.txt` binds release identity to source and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.6
RELEASE_TAG=ghostftp-v0.0.6
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX,ANDROID
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
ANDROID_APK=production-signed
ANDROID_SIGNER_SHA256=<verified signer SHA-256>
ANDROID_SFTP=hidden-until-strict-host-key-verification
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox
BROWSER_DESKTOP_HANDOFF=unsupported
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.6
```

## Authentic runtime evidence

Exact-head UI evidence is source-bound. Maintained workflows capture real Windows, Linux and Android runtime surfaces and assemble a verified evidence bundle containing exact source SHA, expected filenames, byte counts and SHA-256 hashes. Mockups and generated approximations are not release evidence.

The Windows capture is valid for the maintained runner architecture but is not native ARM64 runtime evidence. macOS development-app CI is separate from public release publication and is not notarization evidence.

## Remote release readback

The publish workflow requires the remote GitHub Release asset set to match the exact 21-file allow-list immediately and after a delay. It requires `prerelease=false` and refuses to rewrite an existing tag/release.

The release digest-readback verifier compares GitHub's per-asset SHA-256 digests with the exact source-workflow bundle and checks that `BUILD-METADATA.txt` binds `COMMIT` to the expected release source SHA.

## GitHub Packages readback

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

The exact-version package is verified after push. It is a distribution bundle, not a supported runtime container.

## Latest-only retention verification

Only after the 0.0.6 release transaction succeeds may retention delete superseded public releases/tags/branches/package versions. Retention independently verifies the current `ghostftp-v0.0.6` release is non-draft/non-prerelease, has 21 assets and points to exact current `main` before cleanup. `main` history is never rewritten.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
