> **Ghost FTP 0.99.54 Preview development. Verified published predecessor: v0.99.53. Object-store download response-length validation and staging durability; signed-native, live cloud and all 59 reference-screen acceptance remain pending.**

> **Current development (10 November 2026): Ghost FTP 0.99.42 Preview.** Preserve splash-free Windows/Linux startup and repair the authenticated v0.99.41 release filename/Preview metadata drift without replacing historical bytes. Previous published edition: v0.99.41. Production signing, live providers and 1:1 reference acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.41 Preview.** Windows/Linux open directly in Files without an artificial splash; native QA validates seven real workspaces. Scheduled transfer checks also respond to window wake and connectivity restoration. Previous published edition: v0.99.40 (release verification required). Signing, live providers and 1:1 reference acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.40 Preview.** Repair late GitHub prerelease metadata drift with verified checksums and post-publication regression gates. Public predecessor: v0.99.39. Live-provider, production signing and full pixel-parity acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.39 Preview.** One canonical release asset inventory, exact SHA-256 verification and reliable Preview post-release checks. Public predecessor: v0.99.38. Production signing and 1:1 visual acceptance remain separate.

> **Current development (10 November 2026): Ghost FTP 0.99.38 Preview.** Windows/Linux transfer-history reliability and release safety improvements. Public predecessor: v0.99.37. Production signing and 1:1 visual acceptance remain separate.

> **Current development (9 November 2026): Ghost FTP 0.99.37 Preview.** Windows/Linux DST-safe scheduled transfers, resumable SHA-256-verified release uploads and stricter release gates. Previous published release: v0.99.36. Signed-production status and 1:1 visual parity are not claimed.

> **Current development (9 November 2026): Ghost FTP 0.99.36 Preview.** Windows/Linux/macOS desktop-only product and release train. Security and dead-code regression audits remain mandatory; production and pixel-perfect acceptance require native test evidence. Previously published v0.99.35 remains immutable.

> **Current development (6 November 2026): Ghost FTP 0.99.33 Preview.** Last public edition v0.99.32; Windows/Linux native installers are directly usable without a paid code-signing certificate but need authentic checksums, verified connections and target-device acceptance. Automatic updater remains signed-only; global stable labeling remains gated separately.

> **Current development (6 November 2026): Ghost FTP 0.99.32 Preview.** The preceding published release is v0.99.31. Windows/Linux native file workspace, real connection health, bounded grant imports and resource usage are the focus. Security fixes are source/test changes, not independent penetration-test or full installed-device certification. Historical editions and published assets remain immutable.

> **Authoritative current state (3 November 2026):** Ghost FTP **0.99.31 development Preview**; v0.99.30 is the latest publicly published version. The new cycle audits active Windows/Linux, Android and macOS code, plus maintained Go compatibility tools, security/privacy controls, dead-code reachability and regressions. Older notes retain historical context; no claim of total vulnerability freedom or complete device acceptance.

> **Authoritative current state (2 November 2026):** Ghost FTP **0.99.30 development Preview**; v0.99.29 is the latest publicly published version. Standalone Ghost FTP Server, Demo and CLI are retired from current source and new release packages. Client protocol support remains platform-dependent; security and installed-device validation are not yet certified. Older entries below are historical.

> **Authoritative current state (26 October 2026):** Ghost FTP **0.99.15 Preview**; latest published version **v0.99.14**. Older notes below retain their historical version context.

> Ghost FTP **0.99.4** (release cycle), following published **v0.99.3**. Distribution status is verified from GitHub Releases; unsigned/mobile and notarization limitations remain explicit.

> **Development cycle:** Ghost FTP 0.99.6 Preview. Previous verified public release: v0.99.5. Android responsive reference acceptance is pending.

# Ghost FTP — Recommended Next Work

> Current development: **0.99.4** (premium UI and cloud security), following published **0.99.3**. Historical sections are retained unchanged.

> Earlier 0.94.0/0.93.0 sections remain historical. Current source is **0.99.4** and latest public release **0.99.3**.

This prioritized improvement backlog covers Ghost FTP **0.99.30**. Previous canonical release: **0.99.53**. Items here are recommendations, not claims of completed functionality.

