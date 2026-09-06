# Ghost FTP build and audit scripts

This directory contains the maintained Windows/Linux packaging, security, privacy and verification tooling used by Ghost FTP.

## Canonical stable release path

GitHub Releases are assembled by `.github/workflows/release.yml`. The canonical user-facing installable artifacts are attached to the immutable GitHub Release for `ghostftp-vX.Y.Z`.

Maintained publication is stable-only (`MAJOR >= 1`, `draft=false`, `prerelease=false`). Historical 0.x prereleases remain repository history but the production workflow does not create new prereleases.

Stable releases also publish the same verified `release/` assembly as an OCI distribution bundle to GitHub Packages at `ghcr.io/bren-wp/ghost-ftp`. The package is a distribution surface for automation and integrity verification, not a runtime container and not a second build of the application.

## Important maintained tools

- `release_notes.py` — generates stable release notes from the matching `CHANGELOG.md` section and rejects pre-1.0 publication.
- `make_payload.py` — creates the verified Windows Setup payload.
- `verify_release.py` / `verify_bundle.py` — verify release/bundle structure and integrity.
- `audit_desktop_surface.py` — rejects retired non-desktop application surfaces.
- `audit_platform_contract.py` — enforces Windows/Linux-only application targets.
- `audit_security.py` — security-policy regression checks.
- `audit_privacy.py` — privacy/telemetry regression checks.
- `audit_dependencies.py` — rejects unexpected dependency and tracking-SDK drift.
- `audit_docs.py` — validates active documentation/version/release-policy synchronization.
- `audit_release.py` — validates stable-only Release, truthful signing metadata, artifact and GHCR contracts.
- platform build/package helpers referenced by CI or documented local workflows.

Retired non-desktop/mobile packaging is not part of the maintained Ghost FTP release contract.

## Release identity

Ghost FTP uses root `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Historical tags are immutable and are not reused for another commit. `1.0.0` and later maintained releases are normal GitHub Releases with `prerelease=false`.

## Build invariants

Production workflows disable Go telemetry, use `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off`, and build Windows/Linux from the exact source revision. Public artifacts are assembled only after platform/quality gates pass, then checksummed in `SHA256.txt`.

## Windows signing invariant

Release scripts must never invent signing state.

If protected production Authenticode credentials are configured, every final Windows artifact must verify successfully. If they are not configured, the release metadata must say:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

Development/self-signed credentials are allowed only for CI signing-pipeline tests and must never be represented as production publisher identity. The repository must never contain a production private key/password.

## Release/Package transaction

The canonical workflow:

1. builds and verifies Windows/Linux artifacts;
2. assembles the exact 12-file Release allow-list;
3. publishes/verifies the stable GitHub Release;
4. checks exact `main` state again;
5. builds GHCR from only `release/` using `FROM scratch` and `--network=none`;
6. pushes stable aliases;
7. removes local package images;
8. pulls the semantic-version tag back from GHCR;
9. verifies OCI source/version/revision labels and embedded `SHA256.txt`/`BUILD-METADATA.txt`.

Registry authorization uses a temporary Docker credential directory and must not be copied into package contents.

Do not add an independent release script or package-registry publisher. New distribution logic belongs in the canonical workflow and must preserve tag immutability, checksum generation, exact asset allow-lists, `prerelease=false`, package read-back and explicit cryptographic state.

## Verification

Before publication the maintained workflow/audits verify, as applicable:

- Go formatting, race tests and vet;
- security, privacy, dependency, version, localization and documentation contracts;
- Windows x64/x86 Setup and Portable production builds;
- Linux amd64/arm64/i386 package metadata;
- exact 9-platform-artifact / 12-public-file Release set;
- stable `draft=false` and `prerelease=false`;
- configured Authenticode signatures or explicit unsigned trust metadata;
- GitHub Release remote asset/SHA read-back;
- GitHub Package fresh-pull/metadata/SHA read-back.

A script change that weakens one of these checks is a release/security change, not documentation-only maintenance.

## Security

Never embed production signing keys, tokens, FTP credentials, private-key passphrases, recovery secrets or private certificates in scripts. Signing credentials and registry authorization are supplied only by the protected CI environment and must not be printed to logs or committed to source.
