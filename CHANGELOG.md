# Changelog

## 1.6.0 — Repository-wide integrity and release hygiene

**Focus:** make every tracked repository file part of a fail-closed release-quality contract while preserving the verified transport, security, privacy and shared-hosting behavior from 1.5.0.

- Added `scripts/audit_repository.py`, which enumerates the repository with `git ls-files` and validates every tracked path/file rather than relying only on feature-specific allowlists.
- Added cross-platform path checks for case-insensitive collisions, Windows-reserved names, unsafe/control characters, overlong paths and tracked symlinks so a source tree cannot be valid only on one developer filesystem.
- Added repository hygiene checks that reject committed build/cache outputs such as `dist/`, Android build output, Gradle state, iOS build output, coverage and temporary directories.
- Added strict text-file checks for UTF-8, BOMs, unexpected NUL/binary content, trailing whitespace, missing final newlines and unresolved merge-conflict markers.
- Extended current-release drift detection across all tracked text files for explicit `Current release:` / `Trenutačno izdanje:` markers, using root `VERSION` as the only production release source.
- Added unit regressions for path collisions, generated/reserved paths, symlink modes, BOM/trailing-whitespace/final-newline handling and stale/current release markers.
- Fixed the Android 1.5.0 release-candidate lint failure by removing the obsolete `R.string.connected_to` resource after the diagnostics-aware connection summary replaced it; no lint baseline or suppression was introduced.
- Updated root release documentation and CI/release guidance so repository-wide integrity is a mandatory gate before packaging.
- Preserved 1.5.0 shared-hosting diagnostics, no-auto-navigation behavior, Android/iOS transfer progress, safe batch-stop behavior, SFTP host-key verification, FTPS platform trust, PASV redirect protections and credential-lifetime boundaries.
- Kept the canonical Windows x64/x86, Linux amd64/arm64/i386, macOS Universal, Android and iOS build/release matrix mandatory for 1.6.0.

## 1.5.0 — Shared-hosting connection diagnostics

**Focus:** make ByFTP explain the authenticated hosting environment immediately after connection without adding scanners, hidden probes, credential exposure or automatic path changes.

- Added a transport-neutral `ConnectionDiagnostics` model derived from the same initial remote listing the desktop engine already performs to validate a new connection; no additional server request is issued for diagnostics.
- Added deterministic common web-root detection in the order `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`, while ignoring desktop symlink candidates and never treating files as web roots.
- Diagnostics report only non-secret derived facts: secure versus plain FTP transport, account-root versus SFTP-home mode, detected web-root name and initial root-entry count. Host, username, passwords, passphrases, fingerprints, certificate material and server banners are excluded.
- Windows now surfaces secure/plain transport plus detected web-root/account-home status after a successful connection without changing the existing profile-selected or user-selected remote start path.
- Android derives the same diagnostics from its existing first `list("/")` result and shows them in the connected summary; it does not persist diagnostics or automatically open a detected web root.
- iOS derives diagnostics from its existing initial FTP/implicit-FTPS listing and shows a native SwiftUI **Shared hosting** section that explicitly states a detected path is not opened or saved automatically.
- Added Go, Android JUnit, iOS model and Windows-only regressions for web-root priority, secure/plain transport reporting, account/home semantics and rejection of invalid web-root candidates.
- Expanded Android, iOS and desktop security audits so diagnostics cannot silently gain secret fields, independent network activity, automatic web-root navigation or persistence behavior.
- Added regression coverage for common shared-hosting failure classes already mapped by ByFTP's safe user-error layer: FTP 425 data-channel failures, TLS/certificate failures and 552 quota errors; low-level tool/server details remain hidden.
- Preserved all 1.4.0 mobile transfer-progress and safe batch-stop behavior, Android SFTP host-key pinning, FTPS platform trust/endpoint checks, iOS PASV-host redirect blocking, canonical path validation and external production-signing boundaries.
- Kept the canonical Windows/Linux/macOS/Android/iOS build and release matrix mandatory for 1.5.0.

## 1.4.0 — Mobile transfer progress and safe batch control

**Focus:** make long Android and iOS transfers observable and controllable without weakening the existing FTP/FTPS/SFTP transport, trust, path or credential boundaries.

