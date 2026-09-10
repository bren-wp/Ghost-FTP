# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="108">
</p>

<p align="center"><strong>Production behavior, security boundaries, UI evidence and release engineering for Ghost FTP.</strong></p>

- **Current Ghost FTP release: 0.0.3**
- Development status: **Active**
- Release channel: **Current**
- GitHub Release policy: **PRERELEASE=false**
- Public version retention: **latest release only**
- Platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Languages: **24 selectable local languages**
- Release shape: **14 platform artifacts / 17 public files**
- Verified bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.3`

The root [`VERSION`](../VERSION) file is the authoritative production version source. Active documentation describes the 0.0.3 current release line and its exact Windows/Linux packaging contract.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Understand all current settings | [`SETTINGS.md`](SETTINGS.md) |
| Verify UI behavior and screenshots | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Understand security boundaries | [`SECURITY.md`](SECURITY.md) |
| Understand privacy/no-telemetry behavior | [`PRIVACY.md`](PRIVACY.md) |
| Understand architecture/Core ownership | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Verify Windows/Linux parity | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Validate a downloaded release | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| Understand release/version lifecycle | [`VERSIONING.md`](VERSIONING.md) and [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| Build/test/contribute | [`TESTING.md`](TESTING.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md) |

## Authentic visual reference

Documentation media is **repository-local**. Production screenshots come from the real verified Windows application payload and are checked by the authentic-UI workflow.

<table>
<tr>
<td width="50%" valign="top"><strong>Main Workspace</strong><br><br><img src="images/ghost-ftp-main-workspace.png" alt="Ghost FTP Main Workspace"></td>
<td width="50%" valign="top"><strong>Site Manager</strong><br><br><img src="images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
</tr>
<tr>
<td width="50%" valign="top"><strong>Settings</strong><br><br><img src="images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
<td width="50%" valign="top"><strong>About</strong><br><br><img src="images/ghost-ftp-about.png" alt="Ghost FTP About"></td>
</tr>
</table>

## Current 0.0.3 capability contract

Ghost FTP provides FTP/FTPS/SFTP, native Windows/Linux frontends over the same typed engine, Site Manager, protected credential handling, transfer queue lifecycle, independent upload/download bandwidth ceilings, Remote Edit, non-destructive folder filtering, bounded recursive search, conservative directory comparison, synchronized safe navigation, 24-language local UI, rooted local-path protections and strict FTPS/SFTP trust behavior without application telemetry or a hidden backend.

## Current publication identity

```text
VERSION=0.0.3
TAG=ghostftp-v0.0.3
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

Windows:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Linux publication is the canonical Debian/Ubuntu/Fedora/Portable set documented in [`INSTALLATION.md`](INSTALLATION.md) and built by `linux/BUILD-DISTROS.sh`.

After publication and remote read-back succeed, `.github/workflows/release-retention.yml` preserves only the latest verified Ghost FTP release/tag/canonical release branch and exact-version GHCR bundle while leaving `main` commit history untouched.

## Engineering references

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — components and trust boundaries.
- [`SETTINGS.md`](SETTINGS.md) — validated settings including upload/download KiB/s ceilings.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux behavioral and packaging parity.
- [`TESTING.md`](TESTING.md) — exact-head CI, package lifecycle and regression suites.
- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — maintained public release history.
- [`PACKAGES.md`](PACKAGES.md) — GHCR distribution-bundle contract.
- [`ROADMAP.md`](ROADMAP.md) — next power-user work.

Only the current public version is retained by release infrastructure after successful successor verification; Git history remains engineering provenance.
