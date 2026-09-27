# Changelog

## 0.17.0 — development — 27 September 2026

0.17.0 is the active development cycle. It is not published as a GitHub Release yet.

- Completed and verified the canonical public-history migration while keeping `version.json` as the active version source of truth.
- Removed reliance on legacy release-candidate naming from the active development/release model.
- Hardened Windows background helper processes so Ghost FTP does not flash unintended CMD/PowerShell windows during normal GUI operations.
- Added Windows regression checks for hidden helper process creation and shell-free PATH detection.
- Made the Android platform gate run on every pull request to `main`, so Windows/Linux/Android validation can refer to the same exact PR HEAD.
- Updated version progression checks so meaningful fixes can remain within the active development version instead of requiring a version-only bump.
- Kept the committed Cargo dependency graph and exact-SHA quality/protocol/platform gates mandatory.
- Synchronized active documentation with the canonical `0.x` train and current repository/workflow layout.

See [docs/releases/0.17.0.md](docs/releases/0.17.0.md).

## 0.16.0 — published — 26 September 2026

0.16.0 is the latest published GitHub release at the start of the 0.17.0 development cycle.

See [docs/releases/0.16.0.md](docs/releases/0.16.0.md) and the canonical GitHub Releases page for the published asset set and checksums.

## Older canonical history

Canonical release notes for `0.1.0` through `0.15.0` are retained under `docs/releases/`.

The older `2.1.1-rc.*` identifiers are historical aliases only. Their verified mapping/provenance is recorded in `docs/releases/version-map.json`; they are not the active version scheme.
