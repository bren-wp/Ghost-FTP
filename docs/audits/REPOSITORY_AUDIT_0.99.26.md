# Ghost FTP 0.99.26 — repository audit and code hardening

**Status:** Source-level audit of the 0.99.25 baseline and a 0.99.26 Preview candidate. All findings below require exact-HEAD CI. The audit is not a penetration test or production certification.

## Inventory and methodology

The tracked Git tree on the baseline contains **868 blobs** (1,003 tree entries), including **84 Rust**, **116 TypeScript/TSX**, **40 Kotlin**, **27 Swift**, **13 Go** and **251 Markdown** files in `docs/` (plus Markdown elsewhere). Methods: source-tree enumeration; conservative reachability/reference inventory; Rust, TS, Kotlin, Swift and Go test/build gates; platform E2E; release/artifact provenance; static trust-bypass and dangerous-pattern searches. Runtime-only/reflection/manifest bindings are not automatically removable dead code.

The repository's `tools/audits/full-dead-code-audit.mjs` and `tools/audits/full-security-code-audit.mjs` produce exhaustive checked-in source inventories and advisory candidates. The CI scripts record findings. Dynamic memory, fuzzing, mobile instrumentation, penetration testing, credential recovery, Apple/Google signing and actual device upgrades are distinct acceptance stages.

## Verified source findings and corrections in this PR

1. **High — local Agent Bridge HTTP request boundary:** previously emitted permissive `Access-Control-Allow-Origin: *`, accepted invalid or duplicated `Content-Length` ambiguously and did not bound the total request-read duration. Restrict browser Origin requests, remove permissive CORS, reject transfer-encoding and ambiguous framing, cap request time and bodies, and add loopback parser regression tests.
2. **Medium — incomplete HTTP bodies:** previously accepted EOF before the declared body length. Reject truncated bodies before routing.
3. **Code hygiene — security scan coverage:** original scanner found only the first occurrence of each dangerous-pattern class per file. Enumerate all matches, add blocking patterns for unsafe localhost CORS, embedded private-key literals and remote script piped to a shell, and require HTTP parser hardening invariants.
4. **Documentation / dead-content — obsolete guides and dead links:** remove only unreferenced historical audit/phase documents superseded by current guidance. Keep publication records, legal notices, protocol references, configuration/branding evidence, platform user guides and archived git history. Replace the outdated documentation index and simplify entry points.
5. **Version integrity:** `v0.99.25` is a published Preview. This work advances all workspace metadata to `v0.99.26`, without overwriting earlier tags or distributable assets.

## Unresolved production/security blockers

- No independent demonstration of hardware-level credential recovery, production Windows Authenticode/Apple notarization, signed Android Play update, installer downgrade/upgrade reliability or all-device UI correctness.
- Native SwiftUI macOS FTPS/SFTP remains deliberately unsupported pending validated implementation.
- A source scanner cannot establish that private helpers are dead when used via reflection, manifests, OS callbacks, build scripts, macros, or dynamic plugin entry points. None are deleted solely on name counts.
- Historical `v0.99.24` public release asset inventory requires separate reconciliation against the originally published 25-artifact record; the live API showed 13 at audit time. Do not silently replace or delete historical artifacts.
- Audit completion requires the exact PR HEAD to pass native Rust, Android, Swift, Go, TypeScript, CI security and FTP/FTPS/SFTP protocol tests; issue publication remains blocked until required checks pass.

## Documentation retention policy

Retain current user guides, legal/security notices, platform/build/release procedures, canonical release notes, reference evidence consumed by CI and a small current audit index. Historical obsolete analyses with no inbound references may be removed from the active tree; commit history preserves provenance. Never delete released asset/tag history or CI-critical resources to reduce documentation counts.

## Post-merge historical release integrity incident (1 November 2026)

The metadata workflow on main SHA `8f055c82a5524c586fd27d60fbe323b9da9251fe` failed because `v0.99.24` exposes **13 assets** in the GitHub Releases API, whereas its published client-only artifact contract requires at least 21 (an earlier release manifest referenced 25). The 13 observable assets retain API-provided IDs, sizes and SHA-256 digest values; this **does not validate the missing assets** and cannot establish who or what removed them. The published metadata still marks the incomplete release as `prerelease=false`.

The reproducible incident snapshot is `/.github/scripts/v0.99.24-observed-assets.json`. CI validates the *exact present subset*, keeps the issue visible with a workflow warning, and refuses modifications or automatic reclassification of `v0.99.24` while its original complete inventory cannot be verified. Any drift from the observed subset is a blocking failure. Its historical tag is not moved, no old assets are overwritten, and the original disappearance remains an open investigation requiring independent original artifact and checksum evidence.

Separately, `v0.99.25` retains all 25 artifacts but was observed with `prerelease=false`. The metadata corrector now verifies its exact source tag and 25 digested assets and restores its proper Preview classification. This does not excuse the separate incident affecting `v0.99.24`.
