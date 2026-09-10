# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="128">
</p>

<p align="center"><strong>Fast, private and security-focused FTP/FTPS/SFTP for Windows and Linux.</strong></p>

<p align="center">
  Native desktop UI · Site Manager · Remote Edit · Safe transfer queue · Bandwidth controls · 24 local languages · No application telemetry
</p>

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**. It combines a focused dual-pane workspace with **FTP, FTPS and SFTP**, saved profiles, protected credential handling, bounded transfer management, bandwidth-aware transport controls and a built-in remote text editor. The design target is a modern professional transfer workstation: fewer ambiguous controls, safer defaults, clear state and strong failure handling without a mandatory product account or hidden cloud backend.

- Current Ghost FTP version: **0.0.3**
- Development status: **Active**
- Release channel: **Current**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Repository: https://github.com/bren-wp/Ghost-FTP
- Public release identity: `ghostftp-v0.0.3`, `prerelease=false`
- Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.3`

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are **repository-local assets**. The maintained screenshots are captured from the real production Windows application payload by the authentic-UI workflow; mockups and generated approximations are not accepted as production UI evidence. No remote badge, tracking pixel, icon CDN or webfont is required to render this README.

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
- Independent upload and download bandwidth ceilings expressed in binary KiB/s, with `0 = unlimited`.
- Conservative aggregate directional bandwidth allocation across configured worker slots rather than multiplying a configured limit by the number of simultaneous transfers.
- Real transport enforcement: curl `limit-rate` for FTP/FTPS and OpenSSH `sftp -l` for SFTP.
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

Ghost FTP is not developed by cloning another client screen-for-screen. New capabilities are accepted only when the engine, Windows UI, Linux UI, tests, privacy/security model and documentation agree on the behavior. Current improvement priorities include stronger queue control, large-directory responsiveness, bookmarks, verified resume, multi-session workflows and carefully bounded proxy/jump-host support. Features are not advertised as shipped until their complete runtime path is implemented and tested.

See the [Roadmap](docs/ROADMAP.md) for the maintained power-user plan.

## 0.0.3

Ghost FTP 0.0.3 advances the current public line with real bandwidth-aware upload/download controls and the completed packaging transition. Windows now exposes exactly two self-contained universal public executables while verified native x64/x86 payloads remain internal staging inputs. Linux publication uses canonical Debian, Ubuntu, Fedora and distro-neutral Portable families. The release keeps the established FTP/FTPS/SFTP trust model, Remote Edit safeguards, local path confinement and no-telemetry/privacy contract. Semantic major version `0` does not by itself mark this project release as a GitHub prerelease.

## Protocols

### FTPS — fresh default

A fresh connection uses explicit FTPS on port 21. TLS certificate and hostname validation remain enabled. Failed TLS negotiation is not silently converted to plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained trusted executable/AskPass boundary. The configured bandwidth ceiling is converted conservatively for OpenSSH `sftp -l` so rounding cannot exceed the scheduler budget.

### FTP — explicit compatibility

Plain FTP remains available only as an explicit compatibility choice for legacy servers that intentionally require unencrypted FTP.

## Settings that have runtime effect

The Settings surfaces are backed by one validated configuration model rather than decorative UI state:

- **Parallel transfers:** 1–8, default 2.
- **Upload bandwidth:** 0–1,048,576 KiB/s, default 0 = unlimited.
- **Download bandwidth:** 0–1,048,576 KiB/s, default 0 = unlimited.
- **Connection timeout:** 5–60 seconds, default 15.
- **Automatic retries:** 0–3, default 0.
- **Retry delay:** 1–30 seconds, default 3.
- **Conflict policy:** skip / replace / replace with recovery backup.
- **Delete confirmation:** enabled by default.
- **Appearance:** maintained Windows Classic Light/Dark behavior.
- **Language:** 24 local languages, English fallback.

Bandwidth limits are aggregate directional ceilings. A transfer attempt snapshots its effective budget when it starts, so saving a new value does not mutate a running transport process; new attempts observe the saved policy. Missing legacy settings are migrated to safe canonical defaults and explicit invalid values remain validation failures. See [Settings](docs/SETTINGS.md).

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally.

See [Localization](docs/LOCALIZATION.md).

## Windows installation

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Both public Windows files are self-contained x86-compatible universal bootstraps. The production build still creates and verifies native x64 and x86 Setup/Portable payloads internally. The public bootstrap selects the native payload from Windows system architecture information using `GetNativeSystemInfo`, verifies the staged payload before execution and performs no runtime download. Architecture-specific staging EXEs must not leak into the public artifact directory.

Setup retains the integrated uninstall path and does not require a permanent separate uninstaller binary. Portable requires no installer registration. Production Authenticode is optional: when a trusted certificate is configured, signatures must verify; otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux installation

Canonical 0.0.3 Linux files are built by `linux/BUILD-DISTROS.sh`:

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

Matching Debian, Ubuntu, Fedora and Portable variants reuse one compiled production executable per architecture, and release CI compares the extracted executable byte-for-byte. Native package-manager/runtime/GUI verification is maintained for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; additional architectures retain exact-head build, package-metadata, extraction and binary-parity coverage without an unsupported native-install claim.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

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

The GHCR object is not a supported runtime container. The release workflow validates the exact 17-file allow-list immediately and again after a delay. After successful publication and remote verification, the release-retention workflow removes older Ghost FTP releases, tags, superseded canonical release branches and obsolete package versions while retaining the current package. The release-branch trigger waits for the exact canonical release result and explicitly verifies the retention result. **Only the latest public Ghost FTP version is retained.** Git history on `main` is not rewritten.

See [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Versioning](docs/VERSIONING.md).

## Artifact verification

Every public release contains `SHA256.txt`. `BUILD-METADATA.txt` binds version, release tag, source commit, platform set, Windows signing state, language count and packaging shape to the verified release assembly.

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
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

The older `linux/BUILD.sh` generic DEB/portable path remains a CI compatibility build, not the canonical 0.0.3 public Linux allow-list.

## Quality gates

The production CI/release path checks Go formatting, race tests, unit/integration tests, vet, dependency policy, privacy/security audits, platform parity, localization, release/version documentation contracts, universal Windows packaging, Linux amd64/arm64/i386 compatibility packaging, canonical Debian/Ubuntu/Fedora/Portable metadata and binary parity, distro lifecycle smoke tests and authentic Windows UI screenshots. A green source test is not enough by itself for publication: exact source identity, remote asset read-back, GHCR read-back and retention are separately fail-closed.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
