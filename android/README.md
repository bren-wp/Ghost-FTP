# Ghost FTP for Android

The Android project provides the native **Ghost FTP** client for Android devices.

## Build

The CI build uses Java 17 and Gradle 9.7.1:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon --stacktrace
```

The public GitHub Release publishes one installable APK named:

```text
Ghost-FTP-X.Y.Z-Android.apk
```

The public CI artifact is debug-signed for testing/sideloading. Production Play distribution requires a private release keystore and is intentionally outside the public repository.

## Branding and compatibility

The public app name is **Ghost FTP**. The existing Android application/package identifier can retain the legacy `com.ghostftp.client` value so upgrades and installed-app identity remain stable. It is an internal compatibility identifier, not public branding.

## Security

Use FTPS or SFTP where possible. SFTP host-key fingerprints should be verified before trusting a remote host. Never commit release signing keys or connection credentials to this repository.
