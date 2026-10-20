## 0.95.53 — Audited security, integrity and latest-ten release retention (UNRELEASED Preview) — 20 October 2026

- Audit cross-platform source and release workflows for dangerous TLS/SSH bypass patterns, credentials, unsafe file replacement, recovery lineage, stale code, version drift and installer metadata. Static checks are not a substitute for penetration tests.
- README Windows screenshots updated to seven actual v0.95.52 running-application captures from exact release workflow SHA and SHA-256 manifest, rather than older v0.95.51 or v0.95.18 pixels.
- Verify original v0.95.51 release inventory and original tag before correcting misclassified Preview metadata; label retained unsigned/ad-hoc Preview releases accurately.
- Re-execute owner-authorized deletion of historical GitHub releases and matching tags only after the new release is fully published, with safeguards for ten retained editions and screenshot provenance.
- Strengthen macOS native FTP filesystem/data consistency and cancellation safety; native FTPS/SFTP remains a separately gated implementation until actual certified handshake and live E2E are complete.

## 0.95.52 — Recovery, original QA screenshots and data integrity (UNRELEASED Preview) — 20 October 2026

- Correct fail-closed draft release staging by immutable release ID, not a draft tag lookup that can return HTTP 404.
- Recover the incomplete public v0.95.51 as a verified Preview, preserving its exact source tag.
- Restore genuine Windows QA screenshot provenance and prepare controlled ten-release retention after verifying all required dependencies.
- Improve macOS FTP upload source consistency checks; FTPS and SFTP are still not supported in native macOS and cannot be claimed complete.
- Preserve existing profiles, database migration paths, credentials and public Brendigo branding.

## 0.95.51 — Automatic gated Preview promotion and integrity (UNRELEASED Preview) — 20 October 2026

- Automatically publish a new Preview prerelease after each versioned merge to main, including Android-only/macOS-only/documentation changes, only when all exact-SHA CI gates pass. Reject PRs without version bumps or when the preceding main version has not been published.
- Harden SSH `known_hosts` trust-record replacement to reject symlink/hardlink aliases and stale inode snapshots; preserve existing records on failure.
- Stage GitHub Releases as drafts and verify upload SHA-256, all remote digests, asset inventory and source SHA before public Preview status.
- Preserve public releases and published tags. Document the v0.95.50 metadata mismatch with Preview/unsigned assets.
- Stable release remains blocked by platform signing, notarization and real-device acceptance.

## 0.95.50 — Windows/Linux filesystem and protocol security fixes (published with Preview-grade assets) — 20 October 2026

- Correct SQLite snapshot preservation, encrypted backup verification and recovery copies, Linux PATH and SSH known_hosts no-follow safeguards.
- Guard Mirror sync roots and downloads from symlink/root confusion, Windows device filenames and traversal.
- Bound WebDAV XML and cloud JSON responses, restrict OAuth bearer requests to trusted origins, prevent cyclic Drive/OneDrive pagination.
- Expand isolated FTP/FTPS/SFTP and Windows filesystem regression checks; final protocol security, restart/recovery, vendor cloud accounts and branch-protection acceptance remain unverified.
- Keep all historical tags, published v0.95.49 and user data intact. This release contains **Preview-grade artifacts**, and restore is not globally atomic.

## 0.95.49 — Windows/Linux credential vault migration and recovery (unreleased security preview) — 19 October 2026

- Preserve published v0.95.48 and Windows Credential Manager service/account identifiers; reject silent partial vault deletes and fail-closed API-key status when the vault is locked.
- Gradual Linux keyutils → D-Bus Secret Service migration using private stdin and verified read-back; do not delete a previous secret before verification. New writes require available persistent Secret Service.
- Shared vault API for profile passwords, cloud OAuth tokens and encrypted backup/restore; manifest-first secret records and checked cleanup, with fault-injection regression cases.
- Preserve existing backup version-1 Argon2id/AES-256-GCM format and migration compatibility; improve staged restore validation and keychain rollback behavior.
- Live Windows and Linux secret vault/locked wallet, concurrent processes, crash injection and recovery acceptance remain REQUIRED before merge/release.

## 0.95.48 — Windows/Linux cloud connection wizard and defensive HTTP integration (development preview) — 19 October 2026

- Extend Sites → Connect Cloud with guided S3/S3-compatible, Backblaze B2, Cloudflare R2, Google Cloud Storage, WebDAV and Nextcloud credential forms, a GCS JSON file picker, automatic B2/Nextcloud endpoint assembly and read-only connection probes.
- Show MEGA and Proton Drive as official external tooling only, clearly marking their natively unsupported state; Proton SDK is not ready for third-party commercial production use.
- In Sites, connect a Google Drive, OneDrive, Dropbox or Box account using browser-based OAuth2 PKCE rather than providing account passwords to Ghost FTP.
- Detect whether the application build includes public provider OAuth Client IDs; show safe manual public Client ID setup where required. Keep tokens and per-account IDs in the OS credential store for post-restart refresh, with backward-compatible token JSON.
- Guard against failed profile-save keychain orphans, preserve existing saved accounts, and bound Box listing pagination.
- Redact provider HTTP response bodies and sensitive URL paths from Box, Google Drive and Shopify diagnostics; preserve existing connections and user data.
- Provider-console registrations, real cloud consent and restart/refresh tests remain unverified; this is a Preview and is not a claim of full end-to-end account acceptance.

## 0.95.47 — Repository-wide cross-platform source reachability and dead-code audit (development preview) — 19 October 2026

- Add conservative advisory source inventory for Windows/Linux Rust/React, Android Kotlin and resource files, macOS Swift, Go runtime/installer and operational scripts, with JSON evidence in CI.
- Add regression tests for symbol matching, false-positive candidate classification, OS entrypoint protection and symlink-safe scan traversal.
- Preserve all installed-release/upgrade, credential, migration, signed-bundle, native platform and dynamic runtime entrypoints. No unproven dead-code removal or full-device coverage claim.
- Continue Android Lint/R8, SwiftPM, Rust Clippy, TypeScript and Go regression gates; verify exact SHA before merge or release.

## 0.95.46 — Windows/Linux transfer reliability and bounded cloud uploads (development preview) — 19 October 2026

- Remove stale asynchronous concurrency shrink races when changing transfer parallelism between 1/4/8/16/32 active slots.
- Reject truncated S3-compatible multipart parts; abort multipart sessions on failure and verify exact source sizes instead of completing incomplete objects.
- Stream Box uploads without buffering whole multi-GiB files; limit legacy Google Drive buffered uploads to 32 MiB until resumable large-file upload receives real account verification.
- Verify OneDrive chunk-source lengths fail closed; retain existing TLS, SSH host-key and user-data security policy.
- Regression coverage for source mutation, >4 GiB offset arithmetic, cloud buffering limits and dynamic concurrency; provider-account, throughput, CPU/RSS and GUI responsiveness results remain unverified.
- Advance canonical preview source to 0.95.46 after the published immutable v0.95.45. No release or merge is claimed without exact final SHA CI success.

## 0.95.45 — Windows/Linux crash and encrypted-backup recovery hardening (development preview) — 19 October 2026

- Restore the canonical release train after the immutable published `v0.95.44` tag: version 0.95.45 follows published 0.95.44 without moving or deleting tags.
- Hardening on Windows/Linux desktop: atomic private backup writes; SQLite integrity validation before OS credential mutation; fail-closed staged restore and recovery of original database/WAL/SHM.
- Protect authentication information in Rust debug output, prevent duplicate editor IPC launches, and stop keyring-clear failures from being reported as success.
- Preserve existing profiles, migrations and Brendigo branding. Shared Android version metadata is synchronized for repository CI; Android and macOS feature implementations are unchanged.
- **Development preview only.** Claims of crash resolution or release readiness require real user-machine verification and all six CI workflows on a single exact final commit.

## 0.95.44 — Transfer Center safe selector extraction (development preview) — 19 October 2026

- Decouple Transfer Center status counts, tab/search/direction/time selectors and immutable recent-history projection from the 1,117-line React component into a pure reusable view-model.
- Keep actual backend queue/IPC actions untouched; the extracted logic retains references to source transfers, preserves original row order and does not change confirmed transfer state.
- Add negative/boundary Node tests for failed/retry selection, status counters, hour/day boundaries, immutability and 10,000 synthetic transfers. CI runs them in the desktop UI contract suite.
- Advance shared source metadata to 0.95.44 Preview following immutable public v0.95.43; no production signing, UI study, external E2E or new release is claimed.

## 0.95.43 — Fail-closed release readiness and Preview classification (development preview) — 18 October 2026

- Block any stable-channel release until independently verified Windows Authenticode, Android Play App Signing/upgrade and macOS Developer ID/notarization are wired into release gates.
- Classify future Preview GitHub releases as **prerelease**, not stable/latest. Use the latest public nondraft publication, including prereleases, for immutable release lineage.
- Add negative release-channel regression tests to quality CI; retain all-or-nothing desktop updater signature policy.
- Preserve approved brand/platform code and user data; 75-screen visual acceptance and signed physical installer lifecycle are still unverified. No release is created by this PR.

## 0.95.42 — Post-publication signing source alignment (development preview) — 18 October 2026

- Harden the desktop OAuth PKCE login with a per-login CSRF `state`, strict callback validation, duplicate-parameter rejection and token diagnostic redaction.
- Refuse provider authorization overrides of security-critical OAuth fields; negative tests cover missing/mismatching/duplicate state and callback errors.
- Preserve the single, protected Android Play-upload signing workflow already merged via PR #189, including six-family exact-SHA verification and fail-closed key handling.
- Align source/build version 0.95.42, Android versionCode 954200, Cargo and platform metadata after immutable v0.95.41 publication.
- No production keystore was provided; Play AAB acceptance, previous-version upgrade and secure macOS FTPS/SFTP remain unverified.
- This is a development Preview and does not open or modify any public release.

## 0.95.41 — Bounded fail-closed HTTP autoindex inventories (development preview) — 18 October 2026

- Stop accepting HTML login/error/partial pages as successful empty HTTP directory listings. Enforce recognizable complete directory-index markup.
- Bound the HTTP response while receiving chunks before allocating/parsing the full listing. Reject oversized JSON/HTML source inventories and large aggregate filename sets.
- Refuse duplicate names, malformed URL/Unicode names, nested traversal hrefs, invalid JSON entry types and truncated/multi-anchor HTML rows instead of silently omitting files.
- Add negative tests for malformed/index-shaped HTML, duplicate records, oversized listings and invalid JSON names, preserving valid nginx and Apache listing fixtures.
- Keep published v0.95.40 immutable and treat this as a preview PR candidate pending live HTTP compatibility and all six exact-SHA CI gates.

## 0.95.40 — OS vault backup/restore fail-closed security hardening (development preview) — 18 October 2026

- Abort encrypted backup export when the OS credential vault is unreadable or a manifest credential is missing instead of silently creating an incomplete backup.
- Reject legacy/plaintext profile auth values embedded in restored backup metadata; do not reintroduce secrets to profiles.json.
- Verify and restore OS credential records before staging backup configuration and database snapshots; fail on locked vault or write errors, with best-effort rollback of partially changed vault entries.
- Add fault-injection regression coverage for locked credential stores, partial restoration, invalid profile metadata and preservation of existing local profiles.
- Retain the v0.95.39 published tag and existing encrypted archive format. Full crash-atomic restore across multiple files, live vault tests and production release acceptance remain outstanding.

## 0.95.39 — Fail-closed WebDAV/HTTP directory inventory (development preview) — 18 October 2026

