# Ghost FTP installation

Ghost FTP **0.0.8** is the current release target. The requested distribution is the no-secret 0.0.8 build; root `VERSION` remains authoritative. The existing **0.0.7** release remains the protected prior public baseline and is not rewritten. Root `VERSION` is the authoritative build/version source.

## Canonical 0.0.8 release packages

The 0.0.8 no-secret publication contains **14 platform artifacts / 17 public files**: two unsigned Windows executables, six Linux bundles, one installable CI debug-signed Android APK, one ad-hoc signed universal macOS validation app, four browser-helper ZIPs and three metadata/verification files.

### Windows

```text
Ghost-FTP-0.0.8-Setup.exe
Ghost-FTP-0.0.8-Portable.exe
```

Both are self-contained universal launchers carrying native **x64, x86 and ARM64** Ghost FTP payloads. The launcher selects the appropriate embedded payload locally; no architecture-specific runtime download is required.

The no-secret Windows distribution records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=unsigned
```

`WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` is an evidence boundary: ARM64 is cross-built and structurally verified, but the maintained Windows CI runner does not claim native ARM64 execution.

### Linux

Every supported distribution receives exactly **one Installer and one Portable archive**. Both carry amd64, arm64 and i386 payloads and select the native payload on the local machine.

```text
Ghost-FTP-0.0.8-Linux-Debian-Installer.run
Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.8-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.8-Linux-Fedora-Installer.run
Ghost-FTP-0.0.8-Linux-Fedora-Portable.tar.gz
```

Native installer/runtime/GUI verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. ARM64 and i386 payloads are exact-head build/package verified; native runtime execution is not claimed without a maintained runner or emulator proving it.

Installer example:

```bash
chmod +x Ghost-FTP-0.0.8-Linux-Debian-Installer.run
sudo ./Ghost-FTP-0.0.8-Linux-Debian-Installer.run
```

The installer checks required runtime tools and CA trust before installation. Debian/Ubuntu require `ca-certificates`, `curl` and `openssh-client`; Fedora requires `ca-certificates`, `curl` and `openssh-clients`. A real `ghostftp-uninstall` command is installed and removes Ghost FTP application files without deleting user configuration or server data.

Portable example:

```bash
tar -xzf Ghost-FTP-0.0.8-Linux-Debian-Portable.tar.gz
cd Ghost-FTP-0.0.8-Linux-Debian-Portable
./ghostftp
```

### Android

```text
Ghost-FTP-0.0.8-Android.apk
```

There is exactly one public Android APK. In the no-secret 0.0.8 distribution it is the installable CI debug-signed build. `apksigner` verifies its signature and `BUILD-METADATA.txt` records the observed certificate SHA-256 fingerprint; no protected Android publisher identity is claimed.

Android compatibility is defined by the application's maintained `minSdk`/target SDK and tested devices; no APK can truthfully support literally every historical Android version. Ghost FTP aims for the broadest safe compatibility supported by its Android APIs and dependencies.

Android supports FTP and strict explicit FTPS with certificate/hostname validation and uses the Android **Storage Access Framework**. **SFTP remains intentionally hidden** until a maintained implementation provides strict host-key verification/pinning and fails closed on unknown or mismatched hosts.

### Browser helper packages

```text
Ghost-FTP-0.0.8-Chrome-Extension.zip
Ghost-FTP-0.0.8-Edge-Extension.zip
Ghost-FTP-0.0.8-Firefox-Extension.zip
Ghost-FTP-0.0.8-Opera-Extension.zip
```

Each package is built from the shared local-only helper runtime and a browser-specific manifest under `extensions/<browser>/`. Official packages request zero browser permissions and zero host permissions. They parse/sanitize user-entered FTP/FTPS/SFTP targets locally and support a sanitized browser-to-desktop handoff through the registered `ghostftp:` protocol on supported Windows installs; they never carry passwords, private-key passphrases, private keys, source query data or fragments, and they never auto-connect.

### macOS

The public no-secret macOS artifact is `Ghost-FTP-0.0.8-macOS.app.zip`. It is ad-hoc signed and verified as a universal arm64+x86_64 validation app. It is **not Developer ID signed** and **not Apple notarized**, so macOS may require an explicit user override.

## Windows Setup

1. Download `Ghost-FTP-0.0.8-Setup.exe`.
2. Verify SHA-256 against `SHA256.txt`.
3. Confirm `WINDOWS_AUTHENTICODE=unsigned` in `BUILD-METADATA.txt`; Windows may show an Unknown Publisher/SmartScreen warning.
4. Run Setup as the intended user.
5. Uninstall through the integrated installed-application path.

Setup embeds x64/x86/ARM64 application payloads and does not require a second architecture-specific installer.

## Windows Portable

Run `Ghost-FTP-0.0.8-Portable.exe` directly. It performs no installer registration and selects the embedded native payload without downloading another executable.

## Android installation

1. Download `Ghost-FTP-0.0.8-Android.apk` from the canonical GitHub Release.
2. Verify SHA-256 against `SHA256.txt`.
3. Verify the APK signature and compare its certificate fingerprint with the value recorded in `BUILD-METADATA.txt`.
4. Install through Android's normal package installer after explicitly allowing the chosen download source if device policy requires it.

Ghost FTP does not request broad all-files access; local access remains SAF-scoped.

## Browser helper installation

Use the package for the target browser. Publication of the ZIPs does not imply Chrome Web Store, Edge Add-ons, Firefox AMO or Opera Add-ons approval/signing unless that store has separately accepted the exact package.

## Verification

Before accepting an official 0.0.8 artifact:

1. confirm `VERSION=0.0.8`, `TAG=ghostftp-v0.0.8`, `CHANNEL=Current` and `PRERELEASE=false`;
2. verify the filename belongs to the canonical **17-file** set;
3. verify SHA-256 against `SHA256.txt`;
4. verify `BUILD-METADATA.txt` binds the bundle to the exact release-source commit;
5. for Windows, confirm the intentionally unsigned state rather than assuming trusted Authenticode;
6. for Android, verify the APK signature and recorded CI signer fingerprint without treating it as a persistent publisher identity;
7. preserve documented evidence boundaries for Windows ARM64, Linux ARM64/i386, Android SFTP and macOS ad-hoc/not-notarized distribution.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
