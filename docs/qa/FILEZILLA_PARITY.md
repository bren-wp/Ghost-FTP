# Ghost FTP / FileZilla functional parity

Last reviewed: 2026-10-14 (0.95.18 PR #152; final CI pending)
Benchmark: FileZilla Client
Reference: https://filezilla-project.org/client_features.php

FileZilla is used only as a functional benchmark. Ghost FTP must not copy FileZilla UI, branding or visual identity.

This tracker deliberately errs toward **Partial** or **Missing**. A capability moves to **Equivalent** only after the Ghost FTP implementation and regression tests cover the relevant behavior. A capability moves to **Ghost FTP better** only after reproducible evidence demonstrates a concrete advantage.

## Evidence matrix — 0.95.18 development candidate

Scope: **free FileZilla Client**, not commercial FileZilla Pro. Feature reference: [official FileZilla Client features](https://filezilla-project.org/client_features.php) (the JavaScript-only page may not expose text in a headless environment) and [FileZilla documentation index](https://filezillapro.com/docs/v3/) (differentiate Pro-specific features). Items are not marked EQUIVALENT solely because a method or UI button exists. The latest **published** Ghost FTP baseline is v0.95.17; this branch has **unreleased**, not yet final-CI-verified 0.95.18 code.

| Function | FileZilla Client | Ghost FTP implementation | Evidence | Work needed | E2E | Status |
| --- | --- | --- | --- | --- | --- | --- |
| FTP passive control and file transfers | Supported | Rust session plus FTP RemoteFs | Real loopback FTP fixture at v0.95.17, CI run 38619915744 | Additional real FTP server and NAT/IPv6 tests | Local FTP yes | PARTIAL |
| Explicit FTPS private data channel | Supported | Rust TLS FTP stream and guarded data-channel retry | FTPS local server, hostname mismatch rejection (run 38619915744) | Expiry/revocation/chain and multiple server matrices | Local FTPS yes | PARTIAL |
| SFTP / SSH keys / fingerprint changes | Supported | Russh SFTP and OS known-hosts | Real OpenSSH password/encrypted-key auth and changed-key regression (run 38619915744) | Agent, ProxyJump and keyboard-interactive provider matrix | Local OpenSSH yes | PARTIAL |
| IIS/MS-DOS and Unix FTP LIST | Required server compatibility | Strict DOS and Unix row parse; unknown lines abort instead of silently dropping | `src-tauri/src/remotefs/ftp.rs` DOS/Unix unit tests on PR #152 | Finish exact-SHA Rust CI and real Windows IIS FTP run | IIS: not yet | UNVERIFIED |
| FEAT, MLSD/MLST and LIST fallback | Supported by FTP ecosystem | Current production LIST parser, guarded dot visibility | `src-tauri/src/remotefs/ftp.rs` | Negotiate/verify MLSD and multiple daemon behavior | LIST on local fixtures | PARTIAL |
| Active FTP PORT/EPRT and IPv6 | Supported | No documented E2E for active IPv6/NAT paths | No active network fixture coverage | Implement/configure/test active data channel and NAT safety | No | UNVERIFIED |
| Transfer queue and retries | Supported | Existing Rust queue, TS Transfer Center, SQLite recovery snapshots | `docs/qa/TRANSFERS.md`; 0.95.17 tests | Per-site controls, crash, concurrency and real network failure matrix | Partial | PARTIAL |
| Interrupted upload/download resume | Supported | FTP REST/SFTP range and resume primitives | Protocol E2E v0.95.17 tests multi-chunk resume | Verify large files, cross-protocol loss/cancel and identity changes | Local subset yes | PARTIAL |
| Synchronized browsing and directory compare | Supported | Existing compare/sync modules | `docs/product/FEATURES.md`, inspection pending | Bidirectional root traversal with security and UI tests | Not established | UNVERIFIED |
| Filename/directory filtering and remote search | Supported | Existing search/filter surfaces, completeness not reviewed here | Source audit needed | Safe bounded regex, multi-rule presets and recursive cancellation tests | Not established | UNVERIFIED |
| Site Manager, bookmarks and importing | Supported | Existing profile store and site UI | `docs/product/FEATURES.md` | FileZilla XML import security, per-site options and migrations | Partial | PARTIAL |
| File editing and drag/drop | Supported | Existing managed temporary editor and drag/drop UI | `docs/qa/TRANSFERS.md` | Remote conflict, editor atomic-save and OS integration tests | Not established | PARTIAL |
| Network Configuration Wizard | Supported | No verified parity in current settings | No reproducible end-to-end wizard | Implement real network/DNS/control/data diagnostics without router changes | No | MISSING |
| SOCKS5/HTTP CONNECT proxy | Supported | No verified complete proxy support | No reproducible proxy fixture tests | Securely implement/authenticate proxy with DNS and data-channel tests | No | UNVERIFIED |
| Android SAF interrupted cloud recovery | Ghost FTP extension | Fail-closed, private document URI journal and recovery choices | Android 0.95.17 instrumentation success (run 38619915768) | Physical Android 8–15 and provider errors/permission acceptance, issue #127 | Emulator only | PARTIAL |
| Android/macOS cloud account OAuth | Separate FileZilla Pro-type extension | Ephemeral PKCE foundations; no end-to-end account browsing | Platform unit tests, issue #128 | Registered clients, token vault, list/paging, refresh, revocation and provider tests | No | PARTIAL |
| 75-screen premium visual target | Ghost FTP-specific design, not FileZilla parity | Reference registry = concepts; real build capture registry separate | `docs/qa/PREMIUM_REFERENCE_75.md` | Exact screen-by-screen pixel and click QA on each native OS | Not complete | UNVERIFIED |

The matrix deliberately distinguishes **source implementation**, **validated local test**, and **real cross-vendor interoperability**. Report actual source SHA and workflow ID before upgrading statuses.

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
