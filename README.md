# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="148">
</p>

<h3 align="center">Your servers. Your files. Your control.</h3>

<p align="center">
A privacy-first FTP, FTPS and SFTP workspace built for direct, professional file transfer — without telemetry, advertising, mandatory product accounts or a hidden storage cloud.
</p>

<p align="center">
  <a href="https://ghostftp.com"><strong>Website</strong></a> ·
  <a href="https://ghostftp.com/ftp/"><strong>Web FTP</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

---

## File transfer without the cloud in the middle

Ghost FTP is designed for people who want a capable file-transfer workspace without turning their server credentials, files or activity into somebody else's product.

Connect to the infrastructure you choose.

Manage files using real transfer and remote-file operations.

Keep native transfer traffic between your device and the server you selected.

No Ghost FTP storage cloud.
No advertising.
No behavioral analytics.
No mandatory Ghost FTP account.
No automatic crash-upload backend.

**Just file transfer with clear security and privacy boundaries.**

---

## Why Ghost FTP?

### Direct server workflows

Ghost FTP connects to infrastructure you control or choose rather than requiring files to pass through a proprietary storage service.

Native applications communicate with the destination server selected by the user.

### Privacy by design

The application is designed without:

* application telemetry
* behavioral analytics
* advertising
* fingerprinting
* hidden synchronization
* automatic crash uploads
* mandatory product accounts

Your file-transfer client should not become another data-collection platform.

### Security that fails closed

Security-sensitive protocols are handled explicitly rather than hidden behind reassuring UI labels.

* Explicit FTPS validates TLS certificates and hostname identity.
* Desktop SFTP uses strict host-key trust and pinning.
* Web SFTP requires an expected SHA-256 server host-key fingerprint before password authentication.
* Production releases fail closed when required signing identities are unavailable.

### Real transfer controls

Ghost FTP is built around actual transfer operations, including:

* transfer queue lifecycle
* cancellation
* retry
* pause and resume where technically supported
* priority ordering
* truthful transfer progress

### Real file management

The interface is backed by actual engine capability rather than decorative controls.

Available workflows include:

* local and remote navigation
* filtering
* search
* directory comparison
* bookmarks
* remote file operations
* Remote Edit
* upload and download
* directory creation
* rename
* delete
* CHMOD where supported

---

# One product. Multiple platforms.

Ghost FTP is being developed as a cross-platform file-transfer workspace with platform capabilities documented honestly rather than advertised as identical where they are not.

| Platform            | Status                                       | What you get                                                                                      |
| ------------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **Windows**         | Public release target / reference experience | Universal Setup and Portable applications with x64, x86 and ARM64 payloads                        |
| **Linux**           | Active / public release target               | Debian, Ubuntu and Fedora Installer + Portable packages                                           |
| **Android**         | Active / public release target               | Production-signed APK with FTP and strict explicit FTPS                                           |
| **macOS**           | Active development                           | Native AppKit frontend; production package waits for proven Developer ID signing and notarization |
| **Browser helpers** | Public release target                        | Chrome, Edge, Firefox and Opera packages                                                          |
| **Website**         | Active                                       | Self-contained Ghost FTP product website                                                          |
| **Web FTP**         | Active                                       | Browser workspace for FTP, FTPS and SFTP with explicit server-side trust boundaries               |

---

# See Ghost FTP in action

The repository includes authentic exact-head runtime captures for the maintained Windows, Linux and Android application surfaces.

These screenshots are repository-local runtime evidence — not generated product mockups.

## Windows

![Ghost FTP 0.0.6 Windows main workspace](docs/images/0.0.6/ghost-ftp-main-workspace.png)

<table>
<tr>
<td width="50%" valign="top">
<strong>Site Manager</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager">
</td>

<td width="50%" valign="top">
<strong>Settings</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-settings.png" alt="Ghost FTP Settings">
</td>
</tr>

<tr>
<td width="50%" valign="top">
<strong>Bookmarks</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-bookmarks.png" alt="Ghost FTP Bookmarks">
</td>

<td width="50%" valign="top">
<strong>Product identity</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-about.png" alt="Ghost FTP About">
</td>
</tr>
</table>

## Linux

<table>
<tr>
<td width="50%" valign="top">
<strong>Main workspace</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-linux-main-workspace.png" alt="Ghost FTP Linux main workspace">
</td>

<td width="50%" valign="top">
<strong>Bookmarks</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-linux-bookmarks.png" alt="Ghost FTP Linux Bookmarks">
</td>
</tr>

<tr>
<td colspan="2" align="center">
<strong>Settings</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-linux-settings.png" alt="Ghost FTP Linux Settings" width="820">
</td>
</tr>
</table>

## Android

<table>
<tr>
<td width="33%" valign="top">
<strong>Files</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-files.png" alt="Ghost FTP Android Files">
</td>

<td width="33%" valign="top">
<strong>Sites</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-sites.png" alt="Ghost FTP Android Sites">
</td>

<td width="33%" valign="top">
<strong>Transfers</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-transfers.png" alt="Ghost FTP Android Transfers">
</td>
</tr>

