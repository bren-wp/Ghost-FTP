<p align="center">
  <img src="branding/ghostftp-logo.svg" alt="Ghost FTP" width="720">
</p>

<h1 align="center">Ghost FTP</h1>
<p align="center"><strong>More Than Transfer. Total Control.</strong></p>
<p align="center">A privacy-first desktop file-transfer workspace for Windows and Linux.</p>

<p align="center">
  <a href="https://ghostftp.com/">Official Website</a> ·
  <a href="https://ghostftp.com/download/">Downloads</a> ·
  <a href="https://ghostftp.com/docs/">Documentation</a> ·
  <a href="https://ghostftp.com/security/">Security</a> ·
  <a href="https://ghostftp.com/support/">Support</a>
</p>

---

## Ghost FTP 2.1

Ghost FTP is a modern desktop client built around fast file operations, secure server access, a dual-pane workflow and a consistent Windows/Linux interface. The application is branded exclusively as **Ghost FTP**: application names, package names, Rust crates, deep links, data paths, UI copy, build scripts, bundle identifiers and product-facing URLs use the Ghost FTP namespace.

The primary interface language is **English**. **Croatian / Hrvatski** is available from the language selector and Settings → Language. Public product links resolve to **ghostftp.com**.

### Supported desktop platforms

| Platform | Distribution target | Status |
|---|---|---|
| Windows 10 / 11 x64 | `.exe` NSIS installer, `.msi` | Tauri target configured |
| Linux x86_64 | `.AppImage`, `.deb`, `.rpm` | Tauri target configured |

### Core capabilities

- FTP, FTPS and SFTP/SSH workflows
- Dual-pane local/remote file browser
- Saved server profiles and Site Manager
- Upload/download transfer queue with pause, retry and progress tracking
- Remote editing, directory synchronization and comparison tools
- SSH keys, OS keychain-backed credential storage and host-key verification
- Transfer throttling, concurrency controls and overwrite policies
- Terminal and remote management utilities
- Signed update configuration using the official Ghost FTP update endpoint
- English primary UI plus Croatian language support
- No Ghost FTP telemetry layer and no third-party product branding in the UI

---

## Approved visual system

The desktop frontend and README follow the supplied Ghost FTP concept set: deep navy surfaces, electric-blue focus/accent states, soft cyan ghost mark, thin blue borders, compact Windows-style controls and dense professional file-management layouts.

### Main file manager

![Ghost FTP main file manager](docs/screenshots/01-main-file-manager.png)

### Site Manager

![Ghost FTP Site Manager](docs/screenshots/02-site-manager.png)

### New Connection

![Ghost FTP New Connection](docs/screenshots/03-new-connection.png)

### Preferences

![Ghost FTP preferences](docs/screenshots/04-preferences.png)

### Transfer Center

![Ghost FTP Transfer Center](docs/screenshots/05-transfer-center.png)

### File properties and permissions

![Ghost FTP file properties](docs/screenshots/06-file-properties.png)

### About, updates and help

![Ghost FTP About screen](docs/screenshots/07-about-updates.png)

### Windows and Linux distribution concept

![Ghost FTP Windows and Linux distribution](docs/screenshots/08-windows-linux-install.png)

### Brand identity

![Ghost FTP brand identity board](docs/screenshots/09-brand-board.png)

### Website loading screen

![Ghost FTP website loading page](docs/screenshots/10-web-loading.png)

---

## Project structure

```text
Ghost-FTP/
├─ branding/                 Ghost FTP vector marks
├─ build/                    Windows and Linux build helpers
├─ docs/screenshots/         Approved UI reference artwork
├─ packages/file-ui/         Ghost FTP file-browser component package
├─ scripts/                  Build/verification utilities
├─ src/                      React/Tauri desktop frontend
├─ src-tauri/                Rust/Tauri application and transfer engine
│  ├─ ghostftp-cli/          Ghost FTP command-line client
│  ├─ ghostftp-agent-proto/  Ghost FTP agent protocol
│  └─ ghostftp-agentd/       Ghost FTP agent daemon
└─ web/                      Static Ghost FTP loading website
```

The application bundle identifier is:

```text
com.ghostftp.desktop
```

The custom deep-link scheme is:

```text
ghostftp://
```

The local application database uses the Ghost FTP namespace (`ghostftp.db`).

---

## Development

Requirements for the desktop source build:

- Node.js 20+ / npm
- Rust stable + Cargo
- Tauri 2 platform prerequisites
- Windows: WebView2 and Visual Studio Build Tools for native Windows bundles
- Linux: WebKitGTK/Tauri build dependencies for the target distribution

Install frontend dependencies:

```bash
npm install
```

Run the desktop frontend in development mode:

```bash
npm run tauri dev
```

Run the web frontend preview:

```bash
npm run dev
```

### Windows packages

From Windows PowerShell:

```powershell
./build/windows.ps1
```

Or directly:

```powershell
npm run build:windows
```

Configured outputs are NSIS `.exe` and MSI packages under Tauri's `target/release/bundle/` tree.

### Linux packages

```bash
chmod +x build/linux.sh
./build/linux.sh
```

Or directly:

```bash
npm run build:linux
```

Configured outputs are `.AppImage`, `.deb` and `.rpm` packages under Tauri's `target/release/bundle/` tree.

---

## Web loading page

The `web/` directory is dependency-free and can be uploaded directly to ordinary shared hosting. Its viewport uses the approved Ghost FTP loading artwork without altering the supplied layout. Accessible interaction regions map the approved navigation and cards to official Ghost FTP pages.

Important public destinations:

- `https://ghostftp.com/`
- `https://ghostftp.com/download/`
- `https://ghostftp.com/docs/`
- `https://ghostftp.com/security/`
- `https://ghostftp.com/support/`

No public web link in the supplied Ghost FTP page points to another product repository or old application brand.

---

## Privacy and security principles

Ghost FTP is designed so credentials and server details remain under the user's control. The desktop architecture uses local profile storage, OS credential facilities where supported, SSH host-key verification and signed update metadata. Production deployments should keep the update signing key and distribution pipeline under the Ghost FTP release infrastructure.

See the official security information at **ghostftp.com/security/** and privacy information at **ghostftp.com/privacy/**.

---

## Brand rules

The canonical product name is always **Ghost FTP**. Internal machine-safe identifiers use `ghostftp` or `GHOSTFTP`. New modules, packages, storage keys, binaries, environment variables and deep links should follow that namespace.

Primary visual tokens used by the approved interface:

```text
Deep Navy      #061B2E
Panel Navy     #071F35
Electric Blue  #31AAFF
Action Blue    #027BF4
Ice White      #E8F6FF
Muted Cyan     #AACCE5
Success        #26DC8F
Danger         #FF5A65
```

---

**Ghost FTP — Files Move Forward.**
