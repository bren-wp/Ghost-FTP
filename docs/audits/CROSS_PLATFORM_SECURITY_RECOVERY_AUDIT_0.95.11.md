# Ghost FTP 0.95.11 — Cross-platform security, stability, dead-code and reference acceptance audit

Date: 13 October 2026. Development PR: #145 (Draft). Baseline: merged main `c8daf78da7cd74a621a17ac32f3c082c17679204`. Published version remains **v0.95.10**. Changes here are **not** part of an existing release.

## Source and design inventory

A recursive Git tree inventory of the baseline contains **712 tree entries** (including directories), among them **85 Rust**, **51 TSX**, **56 TS**, **23 Swift**, **18 Kotlin** and **11 Go** source files. These are inventory counts, **not proof that every statement in each module has been manually audited**.

The user-provided `GhostFTP-Premium-Brand-UI(6).zip` contains **75 concept reference screens**:
- Windows: **30** designs, **1290 × 852**.
- Linux: **29** designs, **1290 × 852**.
- Android: **16** designs, **480 × 960**.
- macOS: **0** direct concept images. Align with the shared premium brand but do not claim literal macOS screenshot parity.

See `docs/qa/PREMIUM_REFERENCE_75.json` and `docs/qa/premium-75-screen-contract.json` for all 75 individual identifiers and linked implementation surfaces. The ZIP contains static concepts and illustrative fake file names/data; never embed those mockups as interactive full-screen UI.

The design uses #070E1A / #0B1E36 / #132D52, #38ABFF accents, #EAF6FF text and #26DC8F/#FF7284 status cues, with Inter/Segoe UI/Noto Sans/system fallbacks. Using these colors alone is not evidence of 1:1 parity. Each screen requires installed-build pixel captures, proportional layout, scaling, keyboard and click-through checks.

## Verified source findings and corrective changes in this PR

| Priority | Platform | Finding | Correction | Evidence needed to close |
| --- | --- | --- | --- | --- |
| P0 data loss | Windows/Linux desktop Rust core | When primary `profiles.json` is corrupt, loading the backup as in-memory data then writing could rotate the corrupt primary over the only known-good backup. When both copies failed, a later save could replace the original data with an empty inventory. | Read-only recovery state refuses upsert/reorder/delete; private uniquely named preservation of corrupted original and backup; valid backup remains readable. | New Rust regression tests and exact-SHA CI; independent profile restore drills |
| P1 confidentiality | Windows/Linux desktop Rust core | Recovery snapshots previously used potentially colliding second-resolution filenames and default permissions on Unix. | Separate UUID-based snapshots, exclusive create and 0600 mode; preserve both corrupt files independently. | Unix permissions assertions, Windows filesystem QA |
| P1 staged-write integrity | Windows/Linux desktop Rust core | Predictable `profiles.json.tmp` with truncating open could follow stale files or symlinks. | UUID-based temporary name, exclusive create, no truncation of an existing target. | Rust collision regression and write-interruption tests |
| P1 host identity | Windows/Linux SSH/SFTP `known_hosts` | Host text containing newlines, commas, control characters, wildcard syntax or a zero port could be written as trusted OpenSSH host patterns or additional records. | Shared validation before check/append/replace; reject ambiguous/pattern-based host fields; propagate parent-directory creation/permission errors. | Host injection and literal IPv4/IPv6/DNS unit tests, real SFTP fingerprint E2E |
| P2 compatibility | All platforms | Cross-platform feature surfaces and design references differ; source builds do not certify equal feature behavior. | Maintain an explicit status matrix, do not label unsupported functionality as complete. | Real installations, keyboard, assistive technology and actual server/provider flows |

**Recovery limitation:** A corrupt primary (even with a healthy backup) now opens read-only until the owner restores the original profile metadata. This intentionally favors preserving the sole known-good copy over silent data loss. A safe, separately verified in-app recovery flow would improve usability.

## CI and dead-code auditing

Required exact-final-SHA workflows:
1. Ghost FTP quality: TypeScript typecheck/build, frontend UX contracts, dead-code checks, Rust tests/Clippy and auxiliary compatibility tools.
2. Ghost FTP native build: genuine Windows NSIS/MSI plus Linux DEB/RPM/AppImage, artifact checks and package/launch smoke where configured.
3. Validate Windows hardening: helper process, runtime security and Windows-specific tests.
4. Ghost FTP protocol E2E: real FTP/explicit FTPS/SFTP roundtrips.
5. Ghost FTP macOS: Swift tests, release build and preview artifact verification.
6. Ghost FTP Android: lint, JVM unit tests, APK/AAB build, API-35 emulator UI tests and non-debuggable preview install smoke.

CI gives scoped executable coverage; it is **not a full proof of no dead code**, vulnerability absence or a 1:1 visual match. Before removal, cross-check public APIs, dynamic Tauri command names, Swift/Kotlin callbacks, shell invocation, reflection, persistence migrations and platform conditionals. Preserve any code whose reachability is not understood. Compare traces of bundled packages, not only repository scans.

## Open issue acceptance — do not close prematurely

### #127: Android SAF partial output and cancellation (P0)
The current implementation stages HTTP content in private cache, limits transfers to 1 GiB, checks provider SHA-256 where available, rereads the SAF output, attempts to delete a newly created failed destination and exposes a cancellation action. This **does not** establish provider-atomic output, lifecycle-stable cleanup after process death or Android 8–15/device coverage. Test permission revocation, zero/partial writes, 1 GiB, timeout, storage exhaustion and the inability to delete a provider-managed URI. ETag is not generally an object checksum. Keep issue open until all acceptance tests are evidenced.

### #128: Account OAuth browsing for macOS/Android (P1)
A signed PUT/GET cloud object URL is **not** Google Drive, OneDrive, Box or Dropbox account browsing. Completing this issue needs system-browser Authorization Code + PKCE/state, scope minimization, protected token stores, listing/pagination, safe operations, refresh/401/403/429/5xx handling, sign-out/revocation, test accounts and physical-device QA. Current signed-object UI does not meet the account-OAuth acceptance criteria. Keep issue open.

## Platform limitation / release decision

| Platform | Priority | Evidence from CI/source | Unverified or blocked |
| --- | --- | --- | --- |
| Windows | 1 | Shared Tauri UI, Rust FTP/FTPS/SFTP, Windows bundling and hardening workflow | Every premium control, 30-screen installed 1:1 captures, signing on target machines and malicious filesystem injection QA |
| Linux | 1 | Shared Tauri UI, Rust backend, AppImage/DEB/RPM workflow | 29-screen genuine capture comparison, distro differences, file permissions and package lifecycle on additional distributions |
| macOS | 1 | SwiftUI workspace, real FTP and macOS Swift/build workflow | Explicit FTPS/SFTP engines still blocked; 0 direct macOS mockups, Developer ID/notarization and actual device layout |
| Android | 2 | Kotlin FTP/FTPS/SFTP, Gradle/emu CI, signed URL transfer with bounded staging | 16-screen 1:1, Android 8–15 physical device QA, SAF issue #127, OAuth issue #128, production upgrade signing |

**No release/issue closure authorization:** All six workflows must pass on the exact final PR commit; every package must be inspected and actual user-visible interactions verified. If any blocker remains, keep the PR as Draft and preserve the immutable v0.95.10 tag and artifacts. Do not change version numbers or create a public release merely to satisfy a cadence.
