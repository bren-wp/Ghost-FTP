# Ghost FTP Build Status — 20 September 2026

## Authoritative status

**Ghost FTP 2.1.1 RC12 — native Ghost FTP desktop release candidate. NOT FINAL.**

RC12 continues the existing Ghost FTP source and does not replace the project with a new implementation or screenshot-driven runtime.

## Repository and build organization

- Production desktop source: `ghostftp-desktop/`
- Developer runtime tooling: `tools/ghostftp-runtime/`
- Installer support tooling: `tools/ghostftp-installer/`
- Website source: `website/`
- Update templates: `updates/`
- Native build workflow: `.github/workflows/ghostftp-build.yml`
- Quality workflow: `.github/workflows/ghostftp-quality.yml`
- Release workflow: `.github/workflows/ghostftp-release.yml`

A small number of internal framework-required names remain inside `ghostftp-desktop/` because renaming them would create unnecessary build risk.

## RC12 source improvements

- Reorganized the repository around GhostFTP-branded top-level source paths.
- Standardized release filenames by platform, architecture, role and version.
- Rebuilt the main README as a product/marketing landing page with local logo, icons and screenshots.
- Added dedicated naming and project-status documentation.
- Corrected historical `Ghost FTP` / `GhostFTP` managed PATH alias handling without removing spaces from unrelated Windows paths.
- Removed the browser-host GUI from the production release path.
- Preserved the custom frameless Ghost FTP desktop window and canonical 1290×852 reference geometry.
- Preserved adaptive smaller-window behavior down to the documented minimum target.
- Corrected transfer terminal-state actions and active queue filtering.
- Added Retry All for failed transfers.
- Added semantic progress controls and reduced-motion handling.
- Exposed browser-layout, hidden-file, remote-preview, speed-limit, download-folder and editor settings.
- Hardened corrupt profile/database startup recovery.
- Kept profile secrets outside ordinary profile JSON and OS-keychain-backed where supported.
- Kept the 14-language selector and canonical dictionary coverage model.
- Improved website local icon use and loading-timer cleanup.

## Quality gates

The RC10 quality workflow requires:

- Go tests and `go vet` for GhostFTP runtime/installer tooling;
- JavaScript syntax checks;
- website markup/asset policy checks;
- TypeScript typecheck;
- production frontend build;
- Rust formatting;
- Rust workspace check;
- Rust workspace tests;
- Clippy with warnings denied;
- legacy/demo-branding rejection.

An earlier RC9 quality run correctly exposed PATH-integration failures, and a later RC9 run narrowed the remaining gate to two CLI Clippy findings. RC12 retains the corrected PATH logic and CLI lint fixes. Current publication remains gated on a fully successful quality run for the exact RC12 source revision.

## Native RC12 build targets

### Windows x64

- native portable EXE;
- NSIS Setup EXE.

### Linux x86-64

- native executable;
- AppImage;
- DEB;
- RPM.

MSI is intentionally omitted from prerelease builds because the current MSI version path rejects prerelease identifiers such as `rc.12`.

## Public release artifact policy

Only the native Ghost FTP desktop build is accepted as the end-user GUI release.

The old compatibility browser shell that displayed a visible `127.0.0.1` origin/address strip is not an accepted production artifact.

## Gates still blocking FINAL

FINAL remains blocked until documented evidence exists for:

1. Real FTP end-to-end transfers.
2. Real FTPS end-to-end transfers and certificate failures.
3. Real SFTP password/private-key transfers and host-key behavior.
4. Windows 10/11 titlebar screenshots from installed and portable builds.
5. Windows clean install / upgrade / reinstall / uninstall acceptance.
6. Final responsive and pixel comparison at every required reference size.
7. Production signing decision/validation.

Build and packaging success alone does not convert RC10 into FINAL.


## RC12 fidelity/stability pass

The RC12 branch additionally hardens menu stacking, reference dialog geometry, external-link handling and explicit notification permission flow. Real FTP/FTPS/SFTP workflow evidence exists for an earlier RC12 head, but release status must be reevaluated on the final RC12 commit; native screenshot and installer lifecycle acceptance remain separate gates.
