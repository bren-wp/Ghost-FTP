# Ghost FTP GitHub Packages

Ghost FTP **0.0.8** is already published through the exact-main **no-secret GitHub Release** path. That no-secret publication does **not** publish or update a GHCR package.

A separate protected production-signing workflow can publish a verified distribution bundle to GitHub Packages only when its Windows, Android and macOS production-signing gates all succeed.

## Protected package reference

```text
ghcr.io/bren-wp/ghost-ftp:<VERSION>
```

For version 0.0.8 the protected-path identity is:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.8
```

The presence of that name in documentation or workflow configuration is not evidence that a package was actually published. Package existence must be established by the protected workflow's own successful push and readback verification.

The GHCR object is a **distribution bundle, not a runtime container**. It must never be treated as an application backend, relay, account service, storage service or transfer service.

## Two release modes

### Published 0.0.8 no-secret distribution

The already-published 0.0.8 GitHub Release contains **14 platform artifacts / 17 public files**:

- Windows: universal Setup and Portable, intentionally unsigned;
- Linux: Debian, Ubuntu and Fedora Installer + Portable bundles;
- Android: installable release APK signed with a temporary one-run compatibility certificate;
- macOS: ad-hoc signed universal validation app, not Developer ID signed and not notarized;
- browser helpers: Chrome, Edge, Firefox and Opera ZIPs;
- metadata: `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The no-secret workflow reads no production signing secrets and does not publish GHCR.

### Protected production-signing distribution

The protected `.github/workflows/release.yml` path remains fail-closed. It requires:

- trusted Windows Authenticode;
- the protected Android publisher identity with exact signer-fingerprint verification;
- Developer ID signing, Hardened Runtime, Apple notarization, stapling and Gatekeeper verification for macOS;
- the same Linux and browser-helper verification contracts;
- exactly **17 public files** in the protected release directory.

Only after those gates succeed does that workflow publish the release-directory distribution bundle to GHCR and verify the exact-version package.

## Protected package metadata

When the protected production workflow succeeds, its `BUILD-METADATA.txt` records values such as:

```text
VERSION=<VERSION>
RELEASE_TAG=ghostftp-v<VERSION>
RELEASE_CHANNEL=current
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS,BROWSER_HELPER
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_AUTHENTICODE=signed
LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386
LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386
LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386
LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386
LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386
LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386
ANDROID_APK=production-signed
ANDROID_SIGNER_SHA256=<verified protected publisher certificate SHA-256>
ANDROID_SFTP=hidden-until-strict-host-key-verification
MACOS_APP=developer-id-signed-notarized-universal-arm64-x86_64
MACOS_SIGNING=developer-id-hardened-runtime-notarized-stapled
BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera
BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/<owner>/ghost-ftp:<VERSION>
```

Those values describe the **protected production path**, not the trust state of the already-published 0.0.8 no-secret release.

## Published 0.0.8 trust metadata

For the actual no-secret 0.0.8 release, the corresponding trust boundaries are:

```text
WINDOWS_AUTHENTICODE=unsigned
ANDROID_APK=temporary-compatibility-certificate
ANDROID_SIGNER_SHA256=<verified temporary compatibility certificate fingerprint>
MACOS_SIGNING=ad-hoc-not-notarized
ANDROID_SFTP=hidden-until-strict-host-key-verification
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

Code signing and protocol security are independent. The absence of production publisher credentials never permits weaker TLS, SFTP host-key verification, credential handling or path-safety behavior.

## Integrity and retention

`SHA256.txt` binds every public release file except itself. GitHub Release digest readback verifies the exact remote asset set and source identity.

The protected package workflow uses exact-version package identity before semantic aliases. Retention must preserve the immutable `ghostftp-v0.0.7` baseline and the published `ghostftp-v0.0.8` release/tag. It must never rewrite `main` history or mutate an existing published release.

The no-secret 0.0.8 publication does not perform destructive GHCR retention because it does not publish GHCR.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
