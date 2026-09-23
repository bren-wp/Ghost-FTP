# Ghost FTP Privacy

Ghost FTP is designed as a desktop file-transfer client. Connection profiles and application settings are stored on the user's device. Files transferred through the native protocol engine are sent between the user's device and the endpoints the user chooses; Ghost FTP does not need a Brendigo-hosted proxy to perform ordinary FTP, FTPS or SFTP transfers.

## Credentials

Where supported by the native build, secrets should be stored through operating-system credential facilities or protected references instead of plain profile JSON. Password fields are masked in the UI. Passwords, SSH private-key contents and equivalent secrets must not be written to normal application or crash logs.

## Telemetry

Analytics and telemetry are disabled by default in the supplied product configuration. The application should not silently enable analytics. If an optional diagnostic or usage feature is introduced later, it must be explicit, documented and separable from core transfer functionality.

## Network activity

User-requested connections necessarily disclose technical connection information to the selected server and the network infrastructure required to reach it. Update checks contact the official Ghost FTP update service at `ghostftp.com`. Links opened from Help/About are restricted to the official Ghost FTP site in the product-facing code audited for this package.

## Local data

Local settings, profile metadata, transfer state and required caches are stored on the user's device. The profile store uses staged writes with a backup/recovery path to reduce corruption risk.

Terminal suggestion history is **session-only** in RC18: entered shell commands are not persisted to WebView storage, and credential-looking commands are excluded even from the in-memory suggestion list. Notification-center history is also session-only. On upgrade, Ghost FTP removes legacy terminal/notification history keys created by older release candidates.

User-facing diagnostic text passes through credential redaction for common password, passphrase, token, API-key, Authorization/Bearer, credential-URL and private-key patterns. This filtering is defense in depth and does not replace the rule that secrets should not be included in errors or logs in the first place.

Users remain responsible for backups of their own files and credentials.

For official privacy information, use **https://ghostftp.com/privacy/**. This project document describes the supplied software configuration and is not a substitute for the public website policy that applies to hosted Ghost FTP services.
