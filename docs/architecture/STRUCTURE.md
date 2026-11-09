# Ghost FTP repository structure

**Active platforms:** Windows, Linux and macOS desktop. The standalone server, demo and CLI applications are retired. This is a desktop client repository.

```text
Ghost-FTP/
├── ghostftp-desktop/          Windows/Linux Tauri, React/TypeScript and Rust engine
├── macos/                     Native SwiftUI desktop client
├── tools/
│   ├── ghostftp-runtime/      Go compatibility/runtime helpers
│   └── ghostftp-installer/    Go compatibility installer helpers
├── updates/                   Signed desktop updater metadata and tooling
├── assets/                    Shared desktop branding
├── docs/                      Product, QA, security, legal and release documentation
├── .github/                   CI, release governance and audit scripts
├── version.json               Canonical release metadata
├── README.md
├── SECURITY.md
├── LICENSE.txt
└── EULA.txt
```

## Source ownership

`ghostftp-desktop/src-tauri` owns the Windows/Linux native transfer engine, session security, key handling, data protection and app packaging. `ghostftp-desktop/src` owns the desktop UI. `macos/` contains the independently maintained native SwiftUI client. `tools/` contains compatibility helpers and is **not** the public desktop GUI. `updates/` contains the signed-update service contract, not a separate public website.

## Expected desktop packages

Windows x64: portable EXE, NSIS Setup EXE and MSI. Linux x86_64: native binary, AppImage, DEB and RPM. macOS: architecture-specific app ZIP/DMG Preview where built. Every release package must be verified against the same exact-source commit and SHA-256 manifest. Signing/notarization requirements remain separate from installability.

Historical release archives are retained as immutable evidence, not active platform definitions. Do not rename persistent app IDs, updater keys, deep-link schemes, profile or keychain locations without migration tests.
