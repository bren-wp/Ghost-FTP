# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="128">
</p>

<p align="center"><strong>Fast, private and security-focused FTP/FTPS/SFTP for Windows and Linux.</strong></p>

<p align="center">
  Native desktop UI · Site Manager · Remote Edit · Safe transfer queue · 24 local languages · No application telemetry
</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It combines a focused dual-pane workspace with **FTP, FTPS and SFTP**, saved profiles, protected credential handling, bounded transfer management and a built-in remote text editor. The design target is a modern professional transfer workstation: fewer ambiguous controls, safer defaults, clear state and strong failure handling without a mandatory product account or hidden cloud backend.

- Current Ghost FTP version: **0.0.2**
- Development status: **Active**
- Release channel: **Current**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are **repository-local assets**. The screenshots are captured from the production Windows x64 Portable build by the maintained authentic-UI workflow; mockups and generated approximations are not accepted as production UI evidence. No remote badge, tracking pixel, icon CDN or webfont is required to render this README.

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

See [Reference UI](docs/REFERENCE-UI.md) for screenshot provenance and the production visual contract.

## What Ghost FTP already does

### Transfer workstation

- Local and Remote panes with independent navigation and sorting.
- Upload/download queue with bounded concurrency.
- Pause, resume, cancel, retry and clear-finished lifecycle.
- Progress, transferred bytes, speed and ETA based on real transfer events.
- Recursive directory transfer through the shared engine.
- Non-destructive current-folder filtering and bounded recursive local/server search.
- Conservative local/server directory comparison with synchronized navigation for safely proven paired directories.
- Explicit conflict policy: **Skip**, **Replace**, or **Replace + recovery backup**.
- Safe staged activation/rollback rather than direct destructive overwrite.
- Automatic retry only for errors classified as retryable; trust, permission, validation and unsafe-path failures do not become blind retry loops.

### Server and profile workflow

- FTP, explicit FTPS and SFTP.
- Site Manager with saved profiles.
- Password, SFTP private-key and passphrase workflows.
- Per-save credential persistence decision instead of hidden automatic secret storage.
- Connection diagnostics with bounded user-safe error reporting.
- Strict connection-generation/identity binding so queued work and synchronized comparison state cannot silently migrate to a different server session.

### Remote Edit

A regular remote text file can be opened directly from the Remote pane and edited without launching an external editor. The shared Windows/Linux engine provides:

- bounded editing up to the maintained Remote Edit size limit;
- UTF-8/text validation and binary rejection;
- LF/CRLF/CR preservation and mixed-line-ending rejection;
- SHA-256 revision tokens and conflict detection;
- transaction-style upload with read-back verification;
- remote permission preservation when the server exposes a trustworthy mode;
- metadata refresh after a successful save while retaining the edited-file selection.

### Security and privacy baseline

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, strict SFTP host-key verification/pinning, protected-secret lifetime rules, local root/path protections, staged transfer activation/rollback, trusted Linux transport/AskPass provenance and exact-object Windows cleanup where ownership must be proven.

Ghost FTP includes **no application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, mandatory Ghost FTP account or hidden profile synchronization**. Go telemetry is disabled in production CI and release workflows.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Architecture](docs/ARCHITECTURE.md).

## Design goals beyond legacy FTP clients

Ghost FTP is not developed by cloning another client screen-for-screen. New capabilities are accepted only when the engine, Windows UI, Linux UI, tests, privacy/security model and documentation agree on the behavior. Current improvement priorities include stronger queue control, large-directory responsiveness, navigation productivity and bandwidth-aware transfer controls. Features are not advertised as shipped until their complete runtime path is implemented and tested.

See the [Roadmap](docs/ROADMAP.md) for the maintained power-user plan.

## 0.0.2

Ghost FTP 0.0.2 advances the current public line with post-0.0.1 reliability hardening, non-destructive current-folder filtering, bounded recursive local/server search and conservative directory comparison with synchronized navigation on Windows and Linux. The release keeps the existing FTP/FTPS/SFTP trust model, Remote Edit safeguards, local path confinement and no-telemetry/privacy contract. Semantic major version `0` does not by itself mark this project release as a GitHub prerelease.

## Protocols

### FTPS — fresh default

A fresh connection uses explicit FTPS on port 21. TLS certificate and hostname validation remain enabled. Failed TLS negotiation is not silently converted to plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained trusted executable/AskPass boundary.

### FTP — explicit compatibility

Plain FTP remains available only as an explicit compatibility choice for legacy servers that intentionally require unencrypted FTP.

## Settings that have runtime effect

The Settings surfaces are backed by one validated configuration model rather than decorative UI state:

- **Parallel transfers:** 1–8, default 2.
- **Connection timeout:** 5–60 seconds, default 15.
- **Automatic retries:** 0–3, default 0.
- **Retry delay:** 1–30 seconds, default 3.
- **Conflict policy:** skip / replace / replace with recovery backup.
- **Delete confirmation:** enabled by default.
- **Appearance:** maintained Windows Classic Light/Dark behavior.
- **Language:** 24 local languages, English fallback.

Missing legacy settings are migrated to safe canonical defaults; explicit invalid values remain validation failures. See [Settings](docs/SETTINGS.md).

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally.

See [Localization](docs/LOCALIZATION.md).

## Windows installation

```text
Ghost-FTP-0.0.2-Setup-x64.exe
Ghost-FTP-0.0.2-Setup-x86.exe
Ghost-FTP-0.0.2-Setup-x32.exe
Ghost-FTP-0.0.2-Portable-x64.exe
Ghost-FTP-0.0.2-Portable-x86.exe
```

`x32` is a byte-identical compatibility alias of the verified x86 Setup build; it is not a separate architecture build. Production Authenticode is optional. When a trusted certificate is configured, signatures must verify. Otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical 0.0.2 Linux files are:

```text
Ghost-FTP-0.0.2-Linux-amd64.deb
Ghost-FTP-0.0.2-Linux-arm64.deb
Ghost-FTP-0.0.2-Linux-i386.deb
Ghost-FTP-0.0.2-Linux-multiarch.zip
Ghost-FTP-0.0.2-Linux-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-i386.tar.gz
```

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages remain verification artifacts and are not canonical release files.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Releases

Ghost FTP 0.0.2 uses the canonical **12 platform artifacts / 15 public files** release shape: five Windows files, seven Linux files, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The public release identity is:

```text
ghostftp-v0.0.2
prerelease=false
```

The same verified release directory is published as a distribution-only GHCR bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.2
```

The GHCR object is not a supported runtime container. After a new Ghost FTP release is successfully published and remotely verified, the release-retention workflow removes older Ghost FTP GitHub releases, tags, superseded release branches and obsolete package versions while retaining the current release package. The release-branch trigger also waits for the canonical release result and explicitly verifies the canonical retention result, so the latest-only lifecycle is not dependent on a single downstream event notification. **Only the latest public Ghost FTP version is retained.**

See [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Versioning](docs/VERSIONING.md).

## Artifact verification

Every public release contains `SHA256.txt`. `BUILD-METADATA.txt` binds version, release tag, source commit, platform set, Windows signing state and language count to the verified release assembly.

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

Canonical Linux packages:

```bash
bash linux/BUILD.sh
```

## Quality gates

The production CI/release path checks Go formatting, race tests, unit/integration tests, vet, dependency policy, privacy/security audits, platform parity, localization, release/version documentation contracts, Windows x64/x86 packaging, Linux amd64/arm64/i386 packaging, distro lifecycle smoke tests and authentic Windows UI screenshots. A green source test is not enough by itself for publication: release identity, remote asset read-back and retention are separately fail-closed.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
