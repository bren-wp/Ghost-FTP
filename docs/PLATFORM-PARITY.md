# Ghost FTP platform parity

Ghost FTP **0.0.6** treats parity as **equivalent real capability with honest platform-native UX**, not pixel-copying unsupported controls.

## Windows and Linux desktop parity

Windows and Linux use the same typed `internal/api.Engine` and shared protocol, transfer, profile, settings, localization, Remote Edit and security layers. Both expose FTP, explicit FTPS, SFTP password/private-key flows, strict host-key trust, navigation, transfers, remote file operations, filters/sorting/search/comparison, bookmarks/start directories and Remote Edit.

Both desktop frontends expose real queue lifecycle including pause/resume where supported, cancel, retry, clear and queued Top/Up/Down/Bottom ordering. Progress/speed/ETA are displayed only from actual transfer state.

Windows public output is one universal Setup + one universal Portable, each containing **x64, x86 and ARM64** payloads. The release metadata preserves the evidence boundary explicitly:

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 marker means the maintained CI proves cross-build/package/resource integrity without falsely claiming native Windows ARM64 runtime execution.

Linux public output is exactly six bundles: Installer + Portable for Debian/Ubuntu/Fedora, each carrying amd64/arm64/i386 payloads.

## Android parity boundary

Android is a public 0.0.6 application with Android-native Files, Sites, Bookmarks, Transfers, Settings and About surfaces. It uses Storage Access Framework local authority, FTP + strict explicit FTPS, bounded file/search/comparison/Remote Edit behavior and Android lifecycle ownership.

Android SFTP remains hidden until strict native host-key identity verification exists. Production release packaging remains fail-closed on signing and does not weaken that protocol boundary.

## macOS parity boundary

macOS is an active native AppKit development frontend over the shared engine. Source/build parity does not imply public distribution parity. It remains outside the public release until real Developer ID Application signing and Apple notarization succeed.

## Browser helper boundary

Chrome, Edge, Firefox and Opera helpers are public companion ZIPs built from one shared local runtime. They retain zero broad browser/host permissions and do not implement FTP/FTPS/SFTP transport. On supported installed Windows builds, an explicit **Open in Ghost FTP** action performs a sanitized browser-to-desktop handoff through `ghostftp://connect`. Only protocol, host, optional port, optional username and remote path are transferred; credentials, private-key material, source query data and fragments are excluded, and the desktop does not auto-connect.

## Retired application surfaces

The repository website, Web FTP runtime and PWA application surface are retired and intentionally absent from the maintained product tree. They are not part of the Ghost FTP 0.0.6 runtime, release or platform-parity contract.

## Theme and visual direction

Windows remains the canonical desktop visual reference. Dark uses the maintained deep navy palette; Classic Light uses the dirty/off-white gray palette rather than pure white. Native platform frontends follow the same Ghost FTP visual language while preserving platform-appropriate controls and interaction patterns.

## Release parity

Every public release stage is independently required. A successful Windows build cannot substitute for failed Linux, Android or browser publication.

The Ghost FTP 0.0.6 release contract is **13 platform artifacts / 16 public files**:

- Windows: 2;
- Linux: 6;
- Android: 1;
- Browser helpers: 4;
- release metadata: 3.

macOS remains active source/build validation only and is not a public release artifact for 0.0.6. Retired web surfaces do not contribute release artifacts.

See [Architecture](ARCHITECTURE.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
