# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="148">
</p>

<p align="center"><strong>Your servers. Your files. No cloud middleman.</strong></p>

<p align="center">
  A fast, native and privacy-first FTP/FTPS/SFTP workspace for professionals who want direct control over their infrastructure — without telemetry, mandatory accounts or a Ghost FTP cloud relay.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/Ghost-FTP/releases"><strong>Download Ghost FTP</strong></a> ·
  <a href="https://ghostftp.com"><strong>ghostftp.com</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Installation</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a>
</p>

Ghost FTP is built for real server work: connect securely, browse local and remote files, transfer complete directory trees, edit remote text safely, manage reusable sites and bookmarks, control large transfer queues and keep connection state understandable. The maintained desktop engine supports **FTP, explicit FTPS and SFTP** with strict transport identity checks and no silent secure-to-plain fallback.

- Current Ghost FTP version: **0.0.5**
- Development status: **Active**
- Release channel: **Current**
- Public release platforms: **Windows and Linux**
- Active native source platforms: **Windows, Linux, Android and macOS**
- Default language: **English**
- Selectable local desktop languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Repository: https://github.com/bren-wp/Ghost-FTP
- Public release identity: `ghostftp-v0.0.5`, `prerelease=false`
- Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.5`
- Public release shape: **14 platform artifacts / 17 public files**

## See Ghost FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

<table>
<tr>
<td width="50%" valign="top"><strong>Site Manager</strong><br><br><img src="docs/images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
<td width="50%" valign="top"><strong>Settings</strong><br><br><img src="docs/images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
</tr>
<tr>
<td colspan="2" align="center"><strong>About / verified product identity</strong><br><br><img src="docs/images/ghost-ftp-about.png" alt="Ghost FTP About" width="620"></td>
</tr>
</table>

The logo and UI media rendered by this README are **repository-local assets**. Maintained runtime evidence is captured from real native Windows, Linux and Android application surfaces through exact-head CI. Mockups, generated approximations and screenshots from a different source revision are not production evidence. No remote badge image, tracking pixel, icon CDN or remote webfont is required to render this README.

See [Reference UI](docs/REFERENCE-UI.md) for screenshot provenance and the production visual-evidence contract.

## Built for serious file workflows

| Capability | Ghost FTP 0.0.5 |
| --- | --- |
| Secure connections | FTP compatibility, explicit FTPS with certificate/hostname verification, SFTP with strict host-key verification/pinning on maintained desktop platforms |
| Native workspace | Local and Remote panes, Site Manager, bookmarks, start directories, filtering, deterministic sorting and bounded recursive search |
| Transfers | File and directory-tree upload/download, pause/resume/cancel/retry, truthful progress/speed/ETA and staged commit/rollback behavior |
| Queue control | Clear Finished plus queued **Top / Up / Down / Bottom** ordering without mutating running/terminal history |
| Bandwidth policy | Independent aggregate upload/download ceilings with real curl/OpenSSH transport enforcement |
| Remote Edit | Bounded text editing with UTF-8/binary validation, line-ending preservation, revision conflict detection and verified read-back |
| File management | New Folder, Rename, Delete, server permissions where supported and stale-view rejection around mutations |
| Directory tools | Current-folder filtering, bounded recursive search, conservative directory comparison and synchronized navigation |
| Profiles | Site Manager profiles with explicit saved-credential consent and account-bound remote starts |
| Privacy | No application analytics, advertising, fingerprinting, tracking pixels, automatic crash upload or hidden Ghost FTP backend |

## Why Ghost FTP

### Direct by design

Your FTP/FTPS/SFTP traffic goes to the server you configure. Ghost FTP is not a mandatory cloud intermediary and does not require a product account to use the native client.

### Fail-closed security boundaries

Secure transports stay secure. FTPS certificate/hostname validation and desktop SFTP host-key verification are not silently bypassed. Credential lifetime, path confinement, transfer staging and platform-specific helper provenance are treated as security boundaries rather than convenience options.

### Native instead of web-wrapped

Windows and Linux are maintained public native desktop release platforms. Android and macOS are active native source/development surfaces with their own platform-specific lifecycle and storage/security constraints. Ghost FTP does not hide a browser backend behind a desktop shell.

### Professional transfer controls

Queue ordering, directional bandwidth limits, retries, conflict policy, staged writes, Remote Edit and deterministic listing/search behavior are wired to runtime logic and regression coverage rather than decorative controls.

### Local-first privacy

Ghost FTP has no application telemetry, advertising SDK, tracking backend or mandatory cloud profile store. Production workflows explicitly disable Go telemetry. Saved credentials are opt-in and remain under platform-local protection boundaries.

## Platform status

| Platform | Status | Current contract |
| --- | --- | --- |
| **Windows** | Public release | Universal Setup + Portable; verified internal x64/x86 payloads; official publication requires trusted Authenticode |
| **Linux** | Public release | Debian, Ubuntu, Fedora and Portable packages for the canonical architecture set |
| **Android** | Active development | Installable development APK; FTP + strict explicit FTPS; SAF-scoped local storage; SFTP remains hidden until strict native host-key verification exists |
| **macOS** | Active development | Native AppKit frontend using the shared engine; universal development app; separate fail-closed Developer ID signing/notarization path |
| **Browser helper** | Source companion | Local parsing/copy helper for supported FTP-family targets; no supported browser-to-desktop launch/handoff contract today |

The current public GitHub Release remains the explicit Windows/Linux **14 platform artifacts / 17 public files** contract. Android, macOS and browser-helper source are not silently counted as public release artifacts.

## Connect your way

- **FTP** for explicit legacy compatibility.
- **FTPS** with certificate and hostname validation; a failed secure connection is never silently retried as plain FTP.
- **SFTP** with strict host-key verification/pinning plus password or private-key authentication on maintained desktop platforms.
- Saved Site Manager profiles with explicit credential-persistence consent instead of hidden secret storage.
- Local and remote bookmarks plus profile start directories with account/session revalidation.

Fresh desktop Quick Connect resolves to explicit FTPS on port 21. Plain FTP must be selected intentionally where compatibility requires it.

## Move files with real queue control

Ghost FTP can upload/download single files and recursive directory trees while keeping queue state explicit and auditable.

- Pause, resume, cancel, retry and clear finished transfers.
- Reorder queued work **Top / Up / Down / Bottom** without rewriting running or terminal job history.
- Display transfer progress, transferred bytes, speed and ETA from real transfer events.
- Configure independent upload/download bandwidth ceilings in binary KiB/s, including `0 = unlimited`.
- Treat configured bandwidth as a conservative aggregate directional ceiling across worker slots.
- Choose **Skip**, **Replace** or **Replace + recovery backup** conflict behavior.
- Use staged activation/rollback instead of direct destructive overwrite where the maintained transfer path requires it.

## Remote Edit without leaving the client

Ghost FTP includes built-in **Remote Edit** for supported regular text files. The maintained contract provides:

- bounded text size;
- UTF-8/binary validation;
- LF/CRLF/CR preservation;
- mixed-line-ending rejection;
- SHA-256 revision/conflict detection;
- verified upload/read-back;
- remote permission preservation when trustworthy metadata exists;
- serialized edit-session ownership so stale asynchronous operations cannot create a second conflicting editor session.

Open, edit, **Save / Reload / Close**, then return to the normal file workflow.

## Find, filter and compare confidently

Ghost FTP keeps instant filtering separate from recursive I/O:

- current-folder filtering operates only on the already loaded snapshot;
- sorting is deterministic by Name, Type, Size and Modified, plus remote Permissions where available;
- recursive search is explicit, bounded and cancellable;
- activating a search result performs a fresh parent listing before selection becomes authoritative;
- directory comparison uses conservative `same/local_only/remote_only/newer_local/newer_remote/conflict/unknown` semantics;
- synchronized navigation is available only for safely proven paired ordinary directories.

## Security is part of the product

Ghost FTP preserves:

- FTPS certificate and hostname validation;
- strict SFTP host-key verification/pinning on maintained desktop platforms;
- no silent secure-to-plain downgrade;
- protected saved-secret lifetime rules;
- local path containment and symlink/reparse safety;
- staged transfer activation/rollback;
- account/session binding for saved remote paths;
- trusted Linux transport/AskPass provenance;
- exact-object/ownership-aware Windows cleanup;
- privacy-safe error classification instead of raw credential-bearing tool/server diagnostics.

Android 0.0.5 additionally sanitizes FTP authentication errors so server-controlled replies cannot echo a submitted password into user-facing errors.

See [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md) and [Architecture](docs/ARCHITECTURE.md).

## Privacy without fine print

Ghost FTP includes **no application analytics, advertising, fingerprinting, tracking pixels, automatic crash upload, mandatory Ghost FTP account or hidden profile synchronization**.

Connection/profile state remains local. Browser companion source does not scrape ordinary tabs, request broad host permissions or store FTP credentials. Development/build infrastructure may contact the explicit package/signing/notarization services required for CI/release engineering, but the Ghost FTP application itself has no hidden product telemetry backend.

## 24 local desktop languages

**English** is the canonical default and fallback. The maintained desktop registry exposes **24 selectable local languages**:

English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.

Language selection and translation resolution happen locally. See [Localization](docs/LOCALIZATION.md).

## Ghost FTP 0.0.5 highlights

Ghost FTP 0.0.5 is a reliability, lifecycle, privacy and distribution-quality release built on the complete 0.0.4 desktop feature line.

- **Windows lifecycle hardening** — profile persistence, file mutations and Remote Edit sessions reject duplicate/re-entrant operations and preserve application shutdown semantics across nested modal loops.
- **Android lifecycle hardening** — in-flight FTP/FTPS connections are owned by the current Activity, aborted during destruction and prevented from reviving stale UI/session state.
- **Android authentication privacy** — server-controlled failed-authentication replies are sanitized before reaching user-facing exceptions.
- **macOS native parity** — the maintained AppKit frontend uses the shared engine and has a complete native development action inventory; universal development builds are continuously validated.
- **Browser connection helper** — Chromium-family and Firefox source packages parse supported `ftp://`, `ftps://` and `sftp://` targets locally and can copy a safe target; desktop launch/handoff is not yet a supported contract.
- **Release hardening** — official public Windows publication now requires trusted Authenticode and fails closed when production signing material or signature verification is unavailable.
- **Release documentation accuracy** — current public distribution remains **14 platform artifacts / 17 public files** for Windows/Linux.

