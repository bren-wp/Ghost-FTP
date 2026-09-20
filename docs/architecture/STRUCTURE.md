# Ghost FTP Repository Structure

Ghost FTP keeps user-facing names branded as **Ghost FTP** / **GhostFTP** while preserving a small number of framework-specific internal paths that are safer to leave stable.

## Top-level layout

```text
Ghost-FTP-Premium/
├── desktop-tauri/      production desktop application source
├── runtime/            compatibility/developer runtime tooling
├── installer/          installer development/support tooling
├── website/            ghostftp.com website source
├── docs/               product, QA, build, legal and audit documentation
├── .github/workflows/  source audit, native build and release automation
├── README.md
├── CHANGELOG.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

## Naming policy

Use **Ghost FTP** in user-facing copy.

Use **GhostFTP** in:

- release filenames;
- package artifacts;
- technical product identifiers;
- generated archives;
- CI artifact labels.

Examples:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC9.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC9.exe`
- `GhostFTP-Linux-x86_64-v2.1.1-RC9.AppImage`
- `GhostFTP-v2.1.1-RC9-Desktop-Source.zip`

## Why some internal paths still mention the framework

The `desktop-tauri/` and `src-tauri/` directory names are build-system conventions already referenced by package scripts, Rust configuration, GitHub Actions and tool defaults.

Renaming those paths only to hide the framework name would create a large, low-value migration surface and could break:

- build scripts;
- updater/bundle configuration;
- CI cache paths;
- packaging commands;
- developer tooling;
- framework defaults.

For that reason, RC9 keeps those internal paths stable. Public files and documentation avoid framework-first naming wherever possible.

## Documentation layout

```text
docs/
├── assets/       screenshots and brand/reference media
├── audits/       code, security, language, branding and project audits
├── build/        build status and runtime notes
├── guides/       install, uninstall, support and update guidance
├── legal/        privacy and third-party notices
├── product/      implemented features, roadmap and UI/UX contract
├── qa/           interaction, installer, transfer, responsive and pixel QA
└── releases/     release notes and release checksums
```

The repository root is intentionally kept small and product-oriented.
