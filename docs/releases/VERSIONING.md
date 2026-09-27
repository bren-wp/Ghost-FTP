# Ghost FTP Versioning

Ghost FTP uses semantic versioning with a pre-1.0 development train.

## Rules

- Root `version.json` is the single source of truth for the active product version/build metadata.
- A meaningful new development/feature cycle advances the minor component: `0.15.0 → 0.16.0 → 0.17.0`.
- A hotfix to an already published release advances the patch component.
- Meaningful fixes may remain within the already-active development version; version-only commits are not required.
- `1.0.0` is reserved for the first production-stable release.
- CI rejects metadata drift and invalid version progression.
- Publication is sequential: the declared previous canonical release must exist before the next release is published.

## Current state

- Active source: **0.17.0 development**
- Previous canonical version: **0.16.0**
- Latest published GitHub release: **0.16.0**
- `0.17.0` is not published until `v0.17.0` exists and release asset verification succeeds.

## Published-history migration

The public GitHub history was migrated/verified on 27 September 2026.

- Previously published legacy releases were mapped in order to canonical `v0.1.0` through `v0.14.0`.
- `v0.15.0` and `v0.16.0` are canonical releases in the new train.
- Legacy `2.1.1-rc.*` identifiers are historical aliases/provenance only.
- Their verified mapping is recorded in `version-map.json`.
- Historical release-note files with legacy identifiers are retained as archive snapshots and must not drive active build/release instructions.
- One-time migration/backfill workflows are removed after verification and must not remain part of the active release surface.

## Provenance

`version-map.json` records the legacy-to-canonical mapping and original source provenance for migrated releases. Canonical GitHub Releases are the authoritative public asset/download history.
