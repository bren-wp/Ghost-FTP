# Ghost FTP platform parity — 0.92.0

Last reviewed: 10 October 2026. Latest **verified published release at the start of this feature**: **v0.91.1**. v0.92.0 is unreleased until all exact-SHA CI, merged-main gates and the canonical release workflow pass. This matrix describes source-level availability, not installed-build 1:1 or production signing across all platforms.

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

## Reference and platform differences

- User-supplied design ZIP includes **75 concept images**: **30 Windows, 29 Linux, 16 Android**. macOS has no supplied reference screenshots. These are visual targets, not real captures or evidence of operational interactions.
- v0.91.1 Windows/Linux Site Manager protects drafts against accidental dismissal; Android improves username-only URL privacy; macOS retains corrupt profile bytes and checks imported identity collisions. The changes improve real behavior but do not establish identical UI implementations.
- Keep the 75-screen per-screen acceptance entries pending until actual installed app captures, input interactions, keyboard/a11y checks and matching-scale comparisons are documented.

## 0.92.0 macOS source improvement

macOS now includes an advisory matching-endpoint filter with a duplicate count in the Sites workspace and a live warning while editing saved connection settings. This matches Windows/Linux identity-detection semantics without accessing Keychain secrets or modifying stored entries. It does not make Android profile inventory equivalent, enable macOS FTPS/SFTP operations, or certify the 75 concept-reference screenshots as pixel-perfect installed-device captures.
