# Ghost FTP Android

Native Android client source lives entirely under this `android/` directory.

## Ghost FTP 0.0.7 release status

Ghost FTP **0.0.7** publishes one production-signed Android artifact in the canonical GitHub Release:

```text
Ghost-FTP-0.0.7-Android.apk
```

The repository root `VERSION` is the canonical release identity. Android `versionName` equals that root version exactly; validation builds keep the same visible product version while using an isolated debug application ID where required for CI testing.

Pull-request and branch CI build the standard Android test variant and an unsigned release APK for verification. These CI outputs are validation inputs only and are not public release artifacts. The public 0.0.6 APK is produced exclusively by the protected release workflow after publisher signing and certificate-fingerprint verification.

The canonical release workflow requires protected Android signing credentials and verifies the signing certificate SHA-256 fingerprint before publication. Production signing material is never committed to the repository.

## Current capability

- Native Android Java UI.
- Canonical **Dark** appearance is the fresh-install default with charcoal/blue-black surfaces and gold/amber actions; a neutral gray **Light** appearance remains available in Settings.
- Phone navigation uses a real left navigation drawer; wide/tablet layouts use the same destinations as a persistent sidebar.
- Active destinations: **Files**, **Connections**, **Transfer Queue**, **Settings**, **Bookmarks**, **Connection info** and **About**.
- Local vector assets; no remote fonts, tracking assets or emoji-as-navigation icons.
- **FTP and explicit FTPS Quick Connect**.
- FTPS uses the platform trust store and strict hostname verification on control and protected passive data channels; there is no trust-all fallback.
- FTP remains available only as an explicitly unencrypted compatibility choice.
- Local navigation uses Android **Storage Access Framework** (`ACTION_OPEN_DOCUMENT_TREE`); the app does not request broad all-files storage access.
- MLSD directory listing over EPSV/PASV with fail-closed passive-data validation.
- Binary upload/download with staged same-directory/same-provider commit behavior and explicit cancellation ownership.
- Local SAF create-directory, rename and delete operations.
- Remote FTP/FTPS create-directory, rename, delete and `SITE CHMOD` operations with path/name validation, confirmation and fresh-list readback.
- Saved connections/bookmarks persist only non-secret identity/navigation metadata; passwords remain memory-only.
- Quick Connect endpoint metadata is opt-in on fresh installs; Settings can also toggle file-size display, delete confirmation and restore app defaults without removing saved connections or SAF folder authority.
- **Connection info** is privacy-safe runtime diagnostics: connection state, protocol/security mode and transfer state only; host, username, passwords, keys and saved paths are excluded.
- No telemetry, analytics, ads, automatic crash-report upload or Ghost FTP relay/backend.

See [`UI-UX.md`](UI-UX.md) for navigation and per-surface ownership.

## SFTP security boundary

**SFTP is intentionally not exposed** on Android. Ghost FTP desktop requires strict host-key verification/pinning; Android will not present SFTP until equivalent strict, maintained host-key identity verification exists and is tested. There is no silent SFTP-to-FTP/FTPS fallback.

The public 0.0.6 APK does not change this boundary. Production signing proves publisher/package identity, not protocol safety.

## Remote Desktop boundary

Remote Desktop is intentionally absent because there is no reviewed Android RDP runtime owner. Ghost FTP does not expose a decorative or non-functional RDP destination.

## Upload commit safety

Uploads do not stream directly into the requested final remote name. A random `.ghostftp-upload-<uuid>.part` staging object is used first. The final `RNFR`/`RNTO` sequence is attempted only after successful transfer completion and the synchronized transfer commit gate grants finalization.

A cancellation that wins before commit prevents final-name publication. If finalization already owns the irreversible commit phase, the UI stops claiming cancellation is possible. Ambiguous data/control failures hard-close the FTP session instead of reusing unknown protocol state.

## Download commit safety

Downloads use a temporary SAF `.ghostftp-download-<uuid>.part` document and only receive the requested final display name after protocol completion, a second fresh conflict check, commit-gate ownership and exact display-name readback. Failed/cancelled transfers best-effort remove staging documents and do not report success before exact-name verification.

All local work remains inside the user-granted SAF tree.

## Transfer lifecycle

Transfer state is bound to the Activity/session generation and the exact transfer commit gate. Stale worker callbacks from a cancelled/destroyed/replaced lifecycle cannot publish completion into newer UI/session state.

During active data I/O, cancellation closes the active data/control sockets without waiting on the synchronized protocol monitor. A cancelled session is not reused. `Finalizing…` is used when irreversible finalization has already begun.

## Passive data-channel boundary

Malformed EPSV/PASV replies, invalid passive ports, TCP data-connect failure or FTPS data-channel TLS failure close the FTP session and require reconnect. There is no fallback to an unprotected FTPS data channel and no disabled hostname verification.

## Saved-connection and bookmark boundary

Saved connections remain non-secret. Persisted identity/navigation state is bounded and tied to `(protocol, host, port, username)`. Changing that remote identity clears server-specific starts/bookmarks rather than carrying them to a different endpoint. Local starts/bookmarks require persisted SAF permission and a fresh provider query before becoming authoritative.

## Authentic Android screenshots

Authentic UI evidence is captured from the exact-source built APK in an Android emulator. Mockups, generated images and manually composed approximations are not production evidence. The maintained cross-platform evidence bundle records exact source SHA and SHA-256 hashes.

## CI validation build

The maintained Android CI uses Gradle 8.9 and Android SDK 35. It runs unit tests, debug/release lint, a standard Android test build, unsigned release construction and an isolated `apksigner` verification pass.

No CI validation APK is presented as a public Ghost FTP release artifact or publisher-signed package.

## Production release signing

The public 0.0.6 publication path requires:

```text
GHOSTFTP_ANDROID_KEYSTORE_BASE64
GHOSTFTP_ANDROID_KEYSTORE_PASSWORD
GHOSTFTP_ANDROID_KEY_ALIAS
GHOSTFTP_ANDROID_KEY_PASSWORD
GHOSTFTP_ANDROID_CERT_SHA256
```

The workflow builds the unsigned release APK, signs it using Android `apksigner`, verifies the APK and requires the signer certificate SHA-256 digest to match the protected expected fingerprint. The temporary keystore is removed after the job.

The production workflow fails closed if credentials are absent or invalid and never generates a replacement publisher identity.
