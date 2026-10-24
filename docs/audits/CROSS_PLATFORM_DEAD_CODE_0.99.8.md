# Ghost FTP 0.99.8 — cross-platform source-level dead-code inventory, corrections and acceptance

Previous published release: **v0.99.7**. Source train: **0.99.8**. Target publication is gated by the exact-source GitHub Actions builds.

## Confirmed fixes
- Reconcile the Android Files action strip with the reference: the live 480×960 CI capture had its bottom **More** caption clipped while the other four buttons were legible. Equalize padding and add real TextView baseline assertions across six instrumented phone/tablet viewports.
- Reject batch renaming when numeric placeholders overflow across a batch. Protect existing source files from case-insensitive and Unicode-normalized destination collisions before starting any FTP/SFTP operation.
- Fix Rust dead-code scan false negatives: `#[cfg(test)] mod` previously suppressed all production symbols after the test module. Scan precisely within its lexical brace span; add nested/attribute regressions.
- Extend TypeScript scan to unexported arrow declarations and Go scan to receiver methods. All findings are **review candidates only**, never automatic deletion.

## Cross-platform dead-code baseline
Published v0.99.7 at main commit `d7a24f1cb4e89ed2f2cd080e595ed6f528257bbc` has a successful exact-source full audit artifact: 730 files, 117 TS/TSX, 87 Rust, 16 production Kotlin, 20 Kotlin tests, 54 Android resources, 16 Swift and 9 Swift tests, 12 Go, 68 operational scripts. The previous scanner reported zero orphaned Kotlin/Swift modules, Android resources, operational scripts and symbol candidates; the improved auditor checks formerly missed classes of symbols.

## Gates and open limitations
Require successful Android Gradle lint/JVM/Android 16 emulator, Rust cargo/Clippy, Swift build/test, Go build/test/vet, TypeScript build/dead-code graph, security audit, FTP/FTPS/SFTP E2E, and signed Preview inventory. **All 16 Android screens are not certified 1:1**: no invented profiles, server files, cloud authorizations or synthetic transfer metrics may be inserted for design resemblance. Production certification and real-device crash reproduction remain open.
