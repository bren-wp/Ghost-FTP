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
