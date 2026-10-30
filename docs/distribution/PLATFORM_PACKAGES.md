# Ghost FTP 0.99.24 — client-only Preview package inventory

The **Ghost FTP graphical client** is the only shipped product starting with 0.99.24. The independent Server, Demo and CLI packages have been retired from the current source and release workflow. Previously published versions are historical and remain available; removal from source does not uninstall historical installations from user machines or delete their data.

| Platform | Client packages |
| --- | --- |
| Windows x64 | `GhostFTP-Windows-x64-Portable-v0.99.24.exe`, `GhostFTP-Windows-x64-Setup-v0.99.24.exe`, `GhostFTP-Windows-x64-Setup-v0.99.24.msi`, ZIP bundle |
| Linux x86_64 | Native ELF, AppImage, tar.gz, DEB and RPM |
| macOS ARM64 / Intel | Per-architecture installable Preview DMGs and ZIPs; separate source-oriented macOS preview ZIP |
| Android | Installable Preview APK, unsigned release APK and unsigned AAB |

## Release inventory and security

This cycle requires **25 nonempty SHA-256-verified release assets**, including client OS packages, native Windows QA ZIP, source/documentation archives and `GhostFTP-v0.99.24-SHA256SUMS.txt`. Signed desktop updater files, when configured, must be complete and verified as a set; they are optional Preview assets and are not represented as Authenticode/Apple notarization/Play Store signing. A checksum is **not** a publisher signature.

The client supports FTP/FTPS/SFTP where the relevant platform implements and tests those protocols; do not claim uniform platform parity. In particular, the separately implemented macOS SwiftUI transport may still restrict FTPS and SFTP while its trusted-identity path is unverified. Plain FTP is unencrypted. Check the actual CI and installed-device results before declaring production readiness.

## Compatibility

- Retired standalone product source trees and release jobs are removed. Previous editions are immutable and can still be downloaded from [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases).
- This change does **not** automatically delete historical Server/CLI installations, credentials, service registrations, user files or backups.
- The GUI's own credential storage, session management and authenticated client protocol engines remain. Do not substitute simulated transfers for real integration tests.
- Installers, upgrade and rollback flows must preserve existing user data; installation sign-off requires a real device, not compilation alone.
