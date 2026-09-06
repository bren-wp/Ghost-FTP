# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later maintained releases. Verification covers source identity, GitHub Release state, per-file SHA-256 values, truthful Windows signing state, Linux package metadata and GitHub Packages registry read-back.

## Expected release identity

```text
VERSION=1.0.0
TAG=ghostftp-v1.0.0
TITLE=Ghost FTP 1.0.0
DRAFT=false
PRERELEASE=false
```

The maintained publication workflow does not create new prereleases. Historical 0.x prereleases remain history only.

## Source revision

`BUILD-METADATA.txt` contains the source commit. The GitHub Release tag must resolve to that exact commit. The release workflow proves that `main` still points to the release commit before/after Release publication and before GHCR publication.

## Public file set

The expected contract is **9 platform artifacts** and **12 public files** total. Extra or missing files fail publication read-back.

The platform artifacts are five Windows files and four Linux files. The remaining verification files are:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## SHA-256 verification

Download all Release files into one directory and verify `SHA256.txt`.

Linux:

```bash
sha256sum -c SHA256.txt
```

Windows PowerShell:

```powershell
Get-FileHash .\Ghost-FTP-1.0.0-Setup-x64.exe -Algorithm SHA256
```

Compare the result with the exact filename entry in `SHA256.txt`.

A matching filename alone is not proof of integrity. Verify both the digest and the official Release/tag provenance.

## Windows Authenticode state

`BUILD-METADATA.txt` is authoritative for the intended release signing state.

### Signed state

If it contains:

```text
WINDOWS_AUTHENTICODE=signed
WINDOWS_TRUST_MODE=authenticode+sha256+github-provenance
```

verify both SHA-256 and Authenticode:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.0.0-Setup-x64.exe | Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

The release workflow already requires every configured production signature to validate before publication, but end-user verification is still recommended.

### Explicit unsigned state

If it contains:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

an Authenticode signer is intentionally not claimed. Verify the official GitHub Release/tag, source commit and `SHA256.txt`. Do not interpret an unsigned binary as a confirmed BRENDIGO LTD publisher signature.

Ghost FTP never substitutes a self-signed/development certificate and labels it as a production publisher identity.

## x86/x32 alias verification

The Setup files:

```text
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Setup-x32.exe
```

must be byte-identical. Their SHA-256 values must match exactly.

## Linux DEB verification

For each Linux package:

```bash
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.0.0-Linux-amd64.deb Architecture
```

Expected values:

```text
Package: ghost-ftp
Version: 1.0.0
Architecture: amd64
```

Repeat for arm64/i386 and ensure the metadata matches each filename. The multiarch ZIP must contain exactly the three verified DEB packages.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.0.0`;
- title is `Ghost FTP 1.0.0`;
- `draft` is false;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 12-file allow-list;
- remote `SHA256.txt` matches the downloaded artifacts.

The production workflow performs immediate and delayed remote read-back and compares the published SHA manifest byte-for-byte with the local generated manifest.

## GitHub Packages verification

Stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

The package is a verified OCI **distribution bundle**, not a runtime container. It contains `/ghostftp-release/` with the corresponding Release file set.

For reproducible automation, resolve the semantic tag to its immutable OCI digest and pin that digest. Validate OCI labels for source/version/revision and then verify the embedded `SHA256.txt`.

The production workflow performs a stronger read-back itself: after pushing aliases it removes the local image, pulls the semantic-version tag from GHCR, checks source/version/revision labels and compares `SHA256.txt` plus `BUILD-METADATA.txt` byte-for-byte with the verified Release assembly.

## Privacy verification

A Release/package must not contain:

- saved site profiles;
- plaintext passwords;
- private-key passphrases;
- signing private keys/PFX material;
- user files or local application data;
- developer worktrees, caches or temporary package credentials.

The GHCR build copies only `release/`, uses Docker build networking disabled and uses a temporary Docker credential directory that is removed after publication.

## CI/release gate verification

Before trusting a new stable version, verify that the exact revision passed:

- `go test -race ./...`;
- `go vet ./...`;
- formatting checks;
- repository/platform/dependency audits;
- security and privacy audits;
- localization and documentation audits;
- stable release-contract audit;
- Python regression suite;
- Windows x64/x86 Setup + Portable build;
- Linux amd64/arm64/i386 build;
- configured Authenticode verification when a production certificate is present;
- GitHub Release upload/read-back;
- GitHub Package push/pull/read-back.

## Failure interpretation

A failed gate is meaningful. Do not bypass a mismatch, invent a signature or manually relabel a failed candidate as verified. Fix source, documentation, package/release configuration or protected signing configuration and rerun the exact revision.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
