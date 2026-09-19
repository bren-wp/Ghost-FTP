# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="112">
</p>

<h2 align="center">One client. Three platforms. Zero friction.</h2>

<p align="center">
  A fast, privacy-first FTP / FTPS / SFTP file-transfer client for <strong>Windows</strong>, <strong>Linux</strong> and <strong>Android</strong>.
</p>

<p align="center">
  <img alt="CI" src="https://github.com/bren-wp/Ghost-FTP/actions/workflows/ci.yml/badge.svg">
  <img alt="CodeQL" src="https://github.com/bren-wp/Ghost-FTP/actions/workflows/codeql.yml/badge.svg">
  <img alt="Govulncheck" src="https://github.com/bren-wp/Ghost-FTP/actions/workflows/govulncheck.yml/badge.svg">
</p>

<p align="center">
  <a href="docs/INSTALLATION.md"><strong>Installation</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/REFERENCE-UI.md"><strong>UI reference</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

<p align="center">
  <strong>Current source version: 0.0.9</strong> · English primary · Croatian secondary · 24 desktop languages · no telemetry
</p>

---

## File transfer without the clutter

Ghost FTP is built around a focused Files workspace instead of a dashboard full of secondary controls. The maintained product line is deliberately limited to **Windows, Linux and Android**.

The primary workspace keeps the same hierarchy across platforms:

**connection → toolbar → local/remote files → transfer queue → live status**

The supplied Windows, Linux and Android references are the design masters. Visual parity is accepted only after an **authentic runtime screenshot from the exact source commit** has been reviewed; generated mockups are never used as proof that the application matches the reference.

---

## Authentic runtime UI

### Windows

![Ghost FTP Windows Files workspace](docs/images/ghost-ftp-main-workspace.png)

### Linux

![Ghost FTP Linux Files workspace](docs/images/ghost-ftp-linux-main-workspace.png)

### Android

![Ghost FTP Android Files workspace](docs/images/ghost-ftp-android-files.png)

These are repository-local application captures. Version-bound evidence and provenance are retained under `docs/images/`, while CI also produces exact-head Windows, Linux and Android screenshot artifacts for review before release.

---

## What Ghost FTP includes

<table>
<tr>
<td width="33%" valign="top"><strong>Files</strong><br><sub>Dual-pane local/server browsing, Back/Forward history, refresh, folders, rename, delete, permissions, filters, recursive search, comparison and remote editing where supported.</sub></td>
<td width="33%" valign="top"><strong>Transfers</strong><br><sub>Upload/download, real progress, speed, ETA, queued/running/completed/failed/cancelled states, retry, cancellation, pause/resume and queue priority where supported by the platform engine.</sub></td>
<td width="33%" valign="top"><strong>Connections</strong><br><sub>Quick Connect, saved sites, reconnect/disconnect and secure handling of host, port, username, protected credentials and key material.</sub></td>
</tr>
<tr>
<td width="33%" valign="top"><strong>Security</strong><br><sub>Strict desktop SFTP host-key trust, fail-closed mismatch handling and real FTPS certificate/hostname validation. Security checks are not disabled for UI parity.</sub></td>
<td width="33%" valign="top"><strong>Privacy</strong><br><sub>No analytics, telemetry, ads, hidden synchronization service, mandatory Ghost FTP account or transfer proxy.</sub></td>
<td width="33%" valign="top"><strong>Release integrity</strong><br><sub>Exact-head CI, CodeQL, Govulncheck, signing checks, package verification, SHA-256 checksums and immutable published versions.</sub></td>
</tr>
</table>

---

## Reference workspace

### Desktop

The Windows and Linux Files screen follows the same product hierarchy:

```text
Ghost FTP
├─ Files
├─ Connections
├─ Transfer Queue
└─ Settings

Connection
[server / connection URI] [Connected / Disconnected] [Quick Connect]

Toolbar
Back · Forward · Refresh · New Folder · Upload · Download · Bookmarks · More

Workspace
Local Files                         Remote Files
Name · Size · Modified              Name · Size · Modified · Permissions

Transfer Queue
Active · Completed · Failed · Clear Completed
File · Direction · Progress · Status · Speed · ETA
```

Advanced operations remain real but are moved into **More**, context menus or focused dialogs when keeping them permanently visible would damage the master layout.

### Android

Android follows the mobile master hierarchy:

- Ghost FTP header and live connection state;
- current site/server card;
- Back / Forward / Refresh / New Folder / Upload;
- Download / Bookmarks / More;
- Local Files and Remote Files, side-by-side where screen width allows;
- Transfer Queue;
- bottom navigation: **Files · Sites · Bookmarks · Transfers · Settings**.

