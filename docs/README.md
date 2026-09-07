# Ghost FTP documentation

- **Current Ghost FTP release: 1.1.4**
- Development status: **Stable**
- GitHub Release policy: **prerelease=false**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**

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

## Ghost FTP 1.1.4 contract

Ghost FTP 1.1.4 is a backward-compatible native Windows/Linux maintenance release focused on settings correctness, desktop responsiveness and local-only UI resources.

Key 1.1.4 changes:

- Windows language selection closes its dropdown immediately after selection;
- language persistence no longer blocks the native message loop;
- Language and Settings writes share one UI-side serialization gate to prevent stale whole-settings overwrites;
- Windows Settings consumes canonical backend limits/defaults instead of duplicated numeric ranges;
- invalid prompt state is normalized to safe defaults before display;
- non-language Settings saves avoid unnecessary localization/list/layout rebuilds;
- duplicate language layout refinement was removed;
- idle Windows transfer polling performs no selection-map, summary, action-state or list redraw work when there are no new transfer events;
- native icon registration is deduplicated and remains local, using Segoe Fluent Icons with Segoe MDL2 compatibility fallback;
- deterministic process-tree cancellation regression coverage replaces fixed timing assumptions;
- repository cleanup retains files only when they have an active build, platform, security, test, package or release role;
- no telemetry, analytics, advertising, tracking, remote icon runtime or new external Go module dependency is introduced.

The release preserves existing transport and filesystem protections: FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret lifetime rules, transfer generation/cancel/retry safeguards, root-bound local download activation and destructive-operation containment.

## Stable publication

A stable 1.1.4 publication is a normal GitHub Release with `prerelease=false` and immutable tag:

```text
ghostftp-v1.1.4
```

Windows artifacts:

```text
Ghost-FTP-1.1.4-Setup-x64.exe
Ghost-FTP-1.1.4-Setup-x86.exe
Ghost-FTP-1.1.4-Setup-x32.exe
Ghost-FTP-1.1.4-Portable-x64.exe
Ghost-FTP-1.1.4-Portable-x86.exe
```

Linux artifacts:

```text
Ghost-FTP-1.1.4-Linux-amd64.deb
Ghost-FTP-1.1.4-Linux-arm64.deb
Ghost-FTP-1.1.4-Linux-i386.deb
Ghost-FTP-1.1.4-Linux-multiarch.zip
```

GitHub Package:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.4
```

Stable aliases `1.1`, `1` and `latest` are updated only after successful publication and registry read-back. The package contains `/ghostftp-release/` and is a distribution bundle, not an application runtime container.

Production Authenticode remains optional. If a trusted certificate is configured, Windows signatures are verified fail-closed. Otherwise the release is explicitly unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`; a generated/self-signed identity is never represented as a trusted production publisher.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative engineering narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public release change log used by release-note generation.

Historical version references and Beta terminology inside historical documents describe their original release state; they are not the current support status.

## Release verification rule

A release is complete only after the exact source revision passes Core, Windows and Linux gates and the immutable tag, GitHub Release asset set, `SHA256.txt`, metadata and stable GitHub Package have all been read back successfully.

## UI evidence rule

When a release materially changes Windows layout or visual rendering, authentic screenshots must come from the real production Windows x64 Portable executable. Ghost FTP 1.1.4 changes interaction behavior and local icon plumbing but does not introduce a new application layout.

## Privacy-safe documentation rule

Documentation and build logs must never contain real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only.

## Source-of-truth hierarchy

When active documentation and implementation appear to disagree, verify in this order:

1. current `VERSION` and source code;
2. security/privacy/release audit scripts;
3. CI and release workflow behavior;
4. active documentation;
5. historical release notes.
