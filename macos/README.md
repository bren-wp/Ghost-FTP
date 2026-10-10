# Ghost FTP for macOS

The `macos/` directory is the dedicated SwiftUI macOS client.

The first development phase establishes a real macOS application target, profile validation, non-secret profile persistence, optional password storage in macOS Keychain, transport-security messaging and TCP endpoint reachability checks. The endpoint check is intentionally not presented as a successful FTP/FTPS/SFTP login.

Protocol session work must preserve these rules:

- FTP is explicitly identified as unencrypted transport.
- Explicit FTPS must perform AUTH TLS and validate the certificate and hostname before file operations.
- SFTP must verify the SSH host key before authentication or file operations.
- Passwords and future private-key passphrases must not be written to UserDefaults/profile JSON.
- A TCP reachability check is not evidence that authentication or protocol negotiation succeeded.

## Saved-site restore safety (0.92.1)

Saved-site backups never contain Keychain passwords. An imported profile UUID
that already belongs to a saved site may update its display name or connection
preferences only when protocol, hostname, port and username remain unchanged.
A conflicting import is rejected in full before mutating any saved sites;
Keychain credentials remain untouched. Add a different destination as a new
site rather than restoring it over an existing UUID.

## Keychain integrity in the 0.92.2 patch branch

Replacing an existing password uses an in-place Keychain update, not
delete-then-add. If the update fails, the previous credential remains available.
A missing item is inserted, with one update retry if concurrent insertion
reports a duplicate. Site Save and Delete both stop without changing the saved
profile if the relevant Keychain operation fails. Both deletion paths
(Files and Sites) display a non-secret error rather than silently orphaning
credentials. These protections are exercised with injected Security
`OSStatus` failures in the Swift unit tests; production Keychain permission
acceptance remains a target-device verification item.

## Failure-safe Keychain updates (planned 0.92.2 patch)

Ghost FTP updates existing macOS Keychain password entries in place rather than
deleting the previous credential before attempting to add a replacement. Failed
writes retain the existing entry, while missing entries can be created and a
concurrent duplicate insertion can retry the update. Saved-site edits and both
Files/Sites deletion paths do not commit profile changes if their Keychain
write/removal fails; a non-secret error message is shown instead.

Swift regression tests exercise Keychain OSStatus error paths without depending
on a real test-runner Keychain. Production permissions and upgrade acceptance
still require macOS device testing.

## Build and test

From the repository root:

    bash macos/scripts/check-macos-contract.sh
    swift test --package-path macos
    swift build --package-path macos -c release
    bash macos/scripts/package-app.sh

`package-app.sh` creates an ad-hoc-signed preview application bundle and ZIP for CI validation. It is not a Developer ID/notarized production distribution.

Production signing/notarization and full FTP/explicit FTPS/SFTP session/file-transfer engines remain gated until their implementation and security tests are complete.

## Protocol and release status (0.30.14)

The premium SwiftUI sidebar now displays the bundled Ghost FTP application icon alongside the product name. The icon is sourced from the packaged `GhostFTP.icns` and is not a static mockup or a placeholder symbol.

### Protocol groundwork (0.30.13)

The SwiftUI Preview supports plain FTP browsing and upload/download. The 0.30.13 source fixes RETR/STOR filename interpolation, staged-download naming, rejects unsafe server listing names and bounds FTP control replies. Explicit FTPS and SFTP file operations remain blocked until real TLS certificate/hostname verification and SSH host-key verification are complete. CI-green ad-hoc signing is not Developer ID signing or Apple notarization; this is not a production macOS package.
