# Changelog

## 0.0.5 - 2026-09-12

### Windows lifecycle and re-entry reliability

- Added explicit profile-mutation ownership so encrypted profile save/delete work cannot be launched concurrently from duplicate or stale commands, and application close cannot terminate the process while that persistence mutation is still in flight.
- Hardened shared Windows modal loops so nested premium dialogs and the built-in Remote Text Editor preserve process-level `WM_QUIT` rather than swallowing an application shutdown request.
- Added local and remote file-mutation guards around create-directory, rename, delete and remote permission changes. Conflicting mutation controls are disabled only for the relevant side while unrelated navigation/transfer work keeps its existing availability policy.
- Preserved code-level guards behind UI enablement so stale command delivery cannot bypass the mutation state.
- Kept Remote Edit session work serialized across open/save/reload/close cycles so one editor session cannot be re-entered into a parallel stale session while asynchronous work is completing.

### Android lifecycle stability

- Bound every in-flight FTP/FTPS connection attempt to the owning Activity lifecycle.
- Added a non-blocking pending-session abort path so Activity destruction/recreation can cancel the socket-open/TLS connection race without waiting on UI-thread cleanup.
- Reject stale connection success/error callbacks after the owning Activity is destroyed, preventing a late connection result from restoring a session or mutating an obsolete UI instance.
- Preserved strict explicit FTPS certificate/hostname validation, SAF-only local storage, staged transfer commit/cancellation safeguards and the existing Android development-only publication boundary.

### Browser companion source

- Added optional Ghost FTP companion extension source packages under `ekstenzije/` for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox.
- Kept the companion scope intentionally narrow: supported `ftp://`, `ftps://` and `sftp://` links can be handed into the Ghost FTP workflow without application telemetry, remote executable code, credential persistence, tab scraping or broad host permissions.
- Kept browser companion packages outside the canonical 17-file Windows/Linux GitHub Release allow-list; they are source companion surfaces, not a hidden web backend or extra release binary family.

### Documentation and release engineering

- Advanced root `VERSION` to **0.0.5** while preserving the current GitHub Release channel (`prerelease=false`) and `ghostftp-vX.Y.Z` namespace.
- Rewrote the main README around user value and clear download/security/privacy positioning while keeping every public feature claim bound to implemented and tested behavior.
- Corrected `scripts/release_notes.py` so generated release notes match the actual canonical distribution: two universal Windows executables plus twelve Debian/Ubuntu/Fedora/Portable Linux artifacts, for **14 platform artifacts / 17 public files**.
- Added a regression contract that rejects the retired architecture-specific Windows filenames, old generic Linux/multiarch release names and obsolete 12/15 release counts from generated current release notes.
- Preserved fresh release builds, exact asset read-back, GHCR distribution-bundle verification and latest-only release/tag/branch/package retention.

### Security and privacy

- Preserved strict FTPS certificate/hostname validation, strict desktop SFTP host-key verification/pinning, trusted Linux transport/AskPass provenance, rooted local path/transfer protections, protected-secret lifetime rules and no silent secure-to-plain downgrade.
- Preserved the no-telemetry, no-analytics, no-advertising, no-fingerprinting, no-automatic-crash-upload, no-hidden-backend and no-mandatory-account contract across Windows, Linux, Android and the new browser companion source.

## 0.0.4 - 2026-09-11

### Windows and Linux desktop parity

- Completed Linux Light/Dark appearance parity using the same validated `Appearance` setting and shared palette contract as Windows; persisted appearance is applied before the first Linux frame and changes take effect immediately after saving Settings.
- Added Linux file-pane sorting through the same shared `itemlist.SortBy` engine used by Windows: Name, Type, Size and Modified are available on both panes, with Permissions additionally available for the remote pane; ascending/descending cycles retain directories-first ordering and restore selection by item name.
- Kept filtering and sorting composable and non-destructive. A fresh directory snapshot remains authoritative, visible rows are produced by `sort(filter(snapshot))`, and row-indexed actions continue to target only the visible slice.
- Preserved existing bounded recursive search and conservative directory-comparison modal geometry while adding the Linux sort control to the normal file-pane toolbar.
- Added Linux profile credential-save parity. Password/private-key passphrase persistence remains explicit, requires a bounded second confirmation, clears plaintext UI state after the operation and continues to use the protected local profile store and hardened AskPass/provenance boundary.
- Added Linux settings and sorting regression coverage so appearance persistence, sort order, remote permission ordering, unknown-metadata placement, filter composition and selection restoration cannot silently regress.

