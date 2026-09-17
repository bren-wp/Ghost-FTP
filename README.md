# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="148">
</p>

<h3 align="center">Your servers. Your files. Your control.</h3>

<p align="center">
A privacy-first FTP, FTPS and SFTP workspace for direct professional file transfer without telemetry, advertising, mandatory product accounts or a hidden storage cloud.
</p>

<p align="center">
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

---

## Product scope

Ghost FTP connects native applications directly to infrastructure selected by the user. The maintained repository product surfaces are Windows, Linux, Android, macOS source, and local browser helper packages.

The retired website and Web FTP implementation are intentionally not part of this repository or product runtime.

Ghost FTP contains no application telemetry, behavioral analytics, advertising, fingerprinting, hidden synchronization, mandatory Ghost FTP account, automatic crash-upload backend, or Ghost FTP storage cloud.

## Security model

- FTP is available as an intentional unencrypted compatibility protocol.
- Explicit FTPS validates certificate trust and server hostname identity.
- Desktop SFTP uses strict SSH host-key trust and pinning.
- Android exposes FTP and strict explicit FTPS; SFTP remains hidden until strict host-key verification is maintained on Android.
- The canonical production publication path fails closed when its required signing identities are unavailable.
- Native transfer traffic goes to the server selected by the user.

## File and transfer operations

Ghost FTP provides real file-management and transfer operations rather than simulated controls:

- local and remote navigation
- filtering and search
- directory comparison
- bookmarks
- upload and download
- transfer queue lifecycle, retry, cancellation and priority
- Remote Edit
- create directory
- rename
- delete
- CHMOD where supported

## Supported product surfaces

| Platform | Status | Distribution / boundary |
| --- | --- | --- |
| **Windows** | Current release target | Universal Setup and Portable applications with x64, x86 and ARM64 payloads |
| **Linux** | Current release target | Debian, Ubuntu and Fedora Installer + Portable bundles |
| **Android** | Current release target | APK with FTP and strict explicit FTPS |
| **macOS** | Maintained native source | Public package is withheld until Developer ID Application signing and Apple notarization are verified |
| **Browser helpers** | Current release target | Chrome, Edge, Firefox and Opera local helper packages |

Browser helpers have zero browser permissions and zero host permissions. Windows installed builds support a **sanitized browser-to-desktop handoff** through the registered `ghostftp:` protocol. Only protocol, host, optional port, optional username and remote path are handed to the desktop app; passwords, private-key passphrases, private keys, query data and fragments are excluded. The helper never performs FTP/FTPS/SFTP transport itself and never auto-connects.

---

## Runtime evidence

These repository-local screenshots document maintained Windows, Linux and Android application surfaces. They are runtime evidence captured by repository workflows, not generated product mockups.

### Windows

![Ghost FTP 0.0.7 Windows main workspace](docs/images/ghost-ftp-main-workspace.png)

### Linux

![Ghost FTP 0.0.7 Linux main workspace](docs/images/ghost-ftp-linux-main-workspace.png)

### Android

![Ghost FTP 0.0.7 Android files](docs/images/ghost-ftp-android-files.png)

See [Reference UI](docs/REFERENCE-UI.md) for the complete runtime-evidence contract.

---

## Ghost FTP 0.0.7

Current source version: **0.0.7**  
Release channel: **Current**  
Product status: **Current**  
Last actually published GitHub Release: **0.0.7**  
Public release tag: **ghostftp-v0.0.7**  
Release source SHA: **f7dc8a6947279dc773fe58e9ff264c1aa25bf477**  
Prerelease: **false**

The published 0.0.7 release contains **13 platform artifacts / 16 public files**.

> `main` may contain post-release hardening newer than the 0.0.7 release source SHA. Those changes do not retroactively modify the already-published 0.0.7 binaries.

### Windows

```text
Ghost-FTP-0.0.7-Setup.exe
Ghost-FTP-0.0.7-Portable.exe
```