<tr>
<td width="33%" valign="top">
<strong>Bookmarks</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-bookmarks.png" alt="Ghost FTP Android Bookmarks">
</td>

<td width="33%" valign="top">
<strong>Settings</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-settings.png" alt="Ghost FTP Android Settings">
</td>

<td width="33%" valign="top">
<strong>About</strong><br><br>
<img src="docs/images/0.0.6/ghost-ftp-android-about.png" alt="Ghost FTP Android About">
</td>
</tr>
</table>

See [Reference UI](docs/REFERENCE-UI.md) for the complete 15-image evidence contract.

---

# Desktop file transfer without unnecessary distractions

Ghost FTP focuses on the workflow that matters:

**connect → navigate → transfer → manage → disconnect**

The product is built for users who want their file-transfer application to behave like an infrastructure tool rather than an advertising platform, cloud drive or account ecosystem.

Whether you're maintaining a website, working with hosting infrastructure, managing remote servers or moving files between systems, Ghost FTP keeps the workflow centered on the destination you selected.

---

# Web FTP

## File management from the browser

Ghost FTP also includes a responsive Web FTP workspace for:

* FTP
* explicit FTPS
* SFTP
* remote directory navigation
* upload
* download
* create directory
* rename
* delete
* CHMOD
* bounded Remote Edit

No Ghost FTP user registration database is required.

### An honest browser security model

Web FTP does **not** pretend that browsers can directly open raw FTP, FTPS or SFTP sockets.

They cannot.

Instead, Ghost FTP uses an explicit server-assisted model:

1. Connection values remain in browser page memory rather than a persistent Ghost FTP account database.
2. The browser sends the requested operation over HTTPS to `web/ftp/api.php`.
3. The Ghost FTP web host performs that operation.
4. The transport is closed after the operation.
5. The supplied application does not persist connection passwords or transfer history.
6. FTP and FTPS destinations are checked against SSRF protections.
7. Explicit FTPS requires TLS peer and hostname verification.
8. SFTP requires the expected SHA-256 host-key fingerprint before authentication.

A public deployment should keep private and reserved destination blocking enabled and run behind HTTPS.

See [Web / Web FTP documentation](docs/WEB.md) for the complete trust model.

---

# Security without vague promises

Ghost FTP documents its security model at the protocol and platform level.

## FTP

Traditional FTP is supported as an intentional **unencrypted compatibility mode**.

It is not presented as secure transport.

## Explicit FTPS

Explicit FTPS validates:

* TLS certificate trust
* server hostname identity

Silent downgrade is not accepted.

## Desktop SFTP

Desktop SFTP uses strict server host-key trust and pinning.

## Android

Android currently exposes FTP and strict explicit FTPS.

SFTP remains hidden until a maintained strict host-key verification implementation exists.

## Web SFTP

Web SFTP requires the expected:

```text
SHA256:...
```

server host-key fingerprint **before password authentication**.

Read the complete [Security documentation](docs/SECURITY.md).

---

# Privacy that is part of the architecture

Ghost FTP does not require a Ghost FTP product account.

The application does not contain:

* application telemetry
* behavioral analytics
* advertising
* fingerprinting
* hidden synchronization
* automatic crash uploading

For native applications, transfer traffic goes to the server selected by the user.

## Web FTP privacy boundary

Web FTP necessarily has a different runtime boundary.

Because browser JavaScript cannot communicate directly over raw FTP, FTPS or SFTP, the trusted Ghost FTP web host executes each requested server operation.

The supplied application is designed to avoid durable storage of:

* connection passwords
* connection profiles
* transfer history

The web hosting operator and its infrastructure nevertheless remain part of the Web FTP trust boundary.

That distinction is intentional and documented instead of being hidden behind a misleading “direct connection” claim.

Read the complete [Privacy documentation](docs/PRIVACY.md).

---

# 24 desktop languages

Ghost FTP currently maintains **24 selectable desktop languages**.

English is the canonical default and fallback.

Supported desktop catalogs:

* English
* Croatian
* German
* French
* Spanish
* Turkish
* Greek
* Portuguese
* Chinese
* Russian
* Hindi
* Japanese
* Italian
* Polish
* Dutch
* Czech
* Ukrainian
* Swedish
* Romanian
* Hungarian
* Danish
* Finnish
* Norwegian
* Korean

Localization claims remain platform-specific.

A platform is described as fully localized only when its maintained UI and catalog coverage pass the corresponding audits.

---

# Ghost FTP 0.0.6

**Current source version:** `0.0.6`
**Release channel:** Current
**Development status:** Active
**Last actually published GitHub Release:** `0.0.5`
**Next public release target:** `ghostftp-v0.0.6`
**Prerelease:** `false`

The planned Ghost FTP 0.0.6 publication contract contains:

**13 platform artifacts / 16 public files**

---

# Downloads planned for 0.0.6

## Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Both packages carry native:

```text
x64
x86
ARM64
```

payloads internally.

Separate architecture-specific public Windows executables are intentionally not part of the release contract.

---

