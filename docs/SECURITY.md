# Security

ByFTP keeps transport, credential, remote-path, account-state, archive-processing and filesystem checks fail-closed.

**Current release: 1.9.0**

## Desktop

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use stable local snapshots, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are DPAPI protected. External processes receive a minimized environment and bounded output. AskPass remains parent/token constrained and clears inherited credential state before secret use.

Windows 1.9.0 Setup does not build or install a standalone `Uninstall.exe`. The Setup payload contains only verified `ByFTP.exe` plus a schema-2 integrity manifest. Upgrade rollback covers the application/App Paths transaction, while cleanup of a legacy pre-1.8.0 `Uninstall.exe` happens only after the new application commit and refuses unsafe symlink/reparse/non-regular paths. `BUILD-WINDOWS.ps1`, `scripts/package_windows_bundles.ps1` and public release staging independently reject uninstall-named generated assets.

The maintained desktop toolchain is Go 1.27.1 and all Windows/Linux/macOS CI/release jobs pin that exact version with local toolchain mode and Go telemetry disabled.

## Android

Android uses a separate native protocol boundary:

- SFTP requires an OpenSSH-style `SHA256:` host-key fingerprint.
- The fingerprint is Base64-decoded and must be exactly a 32-byte SHA-256 digest before SSHJ receives the canonical value.
- Permissive/promiscuous SFTP verifiers are forbidden.
- Explicit/implicit FTPS use platform trust, endpoint/hostname checking and `PROT P`.
- ByFTP Android source is audited against custom trust-all `X509TrustManager` and permissive hostname-verifier patterns.
- Plain FTP is retained only for explicit compatibility and is unencrypted.
- FTP/FTPS map UI `/` to the authenticated login/account root; unavailable `PWD` falls back to login-relative paths.
- Remote paths fail closed on traversal, `.`/`..`, duplicate separators, backslashes, NULs and noncanonical single-component names instead of rewriting them.
- Host/port validation is bounded and username/password control characters are rejected.
- Passwords and passphrases remain session-only and are never written to preferences, databases, files or a project backend.
- Android may remember only the last successful non-secret connection metadata in app-private preferences: protocol, host, port, username and SFTP fingerprint.
- The password field is cleared after every connect attempt and again during Activity teardown.
- Storage Access Framework is used instead of broad storage permissions; backup/device-transfer rules exclude app data.
- Multi-file upload validates every target name before transfer and rejects duplicate target names in the same batch.
- Active/pending clients and file-picker state are cleaned during lifecycle teardown; stale UI callbacks are ignored through the lifecycle guard.

Android SFTP private-key import remains deferred until Android Keystore-backed handling, import validation and migration semantics are implemented and audited.

The 1.9.0 Android build uses **AGP 9.4.0**, **Gradle 9.7.1**, JDK 17, API 37 and build-tools 36.0.0. The toolchain update does not relax TLS, SSH, lint, lifecycle or signing invariants.

## iOS

The native iOS release supports FTP and implicit FTPS through Apple Network.framework.

