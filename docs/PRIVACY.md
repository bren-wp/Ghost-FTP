# Privacy

ByFTP contains no advertising, analytics SDK, application telemetry or mandatory cloud account. Connection details are used only to contact the server selected by the user.

## Desktop

Saved profiles and settings remain in the user's local application-data directory. Windows saved credentials use DPAPI. Production Go build gates keep Go telemetry disabled. External helper processes receive only the minimized environment required for the operation.

## Android

The Android client does not persist connection passwords or SSH secrets. It uses Android document-provider URIs for user-selected upload/download files and does not request broad device-storage access.

Cloud-backup and device-transfer extraction rules explicitly exclude the application's root, file, database, shared-preference and external app-data domains. This prevents future local state from silently entering Android backup/migration flows.

The Android manifest disables generic cleartext traffic for platform-aware network stacks. ByFTP does not include Firebase Analytics, advertising SDKs, a project-controlled runtime API or a ByFTP backend service.

Active/pending network clients are closed when the Activity is destroyed and late main-thread callbacks are ignored. Pending download-picker paths are cleared after every picker result, disconnect and Activity destruction so stale local UI state does not retain an unintended remote target.

Release APK generation does not change the privacy model: the debug and unsigned release APKs are built from the same audited source, and neither introduces a ByFTP cloud service or telemetry endpoint.

The product does not send usage events to the project repository or support endpoint.
