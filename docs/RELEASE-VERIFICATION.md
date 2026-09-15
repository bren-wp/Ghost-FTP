# Ghost FTP release verification

Ghost FTP **0.0.6** is the current published release. The existing `ghostftp-v0.0.6` tag/release is immutable and must not be rewritten by cleanup work.

The canonical 0.0.6 publication contains **13 platform artifacts / 16 public files**.

## 0.0.6 release identity

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
TITLE=Ghost FTP 0.0.6
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

## Canonical public files

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
Ghost-FTP-0.0.6-Android.apk
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Canonical release dispatch

Release branches use `release/ghostftp-vX.Y.Z`. Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. A push to `main`, including a `VERSION` change, **must never publish a release directly**.

A future release source must equal the intended verified commit and pass all required exact-head and post-merge gates. Existing release tags/assets must fail closed on collision rather than be replaced.

## Windows verification

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Official publication has no unsigned fallback. Production signing material remains outside repository source.

## Linux verification

The six Linux public bundles cover Debian, Ubuntu and Fedora, each with Installer + Portable form and embedded amd64/arm64/i386 payloads. Native runtime execution is claimed only where maintained evidence actually executes the package.

## Android verification

The public APK is `Ghost-FTP-0.0.6-Android.apk`. Production publication requires:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_CERT_SHA256
```

`apksigner verify --verbose --print-certs` must succeed and the signer certificate SHA-256 must match `GHOSTFTP_ANDROID_CERT_SHA256` exactly. CI-only identities are not production evidence. Android SFTP remains hidden until strict maintained host-key verification/pinning exists.

## Browser verification

Published 0.0.6 packages exist for Chrome, Edge, Firefox and Opera. Any 0.0.7 native-messaging architecture must add explicit tests for least privilege, manifest/host registration, bounded message validation, local-only credential handling and absence of remote code/telemetry.

## Build metadata

At minimum, published metadata identifies:

```text
BRAND=Ghost FTP
VERSION=0.0.6
RELEASE_TAG=ghostftp-v0.0.6
RELEASE_CHANNEL=current
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS
ANDROID_APK=production-signed
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.6
```

## Integrity and readback

`SHA256.txt` binds downloaded files to release bytes. Release automation validates the exact public allow-list and remote digest/readback state. Authentic Windows/Linux/Android runtime evidence remains exact-head bound; mockups are not execution evidence.

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

The GHCR object is a distribution bundle, not a runtime backend.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
