# Ghost FTP Versioning

Current development edition: **0.99.16 Preview**. Previous published canonical release: **v0.99.15**..

Ghost FTP uses semantic versioning with a pre-1.0 development train.

## Rules

- Root `version.json` is the single source of truth for the active product version/build metadata.
- A meaningful new development/feature cycle may advance to a later SemVer line chosen for the planned scope. CI requires the new version to be strictly newer than the preceding source version and `previousVersion` to identify that exact source base.
- A hotfix to an already published release advances the patch component.
- Meaningful fixes may remain within an already-active unpublished development version; version-only commits are not required.
- Once a version tag is published, that tag is immutable and may not be retargeted to newer source.
- `1.0.0` is reserved for the first production-stable release.
- CI rejects metadata drift and invalid version progression.
- `previousPublishedVersion` independently records the last verified public GitHub Release. Publication requires GitHub's actual latest published release to match it; merged-but-unpublished versions must not block the release indefinitely or be misrepresented as published.
- Version synchronization also refreshes the committed canonical Cargo.lock so Cargo metadata and lock state move atomically.

## Required new version train (owner decision 21 October 2026)

- Latest verified public edition before development 0.99.4: **0.99.3** (Preview, 21 artifacts on GitHub Releases).
- Authorized history: 0.95.53 → 0.99.0 → 0.99.1 → 0.99.2 → 0.99.3. **Next edition in this sequence: 0.99.9**.
- Subsequent editions must advance consecutively: **0.99.4** through **0.99.20**. No skipped or repeated version number, no 0.99.21.
- After **0.99.20**, the next allowed version is **1.0.0**. Production signing, notarization, device installation and security acceptance must pass independently before changing the channel from Preview to stable.
- Previously published editions and all existing Git tags are permanent. **Never delete a release or tag during normal builds and releases.**
- Normal post-1.0 versioning uses sequential patches and minor feature releases; no immutable tag can be retargeted.

## Current cycle

- Active source/release cycle: **0.99.14**
- Previous canonical release: **0.99.13**
- Previous source version: **0.99.8** (verified prior published release).
- Historical versions **0.30.11** and **0.30.12** had merged-but-unpublished source transitions; their provenance remains in the release history.
- Live publication state: GitHub Releases is authoritative.

## Published-history migration

The public GitHub history was migrated/verified on 27 September 2026.

- Previously published legacy releases were mapped in order to canonical `v0.1.0` through `v0.14.0`.
- `v0.15.0` through `v0.19.0` are canonical releases in the new train.
- Legacy `2.1.1-rc.*` identifiers are historical aliases/provenance only.
- Their verified mapping is recorded in `version-map.json`.
- Historical release-note files with legacy identifiers are retained as archive snapshots and must not drive active build/release instructions.

## Release discovery

The canonical download entry point is [**Releases**](https://github.com/bren-wp/Ghost-FTP/releases). GitHub's `/releases/latest` endpoint excludes Preview packages and must **not** be used to find this project's newest prerelease. After each successful exact-SHA release, CI verifies its public tag, latest numeric SemVer, first-page discoverability, all required installers and SHA-256 digests. Older releases and tags are never deleted.

## Provenance

`version-map.json` records legacy-to-canonical provenance for migrated releases. Canonical GitHub Releases are the authoritative public asset/download history.
