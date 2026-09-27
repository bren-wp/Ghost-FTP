# Ghost FTP Android parity

Ghost FTP Android follows the same product identity and file-action model as the Windows/Linux app while using a mobile-first native Kotlin layout.

## Current identity contract

- Product name: **Ghost FTP**
- Brand owner label: **Brendigo**
- Active source version: **0.19.0**
- Previous canonical release: **0.18.0**
- Version source of truth: root `version.json`
- Android source: `android/`
- Canonical release asset: `GhostFTP-Android-v<version>.apk`

Android UI must not display legacy release-candidate badges or maintain an independent product version scheme.

## Interface contract

Android keeps the same primary product model as desktop where it makes sense on mobile:

- one application surface;
- Ghost mark and Ghost FTP wordmark;
- Files workspace;
- connection surface for FTP, explicit FTPS and SFTP;
- SFTP host-key fingerprint input/verification;
- remote listing with folder navigation and file selection;
- Refresh, Upload, Download, New Folder and Delete actions;
- guarded confirmation for remote writes/removals;
- bounded activity log;
- clear disconnect behavior;
- no required account;
- no required telemetry.

## Mobile layout rules

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

## CI/release contract

The **Ghost FTP Android** workflow runs for every pull request to `main` and every `main` push, so both PR acceptance and release orchestration have an exact-SHA Android gate.

It must:

- run the Android production contract;
- run an Android emulator click-through smoke for core workspace/action validation;
- lint/build debug, release and preview variants;
- verify the preview APK with `apksigner`;
- confirm the release-check APK is unsigned;
- confirm the preview package id;
- upload the installable preview APK artifact.

The canonical Ghost FTP release workflow consumes the verified installable APK for the exact source SHA and publishes it with Windows/Linux/source/checksum assets.

Stable/FINAL mobile status additionally requires device-level install/upgrade/storage/protocol acceptance and a production signing-key continuity decision.
