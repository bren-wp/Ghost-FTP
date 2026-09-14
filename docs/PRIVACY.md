# Ghost FTP privacy

Ghost FTP **0.0.6** is designed without application telemetry, advertising, behavioral analytics, fingerprinting, hidden crash upload or a mandatory Ghost FTP account.

## Native application model

Windows, Linux, Android and the macOS development frontend send user-directed protocol traffic to the server the user selected. Ghost FTP does not require a product storage cloud or synchronization backend.

The application does not intentionally collect usage analytics, advertising identifiers, filenames for marketing, saved server profiles for cloud sync, passwords/passphrases for a Ghost FTP backend, or machine inventory for an account service.

## Browser helper

The official Chrome, Edge, Firefox and Opera helpers parse/copy explicitly entered FTP-family targets locally. They request zero broad browser/host permissions, contain no analytics or remote code, do not persist credentials and have no supported browser-to-desktop launch/handoff.

## Web FTP privacy boundary

Web FTP is intentionally documented separately because browser JavaScript cannot open raw FTP/FTPS/SFTP sockets. The browser sends one requested operation and the entered connection data over HTTPS to the trusted Ghost FTP web host. PHP then opens the remote protocol connection and performs the operation.

The supplied application:

- has no registration or user account database;
- does not use localStorage, IndexedDB or cookies for connection credentials;
- keeps connection values in page memory;
- does not intentionally persist passwords, remote listings or transfer history;
- does not intentionally write request credentials or transfer contents to application logs;
- closes the protocol transport at the end of the request.

The hosting server necessarily sees credentials and transfer data in memory while performing the operation. Reverse proxies, PHP runtime, OS/network infrastructure and the hosting provider remain inside that trust boundary and may have independent logs or monitoring. Deploy Web FTP only on infrastructure you trust, require HTTPS and disable request-body logging.

## Saved credentials

Native saved credentials are opt-in and platform-local. Windows uses the current-user protection boundary; Linux uses maintained protected-secret handling; macOS development uses Keychain-backed protection; current Android saved-site state remains intentionally non-secret.

Declining persistence can still save non-secret profile fields without turning a newly entered password into durable state.

## Transfer data

Ghost FTP necessarily reads files selected for upload and writes data selected for download. Native transfer data moves only between the local device and the selected server through the selected protocol.

For Web FTP, transfer bytes pass through the trusted web host because of the browser transport limitation. Temporary download/editor files use the operating-system temporary directory and are deleted by request cleanup paths; operators should still configure secure temporary storage and process isolation.

## Diagnostics and logs

User-facing errors are designed to avoid reproducing passwords/passphrases or protected profile payloads. System tools and remote servers may still emit host/path/account details, so logs should be treated as potentially sensitive and minimized before sharing.

## Release infrastructure

Official artifacts are built in GitHub Actions from repository source. The canonical 0.0.6 public allow-list is **13 platform artifacts / 16 public files**. The GHCR object `ghcr.io/bren-wp/ghost-ftp:0.0.6` is a verified distribution bundle, not a runtime product backend.

Production signing keys, PFX files, Android keystores and Apple notarization credentials are external protected release secrets and must never be committed to repository source or embedded in public artifacts.

## Documentation and screenshots

Authentic UI evidence comes from the read-only exact-head Windows/Linux/Android screenshot workflow. The verified 0.0.6 evidence set contains 15 runtime images and records source SHA, workflow run, byte count and SHA-256 values. Mockups or generated approximations are not substitutes for runtime evidence.

The product website reuses byte-identical repository copies of selected verified images; it does not load remote tracking pixels or analytics media.

## Third parties

A remote server can observe protocol traffic necessary for the connection and follows its operator's policy. GitHub, operating systems, hosting providers, timestamp/signing/notarization services and other infrastructure have their own independent policies. Those services are not Ghost FTP application telemetry.

## No sale of user data

Ghost FTP contains no advertising data-broker integration and no product backend designed to sell application usage, profile, transfer or browsing data.

See [Security](SECURITY.md), [Web](WEB.md), [Dependencies](DEPENDENCIES.md), [Signing](SIGNING.md) and [Packages](PACKAGES.md).
