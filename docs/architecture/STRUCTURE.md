# Ghost FTP Repository Structure

Ghost FTP keeps the repository product-oriented. Public/source-root naming is **Ghost FTP / GhostFTP** first; framework-specific names are confined to internal locations where changing them would create build risk.

## Top-level layout

```text
Ghost-FTP/
├── android/                   Native Android application
├── ghostftp-desktop/          Production Windows/Linux desktop source
├── tools/
│   ├── ghostftp-runtime/      Support/compatibility runtime tooling
│   └── ghostftp-installer/    Support/compatibility installer tooling
├── website/                   ghostftp.com website source
├── updates/                   Update-manifest templates/tools
├── docs/                      Product, QA, build, legal and historical docs
├── version.json               Canonical active version/build metadata
├── .github/workflows/         Quality, build, Android and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

## Source-root responsibilities

### `ghostftp-desktop/`

Authoritative production desktop GUI/protocol engine: React/TypeScript UI, Rust backend, file-transfer engine, native window integration, updater, credential handling, agent components and packaging.

### `android/`

Authoritative native Android source and mobile protocol/file workflows.

### `tools/ghostftp-runtime/`

Support/compatibility tooling. It must not replace or be described as the production desktop GUI.

### `tools/ghostftp-installer/`

Support/compatibility installer tooling. Production Windows end-user packaging comes from the native desktop/Tauri NSIS build.

### `website/`

Static ghostftp.com source and local assets.

### `updates/`

Update-manifest templates and release metadata helpers.

### `docs/`

Documentation grouped by purpose:

```text
docs/
├── assets/        Brand and visual QA media
├── architecture/ Repository structure/naming rules
├── audits/        Historical point-in-time audits
├── build/         Current build status/runtime notes
├── development/   Source-build guidance
├── guides/        Installation, updates, support and uninstall
├── legal/         Privacy and third-party notices
├── mobile/        Android parity/contract
├── product/       Features, status, roadmap and UI/UX contract
├── qa/            Interaction, installer, protocol, responsive and visual QA
├── release/       Current canonical release process
└── releases/      Canonical release notes + historical provenance
```

## Naming policy

Use **Ghost FTP** for user-facing copy and **GhostFTP** for technical/public artifact filenames.

Examples:

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`
- `GhostFTP-Linux-x86_64-v<version>.AppImage`
- `GhostFTP-Android-v<version>.apk`

See [NAMING.md](NAMING.md).

## Internal framework names

Framework-required names remain where required, such as `src-tauri/`, `tauri.conf.json` and `@tauri-apps/*`.

## Stability principle

Repository cleanup must not break persisted product identifiers or installed-client compatibility. Bundle IDs, updater identifiers, deep-link schemes and data locations are migration-sensitive.
