# Ghost FTP GitHub Releases

Ghost FTP **0.0.6** is the current published GitHub Release. `ghostftp-v0.0.6` was published on 14 September 2026 as a non-prerelease release. It is immutable release history and must not be retagged, rewritten or republished by 0.0.7 cleanup work.

## Release identity

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
TITLE=Ghost FTP 0.0.6
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

## Canonical release trigger

A push to `main`, including a `VERSION` change, **does not publish a release directly**. Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. Release branches use:

```text
release/ghostftp-vX.Y.Z
```

A release branch is accepted only when its semantic version equals root `VERSION` and its SHA equals the intended verified source. Publication must fail closed on pre-existing tag/release identity rather than overwrite it.

## Published 0.0.6 public files

Ghost FTP 0.0.6 contains **13 platform artifacts / 16 public files**.

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

## Signing and evidence gates

Official Windows publication requires trusted Authenticode on both public executables. Android publication requires the protected production keystore and exact signer certificate SHA-256 equality with `GHOSTFTP_ANDROID_CERT_SHA256`. Browser packages are deterministic release artifacts. macOS remains outside the public release until Developer ID signing and notarization are proven.

Release verification binds artifacts to exact source identity, validates the public allow-list and SHA-256 metadata, and performs remote asset readback. Development or CI-only signing identities are never substituted for production publisher identities.

## Distribution bundle

The verified release directory is also represented as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

This is a distribution bundle, not a supported runtime container or credential relay.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
