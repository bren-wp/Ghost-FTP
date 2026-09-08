# Ghost FTP release verification

This document defines verification for Ghost FTP 1.x Stable releases. The current maintained release is **1.1.7 Stable**. Verification covers source identity, Windows signing state, Linux package metadata/parity, SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Published 1.1.7 release identity

```text
VERSION=1.1.7
TAG=ghostftp-v1.1.7
TITLE=Ghost FTP 1.1.7
PRERELEASE=false
```

Ghost FTP 1.1.7 is Stable and must not be marked as a prerelease. Historical published tags are immutable identities and must not be moved, reused or rewritten.

## Source revision and canonical publication flow

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. Canonical sequence:

1. release-prep PR passes exact-head Core, Windows, Linux, distro and authentic Windows UI evidence;
2. the PR is merged;
3. post-merge gates pass on the exact current `main` SHA;
4. `release/ghostftp-vX.Y.Z` is created from that exact `main` SHA;
5. `.github/workflows/release-branch-trigger.yml` verifies branch SHA equals current `main` and branch version equals `VERSION`;
6. the trigger dispatches `.github/workflows/release.yml` through `workflow_dispatch` with the expected-version guard;
7. publication is accepted only after tag, GitHub Release assets and GHCR package read-back succeed.

A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Published 1.1.7 public file set

The 1.1.7 contract is **12 platform artifacts** and **15 public files** total.

Windows:

```text
Ghost-FTP-1.1.7-Setup-x64.exe
Ghost-FTP-1.1.7-Setup-x86.exe
Ghost-FTP-1.1.7-Setup-x32.exe
Ghost-FTP-1.1.7-Portable-x64.exe
Ghost-FTP-1.1.7-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.7-Linux-amd64.deb
Ghost-FTP-1.1.7-Linux-arm64.deb
Ghost-FTP-1.1.7-Linux-i386.deb
Ghost-FTP-1.1.7-Linux-multiarch.zip
Ghost-FTP-1.1.7-Linux-amd64.tar.gz
Ghost-FTP-1.1.7-Linux-arm64.tar.gz
Ghost-FTP-1.1.7-Linux-i386.tar.gz
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

The immutable 1.1.6 release remains historical with its original 9 platform artifacts / 12 public files; 1.1.7 does not mutate that release.

## Linux DEB/portable parity verification

For each `amd64`, `arm64` and `i386` pair, production must fail closed unless both `.deb` and `.tar.gz` exist, package metadata is valid, portable structure is complete and the matching `ghostftp` executables are byte-identical. Final metadata must record `LINUX_PORTABLE=amd64,arm64,i386`, `PUBLIC_PLATFORM_ARTIFACTS=12` and `PUBLIC_RELEASE_FILES=15`.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI artifacts are separate verification outputs and are not canonical 1.1.7 release assets.

## SHA-256 verification

`SHA256.txt` contains hashes for every other public release file. Verify both digest and official release location; matching filenames alone are not proof of integrity.

## Windows Authenticode

If `BUILD-METADATA.txt` says `WINDOWS_AUTHENTICODE=signed`, protected production signing material was configured and every Setup/Portable signature was required to verify. If it says `WINDOWS_AUTHENTICODE=unsigned`, the release intentionally contains unsigned Windows artifacts. This is a **truthful supported publication state**.

The production workflow **does not create a self-signed production identity**. The release gate requires **explicit unsigned metadata when no production certificate is configured**.

## x86/x32 alias verification

`Ghost-FTP-1.1.7-Setup-x86.exe` and `Ghost-FTP-1.1.7-Setup-x32.exe` must be byte-identical.

## Linux package verification

For every 1.1.7 DEB verify package name `ghost-ftp`, version `1.1.7`, correct architecture, `Homepage: https://ghostftp.com` and `Maintainer: Ghost FTP <https://ghostftp.com>`.

```bash
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Architecture
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Homepage
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Maintainer
```

Fedora supplemental metadata must likewise use the Ghost FTP product identity and `https://ghostftp.com`.

## Public branding verification

Active runtime, package, support and release metadata must use only **Ghost FTP** and `https://ghostftp.com`. Author/publisher identity is intentionally confined to the application's About surface. Legal and historical records retain their original text but are not active product branding.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.1.7`;
- title is `Ghost FTP 1.1.7`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 15-file allow-list;
- all three generic Linux tar.gz files are present;
- `SHA256.txt` verifies downloaded content;
- `BUILD-METADATA.txt` reports Windows signing state truthfully;
- release notes correspond to the `CHANGELOG.md` 1.1.7 section.

## GitHub Packages verification

Stable 1.1.7 is published at:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

The package is a verified distribution bundle, not a runtime container. Compatible aliases `1.1`, `1` and `latest` are updated only after semantic-version publication and registry read-back.

## 1.1.7 runtime and security verification

The release must preserve:

- application-owned DecisionCard primary paths with stock Windows fallback only if custom creation fails;
- shared native modal palette, DPI, keyboard and owner-modality behavior;
- adaptive DecisionCard geometry for longer localized/security text;
- runtime-localized native SSH private-key/folder picker labels;
- 24-language OK/Cancel/Yes/No and profile privacy/security copy;
- unchanged credential retain/remove/automatic-clear semantics;
- FTPS certificate/hostname validation and secure-to-plain no-downgrade;
- SFTP host-key verification/pinning and protected-secret ownership/lifetime rules;
- rooted local/transfer staging, destination revalidation, rollback and cancellation/retry generation safeguards;
- no telemetry, analytics, advertising/tracking SDK, hidden product network service or new external Go module dependency.

## Authentic Windows UI evidence

The exact final 1.1.7 release-prep revision requires authentic screenshots from the real production Windows x64 Portable executable for Main Workspace, Site Manager, Settings and About. Review them for clipping, overlap, stale branding, theme consistency and correct 1.1.7 presentation. Mockups are not accepted.

## CI/release gate verification

The 1.1.7 revision must pass exact PR-head CI, post-merge CI on exact `main`, `go test -race ./...`, `go vet ./...`, formatting, repository/platform/dependency/security/privacy/localization/documentation/release audits, the full Python regression suite, Windows x64/x86 Setup + Portable builds, Linux amd64/arm64/i386 DEB + tar.gz construction/parity, distro package CI, Debian 13/Ubuntu 26.04/Fedora 44 lifecycle smoke, authentic Windows UI evidence, signing-state verification, canonical release-branch checks, GitHub Package read-back and GitHub Release asset/tag/prerelease read-back.

## Failure interpretation

A failed gate is meaningful. Do not bypass integrity checks or relabel a failed build as Stable. Fix source, documentation, packaging, configured signing identity or publication infrastructure and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
