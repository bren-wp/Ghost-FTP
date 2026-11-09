> **Current development (9 November 2026): Ghost FTP 0.99.37 Preview.** Windows/Linux DST-safe scheduled transfers, resumable SHA-256-verified release uploads and stricter release gates. Previous published release: v0.99.36. Signed-production status and 1:1 visual parity are not claimed.

# Ghost FTP build and QA status

- **Active source/release cycle:** Ghost FTP **0.99.37** Preview.
- **Previous canonical release:** Ghost FTP **0.99.36**.
- **Active platforms:** Windows, Linux and macOS desktop.
- **Primary sources:** `ghostftp-desktop/`, `macos/`; Go compatibility helpers in `tools/`.
- **Windows/Linux:** React, TypeScript, Rust/Tauri; portable EXE, NSIS, MSI, standalone Linux binary, AppImage, DEB and RPM
- **macOS:** SwiftUI preview and shared-desktop package builds; Developer ID notarization and real-device acceptance not certified
- **Release integrity:** exact source SHA, immutable version history, SHA-256 artifact manifest, signed automatic updater policy

## Required CI verification

1. `Ghost FTP quality`: frontend checks, code audit, dead-code audit, desktop UX, security and version governance
2. `Ghost FTP protocol E2E`: real localhost FTP, explicit FTPS, SFTP roundtrips, trust policy and disconnect/recovery tests
3. `Ghost FTP native build`: Windows/Linux native builds, installer/packaging and capture evidence
4. `Validate Windows hardening`: native filesystem, credentials and Windows security regression tests
5. `Ghost FTP macOS` and `Ghost FTP shared macOS desktop`: Swift runtime, package, platform and security checks

Tests must pass against the **exact last commit** and final artifacts, not a prior successful build. Stale screenshots are not substitute evidence. The active 30 Windows + 29 Linux references remain unverified until native capture and interaction assessment.

## Known acceptance gaps

The static audit does not validate every public server, remote hash, very large transfer, credential vault on every OS, cloud OAuth refresh, TLS certificate revocation, or macOS FTPS/SFTP availability. Signed release requirements and user data migration must be independently verified. Refer to [0.99.36 release notes](../releases/0.99.36.md) and [desktop audit](../audits/FULL_DESKTOP_CODE_SECURITY_DEAD_CODE_0.99.36.md) when the audit report is available.
