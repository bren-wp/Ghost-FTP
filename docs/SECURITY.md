# Security

Ghost FTP security rules are intentionally fail-closed.

## Protocols

### FTP

FTP is an intentional compatibility protocol and is not encrypted.

### Explicit FTPS

FTPS must validate:

- certificate trust;
- hostname identity;
- configured TLS behavior.

### SFTP

Windows and Linux desktop SFTP uses strict SSH host-key trust and pinning.

Android SFTP remains hidden until the Android transport has maintained strict host-key verification.

## Credentials

Credential persistence is opt-in and platform-local.

Never log plaintext passwords, private-key passphrases, private keys or production signing secrets.

## File operations

Destructive actions must require explicit user intent and must respect the configured delete-confirmation policy.

Remote permissions and Remote Edit operate through the active authenticated connection and may not fabricate local-only success states.

## Release security

Protected production publication must fail when required publisher identities are unavailable.

No workflow may generate a replacement long-lived production identity and silently publish it as official.

## Privacy

Ghost FTP contains no application telemetry, advertising or hidden transfer proxy.

## Reporting

Provide:

- Ghost FTP version;
- platform;
- protocol;
- synthetic reproduction steps;
- privacy-safe logs.

Do not include real credentials or server-private data.
