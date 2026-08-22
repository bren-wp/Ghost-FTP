# Release verification

Use this checklist before trusting a ByFTP release.

## Version and source

Confirm the release tag is `v<version>` and that the tagged `VERSION` file contains the same semantic version. The intended release commit should match the publisher metadata. Published tags are not moved.

## Hashes

Download `SHA256.txt` from the release and calculate SHA-256 for the selected artifact with a trusted local tool. The digest must match exactly.

Windows ZIP bundles contain their own `BUNDLE-SHA256.txt`. The build workflow validates every payload entry against that manifest and rejects duplicate, unsafe, encrypted or unexpected internal entries.

## Platform artifacts

Expected public artifacts include Windows x64/x86 portable and setup executables plus ZIP bundles, Linux amd64/arm64/i386 DEBs and a macOS Universal PKG. Build metadata and release notes are published alongside them.

## Signatures

A cryptographic hash proves file identity relative to the published manifest; it does not by itself prove publisher identity. Verify Authenticode or Apple signing/notarization only when the release explicitly claims those credentials are configured.

## If verification fails

Do not run the artifact. Re-download from the official repository and compare again. Report reproducible mismatches without posting sensitive credentials.