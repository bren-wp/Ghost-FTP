# Ghost FTP desktop updates

Ghost FTP is maintained for Windows and Linux desktop. Manual Windows EXE/MSI and Linux DEB/RPM installers are distributed through the [Ghost FTP Update Center](https://update.ghostftp.com/) once the verified web root is deployed. The source repository retains the full release inventory and release notes.

## Signed in-app updates

Where a signed updater package is published, verify its update signature, HTTPS metadata and version progression before applying it. Reject altered signatures, downgrades and untrusted update endpoints. A lack of signed update offers is **not** evidence that no newer manual installer exists.

## Manual updates

Download the matching OS package and `SHA256SUMS.txt` from the same deployed update-center version, verify their SHA-256, finish running transfers, back up profiles securely, then upgrade without deleting application data or credential-vault records. Check saved profiles, host keys, protected credentials and uploads/downloads after launch.

Windows portable/NSIS/MSI and Linux AppImage/DEB/RPM upgrades have different OS integration and uninstall rules; verify the matching package.

See [Installation](INSTALLATION.md) and the [release process](../release/PROCESS.md).

The desktop now checks the live native transfer list and waiting queue before invoking the signed installer and again before relaunch. Queued, running, paused or unknown transfer states and failed snapshots postpone the action. Local scheduled retries are deferred while installation is in progress or awaiting restart. This is a desktop preflight, not a native global admission barrier: bridge, sync or user operations can race with the snapshot. Do not start new work during an update. Native exclusive admission, installed-device rollback and crash-recovery acceptance remain outstanding; do not infer preservation of all data from a successful preflight.
