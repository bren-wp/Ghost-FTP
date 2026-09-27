<div align="center">

<img src="ghostftp-desktop/branding/ghostftp-logo.svg" alt="Ghost FTP" width="430">

### More Than Transfer. Total Control.

**A modern, privacy-first FTP / FTPS / SFTP client for Windows, Linux and Android.**

[Website](https://ghostftp.com/) ·
[Releases](https://github.com/bren-wp/Ghost-FTP/releases) ·
[Documentation](docs/README.md) ·
[Security](SECURITY.md) ·
[Changelog](CHANGELOG.md)

</div>

<p align="center">
  <a href="docs/assets/screenshots/ghostftp-native-files.png"><img src="docs/assets/screenshots/ghostftp-native-files.png" alt="Ghost FTP native Files workspace" width="100%"></a>
</p>

<p align="center"><sub><strong>Release-proven native application screenshot.</strong> Local documentation images are CI-verified to be byte-identical to files contained in the newest published Ghost FTP release tag.</sub></p>

## Current status

- **Active source/release cycle:** `0.18.0`.
- **Previous canonical release:** `0.17.0`.
- **Live published release status:** [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases) is authoritative and is queried dynamically by CI.
- **Version source of truth:** root `version.json`.
- **Production desktop source:** `ghostftp-desktop/`.
- **Production Android source:** `android/`.

Ghost FTP uses a pre-1.0 semantic-version train. Meaningful development cycles advance the minor version, published hotfixes advance the patch version, and version-only commits are not used.

## Documentation image provenance

README and active documentation must never show screenshots or local product images that are newer than the latest published Ghost FTP release.

The quality workflow:

- resolves the newest non-draft GitHub Release tag;
- scans repository Markdown image references;
- validates the full `docs/assets/screenshots/` set;
- compares every local image Git blob with the same path in that release tag;
- fails if a referenced image is missing from the latest release or has drifted.

This keeps documentation visuals tied to released product state instead of unreleased UI.

## Core product

Ghost FTP combines secure server connections, dual-pane file management, saved sites, transfer queues, synchronization, checksums, permissions, terminal tools and server workflows in one native product.

<table>
<tr><td width="52"><img src="website/assets/icons/security.svg" width="30" alt=""></td><td><strong>Secure connections</strong><br>FTP, explicit FTPS and SFTP with native TLS/SSH verification paths and protected credential handling where supported.</td></tr>
<tr><td><img src="website/assets/icons/speed.svg" width="30" alt=""></td><td><strong>Fast transfers</strong><br>Concurrent queues, pause/resume, retries, conflict handling and bandwidth controls.</td></tr>
<tr><td><img src="website/assets/icons/features.svg" width="30" alt=""></td><td><strong>One workspace</strong><br>Local and remote browsing, Sites, permissions, checksums, sync, search and terminal tools.</td></tr>
<tr><td><img src="website/assets/icons/privacy.svg" width="30" alt=""></td><td><strong>Privacy first</strong><br>No required analytics or telemetry. Sensitive profile secrets stay outside ordinary profile JSON where supported.</td></tr>
<tr><td><img src="website/assets/icons/download.svg" width="30" alt=""></td><td><strong>Desktop + mobile</strong><br>Windows, Linux and Android artifacts are generated and verified through GitHub Actions.</td></tr>
</table>

## Platform scope

### Windows x64

The production Windows application is the native Tauri desktop build. Release packaging provides a portable executable and an NSIS Setup executable. Background helper processes are hardened to avoid unintended console-window flashes.

### Linux x86-64

The native desktop build produces the executable plus AppImage, DEB and RPM packages.

### Android

The native Kotlin application supports FTP, explicit FTPS and SFTP file workflows. CI verifies the installable preview APK with `apksigner`; the separate unsigned release variant is a validation artifact and is not the canonical end-user APK.

Production mobile signing policy remains a stable-release acceptance item.

## Build

Desktop frontend and type checks:

```bash
cd ghostftp-desktop
npm ci
npm run check:i18n
npm run check:ui
npm run check
npm run build
```

Windows bundle:

```bash
cd ghostftp-desktop
npm run build:windows
```

Linux bundles:

```bash
cd ghostftp-desktop
npm run build:linux
```

Android validation/build:

```bash
gradle -p android lintDebug lintRelease lintPreview assembleDebug assembleRelease assemblePreview
```

## Required PR gates

Changes intended for `main` are validated against the exact PR HEAD by:

- **Ghost FTP quality** — version/docs/image provenance, frontend, Go and Rust workspace checks.
- **Ghost FTP protocol E2E** — real FTP, explicit FTPS and SFTP roundtrips.
- **Ghost FTP native build** — Windows/Linux production bundles plus Windows native-window QA evidence.
- **Ghost FTP Android** — Android contract, lint/build, signature/package and APK artifact verification.
- **Validate Windows hardening** — Windows-specific process, UI, Rust and NSIS hardening.

A successful compile alone is not a stable/FINAL acceptance claim.

## Release model

The canonical release workflow is **Ghost FTP release**. It is triggered by the successful production native build on `main`, waits for the remaining exact-SHA gates, consumes that same build's Windows/Linux artifacts, adds the verified Android APK and native QA evidence, and publishes normalized source/documentation/checksum assets.

Existing version tags are immutable: publication refuses to retarget a tag to different source.

## Repository layout

```text
Ghost-FTP/
├── android/                 Native Android application
├── ghostftp-desktop/        Production Windows/Linux desktop application
├── website/                 ghostftp.com source
├── updates/                 Update channels, manifest schema and tools
├── tools/                   Developer/runtime/installer support tooling
├── docs/                    Product, QA, release, legal and development docs
├── version.json             Canonical active version/build metadata
├── .github/workflows/       Quality, platform build and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

Use **Ghost FTP** in user-facing copy and **GhostFTP** in executable/archive names. Framework-specific names remain only where the build ecosystem requires them internally.

<div align="center">

**Ghost FTP**  
*More Than Transfer. Total Control.*  
**Simple. Secure. Powerful.**

Publisher: **Brendigo**  
Official website: **https://ghostftp.com/**

</div>
