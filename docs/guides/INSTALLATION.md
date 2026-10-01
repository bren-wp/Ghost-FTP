# Ghost FTP Installation

GitHub Releases is the canonical source for published Ghost FTP binaries and checksums.

Use the newest non-draft GitHub Release as the canonical published build. The active source/release cycle is **0.20.7** and the previous canonical release is **0.20.6**. Publication of 0.20.7 is authoritative only when the immutable `v0.20.7` GitHub Release/tag exists.

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

The interactive Setup is branded with Ghost FTP artwork and follows a guided Windows installer flow. Before installation it presents the Ghost FTP EULA from the repository root; installation must not continue interactively unless the user accepts the licence terms. The installer uses the Ghost FTP application icon, branded header/sidebar artwork, a Ghost FTP Start Menu folder, and automatically follows the supported Windows locale (English/Croatian installer resources are enabled).

Android production builds are intentionally delivered without a private release-signing key. The unsigned production package is build evidence and is not directly installable by Android; CI separately keeps a non-debuggable installable preview, signed only with the standard development key, for functional install/reinstall/launch verification.

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

- `GhostFTP-Android-v<version>.apk.unsigned`
- `GhostFTP-Android-v<version>-Installable-Preview.apk`

The release workflow obtains both Android artifacts from the verified Android CI run for the same source SHA. `GhostFTP-Android-v<version>.apk.unsigned` is the production `com.ghostftp.android` build with no private release signature. `GhostFTP-Android-v<version>-Installable-Preview.apk` is kept separately as the non-debuggable install/reinstall/launch-tested package and uses only the standard development signing identity.

Android itself requires a signature before an APK can be installed. Therefore the intentionally unsigned production artifact is not presented as directly installable; no private keystore, signing password, user PIN or activation code is required by this project.

## Building instead of installing

Source-build instructions are maintained in [../development/BUILDING.md](../development/BUILDING.md).

The authoritative source directories are:

- desktop: `ghostftp-desktop/`
- Android: `android/`

Support/compatibility tooling under `tools/` must not be confused with the production desktop packages described above.
