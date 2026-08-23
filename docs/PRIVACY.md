# Privacy

ByFTP contains no advertising, analytics SDK, application telemetry or mandatory cloud account. Connection details are used only to contact the server selected by the user.

Desktop saved profiles and settings remain in the user's local application-data directory. Windows saved credentials use DPAPI. Production Go build gates keep Go telemetry disabled.

## Android

The Android 1.1.0 client does not persist connection passwords or SSH secrets. It uses Android document-provider URIs for user-selected upload/download files and does not request broad device-storage access.

Android cloud-backup and device-transfer extraction rules explicitly exclude the application's root, file, database, shared-preference and external app-data domains. This prevents future local app state from silently entering Android backup/migration flows if additional non-secret settings are introduced later.

The Android manifest disables generic cleartext traffic for platform-aware network stacks. ByFTP does not include Firebase Analytics, advertising SDKs, a project-controlled runtime API or a ByFTP backend service.

Active/pending network clients are closed when the Activity is destroyed and late main-thread callbacks are ignored, reducing the chance of stale UI/lifecycle state retaining a live server session.

The product does not send usage events to the project repository or support endpoint.
