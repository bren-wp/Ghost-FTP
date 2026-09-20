<div align="center">

<img src="ghostftp-desktop/branding/ghostftp-logo.svg" alt="Ghost FTP" width="430">

### More Than Transfer. Total Control.

**A modern, privacy-first FTP / FTPS / SFTP desktop client for Windows and Linux.**

[Official website](https://ghostftp.com/) ·
[RC9 release](https://github.com/bren-wp/Ghost-FTP-Premium/releases/tag/v2.1.1-rc.9) ·
[Documentation](docs/README.md) ·
[Security](SECURITY.md) ·
[Changelog](CHANGELOG.md)

</div>

<p align="center">
  <img src="docs/assets/screenshots/ghostftp-main-file-manager.webp" alt="Ghost FTP main file manager" width="100%">
</p>

## Ghost FTP

Ghost FTP is built for developers, agencies, hosting teams and power users who want **fast file transfer without a legacy-looking interface**.

It combines a dual-pane file manager, saved server profiles, secure FTP/FTPS/SFTP connections, transfer queues, synchronization, terminal tools, checksums, permissions, diagnostics and server-management workflows in one consistent desktop product.

The production desktop source lives in **`ghostftp-desktop/`**. Public binaries, archives, documentation and technical identifiers use **Ghost FTP / GhostFTP** naming. Framework-specific names are kept only where the underlying build ecosystem requires them.

## Why Ghost FTP

<table>
<tr>
<td width="50"><img src="ghostftp-web/assets/icons/security.svg" width="30" alt=""></td>
<td><strong>Secure connections</strong><br>FTP, FTPS and SFTP with native TLS/SSH verification paths and protected credential storage where supported.</td>
</tr>
<tr>
<td><img src="ghostftp-web/assets/icons/speed.svg" width="30" alt=""></td>
<td><strong>Fast transfers</strong><br>Concurrent queues, pause/resume, retries, bandwidth controls, conflict handling and clear transfer states.</td>
</tr>
<tr>
<td><img src="ghostftp-web/assets/icons/features.svg" width="30" alt=""></td>
<td><strong>Modern workflow</strong><br>Dual local/remote panes, Site Manager, bookmarks, tags, permissions, checksums, duplicate tools, sync and terminal features.</td>
</tr>
<tr>
<td><img src="ghostftp-web/assets/icons/privacy.svg" width="30" alt=""></td>
<td><strong>Privacy first</strong><br>No required analytics or telemetry. Sensitive profile secrets stay outside ordinary profile JSON.</td>
</tr>
<tr>
<td><img src="ghostftp-web/assets/icons/globe.svg" width="30" alt=""></td>
<td><strong>14 languages</strong><br>English, Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip.</td>
</tr>
<tr>
<td><img src="ghostftp-web/assets/icons/download.svg" width="30" alt=""></td>
<td><strong>Windows + Linux</strong><br>Windows portable EXE and Setup plus Linux native binary, AppImage, DEB and RPM packages.</td>
</tr>
</table>

## One visual system

Ghost FTP is designed as one product across the Main File Manager, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About surfaces.

<p align="center">
  <img src="docs/assets/screenshots/ghostftp-site-manager.webp" alt="Ghost FTP Site Manager" width="49%">
  <img src="docs/assets/screenshots/ghostftp-new-connection.webp" alt="Ghost FTP New Connection" width="49%">
</p>

<p align="center">
  <img src="docs/assets/screenshots/ghostftp-preferences.webp" alt="Ghost FTP Preferences" width="49%">
  <img src="docs/assets/screenshots/ghostftp-transfer-center.webp" alt="Ghost FTP Transfer Center" width="49%">
</p>

<p align="center">
  <img src="docs/assets/screenshots/ghostftp-file-properties.webp" alt="Ghost FTP File Properties" width="49%">
  <img src="docs/assets/screenshots/ghostftp-about.webp" alt="Ghost FTP About" width="49%">
</p>

The canonical desktop reference is **1290×852**. Smaller windows adapt through compact spacing, narrower rails, contained scrolling and viewport-safe dialogs instead of hiding critical controls.

The reference media under `docs/assets/screenshots/` is used for documentation and visual QA only. The running application uses real Ghost FTP components and controls — never a screenshot background or click-hotspot implementation.

See [UI/UX contract](docs/product/UI_UX.md), [Pixel Parity QA](docs/qa/PIXEL_PARITY.md), [Responsive QA](docs/qa/RESPONSIVE.md) and [Titlebar QA](docs/qa/TITLEBAR.md).

## Core workflow

**Connect.** Use Quick Connect for a temporary session or save a reusable server profile in Site Manager.

**Browse.** Work with local and remote files side by side, including hidden server files when enabled.

**Transfer.** Upload, download, pause, resume, retry, cancel and inspect queue state without leaving the main workspace.

**Control.** Use permissions, checksums, sync, terminal, search, duplicate detection, disk tools and server diagnostics from the same application.

## Implemented now

The RC9 source includes:

- FTP, FTPS and SFTP profiles and Quick Connect;
- Site Manager with favorites, folders, tags, bookmarks and recent-server metadata;
- dual-pane file management with upload/download, rename, delete, duplicate and folder operations;
- SHA-256 checksums and supported chmod/permission workflows;
- concurrent transfer queue, retry-all, pause/resume/cancel and bandwidth controls;
- folder synchronization, directory comparison, duplicate discovery and disk analysis;
- integrated terminal, command palette, snippets and keyboard shortcuts;
- OS-backed credential handling where supported;
- signed desktop updater path;
- native Windows and Linux packaging;
- 14-language selector coverage.

Full implementation detail: [Implemented Features](docs/product/FEATURES.md).

## Recommended next work

The project intentionally separates **implemented functionality** from **recommended future work**.

The highest-value remaining work includes native Windows 10/11 visual acceptance, real FTP/FTPS/SFTP end-to-end test servers, full installer lifecycle QA, signed production binaries, SBOM/dependency scanning, additional transfer scheduling/priority controls and expanded accessibility validation.

See [Project Status & Next Work](docs/product/STATUS_AND_NEXT.md) and [Roadmap](docs/product/ROADMAP.md).

## Security and privacy

- Profile passwords and private-key passphrases use OS credential facilities where supported.
- Sensitive credentials are not intended for application logs.
- SSH host-key and TLS certificate verification paths remain part of the native protocol engine.
- The updater is configured for signed package verification.
- Browser-host compatibility executables are not accepted as production GUI builds.
- The visible `127.0.0.1` browser/origin bar from earlier fallback builds is excluded from the production release path.

Read [Security](SECURITY.md), [Privacy](docs/legal/PRIVACY.md) and [Security Audit](docs/audits/SECURITY.md).

## Downloads — Ghost FTP 2.1.1 RC9

### Windows x64

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC9.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC9.exe`
- `GhostFTP-Windows-x64-v2.1.1-RC9.zip`

### Linux x86-64

- `GhostFTP-Linux-x86_64-v2.1.1-RC9`
- `GhostFTP-Linux-x86_64-v2.1.1-RC9.AppImage`
- `GhostFTP-Linux-amd64-v2.1.1-RC9.deb`
- `GhostFTP-Linux-x86_64-v2.1.1-RC9.rpm`
- `GhostFTP-Linux-x86_64-v2.1.1-RC9.tar.gz`

### Source and documentation

- `GhostFTP-v2.1.1-RC9-Source.zip`
- `GhostFTP-v2.1.1-RC9-Desktop-Source.zip`
- `GhostFTP-v2.1.1-RC9-Web.zip`
- `GhostFTP-v2.1.1-RC9-Documentation.zip`
- `GhostFTP-v2.1.1-RC9-SHA256SUMS.txt`

All published release files are accompanied by SHA-256 checksums.

> RC9 is a **release candidate**, not a FINAL label. FINAL remains gated by documented native Windows titlebar/pixel acceptance, Windows installer lifecycle QA and real FTP/FTPS/SFTP end-to-end transfer evidence.

## Repository map

```text
Ghost-FTP-Premium/
├── ghostftp-desktop/      Production Ghost FTP desktop application
├── ghostftp-runtime/      Developer / compatibility runtime tooling
├── ghostftp-installer/    Installer support and compatibility tooling
├── ghostftp-web/          ghostftp.com website source
├── ghostftp-updates/      Update manifest templates
├── docs/
│   ├── assets/            Brand and visual QA media
│   ├── architecture/      Repository and naming architecture
│   ├── audits/            Code, branding, language and security audits
│   ├── build/             Authoritative build status
│   ├── guides/            Install, uninstall, updates and support
│   ├── legal/             Privacy and third-party notices
│   ├── product/           Features, status, roadmap and UI/UX contract
│   ├── qa/                Interaction, installer, transfer and visual QA
│   └── releases/          Release notes and release-specific records
├── .github/workflows/     GhostFTP build, quality and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

## Naming policy

- **Ghost FTP** — user-facing product name.
- **GhostFTP** — filenames, archives, packages and technical product identifiers.
- Framework-specific names remain only where technically required by tooling, such as `src-tauri`, `tauri.conf.json` and framework package dependencies.

See [Repository Structure](docs/architecture/STRUCTURE.md) and [Naming Policy](docs/architecture/NAMING.md).

## Documentation

- [Documentation hub](docs/README.md)
- [Implemented features](docs/product/FEATURES.md)
- [Project status & next work](docs/product/STATUS_AND_NEXT.md)
- [Roadmap](docs/product/ROADMAP.md)
- [UI/UX contract](docs/product/UI_UX.md)
- [Build status](docs/build/STATUS.md)
- [Installation](docs/guides/INSTALLATION.md)
- [Support](docs/guides/SUPPORT.md)
- [Update policy](docs/guides/UPDATES.md)
- [Code audit](docs/audits/CODE.md)
- [Security audit](docs/audits/SECURITY.md)
- [RC9 release notes](docs/releases/2.1.1-rc.9.md)

## Brand

<p align="center">
  <img src="docs/assets/screenshots/ghostftp-brand-board.webp" alt="Ghost FTP brand identity board" width="100%">
</p>

<div align="center">

**Ghost FTP**  
*More Than Transfer. Total Control.*  
**Simple. Secure. Powerful.**

Publisher: **Brendigo LTD** / **Brendigo, obrt za programiranje**  
Official website: **https://ghostftp.com/**

</div>

Copyright © 2026 Brendigo LTD and Brendigo, obrt za programiranje. All rights reserved.