- Reject incomplete or malformed WebDAV XML and non-207 PROPFIND listings rather than returning partial or empty directory inventories that might enable unsafe synchronization.
- Reject malformed nginx JSON autoindex responses rather than treating them as empty folders; preserve legitimate empty JSON arrays.
- Add targeted regression coverage for truncated XML/JSON and update preview/release metadata without retargeting published v0.95.38.

## 0.95.38 — Atomic SQLite sync inventories and fail-closed folder scans (development preview) — 18 October 2026

- Reject unreadable or partial directory scans when planning synchronization; a missing/unreadable source subdirectory can no longer be silently interpreted as files that should be removed from the server.
- Refuse to plan from a fabricated empty SQLite sync history when the database is unavailable or corrupted.
- Replace an entire source inventory for one pair in a single SQLite transaction, with prepared INSERT statements and rollback on write failure. Refuse entries larger than SQLite signed 64-bit integer limits.
- Skip source index replacement and preserve prior last-synced timestamps when Mirror deletion guards withhold destructive work.
- Add Rust regression coverage for strict scans, per-pair isolation, malformed inventory size and transactional rollback.
- Retain existing Windows native OS credential storage, Linux Secret Service, live folder watching, CMD/.NET automation and built-in PHP/JS/HTML/CSS editor; they remain subject to independent native acceptance and throughput benchmarks.

## 0.95.37 — SQLite watched-sync settings and integrated remote source editor (development preview) — 18 October 2026

- Store continuous folder-sync configuration transactionally in the existing local `ghostftp.db` (SQLite); migrate legacy `foldersync.json` once without deleting it, and refuse to silently discard malformed configs.
- Add an in-app remote UTF-8 source editor (context menu → Edit code in Ghost FTP), including PHP, JavaScript, HTML and CSS token highlighting, search, automatic bracket pairs, indentation and Ctrl+S.
- Restrict editor file IPC to Rust-issued active edit session IDs, refuse binary/non-UTF-8 files and files over 2 MiB; remote save completion still requires a successful watcher upload event.
- Keep saved passwords/key passphrases in the native OS credential store; SQLite is for configuration metadata, **not** an encryption replacement.
- Add migration corruption and editor token-preservation regressions, with automated Windows/Linux build and protocol acceptance gates.

## 0.95.36 — Saved credential safety, reliable live sync and script automation (development preview) — 18 October 2026

- Enforce fail-closed profile serialization at the storage layer: new plaintext passwords and private-key passphrases cannot reach `profiles.json`. Existing keyring use remains unchanged (Windows-native and Linux-native).
- Fix a continuous-sync integrity bug: any failed/canceled/missing/timed-out transfer stops the sync from claiming success or advancing its source index; a later poll can retry.
- Ignore Ghost FTP's own incomplete download and delta staging files during continuous sync; notify watcher errors trigger an immediate rescan.
- Harden remote edit temp directory permissions on Linux; reject symlink/special-node auto-uploads; coalesce editor events using bounded memory, with a 10-second failed-save retry backstop.
- Add credential-free CMD, PowerShell and .NET FileSystemWatcher script generation in Sync & Backup; exclude-pattern, Mirror and on-demand pairs fail closed instead of silently changing semantics.
- Cover sync success/failure, profile plaintext refusal, staging exclusion and script injection behavior in automated regressions.

## 0.95.35 — Windows/Linux transfer safety, crash-resistant backups, cross-pane UI and audit (development preview) — 18 October 2026

- Use link-aware local destination and upload-source checks; refuse symlink/special-node transfers and unsafe Unix backslash basenames rather than silently following or renaming them.
- Preserve existing staged-download backups and use Windows/Linux atomic no-replace moves for staging, backups and restoration. Never silently overwrite a previously saved backup.
- Inventory directory uploads fully before creating remote directories; reject special entries, non-UTF-8/unsafe names, excessive nesting, entry counts and metadata size.
- Validate cross-pane drag/drop payload structure and originating session ID, preventing stale-source transfers when remote connections change.
- Add Windows/Linux automated regression coverage for occupied backup files, directories, staging links, dangling links, rename collisions and inventory boundaries.
- Document an evidence-based desktop QA/dead-code audit and explicit FileZilla feature gaps without claiming all 75 screen references are accepted.

## 0.95.34 — FTP directory inventory limits and safer SSH host verification (development preview) — 18 October 2026

- Bound FTP/FTPS LIST inventories to 100,000 rows, 32 MiB raw response bytes and 16 MiB aggregate filename bytes, refusing oversized server listings before parsing.
- FTP overwrite/existence decisions strictly validate the **whole** server inventory; later malformed or duplicate rows cannot be silently ignored after an earlier matching filename.
- Deduplicate SSH host-key prompt IDs, serialize decisions and dismiss only the completed request so double-clicks or slow IPC replies cannot answer the wrong host.
- Changed SSH host fingerprints now require explicit acknowledgement of verification with the server operator over a trusted channel.
- Preserve seven real historical Windows application screenshots and the published v0.95.33 predecessor. Manual pixel parity and universal FileZilla superiority are not claimed.

## 0.95.33 — Windows/Linux protocol security, privacy and transfer UI polish (unreleased preview) — 18 October 2026

- Fail closed on malformed, duplicate, oversized or ambiguous SFTP directory-listing names (100,000 entries / 16 MiB UTF-8 name bytes).
- Refuse implicit symbolic-link and unsupported-node transfers; guard duplicate basenames in one transfer batch when confirmation prompts are disabled.
- Redact diagnostics consistently in the shared File Manager, transfer failure notifications and Transfer Center.
- Add accessible transfer progress semantics and consistent transfer-history filtering.
- Correct the main README to show seven authentic Windows screenshots with the original captured-source/version provenance.
- Preserve published v0.95.32; CI and real-device review determine whether any new candidate may be released.

## 0.95.32 — SFTP recursive deletion safety (unreleased development preview) — 17 October 2026

- Use SFTP LSTAT to unlink directory symlinks without traversing their targets.
- Inventory SFTP recursive deletions before removing anything, with depth/entry/path-byte limits and type-recheck safety.
- Add Rust resource-limit regressions and real OpenSSH SFTP symlink-target preservation coverage.
- Previously published v0.95.31 is immutable; exact-head CI and manual Windows/Linux acceptance still required.

## 0.95.31 — bounded Windows/Linux local directory inventory (unreleased development preview) — 17 October 2026

- Prevent runaway memory growth and incomplete sync/conflict inventories: fail closed for more than 100,000 local entries or 16 MiB cumulative UTF-8 filename bytes.
- Regression tests cover the exact resource limits and international filename enumeration.
- Preserve Ghost FTP premium UI and published v0.95.30; candidate builds/CI pending.

## 0.95.30 — Windows/Linux sync safety, reliability and performance (unreleased Preview) — 17 October 2026

- Windows/Linux: coalesce filesystem watcher storms into a one-slot notification queue; repeated events no longer build an unbounded pending list.
- Windows/Linux: replace recursive, potentially exponentially costly exclude/glob matching with a linear-memory wildcard state machine while retaining existing '*', '**' and '?' semantics.
- Windows/Linux: bound stale delta-temp cleanup to 100,000 entries and 64 levels, skip symlinked folders/files and special nodes rather than following linked paths outside the sync root.
- Limit continuous watcher debounce to five seconds to prevent starvation; Mirror roots reject dot/parent traversal in remote/local paths before any sync plan.
- Add Rust tests for 50,000 coalesced notifications, adversarial globs, Unicode file names, Mirror root traversal and Unix symlink escapes.
- Canonical development metadata advances to v0.95.30 Preview; published v0.95.29 remains immutable. Exact-SHA CI and manual Windows/Linux acceptance remain pending.

## 0.95.29 — Windows/Linux atomic file actions, connection races and transfer UI performance (unreleased development preview) — 17 October 2026

- Windows/Linux: implement atomic local rename without replacing existing files, directories or symlinks. Linux uses renameat2(RENAME_NOREPLACE); Windows uses MoveFileW without replace flags. Protect the original source and destination on collision.
- Linux: replace path-based chmod preflight/set_permissions with O_NOFOLLOW opening, fd metadata validation and fchmod on the same opened inode, removing final-component symlink-swap races.
- Connection management: last user tab/connect intent wins over slower handshakes; stale connection health replies and profile reload snapshots cannot overwrite newer information; Agent Bridge focus updates are serialized and coalesced.
- Transfer rendering: maintain an exact active/queued/paused count in the Zustand transfer store rather than O(n) list scans on every progress event in both file-browser layouts.
- File browsing: reject stale directory responses after rapid navigation, refresh or connection switches; clear obsolete file rows/selections immediately so failed reads cannot expose actions targeting a previous directory.
- Add native Rust regressions for occupied rename destinations and dangling symlinks, and CI source contracts for concurrency and UI stability.
- This is unreleased Preview source pending full exact-SHA CI and manual Windows 10/11 / Linux distribution acceptance. Stable, signed and full FileZilla parity are not claimed.

## 0.95.28 — macOS FTP command argument safety parity (published preview) — 17 October 2026

- macOS native FTP: reject all ASCII C0 control codepoints and DEL in FTP control-channel arguments, including usernames, passwords, navigation and remote file transfer names; previously only CR, LF and NUL were refused.
- Preserve Unicode remote filenames and printable spaces without lossy normalization or changes to existing premium SwiftUI design.
- Expand native Swift XCTest regressions for TAB, ESC/ANSI sequences, BS, DEL and additional control bytes.
- Advance shared version metadata to 0.95.28 Preview after v0.95.27; this patch does not implement secure native FTPS/SFTP, signing/notarization or full FileZilla parity.

## 0.95.27 — bounded FTP recursive delete inventory and regression tests (published preview) — 17 October 2026

- Windows/Linux FTP and explicit FTPS: cap recursive delete tree inventory to 25,000 entries, 64 levels and 8 MiB cumulative target-path bytes before executing any DELE/RMD.
- Fail closed on duplicate paths, malformed directory-list entries or incomplete dotfile inventory; retain bottom-up deletion order for valid trees.
- Add Rust tests for nested/hidden files, duplicate or malformed entries, over-deep chains, excessive inventory entries and LIST -a rejection.
- Advance source/build metadata to 0.95.27 Preview while leaving the published 0.95.26 tag and assets immutable. Windows/Linux installer and real third-party FTP acceptance remain subject to exact-head CI.

## 0.95.26 — FTP command-path validation and security regressions (published preview) — 17 October 2026

- Windows/Linux: prevent control-character-bearing FTP paths from reaching LIST, existence probes, RNFR/RNTO, DELE/RMD, MKD or SITE CHMOD command construction.
- FTP: require nonempty command targets and reject CR/LF/NUL/TAB/DEL and other ASCII controls; preserve legitimate Unicode and spaces in filenames.
- QA: add Rust regressions in both the remote filesystem and low-level FTP stream modules for rejected protocol-control path operands and accepted international filenames. Actual third-party daemon interoperability and production acceptance remain pending.
- Development source version advances to 0.95.26 Preview without changing the immutable published v0.95.25 tag; no Android or native macOS feature change is implied.

## 0.95.25 — Windows/Linux filesystem safety and adaptive menus (published preview) — 17 October 2026

