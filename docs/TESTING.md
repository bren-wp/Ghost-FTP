# Testing and quality gates

Tests, audits and native platform builds are release requirements for Ghost FTP, not optional evidence.

## Continuous integration

`.github/workflows/ci.yml` runs on pull requests and `main`. The current matrix contains six independent jobs:

- core/security/privacy/web;
- Android;
- Linux;
- Windows x64/x86;
- macOS Universal;
- iOS arm64 unsigned.

A failure in a platform-specific job blocks merge/release readiness even when the shared Go core passes.

## Core Go gates

The maintained CI toolchain is Go **1.27.1**. Core verification includes:

```bash
go telemetry off
gofmt
go test ./...
go test -race ./...
go vet ./...
```

Production scripts keep `GOTOOLCHAIN=local`, disable module downloads and require the dependency-free Go module graph. Go telemetry must be explicitly disabled before production builds.

## Repository and version audits

`scripts/audit_repository.py` validates all tracked paths/files for release hygiene. It rejects non-portable paths, case-insensitive collisions, tracked symlinks, generated/cache output, common OS/editor artifacts, malformed UTF-8 text, trailing whitespace and merge-conflict markers.

`scripts/audit_version.py` enforces root `VERSION` as the canonical production version source and verifies version binding for desktop, Linux, macOS, Android, iOS and web metadata. Ghost FTP starts at **1.0.0** and release tags use `ghostftp-vX.Y.Z`.

`scripts/audit_docs.py` validates local Markdown/HTML links and requires all maintained technical documents to be indexed from `docs/README.md` and the root README.

## Security and privacy gates

`scripts/audit_security.py` protects connection, credential, path, temporary-file, process-lifecycle and transfer invariants in the desktop core.

`scripts/audit_privacy.py` rejects application telemetry/vendor markers, forbidden runtime network libraries/fixed HTTP(S) destinations and insecure secret handling. The application policy remains user-selected server communication only.

Web security is additionally executed through `scripts/audit_web.py`, which validates:

- PHP 8.1+ availability and syntax for every PHP file;
- JavaScript/service-worker syntax with Node;
- all PHP regression tests under `GhostFTP WEB/tests/`;
- root/web/composer/PWA version consistency;
- Ghost FTP public manifest/cache branding;
- runtime storage exclusion from tracked/release files;
- strict session cookies, CSRF/session rotation and cross-site POST blocking;
- CSP/HSTS/noindex/security headers;
- remote path and host fail-closed validation;
- FTP/SFTP secret cleanup;
- SFTP SHA-256 host fingerprint verification;
- direct HTTP denial for the storage directory.

The `GhostFTP WEB` path is a legacy source-directory identifier only; the public application is Ghost FTP.

## Windows gate

`BUILD-WINDOWS.ps1` is the canonical Windows production build entry point. It runs brand/localization/version/docs/security/privacy/release audits and Python regression tests before compiling x64 and x86 binaries.

Windows validation covers:

- application-only installer payload integrity;
- rollback-aware installation transactions;
- path/reparse safety;
- PE architecture and GUI subsystem;
- ASLR/NX and architecture-appropriate PE mitigations;
- embedded icon, VERSIONINFO and application manifest;
- absence of telemetry/vendor signatures;
- Authenticode signing status reporting;
- no standalone uninstaller binary.

Legacy `GhostFTP.exe`/registry identifiers may remain only where required to upgrade existing installations. User-facing product metadata must identify Ghost FTP.

## Linux gate

Linux CI runs `bash linux/BUILD.sh` and requires non-empty amd64, arm64 and i386 Debian packages. Each package must report:

- package name `ghost-ftp`;
- canonical root `VERSION`;
- correct Debian architecture.

The packages use the shared desktop core and expose the `ghostftp` command plus the Ghost FTP desktop entry.

## macOS gate

macOS CI runs `bash macos/BUILD.sh`, building amd64 and arm64 desktop binaries and combining them into the Universal package. The resulting PKG must be non-empty and use canonical version metadata.

Production signing/notarization is a separate publisher-credential concern; an unsigned CI artifact must never be represented as notarized or signed.

## Android gate

Android CI uses JDK 17, Gradle **9.7.1** and Android Gradle Plugin **9.4.0**. It runs unit tests, lint and assembles an installable debug-signed APK:

```bash
gradle -p android :app:testDebugUnitTest :app:lintDebug :app:assembleDebug --no-daemon --stacktrace
```

The Android source/audit coverage protects connection validation, credential/control-character rejection, canonical remote paths/names, fingerprint validation, storage/backup policy, lifecycle cleanup and version binding.

A debug-signed CI APK is installable but is not represented as Play Store production-signed.

## iOS gate

iOS CI runs:

```bash
bash ios/BUILD.sh
```

The build compiles a real arm64 iPhoneOS application, derives marketing version from root `VERSION`, validates the bundle and packages an unsigned IPA. Tests/audits protect path handling, connection input, platform TLS behavior, session generation/lifecycle cleanup and archive structure.

An unsigned IPA requires a legitimate Apple signing identity and provisioning profile before normal device/TestFlight/App Store distribution.

## Web package gate

`scripts/package_web.py` creates:

```text
Ghost-FTP-X.Y.Z-Web.zip
```

The ZIP contains tracked production web files only. The packager verifies canonical VERSION, `brendigo/ghost-ftp-web` Composer metadata, the `ghostftp-static-vX.Y.Z` service-worker cache namespace and safe archive paths. Runtime `users.json`, configuration secrets and temporary/log files must not enter the package.

`scripts/test_package_web.py` supplies deterministic packaging regression coverage.

## Python tooling regressions

Repository hardening and packaging regressions are run with:

```bash
python -m unittest discover -s scripts -p 'test_*.py'
```

These tests protect maintenance/security assumptions such as raw input preservation, installer transaction behavior, platform lifecycle cleanup, package validation and Ghost FTP release/tag rules.

## Production release gate

`.github/workflows/release.yml` is the single production publication path. Publication waits for quality, Windows, Linux, macOS, Android and iOS jobs.

The release assembles exactly eight platform packages:

1. `Ghost-FTP-X.Y.Z-Setup-x64.exe`
2. `Ghost-FTP-X.Y.Z-Setup-x86.exe`
3. `Ghost-FTP-X.Y.Z-Setup-x32.exe` — byte-identical alias of x86
4. `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`
5. `Ghost-FTP-X.Y.Z-macOS-Universal.pkg`
6. `Ghost-FTP-X.Y.Z-Android.apk`
7. `Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa`
8. `Ghost-FTP-X.Y.Z-Web.zip`

It then adds `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, producing exactly **11 public release files**.

Before creating/updating a Release, the workflow verifies that `main` still points to the build commit. If a `ghostftp-vX.Y.Z` tag already exists on another commit, publication fails instead of rewriting history.
