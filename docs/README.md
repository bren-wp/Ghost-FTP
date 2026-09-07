# Ghost FTP documentation

- **Current Ghost FTP release: 1.1.2**
- Development status: **Stable**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**

This directory contains the maintained engineering, operations, privacy, security, release and user documentation for Ghost FTP. The root [`VERSION`](../VERSION) file is the authoritative production version source.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component boundaries, protocol architecture, persistence, transfer and release design.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — workstation visual/interaction reference, primary Classic Light palette and authentic screenshot contract.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavior parity contract.
- [`SETTINGS.md`](SETTINGS.md) — persisted settings, primary-light appearance policy, normalization and recovery behavior.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language offline localization model, including credential-persistence consent.

## Installation and distribution

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux DEB installation/upgrade guidance.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical GitHub Release structure and release-channel rules.
- [`PACKAGES.md`](PACKAGES.md) — stable GHCR distribution bundle published through GitHub Packages.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — artifact, metadata, SHA-256, signing-state and package verification.
- [`SIGNING.md`](SIGNING.md) — optional protected Authenticode signing model and truthful unsigned-release policy.
- [`VERSIONING.md`](VERSIONING.md) — semantic versioning and stable/prerelease policy.

The stable workflow publishes **9 platform artifacts** and **12 public files** on each canonical GitHub Release. The same verified release directory is mirrored to GitHub Packages as a non-runtime OCI distribution bundle.

## Security and privacy

- [`SECURITY.md`](SECURITY.md) — transport, secure protocol defaults, SFTP trust, path validation, staged transfer, protected-secret ownership and process boundaries.
- [`PRIVACY.md`](PRIVACY.md) — no-telemetry policy, local data handling, explicit credential-persistence consent, diagnostics redaction and distribution privacy.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency/runtime-tool policy and offline Go build boundary.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — notices for platform/runtime tooling used by Ghost FTP.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — Go race tests, real loopback FTP manager/protocol regressions, UI/runtime checks, authentic screenshot verification and Python repository audits.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution and release-quality expectations.
- [`ROADMAP.md`](ROADMAP.md) — post-1.0 maintenance priorities and product constraints.
- [`SUPPORT.md`](SUPPORT.md) — support information and safe issue-reporting guidance.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative release narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public version-by-version change log.

Historical sections intentionally preserve older version numbers and Beta terminology. They are history, not the current support state.

## Stable 1.1.2 release contract

Ghost FTP 1.1.2 is a backward-compatible Windows/Linux native-UI, localization and release-evidence hardening release built on the 1.1.x security baseline. Windows application navigation is centralized in the canonical left sidebar, the duplicated native top menu is removed, and the operational FTP workspace retains only real connection/file/transfer actions.

Prompt, option/language/settings and About surfaces use the maintained application-owned Light/Dark-aware native dialog shell. Runtime About is a dedicated native card using public **Ghost FTP** naming, **BRENDIGO LTD** publisher metadata and only official Brendigo destinations. It does not use WebView, GitHub runtime links or hidden network requests. Its layout reserves enough room for longer localized headings and body content.

The canonical 24-language registry remains unchanged. Every supported language has valid FTPS explicit/implicit mode text and Site Manager/Diagnostics navigation coverage, with English as the fallback. Adaptive compact-button rendering prevents accidental clipped labels while preserving intentional icon-only secondary actions when space is genuinely constrained.

A stable 1.1.2 publication is a normal GitHub Release with `prerelease=false`. Windows Authenticode remains optional: when a trusted production certificate is configured the workflow signs and verifies Windows artifacts; otherwise the release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`. Linux packages are generated and metadata-verified for amd64, arm64 and i386.

The production workflow publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.2
```

with stable aliases `1.1`, `1` and `latest` only after successful registry publication and read-back. The package contains `/ghostftp-release/` and is a distribution bundle, not an application runtime container.

No telemetry, analytics, advertising, tracking, hidden network service or new external Go module dependency is introduced by 1.1.2. Existing FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret ownership/lifetime rules, transfer generation/cancel/retry safeguards, randomized staging/rollback, destination revalidation and filesystem containment protections remain part of the release contract.

## Connection verification rule

The maintained regression suite distinguishes transport-level protocol tests from application-lifecycle tests. Real loopback FTP coverage exercises login/list/mkdir/upload/rename/download/delete at the adapter layer and also verifies `remote.Manager.Connect` publishes a session only after successful authentication and initial listing. Wrong credentials and explicit-FTPS-to-plaintext endpoints must not produce a connected state. SFTP host-key and credential lifecycle behavior is covered separately by deterministic local tests; documentation must not claim an external live SFTP/FTPS service test when one was not performed.

## UI evidence rule

The maintained screenshots in `docs/images/` must come from the dedicated authentic screenshot workflow, which builds and launches the real Windows x64 Portable executable. Mockups or generated approximations are not valid production evidence. For the 1.1.2 line the required evidence set is Main Workspace, Site Manager, Settings and About. The final release-version commit must regenerate this evidence from the actual 1.1.2 executable, and all four images must be visually reviewed for clipping, overlap, accidental ellipsis, mixed Dark/Light surfaces, spacing, sidebar/header quality, About branding/links and the correct version label.

## Privacy-safe documentation rule

Documentation and build logs must never include real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only. Connection diagnostics should describe categories and remediation without reproducing credentials.

## Source-of-truth hierarchy

When documentation and implementation appear to disagree, verify in this order:

1. current `VERSION` and source code;
2. security/privacy/release audit scripts;
3. CI and release workflow behavior;
4. current active documentation;
5. historical release notes.

A public release is complete only when the exact source revision passes all required gates and GitHub Release, tag and stable GitHub Package publication have been read back successfully.
