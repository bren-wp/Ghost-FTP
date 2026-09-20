# Ghost FTP Installer QA — RC12

## Production packaging

Windows production packaging is generated from the authoritative Ghost FTP desktop build and publishes:

- `GhostFTP-Windows-x64-Portable-v2.1.1-RC12.exe`
- `GhostFTP-Windows-x64-Setup-v2.1.1-RC12.exe`

The Setup package is generated as an NSIS bundle. MSI is intentionally omitted for prerelease versions because the prerelease version identifier is not accepted by the MSI/WiX version path used here.

Linux production packaging publishes the native executable, AppImage, DEB and RPM.

## Product policy

- Browser-host compatibility executables are not production GUI releases.
- Windows production builds must open as the Ghost FTP desktop window, not a visible localhost browser shell.
- Installed product metadata must identify Ghost FTP / Brendigo correctly.
- Uninstall behavior must match the documented product policy.

## Windows lifecycle acceptance still required

Before FINAL, test on Windows 10 and Windows 11:

1. Clean install.
2. EULA presentation/acceptance.
3. Installation location and invalid/unwritable paths.
4. Start Menu / desktop shortcut behavior.
5. Launch-after-install.
6. Upgrade from the previous RC.
7. Reinstall / repair-equivalent behavior.
8. Cancel during setup.
9. Rollback after failed replacement.
10. Apps & Features metadata.
11. Uninstall.
12. User-data handling after uninstall.
13. Portable executable startup.
14. Installed and portable custom-titlebar behavior.

Status: **production installer builds are CI-generated; full Windows lifecycle acceptance remains a FINAL gate.**
