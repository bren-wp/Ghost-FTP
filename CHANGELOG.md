# Changelog

## 0.18.0 — development — 27 September 2026

0.18.0 is a meaningful development cycle after published 0.17.0.

- Added CI provenance validation that keeps README/documentation images byte-identical to the newest published Ghost FTP release tag.
- Consolidated duplicate Windows/Linux Tauri build paths into one canonical production native build.
- Kept Windows native-window screenshot QA in the canonical build and added its evidence archive to release packaging.
- Removed the obsolete duplicate native build workflow and stale RC-era workflow references.
- Removed hard-coded `v2.1.1-Preview` source package naming from the active build path.
- Fixed version synchronization so ignored release-note paths no longer break the bot commit.
- Made version synchronization refresh and commit the canonical Cargo.lock atomically with Cargo metadata changes.
- Kept exact-SHA quality, real protocol E2E, Android and Windows hardening around the canonical native build.
- Continued the no-version-only policy.

See [docs/releases/0.18.0.md](docs/releases/0.18.0.md).

## 0.17.0 — published — 27 September 2026

- Completed the canonical public-history migration.
- Hardened Windows background helper processes against unintended console flashes.
- Added Windows helper-process regression checks.
- Made Android validation run on every pull request to `main`.
- Allowed meaningful fixes to remain within the active development version without forcing a version-only bump.
- Published normalized Windows, Linux, Android, source, documentation and SHA-256 assets.

See [docs/releases/0.17.0.md](docs/releases/0.17.0.md).

## Older canonical history

Canonical release notes for `0.1.0` through `0.16.0` are retained under `docs/releases/`.

The older `2.1.1-rc.*` identifiers are historical aliases only. Their verified mapping/provenance is recorded in `docs/releases/version-map.json`; they are not the active version scheme.
