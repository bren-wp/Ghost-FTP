# Ghost FTP Android parity

Ghost FTP Android follows the same product identity and file-action model as the Windows/Linux app while using a mobile-first native Kotlin layout.

## Current identity contract

- Product name: **Ghost FTP**
- Brand owner label: **Brendigo**
- Active source version: **0.30.6**
- Previous canonical release: **0.30.5**
- Version source of truth: root `version.json`
- Android source: `android/`
- Canonical unsigned production asset: `GhostFTP-Android-v<version>.apk.unsigned`
- Installability evidence asset: `GhostFTP-Android-v<version>-Installable-Preview.apk`

Android UI must not display legacy release-candidate badges or maintain an independent product version scheme.

## Android 15 system UI contract

- API 35+ uses enforced edge-to-edge layout with explicit system-bar inset padding on the root view.
- API 35 theme resources do not depend on deprecated status/navigation bar color attributes.
- System-bar icon appearance remains dark-theme appropriate.
- SFTP password handoff uses JSch's byte-array API; the temporary UTF-8 buffer is zeroed immediately after JSch copies it.

## Interface contract

Android keeps the same primary product model as desktop where it makes sense on mobile:

- one application surface;
- Ghost mark and Ghost FTP wordmark;
- Files workspace;
- Sites workspace/navigation;
- Transfers workspace/navigation;
- Settings security/privacy surface;
- Help & About product/release surface;
- connection surface for FTP, explicit FTPS and SFTP;
- SFTP host-key fingerprint input/verification;
- remote listing with folder navigation and file selection;
- Refresh, Upload, Download, New Folder, Rename and Delete actions;
- guarded confirmation for remote writes/removals;
- bounded activity log;
- clear disconnect behavior;
- no required account;
- no required telemetry.

## Mobile layout rules

- Use a persistent left navigation rail for Files, Sites, Transfers, Settings and Help & About.
- Render only the active workspace in the main content area instead of stacking every workspace in one long screen.
- Keep the selected workspace across Activity recreation while never persisting the password or authenticated session.
- Return to Files after a successful connection and move to Transfers after the document picker returns an upload selection.
- Prioritize one-handed use and readable touch targets.
- Keep connection/session state visible.
- Avoid desktop-only window controls/copy.
- Keep transfer/file actions consistent with desktop terminology.
- Do not persist passwords in ordinary application storage.
- Do not show sample hosts, sample accounts, demo copy or placeholder production text.

## Protocol contract

- FTP opens a real session and supports listing, download, upload, delete and remote-folder creation.
- FTP/explicit FTPS enforce timeouts, passive mode and binary transfers.
- Explicit FTPS keeps `PBSZ 0` and protected data channel `PROT P`.
- FTP/FTPS attempt logout/disconnect cleanup.
- SFTP verifies the supplied SHA-256 host-key fingerprint and keeps strict host-key checking enabled.
- SFTP supports listing, download, upload, delete and remote-folder creation.
- SFTP channels/sessions use timeouts and cleanup.
- Passwords remain in memory only for the active session/action and are cleared on disconnect.
- Android downloads use Android download storage.
- Android uploads use the document picker.
- Unsafe remote path segments such as `.` and `..` are rejected before transfer execution.
- Mutating operations refresh the listing after success.
- Rename uses the real FTP/FTPS/SFTP rename operation; Delete supports files and empty folders without recursive deletion.
- Long-press folder selection exposes folder rename/delete without replacing normal tap-to-open navigation.
- Settings exposes working Clear Activity, Reset Transfers, Reset Connection and Disconnect controls.

## 0.30.1 staged-upload safety

- FTP/FTPS only treats an upload target as absent after a successful parent-directory listing proves the target is missing.
- SFTP only treats an upload target as absent after an explicit no-such-file result from `lstat`.
- Promotion is refused if a target appears while an upload is in progress.
- Failed automatic restoration reports the preserved remote backup path for manual recovery instead of hiding the recovery failure.

## 0.20.9 validation hardening

- `testDebugUnitTest` runs in the canonical Android workflow before lint/build and emulator instrumentation.
- `ConnectionModelTest` covers cooperative cancellation, protocol-index fallback, host normalization and embedded-credential/port rejection, invalid ports, remote dot-segment rejection, root-delete protection and remote path joining.
- Activity recreation remains covered by instrumentation and restores only non-secret UI state while clearing the password/authenticated session.

## 0.20.0 reliability hardening

- Uploads are staged to a temporary remote object and promoted only after the transfer completes; an existing target is preserved for rollback while promotion is in progress.
- Downloads are staged locally and promoted only after completion, so a failed transfer does not destroy the existing destination.
- Activity destruction/disconnect cancellation propagates through upload, download, delete and folder-creation operations.
- Upload input streams are owned and closed at the controller boundary even when cancellation wins before protocol setup.
- Persisted SAF access keeps only valid READ grant modes in `takePersistableUriPermission` while the picker intent still requests persistable access.
- Production Android package metadata is verified even though the release-check APK is intentionally unsigned.

## CI/release contract

The **Ghost FTP Android** workflow runs for every pull request to `main` and every `main` push, so both PR acceptance and release orchestration have an exact-SHA Android gate.

It must:

- run the Android production contract;
- run an Android emulator click-through smoke for core workspace/action validation;
- lint/build debug, release and preview variants;
- verify the preview APK with `apksigner`;
- confirm the production release-check APK is unsigned on every cycle;
- confirm the preview package id;
- verify the unsigned production APK is `com.ghostftp.android`, non-debuggable and has the expected version/SDK/launcher metadata;
- clean-install, reinstall and launch-smoke the non-debuggable installable preview on the emulator;
- upload the verified Android artifact bundle for exact-SHA release consumption.

For publication, the canonical Ghost FTP release workflow requires both Android artifacts from the exact source SHA: the intentionally unsigned production `com.ghostftp.android` build and the non-debuggable installable preview used for emulator install/reinstall/launch proof. No private Android signing secrets are part of the release pipeline.

Stable/FINAL mobile status additionally requires device-level install/upgrade/storage/protocol acceptance. Android itself still requires a signature for any APK that is installed on a device, so the unsigned production artifact is distributed as build evidence rather than falsely labeled installable.
