# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later stable releases. The current maintained release is **1.1.6 Stable**. Verification covers source identity, Windows signing state, Linux package metadata, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Published 1.1.6 release identity

```text
VERSION=1.1.6
TAG=ghostftp-v1.1.6
TITLE=Ghost FTP 1.1.6
PRERELEASE=false
```

Ghost FTP 1.1.6 is published Stable and must not be marked as a prerelease. Published Stable tags through `ghostftp-v1.1.6` are historical identities and must not be moved, reused or rewritten.

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

Ghost FTP 1.1.6 completed this canonical flow. Future releases must independently complete the same gates.

## Published 1.1.6 public file set

The immutable 1.1.6 contract is **9 platform artifacts** and **12 public files** total.

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

Post-1.1.6 source/CI builds also create package-manager-neutral Linux `.tar.gz` archives for amd64, arm64 and i386. Those source outputs are not part of the immutable 1.1.6 asset set.

## Maintained next-release public file contract

The current source release workflow is prepared for a later version with **12 platform artifacts** and **15 public files** total. The additional Linux assets are:

```text
Ghost-FTP-X.Y.Z-Linux-amd64.tar.gz
Ghost-FTP-X.Y.Z-Linux-arm64.tar.gz
Ghost-FTP-X.Y.Z-Linux-i386.tar.gz
```

The future-version release job must fail closed unless, for every Linux architecture:

- both the `.deb` and `.tar.gz` exist and are non-empty;
- DEB package/version/architecture metadata is valid;
- the portable archive contains `ghostftp`, `ghost-ftp.desktop`, `ghost-ftp.png`, `LICENSE` and `README.md`;
- the DEB-installed `/usr/bin/ghostftp` and portable `ghostftp` are byte-identical;
- all three tarballs are staged into the final `release/` directory;
- `BUILD-METADATA.txt` records `LINUX_PORTABLE=amd64,arm64,i386`, `PUBLIC_PLATFORM_ARTIFACTS=12` and `PUBLIC_RELEASE_FILES=15`;
- `SHA256.txt` covers every public file except itself;
- the immediate and delayed GitHub Release read-back asset sets exactly match the 15-file allow-list.

These are source workflow requirements for the next release version, not evidence that another version is already published. They do not modify 1.1.6.

## SHA-256 verification

`SHA256.txt` contains hashes for every other public release file. On Linux use `sha256sum -c SHA256.txt`. On Windows use `Get-FileHash -Algorithm SHA256` and compare each value with the manifest. A matching filename alone is not proof of authenticity; verify both the digest and official release location.

## Windows Authenticode

Read `WINDOWS_AUTHENTICODE` from `BUILD-METADATA.txt` before interpreting Windows signature state.

If metadata says `WINDOWS_AUTHENTICODE=signed`, the protected workflow used a configured production certificate and every Setup/Portable artifact was required to pass Authenticode verification during the production build.

If metadata says `WINDOWS_AUTHENTICODE=unsigned`, the release intentionally contains unsigned Windows artifacts. This is a **truthful supported publication state**, not a failed or partially signed release.

The production workflow **does not create a self-signed production identity**. Short-lived self-signed certificates are permitted only for CI signing smoke tests. The verification gate requires **explicit unsigned metadata when no production certificate is configured**.

## x86/x32 alias verification

`Ghost-FTP-1.1.6-Setup-x86.exe` and `Ghost-FTP-1.1.6-Setup-x32.exe` must be byte-identical. The release workflow compares their SHA-256 values before publication. Future releases preserve the same alias rule.

## Linux DEB verification

For each published 1.1.6 Linux package, verify package name, version, architecture, Homepage and Maintainer. Expected package name is `ghost-ftp`; version must equal `1.1.6`; architecture must match the file suffix; Homepage must be `https://ghostftp.com`; publisher metadata must identify **BRENDIGO LTD**.

Example:

```bash
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Architecture
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Homepage
dpkg-deb -f Ghost-FTP-1.1.6-Linux-amd64.deb Maintainer
```

For a future release that includes `.tar.gz`, also extract both formats and compare their `ghostftp` executables byte-for-byte. The portable archive is not a substitute for DEB metadata verification; both production formats have separate structural checks and a shared-binary parity check.

## GitHub Release verification

For published 1.1.6 confirm that:

- tag is `ghostftp-v1.1.6`;
- title is `Ghost FTP 1.1.6`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the historical 12-file allow-list;
- `SHA256.txt` verifies downloaded content;
- `BUILD-METADATA.txt` truthfully reports `WINDOWS_AUTHENTICODE=signed` or `unsigned`;
- release notes correspond to the `CHANGELOG.md` 1.1.6 section.

For a later version using the maintained source contract, remote assets must instead exactly match that version's 15-file allow-list, including all three Linux `.tar.gz` files. Do not infer a future release's existence from source code alone.

The production workflow performs immediate and delayed Release read-back. Manual verification remains useful before broad deployment.

## GitHub Packages verification

Stable 1.1.6 is published at:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.6
```

The package is a verified distribution bundle, not a runtime container. OCI metadata must identify the Ghost FTP source repository, stable version and release source revision. Stable aliases `1.1`, `1` and `latest` were updated after publication and registry read-back succeeded.

For a future release, GHCR mirrors the exact verified release assembly. Therefore Linux tarballs become part of that future bundle only if they passed the same release allow-list, SHA-256 generation and GitHub Release read-back gates.

## 1.1.6 runtime and security verification

The published release preserves the maintained runtime contract while applying the 1.1.6 hardening changes:

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

Post-1.1.6 `main` contains additional hardening such as rooted tree-download directory preparation and additional Linux packaging/release-contract work. Those later changes are verified separately and are not retroactively attributed to 1.1.6.

## Authentic Windows UI evidence

The 1.1.6 release-prep changed the public version presentation. The exact final release-prep source produced authentic screenshots from the real Windows x64 Portable executable for:

- Main Workspace;
- Site Manager;
- Settings;
- About.

The images were visually reviewed for clipping, overlap, stale branding and correct 1.1.6 version presentation. Mockups or generated approximations were not accepted as release evidence.

A later release that changes public Windows UI or version presentation must independently produce authentic exact-head Windows x64 Portable evidence when required by the maintained release policy.

## CI/release gate verification

The published 1.1.6 revision passed:

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

The maintained next-release contract additionally requires production Linux `.tar.gz` construction, structure verification and executable parity against each matching DEB before publication.

## Privacy verification

A release/package must not contain saved profiles, plaintext passwords, private-key passphrases, signing private keys/PFX material, user files, local application data or developer-machine secrets. The GHCR build copies only the already assembled release allow-list and build networking is disabled.

## Failure interpretation

A failed gate is meaningful. Do not manually relabel a failed build as stable or bypass integrity checks to make a release appear complete. Fix the source, documentation, packaging, configured signing identity or publication infrastructure and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
