# Ghost FTP desktop updates

Ghost FTP is maintained for Windows and Linux desktop. Manual Windows EXE/MSI and Linux DEB/RPM installers are distributed through the [Ghost FTP Update Center](https://update.ghostftp.com/) once the verified web root is deployed. The source repository retains the full release inventory and release notes.

## Signed in-app updates

Where a signed updater package is published, verify its update signature, HTTPS metadata and version progression before applying it. Reject altered signatures, downgrades and untrusted update endpoints. A lack of signed update offers is **not** evidence that no newer manual installer exists.

## Manual updates

Download the matching OS package and `SHA256SUMS.txt` from the same deployed update-center version, verify their SHA-256, finish running transfers, back up profiles securely, then upgrade without deleting application data or credential-vault records. Check saved profiles, host keys, protected credentials and uploads/downloads after launch.

Windows portable/NSIS/MSI and Linux AppImage/DEB/RPM upgrades have different OS integration and uninstall rules; verify the matching package.

See [Installation](INSTALLATION.md) and the [release process](../release/PROCESS.md).

The desktop now checks the live native transfer list and waiting queue before invoking the signed installer and again before relaunch. Queued, running, paused or unknown transfer states and failed snapshots postpone the action. Local scheduled retries are deferred while installation is in progress or awaiting restart. Before signed installation, the Rust transfer manager now also acquires an exclusive admission guard covering file/directory preparation, retry and worker registration. The guard remains held until process exit, including an uncertain installation failure. A canceled or aborted worker may still perform blocking cleanup, so in-app installation is conservatively refused until a fresh process. Remote editor sessions, their upload tasks and atomic local writes now hold shared maintenance admission. Backup export/import and changes to folder-sync configuration also participate; configured enabled pairs and staged backup restores prevent installation. Aborted background sync workers require a fresh process before updating. Independent virtual filesystem callbacks and other non-manager operations still require acceptance testing; arbitrary direct plugin calls remain outside this guard. Installed-device rollback, recovery and preservation acceptance remain outstanding.

The native update path records a credential-free source/target intent in the existing SQLite database before installation. If the record cannot be written, installation is refused. On a subsequent launch, a version below the recorded target produces a recovery notice; reaching the target is not proof of data integrity. The journal does not run installers, restore databases or authorize downgrade. An interrupted or uncertain installation still requires manual inspection and a verified backup. Atomic binary rollback remains unimplemented.

The native update intent is reserved from general settings IPC writes, deletes and bulk imports. Failed preparation releases the native guard before installation. No unencrypted backup or automatic restore was added.

Manual and bridge Mirror synchronization hold shared native maintenance admission until their operation returns. Destination deletions now require confirmed successful completion of all planned copies; failed, skipped, canceled, missing or timed-out rows withhold deletions. A delete error is returned, not silently reported as a successful sync. This does not make a whole Mirror plan transactional: earlier successful copies or deletes are not rolled back. Server archive downloads use a private temporary directory and retain the remote archive if completion is unconfirmed; failed or interrupted archives may require manual remote cleanup.

The WebView no longer has direct updater-plugin permissions. The separate Rust native_update module resolves a fresh signed offer for the selected version, downloads/verifies before acquiring exclusive maintenance, records the attempt and installs through the configured Tauri plugin. Preparation/download failures do not retain admission; uncertain installation failures do. The signed endpoint and embedded key are unchanged. This is not a custom package-manager rollback implementation.

Before native installation, the recovery journal is committed under SQLite FULL durability, preceded by a quick integrity check and followed by a FULL WAL checkpoint. Incomplete checkpoints, corruption or disabled durability prevent the installer from running. The check is not an encrypted backup and does not restore prior binaries or prove data preservation across an installed update. A persisted intent may remain after a checkpoint failure; it indicates an attempted preparation, not that the installer ran.
