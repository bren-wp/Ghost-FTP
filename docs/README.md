# Ghost FTP documentation

- **Current Ghost FTP release: 1.1.6**
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

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux installation/upgrade guidance.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical GitHub Release structure and release-channel rules.
- [`PACKAGES.md`](PACKAGES.md) — stable GHCR distribution bundle.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — artifact, metadata, SHA-256 and signing-state verification.
- [`SIGNING.md`](SIGNING.md) — optional protected Authenticode signing and truthful unsigned-release policy.
- [`VERSIONING.md`](VERSIONING.md) — semantic versioning and stable/prerelease rules.

The **published 1.1.6** release contains **9 platform artifacts** and **12 public files**. The maintained source release workflow for the next version expects **12 platform artifacts / 15 public files**, adding verified Linux `.tar.gz` archives for amd64, arm64 and i386 while preserving the historical 1.1.6 asset set. The same verified release directory is mirrored to GitHub Packages as a non-runtime OCI distribution bundle.

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

## Ghost FTP 1.1.6 contract

Ghost FTP 1.1.6 is a backward-compatible Windows/Linux maintenance release focused on filesystem race hardening and stronger SFTP/remote-cleanup trust guarantees.

Key 1.1.6 changes:

- recursive local deletion traverses verified child directories through opened root handles rather than mutable pathnames;
- local `Mkdir` is anchored to an opened `os.Root`, preventing a late base-directory pathname swap from redirecting creation;
- SFTP SHA-256 host-key fingerprints are computed directly from the selected scanned key blob in memory;
- the key blob's embedded algorithm must match the algorithm declared by `ssh-keyscan`;
- fingerprint calculation no longer relies on a temporary key pathname reopened by `ssh-keygen`;
- remote cleanup treats text diagnostics as untrusted and does not accept `No such file`/`not found` text alone as proof that staging state is absent;
- successful delete or curl's structured `REMOTE_FILE_NOT_FOUND` result can confirm absence, while non-zero SFTP cleanup remains fail-closed;
- deterministic regression coverage proves the confirmed path-swap, fingerprint-binding and spoofed-cleanup cases;
- no telemetry, analytics, advertising, tracking, hidden product service or new external Go module dependency is introduced.

The release preserves existing FTPS certificate/hostname validation, SFTP host-key pinning, protected-secret lifetime rules, transfer generation/cancel/retry safeguards, root-bound local download activation, Site Manager secret/trust isolation and the Windows/Linux native desktop boundary.

## Stable publication

The published 1.1.6 release is a normal GitHub Release with `prerelease=false` and immutable tag:

```text
ghostftp-v1.1.6
```

Windows artifacts:

```text
Ghost-FTP-1.1.6-Setup-x64.exe
Ghost-FTP-1.1.6-Setup-x86.exe
Ghost-FTP-1.1.6-Setup-x32.exe
Ghost-FTP-1.1.6-Portable-x64.exe
Ghost-FTP-1.1.6-Portable-x86.exe
```

Linux artifacts:

```text
Ghost-FTP-1.1.6-Linux-amd64.deb
Ghost-FTP-1.1.6-Linux-arm64.deb
Ghost-FTP-1.1.6-Linux-i386.deb
Ghost-FTP-1.1.6-Linux-multiarch.zip
```

GitHub Package:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.6
```

Stable aliases `1.1`, `1` and `latest` were updated only after successful publication and registry read-back. The package contains `/ghostftp-release/` and is a distribution bundle, not a runtime container.

Production Authenticode remains optional. If a trusted certificate is configured, Windows signatures are verified fail-closed. Otherwise the release is explicitly unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`; a generated/self-signed identity is never represented as a trusted production publisher.

## Next-release Linux portable contract

The maintained release workflow now stages package-manager-neutral Linux tarballs for `amd64`, `arm64` and `i386` alongside the matching DEBs. Before a future version can publish, production release CI verifies archive structure and byte-for-byte parity between each portable `ghostftp` executable and `/usr/bin/ghostftp` from its DEB. The final future-version contract is 12 platform artifacts / 15 public files with exact remote asset read-back. These files are not retroactively added to 1.1.6.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative engineering narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public release change log used by release-note generation.

Historical version references describe their original release state and are not rewritten as current product behavior.

## Release verification rule

A release is complete only after the exact source revision passes Core, Windows and Linux gates and the immutable tag, GitHub Release asset set, `SHA256.txt`, metadata and stable GitHub Package have all been read back successfully.

For 1.1.6 the canonical publication branch was `release/ghostftp-v1.1.6`, created only from the exact post-merge `main` SHA that passed the complete quality gate. A later release must repeat this process independently; current source workflow readiness is not publication evidence.

## UI evidence rule

The 1.1.6 release-prep changed the public version displayed by the built application. Authentic screenshots therefore came from the real production Windows x64 Portable executable for Main Workspace, Site Manager, Settings and About and were visually reviewed on the exact final release-prep source revision.

Future public Windows UI/version changes remain subject to the maintained authentic exact-head evidence rule.

## Privacy-safe documentation rule

Documentation and build logs must never contain real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only.
