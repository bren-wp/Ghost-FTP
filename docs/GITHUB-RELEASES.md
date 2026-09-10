# Ghost FTP GitHub Releases

Ghost FTP **0.0.3** is the current published release contract. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

```text
Tag: ghostftp-v0.0.3
Title: Ghost FTP 0.0.3
Prerelease: false
```

Root `VERSION` is authoritative. A normal push to `main`, including a `VERSION` change, does not publish a release directly.

## Canonical release trigger

`release.yml` is `workflow_dispatch`-only. The canonical release branch namespace is:

```text
release/ghostftp-vX.Y.Z
```

For the current candidate:

```text
release/ghostftp-v0.0.3
```

`.github/workflows/release-branch-trigger.yml` accepts that branch only when its semantic version matches root `VERSION` and its SHA equals exact current `main`. It dispatches the canonical publication workflow, identifies the new exact-main run, waits for terminal `success`, then dispatches and verifies latest-only retention. A successful dispatch request alone is not treated as a successful release.

## 0.0.3 public files

Ghost FTP 0.0.3 exposes **14 platform artifacts**.

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

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **17 public files** total.

## Exact-head and immutable identity rules

Before publication the workflow compares current `main` with the release source SHA and checks again before delayed remote read-back. The requested `ghostftp-v0.0.3` tag/release must not already exist. Publication never uses `--clobber`, never rewrites an existing release tag and never retroactively changes the asset set of a published release.

## Windows contract

The public download surface contains exactly two universal executables. Native x64/x86 Setup/Portable artifacts remain internal staging evidence and must not leak into the release directory. Authenticode is optional; configured trusted signatures must verify, otherwise metadata records `WINDOWS_AUTHENTICODE=unsigned`.

## Linux contract

`linux/BUILD-DISTROS.sh` is the canonical Linux release builder. It produces Debian/Ubuntu/Fedora/Portable families and reuses one binary per architecture across matching package variants. The release workflow verifies package metadata and executable parity.

Native lifecycle coverage remains Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. Other architectures retain build/parity coverage without an unsupported native-install claim.

## Artifact allow-list and read-back

The publish job records:

```text
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The remote GitHub Release asset set must match the exact 17-file allow-list immediately and again after a delay. For 0.0.3 it requires `prerelease=false`.

## Latest-only retention

After successful remote verification, `.github/workflows/release-retention.yml` enforces that **only the latest public Ghost FTP version remains**. It removes superseded Ghost FTP Releases, tags, canonical release branches and obsolete container package versions while leaving `main` commit history untouched. Cleanup independently verifies the current release is non-draft, `prerelease=false`, has exactly 17 assets and points to current `main` before destructive cleanup.

## GitHub Packages

The same verified release directory is published as the distribution-only bundle:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.3
```

It is not a runtime container. The exact semantic-version tag is the immutable verification identity for this release transaction.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
