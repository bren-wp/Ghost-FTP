# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later stable releases. The current maintained release is **1.1.6 Stable**. Verification covers source identity, Windows signing state, Linux package metadata, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Expected 1.1.6 release identity

```text
VERSION=1.1.6
TAG=ghostftp-v1.1.6
TITLE=Ghost FTP 1.1.6
PRERELEASE=false
```

A stable release must not be marked as a prerelease. Previously published Stable tags through `ghostftp-v1.1.5` remain historical identities and must not be moved or reused.

## Source revision and canonical publication flow

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. The release workflow proves that `main` still points to the release commit immediately before and after publication.

Publication is not triggered merely because `VERSION` changed on `main`. The canonical sequence is:

1. the release-prep PR passes exact-head Core, Windows and Linux CI plus any required authentic UI evidence;
2. the PR is merged;
3. post-merge CI passes on the exact current `main` SHA;
4. `release/ghostftp-vX.Y.Z` is created from that exact `main` SHA;
5. `.github/workflows/release-branch-trigger.yml` verifies that the branch commit equals current `main` and that the branch version equals `VERSION`;
6. only then does the branch trigger dispatch `.github/workflows/release.yml` on `main` with the expected version guard;
7. publication is accepted only after tag, GitHub Release assets and GHCR package read-back all succeed.

`release.yml` is publication-only and `workflow_dispatch`-only. A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Public file set

The expected contract is **9 platform artifacts** and **12 public files** total. Extra or missing files fail publication read-back.

Windows:

```text
Ghost-FTP-1.1.6-Setup-x64.exe
Ghost-FTP-1.1.6-Setup-x86.exe
Ghost-FTP-1.1.6-Setup-x32.exe
Ghost-FTP-1.1.6-Portable-x64.exe
Ghost-FTP-1.1.6-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.6-Linux-amd64.deb
Ghost-FTP-1.1.6-Linux-arm64.deb
Ghost-FTP-1.1.6-Linux-i386.deb
Ghost-FTP-1.1.6-Linux-multiarch.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## SHA-256 verification

`SHA256.txt` contains hashes for every other public release file. On Linux use `sha256sum -c SHA256.txt`. On Windows use `Get-FileHash -Algorithm SHA256` and compare each value with the manifest. A matching filename alone is not proof of authenticity; verify both the digest and official release location.

## Windows Authenticode

Read `WINDOWS_AUTHENTICODE` from `BUILD-METADATA.txt` before interpreting Windows signature state.

If metadata says `WINDOWS_AUTHENTICODE=signed`, the protected workflow used a configured production certificate and every Setup/Portable artifact was required to pass Authenticode verification during the production build.

If metadata says `WINDOWS_AUTHENTICODE=unsigned`, the release intentionally contains unsigned Windows artifacts. This is a **truthful supported publication state**, not a failed or partially signed release.

The production workflow **does not create a self-signed production identity**. Short-lived self-signed certificates are permitted only for CI signing smoke tests. The verification gate requires **explicit unsigned metadata when no production certificate is configured**.

## x86/x32 alias verification

`Ghost-FTP-1.1.6-Setup-x86.exe` and `Ghost-FTP-1.1.6-Setup-x32.exe` must be byte-identical. The release workflow compares their SHA-256 values before publication.

## Linux DEB verification

For each Linux package, verify package name, version, architecture, Homepage and Maintainer. Expected package name is `ghost-ftp`; version must equal `1.1.6`; architecture must match the file suffix; Homepage must be `https://ghostftp.com`; publisher metadata must identify **BRENDIGO LTD**.

Example:

```bash
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Architecture
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Homepage
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Maintainer
```

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.1.6`;
- title is `Ghost FTP 1.1.6`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 12-file allow-list;
- `SHA256.txt` verifies downloaded content;
- `BUILD-METADATA.txt` truthfully reports `WINDOWS_AUTHENTICODE=signed` or `unsigned`;
- release notes correspond to the `CHANGELOG.md` 1.1.6 section.

The production workflow performs immediate and delayed Release read-back. Manual verification remains useful before broad deployment.

## GitHub Packages verification

Stable 1.1.6 publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.6
```

The package is a verified distribution bundle, not a runtime container. OCI metadata must identify the Ghost FTP source repository, stable version and release source revision. Stable aliases `1.1`, `1` and `latest` are updated only after publication and registry read-back succeed.

## 1.1.6 runtime and security verification

The release candidate must preserve the maintained runtime contract while applying the 1.1.6 hardening changes:

- recursive local delete cannot traverse a swapped pathname because descent is rooted through opened filesystem handles;
- local directory creation remains anchored to the originally opened base directory across pathname replacement;
- SFTP host-key fingerprint presentation is derived directly from the exact selected scanned key blob;
- the embedded public-key algorithm must match the declared algorithm;
- remote staging cleanup does not accept server/tool text alone as proof of absence;
- non-zero SFTP cleanup remains uncertain/fail-closed and therefore blocks automatic retry;
- FTPS certificate/hostname validation and secure-to-plain no-downgrade behavior remain active;
- SFTP host-key verification/pinning and protected-secret ownership/lifetime rules remain enforced;
- local downloads preserve `LocalRoot` and root-bound staging/activation/rollback safeguards;
- Site Manager Duplicate does not silently clone saved password/passphrase material or host-key trust state;
- cancellation/retry/connection-generation safeguards remain enabled;
- no telemetry, analytics, advertising/tracking SDK, hidden product network service or new external Go module dependency is introduced.

## Authentic Windows UI evidence

The 1.1.6 release-prep changes the public version presentation. The exact final release-prep source must produce authentic screenshots from the real Windows x64 Portable executable for:

- Main Workspace;
- Site Manager;
- Settings;
- About.

The images must be visually reviewed for clipping, overlap, stale branding and correct 1.1.6 version presentation. Mockups or generated approximations are not accepted as release evidence.

## CI/release gate verification

Before trusting 1.1.6, inspect that the exact revision passed:

- exact PR-head CI before merge;
- post-merge CI on the exact `main` SHA;
- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- repository/platform/dependency audits;
- security and privacy audits;
- localization and documentation audits;
- release contract audit;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production package build and artifact verification;
- Linux amd64/arm64/i386 production package build and DEB verification;
- authentic Windows UI evidence from the exact revision;
- Authenticode verification when a production certificate is configured;
- explicit unsigned metadata when no production certificate is configured;
- canonical release branch equality/version checks before publication dispatch;
- GitHub Package push/read-back;
- GitHub Release asset/tag/prerelease read-back verification.

## Privacy verification

A release/package must not contain saved profiles, plaintext passwords, private-key passphrases, signing private keys/PFX material, user files, local application data or developer-machine secrets. The GHCR build copies only the already assembled release allow-list and build networking is disabled.

## Failure interpretation

A failed gate is meaningful. Do not manually relabel a failed candidate as stable or bypass integrity checks to make a release appear complete. Fix the source, documentation, packaging, configured signing identity or publication infrastructure and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
