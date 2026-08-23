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

`python scripts/audit_android.py` additionally checks mobile security/privacy/version invariants including SFTP fingerprint pinning, FTPS endpoint checking, Storage Access Framework use and absence of broad storage permissions/analytics dependencies.

The production release workflow repeats Android unit/lint/APK compilation as a required release gate. The generated debug APK is build evidence, not a production-signed public package.
