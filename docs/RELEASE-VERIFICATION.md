# Ghost FTP release verification

This document defines how to verify Ghost FTP **1.0.0 Stable** and later stable releases. The current candidate is **1.1.1 Stable**. Verification covers source identity, Windows signing state, Linux package metadata, per-file SHA-256 values, GitHub Release state and GitHub Packages registry state.

## Expected 1.1.1 release identity

```text
VERSION=1.1.1
TAG=ghostftp-v1.1.1
TITLE=Ghost FTP 1.1.1
PRERELEASE=false
```

A stable release must not be marked as a prerelease. Previously published tags, including `ghostftp-v1.0.0` and `ghostftp-v1.1.0`, remain historical identities and must not be moved or reused.

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

Read `WINDOWS_AUTHENTICODE` from `BUILD-METADATA.txt` before interpreting the Windows signature state.

If it is:

```text
WINDOWS_AUTHENTICODE=signed
```

then the protected workflow used a configured production certificate and every Setup/Portable artifact was required to pass Authenticode verification during the production build.

If it is:

```text
WINDOWS_AUTHENTICODE=unsigned
```

then the release intentionally contains unsigned Windows artifacts. This is a truthful supported publication state, not a failed or partially signed release.

On Windows, inspect the signature with PowerShell:

```powershell
Get-AuthenticodeSignature .\Ghost-FTP-1.1.1-Setup-x64.exe | Format-List
```

For a signed release, verify that the signature is valid and that the publisher identity is the expected trusted identity. For an unsigned release, Windows may show SmartScreen/publisher warnings. Do not bypass enterprise or operating-system security policy solely to suppress such warnings.

The release workflow does not create a self-signed production identity. Short-lived self-signed certificates are allowed only in CI signing smoke tests and are not public publisher credentials.

## x86/x32 alias verification

The two Setup names:

```text
Ghost-FTP-1.1.1-Setup-x86.exe
Ghost-FTP-1.1.1-Setup-x32.exe
```

must be byte-identical. The release workflow compares their SHA-256 values before publication.

## Linux DEB verification

For each Linux package, verify:

```bash
dpkg-deb -f Ghost-FTP-1.1.1-Linux-amd64.deb Package
dpkg-deb -f Ghost-FTP-1.1.1-Linux-amd64.deb Version
dpkg-deb -f Ghost-FTP-1.1.1-Linux-amd64.deb Architecture
```

Expected package name is `ghost-ftp`; version must equal `1.1.1`; architecture must match the file suffix. The same checks apply to arm64 and i386.

## GitHub Release verification

Confirm that:

- tag is `ghostftp-v1.1.1`;
- title is `Ghost FTP 1.1.1`;
- `prerelease` is false;
- tag resolves to the documented source commit;
- remote asset names exactly match the 12-file allow-list;
- `SHA256.txt` verifies the downloaded content;
- `BUILD-METADATA.txt` truthfully reports `WINDOWS_AUTHENTICODE=signed` or `unsigned`.

The production workflow performs immediate and delayed Release read-back; manual verification is still useful before broad deployment.

## GitHub Packages verification

Stable 1.1.1 publishes:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.1
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

## Connection and UI verification for 1.1.1

The release candidate must preserve the maintained runtime contract in addition to packaging integrity:

- fresh/fallback appearance is Classic Light while an explicitly saved Dark choice remains respected;
- fresh quick-connect protocol is explicit FTPS on port 21 on Windows and Linux;
- plain FTP remains available only as an explicit compatibility choice and FTPS must not silently downgrade to it;
- loopback protocol regressions exercise authentication, list, mkdir, upload, size/list, rename, download/content equality and delete;
- `remote.Manager.Connect()` regression coverage exercises connect, remote list/operation state and disconnect, including invalid-password and secure-to-plain failure paths;
- credential persistence remains opt-in and privacy-sensitive save prompts use the maintained localization catalog;
- documentation screenshots come from the real Windows x64 Portable executable built by the dedicated screenshot workflow.

## CI/release gate verification

Before trusting a new stable version, inspect that the exact revision passed:

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
- Authenticode verification when a production certificate is configured;
- explicit unsigned metadata when no production certificate is configured;
- GitHub Package push/read-back;
- GitHub Release asset/tag/prerelease read-back verification.

## Failure interpretation

A failed gate is meaningful. Do not manually relabel a failed candidate as stable or bypass integrity checks to make a release appear complete. Fix the source, documentation, packaging, configured signing identity or publication infrastructure and rerun the exact workflow.

See [GitHub Releases](GITHUB-RELEASES.md), [Packages](PACKAGES.md), [Signing](SIGNING.md), [Security](SECURITY.md) and [Privacy](PRIVACY.md).
