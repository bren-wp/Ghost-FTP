# Changelog

## 1.1.5 - 2026-09-08 Stable

### Product and publisher identity

- Made **https://ghostftp.com** the canonical Ghost FTP product website across runtime/package documentation surfaces.
- Kept **BRENDIGO LTD** as the developer/publisher and **https://brendigo.com** as the author website, with `https://brendigo.com/kontakt` as the publisher support destination.
- Updated Windows About so the Ghost FTP product destination is shown separately from the BRENDIGO LTD publisher destination.
- Updated Linux DEB metadata to use `Homepage: https://ghostftp.com` and the BRENDIGO LTD Maintainer identity instead of treating the source repository as the product homepage.

### Public UI branding and localization

- Added one public localization boundary that converts the technical `GhostFTP` compatibility identifier to the user-facing **Ghost FTP** product name before localized strings reach the UI.
- Fixed the Settings title-bar branding from `GhostFTP — Settings` to `Ghost FTP — Settings` and applied the same public-brand guarantee to relevant SFTP, disconnect and terminal strings.
- Added regression coverage across all 24 supported languages so public localized strings cannot silently regress to the technical brand identifier.
- Extended the authentic Windows screenshot workflow so hardening branches and localization changes trigger real x64 Portable evidence.

### Cleanup, packaging and documentation integrity

- Removed the stale version-specific `scripts/test_release_1_1_4_desktop_quality.py` file while preserving maintained coverage in the evergreen `scripts/test_desktop_quality.py` suite.
- Corrected stale current-release documentation in Support, Installation and Packages and aligned release documentation with the canonical root `VERSION`.
- Strengthened documentation regression coverage so README, documentation index, Installation, Packages, Support, GitHub Releases and Release Verification must track the current semantic version and release artifact names.
- Preserved the canonical 9-platform-artifact / 12-public-file stable release contract and the dispatch-only canonical release workflow.

### Security, privacy and verification

- Preserved FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret ownership/lifetime handling, path containment, staged transfer rollback, retry/cancel generation binding and secure-protocol no-downgrade behavior.
- Added no external Go module dependency, telemetry, analytics, advertising, tracking SDK, remote UI dependency or hidden product network service.
- Verified the 1.1.5 hardening line with exact-head Core/Windows/Linux CI, production Windows x64/x86 and Linux amd64/arm64/i386 package builds, Authenticode policy smoke tests and authentic Main Workspace, Site Manager, Settings and About screenshots from the real Windows x64 Portable executable.

### Required verification

The 1.1.5 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- dependency/repository/platform/desktop/localization/security/privacy/documentation/release audits;
- full Python regression suite including official-destination and active-doc version contracts;
- Windows x64/x86 Setup + Portable production builds, Setup-x32 alias verification and release artifact verification;
- Linux amd64/arm64/i386 production builds, DEB verification and multiarch packaging contract;
- Authenticode production-policy verification and private-key pipeline smoke test;
- authentic Windows x64 Portable Main/Site Manager/Settings/About capture and visual review on the exact release-prep head;
- exact-head release-prep PR CI;
- post-merge Core/Windows/Linux verification on the exact `main` SHA;
- exact-main `release/ghostftp-v1.1.5` branch validation;
- immutable `ghostftp-v1.1.5` tag, Stable GitHub Release with `prerelease=false`, exact 12-file asset set and GHCR `1.1.5` distribution-bundle publication/read-back.

## 1.1.4 - 2026-09-08 Stable

### Settings correctness and language UX

- Made the Windows language selector close immediately after a language is selected.
- Moved language persistence out of the native UI message-loop path so saving a locale change does not block interaction.
- Serialized Language and Settings writes through one UI-side gate so an older whole-settings snapshot cannot overwrite a newer save.
- Reused canonical backend min/max/default settings constants in the Windows Settings UI instead of maintaining duplicated validation ranges.
- Normalized invalid or out-of-range prompt state to safe canonical defaults before display.
- Avoided rebuilding localization, local/remote lists and layout for ordinary Settings saves unless the persisted language actually changed.
- Removed the second redundant workspace-layout refinement after Windows language selection.

### Performance, local UI assets and cleanup

- Added an idle fast path to Windows transfer polling: when `TransferEvents` returns no new events, the timer does not allocate a selection map, recompute transfer summary/action state or redraw the transfer list.
- Deduplicated native button/icon registration through one canonical helper.
- Kept Windows icons fully local to the operating system: **Segoe Fluent Icons** is preferred where available with **Segoe MDL2 Assets** as compatibility fallback.
- Added no web font, CDN, third-party icon runtime or external Go module dependency.
- Audited repository cleanup candidates and retained active build/platform fallbacks, security regressions, package assets and release contracts instead of deleting files cosmetically.

