# Ghost FTP roadmap

Ghost FTP **0.0.3** is the current source/release candidate. The roadmap prioritizes correctness, security, privacy, reliability, Windows/Linux parity and measured performance before broad new surface area. The objective is not to reproduce legacy FTP clients screen-for-screen; Ghost FTP should deliver a smaller, clearer and safer professional workflow while adding power-user capabilities only when their complete runtime path is production-ready.

## Current 0.0.3 foundation

The current release gate includes:

- native Windows and Linux desktop clients backed by one typed Engine;
- FTP, explicit FTPS and SFTP workflows;
- SFTP password/key/passphrase authentication and host-key trust;
- local-only profiles with platform-local protected secret handling;
- validated connection timeout, retry, conflict, parallel-transfer and upload/download bandwidth settings;
- transfer generation binding, source snapshots and staged/rollback-oriented operations;
- privacy-safe connection diagnostics;
- truthful transfer progress, speed and ETA;
- aggregate directional bandwidth scheduling with real curl/OpenSSH transport enforcement;
- built-in Remote Edit with bounded text handling, revision/conflict protection and verified save/read-back;
- queue pause/resume/cancel/retry/clear-finished controls;
- non-destructive current-folder filtering on both local and server panes;
- bounded recursive local/server search with cancellation and fresh-list navigation;
- conservative directory comparison and synchronized navigation for proven paired directories;
- two public universal Windows Setup/Portable executables backed by verified internal x64/x86 payloads;
- canonical Debian/Ubuntu/Fedora/Portable Linux release packaging;
- 24-language local catalog with English default/fallback;
- production race/vet/security/privacy/dependency/documentation audits;
- current GitHub Release policy with `prerelease=false`;
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

## 0.0.3 navigation work

Ghost FTP 0.0.3 retains the **non-destructive current-folder filter** introduced in 0.0.2 for both local and server panes on Windows and Linux. It filters only entries already loaded into the pane, performs no filesystem or network scan while filtering, preserves an authoritative unfiltered snapshot, and gives row-indexed actions only the visible filtered slice. Empty input restores the complete snapshot without another listing request. The filter is localized for all 24 supported desktop languages.

Ghost FTP 0.0.3 also retains the **bounded recursive local/server search** that is deliberately separate from the current-folder filter. Recursive search clearly discloses that it reads nested local or server folders, uses the same Unicode-aware matching semantics, incrementally presents bounded result batches, supports cancellation, never intentionally traverses symlink/reparse entries, and treats every result as an informational navigation hint rather than mutation authority. Local traversal is anchored to one `os.OpenRoot` capability; server traversal is bound to one captured remote operation/session. Activating a result performs a fresh listing of its parent and reselects the name only from that fresh listing.

Ghost FTP 0.0.3 retains **conservative directory comparison and synchronized navigation** on Windows and Linux. Comparison is read-only, uses exact-name matching, reports deterministic same/only/newer/conflict/unknown states, treats duplicate names and uncertain metadata fail-closed, and never treats a symlink comparison row as transfer authority. Synchronized navigation is available only for an exact ordinary directory proved present on both sides; both target directories are freshly listed and compared before either visible pane path is committed.

These navigation capabilities remain part of the current 0.0.3 source/release candidate while their original 0.0.2 introduction stays recorded in release history.

## High-value power-user lane

These capabilities are prioritized because they improve real hosting/server workflows. Items not explicitly marked implemented are **planned, not advertised as shipped**, until all acceptance gates below are satisfied.

### P0 — directory comparison and synchronized navigation

**Status: implemented in Ghost FTP 0.0.3.** Windows and Linux expose the same shared comparison states while keeping comparison read-only and separate from ordinary file-operation authority. The capability originated in 0.0.2 and remains part of the current release contract.

Implemented contract:

