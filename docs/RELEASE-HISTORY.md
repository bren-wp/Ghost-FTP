# Ghost FTP release history

## 0.0.3 — 2026-09-10

Ghost FTP 0.0.3 adds real bandwidth-aware transfer controls and completes the public packaging transition while preserving the established security/privacy baseline.

### Application and settings

- Added independent upload/download KiB/s ceilings with `0 = unlimited`.
- Applies aggregate directional budgeting across configured workers and transport-level enforcement for FTP/FTPS/SFTP.
- Exposes the same validated setting model on Windows and Linux.

### Release engineering

- Replaced five architecture-specific public Windows downloads with two universal Setup/Portable executables while retaining native x64/x86 payloads internally.
- Promoted Debian/Ubuntu/Fedora/Portable Linux packages to canonical public release artifacts.
- Expanded the canonical allow-list to **14 platform artifacts / 17 public files**.
- Preserved package metadata/binary parity gates and Debian 13, Ubuntu 26.04 LTS and Fedora 44 native lifecycle/GUI smoke.
- Preserved exact-head/exact-main gating, remote release read-back, GHCR verification and latest-only retention.

### Security and privacy

- Preserves strict FTPS/SFTP trust, rooted local path safeguards, protected credential lifetime, no silent downgrade and no application telemetry/analytics/hidden backend.

## 0.0.2 — 2026-09-10

Ghost FTP 0.0.2 advanced the current public line with reliability hardening, non-destructive current-folder filtering, bounded recursive local/server search and conservative directory comparison with synchronized navigation.

- Added cross-platform filter/search/comparison workflows and 24-language UI coverage.
- Hardened settings migration, release-to-retention orchestration and exact-main workflow identification.
- Used the historical 12 platform artifacts / 15 public files layout with architecture-specific Windows downloads and generic Linux packages.
- Historical release identity: `ghostftp-v0.0.2`, `prerelease=false`.

## 0.0.1 — 2026-09-09

Ghost FTP 0.0.1 started the current public release line with native Windows/Linux FTP/FTPS/SFTP, dual-pane file management, Site Manager, transfer queue, Remote Edit, strict trust/path protections and a no-telemetry privacy baseline.

## Public history retention policy

Only the latest public Ghost FTP version is retained after successful publication and remote verification. Older GitHub Releases, `ghostftp-v*` tags, superseded versioned release branches and obsolete package versions are removed by `.github/workflows/release-retention.yml` while the current package and canonical release branch are retained. Git commit history on `main` is not rewritten.
