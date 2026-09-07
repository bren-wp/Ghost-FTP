# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later stable releases. The current maintained release is **1.1.3 Stable**. Verification covers source identity, Windows signing state, Linux package metadata, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Expected 1.1.3 release identity

```text
VERSION=1.1.3
TAG=ghostftp-v1.1.3
TITLE=Ghost FTP 1.1.3
PRERELEASE=false
```

A stable release must not be marked as a prerelease. Previously published tags, including `ghostftp-v1.0.0`, `ghostftp-v1.1.0`, `ghostftp-v1.1.1` and `ghostftp-v1.1.2`, remain historical identities and must not be moved or reused.

## Source revision and canonical publication flow

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. The release workflow proves that `main` still points to the release commit immediately before and after publication.

Publication is not triggered merely because `VERSION` changed on `main`. The canonical sequence is:

1. feature/release-prep PR passes exact-head CI and any required authentic UI evidence;
2. the PR is merged;
3. post-merge CI passes on the exact current `main` SHA;
4. `release/ghostftp-vX.Y.Z` is created from that exact `main` SHA;
5. `.github/workflows/release-branch-trigger.yml` verifies that the branch commit equals current `main` and that the branch version equals `VERSION`;
6. only then does the branch trigger dispatch `.github/workflows/release.yml` on `main` with the expected version guard.

`release.yml` is publication-only and `workflow_dispatch`-only. A push to `main`, including a change to `VERSION`, must never publish a release directly.

## Public file set

The expected contract is **9 platform artifacts** and **12 public files** total. Extra or missing files fail publication read-back.

Windows:

```text
Ghost-FTP-1.1.3-Setup-x64.exe
Ghost-FTP-1.1.3-Setup-x86.exe
Ghost-FTP-1.1.3-Setup-x32.exe
Ghost-FTP-1.1.3-Portable-x64.exe
Ghost-FTP-1.1.3-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.3-Linux-amd64.deb
Ghost-FTP-1.1.3-Linux-arm64.deb
Ghost-FTP-1.1.3-Linux-i386.deb
Ghost-FTP-1.1.3-Linux-multiarch.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## SHA-256 verification

`SHA256.txt` contains hashes for every other public release file. On Linux:

```bash
sha256sum -c SHA256.txt
```

On Windows, use `Get-FileHash -Algorithm SHA256` and compare each value to the manifest. A matching filename alone is not proof of authenticity; verify the digest and official release location.

## Windows Authenticode

Read `WINDOWS_AUTHENTICODE` from `BUILD-METADATA.txt` before interpreting Windows signature state.

If metadata says:

```text
WINDOWS_AUTHENTICODE=signed
```

then the protected workflow used a configured production certificate and every Setup/Portable artifact was required to pass Authenticode verification during the production build.

If metadata says:

```text
WINDOWS_AUTHENTICODE=unsigned
```

then the release intentionally contains unsigned Windows artifacts. This is a truthful supported publication state, not a failed or partially signed release.

Example inspection:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.1.3-Setup-x64.exe | Format-List
```

The production workflow does not create a self-signed production identity. Short-lived self-signed certificates are permitted only for CI signing smoke tests. The verification gate requires explicit unsigned metadata when no production certificate is configured.

## x86/x32 alias verification

The two Setup names:

```text
Ghost-FTP-1.1.3-Setup-x86.exe
Ghost-FTP-1.1.3-Setup-x32.exe
```

must be byte-identical. The release workflow compares their SHA-256 values before publication.

## Linux DEB verification

For each Linux package, verify:

```bash
dpkg-deb -f Ghost-FTP-1.1.3-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.3-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.3-Linux-amd64.deb Architecture
```

Expected package name is `ghost-ftp`; version must equal `1.1.3`; architecture must match the file suffix. The same checks apply to arm64 and i386.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.1.3`;
- title is `Ghost FTP 1.1.3`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 12-file allow-list;
- `SHA256.txt` verifies downloaded content;
- `BUILD-METADATA.txt` truthfully reports `WINDOWS_AUTHENTICODE=signed` or `unsigned`.

The production workflow performs immediate and delayed Release read-back. Manual verification is still useful before broad deployment.

## GitHub Packages verification

Stable 1.1.3 publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.3
```

The package is a verified distribution bundle, not a runtime container. OCI metadata must identify the Ghost FTP source repository, stable version and release source revision. Stable aliases `1.1`, `1` and `latest` are updated only after publication and registry read-back succeed.

## 1.1.3 runtime/security verification

The release candidate must preserve the maintained runtime contract:

- fresh/fallback Windows appearance is Classic Light and explicitly saved Dark remains respected;
- fresh quick-connect protocol is explicit FTPS on port 21 on Windows and Linux;
- plain FTP remains explicit compatibility only and secure transports never silently downgrade;
- SFTP host-key verification/pinning and protected-secret ownership/lifetime rules remain enforced;
- SFTP batch path operands are escaped so literal remote names are not interpreted as globs or command options;
- remote tree directory preparation rejects symlink and non-directory path components;
- local downloads preserve the selected `LocalRoot` through the transfer layer and use root-bound `os.Root` staging/activation/rollback;
- staging identity/sentinel validation and late `SkipExisting` checks are active;
- Site Manager Duplicate does not silently clone saved password/passphrase material or host-key trust state;
- Windows reserved device-name validation includes DOS superscript-number variants;
- installer/uninstaller registry metadata does not advertise a false quiet uninstall command.

## CI/release gate verification

Before trusting 1.1.3, inspect that the exact revision passed:

- exact PR-head CI before merge;
- post-merge CI on the exact `main` SHA;
- `go test -race ./...`;
- `go vet ./...`;
- formatting checks;
- repository/platform/dependency audits;
- security and privacy audits;
- localization and documentation audits;
- release contract audit;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production package build and artifact verification;
- Linux amd64/arm64/i386 production package build and DEB verification;
- authentic Windows UI evidence from the exact revision whenever Windows UI changed;
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
