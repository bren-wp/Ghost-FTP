# Ghost FTP platform parity

Ghost FTP **0.0.6** treats parity as equivalent real capability with honest platform-native UX, not as a promise that every control exists identically on every operating system.

## Windows and Linux desktop parity

Windows and Linux use the same typed `internal/api.Engine` and shared protocol, transfer, profile, settings, localization, Remote Edit and security layers. Both expose FTP, explicit FTPS, desktop SFTP, navigation, transfers, remote file operations, filtering/sorting/search/comparison, bookmarks/start directories and Remote Edit where the underlying platform contract supports them.

Both desktop frontends expose actual queue lifecycle state rather than decorative progress. User-facing controls must remain wired to real engine behavior and guarded against stale/re-entrant operations.

Windows public output is one universal Setup plus one universal Portable application, each containing x64, x86 and ARM64 payloads. Linux public output is Installer plus Portable bundles for Debian, Ubuntu and Fedora, each carrying amd64/arm64/i386 payloads.

## Android parity boundary

Android is a public 0.0.6 application with Android-native Files, Sites, Bookmarks, Transfers, Settings and About surfaces. It uses Storage Access Framework authority, FTP plus strict explicit FTPS, bounded file/search/comparison/Remote Edit behavior and Android lifecycle ownership.

Android SFTP remains hidden until strict native host-key identity verification exists. The UI must not advertise unavailable protocol support.

## macOS parity boundary

macOS is an active native AppKit development frontend over the shared engine. Source/build parity does not imply public distribution parity. It remains outside the public release until real Developer ID Application signing and Apple notarization succeed.

## Browser helper boundary

Chrome, Edge, Firefox and Opera helpers are public companion ZIPs built from a shared local runtime. They remain narrow parser/copy helpers with no credential store, no telemetry backend and **no supported browser-to-desktop** launch/handoff. They are not protocol engines and must not present themselves as full native-client replacements.

## Theme and user experience

Windows remains the canonical desktop visual reference. Each platform may use native controls and navigation patterns, but naming, product identity, connection semantics, security warnings and capability claims must remain consistent and user-oriented.

Development labels, placeholder controls, debug diagnostics and unsupported feature promises do not belong in normal user-facing surfaces.

## Release parity

Every public release stage is independently required. A successful Windows build cannot substitute for failed Linux, Android or browser publication.

The Ghost FTP 0.0.6 release contract is **13 platform artifacts / 16 public files**:

- Windows: 2 platform artifacts;
- Linux: 6 platform artifacts;
- Android: 1 platform artifact;
- Browser helpers: 4 platform artifacts;
- release metadata: 3 additional public files.

macOS remains development/source only and does not enlarge the public release contract.

See [Architecture](ARCHITECTURE.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
