# Ghost FTP testing and quality gates

Ghost FTP **0.0.4** is validated through layered source, security, native build, packaging, Android APK, UI-action, authentic runtime evidence and release-lifecycle gates.

## Core quality gate

The canonical Core gate requires:

```text
gofmt
go test -race ./...
go vet ./...
```

It also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the complete Python regression suite.

## Protocol and transfer regressions

Tests cover the maintained FTP/FTPS/SFTP desktop engine contract, including:

- explicit FTPS verification and no silent downgrade;
- strict SFTP host-key verification/pinning;
- rooted local path and transfer confinement;
- transfer staging/activation/rollback behavior;
- connection-generation guards;
- privacy-safe diagnostics;
- bounded automatic retry policy;
- queue pause/resume/cancel/retry/clear and queued Top/Up/Down/Bottom reordering lifecycle;
- Remote Edit text/binary, size, revision/conflict, permission, read-back and metadata-refresh behavior;
- navigation bookmark/profile-start account and session revalidation.

## Bandwidth regression contract

The maintained 0.0.4 source provides independent upload/download ceilings as shared runtime policy rather than UI-only state.

Go tests and settings/UI regression contracts require:

- `uploadLimitKiBPerSecond` and `downloadLimitKiBPerSecond` to remain independently persisted and validated;
- explicit values to remain in the bounded range **0–1,048,576 KiB/s**, with **0 = unlimited**;
- missing bandwidth fields from 0.0.2 and older settings to remain migration-safe as unlimited;
- corrupt negative or above-maximum persisted values to normalize safely rather than becoming unintended throttles;
- the transfer scheduler to derive a conservative aggregate directional budget across configured worker slots instead of granting the complete configured ceiling to every concurrent transfer;
- idle slots not to lend temporary burst allowance to a running transfer under the maintained stable-ceiling policy;
- each transfer attempt to snapshot its effective budget when the attempt starts, so a settings save cannot mutate/corrupt an already-running transport process;
- retries/new attempts to sample the currently saved bandwidth policy;
- FTP/FTPS transport enforcement through curl `limit-rate`;
- SFTP enforcement through OpenSSH `sftp -l`, with KiB/s converted conservatively to Kbit/s so flooring cannot exceed the scheduler budget;
- no application busy-wait loop or UI timer to be accepted as transport throttling;
- Windows and Linux settings surfaces to expose the same shared upload/download values with explicit KiB/s units and `0 = unlimited` semantics.

The relevant Go suites cover settings migration/validation, aggregate allocation and transport conversion. Windows/Linux source regression coverage protects the settings-dialog/overlay wiring.

## Current-folder filter and sorting regression contract

The 0.0.4 source includes a non-destructive current-folder filter for the local and server panes. It is deliberately separate from bounded recursive search: filtering only evaluates the already-loaded snapshot and performs no additional directory or network I/O.

The filter gates require:

- shared `internal/itemlist.Filter` behavior with Unicode-aware case-insensitive substring matching and whitespace-delimited AND tokens;
- an empty query to return an independent copy of the full source snapshot;
- filtering and subsequent sorting to leave the authoritative source slice untouched;
- no filesystem, network, engine-listing or recursive-scan call from the shared filter path;
- Windows to keep separate authoritative and visible snapshots, with server snapshots bound to the active connection generation;
- Linux to keep separate authoritative and visible snapshots and discard stale server source data on disconnect;
- rename/delete/upload/download row indices to resolve only against the currently visible filtered slice;
- Windows and Linux filter controls to be rendered and wired, with empty input clearing the filter;
- localized filter copy for all 24 supported desktop languages.

Linux 0.0.4 additionally has source/Go contracts for the shared `internal/itemlist.SortBy` path. Tests require Name, Type, Size and Modified ordering on both panes, remote Permissions ordering, ascending/descending cycles, directories-first behavior, unknown-metadata placement, selection restoration by visible item name, filter+sort composition and non-overlap with the established recursive-search control geometry. Windows uses the same shared sorter through its native list lifecycle.

`scripts/test_file_filter_ui_contract.py` protects the cross-platform source wiring while Go unit tests protect shared matching/copy/sort semantics. Native production builds remain the compile/runtime gate for each frontend.

