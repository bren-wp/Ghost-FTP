# Ghost FTP documentation

<p align="center"><img src="../build/icon.png" alt="Ghost FTP" width="118"></p>

Ghost FTP documentation covers the maintained native applications, browser helper packages, privacy and security boundaries, packaging, signing, testing and release engineering.

## Current status

- Current source version: **0.0.6**
- Release channel: **Current**
- Development status: **Active**
- Last actually published GitHub Release: **0.0.5**
- Next public release target: `ghostftp-v0.0.6`
- Public release shape: **13 platform artifacts / 16 public files**
- Public release targets: **Windows, Linux, Android and browser helper packages**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Browser packages: **Chrome, Edge, Firefox and Opera**
- Desktop languages: **24 selectable local languages**, with English as default/fallback
- Product website: **https://ghostftp.com**
- License: **proprietary commercial software, Brendigo LTD**

The former repository-hosted website and browser protocol client have been retired. The repository no longer contains a `web/` application surface; product transfer functionality is maintained in the native applications and explicitly scoped browser helpers.

## Documentation principles

1. **Describe only maintained behavior.** A platform, feature or release is not called production-ready without matching source and verification evidence.
2. **Keep security boundaries explicit.** FTP is compatibility transport, FTPS validates TLS identity, desktop SFTP keeps strict host-key trust, and Android SFTP stays hidden until an equivalent maintained implementation exists.
3. **Keep privacy claims testable.** No application telemetry, advertising, behavioral analytics, hidden crash upload or mandatory product account is part of the maintained application architecture.
4. **Separate development evidence from production evidence.** CI smoke identities and ad-hoc signing are not public publisher identities.
5. **Keep user documentation aligned with the source tree.** Retired application surfaces, paths and workflows are removed instead of left as stale guidance.

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
| Navigation and bookmarks | [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md) |
| Queue priority | [`QUEUE-PRIORITY.md`](QUEUE-PRIORITY.md) |
| Settings | [`SETTINGS.md`](SETTINGS.md) |
| Support | [`SUPPORT.md`](SUPPORT.md) |
| Linux | [`../linux/README.md`](../linux/README.md) |
| Android | [`../android/README.md`](../android/README.md) |
| macOS development | [`../macos/README.md`](../macos/README.md) |
| Browser helpers | [`../extensions/README.md`](../extensions/README.md) |
| Engineering audit prompt | [`prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md`](prompts/GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md) |

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

Windows publishes two architecture-independent user-facing executables containing x64/x86/ARM64 payloads. Linux publishes Installer and Portable bundles for Debian, Ubuntu and Fedora, each carrying amd64/arm64/i386 payloads. Android publishes one production-signed APK. Browser helpers publish deterministic packages for Chrome, Edge, Firefox and Opera.

## Authentic visual reference

Repository-local runtime screenshots are maintained for Windows, Linux and Android and are tied to source/release evidence. Generated mockups are not accepted as proof that an application runs.

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for the maintained evidence contract.

## Commercial proprietary license

Ghost FTP is not open-source software. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand or white-label the project. Ghost FTP is a copyrighted work of **Brendigo LTD** and is distributed under the repository's custom proprietary commercial [`LICENSE`](../LICENSE).
