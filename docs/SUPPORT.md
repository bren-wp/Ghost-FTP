# Ghost FTP support

Ghost FTP **0.0.5** is the current supported public release line.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm public Windows/Linux builds report version `0.0.5`, or the maintained `0.0.5-dev` Android identity for the development APK.
2. Confirm a public desktop package came from the current GitHub Release, or identify the exact Android CI artifact for development-APK reports.
3. Verify official desktop artifacts against `SHA256.txt`.
4. Reproduce with the same protocol, architecture and operating-system/device class.
5. Remove passwords, passphrases, private keys, server secrets and customer data from diagnostics.

Because the project keeps only the latest public desktop release, support is provided against the current version rather than superseded binary/tag URLs.

## Useful report details

Include privacy-safe information such as OS/distribution or Android API/device/emulator, CPU architecture, Ghost FTP version, package/artifact name, protocol, transfer bandwidth/parallelism when relevant, and the exact action that failed (connect/list/upload/download/sort/filter/search/comparison/bookmark/queue/Remote Edit/profile/settings/appearance/Setup/Portable/uninstall).

For 0.0.5 lifecycle issues, state whether the problem involved repeated commands, closing a dialog/application during an active mutation, Remote Edit save/reload, or Android Activity destruction/recreation during an in-flight connection.

Never post real credentials, private-key contents, saved profile secrets, signing keys or production customer data.

## Windows universal-artifact reports

The current public Windows files are `Ghost-FTP-0.0.5-Setup.exe` and `Ghost-FTP-0.0.5-Portable.exe`. There are no supported public `-x64.exe`, `-x86.exe` or `-x32.exe` downloads in the 0.0.5 release contract.

## Windows signing reports

Always **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**.

When metadata says `WINDOWS_AUTHENTICODE=unsigned`, the **official file is explicitly `unsigned`**; absence of a publisher signature alone is not corruption.

When **metadata says `signed`** and **Windows signature verification fails**, treat it as a release-integrity issue and report the artifact name plus SHA-256.

## Bandwidth and queue reports

Bandwidth ceilings are aggregate directional values in binary KiB/s; `0 = unlimited`. Report upload/download limits, parallelism, protocol, simultaneous transfer count and whether a transfer was already running when settings changed. Queue-order reports should identify the selected queued job and requested Top/Up/Down/Bottom action.

## Sorting/filter/search/comparison reports

Report pane, field/direction, item types/names using synthetic names where possible and whether a current-folder filter was active. Recursive search and directory comparison perform bounded/fresh listing work and are distinct from simple loaded-snapshot filtering.

## Remote Edit reports

Include file size class, encoding/line-ending style if relevant, whether the file changed on the server between open/save, whether metadata refresh failed, and whether the problem involved session re-entry or application close. Never attach a sensitive remote file when synthetic text can reproduce the issue.

## Linux reports

State Debian, Ubuntu, Fedora or Portable plus architecture. Native install/remove/GUI lifecycle is continuously verified only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; other canonical architectures retain build/metadata/extraction/binary-parity coverage.

For SFTP password/passphrase issues, state whether Ghost FTP is package-installed in a root-controlled path or running from a user-writable Portable/per-user path; automatic AskPass secret delivery intentionally fails closed without trusted provenance.

## Android development APK reports

Report Android API level, physical device/emulator, exact source SHA/run when known, FTP/FTPS mode, SAF provider/folder behavior and whether the issue occurs during navigation, connection, listing, transfer/cancellation or Activity lifecycle changes. SFTP remains intentionally hidden until strict native host-key identity verification exists.

## Browser companion reports

For optional extension source, identify browser family/version, the `ftp://`, `ftps://` or `sftp://` scheme used and whether the failure is link recognition or handoff behavior. Do not include live credentials in URLs or reports.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.5` with `prerelease=false`. The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.5`. After a newer release is verified, old release/tag/package identities are intentionally removed by latest-only retention. Git history remains engineering provenance, not a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
