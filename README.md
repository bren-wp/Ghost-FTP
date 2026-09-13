# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="156">
</p>

<p align="center"><strong>Fast native file transfer. Direct to your server. No cloud middleman.</strong></p>

<p align="center">
  Ghost FTP is a privacy-first FTP, FTPS and SFTP workspace for people who manage real infrastructure and want native performance, explicit security boundaries and professional transfer control without telemetry or a mandatory product account.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/Ghost-FTP/releases"><strong>Download</strong></a> ·
  <a href="https://ghostftp.com"><strong>Website</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

---

## ⚡ Why Ghost FTP

| | What you get |
| --- | --- |
| **🚀 Native workflow** | Focused native desktop clients instead of a web app wrapped in a desktop shell. |
| **🔐 Security first** | Explicit FTPS verification, strict desktop SFTP host-key trust, safe path handling and no silent secure-to-plain downgrade. |
| **📦 Serious transfers** | File/tree transfers, queue control, retries, directional bandwidth limits, staged commit/rollback and truthful progress. |
| **✏️ Remote Edit** | Built-in bounded text editing with revision conflict detection and verified read-back. |
| **🧭 Server navigation** | Site Manager, bookmarks, start directories, filtering, recursive search and directory comparison. |
| **🛡️ Local-first privacy** | No application analytics, advertising, tracking backend, mandatory account or hidden Ghost FTP relay. |
| **🌍 Localized** | English-first local catalog with **24 selectable desktop languages**. |
| **✅ Verifiable releases** | SHA-256 metadata, exact-source CI gates and signed-only official Windows publication. |

