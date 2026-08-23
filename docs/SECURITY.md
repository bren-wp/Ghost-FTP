# Security

ByFTP keeps transport, credential and filesystem checks fail-closed.

## Desktop

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use a stable local snapshot before network transfer, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are protected by DPAPI. External processes receive a minimized environment and bounded output handling.

## Android

Android 1.1.0 uses a separate native protocol boundary:

- SFTP refuses to connect without an expected OpenSSH-style `SHA256:` host-key fingerprint.
- The expected fingerprint is passed to SSHJ's built-in host-key verifier; permissive/promiscuous verifiers are forbidden.
- Explicit and implicit FTPS explicitly use the Android/JVM platform trust manager, enable TLS endpoint/hostname checking and protect the data channel with `PROT P`.
- Android application source is audited to reject custom `X509TrustManager` implementations, empty trust callbacks, permissive hostname verifiers and Commons Net trust-all helpers.
- Plain FTP is retained only for explicit server compatibility and is unencrypted; prefer FTPS or SFTP.
- Generic cleartext traffic is disabled in the Android network-security configuration.
- Passwords are session-only and are not written to SharedPreferences, databases, files or a project backend.
- Android local files are selected through the Storage Access Framework; the manifest does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write storage permissions.
- Cloud-backup and device-transfer extraction rules exclude application data domains.
- Host input rejects URLs/path forms and ports are restricted to 1–65535.
- Remote operation names are required to be single path components.
- Network work runs outside the UI thread; active and pending connections are closed during Activity destruction and late UI callbacks are ignored.

Android SFTP private-key import is not present in 1.1.0. It must not be added until stable Android Keystore-backed secret/key handling, import validation and migration behavior have dedicated tests and audit coverage.

Android lint remains fail-closed with warnings treated as errors. The `TrustAllX509TrustManager` lint detector is disabled only because current third-party dependency JARs contain unrelated implementations that trigger dependency-level findings; `scripts/audit_android.py` independently rejects those patterns anywhere in ByFTP Android source and requires explicit platform trust plus endpoint checking in the FTPS adapter.

## Reporting

Report security vulnerabilities through the repository Security policy rather than publishing working secrets or exploit details in a public issue.
