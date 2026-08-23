# Release verification

`VERSION` is the only canonical ByFTP product-version source. Desktop build scripts inject it into binary/package metadata; Android derives `versionName`, `versionCode` and versioned APK filenames from the same file.

## Public artifact contract

A 1.1.1+ release stages exactly these platform artifacts before shared metadata is added:

- Windows x64: Portable EXE, Setup EXE, verified ZIP.
- Windows x86: Portable EXE, Setup EXE, verified ZIP.
- Linux: amd64, arm64 and i386 DEB packages.
- macOS: Universal PKG.
- Android: debug-signed APK and optimized unsigned release APK.

The workflow then adds `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`. Publication is centralized through `scripts/publish_release.ps1`, which re-reads the GitHub Release and requires the final asset set, sizes and GitHub-provided SHA-256 digests to match the locally verified artifacts.

## Android verification

Android publication requires static security/privacy/lifecycle audits, JUnit tests, `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and `scripts/package_android.py` validation.

`ByFTP-<version>-Android-debug.apk` is debug-signed and intended for development/testing installs. `ByFTP-<version>-Android-release-unsigned.apk` is optimized but intentionally unsigned. Neither name implies a production/store signature. A production Android distribution requires a stable private signing identity held outside the repository.

## Check SHA-256

Verify downloads against `SHA256.txt` before redistribution:

- Windows: `Get-FileHash <file> -Algorithm SHA256`
- Linux: `sha256sum <file>`
- macOS: `shasum -a 256 <file>`

Publish or redistribute artifacts from the gated workflow rather than manually rebuilding files outside the verified release run.
