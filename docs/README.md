# Ghost FTP documentation

<p align="center">
  <img src="../build/icon.png" alt="Ghost FTP application icon" width="118">
</p>

<p align="center"><strong>Authoritative product behavior, security boundaries, native-platform status, UI evidence and release engineering for Ghost FTP.</strong></p>

- **Current Ghost FTP release: 0.0.5**
- Development status: **Active**
- Release channel: **Current**
- GitHub Release policy: **PRERELEASE=false**
- Public version retention: **latest release only**
- Public release platforms: **Windows and Linux**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Windows native payloads: **x64, x86 and ARM64 inside the same two public universal executables**
- Desktop protocols: **FTP, FTPS and SFTP**
- Android protocols: **FTP and strict explicit FTPS**; SFTP remains hidden until strict native host-key verification exists
- Desktop languages: **24 selectable local languages**
- Release shape: **14 platform artifacts / 17 public files**
- Product website: **https://ghostftp.com**
- Verified distribution bundle: **ghcr.io/bren-wp/ghost-ftp:0.0.5**

The root [`VERSION`](../VERSION) file is the authoritative production version source. All active release-bound documentation must describe the current 0.0.5 contract unless a section is explicitly labeled historical. Public Windows/Linux publication is separate from Android/macOS development artifacts and browser companion source.

## Documentation principles

Ghost FTP documentation follows four rules:

1. **Current behavior first.** Active documents describe the maintained source and current release contract, not superseded packaging assumptions.
2. **No invented platform status.** Windows/Linux are public release platforms; Android/macOS are active native source/development platforms with separate evidence boundaries. Windows ARM64 is a maintained cross-built native payload, but native ARM64 runtime execution is not claimed without ARM64 runner evidence.
3. **Security claims are fail-closed.** Official Windows publication requires trusted Authenticode; development builds may be unsigned but are not official release evidence.
4. **Visual claims use real evidence.** Documentation media is repository-local and maintained runtime screenshots come from actual application surfaces, not generated mockups.

## Start here

