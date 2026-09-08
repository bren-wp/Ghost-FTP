# Ghost FTP for Linux

Ghost FTP **1.1.6 Stable** is the current published release. The maintained `main` source contains post-1.1.6 hardening and packaging improvements for a later maintenance release. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, settings and localization engine as the Windows application.

This document distinguishes three different artifact contracts so source/CI work is never confused with the immutable published 1.1.6 release:

1. **Published 1.1.6 release assets** — historical and unchanged.
2. **Canonical next-release workflow artifacts** — produced by `.github/workflows/release.yml` from `linux/BUILD.sh`.
3. **Supplemental distro-specific CI artifacts** — produced by `linux/BUILD-DISTROS.sh` and verified independently for Debian, Ubuntu and Fedora.

## Canonical Linux build

```bash
go telemetry off
bash linux/BUILD.sh
```

`linux/BUILD.sh` produces the package set consumed by the canonical release workflow:

```text
dist/Ghost-FTP-X.Y.Z-Linux-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-i386.deb

dist/Ghost-FTP-X.Y.Z-Linux-amd64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-arm64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-i386.tar.gz
```

When `dpkg-deb` is available, DEBs are built for `amd64`, `arm64` and `i386`. Production CI sets `GHOSTFTP_REQUIRE_DEB=1`, so the canonical production build fails closed if DEB tooling is unavailable.

The DEB and portable archive for each architecture are built from the same compiled `ghostftp` executable. CI extracts both and compares the executable byte-for-byte before accepting the Linux production job.

## Supplemental distro-specific build

The maintained source also provides a separate distribution packaging path:

```bash
go telemetry off
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

It compiles exactly one Linux executable per Go architecture and reuses that executable across the matching Debian, Ubuntu, Fedora and Portable packages.

### Portable

```text
dist/Ghost-FTP-X.Y.Z-Linux-Portable-amd64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-Portable-arm64.tar.gz
dist/Ghost-FTP-X.Y.Z-Linux-Portable-i386.tar.gz
```

### Debian

```text
dist/Ghost-FTP-X.Y.Z-Linux-Debian-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-Debian-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-Debian-i386.deb
```

### Ubuntu

```text
dist/Ghost-FTP-X.Y.Z-Linux-Ubuntu-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-Ubuntu-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-Ubuntu-i386.deb
```

### Fedora

```text
dist/Ghost-FTP-X.Y.Z-Linux-Fedora-x86_64.rpm
dist/Ghost-FTP-X.Y.Z-Linux-Fedora-aarch64.rpm
dist/Ghost-FTP-X.Y.Z-Linux-Fedora-i686.rpm
```

Architecture mapping is explicit:

| Go / portable / DEB | RPM |
| --- | --- |
| `amd64` | `x86_64` |
| `arm64` | `aarch64` |
| `i386` | `i686` |

`.github/workflows/linux-distro-packages.yml` verifies package metadata and byte-for-byte executable parity across the distro-specific package family. Debian/Ubuntu DEBs and Fedora RPMs must carry the same production executable as their matching Portable archive for each architecture.

These distro-specific packages are **supplemental maintained CI outputs**. They are not retroactive 1.1.6 assets, and the current canonical release workflow does not yet publish them as release assets. A later release must explicitly integrate and verify them in the release workflow before documentation may call them part of that release's public asset set.

## Native distro installation verification

`.github/workflows/linux-distro-install.yml` performs a separate clean-container installation lifecycle on native x86-64 targets:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

For each target the gate verifies distro identity, package metadata, package-manager dependency resolution, installed runtime tools, package-owned system files, startup of the installed `/usr/bin/ghostftp` under an isolated local Xvfb server, package removal and absence of package-owned system residue.

The GUI smoke test uses a private test HOME, the normal `$HOME/.local/share` data root and a private `XDG_RUNTIME_DIR`; it does not weaken Ghost FTP's production safe-path validation. Xvfb listens locally only (`-nolisten tcp`).

Fedora-specific verification is provider-aware: the RPM `curl` requirement may be satisfied by a provider such as `curl-minimal`, so the gate verifies the installed `curl` executable and its RPM owner. CA trust is derived from the installed `ca-certificates` package instead of assuming one fixed path. The Fedora transaction also clears minimal-container `tsflags=nodocs` so the full RPM payload, including packaged documentation, is tested.

Native install coverage above is intentionally **x86-64 only**. `arm64`/`aarch64` and `i386`/`i686` distro artifacts are still protected by exact-head build, metadata, extraction and byte-parity checks; the maintained CI does not claim native package-manager/runtime installation coverage for those architectures.

## Published 1.1.6 note

The already published Ghost FTP 1.1.6 release is immutable and keeps its original Linux asset set:

```text
Ghost-FTP-1.1.6-Linux-amd64.deb
Ghost-FTP-1.1.6-Linux-arm64.deb
Ghost-FTP-1.1.6-Linux-i386.deb
Ghost-FTP-1.1.6-Linux-multiarch.zip
```

Later portable and distro-specific CI outputs are not retroactively listed as 1.1.6 release assets and do not alter the 1.1.6 tag, checksums, release notes or GHCR bundle.

## Canonical next-release contract

The maintained `.github/workflows/release.yml` currently stages the generic DEBs and generic `.tar.gz` archives from `linux/BUILD.sh`. Together with Windows artifacts and the multiarch ZIP, the current next-release assembly contract is **12 platform artifacts / 15 public files**.

The distro-specific Debian/Ubuntu/Fedora/Portable outputs from `BUILD-DISTROS.sh` are independently verified CI artifacts but are not yet part of that canonical release allow-list. This distinction is deliberate: build support is not equivalent to release publication support.

## Portable use

For the canonical generic archive:

```bash
tar -xzf Ghost-FTP-X.Y.Z-Linux-amd64.tar.gz
cd Ghost-FTP-X.Y.Z-Linux-amd64
./ghostftp
```

For the supplemental distro-neutral archive:

```bash
tar -xzf Ghost-FTP-X.Y.Z-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-X.Y.Z-Linux-Portable-amd64
./ghostftp
```

Both portable layouts contain:

```text
ghostftp
ghost-ftp.desktop
ghost-ftp.png
LICENSE
README.md
```

For optional per-user desktop integration without root privileges:

```bash
mkdir -p "$HOME/.local/bin" \
  "$HOME/.local/share/applications" \
  "$HOME/.local/share/icons/hicolor/512x512/apps"
