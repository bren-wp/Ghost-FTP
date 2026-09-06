# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source baseline: **1.0.0 Stable**.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use:

```text
ghostftp-vMAJOR.MINOR.PATCH
```

For the current release:

```text
VERSION=1.0.0
TAG=ghostftp-v1.0.0
CHANNEL=Stable
PRERELEASE=false
```

## Historical pre-1.0 policy

The maintained public line began at **0.1.0**. Every **0.x.y** release was a **Beta** prerelease. Historical tags/releases remain unchanged for traceability.

Version **1.0.0** is the first **Stable** public release. Stable does not mean development stops; it means the maintained release contract is no longer published as a prerelease and backward compatibility/stability receive normal production-release priority.

## Post-1.0 increments

### PATCH

Use a patch increment for compatible bug fixes, security/privacy hardening, performance improvements, documentation corrections and packaging/release fixes that do not intentionally break supported workflows.

Example:

```text
1.0.1
```

### MINOR

Use a minor increment for backward-compatible functionality, substantial workflow improvements or new optional capabilities.

Example:

```text
1.1.0
```

### MAJOR

Use a major increment for intentionally incompatible product contracts that require clear migration guidance.

## Binary/package identity

The same semantic version is injected into:

- Windows application binaries;
- Windows **Setup** packages;
- Windows **Portable** packages;
- Linux DEB metadata;
- release notes and build metadata;
- GitHub Release tag/title;
- stable GitHub Package version tag.

Source entry points retain `version = "dev"` and receive production versions only through build linker flags.

## Stable GitHub Release rule

For `MAJOR >= 1`, the publication workflow selects the stable channel and does not apply the GitHub prerelease flag. The official stable release must point to the exact `main` commit that passed release quality gates.

Windows Authenticode is an **optional production hardening layer**, not a prerequisite for stable version identity. When a trusted production PFX is configured through protected Actions secrets, every produced Windows artifact must verify successfully or publication fails. When no production certificate is configured, stable publication may continue only with an explicit `WINDOWS_AUTHENTICODE=unsigned` declaration in `BUILD-METADATA.txt`.

Ghost FTP never generates a self-signed production certificate and presents it as a trusted publisher identity.

## GitHub Packages versioning

Stable releases publish the verified distribution bundle using the immutable semantic tag:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

and compatible stable aliases:

```text
1.0
1
latest
```

Downstream automation should prefer the full semantic version or an immutable OCI digest. Historical Beta channel logic does not publish stable package aliases.

## Version source integrity

The release workflow rejects:

- a malformed semantic version;
- a manual version different from `VERSION`;
- an existing release tag that points to a different commit;
- a partially configured production signing identity;
- a configured signing identity whose produced Windows signatures do not verify;
- any attempt to substitute a generated/self-signed production identity for a trusted publisher certificate;
- release output whose expected asset set is incomplete;
- release/package read-back failures.

Absence of a production Authenticode certificate by itself is not a versioning failure. The resulting Windows signing state must instead remain truthfully `unsigned` throughout build metadata, documentation and verification.

## Changelog and release notes

`CHANGELOG.md` must contain a `## <VERSION>` section. `scripts/release_notes.py` extracts only that exact section for the public `RELEASE-NOTES.txt` file.

Historical release notes retain their original version/channel wording. Active documentation must describe the current stable baseline.

## First stable checklist

Ghost FTP 1.0.0 is considered release-ready only when the exact candidate passes:

- Windows and Linux production CI;
- FTP/FTPS/SFTP reliability regressions;
- SFTP host-key trust/fingerprint validation;
- transfer correctness, retry/cancel/recovery and generation binding;
- Windows Setup transaction/rollback and Portable verification;
- localization checks;
- security/privacy/dependency/repository audits;
- release asset allow-list and SHA-256 verification;
- Authenticode verification when a trusted production certificate is configured, or explicit unsigned metadata when it is absent;
- GitHub Release normal-release verification;
- stable GitHub Package publication/read-back;
- current documentation/support/license synchronization.

## Historical numbering

Old 0.x references in `CHANGELOG.md` and `docs/RELEASE-HISTORY.md` are intentional historical records and must not be mass-rewritten to 1.0.0.
