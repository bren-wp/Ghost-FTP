# Ghost FTP support

Ghost FTP **0.0.3** is the current supported public release.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm the application reports version `0.0.3`.
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
- exact package/artifact name when the issue is installation or startup related;
- protocol selection;
- configured upload/download KiB/s limits when transfer throughput is involved;
- configured parallelism when aggregate bandwidth or queue behavior is involved;
- whether the issue occurs on connect, list, upload, download, current-folder filter, recursive search, directory comparison, synchronized navigation, Remote Edit, profile handling, Setup/Portable bootstrap selection or uninstall;
- exact user-visible error category without copying secret-bearing raw server output;
- whether the behavior reproduces with a synthetic test server/file.

Do not post real credentials, private-key contents, saved profile secrets, signing keys or production customer data.

## Windows universal-artifact reports

The current public Windows files are `Ghost-FTP-0.0.3-Setup.exe` and `Ghost-FTP-0.0.3-Portable.exe`. There are no supported public `-x64.exe`, `-x86.exe` or `-x32.exe` downloads in the 0.0.3 release contract.

For startup/bootstrap issues, report the public artifact name, Windows version, CPU/native architecture and SHA-256. Do not extract or redistribute internal staging payloads as substitutes for public release files.

## Windows signing reports

Always **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**.

When metadata says:

```text
WINDOWS_AUTHENTICODE=unsigned
```

the official file is explicitly `unsigned`; do not report the mere absence of a publisher signature as corruption.

When metadata says `signed` and Windows signature verification fails, treat that as a release-integrity problem and report it with the artifact name and SHA-256 value.

## Bandwidth reports

Bandwidth ceilings are aggregate directional settings in binary KiB/s; `0 = unlimited`. A transfer attempt snapshots its effective budget when it starts. If reproducing a bandwidth issue, report upload/download limit values, configured parallelism, protocol, number of simultaneous transfers and whether the affected transfer was already running when the setting changed.

Do not infer that an idle worker slot must lend its allowance to another transfer: the maintained policy intentionally favors a stable aggregate ceiling over opportunistic bursting.

## Remote Edit reports

For Remote Edit issues, include:

- file size class and text encoding if known;
- line-ending style (LF/CRLF/CR) if relevant;
- whether the file changed on the server between open and save;
- whether the save succeeded but list metadata did not refresh;
- whether permissions changed unexpectedly.

Never attach the real sensitive remote file if it contains secrets. Reproduce with synthetic text where possible.

## Linux package reports

State whether the artifact is Debian, Ubuntu, Fedora or Portable and include its architecture suffix. Native install/remove/GUI lifecycle is continuously verified only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; arm64/aarch64 and i386/i686 artifacts retain build, metadata, extraction and binary-parity coverage.

For SFTP password/passphrase behavior, also state whether Ghost FTP came from a root-controlled package-manager installation or a user-writable Portable/per-user path. The latter intentionally cannot claim the same trusted AskPass provenance.

## Security and privacy issues

For credential exposure, trust-verification bypass, path containment, installer/uninstaller ownership or other security-sensitive reports, avoid publishing exploit-sensitive secrets or live credentials in a public issue. Provide the minimum synthetic reproduction needed to identify the problem.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.3` with `prerelease=false`. The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.3`. After a newer release is verified, old Ghost FTP releases/tags and obsolete package versions are intentionally removed by the latest-only retention policy. Git commit history remains available for engineering provenance but is not a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
