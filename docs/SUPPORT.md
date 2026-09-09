# Ghost FTP support

Ghost FTP **0.0.1 Beta** is the current supported public release.

## Before reporting a problem

1. Confirm the application reports version `0.0.1 Beta`.
2. Confirm the package came from the current GitHub Release.
3. Verify the artifact against `SHA256.txt`.
4. Reproduce with the same protocol (FTP, FTPS or SFTP), architecture and operating system.
5. Remove real passwords, passphrases, private keys, server secrets and customer data from any diagnostic material.

Because the project keeps only the latest public release, support is provided against the current version rather than superseded release/tag URLs.

## Useful report details

Include privacy-safe information such as:

- Windows or Linux distribution/version;
- CPU architecture;
- Ghost FTP version;
- protocol selection;
- whether the issue occurs on connect, list, upload, download, Remote Edit, profile handling or uninstall;
- exact user-visible error category without copying secret-bearing raw server output;
- whether the behavior reproduces with a synthetic test server/file.

Do not post real credentials, private-key contents, saved profile secrets, signing keys or production customer data.

## Windows signing reports

Always **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**.

When metadata says:

```text
WINDOWS_AUTHENTICODE=unsigned
```

the official file is explicitly `unsigned`; do not report the mere absence of a publisher signature as corruption.

When metadata says `signed` and Windows signature verification fails, treat that as a release-integrity problem and report it with the artifact name and SHA-256 value.

## Remote Edit reports

For Remote Edit issues, include:

- file size class and text encoding if known;
- line-ending style (LF/CRLF/CR) if relevant;
- whether the file changed on the server between open and save;
- whether the save succeeded but list metadata did not refresh;
- whether permissions changed unexpectedly.

Never attach the real sensitive remote file if it contains secrets. Reproduce with synthetic text where possible.

## Security and privacy issues

For credential exposure, trust-verification bypass, path containment, installer/uninstaller ownership, or other security-sensitive reports, avoid publishing exploit-sensitive secrets or live credentials in a public issue. Provide the minimum synthetic reproduction needed to identify the problem.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.1`. After a newer release is verified, old Ghost FTP releases/tags are intentionally removed by the latest-only retention policy. Git commit history remains available for engineering provenance but is not a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
