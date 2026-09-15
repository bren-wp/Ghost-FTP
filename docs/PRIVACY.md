# Ghost FTP privacy

Ghost FTP **0.0.6** is designed without application telemetry, advertising, behavioral analytics, fingerprinting, hidden crash upload or a mandatory Ghost FTP account.

## Device-to-server model

Windows, Linux, Android and the macOS native source surface send user-directed protocol traffic to the server selected by the user. Ghost FTP does not require a product storage cloud, synchronization backend or remote credential proxy.

The removed Web FTP application is no longer part of this repository. Browser extension development must preserve the same privacy boundary: if native protocol access is required, it must be performed by a local Ghost FTP companion on the user's device, not by a Ghost FTP-operated remote relay.

## Data not collected by the product

Ghost FTP does not intentionally collect:

- usage analytics or behavioral events;
- advertising identifiers or marketing attribution;
- browsing history or page content;
- file names for marketing/analytics;
- saved server profiles for cloud synchronization;
- passwords/passphrases for a Ghost FTP backend;
- device fingerprints or machine inventory for an account service.

## Saved credentials

Credential persistence is opt-in and platform-local. Passwords must never be placed in URLs, plaintext configuration files or logs. When a platform exposes protected credential storage, Ghost FTP uses that boundary rather than inventing a weaker application-level plaintext store.

Changing endpoint, account or key identity must not silently reuse a protected credential under the new identity.

## Browser extensions

Extension code is packaged locally and must not load remote executable JavaScript, analytics SDKs or tracking code. Permissions must remain at the minimum required for implemented functionality.

A future native-messaging bridge may receive connection and file-operation requests only from the installed Ghost FTP extension, process them locally and return bounded results. It must not forward credentials or transfer data to Ghost FTP infrastructure.

## Transfer data

Ghost FTP reads files explicitly selected for upload and writes files selected for download. Transfer bytes move between the local device and the selected destination server through the selected protocol. Temporary/staged files are cleaned after success, cancellation or failure according to platform semantics.

## Diagnostics and logs

User-facing errors must be useful without exposing passwords, passphrases, protected profile payloads, stack traces, internal paths or implementation details. Diagnostic logs should minimize host/account/path data and must never contain plaintext credentials.

## Release infrastructure

Official artifacts are built from repository source in GitHub Actions. Build infrastructure, signing/notarization services and GitHub have their own independent policies; they are not Ghost FTP application telemetry.

Production signing keys, PFX files, Android keystores and Apple notarization credentials remain external protected secrets and must never be committed to source or embedded as reusable secrets in public artifacts.

## Documentation and screenshots

Authentic UI evidence is repository-local and exact-head bound. Mockups or generated approximations are not substitutes for runtime evidence.

## No sale of user data

Ghost FTP contains no advertising data-broker integration and no product backend designed to sell application usage, profile, transfer or browsing data.

See [Security](SECURITY.md), [Dependencies](DEPENDENCIES.md), [Signing](SIGNING.md) and [Packages](PACKAGES.md).