### Stability and verification

- Stabilized process-tree cancellation regression coverage with a deterministic descendant-ready/post-cancel survival handshake rather than a fixed child-side delay, without weakening production cancellation behavior.
- Added dedicated 1.1.4 desktop-quality regression contracts for language dropdown closure, asynchronous settings persistence, settings-write serialization, canonical settings bounds, idle transfer behavior, local icon policy and language-layout deduplication.
- Preserved FTPS certificate/hostname validation, SFTP host-key verification/pinning, protected-secret ownership/lifetime rules, local root containment, staged transfer rollback, retry/cancel generation binding and explicit secure-protocol behavior.
- Preserved zero telemetry, zero analytics/advertising/tracking and zero hidden product network service.

### Required verification

The 1.1.4 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- dependency/repository/platform/desktop/localization/security/privacy/documentation/release audits;
- full Python regression suite including the 1.1.4 desktop-quality contracts;
- Windows x64/x86 Setup + Portable production builds, Setup-x32 alias verification and release artifact verification;
- Linux amd64/arm64/i386 production builds, DEB verification and multiarch packaging contract;
- Authenticode production-policy verification and private-key pipeline smoke test;
- exact-head release-prep PR CI;
- post-merge Core/Windows/Linux verification on the exact `main` SHA;
- exact-main `release/ghostftp-v1.1.4` branch validation;
- immutable `ghostftp-v1.1.4` tag, Stable GitHub Release, exact 12-file asset set and GHCR distribution-bundle publication/read-back.

## 1.1.3 - 2026-09-07 Stable

### Transfer and filesystem hardening

- Preserved the user-selected download `LocalRoot` through transfer execution into FTP, FTPS and SFTP download implementations.
- Moved final local download staging, activation and rollback to Go `os.Root` operations so the selected root remains the filesystem capability boundary during commit.
- Added randomized root-level staging with cryptographic sentinel and file-identity verification before activation.
- Added late nested symlink/junction/path-swap rejection, late `SkipExisting` revalidation and rollback/backup safeguards around final local activation.
- Hardened remote tree preparation so existing path components must be real directories; symlink and non-directory components are rejected before child uploads proceed.

### SFTP, Windows and Site Manager correctness

- Escaped SFTP batch glob metacharacters and leading-option edge cases so literal remote names are not reinterpreted as patterns or options.
- Extended Windows local-name validation to reserved DOS device names using superscript-number variants such as `COM¹`, `COM²`, `COM³`, `LPT¹`, `LPT²` and `LPT³`.
- Added a safe Site Manager **Duplicate** workflow that creates a new unsaved draft without silently copying a profile ID, password, private-key passphrase or SFTP host-key trust fingerprint.
- Removed stale `QuietUninstallString` registration because the current Windows uninstaller is intentionally interactive; upgrades remove the obsolete value transactionally and retain rollback protection.

### Release discipline and verification

- Removed the legacy publication path where a `VERSION` change pushed to `main` could trigger release publication.
- Kept `release.yml` publication-only and `workflow_dispatch`-only.
- Made `release/ghostftp-vX.Y.Z` created from the exact verified `main` SHA the canonical publication trigger with branch/version equality guards.
- Added/updated regression contracts for release triggering, documentation version alignment, remote directory types, local root propagation, staging identity, late redirect/path-swap behavior and privacy/security audit invariants.
- Preserved zero telemetry, zero advertising/tracking, zero hidden product network service and zero external Go module requirements.

### Required verification

The 1.1.3 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- dependency/repository/platform/desktop/localization/security/privacy/documentation/release audits;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production builds, Setup-x32 alias verification and release artifact verification;
- Linux amd64/arm64/i386 production builds, DEB verification and multiarch packaging contract;
- Authenticode production-policy verification and private-key pipeline smoke test;
- exact-head release-prep PR CI;
- post-merge Core/Windows/Linux verification on the exact `main` SHA;
- exact-main `release/ghostftp-v1.1.3` branch validation;
- immutable `ghostftp-v1.1.3` tag, Stable GitHub Release, exact 12-file asset set and GHCR distribution-bundle publication/read-back.

## 1.1.2 - 2026-09-07 Stable

### Native Windows UI and navigation

