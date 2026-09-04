# GhostFTP Privacy

GhostFTP is designed to operate without telemetry or tracking.

## What GhostFTP does not do

- No telemetry or analytics.
- No advertising SDKs.
- No crash-report upload.
- No fingerprinting or user profiling.
- No automatic update checks.
- No background requests to ghostftp.com, brendigo.com, GitHub, or any other service.
- No cloud synchronization of saved servers or settings.

## Network behavior

GhostFTP opens network connections only when the user explicitly connects to an FTP/FTPS server or manually opens a website link from the About window. Demo mode is entirely local and generates no network traffic.

## Local data

Profiles and settings are stored locally. In installed mode they are stored under the current user's local application-data directory. Portable builds store data next to the executable in a `Data` directory when possible.

Passwords are never stored unless **Remember password** is enabled. When enabled, the password is encrypted using Windows DPAPI and can only be decrypted by the same Windows user context.

Author: Brendigo  
Project: https://ghostftp.com  
Author website: https://brendigo.com
