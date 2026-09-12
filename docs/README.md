# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="108">
</p>

<p align="center"><strong>Production behavior, security boundaries, UI evidence and release engineering for Ghost FTP.</strong></p>

- **Current Ghost FTP release: 0.0.5**
- Development status: **Active**
- Release channel: **Current**
- GitHub Release policy: **PRERELEASE=false**
- Public version retention: **latest release only**
- Public release platforms: **Windows and Linux**
- Active native source platforms: **Windows, Linux and Android**
- Protocols: **FTP, FTPS and SFTP** on desktop; **FTP and strict explicit FTPS** on the current Android development surface
- Languages: **24 selectable local desktop languages**
- Release shape: **14 platform artifacts / 17 public files**
- Product website: **https://ghostftp.com**
- Verified distribution bundle identity: **ghcr.io/bren-wp/ghost-ftp:0.0.5**

The root [`VERSION`](../VERSION) file is the authoritative production version source. Release-bound documentation describes the 0.0.5 current line. Public Windows/Linux release publication remains fail-closed and separate from the Android development APK path. Superseded release identities are removed only after the successor has passed source gates, publication, remote read-back and canonical retention.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Understand current settings | [`SETTINGS.md`](SETTINGS.md) |
| Verify UI behavior and screenshots | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Understand security boundaries | [`SECURITY.md`](SECURITY.md) |
| Understand privacy/no-telemetry behavior | [`PRIVACY.md`](PRIVACY.md) |
| Understand architecture/Core ownership | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Understand navigation bookmarks/profile starts | [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md) |
| Verify Windows/Linux parity | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Validate a downloaded release | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| Understand release/version lifecycle | [`VERSIONING.md`](VERSIONING.md) and [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| Build/test/contribute | [`TESTING.md`](TESTING.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Get support | [`SUPPORT.md`](SUPPORT.md) |

## Authentic visual reference

Documentation media is **repository-local**. No remote badge image, tracking pixel, remote icon resource, remote webfont or analytics resource is required when these documents render. Exact-head CI captures real native runtime evidence for Windows, Linux and Android and assembles a read-only verified cross-platform evidence bundle. Mockups/generated approximations are not production evidence.

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

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for provenance and the rule that mockups/generated approximations are not production UI evidence.

## Current 0.0.5 capability contract

Ghost FTP 0.0.5 retains the complete 0.0.4 desktop feature set and adds lifecycle/re-entry hardening plus optional browser companion source.

- FTP, explicit FTPS and SFTP through one typed desktop Core engine.
- Native Windows and Linux frontends consuming the same typed engine behavior.
- Built-in Remote Edit with bounded text validation, revision/conflict protection and verified read-back.
- Transfer queue pause/resume/cancel/retry/clear plus queued Top/Up/Down/Bottom ordering.
- Independent aggregate upload/download bandwidth ceilings with real transport enforcement.
- Current-folder filtering, deterministic sorting, bounded recursive search and conservative directory comparison.
- Local/server bookmarks and profile start directories with account/session revalidation.
- Site Manager profiles with explicit protected credential-save consent.
- Windows mutation/re-entry guards for profile persistence, file mutations and Remote Edit lifecycle.
- Android pending connections owned by the Activity lifecycle so destroyed/recreated UI instances cannot be revived by stale callbacks.
- Optional browser companion source for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox; companion packages remain outside the public 17-file desktop release.
- Strict FTPS verification, strict desktop SFTP host-key verification/pinning and no silent secure-to-plain downgrade.
- **No telemetry, analytics, advertising, tracking or hidden product backend.**

## Current publication identity

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
CHANNEL=Current
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

Windows:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Representative Linux files:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
```

The complete Linux matrix covers Debian/Ubuntu `amd64`, `arm64`, `i386`; Fedora `x86_64`, `aarch64`, `i686`; and Portable `amd64`, `arm64`, `i386`.

Android development CI artifact:

```text
Ghost-FTP-Android.apk
```

The Android artifact is independently exact-head verified but is not counted in the Windows/Linux public release allow-list.

The verified desktop release directory is also distributed as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

This is a distribution bundle, not a runtime container.

## Documentation map

### Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — components, ownership and trust boundaries.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux parity and Android boundary.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — native UI and authentic evidence contract.
- [`SETTINGS.md`](SETTINGS.md) — validated settings and persistence behavior.
- [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md) — bookmark and profile-start contract.
- [`QUEUE-PRIORITY.md`](QUEUE-PRIORITY.md) — queue ordering contract.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency and external-tool policy.

### Security and privacy

- [`SECURITY.md`](SECURITY.md) — protocol, filesystem, installer and release trust boundaries.
- [`PRIVACY.md`](PRIVACY.md) — local-first data handling and no-telemetry contract.
- [`SIGNING.md`](SIGNING.md) — optional production Authenticode and truthful unsigned state.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — third-party notices.

### Distribution and release verification

- [`INSTALLATION.md`](INSTALLATION.md) — Windows/Linux installation and Android development boundary.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical release shape and deterministic retention lifecycle.
- [`PACKAGES.md`](PACKAGES.md) — verified GitHub Packages distribution bundle policy.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — checksums, source identity and signing verification.
- [`VERSIONING.md`](VERSIONING.md) — controlled 0.0.x public version policy.

### Engineering

- [`TESTING.md`](TESTING.md) — exact-head CI, Android APK, authentic UI and release gates.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — engineering rules and PR expectations.
- [`ROADMAP.md`](ROADMAP.md) — future capabilities and acceptance criteria.
- [`SUPPORT.md`](SUPPORT.md) — support and privacy-safe diagnostic guidance.
- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — public-line history.
- [`../CHANGELOG.md`](../CHANGELOG.md) — source for generated release notes.

Only the latest public Ghost FTP version remains in active release infrastructure after retention cleanup succeeds. Git history remains the engineering provenance of earlier work.
