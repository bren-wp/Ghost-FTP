# Ghost FTP — Recommended Next Work

This is the prioritized improvement backlog after Ghost FTP 2.1.1 RC10. Items here are recommendations, not claims of completed functionality.

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

- Per-transfer priority controls in the main queue.
- Scheduled transfers with persistent schedules.
- Optional automatic clear of successful transfers after a configurable delay.
- Transfer-history export.
- Per-server bandwidth limits in addition to the global throttle.
- More accurate rolling transfer speed instead of lifetime-average speed.
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
- Duplicate profile detection.
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

## Website improvements

Recommended:

- Replace the current loading-style landing surface with the full product/download site once release distribution is stable.
- Add structured data for SoftwareApplication.
- Add release/download metadata and checksums.
- Add product screenshots from real native RC/FINAL builds, not design references.
- Add localized legal pages and accessibility statement.
- Add lightweight performance budget CI.

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
