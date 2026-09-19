# Packages

Ghost FTP publishes three maintained native applications: **Windows, Linux and Android**.

## Public application artifacts

### Windows

- `Ghost-FTP-<version>-Setup.exe`
- `Ghost-FTP-<version>-Portable.exe`

Each public Windows launcher carries the maintained x64, x86 and ARM64 native payloads. Native ARM64 runtime execution is not claimed unless dedicated runtime evidence exists.

### Linux

For each of Debian, Ubuntu and Fedora:

- `Ghost-FTP-<version>-Linux-<Distro>-Installer.run`
- `Ghost-FTP-<version>-Linux-<Distro>-Portable.tar.gz`

Each bundle carries amd64, arm64 and i386 payloads and selects the local architecture.

### Android

- `Ghost-FTP-<version>-Android.apk`

Official publication requires the configured production signing identity and signer-fingerprint verification.

## Release metadata

Every release also contains:

- `BUILD-METADATA.txt`
- `RELEASE-NOTES.txt`
- `SHA256.txt`

The active public contract is:

```text
PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID
ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID
PUBLIC_PLATFORM_ARTIFACTS=9
PUBLIC_RELEASE_FILES=12
```

macOS and browser extensions are retired and are excluded from current source/release packaging.

Published versions are immutable. Any later source, UI, behavior, documentation, packaging or release-metadata change requires a higher version.
