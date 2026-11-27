# Ghost FTP — Windows/Linux Stable package inventory

Only Windows and Linux desktop clients are maintained in the current release train. Other historically released packages remain immutable archives, not new distribution targets.

| Platform | Expected packages |
| --- | --- |
| Windows x64 | Portable EXE, NSIS Setup EXE, MSI, ZIP bundle, versioned Update EXE (from 0.99.92) |
| Linux x86_64 | ELF binary, AppImage, DEB, RPM, tar.gz, versioned Update DEB and RPM (from 0.99.92) |

The exact-head GitHub Actions build and release workflow must confirm artifacts are nonempty, match the version and source SHA and appear in the SHA-256 manifest. A checksum is **not** an OS publisher signature. Tauri signed updater assets may be generated only with configured signing keys and must never be published partially.

Retired standalone Server/Demo/CLI release history is preserved and must not be rewritten. Installing this client does not delete historic services, saved profiles, files or OS credentials. Plain FTP remains unencrypted.

Manual update aliases are full installers, not delta patches or new executable wrappers. Use the same installer family as the existing installation; MSI upgrades use the original MSI. AppImage automatic updates use the separate signed native updater contract. Installed-device upgrade/rollback acceptance remains independent of package verification.