## Bounded recursive search regression contract

The maintained 0.0.4 source exposes recursive local/server search as an explicit action rather than an extension of typing into the current-folder filter. The search path is read-only and bounded before it reaches either desktop UI.

Core tests and `scripts/test_recursive_search_ui_contract.py` require:

- `internal/filesearch.Walk` to enforce validated depth, visited-item, result-count, batch-size and timeout limits;
- defaults of depth **12**, **20,000** visited items, **1,000** results, batches of **50** and **20 seconds**, with independent hard ceilings of depth **32**, **50,000** items, **5,000** results, batches of **200** and **60 seconds**;
- Unicode-aware matching to reuse `itemlist.MatchesName` so current-folder and recursive modes do not drift to different case semantics;
- cancellation checks before additional listing/item work and cancellation propagation through the underlying list context;
- local scanning to remain anchored to one `os.OpenRoot` capability and never intentionally traverse symlink/reparse entries;
- remote scanning to own one `remote.Operation(ctx)` for the complete scan, so reconnect cannot move an in-flight search onto a different session;
- remote child names and joined paths to pass the existing validation boundary before traversal;
- incremental result batches rather than a single UI-blocking result accumulation;
- all 24 desktop languages to provide Search, Navigate, Cancel, Close, progress/completion and local/server I/O disclosure copy;
- Windows results to live in dedicated search-result ListViews rather than the authoritative file ListViews;
- Linux to keep normal row-indexed actions modal/disabled while recursive result snapshots are displayed;
- Cancel to retain partial informational results until the user closes the search view;
- activating a search result to clear the target pane's current-folder filter, perform a **fresh parent listing**, and reselect the discovered name only from that authoritative listing;
- no search result object to carry delete, rename, upload or download authority.

The desktop disclosure explicitly states that recursive mode reads nested local/server folders and states the default **20-second / 1,000-result** bound before the scan starts.

## Directory comparison and synchronized-navigation regression contract

The maintained 0.0.4 source exposes directory comparison as a read-only view over freshly listed current local/server directories. The shared classifier performs no filesystem or network I/O and does not grant file-operation authority to comparison rows.

Go tests and `scripts/test_directory_comparison_contract.py` require:

- exact-name pairing rather than automatic case folding across filesystems with potentially different case semantics;
- deterministic `same`, `local_only`, `remote_only`, `newer_local`, `newer_remote`, `conflict` and `unknown` states;
- duplicate exact names to fail closed to `conflict`;
- type mismatches to fail closed to `conflict` and symlinks to fail closed to `unknown`;
- a default **2-second** timestamp tolerance with a hard **5-minute** maximum override;
- directional timestamp comparison that cannot overflow through negation of a saturated minimum duration;
- regular-file `same` or `newer_*` classification only when both sides provide usable modification times; zero-size ambiguity and equal-size files with unknown timestamp remain `unknown`, while unequal sizes with unknown time remain `conflict`;
- the synchronized-directory resolver to accept only an exact `same` entry that is an ordinary non-symlink directory present on both sides;
- Windows to render dedicated comparison ListViews, synchronize comparison-row selection, invalidate stale comparison state on connection-generation changes and disable ordinary rename/delete/upload/download/Remote Edit/CHMOD authority while comparison is active;
- Linux to preserve authoritative pre-filter pane snapshots, restore them after cancellation/listing failure, make comparison modal and consume ordinary workspace clicks while comparison display rows are active;
- local synchronized children to pass `security.SafeLocalChild`, and remote child names/paths to pass the existing remote validation boundary;
- **fresh local and remote listings to complete before either pane path is committed** during “Open both”;
- comparison to recompute from those fresh target snapshots after synchronized navigation;
- all 24 desktop languages to provide comparison action/status/disclosure copy;
- comparison code to contain no upload, download, delete, rename or CHMOD side effect.

This contract deliberately does not infer that equal size means equal content when server timestamps or size presence are unavailable. Those cases stay conservative rather than fabricating equality or freshness.

## Settings regression contract

