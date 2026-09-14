# Ghost FTP GitHub Packages

Ghost FTP **0.0.6** publishes a verified **distribution bundle** to GitHub Packages alongside the canonical GitHub Release.

## Current package reference

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

The workflow also maintains current semantic aliases and `latest`, while the exact `0.0.6` tag is the verification identity for this release transaction.

The GHCR object is a verified **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under `/ghostftp-release/`.

The canonical GitHub Release contains **18 platform artifacts / 21 public files**: two Windows executables, twelve Linux packages/archives, one production-signed Android APK, three deterministic browser-helper ZIPs and three metadata/verification files.

## Publication contract

Package publication occurs only after release quality plus Windows, Linux, Android and browser jobs succeed. The workflow:

- binds version/revision labels to root `VERSION` and exact `GITHUB_SHA`;
- requires trusted Authenticode for official Windows Setup/Portable;
- requires the Android APK to be signed by the protected production publisher and match `GHOSTFTP_ANDROID_SIGNER_SHA256`;
- verifies the Linux distro/Portable matrix and binary parity;
- verifies deterministic Chrome/Edge/Firefox helper ZIPs;
- keeps internal architecture-specific Windows staging executables out of the public directory;
- publishes exact version plus current aliases and `latest`;
- verifies `ghcr.io/bren-wp/ghost-ftp:0.0.6` after push;
- builds only from the same exact 21-file release directory used by GitHub Release publication;
- never uses the package as a hidden product backend or runtime service.

## Current metadata identity

`BUILD-METADATA.txt` records at least:

```text
VERSION=0.0.6
RELEASE_TAG=ghostftp-v0.0.6
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX,ANDROID
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
ANDROID_APK=production-signed
ANDROID_SIGNER_SHA256=<verified public certificate fingerprint>
ANDROID_SFTP=hidden-until-strict-host-key-verification
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox
BROWSER_DESKTOP_HANDOFF=unsupported
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.6
```

Production private-key material is never part of this metadata or package payload.

## Integrity

`SHA256.txt` binds every public file except itself. Release digest readback compares the exact source-workflow bundle with GitHub Release per-asset SHA-256 digests before publication is considered verified.

Windows trust and exact-byte integrity are independent: official Windows artifacts need both valid trusted Authenticode and matching SHA-256. Android similarly requires a matching SHA-256 plus the expected production signing-certificate fingerprint.

## Latest-only retention

After successful release publication/readback, `.github/workflows/release-retention.yml` verifies the current release/tag/main identity and 21-file asset set, then removes obsolete Ghost FTP Releases, tags, release branches and obsolete package versions while retaining current `0.0.6` identities. `main` history is not rewritten.

## Platform boundaries

Android public signing does not expose SFTP without strict maintained host-key verification. Browser packages do not add desktop launch/handoff. macOS remains outside the public bundle until real Developer ID signing and Apple notarization succeed.

GitHub Packages remains distribution infrastructure only. Ghost FTP has no hidden product backend, account service, telemetry endpoint or package-backed runtime dependency.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
