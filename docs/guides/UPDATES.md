> **Windows/Linux unsigned desktop update policy (0.99.33 Preview):** Automatic in-app installation accepts **only signed** Tauri updater packages. A missing signed updater offer does not establish that no newer manually installable desktop release exists. Use Help & About → Updates → **Open official Windows / Linux downloads** to open the exact GitHub [Releases index](https://github.com/bren-wp/Ghost-FTP/releases), verify the matching `SHA256SUMS.txt`, and install manually after a backup. No unverified or unsigned package is installed silently in-app. A paid certificate is not necessary for user-initiated desktop installation, but publisher identity is not cryptographically authenticated without one.

# Ghost FTP Updates

Ghost FTP desktop uses the native Tauri updater path. User-facing application copy intentionally describes this only as the official update service and verified update packages; transport/file-format details remain in operator documentation under `updates/`.

## Desktop contract

- Windows/Linux packages are built from the exact release source SHA.
- In-app updater publication requires real Tauri signatures for both the Windows Setup and Linux AppImage.
- If signing secrets are not configured, verified ordinary packages may still be published as the stable GitHub release. The in-app updater bundle is omitted. If any updater proof is present, both signatures, the exact-version manifest and matching Update-Service package must all be present and consistent.
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
