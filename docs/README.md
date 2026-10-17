# Ghost FTP Documentation

Ghost FTP documentation is organized around the current canonical product state, stable operational guidance and historical provenance.

## Current state

- Active source/release cycle: **0.95.29 (in development, preview)**
- Latest canonical release: **0.95.28**
- Live publication state: GitHub Releases is authoritative
- Version source of truth: `/version.json`
- Production desktop source: `/ghostftp-desktop`
- Production Android source: `/android`

Current status documents use the canonical `0.x` version train. Historical release-candidate identifiers remain only in explicitly historical release records under `docs/releases/`; active audit documents describe the current source line.

All local README/documentation images are checked against the newest non-draft GitHub Release tag. A documentation image that is missing from or differs from that release fails CI.

## Complete platform manuals, current status and security review

- [Full product handbook and status rules](handbook/README.md)
- [Windows app — all source features, settings, cloud connectors and QA](handbook/WINDOWS.md)
- [Linux app — capabilities, packaging, keyring and protocol testing](handbook/LINUX.md)
- [macOS app — SwiftUI workspaces, actual protocol limits and Keychain](handbook/MACOS.md)
- [Android app — Kotlin workspaces, transfers, SAF and mobile privacy](handbook/ANDROID.md)
- [Cross-platform feature-by-feature status matrix](handbook/FEATURE_MATRIX.md)
- [Security/privacy architecture and threat model](handbook/SECURITY_PRIVACY.md)
- [Detailed implementation backlog, dependencies and release criteria](handbook/DELIVERY_ROADMAP.md)
- [Cloud storage provider and security scope](product/CLOUD_STORAGE.md)
- [Cloud storage provider compatibility and verified status criteria](cloud/PROVIDERS.md)
- [OAuth, object credentials and signed URL authentication](cloud/AUTHENTICATION.md)
- [Signed cloud URL threat model and mitigations](security/CLOUD_SIGNED_URL_THREAT_MODEL.md)
- [Release blockers and production readiness](status/PRODUCTION_READINESS.md)

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
- [Cloud live E2E test matrix and evidence requirements](qa/CLOUD_E2E.md)

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

## Product guides and release QA

- [0.94.0 Android transport and cancellation changes](releases/0.94.0.md)
- [Cloud signed URL threat model](security/CLOUD_SIGNED_URL_THREAT_MODEL.md)
- [Android DNS/cancellation/SAF unit and integration acceptance](qa/CLOUD_E2E.md)
- [Production readiness and explicit blockers](status/PRODUCTION_READINESS.md)
