# Ghost FTP signing policy

Ghost FTP distinguishes **build success** from **publisher signing**. GitHub Actions can compile and package the applications, but it cannot legitimately manufacture Windows, Apple or Android publisher identities.

## Windows

Authenticode signing requires a real code-signing certificate and private key held outside the public repository. If no signing identity is configured, Ghost FTP Windows installers remain unsigned and release metadata must say so explicitly.

Never commit signing certificates, PFX passwords or hardware-token credentials to the repository.

## macOS

Developer ID Application/Installer signing and notarization require valid Apple Developer credentials. A CI-built universal `.pkg` without those credentials can be used for controlled testing, but Gatekeeper may warn or block it according to local policy.

Production publication should use private CI secrets or a dedicated signing system and should verify the notarization result before release promotion.

## iOS

The public Ghost FTP pipeline publishes `Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa`. This is intentionally labeled **unsigned**.

Device/TestFlight/App Store distribution requires the appropriate Apple certificate, provisioning profile, bundle identifier and entitlements. Those credentials are deployment secrets, not repository content.

## Android

The public CI pipeline builds an installable debug-signed APK. It is suitable for testing and sideload verification but is not a Play Store production release.

Production Android distribution requires a protected release keystore and signing configuration outside the public repository. The release workflow must never rename a debug-signed APK in a way that implies production-store signing.

## Release metadata

Each Ghost FTP Release contains `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`. These files state the signing/provenance limitations for each platform. `SHA256.txt` provides integrity verification but does not substitute for publisher identity or platform notarization.
