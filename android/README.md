# Ghost FTP Android

Native Android application for Ghost FTP.

## Product identity

- Product: Ghost FTP
- Brand: Brendigo
- Active source/release cycle: 0.20.8
- Previous canonical release: 0.20.7
- Version source of truth: ../version.json
- Release identity is rendered from `ReleaseInfo.kt`; this document does not carry an independent version badge.

## Android workspaces

The Android application uses the same primary product vocabulary as the Windows/Linux application:

- **Files** — remote listing, folder navigation and remote-entry selection.
- **Sites** — FTP, explicit FTPS and SFTP connection form.
- **Transfers** — selected remote entry, upload target, folder creation and rename target.
- **Settings** — working session/privacy controls.
- **Help & About** — product, protocol and verified-release information.

Only the active workspace is visible. Workspace state can survive Activity recreation, while passwords and authenticated sessions never do.

## Shared working actions

The Android action surface is backed by real protocol operations:

- Refresh the active remote listing.
- Upload a selected Android document.
- Download a selected remote file.
- Create a remote folder.
- Rename a remote file or folder.
- Delete a remote file or an empty remote folder.
- Connect and disconnect FTP, explicit FTPS and SFTP sessions.

Normal tap opens a remote folder. Long-press selects a folder for rename/delete without changing the current folder first.

Non-empty folders are not deleted recursively. Root-path delete and rename are blocked.

## Settings actions

Android Settings is an actionable workspace, not an informational placeholder:

- **Clear Activity** removes the bounded session activity log.
- **Reset Transfers** clears transfer/rename targets and the selected local upload document.
- **Reset Connection** disconnects, clears the connection form and restores FTP/port 21 plus remote path `/`.
- **Disconnect** cancels the active operation and clears the authenticated in-memory session.

## Protocol support

### FTP

- Login and remote folder listing.
- Passive binary transfer mode.
- Upload/download.
- File delete and empty-folder delete.
- Folder creation.
- Rename.
- Connect/default/data timeouts.
- Logout/disconnect cleanup.

### Explicit FTPS

- FTP behavior above over explicit TLS.
- `PBSZ 0` and `PROT P` protected data channel.
- Upload/download, create, rename, file delete and empty-folder delete.

### SFTP

- Strict host-key checking with required SHA-256 fingerprint.
- Folder listing.
- Upload/download.
- File delete and empty-folder delete.
- Folder creation.
- Rename.
- Session/channel timeout and cleanup.

## Transfer safety

- Uploads are staged to a temporary remote path and promoted after completion.
- Downloads are staged locally and promoted only after completion.
- Lifecycle/disconnect cancellation propagates through active operations.
- Upload streams are closed at the controller ownership boundary.
- Relative targets are resolved from the current remote folder.
- `.` and `..` target segments are rejected.
- Delete and rename refuse the remote root path.
- Upload and destructive actions require explicit confirmation where applicable.
- Mutating operations refresh the remote listing after success.
- The activity log is capped by `MAX_ACTIVITY_ROWS`.

## Privacy and credentials

- No required analytics or telemetry.
- No required Ghost FTP account.
- Passwords stay in memory only for the active session.
- Passwords are cleared on disconnect and Activity destruction.
- Android view-state persistence is disabled for the password field.
- Non-secret form/transfer state can survive Activity recreation.

## Build and QA

From the repository root:

```bash
gradle -p android lintDebug lintRelease assembleDebug assembleRelease
```

Release-relevant CI additionally runs:

- `android/scripts/check-android-contract.sh`
- Android lint for debug/release
- debug/release APK builds
- emulator click-through instrumentation smoke
- exact-SHA artifact packaging for the canonical release workflow

The contract requires the working shared action surface, rename backends for FTP/FTPS/SFTP, empty-folder delete support, Settings controls, lifecycle safety rules and smoke coverage.

## Release artifacts

Canonical release naming:

- `GhostFTP-Android-v<version>.apk.unsigned` — unsigned production output used for release verification.
- `GhostFTP-Android-v<version>-Installable-Preview.apk` — release-optimized, non-debuggable installable CI preview using the isolated `com.ghostftp.android.preview` application id.

The preview signing identity is intentionally ephemeral. A persistent production upgrade identity requires a persistent signing key and is never silently simulated by the keyless CI workflow.

## Completion gate

An Android release candidate is not accepted merely because it compiles. The exact release SHA must pass the Android production contract, lint/build, emulator click-through smoke and the repository-wide release gates before the canonical GitHub Release can be published.
