# Ghost FTP master development — verified phase ledger (28 October 2026)

This is an **execution record**, not a claim of full product acceptance. Source baseline: main merge `c78756525ac9dedd4fc5cd49bc0dffb85f9a276d` (v0.99.17). PR #242 was merged and v0.99.18 published (all seven exact-head CI checks green). Current work: [PR #243](https://github.com/bren-wp/Ghost-FTP/pull/243) (v0.99.19 development Preview).

| Phase | Source/implementation evidence | Acceptance status and next gate |
| --- | --- | --- |
| 1. Stabilization | PR #241 atomic backup rotation merged; seven exact-head PR workflows passed before merge. PR #242 adds fail-closed recovery tests, symlink rejection and corruption-snapshot deduplication. | **In progress.** v0.99.17 was published and verified. Re-run exact-SHA CI for #242, including Go race detector and concurrent-state stress tests. Hardware/Windows disk-fault injection and the native macOS protocol security issue #223 remain open. |
| 2. Client functionality | v0.99.18 merged #242; v0.99.19 PR #243 checks FTP/FTPS/SFTP destination-occupied rename preflight and real protocol E2E for both unchanged contents. Android Sites CTA opens actual editor. | **INCOMPLETE.** Native Swift macOS FTPS/SFTP issue #223, Android sync/mirror engine, and server-supported atomic rename-if-absent semantics remain unverified; do not imply full client parity. |
| 3. Pixel-perfect 75 screens | v8/v10 checksum verified; v0.99.17 Windows/Linux native QA + v0.99.18 Android debug-only 480×960 QA + [v0.99.19 real Windows/Linux Sync](PHASE3_REAL_WINDOWS_LINUX_SYNC_V0_99_19_2026_10_29.md): [7 actual Android matches](PHASE3_REAL_ANDROID_V0_99_18_2026_10_29.md), [7 Windows](PHASE3_REAL_WINDOWS_V0_99_17_2026_10_28.md), [1 Linux](PHASE3_REAL_LINUX_V0_99_17_2026_10_28.md). | **INCOMPLETE: 17/75 reference-to-runtime screenshot comparisons, 0/75 human-accepted.** Mixed released builds, real disconnected state vs design illustration, no native macOS concept and missing per-screen functional/accessibility sign-off. |
| 4. New Ghost FTP Server | No `ghost-server/` module in baseline tree. | **Not implemented.** Create a separate actual FTP/FTPS server engine, identity/auth/authorization, admin UI, Windows service and portable execution with security tests, not mock screens. |
| 5. Client ↔ Server integration | Existing `.github/workflows/ghostftp-protocol-e2e.yml` covers protocol fixtures. | **Blocked** for the new Ghost FTP Server until an actual server exists. All four clients need identity, transfer integrity, interrupts and authorization tests against it. |
| 6. Security, dead code and performance | Existing `tools/audits/`, `docs/audits/` and platform checks; scoped new finding `docs/audits/FALLBACK_STATE_RECOVERY_2026_10_28.md`. | **Incomplete for the full new scope.** Keep critical issues blocking, preserve user data, document exact-SHA findings and run security, migration, CVE, release chain and performance QA across each target OS. |
| 7. Distribution | v0.99.16 public preview carried 25 assets (client builds, source, checksums and QA); 0.99.17 release published with 25 GitHub assets. | **Not a complete Client+Server release.** Windows Server portable/setup do not yet exist; verify each client/OS package, signing status, checksum and installability separately. |
| 8. GitHub release | PR #241 merged; v0.99.17 source on main; PR #242 staged for v0.99.18. | **Not complete.** Public v0.99.17 is verified; PR #242 exact-SHA gates must pass before merge and any subsequent preview release. Never repoint immutable tags. |

## Deterministic work order

1. v0.99.17 is now publicly published (25 artifacts); keep its immutable tag and source provenance unchanged.
2. Run and fix all PR #242 tests, including Go race detector, cross-platform Rust mutations, native Swift macOS FTP hidden-file preflight and live protocol E2E. No bypassing failing gates.
3. Review open macOS issue #223 and ship protocol identity verification only with real positive/negative E2E.
4. Continue functional UI audit and capture true installed screen states before visual acceptance.
5. Implement Ghost FTP Server as an isolated product, then test platform client/server matrices.
6. Close release security/performance/localization and package gates before calling any full cross-platform release complete.

Evidence and remaining limitations must be updated after each source change; a passing static inventory alone is not feature parity, pixel parity or release readiness.
