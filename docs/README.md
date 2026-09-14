# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP" width="118">
</p>

<p align="center"><strong>Product behavior, platform boundaries, privacy, security and release engineering for Ghost FTP.</strong></p>

## Current status

- Current source version: **0.0.6**
- Release channel: **Current**
- Development status: **Active**
- Last actually published GitHub Release: **0.0.5**
- 0.0.6 release target: `ghostftp-v0.0.6`, `PRERELEASE=false`
- 0.0.6 shape: **13 platform artifacts / 16 public files**
- Public release targets: **Windows, Linux, Android and browser helper packages**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Browser packages: **Chrome, Edge, Firefox and Opera**
- Desktop languages: **24 selectable local languages**, English default/fallback
- Product website: **https://ghostftp.com**
- License: **proprietary commercial software, Brendigo LTD**

0.0.6 is not treated as published until the exact verified `main` SHA succeeds through the protected Windows/Android signing and release-readback transaction. macOS remains development/source-only until real Developer ID Application signing and Apple notarization succeed.

## Documentation principles

1. **Describe what actually exists.** No platform, protocol, architecture or release is called production-ready without evidence supporting that claim.
2. **Security boundaries fail closed.** Official Windows publication requires trusted Authenticode; Android requires the protected publisher identity and exact `GHOSTFTP_ANDROID_CERT_SHA256` fingerprint.
3. **Development evidence is not production evidence.** CI smoke keys, self-signed certificates and ad-hoc macOS signing never substitute for production identities.
4. **Visual evidence must be authentic.** Repository-local screenshots may document the product, but only exact-head runtime captures count as execution evidence. Generated UI mockups never count as proof that an app runs.
5. **Privacy claims are scoped.** Ghost FTP itself has no application telemetry or hidden transfer relay; remote servers, operating systems, GitHub and other third parties remain outside that application-level guarantee.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Product architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Platform capability boundaries | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Security model | [`SECURITY.md`](SECURITY.md) |
| Privacy model | [`PRIVACY.md`](PRIVACY.md) |
| Signing identities | [`SIGNING.md`](SIGNING.md) |
| Release verification | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| GitHub publication flow | [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| Packages / GHCR | [`PACKAGES.md`](PACKAGES.md) |
| Version lifecycle | [`VERSIONING.md`](VERSIONING.md) |
| Localization | [`LOCALIZATION.md`](LOCALIZATION.md) |
| Testing and quality gates | [`TESTING.md`](TESTING.md) |
| UI screenshots/evidence | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Support | [`SUPPORT.md`](SUPPORT.md) |
| Linux | [`../linux/README.md`](../linux/README.md) |
| Android | [`../android/README.md`](../android/README.md) |
| macOS development | [`../macos/README.md`](../macos/README.md) |
| Browser helpers | [`../extensions/README.md`](../extensions/README.md) |

## 0.0.6 public artifact contract

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
Ghost-FTP-0.0.6-Android.apk
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

Canonical identity:

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
```

Windows publishes exactly two architecture-independent user-facing EXEs containing x64/x86/ARM64 payloads. Linux publishes one Installer and one Portable bundle per Debian/Ubuntu/Fedora, each carrying amd64/arm64/i386 payloads. Android publishes one production-signed APK. Browser helpers publish one deterministic ZIP each for Chrome, Edge, Firefox and Opera.

## Platform boundaries

Windows is the reference desktop application surface. Linux remains active development and a public 0.0.6 target with native amd64 install/runtime/GUI evidence on maintained Debian, Ubuntu and Fedora images. Android remains active development and a public 0.0.6 target with FTP + strict explicit FTPS; SFTP stays hidden until strict host-key verification/pinning exists. macOS remains active development with no public production package until Developer ID + notarization are proven.

## Authentic visual reference

<table>
<tr>
<td width="50%"><strong>Main workspace</strong><br><img src="images/ghost-ftp-main-workspace.png" alt="Ghost FTP main workspace"></td>
<td width="50%"><strong>Site Manager</strong><br><img src="images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
</tr>
<tr>
<td width="50%"><strong>Settings</strong><br><img src="images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
<td width="50%"><strong>About</strong><br><img src="images/ghost-ftp-about.png" alt="Ghost FTP About"></td>
</tr>
</table>

Authentic runtime evidence is maintained across **Windows, Linux and Android** and must remain bound to an exact-head source SHA. Repository-local images may describe the product, but generated mockups are never accepted as execution or release evidence.

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for provenance and exact-head evidence rules.

## Commercial proprietary license

Ghost FTP is not open-source software. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand or white-label the project. Ghost FTP is a copyrighted work of **Brendigo LTD** and is distributed under the repository's custom proprietary commercial [`LICENSE`](../LICENSE).

Independent third-party components retain their own licenses; see [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md).
