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

## 1.2.9 — Verbatim profile path persistence

**Focus:** stop the saved-profile backend from silently rewriting valid remote, local and private-key paths before validation or persistence.

- `Profiles.Save` now preserves `RemotePath`, `LocalPath` and `PrivateKeyPath` exactly as supplied by the caller instead of applying `TrimSpace` first.
- Valid leading/trailing spaces in Unix/local filesystem paths and server-side remote path components are no longer lost at the persistence boundary.
- Existing CR/LF/NUL, UTF-8, length and remote-traversal validation remains fail-closed; removing normalization does not weaken path validation.
- Empty remote paths still receive the existing protocol-aware default (`.` for SFTP and `/` for FTP/FTPS), while non-empty paths remain verbatim.
- Added regressions that verify all three persisted path fields remain unchanged and that control-character input is still rejected.
- Preserved Windows UI normalization where it is an explicit UX choice; the backend/API no longer depends on a caller performing that normalization.
- Preserved the verified five-platform build, release, credential-binding and endpoint-identity contract from 1.2.8.

## 1.2.8 — Fail-closed profile persistence input

**Focus:** make saved-profile persistence enforce the same raw connection and SFTP trust boundaries as the runtime/UI instead of silently normalizing caller input before validation.

- `Profiles.Save` now validates and stores `Protocol`, `Host` and `Username` exactly as provided; uppercase/edge-whitespace protocols and noncanonical raw hosts are rejected rather than rewritten.
- Backend-compatible usernames are preserved verbatim, while CR/LF/NUL controls are rejected before they can disappear through normalization.
- Saved SFTP fingerprints are validated in their original canonical OpenSSH `SHA256:` form; edge spaces, tabs and line controls are no longer trimmed into valid pins.
- `Profiles.UpdateFingerprint` now applies the same fail-closed raw fingerprint rule instead of trimming before validation.
- Added persistence regressions for raw protocol, host, username and fingerprint input plus canonical success cases.
- Extended the security audit so future pre-validation normalization in either profile-save or direct fingerprint-update paths fails CI.
- Preserved intentional normalization for profile display names/IDs and left local/remote/private-key path semantics unchanged for a separate audit cycle.
- Preserved the verified five-platform build, signing, release, endpoint-identity and credential-binding contract from 1.2.7.

## 1.2.7 — Raw desktop account and port validation

**Focus:** finish fail-closed Windows connection-field handling by ensuring username and port text reach validation without silent normalization while centralizing duplicated quick-connect/profile-save parsing.

- Added one cross-platform-testable desktop raw connection-input validator shared by Windows quick-connect and profile-save.
- Windows no longer trims username before validation; CR/LF/NUL cannot disappear before security checks, while backend-compatible usernames are preserved verbatim.
- Windows now parses port text without `TrimSpace`, so edge whitespace, tabs and CR/LF are rejected instead of silently normalized into a valid port.
- Preserved the 1.2.6 fail-closed raw-host boundary and canonical endpoint/connection-identity behavior.
- Added regressions for canonical ports, noncanonical raw port text, username controls, verbatim usernames and raw-host rejection.
- Extended the security audit so future Windows pre-validation username/port trimming fails CI.
- Removed duplicated host/username/port validation logic from the two Windows connection-entry paths without changing their existing user-facing error behavior.
- Preserved the verified five-platform build, signing, release and credential-boundary contract from 1.2.6.

## 1.2.6 — Protocol-state cleanup and raw host validation

**Focus:** keep protocol-specific runtime state minimal and restore the intended fail-closed raw-host boundary in the Windows desktop UI.

