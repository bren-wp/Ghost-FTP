# Ghost FTP Release Process

Ghost FTP uses a canonical pre-1.0 semantic-version release train. Root `version.json` is the source of truth.

A successful compile is not equivalent to stable/FINAL acceptance.

## Development/release flow

1. Start from current `main`.
2. Make meaningful product/code/documentation changes.
3. Keep `version.json` and synchronized metadata consistent; do not create version-only commits.
4. Keep the previous canonical release recorded in `previousVersion` (currently `0.30.7` for the 0.30.8 cycle).
5. Open a PR.
6. Require exact-head success for:
   - Ghost FTP quality
   - Ghost FTP protocol E2E
   - Ghost FTP native build
   - Ghost FTP Android
   - Validate Windows hardening
   - Ghost FTP macOS (development Preview gate for 0.30.8)
7. Read and fix concrete workflow logs if any gate fails.
8. Merge only the tested source.
9. On `main`, the successful **Ghost FTP native build** triggers the canonical **Ghost FTP release** workflow.
10. The release job waits for the remaining exact-SHA gates and requires the previous canonical release to exist.
11. Package normalized Windows/Linux/Android/source/documentation/native-QA assets and SHA-256 checksums from the verified source SHA.
12. Publish `v<version>` at that exact source SHA and verify uploaded asset digests/count.
13. Never retarget an existing version tag to different source.

## Canonical asset names

For version `<version>`:

### Windows x64

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Windows-x64-v<version>.zip`
- `GhostFTP-Windows-x64-v<version>-Native-QA.zip`

### Linux x86-64

- `GhostFTP-Linux-x86_64-v<version>`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Linux-amd64-v<version>.deb`
- `GhostFTP-Linux-x86_64-v<version>.rpm`
- `GhostFTP-Linux-x86_64-v<version>.tar.gz`

### Android

- `GhostFTP-Android-v<version>.apk.unsigned`
- `GhostFTP-Android-v<version>-Installable-Preview.apk`

### macOS Preview

- `GhostFTP-macOS-v<version>-Preview.zip`

The macOS artifact is a separately verified ad-hoc-signed development Preview. It is not Developer ID signed/notarized and must not be represented as a production macOS package.

### Signed desktop updater assets

When in-app updater signing is enabled, publication requires all four updater assets below:

- `GhostFTP-Windows-x64-Setup-v<version>.exe.sig`
- `GhostFTP-Linux-x86_64-v<version>.AppImage.sig`
- `GhostFTP-v<version>-latest.json`
- `GhostFTP-v<version>-Update-Service.zip`

The release workflow and final publication script fail closed if this optional updater set is partial or inconsistent. The manifest is validated for the exact version, and the copy packaged inside Update-Service must match the separately published manifest byte-for-byte.

If no updater signatures are present, the stable GitHub release may still publish the normal verified application packages, checksums and QA evidence. In that mode no updater manifest or Update-Service bundle is emitted.

### Source/documentation

- `GhostFTP-v<version>-Source.zip`
- `GhostFTP-v<version>-Desktop-Source.zip`
- `GhostFTP-v<version>-Android-Source.zip`
- `GhostFTP-v<version>-macOS-Source.zip`
- `GhostFTP-v<version>-Updates.zip`
- `GhostFTP-v<version>-Documentation.zip`
- `GhostFTP-v<version>-SHA256SUMS.txt`

## Current release cycle

- Active source/release cycle: **0.30.8**
- Previous canonical release: **0.30.7**
- Live publication state is determined by GitHub Releases and exact tag/source verification.

## Release integrity

- End-user desktop GUI assets come from `ghostftp-desktop/`.
- Canonical Windows/Linux binaries and native QA evidence come from the same successful native build run.
- Android publication must contain the verified unsigned production `com.ghostftp.android` build and the separately verified installable preview from the same release SHA. The unsigned production file must remain explicitly suffixed `.unsigned` so it is never misrepresented as directly installable.
- Support/browser-host tooling is not a production desktop release asset.
- The final publication script verifies the checksum file and independently requires every core Windows/Linux/Android/source/documentation/native-QA asset to exist and be non-empty before it can touch a version tag.
- Optional signed updater assets are all-or-nothing and are verified separately from the required core asset set.
- Publication fails if required artifacts/gates are missing.
- Existing tags are never moved.
- Public filenames use `GhostFTP`; product copy uses `Ghost FTP`.

## Stable / FINAL gates

Do not label an artifact FINAL until evidence exists for required installer lifecycle, target-OS visual acceptance, security failure paths, update/signing verification and accessibility acceptance.

Blocked gates are BLOCKED, not PASS.
