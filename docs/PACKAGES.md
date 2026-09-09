# Ghost FTP GitHub Packages

Ghost FTP **0.0.1** publishes a verified **distribution bundle** to GitHub Packages alongside the canonical GitHub Release.

## Current package reference

```text
ghcr.io/bren-wp/ghost-ftp:0.0.1
```

The release workflow also maintains semantic-version aliases and `latest`, but the exact `0.0.1` tag is the verification identity for this release transaction.

The GHCR object is a verified **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under:

```text
/ghostftp-release/
```

The canonical GitHub Release contains **12 platform artifacts / 15 public files**, including `SHA256.txt`, `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`; the package is built from that same verified release directory.

## Publication contract

Package publication occurs only after the quality, Windows and Linux release jobs complete successfully. The release workflow:

- binds the package version and OCI revision labels to root `VERSION` and exact `GITHUB_SHA`;
- publishes the exact semantic version;
- publishes current aliases derived from the semantic version plus `latest`;
- verifies `ghcr.io/bren-wp/ghost-ftp:0.0.1` after push;
- does not use the package as a hidden application backend or runtime service.

## Latest-only package retention

The project retains only the current public Ghost FTP version. After a successful release publication and remote release read-back, `.github/workflows/release-retention.yml` removes obsolete Ghost FTP package versions.

Retention preserves any package version carrying the exact current semantic-version tag (`0.0.1` for this release) and removes superseded package versions. A missing current package is a retention failure rather than a reason to silently remove package verification.

## Integrity

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` records source commit, version, tag, platform set and Windows signing state.

Authenticode verification **when a trusted production certificate is configured** is fail-closed. When no trusted certificate is configured, Windows files are explicitly unsigned and metadata records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The project never generates a self-signed production identity and presents it as a trusted publisher.

## Publication boundary

GitHub Packages is distribution infrastructure only. Ghost FTP has no hidden product backend, account service, telemetry endpoint or package-backed runtime dependency.

The package and GitHub Release use the same current public release policy: `ghostftp-v0.0.1`, `prerelease=false`, exact 15-file release read-back and latest-only retention after successful verification.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
