> Android 16-screen reference candidate v0.95.23: **preview only**. Real Bookmarks, contextual Settings/Advanced, Add/Edit tabs, native About and stored-profile loading are implemented in source. Pixel-perfect 16/16 acceptance and real-device screenshot comparison are not yet established. Latest published baseline: v0.95.22.

> Ghost FTP **0.95.23** (release cycle), following published **v0.95.22**. Distribution status is verified from GitHub Releases; unsigned/mobile and notarization limitations remain explicit.


### 0.95.23 Android layout parity increment — source only, exact-device acceptance pending

The Android phone UI now uses a full-width page with a 260/280dp overlay drawer under 600dp and a 52dp contextual top bar. Only tablets keep a docked rail. Files-only FTP actions now hide when switching to Sites, Transfers, Sync, Cloud, Settings and About. The live FTP session status is visible only on Files and Sites; instrumented navigation asserts the proper visibility. This removes a concrete source-level source of extra/irrelevant rows. The original 16 design images are not currently mounted for direct comparison; no screen is certified 1:1 yet. The app still needs actual device capture, pixel diff and interaction QA against the user-provided designs.

# Ghost FTP Android parity

## Next Android reference increment — Bookmarks screen #10 (development)

- A real **Bookmarks** workspace now uses the approved dark/violet card system, folder icon, Add action, title and path rows, contextual delete action, and consistent slide-out phone navigation.
- `SavedRemoteBookmarks` stores only a bookmark title, an existing saved-site name, and an absolute validated remote directory; no passwords, private keys, access tokens, sample servers, fake folders or fabricated history.
- A bookmark requires a currently authenticated matching profile to open directly. Otherwise the app routes to the saved-site editor and requires explicit authentication again. It cannot bypass SSH host-key checking.
- Persistent data decoding is size-bounded and mutation fails closed on corruption, traversal, duplicates or malformed names. Removal requires confirmation.
- Source unit regression plus Android 16 emulator click-through test checks empty/no-demo state and Add refusal without a live session.
- The eighth **real** Android QA render is `bookmarks.png` at 480px width. The image is an installed debug Activity tree capture under protected test conditions, not an invented design render. It still needs actual reference pixel-diff and physical device acceptance.

**Remaining:** Connect, Transfer Details, real Queue, remote search, comparison/sync, remote editing, batch actions, and full 16/16 parity. Build/test green alone does not establish 1:1 acceptance.

## 2026-10-16 user reference: sixteen distinct Android screens

Two exact uploaded source images and SHA256 provenance are tracked in `docs/qa/android-16-screen-reference-2026-10-16.json`. Their visual concept is the current mobile-specific reference: violet action buttons, full-width mobile pages and slide-over sidebar. PR #158 begins real native rendering and removes the outdated docked 64dp rail. This is NOT proof that all sixteen screens already match or that functional transfer queues, remote editing and multi-provider cloud sign-in exist. Real installed-app 16/16 pixel comparison is pending.



Current development source: **0.95.22** (security and premium visual QA); previous published release: **0.95.21**. Historical versioned notes below remain unchanged.

> Older 0.94.0/0.93.0 notes below remain historical; current development is **0.95.22**, and the latest public release is **0.95.21**.

Ghost FTP Android follows the same product identity and file-action model as the Windows/Linux app while using a mobile-first native Kotlin layout.

0.92.0 macOS adds an advisory saved-site duplicate filter to join the Windows/Linux 0.91.0 capability. The Android app does not maintain the same persistent saved-profile inventory, so this feature does not imply Android saved-site parity; FTP/FTPS/SFTP connection and transfer capabilities remain as documented.

0.91.1 adds a verified diagnostic privacy contract that hides URL user info, even for username-only FTP/SFTP URLs. The 0.91.0 desktop duplicate-profile filter is still a Windows/Linux-only feature.

## Cloud storage — 0.93.0 candidate

The Android Cloud storage workspace transfers one object at a time using short-lived provider-signed S3-compatible/GCS HTTPS PUT/GET and Azure SAS links. Signed URL credentials are not saved in state/backups; document picker files and staged download safety are used. Transfers are bounded to 1 GiB. Google Drive OAuth browsing and S3 bucket listing are not available on Android.

## Current identity contract

- Product name: **Ghost FTP**
- Brand owner label: **Brendigo**
- Active source version: **0.95.18**
- Previous canonical release: **0.95.22**
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

- Use a persistent left navigation rail for Files, Sites, Transfers, Sync & Backup, Settings and Help & About. On standard phones (360–599 dp), keep the rail docked at 88 dp to match the 480 × 960 premium design reference; on devices narrower than 360 dp, use the overlay to preserve workspace width. At 600 dp and above, use the expanded 184 dp rail.
- Render only the active workspace in the main content area instead of stacking every workspace in one long screen.
- Keep the selected workspace across Activity recreation while never persisting the password or authenticated session; restore never hides a permanently docked rail.
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
