# Ghost FTP desktop updates

Ghost FTP is maintained for Windows, Linux and macOS desktop. Installers and checked release notes are available from [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases).

## Signed in-app updates

Where a signed updater package is published, verify its update signature, HTTPS metadata and version progression before applying it. Reject altered signatures, downgrades and untrusted update endpoints. A lack of signed update offers is **not** evidence that no newer manual installer exists.

## Manual updates

Download both the exact OS package and its `GhostFTP-v<version>-SHA256SUMS.txt` from the same release, verify their SHA-256, finish running transfers, back up profiles securely, then upgrade without deleting application data or credential-vault records. Check saved profiles, host keys, protected credentials and uploads/downloads after launch.

Windows portable/NSIS/MSI and Linux AppImage/DEB/RPM upgrades have different OS integration and uninstall rules; verify the matching package. macOS Preview builds require separate platform checks and are not equivalent to notarized Developer ID releases.

See [Installation](INSTALLATION.md) and the [release process](../release/PROCESS.md).
