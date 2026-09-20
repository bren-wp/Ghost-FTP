<p align="center">
  <img src="branding/ghostftp-logo.svg" alt="Ghost FTP" width="520">
</p>

<h1 align="center">Ghost FTP Native Desktop</h1>
<p align="center"><strong>More Than Transfer. Total Control.</strong></p>
<p align="center">React + Tauri + Rust production desktop application for Windows and Linux.</p>

<p align="center">
  <a href="../README.md">Product README</a> ·
  <a href="../docs/README.md">Documentation</a> ·
  <a href="https://ghostftp.com/">Website</a> ·
  <a href="https://github.com/bren-wp/Ghost-FTP-Premium/releases">Releases</a>
</p>

---

## Production application

This directory contains the **production Ghost FTP desktop application**. The native release line does not launch the interface through a localhost browser shell. Tauri owns the desktop window and the React application renders the real Ghost FTP controls inside it.

Current development line: **2.1.1 RC9**.

### Native distribution targets

| Platform | Native deliverables |
| --- | --- |
| Windows 10/11 x64 | portable `ghostftp.exe`, NSIS Setup EXE |
| Linux x86-64 | native executable, AppImage, DEB, RPM |

MSI is intentionally not part of the prerelease RC pipeline because WiX/MSI does not accept the `rc.N` prerelease version used by this line.

## Core capabilities

- FTP, explicit/implicit FTPS and SFTP workflows
- Dual-pane local/remote file management
- Quick Connect and persistent Site Manager profiles
- Transfer queues with progress, retry, pause/resume and cancellation
- File operations, checksums and permission controls where supported
- SSH-key and host-key verification paths
- OS-protected credentials where supported
- Remote editing, synchronization, comparison and terminal tooling
- Signed Tauri updater integration
- 14 advertised interface languages
- no required analytics or telemetry

## Visual contract

The approved visual references live at **[`../docs/assets/screenshots/`](../docs/assets/screenshots/)**. They are documentation/QA specifications only; they must never be used as runtime screenshot backgrounds or click maps.

<p align="center">
  <img src="../docs/assets/screenshots/main.webp" alt="Ghost FTP approved main-window reference" width="100%">
</p>

Canonical desktop geometry at **1290×852**:

| Surface | Reference size |
| --- | ---: |
| Custom titlebar | 51 px |
| Application menu | 42 px |
| Quick Connect | 50 px |
| Toolbar | 62 px |
| Main file workspace | 416 px |
| Transfer/log band | 191 px |
| Status bar | 40 px |
| Sites rail | 216 px wide |
| New Connection | 752×628 |
| File Properties | 530×770 |

Smaller windows use adaptive layout and contained scrolling. Essential controls must remain reachable rather than disappearing.

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
desktop-tauri/
├── branding/                 approved SVG logo and symbol
├── packages/file-ui/         shared file-browser UI package
├── scripts/                  icon/build/verification utilities
├── src/                      React application
│   ├── components/           application surfaces and reusable controls
│   ├── lib/                  IPC, i18n, commands and helpers
│   ├── stores/               application state
│   └── styles.css            canonical visual and responsive rules
├── src-tauri/                native Rust backend and Tauri packaging
│   ├── ghostftp-agent-proto/
│   ├── ghostftp-agentd/
│   └── ghostftp-cli/
└── web/                      static companion/loading surface
```

Native identifier: `com.ghostftp.desktop`  
Deep-link scheme: `ghostftp://`  
Local database namespace: `ghostftp.db`

## Development

Requirements:

- Node.js **22+**
- npm
- stable Rust + Cargo
- Tauri 2 platform prerequisites
- Windows: WebView2 + Visual Studio native build tools
- Linux: WebKitGTK 4.1 and Tauri packaging dependencies

Install exact dependencies and validate the frontend:

```bash
npm ci
npm run check
npm run build
```

Run the native development application:

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

The repository GitHub Actions pipeline is the authoritative production build environment.

## Quality gates

The repository source audit verifies:

- TypeScript typecheck and Vite production build
- Rust formatting, workspace compilation and tests
- Clippy with warnings denied
- Go tests/vet for compatibility tooling
- JavaScript syntax for companion surfaces
- legacy/demo-branding rejection

The native workflow separately builds Windows and Linux Tauri artifacts. GitHub release publication is gated on successful native build output and source audit evidence.

See:

- [Build status](../docs/build/STATUS.md)
- [Code audit](../docs/audits/CODE.md)
- [Pixel parity](../docs/qa/PIXEL_PARITY.md)
- [Responsive QA](../docs/qa/RESPONSIVE.md)
- [Titlebar QA](../docs/qa/TITLEBAR.md)
- [Transfer QA](../docs/qa/TRANSFERS.md)

## Security and privacy

Credentials and connection details remain under user control. Native profiles use OS-protected credential facilities where supported, SSH host-key verification is explicit, and update packages are verified by the Tauri updater before installation. Sensitive secrets must never be written to logs.

Read [SECURITY.md](../SECURITY.md), [privacy](../docs/legal/PRIVACY.md) and the [update policy](../docs/guides/UPDATES.md).

---

<p align="center">
  <img src="branding/ghostftp-symbol.svg" alt="Ghost FTP symbol" width="64"><br>
  <strong>Ghost FTP — Files Move Forward.</strong>
</p>
