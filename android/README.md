# ByFTP for Android

ByFTP Android is a native Java client isolated from the Go desktop runtime so mobile lifecycle, permissions, networking, APK packaging and release signing boundaries remain explicit and independently testable.

## Current capabilities

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Passive FTP/FTPS and binary transfers.
- FTPS platform certificate-chain validation, endpoint/hostname checking and protected data channels (`PROT P`).
- Mandatory OpenSSH-style `SHA256:` SFTP host-key pinning.
- Remote directory listing/navigation, refresh, create directory, rename and delete.
- Mobile **Go to path** navigation through the same canonical remote-path validator used by file operations.
- Case-insensitive local file filtering with deterministic directory-first sorting.
- Multi-file upload through Android's Storage Access Framework, plus document-provider downloads.
- Byte-level upload/download progress for FTP, FTPS and SFTP using one transport-neutral local stream wrapper.
- Safe **Stop after current file** for multi-file upload; the current remote write completes before the remaining batch is skipped.
- Shared-hosting diagnostics derived from the existing initial root listing, including secure/plain transport and common web-root/account-home context.
- Compact connected-state UI with a touch-friendly Up / Refresh / Menu action surface and 48dp minimum controls.
- App-private persistence of the last **non-secret** connection metadata (protocol, host, port, username and SFTP fingerprint) for faster reconnects.
- Session-only passwords; password/passphrase values are never stored in the connection preset and the password field is cleared after every connect attempt.
- Cloud-backup and device-transfer exclusions for application data, including app-private preferences.
- Generic cleartext traffic disabled for Android platform-aware networking.
- No analytics, advertising SDK, telemetry backend or mandatory cloud account.

## 1.5.0 shared-hosting diagnostics

Version 1.5.0 adds a small `SharedHostingDiagnostics` model that analyzes the same `next.list("/")` result already used to render the first connected file list. No second diagnostic listing, port scan, external service or hidden network destination is added.

The model recognizes common hosting document roots in deterministic priority: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Only directories qualify. Plain FTP is explicitly reported as unencrypted; FTPS and SFTP are reported as secure transports. SFTP uses home-root context while FTP/FTPS use authenticated account-root context.

The connected summary displays the result, but diagnostics are advisory only. ByFTP does not call `openDirectory()` with the detected root and does not write diagnostic state into `ConnectionPresetStore`. A custom hosting document root therefore remains fully under user/profile control.

The diagnostic model contains no password, passphrase, username, fingerprint or network-operation field. `SharedHostingDiagnosticsTest` verifies priority, secure/plain behavior and rejection of a file masquerading as a web-root name. `audit_android.py` independently blocks secret-bearing diagnostics, diagnostic network behavior, automatic navigation and persistence.

## 1.4.0 transfer update

Version 1.4.0 made long mobile transfers observable without changing the reviewed network dependencies or trust boundaries.

`TransferStreams` wraps the existing local `InputStream` and `OutputStream` objects and reports cumulative bytes only after successful reads or writes. Because progress is measured at this common stream boundary, the existing FTP, explicit/implicit FTPS and SFTP clients do not need separate progress implementations. `TransferStreamsTest` verifies that monitored streams preserve payload bytes and report cumulative byte counts correctly.

When the Android document provider exposes a stable file size, the UI displays percentage progress. If the provider cannot supply a reliable size, ByFTP switches to transferred-byte reporting instead of inventing a percentage. Downloads use the size already reported by the remote listing when available.

Multi-file upload adds **Stop after current file**. The request is checked only after the active upload returns successfully; remaining files are then skipped and the directory is refreshed once. The button does not interrupt the transport thread, close the socket, issue a fabricated success state or leave the FTP/FTPS command sequence half-finished merely to simulate instant cancellation.

## 1.3.0 mobile update

Version 1.3.0 turned the Android client into a more practical daily mobile file manager without adding a UI framework or broadening the dependency surface.

The connection form is hidden after a successful login and replaced by a compact server/account summary, leaving substantially more vertical space for files on phones. The main action row is reduced to the high-frequency **Up**, **Refresh** and **Menu** actions; upload, new folder, direct-path navigation, saved-connection removal and disconnect live in the menu instead of being squeezed into small buttons.

Directory contents are sorted with folders first and then names case-insensitively. The filter field operates only on the already loaded directory, does not trigger network requests and does not mutate transport-owned lists. These rules live in `RemoteEntryList` instead of `MainActivity` and are covered by JUnit regressions.

The document picker supports selecting multiple files. ByFTP validates every remote name before starting the batch, rejects duplicate destination names in one selection and performs uploads sequentially on the existing dedicated I/O executor before refreshing the directory once.

