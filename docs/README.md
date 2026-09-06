# Ghost FTP documentation

- **Current Ghost FTP release: 1.0.0**
- Development status: **Stable**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- GitHub Release state: **draft=false, prerelease=false**

This directory contains maintained engineering, operations, privacy, security, release and user documentation for Ghost FTP. The root [`VERSION`](../VERSION) file is the authoritative production version source.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component boundaries, protocol architecture, persistence, transfer and release design.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — workstation visual/interaction reference and authentic screenshot contract.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavior parity contract.
- [`SETTINGS.md`](SETTINGS.md) — persisted settings, normalization and recovery behavior.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language offline localization model.

## Installation and distribution

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux DEB installation/upgrade guidance.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — stable-only GitHub Release structure, tag and read-back rules.
- [`PACKAGES.md`](PACKAGES.md) — GHCR distribution bundle and fresh-pull verification.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — artifact, metadata, SHA-256, signing/trust and package verification.
- [`SIGNING.md`](SIGNING.md) — truthful Windows signed/unsigned trust model and protected Authenticode integration.
- [`VERSIONING.md`](VERSIONING.md) — semantic versioning and stable-only publication policy.

The production workflow publishes **9 platform artifacts** and **12 public files** on each canonical GitHub Release. The same verified release directory is mirrored to GitHub Packages as a non-runtime OCI distribution bundle only after Release verification succeeds.

## Security and privacy

- [`SECURITY.md`](SECURITY.md) — transport, SFTP trust, path validation, staged transfer, secret, process and release boundaries.
- [`PRIVACY.md`](PRIVACY.md) — no-telemetry policy, local data handling, diagnostics redaction and distribution privacy.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency/runtime-tool policy and offline Go build boundary.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — notices for platform/runtime tooling used by Ghost FTP.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — Go race tests, protocol regressions, UI/runtime checks, release/package read-back and Python repository audits.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution and release-quality expectations.
- [`ROADMAP.md`](ROADMAP.md) — post-1.0 maintenance priorities and product constraints.
- [`SUPPORT.md`](SUPPORT.md) — support information and safe issue-reporting guidance.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative historical engineering/release narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public version-by-version change log.

Historical sections intentionally preserve old version numbers and Beta/prerelease terminology. They are history, not the current publication policy.

## Stable 1.0 release contract

Ghost FTP 1.0.0 is the first stable release. Maintained publication requires:

```text
MAJOR >= 1
draft=false
prerelease=false
```

The production workflow rejects pre-1.0 publication, so it no longer emits new prereleases. Historical 0.x prereleases remain immutable provenance.

## Windows trust metadata

Every Release records the actual state, rather than assuming that a binary is signed:

```text
WINDOWS_AUTHENTICODE=signed|unsigned
WINDOWS_TRUST_MODE=...
```

A configured production Authenticode certificate must verify successfully. If no production certificate is configured, publication remains explicitly unsigned and uses SHA-256 + official GitHub Release/tag/source provenance. A development/self-signed certificate is never presented as a production publisher identity.

## GitHub Packages contract

The production workflow publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

plus compatible stable aliases. The package contains `/ghostftp-release/` and is a distribution bundle, not an application runtime container.

After push, the workflow removes its local image, pulls the semantic-version tag from GHCR and verifies OCI source/version/revision labels plus embedded `SHA256.txt` and `BUILD-METADATA.txt` against the verified Release assembly.

## Privacy-safe documentation rule

Documentation, examples and build logs must never include real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only. Connection diagnostics should describe categories/remediation without reproducing credentials.

## Source-of-truth hierarchy

When documentation and implementation appear to disagree, verify in this order:

1. current `VERSION` and source code;
2. security/privacy/release audit scripts;
3. CI and release workflow behavior;
4. current active documentation;
5. historical release notes.

A public release is complete only when the exact source revision passes all required gates and both GitHub Release and GitHub Package remote read-back succeed.
