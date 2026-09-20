# Ghost FTP Repository Structure

Ghost FTP keeps the repository product-oriented and easy to scan. Public/source-root naming is **Ghost FTP / GhostFTP** first; framework-specific names are confined to internal locations where changing them would create unnecessary build risk.

## Top-level layout

```text
Ghost-FTP-Premium/
├── ghostftp-desktop/      Production desktop application source
├── ghostftp-runtime/      Developer / compatibility runtime tooling
├── ghostftp-installer/    Installer support and compatibility tooling
├── ghostftp-web/          ghostftp.com website source
├── ghostftp-updates/      Update-manifest templates
├── docs/                  Product, QA, build, legal and audit documentation
├── .github/workflows/     GhostFTP quality, build and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

## Source-root responsibilities

### `ghostftp-desktop/`

Authoritative production GUI and protocol engine. It contains the React/TypeScript interface, Rust backend, file-transfer engine, native window integration, updater, credential handling, agent components and packaging configuration.

### `ghostftp-runtime/`

Developer and compatibility tooling. It is not accepted as the production desktop GUI and must never replace the native Ghost FTP release path.

### `ghostftp-installer/`

Installer support / compatibility tooling retained for development and historical flows. Public Windows production packaging is generated from the authoritative desktop build.

### `ghostftp-web/`

Static ghostftp.com website source and local website icon assets.

### `ghostftp-updates/`

Update-manifest templates and release update metadata helpers.

### `docs/`

Documentation is grouped by purpose instead of accumulating duplicate files in the repository root.

```text
docs/
├── assets/        Brand and visual QA media
├── architecture/ Repository structure and naming rules
├── audits/        Code, security, language, branding and project audits
├── build/         Authoritative build status and runtime notes
├── guides/        Installation, updates, support and uninstall
├── legal/         Privacy and third-party notices
├── product/       Features, status, roadmap and UI/UX contract
├── qa/            Interaction, installer, transfer, responsive and pixel QA
└── releases/      Release notes and release-specific records
```

## Naming policy

Use **Ghost FTP** for user-facing copy and **GhostFTP** for technical filenames/artifacts.

Examples:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC9.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC9.exe`
- `GhostFTP-Linux-x86_64-v2.1.1-RC9.AppImage`
- `GhostFTP-v2.1.1-RC9-Desktop-Source.zip`

See [NAMING.md](NAMING.md) for the full policy.

## Internal framework names

The outer project directory is deliberately branded as `ghostftp-desktop/`.

A small set of internal framework-required names remains inside it, such as:

- `src-tauri/`
- `tauri.conf.json`
- framework package names under `@tauri-apps/*`

Those names are implementation requirements, not public product naming. They are kept stable to avoid breaking tooling, packaging, updater integration and CI.

## Stability principle

Repository cleanup must not break persisted product identifiers or installed-client compatibility. Bundle IDs, updater identifiers, deep-link schemes and data locations are treated as migration-sensitive even when folders and documentation are reorganized.

The root remains intentionally small and product-oriented.