- Added byte-level upload/download progress to Android by wrapping the existing Storage Access Framework input/output streams in a transport-neutral `TransferStreams` helper.
- The Android progress layer works with the existing FTP, explicit FTPS, implicit FTPS and SFTP clients without adding protocol-specific progress code or changing TLS/host-key verification.
- Added Android progress UI with percentage reporting when the document provider exposes a stable size and transferred-byte fallback when it does not.
- Added Android **Stop after current file** for multi-file uploads. The active file is allowed to complete before the remaining batch is skipped, avoiding mid-command FTP/FTPS socket teardown and partial protocol-state ambiguity.
- Added JUnit regression coverage proving monitored input/output streams preserve payload bytes and report monotonically cumulative transfer counts.
- Added byte-progress callbacks to the existing iOS Network.framework data-socket file send/receive loops and threaded those callbacks through the native FTP actor.
- Added SwiftUI transfer progress to the iOS remote browser with percentage display for known sizes and transferred-byte fallback for unknown sizes.
- Added the same safe **Stop after file** batch behavior on iOS: the current upload completes before the remaining selected files are skipped.
- Preserved iOS protocol claims exactly: FTP and implicit FTPS only. No SFTP/explicit-FTPS shim or new network dependency was introduced.
- Preserved the 1.3.0 password-free mobile connection presets, background/disconnect cleanup, path traversal rejection, PASV-host redirect protection, Android SFTP host-key pinning and external production-signing boundaries.
- Kept the canonical Windows/Linux/macOS/Android/iOS build and release matrix mandatory for 1.4.0.

## 1.3.0 — Mobile file-manager parity and secret-lifetime hardening

**Focus:** turn the native Android and iOS clients into more practical everyday file managers while preserving ByFTP's fail-closed transport, path, privacy and release boundaries.

- Added Android and iOS local filtering/search for the current remote directory plus deterministic directory-first, case-insensitive sorting.
- Added direct **Go to path** navigation on both mobile platforms; user-supplied paths still pass through the existing canonical remote-path validators before any network operation.
- Added multi-file upload on Android through `ACTION_OPEN_DOCUMENT`/`EXTRA_ALLOW_MULTIPLE` and on iOS through the security-scoped multi-selection document importer.
- Batch uploads validate every remote name before transfer, reject duplicate target names in the same selection and refresh the listing once after the complete batch instead of after every file.
- Reworked Android's connected-state UI into a compact mobile file-manager surface with a connection summary, Up / Refresh / Menu actions, a filter field and 48dp minimum touch targets.
- Added a native iOS SwiftUI search surface, compact action menu, path navigation and an iOS-16-compatible empty/filter state without raising the deployment target.
- Added optional restoration of the last successful **non-secret** connection metadata: Android stores protocol/host/port/username/SFTP fingerprint in app-private preferences, while iOS stores protocol/host/port/username in Keychain with `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`.
- Passwords and passphrases remain excluded from mobile persistent presets. Android backup/device transfer remains disabled, and iOS does not use `UserDefaults` for connection metadata or credentials.
- Android now clears the password field after every connection attempt and on Activity teardown; iOS keeps its existing connect/background secret cleanup.
- Shortened mobile secret lifetime further by ensuring persistence callbacks receive pre-extracted non-secret preset data instead of retaining credential-bearing connection configurations just for convenience.
- Removed Android pre-validation trimming from create/rename remote-name flows so invalid edge whitespace is rejected by the canonical name validator instead of silently rewritten.
- Extracted Android remote-entry sort/filter behavior from `MainActivity` into a small testable model helper and added JUnit regression coverage.
- Added iOS model regressions proving serialized connection presets contain no password material and are revalidated before restoration.
- Expanded `audit_android.py` and `audit_ios.py` so CI now requires the new mobile navigation/search/batch-upload behavior and fails if password/passphrase persistence is introduced.
- Kept protocol claims exact: Android continues to support FTP, explicit FTPS, implicit FTPS and SFTP; iOS continues to support FTP and implicit FTPS only, with no permissive SFTP/explicit-FTPS compatibility shim.
- Preserved the canonical five-platform build/release matrix, external production-signing boundaries, no-telemetry policy and the 1.2.9 desktop path-persistence hardening.