- Replaced duplicated application navigation with one **canonical left sidebar** for Language, Site Manager, Settings, Diagnostics and About.
- Removed the retired native top menu and its menu-specific renderer instead of preserving duplicate File/Servers/Transfers/View/Help command surfaces.
- Kept the operational FTP workspace focused on genuine connection, file and transfer actions; no duplicate Connect/Save/Delete/Transfer commands were introduced in navigation chrome.
- Made Prompt, option/language/settings and About surfaces use the shared native Light/Dark-aware dialog shell so the selected appearance does not produce mixed dark-titlebar/white-body windows.
- Replaced runtime About with a dedicated application-owned native information card using **BRENDIGO LTD** publisher metadata, public **Ghost FTP** naming and official Brendigo-only destinations.
- Runtime About contains no GitHub/GitHub Issues destination, WebView or hidden web/network request.
- Increased About heading/body geometry for long localized text and verified the corrected layout through a fresh authentic production screenshot.
- Added adaptive compact-button rendering: wide controls use icon + label, medium controls preserve a full centered label without a decorative icon, and genuinely narrow secondary actions use intentional centered icon-only presentation rather than accidental ellipsis.

### Localization

- Preserved the canonical 24-language registry with English as default/fallback.
- Fixed FTPS explicit/implicit labels so all 24 languages render non-empty mode text; the second half of the language registry can no longer produce `FTPS ()`.
- Added maintained Site Manager and Diagnostics navigation coverage for all 24 languages with fallback regression checks.
- Kept public runtime About branding as **Ghost FTP** while retaining the technical/internal `GhostFTP` identity where appropriate.

### Security, privacy and release evidence

- Added no external Go module dependencies and no telemetry, analytics, advertising, tracking, tracking pixels, external crash-reporting SDK or hidden network service.
- Preserved FTPS certificate/hostname validation and the no-secure-to-plaintext-downgrade rule.
- Preserved SFTP host-key verification/pinning, private-key validation and ownership-aware protected-secret lifetime/transfer semantics.
- Preserved connection-generation binding, safe retry/cancel lifecycle, randomized `.GhostFTP-part-*` staging, destination revalidation, rollback/backup behavior, atomic local replacement and filesystem symlink/junction/reparse/root-containment protections.
- Verified production Windows x64/x86 Setup + Portable and Linux amd64/arm64/i386 builds, release artifact checks, DEB verification and Authenticode pipeline policy checks.
- Generated authentic screenshots from the real production Windows x64 Portable executable for Main Workspace, Site Manager, Settings and About; final publication requires those images to be visually inspected on the exact 1.1.2 source head.

### Required verification

The 1.1.2 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- dependency/repository/platform/desktop/localization/security/privacy/documentation/release audits;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production builds, Setup-x32 alias verification and release artifact verification;
- Linux amd64/arm64/i386 production builds, DEB verification and multiarch packaging contract;
- Authenticode production-policy verification and private-key pipeline smoke test;
- authentic Windows x64 Portable Main/Site Manager/Settings/About screenshot capture and manual visual review;
- exact-head PR mergeability/read-back;
- post-merge Core/Windows/Linux verification on `main`;
- immutable `ghostftp-v1.1.2` tag, Stable GitHub Release, exact 12-file asset set and GHCR distribution-bundle publication/read-back.

## 1.1.1 - 2026-09-07 Stable

### Defaults, UX and privacy

- Made **Classic Light** the actual fresh-install and invalid/missing-state fallback appearance while preserving an explicitly saved Dark preference.
- Made explicit **FTPS on port 21** the fresh/quick-connect protocol on both Windows and Linux; plain FTP remains available only as an explicit compatibility choice and secure protocols never silently downgrade.
- Unified Windows Site Manager and the main Save Profile flow around explicit, localized credential-persistence consent.
- Added privacy-consent copy for all 24 supported UI languages and removed the Site Manager path that could persist newly entered password/private-key passphrase data without the same explicit opt-in used by the main profile flow.
- Kept Settings intentionally compact: no duplicate theme, color, credential or protocol toggles were added.

### Connection and stability verification

- Added a real loopback FTP integration test through the production `remote.Manager.Connect` lifecycle rather than testing only the transport adapter.
- The manager integration coverage verifies successful authentication, the initial remote listing probe, operation admission, connection identity, plaintext-secret redaction from public connection state and clean disconnect behavior.
- Added regression coverage proving a wrong FTP password cannot publish a connected/operational session.
- Added regression coverage proving explicit FTPS refuses a plaintext-only FTP endpoint rather than silently downgrading.
- Retained the existing real FTP protocol lifecycle coverage for list, directory creation, upload, size/list validation, rename, byte-equal download, delete and final listing.

### Security, performance and release discipline

