# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later stable releases. Verification covers source identity, Windows signing, Linux package metadata, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Expected release identity

```text
VERSION=1.0.0
TAG=ghostftp-v1.0.0
TITLE=Ghost FTP 1.0.0
PRERELEASE=false
```

A stable release must not be marked as a prerelease.

## Source revision

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. The release workflow also proves that `main` still points to the release commit immediately before and after publication.

## Public file set

The expected contract is **9 platform artifacts** and **12 public files** total. Extra or missing files fail the publication read-back.

The platform artifacts are five Windows files and four Linux files. The remaining three files are:

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

On Windows, use `Get-FileHash -Algorithm SHA256` and compare each value to the manifest.

Do not treat a matching filename as proof of authenticity; verify the digest and official release location.

## Windows Authenticode

Stable Windows publication is blocked unless the protected workflow reports a trusted signing identity as configured and every Setup/Portable artifact passes Authenticode verification during the production build.

On Windows, inspect the signature with PowerShell:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.0.0-Setup-x64.exe | Format-List
```

A stable file should have a valid signature and the expected publisher identity. If signature validation fails, do not bypass it by disabling operating-system security checks.

## x86/x32 alias verification

The two Setup names:

```text
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Setup-x32.exe
```

must be byte-identical. The release workflow compares their SHA-256 values before publication.

## Linux DEB verification

For each Linux package, verify:

```bash
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Architecture
```

Expected package name is `ghost-ftp`; version must equal `1.0.0`; architecture must match the file suffix.

The same checks apply to arm64 and i386.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.0.0`;
- title is `Ghost FTP 1.0.0`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 12-file allow-list;
- `SHA256.txt` verifies the downloaded content.

The production workflow performs immediate and delayed Release read-back; manual verification is still useful before broad deployment.

## GitHub Packages verification

Stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

The package is a verified distribution bundle, not a runtime container. Its OCI metadata must identify the Ghost FTP source repository, stable version and release source revision.

Recommended automation resolves the full version tag to an immutable OCI digest and pins that digest downstream. After extracting `/ghostftp-release/`, verify its `SHA256.txt` exactly as for a GitHub Release download.

## Privacy verification

A release/package must not contain:

- saved site profiles;
- plaintext passwords;
- private-key passphrases;
- signing private keys/PFX material;
- user files or local application data;
- developer machine paths or secrets.

The GHCR build copies only the already assembled `release/` allow-list, and Docker build networking is disabled.

## CI/release gate verification

Before trusting a new stable version, inspect that the exact revision passed:

- `go test -race ./...`;
- `go vet ./...`;
- formatting checks;
- repository/platform/dependency audits;
- security and privacy audits;
- localization and documentation audits;
- release contract audit;
- Python regression suite;
- Windows production package build;
- Linux production package build;
- Authenticode stable gate;
- GitHub Package push/read-back;
- GitHub Release asset/read-back verification.

## Failure interpretation

A failed gate is meaningful. Do not manually relabel a failed candidate as stable or bypass integrity checks to make a release appear complete. Fix the source, documentation, packaging or protected release configuration and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
