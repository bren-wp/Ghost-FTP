<p align="center">
  <img src="desktop-tauri/branding/ghostftp-logo.svg" alt="Ghost FTP" width="460">
</p>

<h1 align="center">Ghost FTP</h1>

<p align="center"><strong>More Than Transfer. Total Control.</strong></p>

<p align="center">
  A privacy-first, native desktop file-transfer client for Windows and Linux.<br>
  FTP, FTPS and SFTP in one focused interface built for people who manage real servers every day.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/actions/workflows/native-build.yml"><img alt="Native build" src="https://github.com/bren-wp/Ghost-FTP-Premium/actions/workflows/native-build.yml/badge.svg"></a>
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/actions/workflows/source-audit.yml"><img alt="Source audit" src="https://github.com/bren-wp/Ghost-FTP-Premium/actions/workflows/source-audit.yml/badge.svg"></a>
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/releases"><img alt="Release" src="https://img.shields.io/github/v/release/bren-wp/Ghost-FTP-Premium?include_prereleases&label=release"></a>
  <img alt="Platforms" src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux-38ABFF">
  <img alt="Protocols" src="https://img.shields.io/badge/protocols-FTP%20%7C%20FTPS%20%7C%20SFTP-132D52">
  <img alt="Privacy" src="https://img.shields.io/badge/telemetry-none-0B1E36">
</p>

<p align="center">
  <a href="https://ghostftp.com/">Website</a> ·
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/releases">Downloads</a> ·
  <a href="docs/README.md">Documentation</a> ·
  <a href="SECURITY.md">Security</a>
</p>

---

<p align="center">
  <img src="docs/assets/screenshots/main.webp" alt="Ghost FTP main file manager" width="100%">
</p>

## Built for fast, controlled server work

Ghost FTP combines a dual-pane file manager, saved server profiles, Quick Connect, transfer queues, permissions, checksums, synchronization-oriented tooling and detailed connection state in one native desktop application. The production GUI is built with **React + Tauri + Rust**; end-user Windows/Linux releases do not use the old browser-host compatibility shell.

**Why Ghost FTP:**

- **Native desktop experience** — frameless Ghost FTP window, custom titlebar and no visible browser/origin chrome.
- **FTP, FTPS and SFTP** — one workflow for the protocols administrators and creators actually use.
- **Dual-pane file management** — local and remote files stay visible together, with clear transfer direction and state.
- **Transfer control** — queues, pause/resume/cancel/retry flows, progress, speed and status feedback.
- **Server organization** — Site Manager, favorites, tags, folders, bookmarks and recent servers.
- **Integrity and permissions** — SHA-256 where supported, chmod controls and protocol-aware file operations.
- **Privacy-first** — no required analytics or telemetry; credentials use OS-protected storage where supported.
- **Windows + Linux** — native Windows executable/NSIS installer and Linux executable/AppImage/DEB/RPM targets.
- **14 interface languages** — English plus Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip.

## Interface

The Ghost FTP design system uses **Electric Blue**, **Deep Navy**, **Slate Blue** and **Ice White** with a dense, professional server-management layout. The canonical desktop reference is **1290 × 852**; smaller windows use adaptive layout, local scrolling and compact controls rather than hiding essential actions.

### Site Manager

<p align="center">
  <img src="docs/assets/screenshots/site-manager.webp" alt="Ghost FTP Site Manager" width="100%">
</p>

Profiles are organized around real operational context: favorites, recent servers, bookmarks, tags, folders and detailed connection settings.

### New Connection

<p align="center">
  <img src="docs/assets/screenshots/new-connection.webp" alt="Ghost FTP New Connection" width="100%">
</p>

Quick Connect and saved profiles expose protocol, host, port, username, credentials, passive mode and SSH-key options without pushing the user through unnecessary setup screens.

### Transfer Center

<p align="center">
  <img src="docs/assets/screenshots/transfer-center.webp" alt="Ghost FTP Transfer Center" width="100%">
</p>

Transfers are visible as work, not hidden as background activity: direction, progress, speed, ETA, result, actions, bandwidth and logs remain easy to inspect.

### Preferences

<p align="center">
  <img src="docs/assets/screenshots/preferences.webp" alt="Ghost FTP Preferences" width="100%">
</p>

Language, appearance, transfer behavior, connection reliability, security/privacy, updates and integrations are grouped into predictable, reversible settings.

### File Properties & Permissions

<p align="center">
  <img src="docs/assets/screenshots/file-properties.webp" alt="Ghost FTP File Properties and Permissions" width="100%">
