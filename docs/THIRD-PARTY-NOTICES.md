# Third-party notices

Ghost FTP's maintained application platforms are **Windows, Linux and Android**. Chrome, Edge, Firefox and Opera helper ZIPs are supporting local packages.

## Go and operating-system facilities

The desktop applications use the Go toolchain/runtime and operating-system services required by supported functionality.

Desktop SFTP uses maintained operating-system OpenSSH tooling together with Ghost FTP's strict host-key trust boundary.

## Android toolchain

Android builds use the maintained Android SDK / Gradle toolchain. Build-time tooling is not a Ghost FTP telemetry service.

## Browser helper

Browser helpers have no FTP engine. They are local parser/copy helpers.

On supported Windows installs, the explicit Open in Ghost FTP action uses a sanitized `ghostftp:` handoff. It excludes passwords, private-key passphrases, private keys, source query data and fragments, and it does not automatically connect.

## Signing tools

Windows and Android signing tools/credentials are release infrastructure. Production publisher keys are never runtime dependencies and must never be committed to source.

## Retired platform

The previous macOS build/signing toolchain is no longer part of the active product or release dependency boundary.
