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
- Package URLs must use HTTPS and the public application endpoint remains first-party on ghostftp.com.
- Production desktop builds require the Tauri updater signing key; a missing key fails the production native build.
- Tauri v2 updater artifacts are enabled with `bundle.createUpdaterArtifacts: true`.
- The canonical signed release assets are the Windows NSIS installer plus `.sig` and the Linux AppImage plus `.sig`.
- The release workflow refuses publication when either updater signature is missing.
- The website update bridge must verify the release digest and signature pair and expose only first-party Ghost FTP download URLs to applications.
- Update-service failure must never prevent Ghost FTP from starting.

Signing private keys and other secrets do not belong in this repository.


## CI signing contract

The production native build receives the signing key through the GitHub Actions secret `TAURI_SIGNING_PRIVATE_KEY` and the optional `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`. Pull-request validation explicitly disables updater artifact generation so secrets are not required in PR builds. Pushes to `main` fail closed when the private key is unavailable.

The release workflow expects the signed files beside the native bundles and publishes them with canonical names:

- `GhostFTP-Windows-x64-Setup-v<VERSION>.exe`
- `GhostFTP-Windows-x64-Setup-v<VERSION>.exe.sig`
- `GhostFTP-Linux-x86_64-v<VERSION>.AppImage`
- `GhostFTP-Linux-x86_64-v<VERSION>.AppImage.sig`

The signing private key must never be committed to the repository or copied into website packages.
