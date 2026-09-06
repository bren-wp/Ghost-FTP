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

### Windows signing state

Production Authenticode signing is optional. Read `BUILD-METADATA.txt` from the same official release:

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

Ghost FTP 1.0.0 is the first stable release. Existing 0.2.x local settings/profiles are intended to remain compatible. Before upgrading critical systems, keep a copy of local configuration and verify the stable package checksum and, when present, its signing state.

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

### Windows shows an unknown-publisher or SmartScreen warning

First inspect `BUILD-METADATA.txt`. If it says `WINDOWS_AUTHENTICODE=unsigned`, the missing trusted signature is expected for that release. Verify the official release tag and `SHA256.txt`, and follow your Windows/organization security policy rather than disabling protections globally.

If metadata says `WINDOWS_AUTHENTICODE=signed` but signature validation fails, re-download from the official Release and treat the mismatch as a verification failure.

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
4. inspect `WINDOWS_AUTHENTICODE` and verify Authenticode when the release is signed;
5. choose the correct architecture;
6. preserve needed local configuration before upgrade;
7. test the target server using its intended FTP/FTPS/SFTP mode;
8. keep private credentials out of logs and support reports.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
