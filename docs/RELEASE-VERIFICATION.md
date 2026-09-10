# Ghost FTP release verification

The current maintained release is **0.0.2**.

## Published 0.0.2 release identity

```text
VERSION=0.0.2
TAG=ghostftp-v0.0.2
TITLE=Ghost FTP 0.0.2
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The release source must be the exact current `main` commit that passed the complete release gate. The canonical release contains **14 platform artifacts / 17 public files**.

## Canonical public files

Windows:

```text
Ghost-FTP-0.0.2-Setup.exe
Ghost-FTP-0.0.2-Portable.exe
```

The two public Windows executables are offline universal x86/x64 packages. Each wrapper is an x86-compatible bootstrap so it can start on supported 32-bit and 64-bit Windows, contains both verified native x86 and x64 payloads, selects the native payload with `GetNativeSystemInfo`, verifies the staged executable and runs it without a network download.

Linux:

```text
Ghost-FTP-0.0.2-Linux-Debian-amd64.deb
Ghost-FTP-0.0.2-Linux-Debian-arm64.deb
Ghost-FTP-0.0.2-Linux-Debian-i386.deb
Ghost-FTP-0.0.2-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.2-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.2-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.2-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.2-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.2-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.2-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-Portable-i386.tar.gz
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

Architecture-specific Windows x64/x86 executables are internal native build payloads and are not valid public release files. The canonical public Windows files are only `Setup.exe` and `Portable.exe`.

## Build metadata

`BUILD-METADATA.txt` binds the release to its source revision and records at least:

```text
BRAND=Ghost FTP
VERSION=0.0.2
RELEASE_TAG=ghostftp-v0.0.2
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
GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp:0.0.2
```

## Windows native-payload verification

The public universal Windows files are produced only after the legacy native x64/x86 production builder completes its existing offline audits, Go tests/vet, PE-resource processing, native Setup/Portable construction, optional Authenticode signing and per-architecture release verification.

The public bootstrap then embeds those exact verified native files. At runtime it obtains the native Windows processor architecture through `GetNativeSystemInfo` rather than environment variables, stages only the matching payload under the Windows Local AppData known folder, verifies its exact size and SHA-256 before execution, waits for the native process and removes the staged file afterward.

Unsupported processor architectures fail closed. The bootstrap does not fetch binaries from the network.

## Windows Authenticode

The release contract supports a **truthful supported publication state** with or without a configured production signing identity.

When a trusted production certificate is configured, the native payloads and final public universal wrappers must verify. The production workflow **does not create a self-signed production identity**.

When no production certificate is configured, publication uses explicit unsigned metadata:

```text
WINDOWS_AUTHENTICODE=unsigned
```

If metadata says `signed` and Windows signature verification fails, treat the artifact as invalid. If metadata says `unsigned`, do not present the artifact as Authenticode-signed.

## Linux distro verification

`linux/BUILD-DISTROS.sh` builds one shared binary per architecture and packages it into Debian, Ubuntu, Fedora and portable artifacts. Release CI verifies package name/version/architecture, distro markers, dependency contracts and extracts each native package to compare its `ghostftp` executable byte-for-byte with the matching portable binary.

The separate distro install matrix additionally validates install/remove and GUI smoke behavior on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Remote release read-back

The publish workflow requires the remote GitHub Release asset set to match the exact 17-file allow-list immediately and after a delay. For 0.0.2 it requires `prerelease=false`.

A local build alone is not release evidence.

## GitHub Packages read-back

The same verified release directory is published as a distribution-only bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.2
```

The canonical workflow verifies the exact version tag after push. The package is not a supported runtime container and must not be treated as an application service.

## Latest-only retention verification

Only after the 0.0.2 release read-back succeeds does `.github/workflows/release-retention.yml` remove superseded public versions.

Before destructive cleanup, retention verifies the current release is not a draft, has `prerelease=false`, exposes exactly 17 assets and its tag points to current `main`.

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