- Added one `sanitizeProtocolState` boundary in the desktop remote manager so FTP, explicit FTPS and implicit FTPS cannot retain SFTP-only private-key paths, passphrases or host-key fingerprints.
- Applied the same sanitizer before transfer connection-identity hashing, preventing dead/stale SFTP fingerprint state from making an otherwise identical FTP/FTPS reconnect look like a different connection.
- Kept SFTP private-key, passphrase and host-key state unchanged for real SFTP sessions.
- Restored raw Windows host validation in both quick-connect and profile-save paths by passing the untrimmed host text to `ValidateConnection`; leading/trailing host whitespace can no longer disappear before the fail-closed `ValidateHost` check.
- Kept trimming for profile display names, ports and other non-host UX fields where normalization is intentional.
- Extended the security audit to block both protocol-state leakage and future reintroduction of pre-validation Windows host trimming.
- Added regressions covering FTP/FTPS protocol-state stripping, connection-identity stability and preservation of SFTP state.
- Preserved the verified five-platform build, signing, release and credential-boundary contract from 1.2.5.

## 1.2.5 — Canonical endpoint identity and profile input hardening

**Focus:** remove the remaining endpoint-identity drift in the desktop runtime and make SFTP profile/direct-connect trust input fail closed on canonical SHA-256 and UTF-8 boundaries.

- Added one canonical endpoint key in `profilebinding` and reused it for profile matching and transfer connection identity, so DNS trailing-dot, protocol case/whitespace and bracketed/raw IPv6 forms no longer become false "different connection" identities after reconnect.
- Kept username, port and SFTP host-key fingerprint as strict connection-identity boundaries so retry isolation is not weakened while equivalent endpoint spelling is normalized consistently.
- Centralized platform-correct private-key path identity for reuse by lifecycle/security code: Windows remains case-insensitive while Linux/macOS stay exact and fail closed.
- Replaced the weak desktop SFTP fingerprint shape check with a canonical OpenSSH `SHA256:` validator that Base64-decodes the value and requires exactly one 32-byte SHA-256 digest.
- Applied the same fingerprint validator to saved profiles, fingerprint updates and direct SFTP connection input instead of maintaining separate rules.
- Added explicit UTF-8 rejection for profile names, local paths and private-key paths so invalid bytes cannot be silently rewritten by JSON serialization.
- Replaced synthetic noncanonical fingerprint fixtures with real canonical digest fixtures and added regressions for malformed fingerprints, invalid UTF-8 and endpoint identity equivalence.
- Preserved the verified Windows/Linux/macOS/Android/iOS build, signing and release contract from 1.2.4.

## 1.2.4 — Cross-platform credential binding and runtime consistency

**Focus:** keep the verified five-platform release contract intact while removing runtime-policy drift, fixing cross-platform SSH key identity semantics and reducing redundant queue state transitions.

- Fixed desktop SFTP private-key passphrase binding so Windows keeps case-insensitive key-path identity while Linux/macOS use a strict fail-closed path comparison; a case-only path change on a case-sensitive filesystem can no longer inherit the saved passphrase of a different key.
- Added platform-specific regressions for Windows and non-Windows private-key identity behavior while retaining exact endpoint and username credential boundaries.
- Centralized transfer parallelism, retry and connection-timeout limits/defaults in the settings store instead of duplicating magic values across persistence and runtime layers.
- Added one nil-safe `Effective` settings path so connection setup, transfer scheduling and transfer workers use the same conservative defaults if settings storage is temporarily unavailable.
- Made transfer Pause/Resume transitions idempotent so repeated UI actions no longer emit redundant state snapshots or wake scheduling unnecessarily.
- Removed a dead repeated colon condition from raw IPv6 host validation and added canonical IPv6 plus malformed host/port regression coverage.
- Added settings recovery/migration tests for unavailable storage and legacy out-of-range state.
- Preserved FTP/FTPS/SFTP staging, rollback, path-validation, host-key verification, secret-protection and five-platform packaging behavior from 1.2.3.

## 1.2.3 — Toolchain refresh and repository hygiene

**Focus:** keep the verified five-platform runtime behavior intact while updating build toolchains, removing obsolete automation and making each platform directory the canonical build surface.

