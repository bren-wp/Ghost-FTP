# Ghost FTP documentation

- **Current Ghost FTP release: 1.1.3**
- Development status: **Stable**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**

This directory contains the maintained engineering, operations, privacy, security, release and user documentation for Ghost FTP. The root [`VERSION`](../VERSION) file is the authoritative production version source.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component boundaries, protocol architecture, persistence, transfer and release design.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — workstation visual/interaction reference, primary Classic Light palette and authentic screenshot contract.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavior parity contract.
- [`SETTINGS.md`](SETTINGS.md) — persisted settings, appearance policy, normalization and recovery behavior.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language offline localization model.

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
- [`PRIVACY.md`](PRIVACY.md) — no-telemetry policy, local data handling, credential-persistence consent, diagnostics redaction and distribution privacy.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency/runtime-tool policy and offline Go build boundary.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — notices for platform/runtime tooling used by Ghost FTP.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — Go race tests, real loopback FTP manager/protocol regressions, UI/runtime checks and Python repository audits.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution and release-quality expectations.
- [`ROADMAP.md`](ROADMAP.md) — post-1.0 maintenance priorities and product constraints.
- [`SUPPORT.md`](SUPPORT.md) — support information and safe issue-reporting guidance.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative release narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public version-by-version change log.

Historical sections intentionally preserve older version numbers and Beta terminology. They are history, not the current support state.

## Stable 1.1.3 release contract

Ghost FTP 1.1.3 is a backward-compatible Windows/Linux maintenance and security release. It strengthens local download containment, remote tree preparation, SFTP operand handling, Windows platform correctness and canonical release publication without expanding the supported protocol or platform surface.

The release preserves the selected local download root through transfer execution and uses Go `os.Root` for root-bound staging, activation and rollback. Randomized staging uses sentinel and file-identity checks before activation, late nested redirect/path-swap attempts are rejected, and `SkipExisting` is rechecked at commit time. Remote tree upload preparation validates that existing remote path components are directories and rejects symlink/non-directory components.

SFTP batch operands escape glob metacharacters and option-like edge cases. Windows local filename validation covers reserved DOS device names including superscript-number variants. Site Manager provides a safe Duplicate workflow that creates a new draft without silently copying saved secrets or SFTP host-key trust state. Windows uninstall registration no longer advertises a non-existent quiet uninstall path.

A stable 1.1.3 publication is a normal GitHub Release with `prerelease=false`. Windows Authenticode remains optional: when a trusted production certificate is configured the workflow signs and verifies Windows artifacts; otherwise the release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`. Linux packages are generated and metadata-verified for amd64, arm64 and i386.

The production workflow publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.3
```

with stable aliases `1.1`, `1` and `latest` only after successful registry publication and read-back. The package contains `/ghostftp-release/` and is a distribution bundle, not an application runtime container.

No telemetry, analytics, advertising, tracking, hidden network service or new external Go module dependency is introduced by 1.1.3. FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret ownership/lifetime rules, transfer generation/cancel/retry safeguards and filesystem containment protections remain part of the release contract.

## Connection verification rule

The regression suite distinguishes transport-level protocol tests from application-lifecycle tests. Real loopback FTP coverage exercises login/list/mkdir/upload/rename/download/delete at the adapter layer and also verifies `remote.Manager.Connect` publishes a session only after successful authentication and initial listing. Wrong credentials and explicit-FTPS-to-plaintext endpoints must not produce a connected state.

## UI evidence rule

Authentic screenshot evidence is required whenever a release changes Windows UI. The maintained screenshots in `docs/images/` come from the dedicated workflow that builds and launches the real Windows x64 Portable executable. The 1.1.3 release itself does not introduce a new visual layout beyond the already verified Site Manager Duplicate surface inherited from the post-1.1.2 hardening cycle.

## Privacy-safe documentation rule

Documentation and build logs must never include real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only.

## Source-of-truth hierarchy

When documentation and implementation appear to disagree, verify in this order:

1. current `VERSION` and source code;
2. security/privacy/release audit scripts;
3. CI and release workflow behavior;
4. current active documentation;
5. historical release notes.

A public release is complete only when the exact source revision passes all required gates and GitHub Release, tag and stable GitHub Package publication have been read back successfully.
