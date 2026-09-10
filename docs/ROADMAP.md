# Ghost FTP roadmap

Ghost FTP **0.0.1** starts the current public release line. The roadmap prioritizes correctness, security, privacy, reliability, Windows/Linux parity and measured performance before broad new surface area. The objective is not to reproduce legacy FTP clients screen-for-screen; Ghost FTP should deliver a smaller, clearer and safer professional workflow while adding power-user capabilities only when their complete runtime path is production-ready.

## Current 0.0.1 foundation

The current release gate includes:

- native Windows and Linux desktop clients backed by one typed Engine;
- FTP, explicit FTPS and SFTP workflows;
- SFTP password/key/passphrase authentication and host-key trust;
- local-only profiles with platform-local protected secret handling;
- validated connection timeout, retry, conflict and parallel-transfer settings;
- transfer generation binding, source snapshots and staged/rollback-oriented operations;
- privacy-safe connection diagnostics;
- truthful transfer progress, speed and ETA;
- built-in Remote Edit with bounded text handling, revision/conflict protection and verified save/read-back;
- queue pause/resume/cancel/retry/clear-finished controls;
- native Windows Setup/Portable packaging and Linux DEB/portable packaging;
- 24-language local catalog with English default/fallback;
- production race/vet/security/privacy/dependency/documentation audits;
- current GitHub Release publication with `prerelease=false`;
- truthful Windows signing-state metadata with fail-closed verification when trusted production signing is configured;
- current GitHub Packages/GHCR distribution-bundle publication and read-back;
- latest-only release/tag/package retention after a successor is successfully published and verified.

## Quality work before new surface area

The immediate 0.0.x hardening lane includes:

1. deterministic release → remote read-back → retention orchestration, including explicit verification of the exact release and retention workflow runs;
2. regression coverage that rejects visible desktop controls with no action handler;
3. compatibility-safe settings migration without weakening validation of explicit invalid values;
4. connection lifecycle, stale-session state and reconnect correctness;
5. cancel/retry/partial-transfer/interrupted-transfer behavior and actionable queue errors;
6. transfer atomicity, overwrite decisions, temp-file cleanup and application-shutdown behavior;
7. large-directory/list memory and UI responsiveness;
8. further Remote Edit shutdown/disconnect/conflict edge cases;
9. Windows/Linux functional parity for file operations, queue state, shortcuts, settings and error handling;
10. documentation and authentic real-application screenshots synchronized with exact maintained source.

## 0.0.x work implemented after 0.0.1

The maintained source now includes a **non-destructive current-folder filter** for both local and server panes on Windows and Linux. It filters only entries already loaded into the pane, performs no filesystem or network scan while filtering, preserves an authoritative unfiltered snapshot, and gives row-indexed actions only the visible filtered slice. Empty input restores the complete snapshot without another listing request. The filter is localized for all 24 supported desktop languages. This work remains part of the Unreleased source line until a successor release is published and verified; it does not rewrite or redefine the existing 0.0.1 release.

## High-value power-user lane

These capabilities are prioritized because they improve real hosting/server workflows. Remaining items are **planned, not advertised as shipped**, until all acceptance gates below are satisfied.

### P0 — directory comparison and synchronized navigation

A comparison mode should pair the current local and server directories and classify entries as same, local-only, server-only, newer-local, newer-server or conflicting/unknown. Optional synchronized browsing should move the opposite pane only when both sides can resolve the corresponding directory safely.

Acceptance requirements:

- comparison logic lives in shared testable code rather than duplicated frontend logic;
- timestamps with unreliable server precision are treated conservatively;
- symlinks and unsupported metadata never trigger destructive automatic action;
- Windows and Linux expose the same states and disable synchronization when the mapping is ambiguous;
- comparison itself never transfers or deletes data.

### P0 — bounded recursive local/server search

The fast current-folder filtering portion of the original search/filter roadmap is implemented in the Unreleased source line. The remaining search work is an explicitly bounded recursive mode; it must be visually and behaviorally distinct from the current-folder filter so users can tell when Ghost FTP will perform additional filesystem/server I/O.

Acceptance requirements:

- case behavior is explicit and platform-independent where practical;
- recursive server search has strict item, depth and time bounds;
- cancellation is immediate and leaves the connection usable;
- search results cannot make a hidden or stale destructive target actionable;
- large result sets are incrementally presented rather than blocking the UI thread;
- starting a recursive search clearly communicates that additional local/server listing work will occur.

### P0 — bandwidth-aware transfer controls

Add optional upload/download rate limits with **unlimited** as the default. Limits belong in the shared transfer/transport layer, not as decorative frontend timers.

Acceptance requirements:

- separate upload and download limits;
- units are explicit and validated;
- aggregate behavior is defined when parallel transfers are enabled;
- changing a limit cannot corrupt an active transfer;
- no busy-wait throttling and no misleading UI-only speed cap;
- Windows/Linux settings use the same persisted model.

### P0 — queue priority and reorder

Add user-controlled queue priority/reordering while retaining connection-generation safety.

Acceptance requirements:

