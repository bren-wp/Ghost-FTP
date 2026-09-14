# Ghost FTP Connection Helper privacy contract

Ghost FTP Connection Helper processes connection text only inside the extension popup on the user's device.

It does **not** transmit connection targets, hostnames, usernames, passwords, paths or parsed output. It uses **no telemetry**, **no tracking**, no analytics, no advertising, no remote crash reporting, **no remote code**, no cookies, no browser storage, no active-tab access, no browsing history, no host permissions and no background/content script.

The popup keeps values only in its in-memory DOM while it is open. Closing or clearing the popup discards those values. The extension **does not store** pasted or parsed connection data. A supplied password is detected only to display a local warning and is never rendered back into an output field. The generated Safe target excludes URL username/password plus query and fragment data.

The extension **does not read the active tab** and **does not connect to your FTP, FTPS, or SFTP server**. Connections remain the responsibility of a Ghost FTP application or another client explicitly chosen by the user.

The Firefox package declares `browser_specific_settings.gecko.data_collection_permissions.required = ["none"]`. All official Chrome, Edge, Firefox and Opera manifests request an empty `permissions` list and no host permissions. The four packages share the same local runtime and differ only where browser package metadata requires it.
