# Signing and artifact trust

Ghost FTP reports the trust state of each produced artifact explicitly.

## Windows

Protected production publication requires trusted Authenticode signing. Development and no-secret validation paths may produce unsigned artifacts, but those artifacts must be labeled truthfully.

## Android

Protected production publication requires the configured publisher keystore and exact signer fingerprint verification.

A compatibility/debug-signed APK must never be described as production-signed.

## Linux

Linux installer and portable bundles are distributed with checksum verification. Platform package metadata must remain tied to the exact source commit.

## Retired macOS signing path

The former Apple Developer ID / notarization pipeline has been removed together with the retired macOS application and is not part of current release policy.

## Policy

- no false production-signing claims;
- no hidden fallback from protected signing to weaker signing;
- no embedded production private keys;
- no publication when protected signing requirements fail;
- checksum generation happens from the final exact-head release directory.

See [Release verification](RELEASE-VERIFICATION.md) and [Packages](PACKAGES.md).