Both public Windows executables carry x64, x86 and ARM64 payloads internally.

### Linux

```text
Ghost-FTP-0.0.7-Linux-Debian-Installer.run
Ghost-FTP-0.0.7-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.7-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.7-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.7-Linux-Fedora-Installer.run
Ghost-FTP-0.0.7-Linux-Fedora-Portable.tar.gz
```

Each distro bundle carries amd64, arm64 and i386 payloads and selects the local architecture at runtime.

### Android

```text
Ghost-FTP-0.0.7-Android.apk
```

The public 0.0.7 release contains the Android APK and its release manifest records its SHA-256 digest alongside the other published assets.

### Browser helpers

```text
Ghost-FTP-0.0.7-Chrome-Extension.zip
Ghost-FTP-0.0.7-Edge-Extension.zip
Ghost-FTP-0.0.7-Firefox-Extension.zip
Ghost-FTP-0.0.7-Opera-Extension.zip
```

The four packages use one shared local runtime with browser-specific manifests. On supported Windows installs, the explicit **Open in Ghost FTP** action uses the sanitized `ghostftp://connect` handoff described above; credentials never enter that launch URL.

---

## Release integrity

Ghost FTP publication is bound to source identity: a published binary must be traceable to the source SHA from which its release artifacts were assembled.

The **canonical production-signed workflow** remains the stricter publication path. It requires the configured trusted signing identities and fails closed when those identities are unavailable.

The already-published **0.0.7 compatibility release** is a separate release path. Its public tag targets the immutable source SHA shown above, its published asset set is explicit, and release metadata plus `SHA256.txt` provide readback-verifiable integrity for the distributed files.

Canonical release identity for 0.0.7:

```text
VERSION=0.0.7
TAG=ghostftp-v0.0.7
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=13
PUBLIC_RELEASE_FILES=16
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
RELEASE_SOURCE_SHA=f7dc8a6947279dc773fe58e9ff264c1aa25bf477
```

Release metadata files:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

The verified release directory may additionally be represented as an OCI distribution bundle. That object is a distribution artifact, not a runtime product backend.

---

## Privacy

Ghost FTP does not require a Ghost FTP product account and does not contain application telemetry, behavioral analytics, advertising, fingerprinting, hidden synchronization or automatic crash uploading.

Saved credentials are opt-in and platform-local. See [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md) for platform-specific boundaries.

---

## Quality gates

Release-candidate checks include Go formatting/tests/vet, repository/security/privacy/localization/documentation audits, Python regression tests, Windows universal packaging, Linux bundle generation and lifecycle tests, Android build/lint/signing checks, deterministic browser packages, macOS source builds, CodeQL and Govulncheck.

A pull request is merge-ready only when workflows for its exact final head SHA reach terminal success.

---

## Localization

Ghost FTP maintains 24 selectable desktop languages with English as the canonical default and fallback: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.

---

## Commercial proprietary software

Ghost FTP is proprietary commercial software. It is not open-source software and is not distributed under an OSI-approved license.

Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand, white-label or incorporate Ghost FTP into another product.

Ghost FTP and its source code, binaries, user interfaces, documentation, branding, icons and release engineering are copyrighted works of **Brendigo LTD**.

Copyright © 2026 Brendigo LTD. All rights reserved.

See [`LICENSE`](LICENSE) and [`docs/THIRD-PARTY-NOTICES.md`](docs/THIRD-PARTY-NOTICES.md).

---

## Documentation

- [Documentation overview](docs/README.md)
- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Signing](docs/SIGNING.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Testing](docs/TESTING.md)
- [Reference UI](docs/REFERENCE-UI.md)
- [Roadmap](docs/ROADMAP.md)

<p align="center">
<strong>Ghost FTP</strong><br>
Direct file transfer. Clear security boundaries. No cloud middleman.
</p>