- reordering queued jobs never changes the identity of already-running work;
- parent/child tree-transfer ordering remains valid;
- selected jobs can move up/down/top/bottom without duplicate execution;
- pause/resume/cancel/retry semantics remain deterministic;
- queue ordering survives only where doing so cannot resurrect stale server work.

### P1 — navigation bookmarks and profile start directories

Allow reusable local/server bookmarks plus explicit default local/server directories per site profile.

Acceptance requirements:

- bookmarks store paths, never credentials;
- stale local paths and unavailable server paths produce actionable errors;
- profile identity changes cannot inherit unrelated server paths silently;
- quick-connect bookmarks do not create hidden persistent profiles.

### P1 — stronger interrupted-transfer resume

Introduce verified resume where the underlying protocol/server semantics can prove the partial object is the intended object. Unsupported cases must fall back to a safe restart, not append blindly.

Acceptance requirements:

- remote/local object identity and existing length are checked before resume;
- overwrite/backup policy still applies;
- checksum/read-back verification is used where available and appropriate;
- a failed resume never leaves the destination reported as successfully complete.

### P1 — multiple live sessions / tabs

Multiple server sessions can provide a significant productivity gain, but this must not weaken the current single-session generation/identity invariants.

Acceptance requirements:

- every queue item is bound to one explicit session identity;
- tab close cancels/drains only that session's work;
- saved secrets are never copied between tabs implicitly;
- UI status and transfer ownership remain unambiguous;
- resource limits prevent unbounded session creation.

### P1 — proxy / jump-host boundary

Proxy or SSH jump-host support is useful only if transport trust remains explicit. Any implementation must make the destination identity and intermediary identity clear and must not silently weaken TLS or SSH host verification.

### P2 — profile import/export

Portable profile exchange may be added for non-secret site metadata. Secret export must remain opt-in, strongly protected and clearly separated from ordinary profile export; plaintext credential export is not an acceptable default.

## UI/product rules for new features

A visible control is not a feature definition. Every new button, menu item, shortcut or option must have:

1. a clear shared-engine or platform-runtime owner;
2. disabled/enabled-state rules;
3. success, failure and cancellation behavior;
4. user-safe error handling;
5. Windows/Linux exposure or a documented platform reason;
6. localization coverage for user-visible text;
7. unit/integration/regression coverage;
8. updated active documentation;
9. authentic UI evidence when the maintained desktop surface changes.

The maintained UI wiring regression must continue to reject a main desktop button that has no handler.

## Performance direction

Optimization work should target measured hotspots:

- avoid full-workspace redraw when state is unchanged;
- keep UI work out of transfer/network critical paths;
- avoid repeated filesystem scans when a validated snapshot is sufficient;
- keep transfer progress publication bounded and truthful;
- reduce unnecessary allocations/copies in listing and transfer planning;
- incrementally render or virtualize very large directory/search result sets where needed;
- batch UI invalidation rather than repainting for every transfer event;
- prevent goroutine leaks, stale callbacks and incomplete timeout cleanup;
- keep CI deterministic and offline for the Go dependency graph.

## Security/privacy direction

Security hardening should favor deterministic rejection and actionable errors over permissive fallback. Privacy improvements should reduce secret lifetime and diagnostic exposure rather than adding remote reporting.

Release security should improve publisher trust when a real certificate is available without weakening integrity verification or inventing trust when it is not.

New productivity features must preserve the same trust model: search, compare, bookmarks, resume, multi-session and proxy functionality may not introduce hidden cloud state, telemetry or credential replication.

## macOS direction

macOS remains a separate future scope. Documentation must not claim macOS support until the common engine has been audited for Darwin, platform contracts and CI/build gates exist, and a real native macOS frontend is built and tested. Any future macOS implementation must reuse the same FTP/FTPS/SFTP engine, security/privacy rules and Remote Edit logic rather than introducing a third protocol stack.

## Non-negotiable constraints

Future work must preserve:

- no application telemetry/advertising/fingerprinting;
- no mandatory Ghost FTP account;
- no silent secure-transport downgrade;
- SFTP host-key verification/pinning;
- FTPS certificate/hostname verification;
- protected saved-secret handling;
- local path containment and symlink/reparse safety;
- fail-closed transfer cleanup/commit behavior;
- trusted Linux transport and AskPass provenance;
- exact-object/ownership-aware Windows installer/uninstaller cleanup;
- exact source/version binding for public releases;
- truthful Windows signing states: configured trusted signatures verify fail-closed, otherwise current publication remains explicitly `WINDOWS_AUTHENTICODE=unsigned`;
- no generated/self-signed production identity represented as a trusted publisher;
- verified GitHub Release and GitHub Package publication;
- no unreviewed external Go dependencies.

## Release direction

The public sequence is `0.0.1`, `0.0.2`, `0.0.3`, and so on. `0.0.0` is reserved. A current release identity is never rewritten in place. A newer version must pass exact-head and post-merge verification, publication, remote read-back and canonical retention verification before the release lifecycle is considered complete.

## Definition of roadmap completion

A roadmap item is complete only after code, regression tests, security/privacy implications, active documentation and production CI/release gates agree on the behavior. Git commit history remains engineering provenance; the active public release catalog follows the verified latest-only lifecycle.
