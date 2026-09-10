# Ghost FTP release verification

The current maintained release is **0.0.2**.

## Published 0.0.2 release identity

```text
VERSION=0.0.2
TAG=ghostftp-v0.0.2
TITLE=Ghost FTP 0.0.2
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
```

The release source must be the exact current `main` commit that passed the complete release gate. The canonical release contains **12 platform artifacts / 15 public files**.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.2-Setup-x64.exe
Ghost-FTP-0.0.2-Setup-x86.exe
Ghost-FTP-0.0.2-Setup-x32.exe
Ghost-FTP-0.0.2-Portable-x64.exe
Ghost-FTP-0.0.2-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-0.0.2-Linux-amd64.deb
Ghost-FTP-0.0.2-Linux-arm64.deb
Ghost-FTP-0.0.2-Linux-i386.deb
Ghost-FTP-0.0.2-Linux-multiarch.zip
Ghost-FTP-0.0.2-Linux-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-i386.tar.gz
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Canonical release dispatch

The canonical release branch namespace is:

```text
release/ghostftp-vX.Y.Z
```

For this release it is `release/ghostftp-v0.0.2`. The branch must point to the exact fully verified current `main` commit. `.github/workflows/release-branch-trigger.yml` validates that identity and uses `workflow_dispatch` to run the canonical `release.yml` workflow on that same source/version guard.

A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Source verification

Before publication:

1. the release branch must match `release/ghostftp-v0.0.2`;
2. root `VERSION` must equal `0.0.2`;
3. the release branch SHA must equal exact current `main`;
4. exact-head Core/Windows/Linux/distro gates must be successful;
5. authentic Windows UI evidence must come from the exact final source where the workflow is triggered.

The release workflow checks current `main` again immediately before publication and again before delayed remote read-back.

## SHA-256 verification

`SHA256.txt` contains a checksum for every public file except itself. A downloaded artifact is trusted for integrity only when its local hash matches the corresponding entry.

The x32 Setup compatibility alias must be byte-identical to the x86 Setup file.

## Build metadata

`BUILD-METADATA.txt` binds the release to its source revision and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.2
RELEASE_TAG=ghostftp-v0.0.2
RELEASE_CHANNEL=current
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.2
```

## Windows Authenticode

The release contract supports a **truthful supported publication state** with or without a configured production signing identity.

When a trusted production certificate is configured, signatures must verify. The production workflow **does not create a self-signed production identity**.

When no production certificate is configured, publication uses **explicit unsigned metadata when no production certificate is configured**:

```text
WINDOWS_AUTHENTICODE=unsigned
```

If metadata says `signed` and Windows signature verification fails, treat the artifact as invalid. If metadata says `unsigned`, do not present the artifact as Authenticode-signed.

## Remote release read-back

The publish workflow requires the remote GitHub Release asset set to match the exact 15-file allow-list immediately and after a delay. For 0.0.2 it requires `prerelease=false`.

A local build alone is not release evidence.

## GitHub Packages read-back

The same verified release directory is published as a distribution-only bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.2
```

The canonical workflow verifies the exact version tag after push. The package is not a supported runtime container and must not be treated as an application service.

## Latest-only retention verification

Only after the 0.0.2 release read-back succeeds does `.github/workflows/release-retention.yml` remove superseded public versions.

Before destructive cleanup, retention verifies the current release is not a draft, has `prerelease=false`, exposes exactly 15 assets and its tag points to current `main`.

Retention is complete only when:

- exactly one Ghost FTP GitHub Release remains and it is `ghostftp-v0.0.2`;
- exactly one `ghostftp-v*` tag remains and it is `ghostftp-v0.0.2`;
- the current canonical release branch is retained and superseded versioned release branches are removed;
- the current `0.0.2` package version remains and obsolete Ghost FTP package versions are removed;
- retention emits `LATEST_ONLY_RELEASE_RETENTION=YES`.

The `main` commit history is never rewritten by retention cleanup.

## Verification commands

Example local hash verification:

```bash
sha256sum -c SHA256.txt
```

For Windows, inspect `BUILD-METADATA.txt` before deciding whether Authenticode verification is expected.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