## Download Ghost FTP 0.0.5

### Windows

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Both public Windows files are self-contained x86-compatible universal bootstraps carrying verified native x64/x86 Ghost FTP payloads. The selected native payload is chosen from Windows native architecture information, verified before execution and requires no runtime download.

**Official Windows publication is signed-only.** `Publish Ghost FTP` requires the protected trusted Authenticode identity, verifies both finalized executables and accepts only:

```text
WINDOWS_AUTHENTICODE=signed
```

Local development and ordinary CI Windows builds may be unsigned, but those outputs are not official public release artifacts.

### Linux

Canonical 0.0.5 Linux files are built by `linux/BUILD-DISTROS.sh`:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

Matching Debian, Ubuntu, Fedora and Portable variants reuse one compiled production executable per architecture, and release CI compares extracted executable bytes. Native package-manager/runtime/GUI verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. Additional canonical architectures retain exact-head build, package-metadata, extraction and binary-parity coverage without an unsupported native-install claim.

See [Installation](docs/INSTALLATION.md), [Linux documentation](linux/README.md) and [Testing](docs/TESTING.md).

## Android development APK

Android is an active native development surface tied to root `VERSION`. CI builds and verifies:

```text
Ghost-FTP-Android.apk
```

The development app exposes Files, Sites, Bookmarks, Transfers, Settings and About with FTP + strict explicit FTPS and Storage Access Framework-scoped local access. Android SFTP remains intentionally hidden until strict native host-key identity verification has a maintained implementation.

