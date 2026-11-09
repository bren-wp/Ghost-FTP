# Ghost FTP Installation

GitHub Releases is the canonical source for published Ghost FTP binaries and checksums.

Use the [complete official Releases index](https://github.com/bren-wp/Ghost-FTP/releases), not the GitHub `/releases/latest` shortcut (which may omit prereleases). Development source is **0.99.36 Preview**; [v0.99.35](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.99.35) is the last published predecessor. A development source version does not imply a downloadable artifact already exists.

## Verify downloads

For a published version `<version>`, download `GhostFTP-v<version>-SHA256SUMS.txt` together with the required artifact and verify the checksum before installation.

Linux example:

```bash
sha256sum -c GhostFTP-v<version>-SHA256SUMS.txt
```

On Windows, use a SHA-256-capable verification tool and compare against the canonical checksum file.

## Windows x64

Published Windows assets follow these names:

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Windows-x64-v<version>.zip`

The Setup executable is the production **native Tauri NSIS package**. It is not the Go compatibility installer.

The portable executable runs without an installation workflow. The Setup executable installs the application using the native package configuration and should be used when normal installed-app integration is preferred.

The interactive Setup is branded with Ghost FTP artwork and follows a guided Windows installer flow. Before installation it presents the Ghost FTP EULA from the repository root; installation must not continue interactively unless the user accepts the licence terms. The installer uses the Ghost FTP application icon, branded header/sidebar artwork, a Ghost FTP Start Menu folder, and automatically follows the supported Windows locale (English/Croatian installer resources are enabled).


## Linux x86-64

Published Linux assets follow these names:

- `GhostFTP-Linux-x86_64-v<version>`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Linux-amd64-v<version>.deb`
- `GhostFTP-Linux-x86_64-v<version>.rpm`
- `GhostFTP-Linux-x86_64-v<version>.tar.gz`

Use the package format appropriate to the target distribution/environment and verify its checksum before installation.

## Building instead of installing

Source-build instructions are maintained in [../development/BUILDING.md](../development/BUILDING.md).

The authoritative source directories are:

- desktop: `ghostftp-desktop/`

Support/compatibility tooling under `tools/` must not be confused with the production desktop packages described above.


## macOS Desktop — installer without Developer ID

Download the **Apple Silicon (arm64)** or **Intel (x86_64)** DMG Preview from [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases), open it, and drag `Ghost FTP Desktop.app` into `Applications`. It has no paid Apple Developer signature or notarization, so the first launch can require `System Settings → Privacy & Security → Open Anyway`. Do not disable Gatekeeper globally. [Instructions in Croatian and English](MACOS_INSTALLATION.md).

## Unsigned Windows/Linux installation and manual upgrade

The **Windows x64 Portable EXE, native NSIS Setup EXE and MSI**, plus **Linux x86_64 DEB, RPM, AppImage and standalone binary**, are genuine installable packages. A paid code-signing certificate is **not a technical prerequisite** to install them. They are not publisher-authenticated by Authenticode/GPG, so Windows SmartScreen or Linux package-manager reputation/source warnings may appear. **Do not disable your OS security protections globally.**

1. Open the [official Releases index](https://github.com/bren-wp/Ghost-FTP/releases), choose the exact version, and download both your OS installer and `GhostFTP-v<VERSION>-SHA256SUMS.txt` from the **same release**. Never rely on search-engine mirrors or a generic "latest" shortcut.
2. **Windows:** In PowerShell, run `Get-FileHash -Algorithm SHA256 -LiteralPath 'GhostFTP-Windows-x64-Setup-v<VERSION>.exe'`, then compare the hash against the matching filename's value in the downloaded checksum manifest. Select **one** installation format (Setup EXE or MSI); Portable EXE runs without installation.
3. **Linux:** For Debian/Ubuntu-family x86_64 choose `.deb`, for compatible Fedora/openSUSE/RHEL-family systems `.rpm`, and for supported glibc desktop systems AppImage where its runtime requirements are available. Check the matching SHA-256 line before executing. The packaging format alone never proves compatibility with every distribution or Wayland/GTK version.
4. **Before an upgrade:** finish or stop active transfers, create an encrypted Ghost FTP backup and store its recovery password separately, close the app, then install the newer package under the same OS user. Inspect saved profiles, OS keychain entries and your source/destination files after reopening. Do not uninstall by deleting application-data directories.
5. **Linux credentials:** An unlocked Secret Service provider (GNOME Keyring, KWallet or compatible) and `secret-tool` must be available. Check the executable with `command -v secret-tool`. Ghost FTP deliberately does **not** fall back to plaintext password files if the vault cannot be used.
6. **Updates in the app:** Help & About → Updates checks only **signed automatic updater packages**. If none are offered, click **Open official Windows / Linux downloads** to review manual packages. Lack of a signed offer is not proof that your current version is newest. Every manual update needs fresh checksum verification.

**Scope of automated verification:** Windows NSIS/MSI installation/launch/removal and Linux DEB/AppImage GUI smoke plus RPM package verification, protocol E2E, frontend/Rust/security tests and real native screenshot QA. **Not yet certified:** all combinations of Windows builds, GNOME/KDE/Wayland sessions, Linux distributions, locked keyrings, long-running transfer interruption, and live upgrade/rollback on user hardware.
