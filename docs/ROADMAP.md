# Roadmap

Ghost FTP development is focused on **Windows, Linux and Android**.

## Current priorities

1. close visible deltas against the maintained Ghost FTP reference composition;
2. keep every visible control engine-backed;
3. improve text visibility, keyboard/touch usability and modal consistency;
4. keep Windows/Linux behavior aligned without forcing identical platform internals;
5. improve Android mobile ergonomics while preserving its stricter capability boundary;
6. strengthen automated UI regression and authentic screenshot evidence;
7. keep release/signing/security boundaries fail-closed.

## Platform focus

### Windows
Continue native UI/UX polish, packaging reliability, accessibility and keyboard/runtime validation.

### Linux
Continue master-layout polish, distro packaging/install lifecycle validation and native modal behavior.

### Android
Continue native workspace polish, SAF/lifecycle correctness, transfer usability and security-hardening work required before exposing SFTP.

## Retired work

macOS is removed from the active roadmap, source tree and release pipelines.

## Release discipline

Published releases remain immutable. New source changes require a higher product version and fresh exact-source artifacts/checksums.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. Current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