The APK is not part of the public Windows/Linux 17-file release allow-list and is not represented as a production-signed public Android release.

## macOS native development app

macOS is an active native source/development surface using the shared `internal/api.Engine` through an AppKit frontend. The maintained development build produces a universal Intel + Apple Silicon app and validates the complete native action inventory.

The development ZIP is ad-hoc signed for CI/native validation. Public macOS distribution requires the separate fail-closed Developer ID + Apple notarization path documented in [`macos/README.md`](macos/README.md). A successful development build is **not** represented as proof of production signing/notarization.

macOS is not part of the current 17-file public Windows/Linux release allow-list.

## Browser connection helper

The `ekstenzije/` source tree contains privacy-minimal companion packages for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox.

Current supported behavior is deliberately narrow:

- explicitly entered/pasted `ftp://`, `ftps://` and `sftp://` targets are parsed locally;
- embedded credentials are rejected/removed from the safe target path according to the companion contract;
- users can explicitly copy the safe target;
- no FTP credentials are persisted;
- no remote executable code is loaded;
- no ordinary-tab scraping or broad host permission is required;
- **the helper does not currently launch Ghost FTP and no browser-to-desktop handoff contract is implemented.**

Browser companion packages are source/development surfaces and are not counted as public desktop release artifacts.