Responsive layouts must remain usable under smaller widths, landscape and Android text scaling; visual parity never overrides touch-target or security requirements.

---

## Supported protocols

| Protocol | Windows | Linux | Android | Security boundary |
| --- | ---: | ---: | ---: | --- |
| FTP | ✓ | ✓ | ✓ | Unencrypted compatibility protocol |
| FTPS explicit | ✓ | ✓ | ✓ | Certificate and hostname verification |
| FTPS implicit | ✓ | ✓ | — | Exposed only where the maintained engine supports it |
| SFTP | ✓ | ✓ | — | Strict SSH host-key verification / pinning |

Android SFTP is intentionally hidden until strict maintained host-key verification is available there. Ghost FTP does not weaken this boundary to claim feature parity.

---

## Platforms and packages

| Platform | Release package |
| --- | --- |
| **Windows** | Universal Setup EXE + Portable EXE with x64, x86 and ARM64 payloads |
| **Linux** | Debian, Ubuntu and Fedora Installer + Portable bundles with amd64, arm64 and i386 payloads |
| **Android** | Production APK |

The active public release contract is **9 application artifacts / 12 public files** including `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

**macOS is retired. Browser extensions are retired.** Neither is part of the active source-platform or release-artifact contract.

---

## Security

Ghost FTP treats protocol and release security as product behavior, not optional hardening.

- SFTP unknown-host trust remains explicit; pinned-key mismatch fails closed.
- FTPS certificate/hostname errors are not silently bypassed.
- Credentials are not written to logs or release evidence.
- Destructive filesystem operations validate their targets.
- Cancellation and transfer-finalization boundaries avoid committing partial files as completed files.
- Official publication requires the configured Windows and Android signing identities.
- Release workflows refuse to overwrite an existing tag or published release.

Read [Security](docs/SECURITY.md), [Signing](docs/SIGNING.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

---

## Privacy

Ghost FTP has no product analytics or telemetry SDK.

It does not require an application account and does not route file transfers through a Ghost FTP backend. Native clients connect to the server selected by the user. Saved credentials use the platform-specific protected storage path implemented by the application.

Read [Privacy](docs/PRIVACY.md).

---

## Installation

See [Installation](docs/INSTALLATION.md) for platform-specific installation and package verification.

Public packages use the canonical version in the repository root `VERSION` file. The current source identity is:

```text
VERSION=0.0.9
TAG=ghostftp-v0.0.9
CHANNEL=Current
PRERELEASE=false
```

Published tags and assets are immutable.

---

## Build from source

The maintained source builds are Windows, Linux and Android. CI validates:

- Go format, tests, race tests and vet;
- Windows universal Setup and Portable builds;
- Linux Debian/Ubuntu/Fedora universal bundles and install lifecycle;
- Android unit tests, lint and APK build/signing contracts;
- repository, platform, release, privacy and security audits;
- CodeQL and Govulncheck;
- authentic Windows/Linux/Android runtime screenshots.

See [Testing](docs/TESTING.md), [Packages](docs/PACKAGES.md) and [Architecture](docs/ARCHITECTURE.md).

---

## Versioning and releases

Ghost FTP uses immutable semantic versions with `ghostftp-v<version>` tags.

**After a version is published, every subsequent source, UI, behavior, documentation, packaging or release-metadata change must advance to a new version.** Existing tags and release assets are never retagged, replaced or silently rewritten.

See [Versioning](docs/VERSIONING.md), [Release history](docs/RELEASE-HISTORY.md) and [GitHub Releases](docs/GITHUB-RELEASES.md).

---

## Localization

English is the primary/fallback language and Croatian is the secondary maintained language. Desktop localization currently exposes 24 selectable languages from the canonical localization registry. Product/protocol names remain consistent across locales.

---

## Documentation

- [Documentation index](docs/README.md)
- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Reference UI](docs/REFERENCE-UI.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Navigation and bookmarks](docs/NAVIGATION-BOOKMARKS.md)
- [Queue priority](docs/QUEUE-PRIORITY.md)
- [Settings](docs/SETTINGS.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Signing](docs/SIGNING.md)
- [Packages](docs/PACKAGES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Versioning](docs/VERSIONING.md)

---

## License

Ghost FTP is proprietary commercial software with source available for transparency and review under the repository [LICENSE](LICENSE). Public source visibility does not grant redistribution, rebranding, sublicensing or derivative-distribution rights beyond the license and applicable law.

<p align="center"><strong>Ghost FTP</strong><br>One client. Three platforms. Zero friction.</p>
