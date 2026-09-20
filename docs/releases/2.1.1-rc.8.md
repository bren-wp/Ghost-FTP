# Ghost FTP 2.1.1 RC8

Ghost FTP 2.1.1 RC8 replaces the browser-host compatibility GUI path with the native Tauri/WebView desktop release path for Windows and Linux.

## Critical fix

The earlier compatibility executable could open through a Chromium/Edge app shell and expose a visible `127.0.0.1` origin bar. That cannot satisfy the approved Ghost FTP frameless 1:1 design. RC8 does not publish those browser-shell binaries as end-user GUI builds.

Windows and Linux GUI artifacts for RC8 are produced from the native React + Tauri + Rust project in `desktop-tauri/`.

## RC8 changes

- Native frameless Ghost FTP application window remains `decorations(false)`.
- Canonical desktop frame remains 1290×852, with adaptive rules for smaller windows.
- Fixed the encrypted-backup magic-header Rust compilation error.
- Windows prerelease packaging uses NSIS to avoid the MSI prerelease-version restriction.
- Native Windows deliverables: portable `ghostftp.exe` and NSIS Setup EXE.
- Native Linux deliverables: executable, DEB, RPM and AppImage.
- Release publishing is gated on a successful native Windows/Linux GitHub Actions build.
- Source audit and frontend TypeScript/Vite build remain required in CI.
- Compatibility runtime/installer code remains developer tooling and is not accepted as the production GUI release path.

## Release status

RC8 is a release candidate. Native Windows and Linux Tauri builds now pass in GitHub Actions and the Windows portable/NSIS plus Linux executable/DEB/RPM/AppImage artifacts are published. It is not labelled FINAL until the remaining Windows 10/11 screenshot/titlebar, installer upgrade/uninstall, real FTP/FTPS/SFTP end-to-end and final pixel/responsive acceptance tests are completed with evidence.

## Official project

Product: Ghost FTP  
Publisher: Brendigo LTD / Brendigo, obrt za programiranje  
Website: https://ghostftp.com/