- shared `internal/directorycompare` logic compares already-listed snapshots without filesystem/network I/O or input mutation;
- exact-name matching avoids unsafe case folding across filesystems with different case semantics;
- deterministic states are `same`, `local_only`, `remote_only`, `newer_local`, `newer_remote`, `conflict` and `unknown`;
- duplicate exact names fail closed to `conflict`, type mismatches fail closed to `conflict`, and symlinks fail closed to `unknown`;
- regular-file `newer_*` and `same` classification is used only when both sides provide usable modification times; equal-size files with unknown time remain `unknown`, while differing sizes with unknown time remain `conflict`;
- timestamp comparison defaults to a two-second tolerance and rejects an override above five minutes;
- synchronized navigation resolves only a `same` entry that is an ordinary non-symlink directory present on both sides;
- Windows renders comparison in dedicated read-only ListViews and disables normal rename/delete/upload/download/edit/chmod authority while the comparison surface is active;
- Linux uses a modal comparison surface and consumes ordinary workspace clicks while comparison rows are displayed, so row indices cannot leak into normal file actions;
- “Open both” validates the local child and remote name/path, performs fresh local and remote listings, recomputes comparison, and only then commits both pane paths;
- localized comparison controls/status/disclosure copy exists for all 24 supported desktop languages;
- `scripts/test_directory_comparison_contract.py` protects cross-platform wiring, mutation gating and the fresh-list-before-path-commit rule;
- comparison itself never uploads, downloads, deletes, renames, overwrites or changes permissions.

### P0 — bounded recursive local/server search

**Status: implemented in Ghost FTP 0.0.3.** The instant current-folder filter remains I/O-free; recursive search is a separate explicit action because it performs additional local/server listing work. The capability originated in 0.0.2 and remains part of the current release contract.

Implemented contract:

- shared Unicode-aware matching keeps case behavior aligned with current-folder filtering, including Unicode simple-fold cases that lowercasing alone cannot represent;
- defaults are bounded to depth 12, 20,000 visited items, 1,000 results, batches of 50 and 20 seconds; hard ceilings are depth 32, 50,000 visited items, 5,000 results, batches of 200 and 60 seconds;
- cancellation propagates through listing contexts; a remote scan owns one operation/session for its lifetime and cannot jump to a reconnected session;
- local scanning is rooted through `os.OpenRoot`, and symlink/reparse entries are not intentionally traversed;
- Windows uses dedicated read-only search-result ListViews and Linux keeps normal row-indexed actions modal/disabled while recursive snapshots are displayed;
- large result sets are incrementally presented rather than accumulated on the UI thread before first display;
- activating a result closes search mode, performs a fresh parent listing and reselects the discovered name only if it is present in that authoritative listing;
- the UI discloses nested-folder I/O and the default 20-second / 1,000-result bound before search starts.

### P0 — bandwidth-aware transfer controls

**Status: implemented in Ghost FTP 0.0.3.** Upload/download limits are shared validated runtime policy, not decorative frontend timers.

Implemented contract:

- independent upload and download limits use explicit binary KiB/s units with `0 = unlimited`;
- the shared configuration layer validates the maintained `0..1,048,576 KiB/s` range and preserves migration-safe unlimited defaults for older settings;
- the transfer scheduler defines a conservative aggregate directional ceiling across configured worker slots rather than granting every concurrent transfer the complete configured budget;
- each transfer attempt snapshots its effective budget when it starts, so a settings change cannot corrupt a running child process and later/retried attempts can observe the new policy;
- FTP/FTPS use curl `limit-rate` and SFTP uses OpenSSH `sftp -l` after conservative KiB/s→Kbit/s conversion;
- no busy-wait throttling or UI-only speed cap is used;
- Windows and Linux settings expose the same persisted upload/download model with localized `KiB/s` labels.

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

New productivity features must preserve the same trust model: search, compare, bandwidth, bookmarks, resume, multi-session and proxy functionality may not introduce hidden cloud state, telemetry or credential replication.

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