The settings suite verifies that visible runtime options remain bounded and migration-safe.

Examples include:

- parallelism accepts only **1–8** for explicit current values;
- an omitted legacy `parallelism=0` migrates to the canonical default **2**;
- independent upload/download bandwidth values accept only the maintained bounded range and preserve `0 = unlimited`;
- connection timeout, retry count and retry delay stay inside documented bounds;
- unknown persisted conflict-policy state fails closed to conservative recovery behavior;
- one canonical conflict-policy field synchronizes legacy compatibility fields;
- Windows and Linux accept only canonical `light`/`dark` appearance state and preserve Classic Light as the fallback;
- Linux persisted appearance is applied before initial rendering and saving Settings applies the selected shared palette;
- Linux profile password/private-key-passphrase persistence requires the bounded confirmation path and does not weaken AskPass provenance.

This prevents compatibility migration from becoming an excuse to silently accept arbitrary invalid settings.

## Desktop action wiring gate

`scripts/test_ui_action_wiring.py` prevents visible main controls from becoming decorative/dead UI.

Windows checks:

- enumerate main-window `mkButton(...)` command IDs from `createControls`;
- require every such ID to have a `case id...:` handler in `command()`;
- keep queue Cancel/Retry engine failures from being silently discarded.

Linux checks:

- require every maintained main control rectangle (connect/disconnect/profile/settings/navigation/file operations/upload/download/queue actions) to exist in layout construction;
- require the same control to have a mouse click handler;
- verify the Remote Edit button path;
- verify SFTP Trust/Cancel and prompt Apply/Cancel overlay controls are both rendered and handled.

The dedicated current-folder-filter/sort, recursive-search and directory-comparison contracts supplement this generic action-wiring gate because those controls are dynamically inserted into pane/workspace geometry instead of being canonical top-level buttons.

The gate intentionally tests wiring, not just pixels. Runtime behavior remains covered by Go unit/integration tests and authentic UI smoke evidence.

## Android native source and APK gate

The Android source line is independently fail-closed and remains a development APK surface rather than an implicit addition to the public Windows/Linux release asset set.

`.github/workflows/android-apk.yml` requires:

- Android source/security contract tests;
- Java 17 / Android SDK 35 / maintained Gradle toolchain setup;
- Android lint;
- installable APK build;
- APK identity/packaging verification;
- artifact upload as `ghostftp-android-apk`.

Android source contracts protect strict explicit FTPS certificate/hostname verification, no trust-all fallback, SAF-only local storage, non-secret saved-site metadata, staged upload/download final-name commit gates, non-blocking transfer cancellation, semantic navigation accessibility and bounded FTP control/MLSD parsing. SFTP remains hidden until strict Android host-key identity verification exists.

## README/media integrity

README and active documentation use repository-local Ghost FTP icon/screenshot assets. Remote tracking pixels, icon CDNs and mockup images are not accepted as release UI evidence.

## Windows production gate

The Windows production job builds and verifies the public artifacts:

```text
Ghost-FTP-0.0.4-Setup.exe
Ghost-FTP-0.0.4-Portable.exe
```

The builder first produces and verifies native x64/x86 Setup and Portable payloads internally. It then constructs the two public universal bootstraps, selects native architecture from Windows system information, verifies staged embedded bytes and rejects architecture-specific public EXEs. The CI also exercises the Authenticode private-key pipeline policy. Production signing is optional; configured signatures must verify.

## Linux production gate

The regular Core CI continues to build generic DEB/portable compatibility artifacts for `amd64`, `arm64` and `i386` through `linux/BUILD.sh` and compares their executable bytes. This remains useful independent build coverage but is not the canonical 0.0.4 release allow-list.

## Canonical distro package gate

`.github/workflows/linux-distro-packages.yml` builds and verifies the canonical distro artifacts through:

```text
linux/BUILD-DISTROS.sh
```

The 0.0.4 release set contains Debian and Ubuntu DEBs for `amd64`, `arm64`, `i386`; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and distro-neutral Portable tarballs for `amd64`, `arm64`, `i386`. Package metadata and byte-for-byte executable parity across matching variants are fail-closed release requirements. These files are canonical members of the **14 platform artifacts / 17 public files** release allow-list.

