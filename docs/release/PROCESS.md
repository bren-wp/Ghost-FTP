# Ghost FTP Release Process

Ghost FTP uses a canonical pre-1.0 semantic-version release train. Root `version.json` is the source of truth.

A successful compile is not equivalent to stable/FINAL acceptance.

## Development/release flow

1. Start from current `main`.
2. Make meaningful product/code/documentation changes.
3. Keep `version.json` and synchronized metadata consistent; do not create version-only commits.
4. Open a PR.
5. Require exact-head success for:
   - Ghost FTP quality
   - Ghost FTP protocol E2E
   - Ghost FTP native preview build
   - Ghost FTP native build
   - Ghost FTP Android
   - Validate Windows hardening
6. Read and fix concrete workflow logs if any gate fails.
7. Merge only the tested source.
8. On `main`, the canonical **Ghost FTP release** workflow is triggered from the successful native-preview build and verifies the release source/gates before publication.
9. Require the previous canonical release to exist before publishing the next version.
10. Package normalized Windows/Linux/Android/source/documentation assets and SHA-256 checksums from the verified source SHA.
11. Publish/update the version tag `v<version>` at that exact source SHA and verify uploaded asset digests/count.
12. Do not retarget older tags/releases to newer source.

## Canonical asset names

For version `<version>`:

### Windows x64

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Windows-x64-v<version>.zip`

### Linux x86-64

- `GhostFTP-Linux-x86_64-v<version>`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Linux-amd64-v<version>.deb`
- `GhostFTP-Linux-x86_64-v<version>.rpm`
- `GhostFTP-Linux-x86_64-v<version>.tar.gz`

### Android

- `GhostFTP-Android-v<version>.apk`

### Source/documentation

- `GhostFTP-v<version>-Source.zip`
- `GhostFTP-v<version>-Desktop-Source.zip`
- `GhostFTP-v<version>-Android-Source.zip`
- `GhostFTP-v<version>-Website.zip`
- `GhostFTP-v<version>-Updates.zip`
- `GhostFTP-v<version>-Documentation.zip`
- `GhostFTP-v<version>-SHA256SUMS.txt`

## Current release state

- Active source: **0.17.0 development**
- Latest published release: **0.16.0**
- `0.17.0` must not be described as published until `v0.17.0` exists and its canonical assets/digests have been verified.

## Stable / FINAL gates

Do not label an artifact FINAL until evidence exists for required installer lifecycle, target-OS visual acceptance, security failure paths, update/signing verification and accessibility acceptance.

Blocked gates are BLOCKED, not PASS.

## Release integrity

- End-user desktop GUI assets come from `ghostftp-desktop/`.
- Canonical Android APK comes from the verified Android artifact for the same release SHA.
- Support/browser-host tooling is not a production desktop release asset.
- Publication fails if required artifacts/gates are missing.
- Public filenames use `GhostFTP`; product copy uses `Ghost FTP`.
- Legacy release-candidate identifiers are historical aliases only.
