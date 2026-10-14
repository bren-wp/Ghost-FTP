# Ghost FTP / FileZilla functional parity

Last reviewed: 2026-10-15 (v0.95.18 published; post-release FileZilla importer PR pending CI)
Benchmark: FileZilla Client
Reference: https://filezilla-project.org/client_features.php

FileZilla is used only as a functional benchmark. Ghost FTP must not copy FileZilla UI, branding or visual identity.

This tracker deliberately errs toward **Partial** or **Missing**. A capability moves to **Equivalent** only after the Ghost FTP implementation and regression tests cover the relevant behavior. A capability moves to **Ghost FTP better** only after reproducible evidence demonstrates a concrete advantage.

## Post-v0.95.18 FileZilla Site Manager import — development candidate (CI pending)

On the new development branch after published v0.95.18, the existing FileZilla XML importer now maps nested saved-site folder breadcrumbs into the real Ghost FTP `ConnectionProfile.group` field; previously it only displayed them as a note. This reuses the existing Sites group selection: nested FileZilla folders are represented by their full breadcrumb as a single Ghost FTP group label, **not** a claim of a nested tree UI. Existing import preview checkbox, credential-free profile save and premium UI remain unchanged.

The importer also recognizes FileZilla exports whose site display name is direct trailing `<Server>` text rather than a `<Name>` element. An explicit `<Name>` still takes priority. Regression source tests cover two-level folder mapping through `into_profile`, trailing site names, unnamed folders and no password import.

Security boundary: 8 MiB XML input cap, 2,048 accepted site limit, 32 nested folder levels, rejected DTD and malformed text escapes, and rejected structurally incomplete folder/server records. These caps apply before or during parsing; no password/credential XML content is transferred to the preview. These changes do **not** implement FileZilla bookmark-path pairs, default remote directory path decoding, arbitrary platform-specific setting migration, or implicit FTPS protocol support.

**Status: PARTIAL / unverified until exact-head Rust, UI, Android, macOS and native CI results.** A Rust unit test proves the importer's behavior on representative exported XML shapes; real user-import acceptance and every FileZilla version are not claimed. Existing issue #127/#128 and broader protocol/visual parity remain open.

## Evidence matrix — 0.95.18 published baseline

