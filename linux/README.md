# Ghost FTP for Linux

Ghost FTP **0.0.6** is the current public release line. Linux uses the same connection, profile, filesystem, remote-operation, transfer, settings, Remote Edit, sorting/search/comparison and localization engine contracts as the Windows desktop application.

Linux remains a first-class public application platform in the expanded **18 platform artifacts / 21 public files** release. Android is now also public through protected production signing; Chrome/Edge/Firefox helper ZIPs are public companion packages; macOS remains development/source only.

The canonical 0.0.6 Linux release workflow uses `linux/BUILD-DISTROS.sh`. The older `linux/BUILD.sh` remains independent CI compatibility coverage.

This document distinguishes:

1. **Canonical 0.0.6 release artifacts** — produced by `linux/BUILD-DISTROS.sh`, verified by distro packaging and staged by the canonical release workflow.
2. **Generic CI compatibility artifacts** — produced by `linux/BUILD.sh` for independent DEB/portable build and binary-parity coverage.

## Canonical distro-specific build

```bash
go telemetry off
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

It compiles one Linux executable per Go architecture and reuses it across matching Debian, Ubuntu, Fedora and Portable packages.

### Architecture mapping

| Go / portable / DEB | RPM |
| --- | --- |
| `amd64` | `x86_64` |
| `arm64` | `aarch64` |
| `i386` | `i686` |

Canonical distro-specific artifacts are no longer supplemental; they are public release files. `.github/workflows/linux-distro-packages.yml` verifies package metadata, archive structure and byte-for-byte executable parity.

## Canonical 0.0.6 Linux artifacts

```text
Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
Ghost-FTP-0.0.6-Linux-Debian-arm64.deb
Ghost-FTP-0.0.6-Linux-Debian-i386.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-i386.tar.gz
```

Those twelve Linux artifacts are part of the complete 18-product-artifact public set. The verified release directory is also distributed at `ghcr.io/bren-wp/ghost-ftp:0.0.6` as a distribution bundle, not a runtime container.

## Native distro installation verification

`.github/workflows/linux-distro-install.yml` performs clean-container installation lifecycle on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. It verifies package identity, dependencies, installed files, startup of `/usr/bin/ghostftp` under isolated local Xvfb, removal and absence of package-owned system residue.

Native install/runtime/GUI evidence is **x86-64 only**. ARM64/i386/aarch64/i686 artifacts retain exact-head build, metadata, extraction and byte-parity verification; they are not falsely described as natively executed by the maintained matrix.

## Portable use

```bash
tar -xzf Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.6-Linux-Portable-amd64
./ghostftp
```

Portable includes `ghostftp`, desktop entry, icon, LICENSE and README. A user-writable Portable/per-user executable cannot provide the same root-controlled OpenSSH AskPass boundary as a package-installed binary, so automatic SFTP password/passphrase delivery fails closed when trusted provenance is unavailable.

## Installed identity and dependencies

- package: `ghost-ftp`;
- executable: `/usr/bin/ghostftp`;
- desktop entry: `/usr/share/applications/ghost-ftp.desktop`;
- icon: `/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png`.

Debian/Ubuntu declare `ca-certificates`, `curl`, `openssh-client`; Fedora declares `ca-certificates`, `curl`, `openssh-clients`.

## Graphical desktop

With local `DISPLAY`, `ghostftp` starts the native graphical frontend directly against X11/XWayland-compatible display transport — no GTK, Qt, Electron, webview or external Go GUI module.

The workspace includes Quick Connect, FTP/FTPS/SFTP, strict SFTP host-key trust, saved profiles, dual panes, create/rename/delete/permissions, upload/download, shared sorting/filtering, bounded recursive search, conservative directory comparison/synchronized navigation, transfer queue and Top/Up/Down/Bottom priority, Remote Edit, bookmarks/start directories, appearance and validated settings/bandwidth behavior.

Classic Light is the fresh/fallback appearance; Dark is maintained. Persisted appearance is applied before first paint with no remote theme/font runtime.

For headless/terminal operation:

```text
GHOSTFTP_UI=terminal ghostftp
```

## Authentication and protected credentials

Linux supports FTP password, explicit FTPS with strict certificate/hostname validation, SFTP password and private-key authentication plus explicit host-key fingerprint confirmation.

Automatic SFTP password/private-key-passphrase delivery requires trusted package/system installation provenance. Saving newly entered secrets is opt-in with protected local storage and plaintext UI clearing; mutable AskPass provenance never silently weakens this boundary.

## Cross-platform release context

The complete 0.0.6 public release also includes:

- two trusted-Authenticode universal Windows executables;
- one protected production-signed Android APK with exact signer SHA-256 verification;
- three deterministic branded Chrome/Edge/Firefox helper ZIPs;
- three metadata/verification files.

Android SFTP remains hidden until strict maintained host-key verification exists. Browser helpers are local parser/copy packages with no supported browser-to-desktop handoff. macOS CI validates the native universal AppKit development app but does not claim Developer ID/notarized publication.

## 0.0.6 compatibility and security note

0.0.6 preserves the mature Linux protocol/security model while revalidating all canonical distro packages. Release work on Android/browser surfaces does not weaken Linux FTPS/SFTP trust, AskPass provenance, filesystem confinement, transfer staging, package parity or native lifecycle gates.

See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md`, `docs/INSTALLATION.md` and `docs/TESTING.md`.
