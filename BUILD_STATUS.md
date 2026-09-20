# Ghost FTP 2.1.1 RC7 — Build Status

**Status: RC7 / NOT FINAL**

RC7 continues the existing Ghost FTP source tree. It does not use reference screenshots as a runtime surface.

## Completed in this pass

- Reference-parity palette and component polish for the 1290×852 desktop target.
- Full-window Site Manager, Preferences, Transfer Center and About surfaces.
- Reference-sized New Connection and File Properties dialogs.
- Frameless native Tauri configuration retained.
- Windows x64 compatibility Portable and Setup rebuilt.
- Linux x86_64 compatibility executable, tar.gz and DEB rebuilt.
- Runtime and installer `go test ./...` and `go vet ./...` pass.
- Runtime, installer and website JavaScript syntax checks pass.
- Active source branding scan: no Faro, `example.com`, fake Production/Staging/Design Assets/Cloud Server profiles, or Alex placeholder path.

## FINAL gates still open

Native React/Tauri/Rust builds must still be compiled on a complete Rust/Cargo + Node/Tauri environment and then tested on Windows/Linux. The Windows single-titlebar screenshot gate, native FTP/FTPS/SFTP end-to-end tests, native installer upgrade/uninstall acceptance, and native AppImage/RPM output are not marked passed without evidence.

Compatibility binaries are never represented as native Tauri production builds.
