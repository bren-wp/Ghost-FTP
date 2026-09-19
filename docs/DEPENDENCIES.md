# Dependencies

Ghost FTP keeps runtime dependencies narrow and explicit.

## Shared desktop

The desktop applications use the Go standard/runtime dependencies already declared by the repository and operating-system facilities required by supported protocol paths.

Desktop SFTP uses the maintained operating-system OpenSSH tooling and strict host-key trust logic.

## Windows

Windows packaging and signing use Windows-native tooling during build/release. Production signing credentials are external secrets and are never runtime dependencies.

## Linux

Linux build/package tooling is used only to produce validated Debian, Ubuntu and Fedora bundles. Runtime protocol work remains in the maintained application/OS boundary.

## Android

Android builds use the maintained Android/Gradle toolchain. The application does not silently inherit desktop-only dependencies.

## Browser helpers

Browser helpers use no FTP runtime library and do not perform FTP/FTPS/SFTP transport. They are local parser/copy helpers with a sanitized Windows desktop handoff.

## Build-time versus runtime

A platform SDK, compiler, signer or packaging utility required only during CI/build does not become an installed-application runtime dependency.

The retired macOS toolchain is no longer part of the active dependency contract.
