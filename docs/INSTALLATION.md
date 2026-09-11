# Ghost FTP installation

Ghost FTP **0.0.4** is the current published release and this document defines its canonical installation/publication contract. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **14 platform artifacts / 17 public files**.

### Windows

```text
Ghost-FTP-0.0.4-Setup.exe
Ghost-FTP-0.0.4-Portable.exe
```

- Both public files are self-contained x86-compatible universal launchers that carry verified native x64 and x86 Ghost FTP payloads.
- Native architecture is selected from Windows system information through the maintained `GetNativeSystemInfo` path rather than mutable architecture environment variables.
- The selected embedded payload is verified before execution and the bootstrap performs no runtime download.
- Architecture-specific x64/x86 staging executables remain internal build evidence and must not appear in the public release directory.
- Portable requires no installer registration.
- Setup installs the matching native Ghost FTP application payload and uses the integrated uninstall path; no permanent separate uninstaller binary is required.

Production Authenticode is optional. When a trusted production certificate is configured, every public Windows executable must verify. If no production certificate is configured, `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

### Linux

Canonical Linux 0.0.4 release files are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.4-Linux-Debian-amd64.deb
Ghost-FTP-0.0.4-Linux-Debian-arm64.deb
Ghost-FTP-0.0.4-Linux-Debian-i386.deb
Ghost-FTP-0.0.4-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.4-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.4-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.4-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.4-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.4-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.4-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.4-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.4-Linux-Portable-i386.tar.gz
```

Representative contract names are `Linux-Debian-amd64.deb`, `Linux-Ubuntu-amd64.deb`, `Linux-Fedora-x86_64.rpm` and `Linux-Portable-amd64.tar.gz`.

The builder compiles one production `ghostftp` executable per Go architecture and reuses that executable across matching Debian, Ubuntu, Fedora and Portable variants. Release CI extracts package payloads and compares those binaries byte-for-byte.

### Android development APK

Android is an active native source surface, but it is **not** part of the 14-artifact / 17-file public Windows/Linux release allow-list. Exact-head Android CI builds and verifies an installable development artifact:

```text
Ghost-FTP-Android.apk
```

Its Android `versionName` remains bound to root `VERSION` with the maintained `-dev` suffix. It is not represented as a production-signed Android public release until a dedicated production signing/publication contract exists.

## Windows Setup

1. Download `Ghost-FTP-0.0.4-Setup.exe`.
2. Verify its SHA-256 against `SHA256.txt` before installation.
3. Inspect `BUILD-METADATA.txt` and require a valid Authenticode signature only when metadata reports `WINDOWS_AUTHENTICODE=signed`.
4. Run Setup as the intended user.
5. Ghost FTP records its application/uninstall identity and owned shortcut digests.
6. Uninstall remains integrated into the installed `GhostFTP.exe`; no permanent separate uninstaller executable is installed.

The installer preserves foreign or user-modified same-name shortcuts and does not delete an unowned Start Menu parent directory. State-directory, install-directory, registry, shortcut and integrated-uninstall cleanup remains ownership-bound where identity must be proven.

## Windows Portable

`Ghost-FTP-0.0.4-Portable.exe` can be started directly and does not require an installation transaction or registry registration. Universal bootstrap selection does not weaken FTPS/SFTP verification, local path protections, bandwidth policy, Remote Edit limits, Light/Dark appearance behavior or privacy behavior.

## Linux Debian and Ubuntu DEB

Examples:

```bash
sudo apt install ./Ghost-FTP-0.0.4-Linux-Debian-amd64.deb
sudo apt install ./Ghost-FTP-0.0.4-Linux-Ubuntu-amd64.deb
```

The packages install the application binary, desktop entry and icon using the maintained `ghost-ftp` package identity. Debian/Ubuntu metadata declares `ca-certificates`, `curl` and `openssh-client`.

Package-installed Linux builds satisfy the maintained root-controlled executable provenance expected by automatic SFTP AskPass credential delivery, subject to the runtime parent/tool provenance checks. Saved profile credentials remain opt-in and local.

## Linux Fedora RPM

Example:

```bash
sudo dnf install ./Ghost-FTP-0.0.4-Linux-Fedora-x86_64.rpm
```

Fedora metadata declares `ca-certificates`, `curl` and `openssh-clients`. Runtime verification is provider-aware where Fedora may satisfy the `curl` requirement through a compatible package provider.

## Linux portable archive

Example:

```bash
tar -xzf Ghost-FTP-0.0.4-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.4-Linux-Portable-amd64
./ghostftp
```

Portable archives include the executable, desktop metadata, icon, README and LICENSE required by the package contract. A user-writable Portable extraction cannot claim the same root-controlled OpenSSH AskPass provenance as a package-manager installation; automatic SFTP password/private-key-passphrase delivery therefore fails closed when that trusted boundary is unavailable.

## Native distro package lifecycle

Native package-manager/runtime coverage is intentionally **x86-64 only** for the maintained lifecycle environments:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

`.github/workflows/linux-distro-install.yml` performs real install/remove/runtime/GUI smoke in those environments. It verifies package identity, runtime dependencies, package-owned files, startup of installed `/usr/bin/ghostftp` under isolated local Xvfb, removal and absence of package-owned residue.

`arm64`/`aarch64` and `i386`/`i686` artifacts remain canonical release files and receive exact-head build, package metadata, extraction and binary-parity verification; the project does not claim native installation lifecycle coverage for those architectures until a corresponding gate exists.

## Upgrade behavior

A future release is installed over the existing application through the maintained installer transaction. State-directory, install-directory, registry, shortcut and integrated-uninstall identity checks remain fail-closed where ownership must be proven.

The public release catalog follows a latest-only policy: after a new Ghost FTP release is successfully published and verified, older Ghost FTP releases/tags and obsolete package versions are removed. Users should obtain the current release rather than depending on a superseded version URL.

## Runtime behavior after installation

Windows and Linux expose the same supported desktop feature set through native frontends backed by the shared Engine. That includes FTP/FTPS/SFTP connection behavior, Site Manager/profile state, Remote Edit, transfer queue actions, bandwidth controls, bookmarks/start directories, filter/search/directory comparison and shared file ordering semantics.

Linux 0.0.4 additionally applies its persisted Classic Light/Dark appearance before first paint and exposes the same validated appearance policy as Windows. Native widget/layout implementation remains platform-specific.

## Remote Edit after installation

Windows and Linux expose the same Remote Edit engine contract for supported remote text files. Opening/saving a remote file remains subject to size, text/binary, revision/conflict, path confinement, permission-preservation and read-back verification safeguards.

## Bandwidth behavior after installation

Upload/download ceilings are persisted local settings in binary KiB/s. `0` is unlimited. The configured directional budget is applied to actual FTP/FTPS/SFTP transport work, not merely displayed in UI. A running transfer attempt retains the budget sampled at start; later attempts observe newly saved values.

## Verification

Before using an official 0.0.4 package:

1. confirm the requested version is `0.0.4`;
2. verify the file is one of the canonical artifact names above;
3. verify its SHA-256 against `SHA256.txt`;
4. inspect `BUILD-METADATA.txt` for source commit and Windows signing state;
5. where metadata says `signed`, require a valid Authenticode signature on Windows;
6. distinguish native lifecycle coverage from build/parity coverage for non-x86-64 Linux artifacts.

After successful publication, the verified distribution bundle is available at `ghcr.io/bren-wp/ghost-ftp:0.0.4`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
