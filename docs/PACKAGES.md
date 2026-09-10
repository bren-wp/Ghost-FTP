# Ghost FTP GitHub Packages

Ghost FTP **0.0.3** publishes a verified **distribution bundle** to GitHub Packages alongside the canonical GitHub Release.

## Current package reference

```text
ghcr.io/bren-wp/ghost-ftp:0.0.3
```

The GHCR object is a verified **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under:

```text
/ghostftp-release/
```

The canonical GitHub Release contains **14 platform artifacts / 17 public files**, including `SHA256.txt`, `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`; the package is built from that same verified directory.

## Publication contract

Package publication occurs only after the quality, universal Windows and distro-specific Linux release jobs complete successfully. The workflow binds OCI version/revision labels to root `VERSION` and exact `GITHUB_SHA`, publishes the exact semantic version plus current aliases and `latest`, then verifies `ghcr.io/bren-wp/ghost-ftp:0.0.3` after push.

The package is distribution infrastructure only. Ghost FTP has no hidden package-backed runtime service, account backend, telemetry endpoint or online profile dependency.

## Latest-only package retention

After successful GitHub Release publication and remote asset read-back, `.github/workflows/release-retention.yml` removes obsolete Ghost FTP package versions while preserving the package version carrying the current exact semantic-version tag. A missing current package is a retention failure.

## Integrity and Windows signing

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` records source commit, version, tag, platform set and Windows signing state.

Authenticode verification **when a trusted production certificate is configured** is fail-closed. Without a trusted production certificate, Windows files are explicitly unsigned and metadata records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The project never generates a self-signed production identity and presents it as a trusted publisher.

The package and GitHub Release use the same current public policy: `ghostftp-v0.0.3`, `prerelease=false`, exact 17-file read-back and latest-only retention after successful verification.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
