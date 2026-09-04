# Ghost FTP GitHub Releases

Ghost FTP publishes release assets from `.github/workflows/release.yml` after the version change is merged to `main` and all platform quality gates pass.

## Version and tag policy

The Ghost FTP line starts at **1.0.0**. Versions follow Semantic Versioning and patch releases advance sequentially (`1.0.1`, `1.0.2`, ...).

Ghost FTP tags are namespaced:

```text
ghostftp-v1.0.0
ghostftp-v1.0.1
ghostftp-v1.0.2
```

Historical GhostFTP tags remain immutable. The workflow must never force-move an existing tag. Before updating an existing Ghost FTP release, it verifies that the tag already points to the exact release commit; otherwise publication stops.

## Public release contract

For version `X.Y.Z`, Releases contains these platform assets:

```text
Ghost-FTP-X.Y.Z-Setup-x64.exe
Ghost-FTP-X.Y.Z-Setup-x86.exe
Ghost-FTP-X.Y.Z-Setup-x32.exe
Ghost-FTP-X.Y.Z-Linux-multiarch.zip
Ghost-FTP-X.Y.Z-macOS-Universal.pkg
Ghost-FTP-X.Y.Z-Android.apk
Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa
Ghost-FTP-X.Y.Z-Web.zip
```

It also contains:

```text
SHA256.txt
RELEASE-NOTES.txt
BUILD-METADATA.txt
```

`Setup-x32.exe` is intentionally byte-identical to `Setup-x86.exe`; x32 and x86 are two labels for the same 32-bit Windows architecture in this release contract.

The Linux archive contains the verified `amd64`, `arm64` and `i386` Debian packages. The macOS package is universal for Intel x86_64 and Apple Silicon arm64.

## Signing status

Release automation does not invent trust identities.

- Windows installers may be unsigned unless a real Authenticode signing identity is supplied outside the public repository.
- The Android asset produced by public CI is an installable debug-signed APK, not a Play Store production-signed package.
- The iOS asset is an unsigned arm64 IPA that requires valid Apple signing/provisioning for ordinary device, TestFlight or App Store distribution.
- macOS Developer ID signing and notarization likewise require real Apple credentials.

The exact signing/provenance state is recorded in `BUILD-METADATA.txt` and described in `RELEASE-NOTES.txt`.

## Publication safety

The publish job runs only after core, Windows, Linux, macOS, Android and iOS jobs succeed. Immediately before publishing it verifies that `main` still points to the workflow commit. This prevents a delayed run from publishing stale binaries as the newest release.

The job creates `SHA256.txt` after assembling the final public filenames and verifies the expected file count before release creation.

## Manual dispatch

`workflow_dispatch` accepts an optional semantic version, but the requested value must match the repository `VERSION` file. Manual dispatch is therefore not a mechanism for publishing arbitrary or uncommitted versions.
