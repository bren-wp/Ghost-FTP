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

The published Linux files for 1.1.6 are:

```text
Ghost-FTP-1.1.6-Linux-amd64.deb
Ghost-FTP-1.1.6-Linux-arm64.deb
Ghost-FTP-1.1.6-Linux-i386.deb
Ghost-FTP-1.1.6-Linux-multiarch.zip
```

Install the DEB matching the machine architecture with the system package manager. The package installs `ghostftp` and the maintained Linux desktop integration. DEB metadata uses `Homepage: https://ghostftp.com` and the BRENDIGO LTD publisher identity.

Post-1.1.6 source/CI builds additionally create verified package-manager-neutral `.tar.gz` archives for amd64, arm64 and i386. Those archives are not retroactive 1.1.6 release assets; see the Linux documentation for source-build and portable-installation instructions.

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

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated external uninstaller.

### Linux

Remove the `ghost-ftp` package with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data unless the product explicitly offers that operation.

For a manually installed package-manager-neutral source/CI tarball, remove only the files that were installed from that archive; do not treat this as a package-manager transaction.

## Troubleshooting

If Windows shows an unknown-publisher or SmartScreen warning, first inspect `BUILD-METADATA.txt`. If it says `WINDOWS_AUTHENTICODE=unsigned`, verify the official release location and `SHA256.txt`. If metadata says `signed` but signature validation fails, treat that as a release-integrity failure.

For Linux architecture errors, use `amd64`, `arm64` or `i386` according to the target host. For saved-profile decryption failures, re-enter the credential under the correct operating-system user/protection context.

For connection failures, verify protocol, host, port, server policy and system transfer-tool availability without placing real credentials in issue reports.

## Production deployment checklist

1. download from the official stable GitHub Release;
2. verify release tag/version;
3. verify `SHA256.txt`;
4. inspect `WINDOWS_AUTHENTICODE` and verify Authenticode when the release is signed;
5. choose the correct architecture;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode and do not bypass failed TLS by silently switching protocols;
8. keep private credentials out of logs and support reports.

Ghost FTP 1.1.6 is already published as Stable. Its official tag, Release asset set and GHCR distribution bundle were read back by the canonical production workflow; later source/CI improvements do not rewrite that historical release.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
