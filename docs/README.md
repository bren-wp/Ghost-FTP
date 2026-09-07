# Ghost FTP documentation

- **Current Ghost FTP release: 1.1.5**
- Development status: **Stable**
- GitHub Release policy: **prerelease=false**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- Product website: **https://ghostftp.com**
- Developer/publisher: **BRENDIGO LTD — https://brendigo.com**

The root [`VERSION`](../VERSION) file is the authoritative production version source. This directory contains maintained engineering, operations, privacy, security, release and user documentation for Ghost FTP.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component, protocol, persistence, transfer and release boundaries.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — maintained native workstation interaction and screenshot reference.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavior-parity contract.
- [`SETTINGS.md`](SETTINGS.md) — persisted settings, validation, normalization and recovery.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.

## Installation and distribution

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux DEB installation/upgrade guidance.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical GitHub Release structure and release-channel rules.
- [`PACKAGES.md`](PACKAGES.md) — stable GHCR distribution bundle.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — artifact, metadata, SHA-256 and signing-state verification.
- [`SIGNING.md`](SIGNING.md) — optional protected Authenticode signing and truthful unsigned-release policy.
- [`VERSIONING.md`](VERSIONING.md) — semantic versioning and stable/prerelease rules.

The stable workflow publishes **9 platform artifacts** and **12 public files** for each canonical release. The same verified release directory is mirrored to GitHub Packages as a non-runtime OCI distribution bundle.

## Security and privacy

- [`SECURITY.md`](SECURITY.md) — transport security, secure defaults, SFTP trust, path validation, transfer staging, secret ownership and process boundaries.
- [`PRIVACY.md`](PRIVACY.md) — no-telemetry policy, local data handling, credential persistence and diagnostics redaction.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency/runtime-tool policy and offline Go module boundary.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — notices for platform/runtime tooling.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — Go race tests, loopback FTP regressions, native UI/runtime checks and repository audits.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution and release-quality expectations.
- [`ROADMAP.md`](ROADMAP.md) — maintenance priorities and product constraints.
- [`SUPPORT.md`](SUPPORT.md) — support and privacy-safe issue reporting.

## Ghost FTP 1.1.5 contract

Ghost FTP 1.1.5 is a backward-compatible native Windows/Linux maintenance release focused on product/publisher identity correctness, public UI branding consistency, release-documentation integrity and repository cleanup.

Key 1.1.5 changes:

- `ghostftp.com` is the official product website;
- `brendigo.com` is the official author/publisher website and BRENDIGO LTD remains the publisher identity;
- Windows About separates the product destination from the publisher destination;
- public localized UI text returns **Ghost FTP** instead of leaking the internal `GhostFTP` technical identifier;
- the public branding contract is regression-tested across all 24 supported languages;
- Linux DEB metadata uses `Homepage: https://ghostftp.com` and BRENDIGO LTD Maintainer identity;
- stale 1.1.1/1.1.4 current-release documentation in Installation, Packages, Support and release docs is corrected;
- version-drift regression coverage now requires active distribution/support documentation to follow root `VERSION`;
- a stale version-specific desktop-quality test file was removed while its maintained regression coverage stays in the evergreen suite;
- no telemetry, analytics, advertising, tracking, hidden product service or new external Go module dependency is introduced.

The release preserves existing transport and filesystem protections: FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret lifetime rules, transfer generation/cancel/retry safeguards, root-bound local download activation and destructive-operation containment.

## Stable publication

A stable 1.1.5 publication is a normal GitHub Release with `prerelease=false` and immutable tag:

```text
ghostftp-v1.1.5
```

Windows artifacts:

```text
Ghost-FTP-1.1.5-Setup-x64.exe
Ghost-FTP-1.1.5-Setup-x86.exe
Ghost-FTP-1.1.5-Setup-x32.exe
Ghost-FTP-1.1.5-Portable-x64.exe
Ghost-FTP-1.1.5-Portable-x86.exe
```

Linux artifacts:

```text
Ghost-FTP-1.1.5-Linux-amd64.deb
Ghost-FTP-1.1.5-Linux-arm64.deb
Ghost-FTP-1.1.5-Linux-i386.deb
Ghost-FTP-1.1.5-Linux-multiarch.zip
```

GitHub Package:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.5
```

Stable aliases `1.1`, `1` and `latest` are updated only after successful publication and registry read-back. The package contains `/ghostftp-release/` and is a distribution bundle, not an application runtime container.

Production Authenticode remains optional. If a trusted certificate is configured, Windows signatures are verified fail-closed. Otherwise the release is explicitly unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`; a generated/self-signed identity is never represented as a trusted production publisher.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative engineering narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public release change log used by release-note generation.

Historical version references describe their original release state and are not rewritten as current product behavior.

## Release verification rule

A release is complete only after the exact source revision passes Core, Windows and Linux gates and the immutable tag, GitHub Release asset set, `SHA256.txt`, metadata and stable GitHub Package have all been read back successfully.

For 1.1.5 the canonical publication branch is `release/ghostftp-v1.1.5`, created only from the exact post-merge `main` SHA that passed the complete quality gate.

## UI evidence rule

Because 1.1.5 changes public Settings/About branding and version presentation, authentic screenshots must come from the real production Windows x64 Portable executable for Main Workspace, Site Manager, Settings and About and must be visually reviewed on the exact release-prep source revision.

## Privacy-safe documentation rule

Documentation and build logs must never contain real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only.
