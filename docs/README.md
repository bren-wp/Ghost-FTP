# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="118">
</p>

<p align="center"><strong>Authoritative product behavior, security boundaries, native-platform status, UI evidence and release engineering for Ghost FTP.</strong></p>

- **Current Ghost FTP release: 0.0.6**
- Development status: **Active**
- Release channel: **Current**
- GitHub Release policy: **PRERELEASE=false**
- Public version retention: **latest release only**
- Public application platforms: **Windows, Linux and Android**
- Public browser-helper packages: **Chrome, Edge and Firefox**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Windows native payloads: **x64, x86 and ARM64 inside the same two public universal executables**
- Desktop protocols: **FTP, FTPS and SFTP**
- Android protocols: **FTP and strict explicit FTPS**; SFTP remains hidden until strict maintained host-key verification exists
- Desktop languages: **24 selectable local languages**
- Release shape: **18 platform artifacts / 21 public files**
- Product website: **https://ghostftp.com**
- Verified distribution bundle: **ghcr.io/bren-wp/ghost-ftp:0.0.6**

The root [`VERSION`](../VERSION) file is authoritative. Ghost FTP 0.0.6 publishes Windows, Linux, a protected production-signed Android APK and three deterministic browser-helper ZIPs. macOS remains a separately validated development/source frontend until real Developer ID signing and Apple notarization succeed.

## Documentation principles

1. **Current behavior first.** Active documents describe maintained behavior and the current 0.0.6 release contract.
2. **No invented platform status.** Android publication does not expose SFTP without strict host-key verification; macOS development evidence is not production notarization evidence.
3. **Security claims fail closed.** Official Windows publication requires trusted Authenticode; official Android publication requires the protected publisher identity and expected signer SHA-256 fingerprint.
4. **Visual claims use authentic evidence.** Documentation media is **repository-local** and maintained runtime screenshots come from real **Windows, Linux and Android** surfaces captured by **exact-head** CI, not generated **mockup** output.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Current settings | [`SETTINGS.md`](SETTINGS.md) |
| UI behavior and screenshots | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Security | [`SECURITY.md`](SECURITY.md) |
| Privacy | [`PRIVACY.md`](PRIVACY.md) |
| Architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Platform boundaries | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Release verification | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| Signing | [`SIGNING.md`](SIGNING.md) |
| Release/version lifecycle | [`VERSIONING.md`](VERSIONING.md) and [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| Build and test | [`TESTING.md`](TESTING.md) |
| Support | [`SUPPORT.md`](SUPPORT.md) |
| macOS development | [`../macos/README.md`](../macos/README.md) |
| Browser helper | [`../ekstenzije/README.md`](../ekstenzije/README.md) |

## Authentic visual reference

The following maintained media is **repository-local**. The exact-head workflow assembles a **read-only verified cross-platform evidence bundle** from authentic Windows, Linux and Android runtime captures. Generated mockup or manually composed approximations are not production evidence.

<table>
<tr>
<td width="50%"><strong>Main Workspace</strong><br><img src="images/ghost-ftp-main-workspace.png" alt="Ghost FTP Main Workspace"></td>
<td width="50%"><strong>Site Manager</strong><br><img src="images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
</tr>
<tr>
<td width="50%"><strong>Settings</strong><br><img src="images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
<td width="50%"><strong>About</strong><br><img src="images/ghost-ftp-about.png" alt="Ghost FTP About"></td>
</tr>
</table>

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for exact provenance and evidence counts.

## Current 0.0.6 capability contract

Ghost FTP **0.0.6** preserves the maintained Windows/Linux desktop engine and promotes the Android APK into the protected public release contract without relaxing Android protocol safety.

- Windows/Linux desktop: FTP, explicit FTPS and SFTP through the shared typed `internal/api.Engine`.
- Windows: two public universal Setup/Portable EXEs with native x64/x86/ARM64 payloads and required trusted Authenticode.
- Linux: twelve canonical Debian/Ubuntu/Fedora/Portable artifacts.
- Android: production-signed APK with FTP + strict explicit FTPS, SAF-scoped storage and SFTP intentionally hidden until strict host-key verification exists.
- Browser helper: deterministic Chrome/Edge/Firefox ZIPs; local parse/copy only, with no supported browser-to-desktop handoff.
- macOS: universal native development app only; no public distribution claim without Developer ID + notarization evidence.
- Privacy: no telemetry, analytics, advertising, fingerprinting or hidden Ghost FTP backend.

## Current publication identity

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
```

Representative public artifacts:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.6-Android.apk
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

Windows publication records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Android development CI remains separate as `Ghost-FTP-Android-dev.apk`. The public `Ghost-FTP-0.0.6-Android.apk` must be signed by the protected production publisher and match the configured signer SHA-256 fingerprint.

The complete verified release directory is also distributed as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

This is a **distribution bundle**, **not a runtime container**.

## Documentation map

### Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md)
- [`REFERENCE-UI.md`](REFERENCE-UI.md)
- [`SETTINGS.md`](SETTINGS.md)
- [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md)
- [`QUEUE-PRIORITY.md`](QUEUE-PRIORITY.md)
- [`LOCALIZATION.md`](LOCALIZATION.md)
- [`DEPENDENCIES.md`](DEPENDENCIES.md)

### Security, privacy and signing

- [`SECURITY.md`](SECURITY.md)
- [`PRIVACY.md`](PRIVACY.md)
- [`SIGNING.md`](SIGNING.md)
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md)

### Distribution and engineering

- [`INSTALLATION.md`](INSTALLATION.md)
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md)
- [`PACKAGES.md`](PACKAGES.md)
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md)
- [`VERSIONING.md`](VERSIONING.md)
- [`TESTING.md`](TESTING.md)
- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`ROADMAP.md`](ROADMAP.md)
- [`SUPPORT.md`](SUPPORT.md)
- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md)
- [`../CHANGELOG.md`](../CHANGELOG.md)

Only the latest public Ghost FTP version remains in active release infrastructure after retention cleanup succeeds. Git history remains the engineering provenance of earlier work.
