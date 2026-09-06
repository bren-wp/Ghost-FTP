# Changelog

## 1.0.0 - 2026-09-06 Stable

### First stable release

- Promoted the maintained Windows/Linux product line from the historical 0.x Beta channel to **Ghost FTP 1.0.0 Stable**.
- Stable GitHub Releases are normal releases with `draft=false` and `prerelease=false` using the immutable `ghostftp-v1.0.0` tag contract.
- The maintained production workflow is now **stable-only**: it rejects `MAJOR=0` rather than creating any new prerelease.
- Historical 0.x prereleases remain immutable release provenance and are not deleted or rewritten.

### Stability and transfer correctness

- Consolidated the 0.2.x connection/transfer stabilization work into the production baseline.
- Preserved connection-generation guards so stale callbacks/work cannot silently attach to a later session.
- Preserved deterministic transfer state, truthful progress/speed/ETA reporting, cancellation/retry handling and remote/local cleanup behavior.
- Preserved upload-source snapshot and remote commit/revalidation protections where the transport supports them.
- Preserved symlink/reparse-aware local filesystem hardening and bounded recursive operations.
- Kept connection timeout configuration wired to real Windows and Linux connection attempts with validated bounds.
- Preserved keyboard-first local/remote file workflow, sorting/navigation protections and native resize/minimize/maximize behavior.

### Privacy and diagnostics

- Promoted privacy-safe connection diagnostics to the stable contract for FTP, FTPS and SFTP.
- User-facing failures remain categorized without intentionally reproducing passwords, SFTP key passphrases, private-key paths, protected profile payloads or raw child-process diagnostics.
- Production CI/release workflows explicitly execute and verify `go telemetry off`.
- Saved profiles remain local; saved-secret protection remains opt-in and platform-local.
- Release/package tooling uses explicit artifact allow-lists and does not receive runtime FTP/SFTP credentials.

### Security

- Preserved explicit FTP/FTPS/SFTP transport selection with no silent secure-to-plain downgrade.
- Preserved FTPS certificate/hostname validation and SFTP host-key fingerprint trust.
- Preserved validated host/port/path/private-key inputs, process/tool discovery boundaries and path-containment checks.
- Stable distribution continues to block committed private signing material and runs dedicated security/privacy/dependency/repository audits.
- Production Authenticode is used only when a real protected signing identity is configured; configured signatures must verify successfully.
- When no production signing identity is available, the Release is now truthfully recorded as `WINDOWS_AUTHENTICODE=unsigned` with `WINDOWS_TRUST_MODE=sha256+github-release-provenance` instead of failing publication or inventing a publisher identity.
- Development/self-signed Authenticode remains a CI pipeline smoke test only and is never represented as BRENDIGO LTD production identity.

### Windows quality

- Retained the polished native Windows dual-pane workstation, dark chrome, DPI-aware layout, Site Manager and keyboard-first file workflow.
- Retained batched resize relayout and reduced erase/redraw paths that address visible flicker.
- Retained secure architecture-aware OpenSSH discovery for x86 Windows SFTP.
- Setup remains transactional/rollback-oriented with integrated uninstall registration; Portable remains a separate no-install mode.
- Windows release metadata now makes signed versus unsigned trust state machine-readable for administrators and mirrors.

### Linux quality

- Retained the native Linux X11/XWayland-compatible UI backed by the same typed Engine.
- Retained the 24-language shared catalog and validated connection-timeout behavior.
- Retained idle redraw suppression so the full workspace is not needlessly repainted when transfer state has not changed.
- Production DEB packages remain available for amd64, arm64 and i386.

### GitHub Releases

- Preserved the verified **9 platform artifacts / 12 public files** GitHub Release contract.
- Release publication now always enforces stable `draft=false` and `prerelease=false` state for maintained versions.
- Existing tags remain immutable; publication refuses to move an existing `ghostftp-vX.Y.Z` tag to different source.
- The workflow verifies current `main` before publication and again during delayed remote verification.
- Remote Release verification now downloads the published `SHA256.txt` and compares it byte-for-byte with the locally generated manifest.
- Reruns can repair the same exact release/tag asset set while preserving immutable source provenance.

