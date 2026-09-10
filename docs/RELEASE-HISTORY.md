# Ghost FTP release history

## 0.0.3 — 2026-09-10

Ghost FTP 0.0.3 adds real bandwidth-aware transfer controls and completes the public packaging transition while preserving the 0.0.2 security/privacy and navigation baseline.

### Application and settings

- Added independent upload/download bandwidth ceilings on Windows and Linux with explicit binary KiB/s units and `0 = unlimited`.
- Added conservative aggregate directional allocation across configured worker slots so concurrent transfers cannot multiply the configured limit.
- Enforced the resulting attempt budget in the actual FTP/FTPS/SFTP transport path through curl `limit-rate` or OpenSSH `sftp -l`.
- New/retried attempts observe newly saved limits while already-running child processes retain their start-time snapshot.
- Added shared configuration, scheduler, transport conversion and Windows/Linux settings-surface regression coverage.

### Release engineering

- Replaced five architecture-specific public Windows downloads with two universal Setup/Portable executables while retaining verified native x64/x86 payloads internally.
- Bound native Windows selection to system architecture information, verified staged embedded payload bytes and introduced no runtime download.
- Promoted Debian/Ubuntu/Fedora/Portable Linux packages to canonical public release artifacts.
- Expanded the canonical allow-list to **14 platform artifacts / 17 public files**.
- Preserved Debian/Ubuntu/Fedora package metadata/binary parity gates and Debian 13, Ubuntu 26.04 LTS and Fedora 44 native lifecycle/GUI smoke.
- Preserved exact-head/exact-main gating, immediate/delayed release read-back, GHCR verification and latest-only retention.

### Security and privacy

- Preserves strict FTPS certificate/hostname verification and no silent downgrade.
- Preserves strict SFTP host-key verification/pinning and trusted Linux transport/AskPass provenance.
- Preserves rooted local path/transfer protections, staged activation/rollback, protected saved-secret handling and connection-generation binding.
- Preserves the no-telemetry, no-analytics, no-advertising, no-fingerprinting and no-hidden-backend contract.

### Release quality

- Universal Windows Setup and Portable public executables with verified internal x64/x86 payloads.
- Debian DEB amd64/arm64/i386; Ubuntu DEB amd64/arm64/i386; Fedora RPM x86_64/aarch64/i686; Portable tar.gz amd64/arm64/i386.
- **14 platform artifacts / 17 public files**.
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 native lifecycle/GUI smoke.
- Exact-head and exact post-merge workflow verification before publication.
- Current source identity `ghostftp-v0.0.3` with `prerelease=false` once canonical publication is authorized and completed.
- Verified distribution bundle target `ghcr.io/bren-wp/ghost-ftp:0.0.3`.

## 0.0.2 — 2026-09-10

Ghost FTP 0.0.2 advances the current public release line with reliability hardening and production-ready local/server navigation workflows while preserving the 0.0.1 security and privacy baseline.

### Application and navigation

- Added non-destructive current-folder filtering on Windows and Linux without hidden listing/network I/O while filtering the loaded snapshot.
- Added bounded recursive local/server search with explicit nested-folder I/O disclosure, cancellation, hard limits, incremental results and fresh-list navigation.
- Added conservative local/server directory comparison with `same`, `local_only`, `remote_only`, `newer_local`, `newer_remote`, `conflict` and `unknown` states.
- Added synchronized navigation only for exact paired ordinary directories proven safe on both sides; both targets are freshly listed before either pane path is committed.
- Kept comparison read-only: it grants no transfer, delete, rename, overwrite or CHMOD authority.
- Localized the new filter/search/comparison workflows across all 24 supported desktop languages.

### Reliability and review hardening

