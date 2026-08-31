# Code signing

ByFTP separates build integrity from publisher identity. CI can prove what source/version produced an artifact, but it must not fabricate a trusted operating-system publisher identity.

## Windows

Verified Publisher requires a real Authenticode certificate controlled by the legitimate publisher. Until configured in protected CI, verification relies on the gated build, provenance and SHA-256 checksums.

ByFTP 1.8.0 signs/verifies only the public Windows `Setup` and `Portable` executables. The release pipeline no longer builds, embeds or publishes a separate `Uninstall.exe`, so no uninstaller signing identity or signing step exists in the current release contract.

## macOS

macOS trust requires a real Developer ID identity and Apple notarization credentials. The project must not claim signed/notarized status without completing the actual Apple flow.

## Android

The release workflow produces:

- `ByFTP-<version>-Android-debug.apk` — standard Android debug identity, development/testing only.
- `ByFTP-<version>-Android-release-unsigned.apk` — optimized/minified release output without a production signature.

Production Android distribution requires a stable private signing identity managed outside the repository.

## iOS

The release workflow produces:

- `ByFTP-<version>-iOS-arm64-unsigned.ipa` — a real arm64 iPhoneOS application packaged under `Payload/ByFTP.app`, built with code signing disabled.
- `ByFTP-<version>-iOS-arm64-unsigned-app.zip` — the same unsigned `.app` bundle for verification or an external signing workflow.

These artifacts are reproducible pre-signing evidence, **not** App Store/TestFlight packages and not normally installable on a stock iPhone/iPad. Production distribution requires a valid Apple signing certificate/private key plus an appropriate provisioning profile and, for store distribution, the corresponding App Store/TestFlight process.

The public workflow intentionally does not invent a development team, ad-hoc production identity or fake provisioning profile. A future protected signing stage may consume real Apple credentials after the unsigned build has passed the existing package integrity gate.

## Secret handling

Authenticode/Developer ID/Apple Distribution certificates, private keys, Android keystores, `.p12` files, mobile provisioning profiles, passwords and notarization credentials belong in protected signing infrastructure. They must never be committed to the repository, embedded in build scripts or generated as fake production identities.
