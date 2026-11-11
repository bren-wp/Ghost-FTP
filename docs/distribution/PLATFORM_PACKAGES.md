# Ghost FTP 0.99.36 — desktop Preview package inventory

Only Windows and Linux desktop clients are maintained in the current release train. Other historically released packages remain immutable archives, not new distribution targets.

| Platform | Expected packages |
| --- | --- |
| Windows x64 | Portable EXE, NSIS Setup EXE, MSI, ZIP bundle |
| Linux x86_64 | ELF binary, AppImage, DEB, RPM, tar.gz |

The exact-head GitHub Actions build and release workflow must confirm artifacts are nonempty, match the version and source SHA and appear in the SHA-256 manifest. A checksum is **not** an OS publisher signature. Tauri signed updater assets may be generated only with configured signing keys and must never be published partially.

Retired standalone Server/Demo/CLI release history is preserved and must not be rewritten. Installing this client does not delete historic services, saved profiles, files or OS credentials. Plain FTP remains unencrypted.
