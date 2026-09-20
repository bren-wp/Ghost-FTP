# Ghost FTP Installation

## Windows

The compatibility setup artifact is `GhostFTP-Setup.exe` for Windows 10/11 x64. Its wizard is ordered as Welcome → License Agreement → Installation Folder → Additional Tasks → Ready/Install → Complete. Future steps are not clickable; the EULA must be accepted before continuing. The folder can be edited or selected with the folder picker, and the setup validates that the target is absolute, non-root and writable.

Optional tasks include Desktop shortcut, Start Menu shortcut, Apps & Features registration and launch-after-install. Upgrades stage the new executable and keep the previous binary temporarily so a failed commit can roll back instead of deleting the working binary first.

`GhostFTP-Portable.exe` is a standalone compatibility executable. It does not install itself unless the user explicitly runs setup.

## Native Tauri build

For the actual protocol-engine release, install Rust/Cargo, Node/npm and the Tauri v2 prerequisites on the target build system. From `desktop-tauri/`, run `npm ci`, `npm run check`, then the platform build command. Windows bundle targets are NSIS/MSI; Linux targets are DEB/RPM/AppImage.

## Linux compatibility binary

The supplied fallback Linux binary is statically linked. It requires an available Chromium-family browser runtime to host the component UI. It is not a substitute for the native Tauri protocol-engine package and is labelled accordingly in `BUILD_STATUS.md`.
