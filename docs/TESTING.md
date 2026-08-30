# Testing and quality gates

Tests, audits and platform builds are release requirements rather than optional evidence.

## Repository-wide integrity gate

Version 1.6.0 adds a repository-level gate before platform packaging. `scripts/audit_repository.py` enumerates the checkout with `git ls-files -s -z`, so **every tracked path and file** is inspected rather than only files explicitly named by a feature audit.

The gate rejects non-portable or unsafe tracked paths, case-insensitive collisions, Windows-reserved names, tracked symlinks, committed build/cache output and common OS/editor junk. Non-binary files must be strict UTF-8 without a BOM or unexpected NUL bytes, trailing whitespace, missing final newline or unresolved merge-conflict markers. Explicit `Current release:` / `Trenutačno izdanje:` markers must equal root `VERSION`.

`scripts/test_audit_repository.py` contains regression coverage for these rules. `scripts/audit_release.py` invokes the repository audit directly, and `scripts/audit_version.py` verifies that this integration cannot silently disappear.

## Desktop gates

Core checks include `go test ./...`, the race detector, `go vet`, Windows x64/x86 production builds, Linux DEB builds, macOS Universal PKG builds, deterministic brand-asset verification, localization/version/documentation checks and security/privacy/release audits.

The current CI/release baseline uses Go 1.27.0. Linux packaging is tested directly through `linux/BUILD.sh`; macOS packaging is tested directly through `macos/BUILD.sh`. Obsolete platform wrappers under `scripts/` are rejected by the version/privacy/release audits rather than retained as duplicate entry points.

Shared-hosting diagnostics have dedicated Go regressions for deterministic web-root priority, plain-versus-secure transport state, SFTP home/account context and invalid file/symlink candidates. The security audit requires diagnostics to derive from the existing initial connection listing exactly once, rejects secret/network behavior in the diagnostic model and blocks automatic remote-path changes. Windows CI adds `connection_diagnostics_windows_test.go` coverage for user-visible connection status.

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

JUnit coverage includes connection validation, credential-control rejection, fingerprint validation, traversal/noncanonical path rejection, version binding, FTP shared-hosting mapping, transfer byte accounting and `SharedHostingDiagnosticsTest`. The Android audit requires diagnostics to consume the already loaded initial listing, contain no secret/network behavior and never feed a detected web root into `openDirectory()` or persistent connection presets.

Version 1.6.0 additionally keeps Android source/resources under the repository-wide gate; the obsolete `connected_to` string found by Android lint during the 1.5.0 release-candidate pass was removed rather than suppressed.

## iOS gates

iOS has a dedicated macOS runner job. The central quality job first runs `python scripts/audit_ios.py`; the platform job then runs the canonical platform entry point:

```bash
bash ios/BUILD.sh
```

The iOS build performs four stages:

1. Compile/run dependency-free Swift model/path/preset/diagnostic regressions with `xcrun swiftc`.
2. Parse/list the checked-in Xcode project/shared scheme.
3. Build a real generic arm64 `iphoneos` Release `.app` with repository-side code signing disabled and `MARKETING_VERSION` derived from root `VERSION`.
4. Validate/package the result through `scripts/package_ios.py` into a versioned unsigned IPA and unsigned app ZIP.

The app validation requires the expected bundle identifier/version, an executable arm64 Mach-O, a normal `Payload/ByFTP.app` IPA structure, no symlinks and no unsafe archive paths. Generated AppIcon PNG sizes are created from the canonical project icon for the build and removed afterwards instead of being duplicated in Git.

The iOS source audit additionally enforces platform TLS use, PASV-host redirect blocking, bounded I/O, strict path/credential controls, session-generation cleanup, pending-connect cleanup, connect-only transport-password lifetime, stale/failed temporary-download cleanup, no `UserDefaults` credential store, no WebView/analytics endpoint and no global ATS bypass. Shared-hosting diagnostics must remain session-only, derive from the existing initial listing and never auto-navigate or persist derived state.

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

## Packaging and tooling regressions

- `scripts/test_audit_repository.py` tests repository path/text/version hygiene.
- `scripts/test_package_android.py` tests Android APK staging/structure/path safety.
- `scripts/test_package_ios.py` tests iOS bundle validation, IPA/app ZIP structure, version mismatch rejection and symlink rejection.
- `scripts/test_release_notes.py` ensures release notes describe all mobile artifacts and signing limitations.
- `scripts/audit_version.py` enforces the reviewed Go/Gradle pins, canonical platform build entry points, root `VERSION` binding and repository-audit integration.
- `scripts/audit_release.py` requires the repository-wide integrity scan, Linux/macOS/iOS platform build directories and centralized release contract.
- `scripts/audit_security.py` enforces the shared-hosting diagnostic non-secret/single-listing/auto-navigation boundaries alongside credential, path and transport checks.

All Python tool regressions are run with:

```bash
python -m unittest discover -s scripts -p 'test_*.py'
```

## Production release gate

The production workflow repeats central quality plus Windows, Linux, macOS, Android and iOS jobs. Publication cannot start until every job succeeds. The repository-wide integrity audit therefore runs before the publication contract accepts platform artifacts.

Public staging requires the exact 14 platform artifacts before `SHA256.txt`, release notes and build metadata are generated and the centralized publisher is invoked. Debug/unsigned mobile artifacts are never treated as production store-signed software.
