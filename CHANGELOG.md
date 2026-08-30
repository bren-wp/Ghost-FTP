# Changelog

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
- Added shared-hosting FTP login-root mapping on iOS so UI `/` represents the authenticated account namespace and `public_html` stays inside that account root.
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
