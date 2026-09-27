# Ghost FTP Click / Interaction QA

## Automated evidence for 0.19.0

Ghost FTP uses real application components; reference screenshots are never runtime backgrounds or click maps.

### Windows desktop

The canonical Windows native build launches the real Tauri executable and captures fresh native-window evidence for seven critical surfaces:

- Files
- Sites
- New Connection
- Settings
- Transfers
- File Properties
- Help & About

The same build verifies the real NSIS Setup install/uninstall lifecycle and native window geometry.

Source interaction contracts additionally require the primary navigation/actions to remain wired to real stores/handlers rather than dead buttons.

### Linux desktop

Linux uses the same React/TypeScript desktop UI and Rust workspace as Windows. The exact source is typechecked, built and tested once for product behavior, then Linux-specific packaging is separately validated through AppImage metadata, DEB install/remove and RPM metadata checks.

### Android

The Android gate now runs an emulator instrumentation smoke test that opens `MainActivity` and performs real click actions for:

- connection validation;
- disconnect/idle recovery;
- refresh in idle state;
- guarded Download;
- guarded New Folder;
- guarded Delete.

The production contract also verifies Files/Sites/Transfers/Settings/Help & About mobile navigation, upload/document-picker wiring, confirmation guards, lifecycle cleanup and real FTP/FTPS/SFTP protocol calls.

## Manual acceptance still required for stable / FINAL

Automation cannot prove every OS compositor, accessibility technology, language expansion, file picker/provider, server implementation or destructive-operation confirmation on physical devices.

Before a stable/FINAL claim, perform a complete Windows/Linux pointer + keyboard sweep and Android physical-device sweep, including repeated open/close cycles, focus traversal, destructive confirmations, disabled states, file picker/storage behavior and supported languages.

For the pre-1.0 0.19.0 release, automated exact-SHA gates are mandatory and manual stable/FINAL acceptance remains a separate criterion.
