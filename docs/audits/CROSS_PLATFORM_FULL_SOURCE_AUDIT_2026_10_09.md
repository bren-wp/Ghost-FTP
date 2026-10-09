# Ghost FTP — cross-platform dead-code, performance, security and privacy review

Date: 9 October 2026. Branch: `audit/0.30.20-cross-platform-hardening` (unreleased changes against verified public v0.30.19). This audit separates **covered by executable checks**, **changed with regressions**, and **still requiring manual verification**.

## Source inventory

- Windows/Linux desktop React TypeScript modules: **86**, plus **18** shared `file-ui` TS modules.
- Tauri/Rust + crates: **85** Rust sources.
- Android: **5** Kotlin production/test files.
- macOS: **15** Swift production/test files.
- CLI/installer compatibility tooling: **11** Go files.
- Script entrypoints: **32** MJS/shell paths in the prior release source tree; new screenshot workflow/script add to this inventory. Counts are file inventory, not all independently runtime-executed paths.

## Executable dead-code / static coverage

| Area | Existing authoritative audit | Audited limitations |
| --- | --- | --- |
| Windows & Linux | `npm run check:dead`, `tsc --noEmit`, Rust orphan module reachability, Clippy, `cargo check` / tests, npm audit, signed update integrity contract | Dynamic plugin/reflection callbacks cannot be deleted from grep absence; CSS visual selectors require screen inspection |
| Android | Private Kotlin symbol scan, Gradle compile/lint/unit + instrumented click smoke, locale-resource parity, guarded protocol actions | Reachability does not prove every real remote server or 16 Android concept images pixel-aligned |
| macOS | Private Swift declaration scan, `swift test`, release build, resource and transport-safety contract | FTPS/SFTP transport is deliberately unavailable; no supplied macOS visual mockups or notarized artifact |
| Shared Rust / Go / tooling | Source module graph, all-target Rust build/test, Go test/vet, Node/shell operational script syntax/reference audit | Remote protocol/hardware fault injection outside available E2E environments still needs manual tests |

**Dead-code decision:** no unrelated files or legacy migration hooks have been deleted without import/module graph and build evidence. This prevents erasing legitimate compatibility paths. The project already has a release-gated module reachability scanner and Kotlin/Swift private-symbol scanners; the CI logs, not manual declarations, decide whether they pass on this PR.

## Changes made

- Android privacy/security: fail-closed bound for unusually large provided secrets; mask credentials/token data before clipping visible error strings; precompile the redaction regex patterns for fewer allocations. Unit tests cover boundary and oversized payloads.
- macOS security: merged profile import cap enforced before mutating current or stored data; regression protects existing profile set.
- macOS privacy: history accepts only 180-character, control-stripped basename instead of persisting a complete local/remote path; round-trip test.
- Desktop Windows/Linux performance: status clock suspends its one-second update loop while the window is hidden, immediately refreshes when visible. User-facing clock and live transfer flow remain available.
- README image provenance: seven **real installed Windows release** UI captures overwrite only the existing image bytes, with identical `README.md` source and pinned SHA256s. 75-image concepts are not disguised as actual application UI.

## Residual work and blocked acceptance

1. Full **1:1** signoff remains pending installed screenshots at 1290×852, other desktop viewport sizes, Android 480×960, keyboard navigation and click-through tests for every real control. All 75 concept slots are still marked `pending`; the concept ZIP contains no macOS references.
2. Cross-platform features cannot truthfully be treated as identical where macOS FTPS/SFTP are disabled pending certificate/SSH host-key identity verification and notarization.
3. This audit does **not** claim independent cryptographic review, penetration testing, full profiler baselines, removal of historical user-data already persisted by old versions, or personal-device acceptance for every screen.
4. Before merging, verify all six workflows at the **same latest commit**, on merged main, plus genuine packaged artifacts; never re-target an existing release tag.

## Reproducible verification

```bash
cd ghostftp-desktop
npm ci
npm run check:dead
npm run check:ui
npm run check:i18n
npm run check
npm run build
cd ..
bash android/scripts/check-android-contract.sh
bash macos/scripts/check-macos-contract.sh
swift test --package-path macos
```

Android release lint/build/instrumentation and native Windows/Linux packaging/protocol E2E are verified by the associated GitHub Actions workflows; local macOS Swift tools are required for manual reproduction.
