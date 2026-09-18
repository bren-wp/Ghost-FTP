# Privacy

Ghost FTP is designed to perform file-transfer work without product telemetry.

## Data flow

Windows, Linux and Android send user-directed protocol traffic only to infrastructure selected by the user. Ghost FTP does not require a product storage cloud or synchronization backend.

## No telemetry

Ghost FTP does not intentionally collect:

- analytics;
- advertising identifiers;
- behavioral telemetry;
- FTP credentials;
- remote directory contents;
- transfer payloads.

## Saved state

Platform-local settings, connection profiles and bookmarks remain on the user's device according to the platform implementation. Credential storage, where available, uses the maintained local protection boundary.

## Browser helpers

Browser helper packages are local companion tools. Any desktop handoff must be sanitized and must never include passwords, private keys or automatic connection authorization.

## Release infrastructure

Signing keys and publisher credentials are protected release secrets and are never committed to source or shipped as application data.

macOS is not an active product platform and has no current credential/storage contract.
