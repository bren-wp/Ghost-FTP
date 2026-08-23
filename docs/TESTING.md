# Testing and quality gates

The repository treats tests and audits as release requirements.

## Desktop gates

Core checks include `go test ./...`, the Go race detector, `go vet`, Windows x64/x86 builds, Linux package builds, macOS Universal package builds, deterministic brand-asset verification, localization checks, version consistency, documentation links, security/privacy audits and release-pipeline validation.

Installer changes require regression coverage for payload validation, transaction/rollback behavior and settings persistence. Localization changes must preserve English fallback and compatible formatting placeholders.

## Android gates

Android has an independent CI job so mobile build failures cannot hide inside desktop tests. The gate provisions JDK 17 and Gradle 9.5.0, verifies a preinstalled Android API 37 platform and AGP 9.3's supported/default build-tools 36.0.0, then runs:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon --stacktrace
```

Android lint aborts on errors and treats warnings as errors. Release builds enable both code minification and resource shrinking.

The `TrustAllX509TrustManager` lint detector is disabled because current third-party dependency JARs contain implementations that produce dependency-level findings even though ByFTP does not use permissive TLS trust. This is not a blanket security exemption: `python scripts/audit_android.py` scans ByFTP Android source and fails on custom `X509TrustManager`/empty trust callbacks/permissive hostname or SSH verifiers, while also requiring explicit platform FTPS trust and endpoint checking.

`scripts/audit_android.py` also verifies SFTP fingerprint pinning, cleartext-network policy, backup/device-transfer exclusions, Storage Access Framework use, absence of broad storage permissions/analytics dependencies, root-version binding and Activity lifecycle cleanup for active/pending remote clients.

The production release workflow repeats Android unit/lint/APK compilation as a required release gate. CI stores lint reports and the generated debug APK as build evidence. The debug APK is not a production-signed public package.
