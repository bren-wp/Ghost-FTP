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

## Implemented 0.0.8 baseline

Ghost FTP **0.0.8** is the published baseline while the 0.0.9 development line is being validated.

- Navigation bookmarks and profile start directories — Status: implemented in Ghost FTP 0.0.8.
- Queue priority/reordering — Status: implemented in Ghost FTP 0.0.8.
- Non-destructive current-folder filter — Status: implemented in Ghost FTP 0.0.8.
- P0 — bounded recursive local/server search — Status: implemented in Ghost FTP 0.0.8.

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

## Windows architecture scope

Windows universal Setup and Portable packages carry **x64, x86 and ARM64** native payloads. CI validates the ARM64 payload and PE structure without claiming native ARM64 execution:

```text
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```
