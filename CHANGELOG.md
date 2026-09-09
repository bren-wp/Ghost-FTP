# Changelog

## 0.0.1 - 2026-09-09 Beta

### Desktop client

- Unified the Windows and Linux desktop experience around the same typed Ghost FTP engine for FTP, FTPS and SFTP.
- Added the built-in Remote Editor for regular text files on FTP/FTPS/SFTP with Save, Reload and Close flows, UTF-8 validation, bounded editing, conflict detection, revision checks and line-ending preservation.
- Refresh remote list metadata after a successful remote edit while preserving the edited file selection and the successful save result.
- Improved responsive Windows geometry for small work areas, mixed-DPI monitors and negative monitor origins.
- Kept the UI intentionally compact: high-value file actions are exposed without permanently adding extra panels or clutter.

### Security and privacy

- Preserved strict FTPS certificate/hostname verification and explicit no-downgrade behavior.
- Preserved strict SFTP host-key verification and pinning.
- Hardened Linux transport executable and AskPass provenance against user-controlled PATH and executable substitution.
- Hardened settings state-directory identity, Windows installer-directory identity, shortcut ownership, legacy uninstaller ownership and integrated uninstall cleanup.
- Use verified-handle/exact-object deletion for sensitive Windows cleanup paths instead of pathname-only deletion authority.
- Keep telemetry, analytics, advertising, tracking, hidden product services and automatic crash upload disabled.

### Packaging and release quality

- Windows: Setup x64/x86, x32 compatibility alias, Portable x64/x86.
- Linux: DEB and portable tar.gz for amd64/arm64/i386 plus the multiarch ZIP.
- Canonical release shape remains **12 platform artifacts / 15 public files**.
- Debian 13, Ubuntu 26.04 LTS and Fedora 44 lifecycle/GUI smoke remain part of release verification.
- Version 0.0.1 starts a new public Beta line. After a newly published Ghost FTP release is verified, the release-retention workflow removes older Ghost FTP GitHub releases and tags so only the latest public version remains.

### Documentation

- Reset active documentation to the 0.0.1 Beta line.
- Removed the old 1.x public-version narrative from active documentation and release history.
- Kept the production engineering/audit prompt and the `ghostftp.com` dark-theme redesign prompt aligned with the current project contract.

### Release contract

The 0.0.1 Beta candidate must pass exact-head Go formatting/race/vet, repository/platform/security/privacy/documentation/release audits, the Python regression suite, Windows x64/x86 production builds, Linux amd64/arm64/i386 production builds, distro package parity, Debian/Ubuntu/Fedora lifecycle smoke and authentic Windows UI evidence. Publication uses `ghostftp-v0.0.1`, `prerelease=true`, the exact 15-file GitHub Release allow-list and the latest-only public release retention policy.
