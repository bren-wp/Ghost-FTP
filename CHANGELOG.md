# Changelog

## Unreleased

No unreleased changes are currently staged after 0.0.3 release preparation.

## 0.0.3 - 2026-09-10

### Bandwidth-aware transfers

- Added independent upload and download bandwidth ceilings with validated local settings on Windows and Linux.
- Added aggregate directional scheduling so configured KiB/s budgets remain bounded across concurrent transfer workers instead of becoming a per-transfer multiplier.
- Enforced the effective transfer budget in the real transport path: curl `limit-rate` for FTP/FTPS and OpenSSH `sftp -l` for SFTP.
- Kept `0` as the migration-safe unlimited default for settings written by 0.0.2 and older builds.
- Added bounded validation, transport conversion tests, scheduler tests and Windows/Linux settings-surface regression coverage.

### Packaging and release engineering

- Reduced the public Windows download surface to two self-contained files: `Ghost-FTP-0.0.3-Setup.exe` and `Ghost-FTP-0.0.3-Portable.exe`.
- Preserved verified native x64/x86 application payloads internally and added a Windows bootstrap that chooses the native payload from `GetNativeSystemInfo`, performs no runtime download and verifies staged bytes before execution.
- Promoted `linux/BUILD-DISTROS.sh` to the canonical Linux release builder.
- Added canonical Debian and Ubuntu DEBs for `amd64`, `arm64`, `i386`; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and distro-neutral Portable tarballs for `amd64`, `arm64`, `i386`.
- Preserved one compiled executable per architecture across matching Debian/Ubuntu/Fedora/Portable packages and verify byte parity before publication.
- Updated the public release allow-list to **14 platform artifacts / 17 public files** and retained immediate/delayed remote asset read-back plus latest-only retention.
- Kept Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 native install/remove/GUI lifecycle gates.
- Strengthened the universal Windows build with direct fail-closed Go telemetry verification and updated authentic UI evidence to capture the verified internal native x64 payload without leaking architecture-specific EXEs into the public download directory.

### Security and privacy

- Preserved strict FTPS validation, SFTP host-key verification, trusted Linux AskPass provenance, rooted local path protections, protected-secret lifetime rules and no silent protocol downgrade.
- Preserved the no-telemetry/no-analytics/no-hidden-backend contract across the new universal Windows and distro packaging paths.

## 0.0.2 - 2026-09-10

### Reliability and release engineering

- Made the canonical release-branch lifecycle wait for the exact newly dispatched `Publish Ghost FTP` workflow run and require terminal success before retention can begin.
- Added explicit canonical retention dispatch/read-back after a successful publish run so latest-only cleanup does not depend on `workflow_run` event chaining alone.
- Added exact-main workflow-run identification and fail-closed release/retention completion checks.
- Fixed release-note generation so semantic major version zero no longer implies Beta/prerelease and current 0.0.x releases retain their verified GitHub Packages section.
- Hardened Windows release verification so the public artifact directory rejects any unexpected executable outside the canonical Setup/Portable x64/x86 naming contract; the no-permanent-uninstaller claim is backed by an observed artifact check.

### Settings and compatibility

- Safely migrated omitted legacy `parallelism=0` saves to the canonical default of 2 while continuing to reject explicit negative or above-range values.
- Expanded settings regression coverage so compatibility migration cannot silently weaken current validation rules.

### Desktop quality

- Added non-destructive current-folder filtering and bounded recursive local/server search on Windows and Linux.
- Added conservative directory comparison with synchronized navigation only for safely proven paired directories.
- Added localized controls and regression coverage for the maintained navigation workflows across all 24 supported desktop languages.

### Documentation and product media

- Redesigned the root README around repository-local Ghost FTP media and authentic Windows screenshots.
- Reorganized active documentation around installation, settings, security/privacy, architecture, UI evidence, testing and release verification.
