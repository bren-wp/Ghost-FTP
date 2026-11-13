> **Ghost FTP v0.99.59 Preview candidate. Previous public release: v0.99.58. Stable releases still require signed packages and independently verified Windows/Linux acceptance.**

# Ghost FTP build and QA status

- **Active source/release cycle:** Ghost FTP **0.99.59** Preview.
- **Previous canonical release:** Ghost FTP **0.99.58**.
- **Active platforms:** Windows and Linux desktop.
- **Windows/Linux:** React, TypeScript, Rust/Tauri; portable EXE, NSIS, MSI, standalone Linux binary, AppImage, DEB and RPM
- **Release integrity:** exact source SHA, immutable version history, SHA-256 artifact manifest, signed automatic updater policy

## Required CI verification

1. `Ghost FTP quality`: frontend checks, code audit, dead-code audit, desktop UX, security and version governance
2. `Ghost FTP protocol E2E`: real localhost FTP, explicit FTPS, SFTP roundtrips, trust policy and disconnect/recovery tests
3. `Ghost FTP native build`: Windows/Linux native builds, installer/packaging and capture evidence
4. `Validate Windows hardening`: native filesystem, credentials and Windows security regression tests

Tests must pass against the **exact last commit** and final artifacts, not a prior successful build. Stale screenshots are not substitute evidence. The active 30 Windows + 29 Linux references remain unverified until native capture and interaction assessment.

## Known acceptance gaps

