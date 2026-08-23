# Release verification

`VERSION` is the only canonical ByFTP product-version source. Desktop build scripts inject it into binaries/package metadata and Android derives `versionName`/`versionCode` from the same file.

Verify public release files with SHA-256 before redistribution. On Windows use `Get-FileHash`; on Linux use `sha256sum`; on macOS use `shasum -a 256`.

Release automation validates expected desktop filenames, architecture and version metadata. Starting with 1.1.0 Android source validation is also a required publication dependency: static Android audit, unit tests, lint and debug APK compilation must pass.

No production Android APK should be inferred from the Android CI artifact. A public Android package becomes eligible only after a stable external signing identity and signing-verification procedure are documented and gated.

Publish artifacts produced by the gated workflow rather than rebuilding unverified files manually.
