# Ghost FTP for Linux

Ghost FTP **0.0.8** is the active Linux release candidate and remains under active development. The official product website is **https://ghostftp.com**.

Linux uses the shared Ghost FTP connection, profile, filesystem, remote-operation, transfer, settings, Remote Edit, sorting/search/comparison and localization engine contracts used by the Windows reference desktop application. Platform-specific UI and operating-system integration are tested separately rather than assumed equivalent.

## 0.0.6 Linux distribution model

The public 0.0.6 contract intentionally exposes exactly **one Installer and one Portable bundle per supported Linux distribution**:

```text
Ghost-FTP-0.0.8-Linux-Debian-Installer.run
Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.8-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Fedora-Installer.run
Ghost-FTP-0.0.8-Linux-Fedora-Portable.tar.gz
```

There are no architecture-specific public `.deb`, `.rpm` or `Linux-Portable-<arch>.tar.gz` assets in the 0.0.6 release contract. Each of the six bundles carries native **amd64, arm64 and i386** Ghost FTP payloads and selects the matching payload locally from `uname -m`.

The canonical builder is:

```bash
go telemetry off
bash linux/BUILD-DISTROS.sh
```

It produces deterministic archives and self-extracting installers without downloading an architecture-specific Ghost FTP executable at install time.

## Architecture mapping

| Host architecture | Embedded Ghost FTP payload |
| --- | --- |
| `x86_64`, `amd64` | `amd64` |
| `aarch64`, `arm64` | `arm64` |
| `i386`, `i486`, `i586`, `i686`, `x86` | `i386` |

Unsupported CPU architectures fail closed with an explicit error rather than running a mismatched binary.

## Installer use

Debian example:

```bash
chmod +x Ghost-FTP-0.0.8-Linux-Debian-Installer.run
sudo ./Ghost-FTP-0.0.8-Linux-Debian-Installer.run
```

Ubuntu and Fedora use their corresponding `Installer.run` file in exactly the same way.

The installer performs a runtime dependency and CA-trust preflight before writing application files. Required platform tools are:

- Debian/Ubuntu: `ca-certificates`, `curl`, `openssh-client`;
- Fedora: `ca-certificates`, `curl`, `openssh-clients`.

By default the installer uses `/usr/local`. Engineering/test installations can set `GHOSTFTP_PREFIX` to a different writable prefix without weakening path ownership checks.

A successful installation provides `ghostftp` plus a real `ghostftp-uninstall` command. Uninstallation removes Ghost FTP application-owned installed files and intentionally does **not** delete user configuration, saved profiles or data on remote servers.

## Portable use

Debian example:

```bash
tar -xzf Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
cd Ghost-FTP-0.0.8-Linux-Debian-Portable
./ghostftp
```

The Portable root contains the architecture-selecting launcher, `bin/amd64/ghostftp`, `bin/arm64/ghostftp`, `bin/i386/ghostftp`, desktop metadata, icon, README, license and distribution identity.

Portable/per-user execution cannot automatically provide every root-controlled provenance property of a system installation. Security-sensitive SFTP AskPass behavior therefore remains fail-closed when trusted executable provenance cannot be established.

## Native distro verification

`.github/workflows/linux-distro-install.yml` performs real clean-container installer lifecycle and GUI smoke verification on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

The maintained gate verifies target OS identity, dependency installation, CA trust, Ghost FTP installation, installed files, native GUI startup under isolated local Xvfb, application termination, `ghostftp-uninstall`, and absence of installed application residue.

ARM64 and i386 payloads are exact-head build/package/binary-format verified. The project does not claim native ARM64/i386 runtime execution until maintained native runner or emulator evidence proves it.

## Graphical desktop

With a local `DISPLAY`, `ghostftp` launches the native Linux graphical frontend using the maintained X11/XWayland-compatible path. It does not require Electron, a webview, GTK/Qt or an external Go GUI module.

The maintained workspace includes Quick Connect, Site Manager/profile workflows, FTP/FTPS/SFTP, local and remote panes, create/rename/delete/permissions, upload/download, sorting/filtering, bounded recursive search, directory comparison, synchronized navigation, transfer queue controls, Top/Up/Down/Bottom priority, Remote Edit, bookmarks/start directories, appearance and validated settings/bandwidth behavior where supported by the shared engine.

For headless/terminal operation:

```text
GHOSTFTP_UI=terminal ghostftp
```

## Security and authentication

Linux supports:

- FTP as an intentional unencrypted compatibility option;
- explicit FTPS with certificate and hostname validation and no silent downgrade;
- SFTP password/private-key authentication with strict host-key trust/pinning;
- protected saved-secret handling and explicit persistence consent;
- local path/root confinement and staged transfer safety;
- privacy-safe diagnostics rather than replaying credential-bearing tool/server output.

Automatic SFTP password or private-key-passphrase delivery requires trusted installation/provenance conditions. If those conditions cannot be established, the application does not silently weaken host-key or secret-handling policy.

## Appearance and localization

Linux follows the shared Ghost FTP visual language and maintained desktop localization catalog. Dark mode is maintained alongside the light appearance; the broader 0.0.6 UI polish work is moving the light palette toward an off-white/soft-gray surface instead of harsh pure white.

English remains the canonical fallback. A localization is described as complete only when the maintained catalog and platform UI audits prove all required user-facing strings are covered.

## 0.0.6 release context

The complete planned public 0.0.6 set is **13 platform artifacts / 16 public files**:

- Windows: one universal Setup + one universal Portable with x64, x86 and ARM64 payloads;
- Linux: the six Installer/Portable bundles listed above;
- Android: one protected production-signed APK;
- Browser helpers: one deterministic ZIP each for Chrome, Edge, Firefox and Opera;
- metadata: `BUILD-METADATA.txt`, `RELEASE-NOTES.txt`, `SHA256.txt`.

Android SFTP remains hidden until strict maintained host-key verification exists. Browser helpers remain local zero-permission parser/copy companions with no supported browser-to-desktop handoff. macOS remains active development/source until real Developer ID Application signing and Apple notarization succeed.

The verified 0.0.6 release directory is also intended to be distributed as `ghcr.io/bren-wp/ghost-ftp:0.0.8`; this is a distribution bundle, not a runtime container.

## License

Ghost FTP is proprietary commercial software and is not licensed as open source. Copyright © 2026 Brendigo LTD. See the repository [`LICENSE`](../LICENSE) and [`docs/THIRD-PARTY-NOTICES.md`](../docs/THIRD-PARTY-NOTICES.md).

See [`../docs/SECURITY.md`](../docs/SECURITY.md), [`../docs/PLATFORM-PARITY.md`](../docs/PLATFORM-PARITY.md), [`../docs/DEPENDENCIES.md`](../docs/DEPENDENCIES.md), [`../docs/INSTALLATION.md`](../docs/INSTALLATION.md) and [`../docs/TESTING.md`](../docs/TESTING.md).
