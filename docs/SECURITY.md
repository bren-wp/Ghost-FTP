# Security

Ghost FTP security policy applies to the active Windows, Linux and Android applications.

## Protocol trust

- FTPS must verify TLS certificates and hostnames.
- Desktop SFTP must preserve strict host-key trust/pinning.
- Android SFTP remains hidden until an equivalent maintained strict host-key verification boundary exists.
- UI work must never disable or bypass protocol verification.

## Credentials

Credential persistence is opt-in and platform-local where implemented.

- Windows uses the maintained current-user protection boundary.
- Linux uses maintained protected-secret handling and provenance checks.
- Android saved-site state remains intentionally non-secret.

Secrets must not be embedded in screenshots, browser helpers, logs, release metadata or source control.

## File and transfer safety

- normalize and validate paths before file mutation;
- keep local/remote ownership explicit;
- fail closed when required state cannot be loaded;
- do not silently retry destructive operations;
- keep transfer identity and queue lifecycle consistent.

## Release security

Official releases use exact-source build artifacts, checksums and platform-appropriate signing gates. Missing signing credentials must fail the protected publication path rather than produce a falsely trusted artifact.

macOS signing/notarization is no longer part of the active product because macOS support is retired.

## Reporting

Security reports should identify the exact Ghost FTP version, platform and reproduction steps without including real credentials.
