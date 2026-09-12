# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="132">
</p>

<p align="center"><strong>Your servers. Your files. No cloud middleman.</strong></p>

<p align="center">
  A native, privacy-first FTP/FTPS/SFTP client for Windows and Linux — built for people who want fast file workflows, clear control and strong security boundaries without telemetry or a mandatory account.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/Ghost-FTP/releases"><strong>Download Ghost FTP</strong></a> ·
  <a href="https://ghostftp.com"><strong>ghostftp.com</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Installation</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a>
</p>

**Ghost FTP** gives you a focused native workspace for everyday server work: connect, browse, transfer, edit remote text files, manage sites and keep large transfer queues under control. The desktop client supports **FTP, explicit FTPS and SFTP**, uses native Windows/Linux interfaces, keeps application telemetry disabled and does not depend on a Ghost FTP cloud account or hidden backend.

- Current Ghost FTP version: **0.0.5**
- Development status: **Active**
- Release channel: **Current**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Repository: https://github.com/bren-wp/Ghost-FTP
- Public release identity: `ghostftp-v0.0.5`, `prerelease=false`
- Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.5`
- Android source surface: installable development APK tied to root `VERSION`; not part of the public Windows/Linux release allow-list

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

The icon and UI images rendered by this README are **repository-local assets**. Maintained release UI evidence is captured from real native Windows, Linux and Android runtime surfaces through exact-head CI. Mockups and generated approximations are not accepted as production UI evidence. No remote badge, tracking pixel, icon CDN or webfont is required to render this README.

## Why Ghost FTP?

| What you need | What Ghost FTP gives you |
| --- | --- |
| **A focused file-transfer workspace** | Native Local/Remote panes, saved sites, bookmarks, queue controls and clear connection state. |
| **Secure modern protocols** | Explicit FTPS with certificate/hostname validation and SFTP with strict host-key verification/pinning. |
| **Remote changes without tool switching** | Built-in **Remote Edit** for supported text files with conflict detection and verified save/read-back. |
| **Control over busy transfer sessions** | Pause/resume/cancel/retry, queued Top/Up/Down/Bottom ordering, parallelism controls and upload/download bandwidth ceilings. |
| **Privacy by design** | No application analytics, advertising, fingerprinting, tracking pixels, automatic crash upload or hidden product backend. |
| **A client that stays understandable** | Native controls, bounded operations, explicit destructive actions and documented fail-closed behavior instead of silent fallback. |

Ghost FTP is designed for developers, administrators, hosting users, agencies and anyone who regularly moves files between a local machine and remote servers but does not want their connection workflow routed through an application-owned cloud service.

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

## What you can do today

### Connect your way

- **FTP** for explicit legacy compatibility.
- **FTPS** with certificate and hostname validation; a failed secure connection is never silently retried as plain FTP.
- **SFTP** with strict host-key verification/pinning plus password or private-key authentication on the maintained desktop platforms.
- Saved Site Manager profiles with explicit credential-persistence consent rather than hidden secret storage.
- Local and remote bookmarks plus profile start directories with account/session revalidation.

### Move files with real queue control

- Upload/download single files and recursive directory trees.
- Pause, resume, cancel, retry and clear finished transfers.
- Move queued work **Top / Up / Down / Bottom** without mutating running or terminal history.
- See real progress, transferred bytes, speed and ETA from transfer events.
- Configure independent upload/download bandwidth ceilings in binary KiB/s, including `0 = unlimited`.
- Keep configured bandwidth as a conservative aggregate directional ceiling across worker slots.
- Choose explicit conflict behavior: **Skip**, **Replace**, or **Replace + recovery backup**.
- Use staged activation/rollback rather than direct destructive overwrite where the maintained transfer path requires it.

### Edit remote text files in place

Ghost FTP includes a built-in **Remote Edit** workflow for supported regular text files. It provides bounded text handling, UTF-8/binary validation, LF/CRLF/CR preservation, mixed-line-ending rejection, SHA-256 revision/conflict detection, verified upload/read-back and remote permission preservation when trustworthy metadata is available.

No permanent third pane and no mandatory external editor process: open, edit, **Save / Reload / Close**, then return to the normal file workflow.

### Find and compare without turning the UI into a maze

- Non-destructive current-folder filtering over the already loaded snapshot.
- Shared deterministic sorting by Name, Type, Size and Modified, plus remote Permissions.
- Bounded recursive local/server search with cancellation and fresh-list navigation.
- Conservative local/server directory comparison with synchronized navigation only for safely proven paired ordinary directories.

## 0.0.5

Ghost FTP 0.0.5 is a reliability, lifecycle and distribution-quality release built on the 0.0.4 feature line.

- **Windows lifecycle hardening:** profile persistence and file mutations are guarded against duplicate/re-entrant actions; modal loops preserve application shutdown semantics; Remote Edit sessions are protected from parallel re-entry and stale async continuation paths.
- **Android connection lifecycle hardening:** an in-flight FTP/FTPS session is explicitly owned by its Activity instance, is aborted on destruction/recreation, and stale success/error callbacks cannot revive a destroyed UI/session.
- **Browser companion source:** maintained source packages for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox can hand supported `ftp://`, `ftps://` and `sftp://` links into the Ghost FTP workflow without telemetry, remote code, credential storage or broad browsing permissions.
- **Release documentation accuracy:** public release notes now describe the actual universal Windows + Debian/Ubuntu/Fedora/Portable Linux contract: **14 platform artifacts / 17 public files**.
- **Documentation refresh:** the public README and release guidance are rewritten around user value while retaining the same exact security, privacy and verification boundaries.

