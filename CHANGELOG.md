# Changelog

## Unreleased

No unreleased changes are currently staged after 0.0.2.

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
