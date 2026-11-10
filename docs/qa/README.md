> **Current development (10 November 2026): Ghost FTP 0.99.41 Preview.** Windows/Linux open directly in Files without an artificial splash; native QA validates seven real workspaces. Scheduled transfer checks also respond to window wake and connectivity restoration. Previous published edition: v0.99.40 (release verification required). Signing, live providers and 1:1 reference acceptance remain separate.

# Direct Files startup and native QA

Ghost FTP Windows/Linux startup opens the real Files workspace immediately. The retired splash is not an active view or a capture target; historical QA pixels are preserved solely for provenance. Seven current screenshot targets are Files, Sites, New Connection, Transfers, Settings, Sync and About. Capture artifacts remain evidence, not 1:1 visual acceptance. Native application launch, backend initialization and real network operations must still be validated independently.

# Ghost FTP desktop QA index

The supported desktop platforms are Windows, Linux and macOS. Run source-level, protocol-level and installed native-app tests on the same exact commit SHA.

## Source and security QA

- [Transfer protocols](PROTOCOL_E2E.md): FTP, explicit FTPS and SFTP actual localhost roundtrips, trust validation, resume/cancel/failure handling
- [Files, transfers and interactions](CLICK.md), [responsive layouts](RESPONSIVE.md), [keyboard/window integration](TITLEBAR.md)
- [Reference collection](PREMIUM_REFERENCE_75.md): active 59 Windows/Linux concepts (30 + 29), none accepted as genuine installed-app screenshots until evidence exists
- [Runtime evidence registry](PREMIUM_REFERENCE_EVIDENCE.json): screenshot SHA, source SHA, viewport and human acceptance
- [Cross-platform coverage](PLATFORM_PARITY.md) and [native pixel diff](V8_RGB_PIXEL_PRECISION_2026_10_26.md)

## Native acceptance gates

Windows: portable/NSIS/MSI installation, application launch, WebView2, key vault, upload/download, transfer queue and updates. Linux: AppImage/DEB/RPM, GTK/WebKit, keyring, Wayland/X11, file selection and transfers. macOS: SwiftUI workspace, Keychain, FTP and signed cloud transport; FTPS/SFTP remain disabled pending secure implementation.

Code checks and mockup comparisons are not certification for all actual systems, permissions, server types or 1:1 pixel parity.
