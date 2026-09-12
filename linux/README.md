# Ghost FTP for Linux

Ghost FTP **0.0.5** is the current public release line. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, settings, Remote Edit, sorting and localization engine contracts as the Windows application.

The canonical 0.0.5 release workflow uses `linux/BUILD-DISTROS.sh`. The older `linux/BUILD.sh` remains independent CI compatibility coverage.

This document distinguishes:

1. **Canonical 0.0.5 release artifacts** — produced by `linux/BUILD-DISTROS.sh`, verified by distro packaging and staged by canonical release workflow.
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

Canonical families use names `Linux-Debian-amd64.deb`, `Linux-Ubuntu-amd64.deb`, `Linux-Fedora-x86_64.rpm` and `Linux-Portable-amd64.tar.gz` with the current version prefix. `.github/workflows/linux-distro-packages.yml` verifies package metadata and byte-for-byte executable parity. These **distro-specific artifacts are no longer supplemental**; they are canonical release files.

## Canonical 0.0.5 release contract

Twelve Linux files plus two universal Windows executables produce **14 platform artifacts / 17 public files** after metadata/verification files are added.

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

The verified release directory is also distributed at `ghcr.io/bren-wp/ghost-ftp:0.0.5` as a distribution bundle, not a runtime container.

## Native distro installation verification

`.github/workflows/linux-distro-install.yml` performs clean-container installation lifecycle on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. It verifies distro/package identity, dependencies, installed files, startup of `/usr/bin/ghostftp` under isolated local Xvfb, removal and absence of package-owned system residue.

Native install coverage above is **x86-64 only**. Other canonical architectures retain exact-head build, metadata, extraction and binary-parity verification.

## Portable use

```bash
tar -xzf Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.5-Linux-Portable-amd64
./ghostftp
```

Portable includes `ghostftp`, desktop entry, icon, LICENSE and README. A user-writable Portable/per-user executable cannot provide the same root-controlled OpenSSH AskPass boundary as a package-installed binary, so automatic SFTP password/passphrase delivery fails closed where trusted provenance is unavailable.

## Installed identity and dependencies

- package: `ghost-ftp`;
- executable: `/usr/bin/ghostftp`;
- desktop entry: `/usr/share/applications/ghost-ftp.desktop`;
- icon: `/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png`.

Debian/Ubuntu declare `ca-certificates`, `curl`, `openssh-client`; Fedora declares `ca-certificates`, `curl`, `openssh-clients`.

## Graphical desktop

When local `DISPLAY` is available, `ghostftp` starts the native graphical frontend. It is implemented directly against X11/XWayland-compatible display transport without GTK, Qt, Electron, a webview or an external Go GUI module.

The workspace includes Quick Connect, FTP/FTPS/SFTP selection, SFTP host-key trust, saved profiles, dual panes, shared sorting/filter/search/comparison, transfers, queue priority, file operations, Remote Edit, bookmarks/profile starts and validated settings/bandwidth behavior.

Classic Light is the fresh/fallback appearance; Dark is maintained. Persisted appearance is applied before first frame and no remote theme service/browser runtime is involved.

For headless/terminal operation:

```text
GHOSTFTP_UI=terminal ghostftp
```

## Authentication and protected credentials

Linux supports FTP password, explicit FTPS with certificate validation, SFTP password and private-key authentication plus explicit host-key fingerprint confirmation. Automatic SFTP password/private-key-passphrase delivery requires trusted package/system installation provenance.

Saving newly entered password/passphrase material is opt-in with bounded confirmation, protected local storage and plaintext UI clearing. Runtime AskPass provenance remains a separate fail-closed boundary.

## 0.0.5 compatibility note

0.0.5's new lifecycle hardening is concentrated in Windows and Android connection ownership; Linux retains the already verified 0.0.4 desktop feature/security behavior and is rebuilt/revalidated through all canonical distro gates for this release.

See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md`, `docs/INSTALLATION.md` and `docs/TESTING.md`.
