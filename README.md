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

<p align="center"><sub><strong>Actual native application screenshot.</strong> Captured automatically from the Windows native QA workflow at the canonical 1290×852 viewport.</sub></p>

## Ghost FTP 2.1.1 RC23

Ghost FTP 2.1.1 RC23 is the all-platform production-hardening release line for:

- Windows x64 portable executable.
- Windows x64 setup executable.
- Linux x86_64 binary.
- Linux x86_64 AppImage.
- Linux amd64 DEB package.
- Linux x86_64 RPM package.
- Android APK.
- Website, update templates, source, documentation and SHA-256 checksum bundles.

RC23 keeps the Windows/Linux desktop release path, includes the native Android app in the same release train, and updates the public website so the production download, security and support routes exist. Android is built from `/android`, uses the same Ghost FTP / Brendigo product identity, and is released as an installable APK without requiring GitHub signing secrets.

## Core product

Ghost FTP brings secure server connections, dual-pane file management, saved sites, transfer queues, synchronization, checksums, permissions, terminal tools and practical server workflows into one consistent product.

<table>
<tr><td width="52"><img src="website/assets/icons/security.svg" width="30" alt=""></td><td><strong>Secure connections</strong><br>FTP, explicit FTPS and SFTP with native TLS/SSH verification paths and protected credential handling where supported.</td></tr>
<tr><td><img src="website/assets/icons/speed.svg" width="30" alt=""></td><td><strong>Fast transfers</strong><br>Concurrent queues, pause/resume, retries, conflict handling and bandwidth controls.</td></tr>
<tr><td><img src="website/assets/icons/features.svg" width="30" alt=""></td><td><strong>One workspace</strong><br>Local and remote browsing, Site Manager, permissions, checksums, sync, search and terminal tools.</td></tr>
<tr><td><img src="website/assets/icons/privacy.svg" width="30" alt=""></td><td><strong>Privacy first</strong><br>No required analytics or telemetry. Sensitive profile secrets stay outside ordinary profile JSON where supported.</td></tr>
<tr><td><img src="website/assets/icons/download.svg" width="30" alt=""></td><td><strong>Desktop + mobile</strong><br>Windows, Linux and Android release artifacts are generated through GitHub Actions.</td></tr>
</table>

## Android RC23

The Android app is native Kotlin and uses a single mobile-first application surface:

- Header with Ghost FTP RC23 release identity.
- Connection card for FTP, explicit FTPS and SFTP.
- SFTP SHA-256 host key fingerprint field with strict host-key checking.
- Remote workspace with folder listing, tap-to-open folders and tap-to-select files.
- Real transfer actions: download, upload, delete and create remote folder.
- Guarded upload and delete confirmations.
- Bounded activity log and lifecycle-safe UI updates.
- Brendigo footer.
- No analytics, no telemetry and no account requirement.

Android credentials are passed only into the active connection or transfer action. Passwords are cleared on disconnect. FTP/FTPS use production timeouts, passive binary transfers and cleanup; explicit FTPS keeps the protected data channel enabled.

## Website RC23

The `website/` source includes production routes for:

- `/download/` — Windows, Linux, Android and checksum guidance.
- `/security/` — SFTP, FTPS, FTP and privacy safeguards.
- `/support/` — install, verify, connect and issue-reporting guidance.

The sitemap includes the new production routes plus the localized landing pages.

## Build

Desktop:

```bash
cd ghostftp-desktop
npm ci
npm run build
```

Android:

```bash
gradle -p android lintDebug lintRelease assembleDebug assembleRelease
```

## Release gates

RC23 release is valid only after these GitHub Actions pass for the same source commit:

- Ghost FTP quality.
- Ghost FTP protocol E2E.
- Ghost FTP native build RC23.
- Ghost FTP Android.
- RC23 release packaging.

The release must include SHA-256 checksums for the published artifacts. RC23 publication is tied to the latest `main` commit that includes the Windows/Linux/Android version bump, Android production protocol contract, website production routes and RC23 release workflow. RC23 packaging must pull the verified Android APK artifact directly into the GitHub Release so the published asset set includes Windows, Linux and APK files.

## Repository layout

```text
Ghost-FTP/
├── android/                 Native Android application and APK workflow
├── ghostftp-desktop/        Production Windows/Linux desktop application
├── website/                 ghostftp.com source
├── updates/                 Update channels, manifest schema and tools
├── tools/                   Runtime and installer support tooling
├── docs/                    Product, QA, release, legal and development docs
├── .github/workflows/       Quality, native build, Android and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

Use **Ghost FTP** in user-facing copy and **GhostFTP** in executable/archive names. Framework-specific names are kept only where the build ecosystem requires them internally.

<div align="center">

**Ghost FTP**  
*More Than Transfer. Total Control.*  
**Simple. Secure. Powerful.**

Publisher: **Brendigo**  
Official website: **https://ghostftp.com/**

</div>