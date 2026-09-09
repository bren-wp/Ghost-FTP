# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source candidate: **1.1.8 Stable**. Published **1.1.7 Stable** and all earlier Stable/Beta tags remain immutable historical releases until and after the 1.1.8 publication flow completes.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use:

```text
ghostftp-vMAJOR.MINOR.PATCH
```

For the current release candidate:

```text
VERSION=1.1.8
TAG=ghostftp-v1.1.8
CHANNEL=Stable
PRERELEASE=false
```

Published tags, including `ghostftp-v1.1.7`, are immutable release history and must remain on their original release commits.

## Historical pre-1.0 policy

The maintained public line began at **0.1.0**. Every **0.x.y** release was a **Beta** prerelease. Historical tags/releases remain unchanged for traceability.

Version **1.0.0** is the first **Stable** public release.

## Post-1.0 increments

### PATCH

Use a patch increment for compatible bug fixes, security/privacy hardening, performance improvements, documentation corrections and packaging/release fixes that do not intentionally add an incompatible product contract.

Ghost FTP **1.1.8** is a patch release candidate. It strengthens privacy-safe child-process diagnostics, Linux transport/AskPass executable provenance, state-directory identity, Windows installer/uninstaller/shortcut ownership, exact-object cleanup and responsive mixed-DPI geometry while preserving the 1.1 protocol, profile and release-artifact contract.

### MINOR

Use a minor increment for backward-compatible functionality, substantial workflow improvements or new optional capabilities.

Ghost FTP **1.1.0** was a minor release because it added the Classic Light desktop appearance while preserving the 1.x connection/transfer contract.

### MAJOR

Use a major increment for intentionally incompatible product contracts that require clear migration guidance.

## Binary/package identity

The same semantic version is injected into Windows application binaries, Setup/Portable packages, Linux DEB metadata, release notes/build metadata, GitHub Release tag/title and the stable GitHub Package tag.

Source entry points retain `version = "dev"` and receive production versions only through build linker flags.

## Stable GitHub Release rule

For `MAJOR >= 1`, publication uses the Stable channel and `prerelease=false`. The official Stable release must point to the exact `main` commit that passed release quality gates.

The canonical release branch trigger is created only from exact current `main` using:

```text
release/ghostftp-v<version>
```

For this candidate the exact branch is:

```text
release/ghostftp-v1.1.8
```

The trigger rejects a branch whose version differs from `VERSION` or whose commit differs from current `main`.

A `VERSION` bump or ordinary push to `main` does not itself publish a release.

## Windows signing state

Windows Authenticode is an **optional production hardening layer**, not a prerequisite for Stable version identity. When a trusted production PFX is configured through protected Actions secrets, every produced Windows artifact must verify successfully or publication fails. When no production certificate is configured, Stable publication may continue only with explicit unsigned metadata:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Absence of a production Authenticode certificate by itself is not a versioning failure. The resulting Windows signing state must remain truthfully `unsigned` throughout build metadata and verification.

Ghost FTP never generates a self-signed production certificate and presents it as a trusted publisher identity.

## GitHub Packages versioning

Stable 1.1.8 publication uses:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.8
```

with compatible Stable aliases after successful registry publication/read-back:

```text
1.1
1
latest
```

Downstream automation should prefer the full semantic version or immutable OCI digest.

The GHCR object is a verified distribution bundle of the canonical release directory, not a supported runtime container.

## Version source integrity

The release workflow rejects a malformed version, a manual version different from `VERSION`, an existing conflicting tag/release, partially configured signing identity, failed configured signatures, self-signed production substitution, incomplete release assets, source/main drift, or release/package read-back failures.

The active versioning document is part of the version integrity contract. `scripts/audit_version.py` must fail when the current candidate/version/tag/GHCR/checklist markers do not agree with root `VERSION`.

## Changelog and release notes

`CHANGELOG.md` must contain a `## <VERSION>` section. `scripts/release_notes.py` extracts only that exact section for public release notes.

Historical release notes retain their original version/channel wording. Active documentation describes the current source candidate and published Stable history separately.

## 1.1.8 release checklist

The exact candidate must pass:

- Go formatting, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- the complete Python regression suite;
- real loopback FTP and shared connection-manager regressions;
- strict FTPS no-downgrade and SFTP host-key verification/pinning;
- privacy-safe child-process diagnostic classification without raw diagnostic retention/exposure;
- trusted Linux transport and credential-bearing AskPass executable provenance;
- rooted local transfer/filesystem safeguards and state-directory identity validation;
- Windows installer directory, registry, shortcut, legacy-uninstaller and integrated-uninstall ownership/identity checks;
- exact-object verified-handle cleanup where the Windows security contract requires it;
- responsive startup/minimum geometry and mixed-DPI destination-monitor handling;
- Windows x64/x86 Setup + Portable production builds and release-artifact verification;
- Linux amd64/arm64/i386 DEB + tar.gz production builds and executable parity verification;
- supplemental Debian/Ubuntu/Fedora/Portable package metadata/parity CI;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 native install/remove/GUI smoke;
- 24-language localization and authentic Windows UI evidence from the exact final release-prep source;
- release asset allow-list and SHA-256 verification;
- Authenticode verification when configured, otherwise explicit `WINDOWS_AUTHENTICODE=unsigned` metadata;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v1.1.8` branch validation;
- Stable GitHub Release `ghostftp-v1.1.8` with `prerelease=false`, exact 15-file read-back and GHCR `1.1.8` publication/read-back.

## Historical numbering

Old 0.x and prior 1.x references in historical `CHANGELOG.md`, `docs/RELEASE-HISTORY.md`, immutable Git tags/releases and package digests are intentional records and must not be mass-rewritten.
