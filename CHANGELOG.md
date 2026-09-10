# Changelog

## Unreleased

### Reliability and release engineering

- Made the canonical release-branch lifecycle wait for the exact newly dispatched `Publish Ghost FTP` workflow run and require terminal success before retention can begin.
- Added explicit canonical retention dispatch/read-back after a successful publish run so latest-only cleanup does not depend on `workflow_run` event chaining alone.
- Added exact-main workflow-run identification and fail-closed release/retention completion checks.
- Fixed release-note generation so semantic major version zero no longer implies Beta/prerelease and current 0.0.x releases retain their verified GitHub Packages section.
- Hardened Windows release verification so the public artifact directory rejects any unexpected executable outside the canonical Setup/Portable x64/x86 naming contract; the no-permanent-uninstaller claim is now backed by an observed artifact check.

### Settings and compatibility

- Safely migrate omitted legacy `parallelism=0` saves to the canonical default of 2 while continuing to reject explicit negative or above-range values.
- Expanded settings regression coverage so compatibility migration cannot silently weaken current validation rules.

### Desktop quality

- Added a non-destructive local/server current-folder filter on Windows and Linux. Filtering is case-insensitive with Unicode simple-fold behavior, supports multiple whitespace-delimited name tokens, never mutates the authoritative directory snapshot, and performs no hidden filesystem/network scan while the user filters already loaded entries.
- Keep row-indexed rename/delete/upload/download actions bound to the filtered visible slice so an action cannot target a hidden item after filtering; Windows server snapshots are additionally bound to the active connection generation and Linux clears stale server source data on disconnect.
- Added localized filter controls for all 24 supported desktop languages plus shared filter semantics and cross-platform UI wiring regression coverage.
- Added bounded recursive local/server search as a separate explicit action on Windows and Linux. Recursive search performs disclosed nested-folder I/O with validated depth/item/result/batch/time limits, cancellation, incremental result presentation, Unicode-aware matching, root/session confinement and no intentional symlink/reparse traversal.
- Recursive search results are informational navigation snapshots only: Windows renders them in dedicated ListViews, Linux keeps normal row-indexed actions modal/disabled, and “Go to result” performs a fresh parent listing before reselecting the discovered name.
- Added localized recursive-search Search/Navigate/Cancel/Close/progress/disclosure copy for all 24 supported desktop languages and a source regression contract covering hard bounds, platform wiring and fresh-list navigation.
- Added a regression contract that rejects main Windows buttons without command handlers and Linux controls/overlays without click handlers.
- Added a guard against silently discarded Windows queue Cancel/Retry errors.
- Defined complete acceptance criteria for future directory comparison, synchronized browsing, bandwidth control, queue priority, bookmarks, verified resume, multi-session and proxy/jump-host capabilities before they may appear as shipped UI.

### Documentation and product media

- Redesigned the root README around the repository-local Ghost FTP icon and authentic Main Workspace, Site Manager, Settings and About screenshots.
- Reorganized the documentation index around installation, settings, security/privacy, architecture, UI evidence, testing and release verification.
- Added a README/media regression contract that rejects remote image sources, missing local media and loss of authentic screenshot provenance.
- Expanded Settings, Testing, GitHub Releases and Roadmap documentation to match the hardened runtime/release contracts.

## 0.0.1 - 2026-09-09

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
- Version 0.0.1 starts the new current public line. After a newly published Ghost FTP release is verified, the release-retention workflow removes older Ghost FTP GitHub releases and tags so only the latest public version remains.
- The verified distribution bundle is published to `ghcr.io/bren-wp/ghost-ftp:0.0.1` with matching current-version aliases.

### Documentation

- Reset active documentation to the 0.0.1 public line.
- Removed the old 1.x public-version narrative from active documentation and release history.
- Kept the production engineering/audit prompt and the `ghostftp.com` dark-theme redesign prompt aligned with the current project contract.

### Release contract

The 0.0.1 candidate must pass exact-head Go formatting/race/vet, repository/platform/security/privacy/documentation/release audits, the Python regression suite, Windows x64/x86 production builds, Linux amd64/arm64/i386 production builds, distro package parity, Debian/Ubuntu/Fedora lifecycle smoke and authentic Windows UI evidence. Publication uses `ghostftp-v0.0.1`, `prerelease=false`, the exact 15-file GitHub Release allow-list, verified GHCR bundle read-back and the latest-only public release retention policy.
