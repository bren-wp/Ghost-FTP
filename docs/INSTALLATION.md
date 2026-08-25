# Installation

## Verify the release first

Use only artifacts produced by the gated ByFTP release workflow. Compare every downloaded file against `SHA256.txt` before installation or redistribution.

## Windows

Use Setup for a normal per-user installation or Portable when installation is not required. Windows packages are produced for x64 and x86. Setup validates its embedded payload, stages replacements, keeps rollback data during upgrades and creates Windows integration only after integrity checks succeed.

## Linux

Install the official architecture-specific DEB: amd64, arm64 or i386. Verify the package hash before installation or redistribution.

The canonical Linux source/packaging surface is under `linux/`. For a source build on a supported Linux host:

```bash
go telemetry off
bash linux/BUILD.sh
```

This creates the three architecture-specific DEBs in `dist/`. `linux/byftp.desktop` and `linux/debian/control.in` are the source metadata copied/rendered into the packages. `scripts/BUILD-LINUX.sh` is retained only as a compatibility wrapper.

## macOS

Use the gated Universal PKG containing Intel x86_64 and Apple Silicon arm64 payloads. See [Signing](SIGNING.md) for Developer ID/notarization status.

The canonical macOS packaging surface is under `macos/`. For a source build on macOS:

```bash
go telemetry off
bash macos/BUILD.sh
```

The build uses `macos/Info.plist.in` and `macos/launcher.zsh`, creates the Universal app/runtime and emits `dist/ByFTP-<version>-macOS-Universal.pkg`. `scripts/BUILD-MACOS.sh` is retained only as a compatibility wrapper.

## Android

ByFTP supports Android 8.0/API 26 or newer and targets API 37. Official releases contain:

- `ByFTP-<version>-Android-debug.apk` — debug-signed and installable for development/testing.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized/minified release APK that intentionally has no production signature.

The unsigned release APK requires a stable private Android signing identity before production distribution. Never commit signing keys, passwords or keystores.

For source builds install JDK 17, Gradle 9.5.0 and Android SDK platform 37, then run:

```bash
gradle -p android :app:clean :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Android uses the Storage Access Framework, requests no broad storage permission, keeps passwords/SSH secrets session-only and excludes app data from backup/device-transfer flows.

## iOS

ByFTP includes a native SwiftUI application for iOS 16+ on arm64 iPhone/iPad devices. Official release evidence contains:

- `ByFTP-<version>-iOS-arm64-unsigned.ipa` — a real iPhoneOS device application packaged as `Payload/ByFTP.app`, without an Apple signature/provisioning profile.
- `ByFTP-<version>-iOS-arm64-unsigned-app.zip` — the same unsigned `ByFTP.app` bundle packaged for inspection or an external signing workflow.

These files are **not** App Store/TestFlight packages and are not normally installable on a stock device until signed with a valid Apple identity and provisioning configuration. The public repository intentionally contains no `.p12`, private key, provisioning profile or signing password.

For source builds use macOS with Xcode and run:

```bash
bash scripts/BUILD-IOS.sh
```

The script generates required AppIcon sizes from the canonical project icon, runs Swift model/path regressions, validates the shared Xcode scheme, builds a generic arm64 `iphoneos` Release application with repository-side signing disabled, validates bundle/version/architecture and creates both iOS release archives.

The iOS transport supports FTP and implicit FTPS. Explicit FTPS and SFTP remain intentionally unavailable on iOS until separately audited native implementations exist.

## Source builds

Root `VERSION` is the only current production version source. Desktop binaries/packages, Android `versionName`/`versionCode`, iOS marketing/build packaging, release notes and public artifact names are derived from it. Run the complete test/audit/build matrix before distributing a source build.
