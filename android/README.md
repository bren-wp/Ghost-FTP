# ByFTP for Android

Native Android client source for ByFTP. The Android module is intentionally isolated from the Go desktop runtime so mobile lifecycle, Storage Access Framework permissions and network behavior remain explicit, testable and independently buildable.

## Current capabilities

- FTP, explicit FTPS, implicit FTPS and SFTP connections.
- Passive FTP/FTPS and binary transfers.
- FTPS certificate-chain validation through the Android/JVM platform trust store plus endpoint/hostname checking.
- Mandatory OpenSSH-style `SHA256:` host-key fingerprint pinning for SFTP.
- Remote directory listing, navigation, refresh, create directory, rename and delete.
- Upload and download through Android's Storage Access Framework; no broad storage permission is requested.
- Session-only passwords: the Android app does not persist passwords or SSH secrets.
- App backup and device-transfer extraction rules exclude ByFTP application data.
- Generic cleartext traffic is disabled in the Android network-security configuration.
- No analytics, advertising SDK, telemetry backend or mandatory cloud account.

## Security model

Plain FTP is available only for compatibility and does not encrypt credentials or file content. Prefer FTPS or SFTP whenever possible.

For FTPS, ByFTP explicitly uses the platform trust manager and enables endpoint checking. It does not install a permissive `X509TrustManager`, custom trust-all helper or hostname-verification bypass. Private FTP data channels use `PROT P`.

For SFTP, paste the expected OpenSSH-style fingerprint, for example `SHA256:AbCd...`. Connections fail closed when the fingerprint is absent or does not match the server key.

The first Android implementation supports password authentication for SFTP. Private-key import is deliberately not included until Android Keystore-backed key handling and migration semantics are implemented and audited.

The Activity keeps network work off the UI thread, tracks a pending connection separately from the active session and closes both active/pending clients when the Activity is destroyed. Main-thread callbacks are ignored after destruction.

## Privacy and storage

Local upload/download files are selected through Android document providers. ByFTP does not request `MANAGE_EXTERNAL_STORAGE` or legacy broad read/write storage permissions.

Connection passwords and SSH secrets are not written to SharedPreferences, databases, files or a ByFTP backend. Cloud-backup and device-transfer rules explicitly exclude the application's root, file, database, shared-preference and external app-data domains.

## Toolchain

- Android Gradle Plugin 9.3.0
- Gradle 9.5.0
- JDK 17
- compileSdk / targetSdk 37
- minSdk 26 (Android 8.0)
- Apache Commons Net 3.13.0
- SSHJ 0.40.0

The application version is read from the repository root `VERSION`; do not hard-code a separate Android product version.

Release builds enable code minification and resource shrinking. CI treats Android lint warnings as errors and separately audits security/privacy/version/lifecycle invariants.

## Build

From the repository root with Gradle 9.5.0, JDK 17 and Android SDK 37 installed:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon
```

The CI job runs the same unit/lint/build gates and stores lint/APK evidence. Debug APKs are CI/development evidence only. A public production APK must use a stable private Android signing identity; this repository does not commit or fabricate a production signing key.
