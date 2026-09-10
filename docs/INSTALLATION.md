# Ghost FTP installation

Ghost FTP **0.0.3** is the current published release. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **14 platform artifacts / 17 public files**.

### Windows

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Both public executables are self-contained x86-compatible universal bootstraps containing verified native x64 and x86 Ghost FTP payloads. The bootstrap uses Windows system architecture information rather than architecture environment variables, verifies the selected staged payload and performs no runtime download.

- Setup installs the matching native payload and retains the integrated uninstall path; no permanent separate uninstaller binary is required.
- Portable starts the matching native payload without installer registration.
- Architecture-specific staging EXEs are internal build evidence and are not public release downloads.

Production Authenticode is optional. When a trusted production certificate is configured, every public Windows executable must verify. Otherwise `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

### Linux

Canonical Linux release packages are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.3-Linux-Debian-amd64.deb
Ghost-FTP-0.0.3-Linux-Debian-arm64.deb
Ghost-FTP-0.0.3-Linux-Debian-i386.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.3-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-i386.tar.gz
```

Representative contract names include `Linux-Debian-amd64.deb`, `Linux-Ubuntu-amd64.deb`, `Linux-Fedora-x86_64.rpm` and `Linux-Portable-amd64.tar.gz`.

Debian/Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`; Fedora packages declare `ca-certificates`, `curl` and `openssh-clients`. Matching Debian, Ubuntu, Fedora and Portable artifacts reuse the same production executable for a given architecture, and CI verifies byte parity before publication.

## Native distro lifecycle coverage

`.github/workflows/linux-distro-install.yml` performs real package-manager install/remove/runtime/GUI smoke on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

Native package-manager/runtime coverage is intentionally **x86-64 only**. `arm64`/`aarch64` and `i386`/`i686` remain covered by exact-head build, metadata, extraction and byte-parity checks.

## Windows Setup

1. Download `Ghost-FTP-0.0.3-Setup.exe`.
2. Verify its SHA-256 against `SHA256.txt`.
3. Inspect `BUILD-METADATA.txt` for signing state.
4. Run Setup as the intended user.
5. Ghost FTP records its owned application/uninstall and shortcut identity.

The installer preserves foreign or user-modified same-name shortcuts and does not delete an unowned Start Menu parent directory.

## Windows Portable

Verify `Ghost-FTP-0.0.3-Portable.exe` the same way, then run it directly. Portable use does not weaken FTPS/SFTP verification, local path protections, Remote Edit limits, bandwidth policy or privacy behavior.

## Linux examples

Debian:

```bash
sudo apt install ./Ghost-FTP-0.0.3-Linux-Debian-amd64.deb
```

Ubuntu:

```bash
sudo apt install ./Ghost-FTP-0.0.3-Linux-Ubuntu-amd64.deb
```

Portable:

```bash
tar -xzf Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.3-Linux-Portable-amd64
./ghostftp
```

A directly extracted/user-writable Portable executable cannot claim the same root-controlled OpenSSH AskPass provenance as a package-manager installation; automatic SFTP password/private-key-passphrase delivery therefore fails closed when that trusted boundary is unavailable.

## Verification

Before using an official package:

1. confirm the requested version is 0.0.3;
2. verify the file is one of the canonical names above;
3. verify SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit and Windows signing state;
5. where metadata says `signed`, require a valid Authenticode signature on Windows.

The verified distribution bundle is also published at `ghcr.io/bren-wp/ghost-ftp:0.0.3`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
