# Privacy

Ghost FTP is designed to perform file-transfer work without product telemetry.

## Data flow

Windows, Linux and Android send user-directed protocol traffic only to infrastructure selected by the user. Ghost FTP does not require a product storage cloud or synchronization backend.

Ghost FTP does not intentionally collect analytics, advertising identifiers, behavioral telemetry, FTP credentials, remote directory contents or transfer payloads.

Platform-local settings, connection profiles and bookmarks remain on the user's device according to the platform implementation. Credential storage, where available, uses the maintained local protection boundary.


Signing keys and publisher credentials are protected release secrets and are never committed to source or shipped as application data.

macOS is not an active product platform and has no current credential/storage contract.

Browser extensions are retired and are not part of the active Ghost FTP source or release surface.