Current Ghost FTP version: **0.0.5**  
Development status: **Active**  
Release channel: **Current**  
Public release platforms: **Windows and Linux**  
Active native source platforms: **Windows, Linux, Android and macOS**  
Public release identity: `ghostftp-v0.0.5`, `prerelease=false`  
Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.5`  
Public release shape: **14 platform artifacts / 17 public files**

> Ghost FTP keeps release claims narrow on purpose. Android and macOS are active native development/source surfaces, while the current public GitHub Release contract remains Windows + Linux.

## 🖥️ See the product

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

<table>
<tr>
<td width="50%" valign="top"><strong>🗂️ Site Manager</strong><br><br><img src="docs/images/ghost-ftp-site-manager.png" alt="Ghost FTP Site Manager"></td>
<td width="50%" valign="top"><strong>⚙️ Settings</strong><br><br><img src="docs/images/ghost-ftp-settings.png" alt="Ghost FTP Settings"></td>
</tr>
<tr>
<td colspan="2" align="center"><strong>ℹ️ Product identity</strong><br><br><img src="docs/images/ghost-ftp-about.png" alt="Ghost FTP About" width="620"></td>
</tr>
</table>

The logo and UI media rendered by this README are **repository-local assets**. No remote badge image, tracking pixel, icon CDN or remote webfont is required to render this page. Maintained runtime evidence comes from real Windows, Linux and Android application surfaces captured by exact-head CI; generated mockups are not treated as production evidence.

See [Reference UI](docs/REFERENCE-UI.md) for the screenshot and evidence contract.

## 🔌 Connect the way your server requires

Ghost FTP keeps protocol choice explicit:

- **FTP** — available only as an intentional unencrypted compatibility choice.
- **FTPS** — explicit TLS with certificate and hostname verification; a failed secure connection is never silently retried as FTP.
- **SFTP** — strict host-key verification/pinning plus password or private-key authentication on maintained desktop platforms.

Fresh desktop Quick Connect defaults to **explicit FTPS on port 21**. Saved Site Manager profiles use explicit credential-persistence consent rather than silently turning entered secrets into durable state.

## 📂 A real two-pane file workspace

The maintained desktop workflow includes Local and Remote panes with deterministic sorting, current-folder filtering, bounded recursive search, bookmarks, saved start directories and conservative directory comparison.

File-management actions are backed by authoritative runtime state. Create folder, rename, delete and remote permissions revalidate the active path/selection instead of trusting stale visible text. Account-bound remote navigation is revalidated before a saved path becomes authoritative on a new session.

## 📦 Transfer control built for real workloads

Ghost FTP exposes real transfer lifecycle state instead of decorative progress UI:

- single-file and recursive directory-tree upload/download;
- pause, resume, cancel, retry and clear finished;
- queued **Top / Up / Down / Bottom** ordering;
- truthful bytes, speed and ETA from runtime transfer state;
- independent aggregate upload/download limits in KiB/s;
- bounded retry and timeout policy;
- **Skip**, **Replace** and **Replace + recovery backup** conflict handling;
- staged activation/rollback where the maintained transfer path requires it;
- connection/session generation ownership so stale workers cannot publish into a newer session.

## ✏️ Remote Edit without breaking transfer safety

Built-in **Remote Edit** is intentionally bounded. It supports regular text files through a controlled edit lifecycle with UTF-8/binary validation, size limits, line-ending preservation, revision conflict detection, verified upload/read-back and permission preservation when trustworthy metadata exists.

The visible workflow stays simple: **Open → Edit → Save / Reload / Close**. The underlying implementation does not bypass the normal transport/security boundary just because the user is editing text.

## 🔐 Security is a product boundary

Ghost FTP treats security-sensitive behavior as typed runtime policy:

- FTPS certificate and hostname validation;
- strict desktop SFTP host-key trust/pinning;
- no silent FTPS-to-FTP fallback;
- protected saved-secret lifetime and explicit persistence consent;
- local path confinement and symlink/reparse safety;
- staged transfer commit/rollback;
- privacy-safe diagnostics instead of replaying raw credential-bearing tool/server output;
- trusted Linux transport/AskPass provenance;
- exact-object Windows cleanup/ownership checks;
- fail-closed official Windows Authenticode publication.

Android 0.0.5 also sanitizes FTP authentication failures so server-controlled USER/PASS replies cannot echo a submitted password into user-facing exceptions.

Read [Security](docs/SECURITY.md), [Architecture](docs/ARCHITECTURE.md) and [Signing](docs/SIGNING.md).

## 🛡️ Privacy without a hidden backend

Ghost FTP includes no application telemetry, advertising, fingerprinting, tracking backend, automatic crash upload, mandatory Ghost FTP account or hidden profile synchronization service.

User-directed FTP/FTPS/SFTP traffic goes to the server the user configured. Profiles and settings stay local. Production workflows explicitly disable Go telemetry. The browser connection helper parses only explicitly entered targets and does not scrape ordinary tabs, persist FTP credentials or send them to a Ghost FTP service.

Read the complete [Privacy](docs/PRIVACY.md) contract.

## 🌍 24 local desktop languages

**English** is the canonical default/fallback. The maintained local desktop catalog exposes 24 selectable languages:

English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.

Localization is local; Ghost FTP does not need an online translation service to render the UI. See [Localization](docs/LOCALIZATION.md).

## 🧩 Platform status

| Platform | Status | Current contract |
| --- | --- | --- |
| **🪟 Windows** | **Public release** | Universal Setup + Portable; internal x64/x86 payload validation; official publication requires trusted Authenticode. |
| **🐧 Linux** | **Public release** | Debian, Ubuntu, Fedora and Portable families across the canonical architecture set. |
| **🤖 Android** | Active development | Installable development/debug-signed APK; FTP + strict explicit FTPS; SAF-scoped local files; SFTP hidden until verified native host-key identity exists. |
| **🍎 macOS** | Active development | Native AppKit frontend using the shared `internal/api.Engine`; universal development app; separate fail-closed Developer ID/notarization path. |
| **🌐 Browser helper** | Source companion | Manifest V3 local parser/copy helper; no network/storage permissions and no supported browser-to-desktop launch/handoff today. |

Android, macOS and browser-helper source are **not** silently counted in the current Windows/Linux **14 platform artifacts / 17 public files** release allow-list.

## 🆕 Ghost FTP 0.0.5 highlights

Version 0.0.5 focuses on lifecycle correctness, privacy hardening and distribution truthfulness while retaining the established desktop feature set.

- **Windows lifecycle hardening** — profile persistence, file mutations and Remote Edit reject duplicate/re-entrant operations and preserve bounded modal lifecycle behavior.
- **Android lifecycle ownership** — pending FTP/FTPS connections belong to the current Activity and stale callbacks cannot revive obsolete UI/session state.
- **Android auth privacy** — failed authentication discards raw server-controlled replies before user-facing errors are created.
- **macOS native parity** — the AppKit frontend is wired to the shared engine and the maintained action inventory is complete for development validation.
- **Signed-only Windows publication** — official public Setup/Portable publication fails closed when the production Authenticode identity or valid final signatures are unavailable.
- **Release/documentation consistency** — the current public contract stays explicitly Windows/Linux with **14 platform artifacts / 17 public files**.

See [Changelog](CHANGELOG.md) and [Release history](docs/RELEASE-HISTORY.md).

## ⬇️ Download Ghost FTP 0.0.5

### Windows

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Official Windows publication is signed-only and records:

```text
WINDOWS_AUTHENTICODE=signed
```

Local development and ordinary CI Windows builds may remain unsigned, but those outputs are not official public release artifacts.

### Linux

Canonical distro-specific files include:

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

Native package-manager/runtime/GUI verification is maintained for **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. Other canonical architectures retain exact-head build, metadata, extraction and binary-parity verification without an unsupported native-install claim.

See [Installation](docs/INSTALLATION.md) and [Linux](linux/README.md).

## 🤖 Android development APK

Android derives its development identity from root `VERSION` and CI builds the installable development artifact:

```text
Ghost-FTP-Android.apk
```

The current Android surface contains Files, Sites, Bookmarks, Transfers, Settings and About, supports FTP + strict explicit FTPS, uses Android Storage Access Framework for local files and keeps SFTP hidden until equivalent strict host-key verification is implemented and tested.

The APK is not a production-signed public Android release and is outside the 17-file public desktop release allow-list. See [`android/README.md`](android/README.md).

## 🍎 macOS native development app

macOS is an active native AppKit development/source surface using the shared `internal/api.Engine`. The maintained build produces a universal Intel + Apple Silicon development app.

A development build is **not** proof of production distribution readiness. Public macOS distribution requires the separate Developer ID + Apple notarization path documented in [`macos/README.md`](macos/README.md). Ghost FTP does not claim successful production signing/notarization unless that credentialed path actually succeeds.

macOS is outside the current 17-file public Windows/Linux release allow-list.

## 🌐 Browser connection helper

`ekstenzije/` contains privacy-minimal Manifest V3 helper source for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox.

It can parse explicitly supplied `ftp://`, `ftps://` and `sftp://` targets locally and copy a credential-stripped safe target. It requests no broad host, tab, history, storage, scripting or network permissions. **The helper does not currently launch Ghost FTP and no supported browser-to-desktop handoff contract is implemented.**

