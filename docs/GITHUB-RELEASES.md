# GitHub Releases

Ghost FTP releases are immutable, versioned and bound to exact source.

## Current application artifacts

A full supported release may contain:

- Windows Setup
- Windows Portable
- Linux Debian installer + portable
- Linux Ubuntu installer + portable
- Linux Fedora installer + portable
- Android APK
- Chrome, Edge, Firefox and Opera browser-helper ZIPs
- BUILD-METADATA.txt
- RELEASE-NOTES.txt
- SHA256.txt

macOS artifacts are retired and must not be assembled or published by current workflows.

## Release identity

The root `VERSION` file is the canonical product version. The release tag format is:

`ghostftp-v<version>`

A published tag or release asset set must never be rewritten. New source changes require a new version.

## Verification

Current release publication validates exact main/source identity, signing state where required, artifact counts, SHA-256 checksums and the expected allow-list before GitHub Release creation.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