- Made timestamp comparison directional so saturated `time.Duration` arithmetic cannot turn extreme timestamp differences into false equality.
- Treat zero-size comparison ambiguity conservatively as `unknown` where remote metadata cannot prove the size fact was present.
- Windows comparison synchronizes row selection between both dedicated ListViews and invalidates stale comparison state across disconnect/reconnect generation changes.
- Linux comparison preserves authoritative pre-filter snapshots and restores both panes after cancellation or asynchronous listing failure.
- Added cross-platform regression contracts for file-filter, recursive-search, comparison, action wiring and fresh-list synchronized navigation behavior.
- Hardened legacy settings migration so omitted `parallelism=0` becomes the safe default 2 while explicit invalid values remain rejected.

### Release engineering

- The canonical release-branch lifecycle identifies and waits for the exact newly dispatched publish run before retention can begin.
- Canonical retention is explicitly dispatched and verified after successful publication, with the existing `workflow_run` trigger retained as defense in depth.
- Release-note generation no longer infers Beta/prerelease status from semantic major version zero.
- Windows release verification rejects unexpected executable files outside the canonical Setup/Portable x64/x86 contract.

### Security and privacy

- Preserves strict FTPS certificate/hostname verification and no silent downgrade.
- Preserves strict SFTP host-key verification/pinning and trusted Linux transport/AskPass provenance.
- Preserves rooted local path/transfer protections, staged activation/rollback, protected saved-secret handling and connection-generation binding.
- Preserves the no-telemetry, no-analytics, no-advertising, no-fingerprinting and no-hidden-backend contract.

### Release quality

- Windows Setup x64/x86, x32 compatibility alias and Portable x64/x86.
- Linux DEB/tar.gz for amd64/arm64/i386 plus multiarch ZIP.
- **12 platform artifacts / 15 public files**.
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 native lifecycle/GUI smoke.
- Exact-head and exact post-merge workflow verification before publication.
- Historical release identity `ghostftp-v0.0.2` with `prerelease=false`.
- Historical GHCR distribution bundle `ghcr.io/bren-wp/ghost-ftp:0.0.2`.

## 0.0.1 — 2026-09-09

Ghost FTP 0.0.1 started the current public release line.

### Application

- Native Windows and Linux desktop client using the same typed FTP/FTPS/SFTP engine.
- Local/Remote dual-pane file workflow, Site Manager, transfer queue and 24-language local UI contract.
- Built-in Remote Edit for supported remote text files on FTP, FTPS and SFTP.
- Remote Edit uses bounded text handling, revision/conflict detection, line-ending preservation, transactional upload/read-back verification and post-save metadata refresh.

### Security and privacy

- Strict FTPS certificate/hostname verification with no silent downgrade.
- Strict SFTP host-key verification/pinning.
- Trusted Linux transport and credential-bearing AskPass executable provenance.
- Rooted local file/transfer protections and staged activation/rollback.
- State-directory identity pinning.
- Windows installer/uninstaller/shortcut ownership and exact-object cleanup hardening.
- No telemetry, analytics, advertising, tracking, automatic crash upload or hidden product backend.

### UI and platform quality

- Focused, non-cluttered file-management layout.
- Responsive Windows work-area and mixed-DPI geometry.
- Remote Edit exposed as one clear action rather than a permanent additional pane.
- Windows and Linux behavior routed through the same engine contract where platform-native presentation permits.

### Release quality

- Windows Setup x64/x86, x32 alias and Portable x64/x86.
- Linux DEB/tar.gz for amd64/arm64/i386 plus multiarch ZIP.
- **12 platform artifacts / 15 public files**.
- Debian 13, Ubuntu 26.04 LTS and Fedora 44 native lifecycle/GUI smoke.
- Exact-head and exact post-merge workflow verification.
- Historical release identity `ghostftp-v0.0.1` with `prerelease=false`.
- Historical GHCR distribution bundle `ghcr.io/bren-wp/ghost-ftp:0.0.1`.

## Public history retention policy

Only the latest public Ghost FTP version is retained after successful publication and remote verification. Older GitHub Releases, `ghostftp-v*` tags, superseded versioned release branches and obsolete package versions are removed by `.github/workflows/release-retention.yml` while the current release package and current canonical branch are retained.

The Git commit history on `main` is not rewritten by this policy.

The next release must start from the verified current source and advance through a separate release-prep change after the current publication and retention cycle succeeds.
