# Ghost FTP support

Ghost FTP **0.0.5** is the current supported public release line.

Official product and support destination: **https://ghostftp.com**.

## Before reporting a problem

1. Confirm public Windows/Linux builds report version `0.0.5`, or identify the maintained development artifact/source SHA for Android or macOS reports.
2. Confirm a public desktop package came from the current GitHub Release; development artifacts from CI are not public-release substitutes.
3. Verify official desktop artifacts against `SHA256.txt`.
4. For official Windows Setup/Portable files, also verify Authenticode publisher status.
5. Reproduce with the same protocol, architecture and operating-system/device class.
6. Remove passwords, passphrases, private keys, server secrets and customer data from diagnostics.

Because the project keeps only the latest public desktop release, support is provided against the current version rather than superseded binary/tag URLs.

## Useful report details

Include privacy-safe information such as OS/distribution or Android API/device/emulator, CPU architecture, Ghost FTP version, package/artifact name, protocol, transfer bandwidth/parallelism when relevant, and the exact action that failed (connect/list/upload/download/sort/filter/search/comparison/bookmark/queue/Remote Edit/profile/settings/appearance/Setup/Portable/uninstall).

For 0.0.5 lifecycle issues, state whether the problem involved repeated commands, closing a dialog/application during an active mutation, Remote Edit save/reload, transfer Cancel followed by disconnect/reconnect, or Android Activity destruction/recreation during an in-flight connection.

Never post real credentials, private-key contents, saved profile secrets, signing keys or production customer data.

## Windows universal-artifact reports

The current public Windows files are `Ghost-FTP-0.0.5-Setup.exe` and `Ghost-FTP-0.0.5-Portable.exe`. There are no supported public `-x64.exe`, `-x86.exe`, `-x32.exe` or `-arm64.exe` downloads in the 0.0.5 release contract.

Both public files contain internal native x64, x86 and ARM64 payloads. `GetNativeSystemInfo` selects the matching embedded payload and the bootstrap verifies staged bytes before execution; Ghost FTP does not download an architecture-specific runtime component.

A successful current release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

For an ARM64 report, state whether the machine is native Windows on ARM64, the Windows version/build, device model if useful, and whether Setup or Portable was used. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` means project CI currently proves ARM64 cross-build/PE/package/signing mechanics but does not claim native ARM64 execution on the maintained Actions runner. A real ARM64 user report should therefore be clearly identified as device/runtime evidence rather than confused with x64 CI screenshot evidence.

## Windows signing reports

Official public Windows artifacts require trusted Authenticode under the current release contract. `BUILD-METADATA.txt` for a successfully published current release must record:

```text
WINDOWS_AUTHENTICODE=signed
```

If an official Setup or Portable file is unsigned, has no signer certificate or fails Windows signature verification, treat that as a release-integrity issue. Report the artifact name, release tag and SHA-256 value, but do not attach signing material or credentials.

Unsigned local/development or ordinary CI builds are allowed for engineering validation. They are not official public Windows release artifacts and should be identified explicitly as development outputs when reporting problems.

## Bandwidth and queue reports

Bandwidth ceilings are aggregate directional values in binary KiB/s; `0 = unlimited`. Report upload/download limits, parallelism, protocol, simultaneous transfer count and whether a transfer was already running when settings changed. Queue-order reports should identify the selected queued job and requested Top/Up/Down/Bottom action.

If a Cancel problem follows a disconnect/reconnect, include that ordering. Windows intentionally rejects stale cancellation completion from an older `connectionGeneration` rather than allowing it to alter the replacement session's status/queue surface.

## Sorting/filter/search/comparison reports

Report pane, field/direction, item types/names using synthetic names where possible and whether a current-folder filter was active. Recursive search and directory comparison perform bounded/fresh listing work and are distinct from simple loaded-snapshot filtering.

## Remote Edit reports

Include file size class, encoding/line-ending style if relevant, whether the file changed on the server between open/save, whether metadata refresh failed, and whether the problem involved session re-entry or application close. Never attach a sensitive remote file when synthetic text can reproduce the issue.

## Linux reports

State Debian, Ubuntu, Fedora or Portable plus architecture. Native install/remove/GUI lifecycle is continuously verified only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; other canonical architectures retain build/metadata/extraction/binary-parity coverage.

For SFTP password/passphrase issues, state whether Ghost FTP is package-installed in a root-controlled path or running from a user-writable Portable/per-user path; automatic AskPass secret delivery intentionally fails closed without trusted provenance.

## Android development APK reports

Report Android API level, physical device/emulator, exact source SHA/run when known, FTP/FTPS mode, SAF provider/folder behavior and whether the issue occurs during navigation, connection, listing, transfer/cancellation or Activity lifecycle changes. SFTP remains intentionally hidden until strict native host-key identity verification exists.

## macOS development app reports

Report the exact development artifact/source SHA, macOS version and CPU architecture. Development app validation is distinct from a public notarized macOS distribution claim; the current 17-file public release remains Windows/Linux only.

## Browser companion reports

The browser helper currently parses explicitly pasted `ftp://`, `ftps://` and `sftp://` targets locally. It does not launch the desktop client and there is no supported browser-to-desktop handoff contract yet. Report browser family/version, the scheme used and whether parsing, safe-target generation or explicit copy behavior failed. Do not include live credentials in URLs or reports.

## Supported release infrastructure

The current release identity is `ghostftp-v0.0.5` with `prerelease=false`. The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.5`. After a newer release is verified, old release/tag/package identities are intentionally removed by latest-only retention. Git history remains engineering provenance, not a supported binary archive.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Testing](TESTING.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
