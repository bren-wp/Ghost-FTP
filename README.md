# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="112">
</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It provides a professional dual-pane workstation for **FTP, FTPS and SFTP**, local profiles, protected saved-secret handling, bounded transfer management and verified release packaging without application telemetry.

- Current Ghost FTP version: **1.1.8**
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

## 1.1.8 Stable

Ghost FTP 1.1.8 is a backward-compatible Windows/Linux maintenance release focused on security/privacy hardening, exact ownership/provenance checks and responsive multi-monitor stability while preserving the established protocol, profile and packaging contract.

### Privacy-safe diagnostics

- Raw `curl`, `ssh`, `sftp` and `ssh-keyscan` diagnostics are not exposed through generic user-facing errors.
- Child-process output is classified while bounded data is in scope; only privacy-safe semantic state required for user errors, retry decisions and protocol fallback is retained.
- FTP MLSD unsupported-command fallback remains functional without retaining raw server replies.
- SFTP host-key verification and pinning remain strict.

### Linux transport and AskPass provenance

- Linux transport discovery rejects user-controlled `PATH` shadowing and accepts `curl`, `ssh`, `sftp` and `ssh-keyscan` only after trusted root-controlled filesystem provenance checks.
- Credential-bearing AskPass requires trusted executable/parent provenance and fails closed when the helper boundary cannot be proven safe.
- `/proc/self/exe` is used only as an in-process identity oracle; it is not handed to OpenSSH as a credential helper path.

### Windows installation and cleanup ownership

- Settings/profile state remains bound to the originally validated state-directory identity.
- Windows installer transactions retain install-directory identity through backup, activation, rollback and cleanup.
- Desktop/Start Menu shortcuts are ownership-bound by digest and exact-handle verification; foreign or modified same-name shortcuts are preserved.
- The Start Menu parent directory is preserved because shortcut ownership does not imply ownership of its containing directory.
- Legacy and integrated uninstall cleanup require registry/executable ownership proof and exact-object validation rather than pathname/name alone.

### Windows responsive geometry

- Startup and minimum window geometry adapt to the active monitor work area.
- Negative multi-monitor origins and small/effective work areas are preserved safely.
- Mixed-DPI `WM_DPICHANGED` bounds are clamped against the destination monitor selected from the suggested rectangle.

### Security and privacy baseline

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, strict SFTP host-key verification/pinning, protected-secret lifetime rules, local root/path protections, staged transfer activation/rollback and connection-generation guards.

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

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained system-tool integration and trusted executable/AskPass provenance boundary.

### FTP — explicit compatibility

Plain FTP remains available only as an explicit compatibility choice for legacy servers that intentionally require unencrypted FTP.

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally; Ghost FTP does not send filenames, server names, credentials or UI text to an online translation service.

See [Localization](docs/LOCALIZATION.md).

## Windows installation

```text
Ghost-FTP-1.1.8-Setup-x64.exe
Ghost-FTP-1.1.8-Setup-x86.exe
Ghost-FTP-1.1.8-Setup-x32.exe
Ghost-FTP-1.1.8-Portable-x64.exe
Ghost-FTP-1.1.8-Portable-x86.exe
```

`x32` is a byte-identical compatibility alias of the verified x86 Setup build; it is not a separate architecture build. Production Authenticode is optional. When a trusted certificate is configured, signatures must verify. Otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical 1.1.8 Linux files are:

```text
Ghost-FTP-1.1.8-Linux-amd64.deb
Ghost-FTP-1.1.8-Linux-arm64.deb
Ghost-FTP-1.1.8-Linux-i386.deb
Ghost-FTP-1.1.8-Linux-multiarch.zip
Ghost-FTP-1.1.8-Linux-amd64.tar.gz
Ghost-FTP-1.1.8-Linux-arm64.tar.gz
Ghost-FTP-1.1.8-Linux-i386.tar.gz
```

DEB and RPM package metadata use the Ghost FTP product identity and `https://ghostftp.com`. Generic tar.gz archives are package-manager-neutral alternatives built from the same per-architecture executable as the matching DEB.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages remain verification artifacts and are not canonical release files unless the release allow-list is explicitly changed.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Releases and GitHub Packages

Ghost FTP 1.1.8 preserves the canonical **12 platform artifacts / 15 public files** release shape: five Windows files, seven Linux files, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The Stable distribution bundle is:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.8
```

The OCI object mirrors `/ghostftp-release/` from the verified release assembly and is **not a runtime container**.

Published `ghostftp-v1.1.7` and earlier releases remain immutable.

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
