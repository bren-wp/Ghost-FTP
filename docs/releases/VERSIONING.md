# Ghost FTP versioning

Ghost FTP uses semantic versioning with a pre-1.0 development train.

- Every meaningful development/release cycle changes the central version together with real code/product changes; there are no version-only commits.
- A new development/feature release advances the minor component: `0.14.0` → `0.15.0` → `0.16.0`.
- A hotfix to an already published release advances only the patch component: `0.15.0` → `0.15.1` → `0.15.2`.
- `1.0.0` is reserved for the first fully production-stable release.
- Root `version.json` is the single source of truth for the active product version and build metadata.
- CI rejects metadata drift and invalid version progression.
- Historical canonical versions are assigned only to releases that actually exist on GitHub. Missing legacy RC numbers do not consume a canonical `0.x.0` version.
- Legacy `2.1.1-rc.*` identifiers remain compatibility aliases for already-published assets and links.

The verified mapping is recorded in `docs/releases/version-map.json`.


## Published-history migration

The public GitHub release history was migrated and verified on 27 September 2026.

- Legacy published RC releases now use the canonical sequence `v0.1.0` through `v0.14.0`.
- The already published next release remains `v0.15.0`.
- Legacy RC release automation, approval markers and one-time migration tooling were removed after successful verification so they cannot publish or fail against the active 0.x train.
- Historical provenance remains recorded in `version-map.json`; migrated release assets retain their verified GitHub SHA-256 digests.
