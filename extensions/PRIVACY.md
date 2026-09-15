# Ghost FTP Connection Helper privacy contract

Ghost FTP Connection Helper is a local browser companion for the installed Ghost FTP application. It uses the browser's **Native Messaging** mechanism to send explicit user actions to a local Ghost FTP bridge. The extension does not operate a cloud service, proxy or hidden remote endpoint.

The extension uses **no telemetry**, **no tracking**, no analytics, no advertising, no remote crash reporting, **no remote code**, no cookies, no browser storage, no active-tab access, no browsing history, no host permissions and no content scripts.

The only browser permission requested by official Chrome, Edge, Firefox and Opera manifests is `nativeMessaging`. This is used only to communicate with the locally installed Ghost FTP bridge. The extension **does not read the active tab** or page content.

The extension **does not store credentials in browser storage**. Passwords and private-key passphrases entered in the popup exist only in popup memory while required for the explicit operation. Saved credentials are stored only when the user requests it, and that storage is performed by the native Ghost FTP Engine through its protected profile store. Public profile data returned to the extension does not contain saved plaintext passwords or passphrases.

A pasted FTP/FTPS/SFTP URL is parsed locally. A password embedded in that URL is deliberately discarded before the form is filled. Query and fragment data are not used for a connection.

When the user connects, FTP/FTPS/SFTP protocol traffic is performed by the local Ghost FTP Engine and travels **directly** between the user's device and the server selected by the user. Connection targets, credentials, directory listings and transferred file contents are not sent to Ghost FTP infrastructure.

Local file access is initiated by the user through an operating-system folder picker. The native bridge confines local navigation and transfers to the selected root and rejects traversal or symlink escape attempts. The browser extension itself does not receive unrestricted filesystem access.

For SFTP, first contact requires explicit host-key fingerprint verification. Remembered fingerprints remain bound to the saved endpoint, and a changed fingerprint blocks the connection.

The background relay exists only to bridge popup requests and keep real active transfers connected to the local native process. It polls only the local transfer event stream while transfers are queued or running and closes the native port when the UI is closed and no active work remains.

The Firefox package declares `browser_specific_settings.gecko.data_collection_permissions.required = ["none"]`. All four official browser packages share the same local runtime and differ only where browser package metadata requires it.
