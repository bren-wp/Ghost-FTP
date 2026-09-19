# GitHub Releases

Ghost FTP releases are immutable, versioned and bound to exact source.

Current source release line: **0.0.9**. The active release shape is **9 platform artifacts / 12 public files**.

Tag: ghostftp-v0.0.9

Canonical Linux release names for the current source identity include `Ghost-FTP-0.0.9-Linux-Debian-Installer.run`, `Ghost-FTP-0.0.9-Linux-Ubuntu-Portable.tar.gz` and `Ghost-FTP-0.0.9-Linux-Fedora-Installer.run`.

## Current application artifacts

A full supported release may contain:

- Windows Setup
- Windows Portable
- Linux Debian installer + portable
- Linux Ubuntu installer + portable
- Linux Fedora installer + portable
- Android APK
- BUILD-METADATA.txt
- RELEASE-NOTES.txt
- SHA256.txt

macOS and browser-extension artifacts are retired and must not be assembled or published by current workflows.

## Release identity

The root `VERSION` file is the canonical product version. The release tag format is:

`ghostftp-v<version>`

A published tag or release asset set must never be rewritten. New source changes require a new version.

## Exact-head transaction

Release publication is an exact-head transaction: build and verify the precise release commit, refuse tag rewrites, publish only the canonical allow-list, then read back release identity and digests before considering the release verified.

## Verification

Current release publication validates exact main/source identity, signing state where required, artifact counts, SHA-256 checksums and the expected allow-list before GitHub Release creation.

## Windows architecture evidence

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The public Windows Setup and Portable launchers carry x64, x86 and ARM64 native payloads. CI verifies the ARM64 payload structure and PE identity, but does not claim native ARM64 runtime execution; that limitation is recorded explicitly by `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`.