### GitHub Packages

- Publishes the verified release directory to `ghcr.io/bren-wp/ghost-ftp:1.0.0` as an OCI **distribution bundle**, not a runtime container.
- Publishes stable aliases `1.0`, `1` and `latest` while keeping the full semantic version as the recommended automation tag.
- Builds the package from `FROM scratch`, copies only `/ghostftp-release/` and disables Docker build networking.
- GitHub Package publication now occurs **after** the GitHub Release has passed remote verification.
- Docker authentication uses a temporary credential directory and `--password-stdin`, followed by logout/cleanup.
- After push the workflow removes its local image, pulls `:1.0.0` back from GHCR, verifies source/version/revision OCI labels and compares embedded `SHA256.txt` plus `BUILD-METADATA.txt` byte-for-byte with the Release assembly.
- The successful package gate records the immutable registry digest with `PACKAGE_READBACK=PASS`.

### Documentation and license

- Reworked README and the maintained documentation set around the stable-only publication policy and explicit cryptographic trust state.
- Updated installation, signing, security, testing, versioning, Releases, Packages, verification and roadmap documentation.
- Updated the proprietary/source-available license to **Version 1.3**, clarifying official Releases/GHCR distribution and the difference between integrity provenance and an actual publisher signature.
- Preserved the Windows/Linux-only application scope and 24-language local catalog.

### Verification

The 1.0.0 release candidate must pass the exact production gate before publication:

- `go test -race ./...`;
- `go vet ./...`;
- formatting checks;
- brand/repository/platform/desktop/dependency/version/localization/security/privacy/docs/release audits;
- Python regression suite;
- Windows x64/x86 production Setup + Portable build;
- Linux amd64/arm64/i386 production package build;
- configured production Authenticode verification when signing credentials are present;
- truthful explicit unsigned metadata when they are absent;
- GitHub Release asset/state/SHA read-back;
- GitHub Package push, fresh pull and embedded metadata/SHA read-back.

## 0.2.1 - 2026-09-06 Beta

### Windows visual quality and connection reliability

- Improved Windows visual quality, dark native surfaces, Site Manager button consistency and resize flicker behavior.
- Connected the persisted connection timeout to actual Windows/Linux connection attempts.
- Added cancellable Windows connect behavior and architecture-aware x86 OpenSSH discovery.

### Transfers and diagnostics

- Added truthful runtime transfer bytes, progress, average speed and ETA reporting without fabricating unsupported upload percentages.
- Added privacy-safe transport error classification for DNS, timeout/refused, TLS, FTP reply/auth/data-channel and SFTP host-key/auth/key/path failures.
- Kept raw curl/OpenSSH details, secret-like values and private-key paths out of the localized user-facing error boundary.

### Linux parity

- Expanded Linux localization through the shared 24-language registry.
- Reduced unnecessary idle redraw work while keeping the shared transfer/security Engine.

## 0.2.0 - 2026-09-06 Beta

- Consolidated the active application contract around native Windows and Linux clients sharing one FTP/FTPS/SFTP engine.
- Removed retired Web/PWA/mobile/macOS application surfaces from the active product tree.
- Strengthened Setup/Portable behavior, Linux parity, localization, security/privacy audits and the 9-artifact/12-file release contract.
- Kept English as default/fallback across the maintained 24-language registry.

## 0.1.1 - 2026-09-06 Beta

- Added the maintained Linux graphical frontend with shared Engine behavior, queue controls, settings/profile parity and SFTP password/key/passphrase support.
- Added Linux package/runtime smoke coverage and visual parity work while preserving dependency-light native presentation.

## 0.1.0 - 2026-09-06 Beta baseline

- Established the public pre-1.0 Beta line, Windows/Linux dual-pane workstation, local profile protection, transfer queue, native Linux frontend and semantic release discipline.
- Established root `VERSION` as the canonical Windows/Linux package/release version source.

## Historical engineering history

Detailed older engineering/release history is intentionally retained in [`docs/RELEASE-HISTORY.md`](docs/RELEASE-HISTORY.md) and repository Git history. Historical version/platform claims describe the source state at that time and do not override the current Ghost FTP 1.0.0 Stable Windows/Linux contract.
