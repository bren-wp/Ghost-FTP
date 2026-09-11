# Changelog

## Unreleased

### Queue priority and reordering

- Completed four-way queued-transfer priority control with **Top**, **Up**, **Down** and **Bottom** actions through the shared transfer manager and Engine API.
- Reordering rotates only queued scheduler slots, preserving running/terminal history positions, transfer IDs, connection bindings and the relative order of unaffected queued jobs.
- Kept edge moves idempotent and event-free, while a real move emits one complete state snapshot without starting, retrying or cancelling transfer work as a side effect.
- Added Windows and Linux controls with the same queued-only policy, ID-based selection restoration and local copy for all 24 supported desktop languages.
- Added regression coverage for four-way ordering, non-queued slot stability, connection binding, tree-transfer directory-preparation ordering and cross-platform UI wiring.
- This is post-0.0.3 source work for the next release; root `VERSION` and the already published `ghostftp-v0.0.3` release remain unchanged.

### Navigation bookmarks and profile start directories

- Added reusable local and remote bookmarks through a shared non-secret persistence model; remote bookmark identity contains only protocol, host, port and username and is captured from the real active Engine connection.
- Made bookmark activation authoritative only after fresh navigation: local targets must list successfully, while remote targets must match the active account and retain the same connection identity across the server listing.
- Added a native Windows bookmark manager and Linux X11 bookmark overlay with Open, Add local, Add remote, Delete and Close behavior wired through the shared Engine contract.
- Added an additional Windows `connectionGeneration` guard around remote bookmark UI commits and Linux modal/action routing that keeps stored paths from bypassing the existing navigation lifecycle.
- Hardened profile start-directory behavior so an inherited server path cannot silently cross a protocol/host/port/username account boundary; Linux also restores the previous verified local base and requires a successful local listing before a selected profile start becomes pane state.
- Added Go coverage plus `scripts/test_navigation_bookmarks_contract.py` for non-secret persistence, corrupt-state fail-closed behavior, account/session revalidation, profile-start isolation, cross-platform UI wiring and documentation/release boundaries.
- Added `docs/NAVIGATION-BOOKMARKS.md`; this remains post-0.0.3 source work targeted for the next public release, while root `VERSION` and the already published `ghostftp-v0.0.3` release remain unchanged.

## 0.0.3 - 2026-09-10

### Bandwidth-aware transfers

- Added independent upload and download bandwidth ceilings with validated local settings on Windows and Linux.
- Added conservative aggregate directional scheduling so configured KiB/s budgets remain bounded across concurrent transfer worker slots instead of becoming a per-transfer multiplier.
- Enforced the effective transfer budget in the real transport path: curl `limit-rate` for FTP/FTPS and OpenSSH `sftp -l` for SFTP.
- Kept `0` as the migration-safe unlimited default for settings written by Ghost FTP 0.0.2 and older builds.
- Added validation, migration, aggregate-allocation, transport-conversion and Windows/Linux settings-surface regression coverage.
- Defined attempt-scoped bandwidth policy: a running transport keeps the budget sampled when it starts, while new and retried attempts observe the newly saved setting.

### Packaging and release engineering

- Reduced the public Windows download surface to two self-contained files: `Ghost-FTP-0.0.3-Setup.exe` and `Ghost-FTP-0.0.3-Portable.exe`.
- Preserved verified native x64/x86 application payloads internally and added a bootstrap that selects the native payload from `GetNativeSystemInfo`, performs no runtime download and verifies staged bytes before execution.
- Kept integrated uninstall ownership and optional fail-closed Authenticode behavior while preventing architecture-specific staging executables from leaking into the public artifact directory.
- Promoted `linux/BUILD-DISTROS.sh` to the canonical Linux release builder.
- Added canonical Debian and Ubuntu DEBs for `amd64`, `arm64`, `i386`; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and distro-neutral Portable tarballs for `amd64`, `arm64`, `i386`.
- Preserved one compiled executable per architecture across matching Debian/Ubuntu/Fedora/Portable packages and byte-parity verification before publication.
- Updated the public release allow-list to **14 platform artifacts / 17 public files**, including immediate and delayed GitHub Release asset read-back and matching latest-only retention preflight.
- Kept Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 native install/remove/GUI lifecycle gates; additional architectures retain build, metadata, extraction and binary-parity verification.
- Strengthened the universal Windows wrapper with a direct fail-closed Go telemetry check and updated authentic UI evidence to capture the verified internal native x64 payload without making it a public download.

### Security and privacy

- Preserved strict FTPS certificate/hostname validation, SFTP host-key verification/pinning, trusted Linux AskPass provenance, rooted local path protections, protected-secret lifetime rules and no silent secure-to-plain downgrade.
- Preserved the no-telemetry, no-analytics, no-advertising, no-fingerprinting and no-hidden-backend contract across the new bandwidth and packaging paths.

## 0.0.2 - 2026-09-10

### Reliability and release engineering

- Made the canonical release-branch lifecycle wait for the exact newly dispatched `Publish Ghost FTP` workflow run and require terminal success before retention can begin.
- Added explicit canonical retention dispatch/read-back after a successful publish run so latest-only cleanup does not depend on `workflow_run` event chaining alone.
- Added exact-main workflow-run identification and fail-closed release/retention completion checks.
- Fixed release-note generation so semantic major version zero no longer implies Beta/prerelease and current 0.0.x releases retain their verified GitHub Packages section.
- Hardened Windows release verification so the public artifact directory rejects any unexpected executable outside the canonical Setup/Portable x64/x86 naming contract; the no-permanent-uninstaller claim is backed by an observed artifact check.

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
- Added conservative local/server directory comparison on Windows and Linux with deterministic `same`, `local_only`, `remote_only`, `newer_local`, `newer_remote`, `conflict` and `unknown` states. Exact-name matching prevents unsafe case folding, duplicate names fail closed to conflict, symlinks remain unknown, zero-size ambiguity remains unknown, and file freshness is never inferred when either side lacks reliable modification time metadata.
- Added explicit synchronized navigation for comparison rows proven to be ordinary directories on both sides. “Open both” validates local and remote child paths, performs fresh listings of both targets, recomputes comparison, and only then commits both pane paths; comparison itself never transfers, deletes, renames, overwrites or changes permissions.
- Windows comparison uses dedicated read-only ListViews, synchronized row selection and connection-generation invalidation; Linux comparison is modal, preserves authoritative pre-filter snapshots and restores them after cancellation or listing failure.
- Added localized comparison controls/statuses for all 24 supported desktop languages and a cross-platform source regression contract protecting comparison semantics, UI wiring, mutation gating, snapshot recovery and fresh-list synchronized navigation.
- Added a regression contract that rejects main Windows buttons without command handlers and Linux controls/overlays without click handlers.
- Added a guard against silently discarded Windows queue Cancel/Retry errors.
- Defined complete acceptance criteria for future bandwidth control, queue priority, bookmarks, verified resume, multi-session and proxy/jump-host capabilities before they may appear as shipped UI.

### Documentation and product media

- Redesigned the root README around the repository-local Ghost FTP icon and authentic Main Workspace, Site Manager, Settings and About screenshots.
- Reorganized the documentation index around installation, settings, security/privacy, architecture, UI evidence, testing and release verification.
- Added a README/media regression contract that rejects remote image sources, missing local media and loss of authentic screenshot provenance.
- Expanded Settings, Testing, GitHub Releases and Roadmap documentation to match the hardened runtime/release contracts.