- Preserved SFTP host-key trust, owned-vs-borrowed protected-secret lifetime rules, pending-trust cleanup and private-key snapshot protections from 1.1.0.
- Preserved FTPS certificate/hostname verification, retry-generation/session binding, local symlink/reparse protections, bounded transfer events and atomic staging/rollback behavior.
- Verified that Windows transfer refresh remains event-driven rather than repainting the full queue every timer tick, and retained the badge-only connection-state repaint path to avoid whole-window flicker.
- Continued the zero-telemetry, zero-advertising, zero-tracking and zero-new-external-Go-module contract.
- The published `ghostftp-v1.1.0` and `ghostftp-v1.0.0` tags/releases remain historical and are never rewritten by the 1.1.1 line.

### Required verification

The 1.1.1 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- repository/platform/desktop/dependency/version/localization/security/privacy/docs/release audits;
- full Python regression suite;
- real loopback FTP manager/protocol regression coverage;
- Windows x64/x86 Setup + Portable production builds and artifact verification;
- Linux amd64/arm64/i386 production builds and DEB verification;
- authentic Windows x64 Portable screenshot capture/verification for documentation evidence;
- Authenticode pipeline verification under the configured signing policy;
- GitHub Release, tag and GHCR read-back after publication.

## 1.1.0 - 2026-09-07 Stable

### Appearance and desktop quality

- Added **Classic Light** as a restrained, professional light appearance inspired by traditional two-pane FTP clients while retaining Ghost FTP branding, icons and assets.
- Kept appearance configuration intentionally compact: Dark and Classic Light are one canonical choice rather than duplicated background/accent/control toggles.
- Hardened native Windows appearance handling so title bar, menus, combo boxes, list views, headers and owner-drawn buttons do not mix incompatible dark/light surfaces.
- Added appearance model/config migration and validation with regression coverage for missing or invalid persisted values.
- Updated authentic production UI screenshot evidence from the real Windows x64 Portable build.

### Security and stability

- Added explicit SFTP protected-secret ownership so session-owned Linux broker secrets are forgotten on close while borrowed profile credentials remain valid for reconnect.
- Added constructor failure cleanup for session-owned SFTP password/passphrase blobs.
- Hardened pending SFTP host-key trust credential ownership and cleanup across cancel, expiry, mismatch, replacement, disconnect and abandoned connection setup paths.
- Preserved the credential captured for the confirmed trust attempt instead of allowing a stale re-resolved profile blob to silently take precedence.
- Added Linux regression coverage for owned/borrowed secret lifetime and pending trust cleanup.
- Preserved host-key verification, FTPS certificate validation, path/symlink/reparse protections and no secure-to-plain protocol downgrade.

### Release and documentation

- Bumped the maintained stable line to **Ghost FTP 1.1.0** without moving or rewriting `ghostftp-v1.0.0`.
- Updated README, settings/UI documentation, release history and verification guidance for the 1.1.0 appearance and security contract.
- Continued the 9-platform-artifact / 12-public-file Windows/Linux release contract.
- Production Authenticode remains optional and fail-closed when configured; unsigned Windows builds remain explicitly identified rather than using a self-signed production identity.
- No telemetry, analytics, advertising, tracking, hidden network service or new external Go module dependency was added.

### Required verification

The 1.1.0 stable candidate must pass before publication:

- `go test -race ./...`;
- `go vet ./...`;
- Go formatting checks;
- repository/platform/desktop/dependency/version/localization/security/privacy/docs/release audits;
- full Python regression suite;
- Windows x64/x86 Setup + Portable production builds and artifact verification;
- Linux amd64/arm64/i386 production builds and DEB verification;
- Authenticode pipeline verification under the configured signing policy;
- GitHub Release, tag and GHCR read-back after publication.

## 1.0.0 - 2026-09-06 Stable

### First stable release

- Promoted the maintained Windows/Linux product line from the 0.x Beta channel to **Ghost FTP 1.0.0 Stable**.
- Stable GitHub Releases are normal releases (`prerelease=false`) using the immutable `ghostftp-v1.0.0` tag contract.
- Production Authenticode is optional: configured trusted certificates are verified fail-closed, while releases without a production certificate remain explicitly unsigned in `BUILD-METADATA.txt` rather than generating or pretending to use a trusted publisher key.

### Stability and transfer correctness

- Consolidated the 0.2.x connection/transfer stabilization work into the production baseline.
- Preserved connection-generation guards so stale callbacks/work cannot silently attach to a later session.
- Preserved deterministic transfer state, truthful progress/speed/ETA reporting, cancellation/retry handling and remote/local cleanup behavior.
- Preserved source-snapshot and remote commit/revalidation protections for transfer operations where the transport supports them.
- Preserved symlink/reparse-aware local filesystem hardening and bounded recursive operations.
- Kept connection timeout configuration wired to real Windows and Linux connection attempts with validated bounds.

