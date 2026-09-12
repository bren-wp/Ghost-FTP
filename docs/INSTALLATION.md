# Ghost FTP installation

Ghost FTP **0.0.5** is the current published release and this document defines its canonical installation/publication contract. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **14 platform artifacts / 17 public files**.

### Windows

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Both public files are self-contained x86-compatible universal launchers carrying verified native x64/x86 Ghost FTP payloads. Native architecture is selected using the maintained `GetNativeSystemInfo` path, the selected embedded payload is verified before execution, and the bootstrap performs no runtime download. Architecture-specific staging executables remain internal evidence and must not appear in the public release directory.

Setup installs the matching native application payload and keeps the integrated uninstall path; no permanent separate uninstaller executable is required. Portable requires no installer registration.

Production Authenticode is optional. When a trusted production certificate is configured, every public Windows executable must verify. If no production certificate is configured, `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

### Linux

Canonical release packages are built by `linux/BUILD-DISTROS.sh`:

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

Representative contract names are `Ghost-FTP-0.0.5-Linux-Debian-amd64.deb`, `Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb`, `Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm` and `Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz`.

The builder compiles one production `ghostftp` executable per architecture and reuses it across matching Debian, Ubuntu, Fedora and Portable variants. Release CI extracts package payloads and compares those binaries byte-for-byte.

### Android development APK

Android is an active native source surface, but it is **not** part of the 14-artifact / 17-file public Windows/Linux release allow-list. Exact-head Android CI builds and verifies:

```text
Ghost-FTP-Android.apk
```

Its Android version identity remains bound to root `VERSION` with the maintained `-dev` suffix. The APK is not represented as a production-signed public Android release until a dedicated production signing/publication contract exists.

## Windows Setup

1. Download `Ghost-FTP-0.0.5-Setup.exe`.
2. Verify SHA-256 against `SHA256.txt`.
3. Inspect `BUILD-METADATA.txt`; require a valid Authenticode signature only when it reports `WINDOWS_AUTHENTICODE=signed`.
4. Run Setup as the intended user.
5. Uninstall through the integrated installed application path.

Installer cleanup remains ownership-bound for state/install directories, registry values and shortcuts where identity must be proven.

## Windows Portable

Run `Ghost-FTP-0.0.5-Portable.exe` directly. It performs no installer registration. Universal bootstrap selection does not weaken FTPS/SFTP verification, local-path protections, bandwidth policy, Remote Edit limits, lifecycle guards or privacy behavior.

## Linux Debian and Ubuntu

```bash
sudo apt install ./Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
sudo apt install ./Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
```

DEB packages use package identity `ghost-ftp` and declare `ca-certificates`, `curl` and `openssh-client`.

## Linux Fedora

```bash
sudo dnf install ./Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
```

Fedora metadata declares `ca-certificates`, `curl` and `openssh-clients`.

## Linux Portable

```bash
tar -xzf Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.5-Linux-Portable-amd64
./ghostftp
```

Portable archives include the executable, desktop metadata, icon, README and LICENSE. A user-writable Portable extraction cannot claim the same root-controlled OpenSSH AskPass provenance as a package-manager installation; automatic SFTP password/private-key-passphrase delivery therefore fails closed where that trusted boundary is unavailable.

## Native distro package lifecycle

Native package-manager/runtime coverage is intentionally **x86-64 only** for:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

The distro-install workflow performs real install/remove/runtime/GUI smoke in those environments. `arm64`/`aarch64` and `i386`/`i686` artifacts remain canonical release files with exact-head build, package metadata, extraction and binary-parity verification; the project does not claim unsupported native installation lifecycle coverage for them.

## Runtime behavior after installation

Windows and Linux expose the same supported desktop feature line: FTP/FTPS/SFTP, Site Manager/profile state, Remote Edit, transfer queue actions, bandwidth controls, bookmarks/start directories, current-folder filtering, recursive search, directory comparison and shared file sorting.

0.0.5 additionally carries the Windows profile/file/Remote-Edit re-entry hardening completed after 0.0.4. Android connection lifecycle hardening remains independently verified in the development APK.

## Verification

Before using an official 0.0.5 package:

1. confirm the requested version is `0.0.5`;
2. verify the filename is one of the canonical names above;
3. verify SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit and Windows signing state;
5. where metadata says `signed`, require valid Authenticode;
6. distinguish native x86-64 distro lifecycle coverage from build/parity coverage for other architectures.

The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.5`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
