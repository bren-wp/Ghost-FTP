# Ghost FTP for Android

`android/` contains the native Android development client. It is post-0.0.3 source work targeted for a later public release and does **not** retroactively add Android to the already published Ghost FTP 0.0.3 release.

## Current Android scope

The Android client is a native Java application rather than a WebView wrapper. The first maintained source slice includes:

- Quick Connect for FTP, explicit FTPS and SFTP;
- remote directory listing and navigation;
- upload through Android's Storage Access Framework (SAF);
- download through SAF with best-effort deletion of a partially created destination document if the transfer fails;
- new-folder and delete operations;
- conservative symlink handling: Android preview does not open, download or delete a remote symlink;
- no persistent password/passphrase storage;
- no analytics, telemetry, advertising SDK, crash upload or Ghost FTP backend.

The UI uses Android platform widgets only. No AndroidX UI dependency is required.

## Transport security

### FTPS

FTPS is explicit TLS. The client sends `AUTH TLS`, upgrades the control channel with the Android/JVM default trust store, enables endpoint identification for the requested host, then requires `PBSZ 0` and `PROT P` before authenticated file operations. Certificate or hostname verification failure is fatal; there is no automatic downgrade to FTP.

### SFTP

SFTP uses `com.github.mwiede:jsch:2.28.7`, pinned in `app/build.gradle`. Host keys are stored only in the app-private `known_hosts` file. A previously unseen key requires an explicit fingerprint confirmation. A **changed pinned host key is always rejected** by the Ghost FTP trust callback; the connection cannot silently replace trust material.

The upstream JSch fork remains a separate third-party dependency under its own license. It is included only for SSH/SFTP transport and not for analytics or networking to any Ghost FTP service.

### Plain FTP

Plain FTP remains an explicit legacy compatibility choice. Android cleartext transport is enabled because raw FTP is intentionally supported, but selecting FTP displays a warning before credentials are sent. Ghost FTP never silently converts a failed FTPS connection into FTP.

Passive FTP data connections intentionally ignore the server-supplied PASV IP address and connect to the already established control peer plus the negotiated passive port. This prevents a PASV response from redirecting the client to an unrelated host.

## Files and Android storage

Ghost FTP does not request broad filesystem/storage permissions. Upload and download use the system document picker so the user explicitly grants access to the chosen source or destination document.

The app sets `android:allowBackup="false"`. Server credentials are not written to preferences, databases or files. SFTP host-key trust is app-private local state because it is required to detect key changes on later connections.

## Build

The development APK is built with Java 17, Android compile/target SDK 35 and minimum SDK 26.

From the repository root on an environment with Android SDK 35, Build Tools 35.0.0 and Gradle 8.9 available:

```bash
bash android/BUILD.sh
```

The build places the installable debug APK at:

```text
android/dist/Ghost-FTP-Android-debug.apk
```

and writes:

```text
android/dist/Ghost-FTP-Android-debug.apk.sha256
```

Generated APK/checksum files are intentionally ignored by Git. GitHub Actions uploads the same `android/dist/` output as the `ghostftp-android-apk` workflow artifact, preserving a real generated APK without treating a mutable development binary as source code.

The debug APK uses the build environment's debug signing identity and is for development/testing. A future public Android release must use a separately protected production signing key and must bind the signed package, version and checksum to the normal immutable Ghost FTP release lifecycle.

## Dependency boundary

Runtime dependency:

```text
com.github.mwiede:jsch:2.28.7
```

Build dependencies are the Android Gradle Plugin and Gradle itself. The Android workflow pins GitHub Actions to immutable commit SHAs and installs the exact Android SDK/build-tools versions required by this project.
