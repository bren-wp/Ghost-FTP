# Ghost FTP installation

Ghost FTP **1.1.8 Stable** is the current published stable release. Use only official artifacts whose version and SHA-256 values match the corresponding GitHub Release. Historical Stable releases remain immutable and are not rewritten by later source or packaging work.

The official product website is **https://ghostftp.com**.

## Windows

### Setup packages

Canonical 1.1.8 files:

```text
Ghost-FTP-1.1.8-Setup-x64.exe
Ghost-FTP-1.1.8-Setup-x86.exe
Ghost-FTP-1.1.8-Setup-x32.exe
```

`x32` is a compatibility alias of the x86 Setup file and is byte-identical to it. Setup is a per-user installation/maintenance application. It stages and validates payloads before replacement, keeps rollback state through the transaction, writes the maintained uninstall registration and supports update/uninstall from the installed product path.

Ghost FTP 1.1.8 additionally requires installer-directory identity to remain bound through backup, activation, rollback and cleanup. Legacy uninstaller cleanup, shortcuts and integrated uninstall use explicit ownership/executable identity rather than same-name pathname assumptions.

### Portable packages

```text
Ghost-FTP-1.1.8-Portable-x64.exe
Ghost-FTP-1.1.8-Portable-x86.exe
```

Portable mode does not create normal Setup registration. Do not manually mix installed and portable state directories.

### Windows signing state

Production Authenticode signing is optional. Read `BUILD-METADATA.txt` from the same official release.

`WINDOWS_AUTHENTICODE=signed` means a configured trusted production certificate was used and the produced signatures were verified. `WINDOWS_AUTHENTICODE=unsigned` means the official Windows artifacts are intentionally unsigned. The production workflow does not create a self-signed identity merely to make files appear signed.

In both cases verify `SHA256.txt`. For a signed release, also verify the Authenticode publisher.

## Linux

### Canonical release packages

The canonical 1.1.8 Linux release files are:

```text
Ghost-FTP-1.1.8-Linux-amd64.deb
Ghost-FTP-1.1.8-Linux-arm64.deb
Ghost-FTP-1.1.8-Linux-i386.deb
Ghost-FTP-1.1.8-Linux-multiarch.zip
Ghost-FTP-1.1.8-Linux-amd64.tar.gz
Ghost-FTP-1.1.8-Linux-arm64.tar.gz
Ghost-FTP-1.1.8-Linux-i386.tar.gz
```

Install the DEB matching the machine architecture with the system package manager, or use the matching generic tar.gz when a package-manager-neutral portable archive is appropriate. DEB metadata uses the Ghost FTP product identity and `Homepage: https://ghostftp.com`.

The maintained canonical release workflow uses `linux/BUILD.sh`. For `amd64`, `arm64` and `i386`, it builds a generic DEB and package-manager-neutral `.tar.gz` from the same compiled executable. Production verification extracts both formats and requires the portable `ghostftp` executable to be byte-identical to `/usr/bin/ghostftp` from its matching DEB.

Together with the Windows artifacts, Linux multiarch ZIP and three metadata/checksum files, Ghost FTP 1.1.8 preserves the canonical **12 platform artifacts / 15 public files** release shape.

### Linux executable provenance

Ghost FTP 1.1.8 hardens Linux transport discovery so `curl`, `ssh`, `sftp` and `ssh-keyscan` are not trusted merely because they appear in `PATH`. Accepted executables must prove trusted root-controlled filesystem provenance. Credential-bearing OpenSSH AskPass also fails closed if the required executable/parent provenance cannot be established.

### Supplemental distro-specific source/CI packages

The maintained source separately provides `linux/BUILD-DISTROS.sh`. It creates distribution-labelled CI packages in addition to the canonical release path:

```text
Ghost-FTP-X.Y.Z-Linux-Debian-amd64.deb
Ghost-FTP-X.Y.Z-Linux-Debian-arm64.deb
Ghost-FTP-X.Y.Z-Linux-Debian-i386.deb

Ghost-FTP-X.Y.Z-Linux-Ubuntu-amd64.deb
Ghost-FTP-X.Y.Z-Linux-Ubuntu-arm64.deb
Ghost-FTP-X.Y.Z-Linux-Ubuntu-i386.deb

Ghost-FTP-X.Y.Z-Linux-Fedora-x86_64.rpm
Ghost-FTP-X.Y.Z-Linux-Fedora-aarch64.rpm
Ghost-FTP-X.Y.Z-Linux-Fedora-i686.rpm

Ghost-FTP-X.Y.Z-Linux-Portable-amd64.tar.gz
Ghost-FTP-X.Y.Z-Linux-Portable-arm64.tar.gz
Ghost-FTP-X.Y.Z-Linux-Portable-i386.tar.gz
```

