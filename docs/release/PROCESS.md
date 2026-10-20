# Ghost FTP Release Process

Ghost FTP uses a canonical pre-1.0 semantic-version release train. Root `version.json` is the source of truth.

A successful compile is not equivalent to stable/FINAL acceptance.

The current distribution pipeline **only authorizes `preview`**. Until Windows Authenticode, Android Play signing/data-preserving upgrade and macOS Developer ID notarization are mandatory verified release gates, any `stable` channel must fail closed.

Every releasable source version **must use an explicit `preview` or `stable` channel** in `version.json`. The source CI verifies this before merge; the release workflow independently enforces it. Until actual Android 16-screen pixel-parity and physical-device acceptance are complete, use `preview`, not `stable`.

## Development/release flow

1. Start from current `main`.
2. Make meaningful product/code/documentation changes.
3. Keep `version.json` and synchronized metadata consistent; do not create version-only commits.
4. Keep `previousVersion` pointing to the preceding merged source (currently `0.95.53`) and `previousPublishedVersion` pointing to the latest verified public GitHub Release (currently `0.95.53`). Never call an unpublished merge a canonical release.
5. Open a PR only after the preceding `main` version has been publicly released; the PR must set `previousVersion` and `previousPublishedVersion` to the base version. This prevents overlapping merges from skipping an edition.
6. Require exact-head success for:
   - Ghost FTP quality
   - Ghost FTP protocol E2E
   - Ghost FTP native build
   - Ghost FTP Android
   - Validate Windows hardening
   - Ghost FTP macOS (development Preview gate)
7. Read and fix concrete workflow logs if any gate fails.
8. Merge only the tested source.
9. After any merge to `main` (including Android-only, macOS-only or documentation changes), the native build runs without path filtering; if successful, it automatically triggers **Ghost FTP release**, provided every exact-SHA gate passes. No manual dispatch is required.
10. The release job waits for the remaining exact-SHA gates and requires the actual latest public release to match `previousPublishedVersion` pointing to the latest verified public GitHub Release (`0.95.53`). Never call an unpublished merge a canonical release.
11. Package normalized Windows/Linux/Android/source/documentation/native-QA assets and SHA-256 checksums from the verified source SHA.
12. Publish `v<version>` at that exact source SHA and verify uploaded asset digests/count.
13. Never retarget an existing version tag to different source.
14. Every published release and tag is retained indefinitely unless the owner gives a new, explicit deletion instruction. Automatic release/tag pruning is prohibited.

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

If no updater signatures are present, automatically triggered Preview releases may publish labeled development packages, checksums and QA evidence after the exact-SHA gates. In that mode no updater manifest or Update-Service bundle is emitted.

### Source/documentation

- `GhostFTP-v<version>-Source.zip`
- `GhostFTP-v<version>-Desktop-Source.zip`
- `GhostFTP-v<version>-Android-Source.zip`
- `GhostFTP-v<version>-macOS-Source.zip`
- `GhostFTP-v<version>-Updates.zip`
- `GhostFTP-v<version>-Documentation.zip`
- `GhostFTP-v<version>-SHA256SUMS.txt`

## Permanent release and tag preservation

The previous one-time latest-ten cleanup has ended. **No scheduled or post-release job may delete GitHub Releases or tags.** New editions are added alongside all retained historical editions. The train is `0.99.0`, `0.99.1`, …, `0.99.20`, then `1.0.0` subject to production signing and QA acceptance.

## Current release cycle

- Active source/release cycle: **0.99.0**
- Previous canonical release: **0.95.53**
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

## Automatic Preview promotion (0.95.52 onward)

1. Every merged change requires a **new version** in `version.json`, synchronized desktop/Android/macOS/Go metadata and matching release notes. CI rejects unchanged versions on PRs.
2. Merging the tested PR to `main` runs all platform CI workflows. A successful native build automatically starts release promotion (not PR or fork builds).
3. Release promotion checks that the triggering build matches the latest native build and current `main` SHA, then waits for the five remaining exact-SHA gates.
4. Assets are staged in a draft release. Upload, SHA-256, remote digests, asset inventory and source SHA must all verify before publication as **Preview prerelease**.
5. Failed promotion stays unpublished; public release assets and existing tags cannot be overwritten. A failed check must be fixed with a new tested version.
6. Stable promotion remains disabled until mandatory signing, notarization and full target-platform QA are proven.
