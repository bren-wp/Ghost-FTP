# Ghost FTP installation

Ghost FTP **0.0.5** is the current published release and this document defines its canonical installation/publication contract. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **14 platform artifacts / 17 public files**.

### Windows

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Both public files are self-contained universal launchers carrying verified native **x64, x86 and ARM64** Ghost FTP payloads. The public bootstrap remains PE x86 for broad Windows startup compatibility, then resolves the native processor architecture through the maintained `GetNativeSystemInfo` path and selects the matching embedded payload. The selected executable is staged in Local AppData, byte-verified before execution, and no runtime package download occurs.

Architecture-specific staging executables remain internal build evidence and must not appear in the public release directory. In particular, there is no public `*-arm64.exe`; ARM64 support is delivered inside the same two canonical Setup/Portable files.

Setup installs the matching native application payload and keeps the integrated uninstall path; no permanent separate uninstaller executable is required. Portable requires no installer registration.

Official Windows publication requires trusted Authenticode. The canonical `Publish Ghost FTP` workflow requires the protected production signing identity and verifies both public executables before publication. A successful official release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

The ARM64 evidence marker is intentionally conservative. Current Windows CI cross-builds the native ARM64 client/installer payloads, verifies ARM64 PE32+ headers/resources, package routing, embedded-byte integrity and the signing pipeline, but it does not claim native ARM64 execution because the maintained Windows runner is not ARM64.

Local development and ordinary CI Windows builds may remain unsigned, but they are not official public release artifacts.

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

### macOS development app

macOS is also an active native development/source surface tied to root `VERSION`. The maintained macOS workflow builds and verifies the universal development app, but it is **not** part of the current 17-file public release allow-list. Development build success is not a claim of Developer ID signing, notarization or public macOS distribution.

## Windows Setup

1. Download `Ghost-FTP-0.0.5-Setup.exe`.
2. Verify SHA-256 against `SHA256.txt`.
3. Require `WINDOWS_AUTHENTICODE=signed` in `BUILD-METADATA.txt` and a valid Authenticode signature on the executable.
4. Optionally confirm `WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64` and `WINDOWS_SETUP=universal-x86-x64-arm64` in release metadata.
5. Run Setup as the intended user.
6. Uninstall through the integrated installed application path.

Installer cleanup remains ownership-bound for state/install directories, registry values and shortcuts where identity must be proven.

## Windows Portable

Run `Ghost-FTP-0.0.5-Portable.exe` directly. It performs no installer registration. The universal bootstrap selects the matching native x64/x86/ARM64 client without downloading another executable. Bootstrap selection does not weaken FTPS/SFTP verification, local-path protections, bandwidth policy, Remote Edit limits, lifecycle guards or privacy behavior.

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

0.0.5 additionally carries Windows profile/file/Remote-Edit re-entry hardening plus connection-generation ownership for transfer cancellation callbacks. The Windows package contract now embeds native x64, x86 and ARM64 payloads in the same two public executables. Android connection lifecycle hardening remains independently verified in the development APK, and macOS remains independently validated as a native development source surface.

## Verification

Before using an official 0.0.5 package:

1. confirm the requested version is `0.0.5`;
2. verify the filename is one of the canonical names above;
3. verify SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit and platform/signing state;
5. for official Windows Setup/Portable, require `WINDOWS_AUTHENTICODE=signed`, `WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64` and valid Authenticode;
6. treat `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` as an explicit evidence boundary, not as a claim of native ARM64 runtime execution;
7. distinguish native x86-64 Linux distro lifecycle coverage from build/parity coverage for other Linux architectures.

The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.5`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
