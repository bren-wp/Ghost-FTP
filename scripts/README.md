# Ghost FTP build and audit scripts

This directory contains the maintained Windows/Linux packaging, security, privacy and verification tooling used by Ghost FTP.

## Canonical stable release path

GitHub Releases are assembled by `.github/workflows/release.yml`. The canonical user-facing installable artifacts are attached to the immutable GitHub Release for `ghostftp-vX.Y.Z`.

Stable releases also publish the same verified `release/` assembly as an OCI distribution bundle to GitHub Packages at `ghcr.io/bren-wp/ghost-ftp`. The package is a distribution surface for automation and integrity verification, not a runtime container and not a second build of the application.

Keeping Release and Package publication in one workflow prevents version, tag, artifact and provenance drift.

Important maintained tools include:

- `release_notes.py` — generates release notes from the matching `CHANGELOG.md` section.
- `make_payload.py` — creates the verified Windows Setup payload.
- `verify_release.py` / `verify_bundle.py` — verify release/bundle structure and integrity.
- `audit_desktop_surface.py` — rejects retired non-desktop application surfaces.
- `audit_platform_contract.py` — enforces Windows/Linux-only application targets.
- `audit_security.py` — security-policy regression checks.
- `audit_privacy.py` — privacy/telemetry regression checks.
- `audit_dependencies.py` — rejects unexpected dependency and tracking SDK drift.
- `audit_release.py` — validates Stable/Beta release-channel, signing, artifact and package contracts.
- platform build/package helpers referenced by CI or documented local workflows.

Retired non-desktop and mobile application packaging is not part of the maintained Ghost FTP release contract.

## Release identity

Ghost FTP uses `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Historical tags are immutable and are not reused for another commit. `1.0.0` and later stable releases are normal GitHub Releases with `prerelease=false`; pre-1.0 historical development releases remain Beta records.

## Build invariants

Production workflows disable Go telemetry, use `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off`, and build Windows and Linux from the exact source revision. Public artifacts are assembled only after the platform and quality gates pass, then checksummed in `SHA256.txt`.

Stable Windows publication additionally requires the protected trusted Authenticode identity. The repository must never contain the production private key or password.

The GitHub Packages distribution bundle is built with network access disabled from only the already verified `release/` directory. Source worktrees, user data and release secrets must not be copied into that package.

Do not add an independent release script or package-registry publisher. New distribution logic belongs in the canonical release workflow and must preserve tag immutability, checksum generation, exact asset allowlists, stable/prerelease semantics, package read-back verification and explicit signing state.

## Verification

Before publication the maintained workflow and audits verify, as applicable:

- Go formatting, race tests and vet;
- security, privacy, dependency, version, localization and documentation contracts;
- Windows x64/x86 Setup and Portable production builds;
- Linux amd64/arm64/i386 package metadata;
- the exact 9-platform-artifact / 12-public-file release set;
- stable `prerelease=false` state;
- GitHub Release read-back;
- stable GitHub Package registry read-back.

A script change that weakens one of these checks must be treated as a release/security change, not as a documentation-only maintenance edit.

## Security

Never embed production signing keys, tokens, FTP credentials, private-key passphrases, recovery secrets or private certificates in scripts. Signing credentials and registry authorization are supplied only by the protected CI environment and must not be printed to logs or committed to source.
