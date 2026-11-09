# Ghost FTP 0.99.36 — full maintained-desktop source, security and dead-code audit

**Review date:** 9 November 2026  
**Branch:** `refactor/remove-android-platform-v0.99.36`  
**PR:** [#266](https://github.com/bren-wp/Ghost-FTP/pull/266)  
**Source state:** Preview under exact-SHA CI. This is a combined source-and-regression review, **not** a completed third-party penetration test or full live-provider certification.

## 1. Inventory and retirement

The tracked branch tree contains **677 blobs** at the documented review snapshot, spread across the Windows/Linux Tauri/Rust/React client, SwiftUI macOS client, Go compatibility helpers, update metadata, tests, documentation and release automation. No tracked pathname includes the discontinued mobile-platform name. Existing public tags and historical asset attestations are immutable and should not be rewritten.

Removed active mobile application source, Gradle configuration, resource/manifest, Kotlin/Java, device QA, two workflow jobs, mobile-only brand asset and documentation. Removed its code, version and APK/AAB requirements from quality, release, signing, design reference, localization, package and source-reachability contracts. A permanent source gate (`.github/scripts/check-retired-platform.mjs`) now prohibits the platform's project root, workflow files, release-script restoration and current-product package references.

**Retained targets:** Windows (NSIS/MSI/portable), Linux (binary/AppImage/DEB/RPM), macOS (SwiftUI plus shared-desktop CI) and the Go compatibility/runtime tooling. Desktop source registry: **30 Windows + 29 Linux concept screens**; all are pending genuine runtime visual acceptance. macOS has no supplied approved pixel-exact reference.

## 2. Full dead-code audit

- Conservative source graph: TypeScript frontend reachability, Rust module graph and private functions, Swift `@main` and types, Go entrypoints/methods and operational scripts
- Source-level candidates are **advisory only**. Dynamic Tauri IPC, Swift selectors, macros, test entrypoints, Go build tags and cross-workflow shell invocation can evade textual reference checks
- The old mobile launcher requirement was removed from the audit; retained desktop entrypoints are still required
- Run `node --test tools/audits/full-dead-code-audit.test.mjs`, `node tools/audits/full-dead-code-audit.mjs --json`, `npm run check:dead`, Rust Clippy, Swift and Go tests against exact head SHA
- Do not delete source because a single identifier occurrence or a static heuristic reports it unreachable. Verify compiler + runtime + migration compatibility before removal

## 3. Full static security audit scope

| Boundary | Safeguards / review targets | Remaining acceptance |
| --- | --- | --- |
| FTP / explicit FTPS | URL/path injection rejection, SIZE/ABOR semantics, TLS certificate and host identity, no silent downgrade; byte-zero fallback for unverified prefix resume | Live servers (vsftpd/proftpd/IIS), passive NAT, certificate rotation and disconnect injection |
| SFTP / SSH | Approved host-key policy, known_hosts replacement safety, exact transferred/staged bytes and resumed-prefix comparison | Live trusted/mismatched host keys, concurrent equal-size mutation, multi-GB uploads |
| Ghost FTP Agent | Authenticated bridge, permission/scoped write gates, chunk-acknowledgement verification, bounded buffers, zero-byte upload, delta verification | Daemon restart, concurrency, remote-side mutation under load, permission revocation |
| HTTP / WebDAV | HTTPS validation, blocked auth redirects, bounded responses, path canonicalization, fail-closed remote existence checks | Real WebDAV locks, 412 conditional responses, quota and hostile XML |
| Cloud/object/SaaS | Credential redaction, endpoint validation, bounded uploads, provider-specific adapters and no unapproved token logging | Live OAuth tenants, redirect/proxy/DNS rebinding, refresh/revoke, multipart failure recovery |
| Local file operations | Staged downloads, symlink/reparse point protection, safe path normalization, overwrite backup restoration | Race-injection, power loss, user-file permissions, replace atomicity per filesystem |
| Credential vaults | Windows OS credential manager, Linux Secret Service and macOS Keychain; no plaintext fallback where prohibited | Real locked/unavailable vault, upgrade, session expiry, secret cleanup and cross-user isolation |
| Auto-update / release | Signed Tauri updates, no downgrade, HTTPS endpoint and exact source SHA, immutable public tags, SHA-256 release asset manifest | Installer upgrade/rollback, signers and notarization on real supported OS versions |
| Native UI and IPC | Strict Tauri CSP/capabilities, single main window, bounded local bridge, no shell-all/filesystem-all grants | Accessibility, focus trapping, interaction runs, native OS screenshot comparison |

A source pattern scan cannot establish all external attack paths are safe, and a SHA-256 checksum alone does not authenticate the publisher. Plain FTP is **not encrypted**.

## 4. Cross-platform functional and protocol acceptance

- Windows/Linux: `cargo fmt --check`, `cargo check`, `cargo test`, Clippy, TypeScript checks, React build, runtime startup, file picker, transfer center, cancel/retry and native package smoke tests
- macOS: native Swift builds/tests, Keychain and FTP behaviors, packaging, real install/launch and credentialed transfer checks; **FTPS/SFTP are not yet natively implemented**
- Go: `go test ./...`, `go test -race ./...`, `go vet ./...` where applicable
- FTP/FTPS/SFTP: actual localhost protocol E2E, binary and Unicode paths, large-file boundaries, wrong/missing credentials, abort/resume, disconnected servers and host/certificate trust
- Other cloud providers: per-tenant live integration with scoped test account, revocation, upload/download hash and rate-limit behavior must be independently established
- Visual: native real capture at 1290×852 for each of the 59 Windows/Linux concepts, compared with RGBA pixel accounting and manually reviewed keyboard/accessibility/functional states. Zero-percent deviation is **not** established by source inspection.

## 5. High-priority residual risks

1. macOS native FTP support does not imply FTPS/SFTP support; maintain disabled controls where unimplemented.
2. Equal-length files can still mutate *during* a live transfer without a server-side immutable revision or end-to-end checksum.
3. OAuth/cloud service tests require genuine authorized tenants; static adapter code and mocked responses are insufficient.
4. Signed auto-updates must not silently fall back to unsigned installation.
5. Real Windows/Linux packaging and macOS notarization have not been universally verified on all supported OS environments by source audit.
6. Historical public releases remain accessible and may contain assets from retired products; source retirement does not revoke already-installed software.

## 6. Mandatory exact-SHA completion record

Check [PR #266 Actions](https://github.com/bren-wp/Ghost-FTP/pull/266/checks) and associated artifact reports before merging. Every required job must report **success** at the final head SHA. A new commit invalidates earlier successful CI evidence. Do not claim full security certification, 1:1 visual parity or release publication until native, protocol and package acceptance records exist.

This audit is part of the source tree and will remain explicit about all outstanding manual checks instead of silently marking them passed.
