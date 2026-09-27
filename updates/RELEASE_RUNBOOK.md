# Ghost FTP Desktop Update Release Runbook

## Purpose

This is the operator checklist for publishing a Windows/Linux in-app update after the source has passed the normal Ghost FTP release gates.

## One-time repository setup

Configure GitHub Actions repository secrets:

- `TAURI_SIGNING_PRIVATE_KEY`
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if the private key is encrypted

The public verification key is embedded in the desktop application. The private key must stay outside the repository and website.

## Build behavior

Pull requests:

- compile normal NSIS/AppImage/DEB/RPM packages;
- run tests and package lifecycle smoke checks;
- do not receive update-signing secrets.

`main`:

- requires the Tauri signing private key;
- builds through `src-tauri/updater-release.conf.json`;
- produces the normal Windows NSIS Setup and Linux AppImage;
- produces a signature file for each update package;
- uploads packages/signatures as native-build artifacts.

If the signing secret is missing, the canonical main build must fail rather than publish an update that installed clients cannot verify.

## Release workflow

The release workflow:

1. waits for exact-SHA quality, protocol, Android and Windows hardening gates;
2. downloads the signed native artifacts from the successful main build;
3. normalizes versioned public asset names;
4. publishes Windows Setup/AppImage and their signature files;
5. creates the Tauri-compatible web update response from the **contents** of the signature files;
6. validates that response;
7. creates `GhostFTP-v<version>-latest.json`;
8. creates `GhostFTP-v<version>-Web-Update.zip` containing `updates/latest.json`;
9. generates SHA-256 checksums;
10. creates/updates the immutable version release at the exact source SHA;
11. verifies GitHub asset digests.

## Website publication

After the GitHub release succeeds, deploy only the web update package as described in `WEB_DEPLOYMENT.md`.

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
