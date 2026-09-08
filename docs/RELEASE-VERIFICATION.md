# Ghost FTP release verification

This document defines verification for Ghost FTP 1.x Stable releases. The current maintained release is **1.1.7 Stable**. Verification covers source identity, Windows signing state, Linux package metadata/parity, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Published 1.1.7 release identity

```text
VERSION=1.1.7
TAG=ghostftp-v1.1.7
TITLE=Ghost FTP 1.1.7
PRERELEASE=false
```

Ghost FTP 1.1.7 is Stable and must not be marked as a prerelease. Historical published tags are immutable identities and must not be moved, reused or rewritten.

## Source revision and canonical publication flow

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. The production workflow proves that `main` still points to the release commit before and after publication.

Canonical sequence:

1. release-prep PR passes exact-head Core, Windows, Linux, distro and authentic Windows UI evidence;
2. PR is merged;
3. post-merge gates pass on the exact current `main` SHA;
4. `release/ghostftp-vX.Y.Z` is created from that exact `main` SHA;
5. `.github/workflows/release-branch-trigger.yml` verifies branch SHA equals current `main` and branch version equals `VERSION`;
6. only then does the trigger dispatch `.github/workflows/release.yml` through `workflow_dispatch` with the expected-version guard;
7. publication is accepted only after tag, GitHub Release assets and GHCR package read-back succeed.

`release.yml` is publication-only and `workflow_dispatch`-only. **A push to `main`, including a change to `VERSION`, must never publish a release directly.**

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

For each `amd64`, `arm64` and `i386` pair, production must fail closed unless:

- both `.deb` and `.tar.gz` exist and are non-empty;
- DEB package/version/architecture metadata is valid;
- portable archive contains `ghostftp`, `ghost-ftp.desktop`, `ghost-ftp.png`, `LICENSE` and `README.md`;
- DEB-installed `/usr/bin/ghostftp` and portable `ghostftp` are byte-identical;
- all three tarballs are staged into final `release/`;
- metadata records `LINUX_PORTABLE=amd64,arm64,i386`, `PUBLIC_PLATFORM_ARTIFACTS=12` and `PUBLIC_RELEASE_FILES=15`;
- `SHA256.txt` covers every other public file;
- immediate and delayed Release read-back asset sets exactly match the 15-file allow-list.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI artifacts are separate verification outputs and are not canonical 1.1.7 release assets.

## SHA-256 verification

`SHA256.txt` contains hashes for every other public release file. On Linux use `sha256sum -c SHA256.txt`. On Windows use `Get-FileHash -Algorithm SHA256` and compare every value. A matching filename alone is not proof of integrity; verify both digest and official release location.

## Windows Authenticode

Read `WINDOWS_AUTHENTICODE` from `BUILD-METADATA.txt` before interpreting Windows signature state.

If metadata says `WINDOWS_AUTHENTICODE=signed`, protected production signing material was configured and every Setup/Portable signature was required to verify.

If metadata says `WINDOWS_AUTHENTICODE=unsigned`, the release intentionally contains unsigned Windows artifacts. This is a **truthful supported publication state**.

The production workflow **does not create a self-signed production identity**. Short-lived self-signed certificates are permitted only for CI signing smoke tests. The release gate requires **explicit unsigned metadata when no production certificate is configured**.

## x86/x32 alias verification

`Ghost-FTP-1.1.7-Setup-x86.exe` and `Ghost-FTP-1.1.7-Setup-x32.exe` must be byte-identical. Production compares their SHA-256 values before publication.

## Linux package verification

For every 1.1.7 DEB verify package name `ghost-ftp`, version `1.1.7`, correct architecture, Homepage `https://ghostftp.com` and BRENDIGO LTD publisher metadata.

Example:

```bash
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Architecture
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Homepage
dpkg-deb -f Ghost-FTP-1.1.7-Linux-amd64.deb Maintainer
```

For tar.gz, verify archive structure and byte parity with the matching DEB executable.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.1.7`;
- title is `Ghost FTP 1.1.7`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 15-file allow-list;
- all three generic Linux tar.gz files are present;
- `SHA256.txt` verifies downloaded content;
- `BUILD-METADATA.txt` reports `WINDOWS_AUTHENTICODE=signed` or `unsigned` truthfully;
- release notes correspond to the `CHANGELOG.md` 1.1.7 section.

The production workflow performs immediate and delayed Release read-back. Manual verification remains useful before broad deployment.

## GitHub Packages verification

Stable 1.1.7 is published at:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

The package is a verified distribution bundle, not a runtime container. OCI metadata must identify source repository, stable version and release revision. Compatible aliases `1.1`, `1` and `latest` are updated only after semantic-version publication and registry read-back.

GHCR mirrors the exact verified release assembly, so the three Linux tarballs are part of 1.1.7 only after they pass the canonical allow-list, SHA-256 and Release read-back gates.

## 1.1.7 runtime and security verification

The release must preserve the maintained runtime/security contract while applying the 1.1.7 improvements:

- Confirm/Info/Error use the application-owned DecisionCard primary path with stock Windows fallback only if custom creation fails;
- shared native modal palette, DPI, keyboard and owner-modality behavior remains intact;
- DecisionCard geometry expands for longer localized/security text while retaining compact short-dialog geometry;
- native SSH private-key/folder pickers resolve labels from the active locale;
- 24-language OK/Cancel/Yes/No and profile privacy/security copy remain non-empty and runtime-resolved;
- credential retain/remove/automatic-clear semantics are unchanged;
- FTPS certificate/hostname validation and secure-to-plain no-downgrade remain active;
- SFTP host-key verification/pinning and protected-secret ownership/lifetime rules remain enforced;
- rooted local/transfer staging, destination revalidation, rollback and cancellation/retry generation safeguards remain enabled;
- no telemetry, analytics, advertising/tracking SDK, hidden product network service or new external Go module dependency is introduced.

## Authentic Windows UI evidence

The 1.1.7 version/UI change requires authentic screenshots from the real production Windows x64 Portable executable for:

- Main Workspace;
- Site Manager;
- Settings;
- About.

The exact final release-prep screenshots must be visually reviewed for clipping, overlap, stale branding and correct 1.1.7 presentation. Mockups or generated approximations are not accepted.

## CI/release gate verification

The 1.1.7 revision must pass:

- exact PR-head CI before merge;
- post-merge CI on exact `main` SHA;
- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- repository/platform/dependency audits;
- security/privacy/localization/documentation/release audits;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production build/artifact verification;
- Linux amd64/arm64/i386 DEB + tar.gz construction, structure and parity verification;
- supplemental distro package build/parity CI;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 install/remove/GUI smoke;
- authentic exact-head Windows UI evidence;
- Authenticode verification when a production certificate is configured;
- explicit unsigned metadata when no production certificate is configured;
- canonical release branch equality/version checks;
- GitHub Package push/read-back;
- GitHub Release asset/tag/prerelease read-back.

## Privacy verification

A release/package must not contain saved profiles, plaintext passwords, private-key passphrases, signing private keys/PFX material, user files, local application data or developer-machine secrets. GHCR copies only the verified release allow-list and builds with networking disabled.

## Failure interpretation

A failed gate is meaningful. Do not bypass integrity checks or relabel a failed build as Stable. Fix source, documentation, packaging, configured signing identity or publication infrastructure and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
