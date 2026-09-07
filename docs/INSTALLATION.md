# Ghost FTP installation

Ghost FTP **1.1.5 Stable** is the current source candidate and ships as native Windows and Linux packages after the complete release gate succeeds. Use only official artifacts whose version and SHA-256 values match the corresponding published GitHub Release. Published 1.1.4 and earlier releases remain immutable historical releases and are not rewritten.

The official Ghost FTP product website is **https://ghostftp.com**. Ghost FTP is developed and published by **BRENDIGO LTD**; the author's official website is **https://brendigo.com**.

## Windows

### Setup packages

```text
Ghost-FTP-1.1.5-Setup-x64.exe
Ghost-FTP-1.1.5-Setup-x86.exe
Ghost-FTP-1.1.5-Setup-x32.exe
```

`x32` is a compatibility alias of the x86 Setup file and is byte-identical to it.

Setup is a per-user installation/maintenance application. It stages and validates payloads before replacement, keeps rollback state through the transaction, writes the maintained uninstall registration and supports update/uninstall from the installed product path.

### Portable packages

```text
Ghost-FTP-1.1.5-Portable-x64.exe
Ghost-FTP-1.1.5-Portable-x86.exe
```

Portable mode does not create the normal Setup registration and keeps its portable state boundary beside the application as documented by the product. Do not manually mix installed and portable state directories.

### Windows signing state

Production Authenticode signing is optional. Read `BUILD-METADATA.txt` from the same official release.

```text
WINDOWS_AUTHENTICODE=signed
```

means the release workflow used a configured trusted production certificate and verified the produced signatures.

```text
WINDOWS_AUTHENTICODE=unsigned
```

means the official Windows artifacts are intentionally unsigned. The workflow does not generate a self-signed production identity merely to make the files appear signed.

In both cases verify `SHA256.txt`. For a signed release, also verify the Authenticode publisher. Unsigned builds may trigger Windows SmartScreen/publisher warnings depending on local policy.

## Linux

Official Debian packages for 1.1.5 are:

```text
Ghost-FTP-1.1.5-Linux-amd64.deb
Ghost-FTP-1.1.5-Linux-arm64.deb
Ghost-FTP-1.1.5-Linux-i386.deb
```

A convenience archive contains all three verified packages:

```text
Ghost-FTP-1.1.5-Linux-multiarch.zip
```

Install the package matching the machine architecture with the system package manager. The package installs `ghostftp` and the maintained Linux desktop integration. DEB metadata uses `Homepage: https://ghostftp.com` and the BRENDIGO LTD publisher identity.

## Upgrade to 1.1.5

Ghost FTP 1.1.5 is a backward-compatible 1.x maintenance release. Existing 1.x local settings and profiles are intended to remain compatible. The release preserves explicit FTPS/SFTP security behavior, local protected-secret handling, transfer rollback safeguards and the existing appearance migration policy.

Before upgrading critical systems, keep an appropriate backup of local configuration and verify the stable package checksum and, when present, its signing state.

Windows Setup performs a staged replacement with rollback-oriented transaction behavior. Linux upgrades use standard DEB package-manager semantics.

## First connection defaults

A fresh/quick connection starts with **explicit FTPS on port 21**. Plain FTP remains available when a legacy server explicitly requires unencrypted FTP, and SFTP remains available for SSH-based transfer. A failed FTPS negotiation is not silently downgraded to FTP.

## Saved credentials

Saved credentials remain local and are opt-in. An upgrade must not require exporting plaintext passwords. Windows uses the current-user protection boundary; Linux uses the documented local protected storage model with bounded runtime secret handling.

The main Save Profile flow and Windows Site Manager require explicit consent before newly entered password/private-key passphrase values are persisted. Saving non-secret profile details does not itself authorize credential storage.

If a protected secret cannot be decrypted under the current user/device context, Ghost FTP must require the credential again rather than silently weakening protection.

## GitHub Packages

After successful stable publication, the 1.1.5 release is mirrored as:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.5
```

This OCI object is a verified distribution bundle containing `/ghostftp-release/`; it is not a runtime container and is not the normal desktop installation path. See [Packages](PACKAGES.md).

## Verifying files

Each GitHub Release contains `SHA256.txt`. Compare the checksum of every downloaded installer/package before use.

The corresponding `BUILD-METADATA.txt` binds the version, source commit, release tag, platform set, Windows signing state and GitHub Package reference.

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated external uninstaller.

### Linux

Remove the `ghost-ftp` package with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data unless the product explicitly offers that operation.

## Troubleshooting

### Windows shows an unknown-publisher or SmartScreen warning

First inspect `BUILD-METADATA.txt`. If it says `WINDOWS_AUTHENTICODE=unsigned`, the missing trusted signature is expected for that release. Verify the official release tag and `SHA256.txt`, and follow Windows/organization security policy rather than disabling protections globally.

If metadata says `WINDOWS_AUTHENTICODE=signed` but signature validation fails, re-download from the official Release and treat the mismatch as a verification failure.

### Linux package architecture mismatch

Use `amd64`, `arm64` or `i386` according to the target host. Do not force an incompatible DEB architecture.

### Saved profile cannot decrypt

Confirm that the same operating-system user and local secret-protection state are being used. Re-enter the password/passphrase if the original protected context is unavailable.

### Connection fails after installation

Use the privacy-safe connection diagnostics in Ghost FTP. Verify protocol, host, port, server policy and system transfer-tool availability without placing real credentials in issue reports. For a fresh connection, verify whether the server supports explicit FTPS/21 before selecting plain FTP for legacy compatibility.

## Production deployment checklist

1. download from the official stable GitHub Release;
2. verify release tag/version;
3. verify `SHA256.txt`;
4. inspect `WINDOWS_AUTHENTICODE` and verify Authenticode when the release is signed;
5. choose the correct architecture;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode and do not bypass failed TLS by silently switching protocols;
8. keep private credentials out of logs and support reports.

The 1.1.5 filenames above describe the candidate contract; they are not proof of publication. Treat 1.1.5 as downloadable only after the official tag, GitHub Release asset set and GHCR package read-back have succeeded.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
