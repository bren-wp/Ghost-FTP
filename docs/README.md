# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="108">
</p>

<p align="center"><strong>Production behavior, security boundaries, UI evidence and release engineering for Ghost FTP.</strong></p>

- **Current Ghost FTP release: 0.0.1**
- Development status: **Active**
- Release channel: **Current**
- GitHub Release policy: **PRERELEASE=false**
- Public version retention: **latest release only**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- Product website: **https://ghostftp.com**

The root [`VERSION`](../VERSION) file is the authoritative production version source. Active documentation describes the current 0.0.1 public line. Superseded release identities are removed only after the successor has passed source gates, publication, remote read-back and the canonical retention workflow.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Understand all current settings | [`SETTINGS.md`](SETTINGS.md) |
| Verify UI behavior and screenshots | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Understand security boundaries | [`SECURITY.md`](SECURITY.md) |
| Understand privacy/no-telemetry behavior | [`PRIVACY.md`](PRIVACY.md) |
| Understand architecture/Core ownership | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Verify Windows/Linux parity | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Validate a downloaded release | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| Understand release/version lifecycle | [`VERSIONING.md`](VERSIONING.md) and [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| See next power-user work | [`ROADMAP.md`](ROADMAP.md) |
| Build/test/contribute | [`TESTING.md`](TESTING.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md) |

## Authentic visual reference

Documentation media is **repository-local**. No remote badge image, tracking pixel, remote icon resource, remote webfont or analytics resource is required when these documents render. Production screenshots come from the real Windows x64 Portable build and are checked by the authentic-UI workflow.

<table>
<tr>
<td width="50%" valign="top"><strong>Main Workspace</strong><br><br><img src="images/ghost-ftp-main-workspace.png" alt="Ghost FTP Main Workspace"></td>
<td width="50%" valign="top"><strong>Site Manager</strong><br><br><img src="images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
</tr>
<tr>
<td width="50%" valign="top"><strong>Settings</strong><br><br><img src="images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
<td width="50%" valign="top"><strong>About</strong><br><br><img src="images/ghost-ftp-about.png" alt="Ghost FTP About"></td>
</tr>
</table>

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for provenance and the rule that mockups/generated approximations are not production UI evidence.

## Current 0.0.1 capability contract

- FTP, explicit FTPS and SFTP through one typed Core engine.
- Native Windows and Linux frontends consuming the same typed engine behavior.
- Built-in Remote Edit with bounded text validation, revision/conflict protection and verified read-back.
- Transfer queue pause/resume/cancel/retry and truthful progress/speed/ETA.
- Explicit conflict policy with safe staged activation and rollback behavior.
- Site Manager profiles with protected saved-secret handling and connection identity binding.
- Strict SFTP host-key verification/pinning and no silent FTPS downgrade.
- Rooted local path/transfer protections and remote cleanup uncertainty reporting.
- Trusted Linux transport/AskPass provenance.
- Ownership-bound Windows installer/uninstaller/shortcut cleanup.
- Validated local settings for concurrency, timeout, retry, overwrite policy, delete confirmation, appearance and language.
- No telemetry, analytics, advertising, tracking or hidden product backend.

## Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — components, ownership and trust boundaries.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux parity contract.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — native desktop UI and authentic evidence.
- [`SETTINGS.md`](SETTINGS.md) — validated settings and persistence behavior.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency and external-tool policy.

## Security and privacy

- [`SECURITY.md`](SECURITY.md) — protocol, filesystem, installer and release trust boundaries.
- [`PRIVACY.md`](PRIVACY.md) — local-first data handling and no-telemetry contract.
- [`SIGNING.md`](SIGNING.md) — optional production Authenticode and truthful unsigned state.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — third-party notices.

## Installation and releases

- [`INSTALLATION.md`](INSTALLATION.md) — Windows Setup/Portable and Linux installation.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical release shape and deterministic latest-only retention lifecycle.
- [`PACKAGES.md`](PACKAGES.md) — verified GitHub Packages distribution bundle policy.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — checksums, source identity and signing verification.
- [`VERSIONING.md`](VERSIONING.md) — controlled 0.0.x public version policy.

Ghost FTP 0.0.1 uses the canonical **12 platform artifacts / 15 public files** release shape and is published as the current GitHub Release with `prerelease=false`. The same verified release directory is mirrored as a distribution-only bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.1
```

Supplemental distro-specific CI packages built by `linux/BUILD-DISTROS.sh` cover Debian, Ubuntu, Fedora and a distro-neutral Portable family. Native lifecycle/GUI smoke verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. These packages are **not yet part of the canonical release allow-list**.

## Current publication identity

```text
VERSION=0.0.1
TAG=ghostftp-v0.0.1
CHANNEL=Current
PRERELEASE=false
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

After publication and remote read-back succeed, `.github/workflows/release-retention.yml` verifies the current tag/release/15-file set and removes superseded Ghost FTP releases, tags, canonical release branches and package versions. The release-branch trigger additionally waits for both the canonical publish run and canonical retention run to succeed. `main` commit history is not rewritten.

## Quality and engineering

- [`TESTING.md`](TESTING.md) — exact-head CI, package lifecycle and regression suites.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — engineering rules and PR expectations.
- [`ROADMAP.md`](ROADMAP.md) — prioritized next capabilities and acceptance criteria.
- [`SUPPORT.md`](SUPPORT.md) — support information and diagnostic expectations.
- [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md) — engineering audit prompt.
- [`prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md`](prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md) — website theme prompt.

## Release history

- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — current public-line history.
- [`../CHANGELOG.md`](../CHANGELOG.md) — source for generated release notes.

Only the current public version is retained in active release history after retention cleanup succeeds. Git history remains the engineering provenance of earlier work.
