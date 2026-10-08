# Ghost FTP Versioning

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

## Current cycle

- Active source/release cycle: **0.30.13**
- Previous canonical release: **0.30.10**
- Previous source version: **0.30.12** (merged, not published).
- Versions **0.30.11** and **0.30.12** were merged but never published. The next public release must carry their real changes with new, tested fixes.
- Live publication state: GitHub Releases is authoritative.

## Published-history migration

The public GitHub history was migrated/verified on 27 September 2026.

- Previously published legacy releases were mapped in order to canonical `v0.1.0` through `v0.14.0`.
- `v0.15.0` through `v0.19.0` are canonical releases in the new train.
- Legacy `2.1.1-rc.*` identifiers are historical aliases/provenance only.
- Their verified mapping is recorded in `version-map.json`.
- Historical release-note files with legacy identifiers are retained as archive snapshots and must not drive active build/release instructions.

## Provenance

`version-map.json` records legacy-to-canonical provenance for migrated releases. Canonical GitHub Releases are the authoritative public asset/download history.