</p>

Properties expose real metadata, checksums and permission controls where the active backend supports them. Unsupported protocol features return explicit errors instead of fabricated values.

### About, updates and support

<p align="center">
  <img src="docs/assets/screenshots/about.webp" alt="Ghost FTP About and updates" width="100%">
</p>

Version/build information, update state, changelog and support links stay inside the same visual system.

## One app. Every platform.

<p align="center">
  <img src="docs/assets/screenshots/platforms.webp" alt="Ghost FTP Windows and Linux" width="100%">
</p>

| Platform | RC9 distribution targets |
| --- | --- |
| Windows x64 | Native portable EXE, NSIS Setup EXE |
| Linux x86-64 | Native executable, AppImage, DEB, RPM |

GitHub Actions is the authoritative release build path. A release is published only after the native Windows/Linux build and source audit complete successfully.

## Brand system

<p align="center">
  <img src="docs/assets/screenshots/brand-board.webp" alt="Ghost FTP brand identity board" width="100%">
</p>

The product source includes the approved Ghost FTP symbol and horizontal logo in `desktop-tauri/branding/`. Reference screenshots in `docs/assets/screenshots/` are retained as **visual QA specifications**; the application is implemented with real controls and must not use screenshots as interactive UI backgrounds.

## Repository map

```text
Ghost-FTP-Premium/
├── desktop-tauri/        # Production React + Tauri + Rust desktop app
│   ├── branding/         # Approved Ghost FTP logo/symbol
│   ├── packages/         # Shared UI packages
│   ├── src/              # React application
│   └── src-tauri/        # Rust backend and native packaging
├── website/              # Product/web source
├── runtime/              # Developer/compatibility tooling — not production GUI
├── installer/            # Compatibility/installer tooling — not production GUI
├── docs/
│   ├── assets/           # README + visual QA media
│   ├── audits/           # Code, branding, language and security audits
│   ├── build/            # Build status and runtime reference
│   ├── guides/           # Install, uninstall, update and support docs
│   ├── legal/            # Privacy and third-party notices
│   ├── qa/               # Pixel, responsive, titlebar, transfer and click QA
│   └── releases/         # Archived release notes/checksums
└── .github/workflows/    # Audit, native build and release automation
```

## Build from source

Requirements: Node.js 22+, npm, stable Rust/Cargo and the Tauri prerequisites for your operating system.

```bash
cd desktop-tauri
npm ci
npm run check
npm run build
npm run tauri build
```

For production artifacts, use the repository's native build workflow rather than the compatibility runtime.

## Quality gates

RC9 strengthens CI around both the frontend and native workspace:

- `npm ci`, TypeScript typecheck and production Vite build;
- Go tests/vet for compatibility tooling;
- JavaScript syntax checks for runtime/installer/website;
- Rust `cargo fmt --check`, workspace check, tests and Clippy;
- legacy/demo-branding scan;
- native Windows/Linux Tauri bundle build;
- release publication only after a successful native build.

The project does **not** call a build FINAL until the remaining OS-level acceptance evidence is complete. See [build status](docs/build/STATUS.md), [pixel parity](docs/qa/PIXEL_PARITY.md), [responsive QA](docs/qa/RESPONSIVE.md) and [titlebar QA](docs/qa/TITLEBAR.md).

## Security & privacy

Ghost FTP is designed around local configuration, OS-protected credential facilities where supported, TLS/SSH identity verification paths and signed-update verification. Sensitive credentials must not be written to logs.

Read:

- [Security](SECURITY.md)
- [Privacy](docs/legal/PRIVACY.md)
- [Update policy](docs/guides/UPDATES.md)
- [Security audit](docs/audits/SECURITY.md)

## Documentation

Start at **[docs/README.md](docs/README.md)** for the complete documentation map.

## Release status

**Current development line: Ghost FTP 2.1.1 RC9.** RC means release candidate. Native Windows/Linux builds may be production-shaped artifacts, but FINAL remains gated on documented Windows titlebar/pixel verification, installer upgrade/uninstall acceptance and real FTP/FTPS/SFTP end-to-end transfer evidence.

---

<p align="center">
  <img src="desktop-tauri/branding/ghostftp-symbol.svg" alt="Ghost FTP symbol" width="72"><br>
  <strong>Ghost FTP — Files Move Forward.</strong><br>
  <sub>© 2026 Brendigo LTD and Brendigo, obrt za programiranje. All rights reserved.</sub>
</p>
