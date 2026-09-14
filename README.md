# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP application icon" width="156">
</p>

<p align="center"><strong>Fast native file transfer. Direct to your server. No cloud middleman.</strong></p>

<p align="center">
Ghost FTP is a privacy-first FTP/FTPS/SFTP workspace with explicit security boundaries, native desktop clients, a scoped Android client and privacy-minimal browser helpers.
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

- Native Windows and Linux workflows with a shared typed engine.
- FTP and strict explicit FTPS; maintained desktop SFTP uses strict host-key verification/pinning.
- Transfer queue lifecycle, retries, pause/resume/cancel, Top/Up/Down/Bottom priority and truthful runtime progress.
- Site Manager, bookmarks, start directories, filtering, bounded recursive search, directory comparison and Remote Edit.
- No application telemetry, advertising, tracking backend, mandatory account or hidden Ghost FTP relay.
- English-first local catalog with **24 selectable desktop languages**.
- Exact-source CI, SHA-256 metadata, trusted Windows Authenticode and protected Android production signing.

- Current Ghost FTP version: **0.0.6**
- Development status: **Active**
- Release channel: **Current**
- Public release platforms: **Windows, Linux and Android**, plus public Chrome/Edge/Firefox browser-helper packages
- Active native source platforms: **Windows, Linux, Android and macOS**
- Public release identity: `ghostftp-v0.0.6`, `prerelease=false`
- Verified distribution bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.6`
- Public release shape: **18 platform artifacts / 21 public files**

macOS remains a separately validated native development/source frontend. Browser packages remain local parser/copy helpers and do not create a browser-to-desktop launch contract.

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

The logo and UI media rendered by this README are **repository-local** assets. Maintained runtime evidence comes from real **Windows, Linux and Android** application surfaces captured by **exact-head** CI; generated **mockup** or image-generation output is not accepted as production evidence. See [Reference UI](docs/REFERENCE-UI.md).

## 🔌 Protocol and trust model

- **FTP** is an intentional unencrypted compatibility option.
- **FTPS** uses explicit TLS with certificate and hostname validation and no silent downgrade to FTP.
- **SFTP** is available on maintained desktop platforms only with strict host-key verification/pinning.
- **Android SFTP remains hidden** until a maintained strict host-key identity implementation exists and is tested. The 0.0.6 Android release does not weaken that boundary.

Fresh desktop Quick Connect defaults to explicit FTPS on port 21. Saved credentials require explicit persistence consent and remain in the protected local profile store.

## 📂 Workspace and transfer control

Windows and Linux expose local/remote panes, deterministic sorting, current-folder filtering, bounded recursive search, conservative directory comparison, synchronized navigation, bookmarks, start directories, file mutations and Remote Edit. Transfer lifecycle supports recursive trees, pause/resume/cancel/retry, clear finished, Top/Up/Down/Bottom queue ordering, bounded retries/timeouts, conflict policy and independent aggregate upload/download limits in `KiB/s`.

Android 0.0.6 exposes Files, Sites, Bookmarks, Transfers, Settings and About, FTP + strict explicit FTPS, SAF-scoped local storage, local/remote create/rename/delete, remote permissions and bounded transfer lifecycle safeguards. Android public signing does not imply unsupported desktop-only protocol parity.

## 🔐 Security and privacy boundaries

Ghost FTP preserves:

- FTPS certificate/hostname validation;
- strict desktop SFTP host-key trust/pinning;
- no silent secure-to-plain fallback;
- protected saved-secret lifetime and explicit persistence consent;
- local root/path confinement and safe mutation/transfer commit behavior;
- privacy-safe diagnostics rather than replaying credential-bearing server/tool output;
- fail-closed official Windows Authenticode publication;
- fail-closed Android production signing with an expected signer SHA-256 fingerprint;
- no application telemetry, analytics, ads, fingerprinting, automatic crash upload or hidden synchronization backend.

Read [Security](docs/SECURITY.md), [Signing](docs/SIGNING.md) and [Privacy](docs/PRIVACY.md).

## 🌍 24 local desktop languages

**English** is the canonical default/fallback. The maintained local desktop catalog exposes **24 selectable desktop languages**:

English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.

## 🧩 Platform status

| Platform | Status | Current contract |
| --- | --- | --- |
| **Windows** | **Public release** | Universal Setup + Portable containing native x64, x86 and ARM64 payloads; publication requires trusted Authenticode. |
| **Linux** | **Public release** | Debian, Ubuntu, Fedora and Portable package families across the canonical architecture set. |
| **Android** | **Public release** | Production-signed APK; FTP + strict explicit FTPS; SAF-scoped storage; SFTP intentionally hidden until strict host-key verification exists. |
| **macOS** | Active development | Native AppKit frontend using shared `internal/api.Engine`; universal development app only until real Developer ID + notarization succeeds. |
| **Browser helper** | **Public packages** | Deterministic Chrome, Edge and Firefox ZIPs; local parser/copy helper, no network/storage permissions and no supported desktop launch/handoff. |

The two public Windows executables do not multiply by CPU architecture. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` remains deliberate: ARM64 is cross-built and structurally verified, but native Windows ARM64 runtime execution is not claimed without maintained ARM64 runtime evidence.

