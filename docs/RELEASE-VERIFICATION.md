# Ghost FTP release verification

The current maintained release is **0.0.3**.

## Published 0.0.3 release identity

```text
VERSION=0.0.3
TAG=ghostftp-v0.0.3
TITLE=Ghost FTP 0.0.3
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The release source must be the exact current `main` commit that passed the complete release gate. The canonical release contains **14 platform artifacts / 17 public files**.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.3-Linux-Debian-amd64.deb
Ghost-FTP-0.0.3-Linux-Debian-arm64.deb
Ghost-FTP-0.0.3-Linux-Debian-i386.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.3-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.3-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.3-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.3-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.3-Linux-Portable-i386.tar.gz
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Source verification

Before publication:

1. the release branch must match `release/ghostftp-v0.0.3`;
2. root `VERSION` must equal `0.0.3`;
3. the release branch SHA must equal exact current `main`;
4. exact-head Core/Windows/Linux/distro gates must be successful;
5. authentic Windows UI evidence must come from the exact final source revision.

A push to `main`, including a change to `VERSION`, does not publish a release directly.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is trusted for integrity only when its local hash matches the corresponding entry. There is no public x32/x86/x64 Windows alias set in 0.0.3; only the universal Setup and Portable executables are public.

## Build metadata

`BUILD-METADATA.txt` binds the release to its source revision and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.3
RELEASE_TAG=ghostftp-v0.0.3
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX
WINDOWS_SETUP=universal-x86-x64
WINDOWS_PORTABLE=universal-x86-x64
WINDOWS_NATIVE_PAYLOADS=x64,x86
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.3
```

## Windows Authenticode

The release contract supports a **truthful supported publication state** with or without a configured production signing identity. When a trusted production certificate is configured, signatures must verify. The production workflow **does not create a self-signed production identity**.

When no production certificate is configured, publication uses **explicit unsigned metadata when no production certificate is configured**:

```text
WINDOWS_AUTHENTICODE=unsigned
```

If metadata says `signed` and Windows signature verification fails, treat the artifact as invalid.

## Remote release read-back

The publish workflow requires the remote GitHub Release asset set to match the exact 17-file allow-list immediately and after a delay. For 0.0.3 it requires `prerelease=false`. A local build alone is not release evidence.

## GitHub Packages read-back

The same verified release directory is published as a distribution-only bundle at `ghcr.io/bren-wp/ghost-ftp:0.0.3`. The canonical workflow verifies the exact version tag after push. The package is not a supported runtime container.

## Latest-only retention verification

Only after the 0.0.3 release read-back succeeds does `.github/workflows/release-retention.yml` remove superseded public versions. Before destructive cleanup, retention verifies the current release is not a draft, has `prerelease=false`, exposes exactly 17 assets and its tag points to current `main`.

Retention is complete only when the current release/tag/canonical branch and exact-version package remain, superseded identities are removed, and the workflow emits `LATEST_ONLY_RELEASE_RETENTION=YES`. The `main` commit history is never rewritten by retention cleanup.

## Verification commands

Example local hash verification:

```bash
sha256sum -c SHA256.txt
```

For Windows, inspect `BUILD-METADATA.txt` before deciding whether Authenticode verification is expected.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