- Updated desktop CI and production builds from Go 1.26.5 to Go 1.27.0 and raised the module baseline to `go 1.27.0`.
- Updated Android CI and production builds from Gradle 9.5.0 to stable Gradle 9.7.0 while retaining AGP 9.3.0, API 37, Build Tools 36.0.0 and Java 17.
- Added version-audit checks so CI fails if the reviewed Go/Gradle pins or canonical platform build entry points drift.
- Made `ios/BUILD.sh` the canonical iOS build entry point alongside `linux/BUILD.sh` and `macos/BUILD.sh`.
- Updated CI and production release workflows to invoke Linux, macOS and iOS builds directly from their platform directories.
- Removed obsolete `scripts/BUILD-LINUX.sh`, `scripts/BUILD-MACOS.sh` and `scripts/BUILD-IOS.sh` compatibility wrappers.
- Removed the retired `.github/workflows/__byftp_sync.yml` workflow that reconstructed the historical ByFTP 1.0.12 source tree from an obsolete sync branch.
- Kept `scripts/` only for shared audits, local desktop build support, APK/IPA packaging, Windows bundle verification, release-note generation and the fail-closed central release publisher.
- Preserved the exact Windows/Linux/macOS/Android/iOS public artifact contract and existing production-signing boundaries.

## 1.2.2 — Cross-platform lifecycle, path and credential cleanup

**Focus:** harden the already verified five-platform architecture without weakening compatibility, reduce duplicated validation/build logic and generate a new complete desktop/Android/iOS release from one canonical version.

- Desktop host validation now rejects leading/trailing whitespace and protocol control characters on raw direct-connect input instead of trimming before validation.
- Added desktop regressions that keep noncanonical raw hosts fail-closed while preserving canonical DNS, IPv4 and IPv6 forms.
- Moved canonical Linux application packaging into `linux/`: `linux/BUILD.sh`, `linux/byftp.desktop`, `linux/debian/control.in` and the Linux build guide now own DEB packaging for amd64, arm64 and i386.
- Moved canonical macOS application packaging into `macos/`: `macos/BUILD.sh`, `macos/Info.plist.in`, `macos/launcher.zsh` and the macOS build guide now own the Universal application/PKG packaging surface.
- Kept the shared Go desktop protocol/transfer/security core single-sourced under `cmd/` and `internal/`; Linux/macOS platform folders do not duplicate that runtime.
- Reduced legacy `scripts/BUILD-LINUX.sh` and `scripts/BUILD-MACOS.sh` to version-guarded compatibility delegates and added release/privacy/version audits that reject duplicated platform build logic in those wrappers.
- Android remote names now reject CR/LF/NUL plus leading/trailing whitespace consistently.
- Android FTP and SFTP directory listings now share `RemotePaths.validateName` instead of maintaining weaker duplicated name filters.
- Android FTP login-root parsing rejects CR/LF/NUL before trimming server-provided `PWD` data.
- Android FTP and SFTP clients no longer retain the complete `ConnectionConfig` for the session; password references are cleared in `finally` immediately after authentication and again during close.
- Added Android regressions for canonical remote names and server login-root control characters, and expanded the Android audit to enforce the shared validator and connect-only transport password lifetime.
- iOS remote names now fail closed on leading/trailing whitespace and CR/LF/NUL instead of silently trimming user input.
- iOS FTP login-root parsing rejects protocol controls before normalizing server-provided working directories.
- iOS now tracks a client while it is still connecting so disconnect/background cleanup can close both established and pending connections immediately.
- iOS clears the password field even when local connection validation fails.
- iOS download operations now remove their private temporary directory after transfer failure or a stale async result instead of leaving abandoned staging folders.
- Expanded iOS model tests and static audits for canonical names, login-root controls, pending-connection cleanup and temporary-download cleanup.
- Advanced the canonical release version to 1.2.2 while preserving immutable 1.2.1 release assets and the exact Windows/Linux/macOS/Android/iOS packaging contract.

## 1.2.1 — Mobile raw-input hardening and five-platform maintenance

**Focus:** preserve the verified 1.2.0 five-platform release model while tightening Android/iOS endpoint validation before normalization and keeping every desktop/mobile build gate mandatory.

