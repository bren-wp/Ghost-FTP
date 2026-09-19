# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="112">
</p>

<h2 align="center">One client. Three platforms. Zero friction.</h2>

<p align="center">
  Ghost FTP is a privacy-first native file-transfer client for <strong>Windows</strong>, <strong>Linux</strong> and <strong>Android</strong>.
  It combines a focused dual-pane workspace, real transfer-queue controls, bookmarks, saved connections and direct server access without a mandatory Ghost FTP account or hidden storage cloud.
</p>

<p align="center">
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

<p align="center">
  <strong>Current release: 0.0.8</strong> · Current source version: **0.0.8** · 0.0.9 development line · 24 desktop languages · no telemetry
</p>

> Last actually published GitHub Release: **0.0.8**
>
> Release channel: **Current** · Product status: **Current** · Prerelease: **false**
>
> Machine-readable release state: `PRERELEASE=false`
>
> Next-line distribution contract: **13 platform artifacts / 16 public files**

---

## One workspace across Windows, Linux and Android

Ghost FTP follows one reference-driven product hierarchy across all maintained applications:

**Files · Connections / Sites · Transfer Queue / Transfers · Settings**

The Files workspace keeps the same product language on every maintained platform:

- direct connection state and Quick Connect;
- Back, Forward, Refresh, New Folder, Upload, Download, Bookmarks and More;
- Local Files and Remote Files panes;
- Name / Size / Modified columns, plus remote permissions where supported;
- a real Transfer Queue with progress, status, speed and ETA;
- clear success, active and failed states;
- the Ghost Gold identity used consistently in application chrome and packaging.

The Windows, Linux and Android interfaces are intentionally aligned with the supplied Ghost FTP reference screenshots. Release claims are based on authentic runtime captures tied to exact source commits, not generated mockups.

---

## Authentic application screenshots

### Windows

![Ghost FTP Windows workspace](docs/images/ghost-ftp-main-workspace.png)

### Linux

![Ghost FTP Linux workspace](docs/images/ghost-ftp-linux-main-workspace.png)

### Android

![Ghost FTP Android workspace](docs/images/ghost-ftp-android-files.png)

Windows · Linux · Android

These repository-local captures are product evidence, not generated mockups. The version-bound 0.0.8 provenance set remains under `docs/images/0.0.8/`; current root-level captures track the maintained development UI and must come from the authentic screenshot workflow.

More authentic runtime evidence is documented in [Reference UI](docs/REFERENCE-UI.md).

---

## Built for real file work

<table>
<tr>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/transfer.svg" width="52" alt=""><br><strong>Direct transfers</strong><br><sub>Upload, download, retry, cancellation, priority, progress, speed and ETA backed by the real transfer engine.</sub></td>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/security.svg" width="52" alt=""><br><strong>Fail-closed security</strong><br><sub>Strict desktop SFTP host-key trust, verified FTPS identity and explicit distribution trust boundaries.</sub></td>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/privacy.svg" width="52" alt=""><br><strong>Privacy first</strong><br><sub>No telemetry, behavioral analytics, advertising, hidden synchronization service or mandatory Ghost FTP account.</sub></td>
</tr>
<tr>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/platforms.svg" width="52" alt=""><br><strong>Three maintained apps</strong><br><sub>Windows, Linux and Android share one Ghost FTP identity with platform-native lifecycle ownership.</sub></td>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/release.svg" width="52" alt=""><br><strong>Verified release flow</strong><br><sub>Exact-head builds, checksums, explicit signing state, package validation and release readback.</sub></td>
<td width="33%" valign="top" align="center"><img src="docs/images/readme/docs.svg" width="52" alt=""><br><strong>Auditable documentation</strong><br><sub>Security, privacy, packaging, platform parity, runtime evidence and release verification remain version-bound.</sub></td>
</tr>
</table>

---

## Core capabilities

### Files and navigation

- local and remote directory navigation;
- Back / Forward history;
- Refresh and New Folder;
- rename and delete;
- remote permissions / CHMOD where supported;
- filtering, sorting and recursive search;
- directory comparison;
- bookmarks and saved locations;
- Remote Edit where supported.

### Transfers

- upload and download;
- queued, running, completed, failed and cancelled lifecycle states;
- Pause / Resume / Retry / Cancel;
- queue priority controls;
- progress, throughput and ETA;
- conflict handling and overwrite policy;
- clear completed jobs.

### Connections

