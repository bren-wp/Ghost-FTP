# Ghost FTP installation

Ghost FTP **0.0.2** is the current published release. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **14 platform artifacts / 17 public files**.

### Windows

```text
Ghost-FTP-0.0.2-Setup.exe
Ghost-FTP-0.0.2-Portable.exe
```

- `Setup.exe` is one offline universal Windows package for supported x86 and x64 systems.
- `Portable.exe` is one offline universal Windows package for supported x86 and x64 systems.
- Each public package contains the already verified native x86 and x64 Ghost FTP payloads and chooses the compatible payload from the native Windows architecture through `GetNativeSystemInfo`.
- No architecture-specific Windows executable is published separately.
- Portable use requires no installer registration.
- Setup delegates to the verified native setup payload, installs the same Ghost FTP application and preserves the integrated uninstall path.

Production Authenticode is optional. When a trusted production certificate is configured, the native payloads and both final public Windows packages must verify. If no production certificate is configured, `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

### Linux

Debian:

```text
Ghost-FTP-0.0.2-Linux-Debian-amd64.deb
Ghost-FTP-0.0.2-Linux-Debian-arm64.deb
Ghost-FTP-0.0.2-Linux-Debian-i386.deb
```

Ubuntu:

```text
Ghost-FTP-0.0.2-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.2-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.2-Linux-Ubuntu-i386.deb
```

Fedora:

```text
Ghost-FTP-0.0.2-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.2-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.2-Linux-Fedora-i686.rpm
```

Portable Linux:

```text
Ghost-FTP-0.0.2-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-Portable-i386.tar.gz
```

All distro and portable packages for one architecture are built from the same Ghost FTP executable. Release CI verifies package metadata and compares the extracted package executable byte-for-byte with the corresponding portable executable.

## Windows Setup

1. Download `Ghost-FTP-0.0.2-Setup.exe`.
2. Verify its SHA-256 against `SHA256.txt`.
3. Run Setup as the intended user.
4. The bootstrap asks Windows for the native processor architecture, verifies and starts the embedded matching native setup payload; it does not download another executable.
5. The native installer records the Ghost FTP application/uninstall identity and owned shortcut digests.
6. Uninstall remains integrated into the installed `GhostFTP.exe`; no permanent separate uninstaller binary is required.

The installer preserves foreign or user-modified same-name shortcuts and does not delete an unowned Start Menu parent directory.

## Windows Portable

Download `Ghost-FTP-0.0.2-Portable.exe` and start it directly. The bootstrap selects the embedded native x86 or x64 client, verifies the staged payload, runs it for the lifetime of the portable session and removes the staged file after exit.

Portable use does not require an installation transaction or registry registration and does not weaken FTPS/SFTP verification, local path protections, Remote Edit limits or privacy behavior.

## Debian

Example for amd64:

```bash
sudo apt install ./Ghost-FTP-0.0.2-Linux-Debian-amd64.deb
```

## Ubuntu

Example for amd64:

```bash
sudo apt install ./Ghost-FTP-0.0.2-Linux-Ubuntu-amd64.deb
```

Debian and Ubuntu packages carry the maintained `ghost-ftp` package identity while retaining an explicit distribution marker in package metadata.

## Fedora

Example for x86-64:

```bash
sudo dnf install ./Ghost-FTP-0.0.2-Linux-Fedora-x86_64.rpm
```

The Fedora package uses the maintained `ghost-ftp` RPM identity and Fedora dependency naming.

## Linux portable archive

Example for amd64:

```bash
tar -xzf Ghost-FTP-0.0.2-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.2-Linux-Portable-amd64
./ghostftp
```

Portable archives include the executable, desktop metadata, icon, README and LICENSE required by the package contract.

## Native distro lifecycle coverage

Native package-manager/runtime CI coverage is intentionally exercised on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

Release artifacts themselves continue to cover the documented three architecture variants per distro/portable family.

## Upgrade behavior

A future release is installed over the existing application through the maintained native installer transaction selected by the public universal setup bootstrap. State-directory, install-directory, registry, shortcut and integrated-uninstall identity checks remain fail-closed where ownership must be proven.

The public release catalog follows a latest-only policy: after a new Ghost FTP release is successfully published and verified, older Ghost FTP releases/tags and obsolete package versions are removed. Users should therefore obtain the current release from the GitHub Releases page rather than depending on an old version URL.

## Remote Edit after installation

Windows and Linux expose the same Remote Edit engine contract for supported remote text files. Opening/saving a remote file remains subject to size, text/binary, revision/conflict, path confinement, permission-preservation and read-back verification safeguards.

## Verification

Before using an official package:

1. confirm the requested version is 0.0.2;
2. verify the file is one of the canonical artifact names above;
3. verify its SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit, architecture/package contract and Windows signing state;
5. where metadata says `signed`, require a valid Authenticode signature on Windows.

The verified distribution bundle is also published at `ghcr.io/bren-wp/ghost-ftp:0.0.2`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
