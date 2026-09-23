<p align="center">
  <img src="branding/ghostftp-logo.svg" alt="Ghost FTP" width="520">
</p>

<h1 align="center">Ghost FTP Desktop</h1>
<p align="center"><strong>More Than Transfer. Total Control.</strong></p>
<p align="center">Production desktop application for Windows and Linux.</p>

<p align="center">
  <a href="../README.md">Product README</a> ·
  <a href="../docs/README.md">Documentation</a> ·
  <a href="https://ghostftp.com/">Website</a> ·
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/releases">Releases</a>
</p>

---

## Production application

This directory contains the authoritative **Ghost FTP desktop application**.

The release line opens as a real desktop window. It does not launch the production GUI through a localhost browser wrapper, and it must not expose a visible `127.0.0.1` address/origin bar.

Current development line: **2.1.1 RC19**.

## Platform deliverables

| Platform | Deliverables |
| --- | --- |
| Windows 10/11 x64 | portable `ghostftp.exe`, NSIS Setup EXE |
| Linux x86-64 | native executable, AppImage, DEB, RPM |

MSI is intentionally excluded from prerelease RC packaging because the current MSI/WiX version path does not accept the `rc.N` prerelease identifier.

## Core capabilities

- FTP, FTPS and SFTP workflows
- dual-pane local/remote file management
- Quick Connect and saved Site Manager profiles
- transfer queues with progress, retry, pause/resume and cancellation
- file operations, checksums and permissions where supported
- SSH key and host-key verification paths
- OS-protected credentials where supported
- remote editing, synchronization, comparison and terminal tooling
- signed desktop updater path
- 14 advertised interface languages
- no required analytics or telemetry
- terminal suggestion history remains process-memory only and credential-looking commands are excluded

## Visual contract

Approved visual references live under **[`../docs/assets/screenshots/`](../docs/assets/screenshots/)**. They are documentation/QA specifications only and must never be used as runtime screenshot backgrounds or click maps.

<p align="center">
  <img src="../docs/assets/screenshots/ghostftp-main-file-manager.webp" alt="Ghost FTP approved main-window reference" width="100%">
</p>

The canonical desktop reference is **1290×852** and the production window minimum is **480×600**. RC19 keeps one persistent native application window: Files, Sites, Transfers, Sync & Backup, Settings and Help & About switch inside the main workspace; New Connection and File Properties remain transient overlays inside that same window.

Native Windows QA captures all seven critical surfaces at three viewport classes:

| QA class | Viewport |
| --- | ---: |
| Canonical | 1290×852 |
| Compact | up to 720×640 |
| Near minimum | 500×620 |

The QA workflow rejects blank/structureless frames, duplicate captures, unexpected window titles, invalid launch geometry and missing evidence. Smaller layouts use contained scrolling and icon compaction rather than removing primary actions.

### Ghost FTP palette

```text
Electric Blue  #38ABFF
Deep Navy      #0B1E36
Slate Blue     #132D52
Ice White      #EAF6FF
Success        #26DC8F
Danger         #FF5A65
```

## Source layout

```text
ghostftp-desktop/
├── branding/                 Ghost FTP SVG logo and symbol
├── packages/file-ui/         shared file-browser UI package
├── scripts/                  icon/build/verification utilities
├── src/                      React application
│   ├── components/           application surfaces and controls
│   ├── lib/                  IPC, i18n, commands and helpers
│   ├── stores/               application state
│   └── styles.css            visual and responsive rules
├── src-tauri/                internal native Rust/framework build area
│   ├── ghostftp-agent-proto/
│   ├── ghostftp-agentd/
│   └── ghostftp-cli/
└── web/                      static companion/loading surface
```

Native identifier: `com.ghostftp.desktop`  
Deep-link scheme: `ghostftp://`  
Local database namespace: `ghostftp.db`

The `src-tauri/` and `tauri.conf.json` names remain because the framework/toolchain consumes them directly. They are implementation details rather than public Ghost FTP branding.

## Development

Requirements:

- Node.js 22+
- npm
- stable Rust + Cargo
- desktop framework platform prerequisites
- Windows: WebView2 + Visual Studio native build tools
- Linux: WebKitGTK 4.1 and packaging dependencies

Install exact dependencies and validate the frontend:

```bash
npm ci
npm run check
npm run build
```

Run development mode:

```bash
npm run tauri dev
```

Build the current platform:

```bash
npm run tauri build
```

Windows RC package:

```powershell
npm run build:windows
```

Linux packages:

```bash
npm run build:linux
```

GitHub Actions is the authoritative production build environment.

## Quality gates

The repository quality workflow verifies:

- TypeScript typecheck and production frontend build
- Rust formatting, workspace compilation and tests
- Clippy with warnings denied
- Go tests/vet for compatibility tooling
- JavaScript syntax for companion surfaces
- legacy/demo-branding rejection

The native build workflow separately produces the Windows and Linux release artifacts. GitHub release publication is gated on successful build output and quality evidence.

See:

- [Build status](../docs/build/STATUS.md)
- [Code audit](../docs/audits/CODE.md)
- [Pixel parity](../docs/qa/PIXEL_PARITY.md)
- [Responsive QA](../docs/qa/RESPONSIVE.md)
- [Titlebar QA](../docs/qa/TITLEBAR.md)
- [Transfer QA](../docs/qa/TRANSFERS.md)

## Security and privacy

Credentials and connection details remain under user control. Profiles use OS-protected credential facilities where supported, SSH host-key verification is explicit, and update packages follow the signed updater path. Sensitive secrets must never be written to logs.

Read [SECURITY.md](../SECURITY.md), [Privacy](../docs/legal/PRIVACY.md) and [Update Policy](../docs/guides/UPDATES.md).

---

<p align="center">
  <img src="branding/ghostftp-symbol.svg" alt="Ghost FTP symbol" width="64"><br>
  <strong>Ghost FTP — Files Move Forward.</strong>
</p>