## 🆕 Ghost FTP 0.0.6 highlights

Version 0.0.6 completes the next public distribution contract without weakening established security boundaries.

- **Android public release** — adds a production-signed `Ghost-FTP-0.0.6-Android.apk`; publication fails closed if the protected keystore, passwords, alias or expected signer SHA-256 fingerprint are unavailable or invalid.
- **Android parity and lifecycle hardening** — remote/local file mutations, transfer ownership, server-reply redaction, passive-data validation and authentic emulator evidence remain enforced.
- **Browser packages** — adds deterministic Chrome, Edge and Firefox release ZIPs while keeping the helper local-only and without desktop handoff.
- **Windows universal distribution** — preserves the two signed public EXEs with native x64/x86/ARM64 payloads and no runtime architecture download.
- **Linux canonical distribution** — preserves the twelve Debian/Ubuntu/Fedora/Portable artifacts and package/binary-parity verification.
- **Release integrity** — expands the canonical allow-list to **18 platform artifacts / 21 public files**, with exact asset read-back, SHA-256 binding, GHCR distribution bundle and latest-only retention.
- **macOS truthfulness** — macOS remains development-only until real Developer ID signing and Apple notarization are actually proven.

See [Changelog](CHANGELOG.md) and [Release history](docs/RELEASE-HISTORY.md).

## ⬇️ Download Ghost FTP 0.0.6

### Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Official Windows publication records:

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

### Linux

```text
Ghost-FTP-0.0.6-Linux-Debian-amd64.deb
Ghost-FTP-0.0.6-Linux-Debian-arm64.deb
Ghost-FTP-0.0.6-Linux-Debian-i386.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.6-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.6-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.6-Linux-Portable-i386.tar.gz
```

Native install/runtime/GUI evidence is maintained for Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. Other canonical architectures retain build, metadata, extraction and binary-parity verification without an unsupported native-runtime claim.

### Android

```text
Ghost-FTP-0.0.6-Android.apk
```

The APK must be production-signed by the protected Ghost FTP Android publisher identity and its certificate SHA-256 fingerprint must match the protected expected fingerprint before publication. The ordinary development artifact remains explicitly separate as `Ghost-FTP-Android-dev.apk`.

### Browser helper packages

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
```

These packages have **no supported browser-to-desktop** URI/native-messaging handoff. See [`ekstenzije/README.md`](ekstenzije/README.md).

### macOS development

macOS is not a 0.0.6 public release artifact. Public macOS distribution requires real Developer ID signing and Apple notarization; a local/ad-hoc development signature is not production evidence.

## ✅ Verify every download

Canonical public release identity:

```text
VERSION=0.0.6
TAG=ghostftp-v0.0.6
CHANNEL=Current
prerelease=false
PUBLIC_PLATFORM_ARTIFACTS=18
PUBLIC_RELEASE_FILES=21
```

Every public release includes `SHA256.txt`, `BUILD-METADATA.txt` and `RELEASE-NOTES.txt`. Metadata binds the exact source commit, Windows signing state, Android signer fingerprint, platform set and release shape. Verify Windows Authenticode and Android signer identity in addition to SHA-256 checks.

The verified release directory is also published as the distribution-only bundle:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.6
```

The GHCR object is a **distribution bundle**, **not a runtime container**.

See [Release verification](docs/RELEASE-VERIFICATION.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [Packages](docs/PACKAGES.md) and [Versioning](docs/VERSIONING.md).

## 🧪 Quality gates

A release candidate is expected to pass:

```text
gofmt
go test -race ./...
go vet ./...
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

Native/package gates additionally validate Windows, Linux packages, Android development/release build paths, macOS development and authentic Windows, Linux and Android runtime UI evidence. A PR is merge-ready only when workflows for its **exact final head** are successful. Publication is valid only when the exact merged `main` SHA passes post-merge verification and the protected signing release workflow succeeds.

<p align="center"><strong>Ghost FTP — direct file transfer with native control and verifiable boundaries.</strong></p>
