# Privacy

ByFTP contains no advertising, analytics SDK, application telemetry or mandatory cloud account. Connection details are used only to contact the server selected by the user.

## Desktop

Saved profiles/settings remain in the local application-data directory. Windows saved credentials use DPAPI. Production Go builds require Go telemetry to be disabled. External helper processes receive a minimized environment.

## Android

The Android client does not persist connection passwords or SSH secrets. It uses document-provider URIs for user-selected upload/download files and requests no broad device-storage permission.

Cloud-backup and device-transfer rules exclude root, file, database, shared-preference and external app-data domains. Generic cleartext traffic is disabled for platform-aware networking. No Firebase Analytics, advertising SDK, project-controlled runtime API or ByFTP backend service is included.

Active/pending clients and picker state are cleaned during lifecycle teardown. Release APK packaging does not change this privacy model.

## iOS

The native iOS client also has no ByFTP backend, analytics/advertising SDK or fixed runtime HTTP(S) endpoint.

Connection credentials are session-only. The password field is cleared after each connection attempt, the FTP transport clears its own password copy after authentication and the app disconnects when it enters the background. The implementation does not store credentials in `UserDefaults` or another persistent profile store.

Uploads use the system document picker and security-scoped access only for the selected file. Downloads are written to a private temporary directory for the system share/save workflow and can be explicitly cleared; the session store also removes the previous temporary download before creating a new one.

The iOS bundle contains no global App Transport Security bypass. Network traffic is limited to the endpoint entered by the user; PASV response hosts are not trusted as alternative destinations.

Unsigned IPA/app ZIP generation is a packaging/signing property only and does not introduce telemetry, cloud storage or credential persistence.

## Project communication

ByFTP does not send usage events to the repository, support endpoint or a project-operated service. Static repository/support URLs used as product metadata do not trigger automatic network requests.
