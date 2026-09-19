# GitHub Releases

Ghost FTP releases are immutable versioned publications built from the exact expected main commit.

## Current release

Current published version: **0.0.8**  
Current development line: **0.0.9**

## Public artifact shape

The active release shape contains **13 platform artifacts / 16 public files**:

- Windows Setup;
- Windows Portable;
- Debian Installer + Portable;
- Ubuntu Installer + Portable;
- Fedora Installer + Portable;
- Android APK;
- Chrome helper;
- Edge helper;
- Firefox helper;
- Opera helper;
- RELEASE-NOTES.txt;
- BUILD-METADATA.txt;
- SHA256.txt.

## Canonical release dispatch

A release branch follows the form:

`release/ghostftp-vX.Y.Z`

The branch trigger dispatches the protected release workflow. It does not publish a release directly.

The release version must match root `VERSION`, and the source SHA must still match the expected main commit.

## Trust

Official Windows publication requires the configured trusted Authenticode identity.

Official Android publication requires the configured protected signing identity and exact signer fingerprint policy.

Linux and browser-helper artifacts are covered by release checksums and exact-head metadata.

## Retired platform

No macOS artifact is part of the active release allow-list.
