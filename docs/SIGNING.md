# Signing

Ghost FTP separates product behavior from release signing.

## Windows

Official Windows publication requires the configured trusted Authenticode signing boundary. Development/no-key artifacts must be identified truthfully when unsigned.

## Android

Official Android publication requires the protected publisher keystore and verification of the exact signer SHA-256 fingerprint. Compatibility/dev signing must never be described as the permanent publisher identity.

## Linux

Linux installer/portable bundles are verified through exact-source build, packaging/install lifecycle checks and release SHA-256 metadata.

## macOS

macOS support is retired. There is no current Developer ID, notarization, stapling or Gatekeeper publication path in the active repository.

## Fail-closed rule

If a protected signing identity is unavailable, the protected release workflow must fail instead of substituting a generated identity and presenting it as production signing.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md) and [Versioning](VERSIONING.md).
