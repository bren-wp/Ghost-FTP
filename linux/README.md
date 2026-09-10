# Ghost FTP for Linux

Ghost FTP **0.0.3** is the current public release line. Linux uses the same connection, profile, local-filesystem, remote-operation, transfer, bandwidth settings, Remote Edit and localization engine as Windows.

## Canonical 0.0.3 release artifacts

The canonical release workflow uses `linux/BUILD-DISTROS.sh` and publishes twelve Linux artifacts across Debian, Ubuntu, Fedora and a distro-neutral Portable family.

```bash
go telemetry off
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

### Debian

```text
Ghost-FTP-0.0.3-Linux-Debian-amd64.deb
Ghost-FTP-0.0.3-Linux-Debian-arm64.deb
Ghost-FTP-0.0.3-Linux-Debian-i386.deb
```

Representative contract name: `Linux-Debian-amd64.deb`.

### Ubuntu

```text
Ghost-FTP-0.0.3-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-i386.deb
```

Representative contract name: `Linux-Ubuntu-amd64.deb`.

### Fedora

```text
Ghost-FTP-0.0.3-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-i686.rpm
```

Representative contract name: `Linux-Fedora-x86_64.rpm`.

### Portable

```text
Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-i386.tar.gz
```

Representative contract name: `Linux-Portable-amd64.tar.gz`.

Architecture mapping is explicit:

| Go / portable / DEB | RPM |
| --- | --- |
| `amd64` | `x86_64` |
| `arm64` | `aarch64` |
| `i386` | `i686` |

The builder compiles exactly one production executable per Go architecture and reuses that executable across matching Debian, Ubuntu, Fedora and Portable variants. `.github/workflows/linux-distro-packages.yml` validates package metadata and byte-for-byte binary parity.

The old generic `linux/BUILD.sh` remains maintained as a CI compatibility build, but distro-specific artifacts are no longer supplemental: they are the canonical 0.0.3 public release allow-list.

## Native distro installation verification

`.github/workflows/linux-distro-install.yml` performs clean-container package-manager installation, runtime and GUI smoke on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

The gate verifies distro/package identity, dependencies, package-owned system files, startup of installed `/usr/bin/ghostftp` under isolated local Xvfb, package removal and absence of package-owned residue.

Native install coverage is intentionally **x86-64 only**. `arm64`/`aarch64` and `i386`/`i686` retain exact-head build, metadata, extraction and byte-parity verification.

## Canonical release contract

Together with two universal Windows executables, Linux produces the 0.0.3 public assembly contract of **14 platform artifacts / 17 public files**. The same verified release directory is published as `ghcr.io/bren-wp/ghost-ftp:0.0.3`; it is a distribution bundle, not a runtime container.

## Portable use

```bash
tar -xzf Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.3-Linux-Portable-amd64
./ghostftp
```

Portable layouts contain `ghostftp`, `ghost-ftp.desktop`, `ghost-ftp.png`, `LICENSE` and `README.md`.

A directly extracted/user-writable executable cannot provide the same immutable AskPass helper boundary as a package-manager-installed root-controlled executable. Ghost FTP still starts, but automatic SFTP password/private-key-passphrase AskPass delivery fails closed when trusted provenance is unavailable.

## Installed identity and dependencies

- package name: `ghost-ftp`;
- installed executable: `/usr/bin/ghostftp`;
- desktop entry: `/usr/share/applications/ghost-ftp.desktop`;
- desktop icon: `/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png`.

Debian/Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`. Fedora RPMs declare `ca-certificates`, `curl` and `openssh-clients`.

## Desktop, authentication and settings

Linux uses the native X11/XWayland-compatible frontend without GTK, Qt, Electron or a webview. The maintained UI includes Quick Connect, FTP/FTPS/SFTP, profiles, dual file panes, transfers, queue controls, Remote Edit, recursive search/comparison and validated upload/download bandwidth settings.

English is the default/fallback and the maintained registry contains 24 languages. Fresh Quick Connect uses explicit FTPS on port 21; plain FTP remains an explicit compatibility choice and failed FTPS is never silently retried as FTP.

See `docs/SECURITY.md`, `docs/PLATFORM-PARITY.md`, `docs/DEPENDENCIES.md`, `docs/INSTALLATION.md` and `docs/TESTING.md` for the maintained release/security contract.
