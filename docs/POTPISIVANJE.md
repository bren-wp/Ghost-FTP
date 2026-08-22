# Distribution signing

ByFTP separates build correctness from publisher signing.

## Windows

Authenticode can provide publisher identity and tamper detection for Windows executables. A real Brendigo code-signing certificate and protected private key are required. Without those credentials, Windows may not display a Verified Publisher identity even when the executable passed all project tests and SHA-256 verification.

## macOS

Developer ID signing and Apple notarization require a valid Apple Developer identity and notarization credentials. The macOS Universal package can be structurally validated without those credentials, but the project must not claim notarization unless Apple actually accepted the submitted package.

## CI secret handling

Signing keys, certificate passwords and notarization credentials must be stored only in protected CI secret facilities. They must never be committed to the repository, printed to logs or placed in release metadata.

## Verification

Signing is an additional trust signal; it does not replace source, build, test and SHA-256 integrity checks.