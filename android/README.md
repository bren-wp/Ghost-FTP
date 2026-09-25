# Ghost FTP Android

Native Android application for Ghost FTP.

The Android app is aligned with the Windows and Linux Ghost FTP product identity:

- Product: Ghost FTP
- Brand: Brendigo
- Version: 2.1.1-rc.23
- Display: 2.1.1 RC23
- Build: 2026.09.25.23

## Android surface

The Android surface uses a mobile version of the desktop Ghost FTP shell:

- Ghost mark and Ghost FTP wordmark.
- RC23 badge.
- `Files` workspace label.
- Desktop-aligned toolbar actions: Refresh, Upload, Download, New Folder and Delete.
- `Sites` card for FTP, explicit FTPS and SFTP endpoint control.
- SFTP host key fingerprint field for explicit server identity verification.
- `Files` card for the current remote listing.
- Tap-to-open remote folders and tap-to-select remote files.
- `Transfers` card for the selected remote file, upload target and folder target.
- Android document picker upload flow.
- Android download storage for received files.
- Transfer-state text for the last selected, running, completed or failed operation.
- Bounded activity log so long sessions keep a stable mobile layout.
- Brendigo footer.
- Passwords kept in memory only for the active session and cleared on disconnect.
- No analytics, telemetry or required account sign-in.

## Protocol support

The Android app uses native protocol clients:

- FTP remote login, folder listing, download, upload, file delete and folder creation.
- Explicit FTPS remote login, protected data-channel listing, download, upload, file delete and folder creation.
- SFTP remote login, folder listing, download, upload, file delete and folder creation with SHA-256 host key fingerprint verification.

Credentials are passed only into the active connection or transfer action. The app does not add telemetry, accounts or password persistence.

## Production safeguards

RC23 locks the production protocol behavior in the Android contract:

- FTP and explicit FTPS use connect/default/data timeouts.
- FTP and explicit FTPS require login success before file actions.
- FTP and explicit FTPS use passive mode and binary file transfers.
- Explicit FTPS uses `PBSZ 0` and protected data channel mode `PROT P`.
- FTP/FTPS sessions always attempt logout and disconnect cleanup.
- SFTP requires a SHA-256 host key fingerprint, strict host-key checking and session/channel timeouts.
- SFTP channels and sessions are cleaned up after each action.
- Host input is normalized through IDN handling before connecting.

## Safety UX

The Android transfer surface includes guarded actions for higher-risk operations:

- Delete requires an explicit confirmation dialog before the server action runs.
- Upload shows the selected local file and asks for confirmation before writing to the remote target.
- Upload, Delete and New Folder refresh the current Files listing after success.
- Relative remote targets are resolved from the current remote folder.
- Remote targets containing `.` or `..` path segments are rejected before a transfer action starts.
- The activity log is capped by `MAX_ACTIVITY_ROWS` so repeated actions do not expand the mobile layout without limit.

## Build

From the repository root:

```bash
gradle -p android lintDebug lintRelease assembleDebug assembleRelease
```

The pull-request Android workflow runs the Android contract, both lint variants and both APK builds. It uploads installable RC23 APK artifacts and checksums for review.

RC23 Android lint validates minSdk 26 compatibility. API 27+ navigation-bar light/dark behavior is kept in the `values-v27` resource override so the base theme remains valid for Android 8.0 devices.

## Release rule without external keys

Every GitHub release must include an Android APK asset. The RC23 release workflow must download the verified Android workflow artifact for the same release source commit and publish it as `GhostFTP-Android-v2.1.1-RC23.apk` together with the Windows, Linux and checksum assets.

No repository keystore, GitHub secret or manual signing key is required for this release path.

For long-term Android upgrade continuity, a stable signing key can be introduced later. Without a stable saved signing key, Android may treat APKs from different release runs as separately signed builds.

## Completion checklist

Before treating the Android app as release-ready, confirm:

- FTP, explicit FTPS and SFTP can list a remote path.
- The top shell shows Ghost mark, Ghost FTP wordmark, RC23 badge and the `Files` workspace.
- Toolbar action names match desktop: Refresh, Upload, Download, New Folder and Delete.
- Tapping a folder opens that remote path.
- Tapping a file selects it for transfer actions.
- Download saves into Android downloads.
- Upload uses the Android document picker and writes to the selected remote target after confirmation.
- Delete requires confirmation and returns a clear success or failure state.
- New Folder returns a clear success or failure state and refreshes the remote listing.
- Unsafe remote target paths with `.` or `..` segments are rejected.
- FTP/FTPS binary passive transfers remain enforced.
- Explicit FTPS protected data channel remains enforced.
- SFTP strict host-key checking remains active.
- Password is cleared on disconnect.