- Android now rejects CR/LF/NUL control characters in raw host, port, username, password and SFTP fingerprint input before trimming or protocol-library handoff.
- Android SFTP fingerprint canonicalization still requires a real 32-byte SHA-256 digest and now cannot silently trim trailing protocol-control characters.
- Added Android regressions for trailing host/port/username control characters and fingerprint control-character rejection.
- iOS now validates raw host, port, username and password fields before whitespace normalization so edge CR/LF values cannot disappear during trimming.
- Added iOS model regressions for trailing CRLF host, port and username input while retaining existing traversal, login-root and credential-injection tests.
- Preserved the native application layout: Android remains under `android/`; the SwiftUI/Xcode iOS application remains under `ios/`.
- Preserved the public five-platform artifact contract, including Android APKs and real unsigned arm64 iPhoneOS IPA/app ZIP artifacts generated from the canonical `VERSION`.
- Updated README and release metadata to 1.2.1 without modifying the immutable `v1.2.0` release.

## 1.2.0 — Native iOS client and five-platform release matrix

**Focus:** add a real native iOS application and unsigned IPA/app-bundle release artifacts while tightening Android path/fingerprint validation and preserving the existing Windows, Linux and macOS stability gates.

- Added a native SwiftUI iPhone/iPad application under `ios/` with a normal Xcode project and shared scheme; no WebView wrapper or hidden ByFTP backend is used.
- Added iOS FTP and implicit FTPS connections using Apple Network.framework, including binary transfers, protected FTPS data channels, EPSV/PASV passive mode and bounded network reads.
- Added iOS remote browse, upload, download, create-directory, rename and delete workflows with the system document picker and share/save sheet.
- Added shared-hosting FTP login-root mapping on iOS so UI `/` represents the authenticated FTP account root and `public_html` stays inside that account root.
- iOS rejects traversal, duplicate separators, backslashes, NUL/control characters and unsafe server-reported login roots instead of silently rewriting paths.
- iOS ignores server-supplied PASV host redirects and uses only the passive port with the endpoint selected by the user.
- iOS credentials remain session-scoped: UI password state is cleared after connect attempts, the transport actor clears its password copy after authentication, and active sessions disconnect when the app enters the background.
- Added dependency-free iOS Swift model/path regressions plus a real generic arm64 `iphoneos` Xcode Release build gate.
- Added `scripts/BUILD-IOS.sh` to generate AppIcon sizes from the canonical project icon, bind Xcode marketing/build versions to `VERSION`, build with repository-side code signing disabled and verify the resulting arm64 application bundle.
- Added `scripts/package_ios.py` and regressions to validate `Info.plist`, bundle identifier, version, Mach-O executable, symlink safety and archive paths before creating `ByFTP-<version>-iOS-arm64-unsigned.ipa` and `ByFTP-<version>-iOS-arm64-unsigned-app.zip`.
- iOS production signing remains external: repository IPA/app ZIP artifacts are real unsigned device builds, not falsely labeled App Store/TestFlight packages.
- Hardened Android `RemotePaths` so traversal, dot components, duplicate separators, backslashes and noncanonical names fail closed rather than being normalized.
- Hardened Android SFTP fingerprint input by Base64-decoding it and requiring exactly a 32-byte SHA-256 digest before SSHJ receives the canonical OpenSSH fingerprint.
- Android connection input now rejects CR/LF/NUL credential control characters before protocol libraries receive them.
- Expanded Android and iOS audits so path, credential, TLS, lifecycle, signing and package structure guarantees fail closed in CI.
- Expanded the cross-platform release contract to require Windows x64/x86, Linux amd64/arm64/i386, macOS Universal, Android debug/unsigned APK and iOS arm64 unsigned IPA/app ZIP artifacts from the same canonical `VERSION`.
- Updated README, installation, architecture, security, privacy, shared-hosting, testing, signing, release-verification, GitHub Releases, roadmap and build-tool documentation for the five-platform 1.2.0 release.
Before the stable 1.0.0 line, the project used internal development versioning during architecture, packaging and security-hardening work. The detailed pre-1.0 history remains available in Git history; this changelog tracks the public stable semantic line from 1.0.0 onward.
