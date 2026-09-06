# Changelog

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

Detailed older release engineering history is intentionally retained in [`docs/RELEASE-HISTORY.md`](docs/RELEASE-HISTORY.md) and in repository Git history. Historical version/platform claims describe the source state at that time and do not override the current Ghost FTP 1.1.1 Stable Windows/Linux contract.
