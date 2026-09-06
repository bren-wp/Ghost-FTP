# Changelog

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

Detailed older release engineering history is intentionally retained in [`docs/RELEASE-HISTORY.md`](docs/RELEASE-HISTORY.md) and in repository Git history. Historical version/platform claims describe the source state at that time and do not override the current Ghost FTP 1.0.0 Stable Windows/Linux contract.
