# ByFTP for iOS

`ios/` contains the native SwiftUI iPhone/iPad application and its Xcode project. It is not a WebView wrapper and does not share Android UI/runtime code.

## Current capabilities

- FTP on the user-selected server.
- Implicit FTPS with Apple Network.framework TLS, platform trust and endpoint validation.
- Shared-hosting login-root mapping so UI `/` represents the authenticated FTP account root.
- EPSV with PASV fallback; PASV response addresses are intentionally ignored and data connections remain bound to the user-selected host.
- MLSD with LIST fallback.
- Remote browse, refresh, upload, download, create directory, rename and delete.
- File upload through the iOS security-scoped document picker.
- Download to an application temporary file followed by the system share/save sheet.
- Session-only credentials. Password text is cleared from the UI after a connection attempt and no profile secret is written to `UserDefaults` or another persistent store.
- Automatic disconnect when the app enters the background, including a connection that is still being established.

Explicit FTPS and SFTP are **not** claimed by the iOS implementation yet. They remain available on the existing ByFTP desktop and Android clients. Adding either transport to iOS requires a separately audited implementation rather than a permissive compatibility shim.

## 1.2.2 path and lifecycle hardening

ByFTP continues to validate raw host, port, username and password text for CR/LF/NUL protocol-control characters before trimming or canonicalization. Version 1.2.2 applies the same fail-closed principle to remote item names: leading/trailing whitespace and CR/LF/NUL are rejected rather than silently removed.

Server-reported FTP login roots reject CR/LF/NUL before normalization. Traversal, duplicate separators, backslashes, NULs and unsafe login-root components remain blocked.

`SessionStore` now tracks both the established client and a client that is still connecting. Disconnect and background teardown invalidate the generation, drop both references and close both actors, so an in-flight connection does not continue merely because it has not yet become the active session.

The UI password field is cleared even when local connection validation fails. Download staging now has an explicit discard path: failed transfers and stale async results remove their private temporary directory instead of leaving abandoned `ByFTP-<UUID>` folders.

## Open in Xcode

Open:

```text
ios/ByFTP.xcodeproj
```

The checked-in project uses a safe development fallback `MARKETING_VERSION = 0.0.0`. Production/CI builds override it from the repository root `VERSION` file.

The AppIcon asset catalog stores only `Contents.json`. `scripts/BUILD-IOS.sh` generates all required PNG sizes from the canonical repository `build/icon.png`; generated icon files are intentionally ignored by Git.

## Build the unsigned release artifacts

On macOS with Xcode installed:

```bash
bash scripts/BUILD-IOS.sh
```

The script performs model/path regressions, validates the Xcode project and shared scheme, builds a generic arm64 `iphoneos` Release application with signing disabled, verifies bundle version/identifier/architecture and then runs `scripts/package_ios.py`.

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
- Pending and active connections are both invalidated and closed during disconnect/background teardown.
- Failed or stale download staging directories are removed.
- Server-provided passive-mode addresses are not trusted as alternative destinations.
- App-bundle packaging rejects symlinks and validates the bundle identifier, version and Mach-O executable before creating release archives.
