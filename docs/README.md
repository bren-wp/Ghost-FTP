# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="96">
</p>

- **Current Ghost FTP release: 0.0.1**
- Development status: **Beta**
- GitHub Release policy: **prerelease=true**
- Public version retention: **latest release only**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- Product website: **https://ghostftp.com**

The root [`VERSION`](../VERSION) file is the authoritative production version source. Active documentation describes the current 0.0.1 Beta line; superseded public release/tag identities are removed after a newer release is successfully published and verified.

## Authentic visual reference

Documentation media is repository-local. No remote badge image, tracking pixel, remote icon resource, remote webfont or analytics resource is required when these documents render.

### Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

### Site Manager

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

### Settings

![Ghost FTP Settings](images/ghost-ftp-settings.png)

### About

![Ghost FTP About](images/ghost-ftp-about.png)

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for authentic screenshot provenance and the rule that mockups/generated approximations are not production UI evidence.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — architecture and trust boundaries.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux parity contract.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — native desktop UI and authentic evidence.
- [`SETTINGS.md`](SETTINGS.md) — settings and persistence behavior.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.

## Installation and releases

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux installation.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical release shape and latest-only retention.
- [`PACKAGES.md`](PACKAGES.md) — package policy for Beta and future Stable releases.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — checksums, source identity and signing verification.
- [`SIGNING.md`](SIGNING.md) — optional production Authenticode signing.
- [`VERSIONING.md`](VERSIONING.md) — 0.0.x Beta version policy.

Ghost FTP 0.0.1 uses the canonical **12 platform artifacts / 15 public files** release shape. Pre-1.0 releases are Beta GitHub Releases; the Stable GHCR distribution bundle is reserved for Stable publication.

Supplemental distro-specific CI packages built by `linux/BUILD-DISTROS.sh` cover Debian, Ubuntu, Fedora and a distro-neutral Portable family. Native lifecycle/GUI smoke verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. These packages are **not yet part of the canonical release allow-list**.

## Current 0.0.1 capability contract

- FTP, explicit FTPS and SFTP through one typed Core engine.
- Built-in Remote Edit on Windows and Linux using the same bounded/revision-checked engine path.
- Save conflict detection, line-ending preservation, read-back verification and remote metadata refresh.
- Transfer queue pause/resume/cancel/retry behavior.
- Site Manager profiles and protected saved-secret handling.
- Strict SFTP host-key verification/pinning and no silent FTPS downgrade.
- Rooted local path/transfer protections.
- Trusted Linux transport/AskPass provenance.
- Ownership-bound Windows installer/uninstaller/shortcut cleanup.
- No telemetry, analytics, advertising, tracking or hidden product backend.

## Current publication identity

```text
VERSION=0.0.1
TAG=ghostftp-v0.0.1
CHANNEL=Beta
PRERELEASE=true
```

Windows:

```text
Ghost-FTP-0.0.1-Setup-x64.exe
Ghost-FTP-0.0.1-Setup-x86.exe
Ghost-FTP-0.0.1-Setup-x32.exe
Ghost-FTP-0.0.1-Portable-x64.exe
Ghost-FTP-0.0.1-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-0.0.1-Linux-amd64.deb
Ghost-FTP-0.0.1-Linux-arm64.deb
Ghost-FTP-0.0.1-Linux-i386.deb
Ghost-FTP-0.0.1-Linux-multiarch.zip
Ghost-FTP-0.0.1-Linux-amd64.tar.gz
Ghost-FTP-0.0.1-Linux-arm64.tar.gz
Ghost-FTP-0.0.1-Linux-i386.tar.gz
```

After the new release passes remote read-back verification, `.github/workflows/release-retention.yml` removes older Ghost FTP releases, tags, completed release branches and obsolete package versions. `main` commit history is not rewritten.

## Security and privacy

- [`SECURITY.md`](SECURITY.md)
- [`PRIVACY.md`](PRIVACY.md)
- [`DEPENDENCIES.md`](DEPENDENCIES.md)
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md)

## Quality and engineering

- [`TESTING.md`](TESTING.md)
- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`ROADMAP.md`](ROADMAP.md)
- [`SUPPORT.md`](SUPPORT.md)
- [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md)
- [`prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md`](prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md)

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — current public-line history.
- [`../CHANGELOG.md`](../CHANGELOG.md) — source for generated release notes.

Only the current public version is retained in active release history after retention cleanup succeeds.
