# Security

ByFTP keeps transport, credential, remote-path and filesystem checks fail-closed.

## Desktop

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use a stable local snapshot before network transfer, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are protected by DPAPI. External processes receive a minimized environment and bounded output handling. AskPass invocation remains parent/token constrained and its inherited credential environment is cleared before secret use. Remaining startup/AskPass fallback text is English-first; localization cleanup does not alter these security checks.

## Android

Android uses a separate native protocol boundary:

- SFTP refuses to connect without an expected OpenSSH-style `SHA256:` host-key fingerprint.
- The expected fingerprint is passed to SSHJ's built-in host-key verifier; permissive/promiscuous verifiers are forbidden.
- Explicit and implicit FTPS use the Android/JVM platform trust manager, enable TLS endpoint/hostname checking and protect the data channel with `PROT P`.
- Android application source is audited to reject custom `X509TrustManager` implementations, empty trust callbacks, permissive hostname verifiers and Commons Net trust-all helpers.
- Plain FTP is retained only for explicit server compatibility and is unencrypted; prefer FTPS or SFTP.
- After FTP/FTPS login, the server working directory is treated as the UI account root. If `PWD` is unavailable, operations remain login-relative rather than forcing an arbitrary server `/`.
- FTP UI paths reject traversal, empty components, backslashes, NUL characters and noncanonical absolute forms before remote operations.
- Generic cleartext traffic is disabled in the Android network-security configuration for platform-aware networking.
- Passwords are session-only and are not written to SharedPreferences, databases, files or a project backend.
- Android local files are selected through the Storage Access Framework; the manifest does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write storage permissions.
- Cloud-backup and device-transfer extraction rules exclude application data domains.
- Host input rejects URLs/path forms and ports are restricted to 1–65535.
- Remote operation names are required to be single path components.
- Network work runs outside the UI thread; active and pending connections are closed during Activity destruction and late UI callbacks are ignored.
- Pending download-picker state is cleared after every result, disconnect and Activity destruction.

Android SFTP private-key import is not present. It must not be added until stable Android Keystore-backed secret/key handling, import validation and migration behavior have dedicated tests and audit coverage.

Android lint remains fail-closed with warnings treated as errors for debug and release variants. The `TrustAllX509TrustManager` lint detector is disabled only because current third-party dependency JARs contain unrelated implementations that trigger dependency-level findings; `scripts/audit_android.py` independently rejects those patterns anywhere in ByFTP Android source and requires explicit platform trust plus endpoint checking in the FTPS adapter.

## APK integrity and signing

`scripts/package_android.py` validates that Android build outputs are non-empty ZIP/APK containers with `AndroidManifest.xml`, `classes.dex` and `resources.arsc`, rejects duplicate/unsafe archive paths and stages only versioned artifact names.

The public debug APK is signed only with the standard Android debug identity and is for development/testing. The optimized release APK is intentionally unsigned. Production Android distribution requires a stable private signing identity kept outside the repository. See [Signing](SIGNING.md).

## Reporting

Report security vulnerabilities through the repository Security policy rather than publishing working secrets, credentials, production endpoints or exploit details in a public issue.
