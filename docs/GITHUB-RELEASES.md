# Ghost FTP GitHub Releases

Ghost FTP **0.0.6** is the current published release contract. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

```text
Tag: ghostftp-v0.0.6
Title: Ghost FTP 0.0.6
Prerelease: false
```

Root `VERSION` is authoritative. Major version zero is not automatically mapped to GitHub prerelease state.

## Canonical release trigger

A normal push to `main`, including a change to `VERSION`, **does not publish a release directly**. Canonical `release.yml` is `workflow_dispatch`-only. Canonical release branches use:

```text
release/ghostftp-vX.Y.Z
```

For 0.0.6 the branch is `release/ghostftp-v0.0.6`. `.github/workflows/release-branch-trigger.yml` accepts the branch only when the semantic version matches root `VERSION` and the branch SHA equals exact current `main`.

The release-branch trigger snapshots existing release-run IDs, dispatches `release.yml`, identifies the new exact-main `Publish Ghost FTP` run, waits for terminal success, and only then dispatches and waits for `release-retention.yml`. A successful dispatch request alone is not publication evidence.

## 0.0.6 public files

Ghost FTP 0.0.6 exposes **18 platform artifacts / 21 public files**.

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

Android:

```text
Ghost-FTP-0.0.6-Android.apk
```

Browser helper packages:

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

macOS remains a development/source surface and is not included in this public allow-list.

## Exact-head and immutable-current transaction

Before publication, `release.yml` checks that current `main` still equals the workflow source SHA. It repeats the check before delayed remote readback. A moved `main`, pre-existing tag or pre-existing release fails closed rather than rewriting an existing release identity.

After a successful release, retention verifies the 21-file asset count and exact tag/main identity, then removes superseded Ghost FTP Releases, tags, canonical release branches and obsolete GHCR package versions. `main` history is never rewritten.

## Windows signing gate

Official Windows publication requires the protected trusted Authenticode identity. `Publish Ghost FTP` fails when the production PFX or password is unavailable and verifies both public executables with `Get-AuthenticodeSignature` before publication.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

There is no supported unsigned official-publication continuation path. The workflow does not create a replacement self-signed production publisher identity.

## Android production-signing gate

The Android release job builds the unsigned release APK, then requires protected publisher credentials:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_CERT_SHA256
```

It signs `Ghost-FTP-0.0.6-Android.apk` with Android `apksigner`, verifies the APK, extracts the certificate SHA-256 fingerprint and requires it to equal `GHOSTFTP_ANDROID_CERT_SHA256`. The workflow does not generate a production Android publisher identity.

Android SFTP remains hidden until strict maintained host-key verification exists. A valid APK signature is not permission to weaken that protocol boundary.

## Browser helper gate

Chrome, Edge and Firefox packages are rebuilt deterministically from the 0.0.6 source manifests. Their release status does not add a supported browser-to-desktop URI/native-messaging handoff, store approval or automatic update service.

## Linux parity gate

`linux/BUILD-DISTROS.sh` remains the canonical Linux release builder. Package metadata and extracted executable bytes are verified before publication. Native install/remove/runtime/GUI coverage is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Artifact allow-list

`BUILD-METADATA.txt` records the canonical shape:

```text
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
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
```

The remote sorted GitHub Release asset set must match the exact 21-file allow-list immediately and after a delay. `Prerelease: false` remains part of the current channel contract.

The same verified release directory is published as the distribution bundle:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

It is not a supported runtime container.

## What counts as release evidence

A local build, PR CI run, debug APK, development certificate or ad-hoc macOS signature is not official publication evidence. Release evidence requires the canonical exact-main workflow, protected Windows/Android signing verification, exact GitHub Release readback, GHCR publication/readback and the successful retention chain.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
