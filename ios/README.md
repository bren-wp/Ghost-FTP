# ByFTP for iOS

`ios/` contains the native SwiftUI iPhone/iPad application, Xcode project, tests and canonical production build entry point. It is not a WebView wrapper and does not share Android UI/runtime code.

## Current capabilities

- FTP on the user-selected server.
- Implicit FTPS with Apple Network.framework TLS, platform trust and endpoint validation.
- Shared-hosting login-root mapping so UI `/` represents the authenticated FTP account root.
- EPSV with PASV fallback; PASV response addresses are intentionally ignored and data connections remain bound to the user-selected host.
- MLSD with LIST fallback.
- Remote browse, refresh, upload, download, create directory, rename and delete.
- Native searchable remote file list with deterministic directory-first sorting.
- **Go to path** navigation through the same canonical `RemotePath` validator used by file operations.
- Multi-file upload through the iOS security-scoped document picker with one directory refresh after the batch.
- Byte-level upload/download progress reported by the existing Network.framework data-socket file loops.
- Safe **Stop after file** for multi-file uploads; the active FTP transaction completes before remaining selected files are skipped.
- Shared-hosting diagnostics derived from the existing initial FTP/FTPS root listing without an extra network probe.
- Download to an isolated application temporary directory followed by the system share/save sheet.
- Keychain storage of the last **non-secret** connection preset (protocol, host, port and username) using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`.
- Session-only credentials. Password text is cleared after connection attempts and password data is not part of the persistent preset model.
- A touch-friendly SwiftUI Menu for upload, new folder, direct path, saved-preset removal and disconnect.
- Automatic disconnect when the app enters the background, including a connection that is still being established.

Explicit FTPS and SFTP are **not** claimed by the iOS implementation yet. They remain available on the existing ByFTP desktop and Android clients. Adding either transport to iOS requires a separately audited implementation rather than a permissive compatibility shim.

## 1.5.0 shared-hosting diagnostics

Version 1.5.0 analyzes the same initial `list("/")` result that already populates the first remote browser view. `SharedHostingDiagnostics` recognizes common web-root directories in deterministic priority: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Only directories qualify.

The SwiftUI browser displays a **Shared hosting** section after a successful connection. It shows whether the selected transport is secure and, when available, the detected web-root name. The UI explicitly states that ByFTP does **not** open or save that path automatically.

Diagnostics remain session-only. They are cleared on failed connect and disconnect, never written into `ConnectionPresetKeychain`, and do not change the current remote path. The model has no password/passphrase/private-key field and no Network.framework connection behavior of its own.

Model regressions verify implicit FTPS is reported as secure, plain FTP remains visibly insecure, `public_html` wins the documented priority and a file named `public_html` cannot masquerade as a directory web root. `audit_ios.py` blocks secret-bearing diagnostics, independent diagnostic network activity, automatic navigation and persistence.

## 1.4.0 transfer update

Version 1.4.0 made long iPhone/iPad transfers observable while keeping the native FTP/implicit-FTPS transport and trust model unchanged.

`SocketConnection.sendFile` and `receiveToFile` report cumulative bytes after successful Network.framework sends or local-file writes. Those callbacks are threaded through `FTPRemoteClient` into `SessionStore`; no third-party transfer SDK, WebView bridge, hidden backend or alternative network destination was added.

The browser's bottom activity area shows a determinate SwiftUI `ProgressView` when the source/remote size is known and an indeterminate progress indicator plus transferred-byte text when a stable size is unavailable. Progress reporting does not modify FTP commands, passive-host handling, TLS configuration or credential lifetime.

For multi-file uploads, **Stop after file** sets a batch-boundary request. ByFTP completes the active `STOR` transaction and only then skips remaining selected files. It does not cancel the active data socket or close the FTP control connection from the stop-button handler just to create an apparently instant cancellation state.

## 1.3.0 mobile update

Version 1.3.0 substantially improved daily phone/tablet use while preserving the existing native transport boundary.

The remote browser exposes `.searchable` filtering, folders-first sorting, a direct canonical path dialog, multi-file import and a compact bottom connection/activity strip. Filtering is local to the currently loaded directory and does not issue a network request for every keystroke.

The document picker uses `allowsMultipleSelection: true`. All selected names are validated before the batch starts, duplicate destination names in one selection are rejected and every security-scoped URL is released after its upload attempt. The remote directory is refreshed once after a successful batch rather than after every file.

ByFTP can restore the last successful connection metadata without storing its password. `ConnectionPreset` deliberately contains only protocol, host, port and username. It round-trips through `Codable`, is revalidated through `ConnectionConfig` when loaded and is stored in the iOS Keychain with a this-device-only accessibility class. Model regressions verify that serialized preset data cannot contain the session password. `UserDefaults` remains forbidden by the iOS audit.

The connection screen clearly reports when safe metadata was restored and provides a direct **Forget saved connection** control. The browser menu exposes the same removal action while connected.

## Transport and security baseline

Raw host, port, username and password input is checked for CR/LF/NUL controls before normalization; remote names reject edge whitespace and controls; server login roots reject unsafe content; pending connections are disconnectable; password UI state is cleared; and failed/stale download staging is removed.

The connection task derives the non-secret `ConnectionPreset` before asynchronous network completion, so the credential-bearing `ConnectionConfig` is not retained merely for persistence after authentication.

Transfer progress is observational. The progress closures receive only cumulative byte counts; the **Stop after file** request does not own or tear down the active socket. Shared-hosting diagnostics are also observational and consume only the already loaded root entries. Protocol-aware true mid-file cancellation remains deferred until abort semantics and partial-file cleanup can be proven fail-safe.

`ios/BUILD.sh` remains the canonical build entry point. CI and production release jobs invoke it directly, so the Swift source, Xcode project, tests and build contract live together instead of being split between `ios/` and a platform-specific wrapper under `scripts/`.

## Open in Xcode

Open:

```text
ios/ByFTP.xcodeproj
```

The checked-in project uses a safe development fallback `MARKETING_VERSION = 0.0.0`. Production/CI builds override it from the repository root `VERSION` file.

The AppIcon asset catalog stores only `Contents.json`. `ios/BUILD.sh` generates all required PNG sizes from the canonical repository `build/icon.png`; generated icon files are intentionally ignored by Git.

The deployment target remains iOS 16.0. The 1.5.0 SwiftUI diagnostics deliberately avoid newer-only APIs such as `ContentUnavailableView`, and the real iPhoneOS build gate verifies this source against the project deployment target.

## Build the unsigned release artifacts

On macOS with Xcode installed:

```bash
bash ios/BUILD.sh
```

The script performs model/path/preset/diagnostic regressions, validates the Xcode project and shared scheme, builds a generic arm64 `iphoneos` Release application with signing disabled, verifies bundle version/identifier/architecture and then runs `scripts/package_ios.py`.

Generated public build artifacts:

```text
dist/ByFTP-<version>-iOS-arm64-unsigned.ipa
dist/ByFTP-<version>-iOS-arm64-unsigned-app.zip
```

The `.ipa` contains the normal `Payload/ByFTP.app` structure. The ZIP contains the same unsigned `.app` bundle for inspection or external signing workflows.

## Signing limitation

These repository artifacts are deliberately **unsigned** because the repository does not contain an Apple Distribution certificate, private key or provisioning profile. An unsigned IPA is not an App Store/TestFlight package and is not installable on a normal iPhone until it is signed with a valid Apple identity and provisioning configuration outside this repository.

Do not commit `.p12` files, private signing keys, provisioning profiles or passwords to the repository. A future production-signed iOS release should consume those credentials from a protected signing environment and keep the current unsigned build as reproducible pre-signing evidence.

## Security and privacy notes

- Plain FTP is unencrypted and is retained only for compatibility. Prefer implicit FTPS where available.
- The iOS source contains no fixed telemetry/API endpoint and no advertising SDK.
- Global App Transport Security weakening is not enabled.
- Raw endpoint and credential input is checked for CR, LF and NUL before normalization.
- Remote names reject edge whitespace and protocol control characters rather than rewriting them.
- FTP command arguments independently reject CR, LF and NUL control characters.
- Remote UI paths reject traversal and noncanonical components before a command is sent.
- The persistent `ConnectionPreset` has no password/passphrase/secret field and is revalidated before use.
- Preset data uses Keychain `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`; `UserDefaults` remains disallowed for connection state.
- Shared-hosting diagnostics are session-only, non-secret and cannot trigger automatic navigation or persistence.
- Pending and active connections are both invalidated and closed during disconnect/background teardown.
- Failed or stale download staging directories are removed.
- Server-provided passive-mode addresses are not trusted as alternative destinations.
- Transfer progress stays on the existing Network.framework data path; safe batch stopping does not close an active transport mid-file.
- App-bundle packaging rejects symlinks and validates the bundle identifier, version and Mach-O executable before creating release archives.