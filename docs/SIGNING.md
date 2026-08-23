# Code signing

ByFTP separates build integrity from publisher identity. CI can prove what source and version produced an artifact, but it must not fabricate a trusted operating-system publisher identity.

## Windows

Windows Verified Publisher status requires a real Authenticode certificate controlled by the legitimate publisher. Until such a certificate is configured in protected CI secrets, release verification relies on the gated build, release provenance and SHA-256 checksums rather than a simulated publisher identity.

## macOS

macOS trust requires a real Developer ID identity and Apple notarization credentials. The project must not claim notarized status without completing the actual Apple signing/notarization flow.

## Android

The official release workflow produces two Android artifacts:

- `ByFTP-<version>-Android-debug.apk` — signed with the standard Android debug identity and suitable only for development/testing installation.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized/minified release output without a production signature.

A production Android package must be signed with a stable private Android signing identity managed outside the repository, then verified as part of a dedicated signing gate. The debug key is not a production identity and the unsigned release artifact must never be presented as store-ready software.

## Secret handling

Signing certificates, private keys, keystores, passwords and notarization credentials belong in protected CI secret storage. They must never be committed to the repository, embedded in build scripts or generated as fake production identities.
