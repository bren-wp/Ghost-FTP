# Ghost FTP Build Status — 20 September 2026

## Authoritative status

**Ghost FTP 2.1.1 RC10 — repository, branding, visual-parity and native-release hardening. NOT FINAL.**

RC10 continues the existing Ghost FTP codebase. It does not replace the project and it does not use screenshot backgrounds as application UI.

## Repository architecture

Product-facing source paths now use GhostFTP naming:

- `ghostftp-desktop/` — production desktop application.
- `ghostftp-web/` — product website.
- `ghostftp-runtime/` — developer/compatibility runtime tooling.
- `ghostftp-installer/` — developer/setup compatibility tooling.
- `docs/` — documentation, audits, QA evidence and release notes.

The nested `ghostftp-desktop/src-tauri/` directory is retained only because the native desktop framework expects that convention. It is not used as a product-facing name.

## RC10 source improvements

- Reorganized top-level source directories around GhostFTP names without changing their internal behavior.
- Updated CI paths and release packaging to the new source layout.
- Standardized release filenames as `GhostFTP-version-platform-architecture-role`.
- Rewrote the main README as a product/marketing page with logo, visual references, build badges, platform targets and documentation navigation.
- Added a project-status document, feature matrix and recommended roadmap.
- Removed duplicated reference-dialog width rules and made New Connection/File Properties reference heights explicit at the canonical desktop size.
- Kept the 1290×852 canonical desktop geometry and adaptive smaller-window rules.
- Kept real React/Rust controls; reference images are QA specifications only.
- Removed an unused Rust service constant and preserved strict Clippy/warnings-as-errors quality checks.
- Kept corrupted profile/database recovery and OS-protected credential paths from prior RC passes.
- Kept browser-shell compatibility executables excluded from end-user releases.

## RC10 visual contract

Canonical main-window geometry:

- titlebar: 51 px
- application menu: 42 px
- Quick Connect row: 50 px
- toolbar: 62 px
- Sites rail: 216 px
- file workspace: 416 px
- transfer/log band: 191 px
- status bar: 40 px

Reference dialog envelopes:

- New Connection: 752 × 628 px at the canonical desktop viewport.
- File Properties: 530 × 770 px at the canonical desktop viewport.

Responsive rules keep key controls available by compacting or using local scrolling rather than deliberately hiding core actions.

## RC10 automated gates

The current RC10 release workflow requires a successful **GhostFTP Quality Audit** and **GhostFTP Desktop Build** on the same main commit.

Quality audit covers:

- TypeScript typecheck and production frontend build.
- Go tests/vet for GhostFTP developer tooling.
- JavaScript syntax checks.
- Rust formatting.
- Rust workspace check.
- Rust workspace tests.
- Clippy with warnings denied.
- legacy/demo-branding scan.

Desktop build targets:

- Windows x64 portable EXE.
- Windows x64 NSIS Setup EXE.
- Linux x86-64 native executable.
- Linux AppImage.
- Linux DEB.
- Linux RPM.

## RC10 build evidence

This document is updated after the current-main RC10 workflows complete. Until then:

- GhostFTP Quality Audit: **PENDING**
- Windows desktop build: **PENDING**
- Windows Setup: **PENDING**
- Linux desktop build: **PENDING**
- AppImage/DEB/RPM: **PENDING**
- GitHub RC10 pre-release: **PENDING**

## Gates still blocking FINAL

FINAL still requires evidence from actual target systems:

1. Windows 10/11 frameless-titlebar screenshots with no extra browser/origin/native caption strip.
2. Native pixel comparison against all approved visual references.
3. Final responsive screenshots at the required viewport sizes.
4. Windows clean install, upgrade and uninstall acceptance.
5. Real FTP end-to-end connection and file-operation matrix.
6. Real FTPS explicit/implicit TLS connection and transfer matrix.
7. Real SFTP password/key/host-key verification and transfer matrix.
8. Interrupted transfer, reconnect and resume/failure-path testing.

These items are not marked passed without evidence.