Architecture mapping is `amd64 -> x86_64`, `arm64 -> aarch64`, and `i386 -> i686`. The distro packaging workflow verifies metadata and byte-for-byte executable parity between matching Debian, Ubuntu, Fedora and Portable package payloads.

These are supplemental maintained CI artifacts and are **not yet part of the canonical release allow-list**. Build support and CI verification must not be represented as public release publication until a canonical release workflow explicitly stages, allow-lists, hashes, publishes and reads those files back.

### Native distro install verification

The separate install matrix performs a real package-manager lifecycle and installed-GUI smoke verification on:

```text
Debian 13 amd64
Ubuntu 26.04 LTS amd64
Fedora 44 x86_64
```

For those targets it verifies package metadata, dependency resolution, installed package state, runtime tools, package-owned files, startup of installed `/usr/bin/ghostftp` under local Xvfb, package removal and absence of package-owned system residue.

Native package-manager/runtime install coverage is intentionally **x86-64 only**. arm64/aarch64 and i386/i686 package families remain protected by exact-head build, metadata, extraction and byte-parity verification but are not claimed as native install-tested.

See [Linux documentation](../linux/README.md) and [Testing](TESTING.md).

## Upgrade to 1.1.8

Ghost FTP 1.1.8 is a backward-compatible 1.x maintenance release. Existing 1.x local settings and profiles are intended to remain compatible.

The release strengthens privacy-safe diagnostics, Linux transport/AskPass provenance, state-directory identity, Windows installer/uninstaller/shortcut ownership and small-display/mixed-DPI geometry. It does not intentionally change FTP/FTPS/SFTP protocol selection, profile format, canonical artifact count or credential-retain/remove semantics.

Before upgrading critical systems, keep an appropriate backup of local configuration and verify the Stable package checksum and, when present, its signing state.

Windows Setup performs staged replacement with rollback-oriented transaction behavior. Linux DEB upgrades use normal package-manager semantics. Generic tar.gz users should replace only the files supplied by that archive.

## First connection defaults

A fresh/quick connection starts with **explicit FTPS on port 21**. Plain FTP remains available when a legacy server explicitly requires unencrypted FTP, and SFTP remains available for SSH-based transfer. Failed FTPS negotiation is not silently downgraded to FTP.

## Saved credentials

Saved credentials remain local and opt-in. The main Save Profile flow and Windows Site Manager require explicit consent before newly entered password/private-key passphrase values are persisted. Saving non-secret profile details does not itself authorize credential storage.

Changing server/account/private-key identity can clear credentials that no longer belong to that identity. If a protected secret cannot be decrypted under the current user/device context, Ghost FTP requires the credential again instead of weakening protection.

## GitHub Packages

Ghost FTP 1.1.8 is mirrored as:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.8
```

This OCI object is a verified distribution bundle containing `/ghostftp-release/`; it is not a runtime container and is not the normal desktop installation path. See [Packages](PACKAGES.md).

## Verifying files

Each GitHub Release contains `SHA256.txt`. Compare the checksum of every downloaded installer/package before use. `BUILD-METADATA.txt` binds version, source commit, release tag, platform set, Windows signing state and GitHub Package reference.

For supplemental source/CI distro packages, CI success proves the repository packaging contract for the tested revision. It does not turn a CI artifact into an official published release file.

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated permanent external uninstaller.

Ghost FTP removes only artifacts for which the maintained ownership/identity contract succeeds. A foreign same-name shortcut/file or unowned Start Menu parent directory is preserved.

### Linux

Remove the `ghost-ftp` DEB with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data. For a manually installed package-manager-neutral tarball, remove only files installed from that archive.

## Troubleshooting

If Windows shows an unknown-publisher or SmartScreen warning, inspect `BUILD-METADATA.txt`. If it says `WINDOWS_AUTHENTICODE=unsigned`, verify the official release location and `SHA256.txt`. If metadata says `signed` but signature validation fails, treat that as a release-integrity failure.

For Linux architecture errors, use the documented package/architecture mapping. Do not infer native install verification merely because a package successfully builds.

For SFTP password/passphrase failures on Linux, do not bypass the trusted executable/AskPass checks. Verify that required system OpenSSH tools and the installed Ghost FTP executable satisfy the documented provenance boundary.

For connection failures, verify protocol, host, port, server policy and system transfer-tool availability without placing real credentials in issue reports.

## Production deployment checklist

1. download from the official stable GitHub Release;
2. verify release tag/version;
3. verify `SHA256.txt`;
4. inspect `WINDOWS_AUTHENTICODE` and verify Authenticode when the release is signed;
5. choose the correct architecture/package family;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode and do not bypass failed TLS by silently switching protocols;
8. keep private credentials out of logs and support reports;
9. do not represent supplemental CI artifacts as published release assets.

The immutable 1.1.7 tag, assets, checksums and GHCR digest remain historical and are not rewritten by the 1.1.8 release.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Platform parity](PLATFORM-PARITY.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
