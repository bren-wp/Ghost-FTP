# Ghost FTP by Brendigo — Desktop privacy

**Product:** Ghost FTP Preview for Windows and Linux. **Description updated:** 21 November 2026.

Ghost FTP is a local-first desktop file-transfer client. Basic FTP, explicit FTPS and SFTP do not require a Ghost FTP user account or a Brendigo-operated file-transfer proxy. The app handles files between the user's device and the endpoint they select. This description covers the desktop application; websites, cloud services and endpoint operators have their own notices.

## Locally processed information
The client may retain connection profile metadata (such as host, port and protocol), local settings, selected paths, transfer queue/history, recovery records and application caches. Where supported, saved credentials are held through OS credential facilities or protected references, rather than ordinary profile fields. Passwords, SSH private-key contents and tokens should not appear in normal UI logs. OS credential storage, backups, roaming profiles and uninstall behaviour depend on the actual system.

Users can change or delete saved profiles and transfer history through supported app controls; OS-managed secrets and independent backup copies may need separate removal. No blanket local retention period is claimed because it varies by feature and installed version.

## External connections
User-requested file transfers reveal connection metadata and content to the selected server/provider and network intermediaries as required by the chosen protocol. **Plain FTP does not encrypt traffic**; prefer validated FTPS or SFTP for sensitive operations. Optional cloud OAuth and similar login flows communicate with the selected identity provider. Signed file URLs and authorisation tokens are secrets.

Optional update checks contact the configured Ghost FTP update endpoint. Downloading a release through GitHub invokes GitHub's separate service and privacy practices. Clicking support links opens a browser. Core desktop transfers do not require mandatory usage analytics or a Brendigo cloud account.

## Responsibility and inquiries
Ghost FTP does not claim ownership of customer files, profiles or credentials. The server operator, user's organisation and third-party providers are responsible for their respective systems, lawful access, retention, access logs and rights requests. If users voluntarily send information to Brendigo support, that is a separate interaction subject to the relevant service notice.

For product privacy questions, use the [official contact page](https://ghostftp.com/hr/contact). Never send credentials, signing keys, bearer URLs or private file contents in public support tickets. The legal identity, lawful basis, contact details and GDPR Article 13/14 information for separately hosted services must be supplied by the entity actually operating those services; this source file does not invent them.

See [Security](../../SECURITY.md), [Licence](../../LICENSE.txt) and [Third-party notices](THIRD_PARTY_NOTICES.md).
