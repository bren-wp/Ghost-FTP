# Ghost FTP installation

Ghost FTP **0.0.6** is the current published release. GitHub Release `ghostftp-v0.0.6` was published on 14 September 2026. Root `VERSION` remains `0.0.6` while the `production/0.0.7-cleanup` branch prepares the next development cycle; the published release is not retagged or republished.

## Canonical 0.0.6 release packages

The published 0.0.6 contract contains **13 platform artifacts / 16 public files**: two Windows executables, six Linux bundles, one production-signed Android APK, four browser-extension ZIPs and three metadata files.

### Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Both carry x64, x86 and ARM64 application payloads and select the local architecture without downloading another executable. Official publication requires trusted Authenticode.

On the 0.0.7 development branch, the Windows Setup also embeds the matching x64/x86/ARM64 `GhostFTPNativeHost.exe` payload used by the browser extensions. The application, native host and Firefox Native Messaging manifest are installed as one rollback-aware transaction. Setup records verified SHA-256 ownership metadata so integrated uninstall removes only the files installed by that Ghost FTP installation. The native host is Authenticode-signed whenever Windows production signing is configured.

Firefox registration is automatic because the extension has the stable Gecko ID `ghostftp-connection-helper@ghostftp.com`. Chrome, Edge and Opera registration remains fail-closed until exact official store extension IDs are available; Setup never installs wildcard `allowed_origins` and never invents an extension ID.

The Windows Portable executable remains portable: it does not mutate browser Native Messaging registration or silently install a persistent companion.

### Linux

```text
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
```

Each bundle contains amd64, arm64 and i386 application payloads. Native runtime evidence is claimed only for environments that actually execute the package. Installed Linux packages provide a real `ghostftp-uninstall` command.

On the 0.0.7 development branch, the same universal bundles additionally carry amd64/arm64/i386 `ghostftp-native-host` payloads. The default `/usr/local` Installer installs the architecture-matched host under `/usr/local/libexec/ghostftp/` and registers the fixed-ID Firefox Native Messaging manifest in the distro's system Mozilla native-messaging directory when that directory can be written. A custom prefix installs the host but deliberately does not write a system browser registration. Portable archives include the matching host binaries but do not mutate browser configuration.

Linux uninstall removes only the Ghost FTP Firefox manifest whose host path and fixed extension identity match the installed Ghost FTP bridge, then removes the companion and application files. Chromium-family registration is not installed until exact official extension IDs exist; wildcard origins are never generated.

### Android

```text
Ghost-FTP-0.0.6-Android.apk
```

The public APK is production-signed. Publication requires the protected keystore and exact signer certificate SHA-256 match against `GHOSTFTP_ANDROID_CERT_SHA256`.

Android exposes FTP and strict explicit FTPS with certificate/hostname validation and Storage Access Framework file access. SFTP remains intentionally hidden until a maintained implementation provides strict host-key verification/pinning and fails closed on unknown or mismatched hosts.

### Browser extensions

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

The published 0.0.6 browser packages are release history. On the 0.0.7 development branch, all four browser packages use the local `com.ghostftp.bridge` Native Messaging host and the existing Ghost FTP Engine for real FTP/FTPS/SFTP work. Browser JavaScript does not open raw protocol sockets and no Ghost FTP remote credential or file-transfer relay is introduced.

A browser ZIP by itself is intentionally insufficient for live protocol operations: its matching local native host must be installed and registered for that browser. This keeps the browser sandbox boundary explicit instead of simulating connectivity.

### macOS source

macOS remains an active native source surface and is not part of the 16-file public 0.0.6 allow-list. Public distribution requires a real Developer ID Application identity plus successful Apple notarization.

## Verification

Before accepting an official 0.0.6 artifact:

1. confirm `VERSION=0.0.6`, `TAG=ghostftp-v0.0.6`, `CHANNEL=Current` and `PRERELEASE=false`;
2. verify the filename belongs to the canonical **16 public files**;
3. verify SHA-256 against `SHA256.txt`;
4. verify `BUILD-METADATA.txt` binds the bundle to the release-source commit;
5. for Windows, require trusted Authenticode and `WINDOWS_AUTHENTICODE=signed`;
6. for Android, require the protected production certificate fingerprint `GHOSTFTP_ANDROID_CERT_SHA256`;
7. preserve the documented SFTP host-key and platform evidence boundaries.

The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.6`; it is distribution infrastructure, not a runtime backend.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
