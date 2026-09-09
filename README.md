# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="112">
</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It provides a focused dual-pane workstation for **FTP, FTPS and SFTP**, saved profiles, protected credential handling, bounded transfer management and a built-in remote text editor without application telemetry.

- Current Ghost FTP version: **0.0.1**
- Development status: **Active**
- Release channel: **Current**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**

- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are repository-local assets. The README does not load remote badges, tracking pixels, icon CDNs or webfont resources.

## 0.0.1

Ghost FTP 0.0.1 starts the current public release line. It combines the maintained Windows/Linux desktop client, FTP/FTPS/SFTP engine, secure installer/uninstaller behavior, responsive desktop geometry and built-in Remote Edit workflow into one release identity. Semantic major version `0` does not by itself mark this project release as a GitHub prerelease.

### Remote Edit

A regular remote text file can be opened directly from the Remote pane and edited without launching an external editor. The shared engine used by Windows and Linux provides:

- bounded editing up to the maintained Remote Edit size limit;
- UTF-8/text validation and binary rejection;
- LF/CRLF/CR preservation and mixed-line-ending rejection;
- SHA-256 revision tokens and conflict detection;
- transaction-style upload with read-back verification;
- remote permission preservation when the server exposes a trustworthy mode;
- metadata refresh after a successful save while retaining the edited-file selection.

The UI remains intentionally compact: high-value actions are available without permanently adding extra panes or toolbars.

### Security and privacy baseline

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, strict SFTP host-key verification/pinning, protected-secret lifetime rules, local root/path protections, staged transfer activation/rollback, trusted Linux transport/AskPass provenance and exact-object Windows cleanup where ownership must be proven.

Ghost FTP includes no application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, mandatory product account or hidden profile synchronization. Go telemetry is disabled in production CI and release workflows.

See [Security](docs/SECURITY.md) and [Privacy](docs/PRIVACY.md).

## Desktop workflow

Ghost FTP provides:

- a **Local** file pane;
- a **Remote** server pane;
- **Site Manager** for saved profiles;
- built-in Remote Edit for supported text files;
- a transfer queue with pause/resume/cancel/retry lifecycle;
- connection diagnostics and status surfaces;
- keyboard-first navigation and file actions;
- language, appearance, transfer and connection settings.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume the same typed Core behavior.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Authentic UI evidence

The maintained screenshots below are produced from the real production Windows x64 Portable executable. Mockups, generated approximations and manually composed replacements are not accepted as production UI evidence.

### Settings

![Ghost FTP Settings](docs/images/ghost-ftp-settings.png)

### About

![Ghost FTP About](docs/images/ghost-ftp-about.png)

See [Reference UI](docs/REFERENCE-UI.md).

## Protocols

### FTPS — fresh default

A fresh connection uses explicit FTPS on port 21. TLS certificate and hostname validation remain enabled. Failed TLS negotiation is not silently converted to plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained trusted executable/AskPass boundary.

### FTP — explicit compatibility

Plain FTP remains available only as an explicit compatibility choice for legacy servers that intentionally require unencrypted FTP.

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally.

See [Localization](docs/LOCALIZATION.md).

## Windows installation

```text
Ghost-FTP-0.0.1-Setup-x64.exe
Ghost-FTP-0.0.1-Setup-x86.exe
Ghost-FTP-0.0.1-Setup-x32.exe
Ghost-FTP-0.0.1-Portable-x64.exe
Ghost-FTP-0.0.1-Portable-x86.exe
```

`x32` is a byte-identical compatibility alias of the verified x86 Setup build; it is not a separate architecture build. Production Authenticode is optional. When a trusted certificate is configured, signatures must verify. Otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical 0.0.1 Linux files are:

```text
Ghost-FTP-0.0.1-Linux-amd64.deb
Ghost-FTP-0.0.1-Linux-arm64.deb
Ghost-FTP-0.0.1-Linux-i386.deb
Ghost-FTP-0.0.1-Linux-multiarch.zip
Ghost-FTP-0.0.1-Linux-amd64.tar.gz
Ghost-FTP-0.0.1-Linux-arm64.tar.gz
Ghost-FTP-0.0.1-Linux-i386.tar.gz
```

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages remain verification artifacts and are not canonical release files.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Releases

Ghost FTP 0.0.1 uses the canonical **12 platform artifacts / 15 public files** release shape: five Windows files, seven Linux files, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The public release identity is:

```text
ghostftp-v0.0.1
prerelease=false
```

The same verified release directory is published as a distribution-only GHCR bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.1
```

The GHCR object is not a supported runtime container. After a new Ghost FTP release is successfully published and remotely verified, the release-retention workflow removes older Ghost FTP GitHub releases, tags, superseded release branches and obsolete package versions while retaining the current release package. **Only the latest public Ghost FTP version is retained.**

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

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
