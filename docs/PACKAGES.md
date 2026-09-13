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

Android, macOS and browser companion source are independently maintained source/development surfaces and are intentionally not included in this public Windows/Linux release directory or GHCR bundle.

## Publication contract

Package publication occurs only after quality, universal Windows and canonical Linux release jobs succeed. The workflow:

- binds version/revision labels to root `VERSION` and exact `GITHUB_SHA`;
- requires the official Windows Setup and Portable artifacts to pass trusted Authenticode verification;
- publishes the exact semantic version plus current aliases and `latest`;
- verifies `ghcr.io/bren-wp/ghost-ftp:0.0.5` after push;
- builds from the same exact 17-file release directory used for GitHub Release publication;
- never uses the package as a hidden product backend or runtime service.

## Latest-only package retention

After successful release publication/read-back, `.github/workflows/release-retention.yml` removes obsolete Ghost FTP package versions while preserving the current exact `0.0.5` package identity.

## Integrity

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` records source commit, version, tag, platform set, universal/native Windows payload contract, Linux distro families and verified Windows signing state.

Official Windows publication requires trusted Authenticode. `Publish Ghost FTP` fails when the production signing identity is unavailable or either public Windows executable does not verify successfully. A successful official bundle records:

```text
WINDOWS_AUTHENTICODE=signed
```

There is no supported unsigned official publication state under the current contract. Local development and ordinary CI Windows artifacts may be unsigned, but they are not copied into an official release/GHCR bundle unless the public signing gate has subsequently produced and verified the signed release artifacts.

The project never generates a self-signed production identity and presents it as a trusted publisher.

## Distribution-bundle immutability

The package is built only from the already assembled release directory. It does not rebuild Ghost FTP inside the container context, fetch runtime dependencies or mutate platform artifacts after release verification. Version and source-revision labels bind the package to the exact release transaction.

## Publication boundary

GitHub Packages is distribution infrastructure only. Ghost FTP has no hidden product backend, account service, telemetry endpoint or package-backed runtime dependency.

The package and GitHub Release use the same current policy: `ghostftp-v0.0.5`, `prerelease=false`, exact 17-file release read-back, signed-only official Windows publication and latest-only retention after successful verification.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
