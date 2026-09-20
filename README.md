# Ghost FTP 2.1.1 RC7

<p align="center">
  <img src="docs/screenshots/09-brand-board.png" alt="Ghost FTP brand identity" width="100%">
</p>

**Ghost FTP** is a privacy-first desktop file-transfer client for Windows and Linux, built around a native React + Tauri + Rust implementation with FTP, FTPS and SFTP support. The approved UI target is the Ghost FTP visual system in `docs/screenshots/`, with a canonical 1290 × 852 desktop frame and adaptive layouts for smaller windows.

Official product site: **https://ghostftp.com/**  
Publisher: **Brendigo LTD** and **Brendigo, obrt za programiranje**

<p align="center">
  <img src="docs/screenshots/01-main-file-manager.png" alt="Ghost FTP main file manager" width="100%">
</p>

## Product scope

Ghost FTP includes saved connection profiles, Quick Connect, Site Manager, local/remote file browsing, upload and download queues, pause/resume/cancel/retry paths, file operations, permissions where supported, checksum tools, transfer history, preferences, update integration, accessibility hooks and Windows/Linux distribution targets. The native Tauri window is frameless (`decorations(false)`) so the Ghost FTP custom titlebar is the only application titlebar.

The source supports the product protocols and desktop engine in `desktop-tauri/`. The `runtime/` and `installer/` directories are **compatibility fallback hosts** used only when the Rust/Tauri toolchain is unavailable. They use real controls and local APIs, but intentionally do not pretend to provide native FTP/FTPS/SFTP authentication or transfer execution. TCP reachability and local filesystem utilities are the only network/filesystem operations claimed for that fallback.

### RC7 engineering pass

RC7 continues the existing production source and concentrates on visual parity, responsive behavior and truthful file-operation features. Site Manager, Preferences, Transfer Center and About now use dedicated full-window application surfaces instead of generic centered dialogs; New Connection and File Properties retain reference-sized modal geometry. The shared custom Ghost FTP titlebar remains the only intended native titlebar. File Properties now includes real General/Checksums tabs, real SHA-256 support where the active backend supports it, numeric chmod controls and recursive chmod for local/SFTP paths. The compatibility host mirrors the same full-window structure so fallback QA does not drift into a separate visual product.

## Design references

The ten retained QA references cover Main File Manager, Site Manager, New Connection, Preferences, Transfer Center, File Properties, About/Updates/Help, Windows/Linux setup, brand identity and web loading. They are QA/design inputs only; production runtime code does not use a full reference screenshot as an application background or click map.

## Security and privacy

Ghost FTP is configured around local profile/settings storage, OS-protected credential facilities where supported, normal TLS/SSH identity verification paths in the native engine, signed update verification through the Tauri updater, masked credential fields, and no required analytics/telemetry. Sensitive credentials must not be written to application logs. See `SECURITY.md` and `PRIVACY.md`.

## Update system

The native updater is configured for `https://ghostftp.com/updates/latest.json`. The Tauri updater plugin verifies the signed update artifact before installation. A failed update must not replace the currently working installation. See `UPDATE_POLICY.md` and `desktop-tauri/src/stores/updaterStore.ts`.

## Windows and Linux

Native bundle targets are Windows NSIS/MSI and Linux DEB/RPM/AppImage. A native package build requires Rust/Cargo, Node/npm and the Tauri platform prerequisites. This chat environment can compile the Go compatibility hosts but does not contain Rust/Cargo and cannot fetch the missing toolchain, so the runnable binaries produced here are clearly marked **fallback**, not native Tauri releases. See `BUILD_STATUS.md`.

## Languages

English is the primary language. The selector exposes English, Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip. Browser auto-translation is disabled on embedded/web runtime surfaces. The native non-English dictionaries now share the same 185-key canonical coverage set. Translation terminology/clipping is still reviewed during target-OS visual QA; details are tracked in `LANGUAGE_AUDIT.md`.

## Installation and removal

Windows compatibility artifacts include `GhostFTP-Setup.exe` and `GhostFTP-Portable.exe`. The setup flow has six ordered steps and requires EULA acceptance before installation. Removal uses the installed `GhostFTP.exe --uninstall` command / Apps & Features registration and does not create a separate `uninstall.exe`. Linux package removal is handled by the package manager, while a portable binary is removed directly. See `INSTALLATION.md` and `UNINSTALL.md`.

## Build and QA truth

`BUILD_STATUS.md` is authoritative. It records which artifacts were actually built, which checks were executed, and which release gates remain blocked. The absence of an online Windows test host and Rust/Tauri toolchain means the Windows frameless-titlebar screenshot gate, native protocol integration tests, native installer verification and native Linux AppImage/RPM generation are **not** claimed as passed in this package.

## Documentation

- `LICENSE.txt` / `EULA.txt` — commercial software licence agreement.
- `PRIVACY.md` — privacy and local-data model.
- `SECURITY.md` — security architecture and reporting guidance.
- `UPDATE_POLICY.md` — signed update flow and rollback expectations.
- `INSTALLATION.md` / `UNINSTALL.md` — platform deployment and removal.
- `SUPPORT.md` — support routes and diagnostic guidance.
- `CHANGELOG.md` — release history.
- `THIRD_PARTY_NOTICES.md` — interoperability and third-party notice policy.
- `CLICK_QA.md`, `TRANSFER_QA.md`, `INSTALLER_QA.md`, `RESPONSIVE_QA.md`, `PIXEL_QA.md`, `TITLEBAR_QA.md` — release-gate evidence.
- `BRANDING_AUDIT.md`, `CODE_AUDIT.md`, `SECURITY_AUDIT.md`, `LANGUAGE_AUDIT.md` — source audits.

Copyright © 2026 Brendigo LTD and Brendigo, obrt za programiranje. All rights reserved.
