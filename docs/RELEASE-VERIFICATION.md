# Ghost FTP release verification

The current maintained release is **0.0.5**.

## Published 0.0.5 release identity

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
TITLE=Ghost FTP 0.0.5
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
LATEST_ONLY_RELEASE_RETENTION=YES
```

The release source must be the exact current `main` commit that passed the complete release gate. The canonical public release contains **14 platform artifacts / 17 public files**.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

The independently validated `Ghost-FTP-Android.apk` development artifact and browser companion source are not among these 17 public Windows/Linux release files.

## Canonical release dispatch

The canonical branch namespace is `release/ghostftp-vX.Y.Z`; the 0.0.5 release branch is `release/ghostftp-v0.0.5`. It must point to exact fully verified current `main` and its version must match root `VERSION`.

Canonical `.github/workflows/release.yml` is `workflow_dispatch`-only. The branch trigger validates source/version equality, dispatches canonical `release.yml`, waits for the exact newly created release run to finish successfully, then dispatches and verifies retention.

A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Source verification

Before publication:

1. root `VERSION` must equal `0.0.5`;
2. the release branch must equal `release/ghostftp-v0.0.5` and exact current `main`;
3. all exact-head release-prep workflows triggered for the final candidate must be successful;
4. all required post-merge push workflows on the exact merge SHA must be successful;
5. authentic Windows/Linux/Android evidence must bind to the exact source revision;
6. the release workflow's quality, Windows and Linux jobs must succeed again from fresh source.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is trusted for integrity only when its local hash matches the corresponding entry.

There is no public x32/x86/x64 Windows alias set in 0.0.5. Only universal Setup and Portable executables are public; architecture-specific native payloads are internal verified staging inputs.

## Build metadata

`BUILD-METADATA.txt` binds release identity to source and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.5
RELEASE_TAG=ghostftp-v0.0.5
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX
WINDOWS_SETUP=universal-x86-x64
WINDOWS_PORTABLE=universal-x86-x64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.5
```

Android remains an independently validated source platform and is not silently added to the release platform/file count.

## Windows Authenticode

The release contract supports a **truthful supported publication state** with or without a configured production signing identity. When a trusted production certificate is configured, signatures must verify. The production workflow **does not create a self-signed production identity**.

When no production certificate is configured, publication uses explicit unsigned metadata when no production certificate is configured:

```text
WINDOWS_AUTHENTICODE=unsigned
```

If metadata says `signed`, require valid Authenticode. If metadata says `unsigned`, do not represent the file as publisher-signed.

## Linux package verification

Debian/Ubuntu DEB and Fedora RPM metadata must match version, architecture, product URL and maintained dependencies. For each architecture the release workflow compares extracted package `ghostftp` bytes with the matching Portable archive. Native package-manager/runtime lifecycle is separately proven only for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Android development verification

The Android APK workflow verifies source/security/lifecycle contracts, lint, installable APK construction, APK identity verification and artifact upload. This proves the maintained development surface builds cleanly but does not add an APK to the 17-file public Windows/Linux release allow-list.

## Authentic runtime evidence

The exact-head UI workflow captures real Windows, Linux and Android runtime surfaces and assembles one read-only verified evidence bundle. The maintained manifest contains **15 runtime images**: five Windows, three Linux and seven Android. The assembler verifies source SHA, workflow run identity, expected filenames, byte counts and SHA-256 hashes before emitting `AUTHENTIC_UI_EVIDENCE=VERIFIED`.

## Remote release read-back

The publish workflow requires the remote GitHub Release asset set to match the exact 17-file allow-list immediately and after a delay. For 0.0.5 it requires `prerelease=false`. A local build alone is not release evidence.

## GitHub Packages read-back

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The exact version tag is verified after push. The package is not a supported runtime container.

## Latest-only retention verification

Only after the 0.0.5 release read-back succeeds does retention remove superseded public versions. Retention independently verifies current release/tag/main/package identity before destructive cleanup.

Completion requires the current release/tag to be `ghostftp-v0.0.5`, the current canonical release branch to remain, the exact 0.0.5 package version to remain, obsolete Ghost FTP release identities/package versions to be removed and `LATEST_ONLY_RELEASE_RETENTION=YES` to be emitted. `main` history is never rewritten.

## Verification commands

```bash
sha256sum -c SHA256.txt
```

For Windows, inspect `BUILD-METADATA.txt` before deciding whether Authenticode verification is expected.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
