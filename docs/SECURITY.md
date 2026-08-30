# Security

ByFTP keeps transport, credential, remote-path and filesystem checks fail-closed.

## Desktop

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use stable local snapshots, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are DPAPI protected. External processes receive a minimized environment and bounded output. AskPass remains parent/token constrained and clears inherited credential state before secret use.

## Android

Android uses a separate native protocol boundary:

- SFTP requires an OpenSSH-style `SHA256:` host-key fingerprint.
- 1.2.0 Base64-decodes that fingerprint and requires exactly a 32-byte SHA-256 digest before SSHJ receives the canonical value.
- Permissive/promiscuous SFTP verifiers are forbidden.
- Explicit/implicit FTPS use platform trust, endpoint/hostname checking and `PROT P`.
- ByFTP Android source is audited against custom trust-all `X509TrustManager` and permissive hostname-verifier patterns.
- Plain FTP is retained only for explicit compatibility and is unencrypted.
- FTP/FTPS map UI `/` to the authenticated login/account root; unavailable `PWD` falls back to login-relative paths.
- Remote paths fail closed on traversal, `.`/`..`, duplicate separators, backslashes, NULs and noncanonical single-component names instead of rewriting them.
- Host/port validation is bounded and username/password control characters are rejected.
- Passwords and passphrases remain session-only and are never written to preferences, databases, files or a project backend.
- From 1.3.0, Android may remember only the last successful **non-secret** connection metadata in app-private preferences: protocol, host, port, username and SFTP fingerprint. The preset model has no password/passphrase field and is revalidated before restoration.
- The password field is cleared after every connect attempt and again during Activity teardown. Persistence callbacks receive pre-extracted non-secret preset values instead of retaining a credential-bearing `ConnectionConfig` for convenience.
- Storage Access Framework is used instead of broad storage permissions; backup/device-transfer rules exclude app data, including the private preset preferences.
- Multi-file upload validates every target name before transfer, rejects duplicate target names in the same batch and keeps remote path handling inside the same canonical validator used by normal file operations.
- Active/pending clients and file-picker state are cleaned during lifecycle teardown.

Android SFTP private-key import remains deferred until Android Keystore-backed handling, import validation and migration semantics are implemented and audited.

## iOS

The native iOS release supports FTP and implicit FTPS through Apple Network.framework.

- Implicit FTPS uses platform TLS validation and protects the FTP data channel with `PBSZ 0` / `PROT P`.
- No custom trust-all callback or global App Transport Security bypass is used.
- FTP UI paths and server-reported login roots reject traversal, backslashes, duplicate separators, NULs and dot components.
- EPSV is preferred. PASV fallback deliberately ignores the server-provided host and opens the data connection only to the user-selected endpoint, preventing passive-response host redirection and improving NAT/shared-hosting compatibility.
- FTP command names are constrained and command arguments/credentials reject CR/LF/NUL control characters.
- Network reads/listings are bounded; downloads are streamed to temporary files rather than accumulated unbounded in memory.
- Session generation prevents stale asynchronous work from mutating a disconnected/newer session.
- The UI password is cleared after each connect attempt; the FTP actor clears its own password copy after authentication; the app disconnects on background transition.
- From 1.3.0, iOS may remember only non-secret protocol/host/port/username metadata in Keychain using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`. The persistent `ConnectionPreset` has no password field and the restored values are passed through `ConnectionConfig` validation before use.
- Preset persistence uses a pre-extracted non-secret value rather than retaining the original credential-bearing config through the asynchronous connection completion path.
- Upload uses security-scoped document access; batch selections validate all remote names before transfer. Downloaded temporary copies can be explicitly cleared.
- No `UserDefaults` credential store, WebView wrapper, analytics SDK or fixed runtime service endpoint is part of the iOS application.

Explicit FTPS and SFTP are not exposed on iOS until separately audited native implementations exist.

## Mobile package integrity and signing

`scripts/package_android.py` validates required APK structure and rejects unsafe/duplicate archive members before versioned staging.

`scripts/package_ios.py` validates `ByFTP.app` bundle identifier/version, executable presence and Mach-O format, rejects symlinks and unsafe archive paths, then creates the normal `Payload/ByFTP.app` unsigned IPA plus an unsigned app ZIP.

Android debug signing is for development/testing only; the optimized Android release APK is unsigned. iOS IPA/app ZIP artifacts are also unsigned. Production Android and Apple signing identities/provisioning material must stay outside the repository. See [Signing](SIGNING.md).

## Reporting

Report vulnerabilities through the repository Security policy. Never publish working secrets, signing credentials, production endpoints or customer data in a public issue.