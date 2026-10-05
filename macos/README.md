# Ghost FTP for macOS

The `macos/` directory is the dedicated SwiftUI macOS client.

The first development phase establishes a real macOS application target, profile validation, non-secret profile persistence, optional password storage in macOS Keychain, transport-security messaging and TCP endpoint reachability checks. The endpoint check is intentionally not presented as a successful FTP/FTPS/SFTP login.

Protocol session work must preserve these rules:

- FTP is explicitly identified as unencrypted transport.
- Explicit FTPS must perform AUTH TLS and validate the certificate and hostname before file operations.
- SFTP must verify the SSH host key before authentication or file operations.
- Passwords and future private-key passphrases must not be written to UserDefaults/profile JSON.
- A TCP reachability check is not evidence that authentication or protocol negotiation succeeded.

## Build and test

From the repository root:

    bash macos/scripts/check-macos-contract.sh
    swift test --package-path macos
    swift build --package-path macos -c release
    bash macos/scripts/package-app.sh

`package-app.sh` creates an ad-hoc-signed preview application bundle and ZIP for CI validation. It is not a Developer ID/notarized production distribution.

Production signing/notarization and full FTP/explicit FTPS/SFTP session/file-transfer engines remain gated until their implementation and security tests are complete.
