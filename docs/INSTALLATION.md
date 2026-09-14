# Ghost FTP installation

Ghost FTP **0.0.6** is the current published release. Root `VERSION` is the authoritative build/version source.

## Canonical release packages

The canonical public release contains **18 platform artifacts / 21 public files**: two Windows executables, twelve Linux packages/archives, one production-signed Android APK, three browser-helper ZIPs and three release metadata/verification files.

### Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Both public files are self-contained universal launchers carrying verified native **x64, x86 and ARM64** Ghost FTP payloads. `GetNativeSystemInfo` selects the embedded native application payload; no runtime architecture download occurs.

Official Windows publication requires trusted Authenticode. A successful official release records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

`WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` is an explicit evidence boundary: ARM64 is cross-built and structurally verified, not claimed as natively executed by the maintained Windows CI runner.

### Linux

Canonical release packages are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
Ghost-FTP-0.0.6-Linux-Debian-arm64.deb
Ghost-FTP-0.0.6-Linux-Debian-i386.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-i386.tar.gz
```

Native package-manager/runtime/GUI verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. Other canonical architectures retain exact-head build, metadata, extraction and binary-parity verification without an unsupported native-runtime claim.

### Android

The public Android file is:

```text
Ghost-FTP-0.0.6-Android.apk
```

It is a **production-signed public Android release**. Publication requires the protected Android publisher keystore/password/alias credentials and verifies the signer certificate SHA-256 fingerprint against `GHOSTFTP_ANDROID_SIGNER_SHA256`. The ordinary exact-head development artifact remains separate as `Ghost-FTP-Android-dev.apk`.

Android supports FTP and explicit FTPS Quick Connect with certificate/hostname validation, uses the Android **Storage Access Framework**, and keeps **SFTP intentionally hidden** until strict maintained host-key verification exists. Production signing does not change that protocol boundary.

### Browser helper packages

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

The helpers parse/copy explicitly entered FTP/FTPS/SFTP targets locally. They do not provide a supported browser-to-desktop URI/native-messaging handoff and do not introduce a Ghost FTP relay.

### macOS development app

The native **macOS development app** remains outside the 21-file public release. Development build success is not Developer ID signing or notarization proof. Public macOS distribution requires the separate real Developer ID + Apple notarization path.

## Windows Setup

1. Download `Ghost-FTP-0.0.6-Setup.exe`.
2. Verify SHA-256 against `SHA256.txt`.
3. Require `WINDOWS_AUTHENTICODE=signed` in `BUILD-METADATA.txt` and a valid Authenticode signature.
4. Run Setup as the intended user.
5. Uninstall through the integrated installed application path.

Setup embeds the native x64/x86/ARM64 payloads and does not install a separate permanent uninstaller executable.

## Windows Portable

Run `Ghost-FTP-0.0.6-Portable.exe` directly. It performs no installer registration and selects the embedded native payload without downloading another executable.

## Linux examples

```bash
sudo apt install ./Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
sudo apt install ./Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
sudo dnf install ./Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm

tar -xzf Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
cd Ghost-FTP-0.0.6-Linux-Portable-amd64
./ghostftp
```

DEB packages declare `ca-certificates`, `curl` and `openssh-client`; Fedora packages declare `ca-certificates`, `curl` and `openssh-clients`. Portable archives include the executable, desktop metadata, icon, README and LICENSE.

## Android installation

1. Download `Ghost-FTP-0.0.6-Android.apk` from the canonical GitHub Release.
2. Verify SHA-256 against `SHA256.txt`.
3. Verify the APK signature/certificate fingerprint as described in [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md).
4. Install through Android's normal package installer after explicitly permitting the chosen download source if required by the device policy.

Ghost FTP does not request broad all-files access; local files remain scoped through SAF.

## Browser helper installation

Extract only the package intended for the target browser and follow that browser's extension-development/package-loading policy. Publication of the ZIPs does not imply store signing/approval, automatic update service or desktop launch integration.

## Verification

Before using an official 0.0.6 artifact:

1. confirm `VERSION=0.0.6`, `TAG=ghostftp-v0.0.6`, `CHANNEL=Current` and `PRERELEASE=false`;
2. verify the filename belongs to the canonical 21-file set;
3. verify SHA-256 against `SHA256.txt`;
4. verify `BUILD-METADATA.txt` is bound to the exact release source commit;
5. for Windows, require trusted Authenticode and `WINDOWS_AUTHENTICODE=signed`;
6. for Android, verify the production signing certificate SHA-256 fingerprint;
7. keep `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` and Android SFTP's hidden/unsupported status as explicit evidence/security boundaries.

The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.6`; it is distribution infrastructure, not a runtime container.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [GitHub Releases](GITHUB-RELEASES.md).