- Windows/Linux: fail closed on unreadable local directory entries and non-UTF-8 filenames instead of hiding files or creating lossy action targets; add native filesystem regressions for real files and symlinks.
- Windows/Linux: reject lexically root-equivalent local rename/delete targets; Linux chmod preflight refuses symbolic links, special nodes and root paths (race-free descriptor-level chmod remains outside this patch).
- Windows/Linux premium UI: reposition and constrain context menus using measured dimensions on window resize; serialize async menu actions to prevent rapid double-dispatch.
- CI: update Rust formatting and adaptive context-menu contract; preserve immutable published v0.95.24 and advance preview source metadata.
- Windows/Linux native installer and UI acceptance for this source remain under verification. This is not a production-stable claim.

## 0.95.24 — macOS FTP destination safety and first-party desktop metadata (published baseline) — 17 October 2026

- macOS: prevent unintended overwrite of existing downloaded files; publish private, completed FTP downloads using exclusive atomic creation, and replace regular files only with Save Panel confirmation.
- macOS: reject symbolic-link and special-file destinations; retain a healthy control session if completed downloads encounter local publication conflicts.
- macOS: validate FTP reply codes and add unit/real FTP fixture regressions for collision, confirmation and control-channel synchronization.
- Windows/Linux: Tauri installer homepage points to https://ghostftp.com and is enforced by the frontend metadata contract.
- Status: native macOS explicit FTPS and SFTP, FileZilla-equivalent workflows, Developer ID signing and notarization remain release blockers. This cycle is preview, not production stable.

## 0.95.23 — Android premium 16-screen reference: structural fidelity (preview candidate) — 17 October 2026

- Android: approved violet/navy native visual primitives and full-width phone drawer; searchable real Bookmarks list and safe site navigation from saved metadata; no mock site credentials.
- Android: reference Settings screen #15 now uses a 4-by-2 grid of eight functional categories; actual security/legal/transfer/reset actions are available under Advanced, not removed.
- Android: reference About #16 centers the real Ghost FTP logo and running version, and offers Licenses/resources, official Terms and current GitHub Releases updates.
- Android: Add/Edit Site #4 has functional General/Advanced/Sync tabs, safe existing-profile field prepopulation, and no credential restoration. Existing FTP/FTPS/SFTP validation remains active.
- Tests must confirm the interactive transitions, honest empty states and privacy guards. 16/16 pixel-for-pixel and device acceptance remain PENDING. Release must remain preview until actually proven.

## 0.95.22 — Android 16-screen premium reference implementation (unreleased) — 16 October 2026

- Align native phone UI with the two user-supplied sixteen-screen Android reference images: full-width content, a slide-over site/navigation drawer, a 52dp contextual title bar, an Android-only violet premium action palette, real dual-pane Files layout and a five-action fixed footer.
- Replace the old Sites-first editor with a searchable list of real saved-site metadata and an Add/Edit route, without ever inventing sites or persisting passwords.
- Keep remote FTP/FTPS/SFTP file interactions wired to their existing safe controllers; browse app-owned local downloads rather than displaying fictitious access to system-wide storage.
- Update Android runtime UI contracts and real emulator visual/click-through tests. Remaining detailed screens, physical-device pixel comparison and complete all-six-workflow acceptance are NOT claimed.

## 0.95.21 — macOS FTP interoperability, desktop connect budget and Android workspace polish (unreleased) — 16 October 2026

- macOS: real navigation, create-folder and confirmation-protected file/empty-folder deletion (MKD, DELE, RMD) with live FTP fixture coverage; no recursive-delete shortcut.
- macOS: fall back to strict Unix LIST only after an explicit unsupported MLSD response; refuse unknown, ambiguous or hostile entries rather than returning an incomplete remote inventory.
- macOS: render arbitrary UInt64 remote file sizes without crashing on Int64 overflow.
- Windows/Linux: FTP/FTPS control TCP attempts share a bounded overall connect budget across DNS answers, with loopback regressions.
- Android: show Files-only remote-action toolbar only in the Files workspace, and session status only in Files and Sites; instrumented navigation assertions.
- No claim of Android 75-reference 1:1 pixel acceptance or complete macOS secure FTPS/SFTP feature parity; real device and third-party FTP/SSH matrix remain open.

## 0.95.20 — FileZilla SFTP key-auth migration (unreleased) — 15 October 2026

- Preserve FileZilla SFTP `Keyfile` path only when `Logontype=5`; do not read/copy private key material during import.
- Reject stale Keyfile on password/other protocol profiles as authentication settings. Explain unsupported PuTTY PPK files and missing keys in the existing import preview, avoiding false working-key claims.
- Add Rust regression tests for key login, field isolation, PPK and missing-key warnings. FileZilla XML protections introduced in 0.95.19 remain enabled.
- Not a claim of completed multi-server SFTP, SSH agent, or cross-platform parity.

## 0.95.19 — FileZilla Site Manager folder migration and XML safety (unreleased, CI pending) — 15 October 2026

- Preserve saved FileZilla folder breadcrumbs in Ghost FTP's Sites grouping, without changing the premium layout.
- Import FileZilla exported trailing Server display names; preserve explicit Name priority.
- Reject oversized input, excess sites/folder nesting, XML DTDs, malformed entities and incomplete folder/server data.
- Never import external passwords; retain explicit FTPS policy. Add Rust safety and migration regressions.
- **Not a completed FileZilla parity or cross-platform device acceptance claim.** Issues #127/#128 remain open.

## 0.95.18 — Strict IIS/MS-DOS FTP LIST compatibility (under verification) — 14 October 2026

- Rust desktop FTP/explicit FTPS listing parser now recognizes strict Windows IIS/MS-DOS file and directory listing records, including Unicode names and spaces, and preserves exact 64-bit file sizes.
- Invalid or unsupported server LIST rows fail the directory listing instead of silently hiding existing files from browsing, recursive operations and conflict detection.
- Regression tests cover Unix listing continuity, DOS dates/leap years, 12-hour clock, invalid paths, overflow, dot rows and mixed valid/invalid data.
- **Validation scope:** compatibility with a real Windows IIS FTP server remains UNVERIFIED until an independent IIS run is documented. No new UI artwork.
- #127 physical Android SAF acceptance and #128 Android/macOS authenticated cloud accounts remain OPEN; free FileZilla Client equivalence is NOT asserted.

## 0.95.17 — Android SAF recovery QA and real protocol binary edge cases — 14 October 2026