The last successful connection can be restored locally without storing its password. `ConnectionPresetStore` is deliberately restricted to protocol, host, port, username and fingerprint. The store is app-private, application backup/device transfer remains disabled, and `audit_android.py` fails if password/passphrase persistence is introduced.

## Shared-hosting FTP paths

FTP/FTPS records the server working directory immediately after login and treats that directory as the Android UI root `/`. This keeps `public_html` and other account paths inside the authenticated login namespace instead of forcing an unrelated server filesystem root.

If the server cannot report `PWD`, ByFTP falls back to login-relative FTP paths. UI paths are required to be canonical and reject traversal (`..`), empty components, backslashes and NUL characters before a remote operation is issued. Raw server-reported login directories reject CR/LF/NUL before trimming or path normalization.

Diagnostics do not change these mapping rules. A detected root name is only shown in the summary and is never used to bypass `RemotePaths` or silently replace the active directory.

## Security model

Plain FTP remains available only for compatibility and does not encrypt credentials or file contents. Prefer FTPS or SFTP whenever possible.

For FTPS, ByFTP explicitly uses the platform trust manager and endpoint checking. It does not install a permissive `X509TrustManager`, trust-all helper or hostname-verification bypass.

For SFTP, supply the expected OpenSSH-style fingerprint such as `SHA256:AbCd...`. The fingerprint must decode to exactly a 32-byte SHA-256 digest before SSHJ receives it, and the connection fails closed if it is absent, malformed or does not match the server key.

ByFTP validates raw host, port, username, password and fingerprint text for CR/LF/NUL control characters **before** trimming or canonicalization. FTP and SFTP directory entries use the same `RemotePaths.validateName` policy, so edge whitespace and protocol-control characters cannot be accepted by one transport and rejected later by another layer. Mobile rename/create operations pass the exact typed remote name to that validator instead of silently trimming it first.

The FTP/SFTP transport objects do not retain the complete `ConnectionConfig` throughout an active session. Endpoint and trust data are copied separately, while the transport password reference is cleared in `finally` immediately after authentication and again on close. The Activity also clears its password field on both validation failure and completed connection attempts. The post-authentication UI callback carries only explicitly extracted non-secret endpoint metadata, not the credential-bearing `ConnectionConfig`.

Transfer progress is observational: `TransferStreams` does not own or close the transport, does not alter TLS/SSH verification and does not inject thread interruption. Batch stopping happens only at a completed-file boundary. Shared-hosting diagnostics are also observational and operate only on an already loaded listing.

SFTP password authentication remains supported. Private-key import remains intentionally deferred until Android Keystore-backed handling, import validation and migration semantics have dedicated tests and audit coverage.

## Lifecycle and local files

Network work runs on a dedicated single-thread executor rather than the UI thread. The Activity separately tracks pending and active clients; both are closed during destruction, queued work is interrupted and late UI callbacks are ignored.

Download-picker state is cleared immediately after every picker result and also on disconnect/destroy so a stale remote path cannot be reused after a cancelled or incomplete document-provider flow.

Local files are opened only through Android document-provider URIs. ByFTP does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write permissions. Multi-file upload uses `ACTION_OPEN_DOCUMENT` with `EXTRA_ALLOW_MULTIPLE`; downloads continue to use `ACTION_CREATE_DOCUMENT` so the user explicitly controls the destination.

## APK artifacts

The root `VERSION` drives Android `versionName`, `versionCode` and public APK names. CI and the release workflow build and validate both variants:

- `ByFTP-<version>-Android-debug.apk` — installable debug-signed development/test APK.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized, minified and resource-shrunk release APK without a production signature.

The unsigned release APK is **not** a production distribution until it is signed with a stable private Android identity held outside the repository. The project does not commit or fabricate a production keystore.

`scripts/package_android.py` validates APK ZIP structure and required entries before staging either artifact under a versioned name. Release checksums cover both APKs.

## Toolchain

- Android Gradle Plugin 9.3.0
- Gradle 9.7.0
- JDK 17
- compileSdk / targetSdk 37
- minSdk 26 (Android 8.0)
- Apache Commons Net 3.13.0
- SSHJ 0.40.0

No AndroidX/Compose framework was added for the 1.5.0 diagnostic work; the application continues to use platform UI APIs plus the already reviewed network dependencies.

## Build and test

From the repository root with JDK 17, Gradle 9.7.0 and Android SDK 37 installed:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Android lint warnings are treated as errors. The repository also runs `scripts/audit_android.py`, JUnit path/security/version regressions, `SharedHostingDiagnosticsTest`, `RemoteEntryListTest`, `TransferStreamsTest`, and the general security/privacy/release audits.