| Goal | Document |
| --- | --- |
| Install or run Ghost FTP | [`INSTALLATION.md`](INSTALLATION.md) |
| Understand current settings | [`SETTINGS.md`](SETTINGS.md) |
| Verify UI behavior and screenshots | [`REFERENCE-UI.md`](REFERENCE-UI.md) |
| Understand security boundaries | [`SECURITY.md`](SECURITY.md) |
| Understand privacy/no-telemetry behavior | [`PRIVACY.md`](PRIVACY.md) |
| Understand architecture/core ownership | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Understand Windows/Linux/macOS/Android boundaries | [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) |
| Understand bookmarks/profile starts | [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md) |
| Validate a downloaded release | [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) |
| Understand signing | [`SIGNING.md`](SIGNING.md) |
| Understand release/version lifecycle | [`VERSIONING.md`](VERSIONING.md) and [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) |
| Build/test/contribute | [`TESTING.md`](TESTING.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Review current roadmap | [`ROADMAP.md`](ROADMAP.md) |
| Get support | [`SUPPORT.md`](SUPPORT.md) |
| macOS development details | [`../macos/README.md`](../macos/README.md) |
| Browser helper behavior | [`../ekstenzije/README.md`](../ekstenzije/README.md) |

## Authentic visual reference

Documentation media is **repository-local**. No remote badge image, tracking pixel, remote icon resource, remote webfont or analytics resource is required when these documents render.

Exact-head CI captures real native runtime evidence for Windows, Linux and Android and assembles a read-only verified cross-platform evidence bundle. The Windows evidence reflects the maintained Windows runner architecture; it is not silently relabeled as native ARM64 runtime proof. macOS has its own native development-app validation workflow. A development macOS build is not represented as Developer ID/notarized public-distribution evidence.

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

See [`REFERENCE-UI.md`](REFERENCE-UI.md) for provenance and the rule that mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

## Current 0.0.5 capability contract

Ghost FTP 0.0.5 retains the complete maintained desktop feature line and adds lifecycle, privacy and release hardening.

- FTP, explicit FTPS and SFTP through one typed shared desktop engine.
- Native Windows and Linux public-release frontends consuming the same typed engine behavior.
- Windows public Setup/Portable packages with native **x64, x86 and ARM64** payloads selected by `GetNativeSystemInfo` and no runtime download.
- Native macOS AppKit development frontend using the shared engine and complete maintained parity inventory.
- Native Android development client with FTP + strict explicit FTPS, SAF-scoped storage and lifecycle-owned sessions.
- Built-in Remote Edit with bounded text validation, revision/conflict protection and verified read-back.
- Transfer queue pause/resume/cancel/retry/clear plus queued Top/Up/Down/Bottom ordering.
- Windows transfer Add/Retry/Cancel completion ownership bound to the active connection generation.
- Independent aggregate upload/download bandwidth ceilings with real transport enforcement.
- Current-folder filtering, deterministic sorting, bounded recursive search and conservative directory comparison.
- Local/server bookmarks and profile start directories with account/session revalidation.
- Site Manager profiles with explicit protected credential-save consent.
- Windows mutation/re-entry guards for profile persistence, file mutations and Remote Edit lifecycle.
- Android stale-callback protection and authentication-error redaction.
- Privacy-minimal browser helper source for Chromium-family browsers and Firefox; local parsing/copy only, with no supported desktop launch/handoff contract today.
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

### Windows public files

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Official Windows publication requires trusted Authenticode and records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Architecture-specific Windows binaries are internal only. No public `-x64.exe`, `-x86.exe`, `-x32.exe` or `-arm64.exe` is added to the release. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` states that ARM64 is cross-built and verified but not natively executed by the current maintained Windows Actions runner.

Local development and ordinary CI Windows builds may remain unsigned, but they are not official public release artifacts.

### Linux public files

The complete Linux matrix covers:

- Debian: `amd64`, `arm64`, `i386`;
- Ubuntu: `amd64`, `arm64`, `i386`;
- Fedora: `x86_64`, `aarch64`, `i686`;
- Portable: `amd64`, `arm64`, `i386`.

Representative files:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
```

### Android development artifact

```text
Ghost-FTP-Android.apk
```

The Android development APK is independently exact-head verified but is not counted in the public Windows/Linux release allow-list.

### macOS development artifact

The maintained macOS workflow builds a universal native development app. Its development ZIP is not counted in the 17-file public release allow-list and does not claim Developer ID/notarization success. Production macOS distribution requires the separate fail-closed signing/notarization path documented in [`../macos/README.md`](../macos/README.md).

### Distribution bundle

The verified public Windows/Linux release directory is also distributed as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

This is a distribution bundle, not a runtime container.

## Documentation map

### Product and architecture

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — components, ownership, Windows architecture dispatch and trust boundaries.
- [`PLATFORM-PARITY.md`](PLATFORM-PARITY.md) — Windows/Linux public parity plus Android/macOS development boundaries.
- [`REFERENCE-UI.md`](REFERENCE-UI.md) — native UI and authentic evidence contract.
- [`SETTINGS.md`](SETTINGS.md) — validated settings and persistence behavior.
- [`NAVIGATION-BOOKMARKS.md`](NAVIGATION-BOOKMARKS.md) — bookmark and profile-start contract.
- [`QUEUE-PRIORITY.md`](QUEUE-PRIORITY.md) — queue ordering contract.
- [`LOCALIZATION.md`](LOCALIZATION.md) — 24-language local localization model.
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — dependency and external-tool policy.

### Security and privacy

- [`SECURITY.md`](SECURITY.md) — protocol, filesystem, installer and release trust boundaries.
- [`PRIVACY.md`](PRIVACY.md) — local-first data handling and no-telemetry contract.
- [`SIGNING.md`](SIGNING.md) — signed-only official Windows publication and development-build distinction.
- [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md) — third-party notices.

### Distribution and release verification

- [`INSTALLATION.md`](INSTALLATION.md) — Windows universal x64/x86/ARM64 installation plus Linux and development-platform boundaries.
- [`GITHUB-RELEASES.md`](GITHUB-RELEASES.md) — canonical release shape and deterministic retention lifecycle.
- [`PACKAGES.md`](PACKAGES.md) — verified GitHub Packages distribution-bundle policy.
- [`RELEASE-VERIFICATION.md`](RELEASE-VERIFICATION.md) — checksums, source identity, Windows ARM64 evidence boundary and signing verification.
- [`VERSIONING.md`](VERSIONING.md) — controlled current public version policy.

### Engineering

- [`TESTING.md`](TESTING.md) — exact-head CI, Windows x64/x86/ARM64 package gates, Android APK, macOS development app, authentic UI and release gates.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — engineering rules and PR expectations.
- [`ROADMAP.md`](ROADMAP.md) — current capability direction and acceptance criteria.
- [`SUPPORT.md`](SUPPORT.md) — support and privacy-safe diagnostic guidance.
- [`RELEASE-HISTORY.md`](RELEASE-HISTORY.md) — public-line history.
- [`../CHANGELOG.md`](../CHANGELOG.md) — source for generated release notes.

Only the latest public Ghost FTP version remains in active release infrastructure after retention cleanup succeeds. Git history remains the engineering provenance of earlier work.
