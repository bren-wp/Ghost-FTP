# Ghost FTP Android

Native Android client source lives entirely under this `android/` directory.

## Ghost FTP 0.0.6 release status

Ghost FTP **0.0.6** adds a **production-signed public Android release** to the canonical GitHub Release:

```text
Ghost-FTP-0.0.6-Android.apk
```

The repository root `VERSION` remains the canonical release identity. Android release `versionName` equals root `VERSION`; development builds add only the `-dev` suffix and keep the separate package/application-development identity.

Ordinary exact-head CI continues to produce:

```text
Ghost-FTP-Android-dev.apk
```

That development APK may use an ephemeral CI-only signing identity for pipeline validation. It is not the public production APK and is not accepted as publisher evidence.

The canonical release workflow requires protected Android signing credentials and verifies the signing certificate SHA-256 fingerprint before publication. Production signing material is never committed to the repository.

## Current capability

- Native Android Java UI.
- Phone navigation uses a real left navigation drawer; wide/tablet layouts use the same destinations as a persistent sidebar.
- Active destinations: **Files**, **Sites**, **Bookmarks**, **Transfers**, **Settings** and **About**.
- Local vector assets; no remote fonts, tracking assets or emoji-as-navigation icons.
- **FTP and explicit FTPS Quick Connect**.
- FTPS uses the platform trust store and strict hostname verification on control and protected passive data channels; there is no trust-all fallback.
- FTP remains available only as an explicitly unencrypted compatibility choice.
- Local navigation uses Android **Storage Access Framework** (`ACTION_OPEN_DOCUMENT_TREE`); the app does not request broad all-files storage access.
- MLSD directory listing over EPSV/PASV with fail-closed passive-data validation.
- Binary upload/download with staged same-directory/same-provider commit behavior and explicit cancellation ownership.
- Local SAF create-directory, rename and delete operations.
- Remote FTP/FTPS create-directory, rename, delete and `SITE CHMOD` operations with path/name validation, confirmation and fresh-list readback.
- Saved sites/bookmarks persist only non-secret identity/navigation metadata; passwords remain memory-only.
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

## Saved-site and bookmark boundary

Saved sites remain non-secret. Persisted identity/navigation state is bounded and tied to `(protocol, host, port, username)`. Changing that remote identity clears server-specific starts/bookmarks rather than carrying them to a different endpoint. Local starts/bookmarks require persisted SAF permission and a fresh provider query before becoming authoritative.

## Authentic Android screenshots

Authentic UI evidence is captured from the exact-source built APK in an Android emulator. Mockups, generated images and manually composed approximations are not production evidence. The maintained cross-platform evidence bundle records exact source SHA and SHA-256 hashes.

## Development build

The canonical development workflow uses Gradle 8.9 and Android SDK 35 and exercises unit tests, debug/release lint, debug packaging, unsigned release construction and an ephemeral `apksigner` smoke test.

Canonical development artifact:

```text
android/dist/Ghost-FTP-Android-dev.apk
```

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

The production workflow must fail closed if credentials are absent/invalid and must never generate a replacement publisher identity.
