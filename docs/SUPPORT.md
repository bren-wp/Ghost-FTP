# Ghost FTP support

Ghost FTP **0.0.3** is the current supported public release.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm the application reports version `0.0.3`.
2. Confirm the package came from the current GitHub Release.
3. Verify the artifact against `SHA256.txt`.
4. Reproduce with the same protocol (FTP, FTPS or SFTP), architecture and operating system.
5. Remove passwords, passphrases, private keys, server secrets and customer data from diagnostic material.

Useful privacy-safe details include Windows/Linux version, CPU architecture, Ghost FTP version, protocol, operation being performed, configured upload/download KiB/s limits when transfer speed is involved, and the exact user-visible error category without copying secret-bearing raw server output.

## Windows signing reports

Always **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**. When the official file is explicitly `unsigned`, absence of a publisher signature is not corruption. When metadata says `signed` and Windows signature verification fails, treat that as a release-integrity problem and report the artifact name plus SHA-256.

## Security and privacy issues

For credential exposure, trust-verification bypass, path containment, installer/uninstaller ownership or related security-sensitive reports, use synthetic reproductions and never publish live credentials/private keys/customer data.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.3` with `prerelease=false`. The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.3`. The project retains only the latest verified public release/package identity after successor verification; Git commit history remains engineering provenance.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
