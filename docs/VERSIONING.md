# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source baseline: **1.1.0 Stable**.

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
VERSION=1.1.0
TAG=ghostftp-v1.1.0
CHANNEL=Stable
PRERELEASE=false
```

The previously published `ghostftp-v1.0.0` tag is immutable release history and must remain on its original release commit.

## Historical pre-1.0 policy

The maintained public line began at **0.1.0**. Every **0.x.y** release was a **Beta** prerelease. Historical tags/releases remain unchanged for traceability.

Version **1.0.0** is the first **Stable** public release.

## Post-1.0 increments

### PATCH

Use a patch increment for compatible bug fixes, security/privacy hardening, performance improvements, documentation corrections and packaging/release fixes that do not intentionally add product functionality.

Example: `1.1.1`.

### MINOR

Use a minor increment for backward-compatible functionality, substantial workflow improvements or new optional capabilities.

Ghost FTP **1.1.0** is a minor release because it adds the Classic Light desktop appearance while preserving the 1.x connection/transfer contract.

### MAJOR

Use a major increment for intentionally incompatible product contracts that require clear migration guidance.

## Binary/package identity

The same semantic version is injected into Windows application binaries, Setup/Portable packages, Linux DEB metadata, release notes/build metadata, GitHub Release tag/title and the stable GitHub Package tag.

Source entry points retain `version = "dev"` and receive production versions only through build linker flags.

## Stable GitHub Release rule

For `MAJOR >= 1`, publication uses the stable channel and `prerelease=false`. The official stable release must point to the exact `main` commit that passed release quality gates.

The canonical release branch trigger is created only from exact current main using:

```text
release/ghostftp-v<version>
```

The trigger rejects a branch whose version differs from `VERSION` or whose commit differs from current `main`.

Windows Authenticode is an **optional production hardening layer**, not a prerequisite for stable version identity. When a trusted production PFX is configured through protected Actions secrets, every produced Windows artifact must verify successfully or publication fails. When no production certificate is configured, stable publication may continue only with explicit unsigned metadata:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Ghost FTP never generates a self-signed production certificate and presents it as a trusted publisher identity.

## GitHub Packages versioning

Stable 1.1.0 publication uses:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.0
```

with compatible stable aliases after successful registry publication/read-back:

```text
1.1
1
latest
```

Downstream automation should prefer the full semantic version or immutable OCI digest.

## Version source integrity

The release workflow rejects a malformed version, a manual version different from `VERSION`, an existing conflicting tag, partially configured signing identity, failed configured signatures, self-signed production substitution, incomplete release assets, or release/package read-back failures.

Absence of a production Authenticode certificate by itself is not a versioning failure. The resulting Windows signing state must remain truthfully `unsigned` throughout build metadata and verification, with `WINDOWS_AUTHENTICODE=unsigned` recorded when no production certificate is configured.

## Changelog and release notes

`CHANGELOG.md` must contain a `## <VERSION>` section. `scripts/release_notes.py` extracts only that exact section for the public release notes.

Historical release notes retain their original version/channel wording. Active documentation describes the current stable baseline.

## 1.1.0 release checklist

The exact candidate must pass:

- `go test -race ./...` and `go vet ./...`;
- Windows and Linux production CI;
- FTP/FTPS/SFTP reliability regressions;
- SFTP host-key trust and protected-secret lifetime regressions;
- transfer correctness, retry/cancel/recovery and generation binding;
- Windows Setup transaction/rollback and Portable verification;
- Linux amd64/arm64/i386 package and DEB verification;
- localization, appearance and UI stability checks;
- security/privacy/dependency/repository/documentation audits;
- release asset allow-list and SHA-256 verification;
- Authenticode verification when configured, otherwise explicit `WINDOWS_AUTHENTICODE=unsigned` metadata;
- GitHub Release normal-release read-back;
- stable GHCR publication/read-back;
- current documentation and authentic screenshot synchronization.

## Historical numbering

Old 0.x and 1.0.0 references in `CHANGELOG.md` and `docs/RELEASE-HISTORY.md` are intentional historical records and must not be mass-rewritten.
