# ByFTP for iOS

`ios/` contains the native SwiftUI iPhone/iPad application, Xcode project, tests and canonical production build entry point. It is not a WebView wrapper and does not share Android UI/runtime code.

**Current release: 1.9.1**

## Current capabilities

- FTP on the user-selected server.
- Implicit FTPS with Apple Network.framework TLS, platform trust and endpoint validation.
- Shared-hosting login-root mapping so UI `/` represents the authenticated FTP account root.
- EPSV with PASV fallback; PASV response addresses are ignored and data connections remain bound to the user-selected host.
- MLSD with LIST fallback.
- Remote browse, refresh, upload/download, create directory, rename and delete.
- Native search/filter with deterministic directory-first sorting and canonical **Go to path** navigation.
- Multi-file upload through the security-scoped document picker.
- Byte-level transfer progress and safe **Stop after file** batch control.
- Shared-hosting diagnostics derived from the existing initial FTP/FTPS root listing without an extra network probe.
- Temporary download staging followed by the system share/save sheet.
- Keychain storage of non-secret protocol/host/port/username metadata using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`.
- Session-only credentials and automatic disconnect when the app enters the background.

Explicit FTPS and SFTP are **not** claimed by the iOS implementation. Adding either transport requires a separately audited native implementation rather than a permissive compatibility shim.

## 1.9.1 release integration

The iOS app uses the repository root `VERSION` for production `MARKETING_VERSION`, so release 1.9.1 is synchronized with Windows, Linux, macOS, Android and ByFTP WEB. The checked-in Xcode project deliberately keeps `MARKETING_VERSION = 0.0.0` as a safe development fallback; only the canonical build script injects the production release number.

The release matrix requires a real arm64 `iphoneos` Release build plus validation of both unsigned release artifacts before GitHub publication. No Apple signing identity, provisioning profile or private key is stored in the repository.

## Shared-hosting diagnostics

The application analyzes the same initial `list("/")` result that already populates the first remote browser view. `SharedHostingDiagnostics` recognizes common web-root directories in deterministic priority: `public_html`, `httpdocs`, `htdocs`, `www`, `web`, `html`. Only directories qualify.

The SwiftUI browser displays a **Shared hosting** section after a successful connection. It shows whether the selected transport is secure and, when available, the detected web-root name. The UI explicitly states that ByFTP does **not** open or save that path automatically.

Diagnostics are session-only. They are cleared on failed connect/disconnect, never written to `ConnectionPresetKeychain`, and do not alter the current path. The credential-bearing `ConnectionConfig` is not retained merely for diagnostic use; only the protocol value needed by the analyzer is extracted before asynchronous completion.

## Transfer behavior

`SocketConnection.sendFile` and `receiveToFile` report cumulative bytes after successful Network.framework sends or local-file writes. Callbacks flow through `FTPRemoteClient` into `SessionStore`; no third-party transfer SDK, hidden backend or alternative network destination is involved.

When size is known the browser shows determinate progress; otherwise it reports transferred bytes. **Stop after file** is a batch-boundary request: the active `STOR` transaction completes before remaining files are skipped. ByFTP does not tear down the active control/data connection merely to simulate immediate cancellation.

## Transport and security baseline

Raw host, port, username and password input is checked for CR/LF/NUL controls before normalization. Remote names reject edge whitespace/control characters and server login roots reject unsafe content. Pending connections are disconnectable, password UI state is cleared, and failed/stale download staging is removed.

Implicit FTPS uses platform TLS validation and `PBSZ 0` / `PROT P`. No custom trust-all callback or global App Transport Security bypass is used. EPSV is preferred; PASV fallback ignores the server-provided host and connects the data channel only to the endpoint selected by the user.

`ConnectionPreset` contains only protocol, host, port and username, is stored in Keychain with this-device-only accessibility and is revalidated before use. `UserDefaults` remains forbidden for connection state.

## Open in Xcode

Open:

```text
ios/ByFTP.xcodeproj
```

The checked-in project keeps the safe development fallback `MARKETING_VERSION = 0.0.0`. Production and CI builds override it from the repository root `VERSION` file. The deployment target remains iOS 16.0.

`ios/BUILD.sh` is the canonical build entry point. It generates AppIcon sizes from `build/icon.png`, runs model/path/preset/diagnostic regressions, builds a generic arm64 `iphoneos` Release app with repository-side signing disabled, validates the bundle and invokes `scripts/package_ios.py`.

## Release artifacts

```text
dist/ByFTP-<version>-iOS-arm64-unsigned.ipa
dist/ByFTP-<version>-iOS-arm64-unsigned-app.zip
```

These artifacts are deliberately unsigned because the repository contains no Apple Distribution private key or provisioning profile. An unsigned IPA is not an App Store/TestFlight package and requires legitimate external Apple signing before normal device distribution.

## Security and privacy notes

- Plain FTP is unencrypted and retained only for compatibility; prefer implicit FTPS where available.
- No fixed telemetry/API endpoint or advertising SDK is present.
- Remote UI paths reject traversal and noncanonical components before commands are sent.
- FTP command arguments independently reject CR/LF/NUL controls.
- Shared-hosting diagnostics are non-secret, session-only and cannot trigger automatic navigation or persistence.
- Pending and active connections are closed during disconnect/background teardown.
- Failed or stale temporary downloads are cleaned up.
- Server-provided passive-mode hosts are never trusted as alternative destinations.
- App-bundle packaging rejects symlinks and validates identifier, version and Mach-O executable before archive creation.

## Build and test

On macOS with Xcode installed:

```bash
bash ios/BUILD.sh
```

CI also runs `scripts/audit_ios.py`, the repository-wide integrity gate, dependency-free Swift model regressions and real arm64 iPhoneOS packaging validation.
