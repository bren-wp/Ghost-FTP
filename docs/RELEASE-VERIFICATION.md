# Ghost FTP release verification

The current maintained release is **0.0.1 Beta**.

## Canonical identity

```text
VERSION=0.0.1
TAG=ghostftp-v0.0.1
CHANNEL=Beta
PRERELEASE=true
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
```

The release source must be the exact current `main` commit that passed the complete release gate.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.1-Setup-x64.exe
Ghost-FTP-0.0.1-Setup-x86.exe
Ghost-FTP-0.0.1-Setup-x32.exe
Ghost-FTP-0.0.1-Portable-x64.exe
Ghost-FTP-0.0.1-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-0.0.1-Linux-amd64.deb
Ghost-FTP-0.0.1-Linux-arm64.deb
Ghost-FTP-0.0.1-Linux-i386.deb
Ghost-FTP-0.0.1-Linux-multiarch.zip
Ghost-FTP-0.0.1-Linux-amd64.tar.gz
Ghost-FTP-0.0.1-Linux-arm64.tar.gz
Ghost-FTP-0.0.1-Linux-i386.tar.gz
```

Metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Source verification

Before publication:

1. the release branch must match `release/ghostftp-v0.0.1`;
2. root `VERSION` must equal `0.0.1`;
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
VERSION=0.0.1
RELEASE_TAG=ghostftp-v0.0.1
RELEASE_CHANNEL=beta
ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
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

The publish workflow requires the remote GitHub Release asset set to match the exact 15-file allow-list immediately and after a delay. For 0.0.1 it requires `prerelease=true`.

A local build alone is not release evidence.

## Latest-only retention verification

Only after the 0.0.1 release read-back succeeds does `.github/workflows/release-retention.yml` remove superseded public versions.

Retention is complete only when:

- exactly one Ghost FTP GitHub Release remains and it is `ghostftp-v0.0.1`;
- exactly one `ghostftp-v*` tag remains and it is `ghostftp-v0.0.1`;
- completed versioned release branches are removed;
- obsolete Ghost FTP package versions are removed;
- retention emits `LATEST_ONLY_RELEASE_RETENTION=YES`.

The `main` commit history is never rewritten by retention cleanup.

## Beta package behavior

0.0.1 is Beta and therefore does not require a Stable GHCR distribution bundle. Any package versions from superseded numbering lines are removed by retention cleanup.

## Verification commands

Example local hash verification:

```bash
sha256sum -c SHA256.txt
```

For Windows, inspect `BUILD-METADATA.txt` before deciding whether Authenticode verification is expected.

See [GitHub Releases](GITHUB-RELEASES.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
