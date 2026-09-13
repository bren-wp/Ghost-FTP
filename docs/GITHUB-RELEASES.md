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

A normal push to `main`, including a change to `VERSION`, does not publish a release directly. Canonical `release.yml` is **`workflow_dispatch`-only**. Canonical release branches use:

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

A successful workflow-dispatch request by itself is **not** successful publication.

## 0.0.5 public files

Ghost FTP 0.0.5 exposes **14 platform artifacts / 17 public files**.

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

The Android development APK, macOS development app and browser companion source are independently maintained source/development surfaces and are not part of this public Windows/Linux release allow-list.

## Exact-head rule

Before publication the release workflow is bound to current `main`. The publish step checks `main` again immediately before release creation and before delayed remote asset verification. A moved `main` fails publication rather than claiming stale source.

The canonical release branch itself must also equal exact current `main`; an older successful workflow cannot satisfy a new release transaction.

## Immutable-current publication transaction

The requested `ghostftp-v0.0.5` tag/release must not already exist. The workflow refuses to clobber an existing release identity or rewrite published assets.

After the successor is successfully published and remotely verified, `.github/workflows/release-retention.yml` enforces the policy that **only the latest public Ghost FTP version remains**. It removes superseded Ghost FTP Releases, tags, canonical release branches and obsolete GHCR package versions while retaining current identities. `main` history is not rewritten.

## Windows universal artifact and signing gate

The Windows release job builds only two public universal executables. Verified native **x64, x86 and ARM64** Setup/Portable binaries remain internal staging inputs. No public `-x64.exe`, `-x86.exe`, `-x32.exe` or `-arm64.exe` release aliases are allowed.

The public PE x86 bootstrap detects the native Windows processor through `GetNativeSystemInfo`, selects the matching embedded x64/x86/ARM64 payload, verifies the staged bytes and performs no runtime download. ARM64 therefore extends the internal payload set without increasing the two-file Windows public surface or the overall **14 platform artifacts / 17 public files** release shape.

Official Windows publication requires a protected trusted Authenticode identity. `Publish Ghost FTP` fails when the production PFX or password is unavailable, and it verifies both public executables with `Get-AuthenticodeSignature` before staging them for publication.

A successful public release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

`WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` prevents an unsupported claim: CI cross-builds and validates ARM64 PE/resources/package routing/signing mechanics, but current maintained Windows Actions runtime evidence is not native ARM64 execution.

There is no supported unsigned-publication fallback for the official release workflow. Local development and ordinary CI builds may be unsigned, but they are not official public release artifacts.

The production workflow never fabricates a self-signed publisher identity. The CI signing smoke test may use a short-lived development certificate only to prove signing mechanics.

## Linux distro and portable parity gate

`linux/BUILD-DISTROS.sh` is the canonical Linux release builder. Architecture mapping is `amd64 ↔ x86_64`, `arm64 ↔ aarch64` and `i386 ↔ i686`. Package metadata and extracted executable bytes are verified before publication. Native install/remove/runtime/GUI coverage is separately maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.

## Artifact allow-list

`BUILD-METADATA.txt` records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
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

## What counts as release evidence

A local build, successful PR workflow or unsigned development executable is not proof of an official release. Release evidence requires the canonical publish workflow, exact source identity, successful signing verification, remote GitHub Release read-back, GHCR read-back and the documented retention chain.

Likewise, a successful Windows x64 runtime/UI capture does not prove native ARM64 execution. Native ARM64 runtime evidence must come from a maintained ARM64 Windows runner/device and must be identified separately if that coverage is added later.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
