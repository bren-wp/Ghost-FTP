# Changelog

## 1.1.7 - 2026-09-09 Stable

### Native Windows UI and UX

- Fixed helper-window lifecycle so closing Prompt, Settings, Diagnostics, About, Language or DecisionCard surfaces never terminates the main application message loop.
- Unified application-owned modal windows around one DPI-aware owner/modal keyboard contract with Tab, Shift+Tab, Enter and Escape behavior.
- Replaced mixed stock Confirm/Info/Error surfaces with Ghost FTP DecisionCard windows while retaining stock Windows dialogs only as creation-failure fallback.
- Added adaptive DecisionCard heading/body sizing for longer localized and security-sensitive text instead of clipping content into a fixed body area.
- Unified the Windows and Linux desktop Light/Dark palette contract; Classic Light uses restrained neutral surfaces instead of a pure-white workspace.
- Consolidated Windows Settings into one application-owned settings window with inline validation, locale-safe label geometry and standard keyboard navigation.
- Kept Windows icons local to the OS using Segoe Fluent Icons with Segoe MDL2 Assets fallback; no external icon/font runtime was added.

### Localization and branding

- Localized OK, Cancel, Yes and No through the active runtime language across the maintained 24-language contract.
- Localized Save Profile privacy/security decisions without changing credential retain/remove or automatic-clear semantics.
- Localized native SSH private-key and local-folder picker titles/filter labels at the time each picker opens.
- Made **Ghost FTP** the only active runtime, package, support and release-documentation identity; author/publisher identity is intentionally confined to the application **About** surface.
- Removed publisher branding from Debian/RPM package metadata and generic runtime metadata while preserving the About author attribution.

### Linux distribution and verification

- Added canonical generic Linux `.tar.gz` archives for `amd64`, `arm64` and `i386` alongside matching DEBs.
- Production verifies archive structure and byte-for-byte `ghostftp` executable parity between each generic tar.gz and its matching DEB.
- Expanded the canonical release shape to **12 platform artifacts / 15 public files**.
- Added maintained supplemental distro-specific Debian, Ubuntu, Fedora RPM and distro-neutral Portable package construction.
- Added real package-manager install/remove plus installed-GUI smoke verification on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64.
- Preserved build/metadata/extraction/byte-parity verification for arm64/aarch64 and i386/i686 without overstating native install coverage.

### Security, stability and performance

- Preserved explicit FTPS as the fresh secure default with certificate/hostname validation and no silent secure-to-plain downgrade.
- Preserved SFTP host-key verification/pinning, protected-secret ownership/lifetime rules and privacy-safe diagnostics.
- Preserved rooted local filesystem/transfer activation, path-containment, staged commit/rollback and connection-generation safeguards.
- Kept transfer refresh event-driven so idle timer activity does not rebuild the queue when no transfer events exist.
- Added no telemetry, analytics, advertising, tracking SDK, hidden network service, remote UI runtime or external Go module dependency.

### Release contract

The 1.1.7 Stable candidate requires:

- `go test -race ./...`, `go vet ./...` and Go formatting checks;
- brand/repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- complete Python regression suite;
- Windows x64/x86 Setup + Portable production builds and release-artifact verification;
- Linux amd64/arm64/i386 DEB + tar.gz production builds and byte-parity verification;
- supplemental distro-package build/parity CI;
- Debian 13, Ubuntu 26.04 LTS and Fedora 44 native lifecycle/GUI smoke;
- Authenticode policy verification and private-key pipeline smoke test;
- authentic Windows x64 Portable Main/Site Manager/Settings/About screenshots from the exact final release-prep head;
- exact-head PR CI and exact post-merge `main` CI;
- exact-main `release/ghostftp-v1.1.7` branch validation;
- immutable `ghostftp-v1.1.7` Stable GitHub Release with `prerelease=false`, exact 15-file asset set and GHCR `1.1.7` distribution-bundle publication/read-back.

## 1.1.6 - 2026-09-08 Stable

- Hardened local filesystem operations against pathname swaps using opened-root boundaries.
- Bound SFTP host-key fingerprints directly to scanned key material and hardened remote cleanup proof.
- Preserved the historical 9-platform-artifact / 12-public-file release shape.

## 1.1.5 - 2026-09-08 Stable

- Aligned public product destinations around `https://ghostftp.com` and strengthened localization/documentation contracts.
- Preserved secure protocol, credential-lifetime and release verification behavior.

## 1.1.4 - 2026-09-08 Stable

- Improved Settings correctness, language persistence, idle transfer-refresh performance and local icon handling.
- Strengthened deterministic regression coverage and release-quality gates.

## 1.1.3 - 2026-09-07 Stable

- Strengthened rooted transfer activation, remote tree validation, SFTP literal-name handling and Site Manager duplication safety.
- Made release publication dispatch-only and bound to an exact verified `main` revision.

## 1.1.2 - 2026-09-07 Stable

- Consolidated native Windows navigation, modal appearance consistency, localization quality and authentic UI evidence.
- Preserved Windows/Linux production build and security verification.

## 1.1.1 - 2026-09-07 Stable

- Established Classic Light as the fresh fallback, explicit FTPS/21 as the fresh connection default and explicit credential-persistence consent.
- Added real loopback FTP manager/protocol regression coverage and no-downgrade verification.

## 1.1.0 - 2026-09-07 Stable

- Added the maintained Classic Light/Dark appearance model and strengthened SFTP protected-secret ownership.
- Preserved the Stable Windows/Linux distribution contract.

## 1.0.0 - 2026-09-06 Stable

- Promoted Ghost FTP to the first Stable Windows/Linux release with verified FTP/FTPS/SFTP, local profiles, native frontends and release read-back.

## 0.2.1 - 2026-09-06 Beta

- Improved Windows visual quality, connection timeout behavior, Linux localization and privacy-safe diagnostics.

## 0.2.0 - 2026-09-06 Beta

- Consolidated the Windows/Linux dual-pane application contract and packaging/security baseline.

## 0.1.x - 2026-09-06 Beta line

- Established the pre-1.0 Windows/Linux FTP/FTPS/SFTP product baseline.

## Historical engineering history

Detailed engineering history remains in [`docs/RELEASE-HISTORY.md`](docs/RELEASE-HISTORY.md), immutable Git tags/releases and repository history. Historical release identities are never rewritten by later maintenance work.
