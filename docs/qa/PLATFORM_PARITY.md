# Ghost FTP platform parity — 0.30.13

Last reviewed: 8 October 2026. The latest **published** canonical version is 0.30.10; 0.30.11 and 0.30.12 were merged and CI-verified but not released. This matrix describes source evidence, not a claim of new release publication.

| Gate or capability | Windows | Linux | Android | macOS |
| --- | --- | --- | --- | --- |
| Application source | Tauri/React/Rust | Shared desktop Tauri/React/Rust | Kotlin application | SwiftUI application |
| Native CI build | Verified Windows build | Verified Ubuntu build | Verified Gradle builds | Verified Swift build |
| Package form | Portable EXE, NSIS, MSI | AppImage, DEB, RPM, binary | Unsigned production APK and separately signed CI Preview | Ad-hoc-signed Preview ZIP |
| Install/launch validation | Native QA and setup lifecycle smoke | Package lifecycle smoke on Ubuntu | Emulator install, reinstall, launcher and click-through | Build/package verified; production install/notarization unverified |
| Plain FTP transfer | Implemented + shared protocol E2E | Implemented + shared protocol E2E | Implemented; Android checks and emulator smoke | Implemented; 0.30.13 corrects RETR/STOR and staging naming |
| Explicit FTPS | Implemented + protocol E2E | Implemented + protocol E2E | Implemented with hostname validation | **Blocked** until verified AUTH TLS/certificate/hostname and protected data channels |
| SFTP | Implemented + protocol E2E | Implemented + protocol E2E | Implemented with pinned host-key SHA-256 | **Blocked** until verified SSH/SFTP engine and host-key policy |
| Workspace shell | Files, Sites, Transfers, Sync & Backup, Settings, Help & About | Same shared shell | Six workspaces | Six workspaces, not feature-complete |
| Durable transfer recovery | Implemented with SQLite transfer ledger | Same shared engine | **Partial**, no desktop-equivalent durable recovery | Credential-free bounded history, **not** full retry/resume engine |
| Stable production signing | Windows production installer availability depends on code-signing deployment | Binary packages built; package-signing/distribution coverage varies | **Blocked** until persistent release keystore and upgrade proof | **Blocked** until Developer ID and notarization |

## Release acceptance rules

An application is not FINAL solely because its sources compile or CI is green. A release requires the exact current commit to pass Quality, real FTP/FTPS/SFTP protocol E2E, Windows/Linux native builds, Android instrumentation, Windows hardening and macOS unit/build/packaging. This proves only the tested targets and behaviors. It does not prove macOS FTPS/SFTP, Android cross-release signing, broad Linux distribution parity or notarized macOS installation.

The previous public release must be read from GitHub Releases rather than inferred from the most recently merged source. Preserve missing features as explicit blockers; never implement insecure fallbacks to claim parity.
