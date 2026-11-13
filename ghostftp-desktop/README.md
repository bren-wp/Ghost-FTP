# Ghost FTP for Windows & Linux

**More Than Transfer. Total Control.** All your local and remote files in one focused desktop app.

[**Download**](https://github.com/bren-wp/Ghost-FTP/releases) · [Features and real screenshots](../README.md) · [Installation help](../docs/guides/INSTALLATION.md)

![Actual Ghost FTP Windows file manager](../docs/assets/screenshots/ghostftp-native-files.png)

<sub>Genuine earlier-version Windows application capture. [Original provenance](../docs/assets/screenshots/README-NATIVE-CAPTURE.json).</sub>

## Choose your edition

| Windows x64 | Linux x86_64 |
| --- | --- |
| Portable EXE · Setup EXE · MSI | AppImage · DEB · RPM · Native executable · TAR.GZ |

Connect using FTP, explicit FTPS or SFTP. Browse local and remote folders, organize Sites, manage transfer queues, synchronize and compare supported directories, and configure the workspace to suit your workflow. Cloud providers require compatible configuration and appropriate authorization.

**Ghost FTP 0.99.57 Preview** · previous published edition **v0.99.56**. The first Stable release follows verified package signatures and upgrade/install acceptance rather than a label change.

## Build from source

Dependencies: Node.js 22+, npm, Rust/Cargo and the Tauri platform prerequisites (Windows MSVC/WebView2 or Linux WebKitGTK 4.1).

```bash
npm ci
npm run check
npm run build
npm run tauri build
```

Build Windows packages: `npm run build:windows` · Linux packages: `npm run build:linux`.

The canonical published packages are produced by GitHub Actions with platform smoke checks, security and source-quality regression tests, protocol testing and release checksum verification.

[Security policy](../SECURITY.md) · [Development build guide](../docs/development/BUILDING.md) · [Stable acceptance](../docs/release/STABLE_RELEASE_POLICY.md)

**Ghost FTP by BRENDIGO** · https://brendigo.com
