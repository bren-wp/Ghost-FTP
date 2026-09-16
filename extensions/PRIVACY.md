# Ghost FTP Connection Helper privacy contract

Ghost FTP Connection Helper processes connection text inside the extension popup on the user's device. It does **not** send connection data to Ghost FTP servers, analytics providers, advertising systems or any other network service.

It uses **no telemetry**, **no tracking**, no analytics, no advertising, no remote crash reporting, **no remote code**, no cookies, no browser storage, no active-tab access, no browsing history, no host permissions and no background/content script.

The popup keeps values only in its in-memory DOM while it is open. Closing or clearing the popup discards those values. The extension **does not store** pasted or parsed connection data. A supplied password is detected only to display a local warning and is never rendered back into an output field. The generated Safe target excludes URL username/password plus query and fragment data.

When the user explicitly clicks **Open in Ghost FTP**, the extension asks the operating system to open a `ghostftp://connect` URL containing only the validated protocol, host, optional port, optional username and optional remote path. Passwords, private-key passphrases, private keys, source query parameters and source fragments are never included. This operating-system handoff is not a network request by the extension. On supported installed desktop builds, Ghost FTP receives the allowlisted fields in process memory, leaves credential fields empty and does not connect automatically.

The extension **does not read the active tab** and **does not connect to your FTP, FTPS, or SFTP server**. Any actual connection remains the responsibility of the Ghost FTP desktop application after the user reviews or completes the connection fields and initiates it.

The Firefox package declares `browser_specific_settings.gecko.data_collection_permissions.required = ["none"]`. All official Chrome, Edge, Firefox and Opera manifests request an empty `permissions` list and no host permissions. The four packages share the same local runtime and differ only where browser package metadata requires it.
