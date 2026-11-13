# Install Ghost FTP

Download from the [official Ghost FTP Releases index](https://github.com/bren-wp/Ghost-FTP/releases), choose an actually published version and download its matching `GhostFTP-v<VERSION>-SHA256SUMS.txt` manifest. **Current source: 0.99.57 Preview; previous published: v0.99.56.** A source version is not automatically available as a release.

## Windows 10/11 — x64

| Download filename | Purpose |
| --- | --- |
| `GhostFTP-Windows-x64-Portable-v<VERSION>.exe` | Portable application without a guided installer |
| `GhostFTP-Windows-x64-Setup-v<VERSION>.exe` | Normal installation with the branded Setup wizard |
| `GhostFTP-Windows-x64-Setup-v<VERSION>.msi` | MSI installer for Windows deployment |
| `GhostFTP-Windows-x64-v<VERSION>.zip` | Bundle containing the Windows distribution files |

Choose either Setup EXE or MSI for a normal installation. On Windows, compare the downloaded file's SHA-256 against the matching line in the release manifest:

```powershell
Get-FileHash -Algorithm SHA256 -LiteralPath '.\GhostFTP-Windows-x64-Setup-v<VERSION>.exe'
```

## Linux — x86_64 / amd64

| Download filename | Intended use |
| --- | --- |
| `GhostFTP-Linux-x86_64-v<VERSION>.AppImage` | Portable desktop package on compatible Linux systems |
| `GhostFTP-Linux-amd64-v<VERSION>.deb` | Debian/Ubuntu-based packages |
| `GhostFTP-Linux-x86_64-v<VERSION>.rpm` | Compatible Fedora/openSUSE/RHEL-based packages |
| `GhostFTP-Linux-x86_64-v<VERSION>` | Standalone native Linux executable |
| `GhostFTP-Linux-x86_64-v<VERSION>.tar.gz` | Portable archive for manual extraction |

Run `sha256sum DOWNLOADED_FILENAME` and compare with the checksum manifest for that version. If all manifest-listed files are in the same directory, you can also use `sha256sum -c GhostFTP-v<VERSION>-SHA256SUMS.txt`.

Linux AppImage and native packages require compatible GTK/WebKitGTK 4.1, glibc and other system runtime components; AppImage compatibility is not universal. An unlocked Secret Service provider and `secret-tool` are required for protected Linux secret storage; there is no silent plaintext fallback. See the [Linux package matrix](../distribution/LINUX_PACKAGE_MATRIX.md).

## Updates and safety

Finish transfers and make an encrypted backup of important profiles before upgrading. Close the application, install the new version under the same OS user and verify saved connections and files. Do not delete app-data directories when uninstalling.

SHA-256 proves downloaded bytes match the release manifest, **not** that the publisher has digitally signed those bytes. Unsigned packages can trigger OS security prompts; do not disable operating-system protection globally. In-app automatic updates require separately authenticated updater signatures, while manual releases remain available in GitHub Releases.

[Release downloads](https://github.com/bren-wp/Ghost-FTP/releases) · [Uninstall](UNINSTALL.md) · [Updates](UPDATES.md) · [Support](SUPPORT.md)
