# Ghost FTP installation

Ghost FTP **0.0.2** is the current published release. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **12 platform artifacts / 15 public files**.

### Windows

```text
Ghost-FTP-0.0.2-Setup-x64.exe
Ghost-FTP-0.0.2-Setup-x86.exe
Ghost-FTP-0.0.2-Setup-x32.exe
Ghost-FTP-0.0.2-Portable-x64.exe
Ghost-FTP-0.0.2-Portable-x86.exe
```

- `x64` is the native 64-bit Windows build.
- `x86` is the native 32-bit Windows build.
- `x32` is a byte-identical compatibility alias of the verified x86 Setup artifact.
- Portable builds require no installer registration.
- Setup installs the same Ghost FTP application payload and uses the integrated uninstall path.

Production Authenticode is optional. When a trusted production certificate is configured, every produced Windows artifact must verify. If no production certificate is configured, `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

### Linux

```text
Ghost-FTP-0.0.2-Linux-amd64.deb
Ghost-FTP-0.0.2-Linux-arm64.deb
Ghost-FTP-0.0.2-Linux-i386.deb
Ghost-FTP-0.0.2-Linux-multiarch.zip
Ghost-FTP-0.0.2-Linux-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-i386.tar.gz
```

DEB packages and portable tar.gz archives are built from the same per-architecture executable and are compared for byte parity during release CI.

## Windows Setup

1. Download the architecture matching the target system.
2. Verify `SHA256.txt` before installation.
3. Run Setup as the intended user.
4. Ghost FTP records its application/uninstall identity and owned shortcut digests.
5. Uninstall is integrated into the installed `GhostFTP.exe`; no permanent separate uninstaller binary is required.

The installer preserves foreign or user-modified same-name shortcuts and does not delete an unowned Start Menu parent directory.

## Windows Portable

Portable executables can be started directly. They do not require an installation transaction or registry registration. Portable use does not weaken FTPS/SFTP verification, local path protections, Remote Edit limits or privacy behavior.

## Linux DEB

Example:

```bash
sudo apt install ./Ghost-FTP-0.0.2-Linux-amd64.deb
```

The package installs the application binary, desktop entry and icon using the maintained Ghost FTP package identity.

## Linux portable archive

Example:

```bash
tar -xzf Ghost-FTP-0.0.2-Linux-amd64.tar.gz
cd Ghost-FTP-0.0.2-Linux-amd64
./ghostftp
```

Portable archives include the executable, desktop metadata, icon, README and LICENSE required by the package contract.

## Supplemental distro-specific source/CI packages

`linux/BUILD-DISTROS.sh` also builds verification-oriented distro artifacts such as:

```text
Linux-Debian-amd64.deb
Linux-Ubuntu-amd64.deb
Linux-Fedora-x86_64.rpm
Linux-Portable-amd64.tar.gz
```

These supplemental files are **not yet part of the canonical release allow-list**. Their purpose is package metadata/parity and native install/remove/GUI verification.

Native package-manager/runtime coverage is intentionally **x86-64 only** for the following CI lifecycle environments:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

Canonical Linux production artifacts still cover `amd64`, `arm64` and `i386` as documented above.

## Upgrade behavior

A future release is installed over the existing application through the maintained installer transaction. State-directory, install-directory, registry, shortcut and integrated-uninstall identity checks remain fail-closed where ownership must be proven.

The public release catalog follows a latest-only policy: after a new Ghost FTP release is successfully published and verified, older Ghost FTP releases/tags and obsolete package versions are removed. Users should therefore obtain the current release from the GitHub Releases page rather than depending on an old version URL.

## Remote Edit after installation

Windows and Linux expose the same Remote Edit engine contract for supported remote text files. Opening/saving a remote file remains subject to size, text/binary, revision/conflict, path confinement, permission-preservation and read-back verification safeguards.

## Verification

Before using an official package:

1. confirm the requested version is 0.0.2;
2. verify the file is one of the canonical artifact names;
3. verify its SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit and Windows signing state;
5. where metadata says `signed`, require a valid Authenticode signature on Windows.

The verified distribution bundle is also published at `ghcr.io/bren-wp/ghost-ftp:0.0.2`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
