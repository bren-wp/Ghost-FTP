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

The maintained public line began at **0.1.0**. Every **0.x.y** release was a **Beta** prerelease. Historical tags/releases remain immutable for traceability.

Version **1.0.0** is the first **Stable** public release. The current production workflow is stable-only and rejects `MAJOR=0`; it does not publish new prereleases.

## Post-1.0 increments

### PATCH

Use a patch increment for compatible bug fixes, security/privacy hardening, performance improvements, documentation corrections and packaging/release fixes that do not intentionally break supported workflows.

Example: `1.0.1`.

### MINOR

Use a minor increment for backward-compatible functionality, substantial workflow improvements or new optional capabilities.

Example: `1.1.0`.

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
- GitHub Package semantic-version tag.

Source entry points retain development defaults and receive production versions only through the maintained build process.

## Stable GitHub Release rule

For `MAJOR >= 1`, the workflow publishes a normal GitHub Release with:

```text
draft=false
prerelease=false
```

The release must point to the exact `main` commit that passed production quality/build gates. Existing version tags are immutable and cannot be moved to different source.

## Windows signing state is not a version channel

Authenticode availability does not create a separate semantic version or prerelease channel. Instead, the actual state is recorded in `BUILD-METADATA.txt`.

When a protected production certificate exists:

```text
WINDOWS_AUTHENTICODE=signed
WINDOWS_TRUST_MODE=authenticode+sha256+github-provenance
```

When no production certificate exists:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

The workflow never substitutes a self-signed/development certificate and represents it as a production identity. Organizations that require Authenticode can enforce that policy independently from semantic versioning.

## GitHub Packages versioning

Stable releases publish the verified distribution bundle using the semantic tag:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

and compatible aliases:

```text
1.0
1
latest
```

Downstream automation should prefer the full semantic version or immutable OCI digest. The production workflow does not publish pre-1.0 package aliases.

## Version source integrity

The release workflow rejects:

- malformed semantic versions;
- manual version values different from root `VERSION`;
- any current `MAJOR=0` publication request;
- existing release tags pointing to another commit;
- configured Authenticode output that fails validation;
- incomplete/extra release asset sets;
- Release state where `draft` or `prerelease` is true;
- GitHub Release checksum read-back mismatches;
- GitHub Package source/version/revision or embedded manifest mismatches.

## Changelog and release notes

`CHANGELOG.md` must contain a `## <VERSION>` section. `scripts/release_notes.py` extracts only that exact section and rejects pre-1.0 publication.

Historical release notes retain their original version/channel wording. Active documentation describes the current stable baseline.

## Stable release checklist

A stable version is release-ready only when the exact candidate passes:

- Windows and Linux production CI;
- FTP/FTPS/SFTP reliability regressions;
- SFTP host-key trust/fingerprint validation;
- transfer correctness, retry/cancel/recovery and generation binding;
- Windows Setup transaction/rollback and Portable verification;
- localization checks;
- security/privacy/dependency/repository audits;
- release asset allow-list and SHA-256 verification;
- truthful Windows signed/unsigned metadata and configured-signature verification;
- normal GitHub Release (`prerelease=false`) remote read-back;
- GitHub Package push, fresh pull and embedded metadata/checksum read-back;
- current documentation/support/license synchronization.

## Historical numbering

Old 0.x references in `CHANGELOG.md` and `docs/RELEASE-HISTORY.md` are intentional historical records and must not be mass-rewritten to 1.0.0.
