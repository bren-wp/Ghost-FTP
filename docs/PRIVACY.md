# Ghost FTP privacy

Ghost FTP **0.0.6** is designed without application telemetry, advertising, behavioral analytics, fingerprinting, hidden crash upload or a mandatory Ghost FTP account.

## Device-to-server model

Windows, Linux, Android and the macOS native source surface send user-directed protocol traffic to the server selected by the user. Ghost FTP does not require a product storage cloud, synchronization backend or remote credential proxy.

The removed Web FTP application is no longer part of this repository. On the 0.0.7 development branch, browser extensions preserve the same privacy boundary through a local Native Messaging bridge backed by the installed Ghost FTP Engine. FTP/FTPS/SFTP protocol traffic still goes directly from the user's device to the server selected by the user, not through a Ghost FTP-operated relay.

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

Extension code is packaged locally and does not load remote executable JavaScript, analytics SDKs or tracking code. Official browser manifests request exactly `nativeMessaging`; they do not request browser storage, tabs, host permissions or content-script access.

The extension **does not store credentials in browser storage**. Password/passphrase inputs live only in popup memory while required for the explicit operation. If the user elects to save them, persistence is handled by the native Ghost FTP protected profile store. Public profile data returned to the extension contains non-secret metadata and only booleans indicating whether a saved protected secret exists.

A pasted FTP/FTPS/SFTP URL is parsed locally. A password embedded in that URL is deliberately discarded before fields are filled, and URL query/fragment data is not used as connection data.

The Native Messaging background relay exists only to connect explicit extension actions to `com.ghostftp.bridge` and to keep genuine active transfers attached to the local Engine event stream. It does not open a Ghost FTP network service. The native port is closed when there are no browser clients, pending requests or queued/running transfers.

Local file access is rooted to a folder the user selects through the operating-system picker. The native host rejects traversal or symlink escape outside that root.

## Transfer data

Ghost FTP reads files explicitly selected for upload and writes files selected for download. Transfer bytes move directly between the local device and the selected destination server through the selected protocol. Temporary/staged files are cleaned after success, cancellation or failure according to platform semantics.

The browser extension does not receive unrestricted filesystem authority and does not upload directory listings, connection targets, credentials or transfer bytes to Ghost FTP infrastructure.

## Diagnostics and logs

User-facing errors must be useful without exposing passwords, passphrases, protected profile payloads, stack traces, internal paths or implementation details. The browser/native bridge returns errors through the same privacy-safe mapping used by the desktop Engine rather than exposing raw curl, OpenSSH or operating-system diagnostics.

Diagnostic logs should minimize host/account/path data and must never contain plaintext credentials.

## Native host registration privacy

Firefox registration is bound to the fixed extension identity `ghostftp-connection-helper@ghostftp.com`. Chromium-family registration requires explicit extension IDs and exact allowed origins. Wildcard extension origins are not accepted by the maintained generator.

This registration boundary prevents unrelated extensions from being intentionally granted access to the Ghost FTP bridge by the generated manifest.

## Release infrastructure

Official artifacts are built from repository source in GitHub Actions. Build infrastructure, signing/notarization services and GitHub have their own independent policies; they are not Ghost FTP application telemetry.

Production signing keys, PFX files, Android keystores and Apple notarization credentials remain external protected secrets and must never be committed to source or embedded as reusable secrets in public artifacts.

## Documentation and screenshots

Authentic UI evidence is repository-local and exact-head bound. Mockups or generated approximations are not substitutes for runtime evidence.

## No sale of user data

Ghost FTP contains no advertising data-broker integration and no product backend designed to sell application usage, profile, transfer or browsing data.

See [Security](SECURITY.md), [Dependencies](DEPENDENCIES.md), [Signing](SIGNING.md) and [Packages](PACKAGES.md).
