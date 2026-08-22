# ByFTP build and verification tools

This directory contains the helper tools used by the production pipeline. They use standard Go, Python, PowerShell and platform tooling and do not add runtime dependencies to ByFTP.

- `BUILD-LOCAL.sh` — local offline Windows x64 cross-build verification
- `BUILD-LINUX.sh` — Linux amd64/arm64/i386 DEB production builds
- `BUILD-MACOS.sh` — macOS Universal PKG production build
- `generate_brand_assets.py` — reproducible PNG/ICO asset generation and verification
- `audit_localization.py` — English-first runtime/localization contract and full-label UI guards
- `audit_version.py` — verifies `VERSION` as the canonical production version source
- `audit_release_version_guard.py` — blocks production changes to an already published version without a VERSION bump
- `audit_docs.py` — local documentation links, index coverage and version-neutral titles
- `audit_security.py` — filesystem, connection, credential, transfer and lifecycle security invariants
- `audit_privacy.py` — telemetry, runtime network-policy and secret-handling invariants
- `audit_release.py` — static contract for the release workflow, bundles and fail-closed publisher
- `test_release_tools.py` — standard-library regression tests for release ZIP verification
- `make_payload.py` — installer payload creation and integrity
- `pe_resources.py` — PE icon and VERSIONINFO resources
- `verify_release.py` — Windows PE/security verification
- `verify_bundle.py` — final Windows ZIP path, manifest and SHA-256 verification
- `release_notes.py` — release notes generated from the exact matching CHANGELOG section
- `publish_release.ps1` — idempotent GitHub Release publisher with tag/commit and asset SHA-256 verification

The full publishing process is documented in [`docs/IZDAVANJE-NA-GITHUBU.md`](../docs/IZDAVANJE-NA-GITHUBU.md), the release checklist in [`docs/PROVJERA-IZDANJA.md`](../docs/PROVJERA-IZDANJA.md), and the quality layers in [`docs/TESTIRANJE.md`](../docs/TESTIRANJE.md).

Historical script names and documentation paths may remain stable when renaming them would break existing automation or links; English is the canonical content language for current production tooling.