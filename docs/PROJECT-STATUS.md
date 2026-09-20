# Ghost FTP Project Status — 2.1.1 RC10

## Current position

Ghost FTP RC10 is the active native desktop release-candidate line. The production GUI is the `ghostftp-desktop/` application. Browser-host compatibility executables are not accepted as production GUI builds.

The approved visual target is the supplied Ghost FTP reference set in `docs/assets/screenshots/`, with a canonical desktop frame of **1290 × 852**.

## Completed and integrated

### Desktop application
- Frameless Ghost FTP application window with custom titlebar.
- Local + remote dual-pane file workflow.
- Quick Connect and saved connection profiles.
- FTP, FTPS and SFTP native protocol paths.
- Site Manager with favorites, folders, tags, bookmarks and recent-server metadata.
- Transfer queue and control surfaces for progress, speed, ETA, pause/resume/cancel/retry paths.
- File properties, SHA-256 where supported and permission/chmod operations where supported.
- Preferences covering language, appearance, transfers, connection behavior, security/privacy, updates and integrations.
- About/update/support surfaces in the same Ghost FTP design system.
- 14 advertised interface languages with aligned canonical key coverage.
- Adaptive layout rules for smaller windows without deliberately removing essential controls.

### Reliability and security
- Corrupt profile metadata recovery instead of startup failure.
- SQLite state quarantine/recovery path.
- OS-protected credential storage where supported.
- Signed updater configuration.
- Explicit unsupported-operation errors instead of fabricated server metadata.
- Legacy/demo branding scan in CI.
- Rust formatting/check/tests/Clippy quality gate.
- TypeScript frontend build gate.
- Go test/vet gate for GhostFTP developer tooling.

### Distribution
- Windows x64 native portable EXE.
- Windows x64 native Setup EXE.
- Linux x86-64 native executable.
- Linux AppImage.
- Linux DEB.
- Linux RPM.
- SHA-256 release manifest.
- Source, desktop-source, web and documentation archives.

### Repository cleanup
- Production application renamed to `ghostftp-desktop/`.
- Web source renamed to `ghostftp-web/`.
- Compatibility/developer runtime renamed to `ghostftp-runtime/`.
- Setup tooling renamed to `ghostftp-installer/`.
- Documentation grouped under `docs/`.
- Product-facing workflows and release artifact names use GhostFTP naming.
- Old browser-shell release path remains excluded from end-user packages.

## Verified in CI

The RC10 quality pipeline is designed to verify:
- frontend dependency install;
- TypeScript typecheck;
- production frontend build;
- Go tests and vet;
- JavaScript syntax;
- Rust formatting;
- Rust workspace check;
- Rust workspace tests;
- Clippy with warnings denied;
- legacy/demo branding scan;
- native Windows and Linux packaging.

Build status is tracked in [build/STATUS.md](build/STATUS.md).

## Still required before FINAL

These items must not be marked complete without target-system evidence:

1. Windows 10/11 screenshot proof that the production executable shows only the Ghost FTP titlebar and no extra browser/origin/native caption strip.
2. Pixel comparison of the real native app against every supplied reference screen.
3. Final responsive acceptance at all required viewport sizes.
4. Clean Windows install, upgrade and uninstall acceptance on actual Windows hosts.
5. Real FTP connection/upload/download/delete/rename tests.
6. Real explicit/implicit FTPS connection and transfer tests.
7. Real SFTP password/key/host-key verification and transfer tests.
8. Reconnect/resume/failure-path testing against deliberately interrupted servers.

## Release rule

A release candidate can be published when source audit and native packaging pass. **FINAL** is reserved for a build that also passes the OS-level visual, installer-lifecycle and protocol end-to-end gates above.