The browser extension source is an optional companion source surface and is **not** added to the 17-file desktop GitHub Release allow-list.

## Security and privacy are product features

Ghost FTP preserves FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, strict SFTP host-key verification/pinning, protected-secret lifetime rules, local root/path protections, staged transfer activation/rollback, trusted Linux transport/AskPass provenance and exact-object Windows cleanup where ownership must be proven.

Ghost FTP includes **no application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, mandatory Ghost FTP account or hidden profile synchronization**. Go telemetry is disabled in production CI and release workflows.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Architecture](docs/ARCHITECTURE.md).

## Settings that change real behavior

The Settings surfaces are backed by one validated configuration model rather than decorative UI state:

- **Parallel transfers:** 1–8, default 2.
- **Upload bandwidth:** 0–1,048,576 KiB/s, default 0 = unlimited.
- **Download bandwidth:** 0–1,048,576 KiB/s, default 0 = unlimited.
- **Connection timeout:** 5–60 seconds, default 15.
- **Automatic retries:** 0–3, default 0.
- **Retry delay:** 1–30 seconds, default 3.
- **Conflict policy:** skip / replace / replace with recovery backup.
- **Delete confirmation:** enabled by default.
- **Appearance:** maintained Windows and Linux Classic Light/Dark behavior.
- **Language:** 24 local languages, English fallback.

Bandwidth limits are aggregate directional ceilings. A transfer attempt snapshots its effective budget when it starts, so saving a new value does not mutate a running transport process; new attempts observe the saved policy. See [Settings](docs/SETTINGS.md).

## Languages

**English** is the canonical default and fallback language. Ghost FTP provides **24 languages** through one local catalog shared by the Windows and Linux frontends. Language selection and translation resolution happen locally.

See [Localization](docs/LOCALIZATION.md).

## Download Ghost FTP 0.0.5

### Windows

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Both public Windows files are self-contained x86-compatible universal bootstraps carrying verified native x64/x86 application payloads. The selected payload is chosen from Windows native architecture information, verified before execution and requires no runtime download. Setup retains the integrated uninstall path; Portable requires no installer registration.

Production Authenticode is optional: when a trusted certificate is configured, signatures must verify; otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned` truthfully.

### Linux

Canonical 0.0.5 Linux files are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

Matching Debian, Ubuntu, Fedora and Portable variants reuse one compiled production executable per architecture, and release CI compares extracted executable bytes. Native package-manager/runtime/GUI verification is maintained for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; additional architectures retain exact-head build, package-metadata, extraction and binary-parity coverage without an unsupported native-install claim.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Android development APK

Android is an active native development surface tied to root `VERSION`. CI builds and verifies an installable `Ghost-FTP-Android.apk` development artifact with Files, Sites, Bookmarks, Transfers, Settings and About, FTP and strict explicit FTPS, SAF-scoped local access and lifecycle-safe staged transfers.

Android SFTP remains hidden until strict native host-key identity verification has a maintained implementation. The development APK is not included in the public Windows/Linux 17-file release allow-list.

## Browser companion extensions

The `ekstenzije/` source tree contains optional companion extension packages for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox. Their job is deliberately narrow: recognize supported FTP-family links and route them into the Ghost FTP workflow. They do not require a Ghost FTP account, do not store FTP credentials, do not ship remote executable code and do not turn normal web browsing into an application telemetry source.

These companion packages are source surfaces and are not counted as desktop GitHub Release artifacts.

## Releases and verification

Ghost FTP 0.0.5 uses the canonical **14 platform artifacts / 17 public files** release shape: two universal Windows executables, twelve Linux packages/archives, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The public release identity is:

```text
ghostftp-v0.0.5
prerelease=false
```

The same verified release directory is published as a distribution-only GHCR bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The GHCR object is not a supported runtime container. The release workflow validates the exact 17-file allow-list immediately and again after a delay. After successful publication and remote verification, release retention removes superseded Ghost FTP releases/tags/branches/package versions and keeps the latest public version. Git history on `main` is not rewritten.

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` binds the version, tag, source commit, platform set, Windows signing state, language count and packaging shape to the verified release assembly.

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
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

The older `linux/BUILD.sh` generic DEB/portable path remains CI compatibility coverage, not the canonical 0.0.5 public Linux allow-list.

## Quality gates

The production CI/release path checks Go formatting, race tests, unit/integration tests, vet, dependency policy, privacy/security audits, platform parity, localization, release/version documentation contracts, universal Windows packaging, Linux architecture compatibility packaging, canonical Debian/Ubuntu/Fedora/Portable metadata and binary parity, distro lifecycle smoke tests, Android lint/APK contracts and authentic Windows/Linux/Android UI screenshots. Exact source identity, remote asset read-back, GHCR read-back and retention are fail-closed publication gates.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
