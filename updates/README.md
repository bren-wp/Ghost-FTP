# Ghost FTP Desktop Update System

This directory is the **operator/developer source of truth** for the Ghost FTP desktop update service.

The Windows and Linux applications expose only simple product language such as “Check for Updates”, “Download & install”, verification status and restart status. File formats, service paths, signatures and publishing mechanics documented here are not shown in the application UI.

## Current production contract

- Desktop application service: `https://ghostftp.com/updates/latest.json`
- Supported updater targets: `windows-x86_64` and `linux-x86_64`
- Windows update package: canonical signed NSIS Setup executable
- Linux update package: canonical signed AppImage
- Signature verification: mandatory Tauri updater signature verification
- Public key: embedded in the desktop application
- Private key: GitHub Actions secret only; never committed or uploaded to the update-service host
- Android: distributed independently as a verified APK release asset; Android does not use this desktop updater contract

The public service follows the Tauri v2 updater contract: `version`, optional human-readable `notes` / `pub_date`, and per-platform `url` + signature content.

## Directory layout

- `latest.template.json` — canonical public-service template synchronized from `version.json`
- `schema/latest.schema.json` — strict schema matching the public desktop updater response
- `scripts/build-manifest.mjs` — creates a production response from signed package files
- `scripts/verify-manifest.mjs` — validates a generated response before deployment
- `DEPLOYMENT.md` — exact update-service paths, headers, Apache/Nginx examples and atomic upload procedure
- `RELEASE_RUNBOOK.md` — end-to-end release/operator procedure
- `SECURITY.md` — signing-key handling, signature rules and recovery/rotation policy
- `hosting/.htaccess.example` — shared-hosting/Apache example
- `hosting/nginx.conf.example` — Nginx example

The former separate preview/stable template files were removed because the desktop application currently has one canonical public update service. Channel policy belongs in release/version metadata, not in a second incompatible wire format.

## CI/release flow

1. Pull requests build and test normal Windows/Linux packages without access to signing secrets.
2. A canonical build on `main` always produces the normal Windows/Linux packages. When the Tauri signing secret is configured it additionally uses `src-tauri/updater-release.conf.json` to produce signed in-app updater artifacts.
3. With signing enabled, Tauri produces the normal NSIS Setup/AppImage plus their signature files. Without signing, normal packages may still be used for CI/preview QA, but a `stable` release is not publishable.
4. The release workflow verifies all exact-SHA gates.
5. The release workflow normalizes packages and signatures, then generates an update-service response.
6. The release publishes versioned packages/signatures plus `GhostFTP-v<version>-latest.json` and `GhostFTP-v<version>-Update-Service.zip`.
7. `channel: stable` requires both signed Windows/Linux updater artifacts, a verified manifest and the matching Update-Service package. Missing proof aborts before stable tag/release publication. Explicit preview/CI builds may omit that bundle.
8. After the GitHub Release and its asset digests are verified, the update-service operator may atomically replace `/updates/latest.json`.

See `DEPLOYMENT.md` for the update-service hosting procedure.

## Required GitHub Actions secrets for stable publication

- `TAURI_SIGNING_PRIVATE_KEY`
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` when the key is password protected

The private key must never be stored in source, release assets, update-service files, workflow logs or documentation examples.

## Failure behavior

- If the update service is unavailable, Ghost FTP still starts and transfers files normally.
- Quiet background checks never interrupt startup.
- Manual checks show a safe user-facing error without revealing service paths or transport implementation details.
- An update that cannot be cryptographically verified is not installed.
- Older releases remain available in GitHub Releases for manual reinstall/rollback procedures.
