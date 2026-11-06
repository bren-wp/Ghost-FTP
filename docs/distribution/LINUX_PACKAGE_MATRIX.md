# Linux installation: choose the correct Ghost FTP package

**Platform:** Linux **x86_64 / amd64**. These are package-family labels, not
certifications that every distribution/version has been tested. The native
Windows/Linux build currently runs on an Ubuntu 22.04 x86_64 GitHub runner;
installed-device checks across other distributions remain required.

| Linux edition in GitHub Releases | Intended distribution / installation family | Important prerequisite |
| --- | --- | --- |
| `GhostFTP-Linux-amd64-v<VERSION>.deb` | **Debian, Ubuntu, Linux Mint, Pop!_OS**, and compatible Debian-family **amd64** systems | APT/dpkg dependencies, WebKitGTK 4.1 and compatible GTK/GLib; test on the target distro release. |
| `GhostFTP-Linux-x86_64-v<VERSION>.rpm` | **Fedora, openSUSE, RHEL-compatible** RPM-based **x86_64** systems | RPM runtime dependencies, WebKitGTK and GTK availability differ. Package format alone does not prove the build launches. |
| `GhostFTP-Linux-x86_64-v<VERSION>.AppImage` | **Portable / cross-distribution** candidate for many **glibc-based x86_64** desktop systems | Check FUSE/AppImage runtime, compatible glibc, WebKitGTK/GTK libraries and display server. Not guaranteed universal. |
| `GhostFTP-Linux-x86_64-v<VERSION>` / `.tar.gz` | **Manual / advanced** installation on compatible **x86_64** desktop Linux | Not statically linked; administrators must supply all system runtime dependencies and desktop integration. |

Replace `<VERSION>` with the exact published tag number. Verify
`GhostFTP-v<VERSION>-SHA256SUMS.txt` before executing installers. A SHA-256
checksum only verifies integrity against that manifest; it does **not** prove
publisher identity or replace a cryptographic code signature.

**No native package is currently built for** Linux ARM64/aarch64,
32-bit x86, musl-only Alpine, or every NixOS/immutable-container configuration.
Arch Linux, Manjaro, EndeavourOS, Gentoo, elementary OS and others may be
candidates for AppImage/manual installation, but are **not** certified based
solely on an Ubuntu runner. Check compatible dependencies first; do not force a
Debian RPM/DEB package onto an unrelated distribution. On Wayland systems,
GTK/WebKit or XWayland availability can affect behavior.

## Required acceptance before claiming distribution support

For **each distribution and version** named as tested, record its exact image
version, x86_64 architecture, display protocol (X11/Wayland), WebKitGTK/GTK
versions, GNOME Keyring/KWallet state, and the native binary hash. Verify:

1. Install, first launch, File Manager/Sites/Transfers/Settings/Sync navigation,
   resize at 1290×852 and 480×600, and correct keyboard focus.
2. FTP warning, verified FTPS TLS and SFTP SSH host-key checks on actual
   endpoints; failed identity checks must not connect.
3. Queue retry/cancel, interrupted upload/download preservation, symlink and
   overwrite safety, and shutdown/restart without losing profile metadata.
4. OS-backed credentials with locked and unavailable Secret Service; no
   plaintext fallback or secret-bearing logs.
5. Uninstall and reinstall without silently deleting stored user documents,
   connection profiles, or keys.

Current GitHub Actions Linux QA produces **real** Xvfb/WebKit screenshots of
seven core workspaces, not mockups. Those captures prove native rendering under
the CI environment; they do not certify a GNOME/KDE/Wayland distro or pixel
parity with all 29 user-supplied Linux design references.

Related: [Linux handbook](../handbook/LINUX.md) ·
[Native desktop build](../../.github/workflows/ghostftp-build.yml) ·
[Published releases](https://github.com/bren-wp/Ghost-FTP/releases).

## Operational prerequisites for unsigned desktop builds

Running the native binary does not require a purchased signing certificate. Verify the SHA-256 checksum of your package against the matching official release manifest; this validates file integrity, **not** cryptographic publisher identity. Saved credentials require an unlocked Secret Service provider and the `secret-tool` client (the DEB declares `libsecret-tools`, the RPM declares `libsecret`). An unavailable vault is a blocking error, not a reason to store plaintext passwords. Before upgrading, create an encrypted backup and ensure all transfers are finished. The desktop Updates screen offers the complete GitHub Releases index because signed auto-updates and manually downloadable package editions are independent.
