# Release verification

`VERSION` is the only canonical ByFTP product-version source. Desktop builds inject it into binary/package metadata; Android derives `versionName`/`versionCode`; iOS build/packaging derives marketing/build metadata and versioned archive names from the same file.

## Public artifact contract

A 1.2.0 release stages exactly **14 platform artifacts** before shared metadata:

- Windows x64: Portable EXE, Setup EXE, verified ZIP.
- Windows x86: Portable EXE, Setup EXE, verified ZIP.
- Linux: amd64, arm64 and i386 DEBs.
- macOS: Universal PKG.
- Android: debug-signed APK and optimized unsigned release APK.
- iOS: arm64 unsigned IPA and arm64 unsigned `.app` ZIP.

The workflow then adds `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **17 public release files** total. Publication remains centralized through `scripts/publish_release.ps1`, which re-reads the GitHub Release and requires final asset identity, sizes and GitHub-provided SHA-256 digests to match local verified artifacts.

## Android verification

Android publication requires source audits, JUnit, `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and `scripts/package_android.py` validation.

`ByFTP-<version>-Android-debug.apk` is debug-signed for development/testing. `ByFTP-<version>-Android-release-unsigned.apk` is optimized but intentionally unsigned. Production distribution requires an external stable private signing identity.

## iOS verification

iOS publication requires `scripts/audit_ios.py`, Swift model/path regressions, Xcode project/scheme parsing, a real generic arm64 `iphoneos` Release build, bundle identifier/version/architecture checks and `scripts/package_ios.py` validation.

`ByFTP-<version>-iOS-arm64-unsigned.ipa` must contain `Payload/ByFTP.app`, the expected `Info.plist` and a Mach-O `ByFTP` executable. The app bundle and archive must contain no symlinks/unsafe paths. `ByFTP-<version>-iOS-arm64-unsigned-app.zip` contains the same verified unsigned application bundle.

Neither iOS artifact implies Apple signing/provisioning. Production device/App Store/TestFlight distribution requires external Apple signing material.

## Check SHA-256

Verify downloads against `SHA256.txt` before redistribution:

- Windows: `Get-FileHash <file> -Algorithm SHA256`
- Linux: `sha256sum <file>`
- macOS/iOS artifact files: `shasum -a 256 <file>`

Publish or redistribute artifacts from the gated release workflow rather than manually rebuilding files outside the verified run.
