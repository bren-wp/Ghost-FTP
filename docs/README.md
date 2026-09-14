# Ghost FTP documentation

<p align="center"><img src="../build/icon.png" alt="Ghost FTP" width="118"></p>
<p align="center"><strong>Product behavior, platform boundaries, Web FTP, privacy, security and release engineering for Ghost FTP.</strong></p>

## Current status

- Current source version: **0.0.6**
- Release channel: **Current**
- Development status: **Active**
- Last actually published GitHub Release: **0.0.5**
- 0.0.6 release target: `ghostftp-v0.0.6`, `PRERELEASE=false`
- 0.0.6 shape: **13 platform artifacts / 16 public files**
- Public release targets: **Windows, Linux, Android and browser helper packages**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Active web surfaces: **`web/` product site + `web/ftp` browser client**
- Browser packages: **Chrome, Edge, Firefox and Opera**
- Desktop languages: **24 selectable local languages**, English default/fallback
- Product website: **https://ghostftp.com**
- License: **proprietary commercial software, Brendigo LTD**

0.0.6 is not treated as published until the exact verified `main` SHA succeeds through the protected Windows/Android signing and release-readback transaction. macOS remains development/source-only until real Developer ID Application signing and Apple notarization succeed.

## Documentation principles

1. **Describe what actually exists.** No platform, protocol, architecture or release is called production-ready without evidence.
2. **Security boundaries fail closed.** Official Windows publication requires trusted Authenticode; Android requires the protected publisher identity and exact `GHOSTFTP_ANDROID_CERT_SHA256` fingerprint.
3. **Development evidence is not production evidence.** CI smoke keys, self-signed certificates and ad-hoc macOS signing never substitute for production identities.
4. **Visual evidence must be authentic.** Repository-local screenshots may document the product, but only exact-head runtime captures count as execution evidence. Generated UI mockups never count as proof that an app runs.
5. **Privacy claims are scoped.** Native Ghost FTP has no telemetry or hidden transfer relay. Web FTP is explicitly server-assisted because browsers cannot open raw FTP/FTPS/SFTP sockets.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Product architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Platform capability boundaries | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Website and Web FTP | [`WEB.md`](WEB.md) |
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
| Website source | [`../web/README.md`](../web/README.md) |
| Web FTP source | [`../web/ftp/README.md`](../web/ftp/README.md) |
| Engineering audit prompt | [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md) |
| Website design/production prompt | [`prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md`](prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md) |
| Full Web FTP implementation prompt | [`prompts/GHOST-FTP-WEB-APP-PROMPT.md`](prompts/GHOST-FTP-WEB-APP-PROMPT.md) |

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

`web/` and `web/ftp` are active source/deployment surfaces, not additional GitHub Release artifacts, so they do not change the 13/16 binary release count.

## Authentic visual reference

<table>
<tr><td width="50%"><strong>Windows workspace</strong><br><img src="images/0.0.6/ghost-ftp-main-workspace.png" alt="Windows workspace"></td><td width="50%"><strong>Windows Site Manager</strong><br><img src="images/0.0.6/ghost-ftp-site-manager.png" alt="Windows Site Manager"></td></tr>
<tr><td width="50%"><strong>Linux workspace</strong><br><img src="images/0.0.6/ghost-ftp-linux-main-workspace.png" alt="Linux workspace"></td><td width="50%"><strong>Linux Settings</strong><br><img src="images/0.0.6/ghost-ftp-linux-settings.png" alt="Linux Settings"></td></tr>
<tr><td width="50%"><strong>Android Files</strong><br><img src="images/0.0.6/ghost-ftp-android-files.png" alt="Android Files"></td><td width="50%"><strong>Android Transfers</strong><br><img src="images/0.0.6/ghost-ftp-android-transfers.png" alt="Android Transfers"></td></tr>
</table>

Authentic runtime evidence is maintained across **Windows, Linux and Android** and must remain bound to an exact-head source SHA. The **read-only verified cross-platform evidence bundle** is the execution-evidence source for release review. Generated mockups are never accepted as execution or release evidence.

## Commercial proprietary license

Ghost FTP is not open-source software. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand or white-label the project. Ghost FTP is a copyrighted work of **Brendigo LTD** and is distributed under the repository's custom proprietary commercial [`LICENSE`](../LICENSE).