## Native distro lifecycle gate

`.github/workflows/linux-distro-install.yml` verifies native install/remove/runtime/GUI behavior on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

**Native package-manager/runtime coverage is deliberately limited to x86-64.** Additional arm64/aarch64 and i386/i686 artifacts still receive exact-head build, metadata, extraction and binary-parity verification.

## Authentic UI evidence

`.github/workflows/ui-screenshots.yml` captures real exact-head runtime UI on all maintained source platforms used by the evidence contract:

- Windows: Main Workspace, Site Manager, Bookmarks, Settings and About;
- Linux: Main Workspace, Bookmarks and Settings;
- Android: Files, Navigation, Sites, Bookmarks, Transfers, Settings and About.

The final read-only evidence job checks out the exact source SHA, downloads the three platform runtime artifacts, verifies provenance/manifest/file hashes and assembles the `ghostftp-authentic-ui-verified-bundle`. The maintained bundle contains **15 runtime images**. It never commits or pushes evidence back to the tested branch. Mockups, screenshot-color navigation and generated approximations are not accepted as authentic runtime evidence.

A release-prep change affecting `VERSION` or maintained UI must obtain authentic evidence from the exact final source revision where the screenshot workflow is triggered.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every required workflow triggered for its exact final head is `completed/success`. After merge, required `push` workflows are identified by the exact merge SHA and must also finish `completed/success` before release preparation continues.

For a 0.0.4 release-prep change that affects the canonical production build, the expected gates are:

1. Ghost FTP CI;
2. Ghost FTP Android APK;
3. Ghost FTP Linux Distro Packages;
4. Ghost FTP Linux Distro Install Matrix;
5. Ghost FTP Authentic Cross-Platform UI Screenshots.

A green run for an older commit does not satisfy a newer PR head.

## Release publication gate

0.0.4 publication additionally requires:

- exact current `main` release-branch validation;
- canonical release workflow quality/build jobs;
- exact **17-file** GitHub Release allow-list;
- immediate and delayed remote release read-back;
- `prerelease=false` for the current 0.0.x release channel;
- verified `ghcr.io/bren-wp/ghost-ftp:0.0.4` distribution-bundle publication/read-back;
- successful latest-only retention cleanup after publication.

The Android development APK gate is required source validation but does not enlarge the 17-file public Windows/Linux release allow-list.

## Deterministic release-to-retention gate

The release branch trigger does not consider `gh workflow run release.yml` itself a successful release. Its regression/audit contract requires the trigger to:

1. record pre-existing release run IDs;
2. dispatch the canonical release workflow;
3. identify the newly created workflow-dispatch run whose `headSha` equals validated current `main`;
4. wait for that exact run with `gh run watch --exit-status`;
5. require terminal `success`;
6. only then dispatch canonical retention;
7. identify the new exact-main retention run;
8. wait for and require terminal retention `success`.

This closes the class of failure where publication succeeds but downstream `workflow_run` chaining is not emitted for a token-dispatched workflow. The retained `workflow_run` retention trigger remains defense in depth; successful release-branch orchestration requires explicit observed retention completion.

## Retention validation

The retention workflow must leave only the current `ghostftp-v0.0.4` release/tag, retain the current canonical release branch and exact-version GHCR package, remove superseded release branches/package versions, and leave `main` commit history untouched.

Before destructive cleanup it independently verifies current release identity, `draft=false`, `prerelease=false`, exactly **17 assets**, current tag SHA equality with current `main` and the current exact-version package.

## Quality rule for new power-user features

A new feature such as verified resume, multi-session or proxy/jump-host support is not release-ready until all applicable layers exist:

- shared engine/runtime behavior;
- validation and safe defaults;
- Windows and Linux controls or an explicit documented platform limitation;
- localized user-facing copy;
- success/failure/cancel semantics;
- unit/integration/regression tests;
- action-wiring coverage for new controls;
- active documentation;
- authentic screenshot evidence when the maintained UI changes.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Settings](SETTINGS.md), [Roadmap](ROADMAP.md) and [Versioning](VERSIONING.md).
