# Ghost FTP Installation

GitHub Releases is the canonical source for published Ghost FTP binaries and checksums.

At the start of the 0.17.0 development cycle, the latest published release is **0.16.0**.

## Verify downloads

For a published version `<version>`, download `GhostFTP-v<version>-SHA256SUMS.txt` together with the required artifact and verify the checksum before installation.

Linux example:

```bash
sha256sum -c GhostFTP-v<version>-SHA256SUMS.txt
```

On Windows, use a SHA-256-capable verification tool and compare against the canonical checksum file.

## Windows x64

Published Windows assets follow these names:

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Windows-x64-v<version>.zip`

The Setup executable is the production **native Tauri NSIS package**. It is not the Go compatibility installer.

The portable executable runs without an installation workflow. The Setup executable installs the application using the native package configuration and should be used when normal installed-app integration is preferred.

Production-signing policy is tracked separately from package generation; do not treat an unsigned development artifact as a stable/FINAL signing claim.

## Linux x86-64

Published Linux assets follow these names:

- `GhostFTP-Linux-x86_64-v<version>`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Linux-amd64-v<version>.deb`
- `GhostFTP-Linux-x86_64-v<version>.rpm`
- `GhostFTP-Linux-x86_64-v<version>.tar.gz`

Use the package format appropriate to the target distribution/environment and verify its checksum before installation.

## Android

The canonical release asset is:

- `GhostFTP-Android-v<version>.apk`

The release workflow obtains this APK from the verified Android CI artifact for the same source SHA. CI verifies the installable preview signature/package contract before release packaging.

The separate unsigned Android release-check APK is a CI validation artifact and is not the canonical end-user APK.

## Building instead of installing

Source-build instructions are maintained in [../development/BUILDING.md](../development/BUILDING.md).

The authoritative source directories are:

- desktop: `ghostftp-desktop/`
- Android: `android/`

Support/compatibility tooling under `tools/` must not be confused with the production desktop packages described above.
