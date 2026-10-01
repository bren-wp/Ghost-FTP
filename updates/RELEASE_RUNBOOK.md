# Ghost FTP Desktop Update Release Runbook

## Purpose

This is the operator checklist for publishing a Windows/Linux in-app update after the source has passed the normal Ghost FTP release gates.

## One-time repository setup

Configure GitHub Actions repository secrets:

- `TAURI_SIGNING_PRIVATE_KEY`
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if the private key is encrypted

The public verification key is embedded in the desktop application. The private key must stay outside the repository and update-service host.

## Build behavior

Pull requests:

- compile normal NSIS/AppImage/DEB/RPM packages;
- run tests and package lifecycle smoke checks;
- do not receive update-signing secrets.

`main`:

- always builds the normal Windows NSIS Setup, Linux AppImage/DEB/RPM and native executables;
- when the Tauri signing private key is configured, also builds through `src-tauri/updater-release.conf.json`;
- with signing enabled, produces a signature file for the Windows Setup and Linux AppImage;
- uploads the normal packages in either mode and the signature files only when they actually exist.

If the signing secret is missing, the canonical native build remains useful for CI/preview QA, but a `stable` release must stop. Stable publication requires both updater signatures, the verified manifest and the matching Update-Service package; the release workflow must fail before tag/release publication rather than silently omitting them.

## Release workflow

The release workflow:

1. waits for exact-SHA quality, protocol, Android and Windows hardening gates;
2. downloads native artifacts from the successful exact-SHA main build;
3. normalizes versioned public asset names;
4. requires both updater signatures when `channel: stable`;
5. creates the Tauri-compatible update-service response from the **contents** of those signature files;
6. validates the response against the exact version;
7. creates `GhostFTP-v<version>-latest.json` and `GhostFTP-v<version>-Update-Service.zip`;
8. for stable, re-checks all four updater assets and verifies the manifest embedded in Update-Service matches the published manifest;
9. generates SHA-256 checksums;
10. invokes a second fail-closed stable updater check in `publish-release.sh` before touching the tag;
11. creates/updates the immutable version release at the exact source SHA;
12. verifies GitHub asset digests.

## Update-service publication

After the GitHub release succeeds, deploy only the update-service package as described in `DEPLOYMENT.md`.

Recommended hosting model:

- `ghostftp.com` serves only the small current update response;
- GitHub Releases serves immutable versioned Windows/Linux packages.

## Manual acceptance after web deployment

Windows:

1. install the previous published version;
2. open Help & About → Updates;
3. run Check for Updates;
4. confirm the new version is offered;
5. download/install;
6. restart;
7. confirm version/build changed;
8. confirm Sites/settings remain intact.

Linux:

1. run the previous AppImage;
2. check for updates;
3. complete update/restart according to the supported updater flow;
4. confirm the new version and saved configuration.

Failure conditions must show product-language errors only. The application UI must not expose internal service paths, file formats or raw transport errors.

## Android

Android release assets are verified and published by the Android/release workflow, but Android does not consume the desktop Tauri update service described here.
