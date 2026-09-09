# Ghost FTP GitHub Packages

Ghost FTP **0.0.1 Beta** is distributed through the canonical GitHub Release. Pre-1.0 Beta releases do **not** publish a Stable GHCR distribution bundle.

## Current Beta policy

For 0.0.1 the public release identity is:

```text
ghostftp-v0.0.1
prerelease=true
```

The canonical GitHub Release contains **12 platform artifacts / 15 public files**, including `SHA256.txt`, `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`.

## Stable package namespace

The reserved Stable package namespace is:

```text
ghcr.io/bren-wp/ghost-ftp
```

A future Stable release may publish:

```text
ghcr.io/bren-wp/ghost-ftp:<stable-version>
```

The GHCR object is a verified **distribution bundle**, **not a runtime container**. Its payload mirrors the canonical release directory under:

```text
/ghostftp-release/
```

## Latest-only package retention

The project retains only the current public Ghost FTP version. After a successful release publication, `.github/workflows/release-retention.yml` removes obsolete Ghost FTP package versions.

Because 0.0.1 is Beta and does not publish a Stable GHCR bundle, retention removes any package versions left from superseded numbering lines. This prevents the package registry from advertising a version that is no longer part of the current public release line.

For a future Stable release, retention preserves the package version carrying the exact current semantic-version tag and removes older package versions.

## Integrity

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` records source commit, version, tag, platform set and Windows signing state.

Authenticode verification **when a trusted production certificate is configured** is fail-closed. When no trusted certificate is configured, Windows files are explicitly unsigned and metadata records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The project never generates a self-signed production identity and presents it as a trusted publisher.

## Publication boundary

GitHub Packages is distribution infrastructure only. Ghost FTP has no hidden product backend, account service, telemetry endpoint or package-backed runtime dependency.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
