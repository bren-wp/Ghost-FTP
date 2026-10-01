# Ghost FTP Branding Audit — 0.20.9

Scope: current application source, installers, Android package metadata, release-facing documentation and compatibility tooling.

- Product-facing identity remains **Ghost FTP** / **GhostFTP** for artifact names.
- Canonical repository and support links point to `bren-wp/Ghost-FTP` and `ghostftp.com`.
- Desktop bundle identifier remains `com.ghostftp.desktop`; Android production package remains `com.ghostftp.android`.
- Framework/toolchain path names such as `src-tauri` are implementation details, not public product branding.
- The removed website application is not treated as an active product surface.
- Third-party product names are retained only where required for interoperability/imports.

Status: source branding is CI-scanned; target-OS visual acceptance remains a separate QA gate.
