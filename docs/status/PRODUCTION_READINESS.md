# Ghost FTP 0.99.56 — Windows/Linux production readiness

**Current classification: Preview candidate, NOT YET CERTIFIED STABLE.** Last public release: v0.99.55 Preview. The mandatory historical 1:1 visual comparison requirement has been retired by product decision; this does not certify untested security or installation behavior.

| Surface | Automated coverage | Stable acceptance still required |
| --- | --- | --- |
| Windows/Linux Rust/Tauri | Build, workspace tests, Clippy, protocol and security regression gates | Installed app, crash recovery, stored credentials, data-preserving upgrades/rollback |
| Windows native | Packaging, Win32 filesystem tests and native capture checks | Authenticode verification, installer/portable smoke, Windows compatibility |
| Linux native | AppImage/DEB/RPM builds and Linux tests | Package-signature decision, target distro and GTK/WebKit/Wayland keyring/upgrade acceptance |
| Cloud and remote transfers | Bounded streaming, exclusive per-attempt non-resumable staging, full HTTP response / Content-Length verification for six GET adapters, and protocol fixtures | Authorized live provider/OAuth, interrupted uploads, end-to-end file integrity |
| UI functionality | Frontend tests and genuine native-app capture | Keyboard/screen-reader access, resizing/scaling, working controls, dialogs and error states |
| Release integrity | Exact-source CI, immutable tags, artifact inventory and checksums | Independent signing evidence, upgrade verification and owner-approved stable promotion |

**Release gate:** all exact-commit quality, dead-code, security, Rust/frontend/Go, Windows/Linux native build, FTP/FTPS/SFTP E2E and artifact integrity jobs must succeed. Security scans alone are not a penetration test. A stable release additionally requires the independently verified proofs listed in [Stable release policy](../release/STABLE_RELEASE_POLICY.md).

Do not change `version.json.channel` to `stable` or publish a non-prerelease tag before the mandatory stable checks have been implemented and passed.
