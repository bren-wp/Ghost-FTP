# ByFTP for Android

ByFTP Android is a native Java client isolated from the Go desktop runtime so mobile lifecycle, permissions, networking, APK packaging and release signing boundaries remain explicit and independently testable.

## Current capabilities

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Passive FTP/FTPS and binary transfers.
- FTPS platform certificate-chain validation, endpoint/hostname checking and protected data channels (`PROT P`).
- Mandatory OpenSSH-style `SHA256:` SFTP host-key pinning.
- Remote browse, refresh, upload/download, create directory, rename and delete.
- Canonical **Go to path** navigation and deterministic directory-first sorting/filtering.
- Multi-file upload through Android's Storage Access Framework.
- Byte-level transfer progress plus safe **Stop after current file** batch control.
- Shared-hosting diagnostics derived from the existing initial root listing.
- Compact connected-state UI with touch-friendly controls.
- App-private persistence of non-secret protocol/host/port/username/SFTP-fingerprint metadata only.
- Session-only passwords; backup/device-transfer exclusions for application data.
- No analytics, advertising SDK, telemetry backend or mandatory cloud account.

## 1.6.0 repository integrity update

Version 1.6.0 keeps the Android runtime and protocol behavior from 1.5.0 intact while placing every tracked Android source, resource, build and documentation file under the repository-wide integrity gate. The gate rejects non-portable tracked paths, generated build/cache output, invalid text encoding, unresolved merge markers, trailing whitespace and stale explicit current-release markers before release packaging begins.

The 1.6.0 maintenance pass also removes the obsolete `connected_to` string resource that remained after the diagnostics-aware connection summary replaced it. Android lint therefore stays warning-as-error without adding a baseline or suppression.

## Shared-hosting diagnostics

Version 1.5.0 added `SharedHostingDiagnostics`, derived from the same `next.list("/")` result already used to render the first connected file list. No second diagnostic listing, port scan, external service or hidden network destination is introduced.

Common hosting document roots are recognized in deterministic priority: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Only directories qualify. Plain FTP is explicitly reported as unencrypted; FTPS and SFTP are reported as secure transports. SFTP uses home-root context while FTP/FTPS use authenticated account-root context.

Diagnostics are advisory only. ByFTP does not call `openDirectory()` with a detected root and does not write diagnostic state into `ConnectionPresetStore`. The diagnostic model has no password, passphrase, username, fingerprint or network-operation field. JUnit and `audit_android.py` independently enforce those boundaries.

## Transfer behavior

`TransferStreams` wraps existing local `InputStream` and `OutputStream` objects and reports cumulative bytes only after successful reads or writes. Because progress is measured at this common stream boundary, FTP, explicit/implicit FTPS and SFTP do not need separate progress implementations.

When a document provider exposes a stable file size, the UI shows percentage progress. Otherwise ByFTP reports transferred bytes rather than inventing a percentage. **Stop after current file** is checked only after the active upload completes; the current FTP/FTPS/SFTP transaction is not torn down merely to simulate immediate cancellation.

## Shared-hosting FTP paths

FTP/FTPS records the server working directory immediately after login and treats that directory as Android UI root `/`. This keeps `public_html` and other account paths inside the authenticated login namespace instead of forcing an unrelated server filesystem root.

If the server cannot report `PWD`, ByFTP falls back to login-relative FTP paths. UI paths reject traversal (`..`), empty components, backslashes and NUL characters before a remote operation is issued. Raw server-reported login directories reject CR/LF/NUL before trimming or path normalization.

Detected web roots never bypass these mapping rules or silently replace the active directory.

## Security model

Plain FTP remains available only for compatibility and does not encrypt credentials or file contents. Prefer FTPS or SFTP whenever possible.

For FTPS, ByFTP uses platform trust and endpoint checking. It does not install a permissive `X509TrustManager`, trust-all helper or hostname-verification bypass.

For SFTP, the expected OpenSSH-style fingerprint such as `SHA256:AbCd...` must decode to exactly a 32-byte SHA-256 digest before SSHJ receives it. The connection fails closed if the fingerprint is absent, malformed or does not match the server key.

Raw host, port, username, password and fingerprint text is checked for CR/LF/NUL controls before trimming or canonicalization. FTP and SFTP directory entries share `RemotePaths.validateName`, so edge whitespace and protocol controls cannot be accepted by one transport and rejected later by another layer.

Transport objects do not retain the complete credential-bearing `ConnectionConfig` throughout an active session. Password references are cleared immediately after authentication and again on close. `MainActivity` clears its password field after connection attempts and during teardown.

SFTP password authentication remains supported. Private-key import remains deferred until Android Keystore-backed handling, validation and migration semantics have dedicated tests and audit coverage.

## Lifecycle and local files

Network work runs on a dedicated single-thread executor. Pending and active clients are tracked separately and closed during teardown; queued work is interrupted and stale UI callbacks are ignored.

Uploads/downloads use Android document-provider URIs. ByFTP does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write permissions. Multi-file upload uses `ACTION_OPEN_DOCUMENT` with `EXTRA_ALLOW_MULTIPLE`; downloads use `ACTION_CREATE_DOCUMENT` so the user explicitly controls the destination.

## APK artifacts

Root `VERSION` drives Android `versionName`, `versionCode` and public APK names:

- `ByFTP-<version>-Android-debug.apk` — installable debug-signed development/test APK.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized/minified release APK without a production signature.

The unsigned release APK is not a production distribution until signed with a stable private Android identity held outside the repository. `scripts/package_android.py` validates APK ZIP structure and required entries before staging.

## Toolchain

- Android Gradle Plugin 9.3.0
- Gradle 9.7.0
- JDK 17
- compileSdk / targetSdk 37
- minSdk 26 (Android 8.0)
- Apache Commons Net 3.13.0
- SSHJ 0.40.0

No AndroidX/Compose framework is required. The application continues to use platform UI APIs plus the reviewed network dependencies.

## Build and test

From the repository root:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Android lint warnings are errors. CI additionally runs `scripts/audit_android.py`, `scripts/audit_repository.py` through the release-integrity gate, JUnit connection/path/security/diagnostic tests, APK packaging validation and the general security/privacy/release audits.
