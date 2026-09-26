# Ghost FTP Uninstall

Ghost FTP does not require a separate `uninstall.exe`.

On Windows, an installed compatibility build registers an Apps & Features uninstall command that invokes the installed `GhostFTP.exe --uninstall`. That command removes Ghost FTP shortcuts and its uninstall registration, then removes the installation directory after the running process exits. The implementation derives the installed directory from the running installed executable when safe, so a custom installation folder can be removed correctly.

A portable Windows build is removed by closing Ghost FTP and deleting the portable executable and any explicitly created user data the user no longer wants.

For DEB installations, use the distribution package manager, for example `apt remove ghostftp`. Native RPM builds should be removed with the relevant RPM/DNF/Zypper flow. An AppImage is removed by deleting the AppImage and, if created, its optional desktop-integration files.

User-created files on remote servers are never removed by uninstalling the application. Profile/settings data should be retained or removed according to the user's explicit choice and the packaging policy of the final native release.
