# Ghost FTP 0.99.38 — Windows/Linux reliability and security regression audit

**Date:** 10 November 2026. **Scope:** exact-source preview build, release scripts, transfer-history UI and accompanying automated tests.

## Confirmed root causes

- The failed one-time release recovery PR #269 contained the *already published* v0.99.37 version, so its version immutability gate correctly rejected another same-version PR. PR #269 was closed without merging; its recovery job had already correctly published the preceding edition.
- Subsequent release source passed build/protocol/security CI, but `.github/scripts/publish-release.sh` attempted to hash a directory in `dist/*`, failing after the 21 assets had been uploaded. Both release loops now exclude directories and symlinks, retaining exact file-only checksum/inventory policy.
- Transfer Center time filters previously included bogus future and non-finite timestamps in the last-hour/day windows; they now bound valid timestamps at the supplied local snapshot clock.

## Verified by tests / required to verify

- Source-level deterministic timestamp tests include exactly-on-boundary, just-before, future timestamps, NaN and infinity. All-history view is preserved.
- The release-governance test checks both file-selection guards and simulates a package, a staging directory and a symlink.
- PR CI must pass: document/version sync, TypeScript checks, Rust tests/Clippy, Go/security/dead-code audits, FTP/FTPS/SFTP E2E, Windows security, Linux and Windows installers, macOS builds and final package hashes.
- Production runtime E2E against external servers, real cloud accounts, live large transfers, app upgrades and full 59-screen 1:1 pixel comparison are pending separately. Zero known scanner candidates is not equivalent to proof of no vulnerabilities or dead code.

## Marketing assets

README retains real Windows WebView2 and Linux WebKit screenshots with recorded original source SHA and QA provenance. No generated imagery was added.
