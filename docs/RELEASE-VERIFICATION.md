# Release verification

`VERSION` is the only canonical ByFTP product-version source. Desktop builds inject it into binary/package metadata; Android derives `versionName`/`versionCode`; iOS derives marketing/build metadata and archive names; ByFTP WEB mirrors it in its own VERSION/composer/PWA cache metadata.

## Repository source verification

Production verification starts with the tracked source tree itself. `scripts/audit_repository.py` enumerates every tracked Git path/file and rejects non-portable path collisions, Windows-reserved names, tracked symlinks, generated/cache output, invalid UTF-8/text hygiene, unresolved merge-conflict markers and stale explicit current-release markers.

The repository audit is invoked by `scripts/audit_release.py`, and `scripts/audit_version.py` verifies that the integration remains present. A platform package is therefore not accepted merely because it compiles; the exact source tree used to build it must satisfy repository-wide integrity, security, privacy, documentation and release contracts first.

## ByFTP 1.8.0 public artifact contract

A production release stages exactly **14 platform artifacts** before shared metadata:

- Windows x64: Portable EXE, Setup EXE, verified ZIP.
- Windows x86: Portable EXE, Setup EXE, verified ZIP.
- Linux: amd64, arm64 and i386 DEBs.
- macOS: Universal PKG.
- Android: debug-signed APK and optimized unsigned release APK.
- iOS: arm64 unsigned IPA and arm64 unsigned `.app` ZIP.

The workflow then adds `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for **17 public release files** total. Publication remains centralized through `scripts/publish_release.ps1`, which re-reads the GitHub Release and requires final asset identity, sizes and GitHub-provided SHA-256 digests to match local verified artifacts.

No public or internal release artifact named `Uninstall.exe` is part of the 1.8.0 contract.

## Windows app-only Setup verification

Windows production validation builds exactly four public PE files: Portable x64/x86 and Setup x64/x86. `BUILD-WINDOWS.ps1` fails if a generated file name contains `uninstall`.

Setup uses installer payload **schema 2**. `scripts/make_payload.py` writes exactly:

- `ByFTP.exe`
- `manifest.json`

`cmd/installer` rejects duplicate payload entries, unexpected files, legacy schema 1 and a legacy `Uninstall.exe` entry. The manifest must contain exactly one `ByFTP.exe` row whose size and SHA-256 match the embedded payload.

`scripts/verify_release.py` verifies Setup and Portable PE architecture, GUI subsystem, required PE mitigations/resources, telemetry-vendor absence and distinct SHA-256 identities, then emits `UNINSTALLER_BINARY=ABSENT`.

The release ZIP contains only the public Setup/Portable binaries plus release/documentation metadata. `BUILD-METADATA-WINDOWS.txt` and shared `BUILD-METADATA.txt` record `WINDOWS_UNINSTALLER=none`.

## Desktop verification

The central gate runs Go unit/integration tests, `go test -race ./...` and `go vet ./...`. Windows independently builds and verifies both x64 and x86 production packages. Linux independently builds amd64, arm64 and i386 DEBs. macOS independently builds and expands/verifies the Universal PKG. The reviewed native Go toolchain is 1.27.0.

Shared-hosting diagnostics remain covered by their Go and Windows regressions and the security audit: diagnostics must use the existing initial listing, contain no secret/network behavior and never override a saved/user-selected remote path.

## Android verification

Android 1.8.0 uses Android Gradle Plugin 9.3.2, Gradle 9.7.1, JDK 17, compileSdk/targetSdk 37 and build-tools 36.0.0.

Publication requires `scripts/audit_android.py`, JUnit, `lintDebug`, `lintRelease`, `assembleDebug`, `assembleRelease` and `scripts/package_android.py` validation. Lint warnings are errors.

`ByFTP-<version>-Android-debug.apk` is debug-signed for development/testing. `ByFTP-<version>-Android-release-unsigned.apk` is optimized but intentionally unsigned. Production distribution requires an external stable private signing identity.

## iOS verification

iOS publication requires `scripts/audit_ios.py`, Swift model/path/preset/diagnostic regressions, Xcode project/scheme parsing, a real generic arm64 `iphoneos` Release build, bundle identifier/version/architecture checks and `scripts/package_ios.py` validation.

`ByFTP-<version>-iOS-arm64-unsigned.ipa` must contain `Payload/ByFTP.app`, the expected `Info.plist` and a Mach-O `ByFTP` executable. The app bundle and archive must contain no symlinks/unsafe paths. `ByFTP-<version>-iOS-arm64-unsigned-app.zip` contains the same verified unsigned application bundle.

Neither iOS artifact implies Apple signing/provisioning. Production device/App Store/TestFlight distribution requires external Apple signing material.

## ByFTP WEB verification

`scripts/audit_web.py` runs PHP syntax/runtime regressions and JavaScript syntax checks. Release 1.8.0 additionally requires fail-closed app/user/rate-limit/profile/preference recovery, atomic login/registration rate limiting, SFTP host-key pinning and generation-safe password/authentication tests.

The WEB source tree is validated as part of the release matrix even though it has no compiled binary artifact. Its VERSION, Composer metadata and Service Worker cache namespace must match root `VERSION` exactly.

## Check SHA-256

Verify downloads against `SHA256.txt` before redistribution:

- Windows: `Get-FileHash <file> -Algorithm SHA256`
- Linux: `sha256sum <file>`
- macOS/iOS artifact files: `shasum -a 256 <file>`

Publish or redistribute artifacts from the gated release workflow rather than manually rebuilding files outside the verified run.
