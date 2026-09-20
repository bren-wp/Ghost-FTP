# Ghost FTP Updates

This folder contains the public update-channel contract for Ghost FTP.

The desktop application checks the official Ghost FTP update endpoint and accepts update metadata only when it matches the expected channel format. Production update artifacts are intended to be signed before publication.

## Layout

- `channels/preview.template.json` — release-candidate channel template.
- `channels/stable.template.json` — stable channel template.
- `schema/latest.schema.json` — update manifest schema.
- `scripts/build-manifest.mjs` — deterministic manifest generator.
- `scripts/verify-manifest.mjs` — manifest validator.

## Update policy

- Preview and stable channels are separate.
- Every manifest identifies a version, publication time, minimum supported version and platform packages.
- Package URLs must use HTTPS.
- Signatures are required before a manifest is production-ready.
- Older releases remain available in GitHub Releases for manual rollback/install.
- Update-service failure must never prevent Ghost FTP from starting.

Signing private keys and other secrets do not belong in this repository.
