# Browser extension privacy contract

Ghost FTP Connection Helper processes connection text only inside the extension popup on the user's device.

It does not transmit connection targets, hostnames, usernames, passwords, paths, or parsed output. It does not use telemetry, analytics, advertising, remote crash reporting, remote code, cookies, browser storage, active-tab access, browsing history, or host permissions.

The popup keeps values only in its in-memory DOM while it is open. Closing or clearing the popup discards those values. A supplied password is detected only to display a local warning and is never rendered back into any output. The generated safe target excludes all URL user information as well as query and fragment data.

The extension does not connect to FTP, FTPS, or SFTP servers. Connections remain the responsibility of the Ghost FTP application or another client explicitly chosen by the user.
