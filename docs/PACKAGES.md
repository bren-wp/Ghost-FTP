# Ghost FTP GitHub Packages

Ghost FTP **0.0.8** is the active release candidate. After the protected publication transaction succeeds, the exact verified release directory is also published to GitHub Packages as a **distribution bundle**.

## Package reference

```text
ghcr.io/bren-wp/ghost-ftp:0.0.8
```

The exact `0.0.8` package tag is the verification identity for the release transaction. Current semantic aliases and `latest` are maintained only after the exact-version package has been published and verified.

The GHCR object is a **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under `/ghostftp-release/`. Ghost FTP does not use GHCR as a hidden application backend, relay, account service or transfer service.

## 0.0.8 bundle shape

The canonical 0.0.8 release contains **14 platform artifacts / 17 public files**:

- Windows: one universal Setup and one universal Portable executable;
- Linux: one Installer and one Portable archive each for Debian, Ubuntu and Fedora;
- Android: one production-signed APK;
- browser helpers: one deterministic ZIP each for Chrome, Edge, Firefox and Opera;
- metadata: `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The GitHub Package is built from exactly that same verified 16-file release directory. Architecture-specific Windows staging executables and retired architecture-specific Linux `.deb`, `.rpm` and Portable archives are not part of the public bundle.

## Publication contract

Package publication occurs only after release quality plus Windows, Linux, Android and browser jobs succeed. The workflow:

- binds version/revision labels to root `VERSION` and exact `GITHUB_SHA`;
- requires trusted Authenticode for official Windows Setup/Portable;
- requires the Android APK to be signed by the protected production publisher and match `GHOSTFTP_ANDROID_CERT_SHA256`;
- verifies the six Linux universal distro bundles and their embedded amd64/arm64/i386 payloads;
- verifies deterministic Chrome/Edge/Firefox/Opera helper ZIPs;
- keeps internal architecture-specific Windows staging executables out of the public directory;
- creates the exact 16-file release directory before packaging;
- publishes exact version plus current aliases and `latest`;
- verifies `ghcr.io/bren-wp/ghost-ftp:0.0.8` after push;
- never treats the OCI object as a supported runtime container or hidden service.

## Current metadata identity

`BUILD-METADATA.txt` records at least:

```text
VERSION=0.0.8
RELEASE_TAG=ghostftp-v0.0.8
RELEASE_CHANNEL=current
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS
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
LINUX_ARM64_RUNTIME_EVIDENCE=build-and-package-ci
LINUX_I386_RUNTIME_EVIDENCE=build-and-package-ci
ANDROID_APK=production-signed
ANDROID_SIGNER_SHA256=<verified signer certificate SHA-256>
ANDROID_SFTP=hidden-until-strict-host-key-verification
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
BROWSER_DESKTOP_HANDOFF=unsupported
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.8
```

`ANDROID_SIGNER_SHA256` above is release **output metadata** containing the verified public certificate fingerprint. The protected GitHub Actions input secret that the workflow compares against is named **`GHOSTFTP_ANDROID_CERT_SHA256`**. Production private-key material and passwords are never written to metadata or included in the package payload.

## Integrity

`SHA256.txt` binds every public file except itself. Release digest readback compares the exact source-workflow bundle with GitHub Release per-asset SHA-256 digests before publication is considered verified.

Windows trust and exact-byte integrity are independent: official Windows artifacts need both valid trusted Authenticode and matching SHA-256. Android similarly requires matching release bytes plus the expected production signing-certificate fingerprint.

## Latest-only retention

After successful 0.0.8 release publication and remote readback, `.github/workflows/release-retention.yml` independently verifies the current release/tag/main identity and exact **16-file** asset set before removing superseded Ghost FTP Releases, tags, canonical release branches and obsolete package versions. `main` history is never rewritten.

Until that protected transaction succeeds, **0.0.7 remains the last actually published GitHub Release** and 0.0.8 remains a release candidate rather than a falsely advertised published build.

## Platform boundaries

Android production signing does not expose SFTP without strict maintained host-key verification. Browser packages do not add desktop launch/handoff, browser networking permissions or a Ghost FTP relay. macOS is included only as the Developer ID signed, Apple-notarized and stapled universal AppKit archive.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).


macOS: `Ghost-FTP-0.0.8-macOS-notarized.app.zip` (arm64 + x86_64, Developer ID signed, notarized and stapled).
