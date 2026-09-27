# Ghost FTP Installer QA

## Production packaging

Windows production packaging is generated from the authoritative Ghost FTP desktop build and publishes:

- `GhostFTP-Windows-x64-Portable-v<version>.exe`
- `GhostFTP-Windows-x64-Setup-v<version>.exe`

The current production Windows packaging path generates an NSIS Setup bundle. The active workflow does not publish MSI packages.

The Setup contract requires:

- Ghost FTP installer and uninstaller icons;
- Ghost FTP header/sidebar branding artwork;
- `EULA.txt` as the NSIS licence page source;
- English and Croatian installer language resources;
- current-user installation by default, avoiding unnecessary elevation;
- Ghost FTP Start Menu grouping;
- CI validation of the branding/EULA configuration and bitmap dimensions;
- a real silent install/uninstall smoke test of the produced Setup executable.

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
6. Upgrade from the previous published version.
7. Reinstall / repair-equivalent behavior.
8. Cancel during setup.
9. Rollback after failed replacement.
10. Apps & Features metadata.
11. Uninstall.
12. User-data handling after uninstall.
13. Portable executable startup.
14. Installed and portable custom-titlebar behavior.

Status: **production installer builds are CI-generated; full Windows lifecycle acceptance remains a FINAL gate.**