See [`ekstenzije/README.md`](ekstenzije/README.md) and [`ekstenzije/PRIVACY.md`](ekstenzije/PRIVACY.md).

## Verify what you download

The public release identity is:

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
CHANNEL=Current
prerelease=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` binds version, tag, exact source commit, platform set, Windows signing state, language count and package shape to the verified release assembly.

For official Windows Setup and Portable files, verify both SHA-256 and trusted Authenticode. A missing or invalid Windows signature does not satisfy the current official release contract.

The same verified release directory is published as a distribution-only bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The GHCR object is not a supported runtime container. The release workflow validates the exact 17-file allow-list immediately and again after a delay. Latest-only retention removes superseded Ghost FTP public releases/tags/canonical release branches/package versions only after a successor has been successfully published and verified. `main` history is not rewritten.

See [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Signing](docs/SIGNING.md) and [Versioning](docs/VERSIONING.md).

## Build from source

Ghost FTP uses Go **1.27.1**. Root `VERSION` is the authoritative production version source.

```text
go telemetry off
go test -race ./...
go vet ./...
```

Windows release-style packages:

```powershell
.\BUILD-WINDOWS.ps1
```

Canonical Linux release packages:

```bash
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

Native macOS development app:

```bash
bash macos/BUILD.sh
```

Android development APK is built through the maintained Android Gradle/CI contract documented under `android/` and `.github/workflows/android-apk.yml`.

## Quality gates

The maintained CI/release path checks:

- Go formatting, race tests, unit/integration tests and vet;
- repository, platform, dependency and localization policy;
- security, privacy, version, documentation and release audits;
- the complete Python regression suite;
- Windows universal Setup/Portable packaging and signing-pipeline mechanics;
- Linux architecture/package compatibility and canonical distro packaging;
- native distro installation/runtime smoke on the maintained x86-64 matrix;
- Android JVM regressions, lint, installable APK and authentic emulator evidence;
- native universal macOS development-app build/validation;
- CodeQL and Govulncheck;
- exact source identity, public artifact allow-list, SHA-256, GitHub Release read-back, GHCR read-back and retention for public publication.

A green workflow for an older commit does not satisfy a newer candidate. Release-sensitive work follows exact-head and post-merge verification.

## Documentation

Start with the [documentation index](docs/README.md). Core references include:

- [Architecture](docs/ARCHITECTURE.md)
- [Installation](docs/INSTALLATION.md)
- [Settings](docs/SETTINGS.md)
- [Reference UI](docs/REFERENCE-UI.md)
- [Localization](docs/LOCALIZATION.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Signing](docs/SIGNING.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [GitHub Packages](docs/PACKAGES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Versioning](docs/VERSIONING.md)
- [Roadmap](docs/ROADMAP.md)
- [Support](docs/SUPPORT.md)
- [Contributing](docs/CONTRIBUTING.md)
- [macOS development](macos/README.md)
- [Browser helper](ekstenzije/README.md)

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **Ghost FTP**. All rights reserved.
