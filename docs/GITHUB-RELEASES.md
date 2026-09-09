# Ghost FTP GitHub Releases

Ghost FTP **0.0.1 Beta** is the current published Beta release contract. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

```text
Tag: ghostftp-v0.0.1
Title: Ghost FTP 0.0.1 Beta
Prerelease: true
```

Root `VERSION` is authoritative. The workflow rejects a manually supplied version that differs from source.

## Beta publication rule

Versions with major number `0` are Beta and publish with `prerelease=true`. A future major version `1` or greater uses the Stable channel and `prerelease=false`.

A normal push to `main`, including a `VERSION` change, does not publish a release directly.

## Canonical release trigger

`release.yml` is `workflow_dispatch`-only. The canonical release branch namespace is:

```text
release/ghostftp-vX.Y.Z
```

For the current candidate:

```text
release/ghostftp-v0.0.1
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when its semantic version matches root `VERSION` and its SHA equals exact current `main`. It then dispatches the canonical release workflow on `main` with the version guard.

## 0.0.1 public files

Ghost FTP 0.0.1 exposes **12 platform artifacts**.

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

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **15 public files** total.

## Exact-head rule

Before publication the release workflow compares current `main` with the release workflow source SHA. It performs the comparison again after publication before the delayed remote asset read-back. If `main` moves, publication fails rather than claiming stale source.

## Immutable-current publication transaction

The requested `ghostftp-v0.0.1` tag/release must not already exist. The publish workflow never clobbers a release asset or rewrites an existing current release tag.

After the new release is successfully published and remotely verified, `.github/workflows/release-retention.yml` enforces the project policy that **only the latest public Ghost FTP version remains**. It removes:

- older `ghostftp-v*` GitHub Releases;
- older/orphan `ghostftp-v*` tags;
- completed `release/ghostftp-v*` branches;
- obsolete Ghost FTP container package versions.

The cleanup runs only after the canonical `Publish Ghost FTP` workflow succeeds. Repository commit history on `main` is not rewritten.

## Linux portable parity gate

For each Linux architecture the production job requires both a `.deb` and `.tar.gz`, verifies DEB metadata and portable archive structure, and compares the installed/portable `ghostftp` executable byte-for-byte.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages built by `linux/BUILD-DISTROS.sh` remain **not yet part of the canonical release allow-list**.

## Windows signing state

Authenticode signing is optional. If protected production signing secrets are configured, all Windows artifacts must verify successfully. If no production certificate is configured, Windows artifacts remain explicitly unsigned and `BUILD-METADATA.txt` records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The workflow never creates a self-signed production identity and never labels an unsigned artifact as signed.

## Artifact allow-list

The publish job assembles a fresh release directory and records:

```text
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
```

`Ghost-FTP-0.0.1-Setup-x32.exe` is intentionally a byte-identical alias of the verified x86 Setup file.

## Read-back verification

The release transaction compares the remote sorted asset set with the expected allow-list immediately and again after a delay. For 0.0.1 it also requires `prerelease=true`.

Only after this verification succeeds may the retention workflow delete superseded public version identities.

## GitHub Packages

Pre-1.0 Beta releases are not published as the Stable GHCR distribution bundle. If an older Stable package version exists from the previous numbering line, the 0.0.1 retention cleanup removes it so the package registry does not continue advertising an obsolete public version.

Stable package publication, when a future Stable release exists, uses the verified release directory as a distribution bundle and **not a runtime container**.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
