# Ghost FTP 0.99.23 Preview — independent Windows and Linux packages

**Product separation is intentional.** These four brands are distinct executables, installers, repositories *within the same source tree*, runtime data boundaries and use cases. Shared protocol libraries, graphics and semantic versioning do not imply sharing credentials or an unsafe automatically enabled server.

| Product | Windows portable | Windows installer | Linux portable | Linux installer |
| --- | --- | --- | --- | --- |
| **Ghost FTP** graphical client | `GhostFTP-Windows-x64-Portable-v0.99.23.exe` | `GhostFTP-Windows-x64-Setup-v0.99.23.exe` and `.msi` | `GhostFTP-Linux-x86_64-v0.99.23`, `.AppImage`, `.tar.gz` | `.deb`, `.rpm` |
| **Ghost FTP Server** explicit FTPS daemon | `GhostFTP-Server-Windows-x64-Portable-v0.99.23.exe` | `GhostFTP-Server-Windows-x64-Setup-Unsigned-Preview-v0.99.23.exe` | `GhostFTP-Server-Linux-x86_64-v0.99.23`, `.tar.gz` | `.deb`, `.rpm` |
| **Ghost CLI** real command-line client | `GhostCLI-Windows-x64-v0.99.23.exe` | `GhostCLI-Windows-x64-Setup-Unsigned-Preview-v0.99.23.exe` | `GhostCLI-Linux-x86_64-v0.99.23`, `.tar.gz` | `.deb`, `.rpm` |
| **Ghost FTP Demo** (simulated walkthrough) | `GhostFTP-Demo-Windows-x64-Portable-v0.99.23.exe` **only** | Not distributed | Not distributed | Not distributed |

**Linux binaries are native ELF files, not Windows `.exe` files.** AppImage is available for the GUI Client; CLI and Server have independent native executables and tarballs. Windows CLI and Server installers are separate NSIS installers using Ghost FTP desktop branding and EULA notices. All Linux CLI packages install files only, never start a service, change shell configuration or create network credentials. Ghost Server installers do not create users/passwords, expose network interfaces or auto-start a public daemon.

## Real function versus simulation

- **Ghost FTP Client:** graphical transfer and saved-site experience. Uses real protocol backends and platform-specific credentials when supported.
- **Ghost FTP Server:** serves a configured local root using protected explicit FTPS, manual credentials and TLS certificates, with an authenticated loopback-only read-only dashboard. Full RBAC and SFTP *server* are not implemented.
- **Ghost CLI:** authenticates using the *same* client engine as the graphical Windows/Linux app. Commands perform real operations on user-authorized configured endpoints. Supports manual encrypted profile backup export/import. It does **not** silently pair devices or upload secrets to a Brendigo cloud.
- **Ghost FTP Demo:** all listed sites, files, queued upload/download progress and sync planning are simulated locally with no real transfer. Never use demo fixtures to measure network speed or test actual remote permissions.

## Release acceptance, checksum and version

The immutable `version.json` specifies `0.99.23`; public `v0.99.23` must be a Preview (not stable). Every edition must have at least **39 nonempty assets**, including all independent packages shown in the table. The final CI flow validates Windows/Linux native binaries and both branded CLI/Demo installers, standalone Server binary and NSIS install/service lifecycles, Linux tar/DEB/RPM contents, and Demo self-tests. The publisher then independently verifies all staged package SHA-256 digests and public remote digest entries before publishing.

Portable builds do **not** install a system service automatically. Optional code signing, notarization and Android production signing are separate readiness requirements. A published checksum alone is *not* equivalent to a publisher signature.

## Safe usage

1. Choose the correct application and operating-system package, verify the official GitHub release and its SHA-256 file, and review the EULA and release limitations.
2. With **Server**, provide valid own TLS identity and password/admin token files, a restricted OS account, a limited filesystem root, firewall policy and backups **before** enabling service operations. Do not expose the local dashboard publicly.
3. With **Client/CLI**, use only endpoints you are authorized to access. Prefer FTPS/SFTP with certificate/host-key validation; use dry-run before sync and back up before bulk destructive changes.
4. With **Demo**, run the portable binary or installer without credentials. On Windows it opens the local interface automatically; on Linux it prints its local URL, which can be opened in a browser. All progress and accounts are fictional.

See the [Server guide](../../ghost-server/README.md), [Ghost CLI manual](../../ghost-cli/README.md), [Ghost FTP Demo manual](../../ghost-demo/README.md), [privacy notice](../legal/PRIVACY.md), [release notes](../releases/0.99.23.md) and [public security policy](../../SECURITY.md).

**Status:** publication, packaging QA, independent visual sign-off and production certification must be checked from actual current CI/release results. Presence in this document is not proof that an artifact was already published.

## v0.99.23 Control Center implementation

The Windows Server installer now includes a **native Windows Forms Control Center** with actual Windows Service Control Manager status, responsive bounded resizable layout, and service Start/Stop operations (only if the administrator configured the service). A separate Win32 GUI-subsystem EXE is also shipped for standalone access, avoiding the console and any WebView. The Linux Server remains a headless native Go daemon managed by systemd; there is no accepted 1:1 graphical Linux Control Center yet. The main Ghost FTP Client still uses Tauri/system WebView. The Demo remains a separate browser-hosted simulated UI and **is not 1:1 with the client**. Do not confuse matching color tokens with complete visual parity.
