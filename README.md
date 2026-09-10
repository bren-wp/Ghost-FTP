# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="128">
</p>

<p align="center"><strong>Fast, private and security-focused FTP/FTPS/SFTP for Windows and Linux.</strong></p>

<p align="center">Native desktop UI · Site Manager · Remote Edit · Safe transfer queue · Bandwidth controls · 24 local languages · No application telemetry</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It provides a focused dual-pane workspace for **FTP, FTPS and SFTP**, saved profiles, protected credential handling, bounded transfer management, bandwidth-aware uploads/downloads and a built-in remote text editor without a mandatory product account or hidden cloud backend.

- Current Ghost FTP version: **0.0.3**
- Development status: **Active**
- Release channel: **Current**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Public release identity: `ghostftp-v0.0.3`, `prerelease=false`
- Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.3`

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are **repository-local assets**. Authentic screenshots are captured from the verified production Windows native payload by the maintained UI workflow; mockups and generated approximations are not accepted as production UI evidence.

## Product surfaces

<table>
<tr>
<td width="50%" valign="top"><strong>Site Manager</strong><br><br><img src="docs/images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
<td width="50%" valign="top"><strong>Settings</strong><br><br><img src="docs/images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
</tr>
<tr>
<td colspan="2" align="center"><strong>About / product identity</strong><br><br><img src="docs/images/ghost-ftp-about.png" alt="Ghost FTP About" width="620"></td>
</tr>
</table>

See [Reference UI](docs/REFERENCE-UI.md) for screenshot provenance.

## What Ghost FTP does

### Transfer workstation

- Independent local and remote panes with navigation and sorting.
- Upload/download queue with bounded concurrency, pause, resume, cancel, retry and clear-finished lifecycle.
- Truthful progress, transferred bytes, speed and ETA from real transfer events.
- Independent upload/download bandwidth ceilings in KiB/s, with `0 = unlimited`.
- Aggregate directional bandwidth budgeting across configured workers rather than multiplying the configured ceiling per transfer.
- Recursive directory transfer, non-destructive current-folder filtering, bounded recursive search and conservative directory comparison.
- Explicit conflict policy: **Skip**, **Replace**, or **Replace + recovery backup**.
- Safe staged activation/rollback rather than direct destructive overwrite.

### Protocol and profile workflow

- FTP, explicit FTPS and SFTP.
- Site Manager with saved profiles and per-save credential-persistence consent.
- Password, SFTP private-key and passphrase workflows.
- Strict connection-generation/identity binding so queued work cannot silently migrate to a different server session.

### Remote Edit

Supported remote text files can be opened from the Remote pane and edited with bounded UTF-8/text validation, size limits, line-ending preservation, SHA-256 revision tokens, conflict detection, transactional upload, read-back verification and permission preservation when trustworthy metadata is available.

### Security and privacy baseline

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, strict SFTP host-key verification/pinning, protected-secret lifetime rules, local root/path protections, trusted Linux transport/AskPass provenance and exact-object Windows cleanup where ownership must be proven.

Ghost FTP includes **no application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, mandatory Ghost FTP account or hidden profile synchronization**. Go telemetry is disabled in production CI and release workflows.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Architecture](docs/ARCHITECTURE.md) and [Settings](docs/SETTINGS.md).

## 0.0.3

Ghost FTP 0.0.3 adds bandwidth-aware upload/download controls and completes the public packaging transition. Windows now exposes exactly two universal public executables while retaining verified x64/x86 payloads internally. Linux publication is distro-specific across Debian, Ubuntu and Fedora plus a distro-neutral Portable family. The release contract is **14 platform artifacts / 17 public files**.

## Windows installation

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Both public files are self-contained universal x86/x64 launchers. The build embeds verified native x64 and x86 payloads, detects the native Windows architecture through system information, verifies staged payload bytes and performs no runtime download. Setup retains the integrated uninstall path; Portable requires no installer registration.

Production Authenticode is optional. When a trusted production certificate is configured, signatures must verify; otherwise `BUILD-METADATA.txt` truthfully records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical Linux 0.0.3 files are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.3-Linux-Debian-amd64.deb
Ghost-FTP-0.0.3-Linux-Debian-arm64.deb
Ghost-FTP-0.0.3-Linux-Debian-i386.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.3-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-i386.tar.gz
```

For each architecture the matching package variants reuse the same production `ghostftp` executable and CI verifies byte parity. Native install/remove/GUI lifecycle coverage is maintained for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; additional architectures retain build, metadata, extraction and parity coverage.

See [Installation](docs/INSTALLATION.md) and [Linux documentation](linux/README.md).

## Releases

Ghost FTP 0.0.3 uses the canonical **14 platform artifacts / 17 public files** release shape: two universal Windows executables, twelve Linux packages/archives, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The public release identity is:

```text
ghostftp-v0.0.3
prerelease=false
```

The same verified release directory is published as a distribution-only GHCR bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.3
```

Only the latest public Ghost FTP version is retained after successful publication and remote verification. Release/tag history is never rewritten in place.

See [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Versioning](docs/VERSIONING.md).

## Build from source

Ghost FTP uses Go **1.27.1**. Root `VERSION` is the production version source of truth.

```text
go telemetry off
go test -race ./...
go vet ./...
```

Windows release-style packages:

```powershell
.\BUILD-WINDOWS.ps1
```

Canonical Linux release packages:

```bash
bash linux/BUILD-DISTROS.sh
```

## Quality gates

The production path checks formatting, race tests, unit/integration tests, vet, dependency policy, privacy/security audits, platform parity, localization, release/version documentation contracts, universal Windows packaging, Linux package metadata/binary parity, Debian/Ubuntu/Fedora lifecycle smoke and authentic Windows UI evidence. Publication additionally requires exact-main release identity, remote release asset read-back and latest-only retention verification.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md) and [Support](docs/SUPPORT.md).

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
