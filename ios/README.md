# Ghost FTP for iOS

The iOS project provides the native **Ghost FTP** client for iPhone/iPad targets.

## CI build

The repository build script is:

```bash
bash ios/BUILD.sh
```

The public GitHub Release publishes:

```text
Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa
```

The artifact is intentionally labeled unsigned. Normal device, TestFlight or App Store distribution requires a valid Apple signing identity, provisioning profile, bundle identifier and entitlements.

## Branding and compatibility

The app display name is **Ghost FTP**. Legacy Xcode target/project identifiers may remain internally during the rebrand to preserve build and migration compatibility. They must not be used as the public product name.

Never commit Apple signing certificates, provisioning profiles containing private deployment data or account credentials to the public repository.
