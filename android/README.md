# ByFTP for Android

Native Android client source for ByFTP. The Android module is intentionally isolated from the Go desktop runtime so Android lifecycle, Storage Access Framework permissions and mobile networking remain explicit and testable.

## Current capabilities

- FTP, explicit FTPS, implicit FTPS and SFTP connections.
- Passive FTP/FTPS and binary transfers.
- FTPS endpoint/hostname certificate checking.
- Mandatory SHA-256 host-key fingerprint pinning for SFTP.
- Remote directory listing, navigation, refresh, create directory, rename and delete.
- Upload and download through Android's Storage Access Framework; no broad storage permission is requested.
- Session-only passwords: the Android app does not persist passwords or SSH secrets.
- No analytics, advertising SDK, telemetry backend or mandatory cloud account.

## Security model

Plain FTP is available only for compatibility and does not encrypt credentials or file content. Prefer FTPS or SFTP whenever possible.

For SFTP, paste the expected OpenSSH-style fingerprint, for example `SHA256:AbCd...`. Connections fail closed when the fingerprint is absent or does not match the server key.

The first Android implementation supports password authentication for SFTP. Private-key import is deliberately not included until Android keystore-backed key handling and migration semantics are implemented and audited.

## Toolchain

- Android Gradle Plugin 9.3.0
- Gradle 9.5.0
- JDK 17
- compileSdk / targetSdk 37
- minSdk 26 (Android 8.0)
- Apache Commons Net 3.13.0
- SSHJ 0.40.0

The application version is read from the repository root `VERSION`; do not hard-code a separate Android product version.

## Build

From the repository root with Gradle 9.5.0 and Android SDK 37 installed:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon
```

The CI job runs the same unit/lint/build gates. Debug APKs are CI evidence only. A public production APK must use a stable private Android signing identity; this repository does not commit or fabricate a production signing key.
