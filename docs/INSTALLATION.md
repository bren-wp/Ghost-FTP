# Installation

Ghost FTP currently supports **Windows, Linux and Android** application installs.

## Windows

Published Windows builds provide:

- Setup executable
- Portable executable

The universal Windows package carries the maintained architecture payloads described by release metadata. Trust prompts depend on the signing state of the specific release.

## Linux

Linux distribution bundles are produced for Debian, Ubuntu and Fedora families as installer and portable packages. Release metadata records the packaged architectures and validation boundary.

## Android

Android distribution is an APK. The signing identity and signer SHA-256 are verified by the applicable release workflow. Android SFTP remains hidden until strict host-key verification is maintained.

## macOS

There is no current macOS application or supported macOS package. The previous macOS source/build pipeline is retired and removed.

## Verification

For release assets:

1. confirm the release tag/version;
2. verify `SHA256.txt`;
3. use only artifacts listed by the release metadata;
4. respect the documented signing state instead of inferring trust from filenames.

See [Release verification](RELEASE-VERIFICATION.md) and [Signing](SIGNING.md).


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.


## Current public release shape

The active Windows/Linux/Android plus browser-helper publication contract contains **13 platform artifacts / 16 public files** after macOS retirement. Historical published releases remain immutable.

```text
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```