install -m 0755 ghostftp "$HOME/.local/bin/ghostftp"
install -m 0644 ghost-ftp.desktop "$HOME/.local/share/applications/ghost-ftp.desktop"
install -m 0644 ghost-ftp.png "$HOME/.local/share/icons/hicolor/512x512/apps/ghost-ftp.png"
```

Ensure `$HOME/.local/bin` is on `PATH` before launching from the desktop entry. System protocol prerequisites still apply: a usable CA certificate store, `curl` for FTP/FTPS and OpenSSH client tools for SFTP.

## Installed identity and dependencies

- package name: `ghost-ftp`;
- installed executable: `/usr/bin/ghostftp`;
- desktop entry: `/usr/share/applications/ghost-ftp.desktop`;
- desktop icon: `/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png`;
- optional user-local portable executable: `$HOME/.local/bin/ghostftp`.

Debian and Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`. Fedora RPMs declare `ca-certificates`, `curl` and `openssh-clients`. Portable archives intentionally do not bundle package-manager metadata or copies of those system tools.

## Graphical desktop

When a local `DISPLAY` is available, `ghostftp` starts the native Ghost FTP graphical frontend by default. The GUI is implemented directly against X11/XWayland-compatible display transport without GTK, Qt, Electron, a webview or an external Go GUI module.

The graphical workspace includes Quick Connect, FTP/FTPS/implicit-FTPS/SFTP selection, SFTP host-key trust, saved profiles, dual local/server file panes, single-file and tree transfers, queue controls, local/remote file operations, remote permissions and validated transfer settings.

**Classic Light is the canonical Linux appearance.** The Linux frontend does not expose a theme switch whose backend cannot provide complete native runtime switching.

The fresh Quick Connect protocol is **explicit FTPS on port 21**. Plain FTP remains available as an explicit compatibility choice for servers that intentionally require unencrypted FTP; failed FTPS is not silently retried as FTP.

For a headless session, or to explicitly use the hardened command interface, set:

```text
GHOSTFTP_UI=terminal ghostftp
```

A graphical session requires a local X11-compatible display (native X11 or XWayland). File-transfer protocols continue to use the system transport prerequisites documented above.

## Authentication

Linux supports the maintained desktop protocol contract:

- FTP with password authentication;
- explicit FTPS with certificate validation;
- SFTP with password authentication;
- SFTP with a private key and optional passphrase;
- explicit SFTP host-key fingerprint confirmation.

Passwords and key passphrases are cleared from the public connection config after authentication. Runtime protected-secret handles distinguish session-owned and borrowed profile-owned material so session close/failed setup can forget owned secrets without invalidating stored-profile credentials needed for a later reconnect.

The accepted public SFTP fingerprint can remain as non-secret session metadata so a saved profile can retain the verified endpoint identity.

## Connection and transfer parity

Linux uses the same shared remote manager, transfer manager and guarded local filesystem service as Windows. Regression coverage protects successful manager connection, remote listing/operation access and disconnect, invalid FTP credentials, FTPS-to-plaintext failure, generation binding, staged transfers, bounded tree operations, rooted download activation and local destructive-operation safeguards.

The terminal fallback exposes remote/local navigation, file operations, transfers, queue controls, profiles, settings and language selection through typed Engine calls. Its parser does not invoke a shell for Ghost FTP commands and rejects embedded NUL/newline control characters before dispatch.

## Settings and languages

English is the canonical/default language. The maintained registry contains **24 languages**, and Linux uses the same catalogs and fallback normalization as Windows and Setup.

Production build scripts require Go telemetry to be disabled and CI uses controlled Go dependency settings. See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md`, `docs/INSTALLATION.md` and `docs/TESTING.md` for the maintained release/security contract.
