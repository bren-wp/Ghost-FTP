# Privacy

ByFTP contains no advertising, analytics SDK, application telemetry or mandatory cloud account. Connection details are used only to contact the server selected by the user.

Desktop saved profiles and settings remain in the user's local application-data directory. Windows saved credentials use DPAPI. Production Go build gates keep Go telemetry disabled.

The Android 1.1.0 client does not persist connection passwords or SSH secrets. It uses Android document-provider URIs for user-selected upload/download files and does not request broad device-storage access. The Android module does not include Firebase Analytics, advertising SDKs or a ByFTP runtime backend.

The product does not require a fixed project runtime API and does not send usage events to the project repository or support endpoint.
