# Ghost FTP GitHub Packages

Ghost FTP **0.0.5** publishes a verified **distribution bundle** to GitHub Packages alongside the canonical GitHub Release.

## Current package reference

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The release workflow also maintains semantic-version aliases and `latest`, but the exact `0.0.5` tag is the verification identity for this release transaction.

The GHCR object is a verified **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under:

```text
/ghostftp-release/
```

The canonical GitHub Release contains **14 platform artifacts / 17 public files**, including `SHA256.txt`, `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`; the package is built from that same verified release directory.

The Android development APK and optional browser companion source are independently maintained source surfaces and are intentionally not included in this public Windows/Linux release directory or GHCR bundle.

## Publication contract

Package publication occurs only after quality, universal Windows and canonical Linux release jobs succeed. The workflow:

- binds version/revision labels to root `VERSION` and exact `GITHUB_SHA`;
- publishes the exact semantic version plus current aliases and `latest`;
- verifies `ghcr.io/bren-wp/ghost-ftp:0.0.5` after push;
- builds from the same exact 17-file release directory used for GitHub Release publication;
- never uses the package as a hidden product backend or runtime service.

## Latest-only package retention

After successful release publication/read-back, `.github/workflows/release-retention.yml` removes obsolete Ghost FTP package versions while preserving the current exact `0.0.5` package identity.

## Integrity

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` records source commit, version, tag, platform set, universal/native Windows payload contract, Linux distro families and Windows signing state.

Authenticode verification **when a trusted production certificate is configured** is fail-closed. When no trusted certificate is configured, Windows files are explicitly unsigned and metadata records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The project never generates a self-signed production identity and presents it as a trusted publisher.

## Publication boundary

GitHub Packages is distribution infrastructure only. Ghost FTP has no hidden product backend, account service, telemetry endpoint or package-backed runtime dependency.

The package and GitHub Release use the same current policy: `ghostftp-v0.0.5`, `prerelease=false`, exact 17-file release read-back and latest-only retention after successful verification.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
