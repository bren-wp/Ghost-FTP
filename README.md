# Ghost FTP 2.1.1 RC7

**Ghost FTP** is a privacy-first desktop file-transfer client for Windows and Linux. The production architecture is React + Tauri + Rust with FTP, FTPS and SFTP support.

Official product site: **https://ghostftp.com/**  
Publisher: **Brendigo LTD** and **Brendigo, obrt za programiranje**

## Design target

The approved desktop application frame is **1290 × 852 px**. The supplied Ghost FTP reference screens are the visual specification. Runtime UI is built from real controls and components; reference screenshots are never used as the application surface or as hotspot maps.

RC7 tightens the approved Electric Blue / Deep Navy / Slate Blue / Ice White system, frameless Ghost FTP window chrome, Site Manager, Preferences, Transfer Center, New Connection and File Properties geometry while retaining adaptive behavior for smaller windows.

## Architecture

- `desktop-tauri/` — native React/Tauri/Rust application and protocol engine.
- `runtime/` — compatibility host used when the native Rust/Tauri toolchain is unavailable.
- `installer/` — Windows setup compatibility source.
- `website/` — localized Ghost FTP web/loading surface.
- `updates/` — update manifest material.
- `.github/workflows/` — source audit and native build workflows.

The native window remains frameless so Ghost FTP's custom titlebar is the only intended titlebar.

## Release truth

The runnable RC7 Windows/Linux compatibility artifacts are **not** presented as native Tauri builds. Native Windows/Linux compilation, protocol integration tests, Windows titlebar screenshots, and native AppImage/RPM acceptance remain release gates until CI/target-OS evidence exists.

See `BUILD_STATUS.md` for the exact build state.

Copyright © 2026 Brendigo LTD and Brendigo, obrt za programiranje. All rights reserved.
