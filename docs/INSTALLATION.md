# Ghost FTP installation

Ghost FTP **1.1.6 Stable** is the current published stable release. Use only official artifacts whose version and SHA-256 values match the corresponding published GitHub Release. Published Stable releases through 1.1.6 are immutable historical release identities and are not rewritten by later source or packaging work.

The official Ghost FTP product website is **https://ghostftp.com**. Ghost FTP is developed and published by **BRENDIGO LTD**; the author's official website is **https://brendigo.com**.

## Windows

### Setup packages

The published 1.1.6 Windows files are:

```text
Ghost-FTP-1.1.6-Setup-x64.exe
Ghost-FTP-1.1.6-Setup-x86.exe
Ghost-FTP-1.1.6-Setup-x32.exe
```

`x32` is a compatibility alias of the x86 Setup file and is byte-identical to it.

Setup is a per-user installation/maintenance application. It stages and validates payloads before replacement, keeps rollback state through the transaction, writes the maintained uninstall registration and supports update/uninstall from the installed product path.

### Portable packages

```text
Ghost-FTP-1.1.6-Portable-x64.exe
Ghost-FTP-1.1.6-Portable-x86.exe
```

Portable mode does not create the normal Setup registration and keeps its portable state boundary beside the application as documented by the product. Do not manually mix installed and portable state directories.

### Windows signing state

Production Authenticode signing is optional. Read `BUILD-METADATA.txt` from the same official release.

`WINDOWS_AUTHENTICODE=signed` means the release workflow used a configured trusted production certificate and verified the produced signatures. `WINDOWS_AUTHENTICODE=unsigned` means the official Windows artifacts are intentionally unsigned. The workflow does not generate a self-signed production identity merely to make the files appear signed.

In both cases verify `SHA256.txt`. For a signed release, also verify the Authenticode publisher. Unsigned builds may trigger Windows SmartScreen/publisher warnings depending on local policy.

## Linux

### Published 1.1.6 packages

The published Linux files for 1.1.6 are:

```text
Ghost-FTP-1.1.6-Linux-amd64.deb
Ghost-FTP-1.1.6-Linux-arm64.deb
Ghost-FTP-1.1.6-Linux-i386.deb
Ghost-FTP-1.1.6-Linux-multiarch.zip
```

Install the DEB matching the machine architecture with the system package manager. The package installs `ghostftp` and the maintained Linux desktop integration. DEB metadata uses `Homepage: https://ghostftp.com` and the BRENDIGO LTD publisher identity.

These four Linux files are the immutable published 1.1.6 asset set. Later source/CI packaging work does not add files retroactively to the 1.1.6 GitHub Release, checksum manifest or GHCR bundle.

### Canonical next-release source packages

The maintained canonical release workflow still uses `linux/BUILD.sh`. It builds generic DEBs and package-manager-neutral `.tar.gz` archives for `amd64`, `arm64` and `i386` from the same compiled executable per architecture.

The generic portable archives are post-1.1.6 outputs and are not retroactive 1.1.6 release assets. The current next-release assembly contract verifies them and includes them in the canonical future **12 platform artifacts / 15 public files** release shape only when that later version passes the complete release workflow.

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

The matching architecture mapping is `amd64 -> x86_64`, `arm64 -> aarch64`, and `i386 -> i686`. The distro packaging workflow verifies metadata and byte-for-byte executable parity between the matching Debian, Ubuntu, Fedora and Portable package payloads.

These are **supplemental maintained CI artifacts**. They are not published 1.1.6 files and are **not yet part of the canonical release allow-list** in `.github/workflows/release.yml`. Build support and CI verification must not be described as public release publication until a later release workflow explicitly stages, hashes, publishes and reads those files back.

### Native distro install verification

The separate install matrix performs real package-manager lifecycle and installed-GUI smoke verification on:

```text
Debian 13 amd64
Ubuntu 26.04 LTS amd64
Fedora 44 x86_64
```

For those exact targets it verifies package metadata, dependency resolution, installed package state, runtime tools, package-owned files, startup of installed `/usr/bin/ghostftp` under local Xvfb, package removal and absence of package-owned system residue.

