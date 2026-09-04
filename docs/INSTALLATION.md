# Installing Ghost FTP

Download Ghost FTP only from the repository Releases page and verify the matching entry in `SHA256.txt` before installation.

## Windows

Choose one installer:

- `Ghost-FTP-X.Y.Z-Setup-x64.exe` for 64-bit Windows.
- `Ghost-FTP-X.Y.Z-Setup-x86.exe` for 32-bit x86 Windows.
- `Ghost-FTP-X.Y.Z-Setup-x32.exe` is a compatibility alias of the x86 installer and has identical bytes/checksum.

The public product name is **Ghost FTP**. Some migration-sensitive internal identifiers can retain the former `byftp` name so existing profiles and cleanup paths continue to work.

## Linux

Download `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`, extract it and install the Debian package matching the machine architecture:

```text
Ghost-FTP-X.Y.Z-Linux-amd64.deb
Ghost-FTP-X.Y.Z-Linux-arm64.deb
Ghost-FTP-X.Y.Z-Linux-i386.deb
```

The installed Debian package name is `ghost-ftp` and the command is:

```bash
ghostftp
```

Runtime dependencies declared by the package are `ca-certificates`, `curl` and `openssh-client`.

## macOS

Use `Ghost-FTP-X.Y.Z-macOS-Universal.pkg`. The package contains a universal application build for Intel x86_64 and Apple Silicon arm64.

A package built without a configured Apple Developer ID may trigger Gatekeeper warnings. Production Developer ID signing and notarization require real Apple credentials and are never simulated by CI.

## Android

Use `Ghost-FTP-X.Y.Z-Android.apk` for direct testing/installations that allow sideloading.

The public CI artifact is debug-signed. It is installable for testing but is not presented as a Play Store production-signed package. Store distribution requires a private production keystore and the appropriate release process.

## iOS

`Ghost-FTP-X.Y.Z-iOS-arm64-unsigned.ipa` is an unsigned device archive. It cannot replace normal Apple distribution signing. A valid Apple signing identity, provisioning profile and entitlements must be applied for the intended distribution channel.

## Web / shared hosting

Use `Ghost-FTP-X.Y.Z-Web.zip`.

The web client requires a supported PHP environment and writable application storage. Deploy the package over HTTPS, keep the application storage protected from direct web access and complete the setup flow before normal use.

The web client is intentionally marked `noindex` and includes defensive session/security headers. See [Shared hosting](SHARED-HOSTING.md) for deployment details.

## Verify before installation

For every platform:

1. Download the package and `SHA256.txt` from the same Ghost FTP Release.
2. Compute the local SHA-256 digest.
3. Compare it exactly with the value in `SHA256.txt`.
4. Read `BUILD-METADATA.txt` for signing/provenance status.
5. Do not install a package whose checksum or expected release tag does not match.

See [Release verification](RELEASE-VERIFICATION.md) for platform-specific commands.