### Privacy and diagnostics

- Promoted privacy-safe connection diagnostics to the stable contract for FTP, FTPS and SFTP.
- User-facing failures remain categorized without intentionally reproducing passwords, SFTP key passphrases or protected profile secrets.
- Production CI/release workflows explicitly disable Go telemetry.
- Saved profiles remain local; saved-secret protection remains opt-in and platform-local.

### Security

- Preserved explicit FTP/FTPS/SFTP transport selection with no silent secure-to-plain downgrade.
- Preserved FTPS certificate/hostname validation and SFTP host-key fingerprint trust.
- Preserved validated host/port/path/private-key inputs, process/tool discovery boundaries and path-containment checks.
- Stable distribution continues to block committed private signing material and runs dedicated security/privacy/dependency/repository audits.

### Windows quality

- Retained the polished native Windows dual-pane workstation, dark chrome, DPI-aware layout, Site Manager and keyboard-first file workflow.
- Retained batched resize relayout and reduced erase/redraw paths that address visible flicker.
- Retained secure architecture-aware OpenSSH discovery for x86 Windows SFTP.
- Setup remains transactional/rollback-oriented with integrated uninstall registration; Portable remains a separate no-install mode.

### Linux quality

- Retained native Linux X11/XWayland-compatible UI backed by the same typed Engine.
- Retained the 24-language shared catalog and validated connection-timeout behavior.
- Retained idle redraw suppression so the full workspace is not needlessly repainted when transfer state has not changed.
- Production DEB packages remain available for amd64, arm64 and i386.

### Releases and GitHub Packages

- Preserved the verified **9 platform artifacts / 12 public files** GitHub Release contract.
- Fixed release-note generation so it now documents only the active Windows/Linux product scope and no longer emits retired platform/package claims.
- Added stable GitHub Packages publication at `ghcr.io/bren-wp/ghost-ftp:1.0.0`.
- The GitHub Package is a verified OCI **distribution bundle**, not a runtime container, and contains only `/ghostftp-release/` assembled by the release allow-list.
- Added stable aliases `1.0`, `1` and `latest` while keeping the full semantic version as the recommended automation tag.
- Added registry read-back verification and OCI source/version/revision labels.
- Docker networking is disabled while building the release bundle.

### Documentation and license

- Rewrote the active README and documentation set for the 1.0 stable contract.
- Added a dedicated GitHub Packages guide and expanded release verification guidance.
- Updated security, privacy, architecture, installation, versioning, localization, testing, support, dependency and signing documentation.
- Updated the proprietary/source-available license identity to **Ghost FTP / BRENDIGO LTD** and documented official Release/Packages distribution without granting additional redistribution rights.

### Verification

The 1.0.0 release candidate must pass the exact production gate before publication:

- `go test -race ./...`;
- `go vet ./...`;
- formatting checks;
- brand/repository/platform/desktop/dependency/version/localization/security/privacy/docs/release audits;
- Python regression suite;
- Windows x64/x86 production Setup + Portable build;
- Linux amd64/arm64/i386 production package build;
- Authenticode verification when a production certificate is configured;
- explicit `WINDOWS_AUTHENTICODE=unsigned` metadata when no production certificate is configured;
- GitHub Package push/read-back;
- GitHub Release asset/prerelease read-back.

## 0.2.1 - 2026-09-06 Beta

- Improved Windows visual quality, dark native surfaces, Site Manager button consistency and resize flicker behavior.
- Connected the persisted connection timeout to actual Windows/Linux connection attempts.
- Added cancellable Windows connect behavior and architecture-aware x86 OpenSSH discovery.
- Expanded Linux localization and reduced unnecessary idle redraw work.
- Added privacy-safe transport error classification and regression coverage before the 1.0 stabilization pass.

## 0.2.0 - 2026-09-06 Beta

- Consolidated the active application contract around native Windows and Linux clients sharing one FTP/FTPS/SFTP engine.
- Strengthened Setup/Portable, Linux parity, localization, security/privacy audits and the 9-artifact/12-file release contract.

## 0.1.x - 2026-09-06 Beta line

- Established the public pre-1.0 Beta line, Windows/Linux dual-pane workstation, local profile protection, transfer queue, native Linux frontend and versioned release discipline.

## Historical engineering history

Detailed older release engineering history is intentionally retained in [`docs/RELEASE-HISTORY.md`](docs/RELEASE-HISTORY.md) and in repository Git history. Historical version/platform claims describe the source state at that time and do not override the current Ghost FTP 1.1.5 Stable Windows/Linux contract.
