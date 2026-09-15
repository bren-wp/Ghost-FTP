# Ghost FTP platform parity

Ghost FTP **0.0.6** treats parity as equivalent real capability with honest platform-native UX, not pixel-copying controls that the platform cannot support.

## Windows and Linux

Windows and Linux use the same typed `internal/api.Engine` and shared protocol, transfer, profile, settings, localization, Remote Edit and security layers. Both expose real connection/navigation/file operations and queue lifecycle backed by actual engine state.

Windows remains the canonical desktop visual reference. Public Windows output is one universal Setup + one universal Portable, each containing x64/x86/ARM64 payloads. Linux publishes Installer + Portable bundles for Debian, Ubuntu and Fedora with amd64/arm64/i386 payloads.

## Android

Android provides native Files, Sites, Bookmarks, Transfers, Settings and About surfaces with Storage Access Framework authority. FTP and strict explicit FTPS are supported. SFTP remains hidden until strict maintained host-key verification exists.

Lifecycle, network loss, cancellation and background work must follow Android platform rules rather than copying desktop behavior blindly.

## macOS

macOS is an active native AppKit source surface over shared product logic. Source/build parity does not imply public distribution parity; Developer ID signing and Apple notarization remain required before a public macOS package can be claimed.

## Browser extensions

Chrome, Edge, Firefox and Opera share one product identity and a shared local UI/runtime. Browsers do not directly provide the same raw FTP/FTPS/SFTP socket capabilities as native desktop applications, so the 0.0.7 development surface reaches parity through a local Ghost FTP Native Messaging host backed by the existing typed Engine.

The browser workspace now exposes real connection management, saved native profiles, SFTP host-key confirmation, local/remote navigation, create/rename/delete, upload/download, transfer progress, cancellation, retry, pause/resume and disconnect. Those actions are routed into the same Engine and transfer queue rather than simulated in JavaScript.

Browser authority remains intentionally narrower than desktop authority:

- the only browser permission is `nativeMessaging`;
- local file access is rooted to the operating-system-selected folder;
- saved credentials stay in the native protected profile store, not extension storage;
- server traffic goes directly from the local native Engine to the selected server;
- native host registration is bound to explicit extension identities and never wildcard origins.

The bridge binaries are built and architecture-verified for Windows x64/x86/ARM64 and Linux amd64/i386/arm64 by the browser CI. This development build evidence does not retroactively alter the already-published 0.0.6 installer or release artifacts. Production installation/registration must remain gated until its platform installer transaction and official Chromium extension identities are available.

Fake connections, simulated transfers and remote Ghost FTP credential relays are not parity.

## Product language

Across all applications, use consistent names for connection fields, transfer states, confirmation dialogs, errors and actions. End-user surfaces should not expose internal development vocabulary, stack traces or implementation class/module names.

## Theme and accessibility

The maintained dark and soft-light palettes remain consistent across product surfaces. Keyboard navigation, focus order, visible focus states, contrast, hit areas, disabled/loading states and reduced-motion behavior are platform requirements rather than optional polish.

## Published release boundary

The published Ghost FTP 0.0.6 release contract is **13 platform artifacts / 16 public files** across Windows, Linux, Android, four browser packages and three release metadata files. The former repository website/Web FTP surfaces are no longer active source and are not part of future parity work.

See [Architecture](ARCHITECTURE.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
