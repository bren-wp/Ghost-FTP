# Testing and quality gates

Tests, audits and platform builds are release requirements rather than optional evidence.

## Repository-wide integrity gate

The repository-level gate runs before platform packaging. `scripts/audit_repository.py` enumerates the checkout with `git ls-files -s -z`, so **every tracked path and file** is inspected rather than only files explicitly named by a feature audit.

The gate rejects non-portable or unsafe tracked paths, case-insensitive collisions, Windows-reserved names, tracked symlinks, committed build/cache output and common OS/editor junk. Non-binary files must be strict UTF-8 without a BOM or unexpected NUL bytes, trailing whitespace, missing final newline or unresolved merge-conflict markers. Explicit `Current release:` / `Trenutačno izdanje:` markers must equal root `VERSION`.

`scripts/test_audit_repository.py` contains regression coverage for these rules. `scripts/audit_release.py` invokes the repository audit directly, and `scripts/audit_version.py` verifies that this integration cannot silently disappear.

## Desktop gates

Core checks include `go test ./...`, the race detector, `go vet`, Windows x64/x86 production builds, Linux DEB builds, macOS Universal PKG builds, deterministic brand-asset verification, localization/version/documentation checks and security/privacy/release audits.

The current 1.9.0 CI/release baseline uses **Go 1.27.1**. Linux packaging is tested directly through `linux/BUILD.sh`; macOS packaging is tested directly through `macos/BUILD.sh`. Obsolete platform wrappers under `scripts/` are rejected rather than retained as duplicate entry points.

Shared-hosting diagnostics have dedicated regressions for deterministic web-root priority, plain-versus-secure transport state, SFTP home/account context and invalid file/symlink candidates. The security audit requires diagnostics to derive from the existing initial connection listing exactly once, rejects secret/network behavior in the diagnostic model and blocks automatic remote-path changes.

Installer changes require payload/transaction/rollback regressions. The Windows installer contract is application-only: `payload.zip` contains `ByFTP.exe` plus the verified manifest and no generated `Uninstall.exe`. Build, Windows bundle and public staging audits fail if an uninstaller binary, `cmd/uninstaller`, legacy `--uninstaller` payload input or old payload schema returns.

Localization changes must preserve English fallback and formatting-placeholder compatibility.

## Android gates

Android has an independent CI job with **JDK 17, Gradle 9.7.1, Android Gradle Plugin 9.4.0, API 37 and Build Tools 36.0.0**:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:lintRelease :app:assembleDebug :app:assembleRelease --no-daemon --stacktrace
python scripts/package_android.py \
  --debug android/app/build/outputs/apk/debug/app-debug.apk \
  --release android/app/build/outputs/apk/release/app-release-unsigned.apk \
  --output-dir dist
```

Lint treats warnings as errors. The mobile audit independently rejects permissive TLS/SSH patterns, verifies FTPS platform trust/endpoint checking, validates real 32-byte SFTP SHA-256 fingerprints, enforces strict canonical remote paths/names, checks login-root mapping, backup/storage policy, password persistence, picker cleanup and lifecycle behavior.

JUnit coverage includes connection validation, credential-control rejection, fingerprint validation, traversal/noncanonical path rejection, version binding, FTP shared-hosting mapping, transfer byte accounting and shared-hosting diagnostics. `scripts/audit_version.py` also verifies the exact AGP 9.4.0 pin.

## iOS gates

iOS has a dedicated macOS runner job. The central quality job first runs `python scripts/audit_ios.py`; the platform job then runs:

```bash
bash ios/BUILD.sh
```

The iOS build:

1. compiles/runs dependency-free Swift model/path/preset/diagnostic regressions;
2. parses the checked-in Xcode project/shared scheme;
3. builds a real generic arm64 `iphoneos` Release `.app` with repository-side signing disabled and `MARKETING_VERSION` derived from root `VERSION`;
4. validates/packages the result through `scripts/package_ios.py` into a versioned unsigned IPA and unsigned app ZIP.

Validation requires the expected bundle identifier/version, an executable arm64 Mach-O, normal `Payload/ByFTP.app` IPA structure, no symlinks and no unsafe archive paths. The iOS source audit additionally enforces platform TLS use, PASV-host redirect blocking, bounded I/O, strict path/credential controls, session-generation cleanup, pending-connect cleanup and stale temporary-download cleanup.

## WEB gates

The shared-hosting WEB application is checked through `scripts/audit_web.py`. Runtime PHP regressions cover authentication rate limiting, user-registry durability, encrypted profile/config recovery, remote-name validation and fail-closed storage/security invariants.

Release 1.9.0 also permanently tests the staged extraction contract:

- complete archive/topology/existing-remote validation precedes remote mutation;
- every ZIP file is locally materialized before the first remote write;
- the cumulative 512 MiB budget is enforced on actual decompressed bytes;
- staged temp files are cleaned through `finally`;
- WEB diagnostics remain administrator-only.

`scripts/package_web.py` then creates the deployable `ByFTP-<version>-WEB-shared-hosting.zip` from tracked production files only. `scripts/test_package_web.py` verifies exact tracked-source membership, safe archive paths, VERSION/PWA metadata and absence of runtime `users.json`/`config.json` state.

## Linux and macOS packaging gates

Linux CI runs shared Go tests/vet and `bash linux/BUILD.sh`, then requires non-empty amd64, arm64 and i386 DEBs with correct package/version/architecture metadata.

macOS CI runs shared Go tests/vet and `bash macos/BUILD.sh`, then requires a non-empty Universal PKG and expands it with `pkgutil` to validate package structure.

## Packaging and tooling regressions

- `scripts/test_audit_repository.py` tests repository path/text/version hygiene.
- `scripts/test_package_android.py` tests Android APK staging/structure/path safety.
- `scripts/test_package_ios.py` tests iOS bundle validation, IPA/app ZIP structure, version mismatch rejection and symlink rejection.
- `scripts/test_package_web.py` tests deterministic WEB packaging and runtime-state exclusion.
- `scripts/test_stability_hardening.py` protects staged WEB extraction, admin-only diagnostics and dead-code cleanup invariants.
- `scripts/test_release_notes.py` verifies release-note artifact/signing descriptions.
- `scripts/audit_version.py` enforces Go 1.27.1, Gradle 9.7.1, AGP 9.4.0, canonical platform build entry points and root `VERSION` binding.
- `scripts/audit_release.py` requires repository/WEB integrity, app-only Windows Setup, dedicated WEB packaging and the centralized 18-file public release contract.

All Python tool regressions are run with:

```bash
python -m unittest discover -s scripts -p 'test_*.py'
```

## Production release gate

The production workflow repeats central quality plus Windows, Linux, macOS, Android and iOS jobs. The quality job additionally produces the verified WEB release package. Publication cannot start until every prerequisite job succeeds.

Before publication, `scripts/prepare_release.ps1` requires exactly **15 platform artifacts**: Windows 6, Linux 3, macOS 1, Android 2, iOS 2 and WEB 1. It then generates `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, producing exactly **18 public release files**.

Debug/unsigned mobile artifacts are never treated as production store-signed software. Windows release metadata explicitly records `WINDOWS_UNINSTALLER=none`.
