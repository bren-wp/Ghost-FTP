# Ghost FTP build and audit scripts

This directory contains the maintained packaging, security, privacy and verification tooling used by Ghost FTP.

## Canonical release path

GitHub Releases are assembled by `.github/workflows/release.yml`. There is no second PowerShell release publisher: keeping one publication path reduces version/tag drift and stale artifact naming.

Important maintained tools include:

- `release_notes.py` — generates Ghost FTP release notes from `CHANGELOG.md`.
- `package_web.py` — creates the shared-hosting web archive used by CI.
- `audit_security.py` — security-policy regression checks.
- `audit_privacy.py` — privacy/telemetry regression checks.
- platform build/package helpers that are still referenced by CI or local documented workflows.

## Release identity

Ghost FTP uses `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Historical GhostFTP tags are immutable and are not reused.

## Build invariants

Production workflows disable Go telemetry and use controlled dependency resolution. Final release filenames are assembled only after all platform jobs pass, then checksummed in `SHA256.txt`.

Do not add a second script that independently creates or force-updates GitHub Releases. New release logic belongs in the canonical workflow and must preserve tag immutability, checksum generation and explicit signing status.

## Security

Never embed production signing keys, tokens, FTP credentials, recovery secrets or private certificates in scripts. Signing credentials must remain outside the public repository.