See [`ekstenzije/README.md`](ekstenzije/README.md) and [`ekstenzije/PRIVACY.md`](ekstenzije/PRIVACY.md).

## ✅ Verify every download

Canonical public release identity:

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
CHANNEL=Current
prerelease=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` binds the release version, tag, exact source commit, public platform set, Windows signing state and release shape to the verified assembly.

For official Windows Setup and Portable, verify both SHA-256 and trusted Authenticode. A missing or invalid Windows signature is a release-integrity failure under the current public contract.

The verified release directory is also published as the distribution-only bundle:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The GHCR object is **not** a supported runtime container.

See [Release verification](docs/RELEASE-VERIFICATION.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [Packages](docs/PACKAGES.md) and [Versioning](docs/VERSIONING.md).

## 🧪 Quality gates

A release candidate is expected to pass the repository's layered gates, including:

```text
gofmt
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_platforms.py
python scripts/audit_dependencies.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

Native/package gates additionally validate Windows, canonical Linux distro packages, the Android development APK, the universal macOS development app and authentic runtime UI evidence where that evidence workflow applies.

A PR is merge-ready only when required workflows for its **exact final head** are successful. Release work then requires successful required post-merge `main` checks on the exact merge SHA.

## 📚 Documentation

Start with the [documentation index](docs/README.md):

- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Signing](docs/SIGNING.md)
- [Testing](docs/TESTING.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Localization](docs/LOCALIZATION.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Versioning](docs/VERSIONING.md)
- [Support](docs/SUPPORT.md)

Platform-specific development documentation lives in [`android/`](android/README.md), [`macos/`](macos/README.md), [`linux/`](linux/README.md) and [`ekstenzije/`](ekstenzije/README.md).

## 🤝 Contributing

Changes should preserve the shared engine as the source of truth, keep visible controls connected to real runtime behavior, avoid weakening privacy/security boundaries, and update documentation/tests when a public contract changes.

See [Contributing](docs/CONTRIBUTING.md).

---

<p align="center"><strong>Ghost FTP — direct file transfer with native control and verifiable boundaries.</strong></p>
