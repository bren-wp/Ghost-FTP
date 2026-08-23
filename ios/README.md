# ByFTP for iOS

`ios/` contains the native SwiftUI iPhone/iPad application and its Xcode project. It is not a WebView wrapper and does not share Android UI/runtime code.

## Supported in the first iOS release

- FTP on the user-selected server.
- Implicit FTPS with Apple Network.framework TLS, platform trust and endpoint validation.
- Shared-hosting login-root mapping so UI `/` represents the authenticated FTP account root.
- EPSV with PASV fallback; PASV response addresses are intentionally ignored and data connections remain bound to the user-selected host.
- MLSD with LIST fallback.
- Remote browse, refresh, upload, download, create directory, rename and delete.
- File upload through the iOS security-scoped document picker.
- Download to an application temporary file followed by the system share/save sheet.
- Session-only credentials. Password text is cleared from the UI after a connection attempt and no profile secret is written to `UserDefaults` or another persistent store.
- Automatic disconnect when the app enters the background.

Explicit FTPS and SFTP are **not** claimed by the iOS implementation yet. They remain available on the existing ByFTP desktop and Android clients. Adding either transport to iOS requires a separately audited implementation rather than a permissive compatibility shim.

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
- FTP command arguments reject CR, LF and NUL control characters.
- Remote UI paths reject traversal and noncanonical components before a command is sent.
- Server-provided passive-mode addresses are not trusted as alternative destinations.
- App-bundle packaging rejects symlinks and validates the bundle identifier, version and Mach-O executable before creating release archives.
