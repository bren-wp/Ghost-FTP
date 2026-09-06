# Ghost FTP installation

Ghost FTP **1.0.0 Stable** ships as native Windows and Linux packages. Use only official artifacts whose version and SHA-256 values match the corresponding GitHub Release.

## Windows

### Setup packages

```text
Ghost-FTP-1.0.0-Setup-x64.exe
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Setup-x32.exe
```

`x32` is a compatibility alias of the x86 Setup file and is byte-identical to it.

Setup is a per-user installation/maintenance application. It stages and validates payloads before replacement, keeps rollback state through the transaction, writes the maintained uninstall registration and supports update/uninstall from the installed product path.

### Portable packages

```text
Ghost-FTP-1.0.0-Portable-x64.exe
Ghost-FTP-1.0.0-Portable-x86.exe
```

Portable mode does not create the normal Setup registration and keeps its portable state boundary beside the application as documented by the product. Do not mix an installed data directory and portable data directory manually.

### Stable signing

Stable Windows publication is blocked unless the protected release environment supplies the configured trusted Authenticode identity. Verify the Windows digital signature and `SHA256.txt` before deployment.

## Linux

Official Debian packages are:

```text
Ghost-FTP-1.0.0-Linux-amd64.deb
Ghost-FTP-1.0.0-Linux-arm64.deb
Ghost-FTP-1.0.0-Linux-i386.deb
```

A convenience archive contains all three verified packages:

```text
Ghost-FTP-1.0.0-Linux-multiarch.zip
```

Install the package matching the machine architecture with the system package manager. The package installs the `ghostftp` application and desktop integration expected by the maintained Linux build.

## Upgrade to 1.0.0

Ghost FTP 1.0.0 is the first stable release. Existing 0.2.x local settings/profiles are intended to remain compatible. Before upgrading critical systems, keep a copy of local configuration and verify the stable package checksum/signature.

Windows Setup performs a staged replacement and rollback-oriented transaction. Linux upgrades use the standard package manager semantics of the DEB package.

## Saved credentials

Saved credentials remain local and are opt-in. An upgrade must not require exporting plaintext passwords. Windows uses the current-user protection boundary; Linux uses local authenticated encryption with user-private key material.

If a protected secret cannot be decrypted under the current user/device context, Ghost FTP must require the credential again rather than silently weakening protection.

## GitHub Packages

The stable release is also mirrored as:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

This OCI object is a verified distribution bundle containing `/ghostftp-release/`; it is not a runtime container and is not the normal desktop installation path. See [Packages](PACKAGES.md).

## Verifying files

Each GitHub Release contains `SHA256.txt`. Compare the checksum of every downloaded installer/package before use.

The corresponding `BUILD-METADATA.txt` binds the version, source commit, release tag, platform set, signing state and GitHub Package reference.

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated external uninstaller.

### Linux

Remove the `ghost-ftp` package with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data unless the product explicitly offers that operation.

## Troubleshooting

### Windows signature validation fails

Do not bypass stable signature verification. Re-download from the official Release, verify `SHA256.txt`, and confirm that the release itself is the expected `ghostftp-v1.0.0` tag.

### Linux package architecture mismatch

Use `amd64`, `arm64` or `i386` according to the target host. Do not force an incompatible DEB architecture.

### Saved profile cannot decrypt

Confirm that the same operating-system user and local secret-protection state are being used. Re-enter the password/passphrase if the original protected context is unavailable.

### Connection fails after installation

Use the privacy-safe connection diagnostics in Ghost FTP. Verify protocol, host, port, server policy and system transfer-tool availability without pasting real credentials into issue reports.

## Production deployment checklist

1. download from the official stable GitHub Release;
2. verify release tag/version;
3. verify `SHA256.txt`;
4. verify Windows Authenticode when installing on Windows;
5. choose the correct architecture;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode;
8. keep private credentials out of logs and support reports.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