## Release blockers before FINAL

These should be completed before a production FINAL label:

1. **Windows 10 and Windows 11 visual acceptance**
   - Capture the real installed and portable application.
   - Confirm there is no extra Edge/Chromium/native caption above the Ghost FTP custom titlebar.
   - Compare Main, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About against the approved references.

2. **Protocol end-to-end acceptance**
   - FTP upload/download/rename/delete/resume.
   - Explicit and implicit FTPS where supported.
   - SFTP password authentication.
   - SFTP private-key authentication.
   - Host-key mismatch/rejection behavior.
   - TLS certificate failure behavior.
   - Large-file transfer and interrupted-transfer recovery.

3. **Installer lifecycle**
   - Clean install.
   - Upgrade from the previous release candidate.
   - Repair/reinstall behavior.
   - Uninstall from Windows Settings.
   - Verify no separate `uninstall.exe` is introduced if the product policy remains unchanged.
   - Verify user data handling on uninstall.

4. **Responsive/pixel acceptance**
   - 1290×852
   - 1280×800
   - 1024×768
   - 960×720
   - 800×600
   - 768×720
   - 640×720
   - 540×720
   - 480×800

## High-value product improvements

### Transfer intelligence

Recommended:

- Optional automatic clear of successful transfers after a configurable delay.
- Transfer-history export.
- Per-server bandwidth limits in addition to the global throttle.
- Verify-after-transfer checksum option for protocols/backends that can calculate both sides.

### Connection reliability

Recommended:

- Configurable keep-alive interval in the main preferences UI.
- Per-profile retry/reconnect policy.
- Connection health indicator with latency and reconnect status.
- Optional automatic reconnect with a visible countdown.
- Better proxy support if target users need SOCKS/HTTP proxy workflows.

### File workflow

Recommended:

- Multi-rename / batch rename.
- Compare local/remote timestamps and size before transfer.
- Configurable double-click action.
- Safe-delete/trash mode for local files where the OS supports it.
- Remote edit conflict detection if a remote file changes while open locally.
- Better archive preview/extract support where appropriate.

### Site Manager

Recommended:

- Encrypted export/import package for selected sites.
- Android saved-site persistence and eventual duplicate-profile discovery parity, if a secure account-neutral local profile model is implemented. Windows/Linux 0.91.0 and macOS 0.92.0 offer advisory duplicate discovery.
- Connection-profile templates.
- Search by tag, host, username and notes.
- Optional per-profile color/icon.

### UI/UX

Recommended:

- Native Windows 11 snap-layout compatibility testing.
- Full keyboard navigation audit.
- Screen-reader label audit for every toolbar/menu control.
- High-contrast theme acceptance.
- Touch/pen target audit for Windows tablets.
- Persisted column widths in file panes and transfer tables.
- Optional compact toolbar labels.
- User-configurable panel visibility.

### Security

Recommended:

- Signed Windows binaries with a production code-signing certificate.
- Signed Linux repository metadata if an APT/RPM repository is introduced.
- Reproducible-build documentation.
- Dependency vulnerability scanning in CI.
- Secret scanning and SBOM generation.
- Optional encrypted export of settings/profiles with documented recovery rules.

## Repository/engineering improvements

Recommended:

- Keep framework-required internal paths stable unless a rename has a clear benefit and a tested migration.
- Continue using GhostFTP-first public artifact names.
- Add integration-test servers or containers for FTP/FTPS/SFTP CI.
- Add snapshot/visual regression tests for the React surfaces.
- Add cargo/npm dependency update automation with review gates.
- Add release signing and provenance.

## Naming policy

User-facing and release-facing names should use:

- `Ghost FTP` for product copy.
- `GhostFTP` for filenames, packages and technical identifiers.

Framework-specific words such as Tauri should be kept only where technically required by the build ecosystem. Renaming `src-tauri` purely for appearance is not recommended because it increases maintenance/build risk without improving the user experience.


Current development line: 0.99.54. Previous canonical release: 0.99.53. The last published Preview is v0.99.52; see GitHub Releases and `docs/releases/0.99.52.md` for its acceptance limits.
