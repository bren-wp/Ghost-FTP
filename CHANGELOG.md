# Changelog

## 2.1.1 RC23 — 2026-09-25

- Added RC23 release line for Windows, Linux and Android production hardening.
- Bumped desktop, Tauri, Rust crate and Android metadata to `2.1.1-rc.23` / `2.1.1 RC23` / `2026.09.25.23`.
- Merged Android transfer completion, guarded transfer UX and lifecycle stability improvements into `main`.
- Added Android production protocol safeguards for FTP, explicit FTPS and SFTP.
- Locked FTP/FTPS timeouts, passive mode, binary transfers, login validation and cleanup in the Android production contract.
- Locked explicit FTPS `PBSZ 0` and protected data channel `PROT P` in the Android production contract.
- Locked SFTP fingerprint verification, strict host-key checking, connection timeouts and cleanup in the Android production contract.
- Updated Android README and Android parity documentation for RC23 release readiness.
- Requires Android, quality and real FTP/FTPS/SFTP protocol E2E gates before release publication.

## 2.1.1 RC22 — 2026-09-24

- Added RC22 release line for Windows, Linux and Android APK artifacts.
- Bumped desktop, Tauri, Rust crate, update-template and Android metadata to `2.1.1-rc.22` / `2.1.1 RC22` / `2026.09.24.22`.
- Fixed Android Kotlin build errors in FTP timeout handling and EditText single-line setup.
- Kept native Android FTP, explicit FTPS and SFTP remote listing support.
- Kept SFTP SHA-256 host key fingerprint verification and strict host-key checking on Android.
- Added Android APK release workflow without GitHub signing secrets.
- Added RC22 release workflow for Windows/Linux assets, source bundles, documentation bundles and SHA-256 checksums.
- Updated root README, Android README, Android parity docs and RC22 release notes.

## Older releases

Historical release notes are retained under `docs/releases/`.
