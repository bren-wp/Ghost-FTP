# Ghost FTP branding and icon audit — 0.99.11 Preview

Ghost FTP is the sole product name; artifact naming uses GhostFTP. The Brendigo identity and current official repository/support links remain authoritative. Bundle identifiers are `com.ghostftp.desktop` and `com.ghostftp.android`.

## Verified source-level identity controls

- Windows/Linux desktop retain the separately approved cyan accent and the shared 214 px navigation rail; their Tauri icon assets are checked for PNG image geometry, non-truncated content and ICO image-directory integrity during installation of build dependencies.
- Android uses the approved violet reference palette. The adaptive launcher, Android system splash and in-app brand all share the same reviewed Ghost silhouette and violet mark. The Android launcher background uses the approved midnight color.
- Android header navigation and standard action touch surfaces now have a 48 dp minimum target. The native splash and related screen-reader labels come from 24-language resources, with CI parity and placeholder checks.
- Both the Android and shared quality workflows run `tools/audits/check-cross-platform-branding.mjs`; it checks geometry, reviewed symbol shape, palette source values and supported reference inventory.
- Native macOS has no approved pixel-exact reference screen set. SwiftUI remains a separate platform-native implementation, and macOS FTPS/SFTP issue #223 remains open.

**Important:** source code, icon and token checks are *not* measured pixel-by-pixel acceptance. The 30 Windows, 29 Linux and 16 Android design concepts (75 total) require authentic installed-app captures at identical viewport size, scale and state. User data must never be fabricated to match a concept screenshot. The accepted screen count is still zero until comparison evidence has been reviewed.
