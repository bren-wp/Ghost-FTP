# Ghost FTP Documentation

Ghost FTP documentation is organized around the current canonical product state, stable operational guidance and historical provenance.

## Current state

- Active source version: **0.17.0 development preview**
- Latest published release: **0.16.0**
- Version source of truth: `/version.json`
- Production desktop source: `/ghostftp-desktop`
- Production Android source: `/android`
- Canonical downloads: GitHub Releases

Current status documents must use the canonical `0.x` version train. Historical release-candidate identifiers are retained only in explicitly historical release/audit records.

## Product

- [Features](product/FEATURES.md)
- [Project status & recommended next work](product/STATUS_AND_NEXT.md)
- [Roadmap](ROADMAP.md)
- [Detailed product backlog](product/ROADMAP.md)
- [UI / UX principles](product/UI_UX.md)

## Development and release

- [Building from source](development/BUILDING.md)
- [Build status](build/STATUS.md)
- [Release process](release/PROCESS.md)
- [Versioning](releases/VERSIONING.md)
- [QA evidence index](qa/README.md)

## Guides

- [Installation](guides/INSTALLATION.md)
- [Uninstall](guides/UNINSTALL.md)
- [Updates](guides/UPDATES.md)
- [Support](guides/SUPPORT.md)

## Legal

- [Privacy](legal/PRIVACY.md)
- [Third-party notices](legal/THIRD_PARTY_NOTICES.md)
- Commercial licence/EULA: `LICENSE.txt` and `EULA.txt`
- Security reporting: `SECURITY.md`

## Releases and historical records

- Canonical release notes live in [releases](releases/).
- [Version mapping/provenance](releases/version-map.json) records the verified legacy-to-canonical migration.
- Files named `2.1.1-rc.*` under `docs/releases/` are historical snapshots only and must not be used as current build/release instructions.
- `docs/audits/` contains point-in-time audit records. Their version labels describe the audited snapshot, not the active product version.

## Repository areas

- `ghostftp-desktop/` — production desktop application.
- `android/` — production Android application.
- `website/` — public Ghost FTP website.
- `updates/` — update channel contract.
- `tools/` — developer/runtime/installer support tooling; not the authoritative production desktop GUI.
- `docs/` — product documentation and historical records.

Framework-specific internal names remain unchanged only where renaming them would create build compatibility risk.
