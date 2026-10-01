<div align="center">

<img src="ghostftp-desktop/branding/ghostftp-logo.svg" alt="Ghost FTP" width="430">

### More Than Transfer. Total Control.

**A privacy-first FTP / FTPS / SFTP client for Windows, Linux and Android.**

[Releases](https://github.com/bren-wp/Ghost-FTP/releases) ·
[Documentation](docs/README.md) ·
[Security](SECURITY.md) ·
[Privacy](docs/legal/PRIVACY.md) ·
[Changelog](CHANGELOG.md)

</div>

<p align="center">
  <a href="docs/assets/screenshots/ghostftp-native-files.png"><img src="docs/assets/screenshots/ghostftp-native-files.png" alt="Ghost FTP native Files workspace" width="100%"></a>
</p>

<p align="center"><sub><strong>Real native application screenshot.</strong> README/documentation images are CI-verified against the newest published Ghost FTP release tag.</sub></p>

## Application gallery

<table>
<tr>
<td width="50%"><a href="docs/assets/screenshots/ghostftp-native-new-connection.png"><img src="docs/assets/screenshots/ghostftp-native-new-connection.png" alt="Ghost FTP New Connection"></a><br><strong>New Connection</strong></td>
<td width="50%"><a href="docs/assets/screenshots/ghostftp-native-sites.png"><img src="docs/assets/screenshots/ghostftp-native-sites.png" alt="Ghost FTP Sites"></a><br><strong>Sites</strong></td>
</tr>
<tr>
<td><a href="docs/assets/screenshots/ghostftp-native-transfers.png"><img src="docs/assets/screenshots/ghostftp-native-transfers.png" alt="Ghost FTP Transfers"></a><br><strong>Transfers</strong></td>
<td><a href="docs/assets/screenshots/ghostftp-native-settings.png"><img src="docs/assets/screenshots/ghostftp-native-settings.png" alt="Ghost FTP Settings"></a><br><strong>Settings</strong></td>
</tr>
<tr>
<td><a href="docs/assets/screenshots/ghostftp-native-file-properties.png"><img src="docs/assets/screenshots/ghostftp-native-file-properties.png" alt="Ghost FTP File Properties"></a><br><strong>File properties</strong></td>
<td><a href="docs/assets/screenshots/ghostftp-native-about.png"><img src="docs/assets/screenshots/ghostftp-native-about.png" alt="Ghost FTP Help and About"></a><br><strong>Help & About</strong></td>
</tr>
</table>

## Current status

- **Active source/release cycle:** `0.20.4`.
- **Previous canonical release:** `0.20.3`.
- **Version source of truth:** root `version.json`.
- **Production desktop source:** `ghostftp-desktop/` — one native Tauri/React/Rust product used by Windows and Linux.
- **Production Android source:** `android/` — native Kotlin mobile application aligned to the same Files/Sites/Transfers connection and action model.
- **No website application is maintained in this repository.** Releases, source, documentation and support/security material live in GitHub/repository artifacts.

## Product scope

Ghost FTP combines secure server connections, local/remote file management, saved Sites, transfer queues, synchronization, checksums, permissions, terminal tools and server workflows in a single desktop product. Android follows the same core connection/file-action terminology using a touch-first layout.

### Secure connections

FTP, explicit FTPS and SFTP with TLS/SSH verification, protected credential handling where supported, timeout/cleanup controls and guarded remote mutations.

### Files and transfers

Local/remote browsing, upload/download, folder creation, rename/delete/properties, concurrent transfer queues, pause/resume/retry, conflict handling and bandwidth controls. Desktop transfer history is persisted without credentials so interrupted work remains visible after restart; recovered retries restart safely from byte zero unless file identity can be proven. The 0.20.1 transfer-worker race fix, 0.20.2 Android/desktop launch hardening and 0.20.3 async file-action hardening remain in 0.20.4. The current patch modernizes CI action runtimes, removes obsolete Go-cache warnings and makes the Vite configuration compatible with native ESM config loading.

### Productivity

Sites, Sync & Backup, terminal tools, command palette, snippets, shortcuts, checksums, permissions and platform integration.

### Privacy

No required analytics or telemetry. Sensitive diagnostic text is redacted and secrets are kept out of ordinary profile storage where supported.

## Platform scope

### Windows x64

The production Windows application is the native Tauri desktop build. Release packaging provides a portable executable, a branded NSIS Setup and a Windows Installer (`.msi`) package. CI performs real silent install/uninstall smoke coverage for both installer formats before either can reach a release.

### Linux x86-64

Linux uses the same desktop frontend and Rust/native engine as Windows. Release packaging provides the native executable, AppImage, DEB and RPM packages. CI validates package metadata and lifecycle behavior before publication.

### Android

The native Kotlin application supports FTP, explicit FTPS and SFTP connection/listing workflows plus upload, download, new-folder, delete, refresh and guarded session handling. Uploads/downloads use staged replacement, lifecycle cancellation propagates into remote mutations, and persisted document access keeps SAF permissions without storing session credentials. Android keeps the same product terminology and branding while using separate Files, Sites, Transfers, Settings and Help & About workspaces behind a persistent left navigation rail.

## Verification

Desktop:

```bash
cd ghostftp-desktop
npm ci
npm run check:i18n
npm run check:ui
npm run check
npm run build
```

Android:

```bash
gradle -p android lintDebug lintRelease lintPreview assembleDebug assembleRelease assemblePreview
```

Required exact-head gates:

- **Ghost FTP quality**
- **Ghost FTP protocol E2E**
- **Ghost FTP native build**
- **Ghost FTP Android**
- **Validate Windows hardening**

The Windows/Linux build additionally performs Windows NSIS/MSI lifecycle smoke tests, Linux package lifecycle checks, native QA evidence capture and artifact-size budget checks.

## Release model

The canonical **Ghost FTP release** workflow runs only from a successful native build on `main`, waits for every required exact-SHA gate, consumes the same Windows/Linux artifacts, adds the verified Android APK and publishes normalized source/documentation/update/checksum assets.

Version tags are immutable. A published tag is never retargeted to newer source.

## Repository layout

```text
Ghost-FTP/
├── android/                 Native Android application
├── ghostftp-desktop/        Native Windows/Linux application
├── updates/                 Desktop update-service tooling and operator docs
├── tools/                   Support/runtime/installer tooling
├── docs/                    Product, QA, release, legal and development docs
├── version.json             Canonical active version/build metadata
├── .github/workflows/       Quality, build, platform and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

Use **Ghost FTP** in user-facing copy and **GhostFTP** in executable/archive names.

<div align="center">

**Ghost FTP**  
*More Than Transfer. Total Control.*  
**Simple. Secure. Powerful.**

Publisher: **Brendigo**  
Canonical downloads: **GitHub Releases**

</div>
