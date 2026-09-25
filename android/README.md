# Ghost FTP Android

Native Android application source for Ghost FTP.

This Android app lives in `/android` so the Windows and Linux desktop release line remains isolated. The product identity stays aligned with Ghost FTP and Brendigo:

- Product: Ghost FTP
- Brand: Brendigo
- Version: 2.1.1-rc.22
- Display: 2.1.1 RC22
- Build: 2026.09.24.22

## Android surface

The Android surface is a native Kotlin single-activity app with a mobile layout that follows the desktop product structure:

- Header with Ghost FTP RC22 release identity.
- Session status card.
- New connection card for FTP, explicit FTPS and SFTP endpoint control.
- SFTP host key fingerprint field for explicit server identity verification.
- Remote workspace card for current server context and remote folder listing.
- Tap-to-open remote folders and tap-to-select remote files.
- Transfer actions for download, upload, remote file delete and remote folder creation.
- Android document picker upload flow.
- App-private Android download storage for received files.
- Transfer-state text for the last selected, running or completed operation.
- Bounded transfer event log so the mobile surface remains stable during long sessions.
- Brendigo footer.
- Passwords kept in memory only for the current action and cleared on disconnect.
- No analytics, no telemetry and no required account sign-in.

## Protocol support

The Android app uses native protocol clients:

- FTP remote login, folder listing, download, upload, file delete and folder creation.
- Explicit FTPS remote login, protected data-channel listing, download, upload, file delete and folder creation.
- SFTP remote login, folder listing, download, upload, file delete and folder creation with SHA-256 host key fingerprint verification.

Credentials are passed only into the active connection or transfer action. The app does not add telemetry, accounts or ordinary password persistence.

## Safety UX

The Android transfer surface now includes guarded actions for higher-risk operations:

- Remote delete requires an explicit confirmation dialog before the server action runs.
- Upload shows the selected local file and asks for confirmation before writing to the remote target.
- Upload, delete and folder creation refresh the current remote workspace after success.
- Relative remote targets are resolved from the current remote folder.
- Remote targets containing `.` or `..` path segments are rejected before a transfer action starts.
- The transfer queue is capped by `MAX_QUEUE_ROWS` so repeated actions do not make the one-screen mobile UI grow without limit.

## Build

From the repository root:

```bash
gradle -p android lintDebug lintRelease assembleDebug assembleRelease
```

The pull-request Android workflow runs the Android contract, both lint variants and both APK builds. It uploads installable RC22 APK artifacts and checksums for review.

RC22 Android lint validates minSdk 26 compatibility. API 27+ navigation-bar light/dark behavior is kept in the `values-v27` resource override so the base theme remains valid for Android 8.0 devices.

## Release rule without external keys

Every next GitHub release must include an Android APK asset. The RC22 release workflow downloads the verified Android workflow artifact for the same release source commit and publishes it as `GhostFTP-Android-v2.1.1-RC22.apk` together with the Windows, Linux and checksum assets.

No repository keystore, GitHub secret or manual signing key is required for this release path.

For long-term Android upgrade continuity, a stable signing key can be introduced later. Without a stable saved signing key, Android may treat APKs from different release runs as separately signed builds.

## Completion checklist

Before treating the Android app as complete for RC22 follow-up development, confirm:

- FTP, explicit FTPS and SFTP can list a remote path.
- Tapping a folder opens that remote path.
- Tapping a file selects it for transfer actions.
- Download saves into app-private Android downloads.
- Upload uses the Android document picker and writes to the selected remote target after confirmation.
- Remote file delete requires confirmation and returns a clear success or failure state.
- Remote folder creation returns a clear success or failure state and refreshes the remote listing.
- Unsafe remote target paths with `.` or `..` segments are rejected.
- SFTP strict host-key checking remains active.
- Password is cleared on disconnect.
