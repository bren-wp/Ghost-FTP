# Ghost FTP support

Ghost FTP **0.0.4** is the current supported public release line.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm the application reports version `0.0.4` for public Windows/Linux builds, or the maintained `0.0.4-dev` Android identity for the development APK.
2. Confirm a public desktop package came from the current GitHub Release, or identify the exact Android CI artifact when reporting the development APK.
3. Verify official desktop artifacts against `SHA256.txt`.
4. Reproduce with the same protocol, architecture and operating system/device class.
5. Remove real passwords, passphrases, private keys, server secrets and customer data from any diagnostic material.

Because the project keeps only the latest public desktop release, support is provided against the current version rather than superseded release/tag URLs.

## Useful report details

Include privacy-safe information such as:

- Windows or Linux distribution/version, or Android API/device/emulator version for development-APK reports;
- CPU architecture;
- Ghost FTP version;
- exact package/artifact name when the issue is installation or startup related;
- protocol selection;
- configured upload/download KiB/s limits when transfer throughput is involved;
- configured parallelism when aggregate bandwidth or queue behavior is involved;
- whether the issue occurs on connect, list, upload, download, sorting, current-folder filter, recursive search, directory comparison, synchronized navigation, bookmarks/profile starts, queue priority, Remote Edit, profile handling, appearance switching, Setup/Portable bootstrap selection or uninstall;
- exact user-visible error category without copying secret-bearing raw server output;
- whether the behavior reproduces with a synthetic test server/file.

Do not post real credentials, private-key contents, saved profile secrets, signing keys or production customer data.

## Windows universal-artifact reports

The current public Windows files are `Ghost-FTP-0.0.4-Setup.exe` and `Ghost-FTP-0.0.4-Portable.exe`. There are no supported public `-x64.exe`, `-x86.exe` or `-x32.exe` downloads in the 0.0.4 release contract.

For startup/bootstrap issues, report the public artifact name, Windows version, CPU/native architecture and SHA-256. Do not extract or redistribute internal staging payloads as substitutes for public release files.

## Windows signing reports

Always **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**.

When metadata says:

```text
WINDOWS_AUTHENTICODE=unsigned
```

the official file is explicitly `unsigned`; do not report the mere absence of a publisher signature as corruption.

When metadata says `signed` and Windows signature verification fails, treat that as a release-integrity problem and report it with the artifact name and SHA-256 value.

## Bandwidth and queue reports

Bandwidth ceilings are aggregate directional settings in binary KiB/s; `0 = unlimited`. A transfer attempt snapshots its effective budget when it starts. If reproducing a bandwidth issue, report upload/download limit values, configured parallelism, protocol, number of simultaneous transfers and whether the affected transfer was already running when the setting changed.

Do not infer that an idle worker slot must lend its allowance to another transfer: the maintained policy intentionally favors a stable aggregate ceiling over opportunistic bursting.

For queue-ordering issues, identify the selected queued job, requested Top/Up/Down/Bottom action and surrounding job states. Running/terminal history positions are not supposed to be reordered by the queued-only priority controls.

## File sorting/filter/search reports

Windows and Linux use shared item sorting semantics for Name, Type, Size and Modified, plus remote Permissions. Directories remain first. If ordering appears wrong, report the pane, field, direction, item names/types and whether a current-folder filter was active. Do not include sensitive filenames when a synthetic set can reproduce the issue.

Recursive search and directory comparison are distinct from current-folder filtering and perform their own bounded/fresh listing work. Report which mode was active so the issue is not misclassified as a simple sort/filter problem.

## Remote Edit reports

For Remote Edit issues, include:

- file size class and text encoding if known;
- line-ending style (LF/CRLF/CR) if relevant;
- whether the file changed on the server between open and save;
- whether the save succeeded but list metadata did not refresh;
- whether permissions changed unexpectedly.

Never attach the real sensitive remote file if it contains secrets. Reproduce with synthetic text where possible.

## Linux package/UI reports

State whether the artifact is Debian, Ubuntu, Fedora or Portable and include its architecture suffix. Native install/remove/GUI lifecycle is continuously verified only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; arm64/aarch64 and i386/i686 artifacts retain build, metadata, extraction and binary-parity coverage.

For SFTP password/passphrase behavior, also state whether Ghost FTP came from a root-controlled package-manager installation or a user-writable Portable/per-user path. The latter intentionally cannot claim the same trusted AskPass provenance.

For saved-credential reports, distinguish profile persistence from runtime AskPass delivery. The Linux GUI requires explicit bounded confirmation before newly entered password/passphrase material is stored, while runtime SFTP delivery still requires trusted executable/helper/parent provenance.

For appearance issues, report whether Light or Dark was persisted, whether the problem occurs at first paint or after saving Settings, and the display environment (native X11 or XWayland).

## Android development APK reports

Android is an active development surface, not one of the public Windows/Linux release assets. Report Android API level, physical device/emulator, exact APK source SHA/run when known, selected FTP/FTPS mode, SAF provider/folder behavior and whether the issue occurs during navigation, connection, listing, transfer/cancellation or Activity lifecycle changes.

Android SFTP is intentionally hidden until strict native host-key identity verification exists. Do not report its absence as a disabled button regression. Never attach real server credentials or SAF-backed sensitive files.

## Security and privacy issues

For credential exposure, trust-verification bypass, path containment, installer/uninstaller ownership or other security-sensitive reports, avoid publishing exploit-sensitive secrets or live credentials in a public issue. Provide the minimum synthetic reproduction needed to identify the problem.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.4` with `prerelease=false`. The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.4`. After a newer release is verified, old Ghost FTP releases/tags and obsolete package versions are intentionally removed by the latest-only retention policy. Git commit history remains available for engineering provenance but is not a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