Scope: **free FileZilla Client**, not commercial FileZilla Pro. Feature reference: [official FileZilla Client features](https://filezilla-project.org/client_features.php) (the JavaScript-only page may not expose text in a headless environment) and [FileZilla documentation index](https://filezillapro.com/docs/v3/) (differentiate Pro-specific features). Items are not marked EQUIVALENT solely because a method or UI button exists. The latest **published** Ghost FTP baseline is v0.95.17; this branch has **unreleased**, not yet final-CI-verified 0.95.18 code.

| Function | FileZilla Client | Ghost FTP implementation | Evidence | Work needed | E2E | Status |
| --- | --- | --- | --- | --- | --- | --- |
| FTP passive control and file transfers | Supported | Rust session plus FTP RemoteFs | Real loopback FTP fixture at v0.95.17, CI run 38619915744 | Additional real FTP server and NAT/IPv6 tests | Local FTP yes | PARTIAL |
| Explicit FTPS private data channel | Supported | Rust TLS FTP stream and guarded data-channel retry | FTPS local server, hostname mismatch rejection (run 38619915744) | Expiry/revocation/chain and multiple server matrices | Local FTPS yes | PARTIAL |
| SFTP / SSH keys / fingerprint changes | Supported | Russh SFTP and OS known-hosts | Real OpenSSH password/encrypted-key auth and changed-key regression (run 38619915744) | Agent, ProxyJump and keyboard-interactive provider matrix | Local OpenSSH yes | PARTIAL |
| IIS/MS-DOS and Unix FTP LIST | Required server compatibility | Strict DOS date/size, Unix mode/name validation, unknown-row and duplicate-name rejection on Windows/Linux | `src-tauri/src/remotefs/ftp.rs` unit regressions and pyftpdlib DOS socket fixture; latest exact-SHA CI pending | Real Windows IIS, ProFTPD, Pure-FTPd and recovery across live server failures | Loopback DOS fixture (earlier SHA), latest pending | PARTIAL |
| macOS FTP MLSD listing safety and transfer | Required interoperability | Swift MLSD parsing fails closed on unknown/ambiguous entries, supports RFC 3659 cdir/pdir, Unicode, checks duplicate names; FTP session supports real file upload/download | `macos/.../FTPControlSession.swift`, `FTPControlSessionTests.swift` and `.github/workflows/ghostftp-macos.yml` now start a local pyftpdlib FTP daemon for login, NOOP, MLSD, STOR, RETR and byte-equality checks | Confirm new exact-SHA Swift workflow, real vendor matrix, LIST fallback, complete FTPS/SFTP engines | Live macOS socket E2E introduced, latest CI pending | PARTIAL |
| Android FTP/FTPS/SFTP remote inventory | Ghost FTP mobile extension | After FTP LIST, checks real completion status; FTP/SFTP rows reject hostile/duplicate names, show truthful first-24 limit, and mark symlinks/special entries non-actionable | `android/.../ConnectionModel.kt`, `ConnectionModelTest.kt`: JVM regressions for reply codes, unsafe/duplicate entries, limits, FTP symlinks; latest Android CI pending | Full paginated browsing, larger directories, actual Android 8–15 FTP/FTPS/SFTP server matrix, disconnect/retry/lifecycle | Unit source changed; physical device not verified | PARTIAL |
| FEAT, MLSD/MLST and LIST fallback | Supported by FTP ecosystem | Current production LIST parser, guarded dot visibility | `src-tauri/src/remotefs/ftp.rs` | Negotiate/verify MLSD and multiple daemon behavior | LIST on local fixtures | PARTIAL |
| Active FTP PORT/EPRT and IPv6 | Supported | No documented E2E for active IPv6/NAT paths | No active network fixture coverage | Implement/configure/test active data channel and NAT safety | No | UNVERIFIED |
| Transfer queue and retries | Supported | Existing Rust queue, TS Transfer Center, SQLite recovery snapshots | `docs/qa/TRANSFERS.md`; 0.95.17 tests | Per-site controls, crash, concurrency and real network failure matrix | Partial | PARTIAL |
| Interrupted upload/download resume | Supported | FTP REST/SFTP range and resume primitives | Protocol E2E v0.95.17 tests multi-chunk resume | Verify large files, cross-protocol loss/cancel and identity changes | Local subset yes | PARTIAL |
| Synchronized browsing and directory compare | Supported | Existing compare/sync modules | `docs/product/FEATURES.md`, inspection pending | Bidirectional root traversal with security and UI tests | Not established | UNVERIFIED |
| Filename/directory filtering and remote search | Supported | Existing search/filter surfaces, completeness not reviewed here | Source audit needed | Safe bounded regex, multi-rule presets and recursive cancellation tests | Not established | UNVERIFIED |
| Site Manager, bookmarks and importing | Supported | Existing profile store and site UI, FileZilla XML importer with security-aware protocol ID mapping | `src-tauri/src/importers/filezilla.rs` source plus 5 unit regressions added in PR #152 (exact-SHA CI pending) | Implicit FTPS ID 3 safely skipped until a real implicit engine exists; surface skipped entries in UI, validate per-site settings and migrations | CI pending; no live IIS | PARTIAL |
| File editing and drag/drop | Supported | Existing managed temporary editor and drag/drop UI | `docs/qa/TRANSFERS.md` | Remote conflict, editor atomic-save and OS integration tests | Not established | PARTIAL |
| Network Configuration Wizard | Supported | Saved FTP/FTPS sites: measured DNS, direct TCP, authenticated session and passive directory LIST via new Settings control | `src-tauri/src/network_diagnostics.rs`, `commands.rs`, `lib.rs`, `src/components/Settings.tsx`; five unit regressions including conditional IPv6 loopback and site default-path selection; exact-SHA CI pending | Active PORT/EPRT, NAT detection, proxy tests, error-specific TLS diagnostics and real multi-host network fixture | Existing protocol E2E proves underlying FTP LIST; wizard IPC E2E pending | PARTIAL |
| SOCKS5/HTTP CONNECT proxy | Supported | No verified complete proxy support | No reproducible proxy fixture tests | Securely implement/authenticate proxy with DNS and data-channel tests | No | UNVERIFIED |
| Android SAF interrupted cloud recovery | Ghost FTP extension | Private content-URI journal, recovery choices, nonblocking abandoned-picker cleanup and exact-document journal clearance | `MainActivity.kt`, `CloudDestinationJournal.kt`, 2 new JVM regressions in `CloudDestinationJournalTest.kt` (new exact-SHA Android CI pending) | Real Android 8–15 provider fault injection, lifecycle cancellation and storage failure matrix; issue #127 | Emulator/JVM subset only | PARTIAL |
| Android/macOS cloud account OAuth | Separate FileZilla Pro-type extension | Ephemeral PKCE foundations; no end-to-end account browsing | Platform unit tests, issue #128 | Registered clients, token vault, list/paging, refresh, revocation and provider tests | No | PARTIAL |
| 75-screen premium visual target | Ghost FTP-specific design, not FileZilla parity | Reference registry = concepts; real build capture registry separate | `docs/qa/PREMIUM_REFERENCE_75.md` | Exact screen-by-screen pixel and click QA on each native OS | Not complete | UNVERIFIED |

The matrix deliberately distinguishes **source implementation**, **validated local test**, and **real cross-vendor interoperability**. Report actual source SHA and workflow ID before upgrading statuses.

### Android native remote browser reliability (0.95.18 candidate)

The Android FTP browser previously accepted the result of `FTPClient.listFiles()` as a successful listing without checking the server's *final FTP reply*. A denied or failed LIST could therefore look like an empty but successful directory. It now requires a positive final reply, preserving the distinction between a valid empty directory and a listing failure.

Both FTP/FTPS and SFTP list adapters validate every returned filename before truncating the UI to its intentional 24-row limit; they refuse ambiguous duplicate identities and path injection. Unsupported remote entry types and links remain visible where supported but are not offered as ordinary file transfers. The UI states that it shows only the first 24 rows when the server returns more files; **this is transparency, not pagination parity**. FTP pre-overwrite existence checks also refuse duplicate candidate targets. Android JVM regressions cover the changed core behavior, but Android build, emulator and real remote-host interoperability on the latest SHA remain pending.

### 0.95.18 continuation — remote file inventory integrity

- On Windows/Linux, FTP `LIST` parsing now rejects duplicate filenames even if the same name appears in mixed Unix/IIS/DOS records. Duplicate names can cause ambiguous conflict resolution, directory operations and UI selection. Added Rust unit cases; await exact-SHA Rust CI and multi-vendor interoperability.
- On macOS, the Swift `MLSD` parser previously **silently discarded** invalid or hostile entries. It now fails the entire listing instead of presenting an incomplete tree, validates types/sizes/facts, rejects repeated identities, and explicitly does not classify unsupported OS-specific types as normal files. Tests retain Unicode and names with spaces. The macOS workflow now also provisions a real local FTP server and runs an opt-in Swift async protocol roundtrip (login, NOOP, MLSD, STOR, RETR and byte equality); success must be verified on the exact candidate SHA and cannot establish third-party interoperability or encrypted FTPS/SFTP support.
- RFC 3659 permits `cdir`/`pdir` to name actual paths, and permits empty values in optional MLSD facts. Strict parsing accepts these valid cases and ignores pseudo-entries rather than mistaking them for files. Source: https://www.rfc-editor.org/rfc/rfc3659.html
- These are scoped inventory-safety fixes, **not** proof of macOS secure FTPS/SFTP support or broad FileZilla parity. No changes were made to the premium UI layout. Final Swift, Rust, Android and native package workflow results must be checked on the same head SHA.

### Strict Unix LIST parser hardening (CI follow-up)

The Rust workspace suite identified two failing tests where SuppaFTP's Unix parser accepted arbitrary server text such as `LIST malformed entry` as if it were a file. The production Ghost FTP parser now validates the Unix permission/type token *before* calling SuppaFTP, rejects path separators/control characters in file names, and refuses an unsupported listing instead of returning a partial inventory. This also guards recursive deletes and remote-existence/conflict checks that depend on the listing. Valid Unix files/directories, IIS/MS-DOS rows, and traversal cases have source-level regressions. Final exact-SHA test results remain pending; real Microsoft IIS remains unverified.

### Saved-site network diagnostics — 0.95.18 development candidate

A collapsed diagnostics control has been added inside existing **Settings → Connection** without changing the Ghost FTP palette or layout. It intentionally selects an existing saved FTP/FTPS profile rather than accepting an arbitrary URL. Running it explicitly performs:
1. bounded DNS resolution, reporting IPv4/IPv6 *resolved* (not automatically reachable);
2. a direct TCP control-port connect with bounded socket timeouts, recording which family actually connected;
3. the real saved connection path, including mandatory TLS for FTPS and secrets hydrated inside Rust;
4. `LIST` over the authenticated server session via the existing RemoteFs engine, proving a usable passive data channel or reporting its failure. When configured, the saved site's default remote path is used rather than assuming the server's login-home directory can be listed.

This diagnostic never changes OS firewall/router, never exposes secret values in its report, does not support proxy paths and **does not** claim active FTP/NAT or FileZilla wizard equivalence. A failed stage suppresses all subsequent success claims. The Rust helper tests cover real loopback TCP (including optional IPv6), zero targets, saved directory selection and not-run states. Source additions are pending final exact-SHA CI and a first-party UI automation / real multi-host network matrix.

### PR #152 follow-up security assertions (exact-SHA CI pending)

- The FileZilla Site Manager importer now interprets IDs `0` (opportunistic TLS), `3` (implicit FTPS), `4` (required explicit TLS) and `6` (plaintext FTP) according to the upstream protocol enum, not the former inverted mapping. An opportunistic-TLS site is imported as **required explicit FTPS** with a visible explanatory note rather than silently downgraded. An implicit-TLS site is skipped rather than misrepresented as explicit FTPS; the import dialog warns about this limitation. Source: `src-tauri/src/importers/filezilla.rs` plus five regression cases.
- Recursive FTP deletion now refuses to begin if the server rejects `LIST -a`. Bare `LIST` cannot prove hidden entries are included; deleting only visible entries can otherwise leave a partially deleted directory tree. Source: `src-tauri/src/remotefs/ftp.rs` with a guard regression. A real server that rejects `LIST -a`, with existing hidden files, is still required for destructive-operation E2E evidence.
- These are targeted safety improvements, not proof of FileZilla equivalence or complete FTP interoperability.

## Ghost FTP better

No capability is claimed here yet. Marketing superiority is not a substitute for measured evidence.

## Equivalent

No broad category is claimed equivalent yet. Individual protocol primitives may already work, but equivalence requires a sufficiently broad compatibility and recovery matrix.

## Partial

| Capability | Ghost FTP status | Evidence / work still required |
| --- | --- | --- |
| FTP client core | Partial | FTP connection and transfers exist and protocol E2E CI exists. Expand PASV/EPSV/PORT/EPRT, IPv4/IPv6, FEAT, UTF8, MLSD/MLST, LIST fallback, SIZE, MDTM, REST, keepalive, timeout and NAT coverage. |
| Explicit FTPS | Partial | Implemented; continue TLS certificate/hostname validation and real-server interoperability coverage. |
| Implicit FTPS | Partial | Desktop support must be exercised by dedicated regression tests; Android parity is not yet claimed. |
| SFTP | Partial | Password/key paths and host-key protections exist. Continue encrypted-key, keyboard-interactive, changed-host-key, SSH Agent and ProxyJump coverage where applicable. |
| Upload / download queue | Partial | Queue exists. Continue crash/restart/update persistence, retry/cancel races and atomic completion tests. |
| Pause / resume | Partial | Do not claim complete parity until byte-range resume is proven for both directions across every supported FTP/FTPS/SFTP path. |
| Interrupted transfer recovery | Partial | Network disconnect/server restart/timeout recovery requires broader fault-injection evidence. |
| Site management | Partial | Existing Sites/profile surfaces are functional; continue auth/profile migration and compatibility regression tests. |
| Remote file operations | Partial | Rename, delete, properties, folder creation and remote edit paths exist. Continue permission, Unicode, long-path and conflict coverage. |
| Remote editing | Partial | Ghost FTP downloads to a managed temporary location, opens an editor and watches for save/upload. Continue crash and atomic-save edge cases. |
| Search / sync utilities | Partial | Source modules exist, but feature-by-feature benchmark parity is not yet established. |
| Large-file / many-small-file behavior | Partial | Must be backed by repeatable stress tests and memory/resource measurements. |
| Cross-platform desktop packaging | Partial | Windows and Linux native bundles are CI-built and lifecycle-tested. Wider distro/runtime coverage remains incomplete. |
| Android client | Partial / product extension | Native Android app exists and exact release-candidate install/launch is tested. Protocol parity and cross-release signing continuity remain incomplete. |

## Missing

A feature belongs here when no production implementation and test evidence are present. Keep this list evidence-driven; do not infer absence merely because a feature was not reviewed in one cycle.

Current high-priority gaps to close before claiming full FileZilla-class parity:

- a proven full byte-range resume matrix for upload and download across FTP, explicit FTPS, implicit FTPS and SFTP;
- crash/restart/update-safe persistent queue recovery with no stored credentials;
- a broad active/passive FTP compatibility matrix including IPv6 and NAT edge cases;
- complete SSH Agent / ProxyJump coverage where desktop support is intended;
- representative real-server and OS compatibility evidence beyond the current CI fixtures;
- Android cross-release upgrade continuity using an existing secure persistent signing identity.

## Intentionally different

- Ghost FTP keeps its approved Ghost FTP UI/UX and branding; FileZilla UI is not copied.
- Ghost FTP may expose capabilities through different navigation or workflows when that preserves the approved product design.
- Native Android is treated as an additional Ghost FTP platform rather than a reason to imitate FileZilla desktop UI.

## Status discipline

For every parity change:

1. link the implementation path;
2. add or strengthen a regression / interoperability test;
3. run the relevant platform and protocol gates;
4. update this file only after the evidence exists;
5. never promote a status because a feature merely compiles or because a mock reports success.

### 2026-10-15 continuation — bounded diagnostics, SAF crash recovery and visual QA

- **FTP diagnostics (PARTIAL):** `src-tauri/src/network_diagnostics.rs` previously gave each of up to eight resolved addresses a three-second TCP timeout, permitting up to 24 seconds in a multi-address failure. The existing direct-control probe now has an eight-second **total** deadline, retaining the three-second per-address maximum and family detection. Real loopback and zero-budget/refused-address regressions were added. Active FTP, NAT and proxy wizard diagnostics are **not** implemented by this change.
- **Android SAF (#127, PARTIAL):** `CloudDestinationJournal.kt` now treats an abandoned `pending-cloud-download.tmp` staging record as unresolved after process death. New downloads refuse to replace this record; explicit dismissal clears the staged marker without following symlinks, and failed atomic journal writes retain uncertainty rather than deleting the evidence. Two JVM regressions cover a crash-left staging record and staging symlinks. These are not substitutes for real DocumentsProvider fault injection or physical Android 8–15 acceptance.
- **Windows native QA (UNVERIFIED until exact-SHA run):** On 2026-10-14, exact SHA `66b6fa912c5a137063dcdad0de7860a48c6aa818` passed quality, protocol E2E, macOS, Android, and Windows hardening, but **failed** Windows native screenshot QA: both `PrintWindow` and the first CDP capture yielded structureless frames (`main`: one color, zero edge ratio). The workflow now retries **real** WebView2 compositor screenshots with both CDP surface modes and validates each frame using the same strict screenshot contract; it still fails if no real UI appears. It does not substitute mock assets or skip the acceptance gate.
- No broad category is promoted to `EQUIVALENT`, and neither #127 nor #128 is closed. Recheck all six workflows on the **new exact head SHA**, then inspect the Windows screenshot artifacts and physical-device/protocol evidence before merge or release.

### 2026-10-15 Windows hardening regression — exact source fix pending CI

On `b0069334e184ecf7d40221e501dfdb9e61990710`, `Validate Windows hardening` failed at Rust unit `network_diagnostics::tests::a_refused_address_does_not_hide_a_later_reachable_address`: `left: None`, `right: Some("IPv4")` (250 passed, 1 failed, 13 ignored). With two addresses and an intentionally tight two-second total budget, the first timed-out address could legitimately consume the whole budget before the live loopback address was tried. This was a real failed test, not a successful acceptance.

The production TCP control probe now divides the remaining *total* timeout over the remaining DNS answers, capped at three seconds per address. The real-socket fallback regression uses the production eight-second total budget to avoid depending on Windows refused-port timing. A zero-budget test still fails closed. This improves IPv4/IPv6 alternate-address fairness without claiming an active FTP/IPv6 full-stack parity or a NAT wizard.

Verify all six workflows on the **new exact commit** before changing any status, merging or releasing. #127 and #128 remain OPEN.
