# Ghost FTP desktop naming policy

Use **Ghost FTP** for interface labels, product copy and release titles, and **GhostFTP** in executable, archive and CI artifact filenames.

## Canonical desktop package patterns

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.msi`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Linux-amd64-v<version>.deb`
- `GhostFTP-Linux-x86_64-v<version>.rpm`
- `GhostFTP-v<version>-Desktop-Source.zip`

The supported source roots are `ghostftp-desktop/`, `macos/`, `tools/ghostftp-runtime/`, `tools/ghostftp-installer/` and `updates/`. Framework-specific names such as `src-tauri/`, `tauri.conf.json` and `@tauri-apps/*` remain unchanged.

The installed desktop identifier `com.ghostftp.desktop`, `ghostftp://` scheme and user-data locations are migration-sensitive. A filename rebrand is not permission to break installed-client upgrades.
