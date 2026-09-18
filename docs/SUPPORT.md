# Support

When reporting a Ghost FTP issue, include the exact version or development commit SHA, platform, architecture/device, protocol involved and reproducible steps. Remove real credentials from screenshots and logs.

## Windows

Include Windows version, Setup/Portable build type and whether the issue involves DPI, keyboard focus, packaging or a protocol operation.

## Linux

Include distribution/version, desktop session, architecture and whether the issue reproduces in installer or portable packaging.

## Android

Include Android version/device, storage permission/SAF context and the affected native screen. Android SFTP is not an active supported capability until strict host-key verification is maintained.

## Retired macOS reports

macOS is no longer an active application target. Historical macOS artifacts are unsupported and should not be used as the basis for current Ghost FTP behavior.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. Current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
