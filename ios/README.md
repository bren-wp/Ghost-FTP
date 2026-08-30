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
- Download to an isolated application temporary directory followed by the system share/save sheet.
- Keychain storage of the last **non-secret** connection preset (protocol, host, port and username) using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`.
- Session-only credentials. Password text is cleared after connection attempts and password data is not part of the persistent preset model.
- A touch-friendly SwiftUI Menu for upload, new folder, direct path, saved-preset removal and disconnect.
- Automatic disconnect when the app enters the background, including a connection that is still being established.

Explicit FTPS and SFTP are **not** claimed by the iOS implementation yet. They remain available on the existing ByFTP desktop and Android clients. Adding either transport to iOS requires a separately audited implementation rather than a permissive compatibility shim.

## 1.3.0 mobile update

Version 1.3.0 substantially improves daily phone/tablet use while preserving the existing native transport boundary.

The remote browser now exposes `.searchable` filtering, folders-first sorting, a direct canonical path dialog, multi-file import and a compact bottom connection/activity strip. Filtering is local to the currently loaded directory and does not issue a network request for every keystroke.

The document picker uses `allowsMultipleSelection: true`. All selected names are validated before the batch starts, duplicate destination names in one selection are rejected and every security-scoped URL is released after its upload attempt. The remote directory is refreshed once after a successful batch rather than after every file.

ByFTP can now restore the last successful connection metadata without storing its password. `ConnectionPreset` deliberately contains only protocol, host, port and username. It round-trips through `Codable`, is revalidated through `ConnectionConfig` when loaded and is stored in the iOS Keychain with a this-device-only accessibility class. Model regressions verify that serialized preset data cannot contain the session password. `UserDefaults` remains forbidden by the iOS audit.

The connection screen now clearly reports when safe metadata was restored and provides a direct **Forget saved connection** control. The browser menu exposes the same removal action while connected.

## Transport and security baseline

The path/lifecycle hardening introduced before 1.3.0 remains unchanged: raw host, port, username and password input is checked for CR/LF/NUL controls before normalization; remote names reject edge whitespace and controls; server login roots reject unsafe content; pending connections are disconnectable; password UI state is cleared; and failed/stale download staging is removed.

The connection task now derives the non-secret `ConnectionPreset` before asynchronous network completion, so the credential-bearing `ConnectionConfig` is not retained merely for persistence after authentication.

`ios/BUILD.sh` remains the canonical build entry point. CI and production release jobs invoke it directly, so the Swift source, Xcode project, tests and build contract live together instead of being split between `ios/` and a platform-specific wrapper under `scripts/`.

## Open in Xcode

Open:

```text
ios/ByFTP.xcodeproj
```

The checked-in project uses a safe development fallback `MARKETING_VERSION = 0.0.0`. Production/CI builds override it from the repository root `VERSION` file.

The AppIcon asset catalog stores only `Contents.json`. `ios/BUILD.sh` generates all required PNG sizes from the canonical repository `build/icon.png`; generated icon files are intentionally ignored by Git.

The deployment target remains iOS 16.0. The 1.3.0 SwiftUI changes deliberately avoid newer-only APIs such as `ContentUnavailableView`, and the real iPhoneOS build gate continues to verify this source against the project deployment target.

## Build the unsigned release artifacts

On macOS with Xcode installed:

```bash
bash ios/BUILD.sh
```

The script performs model/path/preset regressions, validates the Xcode project and shared scheme, builds a generic arm64 `iphoneos` Release application with signing disabled, verifies bundle version/identifier/architecture and then runs `scripts/package_ios.py`.

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
- Pending and active connections are both invalidated and closed during disconnect/background teardown.
- Failed or stale download staging directories are removed.
- Server-provided passive-mode addresses are not trusted as alternative destinations.
- App-bundle packaging rejects symlinks and validates the bundle identifier, version and Mach-O executable before creating release archives.
