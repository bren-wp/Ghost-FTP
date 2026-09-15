# Ghost FTP privacy

Ghost FTP **0.0.6** is designed without application telemetry, advertising, behavioral analytics, fingerprinting, hidden crash upload or a mandatory Ghost FTP account.

## Native application model

Windows, Linux, Android and the macOS development frontend send user-directed protocol traffic to the server selected by the user. Ghost FTP does not require a product storage cloud, synchronization backend or application analytics service.

The maintained applications do not intentionally collect usage analytics, advertising identifiers, filenames for marketing, saved server profiles for cloud sync, passwords/passphrases for a Ghost FTP backend or machine inventory for an account service.

## Browser helpers

The official Chrome, Edge, Firefox and Opera helpers are deliberately narrow companion tools. They operate on explicitly entered targets, contain no analytics or remote executable code, do not persist credentials and have no supported browser-to-desktop launch/handoff.

## Saved credentials

Native saved credentials are opt-in and platform-local. Windows uses the current-user protection boundary; Linux uses maintained protected-secret handling; macOS development uses Keychain-backed protection; Android saved-site state remains intentionally non-secret.

Declining secret persistence can still preserve non-secret profile fields without turning a newly entered password into durable state.

## Transfer data

Ghost FTP necessarily reads files selected for upload and writes data selected for download. Transfer data moves between the local device and the server selected by the user through the selected protocol. Ghost FTP does not insert a proprietary file-storage or transfer-relay service between those endpoints.

## Diagnostics and logs

User-facing errors are designed to avoid reproducing passwords, passphrases, private-key contents or raw protected profile payloads. External tools and remote servers may still produce host/path/account details, so diagnostics should be treated as potentially sensitive and minimized before sharing.

The application does not intentionally maintain a persistent runtime activity log or automatic crash-upload backend.

## Runtime network policy

Maintained core runtime code is audited against fixed HTTP(S) endpoints and known telemetry/analytics vendor markers. FTP/FTPS/SFTP network operations are user-directed. Build, signing, GitHub publication and operating-system infrastructure are separate from application telemetry.

## Release infrastructure

Official artifacts are built from repository source. The canonical 0.0.6 public allow-list is **13 platform artifacts / 16 public files**. The GHCR object `ghcr.io/bren-wp/ghost-ftp:0.0.6` is a verified distribution bundle, not a runtime product backend.

Production signing keys, PFX files, Android keystores and Apple notarization credentials are external protected release secrets and must never be committed to repository source or embedded as reusable secrets in public artifacts.

## Documentation and screenshots

Authentic UI evidence comes from exact-head Windows/Linux/Android runtime capture workflows. Maintained screenshots record source/release provenance; generated mockups or manually composed approximations are not substitutes for runtime evidence.

## Third parties

A remote server can observe protocol traffic necessary for the connection and follows its operator's policy. GitHub, operating systems, timestamp/signing/notarization services and other build/distribution infrastructure have their own independent policies. Those services are not Ghost FTP application telemetry.

## No sale of user data

Ghost FTP contains no advertising data-broker integration and no product backend designed to sell application usage, profile, transfer or browsing data.

See [Security](SECURITY.md), [Dependencies](DEPENDENCIES.md), [Signing](SIGNING.md) and [Packages](PACKAGES.md).
