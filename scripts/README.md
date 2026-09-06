# Ghost FTP build and audit scripts

This directory contains the maintained Windows/Linux packaging, security, privacy and verification tooling used by Ghost FTP.

## Canonical release path

GitHub Releases are assembled by `.github/workflows/release.yml`. There is one public release publication surface: the immutable GitHub Release for `ghostftp-vX.Y.Z`. Keeping one path prevents version, tag and artifact drift.

Important maintained tools include:

- `release_notes.py` — generates release notes from `CHANGELOG.md`.
- `make_payload.py` — creates the verified Windows Setup payload.
- `verify_release.py` / `verify_bundle.py` — verify release/bundle structure and integrity.
- `audit_desktop_surface.py` — rejects retired Web/PWA application surfaces.
- `audit_platform_contract.py` — enforces Windows/Linux-only application targets.
- `audit_security.py` — security-policy regression checks.
- `audit_privacy.py` — privacy/telemetry regression checks.
- `audit_dependencies.py` — rejects unexpected dependency and tracking SDK drift.
- platform build/package helpers referenced by CI or documented local workflows.

NuGet, Web/PWA and mobile application packaging are not part of the maintained Ghost FTP desktop release contract.

## Release identity

Ghost FTP uses `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Historical GhostFTP tags are immutable and are not reused.

## Build invariants

Production workflows disable Go telemetry and use controlled dependency resolution. Windows and Linux artifacts are assembled only after all platform jobs pass, then checksummed in `SHA256.txt`.

Do not add another script or package registry that independently publishes Ghost FTP. New release logic belongs in the canonical GitHub Release workflow and must preserve tag immutability, checksum generation, read-back verification and explicit signing status.

## Security

Never embed production signing keys, tokens, FTP credentials, recovery secrets or private certificates in scripts. Signing credentials must remain outside the public repository.