- **Safety:** Corrupted or tampered Android SAF download recovery journals now block new signed-object transfers until the user explicitly reviews and dismisses the warning. A second transfer can no longer overwrite a still-unresolved document record.
- **Android UI:** Explicit accessible descriptions for the two recovery actions, and on-emulator click-through tests for the valid/crash-restarted and damaged-record flows, including safe return to the cloud workspace.
- **Live protocol testing:** Real loopback FTP, explicit FTPS and SFTP servers receive deterministic binary data of 1, 65,535, 65,536, 65,537 and 262,147 bytes. Every case must pass upload, list, rename, download, exact-byte comparison and deletion. Existing multimegabyte pause/resume/cancel and host-key/certificate tests remain active.
- **Design QA:** Rechecked the new user-provided 75-screen `GhostFTP-Premium-Brand-UI(7).zip` mockup inventory and the existing real-app capture contract. No screenshots are generated from mockups and 1:1 pixel QA remains separately pending.
- **Limitations:** Full Android/macOS authenticated cloud account operations (#128), real Android 8–15 SAF provider acceptance (#127), and all 75 screen-by-screen installed-build comparisons are not certified. Do not claim FINAL or close these issues from a green CI job alone.

See [0.95.17 release notes](docs/releases/0.95.17.md).

## 0.95.16 — SAF document recovery and cloud authorization foundations — 14 October 2026

- Android: recover uncertain partial SAF download documents following process death with explicit user-controlled cleanup; journal only a bounded content URI in no-backup app storage.
- Android/macOS: implement S256-PKCE and single-use callback authorization primitives for future registered Google Drive OAuth clients, with regression tests.
- README: clarify that the checked-in screenshots were captured at v0.30.19 and must be replaced by verified current-build QA captures.
- Acceptance: #127 requires physical Android 8–15 SAF-provider tests; #128 requires full OAuth token storage, authorized account browsing, upload/download, expiry/revocation and real-provider E2E. Both remain open.

## 0.95.15 — official website support, stricter URL safety and localization improvements — 14 October 2026

- **Official help:** Windows, Linux, Android and macOS Help/About links now use verified ghostftp.com support, contact, getting-started, privacy and terms pages, instead of sending users to GitHub issues/legal markdown.
- **Security:** Exact HTTPS route allowlists prevent credential-bearing, redirected, off-domain or unknown URLs. GitHub remains a separately labelled source-code destination.
- **Language and UX:** Localized cloud/navigation/contact labels on all Android translations and eight official-resource texts on all desktop locales; other copy remains under review.
- **External file editing reliability:** Register native save/error event listeners exactly once when editing multiple files concurrently; preserve cleanup and retry when registration fails.
- **QA:** CI must pass on the exact commit before merge and public release. This release does not claim completed vendor OAuth, 75-screen pixel parity or exhaustive dead-code certification.

See [0.95.15 release notes](docs/releases/0.95.15.md).

## 0.95.14 — safer cloud cancellation and clearer transfer controls — 14 October 2026

- **Android cancellation safety:** Recheck cancellation immediately after each provider/network read, before processing a returned block; a regression test prevents late writes after cancellation.
- **Android cloud UX:** Upload/download picker actions disable while a document picker is pending or a transfer runs, and restore when the operation finishes. Invalid signed links are rejected and wiped from the input.
- **Documentation:** Keep the seven verified native Windows captures as the sole README gallery; advance version metadata and identify the remaining 75-screen pixel-parity and full function-level dead-code audit gaps.

See [0.95.14 release notes](docs/releases/0.95.14.md).

## 0.95.13 — privacy-first Settings and adaptive Android identity — 14 October 2026

- **Android privacy:** Privacy Policy and Terms of use now appear at the bottom of Settings, alongside the Brendigo author credit. These reuse the HTTPS-only allowlisted official links from Help & About.
- **Accessible QA:** Android instrumentation checks the legal links are visible and clickable in Settings and confirms the author row.
- **Adaptive app icon:** Android 8+ launcher icon now uses the approved Ghost FTP mark with the dark brand background; original in-app vector is preserved.
- **Localization:** Localized author attribution across English and 22 existing Android translation sets. Improved main English workspace messaging and Croatian privacy Settings introduction.
- **Production integrity:** Android API 36, separate installable Preview APK and intentionally unsigned release APK/AAB, real emulator/visual QA, desktop build and protocol E2E remain mandatory.
- **Privacy documentation:** Removed stale candidate language and clearly documented the Brendigo identity and account-free transfer architecture.

See [0.95.13 release notes](docs/releases/0.95.13.md).

## 0.95.12 — private profile export, Android 16 compatibility and premium README — 13 October 2026

- **Windows/Linux:** Export connection profile metadata using exclusive private staging, rollback during replacement, and symlink refusal. Added regressions for permissions and complete replacement.
- **Android:** Set API 36 compile/target, update Android Gradle Plugin and Gradle, test with an API 36 emulator; production bundles still require authorized signing and store acceptance.
- **Product:** Rewrite README for users with seven existing authentic desktop screenshots and honest platform compatibility details.
- **QA:** Validate source reachability, types, Rust/Swift/Kotlin tests, real protocol transfers and native installation artifacts on the exact commit before publication. Remaining #127/#128 are not marked resolved.
- **Release lineage:** Following published immutable v0.95.11 (21 artifacts).

See [0.95.12 release notes](docs/releases/0.95.12.md).

## 0.95.11 — post-release connection safety and review fixes (development candidate) — 13 October 2026

- **Windows/Linux:** Compact Files layouts retain Host, Username and a usable Quick Connect action; connection status shows real pending/success/failure/disconnected states.
- **macOS:** Opening an existing saved Site discards an abandoned unsaved New connection draft.
- **Android:** A saved connection's custom port is preserved across deferred protocol-spinner selection callbacks; saved Sites now support editing and confirmed removal, preserving all unrelated profiles and never persisting credentials.
- **Privacy:** The desktop UI store retains redacted metadata for a successful temporary session, not plaintext passwords or private-key passphrases.
- **Verification:** Extended premium Files contract checks. Installed native screenshots, Android physical-device port behavior and macOS selection acceptance remain pending.
- **Release lineage:** Follows published immutable v0.95.10 (21 assets). This source is not a published release.

See [0.95.11 release notes](docs/releases/0.95.11.md).

## 0.95.10 — direct Quick Connect and saved connection profiles (development candidate) — 13 October 2026

- **Windows/Linux:** The Files Quick Connect form now has an in-memory masked password input and connects using the existing ephemeral Rust FTP/FTPS/SFTP backend without opening the New Connection editor. The duplicate header New Connection action is removed.
- **Profile separation:** Sidebar New Connection is a save-only reusable Site editor. Save profile persists only after user action; Test Connection never persists. Saved Sites provide future one-click selection and connection.
- **macOS:** Creating a new Site no longer persists an empty profile. Files offers immediate temporary FTP Connect from the entered credentials, with existing FTPS/SFTP engine limitations stated plainly. Reusable profiles are saved only on Save and passwords use Keychain when explicitly remembered.
- **Android:** Sites supports locally saved reusable profile metadata (never passwords, tokens or key passphrases), selection repopulates the existing form, clears credentials and requires a fresh Connect action.
- **QA:** Static interaction contracts, product CI, platform-specific native build and real protocol testing are required. Real-device UI parity has not been independently certified.
- **Release lineage:** New source v0.95.10 follows published immutable v0.95.9; do not rewrite older tags.

See [0.95.10 release notes](docs/releases/0.95.10.md).

## 0.95.9 — Android/macOS signed-cloud public peer policy alignment (development candidate) — 13 October 2026

- **Security (#126 partial):** Disallow the RFC 7343 ORCHIDv2 `2001:20::/28` IPv6 address family as a signed-cloud socket destination on both Android OkHttp DNS and macOS peer-pinned curl.
- **Regression tests:** Cover the low and high nibble edges of that IPv6 prefix on each platform. Existing mixed-answer, direct-only, TLS-verifying transport and cancellation protections remain intact.
- **Visual QA:** Register the supplied 75-screen Windows/Linux/Android design concepts as explicit *references*, not application captures. Real installed-platform screenshots and behavior must be compared before claiming 1:1 parity.
- **Release lineage:** Advance package and source metadata to `0.95.9` after verified public `v0.95.8`; published tags are immutable. Issues #126–#128 remain open pending full device/provider acceptance.

See [docs/releases/0.95.9.md](docs/releases/0.95.9.md).

## 0.95.8 — Android private cloud stage recovery after process termination (development candidate) — 13 October 2026

- **Crash-residue cleanup:** On cold Android process start, remove strictly named app-private cloud download `*.part` staging files left if the process died before its `finally` block. Activity recreation in a live process never repeats the cleanup.
- **Privacy and safety:** Only regular stage files matching the internal prefix/suffix in `cacheDir` are eligible; no unrelated cache files, folders, symlinks, or SAF destinations are removed.
- **Regression tests:** Verify real temporary-file cleanup, unrelated-file and directory preservation, missing cache directory, and repeat-sweep idempotency; this does not replace physical Android 8–15/provider testing.
- **Release train:** Bump source metadata to `0.95.8` from the immutable verified public `v0.95.7` release. Windows/Linux/macOS/Android artifacts still require exact-head CI gates.

See [docs/releases/0.95.8.md](docs/releases/0.95.8.md).

## 0.95.7 — SAF provider cancellation isolation and release-lineage repair (development) — 13 October 2026

- **Android SAF cancellation (#127, partial):** Close each registered provider stream through a separate daemon worker. A blocked `close()` for one DocumentsProvider cannot prevent another active stream from receiving its close request.
- **Regression coverage:** Add a two-provider JVM cancellation test, alongside the existing lifecycle/race regressions.
- **Immutable release history:** The public v0.95.6 tag remains fixed at `a25403c78cb8d26e0dd57495c23598d81ed2860f`. PR #139 merged after that release; these changes advance the source to 0.95.7 from published v0.95.6, never rewriting published binaries or tags.
- **Acceptance limits:** Real Android 8–15 SAF device tests, provider DNS/proxy/VPN labs, OAuth integration, macOS secure FTPS/SFTP and 75-screen pixel comparisons still require evidence; no blanket 1:1 or production signing claim.

See [docs/releases/0.95.7.md](docs/releases/0.95.7.md).

## 0.95.6 — Android provider stream progress and SAF copy hardening (development) — 13 October 2026

- **Android cancellation stability:** When a transfer is canceled, attempt to close registered SAF provider input/output streams on a daemon worker instead of blocking the Android UI. Synchronize stream registration with cancellation to reject late stream opens; add three race-oriented JVM regressions.
- **Android reliability/security:** Add a bounded copy primitive for SAF uploads, signed cloud network download staging and SAF destination writes. Reject endless zero-byte reads from broken providers, early EOF, size growth and cancellation rather than allowing a hanging copy or optimistic success.
- **Android regression tests:** Add deterministic JVM tests for exact/empty files, truncated and oversized streams, eight recoverable zero-byte reads versus unbounded zero-byte responses, cancellation, impossible length metadata and write failures.
- **Multi-platform release discipline:** Windows/Linux/macOS continue from the merged v0.95.5 source, with no unrelated platform features removed. This is not a claim of full 1:1 parity or provider/device acceptance.
- **Security evidence limitation:** #126 real hostile DNS/proxy/VPN and signed cloud provider lab proof, #127 physical Android SAF failure injection and #128 OAuth browsing remain open.

See [docs/releases/0.95.6.md](docs/releases/0.95.6.md).

## 0.95.5 — cross-platform signed cloud URL privacy and SAF recovery (development) — 12 October 2026

- **Windows/Linux:** Treat signed S3, GCS and Azure SAS object URLs as bearer credentials in diagnostics; hide the entire signed URL and add executable redaction regression cases without removing ordinary URL diagnostics.
- **Android:** Clear signed URL fields before invoking system SAF document pickers, avoid bucket hostname exposure in transfer status, and remove newly created orphaned documents after Activity recreation when temporary authorization is unavailable; warn if provider deletion fails.
- **macOS:** Clear signed object URLs from SwiftUI input before open/save panels appear, including canceled selections, and keep transfer progress messages generic.
- **Version + QA:** Advance source to 0.95.5 without reusing immutable published v0.95.4; retain TS/Rust source-reachability, Kotlin/Swift private-symbol, native platform and signed-updater CI. This is not visual 1:1 or real-provider/device security acceptance.
- **Still open:** #126 network/provider lab, #127 physical Android SAF fault-injection, #128 real OAuth PKCE/account browsing.

See [docs/releases/0.95.5.md](docs/releases/0.95.5.md).

## 0.95.4 — Android post-write SAF integrity security fix (in development) — 12 October 2026

- **Security / Android:** Verify a newly created cloud download document after the SAF output stream closes, using exact byte count and SHA-256 of the destination readback against the staged transfer. Reject silent truncation, same-size corruption, extra bytes, unreadable destinations and cancellation rather than reporting success.
- **Recovery:** The existing Android failure path attempts to remove the new SAF document and warns when deletion fails; providers without atomic/transactional writes remain unverified.
- **Android/macOS cloud routing:** Close gaps in the public-address allowlist for reserved IPv6 benchmarking/documentation/retired ranges; recheck ASCII approved hostname policy at Android's connection-time DNS hook. Unit-test the rejection policy without weakening TLS or accepting private IPs.
- **Windows/Linux desktop security:** Patch the Tauri IPC core from 2.11.4 to 2.11.6, Rust updater to 2.12.0, npm updater to 2.12.0, and signing CLI to 2.12.2. Disable `allowDowngrades`, require cryptographically bound update versions with `requireSignedVersion`, regenerate Cargo/npm lockfiles in GitHub Actions and enforce these in Quality CI. The release still needs verified signed updater packages and all six exact-SHA gates.
- **Tests:** Add six JVM regression scenarios for actual byte integrity (no secrets or remote provider credentials).
- **Accurate acceptance:** Reopen #126, #127 and #128 because real DNS/VPN/provider, SAF physical device and OAuth end-to-end criteria are unmet. Do not confuse a green build with security acceptance.
- **Release lineage:** v0.95.3 remains public and immutable; 0.95.4 is unpublished source.

See [docs/releases/0.95.4.md](docs/releases/0.95.4.md).

## 0.95.3 — real cross-platform Transfers overview (in development) — 12 October 2026

- **Windows/Linux:** Interactive premium metric cards read active, completed and failed transfer counts directly from the existing backend queue; selected cards switch actual filters, with accessible keyboard states and responsive grid sizing.
- **macOS:** Credential-free persisted history supplies In progress, Completed, Failed and Cancelled counters; no synthetic file entries, progress, rate or provider claims.
- **Android:** Transfers workspace displays real, session-only remote-operation outcomes rather than mock transfer statistics.
- **QA:** Shared CI source-contract verification and mandatory platform builds; visual 1:1 acceptance remains pending until installed screenshots and real interactions are reviewed.
- **Release lineage:** Published v0.95.2 remains immutable; 0.95.3 starts a new source candidate.

See [docs/releases/0.95.3.md](docs/releases/0.95.3.md).

## 0.95.2 — post-release cloud cancellation safety and version lineage (in development) — 12 October 2026

- **macOS transfer cancellation:** Carry forward the race fix from the merged post-v0.95.1 PR #133. Curl launch and cancellation now share a process coordination lock; cancel before launch prevents a late network process, and cancellation after launch terminates the active process. Two XCTest regressions and a Quality contract guard this behavior.
- **Release integrity:** The immutable public v0.95.1 tag points to `ad3f9b546600dea53fbf5a9b14b1dd7799d84fcf`; the later merged `main` commit cannot reuse that tag. Advance the canonical source version, Android code, desktop/Rust/Go metadata and lockfiles to 0.95.2, and keep v0.95.1 unchanged.
- **Release gating:** This is source code in development, not a published 0.95.2 release. Require all six exact-head workflows, platform-package validation, security acceptance for #126/#127 and honest premium visual QA before publication. Vendor OAuth (#128) and UI parity (#131) remain outstanding.

See [docs/releases/0.95.2.md](docs/releases/0.95.2.md).

## 0.93.0 — private cloud storage transfer foundations — 10 October 2026

- **Windows/Linux:** Keep the existing S3, GCS, Azure Blob, WebDAV, Google Drive, OneDrive, Dropbox and Box account/file adapters. Custom S3/Azure endpoints now require HTTPS without embedded userinfo, query or fragment.
- **macOS:** A dedicated Cloud storage workspace supports direct signed HTTPS PUT/GET object upload/download, URLSession ephemeral mode, no redirects, a 1 GiB object limit and staged downloads. Signed link values are not persisted.
- **Android:** A Cloud storage workspace supports direct signed HTTPS PUT/GET object transfer through SAF documents, validates public endpoints, rejects redirects, stages downloads and limits each object to 1 GiB. Signed link values are not saved in Activity state or logs.
- **Scope boundary:** This is signed-object transfer on macOS/Android, not Google Drive OAuth or S3 bucket listing on those platforms. Real provider/device tests and exact-commit release CI remain mandatory.

See [docs/releases/0.93.0.md](docs/releases/0.93.0.md).

## 0.92.2 — macOS Keychain password write integrity — 10 October 2026

- **macOS:** Change saved-site passwords with `SecItemUpdate`, falling back to add only when no Keychain item exists; retry updates safely on concurrent duplicate inserts. Do not delete the existing secret ahead of a failed write.
- **macOS UI:** Failed password writes or removals no longer appear successful or remove an otherwise valid saved-site profile. Both the quick editor and Sites workspace report errors without exposing secrets.
- **FTP/FTPS:** When a late ABOR `225` response reaches the post-cancel SIZE check, force an unverified-offset restart rather than consuming a stale control reply as a fatal size error; all other unexpected server failures remain fatal.
- **Regression coverage:** Six XCTest cases exercise update, insert, concurrent collision and failure branches. This is scoped protection, not full device/UI or platform-protocol parity certification.
- **Release lineage:** Builds on publicly released v0.92.1. All six exact-SHA PR and merged-main checks plus verified package/checksum publication are required before publishing v0.92.2; published tags remain immutable.

See [docs/releases/0.92.2.md](docs/releases/0.92.2.md).

## 0.92.1 — macOS saved-site restore credential isolation — 10 October 2026

- **macOS privacy and safety:** Reject an imported saved-site UUID if its protocol, hostname, port or username differs from the already-saved connection. This closes a Keychain-credential reuse path where a backup could silently redirect an existing password to a different server.
- **Atomic restoration:** A conflicting backup changes no saved site, including unrelated entries in the same import. Metadata-only updates for the same connection identity remain supported.
- **User-facing feedback:** The Sync & Backup workspace explains why restoration was blocked without claiming any credentials were modified.
- **Regression coverage:** Swift XCTest cases cover host/protocol/port/username conflicts, full rollback and same-identity metadata updates.
- **Scope:** No change to FTP/FTPS/SFTP trust policy or existing release tags. Android production signing, macOS notarization and installed-device pixel parity remain unverified.

See [docs/releases/0.92.1.md](docs/releases/0.92.1.md).

## 0.92.0 — macOS saved-site duplicate discovery — 10 October 2026

- **macOS Sites:** Added a live **Possible duplicates** filter and per-site labels for saved connections that share protocol, normalized hostname, port and case-sensitive username.
- **macOS Files/editor:** Editing a saved connection now displays an accessible, non-destructive advisory when another saved profile uses the same endpoint and account.
- **Privacy:** Never compare or export Keychain passwords, SSH secrets, profile labels or file paths to infer duplicates; all profiles remain separate.
- **Quality:** Swift XCTest cases cover DNS host casing and trailing dots, username case, protocol/port differences, incomplete records and multiple matching entries.
- **No false parity:** Windows/Linux keep their existing filter; Android's saved-profile inventory differs; FTPS/SFTP remain unavailable for macOS file operations pending trusted identity verification. Pixel-perfect, signed/notarized and cross-device acceptance are not claimed.

See [docs/releases/0.92.0.md](docs/releases/0.92.0.md).

## 0.91.1 — cross-platform profile integrity, privacy and unsaved-edit safety — 9 October 2026

- **Windows/Linux:** Editing a saved site no longer silently drops an unsaved draft when the user switches sites or filters. The site editor has an explicit Cancel control; Escape preserves the draft until Save/Cancel.
- **Android:** Connection error redaction now hides URL usernames, including username-only FTP/FTPS/SFTP authority syntax; plain host diagnostics remain available.
- **macOS:** Corrupt/oversized persisted sites keep their original bytes under a separate recovery key before future writes; imports containing duplicate profile UUIDs fail without mutating saved data.
- **Regression tests:** Android JUnit, macOS XCTest and desktop interaction-contract checks target these failure paths.
- **Release boundary:** No previous release tag is moved, no mockups are presented as real screenshots, and 75-screen click/pixel certification is still pending.

See [docs/releases/0.91.1.md](docs/releases/0.91.1.md).

## 0.91.0 — saved-site duplicate discovery — 9 October 2026

- **Windows/Linux Sites:** New Duplicates filter and count identifies saved profiles sharing a protocol/host/port/username; details show a live advisory while editing.
- **Data preservation:** Duplicate detection never merges or deletes site profiles, bookmarks or separate OS-backed credentials. Ephemeral Quick Connect sessions are excluded.
- **Regression tests:** Production TypeScript identity-matching, case sensitivity, malformed site handling and UI wiring are exercised by the existing desktop quality gate.
- **Release limits:** Previous v0.90.1 remains immutable; existing real README images are preserved. Installed 75-screen acceptance and macOS FTPS/SFTP identity work remain open.

See [docs/releases/0.91.0.md](docs/releases/0.91.0.md).

## 0.90.1 — lossless settings migration and password-generation stability — 9 October 2026

- **Windows/Linux data integrity:** retain the legacy browser settings until every key's exact serialized value is returned by the database; wrong or partial writes cannot delete the recovery backup.
- **Startup reliability:** keep corrupt legacy settings for recovery, but ignore bad JSON without preventing app startup. Existing database preferences still take priority.
- **Security and performance:** reject invalid, non-finite and oversized password lengths before potentially blocking cryptographic generation.
- **Regression coverage:** executable tests cover incomplete/mismatched settings persistence, IPC failures, bad JSON, private legacy history cleanup, and password length boundaries.
- **Release safety:** published v0.90.0 remains immutable, original README gallery and image files are untouched. All exact-SHA workflows and genuine artifact verification remain required.
- **Acceptance limit:** the 75 reference screen pixel/click checks remain pending.

See [docs/releases/0.90.1.md](docs/releases/0.90.1.md).

## 0.90.0 — new version train and cross-platform privacy/transport hardening — 9 October 2026

- **Windows/Linux diagnostic privacy:** Mask credential-bearing FTP/SFTP URL userinfo (username as well as password) without hiding the useful server hostname; add executable regression coverage to the existing UI quality gate.
- **macOS protocol parser safety:** Reject EPSV replies that populate unexpected protocol/address fields; preserve valid alternate delimiter support and add Swift negative-case tests.
- **Release process:** Adopt 0.90.0 → 0.91.0 through 0.99.0 feature sequence; 0.90.1/0.91.1-style patch fixes, 0.99.x → 1.0.0 only after separate production acceptance, and 1.x semantic minor/patch progression. Enforce transitions and test them in CI.
- **Preservation:** Existing Windows/Linux/Android/macOS codebases, README gallery layout, seven verified screenshot files and their SHA256 provenance are unchanged.
- **Evidence limits:** Green source/build checks alone do not prove all 75 supplied reference screens are pixel-perfect or all interactions have passed real-device acceptance.

See [docs/releases/0.90.0.md](docs/releases/0.90.0.md).

## 0.30.21 — bounded desktop diagnostics and strict macOS FTP EPSV replies — 9 October 2026

- **Windows/Linux security & privacy:** Diagnostic redaction discards server-controlled text over 64 KiB before regex processing, masks macOS home-folder user names and hides unterminated private key blocks.
- **Executable regression coverage:** New desktop diagnostic privacy tests run in the existing UI check workflow for credential truncation, oversized responses, auth headers, private paths and private-key output.
- **macOS FTP safety:** Strict RFC 2428 EPSV port-reply structure rejects unexpected trailing fields from untrusted FTP servers, with added XCTest cases.
- **Release integrity:** v0.30.20 remains immutable. All six PR and merged-main CI workflows and canonical artifact verification are mandatory before v0.30.21 publication.
- **Limitations:** 75-screen installed-build visual parity and macOS notarization are not claimed.

See [docs/releases/0.30.21.md](docs/releases/0.30.21.md).

## 0.30.20 — cross-platform security, privacy, performance and verified screenshots — 9 October 2026

- **Windows / Linux:** Stop unnecessary 1-second status-clock refreshes in hidden desktop windows; refresh clock immediately upon return. Normal transfer, connection, and progress monitoring remain unchanged.
- **Android:** Mask passwords, tokens and credentials before clipping untrusted network error text to its 600-character UI budget; bound inspection cost and add regression tests for boundary-spanning secrets.
- **macOS:** Limit merged imported saved profiles to 512 without partial mutation; bound private transfer-history file names, migrate existing stored full-path history to basenames and regression-test recovery.
- **README visual fidelity:** Replace only seven existing Windows image assets with actual CI-verified 0.30.19 installed-Windows QA captures; keep the main README's image positions, HTML layout and written sections.
- **Dead-code audit:** Checked TypeScript import/reachability graph, Rust module graph, Kotlin/Swift private-symbol reachability and platform builds/tests. Removed the one-time screenshot importer after use. This does not justify deleting source only because it appears old.
- **Security and privacy review:** Verified fail-closed macOS unsupported FTPS/SFTP operations, file selection, release image SHA256 provenance and no intentionally seeded fake servers/transfer data. No independent penetration test is claimed.
- **Release integrity:** Corrected the previously merged-but-unpublished 0.30.20 source lineage from 0.30.19 to the correct new immutable v0.30.20 tag; existing v0.30.19 remains unchanged.
- **Acceptance limit:** Source-level checks and native build QA do not prove 75 screenshots are pixel-identical. Windows/Linux/Android 1:1 installed-build visual acceptance remains pending for unmatched screens. No macOS screenshot is present in the reference ZIP.

See [docs/releases/0.30.20.md](docs/releases/0.30.20.md).

## 0.30.19 — functional two-pane file manager and upload picker — 9 October 2026

- **Windows/Linux (Files UI):** With an active connection, Upload now opens a native OS file chooser if no local file is selected, in both single and dual-pane layouts. Existing selected-file upload remains one click.
- **Transfer consistency:** After an active transfer batch drains, both dual-pane file lists refresh actual local and remote directories; this matches existing single-pane refresh behavior.
- **Quality:** Regression checks protect the upload event bridge, connected Upload button, two refresh tokens and existing queue semantics.
- **Preservation:** Existing file operation permissions, overwrite/conflict prompts and secure protocols are unchanged. No demo servers/files/progress.
- **Reference status:** 75-screen visual acceptance is still pending installed-build comparisons; this improves real functionality, not certified 1:1 completion.

See [docs/releases/0.30.19.md](docs/releases/0.30.19.md).

## 0.30.18 — premium Help/About interface alignment — 9 October 2026

- Windows and Linux: Reference-shaped live Help/About product identity and resource cards, real version/build/date/platform values, approved tagline, genuine update action and responsive semantic-token styling; decorative mountain artwork removed.
- Android: Centered official icon/version, real product/security/privacy facts, and localized clickable official resources, accessible on compact screens.
- macOS: Installed app icon, responsive brand/resource panels, truthful FTP-only operation status and working external links. The concept ZIP has no macOS design screenshot.
- CI: New source-level reference-hierarchy regression guard; all existing workflow gates still mandatory.
- Limitation: 75 reference mockups remain pending installed-build pixel acceptance, not claimed as completed 1:1 layouts. No fake connection/activity data is shipped.

See [docs/releases/0.30.18.md](docs/releases/0.30.18.md).

## 0.30.17 — functional help resources and accurate cross-platform messaging — 9 October 2026

- **Windows/Linux:** About and Help now link to the canonical GitHub support, documentation, privacy policy and project releases instead of the retired public site; premium approved slogan restored throughout the real About workspace.
- **Android:** About hyperlinks use the same verified GitHub destinations and remain guarded by the existing HTTPS allowlist. Repository link title and description are localized for all 24 Android languages.
- **macOS:** Help/About links are functional project URLs. Security text distinguishes available FTP operations from FTPS/SFTP, still blocked pending identity verification.
- **CI:** checks the real in-app URLs, approved slogan and the unaffected official update endpoint; retains all six build/protocol/security gates.
- **Honest acceptance:** no pixel-perfect 75-screen claim without installed-build comparisons. macOS remains a non-notarized Preview.

See [docs/releases/0.30.17.md](docs/releases/0.30.17.md).

## 0.30.16 — unified premium live surface colors — 9 October 2026

- Windows/Linux primary sidebar now consumes the actual semantic panel token rather than a divergent hardcoded navy value; user-selected themes continue to work.
- macOS SwiftUI sidebar uses the same exact `#0D192B` panel shade as the canonical Android and desktop dark palette.
- Android rows and input surfaces derive from the existing shared Kotlin palette rather than duplicating RGB values.
- Extends mandatory CI branding checks to lock the shared live colors across all four supported platform implementations.
- No claim of complete 75-screen pixel acceptance: UI screenshots and functionality remain subject to installed-build checks. macOS FTPS/SFTP security limitations remain unchanged.

See [docs/releases/0.30.16.md](docs/releases/0.30.16.md).

## 0.30.15 — premium design parity and cross-platform hardening — 9 October 2026

- Windows/Linux: reference-width premium sidebar, consistent active/disabled keyboard focus states and preserved compact-window behaviour.
- Android: a single approved launcher/vector mark replaces the divergent Canvas ghost in the actual header and sidebar; Android 12–15 platform splash now uses the premium vector and navy canvas.
- macOS: refined SwiftUI connection action and sidebar brand lockup, compact-window minimum and previously merged exclusive `0600` private FTP staging protections.
- All 75 Windows/Linux/Android reference screen IDs are now mapped to live source components; the exact-head CI contract detects missing screens, but pixel acceptance remains pending.
- The source train advances from published v0.30.14 to v0.30.15 without retargeting any public release tags. macOS remains a Preview without FTPS/SFTP operations or notarization.

See [docs/releases/0.30.15.md](docs/releases/0.30.15.md).

## Unreleased — macOS FTP private download staging

- macOS plain-FTP downloads now create unique temporary files with exclusive, no-symlink POSIX creation and owner-only `0600` permissions before streaming any server data.
- Existing staging paths cannot be overwritten, even if an attacker substitutes a symlink. Transfer failures continue to remove incomplete data and close the FTP transport.
- Adds Swift regression coverage for private permissions, collision rejection and symlink protection.
- macOS FTPS/SFTP remain disabled pending complete certificate/hostname and SSH host-key verification; no protocol-security checks were bypassed.

## 0.30.14 — premium cross-platform brand foundations — 8 October 2026

- Applies the supplied premium navy/electric-blue design tokens to the real Windows/Linux, Android and macOS UI components rather than displaying mockups as application backgrounds.
- Refreshes the desktop brand mark frame, Android launcher vector and platform accent/status palettes; retains accessible focus states and reduced-motion behavior.
- Adds a shared design contract identifying screenshot/reference screens, platform-specific layout differences and the remaining measured parity/functional release blockers.
- This is a visual foundation, not a claim of 75 finished production screens or FTPS/SFTP macOS parity.

See [docs/releases/0.30.14.md](docs/releases/0.30.14.md).

## 0.30.13 — macOS FTP reliability and publication lineage — 8 October 2026

- Fixes actual macOS FTP RETR/STOR names and per-download unique staging filenames; adds hostile server-listing filtering, bounded FTP control replies and Swift regression tests.
- Locks the macOS FTP wire command and safety contracts into release-blocking CI.
- Distinguishes the preceding merged source version 0.30.12 from the latest **published** canonical release 0.30.10, and makes release publication verify the actual latest GitHub Release instead of waiting for missing 0.30.11/0.30.12 tags.
- Preserves Windows/Linux protocol and native build gates, Android emulator checks, and macOS Preview limitations without claiming unsupported cross-platform parity.

See [docs/releases/0.30.13.md](docs/releases/0.30.13.md).

## 0.30.12 — privacy and security hardening — 6 October 2026

- Hardens Windows/Linux encrypted backup restore with bounded input/decompression, Ghost FTP-only credential targets, private staging files, stronger new-backup password policy, explicit in-memory secret overwrites, command-layer secret wiping and guaranteed temporary snapshot cleanup.
- Prevents credential-bearing MCP child-process stderr from being reflected into UI diagnostics.
- Adds Android secure-window, tapjacking resistance and removal of persistent document grants while retaining memory-only passwords and strict FTPS/SFTP verification.
- Tightens macOS Keychain storage to unlocked, device-only, non-synchronizing items and validates/bounds password-free profile backup restore.
- Extends release-blocking security contracts and regression tests across desktop, Android and macOS.

See [docs/releases/0.30.12.md](docs/releases/0.30.12.md).

## 0.30.11 — cross-platform workspace parity and connection health — 6 October 2026

- Aligns macOS to the Windows/Linux six-workspace shell: Files, Sites, Transfers, Sync & Backup, Settings and Help & About, plus New connection.
- Removes the separate macOS Settings window and keeps primary navigation inside one persistent app shell.
- Adds credential-free macOS transfer history and password-free site profile backup/restore with regression tests.
- Fixes macOS transfer/save-panel interpolation so actual file names are displayed.
- Adds Android Sync & Backup with system-document JSON export/restore, a 256 KiB restore bound and explicit password exclusion.
- Extends Android emulator smoke to click through the new workspace.
- Adds Windows/Linux live connection-health status with visible Healthy / Checking / Needs attention states and visible-window periodic probes.
- Adds cross-platform workspace parity CI and a macOS private Swift dead-code audit while preserving existing TypeScript, Rust and Android audits.
- Bounds macOS profile restore input and recovers interrupted persisted transfers as Cancelled on the next launch.

See [docs/releases/0.30.11.md](docs/releases/0.30.11.md).

## 0.30.10 — streamed macOS FTP upload and download — 6 October 2026

- Adds real macOS plain-FTP `STOR` upload and `RETR` download over extended-passive data connections.
- Streams file payloads in 64 KiB chunks instead of buffering whole transfers in memory.
- Downloads to sibling `.ghostftp-*.part` staging files and promotes only after a successful FTP completion reply.
- Uses explicit macOS open/save panels and security-scoped URL access for user-selected files.
- Rejects invalid/traversal-style and command-injected remote transfer names before issuing FTP commands.
- Fails the FTP session closed after ambiguous transfer failure so stale control replies cannot contaminate later operations.
- Refreshes MLSD directory state after successful uploads and exposes transfer progress/state in the SwiftUI Preview.
- Keeps FTPS/SFTP on macOS explicitly unfinished until real TLS/SSH verification is implemented.
- Updates the locked `source-map-js` build dependency to 1.2.2 instead of suppressing the high-severity audit finding.
- Corrects npm dev-audit severity classification while keeping every unknown high/critical finding release-blocking.

See [docs/releases/0.30.10.md](docs/releases/0.30.10.md).

## 0.30.9 — real macOS FTP browsing and cross-platform connection health — 6 October 2026

- Hardens connection host parsing across desktop, Android and macOS so URL/credential/path/inline-port/control-character input fails before transport use.
- Adds a real macOS plain-FTP control session with greeting/authentication, TYPE I, PWD/CWD/NOOP/QUIT and bounded timeout/cancellation behavior.
- Adds real macOS EPSV + MLSD directory browsing, typed listing rows and an 8 MiB fail-closed listing-buffer limit.
- Adds live Windows/Linux FTP/FTPS NOOP and SFTP-channel health checks, including saved, Quick Connect and refocused sessions.
- Adds authenticated Android FTP/FTPS NOOP and SFTP PWD health probes without relaxing TLS or SSH host-key verification.
- Keeps macOS FTPS/SFTP, upload/download, Developer ID signing and notarization explicitly unfinished rather than presenting them as production-ready.
- Keeps exact-head Quality, protocol E2E, native build, Android, Windows hardening and macOS gates mandatory before canonical publication.

See [docs/releases/0.30.9.md](docs/releases/0.30.9.md).

## 0.30.8 — macOS foundation and cross-platform diagnostic hardening — 5 October 2026

- Starts the dedicated SwiftUI macOS client with Keychain-backed optional password storage, validated FTP/FTPS/SFTP profiles and a separate macOS CI/preview packaging gate.
- Keeps macOS transport checks honest: TCP reachability is not presented as protocol authentication, plain FTP is warned as unencrypted, and future FTPS/SFTP sessions must validate certificate/hostname or SSH host key before file operations.
- Redacts remaining user-visible Windows/Linux errors across Directory Diff, external editing, host-key prompts, Settings, Sync, Import, Agent Bridge and Skills.
- Adds Android runtime diagnostic sanitization with JVM and production-contract regression coverage while retaining lifecycle, SAF and credential-state protections.
- Extends canonical 0.30.8 publication to wait for the macOS gate and attach a clearly labeled ad-hoc-signed macOS Preview plus macOS source archive.
- Keeps exact-head Quality, real FTP/FTPS/SFTP E2E, Windows/Linux native build, Android and Windows hardening gates mandatory, with the macOS gate added for this release.

See [docs/releases/0.30.8.md](docs/releases/0.30.8.md).

## 0.30.7 — desktop production hardening, transfer safety and localization — 5 October 2026

- Prevents same-name batch transfer races by reserving destination names immediately after successful enqueue.
- Serializes Transfer Center pause/resume/cancel/retry/priority actions per transfer and queue-wide pause/resume actions.
- Makes Sync & Backup mutation failures propagate correctly so failed saves and actions cannot appear successful or close prematurely.
- Uses platform-neutral Sync path hints and strengthens Linux parity coverage for the shared Windows/Linux desktop UI.
- Removes raw Quick Connection and Site Manager connection diagnostics from the WebView console and clears quick-connect secrets before close.
- Replaces placeholder icon generation with fail-closed validation of reviewed production Ghost FTP brand assets.
- Completes Android resource-backed navigation/workspace/session localization parity and adds duplicate/missing resource enforcement.
- Excludes the Android session password field from Autofill in addition to saved Activity state.
- Keeps exact-head Quality, protocol E2E, Windows/Linux native build, Android and Windows hardening gates mandatory.

See [docs/releases/0.30.7.md](docs/releases/0.30.7.md).

## 0.30.6 — transfer integrity, mobile UX and security hardening — 4 October 2026

- Remote overwrite and rename existence probes now fail closed across Dynamics, Shopify, HubSpot, OneDrive, Dropbox, FTP, Agent and object-backed paths instead of treating transport, permission or API failures as proof that a target is absent.
- Agent protocol responses distinguish confirmed not-found conditions from operational failures so destructive overwrite decisions propagate ambiguous errors safely.
- Android now uses a responsive left navigation drawer on compact screens, preserves the persistent navigation rail on larger screens and replaces popup confirmation dialogs with inline in-app confirmation UI.
- Android networking disables global cleartext traffic, explicitly warns when plain FTP is selected and removes unused keyboard-interactive SSH authentication.
- Ghost FTP Agent identity/fingerprint handling now rejects malformed or incorrectly sized X25519 keys and identity-file I/O failures; Unix identity files are enforced as owner-only `0600` instead of ignoring permission-hardening failures.
- Desktop official Support, Documentation, Privacy and EULA destinations are centralized and restricted to approved external URLs.
- Advertised desktop locale coverage now has enforced reference-UI parity across all registered languages, including completion dictionaries for previously incomplete locales.
- Production CI regressions found during the hardening cycle were fixed without weakening formatting, protocol E2E, Android, native build or Windows hardening gates.

See [docs/releases/0.30.6.md](docs/releases/0.30.6.md).

## 0.30.5 — Settings ordering and Android URI lifecycle hardening — 4 October 2026

- Serializes Windows/Linux Settings persistence so rapid preference changes cannot complete durable database writes out of user-action order.
- Prevents failed older preference writes from rolling the UI back over a newer action by tracking per-setting revisions and the last durable value.
- Serializes live transfer-engine IPC updates for concurrency, retries, throttle and delta sync so native runtime state cannot finish out of user-action order.
- Gives Reset to Defaults, notification permission and shell PATH integration one shared async mutation lock and busy state.
- Keeps Reset to Defaults locked through recovery and restores the previous DB snapshot, live transfer-engine settings and shell PATH state before returning failure.
- Prevents Preferences from closing while an OS-level Settings mutation is still running.
- Rejects Android picker URIs that cannot actually be opened for reading instead of presenting an unusable file as selected.
- Revalidates saved Android upload URIs after Activity recreation and clears stale grants fail-closed.
- Adds desktop interaction-contract and Android instrumentation/production-contract coverage for the new lifecycle guards.
- Keeps exact-head Quality, real FTP/FTPS/SFTP E2E, Windows/Linux native build, Android and Windows hardening gates mandatory.

See [docs/releases/0.30.5.md](docs/releases/0.30.5.md).

## 0.30.4 — cross-platform interaction serialization — 4 October 2026

- Serializes Agent Bridge master, endpoint, session-access and approval-policy mutations on Windows/Linux so rapid or conflicting clicks cannot race backend IPC state.
- Disables Bridge mutation controls while an operation is in flight and exposes endpoint Start/Stop busy state.
- Prevents Android Pick file, Reset Transfers and Reset Connection from mutating visible transfer context while a transfer/remote operation is active.
- Makes Android Settings Disconnect reflect the real authenticated-session state instead of remaining clickable while disconnected.
- Adds desktop interaction-contract and Android instrumentation/production-contract coverage for the new state guards.
- Keeps the exact-head Quality, real FTP/FTPS/SFTP E2E, Windows/Linux native build, Android and Windows hardening release gates mandatory.

See [docs/releases/0.30.4.md](docs/releases/0.30.4.md).

## 0.30.3 — Android interaction-state and dead-code hardening — 3 October 2026

- Fixes Android action availability so disconnected sessions no longer expose Refresh, Disconnect or remote file operations as actionable controls.
- Keeps the local Android file picker available before connection while guarding remote Upload until a session exists.
- Adds click-by-click instrumentation for disconnected and invalid-session enabled states.
- Serializes Windows/Linux Sync pair toggle, Sync now and Remove actions so duplicate/conflicting IPC mutations cannot race from rapid clicks.
- Adds an Android private Kotlin symbol audit so declaration-only private functions and fields fail the production contract.
- Keeps runtime npm high/critical findings release-blocking while machine-checking the temporary dev-only upstream `braces` advisory exception.
- Preserves the existing cross-platform exact-head quality, real protocol E2E, native build, Android and Windows hardening release gates.

See [docs/releases/0.30.3.md](docs/releases/0.30.3.md).

## 0.30.2 — cross-platform UX and transfer-integrity hardening — 3 October 2026

0.30.2 follows published 0.30.1 and packages the safety and polish completed after that canonical tag.

- Protects desktop overwrite downloads with sibling staging, backup-before-promotion and rollback so failed/canceled transfers cannot truncate an existing local destination.
- Extends staged local replacement to Ghost FTP Agent whole-file fallback while preserving the hash-verified delta path.
- Makes desktop SFTP Rename conflict probing and directory-upload metadata handling fail closed on ambiguous errors.
- Keeps Android FTP/FTPS/SFTP staged replacement fail closed with explicit existence classification, backup, promotion and rollback recovery.
- Adds a bounded Android instrumentation retry only for the confirmed Package Manager split-APK `Broken pipe` transport failure; real test failures stay blocking.
- Makes the Android connection form protocol-aware: FTP/FTPS/SFTP default ports follow 21/21/22 without overwriting custom ports, and SFTP fingerprint input appears only when relevant.
- Adds click-through instrumentation and production-contract coverage for protocol switching, custom-port preservation and SFTP security-field visibility.
- Compacts the shared Windows/Linux Files toolbar at minimum native widths so all contextual actions remain reachable without horizontal panning.
- Gives the active-site desktop control an explicit accessible name with the current site and action.
- Removes the unused Android `MAX_QUEUE_ROWS` alias and confirms no declaration-only private symbols remain in the audited Android Activity/connection controller, while mandatory desktop source-reachability and unused-local checks remain active.
- Synchronizes the 0.30.2 version/build metadata across desktop, Rust workspace, Android, Go compatibility tools and updater templates.
- Splits npm security validation into a zero-tolerance runtime audit plus a machine-checked dev-only exception for the unresolved upstream `braces` advisory, rejecting any unrelated or runtime high/critical finding.

See [docs/releases/0.30.2.md](docs/releases/0.30.2.md).

## 0.30.1 — staged-upload safety and version-line hardening — 2 October 2026

0.30.1 follows published 0.20.10 and starts a broader hardening cycle.

- Makes Android FTP/FTPS/SFTP staged upload replacement fail closed when an existing remote target cannot be safely preserved.
- Requires explicit remote-target existence verification before promotion instead of interpreting backup-rename failure as target absence.
- Refuses promotion when a target appears during an in-progress upload.
- Reports failed backup restoration and preserves the remote backup path for manual recovery instead of silently discarding recovery failure.
- Adds Android regression coverage and production-contract checks for staged replacement safety.
- Centralizes staged remote replacement across FTP/FTPS/SFTP so backup, promotion, rollback and temporary cleanup use one fail-closed transaction.
- Adds JVM regressions for existence-check denial, backup/promotion failures, rollback success/failure, late target races and SFTP `NO_SUCH_FILE` versus permission/protocol failures.
- Makes desktop SFTP Rename candidate probing fail closed so permission, connection and protocol errors can never be mistaken for an available destination.
- Adds Rust regressions proving only a positively classified missing path may be selected by the bounded Rename candidate search.
- Stops directory uploads from silently skipping local entries whose metadata cannot be read; the operation now fails closed with the affected source path.
- Aligns Agent Bridge directory-upload preflight with the same metadata guard so its approval summary cannot authorize a silently incomplete source tree.
- Stages desktop whole-file downloads into transfer-specific sibling files and promotes them only after successful completion, so failed/canceled overwrite downloads cannot truncate an existing local destination.
- Uses backup-before-promotion rollback for local replacement and routes the Ghost FTP Agent whole-file fallback through the same staged protection while preserving its existing hash-verified delta path.
- Allows intentional forward SemVer jumps only when the new version is strictly newer and `previousVersion` matches the exact published base.
- Adds Windows-hardening concurrency so superseded PR/ref runs are canceled instead of consuming runners.

See [docs/releases/0.30.1.md](docs/releases/0.30.1.md).

## 0.20.10 — protocol edge cases and conflict safety — 2 October 2026

0.20.10 follows published 0.20.9 with real protocol edge-case coverage and stricter fail-closed conflict handling.

- Adds zero-byte, Unicode remote-name and permission-denied FTP/FTPS/SFTP E2E coverage.
- Proves changed SFTP host keys are surfaced as mismatches, rejected by default and replaced only after explicit trust.
- Expands pause/resume coverage to a non-aligned multi-MiB payload spanning repeated chunk boundaries.
- Fixes remote conflict rename semantics for dotfiles and Windows-style paths.
- Prevents exhausted Rename candidate searches from falling back to a colliding existing path.

## 0.20.9 — transfer history and release/updater hardening — 1 October 2026

0.20.9 follows published 0.20.8 with a real transfer-history export and hardened release/updater publication.

- Adds end-to-end desktop CSV transfer-history export through Transfer Center, state, IPC and the persisted Rust transfer ledger.
- Neutralizes spreadsheet-formula prefixes in exported path cells and excludes raw backend error text from the shareable CSV.
- Stable GitHub packages may publish without private updater keys; unsigned in-app updater metadata is never generated.
- When updater signing is enabled, signatures, exact-version manifest and Update-Service package are required as an all-or-nothing set and re-verified in the final publish script before tag mutation.
- Extends source reachability to Rust crate module graphs and referenced CI/update/Android helper scripts.
- Expands operational Node/MJS and shell syntax checks.
- Adds Android JVM unit tests for connection input/path safety and cooperative cancellation, and makes `testDebugUnitTest` part of the Android gate.
- Refreshes active documentation from stale RC/current-cycle claims to the 0.20.9 / previous 0.20.8 line.
- Keeps Android production and installable-preview signing semantics explicitly separate.

See [docs/releases/0.20.9.md](docs/releases/0.20.9.md).

# Changelog

## 0.20.8 — cross-app action parity hardening — 1 October 2026

0.20.8 follows published 0.20.7 with Android action parity and stronger click-through enforcement.

- Adds remote Rename on Android for FTP, explicit FTPS and SFTP using the real protocol clients.
- Expands Android Delete to remove either a remote file or an empty remote folder; non-empty folders are never removed recursively.
- Adds long-press folder selection so directory rename/delete actions can target a folder without navigating into it first.
- Replaces the informational-only Android Settings workspace with working Clear Activity, Reset Transfers, Reset Connection and Disconnect controls.
- Persists the non-secret rename target across Activity recreation while continuing to clear passwords and authenticated session state.
- Expands instrumentation smoke coverage and the Android production contract so these actions cannot silently regress into dead UI.

See [docs/releases/0.20.8.md](docs/releases/0.20.8.md).

## 0.20.7 — dead-code and source-reachability hardening — 1 October 2026

0.20.7 follows published 0.20.6 with repository-wide TypeScript dead-code enforcement and source reachability checks.

- Enables TypeScript unused-local diagnostics for the desktop application and shared file-ui package.
- Adds a production Quality gate that walks desktop TypeScript entrypoints and fails on source files that are not reachable from the active application/package graph.
- Keeps cleanup evidence compiler- and entrypoint-driven so files are deleted only when they are demonstrably unused.
- Preserves the 0.20.6 production bundle budget and all exact-SHA release gates.

The audit removed 11 unreachable frontend files: `IconPicker.tsx`, `KeyboardSettings.tsx`, `ProfileEditor.tsx`, `PromptModal.tsx`, `RemoteControlSettings.tsx`, `ui/Badge.tsx`, `ui/Skeleton.tsx`, `ui/Tooltip.tsx`, `brandIconData.ts`, `brandIcons.tsx` and the legacy host `lib/fileIcons.tsx` shim.

TypeScript also identified 21 unused imports/declarations across active files. Those were removed from FilePane, Settings, Site Manager, Terminal and settingsStore, together with the now-unreferenced breadcrumb parser/helper chain.

See [docs/releases/0.20.7.md](docs/releases/0.20.7.md).

## 0.20.6 — frontend bundle-size hardening — 1 October 2026

0.20.6 follows published 0.20.5 with measurable desktop frontend bundle-size hardening.

- Replaced the eager full Material Icon Theme manifest and ~900-icon glob with a curated offline icon set for common development, document, media, archive and creative file types.
- Preserved Ghost FTP's existing Lucide fallback for file types outside the curated branded set.
- Split file-browser, i18n and generated brand-icon data into dedicated stable production chunks instead of keeping them in the main application chunk.
- Added a hard production JavaScript bundle budget: any emitted JS chunk above 500 KiB fails the desktop build instead of only producing a Vite warning.
- Kept all file icons local to the application; no runtime icon network requests were introduced.

See [docs/releases/0.20.6.md](docs/releases/0.20.6.md).

## 0.20.5 — Android API and edge-to-edge hardening — 1 October 2026

0.20.5 follows published 0.20.4 with Android 15 system-bar compatibility and SFTP credential API hardening.

- Replaced the deprecated JSch String password setter with the byte-array API and zeroed the temporary UTF-8 password buffer immediately after JSch copied it.
- Removed direct deprecated Android status/navigation bar color assignments from the Activity.
- Added Android 15 system-bar inset handling so the programmatic root layout remains clear of enforced edge-to-edge system UI on API 35+.
- Added an API 35 theme override that keeps Ghost FTP's dark system-bar icon contract without deprecated status/navigation bar color attributes.
- Added Android production-contract checks that reject reintroduction of deprecated SFTP password and system-bar APIs.
- Preserved the existing FTP/FTPS/SFTP behavior, Android emulator click-through smoke, clean install/reinstall checks and exact-SHA release gates.

See [docs/releases/0.20.5.md](docs/releases/0.20.5.md).

## 0.20.4 — CI runtime and build-config hardening — 1 October 2026

0.20.4 follows published 0.20.3 with CI runtime modernization and build-configuration cleanup.

- Updated first-party GitHub Actions used by the release-relevant workflows to their current supported major trains: checkout/setup-node/setup-go v7, setup-java v6, setup-android v4, Gradle Actions v6 and upload-artifact v7.
- Disabled setup-go dependency caching in the two helper-tool workflows because the helper modules intentionally have no `go.sum`, eliminating the recurring false cache warning.
- Replaced Vite config `__dirname` usage with ESM-safe `fileURLToPath(import.meta.url)` directory resolution so Vite 8 native config loading no longer reports the compatibility warning.
- Added a CI/runtime regression contract that rejects stale action majors, re-enabled unsupported Go cache behavior and reintroduction of Vite `__dirname`.
- Preserved the existing exact-SHA Quality, protocol E2E, native build, Android and Windows hardening release gates.

See [docs/releases/0.20.4.md](docs/releases/0.20.4.md).

## 0.20.3 — file-action reliability hardening — 1 October 2026

0.20.3 follows published 0.20.2 with shared file-browser interaction hardening and release-documentation alignment.

- Serialized shared file-browser confirmation and prompt actions so asynchronous operations cannot be submitted twice while still running.
- Kept rename, new-folder, chmod and delete dialogs open when the underlying filesystem operation fails, allowing a safe retry instead of presenting a premature close as success.
- Added generic in-dialog failure feedback without exposing raw backend errors; detailed errors continue through the existing file-pane error path.
- Removed redundant prompt focus logic and kept focus trapping/restoration under the shared dialog hook.
- Added interaction-contract guards for shared file-ui async action locking and failure propagation.
- Corrected Android release documentation to match the current unsigned production APK plus separate installable-preview validation policy.

See [docs/releases/0.20.3.md](docs/releases/0.20.3.md).

## 0.20.2 — UI and launch-smoke hardening — 1 October 2026

0.20.2 follows published 0.20.1 and carries the post-0.20.1 Android/desktop polish that was merged after the 0.20.1 tag.

- Reworked Android into exclusive Files, Sites, Transfers, Settings and Help & About workspaces behind a persistent left navigation rail instead of one long stacked screen.
- Preserved only non-secret Android workspace state across Activity recreation while keeping passwords and authenticated sessions out of saved state.
- Fixed Android workspace navigation so the selected workspace is scrolled into the visible content area, including the Transfers viewport case caught by emulator instrumentation.
- Made the Android launcher smoke verify the real package-specific RESUMED Activity state and live PID instead of depending on unstable human-readable `am start -W` output formatting.
- Kept Android production contract, lint/APK build, click-through instrumentation, clean install, reinstall and launch proof as release blockers.
- Hardened the shared Windows/Linux Disconnect action so async disconnect errors are surfaced through the normal redacted error path.
- Centralized Windows/Linux workspace resolution for the app shell, primary sidebar and title bar to remove duplicated navigation-state mapping.
- Retained the 0.20.1 transfer worker-registration race fix and exact-SHA Quality, protocol E2E, native build, Android and Windows hardening gates.

See [docs/releases/0.20.2.md](docs/releases/0.20.2.md).

## 0.20.1 — patch cycle — 30 September 2026

0.20.1 follows canonical 0.20.0. It is not considered published until the exact-source GitHub Release and required assets exist.

- Closed a desktop transfer worker-registration/finalization race where an immediately completed worker could finish before its `JoinHandle` was recorded, leaving stale completed task state in memory.
- Added Tokio regression coverage for the completed-before-registration interleaving and routed download, upload and manual-retry workers through the race-safe registration helper.
- Advanced desktop, Rust workspace, Android, compatibility-tool and updater metadata to 0.20.1 with Android `versionCode` 211030.
- Simplified Android release packaging so it no longer depends on private `ANDROID_RELEASE_*` secrets or a second optional signing branch.
- The Android gate now always verifies an intentionally unsigned production `com.ghostftp.android` APK plus a non-debuggable installable preview used for clean-install/reinstall/launch smoke; canonical release naming keeps the unsigned production artifact explicitly marked `.unsigned`.
- Reworked the Android shell into separate Files, Sites, Transfers, Settings and Help & About workspaces behind a persistent left navigation rail, with workspace state preserved across Activity recreation without persisting credentials.
- Hardened the shared Windows/Linux title-bar Disconnect action so asynchronous disconnect failures are caught locally, redacted through the normal error path and surfaced to the user instead of falling through to the global async handler.
- Centralized the Windows/Linux workspace resolver used by the app shell, primary sidebar and title bar, removing three independent copies of workspace-state mapping and adding a regression guard against reintroduction.
- Refreshed build, QA, product, versioning and release documentation for the patch cycle while keeping documentation imagery pinned to the latest published release until new release-proven captures can replace it.

See [docs/releases/0.20.1.md](docs/releases/0.20.1.md).

## 0.20.0 — release cycle — 30 September 2026

0.20.0 follows canonical 0.19.0; live publication status is determined by GitHub Releases.

- Added a durable, credential-free desktop transfer ledger in SQLite so interrupted queue/history state survives process restart and application updates.
- Recovered transfers now reconnect through the saved profile and OS credential store; cross-process retries restart safely from byte zero when source identity cannot be proven, preventing hybrid/corrupted files.
- Bounded terminal transfer history during runtime while preserving active transfers, avoiding unbounded in-memory and SQLite growth in long-running clients.
- Made FTP and explicit FTPS pause/resume byte-accurate, including cooperative ABOR cleanup, fresh authenticated control-session recovery after interrupted transfers, and verified resumed uploads with safe full-restart fallback when the remote final size is wrong.
- Expanded real FTP/explicit-FTPS protocol E2E coverage for pause/resume, cancellation, ABOR race recovery, control-channel isolation and I/O failure cleanup, plus real OpenSSH SFTP non-zero-offset resume primitive verification.
- Hardened Android transfers with staged upload/download replacement, lifecycle-aware cancellation for upload/download/delete/new-folder actions, and deterministic stream cleanup.
- Corrected Android SAF persisted-permission handling so the picker keeps persistable access while `takePersistableUriPermission` receives only valid READ/WRITE grant modes.
- Tightened Android stable-signing continuity so an existing stable APK must be downloadable, inspectable and signed by the same certificate; failures now block release rather than silently skipping the check.
- Kept Windows console-free helper hardening, Windows NSIS/MSI lifecycle smoke, Linux AppImage/DEB/RPM lifecycle smoke and exact-head release gates mandatory.
- Hardened release CI against transient Ubuntu APT stalls with bounded retries/timeouts and extended exact-SHA gate waiting so valid Windows/Linux/Android release candidates are not dropped by a premature orchestration timeout.
- Updated release/version metadata, documentation and updater templates for 0.20.0.

See [docs/releases/0.20.0.md](docs/releases/0.20.0.md).

## 0.19.0 — release cycle — 27 September 2026

0.19.0 follows canonical 0.18.0; live publication status is determined by GitHub Releases.

- Branded the canonical Windows NSIS Setup with Ghost FTP installer/uninstaller icons, header/sidebar artwork and the root EULA as the interactive licence page.
- Added real silent Setup install/uninstall smoke coverage to the canonical Windows native build.
- Added Linux AppImage/DEB/RPM lifecycle and package metadata checks.
- Updated Android FTP/FTPS and SFTP libraries to maintained releases.
- Switched native release codegen to size-focused `opt-level = "s"` while retaining LTO, one codegen unit, panic abort and stripped symbols.
- Explicitly disabled optional AppImage media-framework bundling because Ghost FTP does not require audio/video playback.
- Reduced CI native artifact staging to final package files instead of expanded bundle directories.
- Added artifact size budgets and build-summary size reporting.
- Fixed updater note synchronization and hardened version-sync against concurrent branch pushes.
- Kept README/documentation imagery pinned to the newest published release tag.
- Removed the obsolete website application source, website CI/release packaging and active website roadmap/docs.
- Added release-proven real native application screenshot gallery to the root README.
- Added Android emulator click-through instrumentation and desktop-aligned Files/Sites/Transfers/Settings/Help & About navigation.
- Made Android run on every `main` push so exact-SHA release orchestration always has a mobile gate.
- Removed a dead preview verifier that depended on a non-existent mock runtime.
- Updated EULA, commercial licence, privacy, security, support and third-party notices for the app-only distribution model.

See [docs/releases/0.19.0.md](docs/releases/0.19.0.md).

## 0.18.0 — published — 27 September 2026

0.18.0 was published after 0.17.0 and established the canonical release-image provenance/native-build flow.

- Added CI provenance validation that keeps README/documentation images byte-identical to the newest published Ghost FTP release tag.
- Consolidated duplicate Windows/Linux Tauri build paths into one canonical production native build.
- Kept Windows native-window screenshot QA in the canonical build and added its evidence archive to release packaging.
- Removed the obsolete duplicate native build workflow and stale RC-era workflow references.
- Removed hard-coded `v2.1.1-Preview` source package naming from the active build path.
- Fixed version synchronization so ignored release-note paths no longer break the bot commit.
- Made version synchronization refresh and commit the canonical Cargo.lock atomically with Cargo metadata changes.
- Kept exact-SHA quality, real protocol E2E, Android and Windows hardening around the canonical native build.
- Continued the no-version-only policy.

See [docs/releases/0.18.0.md](docs/releases/0.18.0.md).

## 0.17.0 — published — 27 September 2026

- Completed the canonical public-history migration.
- Hardened Windows background helper processes against unintended console flashes.
- Added Windows helper-process regression checks.
- Made Android validation run on every pull request to `main`.
- Allowed meaningful fixes to remain within the active development version without forcing a version-only bump.
- Published normalized Windows, Linux, Android, source, documentation and SHA-256 assets.

See [docs/releases/0.17.0.md](docs/releases/0.17.0.md).

## Older canonical history

Canonical release notes for `0.1.0` through `0.16.0` are retained under `docs/releases/`.

The older `2.1.1-rc.*` identifiers are historical aliases only. Their verified mapping/provenance is recorded in `docs/releases/version-map.json`; they are not the active version scheme.