- Implicit FTPS uses platform TLS validation and protects the FTP data channel with `PBSZ 0` / `PROT P`.
- No custom trust-all callback or global App Transport Security bypass is used.
- FTP UI paths and server-reported login roots reject traversal, backslashes, duplicate separators, NULs and dot components.
- EPSV is preferred. PASV fallback ignores the server-provided host and opens the data connection only to the user-selected endpoint.
- FTP command names are constrained and command arguments/credentials reject CR/LF/NUL control characters.
- Network reads/listings are bounded; downloads are streamed to temporary files rather than accumulated unbounded in memory.
- Session generation prevents stale asynchronous work from mutating a disconnected/newer session.
- The UI password is cleared after each connect attempt; the FTP actor clears its own password copy after authentication; the app disconnects on background transition.
- iOS may remember only non-secret protocol/host/port/username metadata in Keychain using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`.
- Upload uses security-scoped document access; failed/stale temporary downloads are cleaned up.
- No `UserDefaults` credential store, WebView wrapper, analytics SDK or fixed runtime service endpoint is part of the iOS application.

Explicit FTPS and SFTP are not exposed on iOS until separately audited native implementations exist.

## ByFTP WEB

Release 1.9.0 treats authentication, encrypted profiles, archive extraction, privileged diagnostics and runtime policy as security state.

- `app.json`, `users.json`, rate-limit counters, encrypted `profiles.json`, privacy-bearing `preferences.json` and legacy migration state fail closed when the primary generation is corrupt or missing. Adjacent `.bak` files remain available only for explicit operator recovery.
- Login and registration consume their rate-limit budget atomically before password/account mutation work. Login consumes the source-IP budget first and stops before account-specific state when the IP is already blocked.
- Saved FTP/SFTP secrets are bound to the exact endpoint/account/key identity. Blank secret fields cannot inherit credentials after host, port, username or private-key identity changes.
- SFTP requires a pinned SHA-256 host fingerprint before `ClientFactory` creates a client; `SftpClient` verifies the connected server key against that pin.
- Password changes and password rehashes use compare-and-swap against the exact password-hash generation that was verified.
- Authentication completion rechecks that verified hash generation under the atomic registry update used to publish login metadata.
- User deletion is two-phase and retryable. The registry keeps an inactive `deleting` row until workspace cleanup is verified; workspace-root symlinks are unlinked rather than traversed.
- First setup and configuration recovery never rotate the encryption key over pre-existing protected data.
- Remote connection target validation resolves and validates exact target IPs; transports connect to those validated targets rather than re-resolving a hostname after the security decision.
- ZIP extraction completes archive topology validation, existing-remote conflict validation and local decompression/materialization before any remote `mkdir` or upload. The 512 MiB limit is checked against actual cumulative decompressed bytes, staged temp files are always cleaned and late archive corruption cannot trigger earlier remote writes.
- `diagnostics.php` requires administrator authorization because runtime/PHP/OpenSSL/hosting capability data is privileged operational information.

ByFTP WEB uses SameSite=Strict/HttpOnly cookies, CSRF tokens, cross-site POST filtering, CSP, no-store responses for sensitive surfaces and explicit no-index protections. Saved connection secrets are encrypted with Sodium secretbox or AES-256-GCM/OpenSSL fallback using an installation-specific 256-bit key.

The deployable WEB release ZIP is built by `scripts/package_web.py` from tracked production files only. The packager rejects symlinks/unsafe paths/case-fold collisions and validates archived VERSION, Composer metadata and PWA cache namespace. Runtime users, config, cache and backup data cannot enter the public archive unless incorrectly committed to Git, which the repository audit separately blocks as generated/runtime state.

## Repository and release integrity

The repository-wide fail-closed audit checks every tracked Git file. It rejects case-insensitive path collisions, Windows-reserved path components, tracked symlinks, committed build/cache artifacts, invalid UTF-8 or unexpected NUL content, BOMs, trailing whitespace, missing final newlines, unresolved merge-conflict markers and stale explicit current-release markers.

The no-uninstaller invariant remains explicit: `cmd/uninstaller`, a Windows build path that produces an uninstall binary, a payload `--uninstaller` option or an uninstaller PE-resource role must not re-enter the maintained release surface.

For 1.9.0, `scripts/prepare_release.ps1` additionally enforces exactly 15 platform artifacts before metadata and 18 final public files. Unexpected or uninstall-named assets fail closed before publication.

The repository audit is local and deterministic. It does not weaken runtime trust rules, execute tracked source files, scan network endpoints or upload repository content.

## Mobile package integrity and signing

`scripts/package_android.py` validates required APK structure and rejects unsafe/duplicate archive members before versioned staging.

`scripts/package_ios.py` validates `ByFTP.app` bundle identifier/version, executable presence and Mach-O format, rejects symlinks and unsafe archive paths, then creates the normal `Payload/ByFTP.app` unsigned IPA plus an unsigned app ZIP.

Android debug signing is for development/testing only; the optimized Android release APK is unsigned. iOS IPA/app ZIP artifacts are also unsigned. Production Android and Apple signing identities/provisioning material must stay outside the repository. See [Signing](SIGNING.md).

## Reporting

Report vulnerabilities through the repository Security policy. Never publish working secrets, signing credentials, production endpoints or customer data in a public issue.
