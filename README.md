# Ghost FTP

<p align="center">
  <img src="build/icon.png" alt="Ghost FTP" width="148">
</p>

<h3 align="center">Your servers. Your files. Your control.</h3>

<p align="center">
A privacy-first FTP, explicit FTPS and SFTP workspace for direct professional file transfer — without telemetry, advertising, mandatory product accounts or a hidden storage cloud.
</p>

<p align="center">
  <a href="https://ghostftp.com"><strong>Website</strong></a> ·
  <a href="docs/INSTALLATION.md"><strong>Install</strong></a> ·
  <a href="docs/SECURITY.md"><strong>Security</strong></a> ·
  <a href="docs/PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="docs/README.md"><strong>Documentation</strong></a>
</p>

---

## File transfer without a product cloud in the middle

Ghost FTP is built for users who want a capable file-transfer workspace without turning server credentials, files or activity into somebody else's product.

- Connect to infrastructure you control or choose.
- Keep native transfer traffic between your device and the selected server.
- Use real file operations, transfer controls and protocol security boundaries.
- No application telemetry or behavioral analytics.
- No advertising.
- No mandatory Ghost FTP account.
- No automatic crash-upload backend.

## Maintained product surfaces

| Platform | Status | Maintained scope |
| --- | --- | --- |
| **Windows** | Public release target / reference desktop | Universal Setup + Portable with x64, x86 and ARM64 payloads |
| **Linux** | Public release target | Debian, Ubuntu and Fedora Installer + Portable bundles |
| **Android** | Public release target | Native application with FTP and strict explicit FTPS |
| **macOS** | Active development | Native AppKit frontend; public distribution waits for trusted signing/notarization |
| **Browser helpers** | Public release target | Chrome, Edge, Firefox and Opera companion packages |

The retired repository-hosted website application and browser protocol client are no longer part of this source tree. The public product website remains available at ghostftp.com, while transfer functionality in this repository is maintained in the native applications and narrowly scoped browser helpers.

## Core capabilities

Desktop workflows include:

- FTP, explicit FTPS and strict desktop SFTP
- local and remote navigation
- upload and download
- transfer queue lifecycle and cancellation
- retry and priority ordering
- filtering, sorting and bounded search
- directory comparison
- bookmarks and start directories
- create, rename and delete operations
- CHMOD where supported
- bounded Remote Edit

Android provides platform-appropriate Files, Sites, Bookmarks, Transfers, Settings and About surfaces with Storage Access Framework local authority. Android SFTP remains hidden until strict maintained host-key verification exists.

## Security model

Traditional FTP is supported as an intentional unencrypted compatibility mode. Explicit FTPS validates certificate and hostname identity and does not silently downgrade to plain FTP. Desktop SFTP uses strict server host-key trust/pinning.

Downloads, destructive local operations, external SSH helpers, protected credentials and release signing are guarded by dedicated security contracts and regression tests. See [Security](docs/SECURITY.md).

## Privacy model

Ghost FTP does not require a product account and contains no application telemetry, behavioral analytics, advertising, fingerprinting, hidden synchronization or automatic crash uploading.

Native transfer traffic is user-directed to the server selected by the user. Browser helper packages are deliberately narrow and have **no supported browser-to-desktop** protocol handoff. See [Privacy](docs/PRIVACY.md).

## Authentic UI evidence

Repository-local runtime screenshots document maintained Windows, Linux and Android surfaces. Generated mockups are not treated as execution evidence.

See [Reference UI](docs/REFERENCE-UI.md) for the maintained evidence contract.

## 24 desktop languages

Ghost FTP maintains 24 selectable desktop languages, with English as canonical fallback. Localization claims remain platform-specific and are enforced by repository audits.

## Ghost FTP 0.0.6

Current source version: **0.0.6**

- Release channel: **Current**
- Development status: **Active**
- Last actually published GitHub Release: **0.0.5**
- Next public release target: `ghostftp-v0.0.6`
- GitHub release prerelease flag: `prerelease=false`
- Current container bundle: `ghcr.io/bren-wp/ghost-ftp:0.0.6`
- Public release contract: **13 platform artifacts / 16 public files**

### Windows

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

### Linux

```text
Ghost-FTP-0.0.6-Linux-Debian-Installer.run
Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run
Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz
Ghost-FTP-0.0.6-Linux-Fedora-Installer.run
Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz
```

### Android

```text
Ghost-FTP-0.0.6-Android.apk
```

Public Android publication requires exact protected `GHOSTFTP_ANDROID_CERT_SHA256` signer verification.

### Browser helpers

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

The helpers use a shared local runtime with browser-specific manifests and retain **no supported browser-to-desktop** launch/handoff behavior.

### Release metadata

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Quality gates

Release-candidate verification includes:

```text
gofmt
go test -race ./...
go vet ./...
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_dependencies.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

A release is accepted only from the exact final head SHA that passed the required gates. Development/self-signed identities are not substitutes for production signing.

## Documentation

Start with [docs/README.md](docs/README.md) for architecture, installation, security, privacy, signing, testing, platform parity and release verification.

## License

Ghost FTP is proprietary commercial software and a copyrighted work of **Brendigo LTD**. Source visibility does not grant a general right to modify, redistribute, sublicense, rebrand or white-label the project. See [LICENSE](LICENSE).
