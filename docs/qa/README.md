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
