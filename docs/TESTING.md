# Testing and quality gates

The repository treats tests, audits and platform builds as release requirements rather than optional evidence.

## Desktop gates

Core checks include `go test ./...`, the Go race detector, `go vet`, Windows x64/x86 builds, Linux package builds, macOS Universal package builds, deterministic brand-asset verification, localization checks, version consistency, documentation links, security/privacy audits and release-pipeline validation.

Installer changes require regression coverage for payload validation, transaction/rollback behavior and settings persistence. Localization changes must preserve English fallback, compatible formatting placeholders and English-first startup/AskPass fallback text.

## Android gates

Android has an independent CI job so mobile failures cannot hide inside desktop tests. It provisions JDK 17 and Gradle 9.5.0, verifies Android API 37 and build-tools 36.0.0, then runs:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Android lint aborts on errors and treats warnings as errors. Release builds enable code minification and resource shrinking.

The `TrustAllX509TrustManager` lint detector is disabled because current third-party dependency JARs contain implementations that produce dependency-level findings even though ByFTP does not use permissive TLS trust. This is not a blanket exemption: `scripts/audit_android.py` scans ByFTP Android source and fails on custom `X509TrustManager`, empty trust callbacks, permissive hostname/SSH verifiers and missing platform FTPS trust/endpoint checks.

The Android audit also verifies SFTP fingerprint pinning, FTP login-root mapping, traversal rejection, cleartext-network policy, backup/device-transfer exclusions, Storage Access Framework use, absence of broad storage/analytics dependencies, root-version binding, picker-state cleanup and Activity lifecycle cleanup.

JUnit coverage includes connection validation, remote path normalization/traversal, version binding and FTP login-root/shared-hosting mapping.

## APK packaging tests

`scripts/test_package_android.py` verifies versioned APK staging and rejects malformed APKs, missing required entries, unsafe archive paths and invalid version values. `scripts/package_android.py` validates both build variants before CI or release artifacts are retained.

## Production release gate

The production workflow repeats the full Android build alongside Windows, Linux, macOS and central quality jobs. It publishes a debug-signed Android APK and an optimized unsigned release APK only after every required job succeeds. The unsigned release artifact is not treated as production-signed software.
