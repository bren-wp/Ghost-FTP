# Ghost FTP Privacy

Ghost FTP is designed as a local-first file-transfer application for Windows, Linux and Android. Connection profiles, settings and transfer state are stored on the user's device. Ordinary FTP, explicit FTPS and SFTP transfers run between the user's device and the endpoint selected by the user; Ghost FTP does not require a Brendigo-hosted transfer proxy.

## Credentials

Where supported by the desktop build, secrets are stored through operating-system credential facilities or protected references instead of ordinary profile data. Password fields are masked. Android session passwords remain in memory for the active session and are cleared on disconnect or Activity destruction.

Passwords, passphrases, SSH private-key contents, tokens and equivalent secrets must not be written to normal application, notification or crash logs.

## Telemetry

Ghost FTP does not require analytics or telemetry for core operation. The supplied production configuration does not silently enable usage tracking.

If optional diagnostics are introduced in a future version, they must be explicit, documented and separable from core transfer functionality.

## Network activity

User-requested connections necessarily disclose connection information to the selected server and the network infrastructure required to reach it. Desktop update checks contact the configured official Ghost FTP update service and accept installation only through the updater's verification path.

Help & About remains inside the native desktop application. Canonical downloads are published through GitHub Releases; the repository does not contain a separate public website application.

## Local data

Local settings, profile metadata, transfer state and required caches are stored on the user's device. Desktop profile writes use staged/backup recovery paths to reduce corruption risk.

Terminal suggestion history is session-only and credential-looking commands are excluded from in-memory suggestions. Notification-center history is session-only. Startup cleanup removes legacy terminal/notification history keys created by older development builds.

User-facing diagnostics pass through credential redaction for common password, passphrase, token, API-key, authorization, credential-URL and private-key patterns. Redaction is defense in depth and does not replace the rule that secrets must not be included in errors or logs.

## Files and servers

Ghost FTP does not claim ownership of files, server data, connection profiles or credentials processed by the application. Users remain responsible for authorization, server identity verification, backups and the consequences of requested file operations.

## Canonical privacy record

This file is the canonical privacy description for the Ghost FTP software configuration distributed from this repository. Hosted services, if introduced separately in the future, require their own service-specific privacy terms.