### Android quality and stability

- Hardened the native Android FTP/FTPS parser with bounded control-line length, multiline reply count/size, MLSD line length and directory-entry count so a malformed or hostile server cannot grow response memory without a maintained limit.
- Preserved strict explicit FTPS certificate and hostname verification, staged upload/download final-name commit gates, non-blocking transfer cancellation, SAF-only local storage access, non-secret saved-site metadata and account-bound remote navigation state.
- Preserved the Android 35 system-bar inset correction and semantic accessibility-only navigation used by authentic emulator evidence; screenshot/color/coordinate navigation fallbacks remain prohibited.
- Android remains an active, installable development APK surface tied to root `VERSION`; it is not silently promoted into the signed Windows/Linux public release allow-list.

### Queue priority and reordering

- Completed four-way queued-transfer priority control with **Top**, **Up**, **Down** and **Bottom** actions through the shared transfer manager and Engine API.
- Reordering rotates only queued scheduler slots, preserving running/terminal history positions, transfer IDs, connection bindings and the relative order of unaffected queued jobs.
- Kept edge moves idempotent and event-free, while a real move emits one complete state snapshot without starting, retrying or cancelling transfer work as a side effect.
- Added Windows and Linux controls with the same queued-only policy, ID-based selection restoration and local copy for all 24 supported desktop languages.
- Added regression coverage for four-way ordering, non-queued slot stability, connection binding, tree-transfer directory-preparation ordering and cross-platform UI wiring.

### Navigation bookmarks and profile start directories

- Added reusable local and remote bookmarks through a shared non-secret persistence model; remote bookmark identity contains only protocol, host, port and username and is captured from the real active Engine connection.
- Made bookmark activation authoritative only after fresh navigation: local targets must list successfully, while remote targets must match the active account and retain the same connection identity across the server listing.
- Added a native Windows bookmark manager and Linux X11 bookmark overlay with Open, Add local, Add remote, Delete and Close behavior wired through the shared Engine contract.
- Added an additional Windows `connectionGeneration` guard around remote bookmark UI commits and Linux modal/action routing that keeps stored paths from bypassing the existing navigation lifecycle.
- Hardened profile start-directory behavior so an inherited server path cannot silently cross a protocol/host/port/username account boundary; Linux also restores the previous verified local base and requires a successful local listing before a selected profile start becomes pane state.
- Added Go coverage plus `scripts/test_navigation_bookmarks_contract.py` for non-secret persistence, corrupt-state fail-closed behavior, account/session revalidation, profile-start isolation, cross-platform UI wiring and documentation/release boundaries.
- Added `docs/NAVIGATION-BOOKMARKS.md` and incorporated the completed feature into the 0.0.4 public desktop release line.

### Release engineering and verification

- Advanced root `VERSION` to **0.0.4** while preserving the current GitHub Release channel (`prerelease=false`) and canonical `ghostftp-vX.Y.Z` tag namespace.
- Kept the public release shape at **14 platform artifacts / 17 public files**: two universal Windows executables, twelve canonical Linux distro/Portable artifacts and three metadata/verification files.
- Preserved optional fail-closed production Authenticode: configured trusted signing identities must verify, while an absent production certificate is recorded explicitly as unsigned rather than replaced by a self-signed identity.
- Kept Android outside the Windows/Linux public release asset allow-list until a maintained production signing/publication contract exists.
- Preserved exact-head CI, Android APK, Linux distro install, package parity and authentic cross-platform UI evidence as independent release-readiness gates.

### Security and privacy

- Preserved strict FTPS certificate/hostname validation, SFTP host-key verification/pinning, trusted Linux AskPass executable/parent provenance, account-bound stored credentials, local root/path protections, protected-secret lifetime rules and no silent secure-to-plain downgrade.
- Preserved the no-telemetry, no-analytics, no-advertising, no-fingerprinting, no-hidden-backend and no-automatic-crash-upload contract across Windows, Linux and Android source surfaces.

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

- Preserved strict FTPS certificate/hostname verification, SFTP host-key verification/pinning, trusted Linux AskPass provenance, rooted local path protections, protected-secret lifetime rules and no silent secure-to-plain downgrade.
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
- Added localized recursive-search Search/Navigate/Cancel/Close/progress/disclosure copy for all 24 supported desktop languages and a source regression contract covering hard bounds, platform wiring and fresh-list navigation behavior.
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
