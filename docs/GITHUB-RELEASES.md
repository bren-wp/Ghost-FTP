# Ghost FTP GitHub Releases

Ghost FTP **0.0.8** is the active release candidate. The last actually published GitHub Release is **0.0.7** until the protected 0.0.8 publication workflow completes successfully.

## Release identity

```text
Tag: ghostftp-v0.0.8
Title: Ghost FTP 0.0.8
Prerelease: false
```

Root `VERSION` is authoritative. Major version zero does not automatically make a Ghost FTP release a GitHub prerelease.

## Canonical release trigger

A push to `main`, including a `VERSION` change, **does not publish a release directly**. Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. Canonical release branches use:

```text
release/ghostftp-vX.Y.Z
```

For 0.0.8 the branch is `release/ghostftp-v0.0.8`. The branch trigger accepts it only when its semantic version equals root `VERSION` and its SHA equals exact current `main`.

The trigger snapshots existing workflow runs, dispatches `Publish Ghost FTP`, waits for the exact new run to finish successfully, and only then dispatches and verifies release retention. A successful dispatch request is not publication evidence.

## 0.0.8 public files

Ghost FTP 0.0.8 publishes **13 platform artifacts / 16 public files**.

Windows:

```text
Ghost-FTP-0.0.8-Setup.exe
Ghost-FTP-0.0.8-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.8-Linux-Debian-Installer.run
Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.8-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Fedora-Installer.run
Ghost-FTP-0.0.8-Linux-Fedora-Portable.tar.gz
```

Android:

```text
Ghost-FTP-0.0.8-Android.apk
```

Browser helper packages:

```text
Ghost-FTP-0.0.8-Chrome-Extension.zip
Ghost-FTP-0.0.8-Edge-Extension.zip
Ghost-FTP-0.0.8-Firefox-Extension.zip
Ghost-FTP-0.0.8-Opera-Extension.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

macOS remains an active development/source surface and is not in the public 0.0.8 allow-list until real Developer ID signing and Apple notarization are proven.

## Exact-head transaction

Before publication, `release.yml` requires its source SHA to remain exact current `main`. A moved `main`, pre-existing tag or pre-existing release fails closed rather than rewriting release identity.

After successful publication, retention independently verifies the **16-file** asset count and exact tag/main identity before removing superseded Ghost FTP Releases, tags, canonical release branches and obsolete GHCR versions. `main` history is never rewritten.

## Windows signing gate

Official Windows publication requires the protected trusted Authenticode identity. The workflow fails when the production PFX or password is absent and verifies both public executables with `Get-AuthenticodeSignature` before publication.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

There is no unsigned official-publication fallback and the workflow does not create a replacement self-signed production publisher identity.

## Android production-signing gate

The Android release job requires:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_CERT_SHA256
```

It signs `Ghost-FTP-0.0.8-Android.apk`, runs `apksigner verify --verbose --print-certs`, normalizes the signer certificate SHA-256 digest and requires exact equality with `GHOSTFTP_ANDROID_CERT_SHA256`. The workflow never generates a production Android publisher identity.

Android SFTP remains hidden until strict maintained host-key verification exists. Production signing does not weaken that boundary.

## Browser helper gate

Chrome, Edge, Firefox and Opera packages are rebuilt deterministically from the 0.0.8 source tree under `extensions/`. Official manifests remain zero-permission/zero-host-permission and publication preserves the sanitized `ghostftp://connect` browser-to-desktop handoff on supported Windows installs without adding a remote service or automatic update mechanism.

## Linux universal-distribution gate

`linux/BUILD-DISTROS.sh` builds one Installer and one Portable bundle per Debian, Ubuntu and Fedora. Every bundle carries amd64, arm64 and i386 payloads and chooses the native payload locally.

Native installer/runtime/GUI coverage is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. ARM64/i386 remain build/package verified unless native execution evidence is separately available.

## Artifact allow-list

`BUILD-METADATA.txt` records at least:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386
LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386
LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386
LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386
LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386
LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386
ANDROID_APK=production-signed
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

The sorted remote GitHub Release asset set must match the exact 16-file allow-list immediately and after delayed readback. `Prerelease: false` remains part of the current-channel contract.

The same verified release directory is published as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.8
```

This is a distribution bundle, not a supported runtime container.

## What counts as release evidence

A local build, PR artifact, debug APK, development certificate, self-signed Windows test certificate or ad-hoc macOS signature is not official publication evidence. Release evidence requires the canonical exact-main workflow, protected Windows/Android signing verification, exact GitHub Release readback, GHCR publication/readback and successful retention verification.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
