# Security

ByFTP keeps transport and filesystem checks fail-closed.

## Desktop

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use a stable local snapshot before network transfer, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are protected by DPAPI. External processes receive a minimized environment and bounded output handling.

## Android

Android 1.1.0 uses a separate native protocol boundary:

- SFTP refuses to connect without an expected OpenSSH-style `SHA256:` host-key fingerprint.
- The expected fingerprint is passed to SSHJ's built-in host-key verifier; permissive/promiscuous verifiers are forbidden.
- Explicit and implicit FTPS enable TLS endpoint/hostname checking and private data-channel protection (`PROT P`).
- Plain FTP is retained only for server compatibility and is explicitly unencrypted.
- Passwords are session-only and are not written to SharedPreferences, files or a project backend.
- Android local files are selected through the Storage Access Framework; the manifest does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write storage permissions.
- Host input rejects URLs/path forms and ports are restricted to 1–65535.
- Remote operation names are required to be single path components.

Android SFTP private-key import is not present in 1.1.0. It must not be added until stable Android Keystore-backed secret/key handling, import validation and migration behavior have dedicated tests and audit coverage.

## Reporting

Report security vulnerabilities through the repository Security policy rather than publishing working secrets or exploit details in a public issue.