Debian and Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`. Fedora RPMs declare `ca-certificates`, `curl` and `openssh-clients`. Fedora may satisfy the `curl` capability through a provider such as `curl-minimal`; the verifier therefore validates the installed executable and its RPM owner instead of assuming one provider package name.

Native package-manager/runtime install coverage is intentionally **x86-64 only**. The arm64/aarch64 and i386/i686 package families are still covered by exact-head build, metadata, extraction and byte-parity verification, but are not claimed as native install-tested.

See [Linux documentation](../linux/README.md) and [Testing](TESTING.md) for the exact build/install gate boundaries.

## Upgrade to 1.1.6

Ghost FTP 1.1.6 is a backward-compatible 1.x maintenance release. Existing 1.x local settings and profiles are intended to remain compatible. The release strengthens local filesystem race handling, SFTP fingerprint binding and remote staging cleanup proof without weakening FTPS/SFTP trust or overwrite/rollback safeguards.

Before upgrading critical systems, keep an appropriate backup of local configuration and verify the stable package checksum and, when present, its signing state.

Windows Setup performs a staged replacement with rollback-oriented transaction behavior. Linux upgrades using the published 1.1.6 DEBs use standard DEB package-manager semantics.

## First connection defaults

A fresh/quick connection starts with **explicit FTPS on port 21**. Plain FTP remains available when a legacy server explicitly requires unencrypted FTP, and SFTP remains available for SSH-based transfer. A failed FTPS negotiation is not silently downgraded to FTP.

## Saved credentials

Saved credentials remain local and are opt-in. An upgrade must not require exporting plaintext passwords. Windows uses the current-user protection boundary; Linux uses the documented local protected storage model with bounded runtime secret handling.

The main Save Profile flow and Windows Site Manager require explicit consent before newly entered password/private-key passphrase values are persisted. Saving non-secret profile details does not itself authorize credential storage.

If a protected secret cannot be decrypted under the current user/device context, Ghost FTP must require the credential again rather than silently weakening protection.

## GitHub Packages

The published 1.1.6 release is mirrored as:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.6
```

This OCI object is a verified distribution bundle containing `/ghostftp-release/`; it is not a runtime container and is not the normal desktop installation path. See [Packages](PACKAGES.md).

## Verifying files

Each GitHub Release contains `SHA256.txt`. Compare the checksum of every downloaded installer/package before use. `BUILD-METADATA.txt` binds the version, source commit, release tag, platform set, Windows signing state and GitHub Package reference.

For supplemental source/CI distro packages, CI success proves the repository packaging contract for the tested revision. It does not turn a CI artifact into an official published release file.

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated external uninstaller.

### Linux

Remove the `ghost-ftp` package with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data unless the product explicitly offers that operation.

For a manually installed package-manager-neutral source/CI tarball, remove only the files that were installed from that archive; do not treat this as a package-manager transaction.

## Troubleshooting

If Windows shows an unknown-publisher or SmartScreen warning, first inspect `BUILD-METADATA.txt`. If it says `WINDOWS_AUTHENTICODE=unsigned`, verify the official release location and `SHA256.txt`. If metadata says `signed` but signature validation fails, treat that as a release-integrity failure.

For Linux architecture errors, use the package naming/architecture mapping documented above. Do not infer native install verification for an architecture merely because a package successfully builds.

For saved-profile decryption failures, re-enter the credential under the correct operating-system user/protection context.

For connection failures, verify protocol, host, port, server policy and system transfer-tool availability without placing real credentials in issue reports.

## Production deployment checklist

1. download from the official stable GitHub Release when installing a published Stable version;
2. verify release tag/version;
3. verify `SHA256.txt`;
4. inspect `WINDOWS_AUTHENTICODE` and verify Authenticode when the release is signed;
5. choose the correct architecture;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode and do not bypass failed TLS by silently switching protocols;
8. keep private credentials out of logs and support reports;
9. do not represent supplemental CI artifacts as published release assets.

Ghost FTP 1.1.6 is already published as Stable. Its official tag, Release asset set and GHCR distribution bundle were read back by the canonical production workflow; later source/CI improvements do not rewrite that historical release.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Platform parity](PLATFORM-PARITY.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