## Linux

### Debian

```text
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
```

### Ubuntu

```text
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
```

### Fedora

```text
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
```

Each bundle contains:

```text
amd64
arm64
i386
```

payloads and selects the local architecture at runtime.

Native installer, runtime and GUI evidence is maintained on:

* Debian 13 amd64
* Ubuntu 26.04 LTS amd64
* Fedora 44 x86_64

Cross-built ARM64 and i386 artifacts are not mislabeled as native execution evidence.

---

## Android

```text
Ghost-FTP-0.0.6-Android.apk
```

Publication requires:

* `apksigner` verification
* exact protected `GHOSTFTP_ANDROID_CERT_SHA256` signer fingerprint match

Android compatibility follows the maintained `minSdk`, target SDK and tested APIs rather than claiming compatibility with every Android release ever produced.

---

## Browser helpers

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

All four packages use a shared local runtime with browser-specific manifests.

They require zero browser or host permissions and do not advertise unsupported browser-to-desktop handoff functionality.

Source:

[`extensions/`](extensions/README.md)

---

# Release integrity

Ghost FTP release publication is designed around a simple principle:

**the binary being published must correspond to the exact source that passed verification.**

Version 0.0.6 is published only after:

* the exact final head SHA is merged to `main`
* the complete post-merge gate succeeds
* trusted Windows Authenticode signing succeeds
* the protected Android signing identity matches `GHOSTFTP_ANDROID_CERT_SHA256`
* release assets are published successfully
* published assets pass readback verification without drift

Development, CI-smoke, self-signed or ad-hoc identities are never silently substituted for production signing.

Canonical release identity:

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

Release metadata files:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

The verified release directory is additionally represented as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

This OCI object is a **distribution bundle**, not a supported runtime container.

---

# Tested before publication

Ghost FTP uses automated quality gates across code quality, security, privacy, localization, platform contracts and release integrity.

Release-candidate checks include:

```text
gofmt
go test -race ./...
go vet ./...

python scripts/audit_brand_hardcut.py
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_desktop_surface.py
python scripts/audit_dependencies.py
python scripts/audit_version.py
python scripts/audit_localization.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py

python -m unittest discover -s scripts -p 'test_*.py'
```

The regression suite also covers the Web FTP contract.

When PHP CLI is available on the maintained Linux runner, the pipeline additionally:

* lints every tracked Web FTP PHP source file
* checks SSRF protection markers
* checks TLS trust markers
* checks SFTP trust markers

Platform workflows additionally verify:

* Windows universal packaging
* Windows signing smoke
* Linux bundle generation
* Linux install lifecycle
* Android build and lint
* Android signing pipeline
* deterministic browser packages
* macOS development builds
* CodeQL
* Govulncheck

A pull request is considered merge-ready only when workflows for its **exact final head SHA** reach terminal success.

A production release is valid only when the exact merged `main` SHA passes post-merge verification and the protected production workflow completes publication and readback.

---

# Built around verifiable boundaries

Ghost FTP does not try to make every platform sound identical.

Instead, capability claims follow evidence.

That means:

* unsupported features remain hidden rather than simulated
* security-sensitive requirements fail closed
* development signing is not presented as production signing
* cross-built binaries are not presented as native execution evidence
* Web FTP does not claim impossible browser-direct FTP/SFTP connectivity
* platform differences are documented instead of concealed

For infrastructure software, trust is not a tagline.

**It should be something you can verify.**

---

# Commercial proprietary software

Ghost FTP is proprietary commercial software.

It is **not open-source software** and is not distributed under an OSI-approved license.

Source visibility does not grant a general right to:

* modify
* redistribute
* sublicense
* rebrand
* white-label
* incorporate Ghost FTP into another product

Ghost FTP and its source code, binaries, user interfaces, website, Web FTP implementation, documentation, branding, icons and release engineering are copyrighted works of:

**Brendigo LTD**

Copyright © 2026 Brendigo LTD.
All rights reserved.

See:

* [`LICENSE`](LICENSE)
* [`docs/THIRD-PARTY-NOTICES.md`](docs/THIRD-PARTY-NOTICES.md)

for the controlling licensing terms and independent third-party notices.

---

# Documentation

Start with:

[Documentation overview](docs/README.md)

Then explore:

* [Installation](docs/INSTALLATION.md)
* [Architecture](docs/ARCHITECTURE.md)
* [Platform parity](docs/PLATFORM-PARITY.md)
* [Web / Web FTP](docs/WEB.md)
* [Security](docs/SECURITY.md)
* [Privacy](docs/PRIVACY.md)
* [Signing](docs/SIGNING.md)
* [Release verification](docs/RELEASE-VERIFICATION.md)
* [Testing](docs/TESTING.md)
* [Reference UI](docs/REFERENCE-UI.md)
* [Roadmap](docs/ROADMAP.md)

---

<p align="center">
<strong>Ghost FTP</strong><br>
Direct file transfer. Clear security boundaries. No cloud middleman.
</p>

<p align="center">
<strong>Your servers. Your files. Your control.</strong>
</p>