- Quick Connect;
- saved profiles;
- recent connections;
- reconnect / disconnect;
- edit and duplicate profiles;
- FTP compatibility;
- explicit FTPS with certificate and hostname verification;
- strict desktop SFTP host-key trust.

Android exposes FTP and strict explicit FTPS. Android SFTP remains hidden until strict host-key verification is maintained there.

---

## Maintained platforms

| Platform | Status | Distribution |
| --- | --- | --- |
| **Windows** | Primary | Universal Setup + Portable EXE with x64, x86 and ARM64 payloads |
| **Linux** | Primary | Debian, Ubuntu and Fedora Installer + Portable bundles |
| **Android** | Primary | Installable APK; publisher-signing state is declared explicitly per release |
| **Browser helpers** | Supporting | Local Chrome, Edge, Firefox and Opera helper packages |

The native application surface is now deliberately limited to **Windows, Linux and Android**. The former macOS application, macOS build scripts, signing/notarization workflows and macOS-specific regression tests have been retired from the active source tree.

---

## Security model

- FTP is available only as an intentional unencrypted compatibility protocol.
- Explicit FTPS validates certificate trust and hostname identity.
- Desktop SFTP uses strict SSH host-key trust and pinning.
- Native transfer traffic goes directly to the server selected by the user.
- The application does not proxy transfers through a Ghost FTP cloud.
- Release metadata records the actual signing state instead of implying stronger trust than the artifact has.
- Protected production signing remains fail-closed when publisher credentials are unavailable.

See [Security](docs/SECURITY.md) and [Signing](docs/SIGNING.md).

---

## Privacy

Ghost FTP does not require:

- an application account;
- analytics consent;
- telemetry;
- advertising identifiers;
- behavioral tracking;
- a Ghost FTP storage backend.

Saved secrets stay under platform-appropriate local credential protection. Connection data and transfer data are not sent to a Ghost FTP analytics service.

See [Privacy](docs/PRIVACY.md).

---

## Current release and next development line

**0.0.8** remains the current published release and is immutable.

The **0.0.9** development line concentrates on:

- Windows / Linux / Android visual parity with the supplied master references;
- tighter Files workspace composition;
- consistent popups, menus and secondary surfaces;
- improved readable labels and touch targets;
- cleanup of retired platform code;
- stronger marketing and product documentation;
- exact-head runtime screenshot evidence before release.

A new release version is created only when the validated source state changes and the release evidence is rebuilt from that exact commit.

---

## Build and validation

Ghost FTP CI validates:

- Go formatting, tests and vet;
- Windows Setup + Portable packaging;
- Linux distro bundles and install lifecycle;
- Android lint/build/signing contracts;
- CodeQL and Govulncheck;
- security, privacy, repository and documentation audits;
- authentic Windows / Linux / Android runtime screenshots;
- exact-head release metadata and checksums.

See [Testing](docs/TESTING.md), [Release verification](docs/RELEASE-VERIFICATION.md) and [Packages](docs/PACKAGES.md).

Windows release metadata preserves the architecture evidence boundary:

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The universal Windows packages carry x64, x86 and ARM64 native payloads, while CI does not claim native ARM64 runtime execution without maintained ARM64 hardware/runner evidence.

---

## Documentation

Start with [docs/README.md](docs/README.md).

Key documents:

- [Installation](docs/INSTALLATION.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Reference UI](docs/REFERENCE-UI.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Settings](docs/SETTINGS.md)
- [Navigation and bookmarks](docs/NAVIGATION-BOOKMARKS.md)
- [Queue priority](docs/QUEUE-PRIORITY.md)
- [Testing](docs/TESTING.md)
- [Signing](docs/SIGNING.md)
- [Packages](docs/PACKAGES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Versioning](docs/VERSIONING.md)

---

## Localization

English is the primary and fallback product language. Croatian is the secondary maintained language, and the desktop applications expose **24 selectable desktop languages** from the canonical localization registry. User-facing strings must use the shared localization catalogs instead of platform-specific hardcoded copies where localization is supported.

## Commercial proprietary software

Ghost FTP is proprietary commercial software. It is not open-source software. The source is available for transparency and review under the controlling repository [`LICENSE`](LICENSE); public source visibility does not grant open-source redistribution, rebranding, sublicensing or derivative-distribution rights beyond the license and applicable law.

## Product identity

**Ghost FTP**
**One client. Three platforms. Zero friction.**

The maintained native application targets are Windows, Linux and Android. Browser helpers remain supporting packages rather than additional native applications.

