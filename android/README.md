# ByFTP for Android

ByFTP Android is a native Java client isolated from the Go desktop runtime so mobile lifecycle, permissions, networking, APK packaging and release signing boundaries remain explicit and independently testable.

## Current capabilities

- FTP, explicit FTPS, implicit FTPS and SFTP.
- Passive FTP/FTPS and binary transfers.
- FTPS platform certificate-chain validation, endpoint/hostname checking and protected data channels (`PROT P`).
- Mandatory OpenSSH-style `SHA256:` SFTP host-key pinning.
- Remote directory listing/navigation, refresh, create directory, rename and delete.
- Upload/download through Android's Storage Access Framework without broad storage permissions.
- Session-only passwords; no password or SSH-secret persistence.
- Cloud-backup and device-transfer exclusions for application data.
- Generic cleartext traffic disabled for Android platform-aware networking.
- No analytics, advertising SDK, telemetry backend or mandatory cloud account.

## Shared-hosting FTP paths

Starting with 1.1.1, FTP/FTPS records the server working directory immediately after login and treats that directory as the Android UI root `/`. This keeps `public_html` and other account paths inside the authenticated login namespace instead of forcing an unrelated server filesystem root.

If a server cannot report `PWD`, ByFTP falls back to login-relative FTP paths. UI paths are required to be canonical and reject traversal (`..`), empty components, backslashes and NUL characters before a remote operation is issued.

## Security model

Plain FTP remains available only for compatibility and does not encrypt credentials or file contents. Prefer FTPS or SFTP whenever possible.

For FTPS, ByFTP explicitly uses the platform trust manager and endpoint checking. It does not install a permissive `X509TrustManager`, trust-all helper or hostname-verification bypass.

For SFTP, supply the expected OpenSSH-style fingerprint such as `SHA256:AbCd...`. The connection fails closed if the fingerprint is absent or does not match the server key.

SFTP password authentication is supported. Private-key import remains intentionally deferred until Android Keystore-backed handling, import validation and migration semantics have dedicated tests and audit coverage.

## Lifecycle and local files

Network work runs on a dedicated single-thread executor rather than the UI thread. The Activity separately tracks pending and active clients; both are closed during destruction, queued work is interrupted and late UI callbacks are ignored.

Download-picker state is cleared immediately after every picker result and also on disconnect/destroy so a stale remote path cannot be reused after a cancelled or incomplete document-provider flow.

Local files are opened only through Android document-provider URIs. ByFTP does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write permissions.

## APK artifacts

The root `VERSION` drives Android `versionName`, `versionCode` and public APK names. CI and the release workflow build and validate both variants:

- `ByFTP-<version>-Android-debug.apk` — installable debug-signed development/test APK.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized, minified and resource-shrunk release APK without a production signature.

The unsigned release APK is **not** a production distribution until it is signed with a stable private Android identity held outside the repository. The project does not commit or fabricate a production keystore.

`scripts/package_android.py` validates APK ZIP structure and required entries before staging either artifact under a versioned name. Release checksums cover both APKs.

## Toolchain

- Android Gradle Plugin 9.3.0
- Gradle 9.5.0
- JDK 17
- compileSdk / targetSdk 37
- minSdk 26 (Android 8.0)
- Apache Commons Net 3.13.0
- SSHJ 0.40.0

## Build and test

From the repository root with JDK 17, Gradle 9.5.0 and Android SDK 37 installed:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Android lint warnings are treated as errors. The repository also runs `scripts/audit_android.py`, JUnit path/security/version regressions and the general security/privacy/release audits.
