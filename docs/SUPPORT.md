# Support

When reporting a Ghost FTP issue, include the exact version or development commit SHA, platform, architecture/device, protocol involved and reproducible steps. Remove real credentials from screenshots and logs.

Ghost FTP **0.0.8** is the latest published release. Current source may contain post-0.0.8 development work for 0.0.9; that work does not rewrite the published 0.0.8 tag or assets.

Product website: https://ghostftp.com

## Windows

Include Windows version, Setup/Portable build type and whether the issue involves DPI, keyboard focus, packaging or a protocol operation.

## Linux

Include distribution/version, desktop session, architecture and whether the issue reproduces in installer or portable packaging.

## Android

Include Android version/device, storage permission/SAF context and the affected native screen. Android SFTP is not an active supported capability until strict host-key verification is maintained.

## Retired macOS reports

macOS is no longer an active application target. Historical macOS artifacts are unsupported and should not be used as the basis for current Ghost FTP behavior.

## Windows architecture scope

Windows universal Setup and Portable packages carry **x64, x86 and ARM64** native payloads. CI validates the ARM64 payload and PE structure without claiming native ARM64 execution:

```text
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```
