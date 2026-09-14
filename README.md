# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="150">
</p>

<p align="center"><strong>Your servers. Your files. No cloud middleman.</strong></p>

<p align="center">
A privacy-first desktop and mobile file-transfer workspace for people who want direct control over FTP, FTPS and SFTP workflows without telemetry, advertising or a mandatory Ghost FTP account.
</p>

<p align="center">
  <a href="https://ghostftp.com"><strong>Website</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

---

## Ghost FTP 0.0.6

- Current source version: **0.0.6**
- Release channel: **Current**
- Development status: **Active**
- Last actually published GitHub Release: **0.0.5**
- Next public release target: **ghostftp-v0.0.6**, `prerelease=false`
- Public 0.0.6 target: **13 platform artifacts / 16 public files**
- License: **proprietary commercial software — Copyright © 2026 Brendigo LTD. All rights reserved.**

0.0.6 is published only after the exact merged `main` SHA passes the complete release gate and the protected Windows and Android production-signing identities succeed. Development, CI-smoke, self-signed or ad-hoc identities are never substituted for production signing.

## Why Ghost FTP

- **Direct transfer model.** Ghost FTP connects to infrastructure you choose; it does not require a Ghost FTP storage cloud.
- **Privacy by design.** No application telemetry, advertising, behavioral tracking, fingerprinting or automatic crash upload.
- **Security boundaries that fail closed.** Explicit FTPS validates certificate/hostname identity; maintained desktop SFTP uses strict host-key trust/pinning.
- **Real transfer controls.** Queue lifecycle, retry, cancellation, pause/resume where technically supported, priority ordering and truthful progress reporting.
- **Native workflows.** Windows is the reference desktop experience; Linux, Android and macOS remain active maintained surfaces with capability claims tied to actual evidence.
- **24 desktop languages.** English is the canonical default/fallback and the maintained desktop catalog exposes 24 selectable languages.

## Ghost FTP 0.0.6 product preview

![Ghost FTP 0.0.6 Windows main workspace](docs/images/0.0.6/ghost-ftp-main-workspace.png)

<table>
<tr>
<td width="50%" valign="top"><strong>Windows Site Manager</strong><br><br><img src="docs/images/0.0.6/ghost-ftp-site-manager.png" alt="Ghost FTP 0.0.6 Windows Site Manager"></td>
<td width="50%" valign="top"><strong>Windows Settings</strong><br><br><img src="docs/images/0.0.6/ghost-ftp-settings.png" alt="Ghost FTP 0.0.6 Windows Settings"></td>
</tr>
<tr>
<td width="50%" valign="top"><strong>Linux workspace</strong><br><br><img src="docs/images/0.0.6/ghost-ftp-linux-main-workspace.png" alt="Ghost FTP 0.0.6 Linux main workspace"></td>
<td width="50%" valign="top"><strong>Android Files</strong><br><br><img src="docs/images/0.0.6/ghost-ftp-android-files.png" alt="Ghost FTP 0.0.6 Android Files"></td>
</tr>
<tr>
<td colspan="2" align="center"><strong>Windows product identity</strong><br><br><img src="docs/images/0.0.6/ghost-ftp-about.png" alt="Ghost FTP 0.0.6 About" width="620"></td>
</tr>
</table>

These **repository-local** screenshots are authentic exact-head runtime captures for Windows, Linux and Android on the 0.0.6 source line, imported byte-for-byte from the verified `ghostftp-authentic-ui-verified-bundle` produced by workflow run `34863585111` for source SHA `9adace20030a300c39eb320a97482eb50dfdb9d8`. The repository stores the original provenance manifest and SHA-256 allow-list under [`docs/images/0.0.6/`](docs/images/0.0.6/). Generated mockups, image-generation output and manually substituted screenshots are not accepted as runtime evidence. See [Reference UI](docs/REFERENCE-UI.md).

## Platform status

