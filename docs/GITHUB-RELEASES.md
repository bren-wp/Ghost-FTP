# Ghost FTP GitHub Releases

Ghost FTP **0.0.5** is the current published release contract. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

```text
Tag: ghostftp-v0.0.5
Title: Ghost FTP 0.0.5
Prerelease: false
```

Root `VERSION` is authoritative. Major version zero is not automatically mapped to GitHub prerelease state for this project.

## Canonical release trigger

A normal push to `main`, including a `VERSION` change, does not publish directly. Canonical release branches use:

```text
release/ghostftp-vX.Y.Z
```

For the current candidate:

```text
release/ghostftp-v0.0.5
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when its semantic version matches root `VERSION` and its SHA equals exact current `main`.

The trigger lifecycle is deterministic:

1. snapshot existing `release.yml` workflow-dispatch run IDs;
2. dispatch canonical `release.yml` on `main` with the exact version guard;
3. identify the newly created `Publish Ghost FTP` run whose head SHA equals validated `main`;
4. wait for that exact run and require terminal success;
5. only then dispatch canonical `release-retention.yml`;
6. identify the new exact-main retention run;
7. wait for and require terminal retention success.

A successful workflow dispatch request by itself is **not** successful publication.

## 0.0.5 public files

Ghost FTP 0.0.5 exposes **14 platform artifacts**.

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

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **17 public files** total.

The Android development APK and optional browser companion source are independently maintained but are not part of this public Windows/Linux release allow-list.

## Exact-head rule

Before publication the release workflow is bound to current `main`. The publish step checks `main` again immediately before release creation and before delayed remote asset verification. A moved `main` fails publication rather than claiming stale source.

The canonical release branch itself must also equal exact current `main`; an older successful workflow cannot satisfy a new release transaction.

## Immutable-current publication transaction

The requested `ghostftp-v0.0.5` tag/release must not already exist. The workflow refuses to clobber an existing release identity or rewrite published assets.

After the successor is successfully published and remotely verified, `.github/workflows/release-retention.yml` enforces the policy that **only the latest public Ghost FTP version remains**. It removes superseded Ghost FTP Releases, tags, canonical release branches and obsolete GHCR package versions while retaining current identities. `main` history is not rewritten.

## Windows universal artifact gate

The Windows release job builds only two public universal executables. Verified native x64/x86 Setup/Portable binaries remain internal staging inputs. No public `-x64.exe`, `-x86.exe` or `-x32.exe` release aliases are allowed.

Production Authenticode is optional. If protected trusted signing secrets are configured, signatures must verify. If no production certificate is configured, metadata records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The workflow never fabricates a self-signed production publisher identity.

## Linux distro and portable parity gate

`linux/BUILD-DISTROS.sh` is the canonical Linux release builder. Architecture mapping is `amd64 ↔ x86_64`, `arm64 ↔ aarch64` and `i386 ↔ i686`. Package metadata and extracted executable bytes are verified before publication. Native install/remove/runtime/GUI coverage is separately maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Artifact allow-list

`BUILD-METADATA.txt` records:

```text
WINDOWS_SETUP=universal-x86-x64
WINDOWS_PORTABLE=universal-x86-x64
WINDOWS_NATIVE_PAYLOADS=x64,x86
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

## Read-back and package verification

The remote sorted GitHub Release asset set must match the exact 17-file allow-list immediately and after a delay. For 0.0.5 the workflow requires `prerelease=false`.

The same verified release directory is published as a distribution bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

It is a distribution bundle, not a runtime container. The exact-version package is verified after push and preserved by retention.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
