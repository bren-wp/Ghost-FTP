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

Portable mode does not create the normal Setup registration and keeps its portable state boundary separate from an installed copy.

### Windows signing/trust status

Always read `BUILD-METADATA.txt` from the same Release before deciding how to verify a Windows file.

When the release says:

```text
WINDOWS_AUTHENTICODE=signed
```

verify both Authenticode and `SHA256.txt`.

When it says:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

no production signer is claimed. Verify the official GitHub Release/tag, source commit and `SHA256.txt`. Do not bypass a local enterprise policy that requires signed executables; use a release whose metadata/signature satisfies that policy.

Ghost FTP never substitutes a self-signed/development certificate and presents it as a trusted production publisher identity.

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

Install the package matching the machine architecture with the system package manager. The package installs the `ghostftp` application and maintained desktop integration.

## Upgrade to 1.0.0

Ghost FTP 1.0.0 is the first stable release. Existing 0.2.x local settings/profiles are intended to remain compatible. Before upgrading critical systems, keep a copy of local configuration and verify release provenance/checksums.

Windows Setup performs a staged replacement and rollback-oriented transaction. Linux upgrades use normal DEB/package-manager semantics.

## Saved credentials

Saved credentials remain local and are opt-in. An upgrade must not require exporting plaintext passwords. Windows uses the current-user protection boundary; Linux uses local authenticated encryption with user-private key material.

If a protected secret cannot be decrypted under the current user/device context, Ghost FTP requires the credential again rather than silently weakening protection.

## GitHub Packages

The stable release is also mirrored as:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

This OCI object is a verified distribution bundle containing `/ghostftp-release/`; it is not a runtime container and is not the normal desktop installation path. See [Packages](PACKAGES.md).

## Verifying files

Each GitHub Release contains `SHA256.txt`. Compare the checksum of every downloaded installer/package before use.

`BUILD-METADATA.txt` binds the version, source commit, release tag, Windows signing/trust state, platform set and GitHub Package reference.

## Uninstall

### Windows

Use the registered Ghost FTP uninstall entry. The integrated maintenance/uninstall path belongs to the installed Ghost FTP binary/Setup transaction and does not depend on an unrelated external uninstaller.

### Linux

Remove the `ghost-ftp` package with the distribution package manager. User-local profiles/settings are separate data; removing application binaries does not imply deletion of all user data unless the product explicitly offers that operation.

## Troubleshooting

### Windows file is unsigned

First check `BUILD-METADATA.txt`. If the Release explicitly states `WINDOWS_AUTHENTICODE=unsigned`, compare the file against `SHA256.txt` and verify the official Release/tag/source commit. If your local policy requires an Authenticode-trusted publisher, do not disable that policy; wait for/use a release with a valid production signature.

### Windows signature validation fails on a release marked signed

Treat this as a verification failure. Re-download from the official Release, verify `SHA256.txt` and confirm the expected tag. Do not ignore a signature failure when metadata claims the artifact is signed.

### Linux package architecture mismatch

Use `amd64`, `arm64` or `i386` according to the target host. Do not force an incompatible DEB architecture.

### Saved profile cannot decrypt

Confirm that the same operating-system user and local secret-protection state are being used. Re-enter the password/passphrase if the original protected context is unavailable.

### Connection fails after installation

Use the privacy-safe connection diagnostics in Ghost FTP. Verify protocol, host, port, server policy and system transfer-tool availability without pasting real credentials into public issue reports.

## Production deployment checklist

1. download from the official stable GitHub Release;
2. verify `ghostftp-v1.0.0`, title and `prerelease=false`;
3. verify `SHA256.txt`;
4. read the actual Windows signing/trust state in `BUILD-METADATA.txt`;
5. if marked signed, verify Authenticode; if marked unsigned, apply your organization policy and GitHub/SHA-256 provenance checks;
6. choose the correct architecture;
7. preserve needed local configuration before upgrade;
8. test the intended FTP/FTPS/SFTP server mode;
9. keep private credentials out of logs and support reports.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
