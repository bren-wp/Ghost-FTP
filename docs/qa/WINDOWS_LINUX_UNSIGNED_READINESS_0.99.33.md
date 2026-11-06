# Windows/Linux unsigned desktop acceptance — 0.99.33

**Goal:** reliable and directly installable Windows/Linux clients without mandatory commercial code signing, while preserving verified OS identity, credentials, backup and auto-updater trust.

| Area | Windows x64 acceptance | Linux x86_64 acceptance |
| --- | --- | --- |
| Distribution | Setup NSIS, MSI, portable executable | Debian/Ubuntu-family DEB, RPM family, AppImage/ELF on compatible glibc hosts |
| Native installation | Install, launch genuine Tauri window, NSIS/MSI uninstall and reinstall | DEB and AppImage launch smoke; RPM metadata validated; distro/device runtime testing additionally required |
| Data retention | Profiles, encrypted backup and Windows Credential Manager across restarts/upgrades | Profiles, encrypted backup and unlocked Secret Service across restarts/upgrades |
| Transfer integrity | FTP plaintext warning, FTPS trusted TLS, SFTP pinned SSH host key; interrupted transfer/retry | Same; test GNOME/KDE/Wayland and representative distro systems |
| Manual upgrades | Exact GitHub Releases link, SHA-256, no unsigned in-app install | Same; distro-specific package selection |
| Automatic upgrades | Only publisher-signed artifacts; downgrade prevention | Only publisher-signed artifacts; downgrade prevention |
| CI release gate | All Windows install/uninstall, typecheck, Rust and protocol tests pass | Linux DEB/AppImage native smoke, RPM package checks and real WebKit screenshots |
| Outstanding | Real Windows 10/11 long-running network tests, SmartScreen prompts, backup/restore migration | Fedora/openSUSE/RHEL hardware, locked vault, mixed Wayland desktop environments |

**v0.99.32 baseline:** all seven source-head GitHub Actions workflows passed before that version was merged; its GitHub Release contains actual Windows and Linux package assets. **v0.99.33:** proposed changes and regression tests require a new exact-head CI verdict. Lack of an installer signature does not prevent user-initiated installation; it does reduce publisher identity guarantees. No unconditional production-stable or exact 1:1 pixel certification is implied.

Related: [installer instructions](../guides/INSTALLATION.md), [Linux package matrix](../distribution/LINUX_PACKAGE_MATRIX.md), [public releases](https://github.com/bren-wp/Ghost-FTP/releases).
