# Installation

## Verify the release first

Use only artifacts produced by the gated ByFTP release workflow. Compare the downloaded file against `SHA256.txt` before redistribution or installation.

## Windows

Use the generated Setup executable for a normal per-user installation or Portable when installation is not required. Setup validates its embedded payload before changing the filesystem or registry, stages replacements, keeps rollback data during upgrades and creates Windows integration only after payload integrity checks succeed.

Windows packages are produced for x64 and x86. The first setup screen selects the installer/application language; normal per-user installation does not require administrator rights.

## Linux

Install the architecture-specific DEB from the official release: amd64, arm64 or i386. Verify the package hash before installation or redistribution.

## macOS

Use the gated Universal PKG from the official release. It contains Intel x86_64 and Apple Silicon arm64 payloads. See [Signing](SIGNING.md) for signing/notarization status.

## Android

ByFTP supports Android 8.0/API 26 or newer and targets API 37. Starting with 1.1.1 the official GitHub Release contains two Android APK artifacts:

- `ByFTP-<version>-Android-debug.apk` — debug-signed and installable for development/testing.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized/minified release APK that is intentionally unsigned.

The debug APK is not a production-signed package. The unsigned release APK must be signed with a stable private Android identity before production distribution. Signing keys, passwords and keystores must never be committed to this repository.

For source builds install JDK 17, Gradle 9.5.0 and Android SDK platform 37, then run from the repository root:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

The Android app uses the Storage Access Framework for local files and requests no broad storage permission. Passwords/SSH secrets are session-only, app data is excluded from backup/device-transfer flows, FTPS uses platform certificate trust plus endpoint verification and FTP/FTPS paths remain inside the authenticated login/account root.

## Source builds

Root `VERSION` is the only current production version source. Desktop builds inject it into binary/package metadata; Android derives `versionName`, `versionCode` and versioned APK artifact names from the same file. Run the complete repository tests and audits before distributing any source build.
