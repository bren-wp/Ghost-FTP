# Ghost FTP support

Ghost FTP **0.0.8** is the current supported public release.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm the artifact reports version `0.0.6` or identify the exact maintained development source SHA for a non-public macOS/dev report.
2. Confirm the file came from the canonical `ghostftp-v0.0.8` release when reporting a public Windows/Linux/Android/browser package issue.
3. Verify public artifacts against `SHA256.txt`.
4. For official Windows Setup/Portable, verify trusted Authenticode.
5. For the public Android APK, verify the production signing certificate fingerprint as described in release verification.
6. Reproduce with the same protocol, architecture and OS/device class.
7. Remove passwords, passphrases, private keys, signing material, server secrets and customer data from diagnostics.

Latest-only retention means support targets the current public release rather than superseded binary/tag URLs.

## Windows reports

The public Windows files are:

```text
Ghost-FTP-0.0.8-Setup.exe
Ghost-FTP-0.0.8-Portable.exe
```

There are no supported public architecture-specific EXE aliases. Both files contain native **x64, x86 and ARM64** payloads and record:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

For Windows ARM64 problems, identify real ARM64 device/runtime evidence separately from CI's cross-build/structural evidence.

## Linux reports

State Debian, Ubuntu, Fedora or Portable plus architecture. Native install/remove/GUI lifecycle is continuously verified on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; other canonical architectures have build/metadata/extraction/binary-parity coverage.

For SFTP password/passphrase issues, state whether Ghost FTP is package-installed in a root-controlled path or running from a user-writable Portable path; automatic AskPass secret delivery fails closed without trusted provenance.

## Android public APK reports

Public Android 0.0.6 artifact:

```text
Ghost-FTP-0.0.8-Android.apk
```

Report Android API level, physical device/emulator, FTP/FTPS mode, SAF provider/folder behavior and whether the issue occurs during navigation, listing, file mutation, transfer/cancellation or Activity lifecycle changes.

Android SFTP remains intentionally hidden until strict maintained host-key identity verification exists. A public signed APK does not imply SFTP support.

For development-only issues, identify `Ghost-FTP-Android-dev.apk` and the exact source/run instead of confusing it with the production-signed APK.

## Browser helper reports

The public helper ZIPs target Chrome, Edge and Firefox. The helper parses explicitly supplied FTP/FTPS/SFTP targets locally and can copy a credential-stripped target. There is no supported browser-to-desktop handoff. Report browser family/version and whether parsing, safe-target generation or explicit copy failed; never include live credentials.

## macOS development reports

macOS remains a development/source frontend. Report the exact development artifact/source SHA, macOS version and CPU architecture. Development validation is not proof of Developer ID signing/notarization or public distribution.

## Transfer, queue and bandwidth reports

Bandwidth ceilings are aggregate directional values in binary KiB/s; `0 = unlimited`. Report upload/download limits, parallelism, protocol, simultaneous transfer count and whether a transfer was already running when settings changed. Queue reports should identify Top/Up/Down/Bottom action and lifecycle state.

For cancellation after disconnect/reconnect, include ordering. Session-generation ownership intentionally blocks stale completion from updating replacement state.

## File navigation and Remote Edit reports

For sorting/filter/search/comparison/bookmarks/file-mutation problems, identify local or remote side and use synthetic filenames when possible. Recursive search and comparison perform bounded/fresh listing work rather than operating only on the current visible filter.

For Remote Edit, include file-size class, encoding/line endings if relevant and whether the server file changed between open/save. Never attach sensitive remote contents when synthetic text can reproduce the issue.

## Supported release infrastructure

Current identity:

```text
VERSION=0.0.8
TAG=ghostftp-v0.0.8
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.8
```

After a successor is successfully verified, superseded release/tag/package identities are intentionally removed by latest-only retention. Git history remains engineering provenance rather than a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
