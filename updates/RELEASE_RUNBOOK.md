# Ghost FTP Desktop Update Release Runbook

## Purpose

This is the operator checklist for publishing a Windows/Linux in-app update after the source has passed the normal Ghost FTP release gates.

## Optional in-app updater signing

The normal stable GitHub release does not require private updater keys. To additionally publish the Windows/Linux in-app update bundle, configure these GitHub Actions repository secrets:

- `TAURI_SIGNING_PRIVATE_KEY`
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if the private key is encrypted

The public verification key is embedded in the desktop application. If updater signing is enabled, the private key must stay outside the repository and update-service host.

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

Pull requests remain secret-free and build normal packages. On `main`, missing updater-signing secrets do not block a stable GitHub release; the release contains the verified application packages, checksums and QA evidence but deliberately omits the in-app update-service bundle.

If either updater signature is present, both signatures, the exact-version manifest and the matching Update-Service package become an all-or-nothing set. Partial updater proof fails closed before tag/release publication.

## Release workflow

The release workflow:

1. waits for exact-SHA quality, protocol, Android and Windows hardening gates;
2. downloads native artifacts from the successful exact-SHA main build;
3. normalizes versioned public asset names;
4. detects whether both updater signatures are available;
5. when signed updater artifacts exist, creates the Tauri-compatible update-service response from the signature contents;
6. validates the response against the exact version;
7. when signing is enabled, creates `GhostFTP-v<version>-latest.json` and `GhostFTP-v<version>-Update-Service.zip`;
8. enforces that signatures, manifest and Update-Service package are either all present and consistent or all absent;
9. generates SHA-256 checksums;
10. re-validates the optional updater set in `publish-release.sh` before touching the tag;
11. creates/updates the immutable version release at the exact source SHA;
12. verifies GitHub asset digests.

## Update-service publication

If the release contains the signed Update-Service package, it may be deployed as described in `DEPLOYMENT.md`. A release without that package remains a valid manual GitHub release and must not publish an unsigned in-app update manifest.

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