| Platform | 0.0.6 status | Distribution contract |
| --- | --- | --- |
| **Windows** | Public release target / reference UI | One universal Setup + one universal Portable, each containing x64, x86 and ARM64 payloads. Trusted Authenticode required for publication. |
| **Linux** | Active development, public release target | Debian, Ubuntu and Fedora each receive one Installer + one Portable bundle; each carries amd64, arm64 and i386 payloads. |
| **Android** | Active development, public release target | One production-signed APK. FTP + strict explicit FTPS. SAF-scoped local storage. SFTP remains hidden until strict host-key verification exists. |
| **macOS** | Active development | Native AppKit frontend. Public distribution remains blocked until real Developer ID Application signing + Apple notarization succeed. |
| **Browser helper** | Public release target | Separate deterministic ZIP for Chrome, Edge, Firefox and Opera; zero browser/host permissions and no hidden desktop handoff. |

## 0.0.6 downloads

The following filenames are the **canonical 0.0.6 publication contract**. They are not considered publicly released until `ghostftp-v0.0.6` is actually published by the protected release workflow.

### Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Both files contain native x64, x86 and ARM64 application payloads. Architecture-specific public EXEs are intentionally not emitted.

### Linux

```text
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
```

Each distro bundle contains amd64, arm64 and i386 payloads and performs local architecture selection. Native installer/runtime/GUI evidence is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. ARM64/i386 are not described as natively executed unless that evidence exists.

### Android

```text
Ghost-FTP-0.0.6-Android.apk
```

There is one public APK. Production publication requires `apksigner` verification and an exact protected `GHOSTFTP_ANDROID_CERT_SHA256` certificate fingerprint match. Android compatibility follows the maintained `minSdk`/target SDK and tested APIs; no responsible Android application can claim literal compatibility with every Android version ever released.

### Browser extensions

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

Source lives under [`extensions/`](extensions/README.md), with one browser-specific manifest directory per browser and a shared local-only runtime. The official helper has **no supported browser-to-desktop** URI/native-messaging handoff, no telemetry backend and no network/host permissions.

### Release metadata

```text
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
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The verified release directory is additionally packaged as `ghcr.io/bren-wp/ghost-ftp:0.0.6`. That OCI object is a **distribution bundle**, not a supported runtime container.

## Protocol and trust model

**FTP** is available only as an intentional unencrypted compatibility choice. **Explicit FTPS** validates TLS certificate and hostname identity and never silently downgrades to FTP. **SFTP** is maintained on desktop platforms with strict host-key verification/pinning.

Android SFTP is deliberately hidden in 0.0.6 until a maintained implementation can prove strict host-key identity handling and fail closed on first-contact uncertainty or mismatch. A production APK signature does not change this boundary.

## Privacy

Ghost FTP does not require an account and does not include application telemetry, analytics, advertising, fingerprinting, hidden synchronization or a Ghost FTP transfer relay. Saved credentials require explicit persistence consent and are handled by platform-appropriate protected local storage.

Read [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md) for the exact maintained contract.

## Localization

The maintained desktop catalog exposes **24 selectable desktop languages** with English as the canonical fallback:

English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.

All user-facing strings must remain catalog-backed. A platform is not described as fully localized unless its own maintained UI surface passes the localization audit.

## Build and quality gates

Core release-candidate checks include:

```text
gofmt
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_dependencies.py
python scripts/audit_version.py
python scripts/audit_localization.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Platform workflows add Windows universal-build/signing verification, Linux universal-bundle and distro install/GUI/uninstall verification, Android build/lint/signing-pipeline verification, browser deterministic-package verification, macOS development build checks, CodeQL and Govulncheck.

A PR is merge-ready only when workflows for its **exact final head SHA** are terminal-successful. A release is valid only when the exact merged `main` SHA passes post-merge verification and the production release workflow succeeds.

## Commercial proprietary license

Ghost FTP is **not open-source software** and is not offered under an OSI-approved license. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand, white-label or incorporate Ghost FTP into another product.

Ghost FTP and its source code, binaries, UI, documentation, branding, icons, release engineering and related materials are copyrighted works of **Brendigo LTD**. See [`LICENSE`](LICENSE) for the controlling terms and [`docs/THIRD-PARTY-NOTICES.md`](docs/THIRD-PARTY-NOTICES.md) for independent third-party components.

## Documentation

Start with [`docs/README.md`](docs/README.md). Important references include:

- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Signing](docs/SIGNING.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Testing](docs/TESTING.md)
- [Roadmap](docs/ROADMAP.md)

<p align="center"><strong>Ghost FTP — direct file transfer with verifiable boundaries.</strong></p>
