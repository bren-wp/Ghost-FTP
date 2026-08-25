# Testing and quality gates

Tests, audits and platform builds are release requirements rather than optional evidence.

## Desktop gates

Core checks include `go test ./...`, the race detector, `go vet`, Windows x64/x86 production builds, Linux DEB builds, macOS Universal PKG builds, deterministic brand-asset verification, localization/version/documentation checks and security/privacy/release audits.

The 1.2.3 CI/release baseline uses Go 1.27.0. Linux packaging is tested directly through `linux/BUILD.sh`; macOS packaging is tested directly through `macos/BUILD.sh`. Obsolete platform wrappers under `scripts/` are rejected by the version/privacy/release audits rather than retained as duplicate entry points.

Installer changes require payload/transaction/rollback regressions. Localization changes must preserve English fallback and formatting-placeholder compatibility.

## Android gates

Android has an independent CI job with JDK 17, Gradle 9.7.0, Android Gradle Plugin 9.3.0, API 37 and Build Tools 36.0.0:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Lint treats warnings as errors. The mobile audit independently rejects permissive TLS/SSH patterns, verifies FTPS platform trust/endpoint checking, validates real 32-byte SFTP SHA-256 fingerprints, enforces strict canonical remote paths/names, checks login-root mapping, backup/storage policy, password persistence, picker cleanup and lifecycle behavior.

JUnit coverage includes connection validation, credential-control rejection, fingerprint validation, traversal/noncanonical path rejection, version binding and FTP shared-hosting mapping.

## iOS gates

iOS has a dedicated `macos-14` job. The central quality job first runs `python scripts/audit_ios.py`; the platform job then runs the canonical platform entry point:

```bash
bash ios/BUILD.sh
```

The iOS build performs four stages:

1. Compile/run dependency-free Swift model/path regressions with `xcrun swiftc`.
2. Parse/list the checked-in Xcode project/shared scheme.
3. Build a real generic arm64 `iphoneos` Release `.app` with repository-side code signing disabled and `MARKETING_VERSION` derived from root `VERSION`.
4. Validate/package the result through `scripts/package_ios.py` into a versioned unsigned IPA and unsigned app ZIP.

The app validation requires the expected bundle identifier/version, an executable arm64 Mach-O, a normal `Payload/ByFTP.app` IPA structure, no symlinks and no unsafe archive paths. Generated AppIcon PNG sizes are created from the canonical project icon for the build and removed afterwards instead of being duplicated in Git.

The iOS source audit additionally enforces platform TLS use, PASV-host redirect blocking, bounded I/O, strict path/credential controls, session-generation cleanup, pending-connect cleanup, connect-only transport-password lifetime, stale/failed temporary-download cleanup, no `UserDefaults` credential store, no WebView/analytics endpoint and no global ATS bypass.

## Linux packaging gates

Linux CI runs the shared Go tests/vet and then:

```bash
bash linux/BUILD.sh
```

The gate requires non-empty amd64, arm64 and i386 DEBs and verifies package name, canonical root `VERSION` and architecture metadata. `linux/byftp.desktop` and `linux/debian/control.in` are source-controlled packaging inputs.

## macOS packaging gates

macOS CI runs the shared Go tests/vet and then:

```bash
bash macos/BUILD.sh
```

The gate requires a non-empty Universal PKG and expands it with `pkgutil` to validate package structure. The application metadata and launcher come from `macos/Info.plist.in` and `macos/launcher.zsh` rather than inline generated source.

## Packaging regressions

- `scripts/test_package_android.py` tests Android APK staging/structure/path safety.
- `scripts/test_package_ios.py` tests iOS bundle validation, IPA/app ZIP structure, version mismatch rejection and symlink rejection.
- `scripts/test_release_notes.py` ensures release notes describe all mobile artifacts and signing limitations.
- `scripts/audit_version.py` enforces the reviewed Go/Gradle pins, canonical platform build entry points and root `VERSION` binding.
- `scripts/audit_release.py` requires Linux/macOS/iOS platform build directories and rejects obsolete build wrappers/source-sync workflow files.

All Python tool regressions are run with:

```bash
python -m unittest discover -s scripts -p 'test_*.py'
```

## Production release gate

The production workflow repeats central quality plus Windows, Linux, macOS, Android and iOS jobs. Publication cannot start until every job succeeds. The staging allowlist then requires all 14 platform artifacts before generating shared metadata/checksums and invoking the centralized release publisher.

Debug/unsigned mobile artifacts are never treated as production store-signed software.
