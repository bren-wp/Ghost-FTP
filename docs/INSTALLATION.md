# Installation

## Windows

Use the generated Setup executable for a normal per-user installation. Setup validates its embedded payload before changing the filesystem or registry, stages replacements, keeps rollback data during upgrades and creates Windows integration only after the payload passes integrity checks.

The first setup screen is the language selector. The chosen locale is used by setup and persisted as the initial ByFTP desktop language. Normal per-user installation does not require administrator rights.

## Linux

Use the architecture-specific DEB from the official release for amd64, arm64 or i386. Verify the package hash against `SHA256.txt` before redistribution.

## macOS

Use the gated Universal PKG from the official release. See [Signing](SIGNING.md) for the current signing/notarization status.

## Android

Android source lives under `android/` and supports Android 8.0/API 26 or newer. The project targets API 37.

For development builds install JDK 17, Gradle 9.5.0 and Android SDK platform 37, then run from the repository root:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon
```

The debug APK is generated under `android/app/build/outputs/apk/debug/` and is intended for development/CI verification. Do not redistribute it as a production-signed application.

A public production Android APK is intentionally withheld until a stable private Android signing identity is configured outside the repository. Signing keys, passwords and keystores must never be committed.

The Android app uses the Storage Access Framework for local files and does not need broad storage permission. Connection passwords are session-only in version 1.1.0.

## Source builds

Read the product version from root `VERSION`. Desktop production builds inject it into runtime binaries/package metadata; Android derives `versionName` and `versionCode` from the same file. Run the complete repository tests and audits before distributing any build.
