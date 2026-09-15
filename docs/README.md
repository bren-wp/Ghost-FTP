# Ghost FTP documentation

<p align="center"><img src="../build/icon.png" alt="Ghost FTP" width="118"></p>
<p align="center"><strong>Product behavior, platform boundaries, privacy, security and release engineering for Ghost FTP.</strong></p>

## Current status

- Current source version: **0.0.6**
- Latest published GitHub Release: **0.0.6** (`ghostftp-v0.0.6`, published 14 September 2026)
- Release channel: **Current**
- 0.0.6 shape: **13 platform artifacts / 16 public files**
- Public release targets: **Windows, Linux, Android and browser extension packages**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Browser packages: **Chrome, Edge, Firefox and Opera**
- Desktop languages: **24 selectable local languages**, English default/fallback
- License: **proprietary commercial software, Brendigo LTD**

The published 0.0.6 release is immutable release history. The `production/0.0.7-cleanup` branch prepares the product for the next cycle without rewriting, retagging or republishing 0.0.6.

The repository no longer contains the former `web/` website or Web FTP application. Documentation and tests must not retain live references to those removed source surfaces.

## Documentation principles

1. **Describe what actually exists.** Unsupported capabilities are not represented as working features.
2. **Security-sensitive behavior fails closed.** Production signing and server identity checks are not weakened to pass CI.
3. **Privacy is architectural.** No telemetry, behavioral analytics, advertising, fingerprinting or hidden credential relay is part of the product.
4. **Visual evidence must be authentic.** Repository-local screenshots are bound to an exact source SHA; generated mockups are not runtime proof.
5. **Platform differences remain explicit.** Browser sandbox limitations, Android protocol boundaries and macOS distribution requirements are documented rather than simulated away.

## Start here

| Goal | Document |
| --- | --- |
| Install or verify Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
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
| Dependencies | [`DEPENDENCIES.md`](DEPENDENCIES.md) |
| Testing and quality gates | [`TESTING.md`](TESTING.md) |
| UI screenshots/evidence | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Support | [`SUPPORT.md`](SUPPORT.md) |
| Linux | [`../linux/README.md`](../linux/README.md) |
| Android | [`../android/README.md`](../android/README.md) |
| macOS native source | [`../macos/README.md`](../macos/README.md) |
| Browser extensions | [`../extensions/README.md`](../extensions/README.md) |
| Engineering audit prompt | [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md) |

## Published 0.0.6 artifact contract

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

## Authentic visual reference

The following images are repository-local, exact-head evidence from Windows, Linux and Android. They are not mockups. The read-only verified cross-platform evidence bundle records capture source `9adace20030a300c39eb320a97482eb50dfdb9d8` and workflow run `34863585111`.

<table>
<tr><td width="50%"><strong>Windows workspace</strong><br><img src="images/0.0.6/ghost-ftp-main-workspace.png" alt="Windows workspace"></td><td width="50%"><strong>Windows Site Manager</strong><br><img src="images/0.0.6/ghost-ftp-site-manager.png" alt="Windows Site Manager"></td></tr>
<tr><td width="50%"><strong>Linux workspace</strong><br><img src="images/0.0.6/ghost-ftp-linux-main-workspace.png" alt="Linux workspace"></td><td width="50%"><strong>Linux Settings</strong><br><img src="images/0.0.6/ghost-ftp-linux-settings.png" alt="Linux Settings"></td></tr>
<tr><td width="50%"><strong>Android Files</strong><br><img src="images/0.0.6/ghost-ftp-android-files.png" alt="Android Files"></td><td width="50%"><strong>Android Transfers</strong><br><img src="images/0.0.6/ghost-ftp-android-transfers.png" alt="Android Transfers"></td></tr>
</table>

Authentic runtime evidence is maintained across **Windows, Linux and Android** and remains bound to an exact-head source identity. Generated mockups are never accepted as execution or release evidence.

## Commercial proprietary license

Ghost FTP is proprietary commercial software. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand or white-label the project. Ghost FTP is a copyrighted work of **Brendigo LTD** and is distributed under the repository's [`LICENSE`](../LICENSE).
