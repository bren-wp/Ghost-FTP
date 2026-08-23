# ByFTP build and verification tools

This directory contains the repository's build, audit, release and verification utilities. They are development and CI tools only; they do not add runtime dependencies to the ByFTP application.

## Build tools

- `BUILD-LOCAL.sh` — local/offline cross-build smoke check.
- `BUILD-LINUX.sh` — production Linux DEB builds for amd64, arm64 and i386.
- `BUILD-MACOS.sh` — production macOS Universal application/package build.
- `make_payload.py` — creates the verified Windows installer payload.
- `pe_resources.py` — writes Windows PE icon and VERSIONINFO resources.
- `generate_brand_assets.py` — reproducibly generates and verifies PNG/ICO brand assets.

The canonical Windows production build is [`BUILD-WINDOWS.ps1`](../BUILD-WINDOWS.ps1) in the repository root.

## Audit tools

- `audit_localization.py` — verifies the English-first localization contract, supported language catalogs and version binding.
- `audit_version.py` — verifies that `VERSION` is the single production version source.
- `audit_docs.py` — checks local documentation links, the documentation index and version-neutral document titles.
- `audit_security.py` — protects important filesystem, credential, transfer and session security invariants.
- `audit_privacy.py` — enforces the privacy and network policy.
- `audit_release.py` — validates the production release workflow, platform matrix, bundle contract and centralized publisher.
- `audit_release_version_guard.py` — prevents mutation of already-published version lines.

## Release and verification tools

- `verify_release.py` — validates Windows PE files and release security properties.
- `verify_bundle.py` — fail-closed validation of Windows release ZIP contents, paths and `BUNDLE-SHA256.txt`.
- `release_notes.py` — generates release notes from the matching `CHANGELOG.md` section.
- `publish_release.ps1` — centralized GitHub Release publication with tag/commit and asset integrity checks.
- `test_release_tools.py` — Python standard-library regression tests for release tooling.

## Production rules

The build and release pipeline is intentionally fail-closed:

1. `VERSION` is the only production version source.
2. Go telemetry must be disabled before a production build.
3. Production builds run with `GOPROXY=off` and `GOSUMDB=off`.
4. The current module graph must contain no unexpected external Go modules.
5. Security, privacy, localization, documentation and release audits must pass.
6. Windows bundles are checked against an explicit allowlist and SHA-256 manifest.
7. GitHub Release publication is performed only through `publish_release.ps1`.

## Documentation

See [GitHub releases](../docs/GITHUB-RELEASES.md) for the release workflow, [Release verification](../docs/RELEASE-VERIFICATION.md) for verification requirements, [Testing](../docs/TESTING.md) for the test matrix and [Security](../docs/SECURITY.md) for the security model.
