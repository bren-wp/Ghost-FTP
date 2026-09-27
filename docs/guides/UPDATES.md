# Ghost FTP Updates

Ghost FTP desktop uses the native Tauri updater path. User-facing application copy intentionally describes this only as the official update service and verified update packages; transport/file-format details remain in operator documentation under `updates/`.

## Desktop contract

- Windows/Linux packages are built from the exact release source SHA.
- In-app updater publication requires real Tauri signatures for both the Windows Setup and Linux AppImage.
- If signing secrets are not configured, ordinary GitHub Release packages may still be published, but the in-app update deployment bundle is omitted.
- Existing version tags are immutable.
- Failed update checks or verification must leave the installed application usable.

Operator details:

- `updates/README.md`
- `updates/RELEASE_RUNBOOK.md`
- `updates/SECURITY.md`
- `updates/DEPLOYMENT.md`

## Android

Android is distributed as the verified APK release asset produced by the Android gate. Android does not consume the desktop Tauri updater contract.

## Verification

Always verify the release checksum file before manual installation or redistribution. A release must not be advertised as stable/FINAL merely because an update package exists.
