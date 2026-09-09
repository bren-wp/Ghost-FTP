# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="96">
</p>

- **Current Ghost FTP release: 1.1.7**
- Development status: **Stable**
- GitHub Release policy: **prerelease=false**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- Product website: **https://ghostftp.com**

The root [`VERSION`](../VERSION) file is the authoritative production version source. This directory contains maintained engineering, operations, privacy, security, release and user documentation for Ghost FTP.

## Authentic visual reference

The documentation icon and screenshots are repository-local. They do not load a remote badge, tracking pixel, icon CDN, webfont or analytics resource when the Markdown is rendered.

All four UI captures below come from the real production Windows x64 Portable executable and are maintained by the authentic screenshot workflow. They are evidence of application-owned native windows, not design mockups.

### Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

### Site Manager

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

### Settings

![Ghost FTP Settings](images/ghost-ftp-settings.png)

### About

![Ghost FTP About](images/ghost-ftp-about.png)

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for dimensions, SHA-256 provenance and the rules that prevent generated or stale replacement imagery from being treated as release evidence.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — component, protocol, persistence, transfer and release boundaries.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — maintained native workstation interaction and authentic screenshot reference.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavior-parity contract.
- [`SETTINGS.md`](SETTINGS.md) — persisted settings, validation, normalization and recovery.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.

## Installation and distribution

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux installation/upgrade guidance.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical GitHub Release structure and release-channel rules.
- [`PACKAGES.md`](PACKAGES.md) — Stable GHCR distribution bundle.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — artifact, metadata, SHA-256 and signing-state verification.
- [`SIGNING.md`](SIGNING.md) — optional protected Authenticode signing and truthful unsigned-release policy.
- [`VERSIONING.md`](VERSIONING.md) — semantic versioning and stable/prerelease rules.

Ghost FTP 1.1.7 uses the canonical **12 platform artifacts / 15 public files** release shape: five Windows Setup/Portable files, three Linux DEBs, a Linux multiarch ZIP, three package-manager-neutral Linux tar.gz archives, release metadata, notes and `SHA256.txt`. The same verified release directory is mirrored to GitHub Packages as a non-runtime OCI distribution bundle.

Supplemental distro-specific CI packages built by `linux/BUILD-DISTROS.sh` cover Debian, Ubuntu, Fedora and a distro-neutral Portable family. Native lifecycle/GUI smoke verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. These supplemental packages are **not yet part of the canonical release allow-list** and therefore do not change the 1.1.7 public file count.

## Security and privacy

- [`SECURITY.md`](SECURITY.md) — transport security, secure defaults, SFTP trust, path validation, transfer staging, secret ownership and process boundaries.
- [`PRIVACY.md`](PRIVACY.md) — no-telemetry policy, local data handling, credential persistence and diagnostics redaction.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency/runtime-tool policy and offline Go module boundary.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — notices for platform/runtime tooling.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — Go race tests, loopback FTP regressions, native UI/runtime checks, distro lifecycle tests and repository audits.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution and release-quality expectations.
- [`ROADMAP.md`](ROADMAP.md) — maintenance priorities and product constraints.
- [`SUPPORT.md`](SUPPORT.md) — support and privacy-safe issue reporting.
- [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md) — production engineering, security, functional, UI/UX and release-audit handoff prompt.
- [`prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md`](prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md) — production build/redesign prompt for the official product website using the canonical Ghost FTP dark palette.

## Ghost FTP 1.1.7 contract

Ghost FTP 1.1.7 is a backward-compatible Windows/Linux Stable maintenance release. Key changes include:

- application-owned Windows Confirm/Info/Error DecisionCard surfaces using the same Light/Dark, DPI, owner-modality and keyboard contracts as other Ghost FTP modals;
- adaptive DecisionCard geometry for long localized/security text;
- runtime-localized OK/Cancel/Yes/No, profile privacy/security decisions and native SSH-key/folder pickers across the maintained 24-language contract;
- unchanged credential retain/remove/automatic-clear semantics;
- canonical Linux generic `.tar.gz` output for amd64, arm64 and i386 with executable byte-parity verification against matching DEBs;
- maintained supplemental Debian/Ubuntu/Fedora/Portable package construction and x86-64 native lifecycle/GUI-smoke gates;
- preserved rooted transfer/filesystem safeguards, FTPS/SFTP trust and rollback behavior;
- no telemetry, analytics, advertising, tracking, hidden product service or new external Go module dependency.

## Stable publication

The 1.1.7 stable identity is:

```text
ghostftp-v1.1.7
```

Windows artifacts:

```text
Ghost-FTP-1.1.7-Setup-x64.exe
Ghost-FTP-1.1.7-Setup-x86.exe
Ghost-FTP-1.1.7-Setup-x32.exe
Ghost-FTP-1.1.7-Portable-x64.exe
Ghost-FTP-1.1.7-Portable-x86.exe
```

Linux artifacts:

```text
Ghost-FTP-1.1.7-Linux-amd64.deb
Ghost-FTP-1.1.7-Linux-arm64.deb
Ghost-FTP-1.1.7-Linux-i386.deb
Ghost-FTP-1.1.7-Linux-multiarch.zip
Ghost-FTP-1.1.7-Linux-amd64.tar.gz
Ghost-FTP-1.1.7-Linux-arm64.tar.gz
Ghost-FTP-1.1.7-Linux-i386.tar.gz
```

GitHub Package:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

Production Authenticode remains optional. If a trusted certificate is configured, Windows signatures are verified fail-closed. Otherwise the release is explicitly unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`; a generated/self-signed identity is never represented as a trusted production publisher.

## Historical 1.1.6 contract

The immutable 1.1.6 release remains historical at `ghostftp-v1.1.6`. It used **9 platform artifacts / 12 public files** and did not include the three generic Linux tar.gz archives. 1.1.7 does not rewrite that tag, its assets, checksums, release notes or GHCR digest history.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — cumulative engineering narrative.
- [`../CHANGELOG.md`](../CHANGELOG.md) — public release change log used by release-note generation.

Historical version references describe their original release state and are not rewritten as current product behavior.

## Release verification rule

A release is complete only after the exact source revision passes Core, Windows and Linux gates and the immutable tag, GitHub Release asset set, `SHA256.txt`, metadata and Stable GitHub Package have all been read back successfully.

Canonical publication uses `release/ghostftp-vX.Y.Z` created only from the exact post-merge `main` SHA that passed the complete quality gate. A push to `main` does not itself publish a release.

## UI evidence rule

A public Windows UI/version change requires authentic screenshots from the real production Windows x64 Portable executable for Main Workspace, Site Manager, Settings and About. Mockups or generated approximations are not accepted as release evidence.

## Privacy-safe documentation rule

Documentation and build logs must never contain real passwords, private-key passphrases, protected profile payloads, signing private keys or private user data. Examples use synthetic values only. Rendered documentation media is repository-local; externally hosted images, badge images, tracking pixels, remote icon resources and remote webfont/image dependencies are not permitted in the active documentation surface.
