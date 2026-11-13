# Ghost FTP 0.99.59 — repository-wide risk and quality review

**Date:** 13 November 2026  
**Repository:** [bren-wp/Ghost-FTP](https://github.com/bren-wp/Ghost-FTP)  
**Baseline source:** `1750fc66a1f58611fe28f525bd7e73d9b59047ee` (main, 0.99.58 Preview)  
**Owner/brand:** BRENDIGO · https://brendigo.com

## Scope and boundaries

The complete tracked Git tree was inventoried (**752 entries, 677 tracked files, ~9.18 MB**), split into ~66 Rust-core, 99 React/UI, 45 CI, 35 tools, 10 updater, 317 documentation, 32 other Tauri, 63 other desktop and 10 root files. Selected high-risk workflows, release scripts, runtime configuration, production-readiness notes, cloud provider/QA files and security/dead-code audit implementations were examined. Existing quality, Rust, protocol, hardening and native-build workflows were checked for exact-SHA status. This is **not** a line-by-line manual proof of all source files, real provider testing, or an external penetration test.

## Findings, with priority and acceptance

| Priority | Evidence and finding | Required intervention / acceptance |
| --- | --- | --- |
| P0 — governance | GitHub API reports default branch `main` as **unprotected**; CI is extensive but could be bypassed by an authorized direct push | Enable GitHub branch ruleset: required PR, reviews, status checks, dismiss stale approval, no force push and no bypass except explicit audited emergency policy |
| P0 — release | Version 0.99.58 source is on main; first inspection still showed v0.99.57 as newest published. Native build was in progress | Finish release from exact merged SHA, verify 15 assets, SHA-256 and GitHub digest checks. Never manually forge release success or rewrite old tags |
| P0 — Stable | Stable v1.0.0 signature and installed-machine gates are not evidenced | Authenticode identity, Linux authenticity scheme, signed updater, clean install/upgrade/rollback, real OS tests. Remain Preview until verified |
| P1 — active docs | `docs/qa/CLOUD_E2E.md` and `docs/cloud/PROVIDERS.md` advertised old unsupported client platforms; installer QA described retired products; production readiness was two releases stale | Preserve historical content under audit archive, replace live docs with Windows/Linux only, add new fail-closed current-docs CI test |
| P1 — cloud | Desktop provider adapter source exists, but no documented matrix of complete, authorized live-provider pass results | Per-provider account test, OAuth refresh/revoke, unicode/large files, 429/5xx, TLS/redirect/DNS, checksum and cancellation evidence |
| P1 — transfers | Unit and localhost protocol tests exist; independent concurrent large-file, crash/disk-full and heterogeneous server acceptance remains open | Test cancellation, resume, partial files, overwrite policy, rename/recovery, IPv4/IPv6, TLS and known_hosts rotations without data loss |
| P1 — credentials | Linux secret-tool subprocess protections and Windows OS vault integration have tests; real locked vault behavior and migration/upgrade acceptance remain open | Test locked/unavailable vault, secret deletion, no fallback to plaintext, process lifecycle, concurrent migrations |
| P1 — UI/accessibility | Native captures and UI contracts exist; the full DPI, keyboard/screen reader and viewport matrix is not demonstrated | Test 480x600 to larger screens, 200–400% zoom, tab/focus order, screen reader, reduced motion and responsive dialogs |
| P2 — architecture | `transfer.rs` (~290 KB), `bridge.rs` (~279 KB), `commands.rs` (~99 KB), `i18n.ts` (~202 KB), `styles.css` (~166 KB) are large maintenance hotspots | Split by stable domain APIs with characterization tests and contract-preserving incremental PRs; no bulk refactor |
| P2 — static audits | Security/dead-code scanners run in CI but are heuristics and cannot certify runtime behavior or all vulnerabilities | Keep conservative findings, explain false positives, add compiler/fuzz/integration coverage and external review; remove only proven unreachable code |

## Work performed in this PR

- Updated active cloud provider and E2E documentation to accurately describe the current Windows/Linux client and outstanding real-provider tests.
- Archived original mobile-platform QA and provider matrices instead of deleting historical evidence.
- Corrected active installer and production readiness pages.
- Added a reproducible CI guard that checks active documentation against the declared version, predecessor and supported platforms, with negative regression tests.
- Prepared the next monotonic Preview version and documented evidence limits. Already published tags and release assets are immutable.

## Proposed implementation sequence

1. **Release integrity/governance:** finish current edition; enforce branch protection; complete automated version lineage, artifacts, checksums and post-release verification.
2. **Critical behavior:** transfers, filesystem staged-write/rollback, SSH/FTPS trust and credential vaults; add property/fuzz and crash injection for the most sensitive code.
3. **Provider verification:** genuine scoped provider E2E in isolated accounts; refuse any false certified claims.
4. **Maintainability:** isolate Rust transfer queue/retry/staging, bridge protocol framing, source commands and localized UI modules one subsystem at a time; preserve user profiles and migrations.
5. **Native quality:** accessibility, window scaling, visual review against approved real references, Windows/Linux installer lifecycle.
6. **Stable promotion:** collect real publisher-signing and installation evidence, then build a separate v1.0.0 release from an approved exact commit. No simulated signing.

**Safety rules:** never delete users, profiles, existing OS secrets, stored site connections or public release history; never force push/retag; do not edit main directly. Any future deep source audit must report proven findings separately from unverified risks.
