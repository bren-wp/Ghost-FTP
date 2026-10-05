# Ghost FTP Documentation

Ghost FTP documentation is organized around the current canonical product state, stable operational guidance and historical provenance.

## Current state

- Active source/release cycle: **0.30.8**
- Previous canonical release: **0.30.7**
- Live publication state: GitHub Releases is authoritative
- Version source of truth: `/version.json`
- Production desktop source: `/ghostftp-desktop`
- Production Android source: `/android`

Current status documents use the canonical `0.x` version train. Historical release-candidate identifiers remain only in explicitly historical release records under `docs/releases/`; active audit documents describe the current source line.

All local README/documentation images are checked against the newest non-draft GitHub Release tag. A documentation image that is missing from or differs from that release fails CI.

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
- Files named `2.1.1-rc.*` under `docs/releases/` are historical snapshots only.
- `docs/audits/` contains point-in-time audit records whose labels describe the audited snapshot, not the active version.
