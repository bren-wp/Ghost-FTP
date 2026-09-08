# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="112">
</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It provides a professional dual-pane workstation for **FTP, FTPS and SFTP**, local profiles, protected saved-secret handling, bounded transfer management and verified release packaging without application telemetry.

- Current Ghost FTP version: **1.1.7**
- Development status: **Stable**
- Release channel: **Stable**
- First stable release: **Ghost FTP 1.0.0**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**

- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Packages: https://github.com/users/bren-wp/packages?repo_name=Ghost-FTP
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are repository-local assets. The README does not load remote badges, tracking pixels, icon CDNs or webfont resources.

## 1.1.7 Stable

Ghost FTP 1.1.7 is a backward-compatible Windows/Linux maintenance release focused on native UI consistency, localization, packaging quality and security-preserving stability improvements.

### Windows UI and UX

- Application-owned Confirm, Info and Error DecisionCard windows follow the same Light/Dark palette as the main workspace.
- Native modal surfaces share DPI-aware sizing, owner modality and one keyboard/message-loop contract.
- DecisionCard heading/body geometry expands for longer localized and security-sensitive text instead of clipping it.
- OK, Cancel, Yes and No resolve from the live desktop locale when a dialog opens.
- Save Profile privacy/security decisions are localized across all 24 canonical languages without changing credential-retain/remove semantics.
- Native SSH private-key and local-folder pickers resolve titles and filters from the active runtime language.
- Closing a helper dialog closes only that dialog; only the main window may end the application message loop.
- Authentic Windows x64 Portable screenshots remain the release-evidence source for Main Workspace, Site Manager, Settings and About.

### Linux packaging and verification

The canonical 1.1.7 release adds package-manager-neutral Linux portable archives for `amd64`, `arm64` and `i386` alongside matching DEBs. Production verifies archive structure and byte-for-byte executable parity between each portable archive and `/usr/bin/ghostftp` from its corresponding DEB.

The canonical public release shape is **12 platform artifacts / 15 public files**.

Supplemental distro-specific CI packages built by `linux/BUILD-DISTROS.sh` cover Debian, Ubuntu, Fedora and a distro-neutral Portable family. Native package lifecycle/GUI smoke verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. These distro-labelled CI artifacts are not canonical 1.1.7 release assets.

### Security and privacy

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, SFTP host-key verification/pinning, protected-secret ownership/lifetime rules, path containment, staged transfer activation/rollback, connection-generation guards and privacy-safe diagnostics.

Ghost FTP includes no application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, mandatory product account or hidden profile synchronization. Go telemetry is disabled in production CI and release workflows.

See [Security](docs/SECURITY.md) and [Privacy](docs/PRIVACY.md).

## Desktop workflow

Ghost FTP provides:

- a **Local** file pane;
- a **Remote** server pane;
- **Site Manager** for saved profiles;
- a transfer queue with pause/resume/cancel/retry lifecycle;
- connection diagnostics and status surfaces;
- keyboard-first navigation and file actions;
- language, appearance, transfer and connection settings.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume the same typed Core behavior.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Authentic UI evidence

The maintained screenshots below are produced by `.github/workflows/ui-screenshots.yml` from the real production Windows x64 Portable executable. The workflow disables Go telemetry, builds production packages, captures native windows, rejects invalid/degenerate PNG output and records SHA-256 evidence before repository persistence.

### Settings

![Ghost FTP Settings](docs/images/ghost-ftp-settings.png)

### About

![Ghost FTP About](docs/images/ghost-ftp-about.png)

Mockups, generated approximations and manually composed replacement screenshots are not accepted as production UI evidence. See [Reference UI](docs/REFERENCE-UI.md) for the exact provenance contract.

## Appearance

Classic Light is the fresh/fallback appearance and uses restrained neutral surfaces rather than a pure-white workspace. An explicitly saved Dark preference is preserved. Windows icons remain local to the operating system: Segoe Fluent Icons is preferred where available with Segoe MDL2 Assets as compatibility fallback.

## Protocols

### FTPS — fresh default

A fresh connection uses explicit FTPS on port 21. TLS certificate and hostname validation remain enabled. Failed TLS negotiation is not silently converted to plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained system-tool integration.

### FTP — explicit compatibility

Plain FTP remains available only as an explicit compatibility choice for legacy servers that intentionally require unencrypted FTP.

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally; Ghost FTP does not send filenames, server names, credentials or UI text to an online translation service.

Changing language at runtime updates the maintained Windows UI surfaces and native dialogs. Linux uses the same canonical registry and fallback normalization. Missing or invalid locale state safely resolves to English.

See [Localization](docs/LOCALIZATION.md).

## Windows installation

```text
Ghost-FTP-1.1.7-Setup-x64.exe
Ghost-FTP-1.1.7-Setup-x86.exe
Ghost-FTP-1.1.7-Setup-x32.exe
Ghost-FTP-1.1.7-Portable-x64.exe
Ghost-FTP-1.1.7-Portable-x86.exe
```

`x32` is a byte-identical compatibility alias of the verified x86 Setup build; it is not a separate architecture build. Production Authenticode is optional. When a trusted certificate is configured, signatures must verify. Otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical 1.1.7 Linux files are:

```text
Ghost-FTP-1.1.7-Linux-amd64.deb
Ghost-FTP-1.1.7-Linux-arm64.deb
Ghost-FTP-1.1.7-Linux-i386.deb
Ghost-FTP-1.1.7-Linux-multiarch.zip
Ghost-FTP-1.1.7-Linux-amd64.tar.gz
Ghost-FTP-1.1.7-Linux-arm64.tar.gz
Ghost-FTP-1.1.7-Linux-i386.tar.gz
```

DEB and RPM package metadata use the Ghost FTP product identity and `https://ghostftp.com`. Generic tar.gz archives are package-manager-neutral alternatives built from the same per-architecture executable as the matching DEB.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Releases and GitHub Packages

Ghost FTP 1.1.7 publishes **12 platform artifacts / 15 public files**: five Windows files, seven Linux files, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The Stable distribution bundle is:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

The OCI object mirrors `/ghostftp-release/` from the verified release assembly and is **not a runtime container**.

See [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Artifact verification

Every public release contains `SHA256.txt`. `BUILD-METADATA.txt` binds version, release tag, source commit, platform set, Windows signing state, language count and package reference to the verified release assembly.

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

Supplemental distro-specific CI packages:

```bash
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

Official public artifacts are produced only by the canonical release workflow after exact-source audits, tests and builds succeed.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

Historical release text remains historical and is not rewritten to imply that older immutable tags contained later changes.

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
